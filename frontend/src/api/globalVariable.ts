import request from '@/utils/request'

export interface GlobalVariable {
  id: number
  key: string
  type: string
  value: string
  description: string
  group: string
  source: string
  created_at: string
  updated_at: string
}

let cachedPromise: Promise<GlobalVariable[]> | null = null

/** 获取所有全局变量（页面内缓存，避免重复请求） */
export function getGlobalVariablesApi(): Promise<GlobalVariable[]> {
  if (!cachedPromise) {
    cachedPromise = request.get('/global-variables').then((res) => {
      const data = res.data.data as GlobalVariable[]
      // 30 秒后清缓存
      setTimeout(() => { cachedPromise = null }, 30000)
      return data
    }).catch((err) => {
      cachedPromise = null
      throw err
    })
  }
  return cachedPromise
}

/** 获取全局变量详情 */
export function getGlobalVariableApi(id: number): Promise<GlobalVariable> {
  return request.get(`/global-variables/${id}`).then((res) => res.data.data)
}

/** 创建全局变量 */
export function createGlobalVariableApi(data: Partial<GlobalVariable>): Promise<GlobalVariable> {
  return request.post('/global-variables', data).then((res) => res.data.data)
}

/** 更新全局变量 */
export function updateGlobalVariableApi(id: number, data: Partial<GlobalVariable>): Promise<void> {
  return request.put(`/global-variables/${id}`, data).then((res) => res.data)
}

/** 删除全局变量 */
export function deleteGlobalVariableApi(id: number): Promise<void> {
  return request.delete(`/global-variables/${id}`).then((res) => res.data)
}
