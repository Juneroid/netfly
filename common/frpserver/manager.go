// Package frpserver 封装 frp 服务端（frps）的启停管理与运行统计。
// 直接复用 frp v0.71.0 官方库，协议与命令行版 frps 完全兼容。
package frpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/metrics/mem"
	"github.com/fatedier/frp/server"

	// 注册 frps dashboard 静态资源（embed），管理网页依赖它
	_ "github.com/fatedier/frp/web/frps"
)

// proxyTypes 服务端需要统计展示的全部代理类型。
var proxyTypes = []string{"tcp", "udp", "http", "https", "tcpmux", "stcp", "sudp", "xtcp"}

// Stats 是服务端运行统计快照，供 GUI 展示。
type Stats struct {
	Running           bool           `json:"running"`
	ClientCount       int64          `json:"clientCount"`       // 在线客户端数
	ProxyCount        int64          `json:"proxyCount"`        // 在线代理总数
	ProxyTypeCounts   map[string]int64 `json:"proxyTypeCounts"` // 各类型代理数
	CurConns          int64          `json:"curConns"`          // 当前连接数
	SessionTrafficIn  int64          `json:"sessionTrafficIn"`  // 本次运行流入流量（字节）
	SessionTrafficOut int64          `json:"sessionTrafficOut"` // 本次运行流出流量（字节）
}

// ProxyBrief 是服务端视角的单个代理摘要信息。
type ProxyBrief struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	User            string `json:"user"`
	CurConns        int64  `json:"curConns"`
	TodayTrafficIn  int64  `json:"todayTrafficIn"`
	TodayTrafficOut int64  `json:"todayTrafficOut"`
}

// Manager 管理 frps 服务生命周期，并发安全。
type Manager struct {
	mu      sync.Mutex
	running bool
	svr     *server.Service // 当前服务实例（停止时主动关闭以释放端口）
	cancel  context.CancelFunc
	done    chan struct{}

	// runCount 记录调用 server.NewService 的次数。
	// frp 库每次 NewService 都会重复注册 metrics（同一实例多次注册），
	// 导致后续流量按次数 N 放大，读取时需除以 N 修正。
	runCount int
	baseIn   int64 // 本次会话开始时的流量基线（用于差分）
	baseOut  int64
}

// NewManager 创建服务端管理器。
func NewManager() *Manager {
	return &Manager{}
}

// checkPortFree 预检端口是否可绑定。
// frp 的 NewService 在创建 dashboard 监听后才校验业务端口，
// 若业务端口冲突会导致 dashboard 监听器泄漏，因此这里提前拦截。
func checkPortFree(addr string, port int) error {
	if port <= 0 {
		return nil
	}
	if addr == "" {
		addr = "0.0.0.0"
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(addr, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("端口 %d 无法监听（可能被占用）: %w", port, err)
	}
	return ln.Close()
}

// Start 启动 frps 服务。cfg 应已完成默认值填充（Complete）。
func (m *Manager) Start(cfg *v1.ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return errors.New("服务已在运行中")
	}

	// 端口预检：避免 NewService 半途失败造成监听器泄漏
	if err := checkPortFree(cfg.BindAddr, cfg.BindPort); err != nil {
		return err
	}
	if cfg.WebServer.Port > 0 {
		if err := checkPortFree(cfg.WebServer.Addr, cfg.WebServer.Port); err != nil {
			return err
		}
	}
	if err := checkPortFree(cfg.ProxyBindAddr, cfg.VhostHTTPPort); err != nil {
		return err
	}
	if err := checkPortFree(cfg.ProxyBindAddr, cfg.VhostHTTPSPort); err != nil {
		return err
	}
	if err := checkPortFree(cfg.ProxyBindAddr, cfg.TCPMuxHTTPConnectPort); err != nil {
		return err
	}

	// 记录流量基线与 NewService 调用次数（用于流量修正）
	if stats := mem.StatsCollector.GetServer(); stats != nil {
		m.baseIn = stats.TotalTrafficIn
		m.baseOut = stats.TotalTrafficOut
	}
	m.runCount++

	svr, err := server.NewService(cfg)
	if err != nil {
		return fmt.Errorf("启动 frp 服务失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m.svr = svr
	m.cancel = cancel
	m.done = done
	m.running = true

	go func() {
		defer close(done)
		svr.Run(ctx)
	}()

	// 等待监听器就绪，尽早把启动错误暴露给调用方
	time.Sleep(200 * time.Millisecond)
	return nil
}

// Stop 停止 frps 服务并等待端口释放。
// 注意：frp 的 Service.Run 阻塞在 Accept 循环且不监听 ctx，
// 仅 cancel 无法触发其内部 Close，必须主动调用 svr.Close()
// 关闭全部 listener 后 Run 才能退出、端口才得以释放。
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	svr := m.svr
	cancel := m.cancel
	done := m.done
	m.running = false
	m.svr = nil
	m.cancel = nil
	m.done = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if svr != nil {
		// 主动关闭所有 listener/dashboard/会话，触发 Accept 报错返回
		_ = svr.Close()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
}

// IsRunning 返回服务是否处于运行状态。
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// Stats 采集当前运行统计。流量经过重复注册修正后为真实值。
func (m *Manager) Stats() Stats {
	m.mu.Lock()
	running, runCount, baseIn, baseOut := m.running, m.runCount, m.baseIn, m.baseOut
	m.mu.Unlock()

	st := Stats{Running: running, ProxyTypeCounts: map[string]int64{}}
	if s := mem.StatsCollector.GetServer(); s != nil {
		st.ClientCount = s.ClientCounts
		st.CurConns = s.CurConns
		st.ProxyTypeCounts = s.ProxyTypeCounts
		var total int64
		for _, c := range s.ProxyTypeCounts {
			total += c
		}
		st.ProxyCount = total

		if runCount > 0 {
			st.SessionTrafficIn = (s.TotalTrafficIn - baseIn) / int64(runCount)
			st.SessionTrafficOut = (s.TotalTrafficOut - baseOut) / int64(runCount)
			if st.SessionTrafficIn < 0 {
				st.SessionTrafficIn = 0
			}
			if st.SessionTrafficOut < 0 {
				st.SessionTrafficOut = 0
			}
		}
	}
	return st
}

// Proxies 返回当前在线代理的摘要列表。
func (m *Manager) Proxies() []ProxyBrief {
	out := make([]ProxyBrief, 0)
	for _, t := range proxyTypes {
		for _, p := range mem.StatsCollector.GetProxiesByType(t) {
			out = append(out, ProxyBrief{
				Name:            p.Name,
				Type:            p.Type,
				User:            p.User,
				CurConns:        p.CurConns,
				TodayTrafficIn:  p.TodayTrafficIn,
				TodayTrafficOut: p.TodayTrafficOut,
			})
		}
	}
	return out
}
