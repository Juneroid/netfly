// Package appcfg 提供应用程序配置的持久化能力。
// 配置以 JSON 格式存储在用户数据目录（%APPDATA%\NetFly）下，
// GUI 中的所有操作最终都会写入这里，避免用户手工编辑配置文件。
package appcfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// dataDir 计算应用数据目录：优先用户配置目录，
// 不可用时（如受限环境）回退到临时目录。
func dataDir(name string) (string, error) {
	if base, err := os.UserConfigDir(); err == nil {
		dir := filepath.Join(base, "NetFly", name)
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return dir, nil
		}
	}
	dir := filepath.Join(os.TempDir(), "NetFly", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}
	return dir, nil
}

// Store 负责单个应用配置文件的加载与保存，并发安全。
type Store struct {
	mu   sync.Mutex
	path string
}

// NewStore 创建配置存储器。
// name 用于区分不同应用（如 "server"、"client"），对应不同子目录与文件名。
func NewStore(name string) (*Store, error) {
	dir, err := dataDir(name)
	if err != nil {
		return nil, err
	}
	return &Store{path: filepath.Join(dir, "config.json")}, nil
}

// DataDir 返回该应用的数据目录（日志等文件也放在这里）。
func (s *Store) DataDir() string {
	return filepath.Dir(s.path)
}

// Load 将配置文件内容反序列化到 v 指向的结构体。
// 配置文件不存在时保持 v 原值（调用方可预填默认值）。
func (s *Store) Load(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 首次运行，无配置文件
		}
		return fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}
	return nil
}

// Save 将 v 序列化为带缩进的 JSON 并原子写入配置文件。
// 先写临时文件再重命名，避免写入中断导致配置损坏。
func (s *Store) Save(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入配置临时文件失败: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("保存配置文件失败: %w", err)
	}
	return nil
}
