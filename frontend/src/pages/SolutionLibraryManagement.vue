<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>模板市场</h2>
        <p class="page-subtitle">预制模板包，一键导入阶段、变量、钩子、模板文件的完整组合</p>
      </div>
      <div class="header-actions">
        <el-button @click="showImportDialog">
          <Icon icon="mdi:import" :size="16" /> 导入模板包
        </el-button>
        <el-button type="primary" @click="showCreateDialog">
          <Icon icon="mdi:plus" :size="16" /> 创建模板包
        </el-button>
      </div>
    </div>

    <div class="page-content">
      <el-tabs v-model="activeTab" class="package-tabs">
        <el-tab-pane label="预制方案" name="preset">
          <div class="tab-description">
            <Icon icon="mdi:lightning-bolt" :size="14" /> 官方提供的开箱即用方案
          </div>
        </el-tab-pane>
        <el-tab-pane label="自定义方案" name="custom">
          <div class="tab-description">
            <Icon icon="mdi:pencil" :size="14" /> 用户自建或导入的方案
          </div>
        </el-tab-pane>
      </el-tabs>

      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input v-model="searchText" placeholder="搜索方案名称" clearable style="width: 240px;">
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
          <el-select v-model="selectedCategory" placeholder="全部分类" clearable style="width: 160px;">
            <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
          </el-select>
        </div>
        <div class="table-toolbar-right">
          <span class="total-text">共 {{ filteredPackages.length }} 个方案</span>
        </div>
      </div>

      <div class="package-grid" v-loading="loading">
        <div
          v-for="pkg in filteredPackages"
          :key="pkg.id"
          class="package-card"
        >
          <div class="package-card-header">
            <div class="package-icon">
              <Icon :icon="getCategoryIcon(pkg.category)" :size="28" />
            </div>
            <div class="package-meta">
              <div class="package-name">{{ pkg.name }}</div>
              <div class="package-category">
                <el-tag size="small" type="info" effect="plain">{{ pkg.category || '其他' }}</el-tag>
                <span v-if="pkg.version" class="package-version">v{{ pkg.version }}</span>
              </div>
            </div>
          </div>
          <div class="package-desc">{{ pkg.description || '暂无描述' }}</div>
          <div class="package-stats">
            <span v-if="pkg.stage_count"><Icon icon="mdi:view-column-outline" :size="14" /> {{ pkg.stage_count }} 阶段</span>
            <span v-if="pkg.variable_count"><Icon icon="mdi:code-json" :size="14" /> {{ pkg.variable_count }} 变量</span>
            <span v-if="pkg.hook_count"><Icon icon="mdi:hook" :size="14" /> {{ pkg.hook_count }} 钩子</span>
            <span v-if="pkg.template_count"><Icon icon="mdi:file-document-outline" :size="14" /> {{ pkg.template_count }} 模板</span>
          </div>
          <div class="package-footer">
            <span class="package-author">{{ pkg.author || '未知' }}</span>
            <div class="package-actions">
              <el-button type="primary" link size="small" @click="exportPackage(pkg)">
                <Icon icon="mdi:download" :size="14" /> 导出
              </el-button>
              <el-button type="danger" link size="small" @click="deletePackage(pkg)">
                <Icon icon="mdi:delete-outline" :size="14" />
              </el-button>
            </div>
          </div>
        </div>

        <div v-if="filteredPackages.length === 0 && !loading" class="package-empty">
          <Icon icon="mdi:package-variant-closed" :size="48" />
          <p>暂无模板包</p>
          <p style="font-size: 12px; color: var(--el-text-color-secondary);">创建或导入模板包，快速复用编排好的配置组合</p>
        </div>
      </div>
    </div>

    <!-- 创建模板包对话框 -->
    <el-dialog v-model="createDialogVisible" title="创建模板包" width="500px" destroy-on-close>
      <el-form :model="createForm" label-width="80px" ref="createFormRef" :rules="createRules">
        <el-form-item label="名称" prop="name">
          <el-input v-model="createForm.name" placeholder="如：k8s-deploy-v1.28" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="分类">
              <el-input v-model="createForm.category" placeholder="如：k8s、database" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="版本">
              <el-input v-model="createForm.version" placeholder="如：1.0" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="作者">
          <el-input v-model="createForm.author" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="submitting">创建</el-button>
      </template>
    </el-dialog>

    <!-- 导入模板包对话框 -->
    <el-dialog v-model="importDialogVisible" title="导入模板包" width="600px" destroy-on-close>
      <div class="import-area">
        <el-input
          v-model="importYaml"
          type="textarea"
          :rows="15"
          placeholder="粘贴 orbit-pack YAML 内容..."
          class="import-textarea"
        />
      </div>
      <template #footer>
        <el-button @click="importDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleImport" :loading="importing">导入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as yaml from 'js-yaml'
import {
  getSolutionLibrariesApi,
  createSolutionLibraryApi,
  deleteSolutionLibraryApi,
  exportSolutionLibraryApi,
  importSolutionLibraryApi,
  type SolutionLibrary,
  type OrbitPack,
} from '@/api/solutionLibrary'

const loading = ref(false)
const searchText = ref('')
const selectedCategory = ref('')
const activeTab = ref('custom')
const packages = ref<SolutionLibrary[]>([])

const createDialogVisible = ref(false)
const importDialogVisible = ref(false)
const submitting = ref(false)
const importing = ref(false)
const createFormRef = ref()
const importYaml = ref('')

const createForm = ref({
  name: '',
  description: '',
  category: '',
  version: '',
  author: '',
})

const createRules = {
  name: [{ required: true, message: '请输入模板包名称', trigger: 'blur' }],
}

const filteredPackages = computed(() => {
  let result = packages.value
  // 按 tab 筛选：preset = author 为 'official'，custom = 其他
  if (activeTab.value === 'preset') {
    result = result.filter(p => p.author === 'official')
  } else {
    result = result.filter(p => p.author !== 'official')
  }
  if (selectedCategory.value) {
    result = result.filter(p => p.category === selectedCategory.value)
  }
  if (searchText.value) {
    const kw = searchText.value.toLowerCase()
    result = result.filter(p =>
      p.name.toLowerCase().includes(kw) || (p.description || '').toLowerCase().includes(kw)
    )
  }
  return result
})

const categories = computed(() => {
  const cats = new Set(packages.value.map(p => p.category).filter(Boolean))
  return Array.from(cats).sort()
})

function getCategoryIcon(category: string): string {
  const icons: Record<string, string> = {
    k8s: 'mdi:kubernetes',
    database: 'mdi:database',
    monitoring: 'mdi:chart-line',
    network: 'mdi:network',
    security: 'mdi:shield-lock',
  }
  return icons[category] || 'mdi:package-variant-closed'
}

async function loadData() {
  loading.value = true
  try {
    packages.value = await getSolutionLibrariesApi()
  } catch {
    packages.value = []
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  createForm.value = { name: '', description: '', category: '', version: '', author: '' }
  createDialogVisible.value = true
}

function showImportDialog() {
  importYaml.value = ''
  importDialogVisible.value = true
}

async function handleCreate() {
  try {
    await createFormRef.value?.validate()
  } catch { return }

  submitting.value = true
  try {
    await createSolutionLibraryApi(createForm.value)
    ElMessage.success('创建成功')
    createDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

async function handleImport() {
  if (!importYaml.value.trim()) {
    ElMessage.warning('请粘贴 YAML 内容')
    return
  }

  importing.value = true
  try {
    const pack = yaml.load(importYaml.value) as OrbitPack
    if (!pack || !pack.metadata?.name) {
      ElMessage.error('YAML 格式错误：缺少 metadata.name')
      return
    }
    await importSolutionLibraryApi(pack)
    ElMessage.success('导入成功')
    importDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error('导入失败: ' + (e?.message || '未知错误'))
  } finally {
    importing.value = false
  }
}

async function exportPackage(pkg: SolutionLibrary) {
  try {
    const pack = await exportSolutionLibraryApi(pkg.id)
    const yamlStr = yaml.dump(pack, { indent: 2, lineWidth: -1 })
    // 复制到剪贴板
    await navigator.clipboard.writeText(yamlStr)
    ElMessage.success('已复制 YAML 到剪贴板')
  } catch (e: any) {
    ElMessage.error('导出失败: ' + (e?.message || '未知错误'))
  }
}

async function deletePackage(pkg: SolutionLibrary) {
  try {
    await ElMessageBox.confirm(
      `确定删除方案「${pkg.name}」？\n\n该操作将同时删除该分组下的所有阶段、变量、钩子和模板文件！`,
      '删除确认',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteSolutionLibraryApi(pkg.id)
    ElMessage.success('删除成功')
    loadData()
  } catch {
    // 取消
  }
}

onMounted(loadData)
</script>

<style scoped>
.package-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.package-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 16px;
  background: var(--el-bg-color);
  transition: border-color 0.2s;
}

.package-card:hover {
  border-color: var(--el-color-primary-light-5);
}

.package-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.package-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-primary);
  flex-shrink: 0;
}

.package-meta {
  flex: 1;
  min-width: 0;
}

.package-name {
  font-weight: 600;
  font-size: 15px;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.package-category {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.package-version {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.package-desc {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 12px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.package-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.package-stats span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.package-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.package-author {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.package-actions {
  display: flex;
  gap: 4px;
}

.package-empty {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 60px;
  color: var(--el-text-color-secondary);
}

.import-area {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.import-textarea :deep(textarea) {
  font-family: monospace;
  font-size: 13px;
  line-height: 1.5;
}

.package-tabs {
  margin-bottom: 16px;
}

.tab-description {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 0;
}
</style>
