import request from '@/utils/request'
import type { ApiResponse } from '@/utils/request'
import type { GpuInfo, GpuQueryParams } from './types'

/** 获取GPU列表 */
export function getGpuListApi(params: GpuQueryParams): Promise<ApiResponse<{ list: GpuInfo[]; total: number }>> {
  return request.post('/gpu/list', params).then((res) => res.data)
}

/** 获取GPU详情 */
export function getGpuDetailApi(id: number): Promise<ApiResponse<GpuInfo>> {
  return request.post('/gpu/detail', { id }).then((res) => res.data)
}
