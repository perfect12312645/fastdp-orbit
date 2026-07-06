import request from '@/utils/request'
import type { ApiResponse } from '@/utils/request'
import type { DashboardStats } from './types'

/** 获取仪表盘统计数据 */
export function getDashboardStatsApi(): Promise<ApiResponse<DashboardStats>> {
  return request.post('/dashboard/stats').then((res) => res.data)
}

/** 获取最近活动列表 */
export function getRecentActivityApi(): Promise<ApiResponse<unknown[]>> {
  return request.post('/dashboard/recentActivity').then((res) => res.data)
}
