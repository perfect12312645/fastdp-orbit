import request from '@/utils/request'
import type {
  Workflow,
  WorkflowExecution,
  CreateWorkflowRequest,
} from '@/types/workflow'

/** 获取所有工作流 */
export function getWorkflowsApi(): Promise<Workflow[]> {
  return request.get('/workflows').then((res) => res.data.data)
}

/** 获取工作流详情 */
export function getWorkflowApi(id: number): Promise<Workflow> {
  return request.get(`/workflows/${id}`).then((res) => res.data.data)
}

/** 创建工作流 */
export function createWorkflowApi(data: CreateWorkflowRequest): Promise<Workflow> {
  return request.post('/workflows', data).then((res) => res.data.data)
}

/** 更新工作流 */
export function updateWorkflowApi(id: number, data: CreateWorkflowRequest): Promise<void> {
  return request.put(`/workflows/${id}`, data).then((res) => res.data)
}

/** 删除工作流 */
export function deleteWorkflowApi(id: number): Promise<void> {
  return request.delete(`/workflows/${id}`).then((res) => res.data)
}

/** 执行工作流 */
export function executeWorkflowApi(id: number): Promise<void> {
  return request.post(`/workflows/${id}/execute`).then((res) => res.data)
}

/** 获取执行历史 */
export function getExecutionsApi(workflowId: number): Promise<WorkflowExecution[]> {
  return request.get(`/workflows/${workflowId}/executions`).then((res) => res.data.data)
}

/** 获取执行详情 */
export function getExecutionApi(workflowId: number, executionId: number): Promise<WorkflowExecution> {
  return request.get(`/workflows/${workflowId}/executions/${executionId}`).then((res) => res.data.data)
}

/** 暂停执行 */
export function pauseExecutionApi(workflowId: number, executionId: number): Promise<void> {
  return request.post(`/workflows/${workflowId}/executions/${executionId}/pause`).then((res) => res.data)
}

/** 恢复执行 */
export function resumeExecutionApi(workflowId: number, executionId: number): Promise<void> {
  return request.post(`/workflows/${workflowId}/executions/${executionId}/resume`).then((res) => res.data)
}

/** 终止执行 */
export function cancelExecutionApi(workflowId: number, executionId: number): Promise<void> {
  return request.post(`/workflows/${workflowId}/executions/${executionId}/cancel`).then((res) => res.data)
}

/** 重试整个执行 */
export function retryExecutionApi(workflowId: number, executionId: number): Promise<void> {
  return request.post(`/workflows/${workflowId}/executions/${executionId}/retry`).then((res) => res.data)
}

/** 重试单个阶段 */
export function retryStageApi(workflowId: number, executionId: number, stageId: number): Promise<void> {
  return request.post(`/workflows/${workflowId}/executions/${executionId}/stages/${stageId}/retry`).then((res) => res.data)
}
