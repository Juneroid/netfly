// Package model - 客户端配置模型与 frp 配置转换。
package model

import (
	"fmt"
	"strings"

	"github.com/samber/lo"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// ClientAppConfig 是客户端 GUI 的完整配置。
type ClientAppConfig struct {
	ServerAddr string      `json:"serverAddr"` // 服务端地址（IP 或域名）
	ServerPort int         `json:"serverPort"` // 服务端端口
	Token      string      `json:"token"`      // 认证口令，需与服务端一致
	User       string      `json:"user"`       // 代理名称前缀（多客户端区分）
	LogLevel   string      `json:"logLevel"`   // 日志级别
	AutoStart  bool        `json:"autoStart"`  // 程序启动时自动连接
	Proxies    []ProxyItem `json:"proxies"`    // 代理节点列表
}

// ProxyItem 是单个代理节点的 GUI 配置。
type ProxyItem struct {
	Name           string `json:"name"`           // 节点名称（需唯一）
	Type           string `json:"type"`           // tcp / udp / http / https
	LocalIP        string `json:"localIP"`        // 本地服务地址
	LocalPort      int    `json:"localPort"`      // 本地服务端口
	RemotePort     int    `json:"remotePort"`     // 服务端暴露端口（tcp/udp）
	CustomDomains  string `json:"customDomains"`  // 自定义域名（http/https，逗号分隔）
	SubDomain      string `json:"subDomain"`      // 子域名（需服务端配置 subDomainHost）
	UseEncryption  bool   `json:"useEncryption"`  // 加密传输
	UseCompression bool   `json:"useCompression"` // 压缩传输
	Enabled        bool   `json:"enabled"`        // 是否启用
}

// DefaultClientConfig 返回客户端默认配置。
func DefaultClientConfig() ClientAppConfig {
	return ClientAppConfig{
		ServerAddr: "",
		ServerPort: 7000,
		Token:      "",
		User:       "",
		LogLevel:   "info",
		AutoStart:  false,
		Proxies:    []ProxyItem{},
	}
}

// Validate 校验客户端完整配置（连接参数 + 节点）。
func (c *ClientAppConfig) Validate() error {
	if err := c.ValidateConnection(); err != nil {
		return err
	}
	return c.ValidateProxies()
}

// ValidateConnection 仅校验服务端连接参数。
// 服务未启动时允许地址为空，故仅在连接前调用。
func (c *ClientAppConfig) ValidateConnection() error {
	if strings.TrimSpace(c.ServerAddr) == "" {
		return fmt.Errorf("服务端地址不能为空")
	}
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("服务端端口必须在 1-65535 之间")
	}
	return nil
}

// ValidateProxies 仅校验节点列表（名称唯一性与单节点合法性）。
// 新建/编辑节点时调用，与连接配置无关。
func (c *ClientAppConfig) ValidateProxies() error {
	names := map[string]bool{}
	for i := range c.Proxies {
		p := &c.Proxies[i]
		if err := p.Validate(); err != nil {
			return fmt.Errorf("节点 #%d: %w", i+1, err)
		}
		if names[p.Name] {
			return fmt.Errorf("节点名称 [%s] 重复", p.Name)
		}
		names[p.Name] = true
	}
	return nil
}

// Validate 校验单个节点配置。
func (p *ProxyItem) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("名称不能为空")
	}
	if p.LocalPort <= 0 || p.LocalPort > 65535 {
		return fmt.Errorf("本地端口必须在 1-65535 之间")
	}
	switch p.Type {
	case ProxyTypeTCP, ProxyTypeUDP:
		if p.RemotePort <= 0 || p.RemotePort > 65535 {
			return fmt.Errorf("远程端口必须在 1-65535 之间")
		}
	case ProxyTypeHTTP, ProxyTypeHTTPS:
		if strings.TrimSpace(p.CustomDomains) == "" && strings.TrimSpace(p.SubDomain) == "" {
			return fmt.Errorf("自定义域名和子域名至少填写一项")
		}
	default:
		return fmt.Errorf("不支持的节点类型: %s", p.Type)
	}
	return nil
}

// BuildFRPCConfig 将 GUI 配置转换为 frp 客户端公共配置。
// LoginFailExit 固定为 false：连接失败时保持重试，由 GUI 展示状态。
func (c *ClientAppConfig) BuildFRPCConfig() (*v1.ClientCommonConfig, error) {
	if err := c.ValidateConnection(); err != nil {
		return nil, err
	}
	cfg := &v1.ClientCommonConfig{
		User:          c.User,
		ServerAddr:    c.ServerAddr,
		ServerPort:    c.ServerPort,
		LoginFailExit: lo.ToPtr(false),
		Auth: v1.AuthClientConfig{
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
	if err := cfg.Complete(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// BuildProxyConfigurers 将节点列表转换为 frp 代理配置。
// 仅返回启用状态的节点。
func (c *ClientAppConfig) BuildProxyConfigurers() []v1.ProxyConfigurer {
	out := make([]v1.ProxyConfigurer, 0, len(c.Proxies))
	for i := range c.Proxies {
		p := &c.Proxies[i]
		if !p.Enabled {
			continue
		}
		if cfg := p.buildConfigurer(); cfg != nil {
			out = append(out, cfg)
		}
	}
	return out
}

// buildConfigurer 根据节点类型构建对应的 frp 代理配置。
func (p *ProxyItem) buildConfigurer() v1.ProxyConfigurer {
	base := v1.ProxyBaseConfig{
		Name: p.Name,
		Type: p.Type,
		ProxyBackend: v1.ProxyBackend{
			LocalIP:   p.LocalIP,
			LocalPort: p.LocalPort,
		},
	}
	base.Transport.UseEncryption = p.UseEncryption
	base.Transport.UseCompression = p.UseCompression
	base.Enabled = lo.ToPtr(p.Enabled)

	switch p.Type {
	case ProxyTypeTCP:
		return &v1.TCPProxyConfig{ProxyBaseConfig: base, RemotePort: p.RemotePort}
	case ProxyTypeUDP:
		return &v1.UDPProxyConfig{ProxyBaseConfig: base, RemotePort: p.RemotePort}
	case ProxyTypeHTTP:
		cfg := &v1.HTTPProxyConfig{
			ProxyBaseConfig: base,
			DomainConfig:    p.buildDomainConfig(),
		}
		return cfg
	case ProxyTypeHTTPS:
		return &v1.HTTPSProxyConfig{
			ProxyBaseConfig: base,
			DomainConfig:    p.buildDomainConfig(),
		}
	default:
		return nil
	}
}

// buildDomainConfig 构建域名配置（http/https 类型使用）。
func (p *ProxyItem) buildDomainConfig() v1.DomainConfig {
	domains := make([]string, 0)
	for _, d := range strings.Split(p.CustomDomains, ",") {
		if d = strings.TrimSpace(d); d != "" {
			domains = append(domains, d)
		}
	}
	return v1.DomainConfig{
		CustomDomains: domains,
		SubDomain:     strings.TrimSpace(p.SubDomain),
	}
}
