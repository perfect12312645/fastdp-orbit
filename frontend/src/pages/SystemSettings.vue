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
              <el-form-item
                label="昵称"
                :rules="[{ min: 1, max: 50, message: '昵称长度不超过50个字符', trigger: 'blur' }]"
              >
                <el-input v-model="profileForm.nickname" placeholder="请输入昵称">
                  <template #prefix><Icon icon="mdi:badge-account-outline" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item
                label="邮箱"
                :rules="[{ type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }]"
              >
                <el-input v-model="profileForm.email" placeholder="请输入邮箱">
                  <template #prefix><Icon icon="mdi:email-outline" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="profileLoading" @click="handleSaveProfile">
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
              <p class="section-desc">密码需至少 8 位，包含大小写字母和数字</p>
            </div>
            <el-form
              ref="pwdFormRef"
              :model="pwdForm"
              :rules="pwdRules"
              label-width="100px"
              class="settings-form"
            >
              <el-form-item label="当前密码" prop="oldPassword">
                <el-input v-model="pwdForm.oldPassword" type="password" show-password placeholder="请输入当前密码">
                  <template #prefix><Icon icon="mdi:key-outline" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item label="新密码" prop="newPassword">
                <el-input v-model="pwdForm.newPassword" type="password" show-password placeholder="请输入新密码">
                  <template #prefix><Icon icon="mdi:key-change" :size="16" /></template>
                </el-input>
                <!-- 密码强度 -->
                <div v-if="pwdForm.newPassword" class="pwd-strength-bar">
                  <el-progress
                    :percentage="pwdStrengthLevel.percent"
                    :color="pwdStrengthLevel.color"
                    :stroke-width="6"
                    :show-text="false"
                    style="flex:1"
                  />
                  <span class="pwd-strength-text" :style="{ color: pwdStrengthLevel.color }">
                    强度：{{ pwdStrengthLevel.text }}
                  </span>
                </div>
              </el-form-item>
              <el-form-item label="确认密码" prop="confirmPassword">
                <el-input v-model="pwdForm.confirmPassword" type="password" show-password placeholder="请再次输入新密码">
                  <template #prefix><Icon icon="mdi:key-check" :size="16" /></template>
                </el-input>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="pwdLoading" @click="handleChangePwd">
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
import { ref, reactive, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { updateProfileApi } from '@/api/auth'

const appStore = useAppStore()
const authStore = useAuthStore()
const activeTab = ref('profile')
const profileLoading = ref(false)

// ============ 个人信息 ============
const profileForm = reactive({
  username: '',
  nickname: '',
  email: '',
})

onMounted(() => {
  if (authStore.userInfo) {
    profileForm.username = authStore.userInfo.username || ''
    profileForm.nickname = authStore.userInfo.nickname || ''
    profileForm.email = authStore.userInfo.email || ''
  }
})

async function handleSaveProfile() {
  profileLoading.value = true
  try {
    await updateProfileApi({
      nickname: profileForm.nickname,
      email: profileForm.email,
    })
    // 更新 store 中的 userInfo
    if (authStore.userInfo) {
      authStore.userInfo.nickname = profileForm.nickname
      authStore.userInfo.email = profileForm.email
    }
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    profileLoading.value = false
  }
}

// ============ 修改密码 ============
const pwdFormRef = ref<FormInstance>()
const pwdLoading = ref(false)
const pwdForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})
const pwdRules: FormRules = {
  oldPassword: [
    { required: true, message: '请输入当前密码', trigger: 'blur' },
  ],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '密码长度不能少于8位', trigger: 'blur' },
    {
      validator: (_rule: any, value: string, callback: any) => {
        if (!value) return callback()
        if (!/[A-Z]/.test(value)) return callback(new Error('需包含至少一个大写字母'))
        if (!/[a-z]/.test(value)) return callback(new Error('需包含至少一个小写字母'))
        if (!/[0-9]/.test(value)) return callback(new Error('需包含至少一个数字'))
        if (/[;&$\\|]/.test(value)) return callback(new Error('不能包含特殊字符 ; & $ \\ |'))
        if (value === pwdForm.oldPassword) return callback(new Error('新密码不能与旧密码相同'))
        callback()
      },
      trigger: 'blur',
    },
  ],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (_rule: any, value: string, callback: any) => {
        if (value && value !== pwdForm.newPassword) {
          return callback(new Error('两次输入的密码不一致'))
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
}

const pwdStrength = computed(() => {
  const pwd = pwdForm.newPassword
  let score = 0
  if (pwd.length >= 8) score += 25
  if (/[A-Z]/.test(pwd)) score += 25
  if (/[a-z]/.test(pwd)) score += 25
  if (/[0-9]/.test(pwd)) score += 25
  return score
})

const pwdStrengthLevel = computed(() => {
  if (pwdStrength.value < 50) return { text: '弱', color: '#f56c6c', percent: pwdStrength.value }
  if (pwdStrength.value < 75) return { text: '中', color: '#e6a23c', percent: pwdStrength.value }
  return { text: '强', color: '#67c23a', percent: pwdStrength.value }
})

async function handleChangePwd() {
  if (!pwdFormRef.value) return
  try {
    await pwdFormRef.value.validate()
  } catch {
    return
  }
  pwdLoading.value = true
  try {
    const { changePasswordApi } = await import('@/api/auth')
    await changePasswordApi({
      old_password: pwdForm.oldPassword,
      new_password: pwdForm.newPassword,
    })
    ElMessage.success('密码修改成功')
    pwdForm.oldPassword = ''
    pwdForm.newPassword = ''
    pwdForm.confirmPassword = ''
  } catch (err: any) {
    ElMessage.error(err?.message || '密码修改失败')
  } finally {
    pwdLoading.value = false
  }
}

// ============ 主题 ============
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

/* 密码强度条 */
.pwd-strength-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 6px;
}
.pwd-strength-text {
  font-size: 12px;
  white-space: nowrap;
  min-width: 80px;
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
