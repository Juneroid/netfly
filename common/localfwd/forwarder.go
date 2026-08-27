// Package localfwd 提供基于本地 TCP 转发的流量统计。
// 原理：为每个代理节点在 127.0.0.1 上开启一个转发监听端口，
// 将 frpc 代理的本地地址改写为该端口，转发器把流量双向拷贝到
// 真实本地服务的同时完成字节数统计，实现纯本地、不依赖服务端的
// 流量记录；统计结果按天持久化到磁盘。
package localfwd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const (
	dayFormat   = "2006-01-02"
	flushPeriod = 30 * time.Second // 统计落盘周期
)

// Stat 单个节点的累计流量。
type Stat struct {
	Name string `json:"name"`
	In   int64  `json:"in"`  // 入站：来自服务端方向的字节（请求流量）
	Out  int64  `json:"out"` // 出站：发往服务端方向的字节（响应流量）
}

// Item 转发器输入项（由 GUI 节点配置转换而来）。
type Item struct {
	Name       string // 节点名
	TargetIP   string // 真实本地服务地址
	TargetPort int    // 真实本地服务端口
	TCPBased   bool   // tcp/http/https 为 true；udp 无法走 TCP 转发统计
}

// persistNode 持久化的单节点流量。
type persistNode struct {
	In       int64 `json:"in"`
	Out      int64 `json:"out"`
	TodayIn  int64 `json:"todayIn"`
	TodayOut int64 `json:"todayOut"`
}

// persistData 持久化文件结构。
type persistData struct {
	Day   string                `json:"day"`
	Nodes map[string]persistNode `json:"nodes"`
}

// countWriter 在写入目标的同时累加字节数，实现边转发边统计。
type countWriter struct {
	n *atomic.Int64
	w io.Writer
}

// Write 实现 io.Writer。
func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n.Add(int64(n))
	return n, err
}

// entry 单个节点的转发器实例。
type entry struct {
	name   string
	target string // 真实本地服务 ip:port
	ln     net.Listener
	stopCh chan struct{}
	wg     sync.WaitGroup

	sessionIn  atomic.Int64 // 本次运行入站增量
	sessionOut atomic.Int64 // 本次运行出站增量

	connsMu sync.Mutex
	conns   map[net.Conn]struct{} // 活动连接（停止时统一关闭）
}

// listenPort 返回转发器监听端口。
func (e *entry) listenPort() int {
	if addr, ok := e.ln.Addr().(*net.TCPAddr); ok {
		return addr.Port
	}
	return 0
}

// trackConn 登记活动连接。
func (e *entry) trackConn(c net.Conn) {
	e.connsMu.Lock()
	e.conns[c] = struct{}{}
	e.connsMu.Unlock()
}

// untrackConn 移除活动连接。
func (e *entry) untrackConn(c net.Conn) {
	e.connsMu.Lock()
	delete(e.conns, c)
	e.connsMu.Unlock()
}

// serve 接受来自 frpc 的连接并转发到真实本地服务。
func (e *entry) serve() {
	for {
		conn, err := e.ln.Accept()
		if err != nil {
			return
		}
		e.wg.Add(1)
		go e.handle(conn)
	}
}

// handle 双向拷贝一条连接并实时统计字节。
// 任一方向结束后关闭两端，促使另一方向尽快返回，
// 避免 keep-alive 长连接导致 handle 永不退出。
func (e *entry) handle(src net.Conn) {
	defer e.wg.Done()

	dst, err := net.DialTimeout("tcp", e.target, 5*time.Second)
	if err != nil {
		_ = src.Close()
		return
	}
	e.trackConn(src)
	e.trackConn(dst)
	defer e.untrackConn(src)
	defer e.untrackConn(dst)

	done := make(chan struct{}, 2)
	go func() {
		// src -> dst：来自服务端的请求方向，计入入站
		_, _ = io.Copy(&countWriter{n: &e.sessionIn, w: dst}, src)
		done <- struct{}{}
	}()
	go func() {
		// dst -> src：本地服务的响应方向，计出入站
		_, _ = io.Copy(&countWriter{n: &e.sessionOut, w: src}, dst)
		done <- struct{}{}
	}()

	// 任一方向结束即关闭两端，让另一方向随之退出
	<-done
	_ = src.Close()
	_ = dst.Close()
	<-done
}

// stop 关闭转发器：先关监听端口，再关闭全部活动连接，
// 最后等待所有处理协程退出（连接关闭后 io.Copy 会立即返回）。
func (e *entry) stop() {
	_ = e.ln.Close()
	close(e.stopCh)

	e.connsMu.Lock()
	for c := range e.conns {
		_ = c.Close()
	}
	e.conns = map[net.Conn]struct{}{}
	e.connsMu.Unlock()

	e.wg.Wait()
}

// Manager 管理全部转发器与流量持久化，并发安全。
type Manager struct {
	mu       sync.Mutex
	entries  map[string]*entry
	base     persistData // 已持久化的累计值
	storeDir string      // 持久化目录（空则仅内存统计）
	lastSave time.Time   // 上次落盘时间
}

// NewManager 创建转发管理器并加载历史统计。
// storeDir 为空表示不持久化（仅内存）。
func NewManager(storeDir string) (*Manager, error) {
	m := &Manager{
		entries: map[string]*entry{},
		base:    persistData{Day: time.Now().Format(dayFormat), Nodes: map[string]persistNode{}},
	}
	if storeDir != "" {
		m.storeDir = storeDir
		if err := os.MkdirAll(storeDir, 0o755); err != nil {
			return nil, fmt.Errorf("创建统计目录失败: %w", err)
		}
		m.load()
	}
	return m, nil
}

// storePath 返回统计文件路径。
func (m *Manager) storePath() string {
	return filepath.Join(m.storeDir, "traffic.json")
}

// load 从磁盘加载历史统计，并处理跨天：昨日"今日"数据清零。
func (m *Manager) load() {
	data, err := os.ReadFile(m.storePath())
	if err != nil {
		return
	}
	var p persistData
	if err := json.Unmarshal(data, &p); err != nil {
		return // 文件损坏时忽略历史，重新累计
	}
	today := time.Now().Format(dayFormat)
	if p.Day != today {
		for name := range p.Nodes {
			n := p.Nodes[name]
			n.TodayIn, n.TodayOut = 0, 0
			p.Nodes[name] = n
		}
		p.Day = today
	}
	if p.Nodes == nil {
		p.Nodes = map[string]persistNode{}
	}
	m.base = p
}

// startEntry 创建并启动单个转发器。
func startEntry(it Item) (*entry, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("创建转发端口失败: %w", err)
	}
	e := &entry{
		name:   it.Name,
		target: net.JoinHostPort(it.TargetIP, fmt.Sprintf("%d", it.TargetPort)),
		ln:     ln,
		stopCh: make(chan struct{}),
		conns:  map[net.Conn]struct{}{},
	}
	go e.serve()
	return e, nil
}

// Sync 将转发器集合与节点列表对齐：
// 为 TCP 类节点创建/保留转发器，关闭已移除的，目标地址变化的重建。
func (m *Manager) Sync(items []Item) error {
	want := map[string]Item{}
	for _, it := range items {
		if it.TCPBased && it.Name != "" && it.TargetPort > 0 {
			want[it.Name] = it
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭不再需要的转发器
	for name, e := range m.entries {
		if _, ok := want[name]; !ok {
			e.stop()
			m.absorbLocked(name, e)
			delete(m.entries, name)
		}
	}

	// 创建或按需重建
	for name, it := range want {
		target := net.JoinHostPort(it.TargetIP, fmt.Sprintf("%d", it.TargetPort))
		if e, ok := m.entries[name]; ok {
			if e.target == target {
				continue // 目标未变，复用
			}
			e.stop()
			m.absorbLocked(name, e)
			delete(m.entries, name)
		}
		e, err := startEntry(it)
		if err != nil {
			return err
		}
		m.entries[name] = e
	}
	m.saveLocked(true)
	return nil
}

// absorbLocked 将转发器的本次运行增量并入累计基数。
func (m *Manager) absorbLocked(name string, e *entry) {
	n := m.base.Nodes[name]
	n.In += e.sessionIn.Load()
	n.Out += e.sessionOut.Load()
	n.TodayIn += e.sessionIn.Load()
	n.TodayOut += e.sessionOut.Load()
	m.base.Nodes[name] = n
}

// Ports 返回各节点当前的转发监听端口。
func (m *Manager) Ports() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	ports := make(map[string]int, len(m.entries))
	for name, e := range m.entries {
		ports[name] = e.listenPort()
	}
	return ports
}

// Remove 删除单个节点：关闭其转发器并清空历史统计。
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[name]; ok {
		e.stop()
		delete(m.entries, name)
	}
	delete(m.base.Nodes, name)
	m.saveLocked(true)
}

// StopAll 关闭全部转发器并落盘。
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, e := range m.entries {
		e.stop()
		m.absorbLocked(name, e)
		delete(m.entries, name)
	}
	m.saveLocked(true)
}

// Stats 返回各节点的累计流量（历史基数 + 本次运行增量）。
// 内部按周期落盘并处理跨天滚动。
func (m *Manager) Stats() map[string]Stat {
	m.mu.Lock()
	defer m.mu.Unlock()

	today := time.Now().Format(dayFormat)
	if m.base.Day != today {
		// 跨天：把昨日增量并入累计后清零"今日"
		for name, e := range m.entries {
			m.absorbLocked(name, e)
			e.sessionIn.Store(0)
			e.sessionOut.Store(0)
		}
		for name := range m.base.Nodes {
			n := m.base.Nodes[name]
			n.TodayIn, n.TodayOut = 0, 0
			m.base.Nodes[name] = n
		}
		m.base.Day = today
		m.saveLocked(true)
	} else if time.Since(m.lastSave) > flushPeriod {
		for name, e := range m.entries {
			m.absorbLocked(name, e)
			e.sessionIn.Store(0)
			e.sessionOut.Store(0)
		}
		m.saveLocked(true)
	}

	out := make(map[string]Stat, len(m.base.Nodes)+len(m.entries))
	for name, n := range m.base.Nodes {
		out[name] = Stat{Name: name, In: n.In, Out: n.Out}
	}
	for name, e := range m.entries {
		s := out[name]
		s.Name = name
		s.In += e.sessionIn.Load()
		s.Out += e.sessionOut.Load()
		out[name] = s
	}
	return out
}

// saveLocked 将当前累计值写入磁盘（须持锁调用）。
func (m *Manager) saveLocked(force bool) {
	if m.storeDir == "" {
		return
	}
	if !force && time.Since(m.lastSave) < flushPeriod {
		return
	}
	data, err := json.MarshalIndent(m.base, "", "  ")
	if err != nil {
		return
	}
	tmp := m.storePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, m.storePath())
	m.lastSave = time.Now()
}
