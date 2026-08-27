# 构建脚本：一键打包 NetFly 服务端与客户端 exe
# 用法：在项目根目录执行  .\build.ps1
# 说明：依赖 .tools 目录下的 Go SDK 与 frp 源码（含已构建的 dashboard 前端）

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot

# Go 环境变量
$env:GOROOT = "$root\.tools\go"
$env:GOPATH = "$root\.tools\gopath"
$env:PATH = "$env:GOROOT\bin;$env:GOPATH\bin;$env:PATH"
$env:GOPROXY = "https://goproxy.cn,direct"
$env:npm_config_cache = "$root\.tools\npm-cache"

# 校验环境
if (-not (Test-Path "$env:GOROOT\bin\go.exe")) {
    Write-Error "未找到 Go SDK：$env:GOROOT\bin\go.exe"
}
$wailsCmd = Get-Command wails -ErrorAction SilentlyContinue
if (-not $wailsCmd) {
    Write-Output "首次运行：安装 Wails CLI..."
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
}

# 构建服务端
Write-Output "`n========== 构建 NetFlyServer =========="
Set-Location "$root\server-app"
wails build -platform windows/amd64

# 构建客户端
Write-Output "`n========== 构建 NetFlyClient =========="
Set-Location "$root\client-app"
wails build -platform windows/amd64

Set-Location $root
Write-Output "`n========== 构建完成 =========="
Write-Output "服务端: server-app\build\bin\NetFlyServer.exe"
Write-Output "客户端: client-app\build\bin\NetFlyClient.exe"
