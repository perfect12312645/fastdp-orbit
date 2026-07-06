/** API类型定义 */

/** 通用分页请求参数 */
export interface PageParams {
  page: number
  pageSize: number
}

/** 通用分页响应 */
export interface PageResult<T> {
  list: T[]
  total: number
}

/** 登录请求 */
export interface LoginParams {
  username: string
  password: string
}

/** 登录响应 */
export interface LoginResult {
  token: string
  user: UserInfo
}

/** 修改密码请求 */
export interface ChangePasswordParams {
  old_password: string
  new_password: string
}

/** 用户信息 */
export interface UserInfo {
  id: number
  username: string
  nickname: string
  avatar: string
  role: string
  email: string
  must_change_pwd: boolean
  last_login_at: string | null
}

/** 节点信息 */
export interface NodeInfo {
  id: number
  name: string
  ip: string
  os: string
  cpuCores: number
  memoryMb: number
  diskGb: number
  status: string
  gpuCount: number
  clusterName: string
  createdAt: string
  updatedAt: string
}

/** 节点查询参数 */
export interface NodeQueryParams extends PageParams {
  name?: string
  ip?: string
  status?: string
}

/** 节点创建/更新参数 */
export interface NodeFormData {
  name: string
  ip: string
  os: string
  cpuCores: number
  memoryMb: number
  diskGb: number
  clusterName: string
}

/** GPU信息 */
export interface GpuInfo {
  id: number
  nodeId: number
  nodeName: string
  model: string
  uuid: string
  memoryMb: number
  usedMemoryMb: number
  temperature: number
  utilization: number
  status: string
  driverVersion: string
  cudaVersion: string
}

/** GPU查询参数 */
export interface GpuQueryParams extends PageParams {
  nodeName?: string
  model?: string
  status?: string
}

/** 大模型服务信息 */
export interface ModelServiceInfo {
  id: number
  name: string
  modelName: string
  endpoint: string
  status: string
  gpuCount: number
  replicas: number
  createdAt: string
  updatedAt: string
}

/** 仪表盘统计数据 */
export interface DashboardStats {
  totalNodes: number
  onlineNodes: number
  totalGpus: number
  activeGpus: number
  totalServices: number
  runningServices: number
}
