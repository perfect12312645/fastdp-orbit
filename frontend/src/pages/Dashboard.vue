<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>仪表盘</h2>
        <p class="page-subtitle">自动化运维平台概览</p>
      </div>
      <div class="header-actions">
        <el-button @click="loadData" :loading="loading">
          <Icon icon="mdi:refresh" :size="16" /> 刷新
        </el-button>
      </div>
    </div>

    <div class="page-content">
      <!-- 统计卡片 -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon" style="background: rgba(22, 93, 255, 0.1);">
            <Icon icon="mdi:server-network-outline" :size="28" style="color: #165DFF;" />
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_machines }}</div>
            <div class="stat-label">机器总数</div>
            <div class="stat-sub">在线 {{ stats.online_machines }} 台</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon" style="background: rgba(0, 180, 42, 0.1);">
            <Icon icon="mdi:play-circle-outline" :size="28" style="color: #00B42A;" />
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_workflows }}</div>
            <div class="stat-label">工作流</div>
            <div class="stat-sub">{{ stats.total_stages }} 个阶段</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon" style="background: rgba(255, 125, 0, 0.1);">
            <Icon icon="mdi:check-circle-outline" :size="28" style="color: #FF7D00;" />
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.exec_stats?.total || 0 }}</div>
            <div class="stat-label">执行总数</div>
            <div class="stat-sub">
              <span style="color: #00B42A;">成功 {{ stats.exec_stats?.success || 0 }}</span>
              <span style="margin: 0 6px;">/</span>
              <span style="color: #F53F3F;">失败 {{ stats.exec_stats?.failed || 0 }}</span>
            </div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon" style="background: rgba(114, 46, 209, 0.1);">
            <Icon icon="mdi:harddisk-variant-outline" :size="28" style="color: #722ED1;" />
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_files }}</div>
            <div class="stat-label">存储文件</div>
            <div class="stat-sub">{{ stats.total_variables }} 个变量 / {{ stats.total_hooks }} 个钩子</div>
          </div>
        </div>
      </div>

      <!-- 最近执行记录 -->
      <el-card class="recent-exec-card">
        <template #header>
          <div class="card-header">
            <span class="card-title">
              <Icon icon="mdi:history" :size="18" /> 最近执行
            </span>
            <el-button type="primary" link @click="$router.push('/workflow')">
              查看全部 <Icon icon="mdi:arrow-right" :size="14" />
            </el-button>
          </div>
        </template>

        <el-table :data="stats.recent_execs" style="width: 100%" stripe v-loading="loading">
          <el-table-column label="工作流" prop="workflow_name" min-width="150" />
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small" effect="light" round>
                {{ getStatusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="触发方式" width="100" align="center">
            <template #default="{ row }">
              <span style="font-size: 12px; color: var(--el-text-color-secondary);">{{ row.trigger }}</span>
            </template>
          </el-table-column>
          <el-table-column label="开始时间" width="170">
            <template #default="{ row }">
              <span style="font-size: 13px;">{{ row.started_at }}</span>
            </template>
          </el-table-column>
          <el-table-column label="耗时" width="100" align="center">
            <template #default="{ row }">
              <span style="font-size: 13px;">{{ formatDuration(row.duration_ms) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ row }">
              <el-button type="primary" link size="small" @click="goToExecution(row)">
                详情
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div v-if="!stats.recent_execs?.length && !loading" class="empty-tip">
          暂无执行记录
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import request from '@/utils/request'

const router = useRouter()
const loading = ref(false)

const stats = ref<any>({
  total_machines: 0,
  online_machines: 0,
  total_workflows: 0,
  total_stages: 0,
  total_variables: 0,
  total_hooks: 0,
  total_files: 0,
  recent_execs: [],
  exec_stats: { total: 0, success: 0, failed: 0, running: 0 },
})

async function loadData() {
  loading.value = true
  try {
    const res = await request.get('/dashboard/stats')
    stats.value = res.data.data
  } catch (e) {
    console.error('加载仪表盘数据失败', e)
  } finally {
    loading.value = false
  }
}

function getStatusType(status: string) {
  const map: Record<string, string> = {
    running: 'warning',
    success: 'success',
    failed: 'danger',
    paused: 'info',
    cancelled: 'info',
  }
  return (map[status] || 'info') as any
}

function getStatusLabel(status: string) {
  const map: Record<string, string> = {
    running: '运行中',
    success: '成功',
    failed: '失败',
    paused: '已暂停',
    cancelled: '已终止',
  }
  return map[status] || status
}

function formatDuration(ms: number) {
  if (!ms || ms <= 0) return '-'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  const min = Math.floor(ms / 60000)
  const sec = Math.floor((ms % 60000) / 1000)
  return `${min}m${sec}s`
}

function goToExecution(row: any) {
  // 需要 workflow_id，从 recent_execs 中获取
  // 暂时跳转到工作流列表
  router.push('/workflow')
}

onMounted(loadData)
</script>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.2;
}

.stat-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
}

.stat-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.recent-exec-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.empty-tip {
  text-align: center;
  color: var(--el-text-color-secondary);
  padding: 40px;
  font-size: 13px;
}
</style>
