// Package tray 提供系统托盘（任务栏右下角）能力。
// 基于 energye/systray 运行托盘消息循环。
//
// 关键设计：所有用户回调（点击图标、点击菜单）一律通过 goroutine
// 异步派发，绝不阻塞托盘自身的 Windows 消息循环。否则长时间运行后
// 跨线程的窗口操作可能卡住回调，导致"进程还在、图标还在，但点击
// 托盘没有任何反应"。
package tray

import (
	"errors"
	"sync"

	"github.com/energye/systray"
)

// ErrEmptyIcon 表示未提供托盘图标。
var ErrEmptyIcon = errors.New("托盘图标不能为空")

// Config 托盘配置。
type Config struct {
	Icon    []byte // Windows 下为 .ico 文件字节
	Tooltip string // 鼠标悬停提示
	OnShow  func() // 显示主界面（菜单点击或图标左键单击）
	OnQuit  func() // 退出程序
}

// dispatch 在独立 goroutine 中执行回调，
// 保证 systray 的消息分发线程立即返回、永不阻塞。
func dispatch(fn func()) {
	if fn == nil {
		return
	}
	go fn()
}

// Start 在后台 goroutine 中启动系统托盘。
// 窗口关闭按钮将被应用层拦截为隐藏，程序通过托盘菜单退出。
func Start(cfg Config) error {
	if len(cfg.Icon) == 0 {
		return ErrEmptyIcon
	}

	onReady := func() {
		systray.SetIcon(cfg.Icon)
		systray.SetTooltip(cfg.Tooltip)

		// 左键单击托盘图标 -> 显示主界面（异步，避免阻塞消息循环）
		systray.SetOnClick(func(_ systray.IMenu) {
			dispatch(cfg.OnShow)
		})
		// 双击托盘图标同样显示主界面
		systray.SetOnDClick(func(_ systray.IMenu) {
			dispatch(cfg.OnShow)
		})

		showItem := systray.AddMenuItem("显示主界面", "显示应用程序主窗口")
		showItem.Click(func() {
			dispatch(cfg.OnShow)
		})
		systray.AddSeparator()

		var quitOnce sync.Once
		quitItem := systray.AddMenuItem("退出", "退出程序")
		quitItem.Click(func() {
			quitOnce.Do(func() {
				dispatch(cfg.OnQuit)
				systray.Quit()
			})
		})
	}

	go systray.Run(onReady, func() {})
	return nil
}
