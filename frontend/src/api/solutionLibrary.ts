import request from '@/utils/request'

export interface SolutionLibrary {
  id: number
  name: string
  description: string
  category: string
  version: string
  author: string
  icon: string
  stage_ids: string     // JSON string like "[1,2,3]"
  variable_ids: string
  hook_ids: string
  template_ids: string
  file_ids: string
  workflow_ids: string
  stage_count: number
  variable_count: number
  hook_count: number
  template_count: number
  file_count: number
  workflow_count: number
  created_at: string
  updated_at: string
}

export interface OrbitPack {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    description: string
    category: string
    version: string
    author: string
  }
  materials?: Array<{
    name: string
    size: number
    md5: string
    download_url?: string
  }>
  global_variables?: Array<{
    key: string
    type: string
    value: string
    description: string
    group: string
  }>
  hooks?: Array<{
    name: string
    description?: string
    module: string
    params?: string
    timeout?: number
    ignore_errors?: boolean
    retries?: number
    delay?: number
  }>
  workflow_templates?: Array<{
    name: string
    description?: string
    content: string
  }>
  stages?: Array<{
    name: string
    description?: string
    machine_group?: string
    tasks: Array<{
      ref: number
      name: string
      module: string
      order: number
      params?: string
      when?: string
          hooks?: string
      loop?: string
      timeout?: number
      ignore_errors?: boolean
      retries?: number
      delay?: number
      register?: string
    }>
  }>
  workflows?: Array<{
    name: string
    description?: string
    stage_groups?: Array<{
      name: string
      description?: string
      order: number
      mode: string
      stages?: Array<{
        name: string
        description?: string
        order: number
        machine_group?: string
        tasks?: Array<{
          ref: number
          name: string
          module: string
          order: number
          params?: string
          when?: string
          hooks?: string
          loop?: string
          timeout?: number
          ignore_errors?: boolean
          retries?: number
          delay?: number
          register?: string
        }>
      }>
    }>
    hooks?: Array<{
      name: string
      module: string
      params?: string
      timeout?: number
      ignore_errors?: boolean
      retries?: number
      delay?: number
    }>
  }>
}

/** 获取方案列表 */
export function getSolutionLibrariesApi(category?: string): Promise<SolutionLibrary[]> {
  return request.get('/solution-libraries', { params: { category } }).then((res) => res.data.data)
}

/** 获取方案详情 */
export function getSolutionLibraryApi(id: number): Promise<SolutionLibrary> {
  return request.get(`/solution-libraries/${id}`).then((res) => res.data.data)
}

/** 创建方案 */
export function createSolutionLibraryApi(data: any): Promise<SolutionLibrary> {
  return request.post('/solution-libraries', data).then((res) => res.data.data)
}

/** 更新方案 */
export function updateSolutionLibraryApi(id: number, data: any): Promise<void> {
  return request.put(`/solution-libraries/${id}`, data).then((res) => res.data)
}

/** 删除方案 */
export function deleteSolutionLibraryApi(id: number): Promise<void> {
  return request.delete(`/solution-libraries/${id}`).then((res) => res.data)
}

/** 导出方案 */
export function exportSolutionLibraryApi(id: number): Promise<OrbitPack> {
  return request.get(`/solution-libraries/${id}/export`).then((res) => res.data.data)
}

// 冲突项
export interface ConflictItem {
  type: string
  name: string
  existing_source: string
}

// 冲突检测响应
export interface ConflictResponse {
  has_conflicts: boolean
  conflicts: ConflictItem[]
  summary: {
    stage_count: number
    variable_count: number
    hook_count: number
    template_count: number
    file_count: number
    workflow_count: number
  }
}

// 导入方案（简单导入，只检查方案名称）
export function importSolutionLibraryApi(data: {
  pack: OrbitPack
}): Promise<SolutionLibrary> {
  return request.post('/solution-libraries/import', data).then((res) => res.data.data)
}

// 应用方案（检测冲突 + 根据决策执行）
export function applySolutionLibraryApi(
  id: number,
  decisions?: Record<string, Record<string, string>>
): Promise<ConflictResponse | void> {
  return request.post(`/solution-libraries/${id}/apply`, { decisions }).then((res) => res.data.data)
}
