// Package main - NetFly 服务端应用逻辑层。
// 通过 Wails 绑定向前端暴露配置管理、服务启停、日志与统计接口。
package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"netfly.dev/common/appcfg"
	"netfly.dev/common/frpserver"
	"netfly.dev/common/logbridge"
	"netfly.dev/common/model"
)

// App 服务端应用对象，承载全部业务状态。
type App struct {
	ctx    context.Context
	store  *appcfg.Store      // 配置持久化
	bridge *logbridge.Bridge  // frp 日志桥
	mgr    *frpserver.Manager // frps 服务管理器
	cfg    model.ServerAppConfig
}

// NewApp 初始化应用：加载配置、创建日志桥与服务管理器。
func NewApp() (*App, error) {
	store, err := appcfg.NewStore("server")
	if err != nil {
		return nil, err
	}
	bridge, err := logbridge.New(store.DataDir() + "\\logs")
	if err != nil {
		return nil, err
	}
	bridge.Install("info")

	cfg := model.DefaultServerConfig()
	if err := store.Load(&cfg); err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return &App{
		store:  store,
		bridge: bridge,
		mgr:    frpserver.NewManager(),
		cfg:    cfg,
	}, nil
}

// startup Wails 启动回调：保存上下文、注册日志转发、按需自动启动服务。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// frp 日志实时推送给前端
	a.bridge.OnLine(func(line string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "log", line)
		}
	})

	// 配置了自动启动时，延迟开启服务
	if a.cfg.AutoStart {
		go func() {
			if err := a.StartService(); err != nil {
				runtime.EventsEmit(a.ctx, "log", "自动启动失败: "+err.Error())
			}
		}()
	}
}

// shutdown Wails 退出回调：停止服务并释放资源。
func (a *App) shutdown(_ context.Context) {
	a.mgr.Stop()
	a.bridge.Close()
}

// showWindow 显示主窗口（托盘菜单触发）。
func (a *App) showWindow() {
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
	}
}

// quitApp 退出整个程序（托盘菜单触发）。
func (a *App) quitApp() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// GetConfig 返回当前服务端配置。
func (a *App) GetConfig() model.ServerAppConfig {
	return a.cfg
}

// SaveConfig 校验并保存配置。服务运行中修改的部分需重启服务生效。
func (a *App) SaveConfig(cfg model.ServerAppConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := a.store.Save(cfg); err != nil {
		return err
	}
	a.cfg = cfg
	return nil
}

// StartService 按当前配置启动 frps 服务。
func (a *App) StartService() error {
	if a.mgr.IsRunning() {
		return fmt.Errorf("服务已在运行中")
	}
	// 按配置重设日志级别
	a.bridge.Install(a.cfg.LogLevel)

	frpCfg, err := a.cfg.BuildFRPSConfig()
	if err != nil {
		return err
	}
	if err := a.mgr.Start(frpCfg); err != nil {
		return err
	}
	return nil
}

// StopService 停止 frps 服务。
func (a *App) StopService() {
	a.mgr.Stop()
}

// ServiceRunning 返回服务运行状态。
func (a *App) ServiceRunning() bool {
	return a.mgr.IsRunning()
}

// GetStats 返回服务端运行统计（客户端数、连接数、本次会话流量等）。
func (a *App) GetStats() frpserver.Stats {
	return a.mgr.Stats()
}

// GetProxies 返回当前在线代理列表。
func (a *App) GetProxies() []frpserver.ProxyBrief {
	return a.mgr.Proxies()
}

// GetLogs 返回日志缓冲区内容。
func (a *App) GetLogs() []string {
	return a.bridge.Lines()
}

// OpenDashboard 在默认浏览器中打开 frps 管理网页。
func (a *App) OpenDashboard() {
	if !a.cfg.DashboardEnabled {
		return
	}
	addr := "127.0.0.1"
	if a.cfg.BindAddr != "" && a.cfg.BindAddr != "0.0.0.0" {
		addr = a.cfg.BindAddr
	}
	runtime.BrowserOpenURL(a.ctx,
		fmt.Sprintf("http://%s:%d", addr, a.cfg.DashboardPort))
}
