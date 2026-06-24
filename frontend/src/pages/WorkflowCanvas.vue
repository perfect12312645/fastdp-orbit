<template>
  <div class="canvas-page">
    <!-- 顶部工具栏 -->
    <div class="canvas-toolbar">
      <div class="toolbar-left">
        <el-button @click="goBack" text>
          <Icon icon="mdi:arrow-left" :size="18" /> 返回
        </el-button>
        <el-divider direction="vertical" />
        <el-input
          v-model="workflow.name"
          placeholder="工作流名称"
          class="workflow-name-input"
        />
      </div>
      <div class="toolbar-center">
        <span class="save-status" v-if="saving">保存中...</span>
        <span class="save-status saved" v-else-if="lastSaved">已保存 {{ lastSaved }}</span>
      </div>
      <div class="toolbar-right">
        <el-button @click="showVariables = true">
          <Icon icon="mdi:code-json" :size="16" /> 变量
        </el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          <Icon icon="mdi:content-save" :size="16" /> 保存
        </el-button>
        <el-button type="success" @click="handleExecute" :loading="executing">
          <Icon icon="mdi:play" :size="16" /> 执行
        </el-button>
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
            @dragover.prevent
            @drop="onDropToGroup($event, gi)"
          >
            <div class="column-header">
              <el-input
                v-model="group.name"
                placeholder="阶段组名称"
                class="column-name-input"
              />
              <div class="column-actions">
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
                    </div>
                  </div>
                </template>
              </draggable>
              <div class="add-stage-hint" @click="showAddStageDialog(gi)">
                <Icon icon="mdi:plus" :size="16" /> 添加阶段
              </div>
            </div>
          </div>

          <!-- 添加阶段组按钮 -->
          <div class="add-column" @click="addGroup">
            <Icon icon="mdi:plus" :size="24" />
            <span>添加阶段组</span>
          </div>
        </div>
      </div>

      <!-- 右侧阶段模板面板 -->
      <div class="stage-palette" :class="{ collapsed: paletteCollapsed }">
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

    <!-- 变量对话框 -->
    <el-dialog v-model="showVariables" title="全局变量" width="500px" destroy-on-close>
      <div class="variables-list">
        <div v-for="(v, vi) in workflow.variables" :key="vi" class="variable-item">
          <el-input v-model="v.key" placeholder="变量名" style="width: 120px" />
          <el-select v-model="v.type" style="width: 100px">
            <el-option label="String" value="string" />
            <el-option label="Number" value="number" />
            <el-option label="Bool" value="bool" />
          </el-select>
          <el-input v-model="v.value" placeholder="默认值" style="flex: 1" />
          <el-button type="danger" link size="small" @click="workflow.variables.splice(vi, 1)">
            <Icon icon="mdi:delete" :size="14" />
          </el-button>
        </div>
        <el-button type="primary" text @click="workflow.variables.push({ key: '', type: 'string', value: '' })">
          <Icon icon="mdi:plus" :size="14" /> 添加变量
        </el-button>
      </div>
      <template #footer>
        <el-button @click="showVariables = false">关闭</el-button>
      </template>
    </el-dialog>

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
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import draggable from 'vuedraggable'
import { getWorkflowApi, updateWorkflowApi, executeWorkflowApi } from '@/api/workflow'
import { getMachineGroupsApi, type MachineGroup } from '@/api/machineGroup'
import { getStageTemplatesApi, type StageTemplate as StageTemplateApi } from '@/api/stageTemplate'
import type { WorkflowStageGroup, WorkflowStage, WorkflowTask } from '@/types/workflow'

const route = useRoute()
const router = useRouter()
const workflowId = computed(() => Number(route.params.id))

const saving = ref(false)
const executing = ref(false)
const lastSaved = ref('')
const showVariables = ref(false)
const showAddStage = ref(false)
const templateSearch = ref('')
const machineGroups = ref<MachineGroup[]>([])

const workflow = ref({
  name: '',
  description: '',
  config: '',
  stage_groups: [] as (WorkflowStageGroup & { stages: WorkflowStage[] })[],
  variables: [] as { key: string; type: string; value: string; description?: string; group?: string }[],
  hooks: [] as any[],
})

const newStage = ref({
  name: '',
  description: '',
  machine_group_id: 0,
})

const addStageGroupIndex = ref(0)
const paletteCollapsed = ref(false)

let templates = ref<StageTemplateApi[]>([])

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
    // tasks 是 JSON 字符串，需要解析
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

function onStageReorder(groupIndex: number) {
  // 手动保存，不自动保存
}

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

function confirmAddStage() {
  if (!newStage.value.name) {
    ElMessage.warning('请输入阶段名称')
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
      config: workflow.value.config || '',
      stage_groups: stageGroups,
      variables: workflow.value.variables.map((v) => ({
        key: v.key,
        type: v.type as 'string' | 'number' | 'bool',
        value: v.value,
      })),
      hooks: workflow.value.hooks || [],
    }

    await updateWorkflowApi(workflowId.value, payload)
    lastSaved.value = new Date().toLocaleTimeString()
    ElMessage.success('保存成功')
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleExecute() {
  try {
    await ElMessageBox.confirm(`确认执行工作流「${workflow.value.name}」？`, '执行确认', {
      confirmButtonText: '执行',
      cancelButtonText: '取消',
      type: 'warning',
    })
    executing.value = true
    await executeWorkflowApi(workflowId.value)
    ElMessage.success('工作流已触发执行')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('执行失败')
  } finally {
    executing.value = false
  }
}

function goBack() {
  router.push('/workflow')
}

async function loadData() {
  try {
    const [wf, groups] = await Promise.all([
      getWorkflowApi(workflowId.value),
      getMachineGroupsApi(),
    ])
    machineGroups.value = groups
    workflow.value = {
      name: wf.name,
      description: wf.description,
      config: wf.config,
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
      variables: (wf.variables || []).map((v) => ({
        key: v.key,
        type: v.type,
        value: v.value || '',
        description: v.description,
        group: v.group,
      })),
      hooks: wf.hooks || [],
    }
    lastSaved.value = new Date().toLocaleTimeString()
  } catch {
    ElMessage.error('加载工作流失败')
  }
}

onMounted(() => {
  loadData()
  loadTemplates()
})
</script>

<style scoped>
.canvas-page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
}

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

.canvas-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

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

.canvas-column {
  width: 260px;
  min-width: 260px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 120px);
}

.column-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 12px 4px;
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

.column-mode {
  padding: 0 12px 8px;
}

.column-stages {
  flex: 1;
  overflow-y: auto;
  padding: 0 8px 8px;
}

.stage-drop-zone {
  min-height: 60px;
}

.stage-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px 12px;
  margin-bottom: 8px;
  cursor: grab;
  transition: box-shadow 0.15s;
}

.stage-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.stage-card:active {
  cursor: grabbing;
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

.variables-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.variable-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.danger-item {
  color: var(--el-color-danger) !important;
}
</style>
