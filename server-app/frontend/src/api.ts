/**
 * Wails 后端绑定封装。
 * Wails 运行时会把 Go 绑定注入 window.go.main.App，
 * 这里统一封装并附加 TypeScript 类型。
 */

/** 服务端 GUI 配置（与 Go model.ServerAppConfig 对应） */
export interface ServerAppConfig {
  bindAddr: string
  bindPort: number
  token: string
  vhostHTTPPort: number
  vhostHTTPSPort: number
  dashboardEnabled: boolean
  dashboardPort: number
  dashboardUser: string
  dashboardPassword: string
  logLevel: string
  autoStart: boolean
}

/** 服务端运行统计（与 Go frpserver.Stats 对应） */
export interface ServerStats {
  running: boolean
  clientCount: number
  proxyCount: number
  proxyTypeCounts: Record<string, number>
  curConns: number
  sessionTrafficIn: number
  sessionTrafficOut: number
}

/** 在线代理摘要（与 Go frpserver.ProxyBrief 对应） */
export interface ProxyBrief {
  name: string
  type: string
  user: string
  curConns: number
  todayTrafficIn: number
  todayTrafficOut: number
}

/** Wails 注入的全局对象结构（window.go.main.App） */
interface WailsGlobal {
  main: {
    App: Record<string, (...args: unknown[]) => Promise<unknown>>
  }
}

declare global {
  interface Window {
    go?: WailsGlobal
  }
}

/** 获取后端绑定对象 */
function bind(): Record<string, (...args: unknown[]) => Promise<unknown>> {
  if (!window.go?.main?.App) {
    throw new Error('后端服务未就绪')
  }
  return window.go.main.App
}

/** 调用后端无参方法 */
function call<T>(method: string): Promise<T> {
  return bind()[method]() as Promise<T>
}

/** 调用后端带参方法 */
function callWith<T>(method: string, arg: unknown): Promise<T> {
  return bind()[method](arg) as Promise<T>
}

/** 后端 API（服务端应用） */
export const api = {
  /** 获取当前配置 */
  GetConfig: (): Promise<ServerAppConfig> => call('GetConfig'),
  /** 保存配置 */
  SaveConfig: (cfg: ServerAppConfig): Promise<void> =>
    callWith('SaveConfig', cfg),
  /** 启动 frps 服务 */
  StartService: (): Promise<void> => call('StartService'),
  /** 停止 frps 服务 */
  StopService: (): Promise<void> => call('StopService'),
  /** 服务是否运行中 */
  ServiceRunning: (): Promise<boolean> => call('ServiceRunning'),
  /** 获取运行统计 */
  GetStats: (): Promise<ServerStats> => call('GetStats'),
  /** 获取在线代理列表 */
  GetProxies: (): Promise<ProxyBrief[]> => call('GetProxies'),
  /** 获取日志缓冲 */
  GetLogs: (): Promise<string[]> => call('GetLogs'),
  /** 打开管理网页 */
  OpenDashboard: (): Promise<void> => call('OpenDashboard'),
}

/** 字节数格式化 */
export function formatBytes(n: number): string {
  if (!n || n <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), units.length - 1)
  return (n / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 2) + ' ' + units[i]
}
