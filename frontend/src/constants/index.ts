/** 全局常量定义 */

/** 节点状态枚举 */
export const NodeStatus = {
  ONLINE: 'online',
  OFFLINE: 'offline',
  MAINTENANCE: 'maintenance',
  ERROR: 'error',
} as const

export type NodeStatusType = (typeof NodeStatus)[keyof typeof NodeStatus]

/** 节点状态标签映射 */
export const NodeStatusLabel: Record<NodeStatusType, string> = {
  online: '在线',
  offline: '离线',
  maintenance: '维护中',
  error: '异常',
}

/** 节点状态对应的Element Plus标签类型 */
export const NodeStatusType: Record<NodeStatusType, 'success' | 'info' | 'warning' | 'danger'> = {
  online: 'success',
  offline: 'info',
  maintenance: 'warning',
  error: 'danger',
}

/** GPU状态枚举 */
export const GpuStatus = {
  IDLE: 'idle',
  IN_USE: 'in_use',
  ERROR: 'error',
  OFFLINE: 'offline',
} as const

export type GpuStatusType = (typeof GpuStatus)[keyof typeof GpuStatus]

/** GPU状态标签映射 */
export const GpuStatusLabel: Record<GpuStatusType, string> = {
  idle: '空闲',
  in_use: '使用中',
  error: '异常',
  offline: '离线',
}

/** GPU状态对应的Element Plus标签类型 */
export const GpuStatusType: Record<GpuStatusType, 'success' | 'info' | 'warning' | 'danger'> = {
  idle: 'success',
  in_use: 'info',
  error: 'danger',
  offline: 'warning',
}

/** 分页默认配置 */
export const PaginationConfig = {
  pageSizes: [10, 20, 50, 100],
  defaultPageSize: 20,
  defaultPage: 1,
} as const

/** 主题选项 */
export const ThemeOptions = [
  { label: '浅色模式', value: 'light' },
  { label: '深色模式', value: 'dark' },
] as const
