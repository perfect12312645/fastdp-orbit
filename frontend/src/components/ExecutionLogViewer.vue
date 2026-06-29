<template>
  <el-dialog
    v-model="visible"
    title="执行日志"
    fullscreen
    :close-on-click-modal="false"
    :close-on-press-escape="!isRunning"
    @close="handleClose"
  >
    <template #header>
      <div class="log-header">
        <div class="log-header-left">
          <Icon icon="mdi:console" :size="20" />
          <span class="log-title">执行日志</span>
          <el-tag :type="getStatusType(executionStatus)" size="small">
            {{ getStatusLabel(executionStatus) }}
          </el-tag>
        </div>
        <div class="log-header-right">
          <el-button v-if="isRunning" type="danger" size="small" @click="handleCancel">
            <Icon icon="mdi:stop" :size="14" /> 终止
          </el-button>
          <el-button size="small" @click="visible = false">
            {{ isRunning ? '最小化' : '关闭' }}
          </el-button>
        </div>
      </div>
    </template>

    <div class="log-content" ref="logContainer">
      <!-- 执行信息 -->
      <div class="log-info">
        <div class="log-info-item">
          <span class="label">阶段：</span>
          <span>{{ stageName }}</span>
        </div>
        <div class="log-info-item" v-if="machineGroupName">
          <span class="label">机器分组：</span>
          <span>{{ machineGroupName }}</span>
        </div>
        <div class="log-info-item" v-if="startTime">
          <span class="label">开始时间：</span>
          <span>{{ formatDateTime(startTime) }}</span>
        </div>
      </div>

      <!-- 任务执行日志 -->
      <div class="log-tasks">
        <div
          v-for="(task, index) in taskLogs"
          :key="task.taskId"
          class="log-task"
          :class="getTaskClass(task.status)"
        >
          <div class="log-task-header">
            <div class="log-task-status">
              <Icon :icon="getTaskIcon(task.status)" :size="16" />
            </div>
            <div class="log-task-info">
              <span class="log-task-name">{{ task.name || `任务 ${task.ref}` }}</span>
              <span class="log-task-module">{{ task.module }}</span>
            </div>
            <div class="log-task-time" v-if="task.duration">
              {{ task.duration }}ms
            </div>
          </div>

          <!-- 任务输出 -->
          <div class="log-task-output" v-if="task.output || task.error">
            <pre v-if="task.output" class="log-output">{{ task.output }}</pre>
            <pre v-if="task.error" class="log-error">{{ task.error }}</pre>
          </div>

          <!-- 机器执行详情 -->
          <div class="log-machines" v-if="task.machines && task.machines.length > 0">
            <div
              v-for="machine in task.machines"
              :key="machine.host"
              class="log-machine"
              :class="getMachineClass(machine.status)"
            >
              <div class="log-machine-header">
                <Icon :icon="getTaskIcon(machine.status)" :size="14" />
                <span class="log-machine-host">{{ machine.host }}</span>
                <span class="log-machine-time" v-if="machine.duration">{{ machine.duration }}ms</span>
              </div>
              <div class="log-machine-output" v-if="machine.output || machine.error">
                <pre v-if="machine.output" class="log-output">{{ machine.output }}</pre>
                <pre v-if="machine.error" class="log-error">{{ machine.error }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="taskLogs.length === 0 && !isRunning" class="log-empty">
        <Icon icon="mdi:console-line" :size="48" />
        <p>暂无执行日志</p>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDateTime } from '@/utils/format'

interface TaskLog {
  taskId: number
  ref: number
  name: string
  module: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'skipped'
  output?: string
  error?: string
  duration?: number
  machines?: MachineLog[]
}

interface MachineLog {
  host: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'skipped'
  output?: string
  error?: string
  duration?: number
}

const props = defineProps<{
  executionId: number
  stageName: string
  machineGroupName?: string
}>()

const visible = ref(false)
const executionStatus = ref('pending')
const startTime = ref('')
const taskLogs = ref<TaskLog[]>([])
const logContainer = ref<HTMLElement>()
let eventSource: EventSource | null = null

const isRunning = computed(() => 
  executionStatus.value === 'running' || executionStatus.value === 'paused'
)

// 打开日志对话框
function open(executionId?: number) {
  if (executionId) {
    // 使用传入的 executionId
  }
  visible.value = true
  executionStatus.value = 'running'
  startTime.value = new Date().toISOString()
  taskLogs.value = []
  connectSSE(executionId || props.executionId)
}

// 关闭对话框
function handleClose() {
  if (!isRunning.value) {
    disconnectSSE()
  }
}

// 取消执行
async function handleCancel() {
  try {
    await ElMessageBox.confirm('确认终止执行？', '终止确认', {
      confirmButtonText: '终止',
      cancelButtonText: '取消',
      type: 'warning',
    })
    // 调用取消 API
    // await cancelExecutionApi(...)
    ElMessage.success('已发送终止请求')
  } catch {
    // 取消
  }
}

// 连接 SSE
function connectSSE(executionId?: number) {
  disconnectSSE()
  const id = executionId || props.executionId
  if (!id) return
  
  const url = `/api/v1/executions/${id}/stream`
  eventSource = new EventSource(url)

  eventSource.addEventListener('connected', () => {
    console.log('[SSE] connected to execution', id)
    // 加载执行详情（处理快速完成的情况）
    loadExecutionDetails(id)
  })

  eventSource.addEventListener('execution_status', (e) => {
    const data = JSON.parse(e.data)
    executionStatus.value = data.status
    if (data.status === 'success' || data.status === 'failed' || data.status === 'cancelled') {
      disconnectSSE()
      // 加载最终执行详情
      loadExecutionDetails(id)
    }
  })

  eventSource.addEventListener('stage_status', (e) => {
    const data = JSON.parse(e.data)
    // 更新阶段状态
  })

  eventSource.addEventListener('task_status', (e) => {
    const data = JSON.parse(e.data)
    updateTaskStatus(data.task_id, data.status, {
      output: data.output,
      error: data.error,
      duration: data.duration_ms,
      host: data.host,
    })
  })

  eventSource.onerror = () => {
    if (eventSource?.readyState === EventSource.CLOSED) {
      console.warn('[SSE] connection closed')
      disconnectSSE()
    }
  }
}

// 断开 SSE
function disconnectSSE() {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
}

// 更新任务状态
function updateTaskStatus(taskId: number, status: string, data?: any) {
  let task = taskLogs.value.find(t => t.taskId === taskId)
  if (!task) {
    // 创建新任务条目
    task = {
      taskId: taskId,
      ref: 0,
      name: `任务 ${taskId}`,
      module: '',
      status: status as any,
      machines: [],
    }
    taskLogs.value.push(task)
  }
  
  task.status = status as any
  if (data) {
    if (data.output) task.output = data.output
    if (data.error) task.error = data.error
    if (data.duration) task.duration = data.duration
    
    // 更新或创建机器级别条目
    if (data.host) {
      let machine = task.machines?.find(m => m.host === data.host)
      if (!machine) {
        machine = { host: data.host, status: status as any }
        if (!task.machines) task.machines = []
        task.machines.push(machine)
      }
      machine.status = status as any
      if (data.output) machine.output = data.output
      if (data.error) machine.error = data.error
      if (data.duration) machine.duration = data.duration
    }
  }
  scrollToBottom()
}

// 加载执行详情（处理快速完成的情况）
async function loadExecutionDetails(executionId: number) {
  try {
    const { getExecutionApi } = await import('@/api/workflow')
    // 需要一个 workflowId，但单阶段执行没有 workflowId
    // 使用通用的执行详情 API
    const response = await fetch(`/api/v1/executions/${executionId}`)
    const result = await response.json()
    if (result.code === 0 && result.data) {
      const exec = result.data
      executionStatus.value = exec.status
      
      // 加载任务执行详情
      if (exec.stage_group_executions) {
        for (const sge of exec.stage_group_executions) {
          if (sge.stage_executions) {
            for (const se of sge.stage_executions) {
              if (se.task_executions) {
                for (const te of se.task_executions) {
                  const existingTask = taskLogs.value.find(t => t.taskId === te.task_id)
                  if (!existingTask) {
                    taskLogs.value.push({
                      taskId: te.task_id,
                      ref: te.task?.ref || 0,
                      name: te.task?.name || '',
                      module: te.task?.module || '',
                      status: te.status,
                      output: te.output,
                      error: te.error,
                      duration: te.duration_ms,
                      machines: [{
                        host: te.host,
                        status: te.status,
                        output: te.output,
                        error: te.error,
                        duration: te.duration_ms,
                      }]
                    })
                  }
                }
              }
            }
          }
        }
      }
    }
  } catch (error) {
    console.error('Failed to load execution details:', error)
  }
}

// 滚动到底部
function scrollToBottom() {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

// 状态样式
function getStatusType(status: string) {
  const map: Record<string, string> = {
    running: 'warning',
    success: 'success',
    failed: 'danger',
    cancelled: 'info',
    pending: 'info',
  }
  return (map[status] || 'info') as any
}

function getStatusLabel(status: string) {
  const map: Record<string, string> = {
    running: '执行中',
    success: '成功',
    failed: '失败',
    cancelled: '已终止',
    pending: '等待中',
  }
  return map[status] || status
}

function getTaskClass(status: string) {
  return `log-task--${status}`
}

function getMachineClass(status: string) {
  return `log-machine--${status}`
}

function getTaskIcon(status: string) {
  const map: Record<string, string> = {
    running: 'mdi:loading',
    success: 'mdi:check-circle',
    failed: 'mdi:close-circle',
    pending: 'mdi:clock-outline',
    skipped: 'mdi:skip-next',
  }
  return map[status] || 'mdi:circle-outline'
}

// 暴露方法
defineExpose({
  open,
})
</script>

<style scoped>
.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.log-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.log-title {
  font-size: 16px;
  font-weight: 600;
}

.log-content {
  height: calc(100vh - 120px);
  overflow-y: auto;
  padding: 16px;
  background: #1e1e1e;
  border-radius: 8px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.log-info {
  display: flex;
  gap: 24px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.log-info-item {
  display: flex;
  align-items: center;
  gap: 4px;
  color: rgba(255, 255, 255, 0.7);
}

.log-info-item .label {
  color: rgba(255, 255, 255, 0.5);
}

.log-tasks {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.log-task {
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  overflow: hidden;
  transition: border-color 0.2s;
}

.log-task--running {
  border-color: #e6a23c;
}

.log-task--success {
  border-color: #67c23a;
}

.log-task--failed {
  border-color: #f56c6c;
}

.log-task-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.05);
}

.log-task-status {
  flex-shrink: 0;
}

.log-task-info {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
}

.log-task-name {
  font-weight: 500;
  color: rgba(255, 255, 255, 0.9);
}

.log-task-module {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.5);
  background: rgba(255, 255, 255, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
}

.log-task-time {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
}

.log-task-output {
  padding: 8px 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.log-machines {
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.log-machine {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.log-machine:last-child {
  border-bottom: none;
}

.log-machine-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px 6px 24px;
  background: rgba(255, 255, 255, 0.02);
}

.log-machine--success .log-machine-header {
  background: rgba(103, 194, 58, 0.1);
}

.log-machine--failed .log-machine-header {
  background: rgba(245, 108, 108, 0.1);
}

.log-machine-host {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
}

.log-machine-time {
  margin-left: auto;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.4);
}

.log-machine-output {
  padding: 6px 12px 6px 36px;
}

.log-output {
  margin: 0;
  color: rgba(255, 255, 255, 0.8);
  white-space: pre-wrap;
  word-break: break-all;
}

.log-error {
  margin: 0;
  color: #f56c6c;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: rgba(255, 255, 255, 0.3);
  gap: 12px;
}
</style>
