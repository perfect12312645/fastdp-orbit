<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>阶段管理</h2>
        <p class="page-subtitle">预编排阶段模板，供工作流画布拖拽使用</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateDialog">
          <Icon icon="mdi:plus" :size="16" /> 创建阶段
        </el-button>
      </div>
    </div>

    <div class="page-content">
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input v-model="searchText" placeholder="搜索阶段名称" clearable style="width: 240px;">
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
        </div>
        <div class="table-toolbar-right">
          <span class="total-text">共 {{ filteredStages.length }} 个阶段</span>
        </div>
      </div>

      <div class="stage-cards" v-loading="loading">
        <div
          v-for="stage in filteredStages"
          :key="stage.id"
          class="stage-card"
        >
          <div class="stage-card-header">
            <div class="stage-card-title">
              <Icon icon="mdi:view-column-outline" :size="20" />
              <span>{{ stage.name }}</span>
              <el-tag size="small" type="info" effect="plain" class="version-tag">v{{ stage.version }}</el-tag>
            </div>
            <div class="stage-card-actions">
              <el-button type="primary" link size="small" @click="showVersionHistory(stage)">
                <Icon icon="mdi:history" :size="14" /> 版本
              </el-button>
              <el-button type="primary" link size="small" @click="editStage(stage)">
                <Icon icon="mdi:pencil" :size="14" /> 编辑
              </el-button>
              <el-button type="danger" link size="small" @click="deleteStage(stage)">
                <Icon icon="mdi:delete-outline" :size="14" />
              </el-button>
            </div>
          </div>
          <p v-if="stage.description" class="stage-card-desc">{{ stage.description }}</p>
          <div class="stage-card-info">
            <el-tag size="small" type="info" effect="plain">
              <Icon icon="mdi:server-network" :size="12" /> {{ getMachineGroupName(stage.machine_group_id) || '未指定分组' }}
            </el-tag>
            <el-tag size="small" type="primary" effect="plain">
              {{ getTasks(stage).length }} 个任务
            </el-tag>
          </div>
          <div class="stage-card-tasks" v-if="getTasks(stage).length > 0">
            <div v-for="(task, ti) in getTasks(stage).slice(0, 3)" :key="ti" class="task-mini">
              <span class="task-mini-index">{{ ti + 1 }}</span>
              <span class="task-mini-name">{{ task.name || '未命名' }}</span>
              <el-tag size="small" effect="plain" class="task-mini-module">{{ task.module }}</el-tag>
            </div>
            <div v-if="getTasks(stage).length > 3" class="task-more">
              +{{ getTasks(stage).length - 3 }} 个任务
            </div>
          </div>
        </div>

        <div v-if="filteredStages.length === 0 && !loading" class="stage-empty">
          <Icon icon="mdi:view-column-outline" :size="48" />
          <p>暂无阶段模板</p>
          <el-button type="primary" @click="showCreateDialog">创建阶段</el-button>
        </div>
      </div>
    </div>

    <!-- 创建对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="创建阶段"
      width="700px"
      destroy-on-close
      top="5vh"
    >
      <el-form :model="formData" label-width="100px" ref="formRef" :rules="formRules">
        <el-form-item label="阶段名称" prop="name">
          <el-input v-model="formData.name" placeholder="如：安装 Docker" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="formData.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="目标机器分组" prop="machine_group_id">
          <el-select
            v-model="formData.machine_group_id"
            placeholder="选择机器分组"
            filterable
            style="width: 100%"
            :loading="machineGroupLoading"
          >
            <el-option
              v-for="g in machineGroups"
              :key="g.id"
              :label="g.name"
              :value="g.id"
            >
              <span>{{ g.name }}</span>
              <span style="color: var(--el-text-color-secondary); margin-left: 8px; font-size: 12px">
                {{ g.machines?.length || 0 }} 台机器
              </span>
            </el-option>
          </el-select>
        </el-form-item>

        <el-divider />

        <div class="section-header">
          <div class="section-title">任务列表</div>
          <el-button type="primary" size="small" @click="addTask">
            <Icon icon="mdi:plus" :size="14" /> 添加任务
          </el-button>
        </div>

        <div class="task-list">
          <div v-for="(task, ti) in formData.tasks" :key="ti" class="task-card">
            <div class="task-header">
              <span class="task-index">{{ ti + 1 }}</span>
              <el-input v-model="task.name" placeholder="任务名称" class="task-name-input" />
              <el-button
                type="danger"
                link
                size="small"
                @click="removeTask(ti)"
              >
                <Icon icon="mdi:delete" :size="14" />
              </el-button>
            </div>
            <div class="task-body">
              <div class="task-row">
                <el-select v-model="task.module" placeholder="模块" style="width: 130px">
                  <el-option label="Shell" value="shell" />
                  <el-option label="Systemd" value="systemd" />
                  <el-option label="Package" value="package" />
                  <el-option label="File" value="file" />
                  <el-option label="Template" value="template" />
                  <el-option label="Repo" value="repo" />
                  <el-option label="Blockinfile" value="blockinfile" />
                  <el-option label="Modprobe" value="modprobe" />
                </el-select>
                <el-input-number v-model="task.timeout" :min="0" :max="3600" placeholder="超时(秒)" style="width: 120px" />
                <el-input-number v-model="task.retries" :min="0" :max="10" placeholder="重试次数" style="width: 120px" />
                <el-input-number v-model="task.delay" :min="0" :max="60" placeholder="重试间隔(秒)" style="width: 140px" />
              </div>
              <el-input
                v-model="task.params"
                type="textarea"
                :rows="2"
                placeholder='参数 JSON，如: {"command": "yum install -y docker-ce"}'
              />
              <el-input v-model="task.when" placeholder='条件，如: {{.machine.os_name}} !contains ubuntu' style="width: 100%" />
              <div class="task-row">
                <el-input v-model="task.hook_ids" placeholder='后置钩子 ref，如: [1,3]' style="flex: 1" />
                <el-input v-model="task.register" placeholder="注册变量名" style="width: 150px" />
                <el-checkbox v-model="task.ignore_errors">忽略错误</el-checkbox>
              </div>
            </div>
          </div>
          <div v-if="formData.tasks.length === 0" class="empty-tip">
            点击「添加任务」配置此阶段的操作
          </div>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="submitting">
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑对话框（含修改描述） -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑阶段"
      width="700px"
      destroy-on-close
      top="5vh"
    >
      <el-form :model="editFormData" label-width="100px" ref="editFormRef" :rules="editFormRules">
        <el-form-item label="阶段名称" prop="name">
          <el-input v-model="editFormData.name" placeholder="如：安装 Docker" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editFormData.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="目标机器分组" prop="machine_group_id">
          <el-select
            v-model="editFormData.machine_group_id"
            placeholder="选择机器分组"
            filterable
            style="width: 100%"
            :loading="machineGroupLoading"
          >
            <el-option
              v-for="g in machineGroups"
              :key="g.id"
              :label="g.name"
              :value="g.id"
            >
              <span>{{ g.name }}</span>
              <span style="color: var(--el-text-color-secondary); margin-left: 8px; font-size: 12px">
                {{ g.machines?.length || 0 }} 台机器
              </span>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="修改描述" prop="change_note">
          <el-input v-model="editFormData.change_note" placeholder="必填，描述本次修改内容" />
        </el-form-item>

        <el-divider />

        <div class="section-header">
          <div class="section-title">任务列表</div>
          <el-button type="primary" size="small" @click="addEditTask">
            <Icon icon="mdi:plus" :size="14" /> 添加任务
          </el-button>
        </div>

        <div class="task-list">
          <div v-for="(task, ti) in editFormData.tasks" :key="ti" class="task-card">
            <div class="task-header">
              <span class="task-index">{{ ti + 1 }}</span>
              <el-input v-model="task.name" placeholder="任务名称" class="task-name-input" />
              <el-button
                type="danger"
                link
                size="small"
                @click="removeEditTask(ti)"
              >
                <Icon icon="mdi:delete" :size="14" />
              </el-button>
            </div>
            <div class="task-body">
              <div class="task-row">
                <el-select v-model="task.module" placeholder="模块" style="width: 130px">
                  <el-option label="Shell" value="shell" />
                  <el-option label="Systemd" value="systemd" />
                  <el-option label="Package" value="package" />
                  <el-option label="File" value="file" />
                  <el-option label="Template" value="template" />
                  <el-option label="Repo" value="repo" />
                  <el-option label="Blockinfile" value="blockinfile" />
                  <el-option label="Modprobe" value="modprobe" />
                </el-select>
                <el-input-number v-model="task.timeout" :min="0" :max="3600" placeholder="超时(秒)" style="width: 120px" />
                <el-input-number v-model="task.retries" :min="0" :max="10" placeholder="重试次数" style="width: 120px" />
                <el-input-number v-model="task.delay" :min="0" :max="60" placeholder="重试间隔(秒)" style="width: 140px" />
              </div>
              <el-input
                v-model="task.params"
                type="textarea"
                :rows="2"
                placeholder='参数 JSON，如: {"command": "yum install -y docker-ce"}'
              />
              <el-input v-model="task.when" placeholder='条件，如: {{.machine.os_name}} !contains ubuntu' style="width: 100%" />
              <div class="task-row">
                <el-input v-model="task.hook_ids" placeholder='后置钩子 ref，如: [1,3]' style="flex: 1" />
                <el-input v-model="task.register" placeholder="注册变量名" style="width: 150px" />
                <el-checkbox v-model="task.ignore_errors">忽略错误</el-checkbox>
              </div>
            </div>
          </div>
          <div v-if="editFormData.tasks.length === 0" class="empty-tip">
            点击「添加任务」配置此阶段的操作
          </div>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleEdit" :loading="submitting">
          保存（生成新版本）
        </el-button>
      </template>
    </el-dialog>

    <!-- 版本历史对话框 -->
    <el-dialog
      v-model="versionDialogVisible"
      title="版本历史"
      width="600px"
      destroy-on-close
    >
      <div v-loading="versionLoading">
        <div v-if="versionStage" class="version-current">
          <span>当前版本：<strong>v{{ versionStage.version }}</strong></span>
        </div>
        <el-timeline v-if="versionList.length > 0">
          <el-timeline-item
            v-for="v in versionList"
            :key="v.id"
            :timestamp="formatTime(v.created_at)"
            placement="top"
            :type="v.version === versionStage?.version ? 'primary' : ''"
          >
            <div class="version-item">
              <div class="version-item-header">
                <el-tag size="small" :type="v.version === versionStage?.version ? 'primary' : 'info'" effect="plain">
                  v{{ v.version }}
                </el-tag>
                <span v-if="v.version === versionStage?.version" class="current-badge">当前</span>
              </div>
              <p class="version-note">{{ v.change_note || '无描述' }}</p>
              <el-button
                v-if="v.version !== versionStage?.version"
                type="warning"
                size="small"
                @click="handleRollback(v)"
              >
                回滚到此版本
              </el-button>
            </div>
          </el-timeline-item>
        </el-timeline>
        <div v-else class="version-empty">暂无版本历史</div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getMachineGroupsApi, type MachineGroup } from '@/api/machineGroup'
import {
  getStageTemplatesApi,
  createStageTemplateApi,
  updateStageTemplateApi,
  deleteStageTemplateApi,
  listStageTemplateVersionsApi,
  rollbackStageTemplateApi,
  type StageTemplate,
  type StageTemplateVersion,
} from '@/api/stageTemplate'

interface StageTask {
  name: string
  module: string
  params: string
  timeout: number
  retries: number
  delay: number
  when: string
  hook_ids: string
  register: string
  ignore_errors: boolean
}

const loading = ref(false)
const searchText = ref('')
const stages = ref<StageTemplate[]>([])
const machineGroups = ref<MachineGroup[]>([])
const machineGroupLoading = ref(false)

// 创建对话框
const createDialogVisible = ref(false)
const formData = ref({
  name: '',
  description: '',
  machine_group_id: 0,
  tasks: [] as StageTask[],
})
const formRules = {
  name: [{ required: true, message: '请输入阶段名称', trigger: 'blur' }],
  machine_group_id: [{ required: true, message: '请选择机器分组', trigger: 'change' }],
}

// 编辑对话框
const editDialogVisible = ref(false)
const editingId = ref(0)
const editFormData = ref({
  name: '',
  description: '',
  machine_group_id: 0,
  tasks: [] as StageTask[],
  change_note: '',
})
const editFormRules = {
  name: [{ required: true, message: '请输入阶段名称', trigger: 'blur' }],
  machine_group_id: [{ required: true, message: '请选择机器分组', trigger: 'change' }],
  change_note: [{ required: true, message: '请描述本次修改内容', trigger: 'blur' }],
}

const submitting = ref(false)
const formRef = ref()
const editFormRef = ref()

// 版本历史
const versionDialogVisible = ref(false)
const versionLoading = ref(false)
const versionStage = ref<StageTemplate | null>(null)
const versionList = ref<StageTemplateVersion[]>([])

const filteredStages = computed(() => {
  if (!searchText.value) return stages.value
  const kw = searchText.value.toLowerCase()
  return stages.value.filter(
    (s) => s.name.toLowerCase().includes(kw) || (s.description || '').toLowerCase().includes(kw)
  )
})

function getTasks(stage: StageTemplate): StageTask[] {
  try {
    return JSON.parse(stage.tasks || '[]')
  } catch {
    return []
  }
}

function formatTime(t: string): string {
  if (!t) return ''
  return new Date(t).toLocaleString()
}

async function loadData() {
  loading.value = true
  try {
    stages.value = await getStageTemplatesApi()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function loadMachineGroups() {
  machineGroupLoading.value = true
  try {
    machineGroups.value = await getMachineGroupsApi()
  } catch (e) {
    console.error(e)
  } finally {
    machineGroupLoading.value = false
  }
}

function getMachineGroupName(id: number): string {
  const g = machineGroups.value.find((g) => g.id === id)
  return g?.name || ''
}

function showCreateDialog() {
  formData.value = { name: '', description: '', machine_group_id: 0, tasks: [] }
  createDialogVisible.value = true
  loadMachineGroups()
}

function editStage(stage: StageTemplate) {
  editingId.value = stage.id
  editFormData.value = {
    name: stage.name,
    description: stage.description,
    machine_group_id: stage.machine_group_id,
    tasks: getTasks(stage),
    change_note: '',
  }
  editDialogVisible.value = true
  loadMachineGroups()
}

function addTask() {
  formData.value.tasks.push({
    name: '',
    module: 'shell',
    params: '',
    timeout: 0,
    retries: 0,
    delay: 0,
    when: '',
    hook_ids: '',
    register: '',
    ignore_errors: false,
  })
}

function removeTask(index: number) {
  formData.value.tasks.splice(index, 1)
}

function addEditTask() {
  editFormData.value.tasks.push({
    name: '',
    module: 'shell',
    params: '',
    timeout: 0,
    retries: 0,
    delay: 0,
    when: '',
    hook_ids: '',
    register: '',
    ignore_errors: false,
  })
}

function removeEditTask(index: number) {
  editFormData.value.tasks.splice(index, 1)
}

function validateTasks(tasks: StageTask[]): boolean {
  if (tasks.length === 0) {
    ElMessage.warning('至少需要一个任务')
    return false
  }
  for (const task of tasks) {
    if (!task.name) {
      ElMessage.warning('任务名称不能为空')
      return false
    }
    if (!task.module) {
      ElMessage.warning(`任务「${task.name}」的模块类型不能为空`)
      return false
    }
  }
  return true
}

async function handleCreate() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  if (!validateTasks(formData.value.tasks)) return

  submitting.value = true
  try {
    await createStageTemplateApi({
      name: formData.value.name,
      description: formData.value.description,
      machine_group_id: formData.value.machine_group_id,
      tasks: JSON.stringify(formData.value.tasks),
    })
    ElMessage.success('创建成功')
    createDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

async function handleEdit() {
  try {
    await editFormRef.value?.validate()
  } catch {
    return
  }
  if (!validateTasks(editFormData.value.tasks)) return

  submitting.value = true
  try {
    await updateStageTemplateApi(editingId.value, {
      name: editFormData.value.name,
      description: editFormData.value.description,
      machine_group_id: editFormData.value.machine_group_id,
      tasks: JSON.stringify(editFormData.value.tasks),
      change_note: editFormData.value.change_note,
    })
    ElMessage.success('保存成功，已生成新版本')
    editDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
  } finally {
    submitting.value = false
  }
}

async function deleteStage(stage: StageTemplate) {
  try {
    await ElMessageBox.confirm(
      `确定要删除阶段「${stage.name}」吗？所有版本历史将一并删除。`,
      '删除确认',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteStageTemplateApi(stage.id)
    ElMessage.success('删除成功')
    loadData()
  } catch {
    // 用户取消
  }
}

// ==================== 版本管理 ====================

async function showVersionHistory(stage: StageTemplate) {
  versionStage.value = stage
  versionDialogVisible.value = true
  versionLoading.value = true
  try {
    versionList.value = await listStageTemplateVersionsApi(stage.id)
  } catch (e) {
    console.error(e)
    versionList.value = []
  } finally {
    versionLoading.value = false
  }
}

async function handleRollback(version: StageTemplateVersion) {
  try {
    await ElMessageBox.confirm(
      `确定要回滚到版本 v${version.version} 吗？\n\n当前版本将被保存为历史版本，系统会生成一个新的当前版本。`,
      '回滚确认',
      { confirmButtonText: '确定回滚', cancelButtonText: '取消', type: 'warning' }
    )
    await rollbackStageTemplateApi(versionStage.value!.id, version.version)
    ElMessage.success(`已回滚到 v${version.version}`)
    versionDialogVisible.value = false
    loadData()
  } catch {
    // 用户取消
  }
}

onMounted(loadData)
</script>

<style scoped>
.page-subtitle {
  font-size: var(--font-size-sm);
  color: var(--text-color-secondary);
  margin-top: 4px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.table-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.table-toolbar-left {
  display: flex;
  gap: 8px;
  align-items: center;
}

.total-text {
  font-size: 13px;
  color: var(--text-color-secondary);
}

.stage-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 16px;
}

.stage-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 16px;
  background: var(--el-bg-color);
  transition: box-shadow 0.2s;
}

.stage-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.stage-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.stage-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color-primary);
}

.version-tag {
  font-size: 11px;
  font-weight: 600;
}

.stage-card-actions {
  display: flex;
  gap: 4px;
}

.stage-card-desc {
  font-size: 13px;
  color: var(--text-color-secondary);
  margin-bottom: 12px;
}

.stage-card-info {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.stage-card-tasks {
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 12px;
}

.task-mini {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  font-size: 13px;
}

.task-mini-index {
  width: 18px;
  height: 18px;
  border-radius: 4px;
  background: var(--el-fill-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}

.task-mini-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-mini-module {
  flex-shrink: 0;
}

.task-more {
  text-align: center;
  font-size: 12px;
  color: var(--text-color-secondary);
  padding-top: 4px;
}

.stage-empty {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 0;
  color: var(--text-color-secondary);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.task-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 12px;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.task-index {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: var(--el-color-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}

.task-name-input {
  flex: 1;
}

.task-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.task-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.empty-tip {
  text-align: center;
  color: var(--text-color-secondary);
  font-size: 13px;
  padding: 20px;
}

/* 版本历史样式 */
.version-current {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  font-size: 14px;
}

.version-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.version-item-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.current-badge {
  font-size: 12px;
  color: var(--el-color-primary);
  font-weight: 600;
}

.version-note {
  font-size: 13px;
  color: var(--text-color-secondary);
  margin: 0;
}

.version-empty {
  text-align: center;
  color: var(--text-color-secondary);
  padding: 24px;
}
</style>
