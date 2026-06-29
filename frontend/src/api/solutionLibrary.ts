import request from '@/utils/request'

export interface SolutionLibrary {
  id: number
  name: string
  description: string
  category: string
  version: string
  author: string
  icon: string
  stage_count: number
  variable_count: number
  hook_count: number
  template_count: number
  material_count: number
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
      hook_ids?: string
      loop?: string
      timeout?: number
      ignore_errors?: boolean
      retries?: number
      delay?: number
      register?: string
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
export function createSolutionLibraryApi(data: Partial<SolutionLibrary>): Promise<SolutionLibrary> {
  return request.post('/solution-libraries', data).then((res) => res.data.data)
}

/** 删除方案 */
export function deleteSolutionLibraryApi(id: number): Promise<void> {
  return request.delete(`/solution-libraries/${id}`).then((res) => res.data)
}

/** 导出方案 */
export function exportSolutionLibraryApi(id: number): Promise<OrbitPack> {
  return request.get(`/solution-libraries/${id}/export`).then((res) => res.data.data)
}

/** 导入方案 */
export function importSolutionLibraryApi(pack: OrbitPack): Promise<SolutionLibrary> {
  return request.post('/solution-libraries/import', pack).then((res) => res.data.data)
}
