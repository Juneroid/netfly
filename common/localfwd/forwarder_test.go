// Package localfwd 单元测试：转发、统计、同步与删除。
package localfwd_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"netfly.dev/common/localfwd"
)

// TestForwarderBasic 验证基本转发与流量统计。
func TestForwarderBasic(t *testing.T) {
	// 真实本地服务
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "hello-fwd")
	}))
	defer srv.Close()
	_, portStr, _ := net.SplitHostPort(srv.Listener.Addr().String())
	var targetPort int
	fmt.Sscanf(portStr, "%d", &targetPort)

	// 仅内存统计（不落盘）
	m, err := localfwd.NewManager("")
	if err != nil {
		t.Fatal(err)
	}

	// 同步一个转发器
	if err := m.Sync([]localfwd.Item{{
		Name: "web", TargetIP: "127.0.0.1", TargetPort: targetPort, TCPBased: true,
	}}); err != nil {
		t.Fatal(err)
	}
	ports := m.Ports()
	fwdPort, ok := ports["web"]
	if !ok || fwdPort <= 0 {
		t.Fatalf("转发端口异常: %+v", ports)
	}

	// 通过转发端口访问本地服务
	resp, err := (&http.Client{Timeout: 5 * time.Second}).
		Get(fmt.Sprintf("http://127.0.0.1:%d", fwdPort))
	if err != nil {
		t.Fatal("转发访问失败:", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "hello-fwd" {
		t.Fatalf("转发内容异常: %q", string(body))
	}

	// 等待统计更新后校验流量
	time.Sleep(300 * time.Millisecond)
	stats := m.Stats()
	s, ok := stats["web"]
	if !ok || s.In+s.Out <= 0 {
		t.Fatalf("流量统计异常: %+v", stats)
	}
	t.Logf("web 流量: in=%d out=%d", s.In, s.Out)

	// 删除节点后统计清零
	m.Remove("web")
	if _, ok := m.Stats()["web"]; ok {
		t.Fatal("删除节点后统计未清零")
	}
	if len(m.Ports()) != 0 {
		t.Fatal("删除节点后转发器未关闭")
	}
}

// TestForwarderSyncUpdate 验证目标地址变更时转发器重建、节点移除时关闭。
func TestForwarderSyncUpdate(t *testing.T) {
	srvA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "A")
	}))
	defer srvA.Close()
	srvB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "B")
	}))
	var portA, portB int
	_, pa, _ := net.SplitHostPort(srvA.Listener.Addr().String())
	_, pb, _ := net.SplitHostPort(srvB.Listener.Addr().String())
	fmt.Sscanf(pa, "%d", &portA)
	fmt.Sscanf(pb, "%d", &portB)

	m, _ := localfwd.NewManager("")

	// 先指向 A
	if err := m.Sync([]localfwd.Item{
		{Name: "n1", TargetIP: "127.0.0.1", TargetPort: portA, TCPBased: true},
	}); err != nil {
		t.Fatal(err)
	}
	p1 := m.Ports()["n1"]

	// 改指向 B，端口应重建
	if err := m.Sync([]localfwd.Item{
		{Name: "n1", TargetIP: "127.0.0.1", TargetPort: portB, TCPBased: true},
	}); err != nil {
		t.Fatal(err)
	}
	p2 := m.Ports()["n1"]
	if p2 == 0 {
		t.Fatal("重建后端口异常")
	}
	_ = p1 // 重建后旧端口关闭即可

	// 验证转发到 B
	resp, err := (&http.Client{Timeout: 5 * time.Second}).
		Get(fmt.Sprintf("http://127.0.0.1:%d", p2))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "B" {
		t.Fatalf("目标变更后转发异常: %q", string(body))
	}

	// 空列表同步：全部关闭
	if err := m.Sync(nil); err != nil {
		t.Fatal(err)
	}
	if len(m.Ports()) != 0 {
		t.Fatal("清空后转发器未关闭")
	}
}
