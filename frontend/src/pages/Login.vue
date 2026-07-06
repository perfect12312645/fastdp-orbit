<template>
  <div class="login-page">
    <!-- 背景动效 -->
    <div class="login-bg">
      <div class="bg-gradient"></div>
      <div class="bg-grid"></div>
      <div class="bg-orb orb-1"></div>
      <div class="bg-orb orb-2"></div>
      <div class="bg-orb orb-3"></div>
      <div class="particles">
        <div v-for="i in 20" :key="i" class="particle" :style="particleStyle(i)"></div>
      </div>
    </div>

    <!-- 左侧品牌区 -->
    <div class="login-branding">
      <div class="branding-content">
        <div class="brand-logo">
          <Icon icon="mdi:orbit" :size="40" />
        </div>
        <h1 class="brand-name">fastdp-orbit</h1>
        <p class="brand-subtitle">多机可视化运维平台</p>
        <div class="brand-features">
          <div class="feature-item" v-for="(f, i) in features" :key="i">
            <Icon :icon="f.icon" :size="20" />
            <span>{{ f.text }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧登录卡片 -->
    <div class="login-container">
      <div class="login-card">
        <div class="login-header">
          <div class="login-logo-mobile">
            <Icon icon="mdi:orbit" :size="28" />
          </div>
          <h2 class="login-title">欢迎回来</h2>
          <p class="login-desc">请登录您的账号以继续</p>
        </div>

        <el-form
          ref="loginFormRef"
          :model="loginForm"
          :rules="loginRules"
          class="login-form"
          size="large"
          @keyup.enter="handleLogin"
        >
          <el-form-item prop="username">
            <el-input
              v-model="loginForm.username"
              placeholder="用户名"
              clearable
            >
              <template #prefix>
                <Icon icon="mdi:account-outline" :size="18" />
              </template>
            </el-input>
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="loginForm.password"
              type="password"
              placeholder="密码"
              show-password
              clearable
            >
              <template #prefix>
                <Icon icon="mdi:lock-outline" :size="18" />
              </template>
            </el-input>
          </el-form-item>

          <div class="login-extra">
            <el-checkbox v-model="rememberMe">记住我</el-checkbox>
            <a class="forgot-link" href="javascript:void(0)" @click="showForgotDialog = true">忘记密码？</a>
          </div>

          <el-form-item style="margin-bottom: 0;">
            <el-button
              type="primary"
              class="login-btn"
              :loading="loading"
              @click="handleLogin"
            >
              登 录
            </el-button>
          </el-form-item>
        </el-form>

        <div class="login-footer">
          <span class="footer-text">多机可视化运维平台</span>
        </div>
      </div>
    </div>
  </div>

  <!-- 忘记密码弹窗 -->
  <el-dialog
    v-model="showForgotDialog"
    title="重置密码"
    width="480px"
    align-center
    top="25vh"
  >
    <div class="forgot-body">
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 16px;"
      >
        <template #title>忘记密码后如何重置</template>
      </el-alert>

      <div class="forgot-steps">
        <div class="forgot-step">
          <span class="step-num">1</span>
          <span>SSH 登录到服务器</span>
        </div>
        <div class="forgot-step">
          <span class="step-num">2</span>
          <span>执行以下命令重置密码：</span>
        </div>
        <div class="forgot-code-block">
          <code>orbitctl reset-password admin</code>
          <el-button link type="primary" size="small" @click="copyCmd">复制</el-button>
        </div>
        <div class="forgot-step">
          <span class="step-num">3</span>
          <span>终端会显示新密码，用它登录后请立即修改</span>
        </div>
      </div>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        style="margin-top: 16px;"
      >
        <template #title>
          如果 server.toml 不在默认路径，需加上 --config 参数指定配置文件路径
        </template>
      </el-alert>

      <el-button
        type="primary"
        class="login-btn"
        style="margin-top: 20px;"
        @click="showForgotDialog = false"
      >
        知道了
      </el-button>
    </div>
  </el-dialog>

  <!-- 强制修改密码弹窗 -->
  <el-dialog
    v-model="authStore.needsPasswordChange"
    title="首次登录 - 请修改密码"
    width="440px"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="false"
    align-center
    top="20vh"
  >
    <el-form
      ref="pwdFormRef"
      :model="pwdForm"
      :rules="pwdRules"
      label-width="0"
      size="large"
      @keyup.enter="handleChangePwd"
    >
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        class="pwd-alert"
      >
        <template #title>
          首次登录请修改默认密码（admin123），密码需满足：
          <ul style="margin: 4px 0 0 16px; padding: 0;">
            <li>至少 8 位</li>
            <li>包含大写字母</li>
            <li>包含小写字母</li>
            <li>包含数字</li>
          </ul>
        </template>
      </el-alert>

      <el-form-item prop="oldPassword">
        <el-input
          v-model="pwdForm.oldPassword"
          type="password"
          placeholder="当前密码"
          show-password
        >
          <template #prefix>
            <Icon icon="mdi:lock-outline" :size="18" />
          </template>
        </el-input>
      </el-form-item>

      <el-form-item prop="newPassword">
        <el-input
          v-model="pwdForm.newPassword"
          type="password"
          placeholder="新密码"
          show-password
        >
          <template #prefix>
            <Icon icon="mdi:lock-plus-outline" :size="18" />
          </template>
        </el-input>
      </el-form-item>

      <!-- 密码强度条 -->
      <div v-if="pwdForm.newPassword" class="pwd-strength-bar">
        <el-progress
          :percentage="pwdStrengthLevel.percent"
          :color="pwdStrengthLevel.color"
          :stroke-width="6"
          :show-text="false"
        />
        <span class="pwd-strength-text" :style="{ color: pwdStrengthLevel.color }">
          密码强度：{{ pwdStrengthLevel.text }}
        </span>
      </div>

      <el-form-item prop="confirmPassword">
        <el-input
          v-model="pwdForm.confirmPassword"
          type="password"
          placeholder="确认新密码"
          show-password
        >
          <template #prefix>
            <Icon icon="mdi:check-circle-outline" :size="18" />
          </template>
        </el-input>
      </el-form-item>

      <el-form-item>
        <el-button
          type="primary"
          class="login-btn"
          :loading="pwdLoading"
          @click="handleChangePwd"
          style="width: 100%;"
        >
          确认修改
        </el-button>
      </el-form-item>
    </el-form>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
const authStore = useAuthStore()
const loginFormRef = ref<FormInstance>()
const loading = ref(false)

const SAVED_USER_KEY = 'saved_username'

// 记住我 - 加载已存用户名
const rememberMe = ref(false)
const savedUser = localStorage.getItem(SAVED_USER_KEY)
if (savedUser) {
  rememberMe.value = true
}

const loginForm = reactive({
  username: savedUser || '',
  password: '',
})

const loginRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 50, message: '用户名长度应在2-50个字符之间', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 1, max: 50, message: '密码长度应在1-50个字符之间', trigger: 'blur' },
  ],
}

// 强制改密码
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

// 密码强度计算
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
    await authStore.changePassword({
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

// ============ 忘记密码 ============
const showForgotDialog = ref(false)

function copyCmd() {
  navigator.clipboard.writeText('orbitctl reset-password admin')
  ElMessage.success('已复制命令')
}

// ============ 功能 ============
const features = [
  { icon: 'mdi:view-dashboard-outline', text: '声明式可视化任务编排' },
  { icon: 'mdi:server-network-outline', text: '多机统一管理，快速批量任务执行' },
  { icon: 'mdi:play-circle-outline', text: '丰富运维模块' },
  { icon: 'mdi:package-variant-closed', text: '预制方案即开即用' },
]

function particleStyle(i: number) {
  const left = Math.random() * 100
  const delay = Math.random() * 6
  const duration = 4 + Math.random() * 4
  const size = 2 + Math.random() * 3
  return {
    left: `${left}%`,
    animationDelay: `${delay}s`,
    animationDuration: `${duration}s`,
    width: `${size}px`,
    height: `${size}px`,
  }
}

async function handleLogin() {
  if (!loginFormRef.value) return
  try {
    await loginFormRef.value.validate()
  } catch {
    return
  }
  loading.value = true
  try {
    const needChange = await authStore.login({
      username: loginForm.username,
      password: loginForm.password,
    })
    if (needChange) {
      pwdForm.oldPassword = loginForm.password
    } else {
      ElMessage.success('登录成功')
    }
    // 记住我：勾选则保存用户名，否则清除
    if (rememberMe.value) {
      localStorage.setItem(SAVED_USER_KEY, loginForm.username)
    } else {
      localStorage.removeItem(SAVED_USER_KEY)
    }
  } catch {
    ElMessage.error('登录失败，请检查用户名和密码')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  width: 100%;
  height: 100vh;
  display: flex;
  overflow: hidden;
  position: relative;
}

/* ============ 背景 ============ */
.login-bg {
  position: absolute;
  inset: 0;
  z-index: 0;
}

.bg-gradient {
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, #0F172A 0%, #1E293B 40%, #0F172A 100%);
}

.bg-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(22, 93, 255, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(22, 93, 255, 0.03) 1px, transparent 1px);
  background-size: 40px 40px;
}

.bg-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  animation: orbFloat 8s ease-in-out infinite;
}

.orb-1 {
  width: 500px;
  height: 500px;
  background: rgba(22, 93, 255, 0.15);
  top: -150px;
  right: -100px;
}

.orb-2 {
  width: 350px;
  height: 350px;
  background: rgba(114, 46, 209, 0.12);
  bottom: -100px;
  left: -50px;
  animation-delay: -3s;
}

.orb-3 {
  width: 200px;
  height: 200px;
  background: rgba(0, 180, 42, 0.1);
  top: 50%;
  left: 30%;
  animation-delay: -5s;
}

@keyframes orbFloat {
  0%, 100% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(30px, -20px) scale(1.05); }
}

.particles {
  position: absolute;
  inset: 0;
  overflow: hidden;
}

.particle {
  position: absolute;
  bottom: -10px;
  background: rgba(22, 93, 255, 0.6);
  border-radius: 50%;
  animation: particleRise linear infinite;
}

@keyframes particleRise {
  0% { transform: translateY(0) scale(1); opacity: 0; }
  10% { opacity: 1; }
  90% { opacity: 0.5; }
  100% { transform: translateY(-100vh) scale(0.5); opacity: 0; }
}

/* ============ 品牌区 ============ */
.login-branding {
  position: relative;
  z-index: 1;
  width: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-2xl);
}

.branding-content {
  max-width: 420px;
  animation: slideInLeft 0.8s ease-out;
}

@keyframes slideInLeft {
  from { transform: translateX(-40px); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}

.brand-logo {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  background: var(--gradient-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  margin-bottom: 24px;
  box-shadow: 0 8px 24px rgba(22, 93, 255, 0.3);
}

.brand-name {
  font-size: 36px;
  font-weight: 800;
  color: #fff;
  margin-bottom: 8px;
  letter-spacing: -0.03em;
}

.brand-subtitle {
  font-size: 16px;
  color: rgba(255, 255, 255, 0.5);
  margin-bottom: 48px;
}

.brand-features {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 12px;
  color: rgba(255, 255, 255, 0.7);
  font-size: 14px;
}

.feature-item :deep(svg) {
  color: #4080FF;
}

/* ============ 登录卡片 ============ */
.login-container {
  position: relative;
  z-index: 1;
  width: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}

.login-card {
  width: 100%;
  max-width: 400px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
  padding: 48px 40px;
  backdrop-filter: blur(20px);
  animation: slideInRight 0.8s ease-out;
}

@keyframes slideInRight {
  from { transform: translateX(40px); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}

.login-header {
  text-align: center;
  margin-bottom: 36px;
}

.login-logo-mobile {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: var(--gradient-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  margin-bottom: 20px;
}

.login-title {
  font-size: 24px;
  font-weight: 700;
  color: #fff;
  margin-bottom: 8px;
}

.login-desc {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.5);
}

.login-form {
  width: 100%;
}

.login-form :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  box-shadow: none;
  height: 46px;
}

.login-form :deep(.el-input__wrapper:hover) {
  border-color: rgba(22, 93, 255, 0.4);
}

.login-form :deep(.el-input__wrapper.is-focus) {
  border-color: #165DFF;
  box-shadow: 0 0 0 2px rgba(22, 93, 255, 0.15);
}

.login-form :deep(.el-input__inner) {
  color: #fff;
}

.login-form :deep(.el-input__inner::placeholder) {
  color: rgba(255, 255, 255, 0.35);
}

.login-form :deep(.el-input__prefix .iconify) {
  color: rgba(255, 255, 255, 0.4);
}

.login-extra {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.login-extra :deep(.el-checkbox__label) {
  color: rgba(255, 255, 255, 0.5);
  font-size: 13px;
}

.forgot-link {
  font-size: 13px;
  color: #4080FF;
  text-decoration: none;
  cursor: pointer;
  transition: color 0.2s;
}

.forgot-link:hover {
  color: #69b1ff;
}

.login-btn {
  width: 100%;
  height: 46px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 10px;
  background: var(--gradient-primary);
  border: none;
  letter-spacing: 0.05em;
}

.login-btn:hover {
  opacity: 0.9;
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(22, 93, 255, 0.4);
}

.login-footer {
  text-align: center;
  margin-top: 28px;
  padding-top: 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.footer-text {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.3);
}

/* ============ 响应式 ============ */
@media (max-width: 900px) {
  .login-branding {
    display: none;
  }
  .login-container {
    width: 100%;
  }
}
</style>

<!-- 改密码对话框样式（非 scoped，el-dialog 使用 Teleport） -->
<style>
.pwd-alert {
  margin-bottom: 20px;
}
.pwd-alert ul {
  font-size: 13px;
  line-height: 1.8;
}
.pwd-strength-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: -10px 0 18px;
}
.pwd-strength-bar .el-progress {
  flex: 1;
}
.pwd-strength-text {
  font-size: 12px;
  white-space: nowrap;
  min-width: 80px;
}
.forgot-body {
  padding: 0 4px;
}
.forgot-steps {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.forgot-step {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: #333;
}
.step-num {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #165DFF;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.forgot-code-block {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 10px 14px;
  margin-left: 32px;
}
.forgot-code-block code {
  flex: 1;
  font-size: 14px;
  font-weight: 600;
  color: #165DFF;
  letter-spacing: 0.02em;
}
</style>
