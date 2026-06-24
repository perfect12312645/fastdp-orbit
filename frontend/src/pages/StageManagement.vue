<template>
  <div class="page-container">
    <!-- 列表视图 -->
    <template v-if="viewMode === 'list'">
      <div class="page-header">
        <div>
          <h2>阶段管理</h2>
          <p class="page-subtitle">预编排阶段模板，供工作流画布拖拽使用</p>
        </div>
        <div class="header-actions">
          <el-button type="primary" @click="showCreateEditor">
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
                <el-tag size="small" type="info" effect="plain" class="version-tag">{{ stage.version }}</el-tag>
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
                <span class="task-mini-index">{{ task.ref }}</span>
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
            <el-button type="primary" @click="showCreateEditor">创建阶段</el-button>
          </div>
        </div>
      </div>
    </template>

    <!-- 全屏编辑视图 -->
    <template v-if="viewMode === 'edit'">
      <div class="fullscreen-view">
        <div class="fullscreen-header">
          <div class="fullscreen-header-left">
            <el-button text @click="exitFullscreen">
              <Icon icon="mdi:arrow-left" :size="20" /> 返回列表
            </el-button>
            <el-divider direction="vertical" />
            <span class="fullscreen-title">{{ editingId ? '编辑阶段' : '创建阶段' }}</span>
            <el-tag v-if="editingId && currentStage" size="small" type="info" effect="plain">
              当前版本: {{ currentStage.version }}
            </el-tag>
          </div>
          <div class="fullscreen-header-right">
            <el-button @click="exitFullscreen">取消</el-button>
            <el-button type="primary" @click="handleSave" :loading="submitting">
              <Icon icon="mdi:content-save" :size="16" />
              {{ editingId ? '保存为新版本' : '创建阶段' }}
            </el-button>
          </div>
        </div>

        <div class="fullscreen-content">
          <el-form :model="formData" label-width="90px" ref="formRef" :rules="formRules">
            <div class="form-section">
              <h3 class="form-section-title">基本信息</h3>
              <el-row :gutter="24">
                <el-col :span="12">
                  <el-form-item label="阶段名称" prop="name">
                    <el-input v-model="formData.name" placeholder="如：安装 Docker" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="目标分组" prop="machine_group_id">
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
                </el-col>
              </el-row>
              <el-form-item label="描述">
                <el-input v-model="formData.description" type="textarea" :rows="2" placeholder="可选" />
              </el-form-item>
            </div>

            <div class="form-section">
              <div class="form-section-header">
                <div class="form-section-title-row">
                  <h3 class="form-section-title">任务列表</h3>
                  <el-button type="primary" size="small" @click="addTask">
                    <Icon icon="mdi:plus" :size="14" /> 添加任务
                  </el-button>
                </div>
                <el-radio-group v-model="editMode" size="small">
                  <el-radio-button value="form">
                    <Icon icon="mdi:form-textbox" :size="14" /> 表单
                  </el-radio-button>
                  <el-radio-button value="yaml">
                    <Icon icon="mdi:code-json" :size="14" /> YAML
                  </el-radio-button>
                </el-radio-group>
              </div>

              <!-- 表单模式 -->
              <div v-if="editMode === 'form'" class="task-list">
                <div v-for="(task, ti) in formData.tasks" :key="ti" class="task-card">
                  <div class="task-card-header">
                    <div class="task-card-title">
                      <span class="task-index">{{ task.ref }}</span>
                      <el-input v-model="task.name" placeholder="任务名称" class="task-name-input" />
                    </div>
                    <div class="task-card-actions">
                      <el-button
                        v-if="ti > 0"
                        type="info"
                        link
                        size="default"
                        @click="moveTask(ti, -1)"
                      >
                        <Icon icon="mdi:arrow-up-bold" :size="16" /> 上移
                      </el-button>
                      <el-button
                        v-if="ti < formData.tasks.length - 1"
                        type="info"
                        link
                        size="default"
                        @click="moveTask(ti, 1)"
                      >
                        <Icon icon="mdi:arrow-down-bold" :size="16" /> 下移
                      </el-button>
                      <el-button
                        type="danger"
                        link
                        size="default"
                        @click="removeTask(ti)"
                      >
                        <Icon icon="mdi:delete" :size="16" /> 删除
                      </el-button>
                    </div>
                  </div>
                  <div class="task-card-body">
                    <el-row :gutter="16">
                      <el-col :span="4">
                        <el-form-item label="引用ID" class="task-field">
                          <el-input-number v-model="task.ref" :min="1" style="width: 100%" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item label="模块类型" class="task-field">
                          <el-select v-model="task.module" placeholder="选择模块" style="width: 100%" @change="resetTaskParams(task)">
                            <el-option label="Shell" value="shell" />
                            <el-option label="Systemd" value="systemd" />
                            <el-option label="Package" value="package" />
                            <el-option label="File" value="file" />
                            <el-option label="Template" value="template" />
                            <el-option label="Repo" value="repo" />
                            <el-option label="Blockinfile" value="blockinfile" />
                            <el-option label="Modprobe" value="modprobe" />
                          </el-select>
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item class="task-field">
                          <template #label>
                            <el-tooltip content="0 表示不限制超时" placement="top">
                              <span style="cursor: help">超时(秒) <Icon icon="mdi:information-outline" :size="12" style="vertical-align: middle" /></span>
                            </el-tooltip>
                          </template>
                          <el-input-number v-model="task.timeout" :min="0" :max="3600" style="width: 100%" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item label="重试次数" class="task-field">
                          <el-input-number v-model="task.retries" :min="0" :max="10" style="width: 100%" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="5">
                        <el-form-item label="重试间隔(秒)" class="task-field">
                          <el-input-number v-model="task.delay" :min="0" :max="60" style="width: 100%" />
                        </el-form-item>
                      </el-col>
                    </el-row>

                    <div class="params-section">
                      <el-form-item label="执行参数" class="task-field">
                        <template #label>
                          <span>执行参数</span>
                        </template>
                        <div class="params-kv-list">
                          <div v-for="(key, pi) in Object.keys(task.params)" :key="pi" class="params-kv-row">
                            <span class="params-kv-key">{{ key }}</span>
                            <el-input
                              v-model="task.params[key]"
                              :placeholder="getParamPlaceholder(task.module, key)"
                              class="params-kv-value"
                            />
                          </div>
                          <div v-if="Object.keys(task.params).length === 0" class="params-empty">
                            请先选择模块类型
                          </div>
                        </div>
                      </el-form-item>
                    </div>

                    <el-form-item label="执行条件" class="task-field">
                      <el-input v-model="task.when" placeholder='如: {{.machine.os_name}} !contains ubuntu' style="width: 100%" />
                    </el-form-item>
                    <el-row :gutter="16">
                      <el-col :span="8">
                        <el-form-item label="后置钩子" class="task-field">
                          <el-input v-model="task.hook_ids" placeholder='如: [1, 3]' />
                        </el-form-item>
                      </el-col>
                      <el-col :span="8">
                        <el-form-item label="注册变量" class="task-field">
                          <el-input v-model="task.register" placeholder="变量名" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="8">
                        <el-form-item label="忽略错误" class="task-field">
                          <el-checkbox v-model="task.ignore_errors">即使本任务失败也继续执行后续任务</el-checkbox>
                        </el-form-item>
                      </el-col>
                    </el-row>
                  </div>
                </div>
                <div v-if="formData.tasks.length === 0" class="empty-tip">
                  点击「添加任务」配置此阶段的操作
                </div>
              </div>

              <!-- YAML 模式 -->
              <div v-else class="yaml-editor">
                <el-input
                  v-model="yamlContent"
                  type="textarea"
                  :rows="20"
                  placeholder="在此编辑 YAML 格式的完整配置（基本信息 + 任务列表）"
                  class="yaml-textarea"
                />
                <div class="yaml-actions">
                  <el-button size="small" @click="formatYaml">
                    <Icon icon="mdi:format-align-left" :size="14" /> 格式化
                  </el-button>
                  <el-button size="small" type="warning" @click="yamlToForm">
                    <Icon icon="mdi:transfer-right" :size="14" /> 应用到表单
                  </el-button>
                </div>
              </div>
            </div>
          </el-form>
        </div>
      </div>
    </template>

    <!-- 全屏版本历史视图 -->
    <template v-if="viewMode === 'versions'">
      <div class="fullscreen-view">
        <div class="fullscreen-header">
          <div class="fullscreen-header-left">
            <el-button text @click="exitFullscreen">
              <Icon icon="mdi:arrow-left" :size="20" /> 返回列表
            </el-button>
            <el-divider direction="vertical" />
            <span class="fullscreen-title">版本历史 - {{ versionStage?.name }}</span>
            <el-tag size="small" type="info" effect="plain">
              当前版本: {{ versionStage?.version }}
            </el-tag>
          </div>
        </div>

        <div class="fullscreen-content version-layout">
          <div class="version-sidebar" v-loading="versionLoading">
            <div class="version-list">
              <div
                v-for="v in versionList"
                :key="v.id"
                class="version-item"
                :class="{ 'is-current': v.version === versionStage?.version, 'is-selected': selectedVersion?.id === v.id }"
                @click="selectVersion(v)"
              >
                <div class="version-item-header">
                  <el-tag size="small" :type="v.version === versionStage?.version ? 'primary' : 'info'" effect="plain">
                    {{ v.version }}
                  </el-tag>
                  <span v-if="v.version === versionStage?.version" class="current-badge">当前</span>
                </div>
                <p class="version-note">{{ v.change_note || '无描述' }}</p>
                <span class="version-time">{{ formatTime(v.created_at) }}</span>
              </div>
            </div>
          </div>

          <div class="version-detail">
            <template v-if="selectedVersion">
              <div class="version-detail-header">
                <h3>版本 {{ selectedVersion.version }} 详情</h3>
                <el-button
                  v-if="selectedVersion.version !== versionStage?.version"
                  type="warning"
                  size="small"
                  @click="handleUpdateToVersion(selectedVersion)"
                >
                  <Icon icon="mdi:backup-restore" :size="14" /> 更新到此版本
                </el-button>
              </div>

              <div class="version-detail-content">
                <el-descriptions :column="2" border size="small">
                  <el-descriptions-item label="版本号">{{ selectedVersion.version }}</el-descriptions-item>
                  <el-descriptions-item label="修改描述">{{ selectedVersion.change_note || '无描述' }}</el-descriptions-item>
                  <el-descriptions-item label="阶段名称">{{ selectedVersion.name }}</el-descriptions-item>
                  <el-descriptions-item label="目标分组">{{ getMachineGroupName(selectedVersion.machine_group_id) }}</el-descriptions-item>
                  <el-descriptions-item label="创建时间" :span="2">{{ formatTime(selectedVersion.created_at) }}</el-descriptions-item>
                </el-descriptions>

                <h4 class="detail-section-title">任务列表</h4>
                <div class="version-task-list">
                  <div v-for="(task, ti) in getVersionTasks(selectedVersion)" :key="ti" class="version-task-item">
                    <div class="version-task-header">
                      <span class="task-index">#{{ task.ref }}</span>
                      <span class="version-task-name">{{ task.name || '未命名' }}</span>
                      <el-tag size="small" effect="plain">{{ task.module }}</el-tag>
                    </div>
                    <div class="version-task-body">
                      <div v-if="task.params" class="version-task-params">
                        <span class="params-label">参数:</span>
                        <code>{{ task.params }}</code>
                      </div>
                      <div v-if="task.when" class="version-task-when">
                        <span class="params-label">条件:</span>
                        <code>{{ task.when }}</code>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </template>
            <div v-else class="version-detail-empty">
              <Icon icon="mdi:information-outline" :size="48" />
              <p>选择左侧版本查看详情</p>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 保存确认对话框（仅编辑已有阶段时弹出） -->
    <el-dialog
      v-model="saveDialogVisible"
      title="保存为新版本"
      width="500px"
      destroy-on-close
    >
      <div class="save-dialog-content">
        <div class="save-version-info">
          <div class="save-version-row">
            <span class="save-label">当前版本:</span>
            <el-tag size="small" type="info" effect="plain">
              {{ currentStage?.version }}
            </el-tag>
          </div>
          <div class="save-version-row">
            <span class="save-label">保存后:</span>
            <el-tag size="small" type="success" effect="plain">
              将生成新版本
            </el-tag>
          </div>
        </div>
        <el-form :model="saveFormData" label-width="100px" ref="saveFormRef" :rules="saveFormRules">
          <el-form-item label="修改描述" prop="change_note">
            <el-input
              v-model="saveFormData.change_note"
              type="textarea"
              :rows="3"
              placeholder="请描述本次修改内容，如：添加了安装 Docker 任务"
            />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="saveDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmSave" :loading="submitting">
          确认保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as yaml from 'js-yaml'
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
import { HandledError } from '@/utils/request'

interface StageTask {
  ref: number
  name: string
  module: string
  params: Record<string, string>
  order: number
  timeout: number
  retries: number
  delay: number
  when: string
  hook_ids: string
  register: string
  ignore_errors: boolean
}

// 每种模块支持的参数 key 和占位说明
const MODULE_PARAMS: Record<string, Record<string, string>> = {
  shell: { command: '执行的命令', script: '脚本内容' },
  systemd: { name: '服务名称', action: 'start/stop/restart/enable/disable' },
  package: { name: '包名', state: 'present/absent/latest' },
  file: { src: '源文件路径', dest: '目标路径' },
  template: { src: '模板路径', dest: '目标路径' },
  repo: { name: '仓库名', state: 'present/absent' },
  blockinfile: { path: '文件路径', block: '插入的内容', marker: '标记注释', insertafter: '插入位置' },
  modprobe: { name: '模块名', state: 'present/absent' },
}

function getParamPlaceholder(module: string, key: string): string {
  return MODULE_PARAMS[module]?.[key] || ''
}

function resetTaskParams(task: StageTask) {
  const keys = Object.keys(MODULE_PARAMS[task.module] || {})
  const newParams: Record<string, string> = {}
  for (const k of keys) newParams[k] = ''
  task.params = newParams
}

const loading = ref(false)
const searchText = ref('')
const stages = ref<StageTemplate[]>([])
const machineGroups = ref<MachineGroup[]>([])
const machineGroupLoading = ref(false)

// 视图模式
const viewMode = ref<'list' | 'edit' | 'versions'>('list')

// 全屏编辑
const editingId = ref(0)
const currentStage = ref<StageTemplate | null>(null)
const formData = ref({
  name: '',
  description: '',
  machine_group_id: 0,
  tasks: [] as StageTask[],
})
const formRules = {
  name: [{ required: true, message: '请输入阶段名称', trigger: 'blur' }],
  machine_group_id: [{ required: true, message: '请选择目标分组', trigger: 'change' }],
}

// 编辑模式：form / yaml
const editMode = ref<'form' | 'yaml'>('form')
const yamlContent = ref('')

// 保存对话框（仅编辑已有阶段）
const saveDialogVisible = ref(false)
const saveFormData = ref({ change_note: '' })
const saveFormRules = {
  change_note: [{ required: true, message: '请描述本次修改内容', trigger: 'blur' }],
}
const saveFormRef = ref()

const submitting = ref(false)
const formRef = ref()

// 版本历史
const versionStage = ref<StageTemplate | null>(null)
const versionLoading = ref(false)
const versionList = ref<StageTemplateVersion[]>([])
const selectedVersion = ref<StageTemplateVersion | null>(null)

const filteredStages = computed(() => {
  if (!searchText.value) return stages.value
  const kw = searchText.value.toLowerCase()
  return stages.value.filter(
    (s) => s.name.toLowerCase().includes(kw) || (s.description || '').toLowerCase().includes(kw)
  )
})

function normalizeTasks(rawTasks: any[]): StageTask[] {
  return rawTasks.map((t: any, i: number) => {
    let params: Record<string, string> = {}
    if (typeof t.params === 'object' && t.params !== null) {
      // 新格式：直接是对象
      params = {}
      for (const [k, v] of Object.entries(t.params)) {
        params[k] = String(v ?? '')
      }
    } else if (typeof t.params === 'string' && t.params.trim()) {
      // 旧格式：JSON 字符串，尝试解析
      try {
        const parsed = JSON.parse(t.params)
        if (typeof parsed === 'object' && parsed !== null) {
          params = {}
          for (const [k, v] of Object.entries(parsed)) {
            params[k] = String(v ?? '')
          }
        } else {
          params = { command: t.params }
        }
      } catch {
        params = { command: t.params }
      }
    } else {
      params = { command: '' }
    }
    return {
      ...t,
      params,
      order: t.order || i + 1,
    }
  })
}

function getTasks(stage: StageTemplate): StageTask[] {
  try {
    const raw = JSON.parse(stage.tasks || '[]') as any[]
    return normalizeTasks(raw)
  } catch {
    return []
  }
}

function getVersionTasks(version: StageTemplateVersion): StageTask[] {
  try {
    const raw = JSON.parse(version.tasks || '[]') as any[]
    return normalizeTasks(raw)
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

// ==================== 全屏视图控制 ====================

function exitFullscreen() {
  viewMode.value = 'list'
  editingId.value = 0
  currentStage.value = null
  versionStage.value = null
  selectedVersion.value = null
  editMode.value = 'form'
  yamlContent.value = ''
}

// 创建阶段：直接进入全屏编辑
function showCreateEditor() {
  editingId.value = 0
  currentStage.value = null
  formData.value = { name: '', description: '', machine_group_id: 0, tasks: [] }
  editMode.value = 'form'
  yamlContent.value = ''
  viewMode.value = 'edit'
  loadMachineGroups()
}

// 编辑阶段：进入全屏编辑
function editStage(stage: StageTemplate) {
  editingId.value = stage.id
  currentStage.value = stage
  formData.value = {
    name: stage.name,
    description: stage.description,
    machine_group_id: stage.machine_group_id,
    tasks: getTasks(stage),
  }
  editMode.value = 'form'
  yamlContent.value = ''
  viewMode.value = 'edit'
  loadMachineGroups()
}

// ==================== 任务管理 ====================

function addTask() {
  const maxRef = formData.value.tasks.reduce((max: number, t: StageTask) => Math.max(max, t.ref), 0)
  formData.value.tasks.push({
    ref: maxRef + 1,
    name: '',
    module: 'shell',
    params: { command: '' },
    order: formData.value.tasks.length + 1,
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
  formData.value.tasks.forEach((t, i) => { t.order = i + 1 })
}

function moveTask(index: number, direction: -1 | 1) {
  const tasks = formData.value.tasks
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= tasks.length) return
  const temp = tasks[index]
  tasks[index] = tasks[newIndex]
  tasks[newIndex] = temp
  tasks.forEach((t, i) => { t.order = i + 1 })
}

function validateTasks(tasks: StageTask[]): boolean {
  if (tasks.length === 0) {
    ElMessage.warning('至少需要一个任务')
    return false
  }
  const refSet = new Set<number>()
  for (const task of tasks) {
    if (!task.ref || task.ref <= 0) {
      ElMessage.warning('任务引用ID必须大于0')
      return false
    }
    if (refSet.has(task.ref)) {
      ElMessage.warning(`任务引用ID ${task.ref} 重复`)
      return false
    }
    refSet.add(task.ref)
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

// ==================== YAML 转换 ====================

function formToYaml() {
  try {
    const obj: Record<string, any> = {
      name: formData.value.name,
      machine_group_id: formData.value.machine_group_id,
    }
    if (formData.value.description) obj.description = formData.value.description
    obj.tasks = formData.value.tasks.map(t => {
      const task: Record<string, any> = {
        ref: t.ref,
        name: t.name,
        module: t.module,
        order: t.order,
      }
      if (t.timeout) task.timeout = t.timeout
      if (t.retries) task.retries = t.retries
      if (t.delay) task.delay = t.delay
      if (t.params) task.params = t.params
      if (t.when) task.when = t.when
      if (t.hook_ids) task.hook_ids = t.hook_ids
      if (t.register) task.register = t.register
      if (t.ignore_errors) task.ignore_errors = t.ignore_errors
      return task
    })
    yamlContent.value = yaml.dump(obj, { indent: 2, lineWidth: -1 })
  } catch {
    yamlContent.value = '{}'
  }
}

function yamlToForm() {
  try {
    const parsed = yaml.load(yamlContent.value) as Record<string, any>
    if (!parsed || typeof parsed !== 'object') {
      ElMessage.error('YAML 格式错误：应为对象')
      return
    }
    if (parsed.name !== undefined) formData.value.name = String(parsed.name)
    if (parsed.description !== undefined) formData.value.description = String(parsed.description)
    if (parsed.machine_group_id !== undefined) formData.value.machine_group_id = Number(parsed.machine_group_id)
    if (Array.isArray(parsed.tasks)) {
      formData.value.tasks = normalizeTasks(parsed.tasks)
    }
    editMode.value = 'form'
    ElMessage.success('已应用到表单')
  } catch (e: any) {
    ElMessage.error('YAML 解析失败: ' + (e.message || ''))
  }
}

function formatYaml() {
  try {
    const parsed = yaml.load(yamlContent.value)
    yamlContent.value = yaml.dump(parsed, { indent: 2, lineWidth: -1 })
  } catch (e: any) {
    ElMessage.error('YAML 格式化失败: ' + (e.message || ''))
  }
}

// 切换到 YAML 模式时自动转换
watch(editMode, (mode) => {
  if (mode === 'yaml') {
    formToYaml()
  }
})

// ==================== 保存 ====================

function handleSave() {
  try {
    formRef.value?.validate()
  } catch {
    return
  }
  // 如果当前是 YAML 模式，先应用到表单
  if (editMode.value === 'yaml') {
    yamlToForm()
    if (editMode.value === 'yaml') return // 解析失败则中断
  }
  if (!validateTasks(formData.value.tasks)) return

  // 编辑已有阶段：弹出修改描述对话框
  if (editingId.value) {
    saveDialogVisible.value = true
    saveFormData.value.change_note = ''
  } else {
    // 创建新阶段：直接提交
    doCreate()
  }
}

async function doCreate() {
  submitting.value = true
  try {
    await createStageTemplateApi({
      name: formData.value.name,
      description: formData.value.description,
      machine_group_id: formData.value.machine_group_id,
      tasks: JSON.stringify(formData.value.tasks.map(t => ({ ...t, params: JSON.stringify(t.params) }))),
    })
    ElMessage.success('创建成功')
    exitFullscreen()
    loadData()
  } catch (e: any) {
    if (!(e instanceof HandledError)) ElMessage.error(e?.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

async function confirmSave() {
  try {
    await saveFormRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    await updateStageTemplateApi(editingId.value, {
      name: formData.value.name,
      description: formData.value.description,
      machine_group_id: formData.value.machine_group_id,
      tasks: JSON.stringify(formData.value.tasks.map(t => ({ ...t, params: JSON.stringify(t.params) }))),
      change_note: saveFormData.value.change_note,
    })
    ElMessage.success('保存成功，已生成新版本')
    saveDialogVisible.value = false
    exitFullscreen()
    loadData()
  } catch (e: any) {
    if (!(e instanceof HandledError)) ElMessage.error(e?.message || '保存失败')
  } finally {
    submitting.value = false
  }
}

// ==================== 删除 ====================

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
  selectedVersion.value = null
  viewMode.value = 'versions'
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

function selectVersion(version: StageTemplateVersion) {
  selectedVersion.value = version
}

async function handleUpdateToVersion(version: StageTemplateVersion) {
  try {
    await ElMessageBox.confirm(
      `确定要更新到版本 ${version.version} 吗？\n\n当前版本将被保存为历史版本，系统会生成一个新的当前版本。`,
      '更新确认',
      { confirmButtonText: '确定更新', cancelButtonText: '取消', type: 'warning' }
    )
    await rollbackStageTemplateApi(versionStage.value!.id, version.version)
    ElMessage.success(`已更新到 ${version.version}`)
    exitFullscreen()
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

/* 全屏视图样式 */
.fullscreen-view {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--el-bg-color);
  z-index: 1000;
  display: flex;
  flex-direction: column;
}

.fullscreen-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.fullscreen-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.fullscreen-header-right {
  display: flex;
  gap: 8px;
}

.fullscreen-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color-primary);
}

.fullscreen-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.form-section {
  margin-bottom: 24px;
}

.form-section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color-primary);
  margin-bottom: 16px;
}

.form-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.form-section-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.form-section-title-row .form-section-title {
  margin-bottom: 0;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.task-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 16px;
  background: var(--el-bg-color);
}

.task-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.task-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  padding-left: 90px;
}

.task-card-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.task-card-actions :deep(.el-button) {
  font-size: 13px;
  font-weight: 500;
}

.task-index {
  width: 24px;
  height: 24px;
  border-radius: 4px;
  background: var(--el-color-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}

.task-name-input {
  flex: 1;
}

.task-card-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.task-field {
  margin-bottom: 8px;
}

.task-field :deep(.el-form-item__label) {
  font-size: 12px;
  color: var(--text-color-secondary);
  font-weight: 500;
}

.empty-tip {
  text-align: center;
  color: var(--text-color-secondary);
  font-size: 13px;
  padding: 40px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
}

/* YAML 编辑器样式 */
.yaml-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.yaml-textarea :deep(textarea) {
  font-family: monospace;
  font-size: 13px;
  line-height: 1.6;
}

.yaml-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

/* 版本历史视图样式 */
.version-layout {
  display: flex;
  gap: 24px;
  padding: 24px;
}

.version-sidebar {
  width: 320px;
  flex-shrink: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}

.version-list {
  max-height: calc(100vh - 120px);
  overflow-y: auto;
}

.version-item {
  padding: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  cursor: pointer;
  transition: background 0.2s;
}

.version-item:hover {
  background: var(--el-fill-color-light);
}

.version-item.is-current {
  background: var(--el-color-primary-light-9);
}

.version-item.is-selected {
  background: var(--el-color-primary-light-7);
}

.version-item-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.current-badge {
  font-size: 12px;
  color: var(--el-color-primary);
  font-weight: 600;
}

.version-note {
  font-size: 13px;
  color: var(--text-color-secondary);
  margin: 0 0 4px 0;
}

.version-time {
  font-size: 12px;
  color: var(--text-color-placeholder);
}

.version-detail {
  flex: 1;
  min-width: 0;
}

.version-detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.version-detail-header h3 {
  margin: 0;
  font-size: 18px;
  color: var(--text-color-primary);
}

.version-detail-content {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 24px;
}

.detail-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color-primary);
  margin: 24px 0 12px 0;
}

.version-task-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.version-task-item {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 12px;
}

.version-task-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.version-task-name {
  flex: 1;
  font-weight: 500;
}

.version-task-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}

.version-task-params,
.version-task-when {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.params-label {
  color: var(--text-color-secondary);
  flex-shrink: 0;
}

.version-task-params code,
.version-task-when code {
  background: var(--el-fill-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
}

.version-detail-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-color-secondary);
}

/* 保存对话框样式 */
.save-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.save-version-info {
  background: var(--el-fill-color-light);
  border-radius: 8px;
  padding: 16px;
}

.save-version-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0;
}

.save-label {
  font-size: 14px;
  color: var(--text-color-secondary);
  min-width: 80px;
}

/* 执行参数 key-value 表单 */
.params-section {
  margin-bottom: 8px;
}

.params-kv-list {
  width: 100%;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  overflow: hidden;
}

.params-kv-row {
  display: flex;
  align-items: center;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.params-kv-row:last-child {
  border-bottom: none;
}

.params-kv-key {
  min-width: 120px;
  max-width: 160px;
  padding: 8px 12px;
  background: var(--el-fill-color-lighter);
  font-family: monospace;
  font-size: 13px;
  color: var(--el-color-primary);
  font-weight: 500;
  border-right: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}

.params-kv-value {
  flex: 1;
}

.params-kv-value :deep(.el-input__wrapper) {
  box-shadow: none !important;
  border-radius: 0;
}

.params-kv-value :deep(.el-input__inner) {
  font-family: monospace;
  font-size: 13px;
}

.params-empty {
  padding: 16px;
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
</style>
