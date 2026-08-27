package main

import (
	"embed"
	"fmt"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"netfly.dev/common/tray"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

func main() {
	// 初始化应用（配置存储、日志桥、frp 服务管理器）
	app, err := NewApp()
	if err != nil {
		println("初始化失败:", err.Error())
		fmt.Println("初始化失败:", err)
		return
	}

	// 启动系统托盘：关闭窗口时隐藏到任务栏右下角
	_ = tray.Start(tray.Config{
		Icon:    trayIcon,
		Tooltip: "NetFly 服务端 - frp 内网穿透",
		OnShow:  app.showWindow,
		OnQuit:  app.quitApp,
	})

	// 创建 Wails 应用
	err = wails.Run(&options.App{
		Title:     "NetFly 服务端",
		Width:     1000,
		Height:    720,
		MinWidth:  880,
		MinHeight: 620,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:  &options.RGBA{R: 24, G: 26, B: 32, A: 1},
		HideWindowOnClose: true, // 点关闭按钮时隐藏窗口到托盘而不是退出
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
