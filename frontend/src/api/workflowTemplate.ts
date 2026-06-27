import request from '@/utils/request'

export interface WorkflowTemplate {
  id: number
  name: string
  description: string
  content: string
  variables: string
  created_at: string
  updated_at: string
}

/** 获取所有工作流模板文件 */
export function getWorkflowTemplatesApi(): Promise<WorkflowTemplate[]> {
  return request.get('/workflow-templates').then((res) => res.data.data)
}

/** 获取工作流模板文件详情 */
export function getWorkflowTemplateApi(id: number): Promise<WorkflowTemplate> {
  return request.get(`/workflow-templates/${id}`).then((res) => res.data.data)
}

/** 创建工作流模板文件 */
export function createWorkflowTemplateApi(data: Partial<WorkflowTemplate>): Promise<WorkflowTemplate> {
  return request.post('/workflow-templates', data).then((res) => res.data.data)
}

/** 更新工作流模板文件 */
export function updateWorkflowTemplateApi(id: number, data: Partial<WorkflowTemplate>): Promise<void> {
  return request.put(`/workflow-templates/${id}`, data).then((res) => res.data)
}

/** 删除工作流模板文件 */
export function deleteWorkflowTemplateApi(id: number): Promise<void> {
  return request.delete(`/workflow-templates/${id}`).then((res) => res.data)
}
