// Package model 定义 GUI 层使用的业务配置模型，
// 并提供向 frp v1 官方配置结构的转换，保证与标准 frps/frpc 完全兼容。
package model

import (
	"fmt"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// 支持的代理节点类型。
const (
	ProxyTypeTCP   = "tcp"
	ProxyTypeUDP   = "udp"
	ProxyTypeHTTP  = "http"
	ProxyTypeHTTPS = "https"
)

// ProxyTypes 供前端下拉选择使用的类型列表。
var ProxyTypes = []string{ProxyTypeTCP, ProxyTypeUDP, ProxyTypeHTTP, ProxyTypeHTTPS}

// ServerAppConfig 是服务端 GUI 的完整配置。
type ServerAppConfig struct {
	BindAddr          string `json:"bindAddr"`          // 监听地址，默认 0.0.0.0
	BindPort          int    `json:"bindPort"`          // 服务端通信端口
	Token             string `json:"token"`             // 认证口令
	VhostHTTPPort     int    `json:"vhostHTTPPort"`     // HTTP 穿透端口，0 表示关闭
	VhostHTTPSPort    int    `json:"vhostHTTPSPort"`    // HTTPS 穿透端口，0 表示关闭
	DashboardEnabled  bool   `json:"dashboardEnabled"`  // 是否开启管理网页
	DashboardPort     int    `json:"dashboardPort"`     // 管理网页端口
	DashboardUser     string `json:"dashboardUser"`     // 管理网页登录用户名
	DashboardPassword string `json:"dashboardPassword"` // 管理网页登录密码
	LogLevel          string `json:"logLevel"`          // 日志级别
	AutoStart         bool   `json:"autoStart"`         // 程序启动时自动开启服务
}

// DefaultServerConfig 返回服务端默认配置。
func DefaultServerConfig() ServerAppConfig {
	return ServerAppConfig{
		BindAddr:          "0.0.0.0",
		BindPort:          7000,
		Token:             "",
		VhostHTTPPort:     0,
		VhostHTTPSPort:    0,
		DashboardEnabled:  true,
		DashboardPort:     7500,
		DashboardUser:     "admin",
		DashboardPassword: "admin",
		LogLevel:          "info",
		AutoStart:         false,
	}
}

// Validate 校验服务端配置合法性。
func (c *ServerAppConfig) Validate() error {
	if c.BindPort <= 0 || c.BindPort > 65535 {
		return fmt.Errorf("服务端口必须在 1-65535 之间")
	}
	if c.VhostHTTPPort < 0 || c.VhostHTTPPort > 65535 ||
		c.VhostHTTPSPort < 0 || c.VhostHTTPSPort > 65535 {
		return fmt.Errorf("HTTP/HTTPS 穿透端口必须在 0-65535 之间")
	}
	if c.DashboardEnabled {
		if c.DashboardPort <= 0 || c.DashboardPort > 65535 {
			return fmt.Errorf("管理网页端口必须在 1-65535 之间")
		}
		if c.DashboardPort == c.BindPort {
			return fmt.Errorf("管理网页端口不能与服务端口相同")
		}
		if c.DashboardUser == "" {
			return fmt.Errorf("管理网页用户名不能为空")
		}
	}
	return nil
}

// BuildFRPSConfig 将 GUI 配置转换为 frp 服务端配置。
func (c *ServerAppConfig) BuildFRPSConfig() (*v1.ServerConfig, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	cfg := &v1.ServerConfig{
		BindAddr:       c.BindAddr,
		BindPort:       c.BindPort,
		VhostHTTPPort:  c.VhostHTTPPort,
		VhostHTTPSPort: c.VhostHTTPSPort,
		Auth: v1.AuthServerConfig{
			Method: v1.AuthMethodToken,
			Token:  c.Token,
		},
		Log: v1.LogConfig{
			To:                "console",
			Level:             c.LogLevel,
			MaxDays:           3,
			DisablePrintColor: true,
		},
	}
	if c.DashboardEnabled {
		cfg.WebServer = v1.WebServerConfig{
			Addr:     "0.0.0.0",
			Port:     c.DashboardPort,
			User:     c.DashboardUser,
			Password: c.DashboardPassword,
		}
	}
	if err := cfg.Complete(); err != nil {
		return nil, err
	}
	return cfg, nil
}
