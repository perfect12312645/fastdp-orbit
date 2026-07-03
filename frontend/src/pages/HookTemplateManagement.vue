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
          <el-select v-model="selectedGroup" placeholder="全部来源" clearable style="width: 160px;">
            <el-option v-for="g in availableGroups" :key="g" :label="g || '(默认)'" :value="g" />
          </el-select>
        </div>
        <div class="table-toolbar-right">
          <span class="total-text">共 {{ filteredTemplates.length }} 个钩子</span>
        </div>
      </div>

      <el-table :data="paginatedTemplates" v-loading="loading" stripe>
        <el-table-column label="名称" prop="name" min-width="150" />
        <el-table-column label="模块" prop="module" width="120">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ row.module }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="来源" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.source" size="small" type="info" effect="plain">{{ row.source }}</el-tag>
            <span v-else class="text-muted">-</span>
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

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      width="640px"
      destroy-on-close
    >
      <template #header>
        <div class="hook-template-editor-header">
          <span>{{ editingId ? '编辑钩子' : '创建钩子' }}</span>
          <div class="hook-template-editor-header-actions">
            <el-button @click="dialogVisible = false" size="small">取消</el-button>
            <el-button type="primary" @click="handleSubmit" :loading="submitting" size="small">
              {{ editingId ? '保存' : '创建' }}
            </el-button>
          </div>
        </div>
      </template>
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
                 <el-option label="Shell" value="shell">
                   <div class="module-option"><span class="module-option-name">Shell</span><span class="module-option-desc">执行Shell命令</span></div>
                 </el-option>
                 <el-option label="Script" value="script">
                   <div class="module-option"><span class="module-option-name">Script</span><span class="module-option-desc">执行脚本内容</span></div>
                 </el-option>
                 <el-option label="Systemd" value="systemd">
                   <div class="module-option"><span class="module-option-name">Systemd</span><span class="module-option-desc">管理服务（启动/停止/重启）</span></div>
                 </el-option>
                 <el-option label="Package" value="package">
                   <div class="module-option"><span class="module-option-name">Package</span><span class="module-option-desc">安装/卸载软件包</span></div>
                 </el-option>
                 <el-option label="File" value="file">
                   <div class="module-option"><span class="module-option-name">File</span><span class="module-option-desc">文件/目录操作</span></div>
                 </el-option>
                 <el-option label="Template" value="template">
                   <div class="module-option"><span class="module-option-name">Template</span><span class="module-option-desc">渲染模板并写入文件</span></div>
                 </el-option>
                 <el-option label="Copy" value="copy">
                   <div class="module-option"><span class="module-option-name">Copy</span><span class="module-option-desc">从Server分发文件到Agent</span></div>
                 </el-option>
                 <el-option label="File Pull" value="file_pull">
                   <div class="module-option"><span class="module-option-name">File Pull</span><span class="module-option-desc">从URL拉取文件到Agent</span></div>
                 </el-option>
                 <el-option label="Unarchive" value="unarchive">
                   <div class="module-option"><span class="module-option-name">Unarchive</span><span class="module-option-desc">解压文件</span></div>
                 </el-option>
                 <el-option label="Repo" value="repo">
                   <div class="module-option"><span class="module-option-name">Repo</span><span class="module-option-desc">管理YUM/APT仓库</span></div>
                 </el-option>
                 <el-option label="Blockinfile" value="blockinfile">
                   <div class="module-option"><span class="module-option-name">Blockinfile</span><span class="module-option-desc">在文件中插入/更新文本块</span></div>
                 </el-option>
                 <el-option label="Lineinfile" value="lineinfile">
                   <div class="module-option"><span class="module-option-name">Lineinfile</span><span class="module-option-desc">在文件中插入/替换/删除行</span></div>
                 </el-option>
                 <el-option label="Cfssl" value="cfssl">
                   <div class="module-option"><span class="module-option-name">Cfssl</span><span class="module-option-desc">生成TLS证书</span></div>
                 </el-option>
                 <el-option label="Image" value="image">
                   <div class="module-option"><span class="module-option-name">Image</span><span class="module-option-desc">管理容器镜像</span></div>
                 </el-option>
                 <el-option label="Modprobe" value="modprobe">
                   <div class="module-option"><span class="module-option-name">Modprobe</span><span class="module-option-desc">加载/卸载内核模块</span></div>
                 </el-option>
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
              <el-input
                v-if="!isMultilineParam(form.module, key)"
                v-model="formParams[key]"
                :placeholder="getParamPlaceholder(form.module, key)"
                class="params-kv-value"
              />
              <el-input
                v-else
                v-model="formParams[key]"
                type="textarea"
                :rows="4"
                :placeholder="getParamPlaceholder(form.module, key)"
                class="params-kv-value"
              />
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
  shell: { command: '执行的命令 [必填]' },
  script: { script: '脚本内容（与script_file二选一）', script_file: '脚本文件路径（与script二选一）' },
  systemd: { name: '服务名称 [必填，reload操作可不填]', action: '操作类型 [必填]: start/stop/restart/reload/status/enable/disable' },
  package: { action: '操作类型 [必填]: install/remove/update/check/localinstall', name: '包名（多包逗号分隔，localinstall时填文件路径）[必填]' },
  file: { path: '目标路径（绝对路径）[必填]', action: '操作类型 [必填]: create/delete/touch/symlink', type: '文件类型（create/symlink时必填）: file/directory', src: '符号链接源路径（symlink时必填）', mode: '权限模式（可选），如 0644', owner: '所有者UID（可选）', group: '所属组GID（可选）', recurse: '递归创建目录（可选）: true/false', force: '强制删除非空目录（可选）: true/false', backup: '操作前备份（可选）: true/false' },
  file_pull: { url: '文件URL（支持http/https）[必填]', dest: '目标路径（绝对路径，以/结尾则自动提取文件名）[必填]', md5: '文件MD5（可选，用于校验）' },
  template: { src: '选择模板文件（与content二选一，引擎层渲染为content）', content: 'Go template模板内容（与src二选一，直接填写时使用）', dest: '目标路径（绝对路径）[必填]', append: '追加模式（可选）: true/false，默认false覆盖' },
  repo: { action: '操作类型 [必填]: add/remove/test/backup/restore/makecache', name: '仓库名称（add/remove时必填）', url: '仓库URL（add/test时必填）' },
  blockinfile: { action: '操作类型 [必填]: ensure/delete', path: '目标文件路径 [必填]', content: '文本块内容（ensure时必填，支持换行）', backup: '操作前备份（可选）: true/false' },
  lineinfile: { path: '目标文件路径（绝对路径）[必填]', regexp: '匹配行的正则表达式 [必填]', line: '目标行内容（insert/replace时必填）', action: '操作类型 [必填]: insert/replace/delete', backrefs: '启用正则反向引用（仅replace时有效）: true/false', insertbefore: '插入到匹配行前（可选）: true/false，默认false插入到匹配行后' },
  modprobe: { module: '内核模块名（与loop二选一）', loop: '模块列表（逗号分隔，与module二选一）', action: '操作类型（可选）: load/remove，默认load', options: '模块加载选项（可选）' },
  cfssl: { action: '操作类型 [必填]: generate_ca/generate_cert', csr_path: 'CSR配置文件路径 [必填]', output_dir: '证书输出目录 [必填]', basename: '输出文件名前缀 [必填]', ca_cert: 'CA证书路径（generate_cert时必填）', ca_key: 'CA私钥路径（generate_cert时必填）', config_file: 'cfssl配置文件（generate_cert时必填）', profile: '配置profile名称（generate_cert时必填）' },
  image: { action: '操作类型 [必填]: load/push/remove/pull', tag: '镜像标签（如nginx:latest）[必填]', path: '镜像文件路径（load时必填，绝对路径）' },
  unarchive: { src: '压缩文件路径（绝对路径）[必填]', dest: '目标目录（绝对路径）[必填]', strip_components: '去除路径层级数（可选），如1' },
  copy: { src: 'Server端源文件路径（绝对路径）[必填]', dest: 'Agent端目标路径（绝对路径）[必填]', type: '类型（可选）: file/dir', recursive: '递归复制（可选）: true/false', mode: '文件权限（可选），如 0644' },
}

// 需要多行输入的参数（textarea）
const MULTILINE_PARAMS: Record<string, string[]> = {
  blockinfile: ['content'],
  script: ['script'],
  template: ['content'],
}

function isMultilineParam(module: string, key: string): boolean {
  return MULTILINE_PARAMS[module]?.includes(key) ?? false
}

function getParamPlaceholder(module: string, key: string): string {
  return MODULE_PARAMS[module]?.[key] || ''
}

const loading = ref(false)
const searchText = ref('')
const selectedGroup = ref('')
const templates = ref<HookTemplate[]>([])
const currentPage = ref(1)
const pageSize = ref(20)

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

watch([searchText, selectedGroup], () => {
  currentPage.value = 1
})

const filteredTemplates = computed(() => {
  let result = templates.value
  if (selectedGroup.value) {
    result = result.filter(t => t.source === selectedGroup.value)
  }
  if (searchText.value) {
    const kw = searchText.value.toLowerCase()
    result = result.filter(
      (t) => t.name.toLowerCase().includes(kw) || (t.description || '').toLowerCase().includes(kw)
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

/* 模块选项样式 */
.module-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 2px 0;
}
.module-option-name {
  font-weight: 500;
  min-width: 80px;
}
.module-option-desc {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.hook-template-editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.hook-template-editor-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
