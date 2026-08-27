/**
 * Wails 后端绑定封装（客户端应用）。
 * Wails 运行时会把 Go 绑定注入 window.go.main.App，
 * 这里统一封装并附加 TypeScript 类型。
 */

/** 客户端 GUI 配置（与 Go model.ClientAppConfig 对应） */
export interface ClientAppConfig {
  serverAddr: string
  serverPort: number
  token: string
  user: string
  logLevel: string
  autoStart: boolean
  proxies: ProxyItem[]
}

/** 代理节点配置（与 Go model.ProxyItem 对应） */
export interface ProxyItem {
  name: string
  type: 'tcp' | 'udp' | 'http' | 'https'
  localIP: string
  localPort: number
  remotePort: number
  customDomains: string
  subDomain: string
  useEncryption: boolean
  useCompression: boolean
  enabled: boolean
}

/** 节点运行状态（与 Go frpclient.ProxyState 对应） */
export interface ProxyState {
  name: string
  type: string
  status: string
  err: string
  remoteAddr: string
}

/** 节点网络统计（与 Go NodeNetStat 对应） */
export interface NodeNetStat {
  name: string
  trafficIn: number
  trafficOut: number
  rateIn: number
  rateOut: number
}

/** 网络统计汇总（与 Go NetworkStats 对应，本地统计） */
export interface NetworkStats {
  connState: string
  nodes: NodeNetStat[]
  totalIn: number
  totalOut: number
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
function callWith<T>(method: string, ...args: unknown[]): Promise<T> {
  return (bind()[method](...args) as Promise<T>)
}

/** 后端 API（客户端应用） */
export const api = {
  /** 获取当前配置 */
  GetConfig: (): Promise<ClientAppConfig> => call('GetConfig'),
  /** 保存配置 */
  SaveConfig: (cfg: ClientAppConfig): Promise<void> =>
    callWith('SaveConfig', cfg),
  /** 连接服务端并启动服务 */
  StartService: (): Promise<void> => call('StartService'),
  /** 断开连接 */
  StopService: (): Promise<void> => call('StopService'),
  /** 服务是否运行中 */
  ServiceRunning: (): Promise<boolean> => call('ServiceRunning'),
  /** 获取连接状态 */
  GetConnState: (): Promise<string> => call('GetConnState'),
  /** 获取全部节点运行状态 */
  GetProxyStates: (): Promise<ProxyState[]> => call('GetProxyStates'),
  /** 新增节点 */
  AddProxy: (item: ProxyItem): Promise<void> => callWith('AddProxy', item),
  /** 更新节点 */
  UpdateProxy: (name: string, item: ProxyItem): Promise<void> =>
    callWith('UpdateProxy', name, item),
  /** 删除节点 */
  DeleteProxy: (name: string): Promise<void> => callWith('DeleteProxy', name),
  /** 启停节点 */
  ToggleProxy: (name: string): Promise<void> => callWith('ToggleProxy', name),
  /** 获取日志缓冲 */
  GetLogs: (): Promise<string[]> => call('GetLogs'),
  /** 获取网络统计（本地流量与速率） */
  GetNetworkStats: (): Promise<NetworkStats> => call('GetNetworkStats'),
}

/** 连接状态文案 */
export const connStateText: Record<string, string> = {
  idle: '未连接',
  connecting: '连接中...',
  online: '已连接',
  offline: '连接断开，重试中',
}

/** 字节数格式化 */
export function formatBytes(n: number): string {
  if (!n || n <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), units.length - 1)
  return (n / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 2) + ' ' + units[i]
}

/** 速率格式化（B/s） */
export function formatRate(n: number): string {
  return formatBytes(Math.max(0, n)) + '/s'
}
