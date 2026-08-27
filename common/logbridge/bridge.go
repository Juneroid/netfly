// Package logbridge 将 frp 内部日志桥接到 GUI 界面与本地文件。
// frp 使用 fatedier/golib/log 全局 Logger，这里通过 WithOutput
// 把日志重定向到自定义 Writer，再分发给：内存环形缓冲（GUI 展示）、
// 磁盘日志文件（持久化）、关键字订阅者（用于判断连接状态等）。
package logbridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	golog "github.com/fatedier/golib/log"

	frplog "github.com/fatedier/frp/pkg/util/log"
)

const (
	maxLines      = 3000 // 环形缓冲最大行数
	maxBufferLen  = 1 << 20
	logFileFormat = "2006-01-02"
)

// Bridge 是日志桥接器，实现 io.Writer 接收 frp 日志输出。
type Bridge struct {
	mu       sync.Mutex
	lines    []string       // 环形缓冲
	pos      int            // 下一个写入位置
	subs     []func(string) // 行订阅回调
	kwMu     sync.RWMutex
	keywords map[string][]func() // 关键字订阅（关键字 -> 回调列表）
	logFile  *os.File            // 磁盘日志文件句柄
	logDay   string              // 当前日志文件日期（按天滚动）
}

// New 创建日志桥接器，并可选开启磁盘日志输出。
// logDir 为空表示不写磁盘日志。
func New(logDir string) (*Bridge, error) {
	b := &Bridge{keywords: make(map[string][]func())}
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %w", err)
		}
		b.lines = make([]string, 0, maxLines)
		if err := b.openLogFile(logDir); err != nil {
			return nil, err
		}
	}
	if b.lines == nil {
		b.lines = make([]string, 0, maxLines)
	}
	return b, nil
}

// openLogFile 打开（或按天滚动）磁盘日志文件。
func (b *Bridge) openLogFile(logDir string) error {
	day := time.Now().Format(logFileFormat)
	if b.logFile != nil && day == b.logDay {
		return nil
	}
	if b.logFile != nil {
		_ = b.logFile.Close()
	}
	f, err := os.OpenFile(filepath.Join(logDir, day+".log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	b.logFile = f
	b.logDay = day
	return nil
}

// Install 将本桥接器安装为 frp 全局 Logger 的输出目标。
// level 为日志级别：trace/debug/info/warn/error。
func (b *Bridge) Install(level string) {
	// 先用 frp 官方初始化函数设置级别，再覆盖输出目标，
	// 保证与 frp 内部日志行为（级别过滤等）完全一致。
	frplog.InitLogger("console", level, 3, true)
	consoleWriter := golog.NewConsoleWriter(golog.ConsoleConfig{Colorful: false}, b)
	frplog.Logger = frplog.Logger.WithOptions(golog.WithOutput(consoleWriter))
}

// Write 实现 io.Writer：按行切分日志并分发。
func (b *Bridge) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	n = len(p)
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		b.dispatch(line)
	}
	return n, nil
}

// dispatch 单行日志的存储与分发，同时处理日志文件按天滚动。
func (b *Bridge) dispatch(line string) {
	b.mu.Lock()
	// 环形缓冲写入
	if len(b.lines) < maxLines {
		b.lines = append(b.lines, line)
	} else {
		b.lines[b.pos] = line
		b.pos = (b.pos + 1) % maxLines
	}
	// 磁盘日志（带时间戳）
	if b.logFile != nil {
		day := time.Now().Format(logFileFormat)
		if day != b.logDay {
			dir := filepath.Dir(b.logFile.Name())
			_ = b.openLogFile(dir)
		}
		if b.logFile != nil {
			_, _ = b.logFile.WriteString(
				time.Now().Format("15:04:05.000") + " " + line + "\n")
		}
	}
	subs := make([]func(string), len(b.subs))
	copy(subs, b.subs)
	b.mu.Unlock()

	for _, fn := range subs {
		fn(line)
	}

	// 关键字分发
	b.kwMu.RLock()
	var hits []func()
	for kw, fns := range b.keywords {
		if strings.Contains(line, kw) {
			hits = append(hits, fns...)
		}
	}
	b.kwMu.RUnlock()
	for _, fn := range hits {
		fn()
	}
}

// Lines 返回缓冲区内最近的日志行（时间正序）。
func (b *Bridge) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.lines) < maxLines {
		out := make([]string, len(b.lines))
		copy(out, b.lines)
		return out
	}
	out := make([]string, 0, maxLines)
	out = append(out, b.lines[b.pos:]...)
	out = append(out, b.lines[:b.pos]...)
	return out
}

// OnLine 注册行订阅回调，每产生一行日志调用一次。
func (b *Bridge) OnLine(fn func(line string)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = append(b.subs, fn)
}

// OnKeyword 注册关键字订阅：日志行包含 keyword 时触发 fn。
// 用于从 frp 日志中提取连接状态变化等事件。
func (b *Bridge) OnKeyword(keyword string, fn func()) {
	b.kwMu.Lock()
	defer b.kwMu.Unlock()
	b.keywords[keyword] = append(b.keywords[keyword], fn)
}

// Close 关闭磁盘日志文件。
func (b *Bridge) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.logFile != nil {
		_ = b.logFile.Close()
		b.logFile = nil
	}
}
