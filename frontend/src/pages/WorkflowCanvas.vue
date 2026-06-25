<template>
  <div class="canvas-page">
    <!-- 顶部工具栏 -->
    <div class="canvas-toolbar">
      <div class="toolbar-left">
        <el-button @click="goBack" text>
          <Icon icon="mdi:arrow-left" :size="18" /> 返回
        </el-button>
        <el-divider direction="vertical" />
        <template v-if="editMode">
          <el-input
            v-model="workflow.name"
            placeholder="工作流名称"
            class="workflow-name-input"
          />
        </template>
        <template v-else>
          <span class="workflow-title">{{ workflow.name }}</span>
        </template>
        <template v-if="!editMode && lastSaved">
          <span class="save-status saved">已保存 {{ lastSaved }}</span>
        </template>
      </div>
      <div class="toolbar-center">
        <!-- 执行状态栏 -->
        <div v-if="!editMode && currentExecution" class="execution-status-bar">
          <el-tag :type="getExecStatusType(currentExecution.status)" effect="dark" size="small">
            <Icon :icon="getExecStatusIcon(currentExecution.status)" :size="12" class="spin-icon" />
            {{ getExecStatusLabel(currentExecution.status) }}
          </el-tag>
          <span v-if="currentExecution.status === 'running' || currentExecution.status === 'paused'" class="exec-progress">
            {{ executionProgress }}
          </span>
          <span v-if="currentExecution.error" class="exec-error">{{ currentExecution.error }}</span>
        </div>
      </div>
      <div class="toolbar-right">
        <template v-if="editMode">
          <el-button @click="exitEditMode">
            <Icon icon="mdi:cancel" :size="16" /> 取消
          </el-button>
          <el-button type="primary" @click="handleSave" :loading="saving">
            <Icon icon="mdi:content-save" :size="16" /> 保存
          </el-button>
        </template>
        <template v-else>
          <el-button @click="enterEditMode" :disabled="!canEdit">
            <Icon icon="mdi:pencil" :size="16" /> 编排
          </el-button>
          <el-button @click="showExecHistory = true">
            <Icon icon="mdi:history" :size="16" /> 历史
          </el-button>
          <!-- 执行中：显示暂停/终止 -->
          <template v-if="currentExecution?.status === 'running'">
            <el-button type="warning" @click="handlePause">
              <Icon icon="mdi:pause" :size="16" /> 暂停
            </el-button>
            <el-button type="danger" @click="handleCancel">
              <Icon icon="mdi:stop" :size="16" /> 终止
            </el-button>
          </template>
          <!-- 已暂停：显示继续/终止 -->
          <template v-else-if="currentExecution?.status === 'paused'">
            <el-button type="success" @click="handleResume">
              <Icon icon="mdi:play" :size="16" /> 继续
            </el-button>
            <el-button type="danger" @click="handleCancel">
              <Icon icon="mdi:stop" :size="16" /> 终止
            </el-button>
          </template>
          <!-- 无执行或已结束：显示执行按钮 -->
          <template v-else>
            <el-button type="success" @click="handleExecute" :disabled="!canExecute">
              <Icon icon="mdi:play" :size="16" /> 执行
            </el-button>
          </template>
        </template>
      </div>
    </div>

    <div class="canvas-body">
      <!-- 画布区域 -->
      <div class="canvas-area" ref="canvasArea">
        <div class="canvas-columns">
          <div
            v-for="(group, gi) in workflow.stage_groups"
            :key="gi"
            class="canvas-column"
            :class="{ 'column-running': isGroupRunning(gi), 'column-success': isGroupSuccess(gi), 'column-failed': isGroupFailed(gi) }"
            @dragover.prevent="editMode && $event.preventDefault()"
            @drop="editMode && onDropToGroup($event, gi)"
          >
            <div class="column-header">
              <template v-if="editMode">
                <el-input
                  v-model="group.name"
                  placeholder="阶段组名称"
                  class="column-name-input"
                />
              </template>
              <template v-else>
                <span class="column-name">{{ group.name }}</span>
                <Icon
                  v-if="getGroupStatus(gi)"
                  :icon="getExecStatusIcon(getGroupStatus(gi)!)"
                  :size="16"
                  :class="'status-' + getGroupStatus(gi)"
                  class="group-status-icon"
                />
              </template>
              <div class="column-actions" v-if="editMode">
                <el-dropdown trigger="click">
                  <el-button link size="small">
                    <Icon icon="mdi:dots-vertical" :size="16" />
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item @click="toggleGroupMode(gi)">
                        <Icon :icon="group.mode === 'parallel' ? 'mdi:swap-horizontal' : 'mdi:swap-vertical'" :size="14" />
                        {{ group.mode === 'parallel' ? '切换为顺序执行' : '切换为并行执行' }}
                      </el-dropdown-item>
                      <el-dropdown-item divided @click="removeGroup(gi)" class="danger-item">
                        <Icon icon="mdi:delete-outline" :size="14" /> 删除阶段组
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </div>
            <div class="column-mode">
              <el-tag size="small" :type="group.mode === 'parallel' ? 'warning' : 'info'" effect="plain">
                {{ group.mode === 'parallel' ? '并行' : '顺序' }}
              </el-tag>
            </div>
            <div class="column-stages">
              <template v-if="editMode">
                <draggable
                  v-model="group.stages"
                  group="stages"
                  item-key="id"
                  class="stage-drop-zone"
                  ghost-class="stage-ghost"
                  :animation="200"
                  @change="onStageReorder(gi)"
                >
                  <template #item="{ element: stage, index: si }">
                    <div class="stage-card">
                      <div class="stage-card-header">
                        <span class="stage-card-name">{{ stage.name || '未命名' }}</span>
                        <el-button
                          type="danger"
                          link
                          size="small"
                          @click="removeStage(gi, si)"
                        >
                          <Icon icon="mdi:close" :size="14" />
                        </el-button>
                      </div>
                      <div class="stage-card-meta">
                        <el-tag size="small" effect="plain">{{ stage.machine_group_name || '未指定' }}</el-tag>
                        <span class="stage-task-count">{{ stage.tasks.length }} 个任务</span>
                        <el-tag v-if="stage.template_version" size="small" type="info" effect="plain" class="version-tag">
                          {{ stage.template_version }}
                        </el-tag>
                      </div>
                    </div>
                  </template>
                </draggable>
                <div class="add-stage-hint" @click="showAddStageDialog(gi)">
                  <Icon icon="mdi:plus" :size="16" /> 添加阶段
                </div>
              </template>
              <template v-else>
                <div
                  v-for="(stage, si) in group.stages"
                  :key="si"
                  class="stage-card stage-card-view"
                  :class="getStageCardClass(gi, si)"
                  @click="openStageDetail(gi, si)"
                >
                  <div class="stage-card-header">
                    <Icon :icon="getStageStatusIcon(gi, si)" :size="16" :class="'status-' + (getStageStatus(gi, si) || 'pending')" />
                    <span class="stage-card-name">{{ stage.name || '未命名' }}</span>
                    <el-tag v-if="stage.template_version" size="small" type="info" effect="plain" class="version-tag">
                      {{ stage.template_version }}
                    </el-tag>
                    <el-button
                      v-if="getStageStatus(gi, si) === 'failed'"
                      type="warning"
                      link
                      size="small"
                      @click.stop="handleRetryStage(getStageExecId(gi, si))"
                    >
                      <Icon icon="mdi:restart" :size="14" />
                    </el-button>
                  </div>
                  <div class="stage-card-meta">
                    <el-tag size="small" effect="plain">{{ stage.machine_group_name || '未指定' }}</el-tag>
                    <span class="stage-task-count">{{ stage.tasks.length }} 个任务</span>
                    <span v-if="getStageDuration(gi, si)" class="stage-duration">{{ getStageDuration(gi, si) }}</span>
                  </div>
                </div>
              </template>
            </div>
          </div>

          <!-- 添加阶段组按钮 (编辑模式) -->
          <div v-if="editMode" class="add-column" @click="addGroup">
            <Icon icon="mdi:plus" :size="24" />
            <span>添加阶段组</span>
          </div>
        </div>

        <!-- 执行历史 (非编辑模式，无当前执行时显示最近历史) -->
        <div v-if="!editMode && !currentExecution && recentExecutions.length > 0" class="recent-executions">
          <h4>最近执行</h4>
          <div class="exec-list">
            <div
              v-for="exec in recentExecutions"
              :key="exec.id"
              class="exec-item"
              @click="goToExecution(exec.id)"
            >
              <el-tag :type="getExecStatusType(exec.status)" size="small">{{ getExecStatusLabel(exec.status) }}</el-tag>
              <span class="exec-trigger">{{ exec.trigger || 'system' }}</span>
              <span class="exec-time">{{ formatDateTime(exec.started_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧阶段模板面板 (仅编辑模式) -->
      <div v-if="editMode" class="stage-palette" :class="{ collapsed: paletteCollapsed }">
        <div class="palette-toggle" @click="paletteCollapsed = !paletteCollapsed">
          <Icon :icon="paletteCollapsed ? 'mdi:chevron-left' : 'mdi:chevron-right'" :size="16" />
        </div>
        <template v-if="!paletteCollapsed">
          <div class="palette-header">
            <span>阶段模板</span>
            <el-button type="primary" size="small" text @click="refreshTemplates">
              <Icon icon="mdi:refresh" :size="14" />
            </el-button>
          </div>
          <div class="palette-search">
            <el-input v-model="templateSearch" placeholder="搜索" clearable size="small">
              <template #prefix>
                <Icon icon="mdi:magnify" :size="14" />
              </template>
            </el-input>
          </div>
          <div class="palette-list">
            <div
              v-for="tpl in filteredTemplates"
              :key="tpl.id"
              class="palette-item"
              draggable="true"
              @dragstart="onTemplateDragStart($event, tpl)"
            >
              <Icon icon="mdi:view-column-outline" :size="16" />
              <div class="palette-item-info">
                <span class="palette-item-name">{{ tpl.name }}</span>
                <span class="palette-item-meta">{{ tpl.version }}</span>
              </div>
            </div>
            <div v-if="filteredTemplates.length === 0" class="palette-empty">
              暂无模板，请先在阶段管理中创建
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- 执行历史对话框 -->
    <el-dialog v-model="showExecHistory" title="执行历史" width="700px" destroy-on-close @open="loadExecHistory">
      <el-table :data="execHistory" v-loading="loadingExecHistory" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getExecStatusType(row.status)" size="small">{{ getExecStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="trigger" label="触发者" width="100" />
        <el-table-column prop="error" label="错误信息" min-width="150" show-overflow-tooltip />
        <el-table-column prop="started_at" label="开始时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.started_at) }}</template>
        </el-table-column>
        <el-table-column prop="finished_at" label="结束时间" width="170">
          <template #default="{ row }">{{ row.finished_at ? formatDateTime(row.finished_at) : '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="goToExecution(row.id)">
              <Icon icon="mdi:eye" :size="14" /> 详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 阶段详情抽屉 -->
    <el-drawer
      v-model="showStageDetail"
      :title="stageDetailTitle"
      size="500px"
      destroy-on-close
    >
      <template v-if="stageDetailData">
        <div class="stage-detail-section">
          <div class="detail-header">
            <el-tag :type="getExecStatusType(stageDetailData.status)" effect="dark">
              {{ getExecStatusLabel(stageDetailData.status) }}
            </el-tag>
            <span v-if="stageDetailData.error" class="detail-error">{{ stageDetailData.error }}</span>
          </div>
          <div v-if="stageDetailData.task_executions?.length" class="task-list">
            <div
              v-for="te in stageDetailData.task_executions"
              :key="te.id"
              class="task-detail-card"
            >
              <div class="task-detail-header">
                <Icon :icon="getTaskIcon(te.status)" :size="14" :class="'status-' + te.status" />
                <span class="task-detail-name">{{ te.task?.name || `Task #${te.task_id}` }}</span>
                <span class="task-detail-host">{{ te.host }}</span>
                <el-tag :type="getExecStatusType(te.status)" size="small">{{ getExecStatusLabel(te.status) }}</el-tag>
              </div>
              <div v-if="te.duration_ms" class="task-detail-duration">
                耗时 {{ te.duration_ms }}ms
              </div>
              <div v-if="te.output" class="task-output">
                <div class="output-label">输出</div>
                <pre>{{ te.output }}</pre>
              </div>
              <div v-if="te.error" class="task-error">
                <div class="output-label">错误</div>
                <pre>{{ te.error }}</pre>
              </div>
            </div>
          </div>
          <div v-else class="no-data">暂无任务执行数据</div>
        </div>
      </template>
    </el-drawer>

    <!-- 手动添加阶段对话框 -->
    <el-dialog v-model="showAddStage" title="添加阶段" width="500px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item label="阶段名称">
          <el-input v-model="newStage.name" placeholder="如：安装 Docker" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="newStage.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="目标分组">
          <el-select v-model="newStage.machine_group_id" placeholder="选择机器分组" filterable style="width: 100%">
            <el-option
              v-for="g in machineGroups"
              :key="g.id"
              :label="g.name"
              :value="g.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddStage = false">取消</el-button>
        <el-button type="primary" @click="confirmAddStage">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, triggerRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import draggable from 'vuedraggable'
import {
  getWorkflowApi,
  updateWorkflowApi,
  executeWorkflowApi,
  getExecutionsApi,
  getExecutionApi,
  pauseExecutionApi,
  resumeExecutionApi,
  cancelExecutionApi,
  retryStageApi,
} from '@/api/workflow'
import { getMachineGroupsApi, type MachineGroup } from '@/api/machineGroup'
import { getStageTemplatesApi, type StageTemplate as StageTemplateApi } from '@/api/stageTemplate'
import { getHookTemplatesApi, type HookTemplate } from '@/api/hookTemplate'
import { formatDateTime } from '@/utils/format'
import type {
  WorkflowStageGroup,
  WorkflowStage,
  WorkflowExecution,
  WorkflowStageGroupExecution,
  WorkflowStageExecution,
} from '@/types/workflow'

const route = useRoute()
const router = useRouter()
const workflowId = computed(() => Number(route.params.id))

// ==================== 模式控制 ====================
const editMode = ref(false)
const paletteCollapsed = ref(false)

// ==================== 数据状态 ====================
const saving = ref(false)
const lastSaved = ref('')
const showAddStage = ref(false)
const templateSearch = ref('')
const machineGroups = ref<MachineGroup[]>([])
const templates = ref<StageTemplateApi[]>([])
const hookTemplates = ref<HookTemplate[]>([])

const workflow = ref({
  name: '',
  description: '',
  stage_groups: [] as (WorkflowStageGroup & { stages: WorkflowStage[] })[],
})

// ==================== 执行状态 ====================
const currentExecution = ref<WorkflowExecution | null>(null)
const recentExecutions = ref<WorkflowExecution[]>([])
let eventSource: EventSource | null = null

const canExecute = computed(() => {
  if (!workflow.value.stage_groups.length) return false
  return !currentExecution.value || !['running', 'paused'].includes(currentExecution.value.status)
})

const canEdit = computed(() => {
  return !currentExecution.value || !['running', 'paused'].includes(currentExecution.value.status)
})

const executionProgress = computed(() => {
  if (!currentExecution.value) return ''
  const groups = currentExecution.value.stage_group_executions || []
  const completed = groups.filter((g) => g.status === 'success' || g.status === 'failed' || g.status === 'skipped').length
  return `${completed}/${groups.length} 阶段组`
})

// ==================== 执行状态映射 ====================
// key: stage_id -> WorkflowStageExecution
const stageExecMap = computed(() => {
  const map = new Map<number, WorkflowStageExecution>()
  if (!currentExecution.value) return map
  for (const sge of currentExecution.value.stage_group_executions || []) {
    for (const se of sge.stage_executions || []) {
      map.set(se.stage_id, se)
      void se.status
    }
  }
  return map
})

// key: group_id -> WorkflowStageGroupExecution
const groupExecMap = computed(() => {
  const map = new Map<number, WorkflowStageGroupExecution>()
  if (!currentExecution.value) return map
  for (const sge of currentExecution.value.stage_group_executions || []) {
    map.set(sge.group_id, sge)
    void sge.status
  }
  return map
})

function getStageStatus(gi: number, si: number): string | null {
  const stage = workflow.value.stage_groups[gi]?.stages[si]
  if (!stage?.id) return null
  return stageExecMap.value.get(stage.id)?.status || null
}

function getStageExecId(gi: number, si: number): number | null {
  const stage = workflow.value.stage_groups[gi]?.stages[si]
  if (!stage?.id) return null
  return stageExecMap.value.get(stage.id)?.id || null
}

function getStageDuration(gi: number, si: number): string | null {
  const stage = workflow.value.stage_groups[gi]?.stages[si]
  if (!stage?.id) return null
  const exec = stageExecMap.value.get(stage.id)
  if (!exec?.started_at) return null
  const start = new Date(exec.started_at).getTime()
  const end = exec.finished_at ? new Date(exec.finished_at).getTime() : Date.now()
  const sec = Math.round((end - start) / 1000)
  if (sec < 60) return `${sec}s`
  return `${Math.floor(sec / 60)}m${sec % 60}s`
}

function getGroupStatus(gi: number): string | null {
  const group = workflow.value.stage_groups[gi]
  if (!group?.id) return null
  return groupExecMap.value.get(group.id)?.status || null
}

function isGroupRunning(gi: number) { return getGroupStatus(gi) === 'running' }
function isGroupSuccess(gi: number) { return getGroupStatus(gi) === 'success' }
function isGroupFailed(gi: number) { return getGroupStatus(gi) === 'failed' }

function getStageCardClass(gi: number, si: number): string {
  const status = getStageStatus(gi, si)
  if (status) return `stage-${status}`
  return ''
}

function getStageStatusIcon(gi: number, si: number): string {
  const status = getStageStatus(gi, si)
  if (!status) return 'mdi:circle-outline'
  const map: Record<string, string> = {
    running: 'mdi:loading',
    success: 'mdi:check-circle',
    failed: 'mdi:close-circle',
    pending: 'mdi:clock-outline',
    skipped: 'mdi:skip-next',
  }
  return map[status] || 'mdi:circle-outline'
}

// ==================== 状态标签 ====================
function getExecStatusType(status: string) {
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

function getExecStatusLabel(status: string) {
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

function getExecStatusIcon(status: string) {
  const map: Record<string, string> = {
    running: 'mdi:loading',
    success: 'mdi:check-circle',
    failed: 'mdi:close-circle',
    paused: 'mdi:pause-circle',
    cancelled: 'mdi:stop-circle',
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

// ==================== 模式切换 ====================
function enterEditMode() {
  if (!canEdit.value) {
    ElMessage.warning('工作流正在执行中，无法编排')
    return
  }
  editMode.value = true
}

function exitEditMode() {
  editMode.value = false
  loadData()
}

// ==================== 执行操作 ====================
async function handleExecute() {
  try {
    await ElMessageBox.confirm(`确认执行工作流「${workflow.value.name}」？`, '执行确认', {
      confirmButtonText: '执行',
      cancelButtonText: '取消',
      type: 'warning',
    })
    const res = await executeWorkflowApi(workflowId.value) as any
    ElMessage.success('工作流已触发执行')
    if (res?.data?.execution_id) {
      await loadExecutionById(res.data.execution_id)
      connectSSE(res.data.execution_id)
    } else {
      await loadLatestExecution()
      if (currentExecution.value) {
        connectSSE(currentExecution.value.id)
      }
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('执行失败')
  }
}

async function handlePause() {
  if (!currentExecution.value) return
  try {
    await ElMessageBox.confirm('确认暂停执行？', '暂停确认', {
      confirmButtonText: '暂停',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await pauseExecutionApi(workflowId.value, currentExecution.value.id)
    ElMessage.success('已暂停')
    await loadExecutionById(currentExecution.value.id)
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('暂停失败')
  }
}

async function handleResume() {
  if (!currentExecution.value) return
  try {
    await resumeExecutionApi(workflowId.value, currentExecution.value.id)
    ElMessage.success('已恢复')
    await loadExecutionById(currentExecution.value.id)
    connectSSE(currentExecution.value.id)
  } catch (e) {
    ElMessage.error('恢复失败')
  }
}

async function handleCancel() {
  if (!currentExecution.value) return
  try {
    await ElMessageBox.confirm('确认终止执行？终止后不可恢复。', '终止确认', {
      confirmButtonText: '终止',
      cancelButtonText: '取消',
      type: 'error',
    })
    await cancelExecutionApi(workflowId.value, currentExecution.value.id)
    ElMessage.success('已终止')
    await loadExecutionById(currentExecution.value.id)
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('终止失败')
  }
}

async function handleRetryStage(stageExecId: number | null) {
  if (!currentExecution.value || !stageExecId) return
  try {
    await retryStageApi(workflowId.value, currentExecution.value.id, stageExecId)
    ElMessage.success('已重试该阶段')
    await loadExecutionById(currentExecution.value.id)
    connectSSE(currentExecution.value.id)
  } catch (e) {
    ElMessage.error('重试失败')
  }
}

function goToExecution(execId: number) {
  router.push(`/workflow/${workflowId.value}/executions/${execId}`)
}

// ==================== SSE 实时推送 ====================
let nextTempId = -1

function findGroupIdForStage(stageId: number): number | null {
  for (const group of workflow.value.stage_groups) {
    if (group.id && group.stages.some(s => s.id === stageId)) {
      return group.id
    }
  }
  return null
}

function ensureGroupInExecution(groupId: number, status: string): WorkflowStageGroupExecution[] {
  const groups = currentExecution.value?.stage_group_executions || []
  const idx = groups.findIndex(g => g.group_id === groupId)
  if (idx >= 0) return groups
  const newGroup: WorkflowStageGroupExecution = {
    id: nextTempId--,
    execution_id: currentExecution.value?.id || 0,
    group_id: groupId,
    status: status as any,
    error: '',
    started_at: null,
    finished_at: null,
    stage_executions: [],
  }
  return [...groups, newGroup]
}

function connectSSE(executionId: number) {
  disconnectSSE()
  const url = `/api/v1/executions/${executionId}/stream`
  eventSource = new EventSource(url)

  eventSource.addEventListener('connected', () => {
    console.log('[SSE] connected to execution', executionId)
  })

  eventSource.addEventListener('execution_status', (e) => {
    const data = JSON.parse(e.data)
    if (currentExecution.value && currentExecution.value.id === data.execution_id) {
      const updated = { ...currentExecution.value, status: data.status }
      if (data.error) updated.error = data.error
      currentExecution.value = updated
    }
  })

  eventSource.addEventListener('group_status', (e) => {
    const data = JSON.parse(e.data)
    if (currentExecution.value && currentExecution.value.id === data.execution_id) {
      const groups = ensureGroupInExecution(data.group_id, data.status)
      const idx = groups.findIndex(g => g.group_id === data.group_id)
      if (idx >= 0) {
        const updatedGroups = [...groups]
        updatedGroups[idx] = { ...updatedGroups[idx], status: data.status }
        currentExecution.value = { ...currentExecution.value, stage_group_executions: updatedGroups }
      }
    }
  })

  eventSource.addEventListener('stage_status', (e) => {
    const data = JSON.parse(e.data)
    if (currentExecution.value && currentExecution.value.id === data.execution_id) {
      let groups = currentExecution.value.stage_group_executions || []
      const gi = groups.findIndex(sge =>
        sge.stage_executions?.some(se => se.stage_id === data.stage_id)
      )
      if (gi >= 0) {
        const stages = groups[gi].stage_executions || []
        const si = stages.findIndex(se => se.stage_id === data.stage_id)
        if (si >= 0) {
          const updatedGroups = [...groups]
          const updatedStages = [...stages]
          updatedStages[si] = { ...updatedStages[si], status: data.status }
          updatedGroups[gi] = { ...updatedGroups[gi], stage_executions: updatedStages }
          currentExecution.value = { ...currentExecution.value, stage_group_executions: updatedGroups }
        }
      } else {
        const workflowGroupId = findGroupIdForStage(data.stage_id)
        if (workflowGroupId) {
          groups = ensureGroupInExecution(workflowGroupId, 'running')
          const groupIdx = groups.findIndex(g => g.group_id === workflowGroupId)
          const newStage = {
            id: nextTempId--,
            stage_group_execution_id: groups[groupIdx]?.id || 0,
            stage_id: data.stage_id,
            status: data.status as any,
            error: '',
            started_at: null,
            finished_at: null,
            task_executions: [],
          }
          const updatedGroups = [...groups]
          updatedGroups[groupIdx] = {
            ...updatedGroups[groupIdx],
            stage_executions: [...(updatedGroups[groupIdx].stage_executions || []), newStage],
          }
          currentExecution.value = { ...currentExecution.value, stage_group_executions: updatedGroups }
        } else {
          console.warn('[SSE] stage not found in any group, ignoring', data.stage_id)
        }
      }
    }
  })

  eventSource.onerror = () => {
    if (eventSource?.readyState === EventSource.CLOSED) {
      console.warn('[SSE] connection closed, falling back to polling')
      disconnectSSE()
      startPolling()
    }
  }
}

function disconnectSSE() {
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
}

// ==================== 轮询降级（SSE 不可用时） ====================
let pollingTimer: ReturnType<typeof setInterval> | null = null

function startPolling() {
  stopPolling()
  pollingTimer = setInterval(async () => {
    if (!currentExecution.value) { stopPolling(); return }
    const status = currentExecution.value.status
    if (status === 'running' || status === 'paused') {
      await loadExecutionById(currentExecution.value.id)
    } else {
      stopPolling()
    }
  }, 3000)
}

function stopPolling() {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

async function loadExecutionById(execId: number) {
  try {
    currentExecution.value = await getExecutionApi(workflowId.value, execId)
  } catch {
    currentExecution.value = null
  }
}

async function loadLatestExecution() {
  try {
    const execs = await getExecutionsApi(workflowId.value)
    if (execs && execs.length > 0) {
      await loadExecutionById(execs[0].id)
      if (currentExecution.value && (currentExecution.value.status === 'running' || currentExecution.value.status === 'paused')) {
        connectSSE(currentExecution.value.id)
      }
    } else {
      currentExecution.value = null
    }
  } catch {
    currentExecution.value = null
  }
}

async function loadRecentExecutions() {
  try {
    const execs = await getExecutionsApi(workflowId.value)
    recentExecutions.value = (execs || []).slice(0, 5)
  } catch {
    recentExecutions.value = []
  }
}

// ==================== 执行历史 ====================
const showExecHistory = ref(false)
const loadingExecHistory = ref(false)
const execHistory = ref<WorkflowExecution[]>([])

async function loadExecHistory() {
  loadingExecHistory.value = true
  try {
    execHistory.value = await getExecutionsApi(workflowId.value)
  } catch {
    execHistory.value = []
  } finally {
    loadingExecHistory.value = false
  }
}

// ==================== 阶段详情 ====================
const showStageDetail = ref(false)
const stageDetailData = ref<WorkflowStageExecution | null>(null)
const stageDetailTitle = computed(() => {
  if (!stageDetailData.value) return '阶段详情'
  return stageDetailData.value.stage?.name || `Stage #${stageDetailData.value.stage_id}`
})

function openStageDetail(gi: number, si: number) {
  const stage = workflow.value.stage_groups[gi]?.stages[si]
  if (!stage?.id) return
  const exec = stageExecMap.value.get(stage.id)
  if (!exec) {
    ElMessage.info('该阶段暂无执行数据')
    return
  }
  stageDetailData.value = exec
  showStageDetail.value = true
}

// ==================== 模板相关 ====================

const filteredTemplates = computed(() => {
  if (!templateSearch.value) return templates.value
  const kw = templateSearch.value.toLowerCase()
  return templates.value.filter((t) => t.name.toLowerCase().includes(kw))
})

async function loadTemplates() {
  try {
    templates.value = await getStageTemplatesApi()
  } catch {
    templates.value = []
  }
}

function refreshTemplates() {
  loadTemplates()
  ElMessage.success('已刷新')
}

function onTemplateDragStart(e: DragEvent, tpl: StageTemplateApi) {
  e.dataTransfer?.setData('application/json', JSON.stringify(tpl))
  e.dataTransfer!.effectAllowed = 'copy'
}

function onDropToGroup(e: DragEvent, groupIndex: number) {
  const data = e.dataTransfer?.getData('application/json')
  if (!data) return
  try {
    const tpl: StageTemplateApi = JSON.parse(data)
    let rawTasks: any[] = []
    try {
      rawTasks = JSON.parse(tpl.tasks || '[]')
    } catch {
      rawTasks = []
    }
    const stage: WorkflowStage = {
      name: tpl.name,
      description: tpl.description,
      order: workflow.value.stage_groups[groupIndex].stages.length + 1,
      machine_group_id: tpl.machine_group_id,
      machine_group_name: '',
      template_version: tpl.version,
      tasks: rawTasks.map((t: any, i: number) => ({
        ref: 0,
        name: t.name || '',
        module: t.module || 'shell',
        params: typeof t.params === 'object' ? JSON.stringify(t.params) : (t.params || ''),
        order: i + 1,
        when: t.when || '',
        hook_ids: t.hook_ids || '',
        loop: '',
        timeout: t.timeout || 0,
        ignore_errors: t.ignore_errors ?? false,
        retries: t.retries || 0,
        delay: t.delay || 0,
        register: t.register || '',
      })),
    }
    workflow.value.stage_groups[groupIndex].stages.push(stage)
  } catch {
    // ignore
  }
}

function onStageReorder(_groupIndex: number) {}

// ==================== 阶段组/阶段操作 ====================
function addGroup() {
  workflow.value.stage_groups.push({
    name: `阶段组 ${workflow.value.stage_groups.length + 1}`,
    description: '',
    order: workflow.value.stage_groups.length + 1,
    mode: 'sequential',
    stages: [],
  })
}

function removeGroup(index: number) {
  workflow.value.stage_groups.splice(index, 1)
}

function toggleGroupMode(index: number) {
  const group = workflow.value.stage_groups[index]
  group.mode = group.mode === 'sequential' ? 'parallel' : 'sequential'
}

function removeStage(groupIndex: number, stageIndex: number) {
  workflow.value.stage_groups[groupIndex].stages.splice(stageIndex, 1)
}

function showAddStageDialog(groupIndex: number) {
  addStageGroupIndex.value = groupIndex
  newStage.value = { name: '', description: '', machine_group_id: 0 }
  showAddStage.value = true
}

const newStage = ref({ name: '', description: '', machine_group_id: 0 })
const addStageGroupIndex = ref(0)

function confirmAddStage() {
  if (!newStage.value.name) {
    ElMessage.warning('请输入阶段名称')
    return
  }
  if (!newStage.value.machine_group_id) {
    ElMessage.warning('请选择机器分组')
    return
  }
  const group = workflow.value.stage_groups[addStageGroupIndex.value]
  group.stages.push({
    name: newStage.value.name,
    description: newStage.value.description,
    order: group.stages.length + 1,
    machine_group_id: newStage.value.machine_group_id,
    machine_group_name: machineGroups.value.find((g) => g.id === newStage.value.machine_group_id)?.name || '',
    tasks: [],
  })
  showAddStage.value = false
}

// ==================== 保存 ====================
async function handleSave() {
  if (!workflow.value.name) {
    ElMessage.warning('请输入工作流名称')
    return
  }

  saving.value = true
  try {
    let globalRef = 1
    const stageGroups = workflow.value.stage_groups.map((g, gi) => ({
      name: g.name,
      description: g.description || '',
      order: gi + 1,
      mode: g.mode || 'sequential',
      stages: g.stages.map((s, si) => ({
        name: s.name,
        description: s.description || '',
        order: si + 1,
        machine_group_id: s.machine_group_id || 0,
        template_version: s.template_version || '',
        tasks: (s.tasks || []).map((t, ti) => ({
          ref: globalRef++,
          name: t.name,
          module: t.module,
          params: t.params || '',
          order: ti + 1,
          when: t.when || '',
          hook_ids: t.hook_ids || '',
          loop: t.loop || '',
          timeout: t.timeout || 0,
          ignore_errors: t.ignore_errors ?? false,
          retries: t.retries || 0,
          delay: t.delay || 0,
          register: t.register || '',
        })),
      })),
    }))

    const payload = {
      name: workflow.value.name,
      description: workflow.value.description,
      stage_groups: stageGroups,
      hooks: buildHooksSnapshot(stageGroups),
    }

    const updatedWf = await updateWorkflowApi(workflowId.value, payload)
    lastSaved.value = new Date(updatedWf.updated_at).toLocaleTimeString()
    await loadData()
    editMode.value = false
    ElMessage.success('保存成功')
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function goBack() {
  router.push('/workflow')
}

function buildHooksSnapshot(stageGroups: any[]) {
  const hookNames = new Set<string>()
  for (const g of stageGroups) {
    for (const s of g.stages || []) {
      for (const t of s.tasks || []) {
        if (t.hook_ids) {
          try {
            const arr = JSON.parse(t.hook_ids)
            if (Array.isArray(arr)) arr.forEach((n: string) => hookNames.add(n))
          } catch { /* skip */ }
        }
      }
    }
  }
  const hooks: any[] = []
  for (const name of hookNames) {
    const ht = hookTemplates.value.find((h) => h.name === name)
    if (ht) {
      hooks.push({
        name: ht.name,
        module: ht.module,
        params: ht.params || '',
        timeout: ht.timeout,
        ignore_errors: ht.ignore_errors,
        retries: ht.retries,
        delay: ht.delay,
      })
    }
  }
  return hooks
}

async function loadData() {
  try {
    const [wf, groups, hooksData] = await Promise.all([
      getWorkflowApi(workflowId.value),
      getMachineGroupsApi(),
      getHookTemplatesApi().catch(() => [] as HookTemplate[]),
    ])
    machineGroups.value = groups
    hookTemplates.value = hooksData
    workflow.value = {
      name: wf.name,
      description: wf.description,
      stage_groups: (wf.stage_groups || []).map((g) => ({
        ...g,
        stages: (g.stages || []).map((s) => ({
          ...s,
          machine_group_name: groups.find((mg) => mg.id === s.machine_group_id)?.name || '',
          tasks: (s.tasks || []).map((t) => ({
            ...t,
            ignore_errors: t.ignore_errors ?? false,
            retries: t.retries ?? 0,
            delay: t.delay ?? 0,
            register: t.register || '',
          })),
        })),
      })),
    }
    lastSaved.value = new Date(wf.updated_at).toLocaleTimeString()
  } catch {
    ElMessage.error('加载工作流失败')
  }
}

onMounted(async () => {
  await loadData()
  await Promise.all([loadTemplates(), loadLatestExecution(), loadRecentExecutions()])
})

onUnmounted(() => {
  disconnectSSE()
  stopPolling()
})
</script>

<style scoped>
.canvas-page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
}

/* ==================== Toolbar ==================== */
.canvas-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}

.toolbar-left,
.toolbar-center,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.workflow-title {
  font-size: 16px;
  font-weight: 600;
}

.workflow-name-input {
  width: 240px;
}

.workflow-name-input :deep(.el-input__inner) {
  font-size: 16px;
  font-weight: 600;
  border: none;
  background: transparent;
}

.workflow-name-input :deep(.el-input__inner):focus {
  border-bottom: 2px solid var(--el-color-primary);
  border-radius: 0;
}

.save-status {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.save-status.saved {
  color: var(--el-color-success);
}

/* ==================== Execution Status Bar ==================== */
.execution-status-bar {
  display: flex;
  align-items: center;
  gap: 10px;
}

.exec-progress {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.exec-error {
  font-size: 12px;
  color: var(--el-color-danger);
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.spin-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* ==================== Canvas Body ==================== */
.canvas-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.canvas-area {
  flex: 1;
  overflow-x: auto;
  overflow-y: auto;
  padding: 24px;
}

.canvas-columns {
  display: flex;
  gap: 16px;
  min-height: 100%;
  align-items: flex-start;
}

/* ==================== Stage Palette ==================== */
.stage-palette {
  width: 220px;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-lighter);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  position: relative;
  transition: width 0.2s;
}

.stage-palette.collapsed {
  width: 36px;
}

.palette-toggle {
  position: absolute;
  left: -1px;
  top: 50%;
  transform: translate(-100%, -50%);
  width: 24px;
  height: 48px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-right: none;
  border-radius: 6px 0 0 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 10;
  transition: background 0.15s;
}

.palette-toggle:hover {
  background: var(--el-fill-color-light);
}

.stage-palette.collapsed .palette-toggle {
  left: 0;
  transform: translateY(-50%);
  border-radius: 6px 0 0 6px;
}

.palette-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  font-weight: 600;
  font-size: 14px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.palette-search {
  padding: 8px 12px;
}

.palette-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.palette-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: grab;
  margin-bottom: 4px;
  border: 1px dashed var(--el-border-color);
  transition: all 0.15s;
  font-size: 13px;
}

.palette-item:hover {
  background: var(--el-fill-color-light);
  border-color: var(--el-color-primary);
}

.palette-item:active {
  cursor: grabbing;
}

.palette-item-info {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.palette-item-name {
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.palette-item-meta {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.palette-empty {
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  padding: 20px;
}

/* ==================== Canvas Column ==================== */
.canvas-column {
  width: 280px;
  min-width: 280px;
  background: var(--el-bg-color);
  border: 2px solid var(--el-border-color-lighter);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 120px);
  transition: border-color 0.3s, box-shadow 0.3s;
}

.canvas-column.column-running {
  border-color: var(--el-color-warning);
  box-shadow: 0 0 0 1px var(--el-color-warning-light-5);
}

.canvas-column.column-success {
  border-color: var(--el-color-success-light-5);
}

.canvas-column.column-failed {
  border-color: var(--el-color-danger);
  box-shadow: 0 0 0 1px var(--el-color-danger-light-5);
}

.column-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 14px 4px;
}

.column-name {
  font-weight: 600;
  font-size: 14px;
  flex: 1;
}

.column-name-input {
  flex: 1;
}

.column-name-input :deep(.el-input__inner) {
  font-weight: 600;
  border: none;
  background: transparent;
}

.column-name-input :deep(.el-input__inner):focus {
  border-bottom: 1px solid var(--el-color-primary);
  border-radius: 0;
}

.group-status-icon {
  margin-left: 8px;
}

.column-mode {
  padding: 0 14px 8px;
}

.column-stages {
  flex: 1;
  overflow-y: auto;
  padding: 0 8px 8px;
}

.stage-drop-zone {
  min-height: 60px;
}

/* ==================== Stage Card (Edit Mode) ==================== */
.stage-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px 12px;
  margin-bottom: 8px;
  transition: box-shadow 0.15s, border-color 0.2s;
}

.stage-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.stage-card:active {
  cursor: grabbing;
}

/* ==================== Stage Card (View Mode) ==================== */
.stage-card-view {
  cursor: pointer;
}

.stage-card-view:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border-color: var(--el-color-primary-light-5);
}

.stage-card-view.stage-running {
  border-color: var(--el-color-warning);
  background: rgba(230, 162, 60, 0.04);
}

.stage-card-view.stage-success {
  border-color: var(--el-color-success-light-5);
  background: rgba(103, 194, 58, 0.04);
}

.stage-card-view.stage-failed {
  border-color: var(--el-color-danger);
  background: rgba(245, 63, 63, 0.04);
}

.stage-card-view.stage-pending {
  opacity: 0.6;
}

.stage-card-view.stage-skipped {
  opacity: 0.4;
}

.stage-ghost {
  opacity: 0.5;
  border: 2px dashed var(--el-color-primary);
}

.stage-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  gap: 6px;
}

.stage-card-name {
  font-weight: 500;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.stage-card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.stage-task-count {
  color: var(--el-text-color-secondary);
}

.version-tag {
  margin-left: auto;
}

.stage-duration {
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

.add-stage-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 10px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}

.add-stage-hint:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

.add-column {
  width: 200px;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px 20px;
  border: 2px dashed var(--el-border-color);
  border-radius: 10px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.add-column:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

/* ==================== Recent Executions ==================== */
.recent-executions {
  margin-top: 32px;
  padding-top: 24px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.recent-executions h4 {
  margin: 0 0 12px;
  font-size: 14px;
  color: var(--el-text-color-regular);
}

.exec-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.exec-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}

.exec-item:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
}

.exec-trigger {
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.exec-time {
  margin-left: auto;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* ==================== Stage Detail Drawer ==================== */
.stage-detail-section {
  padding: 0;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.detail-error {
  color: var(--el-color-danger);
  font-size: 13px;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.task-detail-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
}

.task-detail-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-detail-name {
  font-weight: 500;
  font-size: 13px;
  flex: 1;
}

.task-detail-host {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.task-detail-duration {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.output-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
  text-transform: uppercase;
}

.task-output,
.task-error {
  margin-top: 8px;
  border-radius: 4px;
  font-size: 12px;
  overflow-x: auto;
}

.task-output pre,
.task-error pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 200px;
  overflow-y: auto;
}

.task-output {
  background: var(--el-fill-color-lighter);
  padding: 8px;
}

.task-error {
  background: rgba(245, 63, 63, 0.06);
  color: var(--el-color-danger);
  padding: 8px;
}

.no-data {
  text-align: center;
  color: var(--el-text-color-secondary);
  padding: 40px 0;
}

/* ==================== Status Colors ==================== */
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
.danger-item {
  color: var(--el-color-danger) !important;
}
</style>
