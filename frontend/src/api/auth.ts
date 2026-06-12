import request from '@/utils/request'
import type { ApiResponse } from '@/utils/request'
import type { LoginParams, LoginResult, UserInfo } from './types'

/** 用户登录 */
export function loginApi(data: LoginParams): Promise<ApiResponse<LoginResult>> {
  return request.post('/auth/login', data).then((res) => res.data)
}

/** 获取当前用户信息 */
export function getUserInfoApi(): Promise<ApiResponse<UserInfo>> {
  return request.post('/auth/userinfo').then((res) => res.data)
}

/** 退出登录 */
export function logoutApi(): Promise<ApiResponse<null>> {
  return request.post('/auth/logout').then((res) => res.data)
}

/** 修改密码 */
export function changePasswordApi(data: { oldPassword: string; newPassword: string }): Promise<ApiResponse<null>> {
  return request.post('/auth/changePassword', data).then((res) => res.data)
}
