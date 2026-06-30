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
          :class="getTaskClass(task.status, task.changed)"
        >
          <div class="log-task-header">
            <div class="log-task-status">
              <Icon :icon="getTaskIcon(task.status, task.changed)" :size="16" :style="{ color: getTaskIconColor(task.status, task.changed) }" />
            </div>
            <div class="log-task-info">
              <span class="log-task-ref">任务-{{ task.ref }}</span>
              <span class="log-task-name">{{ task.name || '未命名任务' }}</span>
              <span class="log-task-module">{{ task.module }}</span>
            </div>
            <div class="log-task-time" v-if="task.duration">
              {{ task.duration }}ms
            </div>
          </div>

          <!-- 任务输出（仅在没有机器数据时显示） -->
          <div class="log-task-output" v-if="(task.output || task.error) && (!task.machines || task.machines.length === 0)">
            <pre v-if="task.output" class="log-output">{{ task.output }}</pre>
            <pre v-if="task.error" class="log-error">{{ task.error }}</pre>
            <pre v-if="task.trace" class="log-trace">{{ task.trace }}</pre>
          </div>

          <!-- 机器执行详情 -->
          <div class="log-machines" v-if="task.machines && task.machines.length > 0">
            <div
              v-for="machine in task.machines"
              :key="machine.host"
              class="log-machine"
              :class="getMachineClass(machine.status, machine.changed)"
            >
              <div class="log-machine-header">
                <Icon :icon="getTaskIcon(machine.status, machine.changed)" :size="14" :style="{ color: getTaskIconColor(machine.status, machine.changed) }" />
                <span class="log-machine-host">{{ machine.host }}</span>
                <span class="log-machine-time" v-if="machine.duration">{{ machine.duration }}ms</span>
              </div>
              <div class="log-machine-output" v-if="machine.output || machine.error">
                <pre v-if="machine.output" class="log-output">{{ machine.output }}</pre>
                <pre v-if="machine.error" class="log-error">{{ machine.error }}</pre>
                <pre v-if="machine.trace" class="log-trace">{{ machine.trace }}</pre>
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
  trace?: string
  errorCode?: number
  changed?: boolean
  duration?: number
  machines?: MachineLog[]
}

interface MachineLog {
  host: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'skipped'
  output?: string
  error?: string
  trace?: string
  errorCode?: number
  changed?: boolean
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
    loadStageExecutionDetails(id)
  })

  eventSource.addEventListener('execution_status', (e) => {
    const data = JSON.parse(e.data)
    executionStatus.value = data.status
    if (data.status === 'success' || data.status === 'failed' || data.status === 'cancelled') {
      disconnectSSE()
      // 加载最终执行详情
      loadStageExecutionDetails(id)
    }
  })

  eventSource.addEventListener('stage_status', (e) => {
    const data = JSON.parse(e.data)
    // 更新阶段状态
  })

  eventSource.addEventListener('task_status', (e) => {
    const data = JSON.parse(e.data)
    updateTaskStatus(data.ref || data.task_id, data.status, {
      ref: data.ref,
      taskName: data.task_name,
      output: data.output,
      error: data.error,
      trace: data.trace,
      errorCode: data.error_code,
      changed: data.changed,
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

// 更新任务状态（用 ref 做 key，同一任务在不同机器上有不同 taskId）
function updateTaskStatus(taskRef: number, status: string, data?: any) {
  let task = taskLogs.value.find(t => t.ref === taskRef)
  if (!task) {
    task = {
      taskId: taskRef,
      ref: data?.ref || taskRef,
      name: data?.taskName || `任务 ${taskRef}`,
      module: '',
      status: status as any,
      machines: [],
    }
    taskLogs.value.push(task)
  }
  
  // 更新任务级别状态（不设置 output，output 只在机器级别）
  task.status = status as any
  if (data) {
    if (data.taskName) task.name = data.taskName
    if (data.error) task.error = data.error
    if (data.trace) task.trace = data.trace
    if (data.errorCode) task.errorCode = data.errorCode
    if (data.changed !== undefined) task.changed = data.changed
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
      if (data.trace) machine.trace = data.trace
      if (data.errorCode) machine.errorCode = data.errorCode
      if (data.changed !== undefined) machine.changed = data.changed
      if (data.duration) machine.duration = data.duration
    }
  }
  scrollToBottom()
}

// 加载单阶段执行详情（处理快速完成的情况）
async function loadStageExecutionDetails(executionId: number) {
  try {
    const { getStageExecutionApi } = await import('@/api/stageTemplate')
    const exec = await getStageExecutionApi(executionId)
    if (!exec) return

    executionStatus.value = exec.status
    
    // 按 task_ref 分组收集任务执行记录
    const taskMap = new Map<number, any>()
    
    if (exec.task_executions) {
      for (const te of exec.task_executions) {
        const taskRef = te.task_ref
        if (!taskMap.has(taskRef)) {
          taskMap.set(taskRef, {
            taskId: te.task_id || taskRef,
            ref: taskRef,
            name: te.task_name || '',
            module: te.task_module || '',
            status: te.status,
            output: '',
            error: '',
            duration: 0,
            changed: te.changed,
            machines: []
          })
        }
        const task = taskMap.get(taskRef)!
        // 添加机器执行记录
        task.machines.push({
          host: te.host,
          status: te.status,
          output: te.output,
          error: te.error,
          trace: te.trace,
          errorCode: te.error_code,
          changed: te.changed,
          duration: te.duration_ms,
        })
        // 更新任务状态
        if (te.status === 'failed') {
          task.status = 'failed'
          task.error = te.error
          task.trace = te.trace
        } else if (te.status === 'success' && task.status !== 'failed') {
          task.status = 'success'
          task.changed = task.changed || te.changed
        }
        task.duration = Math.max(task.duration, te.duration_ms || 0)
      }
    }
    
    // 更新任务日志（合并已有数据）
    for (const [taskRef, task] of taskMap) {
      const existing = taskLogs.value.find(t => t.ref === taskRef)
      if (existing) {
        existing.status = task.status
        existing.changed = task.changed
        existing.duration = task.duration
        if (task.error) existing.error = task.error
        if (task.trace) existing.trace = task.trace
        // 合并机器记录
        for (const machine of task.machines) {
          const existingMachine = existing.machines?.find(m => m.host === machine.host)
          if (existingMachine) {
            Object.assign(existingMachine, machine)
          } else {
            if (!existing.machines) existing.machines = []
            existing.machines.push(machine)
          }
        }
      } else {
        taskLogs.value.push(task)
      }
    }
  } catch (error) {
    console.error('Failed to load stage execution details:', error)
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

function getTaskClass(status: string, changed?: boolean) {
  if (status === 'success' && changed === false) return 'log-task--unchanged'
  return `log-task--${status}`
}

function getMachineClass(status: string, changed?: boolean) {
  if (status === 'success' && changed === false) return 'log-machine--unchanged'
  return `log-machine--${status}`
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

// 获取图标颜色
function getTaskIconColor(status: string, changed?: boolean): string {
  if (status === 'success' && changed === false) return '#f5d76e' // 柔和黄
  const map: Record<string, string> = {
    running: '#e6a23c',
    success: '#67c23a',
    failed: '#f56c6c',
    pending: '#909399',
    skipped: '#909399',
  }
  return map[status] || '#909399'
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

.log-task--unchanged {
  border-color: #f5d76e;
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

.log-task-ref {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
  background: rgba(255, 255, 255, 0.1);
  padding: 1px 6px;
  border-radius: 4px;
  font-weight: 500;
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

.log-machine--unchanged .log-machine-header {
  background: rgba(245, 215, 110, 0.15);
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

.log-trace {
  margin: 0;
  color: #e6a23c;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  opacity: 0.8;
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
