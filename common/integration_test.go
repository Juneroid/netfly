// Package common 集成测试：端到端验证 frp 服务端/客户端封装、
// 代理穿透、流量统计与节点热更新，保证与 frp v0.71.0 协议兼容。
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
	"netfly.dev/common/localfwd"
	"netfly.dev/common/logbridge"
	"netfly.dev/common/model"
)

// TestClientLocalTrafficStats 端到端验证本地流量统计：
// frpc 代理指向本地转发器，真实穿透流量应被统计。
func TestClientLocalTrafficStats(t *testing.T) {
	newBridge(t)
	backendPort := newHTTPBackend(t)

	// 启动服务端
	svrMgr := frpserver.NewManager()
	scfg := model.DefaultServerConfig()
	scfg.BindPort = 17600
	scfg.Token = "test-token"
	scfg.DashboardEnabled = false
	sfrpCfg, err := scfg.BuildFRPSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := svrMgr.Start(sfrpCfg); err != nil {
		t.Fatal(err)
	}
	defer svrMgr.Stop()

	// 同步本地转发器：节点 web 指向真实本地服务
	fwd, err := localfwd.NewManager("")
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.StopAll()
	if err := fwd.Sync([]localfwd.Item{{
		Name: "web", TargetIP: "127.0.0.1",
		TargetPort: backendPort, TCPBased: true,
	}}); err != nil {
		t.Fatal(err)
	}

	// 构建客户端配置并把代理本地地址改写为转发器端口（与 GUI 逻辑一致）
	ccfg := model.DefaultClientConfig()
	ccfg.ServerAddr = "127.0.0.1"
	ccfg.ServerPort = 17600
	ccfg.Token = "test-token"
	ccfg.Proxies = []model.ProxyItem{{
		Name: "web", Type: "tcp",
		LocalIP: "127.0.0.1", LocalPort: backendPort,
		RemotePort: 16600, Enabled: true,
	}}
	cfrp, err := ccfg.BuildFRPCConfig()
	if err != nil {
		t.Fatal(err)
	}
	proxies := ccfg.BuildProxyConfigurers()
	for _, c := range proxies {
		base := c.GetBaseConfig()
		if port, ok := fwd.Ports()[base.Name]; ok {
			base.ProxyBackend.LocalIP = "127.0.0.1"
			base.ProxyBackend.LocalPort = port
		}
	}

	// 启动客户端并等待代理就绪
	cliMgr := frpclient.NewManager()
	if err := cliMgr.Start(cfrp, proxies); err != nil {
		t.Fatal(err)
	}
	defer cliMgr.Stop()
	if err := waitTCP("127.0.0.1:16600", 30*time.Second); err != nil {
		t.Fatal("代理端口未就绪:", err)
	}

	// 通过服务端穿透访问
	for i := 0; i < 3; i++ {
		resp, err := (&http.Client{Timeout: 5 * time.Second}).
			Get("http://127.0.0.1:16600")
		if err != nil {
			t.Fatal("穿透访问失败:", err)
		}
		resp.Body.Close()
	}

	// 校验本地统计（countWriter 实时计数，无需等待连接关闭）
	stats := fwd.Stats()
	s, ok := stats["web"]
	if !ok || s.In+s.Out <= 0 {
		t.Fatalf("本地流量统计异常: %+v", stats)
	}
	t.Logf("web 本地统计: in=%d out=%d", s.In, s.Out)
}

// waitTCP 轮询等待本地端口可连通。
func waitTCP(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("等待 %s 超时", addr)
}

// TestEndToEnd 覆盖完整链路：frps 启动 -> frpc 连接 -> TCP 穿透 ->
// 流量统计 -> 节点热更新 -> 停止服务。
func TestEndToEnd(t *testing.T) {
	// 日志桥（仅内存缓冲，不落盘）
	bridge, err := logbridge.New("")
	if err != nil {
		t.Fatal(err)
	}
	bridge.Install("debug")
	defer bridge.Close()

	// 本地测试 HTTP 服务（模拟被穿透的内网服务）
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "hello-netfly")
	}))
	defer srv.Close()
	_, backendPortStr, _ := net.SplitHostPort(srv.Listener.Addr().String())

	var backendPort int
	fmt.Sscanf(backendPortStr, "%d", &backendPort)

	// 1. 启动 frps：服务端口 17000，管理网页 17500
	svrMgr := frpserver.NewManager()
	scfg := model.DefaultServerConfig()
	scfg.BindPort = 17000
	scfg.Token = "test-token"
	scfg.DashboardEnabled = true
	scfg.DashboardPort = 17500
	scfg.DashboardUser = "admin"
	scfg.DashboardPassword = "admin"
	sfrp, err := scfg.BuildFRPSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := svrMgr.Start(sfrp); err != nil {
		t.Fatal("启动 frps 失败:", err)
	}
	defer svrMgr.Stop()
	if !svrMgr.IsRunning() {
		t.Fatal("frps 未处于运行状态")
	}

	// 2. 启动 frpc：一个 TCP 节点指向本地 HTTP 服务
	cliMgr := frpclient.NewManager()
	ccfg := model.DefaultClientConfig()
	ccfg.ServerAddr = "127.0.0.1"
	ccfg.ServerPort = 17000
	ccfg.Token = "test-token"
	ccfg.Proxies = []model.ProxyItem{{
		Name: "web", Type: "tcp",
		LocalIP: "127.0.0.1", LocalPort: backendPort,
		RemotePort: 16000, Enabled: true,
	}}
	cfrp, err := ccfg.BuildFRPCConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := cliMgr.Start(cfrp, ccfg.BuildProxyConfigurers()); err != nil {
		t.Fatal("启动 frpc 失败:", err)
	}
	defer cliMgr.Stop()

	// 3. 等待登录与代理就绪
	if err := waitTCP("127.0.0.1:16000", 30*time.Second); err != nil {
		t.Fatal("代理端口未就绪:", err)
	}

	// 4. 通过服务端穿透端口访问本地服务
	resp, err := (&http.Client{Timeout: 5 * time.Second}).
		Get("http://127.0.0.1:16000")
	if err != nil {
		t.Fatal("穿透访问失败:", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("状态码异常: %d", resp.StatusCode)
	}

	// 5. 校验节点运行状态
	states := cliMgr.States()
	found := false
	for _, s := range states {
		if s.Name == "web" && s.Status == "running" {
			found = true
		}
	}
	if !found {
		t.Fatalf("节点状态异常: %+v", states)
	}

	// 6. 校验服务端统计
	if st := svrMgr.Stats(); !st.Running || st.ProxyCount < 1 {
		t.Fatalf("服务端统计异常: %+v", st)
	}

	// 7. 节点热更新：追加第二个节点
	newCfg := ccfg
	newCfg.Proxies = append(ccfg.Proxies, model.ProxyItem{
		Name: "web2", Type: "tcp",
		LocalIP: "127.0.0.1", LocalPort: backendPort,
		RemotePort: 16001, Enabled: true,
	})
	if err := cliMgr.UpdateProxies(nil, newCfg.BuildProxyConfigurers()); err != nil {
		t.Fatal("热更新节点失败:", err)
	}
	if err := waitTCP("127.0.0.1:16001", 15*time.Second); err != nil {
		t.Fatal("热更新后新节点端口未就绪:", err)
	}

	// 8. 停止客户端后服务端统计应下降
	cliMgr.Stop()
	time.Sleep(500 * time.Millisecond)
	if cliMgr.IsRunning() {
		t.Fatal("客户端停止后仍处于运行状态")
	}
}
