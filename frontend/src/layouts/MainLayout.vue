<template>
  <el-container class="main-layout">
    <!-- 左侧侧边栏 -->
    <el-aside :width="appStore.sidebarCollapsed ? '64px' : '220px'" class="sidebar">
      <div class="sidebar-logo">
        <div class="logo-icon">
          <Icon icon="mdi:orbit" :size="24" />
        </div>
        <transition name="fade">
          <span v-if="!appStore.sidebarCollapsed" class="logo-text">fastdp-orbit</span>
        </transition>
      </div>

      <el-menu
        :default-active="activeMenu"
        :collapse="appStore.sidebarCollapsed"
        :collapse-transition="false"
        router
        class="sidebar-menu"
      >
        <template v-for="item in menuItems" :key="item.path">
          <el-menu-item :index="item.path">
            <Icon :icon="item.icon" :size="20" />
            <template #title>{{ item.title }}</template>
          </el-menu-item>
        </template>
      </el-menu>

      <div class="sidebar-footer">
        <Icon icon="mdi:logout" :size="18" class="logout-icon" @click="authStore.logout()" />
      </div>
    </el-aside>

    <!-- 右侧主区域 -->
    <el-container class="main-area">
      <!-- 顶部导航栏 -->
      <el-header class="topbar" height="60px">
        <div class="topbar-left">
          <div class="collapse-btn" @click="appStore.toggleSidebar">
            <Icon :icon="appStore.sidebarCollapsed ? 'mdi:menu' : 'mdi:menu-open'" :size="20" />
          </div>
          <el-breadcrumb separator="/" class="topbar-breadcrumb">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentRoute.meta.title && currentRoute.path !== '/dashboard'">
              {{ currentRoute.meta.title }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="topbar-right">
          <!-- 全屏 -->
          <div class="topbar-action" @click="toggleFullscreen">
            <Icon :icon="isFullscreen ? 'mdi:fullscreen-exit' : 'mdi:fullscreen'" :size="18" />
          </div>

          <!-- 主题切换 -->
          <div class="topbar-action" @click="appStore.toggleTheme">
            <Icon :icon="appStore.theme === 'light' ? 'mdi:weather-night' : 'mdi:weather-sunny'" :size="18" />
          </div>

          <!-- 通知 -->
          <el-badge :value="3" :max="99" class="notification-badge">
            <div class="topbar-action">
              <Icon icon="mdi:bell-outline" :size="18" />
            </div>
          </el-badge>

          <!-- 用户信息 -->
          <el-dropdown trigger="click" @command="handleUserCommand">
            <div class="user-info">
              <div class="user-avatar">
                {{ authStore.userInfo?.nickname?.charAt(0) || 'A' }}
              </div>
              <span class="user-name">{{ authStore.userInfo?.nickname || '管理员' }}</span>
              <Icon icon="mdi:chevron-down" :size="14" />
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <Icon icon="mdi:account-outline" :size="16" /> 个人信息
                </el-dropdown-item>
                <el-dropdown-item command="settings">
                  <Icon icon="mdi:cog-outline" :size="16" /> 系统设置
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  <Icon icon="mdi:logout" :size="16" /> 退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 主内容区 -->
      <el-main class="main-content">
        <router-view v-slot="{ Component }">
          <transition name="slide-up" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()

const isFullscreen = ref(false)

const menuItems = [
  { path: '/dashboard', title: '仪表盘', icon: 'mdi:view-dashboard-outline' },
  { path: '/node', title: '节点管理', icon: 'mdi:server-network-outline' },
  { path: '/model-service', title: '大模型服务', icon: 'mdi:robot-outline' },
  { path: '/storage', title: '存储管理', icon: 'mdi:harddisk-variant-outline' },
  { path: '/cluster', title: '集群部署', icon: 'mdi:cluster-outline' },
  { path: '/workflow', title: '工作流', icon: 'mdi:play-circle-outline' },
  { path: '/stages', title: '阶段管理', icon: 'mdi:view-column-outline' },
  { path: '/global-variables', title: '全局变量', icon: 'mdi:code-json' },
  { path: '/hook-templates', title: '钩子管理', icon: 'mdi:hook' },
  { path: '/testing', title: '自动化测试', icon: 'mdi:test-tube-outline' },
  { path: '/settings', title: '系统设置', icon: 'mdi:cog-outline' },
]

const activeMenu = computed(() => route.path)
const currentRoute = computed(() => route)

function toggleFullscreen() {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen()
    isFullscreen.value = true
  } else {
    document.exitFullscreen()
    isFullscreen.value = false
  }
}

function handleUserCommand(command: string) {
  switch (command) {
    case 'profile':
    case 'settings':
      router.push('/settings')
      break
    case 'logout':
      authStore.logout()
      break
  }
}
</script>

<style scoped>
.main-layout {
  height: 100vh;
  overflow: hidden;
}

/* ============ 侧边栏 ============ */
.sidebar {
  background: linear-gradient(180deg, #0F172A 0%, #1E293B 100%);
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  position: relative;
  z-index: 20;
}

.sidebar-logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 16px;
  gap: 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
}

.logo-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--gradient-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  flex-shrink: 0;
}

.logo-text {
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  white-space: nowrap;
  letter-spacing: -0.02em;
  background: linear-gradient(90deg, #fff 0%, #94A3B8 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.sidebar-menu {
  flex: 1;
  overflow-y: auto;
  border-right: none;
  background: transparent !important;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 220px;
}

.sidebar-menu .el-menu-item {
  height: 48px;
  line-height: 48px;
  margin: 4px 8px;
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.6);
  transition: all 0.2s;
}

.sidebar-menu .el-menu-item:hover {
  background: rgba(255, 255, 255, 0.08) !important;
  color: #fff;
}

.sidebar-menu .el-menu-item.is-active {
  background: rgba(22, 93, 255, 0.2) !important;
  color: #4080FF !important;
}

.sidebar-menu .el-menu-item.is-active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  background: #4080FF;
  border-radius: 0 3px 3px 0;
}

.sidebar-footer {
  padding: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  justify-content: center;
}

.logout-icon {
  color: rgba(255, 255, 255, 0.4);
  cursor: pointer;
  transition: color 0.2s;
  padding: 6px;
  border-radius: 6px;
}

.logout-icon:hover {
  color: var(--el-color-danger);
  background: rgba(245, 63, 63, 0.1);
}

/* ============ 主区域 ============ */
.main-area {
  overflow: hidden;
  background: var(--el-bg-color-page);
}

/* ============ 顶部导航 ============ */
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
  z-index: 10;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  cursor: pointer;
  color: var(--el-text-color-regular);
  transition: all 0.2s;
}

.collapse-btn:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}

.topbar-breadcrumb {
  font-size: var(--font-size-sm);
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

.topbar-action {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  cursor: pointer;
  color: var(--el-text-color-regular);
  transition: all 0.2s;
}

.topbar-action:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}

.notification-badge {
  line-height: 1;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 10px;
  border-radius: 8px;
  transition: background-color 0.2s;
  margin-left: 8px;
}

.user-info:hover {
  background-color: var(--el-fill-color-light);
}

.user-avatar {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--gradient-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: var(--font-weight-semibold);
}

.user-name {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--el-text-color-primary);
}

/* ============ 主内容区 ============ */
.main-content {
  overflow-y: auto;
  padding: 0;
}
</style>
