<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>系统设置</h2>
        <p class="page-subtitle">管理个人信息和系统偏好</p>
      </div>
    </div>

    <div class="page-content settings-content">
      <el-tabs v-model="activeTab" class="settings-tabs">
        <el-tab-pane name="profile">
          <template #label>
            <span class="tab-label">
              <Icon icon="mdi:account-outline" :size="16" /> 个人信息
            </span>
          </template>
          <div class="settings-form-wrapper">
            <div class="form-section">
              <h3 class="section-title">基本信息</h3>
              <p class="section-desc">更新您的个人资料和联系信息</p>
            </div>
            <el-form :model="profileForm" label-width="100px" class="settings-form">
              <el-form-item label="用户名">
                <el-input v-model="profileForm.username" disabled>
                  <template #prefix><Icon icon="mdi:account-outline" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item label="昵称">
                <el-input v-model="profileForm.nickname" placeholder="请输入昵称">
                  <template #prefix><Icon icon="mdi:badge-account-outline" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item label="邮箱">
                <el-input v-model="profileForm.email" placeholder="请输入邮箱">
                  <template #prefix><Icon icon="mdi:email-outline" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item>
                <el-button type="primary">
                  <Icon icon="mdi:content-save-outline" :size="16" /> 保存修改
                </el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <el-tab-pane name="password">
          <template #label>
            <span class="tab-label">
              <Icon icon="mdi:lock-outline" :size="16" /> 修改密码
            </span>
          </template>
          <div class="settings-form-wrapper">
            <div class="form-section">
              <h3 class="section-title">安全设置</h3>
              <p class="section-desc">定期更换密码以保障账号安全</p>
            </div>
            <el-form :model="passwordForm" label-width="100px" class="settings-form">
              <el-form-item label="当前密码">
                <el-input v-model="passwordForm.oldPassword" type="password" show-password placeholder="请输入当前密码">
                  <template #prefix><Icon icon="mdi:key-outline" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item label="新密码">
                <el-input v-model="passwordForm.newPassword" type="password" show-password placeholder="请输入新密码">
                  <template #prefix><Icon icon="mdi:key-change" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item label="确认密码">
                <el-input v-model="passwordForm.confirmPassword" type="password" show-password placeholder="请再次输入新密码">
                  <template #prefix><Icon icon="mdi:key-check" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item>
                <el-button type="primary">
                  <Icon icon="mdi:lock-check-outline" :size="16" /> 修改密码
                </el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <el-tab-pane name="theme">
          <template #label>
            <span class="tab-label">
              <Icon icon="mdi:palette-outline" :size="16" /> 主题设置
            </span>
          </template>
          <div class="settings-form-wrapper">
            <div class="form-section">
              <h3 class="section-title">外观定制</h3>
              <p class="section-desc">选择您偏好的界面风格</p>
            </div>
            <div class="theme-options">
              <div
                class="theme-card"
                :class="{ 'theme-card--active': appStore.theme === 'light' }"
                @click="handleThemeChange('light')"
              >
                <div class="theme-preview theme-preview--light">
                  <div class="preview-sidebar"></div>
                  <div class="preview-content">
                    <div class="preview-header"></div>
                    <div class="preview-body">
                      <div class="preview-card"></div>
                      <div class="preview-card"></div>
                    </div>
                  </div>
                </div>
                <div class="theme-info">
                  <Icon icon="mdi:weather-sunny" :size="20" />
                  <span>浅色模式</span>
                </div>
                <Icon v-if="appStore.theme === 'light'" icon="mdi:check-circle" :size="20" class="theme-check" />
              </div>

              <div
                class="theme-card"
                :class="{ 'theme-card--active': appStore.theme === 'dark' }"
                @click="handleThemeChange('dark')"
              >
                <div class="theme-preview theme-preview--dark">
                  <div class="preview-sidebar"></div>
                  <div class="preview-content">
                    <div class="preview-header"></div>
                    <div class="preview-body">
                      <div class="preview-card"></div>
                      <div class="preview-card"></div>
                    </div>
                  </div>
                </div>
                <div class="theme-info">
                  <Icon icon="mdi:weather-night" :size="20" />
                  <span>深色模式</span>
                </div>
                <Icon v-if="appStore.theme === 'dark'" icon="mdi:check-circle" :size="20" class="theme-check" />
              </div>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { Icon } from '@iconify/vue'
import { useAppStore } from '@/stores/app'
import { ElMessage } from 'element-plus'

const appStore = useAppStore()
const activeTab = ref('profile')

const profileForm = reactive({
  username: 'admin',
  nickname: '管理员',
  email: 'admin@example.com',
})

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

function handleThemeChange(val: string) {
  if (val === 'light' || val === 'dark') {
    appStore.toggleTheme()
    ElMessage.success(`已切换为${val === 'light' ? '浅色' : '深色'}模式`)
  }
}
</script>

<style scoped>
.page-subtitle {
  font-size: var(--font-size-sm);
  color: var(--text-color-secondary);
  margin-top: 4px;
}

.settings-content {
  padding: 0;
  overflow: hidden;
}

.settings-tabs {
  height: 100%;
}

.settings-tabs :deep(.el-tabs__header) {
  padding: 0 var(--spacing-lg);
  margin: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.settings-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.settings-tabs :deep(.el-tabs__item) {
  height: 52px;
  line-height: 52px;
}

.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.settings-form-wrapper {
  padding: var(--spacing-xl) var(--spacing-2xl);
  max-width: 600px;
}

.form-section {
  margin-bottom: var(--spacing-xl);
}

.section-title {
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color-heading);
  margin-bottom: 4px;
}

.section-desc {
  font-size: var(--font-size-sm);
  color: var(--text-color-secondary);
}

.settings-form :deep(.el-input__wrapper) {
  border-radius: 8px;
}

/* ============ 主题卡片 ============ */
.theme-options {
  display: flex;
  gap: var(--spacing-lg);
  margin-top: var(--spacing-md);
}

.theme-card {
  width: 200px;
  border: 2px solid var(--el-border-color-lighter);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s;
  position: relative;
  overflow: hidden;
}

.theme-card:hover {
  border-color: var(--el-color-primary-light-5);
}

.theme-card--active {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px rgba(22, 93, 255, 0.1);
}

.theme-check {
  position: absolute;
  top: 8px;
  right: 8px;
  color: var(--el-color-primary);
}

.theme-preview {
  height: 120px;
  display: flex;
  overflow: hidden;
  border-radius: 10px 10px 0 0;
}

.theme-preview--light {
  background: #F7F8FA;
}

.theme-preview--dark {
  background: #1D2129;
}

.preview-sidebar {
  width: 40px;
  flex-shrink: 0;
}

.theme-preview--light .preview-sidebar {
  background: #1D2129;
}

.theme-preview--dark .preview-sidebar {
  background: #0F172A;
}

.preview-content {
  flex: 1;
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.preview-header {
  height: 12px;
  border-radius: 3px;
}

.theme-preview--light .preview-header {
  background: #fff;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}

.theme-preview--dark .preview-header {
  background: #2D3548;
}

.preview-body {
  flex: 1;
  display: flex;
  gap: 4px;
}

.preview-card {
  flex: 1;
  border-radius: 3px;
}

.theme-preview--light .preview-card {
  background: #fff;
}

.theme-preview--dark .preview-card {
  background: #2D3548;
}

.theme-info {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}
</style>
