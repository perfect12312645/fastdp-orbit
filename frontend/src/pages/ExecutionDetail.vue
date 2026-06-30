<template>
  <div class="page-container">
    <div class="page-header">
      <div class="header-left">
        <el-button link @click="goBack">
          <Icon icon="mdi:arrow-left" :size="18" /> 返回
        </el-button>
        <h2>执行详情</h2>
      </div>
      <div class="header-actions" v-if="execution">
        <template v-if="execution.status === 'running'">
          <el-button type="warning" @click="handlePause">
            <Icon icon="mdi:pause" :size="16" /> 暂停
          </el-button>
          <el-button type="danger" @click="handleCancel">
            <Icon icon="mdi:stop" :size="16" /> 终止
          </el-button>
        </template>
        <template v-else-if="execution.status === 'paused'">
          <el-button type="success" @click="handleResume">
            <Icon icon="mdi:play" :size="16" /> 恢复
          </el-button>
          <el-button type="danger" @click="handleCancel">
            <Icon icon="mdi:stop" :size="16" /> 终止
          </el-button>
        </template>
        <template v-else-if="execution.status === 'failed'">
          <el-button type="primary" @click="handleRetryExecution">
            <Icon icon="mdi:restart" :size="16" /> 重新执行
          </el-button>
        </template>
      </div>
    </div>

    <div class="page-content" v-loading="loading">
      <template v-if="execution">
        <!-- 基本信息 -->
        <div class="info-bar">
          <div class="info-item">
            <span class="label">状态</span>
            <el-tag :type="getStatusType(execution.status)" size="default">
              {{ getStatusLabel(execution.status) }}
            </el-tag>
          </div>
          <div class="info-item">
            <span class="label">触发者</span>
            <span>{{ execution.trigger || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="label">开始时间</span>
            <span>{{ formatDateTime(execution.started_at) }}</span>
          </div>
          <div class="info-item">
            <span class="label">结束时间</span>
            <span>{{ execution.finished_at ? formatDateTime(execution.finished_at) : '-' }}</span>
          </div>
          <div class="info-item error" v-if="execution.error">
            <span class="label">错误</span>
            <span>{{ execution.error }}</span>
          </div>
        </div>

        <!-- 阶段组执行详情 -->
        <div class="stages-container">
          <div
            v-for="(sge, sgei) in execution.stage_group_executions"
            :key="sge.id"
            class="stage-card"
          >
            <div class="stage-header">
              <div class="stage-status">
                <Icon :icon="getStageIcon(sge.status)" :size="20" :class="'status-' + sge.status" />
              </div>
              <div class="stage-info">
                <span class="stage-name">阶段组 {{ sgei + 1 }}: {{ sge.group?.name || `Group #${sge.group_id}` }}</span>
                <span class="stage-time" v-if="sge.started_at">
                  {{ formatDateTime(sge.started_at) }}
                  <template v-if="sge.finished_at"> ~ {{ formatDateTime(sge.finished_at) }}</template>
                </span>
              </div>
              <el-tag :type="getStatusType(sge.status)" size="small">
                {{ getStatusLabel(sge.status) }}
              </el-tag>
              <el-button
                v-if="sge.status === 'failed' && execution.status !== 'cancelled'"
                type="warning"
                link
                size="small"
                @click="handleRetryStage(sge.id)"
              >
                <Icon icon="mdi:restart" :size="14" /> 重试
              </el-button>
            </div>

            <!-- 子阶段列表 -->
            <div class="tasks-container" v-if="sge.stage_executions?.length">
              <div
                v-for="(se, sei) in sge.stage_executions"
                :key="se.id"
                class="task-item"
              >
                <div class="task-header">
                  <Icon :icon="getTaskIcon(se.status)" :size="16" :class="'status-' + se.status" />
                  <span class="task-name">{{ se.stage?.name || `Stage #${se.stage_id}` }}</span>
                  <span class="task-host">{{ se.task_executions?.length || 0 }} 个任务</span>
                  <el-tag :type="getStatusType(se.status)" size="small">
                    {{ getStatusLabel(se.status) }}
                  </el-tag>
                </div>

                <!-- 任务执行列表 - 按 ref 分组 -->
                <div class="subtasks-container" v-if="se.task_executions?.length">
                  <div
                    v-for="group in groupTasksByRef(se.task_executions)"
                    :key="group.ref"
                    class="subtask-item"
                  >
                    <div class="task-header">
                      <Icon :icon="getTaskIcon(group.machines[0]?.status, group.machines[0]?.changed)" :size="14" :class="'status-' + group.machines[0]?.status" />
                      <span class="task-name">任务-{{ group.ref }} {{ group.name }}</span>
                      <span class="task-module" v-if="group.module">{{ group.module }}</span>
                    </div>
                    <div class="subtask-machines">
                      <div
                        v-for="(te, tei) in group.machines"
                        :key="te.id"
                        class="subtask-machine"
                        :class="getTaskClass(te.status, te.changed)"
                      >
                        <div class="machine-header">
                          <span class="task-host">{{ te.host }}</span>
                          <span class="task-duration" v-if="te.duration_ms">{{ te.duration_ms }}ms</span>
                        </div>
                        <div class="task-output" v-if="te.output">
                          <pre>{{ te.output }}</pre>
                        </div>
                        <div class="task-error" v-if="te.error">
                          <pre>{{ te.error }}</pre>
                        </div>
                        <div class="task-trace" v-if="te.trace">
                          <pre>{{ te.trace }}</pre>
                        </div>
                        <div class="task-hook-status" v-if="te.hook_status && te.hook_status !== 'none'">
                          <span class="hook-label">钩子:</span>
                          <el-tag :type="te.hook_status === 'success' ? 'success' : te.hook_status === 'failed' ? 'danger' : 'warning'" size="small">
                            {{ te.hook_status === 'success' ? '成功' : te.hook_status === 'failed' ? '失败' : '执行中' }}
                          </el-tag>
                          <span class="hook-error" v-if="te.hook_error">{{ te.hook_error }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getExecutionApi,
  pauseExecutionApi,
  resumeExecutionApi,
  cancelExecutionApi,
  retryExecutionApi,
  retryStageApi,
} from '@/api/workflow'
import { formatDateTime } from '@/utils/format'
import type { WorkflowExecution } from '@/types/workflow'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const execution = ref<WorkflowExecution | null>(null)
let refreshTimer: ReturnType<typeof setInterval> | null = null
let eventSource: EventSource | null = null

const workflowId = computed(() => Number(route.params.id))
const executionId = computed(() => Number(route.params.eid))

const isRunning = computed(() => {
  if (!execution.value) return false
  if (execution.value.status === 'running' || execution.value.status === 'paused') return true
  return execution.value.stage_group_executions?.some(
    (sge) => sge.status === 'running'
  ) ?? false
})

// 按 task_ref 分组任务执行记录
function groupTasksByRef(taskExecutions: any[]) {
  const groups = new Map<number, { ref: number; name: string; module: string; machines: any[] }>()
  for (const te of taskExecutions) {
    const ref = te.task?.ref || te.task_id
    if (!groups.has(ref)) {
      groups.set(ref, {
        ref,
        name: te.task?.name || `Task #${te.task_id}`,
        module: te.task?.module || '',
        machines: [],
      })
    }
    groups.get(ref)!.machines.push(te)
  }
  return Array.from(groups.values())
}

async function loadExecution() {
  loading.value = true
  try {
    execution.value = await getExecutionApi(workflowId.value, executionId.value)
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function connectSSE() {
  disconnectSSE()
  const url = `/api/v1/executions/${executionId.value}/stream`
  eventSource = new EventSource(url)
  eventSource.addEventListener('connected', () => {
    console.log('[SSE] connected to execution', executionId.value)
  })
  eventSource.addEventListener('execution_status', (e) => {
    const data = JSON.parse(e.data)
    if (execution.value && execution.value.id === data.execution_id) {
      execution.value = { ...execution.value, status: data.status, error: data.error || execution.value.error }
      if (data.status === 'success' || data.status === 'failed' || data.status === 'cancelled') {
        disconnectSSE()
        loadExecution()
      }
    }
  })
  eventSource.addEventListener('group_status', (e) => {
    const data = JSON.parse(e.data)
    if (!execution.value || execution.value.id !== data.execution_id) return
    const groups = execution.value.stage_group_executions || []
    const idx = groups.findIndex(g => g.group_id === data.group_id)
    if (idx >= 0) {
      const updated = [...groups]
      updated[idx] = { ...updated[idx], status: data.status }
      execution.value = { ...execution.value, stage_group_executions: updated }
    }
  })
  eventSource.addEventListener('stage_status', (e) => {
    const data = JSON.parse(e.data)
    if (!execution.value || execution.value.id !== data.execution_id) return
    const groups = execution.value.stage_group_executions || []
    for (let gi = 0; gi < groups.length; gi++) {
      const stages = groups[gi].stage_executions || []
      const si = stages.findIndex(s => s.stage_id === data.stage_id)
      if (si >= 0) {
        const updatedGroups = [...groups]
        const updatedStages = [...stages]
        updatedStages[si] = { ...updatedStages[si], status: data.status }
        updatedGroups[gi] = { ...updatedGroups[gi], stage_executions: updatedStages }
        execution.value = { ...execution.value, stage_group_executions: updatedGroups }
        return
      }
    }
  })
  eventSource.onerror = () => {
    if (eventSource?.readyState === EventSource.CLOSED) {
      console.warn('[SSE] connection closed, falling back to polling')
      disconnectSSE()
      startAutoRefresh()
    }
  }
}

function disconnectSSE() {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
}

function startAutoRefresh() {
  stopAutoRefresh()
  refreshTimer = setInterval(() => {
    loadExecution()
  }, 3000)
}

function stopAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

async function handlePause() {
  try {
    await ElMessageBox.confirm('确认暂停执行？', '暂停确认', {
      confirmButtonText: '暂停',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await pauseExecutionApi(workflowId.value, executionId.value)
    ElMessage.success('已暂停')
    await loadExecution()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('暂停失败')
  }
}

async function handleResume() {
  try {
    await resumeExecutionApi(workflowId.value, executionId.value)
    ElMessage.success('已恢复')
    await loadExecution()
    connectSSE()
  } catch (e) {
    ElMessage.error('恢复失败')
  }
}

async function handleCancel() {
  try {
    await ElMessageBox.confirm('确认终止执行？终止后不可恢复。', '终止确认', {
      confirmButtonText: '终止',
      cancelButtonText: '取消',
      type: 'error',
    })
    await cancelExecutionApi(workflowId.value, executionId.value)
    ElMessage.success('已终止')
    await loadExecution()
    disconnectSSE()
    startAutoRefresh()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('终止失败')
  }
}

async function handleRetryExecution() {
  try {
    await ElMessageBox.confirm('确认重新执行？将从头开始执行。', '重新执行确认', {
      confirmButtonText: '重新执行',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await retryExecutionApi(workflowId.value, executionId.value)
    ElMessage.success('已重新触发执行')
    await loadExecution()
    connectSSE()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('重新执行失败')
  }
}

async function handleRetryStage(stageId: number) {
  try {
    await retryStageApi(workflowId.value, executionId.value, stageId)
    ElMessage.success('已重试该阶段')
    await loadExecution()
    connectSSE()
  } catch (e) {
    ElMessage.error('重试失败')
  }
}

function goBack() {
  router.push('/workflow')
}

function getStatusType(status: string) {
  const map: Record<string, string> = {
    running: 'warning',
    success: 'success',
    failed: 'danger',
    paused: 'info',
    cancelled: 'info',
    pending: 'info',
    skipped: 'info',
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
    pending: '等待中',
    skipped: '已跳过',
  }
  return map[status] || status
}

function getStageIcon(status: string) {
  const map: Record<string, string> = {
    running: 'mdi:loading',
    success: 'mdi:check-circle',
    failed: 'mdi:close-circle',
    pending: 'mdi:clock-outline',
    skipped: 'mdi:skip-next',
  }
  return map[status] || 'mdi:circle-outline'
}

function getTaskIcon(status: string, changed?: boolean) {
  if (status === 'success' && changed === false) return 'mdi:check-circle-outline'
  const map: Record<string, string> = {
    running: 'mdi:loading',
    success: 'mdi:check-circle',
    failed: 'mdi:close-circle',
    pending: 'mdi:clock-outline',
    skipped: 'mdi:skip-next',
  }
  return map[status] || 'mdi:circle-outline'
}

function getTaskClass(status: string, changed?: boolean) {
  if (status === 'success' && changed === false) return 'subtask-unchanged'
  return `subtask-${status}`
}

onMounted(() => {
  loadExecution().then(() => {
    if (isRunning.value) {
      connectSSE()
    }
    // 不再自动刷新 - 只在 SSE 连接时实时更新
  })
})

onUnmounted(() => {
  disconnectSSE()
  stopAutoRefresh()
})
</script>

<style scoped>
.page-container {
  padding: 24px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 8px 0 0;
  font-size: 22px;
  font-weight: 600;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.page-content {
  flex: 1;
  background: var(--el-bg-color);
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.info-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-item .label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.info-item.error {
  width: 100%;
  color: var(--el-color-danger);
}

.stages-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.stage-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}

.stage-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--el-fill-color-lighter);
}

.stage-status {
  flex-shrink: 0;
}

.stage-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stage-name {
  font-weight: 600;
  font-size: 14px;
}

.stage-time {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.tasks-container {
  padding: 8px;
}

.task-item {
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.task-item:last-child {
  border-bottom: none;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-name {
  font-weight: 500;
}

.task-host {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.task-duration {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.task-output,
.task-error,
.task-trace {
  margin-top: 8px;
  padding: 8px;
  border-radius: 4px;
  font-size: 12px;
  overflow-x: auto;
}

.task-output {
  background: var(--el-fill-color-lighter);
}

.task-error {
  background: rgba(245, 63, 63, 0.06);
  color: var(--el-color-danger);
}

.task-trace {
  background: rgba(230, 162, 60, 0.06);
  color: var(--el-color-warning);
  font-size: 11px;
}

.task-output pre,
.task-error pre,
.task-trace pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.task-hook-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  font-size: 12px;
}

.hook-label {
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}

.hook-error {
  color: var(--el-color-danger);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.subtasks-container {
  padding: 4px 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.subtask-item {
  padding: 8px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.subtask-item:last-child {
  border-bottom: none;
}

.task-module {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-lighter);
  padding: 1px 6px;
  border-radius: 4px;
  margin-left: 4px;
}

.subtask-machines {
  margin-left: 24px;
  margin-top: 4px;
}

.subtask-machine {
  padding: 6px 8px;
  border-left: 3px solid transparent;
  margin-bottom: 4px;
}

.subtask-machine:last-child {
  margin-bottom: 0;
}

.subtask-success {
  border-left-color: var(--el-color-success);
}

.subtask-unchanged {
  border-left-color: #e6a23c;
}

.subtask-failed {
  border-left-color: var(--el-color-danger);
}

.subtask-running {
  border-left-color: var(--el-color-warning);
}

.machine-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 4px;
}

.status-running {
  color: var(--el-color-warning);
}

.status-success {
  color: var(--el-color-success);
}

.status-failed {
  color: var(--el-color-danger);
}

.status-pending {
  color: var(--el-text-color-secondary);
}

.status-skipped {
  color: var(--el-text-color-secondary);
}
</style>
