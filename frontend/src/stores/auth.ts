import { defineStore } from 'pinia'
import { ref } from 'vue'
import { loginApi, getUserInfoApi, logoutApi } from '@/api/auth'
import type { LoginParams, UserInfo } from '@/api/types'
import { getToken, setToken, removeToken } from '@/utils/auth'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(getToken() || '')
  const userInfo = ref<UserInfo | null>(null)
  const isLoggedIn = ref(!!getToken())

  /** 登录 */
  async function login(params: LoginParams) {
    const res = await loginApi(params)
    token.value = res.data.token
    userInfo.value = res.data.user
    setToken(res.data.token)
    isLoggedIn.value = true
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
      removeToken()
      router.push('/login')
    }
  }

  return {
    token,
    userInfo,
    isLoggedIn,
    login,
    fetchUserInfo,
    logout,
  }
})
