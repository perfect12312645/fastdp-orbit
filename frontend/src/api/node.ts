import request from '@/utils/request'
import type { ApiResponse } from '@/utils/request'
import type { NodeInfo, NodeQueryParams, NodeFormData } from './types'

/** 获取节点列表 */
export function getNodeListApi(params: NodeQueryParams): Promise<ApiResponse<{ list: NodeInfo[]; total: number }>> {
  return request.post('/node/list', params).then((res) => res.data)
}

/** 获取节点详情 */
export function getNodeDetailApi(id: number): Promise<ApiResponse<NodeInfo>> {
  return request.post('/node/detail', { id }).then((res) => res.data)
}

/** 创建节点 */
export function createNodeApi(data: NodeFormData): Promise<ApiResponse<NodeInfo>> {
  return request.post('/node/create', data).then((res) => res.data)
}

/** 更新节点 */
export function updateNodeApi(data: NodeFormData & { id: number }): Promise<ApiResponse<NodeInfo>> {
  return request.post('/node/update', data).then((res) => res.data)
}

/** 删除节点 */
export function deleteNodeApi(id: number): Promise<ApiResponse<null>> {
  return request.post('/node/delete', { id }).then((res) => res.data)
}

/** 批量删除节点 */
export function batchDeleteNodeApi(ids: number[]): Promise<ApiResponse<null>> {
  return request.post('/node/batchDelete', { ids }).then((res) => res.data)
}
