import request from '@/utils/request'

export interface StageTemplate {
  id: number
  name: string
  description: string
  machine_group_id: number
  tasks: string // JSON string
  version: number
  created_at: string
  updated_at: string
}

export interface StageTemplateVersion {
  id: number
  template_id: number
  version: number
  name: string
  description: string
  machine_group_id: number
  tasks: string
  change_note: string
  created_at: string
}

/** 获取所有阶段模板 */
export function getStageTemplatesApi(): Promise<StageTemplate[]> {
  return request.get('/stage-templates').then((res) => res.data.data)
}

/** 获取阶段模板详情 */
export function getStageTemplateApi(id: number): Promise<StageTemplate> {
  return request.get(`/stage-templates/${id}`).then((res) => res.data.data)
}

/** 创建阶段模板 */
export function createStageTemplateApi(data: Partial<StageTemplate>): Promise<StageTemplate> {
  return request.post('/stage-templates', data).then((res) => res.data.data)
}

/** 更新阶段模板（强制生成新版本，必须提供 changeNote） */
export function updateStageTemplateApi(id: number, data: Partial<StageTemplate> & { change_note: string }): Promise<void> {
  return request.put(`/stage-templates/${id}`, data).then((res) => res.data)
}

/** 删除阶段模板 */
export function deleteStageTemplateApi(id: number): Promise<void> {
  return request.delete(`/stage-templates/${id}`).then((res) => res.data)
}

/** 获取阶段模板版本历史 */
export function listStageTemplateVersionsApi(id: number): Promise<StageTemplateVersion[]> {
  return request.get(`/stage-templates/${id}/versions`).then((res) => res.data.data)
}

/** 回滚到指定版本 */
export function rollbackStageTemplateApi(id: number, version: number): Promise<void> {
  return request.post(`/stage-templates/${id}/rollback`, { version }).then((res) => res.data)
}
