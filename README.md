# NetFly

基于 [frp](https://github.com/fatedier/frp)（v0.71.0）打造的 Windows 图形化内网穿透工具，提供**服务端**与**客户端**两个独立的 exe 程序。

双击即用、全程窗口操作（无需手写配置文件）、可关闭到任务栏托盘后台运行。

[中文](#中文说明) | [English](#english)

---

## 中文说明

### 项目简介

NetFly 将 frp 的命令行程序封装为原生 Windows 桌面应用（Wails v2 + Vue 3 + Element Plus），目标是让不熟悉配置文件的用户也能轻松完成内网穿透部署：

- **协议 100% 兼容官方 frp**：NetFly 服务端可直接对接标准 frpc，NetFly 客户端也可直接对接任意标准 frps
- **全部操作图形化**：端口、口令、节点等均在窗口中配置，自动持久化
- **托盘常驻**：点击窗口关闭按钮仅隐藏到任务栏右下角，从托盘菜单真正退出

### 功能特性

**服务端（NetFlyServer.exe）**

| 功能 | 说明 |
|---|---|
| 服务端口 | 设置 frp 绑定端口（默认 7000） |
| 认证口令 | Token 鉴权，客户端需填一致口令 |
| 管理网页 | 可选开启 dashboard（默认 7500），浏览器查看在线客户端与流量 |
| 启停控制 | 一键开启/关闭服务，支持开机自动启动 |
| 日志显示 | 实时滚动显示运行日志，同时在本地留存 |
| 运行状态 | 显示当前连接数、代理数、端口占用情况 |

**客户端（NetFlyClient.exe）**

| 功能 | 说明 |
|---|---|
| 连接服务端 | 配置服务端地址/端口/口令，状态灯实时显示连接状态（未连接/连接中/已连接/重试中） |
| 节点管理 | 新建/编辑/删除/启停节点，支持 TCP、UDP、HTTP、HTTPS 四种类型 |
| 热更新 | 服务运行中新增/修改/删除节点**即时生效，不断线** |
| 本地流量统计 | 入站/出站实时速率、按节点累计流量，纯本地统计**不依赖服务端**，跨天持久保存 |
| 日志显示 | 实时查看连接与代理日志 |
| 启停控制 | 一键连接/断开，支持开机自动连接 |

### 使用方式

**第一步：服务端（部署在公网机器上）**

1. 双击运行 `NetFlyServer.exe`
2. 进入「设置」，填写服务端口（默认 7000）和认证口令（Token）
3. 需要网页监控时开启「管理网页」，设置端口与账号密码
4. 点击「启动服务」，状态变为运行中即可
5. 关闭窗口会最小化到托盘继续服务；从托盘菜单「退出」才会真正停止

**第二步：客户端（部署在内网机器上）**

1. 双击运行 `NetFlyClient.exe`
2. 进入「设置」，填写服务端地址、端口、Token（与服务端一致）
3. 进入「节点管理」，新建节点，例如把本机远程桌面暴露出去：
   - 类型：TCP，本地地址：`127.0.0.1`，本地端口：`3389`，远程端口：`16000`
4. 回到「运行状态」，点击「连接服务」
5. 之后从公网访问 `服务端IP:16000` 即等于访问内网机器的 3389 端口

**常用场景示例**

| 场景 | 节点类型 | 本地端口 | 远程端口 |
|---|---|---|---|
| 远程桌面 | TCP | 3389 | 任意未占用端口 |
| 内部网站 | HTTP | 80 | 需填域名（customDomains） |
| SSH 终端 | TCP | 22 | 任意未占用端口 |

### 数据与文件位置

| 内容 | 路径 |
|---|---|
| 服务端配置/日志/统计 | `%APPDATA%\NetFly\server\` |
| 客户端配置/日志/流量 | `%APPDATA%\NetFly\client\` |
| 客户端流量记录 | `%APPDATA%\NetFly\client\traffic.json`（按天累计） |

配置修改即时保存，删除目录即恢复出厂状态。

### 从源码构建

要求：Windows + 网络可用（首次会下载 Wails CLI 与前端依赖）。

```powershell
# 在项目根目录执行
.\build.ps1
```

脚本使用项目内置的 `.tools` 目录（Go SDK、frp 源码），无需自行安装 Go。产物：

- `server-app\build\bin\NetFlyServer.exe`
- `client-app\build\bin\NetFlyClient.exe`

### 常见问题

**两个程序能同时运行在一台机器上吗？**
可以。两者数据目录、端口完全隔离，本机自连（客户端填 `127.0.0.1:7000`）也是常用调试方式。注意节点的远程端口不要与服务端自身端口（7000/7500）相同。

**为什么 UDP 节点没有流量统计？**
本地流量统计基于 TCP 转发实现，UDP 节点显示 `--`，穿透功能本身不受影响。

**连接标准 frps（非 NetFly 服务端）时流量统计可用吗？**
可用。流量统计完全在客户端本地完成，与服务端类型无关。

**忘记 Token 怎么办？**
在服务端「设置」中查看或重设；客户端也填同样的值即可。

---

## English

### Overview

NetFly is a Windows GUI toolkit for NAT traversal / reverse proxy, built on top of [frp](https://github.com/fatedier/frp) (v0.71.0). It ships as two standalone executables: a **server** and a **client**.

Double-click to run, configure everything in the window (no hand-edited config files), and minimize to the system tray for background operation.

### Features

**Server (NetFlyServer.exe)**

- Bind port & token configuration (default port 7000)
- Optional dashboard web UI (default port 7500) with basic auth
- One-click start/stop, optional auto-start on launch
- Live log streaming with on-disk retention
- Runtime stats: connections, proxies, port usage

**Client (NetFlyClient.exe)**

- Connect to frps with address/port/token; live connection state indicator
- Create/edit/delete/enable/disable proxies (TCP / UDP / HTTP / HTTPS)
- Hot updates: proxy changes apply instantly while connected, no reconnect
- **Local traffic statistics**: per-proxy in/out rates and cumulative bytes, counted entirely on the client side (works with any standard frps), persisted across restarts and rolled over daily
- Live log streaming
- One-click connect/disconnect, optional auto-connect on launch

### Usage

**Server (deploy on a public machine)**

1. Run `NetFlyServer.exe`
2. Open *Settings*, set the bind port (default 7000) and token
3. Optionally enable the dashboard web UI
4. Click *Start Service*
5. Closing the window minimizes to tray; use *Exit* in the tray menu to fully stop

**Client (deploy on an internal machine)**

1. Run `NetFlyClient.exe`
2. Open *Settings*, enter the server address, port, and token
3. Open *Proxies*, create one. Example — expose Remote Desktop:
   - Type: TCP, Local: `127.0.0.1:3389`, Remote port: `16000`
4. Open *Dashboard*, click *Connect*
5. Accessing `server-ip:16000` from the public network now reaches the internal machine's port 3389

### Data Locations

| Item | Path |
|---|---|
| Server config/logs | `%APPDATA%\NetFly\server\` |
| Client config/logs/traffic | `%APPDATA%\NetFly\client\` |

### Build from Source

Requires Windows and network access (downloads Wails CLI and frontend deps on first run):

```powershell
.\build.ps1
```

The script uses the bundled `.tools` directory (Go SDK, frp source), so no Go installation is needed. Outputs:

- `server-app\build\bin\NetFlyServer.exe`
- `client-app\build\bin\NetFlyClient.exe`

### FAQ

**Can both apps run on the same machine?**
Yes — fully isolated data dirs and ports. Pointing the client to `127.0.0.1:7000` is a common way to test locally. Avoid assigning remote ports equal to the server's own ports (7000/7500).

**Why do UDP proxies show no traffic?**
Local statistics are TCP-forwarding based. UDP proxies work fine for tunneling but display `--`.

**Does traffic statistics work with a standard frps?**
Yes — statistics are collected locally in the client and do not depend on the server implementation.

### Compatibility & Credits

- Protocol layer: [frp](https://github.com/fatedier/frp) v0.71.0 (Apache-2.0)
- Desktop framework: [Wails](https://github.com/wailsapp/wails) v2
- UI: Vue 3 + Element Plus
