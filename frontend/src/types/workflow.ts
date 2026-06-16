export interface Workflow {
  id: number
  name: string
  description: string
  config: string
  created_by: string
  created_at: string
  updated_at: string
  stages: WorkflowStage[]
}

export interface WorkflowStage {
  id?: number
  workflow_id?: number
  name: string
  description: string
  order: number
  retry_policy: string
  max_retries: number
  continue_on_error: boolean
  tasks: WorkflowTask[]
}

export interface WorkflowTask {
  id?: number
  stage_id?: number
  name: string
  module: string
  params: string
  host: string
  order: number
  when: string
  hooks: string
  loop: string
  timeout: number
}

export interface WorkflowExecution {
  id: number
  workflow_id: number
  status: 'running' | 'success' | 'failed' | 'paused' | 'cancelled'
  trigger: string
  error: string
  started_at: string
  finished_at: string | null
  created_at: string
  stage_executions: WorkflowStageExecution[]
}

export interface WorkflowStageExecution {
  id: number
  execution_id: number
  stage_id: number
  stage?: WorkflowStage
  status: 'pending' | 'running' | 'success' | 'failed' | 'skipped'
  error: string
  started_at: string | null
  finished_at: string | null
  task_executions: WorkflowTaskExecution[]
}

export interface WorkflowTaskExecution {
  id: number
  stage_execution_id: number
  task_id: number
  task?: WorkflowTask
  status: 'pending' | 'running' | 'success' | 'failed'
  output: string
  error: string
  duration_ms: number
  started_at: string | null
  finished_at: string | null
}

export interface CreateWorkflowRequest {
  name: string
  description: string
  config: string
  stages: {
    name: string
    description: string
    order: number
    retry_policy: string
    max_retries: number
    continue_on_error: boolean
    tasks: {
      name: string
      module: string
      params: string
      host: string
      order: number
      when: string
      hooks: string
      loop: string
      timeout: number
    }[]
  }[]
}
