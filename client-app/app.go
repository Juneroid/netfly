// Package main - NetFly 客户端应用逻辑层。
// 通过 Wails 绑定向前端暴露服务端连接、节点管理、日志、网络统计接口。
// 流量统计由本地 TCP 转发器（localfwd）完成，不依赖服务端。
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	v1 "github.com/fatedier/frp/pkg/config/v1"

	"netfly.dev/common/appcfg"
	"netfly.dev/common/frpclient"
	"netfly.dev/common/localfwd"
	"netfly.dev/common/logbridge"
	"netfly.dev/common/model"
	"netfly.dev/common/winui"
)

// windowTitle 主窗口标题，用于 Win32 查找句柄（须与 main.go 一致）。
const windowTitle = "NetFly 客户端"

// NodeNetStat 单个节点的网络统计（流量与实时速率）。
type NodeNetStat struct {
	Name       string  `json:"name"`
	TrafficIn  int64   `json:"trafficIn"`  // 累计入站字节（来自服务端方向）
	TrafficOut int64   `json:"trafficOut"` // 累计出站字节（发往服务端方向）
	RateIn     float64 `json:"rateIn"`     // 实时入站速率 B/s
	RateOut    float64 `json:"rateOut"`    // 实时出站速率 B/s
}

// NetworkStats 网络统计汇总（本地统计，始终可用）。
type NetworkStats struct {
	ConnState string        `json:"connState"` // 连接状态
	Nodes     []NodeNetStat `json:"nodes"`
	TotalIn   int64         `json:"totalIn"`
	TotalOut  int64         `json:"totalOut"`
}

// App 客户端应用对象，承载全部业务状态。
type App struct {
	ctx    context.Context
	store  *appcfg.Store      // 配置持久化
	bridge *logbridge.Bridge  // frp 日志桥
	mgr    *frpclient.Manager // frpc 服务管理器
	fwd    *localfwd.Manager  // 本地转发器（流量统计）
	cfg    model.ClientAppConfig

	// curGen 当前 frpc 实例代号：日志回调据此忽略旧实例残留日志
	curGen int

	// 速率计算所需的上次采样
	rateMu     sync.Mutex
	lastSample map[string]localfwd.Stat
	lastTime   time.Time
}

// NewApp 初始化应用：加载配置、创建日志桥、转发器与服务管理器。
func NewApp() (*App, error) {
	store, err := appcfg.NewStore("client")
	if err != nil {
		return nil, err
	}
	bridge, err := logbridge.New(store.DataDir() + "\\logs")
	if err != nil {
		return nil, err
	}
	bridge.Install("info")

	fwd, err := localfwd.NewManager(store.DataDir())
	if err != nil {
		return nil, err
	}

	cfg := model.DefaultClientConfig()
	if err := store.Load(&cfg); err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return &App{
		store:      store,
		bridge:     bridge,
		mgr:        frpclient.NewManager(),
		fwd:        fwd,
		cfg:        cfg,
		lastSample: map[string]localfwd.Stat{},
	}, nil
}

// startup Wails 启动回调：注册日志转发、连接状态监听、按需自动启动。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// frp 日志实时推送给前端
	a.bridge.OnLine(func(line string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "log", line)
		}
	})

	// 通过日志关键字跟踪与服务端的连接状态（frp v0.71.0 固定文案）。
	// 仅当回调属于当前运行实例时才生效：
	// 旧实例停止时的残留日志（如 context canceled 触发的 error）不影响新状态。
	onConnEvent := func(state frpclient.ConnState) {
		if a.mgr.IsRunning() && a.mgr.Generation() == a.curGen {
			a.mgr.SetConnState(state)
			a.emitConnState(state)
		}
	}
	a.bridge.OnKeyword("login to server success", func() { onConnEvent(frpclient.ConnStateOnline) })
	a.bridge.OnKeyword("try to connect to server", func() { onConnEvent(frpclient.ConnStateConnecting) })
	a.bridge.OnKeyword("connect to server error", func() { onConnEvent(frpclient.ConnStateOffline) })

	// 配置了自动启动时，延迟连接服务
	if a.cfg.AutoStart {
		go func() {
			if err := a.StartService(); err != nil {
				runtime.EventsEmit(a.ctx, "log", "自动连接失败: "+err.Error())
			}
		}()
	}
}

// emitConnState 向前端推送连接状态变化。
func (a *App) emitConnState(s frpclient.ConnState) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "connState", string(s))
	}
}

// shutdown Wails 退出回调：停止服务并释放资源。
func (a *App) shutdown(_ context.Context) {
	a.mgr.Stop()
	a.fwd.StopAll()
	a.bridge.Close()
}

// showWindow 显示并前置主窗口（托盘菜单/图标点击触发）。
// 该方法在独立 goroutine 中被调用，不阻塞托盘消息循环。
// 先用 Win32 强制恢复+前置（解决长期隐藏后 SW_SHOW 不生效），
// 再用 Wails 运行时兜底。
func (a *App) showWindow() {
	if hwnd := winui.FindWindowByTitle(windowTitle); hwnd != 0 {
		winui.WakeWindow(hwnd)
	}
	if a.ctx != nil {
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowShow(a.ctx)
	}
}

// quitApp 退出整个程序（托盘菜单触发）。
func (a *App) quitApp() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// GetConfig 返回当前客户端配置。
func (a *App) GetConfig() model.ClientAppConfig {
	return a.cfg
}

// SaveConfig 校验并保存配置。
// 允许服务端地址暂为空（稍后在设置中填写），仅校验节点合法性。
func (a *App) SaveConfig(cfg model.ClientAppConfig) error {
	if err := cfg.ValidateProxies(); err != nil {
		return err
	}
	if err := a.store.Save(cfg); err != nil {
		return err
	}
	a.cfg = cfg
	return nil
}

// fwdItems 将启用的节点转换为转发器输入项（UDP 不支持本地统计）。
func fwdItems(proxies []model.ProxyItem) []localfwd.Item {
	items := make([]localfwd.Item, 0, len(proxies))
	for _, p := range proxies {
		if !p.Enabled {
			continue
		}
		items = append(items, localfwd.Item{
			Name:       p.Name,
			TargetIP:   p.LocalIP,
			TargetPort: p.LocalPort,
			TCPBased:   p.Type != model.ProxyTypeUDP,
		})
	}
	return items
}

// rewriteLocalAddr 将代理的本地地址改写为本地转发器端口，
// 使流量经过转发器完成统计。
func rewriteLocalAddr(cfgs []v1.ProxyConfigurer, ports map[string]int) {
	for _, c := range cfgs {
		base := c.GetBaseConfig()
		if base == nil {
			continue
		}
		if port, ok := ports[base.Name]; ok {
			base.ProxyBackend.LocalIP = "127.0.0.1"
			base.ProxyBackend.LocalPort = port
		}
	}
}

// buildProxiesWithForwarders 同步转发器并返回改写本地地址后的代理配置。
func (a *App) buildProxiesWithForwarders(cfg model.ClientAppConfig) ([]v1.ProxyConfigurer, error) {
	if err := a.fwd.Sync(fwdItems(cfg.Proxies)); err != nil {
		return nil, err
	}
	cfgs := cfg.BuildProxyConfigurers()
	rewriteLocalAddr(cfgs, a.fwd.Ports())
	return cfgs, nil
}

// StartService 按当前配置连接服务端并启动全部启用的节点。
// 连接参数（地址/端口）在此处才做强校验。
func (a *App) StartService() error {
	if a.mgr.IsRunning() {
		return fmt.Errorf("服务已在运行中")
	}
	if err := a.cfg.ValidateConnection(); err != nil {
		return err
	}
	a.bridge.Install(a.cfg.LogLevel)

	common, err := a.cfg.BuildFRPCConfig()
	if err != nil {
		return err
	}
	proxies, err := a.buildProxiesWithForwarders(a.cfg)
	if err != nil {
		return err
	}
	if err := a.mgr.Start(common, proxies); err != nil {
		return err
	}
	a.curGen = a.mgr.Generation()
	a.resetRateSample()
	return nil
}

// StopService 断开连接并停止服务（转发器统计随轮询自动落盘）。
func (a *App) StopService() {
	a.mgr.Stop()
	a.fwd.StopAll()
	a.emitConnState(frpclient.ConnStateIdle)
}

// ServiceRunning 返回服务运行状态。
func (a *App) ServiceRunning() bool {
	return a.mgr.IsRunning()
}

// GetConnState 返回与服务端的连接状态；服务未运行时一律视为未连接。
func (a *App) GetConnState() string {
	if !a.mgr.IsRunning() {
		return string(frpclient.ConnStateIdle)
	}
	return string(a.mgr.ConnState())
}

// GetProxyStates 返回全部节点的运行状态（本地 frpc 实例查询）。
func (a *App) GetProxyStates() []frpclient.ProxyState {
	return a.mgr.States()
}

// applyProxies 校验保存并更新节点列表（运行中热更新即时生效）。
// 仅校验节点本身，服务端地址是否已配置不影响节点管理。
func (a *App) applyProxies(proxies []model.ProxyItem) error {
	cfg := a.cfg
	cfg.Proxies = proxies
	if err := cfg.ValidateProxies(); err != nil {
		return err
	}
	if err := a.store.Save(cfg); err != nil {
		return err
	}
	a.cfg = cfg

	// 同步转发器并把代理本地地址改写为转发端口
	frpProxies, err := a.buildProxiesWithForwarders(cfg)
	if err != nil {
		return err
	}
	// 运行中热更新；未运行时仅记录，供界面展示与下次启动
	return a.mgr.UpdateProxies(nil, frpProxies)
}

// AddProxy 新增节点。
func (a *App) AddProxy(item model.ProxyItem) error {
	if err := item.Validate(); err != nil {
		return err
	}
	proxies := append(append([]model.ProxyItem{}, a.cfg.Proxies...), item)
	return a.applyProxies(proxies)
}

// UpdateProxy 更新指定名称的节点。
func (a *App) UpdateProxy(name string, item model.ProxyItem) error {
	if err := item.Validate(); err != nil {
		return err
	}
	proxies := append([]model.ProxyItem{}, a.cfg.Proxies...)
	for i := range proxies {
		if proxies[i].Name == name {
			proxies[i] = item
			return a.applyProxies(proxies)
		}
	}
	return fmt.Errorf("节点 [%s] 不存在", name)
}

// DeleteProxy 删除指定名称的节点（连同其流量统计记录）。
func (a *App) DeleteProxy(name string) error {
	proxies := make([]model.ProxyItem, 0, len(a.cfg.Proxies))
	found := false
	for _, p := range a.cfg.Proxies {
		if p.Name == name {
			found = true
			continue
		}
		proxies = append(proxies, p)
	}
	if !found {
		return fmt.Errorf("节点 [%s] 不存在", name)
	}
	// 删除后先生效配置，再清理该节点的统计记录
	if err := a.applyProxies(proxies); err != nil {
		return err
	}
	a.fwd.Remove(name)
	return nil
}

// ToggleProxy 启用或停用指定节点（运行中即时生效）。
func (a *App) ToggleProxy(name string) error {
	proxies := append([]model.ProxyItem{}, a.cfg.Proxies...)
	for i := range proxies {
		if proxies[i].Name == name {
			proxies[i].Enabled = !proxies[i].Enabled
			return a.applyProxies(proxies)
		}
	}
	return fmt.Errorf("节点 [%s] 不存在", name)
}

// GetLogs 返回日志缓冲区内容。
func (a *App) GetLogs() []string {
	return a.bridge.Lines()
}

// resetRateSample 重置速率采样基线（服务启动时调用）。
func (a *App) resetRateSample() {
	a.rateMu.Lock()
	a.lastSample = map[string]localfwd.Stat{}
	a.lastTime = time.Time{}
	a.rateMu.Unlock()
}

// GetNetworkStats 返回当前配置节点的本地流量统计与实时速率。
// UDP 节点不支持本地统计，无数据。
func (a *App) GetNetworkStats() NetworkStats {
	stats := NetworkStats{
		ConnState: a.GetConnState(),
		Nodes:     []NodeNetStat{},
	}

	all := a.fwd.Stats()

	a.rateMu.Lock()
	now := time.Now()
	validSample := !a.lastTime.IsZero() && now.After(a.lastTime)
	elapsed := now.Sub(a.lastTime).Seconds()
	last := a.lastSample

	for _, p := range a.cfg.Proxies {
		if p.Type == model.ProxyTypeUDP {
			continue
		}
		s, ok := all[p.Name]
		if !ok {
			continue
		}
		node := NodeNetStat{
			Name:       p.Name,
			TrafficIn:  s.In,
			TrafficOut: s.Out,
		}
		if validSample && elapsed > 0 {
			if prev, ok := last[p.Name]; ok {
				node.RateIn = float64(s.In-prev.In) / elapsed
				node.RateOut = float64(s.Out-prev.Out) / elapsed
				if node.RateIn < 0 {
					node.RateIn = 0
				}
				if node.RateOut < 0 {
					node.RateOut = 0
				}
			}
		}
		stats.Nodes = append(stats.Nodes, node)
		stats.TotalIn += s.In
		stats.TotalOut += s.Out
	}

	// 更新采样基线
	sample := map[string]localfwd.Stat{}
	for _, n := range stats.Nodes {
		sample[n.Name] = localfwd.Stat{Name: n.Name, In: n.TrafficIn, Out: n.TrafficOut}
	}
	a.lastSample = sample
	a.lastTime = now
	a.rateMu.Unlock()

	return stats
}
