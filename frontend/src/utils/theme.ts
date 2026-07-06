const THEME_KEY = 'theme'

export type ThemeType = 'light' | 'dark'

/** 获取当前主题 */
export function getTheme(): ThemeType {
  const theme = localStorage.getItem(THEME_KEY) as ThemeType | null
  return theme || 'light'
}

/** 设置主题 */
export function setTheme(theme: ThemeType): void {
  localStorage.setItem(THEME_KEY, theme)
  applyTheme(theme)
}

/** 应用主题到DOM */
export function applyTheme(theme: ThemeType): void {
  const html = document.documentElement
  if (theme === 'dark') {
    html.classList.add('dark')
  } else {
    html.classList.remove('dark')
  }
}

/** 切换主题 */
export function toggleTheme(): ThemeType {
  const current = getTheme()
  const next: ThemeType = current === 'light' ? 'dark' : 'light'
  setTheme(next)
  return next
}
