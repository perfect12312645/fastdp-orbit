const TOKEN_KEY = 'token'

/** 获取token */
export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

/** 存储token */
export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

/** 移除token */
export function removeToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

/** 判断是否已登录（临时跳过登录验证） */
export function isLoggedIn(): boolean {
  // TODO: 后端登录接口实现后恢复验证
  return true
}
