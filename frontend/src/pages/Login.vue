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
        <p class="brand-subtitle">异构算力运维平台</p>
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
            <a class="forgot-link" href="javascript:void(0)">忘记密码？</a>
          </div>

          <el-form-item>
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
          <span class="footer-text">GPU集群可视化运维平台</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { Icon } from '@iconify/vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const loginFormRef = ref<FormInstance>()
const loading = ref(false)
const rememberMe = ref(false)

const features = [
  { icon: 'mdi:chip-outline', text: 'GPU资源实时监控' },
  { icon: 'mdi:server-network-outline', text: '集群节点统一管理' },
  { icon: 'mdi:robot-outline', text: '大模型服务编排' },
  { icon: 'mdi:chart-line', text: '可视化数据分析' },
]

const loginForm = reactive({
  username: '',
  password: '',
})

const loginRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 50, message: '用户名长度应在2-50个字符之间', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 50, message: '密码长度应在6-50个字符之间', trigger: 'blur' },
  ],
}

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
    await authStore.login({
      username: loginForm.username,
      password: loginForm.password,
    })
    ElMessage.success('登录成功')
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
