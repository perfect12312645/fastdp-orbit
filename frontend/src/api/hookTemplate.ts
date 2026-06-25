import request from '@/utils/request'

export interface HookTemplate {
  id: number
  name: string
  description: string
  module: string
  params: string
  timeout: number
  ignore_errors: boolean
  retries: number
  delay: number
  created_at: string
  updated_at: string
}

/** 获取所有钩子模板 */
export function getHookTemplatesApi(): Promise<HookTemplate[]> {
  return request.get('/hook-templates').then((res) => res.data.data)
}

/** 获取钩子模板详情 */
export function getHookTemplateApi(id: number): Promise<HookTemplate> {
  return request.get(`/hook-templates/${id}`).then((res) => res.data.data)
}

/** 创建钩子模板 */
export function createHookTemplateApi(data: Partial<HookTemplate>): Promise<HookTemplate> {
  return request.post('/hook-templates', data).then((res) => res.data.data)
}

/** 更新钩子模板 */
export function updateHookTemplateApi(id: number, data: Partial<HookTemplate>): Promise<void> {
  return request.put(`/hook-templates/${id}`, data).then((res) => res.data)
}

/** 删除钩子模板 */
export function deleteHookTemplateApi(id: number): Promise<void> {
  return request.delete(`/hook-templates/${id}`).then((res) => res.data)
}
