<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>钩子管理</h2>
        <p class="page-subtitle">可复用的后置钩子，供工作流任务引用</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateDialog">
          <Icon icon="mdi:plus" :size="16" /> 创建钩子
        </el-button>
      </div>
    </div>

    <div class="page-content">
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input v-model="searchText" placeholder="搜索钩子名称" clearable style="width: 240px;">
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
        </div>
        <div class="table-toolbar-right">
          <span class="total-text">共 {{ filteredTemplates.length }} 个钩子</span>
        </div>
      </div>

      <el-table :data="filteredTemplates" v-loading="loading" stripe>
        <el-table-column label="名称" prop="name" min-width="150" />
        <el-table-column label="模块" prop="module" width="120">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ row.module }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="描述" prop="description" min-width="200" show-overflow-tooltip />
        <el-table-column label="超时" width="80" align="center">
          <template #default="{ row }">
            {{ row.timeout || '不限' }}
          </template>
        </el-table-column>
        <el-table-column label="重试" width="70" align="center">
          <template #default="{ row }">
            {{ row.retries }}
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

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑钩子' : '创建钩子'"
      width="640px"
      destroy-on-close
    >
      <el-form :model="form" label-width="90px" ref="formRef" :rules="formRules">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="钩子名称" prop="name">
              <el-input v-model="form.name" placeholder="如：重启docker" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="模块类型" prop="module">
               <el-select v-model="form.module" placeholder="选择模块" style="width: 100%">
                 <el-option label="Shell" value="shell" />
                 <el-option label="Script" value="script" />
                 <el-option label="Systemd" value="systemd" />
                 <el-option label="Package" value="package" />
                 <el-option label="File" value="file" />
                 <el-option label="Template" value="template" />
                 <el-option label="Repo" value="repo" />
                 <el-option label="Blockinfile" value="blockinfile" />
                 <el-option label="Lineinfile" value="lineinfile" />
                 <el-option label="File Pull" value="file_pull" />
                 <el-option label="Cfssl" value="cfssl" />
                 <el-option label="Image" value="image" />
                 <el-option label="Unarchive" value="unarchive" />
                 <el-option label="Copy" value="copy" />
                 <el-option label="Modprobe" value="modprobe" />
               </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="执行参数">
          <div class="params-kv-list">
            <div v-for="(key, pi) in Object.keys(formParams)" :key="pi" class="params-kv-row">
              <span class="params-kv-key">{{ key }}</span>
              <el-input v-model="formParams[key]" :placeholder="getParamPlaceholder(form.module, key)" class="params-kv-value" />
            </div>
            <div v-if="Object.keys(formParams).length === 0" class="params-empty">请先选择模块类型</div>
          </div>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="超时(秒)">
              <el-input-number v-model="form.timeout" :min="0" :max="3600" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="重试次数">
              <el-input-number v-model="form.retries" :min="0" :max="10" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="重试间隔">
              <el-input-number v-model="form.delay" :min="0" :max="60" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="忽略错误">
          <el-checkbox v-model="form.ignore_errors">即使钩子失败也继续</el-checkbox>
        </el-form-item>
      </el-form>
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
import { ref, computed, onMounted, watch } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getHookTemplatesApi,
  createHookTemplateApi,
  updateHookTemplateApi,
  deleteHookTemplateApi,
  type HookTemplate,
} from '@/api/hookTemplate'
import { HandledError } from '@/utils/request'

const MODULE_PARAMS: Record<string, Record<string, string>> = {
  shell: { command: '执行的命令' },
  script: { script: '脚本内容', script_file: '脚本文件路径（可选）' },
  systemd: { name: '服务名称', action: 'start/stop/restart/enable/disable' },
  package: { name: '包名', state: 'present/absent/latest' },
  file: { src: '源文件路径', dest: '目标路径' },
  file_pull: { url: '文件URL [必填]', md5: '文件MD5 [必填]', dest: '目标路径', type: '类型[file/dir]' },
  template: { src: '模板路径', dest: '目标路径' },
  repo: { name: '仓库名', state: 'present/absent' },
  blockinfile: { path: '文件路径', block: '插入的内容', marker: '标记注释', insertafter: '插入位置' },
  lineinfile: { path: '文件路径 [必填]', regexp: '匹配正则 [必填]', line: '目标行 [必填]', action: 'insert/replace/delete [必填]', backrefs: '反向引用', insertbefore: '插入位置' },
  modprobe: { name: '模块名', state: 'present/absent' },
  cfssl: { action: '操作类型 [必填]: generate_ca/generate_cert' },
  image: { action: '操作类型 [必填]: load/push/remove/pull', image: '镜像名称 [必填]' },
  unarchive: { src: '源文件路径 [必填]', dest: '目标路径 [必填]', strip_components: '去除路径层级' },
  copy: { src: 'Server端源文件路径（绝对路径）[必填]', dest: 'Agent端目标路径（绝对路径）[必填]', type: '类型[file/dir]', recursive: '递归', mode: '文件权限' },
}

function getParamPlaceholder(module: string, key: string): string {
  return MODULE_PARAMS[module]?.[key] || ''
}

const loading = ref(false)
const searchText = ref('')
const templates = ref<HookTemplate[]>([])

const dialogVisible = ref(false)
const editingId = ref(0)
const submitting = ref(false)
const formRef = ref()

const form = ref({
  name: '',
  description: '',
  module: 'shell',
  timeout: 0,
  ignore_errors: false,
  retries: 0,
  delay: 0,
})

const formParams = ref<Record<string, string>>({})

const formRules = {
  name: [
    { required: true, message: '请输入钩子名称', trigger: 'blur' },
    {
      validator: (_rule: any, value: string, callback: any) => {
        if (!value) { callback(); return }
        const existing = templates.value.find(t => t.name === value && t.id !== editingId.value)
        if (existing) {
          callback(new Error(`钩子名称「${value}」已存在`))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  module: [{ required: true, message: '请选择模块类型', trigger: 'change' }],
}

watch(() => form.value.module, (mod) => {
  const keys = Object.keys(MODULE_PARAMS[mod] || {})
  const newParams: Record<string, string> = {}
  for (const k of keys) newParams[k] = ''
  formParams.value = newParams
})

const filteredTemplates = computed(() => {
  if (!searchText.value) return templates.value
  const kw = searchText.value.toLowerCase()
  return templates.value.filter(
    (t) => t.name.toLowerCase().includes(kw) || (t.description || '').toLowerCase().includes(kw)
  )
})

async function loadData() {
  loading.value = true
  try {
    templates.value = await getHookTemplatesApi()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  editingId.value = 0
  form.value = { name: '', description: '', module: 'shell', timeout: 0, ignore_errors: false, retries: 0, delay: 0 }
  formParams.value = { command: '' }
  dialogVisible.value = true
}

function editTemplate(t: HookTemplate) {
  editingId.value = t.id
  form.value = {
    name: t.name,
    description: t.description,
    module: t.module,
    timeout: t.timeout,
    ignore_errors: t.ignore_errors,
    retries: t.retries,
    delay: t.delay,
  }
  try {
    const parsed = JSON.parse(t.params || '{}')
    formParams.value = {}
    for (const [k, v] of Object.entries(parsed)) {
      formParams.value[k] = String(v ?? '')
    }
  } catch {
    formParams.value = { command: t.params || '' }
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
    const data = {
      ...form.value,
      params: JSON.stringify(formParams.value),
    }
    if (editingId.value) {
      await updateHookTemplateApi(editingId.value, data)
      ElMessage.success('保存成功')
    } else {
      await createHookTemplateApi(data)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadData()
  } catch (e: any) {
    if (!(e instanceof HandledError)) ElMessage.error(e?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function deleteTemplate(t: HookTemplate) {
  try {
    await ElMessageBox.confirm(
      `确定要删除钩子「${t.name}」吗？`,
      '删除确认',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteHookTemplateApi(t.id)
    ElMessage.success('删除成功')
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
