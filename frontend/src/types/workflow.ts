export interface Workflow {
  id: number
  name: string
  description: string
  config: string
  created_by: string
  created_at: string
  updated_at: string
  stage_groups: WorkflowStageGroup[]
  variables: WorkflowVariable[]
  hooks: WorkflowHook[]
}

export interface WorkflowStageGroup {
  id?: number
  workflow_id?: number
  name: string
  description: string
  order: number
  mode: 'sequential' | 'parallel'
  stages: WorkflowStage[]
}

export interface WorkflowStage {
  id?: number
  stage_group_id?: number
  name: string
  description: string
  order: number
  machine_group_id?: number
  machine_group_name?: string
  tasks: WorkflowTask[]
}

export interface WorkflowTask {
  id?: number
  stage_id?: number
  ref: number
  name: string
  module: string
  params: string
  order: number
  when: string
  hook_ids: string
  loop: string
  timeout: number
  ignore_errors: boolean
  retries: number
  delay: number
  register: string
}

export interface WorkflowVariable {
  id?: number
  workflow_id?: number
  key: string
  type: 'string' | 'number' | 'bool'
  value?: string
  description?: string
  group?: string
}

export interface WorkflowHook {
  id?: number
  workflow_id?: number
  ref: number
  name: string
  module: string
  params: string
  host: string
  timeout: number
  when: string
  loop: string
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
  stage_group_executions: WorkflowStageGroupExecution[]
}

export interface WorkflowStageGroupExecution {
  id: number
  execution_id: number
  group_id: number
  group?: WorkflowStageGroup
  status: 'pending' | 'running' | 'success' | 'failed' | 'skipped'
  error: string
  started_at: string | null
  finished_at: string | null
  stage_executions: WorkflowStageExecution[]
}

export interface WorkflowStageExecution {
  id: number
  stage_group_execution_id: number
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
  host: string
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
  stage_groups: {
    name: string
    description: string
    order: number
    mode: 'sequential' | 'parallel'
    stages: {
      name: string
      description: string
      order: number
      machine_group_id: number
      tasks: {
        ref: number
        name: string
        module: string
        params: string
        order: number
        when: string
        hook_ids: string
        loop: string
        timeout: number
        ignore_errors: boolean
        retries: number
        delay: number
        register: string
      }[]
    }[]
  }[]
  variables: {
    key: string
    type: 'string' | 'number' | 'bool'
    value?: string
    description?: string
    group?: string
  }[]
  hooks: {
    ref: number
    name: string
    module: string
    params: string
    timeout: number
    when: string
    loop: string
    ignore_errors: boolean
    retries: number
    delay: number
    register: string
  }[]
}
