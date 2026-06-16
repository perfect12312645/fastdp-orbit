import request from '@/utils/request'

export interface DiskInfo {
  device: string
  type: string
  total_gb: number
}

export interface NetworkInfo {
  name: string
  ip: string
  mac: string
  speed: number
  status: string
}

export interface GPUInfo {
  name: string
  count: number
  driver_version: string
}

export interface MachineInfo {
  ip: string
  port: number
  status: string
  hostname: string
  virtualization: string
  uptime_seconds: number
  system_time: string
  hardware_time: string
  os_name: string
  os_version: string
  kernel: string
  arch: string
  cpu_model: string
  cpu_cores: number
  memory_kb: number
  swap_kb: number
  gateway: string
  firewall_status: string
  firewall_enabled: string
  timezone: string
  disks: DiskInfo[]
  networks: NetworkInfo[]
  gpus: GPUInfo[] | null
}

/** 获取所有机器信息（从缓存读取） */
export function getMachinesApi(): Promise<MachineInfo[]> {
  return request.get('/machines').then((res) => res.data.data)
}

/** 同步硬件信息（触发gRPC刷新） */
export function syncHardwareApi(): Promise<MachineInfo[]> {
  return request.get('/machines/sync-hardware').then((res) => res.data.data)
}

/** 删除机器记录 */
export function deleteMachineApi(ip: string, port: number): Promise<string> {
  return request.delete(`/machines/${ip}/${port}`).then((res) => res.data.data)
}

/** 远程执行命令 */
export function execOnMachineApi(ip: string, port: number, command: string): Promise<string> {
  return request.post(`/machines/${ip}/${port}/exec`, { command }).then((res) => res.data.data)
}
