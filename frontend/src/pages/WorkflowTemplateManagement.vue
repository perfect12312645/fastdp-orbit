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
            <el-select v-model="selectedGroup" placeholder="全部来源" clearable style="width: 160px;">
              <el-option v-for="g in availableGroups" :key="g" :label="g || '(默认)'" :value="g" />
            </el-select>
          </div>
          <div class="table-toolbar-right">
            <span class="total-text">共 {{ filteredTemplates.length }} 个模板</span>
          </div>
        </div>

        <el-table :data="paginatedTemplates" v-loading="loading" stripe>
          <el-table-column label="名称" prop="name" min-width="150" />
          <el-table-column label="描述" prop="description" min-width="200" show-overflow-tooltip />
          <el-table-column label="来源" width="120">
            <template #default="{ row }">
              <el-tag v-if="row.source" size="small" type="info" effect="plain">{{ row.source }}</el-tag>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>
        <el-table-column label="更新时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.updated_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button type="success" link size="small" @click="showPreviewDialog(row)">
              <Icon icon="mdi:eye" :size="14" /> 预览
            </el-button>
            <el-button type="primary" link size="small" @click="editTemplate(row)">
              <Icon icon="mdi:pencil" :size="14" /> 编辑
            </el-button>
            <el-button type="danger" link size="small" @click="deleteTemplate(row)">
              <Icon icon="mdi:delete-outline" :size="14" />
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper" v-if="filteredTemplates.length > pageSize">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="filteredTemplates.length"
          layout="total, sizes, prev, pager, next"
          @size-change="currentPage = 1"
        />
      </div>
    </div>

    <!-- 全屏编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      fullscreen
      destroy-on-close
      :show-close="false"
      class="template-editor-dialog"
    >
      <template #header>
        <div class="template-editor-header">
          <span>{{ editingId ? '编辑模板' : '创建模板' }}</span>
          <div class="template-editor-header-actions">
            <el-button @click="dialogVisible = false" size="small">取消</el-button>
            <el-button type="primary" @click="handleSubmit" :loading="submitting" size="small">
              {{ editingId ? '保存' : '创建' }}
            </el-button>
          </div>
        </div>
      </template>

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
                  <el-button size="small" @click="showPreviewDialog">
                    <Icon icon="mdi:eye" :size="14" /> 预览
                  </el-button>
                </div>
                <codemirror
                  v-model="form.content"
                  :style="{ height: '100%', width: '100%' }"
                  :extensions="codemirrorExtensions"
                  :tab-size="2"
                  :indent-with-tab="true"
                  placeholder="使用 Go template 语法，如：{{.Machine.ip}}、{{.GlobalVariable.key}}"
                  @ready="onEditorReady"
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
              <div class="var-ref-item" v-for="g in machineGroupsForPicker.slice(0, 2)" :key="g.name">
                <code @click="insertGroupsLoop(g.name)">{{ groupsVarText(g.name) }}</code>
                <span>{{ g.count }} 台机器</span>
              </div>
              <div class="var-ref-hint" v-if="machineGroupsForPicker.length > 2">
                共 {{ machineGroupsForPicker.length }} 个分组，使用「插入变量」按钮查看全部
              </div>
              <div class="var-ref-hint">
                循环内可用属性：ip, hostname, os_name, os_version, arch, kernel, cpu_model, cpu_cores, memory_kb, swap_kb, gateway, virtualization, timezone<br>
                嵌套列表：gpus[].name/count/driver_version、disks[].device/type/total_gb、networks[].name/ip/mac/speed/status
              </div>
            </div>
            <div class="var-ref-group">
              <div class="var-ref-title">全局变量</div>
              <div class="var-ref-item" v-for="v in globalVars.slice(0, 2)" :key="v.key">
                <code @click="insertVariable(globalVarExpr(v.key))">{{ globalVarText(v.key) }}</code>
                <span>{{ v.description || v.key }}</span>
              </div>
              <div class="var-ref-hint" v-if="globalVars.length > 2">
                共 {{ globalVars.length }} 个变量，使用「插入变量」按钮查看全部
              </div>
              <div class="var-ref-hint" v-else-if="globalVars.length === 0">
                暂无全局变量，可在「全局变量」页面创建
              </div>
            </div>

            <h4 style="margin-top: 16px;">自定义函数</h4>
            <div class="var-ref-group">
              <div class="var-ref-title">文件与编码</div>
              <div class="var-ref-item">
                <code @click="insertVariable(funcExpr('lookup', '/path/to/file'))">lookup</code>
                <span>读取文件内容</span>
              </div>
              <div class="var-ref-hint">
                示例：&#123;&#123; lookup "/etc/kubernetes/pki/ca.pem" &#125;&#125;<br>
                管道：&#123;&#123; lookup "/path" | b64encode &#125;&#125;
              </div>
              <div class="var-ref-item">
                <code @click="insertVariable(funcExpr('b64encode', 'text'))">b64encode</code>
                <span>Base64编码</span>
              </div>
              <div class="var-ref-item">
                <code @click="insertVariable(funcExpr('lower', 'STRING'))">lower</code>
                <span>转小写</span>
              </div>
            </div>
            <div class="var-ref-group">
              <div class="var-ref-title">常用管道语法</div>
              <div class="var-ref-hint">
                <code>default</code>：&#123;&#123; .Key | default "N/A" &#125;&#125;<br>
                <code>printf</code>：&#123;&#123; printf "%s:%d" .ip .port &#125;&#125;<br>
                <code>if</code>：&#123;&#123; if .Key &#125;&#125;...&#123;&#123; else &#125;&#125;...&#123;&#123; end &#125;&#125;<br>
                <code>range</code>：&#123;&#123; range .List &#125;&#125;...&#123;&#123; end &#125;&#125;<br>
                <code>index</code>：&#123;&#123; index .Groups "name" 0 &#125;&#125;
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 预览对话框 -->
    <el-dialog
      v-model="previewVisible"
      title="模板预览"
      width="700px"
      destroy-on-close
    >
      <div class="preview-toolbar">
        <span style="font-size: 13px; color: var(--el-text-color-secondary);">选择机器（使用该机器数据渲染）：</span>
        <el-select-v2
          v-model="previewMachineId"
          :options="machineOptions"
          placeholder="选择机器"
          filterable
          clearable
          style="width: 300px; margin-left: 8px;"
          @change="doPreview"
        />
      </div>
      <div class="preview-result" v-loading="previewLoading">
        <pre v-if="previewContent">{{ previewContent }}</pre>
        <div v-else class="preview-empty">选择机器后自动渲染预览</div>
      </div>
      <template #footer>
        <el-button @click="previewVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
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
  previewWorkflowTemplateApi,
  type WorkflowTemplate,
} from '@/api/workflowTemplate'
import VariablePicker from '@/components/VariablePicker.vue'

const codemirrorExtensions = [oneDark]

const loading = ref(false)
const searchText = ref('')
const selectedGroup = ref('')
const templates = ref<WorkflowTemplate[]>([])
const globalVars = ref<GlobalVariable[]>([])
const machineGroups = ref<MachineGroup[]>([])
const currentPage = ref(1)
const pageSize = ref(20)

const filteredTemplates = computed(() => {
  let result = templates.value
  if (selectedGroup.value) {
    result = result.filter(t => t.source === selectedGroup.value)
  }
  if (searchText.value) {
    const kw = searchText.value.toLowerCase()
    result = result.filter(
      (t) => t.name.toLowerCase().includes(kw) || t.description.toLowerCase().includes(kw)
    )
  }
  return result
})

const availableGroups = computed(() => {
  const groups = new Set(templates.value.map(t => t.source).filter(Boolean))
  return Array.from(groups).sort()
})

const paginatedTemplates = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredTemplates.value.slice(start, start + pageSize.value)
})

watch([searchText, selectedGroup], () => {
  currentPage.value = 1
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

function funcExpr(name: string, arg: string) {
  return `{{ ${name} "${arg}" }}`
}

// ==================== 编辑器 ====================
const editorView = ref<any>(null)

function onEditorReady(payload: { view: any }) {
  editorView.value = payload.view
}

function insertVariable(expr: string) {
  if (editorView.value) {
    // 在光标位置插入
    const view = editorView.value
    const pos = view.state.selection.main.head
    view.dispatch({
      changes: { from: pos, insert: expr },
      selection: { anchor: pos + expr.length },
    })
    // 同步到 form.content
    form.value.content = view.state.doc.toString()
  } else {
    // 降级：追加到末尾
    form.value.content += expr
  }
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

// 预览相关
const previewVisible = ref(false)
const previewMachineId = ref(0)
const previewContent = ref('')
const previewLoading = ref(false)
const previewTemplate = ref<WorkflowTemplate | null>(null)

// 机器列表（用于预览选择）
const allMachines = ref<any[]>([])

const machineOptions = computed(() => [
  { value: 0, label: '使用示例数据' },
  ...allMachines.value.map((m: any) => ({
    value: m.id,
    label: `${m.hostname || m.ip} (${m.ip}:${m.port})`,
  }))
])

function showPreviewDialog(row: WorkflowTemplate) {
  previewTemplate.value = row
  previewMachineId.value = 0
  previewContent.value = ''
  previewVisible.value = true
  // 加载机器列表
  loadMachinesForPreview()
}

async function loadMachinesForPreview() {
  try {
    const { getMachinesApi } = await import('@/api/machine')
    allMachines.value = await getMachinesApi()
  } catch {
    allMachines.value = []
  }
  doPreview()
}

async function doPreview() {
  const content = previewTemplate.value?.content || form.value.content
  if (!content?.trim()) {
    previewContent.value = ''
    return
  }
  previewLoading.value = true
  try {
    previewContent.value = await previewWorkflowTemplateApi(content, previewMachineId.value || undefined)
  } catch (e: any) {
    previewContent.value = '渲染失败: ' + (e?.message || '未知错误')
  } finally {
    previewLoading.value = false
  }
}

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
.page-subtitle {
  font-size: var(--font-size-sm);
  color: var(--text-color-secondary);
  margin-top: 4px;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.template-editor-dialog :deep(.el-dialog__body) {
  padding: 0;
  height: calc(100vh - 106px);
  overflow: hidden;
}

.template-editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.template-editor-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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

/* 预览样式 */
.preview-toolbar {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.preview-result {
  min-height: 200px;
  max-height: 500px;
  overflow: auto;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  padding: 12px;
}

.preview-result pre {
  margin: 0;
  font-family: monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}

.preview-empty {
  text-align: center;
  color: var(--el-text-color-placeholder);
  padding: 40px;
  font-size: 13px;
}
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
