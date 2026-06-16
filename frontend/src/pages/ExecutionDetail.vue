<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <el-button link @click="goBack">
          <Icon icon="mdi:arrow-left" :size="18" /> 返回
        </el-button>
        <h2>执行详情</h2>
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

        <!-- 阶段执行详情 -->
        <div class="stages-container">
          <div
            v-for="(se, sei) in execution.stage_executions"
            :key="se.id"
            class="stage-card"
          >
            <div class="stage-header">
              <div class="stage-status">
                <Icon :icon="getStageIcon(se.status)" :size="20" :class="'status-' + se.status" />
              </div>
              <div class="stage-info">
                <span class="stage-name">阶段 {{ sei + 1 }}: {{ se.stage?.name || `Stage #${se.stage_id}` }}</span>
                <span class="stage-time" v-if="se.started_at">
                  {{ formatDateTime(se.started_at) }}
                  <template v-if="se.finished_at"> ~ {{ formatDateTime(se.finished_at) }}</template>
                </span>
              </div>
              <el-tag :type="getStatusType(se.status)" size="small">
                {{ getStatusLabel(se.status) }}
              </el-tag>
            </div>

            <!-- 任务列表 -->
            <div class="tasks-container" v-if="se.task_executions?.length">
              <div
                v-for="(te, tei) in se.task_executions"
                :key="te.id"
                class="task-item"
              >
                <div class="task-header">
                  <Icon :icon="getTaskIcon(te.status)" :size="16" :class="'status-' + te.status" />
                  <span class="task-name">{{ te.task?.name || `Task #${te.task_id}` }}</span>
                  <span class="task-host">{{ te.task?.host }}</span>
                  <span class="task-duration" v-if="te.duration_ms">{{ te.duration_ms }}ms</span>
                  <el-tag :type="getStatusType(te.status)" size="small">
                    {{ getStatusLabel(te.status) }}
                  </el-tag>
                </div>
                <div class="task-output" v-if="te.output">
                  <pre>{{ te.output }}</pre>
                </div>
                <div class="task-error" v-if="te.error">
                  <pre>{{ te.error }}</pre>
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
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { getExecutionApi } from '@/api/workflow'
import { formatDateTime } from '@/utils/format'
import type { WorkflowExecution } from '@/types/workflow'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const execution = ref<WorkflowExecution | null>(null)

async function loadExecution() {
  loading.value = true
  try {
    const workflowId = Number(route.params.id)
    const executionId = Number(route.params.eid)
    execution.value = await getExecutionApi(workflowId, executionId)
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
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

function getTaskIcon(status: string) {
  const map: Record<string, string> = {
    running: 'mdi:loading',
    success: 'mdi:check',
    failed: 'mdi:close',
    pending: 'mdi:circle-outline',
  }
  return map[status] || 'mdi:circle-outline'
}

onMounted(loadExecution)
</script>

<style scoped>
.page-container {
  padding: 24px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 8px 0 0;
  font-size: 22px;
  font-weight: 600;
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
.task-error {
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

.task-output pre,
.task-error pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
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
