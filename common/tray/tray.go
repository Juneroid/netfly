// Package tray 提供系统托盘（任务栏右下角）能力。
// 基于 energye/systray，在独立 goroutine 中运行消息循环，
// 与 Wails 主窗口事件循环互不干扰。
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

// Start 在后台 goroutine 中启动系统托盘。
// 窗口关闭按钮将被应用层拦截为隐藏，程序通过托盘菜单退出。
func Start(cfg Config) error {
	if len(cfg.Icon) == 0 {
		return ErrEmptyIcon
	}

	onReady := func() {
		systray.SetIcon(cfg.Icon)
		systray.SetTooltip(cfg.Tooltip)

		// 左键单击托盘图标 -> 显示主界面
		systray.SetOnClick(func(_ systray.IMenu) {
			if cfg.OnShow != nil {
				cfg.OnShow()
			}
		})

		systray.AddMenuItem("显示主界面", "显示应用程序主窗口")
		systray.AddSeparator()

		var quitOnce sync.Once
		quitItem := systray.AddMenuItem("退出", "退出程序")
		quitItem.Click(func() {
			quitOnce.Do(func() {
				if cfg.OnQuit != nil {
					cfg.OnQuit()
				}
				systray.Quit()
			})
		})
	}

	go systray.Run(onReady, func() {})
	return nil
}
