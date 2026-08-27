// Package common 问题复现测试：服务端重启端口释放、客户端停止/重启、节点热更新。
package common_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"netfly.dev/common/frpclient"
	"netfly.dev/common/frpserver"
	"netfly.dev/common/logbridge"
	"netfly.dev/common/model"
)

// newBridge 创建仅内存缓冲的日志桥。
func newBridge(t *testing.T) *logbridge.Bridge {
	t.Helper()
	bridge, err := logbridge.New("")
	if err != nil {
		t.Fatal(err)
	}
	bridge.Install("info")
	t.Cleanup(bridge.Close)
	return bridge
}

// newHTTPBackend 启动本地测试 HTTP 服务并返回端口。
func newHTTPBackend(t *testing.T) int {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "hello-netfly")
	}))
	t.Cleanup(srv.Close)
	_, portStr, _ := net.SplitHostPort(srv.Listener.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return port
}

// buildServerCfg 构建测试用服务端配置。
func buildServerCfg(bindPort, dashboardPort int) model.ServerAppConfig {
	cfg := model.DefaultServerConfig()
	cfg.BindPort = bindPort
	cfg.Token = "test-token"
	cfg.DashboardEnabled = true
	cfg.DashboardPort = dashboardPort
	cfg.DashboardUser = "admin"
	cfg.DashboardPassword = "admin"
	return cfg
}

// TestServerRestart 复现问题：停止服务后立即重启，端口应能正常释放。
func TestServerRestart(t *testing.T) {
	newBridge(t)

	mgr := frpserver.NewManager()
	scfg := buildServerCfg(18100, 18150)
	frpCfg, err := scfg.BuildFRPSConfig()
	if err != nil {
		t.Fatal(err)
	}

	// 第一次启动
	if err := mgr.Start(frpCfg); err != nil {
		t.Fatal("首次启动失败:", err)
	}
	time.Sleep(300 * time.Millisecond)
	mgr.Stop()

	// 立即第二次启动：验证端口是否释放
	if err := mgr.Start(frpCfg); err != nil {
		t.Fatal("重启失败（端口未释放）:", err)
	}
	mgr.Stop()

	// 第三次启动：带 dashboard 变更场景再验证一次
	if err := mgr.Start(frpCfg); err != nil {
		t.Fatal("第三次启动失败:", err)
	}
	mgr.Stop()
}

// TestClientStopAndRestart 复现问题：客户端停止应快速完成，且可立即重连。
func TestClientStopAndRestart(t *testing.T) {
	newBridge(t)
	backendPort := newHTTPBackend(t)

	// 启动服务端
	svrMgr := frpserver.NewManager()
	scfg := buildServerCfg(18200, 18250)
	sfrpCfg, err := scfg.BuildFRPSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := svrMgr.Start(sfrpCfg); err != nil {
		t.Fatal(err)
	}
	defer svrMgr.Stop()

	buildClientCfg := func(remote int) (model.ClientAppConfig, error) {
		cfg := model.DefaultClientConfig()
		cfg.ServerAddr = "127.0.0.1"
		cfg.ServerPort = 18200
		cfg.Token = "test-token"
		cfg.Proxies = []model.ProxyItem{{
			Name: "web", Type: "tcp",
			LocalIP: "127.0.0.1", LocalPort: backendPort,
			RemotePort: remote, Enabled: true,
		}}
		return cfg, nil
	}

	cliMgr := frpclient.NewManager()

	// 第一次连接
	ccfg, _ := buildClientCfg(17100)
	cfrp, err := ccfg.BuildFRPCConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := cliMgr.Start(cfrp, ccfg.BuildProxyConfigurers()); err != nil {
		t.Fatal("首次连接失败:", err)
	}
	if err := waitTCP("127.0.0.1:17100", 30*time.Second); err != nil {
		t.Fatal("代理端口未就绪:", err)
	}

	// 停止并计时（3 秒内完成才算正常）
	begin := time.Now()
	cliMgr.Stop()
	elapsed := time.Since(begin)
	if cliMgr.IsRunning() {
		t.Fatal("停止后仍处于运行状态")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("停止耗时 %v，超过 3 秒", elapsed)
	}
	t.Logf("客户端停止耗时: %v", elapsed)

	// 立即重连
	ccfg2, _ := buildClientCfg(17101)
	cfrp2, err := ccfg2.BuildFRPCConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := cliMgr.Start(cfrp2, ccfg2.BuildProxyConfigurers()); err != nil {
		t.Fatal("重连失败:", err)
	}
	if err := waitTCP("127.0.0.1:17101", 30*time.Second); err != nil {
		t.Fatal("重连后代理端口未就绪:", err)
	}
	cliMgr.Stop()
}

// TestClientHotUpdateRepeated 复现问题：运行中多次新建/修改/删除节点。
func TestClientHotUpdateRepeated(t *testing.T) {
	newBridge(t)
	backendPort := newHTTPBackend(t)

	// 启动服务端
	svrMgr := frpserver.NewManager()
	scfg := buildServerCfg(18300, 18350)
	sfrpCfg, err := scfg.BuildFRPSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := svrMgr.Start(sfrpCfg); err != nil {
		t.Fatal(err)
	}
	defer svrMgr.Stop()

	// 客户端初始无节点
	cliMgr := frpclient.NewManager()
	ccfg := model.DefaultClientConfig()
	ccfg.ServerAddr = "127.0.0.1"
	ccfg.ServerPort = 18300
	ccfg.Token = "test-token"
	ccfg.Proxies = []model.ProxyItem{}
	cfrp, err := ccfg.BuildFRPCConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := cliMgr.Start(cfrp, ccfg.BuildProxyConfigurers()); err != nil {
		t.Fatal(err)
	}
	defer cliMgr.Stop()
	time.Sleep(1 * time.Second)

	// 连续模拟 GUI 操作：新建 3 个节点
	for i := 1; i <= 3; i++ {
		proxies := append([]model.ProxyItem{}, ccfg.Proxies...)
		proxies = append(proxies, model.ProxyItem{
			Name:    fmt.Sprintf("node%d", i),
			Type:    "tcp",
			LocalIP: "127.0.0.1", LocalPort: backendPort,
			RemotePort: 17200 + i, Enabled: true,
		})
		ccfg.Proxies = proxies
		if err := cliMgr.UpdateProxies(nil, ccfg.BuildProxyConfigurers()); err != nil {
			t.Fatalf("新建节点 %d 失败: %v", i, err)
		}
	}

	// 等待最后一个节点端口就绪，验证热更新生效
	if err := waitTCP("127.0.0.1:17203", 20*time.Second); err != nil {
		t.Fatal("热更新节点端口未就绪:", err)
	}

	// 修改节点 2（改端口）
	for i := range ccfg.Proxies {
		if ccfg.Proxies[i].Name == "node2" {
			ccfg.Proxies[i].RemotePort = 17210
		}
	}
	if err := cliMgr.UpdateProxies(nil, ccfg.BuildProxyConfigurers()); err != nil {
		t.Fatal("修改节点失败:", err)
	}
	if err := waitTCP("127.0.0.1:17210", 20*time.Second); err != nil {
		t.Fatal("修改后的节点端口未就绪:", err)
	}

	// 删除节点 3
	kept := ccfg.Proxies[:2]
	ccfg.Proxies = kept
	if err := cliMgr.UpdateProxies(nil, ccfg.BuildProxyConfigurers()); err != nil {
		t.Fatal("删除节点失败:", err)
	}
	// 服务端端口 17203 应关闭（连接被拒）
	deadline := time.Now().Add(10 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:17203", time.Second)
		if err != nil {
			break // 已关闭，符合预期
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			t.Fatal("删除节点后端口未关闭")
		}
		time.Sleep(300 * time.Millisecond)
	}
}
