// Package frpclient 封装 frp 客户端（frpc）的启停、节点热更新与状态查询。
// 直接复用 frp v0.71.0 官方库，协议与命令行版 frpc 完全兼容，
// 可连接本程序的服务端，也可连接任意标准 frps。
package frpclient

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// ConnState 表示客户端与服务端的连接状态。
type ConnState string

const (
	ConnStateIdle       ConnState = "idle"       // 未启动
	ConnStateConnecting ConnState = "connecting" // 连接中
	ConnStateOnline     ConnState = "online"     // 已连接
	ConnStateOffline    ConnState = "offline"    // 连接断开（重试中）
)

// ProxyState 是单个代理节点的运行状态快照。
type ProxyState struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"` // new/wait start/start error/running/check failed/closed
	Err        string `json:"err"`
	RemoteAddr string `json:"remoteAddr"`
}

// Manager 管理 frpc 服务生命周期与节点配置，并发安全。
type Manager struct {
	mu      sync.Mutex
	running bool
	svr     *client.Service
	cancel  context.CancelFunc
	done    chan struct{}

	// gen 实例代号，每次 Start 递增。
	// 用于区分日志归属：旧实例退出时的残留日志不应影响新实例的状态。
	gen int

	connMu    sync.Mutex
	connState ConnState

	common  *v1.ClientCommonConfig // 当前生效的公共配置
	proxies []v1.ProxyConfigurer   // 当前生效的节点列表
}

// NewManager 创建客户端管理器。
func NewManager() *Manager {
	return &Manager{}
}

// SetConnState 更新连接状态（由日志桥的关键字监听回调）。
func (m *Manager) SetConnState(s ConnState) {
	m.connMu.Lock()
	m.connState = s
	m.connMu.Unlock()
}

// ConnState 返回当前连接状态。
func (m *Manager) ConnState() ConnState {
	m.connMu.Lock()
	defer m.connMu.Unlock()
	return m.connState
}

// Start 启动 frpc 服务。
// common 应已完成默认值填充；proxies 为初始节点列表。
func (m *Manager) Start(common *v1.ClientCommonConfig, proxies []v1.ProxyConfigurer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return errors.New("服务已在运行中")
	}

	// 构建配置源聚合器（frp v0.71.0 要求的配置载入方式）
	configSource := source.NewConfigSource()
	if err := configSource.ReplaceAll(proxies, nil); err != nil {
		return fmt.Errorf("载入节点配置失败: %w", err)
	}
	aggregator := source.NewAggregator(configSource)

	svr, err := client.NewService(client.ServiceOptions{
		Common:                 common,
		ConfigSourceAggregator: aggregator,
	})
	if err != nil {
		return fmt.Errorf("创建 frp 客户端失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m.svr = svr
	m.cancel = cancel
	m.done = done
	m.running = true
	m.gen++
	m.common = common
	m.proxies = proxies

	m.SetConnState(ConnStateConnecting)

	go func() {
		defer close(done)
		// LoginFailExit 已设为 false，Run 将阻塞直至 ctx 取消
		_ = svr.Run(ctx)
	}()

	// 给首次登录一点时间，让状态尽快反馈
	time.Sleep(300 * time.Millisecond)
	return nil
}

// Stop 停止 frpc 服务并等待资源释放。
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

	m.SetConnState(ConnStateIdle)

	if cancel != nil {
		cancel()
	}
	if svr != nil {
		svr.Close()
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

// Generation 返回当前实例代号（每次 Start 递增）。
// 调用方可用它区分新旧实例：代号不匹配的回调应忽略。
func (m *Manager) Generation() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gen
}

// UpdateProxies 更新节点列表。
// 服务运行中：热更新即时生效；未运行：仅记录，下次 Start 时生效。
// common 传 nil 表示沿用启动时的公共配置。
func (m *Manager) UpdateProxies(common *v1.ClientCommonConfig, proxies []v1.ProxyConfigurer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		// 未运行：只更新内部记录，供 States() 展示与下次启动使用
		if common != nil {
			m.common = common
		}
		m.proxies = proxies
		return nil
	}

	if m.svr == nil {
		return errors.New("服务未运行")
	}
	if common == nil {
		common = m.common
	}
	if err := m.svr.UpdateConfigSource(common, proxies, nil); err != nil {
		return fmt.Errorf("更新节点失败: %w", err)
	}
	m.common = common
	m.proxies = proxies
	return nil
}

// Proxies 返回当前生效的节点配置名称列表。
func (m *Manager) Proxies() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.proxies))
	for _, p := range m.proxies {
		if base := p.GetBaseConfig(); base != nil {
			names = append(names, base.Name)
		}
	}
	return names
}

// States 返回所有已配置节点的运行状态。
func (m *Manager) States() []ProxyState {
	m.mu.Lock()
	svr := m.svr
	proxies := make([]v1.ProxyConfigurer, len(m.proxies))
	copy(proxies, m.proxies)
	m.mu.Unlock()

	out := make([]ProxyState, 0, len(proxies))
	if svr == nil {
		for _, p := range proxies {
			base := p.GetBaseConfig()
			if base == nil {
				continue
			}
			out = append(out, ProxyState{
				Name:   base.Name,
				Type:   base.Type,
				Status: proxy.ProxyPhaseClosed,
			})
		}
		return out
	}

	exporter := svr.StatusExporter()
	for _, p := range proxies {
		base := p.GetBaseConfig()
		if base == nil {
			continue
		}
		st := ProxyState{Name: base.Name, Type: base.Type, Status: proxy.ProxyPhaseClosed}
		if ws, ok := exporter.GetProxyStatus(base.Name); ok && ws != nil {
			st.Status = ws.Phase
			st.Err = ws.Err
			st.RemoteAddr = ws.RemoteAddr
		}
		out = append(out, st)
	}
	return out
}
