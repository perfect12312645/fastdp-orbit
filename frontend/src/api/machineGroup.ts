import request from '@/utils/request'
import type { MachineInfo } from '@/api/machine'

export interface MachineGroup {
  id: number
  name: string
  description: string
  machines: MachineInfo[]
}

/** 获取所有机器分组 */
export function getMachineGroupsApi(): Promise<MachineGroup[]> {
  return request.get('/machine-groups').then((res) => res.data.data)
}

/** 获取机器分组详情 */
export function getMachineGroupApi(id: number): Promise<MachineGroup> {
  return request.get(`/machine-groups/${id}`).then((res) => res.data.data)
}

/** 创建机器分组 */
export function createMachineGroupApi(data: { name: string; description: string; machine_ids: number[] }): Promise<MachineGroup> {
  return request.post('/machine-groups', data).then((res) => res.data.data)
}

/** 更新机器分组 */
export function updateMachineGroupApi(id: number, data: { name: string; description: string; machine_ids: number[] }): Promise<MachineGroup> {
  return request.put(`/machine-groups/${id}`, data).then((res) => res.data.data)
}

/** 删除机器分组 */
export function deleteMachineGroupApi(id: number): Promise<void> {
  return request.delete(`/machine-groups/${id}`).then((res) => res.data)
}
