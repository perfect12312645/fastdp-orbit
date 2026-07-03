import request from '@/utils/request'
import type { ApiResponse } from '@/utils/request'
import type { LoginParams, LoginResult, UserInfo } from './types'

/** 用户登录 */
export function loginApi(data: LoginParams): Promise<ApiResponse<LoginResult>> {
  return request.post('/auth/login', data).then((res) => res.data)
}

/** 获取当前用户信息 */
export function getUserInfoApi(): Promise<ApiResponse<UserInfo>> {
  return request.get('/auth/user-info').then((res) => res.data)
}

/** 退出登录 */
export function logoutApi(): Promise<ApiResponse<null>> {
  return request.post('/auth/logout').then((res) => res.data)
}

/** 修改密码 */
export function changePasswordApi(data: { old_password: string; new_password: string }): Promise<ApiResponse<null>> {
  return request.post('/auth/change-password', data).then((res) => res.data)
}

/** 更新个人信息 */
export function updateProfileApi(data: { nickname: string; email: string }): Promise<ApiResponse<null>> {
  return request.put('/auth/profile', data).then((res) => res.data)
}
