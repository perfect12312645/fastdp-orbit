<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>模板文件管理</h2>
        <p class="page-subtitle">可复用的配置模板，供阶段编排中 template 模块选取</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateDialog">
          <Icon icon="mdi:plus" :size="16" /> 创建模板
        </el-button>
      </div>
    </div>

    <div class="page-content">
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input v-model="searchText" placeholder="搜索模板名称" clearable style="width: 240px;">
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
        </div>
        <div class="table-toolbar-right">
          <span class="total-text">共 {{ filteredTemplates.length }} 个模板</span>
        </div>
      </div>

      <el-table :data="filteredTemplates" v-loading="loading" stripe>
        <el-table-column label="名称" prop="name" min-width="150" />
        <el-table-column label="描述" prop="description" min-width="200" show-overflow-tooltip />
        <el-table-column label="更新时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.updated_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="editTemplate(row)">
              <Icon icon="mdi:pencil" :size="14" /> 编辑
            </el-button>
            <el-button type="danger" link size="small" @click="deleteTemplate(row)">
              <Icon icon="mdi:delete-outline" :size="14" />
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 全屏编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑模板' : '创建模板'"
      fullscreen
      destroy-on-close
      class="template-editor-dialog"
    >
      <div class="template-editor-layout">
        <!-- 左侧编辑区 -->
        <div class="template-editor-main">
          <el-form :model="form" label-width="80px" ref="formRef" :rules="formRules">
            <el-row :gutter="16">
              <el-col :span="12">
                <el-form-item label="模板名称" prop="name">
                  <el-input v-model="form.name" placeholder="如：nginx.conf" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="描述">
                  <el-input v-model="form.description" placeholder="可选" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="模板内容" prop="content" class="editor-form-item">
              <div class="editor-wrapper">
                <div class="editor-toolbar">
                  <VariablePicker
                    button-type="primary"
                    button-text="插入变量"
                    :machine-groups="machineGroupsForPicker"
                    @select="(expr: string) => { insertVariable(expr) }"
                  />
                </div>
                <codemirror
                  v-model="form.content"
                  :style="{ height: '100%', width: '100%' }"
                  :extensions="codemirrorExtensions"
                  :tab-size="2"
                  :indent-with-tab="true"
                  placeholder="使用 Go template 语法，如：{{.Machine.ip}}、{{.GlobalVariable.key}}"
                />
              </div>
            </el-form-item>
          </el-form>
        </div>

        <!-- 右侧变量参考 -->
        <div class="template-editor-sidebar">
          <div class="sidebar-section">
            <h4>变量参考</h4>
            <div class="var-ref-group">
              <div class="var-ref-title">机器属性</div>
              <div class="var-ref-item" v-for="v in machineVars" :key="v.key">
                <code @click="insertVariable(machineVarExpr(v.key))">{{ machineVarText(v.key) }}</code>
                <span>{{ v.label }}</span>
              </div>
            </div>
            <div class="var-ref-group">
              <div class="var-ref-title">机器列表属性（循环）</div>
              <div class="var-ref-item">
                <code @click="insertVariable(gpuLoopExpr())">gpus</code>
                <span>GPU列表</span>
              </div>
              <div class="var-ref-item">
                <code @click="insertVariable(diskLoopExpr())">disks</code>
                <span>磁盘列表</span>
              </div>
              <div class="var-ref-item">
                <code @click="insertVariable(networkLoopExpr())">networks</code>
                <span>网卡列表</span>
              </div>
              <div class="var-ref-hint">
                循环内可用属性：<br>
                gpus: name, count, driver_version<br>
                disks: device, type, total_gb<br>
                networks: name, ip, mac, speed, status
              </div>
            </div>
            <div class="var-ref-group">
              <div class="var-ref-title">当前分组</div>
              <div class="var-ref-item">
                <code @click="insertVariable(groupVarExpr())">{{ groupVarText() }}</code>
                <span>分组名称</span>
              </div>
            </div>
            <div class="var-ref-group">
              <div class="var-ref-title">所有分组（循环）</div>
              <div class="var-ref-item" v-for="g in machineGroupsForPicker" :key="g.name">
                <code @click="insertGroupsLoop(g.name)">{{ groupsVarText(g.name) }}</code>
                <span>{{ g.count }} 台机器</span>
              </div>
              <div class="var-ref-hint">
                循环内可用属性：ip, hostname, os_name, os_version, arch, kernel, cpu_model, cpu_cores, memory_kb, swap_kb, gateway, virtualization, timezone<br>
                嵌套列表：gpus[].name/count/driver_version、disks[].device/type/total_gb、networks[].name/ip/mac/speed/status
              </div>
            </div>
            <div class="var-ref-group">
              <div class="var-ref-title">全局变量</div>
              <div class="var-ref-item" v-for="v in globalVars" :key="v.key">
                <code @click="insertVariable(globalVarExpr(v.key))">{{ globalVarText(v.key) }}</code>
                <span>{{ v.description || v.key }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          {{ editingId ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Codemirror } from 'vue-codemirror'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView } from '@codemirror/view'
import { formatDateTime } from '@/utils/format'
import { getGlobalVariablesApi, type GlobalVariable } from '@/api/globalVariable'
import { getMachineGroupsApi, type MachineGroup } from '@/api/machineGroup'
import {
  getWorkflowTemplatesApi,
  createWorkflowTemplateApi,
  updateWorkflowTemplateApi,
  deleteWorkflowTemplateApi,
  type WorkflowTemplate,
} from '@/api/workflowTemplate'
import VariablePicker from '@/components/VariablePicker.vue'

const codemirrorExtensions = [oneDark]

const loading = ref(false)
const searchText = ref('')
const templates = ref<WorkflowTemplate[]>([])
const globalVars = ref<GlobalVariable[]>([])
const machineGroups = ref<MachineGroup[]>([])

const filteredTemplates = computed(() => {
  if (!searchText.value) return templates.value
  const kw = searchText.value.toLowerCase()
  return templates.value.filter(
    (t) => t.name.toLowerCase().includes(kw) || t.description.toLowerCase().includes(kw)
  )
})

const machineGroupsForPicker = computed(() => {
  return machineGroups.value.map(g => ({
    name: g.name,
    count: g.machines?.length || 0,
  }))
})

const machineVars = [
  { key: 'ip', label: 'IP 地址' },
  { key: 'hostname', label: '主机名' },
  { key: 'os_name', label: '操作系统' },
  { key: 'os_version', label: '系统版本' },
  { key: 'arch', label: '架构' },
  { key: 'kernel', label: '内核版本' },
  { key: 'cpu_model', label: 'CPU 型号' },
  { key: 'cpu_cores', label: 'CPU 核数' },
  { key: 'memory_kb', label: '内存(KB)' },
  { key: 'swap_kb', label: 'Swap(KB)' },
  { key: 'gateway', label: '网关' },
  { key: 'virtualization', label: '虚拟化类型' },
  { key: 'timezone', label: '时区' },
]

// 变量文本生成函数
function machineVarExpr(key: string) {
  return `{{ .Machine.${key} }}`
}

function machineVarText(key: string) {
  return `{{ .Machine.${key} }}`
}

function gpuLoopExpr() {
  return `{{ range $gpuIdx, $gpu := .Machine.gpus }}
GPU{{$gpuIdx}}: {{$gpu.name}} 数量:{{$gpu.count}} 驱动:{{$gpu.driver_version}}
{{ end }}`
}

function diskLoopExpr() {
  return `{{ range $diskIdx, $disk := .Machine.disks }}
磁盘{{$diskIdx}}: {{$disk.device}} {{$disk.total_gb}}GB
{{ end }}`
}

function networkLoopExpr() {
  return `{{ range $netIdx, $net := .Machine.networks }}
网卡{{$netIdx}}: {{$net.ip}} {{$net.mac}}
{{ end }}`
}

function groupVarExpr() {
  return '{{ .Group.name }}'
}

function groupVarText() {
  return '{{ .Group.name }}'
}

function groupsVarExpr(name: string) {
  return `{{ .Groups.${name} }}`
}

function groupsVarText(name: string) {
  return `{{ .Groups.${name} }}`
}

function globalVarExpr(key: string) {
  return `{{ .GlobalVariable.${key} }}`
}

function globalVarText(key: string) {
  return `{{ .GlobalVariable.${key} }}`
}

// ==================== 编辑器 ====================
const editorView = ref<any>(null)

function insertVariable(expr: string) {
  form.value.content += expr
}

function insertGroupsLoop(groupName: string) {
  const template = `{{ range $index, $value := .Groups.${groupName} }}
IP: {{ $value.ip }}
主机名: {{ $value.hostname }}
CPU核心: {{ $value.cpu_cores }}

{{ range $gpuIdx, $gpu := $value.gpus }}
GPU{{$gpuIdx}}: {{$gpu.name}} 数量:{{$gpu.count}} 驱动:{{$gpu.driver_version}}
{{ end }}

{{ range $diskIdx, $disk := $value.disks }}
磁盘{{$diskIdx}}: {{$disk.device}} {{$disk.total_gb}}GB
{{ end }}

{{ range $netIdx, $net := $value.networks }}
网卡{{$netIdx}}: {{$net.ip}} {{$net.mac}}
{{ end }}
{{ end }}`
  insertVariable(template)
}

// ==================== CRUD ====================
const dialogVisible = ref(false)
const editingId = ref(0)
const submitting = ref(false)
const formRef = ref()
const form = ref({
  name: '',
  description: '',
  content: '',
})

const formRules = {
  name: [
    { required: true, message: '请输入模板名称', trigger: 'blur' },
    {
      validator: (_rule: any, value: string, callback: any) => {
        if (!value) { callback(); return }
        const existing = templates.value.find(t => t.name === value && t.id !== editingId.value)
        if (existing) {
          callback(new Error(`模板名称「${value}」已存在`))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  content: [
    { required: true, message: '请输入模板内容', trigger: 'blur' },
  ],
}

async function loadData() {
  loading.value = true
  try {
    const [templatesData, varsData, groupsData] = await Promise.all([
      getWorkflowTemplatesApi(),
      getGlobalVariablesApi().catch(() => []),
      getMachineGroupsApi().catch(() => []),
    ])
    templates.value = templatesData
    globalVars.value = varsData
    machineGroups.value = groupsData
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  editingId.value = 0
  form.value = { name: '', description: '', content: '' }
  dialogVisible.value = true
}

function editTemplate(row: WorkflowTemplate) {
  editingId.value = row.id
  form.value = {
    name: row.name,
    description: row.description,
    content: row.content,
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    if (editingId.value) {
      await updateWorkflowTemplateApi(editingId.value, form.value)
      ElMessage.success('保存成功')
    } else {
      await createWorkflowTemplateApi(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function deleteTemplate(row: WorkflowTemplate) {
  try {
    await ElMessageBox.confirm(`确认删除模板「${row.name}」？`, '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'error',
    })
    await deleteWorkflowTemplateApi(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

onMounted(loadData)
</script>

<style scoped>
.template-editor-dialog :deep(.el-dialog__body) {
  padding: 0;
  height: calc(100vh - 120px);
  overflow: hidden;
}

.template-editor-layout {
  display: flex;
  height: 100%;
}

.template-editor-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow: hidden;
}

.template-editor-main .el-form {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.editor-form-item {
  flex: 1;
  margin-bottom: 0;
}

.editor-form-item :deep(.el-form-item__content) {
  height: 100%;
}

.editor-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  overflow: hidden;
  width: 100%;
  max-width: 100%;
}

.editor-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}

.editor-wrapper :deep(.cm-editor) {
  height: 100%;
  width: 100%;
  max-width: 100%;
}

.editor-wrapper :deep(.cm-scroller) {
  overflow: auto;
}

.editor-wrapper :deep(.cm-content) {
  max-width: 100%;
}

.template-editor-sidebar {
  width: 320px;
  border-left: 1px solid var(--el-border-color);
  background: var(--el-fill-color-blank);
  overflow-y: auto;
  padding: 16px;
}

.sidebar-section h4 {
  margin: 0 0 16px 0;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.var-ref-group {
  margin-bottom: 16px;
}

.var-ref-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.var-ref-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  font-size: 12px;
}

.var-ref-item code {
  font-family: monospace;
  font-size: 11px;
  color: var(--el-color-primary);
  background: var(--el-fill-color);
  padding: 2px 6px;
  border-radius: 3px;
  cursor: pointer;
}

.var-ref-item code:hover {
  background: var(--el-color-primary-light-9);
}

.var-ref-item span {
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.var-ref-hint {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  margin-top: 4px;
  padding: 4px 8px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  line-height: 1.4;
}

.total-text {
  color: #909399;
  font-size: 13px;
}
</style>
