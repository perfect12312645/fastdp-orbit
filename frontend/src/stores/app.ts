import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getTheme, setTheme, applyTheme, type ThemeType } from '@/utils/theme'

export const useAppStore = defineStore('app', () => {
  /** 侧边栏是否折叠 */
  const sidebarCollapsed = ref(false)
  /** 当前主题 */
  const theme = ref<ThemeType>(getTheme())

  /** 初始化主题（从localStorage恢复） */
  function initTheme() {
    const saved = getTheme()
    theme.value = saved
    applyTheme(saved)
  }

  /** 切换主题 */
  function toggleTheme() {
    const next: ThemeType = theme.value === 'light' ? 'dark' : 'light'
    theme.value = next
    setTheme(next)
  }

  /** 切换侧边栏折叠状态 */
  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  return {
    sidebarCollapsed,
    theme,
    initTheme,
    toggleTheme,
    toggleSidebar,
  }
})
