import { defineStore } from 'pinia'
import { ref } from 'vue'
import { loginApi, getUserInfoApi, logoutApi, changePasswordApi } from '@/api/auth'
import type { LoginParams, UserInfo, ChangePasswordParams } from '@/api/types'
import { getToken, setToken, removeToken } from '@/utils/auth'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(getToken() || '')
  const userInfo = ref<UserInfo | null>(null)
  const isLoggedIn = ref(!!getToken())
  const needsPasswordChange = ref(false)

  /** 登录 - 返回是否需强制改密码 */
  async function login(params: LoginParams): Promise<boolean> {
    const res = await loginApi(params)
    token.value = res.data.token
    userInfo.value = res.data.user
    setToken(res.data.token)
    isLoggedIn.value = true

    if (res.data.user.must_change_pwd) {
      needsPasswordChange.value = true
      return true // 需要改密码
    }

    needsPasswordChange.value = false
    router.push('/')
    return false
  }

  /** 修改密码 */
  async function changePassword(params: ChangePasswordParams): Promise<void> {
    await changePasswordApi(params)
    // 改密码成功后清除标记
    needsPasswordChange.value = false
    if (userInfo.value) {
      userInfo.value.must_change_pwd = false
    }
    router.push('/')
  }

  /** 获取用户信息 */
  async function fetchUserInfo() {
    const res = await getUserInfoApi()
    userInfo.value = res.data
  }

  /** 退出登录 */
  async function logout() {
    try {
      await logoutApi()
    } catch {
      // 即使请求失败也执行本地清理
    } finally {
      token.value = ''
      userInfo.value = null
      isLoggedIn.value = false
      needsPasswordChange.value = false
      removeToken()
      router.push('/login')
    }
  }

  return {
    token,
    userInfo,
    isLoggedIn,
    needsPasswordChange,
    login,
    changePassword,
    fetchUserInfo,
    logout,
  }
})
