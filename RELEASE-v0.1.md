# NetFly v0.1.0 发行说明 | Release Notes

> 发布日期：2026-08-27 | 首个公开发行版本 | First Public Release

[中文](#中文) | [English](#english)

---

## 中文

### 版本概要

NetFly v0.1.0 是首个正式发行版本。它将 frp（v0.71.0）封装为两个开箱即用的 Windows 图形化程序，实现零配置文件的内网穿透部署：双击运行、窗口内完成全部操作、关闭窗口即最小化到托盘后台常驻。

协议层与官方 frp 完全兼容：NetFly 服务端可对接标准 frpc，NetFly 客户端可对接任意标准 frps。

### 下载文件

| 文件 | 说明 | 大小 |
|---|---|---|
| `NetFlyServer.exe` | 服务端，部署在公网机器 | 约 24 MB |
| `NetFlyClient.exe` | 客户端，部署在内网机器 | 约 20 MB |

单文件绿色程序，无需安装、无运行时依赖捆绑。

### 系统要求

- Windows 10 / 11（64 位）
- WebView2 运行时（Win10/11 默认自带，缺失时系统会自动引导安装）

### 本版本功能

**服务端**

- 服务端口、认证口令（Token）可视化配置
- 管理网页（Dashboard）：可配置端口与账号密码，浏览器查看在线客户端与流量
- 一键开启/关闭服务，支持程序启动时自动开启
- 实时日志窗口，日志同时在本地文件留存
- 运行状态总览：连接数、代理数
- 托盘常驻：窗口关闭仅隐藏，托盘菜单退出才真正停止

**客户端**

- 服务端地址/端口/口令连接配置，实时连接状态指示（未连接 / 连接中 / 已连接 / 重试中）
- 节点管理：新建、编辑、删除、启停，支持 TCP / UDP / HTTP / HTTPS 四种类型
- 运行中修改节点**热更新即时生效，无需断线重连**
- 本地流量统计：各节点入站/出站实时速率、累计流量；纯客户端本地统计，对接任何 frps 均可用；按天持久化保存
- 实时日志窗口
- 托盘常驻，支持程序启动时自动连接

### 快速上手

1. 公网机器运行 `NetFlyServer.exe` -> 设置端口与 Token -> 启动服务
2. 内网机器运行 `NetFlyClient.exe` -> 设置页填入服务端地址/端口/Token
3. 节点管理新建节点（如 TCP `127.0.0.1:3389` -> 远程端口 `16000`）
4. 运行状态页点击「连接服务」-> 公网访问 `服务端IP:16000`

详细说明见项目 [README.md](README.md)。

### 已知限制

- UDP 节点不支持流量统计（界面显示 `--`），穿透功能不受影响
- 流量统计基于 TCP 本地转发，`127.0.0.1` 上会为每个 TCP 类节点开放一个临时转发端口
- 同机同时运行服务端与客户端无冲突，但节点远程端口请避开 7000/7500 等服务端自身端口
- 暂未提供开机自启（注册表/计划任务）与程序自动更新，计划在后续版本提供

### 质量验证

- 端到端集成测试：frps 启动 -> frpc 认证连接 -> TCP 穿透 -> 本地流量统计 -> 运行中热更新 -> 服务停止端口释放，全链路通过
- 专项回归：服务端连续启停端口正常释放、客户端停止后立即可重连、连续增删改节点正常

---

## English

### Overview

NetFly v0.1.0 is the first official release. It wraps frp (v0.71.0) into two standalone Windows GUI apps, bringing config-file-free NAT traversal deployment: double-click to run, configure everything in the window, and the app stays in the system tray after closing.

The protocol layer is fully compatible with official frp: the NetFly server accepts standard frpc clients, and the NetFly client works with any standard frps.

### Downloads

| File | Purpose | Size |
|---|---|---|
| `NetFlyServer.exe` | Server, deploy on a public machine | ~24 MB |
| `NetFlyClient.exe` | Client, deploy on an internal machine | ~20 MB |

Single-file portable executables - no installer, no bundled runtime.

### Requirements

- Windows 10 / 11 (64-bit)
- WebView2 runtime (preinstalled on Win10/11; auto-guided install if missing)

### What's in this Version

**Server**

- Visual configuration of bind port and auth token
- Optional dashboard web UI with configurable port and credentials
- One-click start/stop; optional auto-start on launch
- Live log view with on-disk retention
- Runtime overview: connections and proxy count
- Tray-resident: closing the window hides it; exit via tray menu

**Client**

- Server address/port/token configuration with live connection state
- Proxy management: create / edit / delete / toggle, supporting TCP / UDP / HTTP / HTTPS
- **Hot updates**: proxy changes apply instantly while connected
- **Local traffic statistics**: per-proxy in/out rates and cumulative counters, collected entirely client-side (works with any frps), persisted and rolled over daily
- Live log view; optional auto-connect on launch

### Quick Start

1. Run `NetFlyServer.exe` on the public machine -> set port & token -> start
2. Run `NetFlyClient.exe` on the internal machine -> enter server address/port/token
3. Create a proxy (e.g. TCP `127.0.0.1:3389` -> remote port `16000`)
4. Click *Connect* -> access `server-ip:16000` from the public network

See [README.md](README.md) for full documentation.

### Known Limitations

- UDP proxies do not support traffic statistics (shown as `--`); tunneling works fine
- Statistics use local TCP forwarding, which opens a temporary port on `127.0.0.1` per TCP proxy
- Running both apps on one machine is fine, but avoid remote ports that collide with the server's own ports (e.g. 7000/7500)
- OS-level auto-start on boot and auto-update are not yet provided - planned for future releases

### Quality Assurance

- End-to-end integration test: frps start -> frpc authenticated connection -> TCP tunneling -> local traffic accounting -> in-place hot updates -> clean port release on stop
- Targeted regression: repeated server restart releases ports; client stops instantly and reconnects cleanly; repeated proxy add/edit/delete verified
