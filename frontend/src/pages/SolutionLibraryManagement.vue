<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>方案库</h2>
        <p class="page-subtitle">一键导入阶段、变量、钩子、模板文件的完整组合</p>
      </div>
      <div class="header-actions">
        <el-button @click="showImportDialog">
          <Icon icon="mdi:import" :size="16" /> 导入方案
        </el-button>
        <el-button type="primary" @click="showCreateDialog">
          <Icon icon="mdi:plus" :size="16" /> 创建方案
        </el-button>
      </div>
    </div>

    <div class="page-content">
      <el-tabs v-model="activeTab" class="package-tabs">
        <el-tab-pane label="预制方案" name="preset" />
        <el-tab-pane label="自定义方案" name="custom" />
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
              <el-button type="warning" link size="small" @click="handleApply(pkg)">
                <Icon icon="mdi:play-circle-outline" :size="14" /> 应用
              </el-button>
              <el-button type="info" link size="small" @click="viewPackage(pkg)">
                <Icon icon="mdi:eye-outline" :size="14" /> 查看
              </el-button>
              <el-button type="primary" link size="small" @click="editPackage(pkg)">
                <Icon icon="mdi:pencil" :size="14" /> 编辑
              </el-button>
              <el-button type="success" link size="small" @click="exportPackage(pkg)">
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
          <p>暂无方案</p>
          <p style="font-size: 12px; color: var(--el-text-color-secondary);">创建或导入方案，快速复用编排好的配置组合</p>
        </div>
      </div>
    </div>

    <!-- 创建方案 - 全屏对话框 -->
    <el-dialog v-model="createDialogVisible" fullscreen :close-on-click-modal="false" destroy-on-close>
      <template #header>
        <div class="create-header">
          <div class="create-header-left">
            <Icon icon="mdi:package-variant-closed" :size="20" />
            <span class="create-title">{{ editingPackageId ? '编辑方案' : '创建方案' }}</span>
          </div>
          <div class="create-header-right">
            <el-button @click="createDialogVisible = false">取消</el-button>
            <el-button type="primary" @click="handleCreate" :loading="submitting">
              <Icon icon="mdi:content-save" :size="16" /> {{ editingPackageId ? '保存修改' : '创建方案' }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="create-content">
        <!-- 基本信息 -->
        <div class="create-section">
          <h3 class="section-title">基本信息</h3>
          <el-form :model="createForm" label-width="80px" ref="createFormRef" :rules="createRules">
            <el-row :gutter="16">
              <el-col :span="12">
                <el-form-item label="方案名称" prop="name">
                  <el-input v-model="createForm.name" placeholder="如：k8s-deploy-v1.28" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="分类">
                  <el-input v-model="createForm.category" placeholder="如：k8s、database" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="描述">
              <el-input v-model="createForm.description" type="textarea" :rows="2" placeholder="可选" />
            </el-form-item>
            <el-row :gutter="16">
              <el-col :span="12">
                <el-form-item label="版本">
                  <el-input v-model="createForm.version" placeholder="如：1.0" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="作者">
                  <el-input v-model="createForm.author" placeholder="可选" />
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
        </div>

        <!-- 选择内容 -->
        <div class="create-section">
          <h3 class="section-title">选择内容 <span class="section-hint">（阶段管理至少选一个）</span></h3>
          
          <div class="module-grid">
            <!-- 阶段管理 -->
            <div class="module-card">
              <div class="module-card-header">
                <Icon icon="mdi:view-column-outline" :size="20" />
                <span class="module-card-title">阶段管理</span>
                <el-tag size="small" type="danger" effect="plain">必选</el-tag>
                <el-link type="primary" :underline="false" size="small" class="select-all-btn" @click="toggleSelectAll('stages')">
                  {{ isAllSelected('stages') ? '取消全选' : '全选' }}
                </el-link>
                <el-input v-model="stageSearch" placeholder="搜索..." clearable size="small" style="width: 120px; margin-left: 8px;" />
              </div>
              <div class="module-card-body">
                <el-checkbox-group v-model="createForm.stage_ids">
                  <el-checkbox 
                    v-for="item in filteredStages" 
                    :key="item.id" 
                    :label="item.id"
                    class="module-checkbox"
                  >
                    {{ item.name }}
                  </el-checkbox>
                </el-checkbox-group>
                <div v-if="filteredStages.length === 0" class="module-empty">暂无阶段模板</div>
              </div>
            </div>

            <!-- 全局变量 -->
            <div class="module-card">
              <div class="module-card-header">
                <Icon icon="mdi:code-json" :size="20" />
                <span class="module-card-title">全局变量</span>
                <el-tag size="small" type="info" effect="plain">可选</el-tag>
                <el-link type="primary" :underline="false" size="small" class="select-all-btn" @click="toggleSelectAll('variables')">
                  {{ isAllSelected('variables') ? '取消全选' : '全选' }}
                </el-link>
                <el-input v-model="variableSearch" placeholder="搜索..." clearable size="small" style="width: 120px; margin-left: 8px;" />
              </div>
              <div class="module-card-body">
                <el-checkbox-group v-model="createForm.variable_ids">
                  <el-checkbox 
                    v-for="item in filteredVariables" 
                    :key="item.id" 
                    :label="item.id"
                    class="module-checkbox"
                  >
                    {{ item.key }} <span class="module-checkbox-desc">{{ item.description || item.value }}</span>
                  </el-checkbox>
                </el-checkbox-group>
                <div v-if="filteredVariables.length === 0" class="module-empty">暂无全局变量</div>
              </div>
            </div>

            <!-- 钩子管理 -->
            <div class="module-card">
              <div class="module-card-header">
                <Icon icon="mdi:hook" :size="20" />
                <span class="module-card-title">钩子管理</span>
                <el-tag size="small" type="info" effect="plain">可选</el-tag>
                <el-link type="primary" :underline="false" size="small" class="select-all-btn" @click="toggleSelectAll('hooks')">
                  {{ isAllSelected('hooks') ? '取消全选' : '全选' }}
                </el-link>
                <el-input v-model="hookSearch" placeholder="搜索..." clearable size="small" style="width: 120px; margin-left: 8px;" />
              </div>
              <div class="module-card-body">
                <el-checkbox-group v-model="createForm.hook_ids">
                  <el-checkbox 
                    v-for="item in filteredHooks" 
                    :key="item.id" 
                    :label="item.id"
                    class="module-checkbox"
                  >
                    {{ item.name }} <span class="module-checkbox-desc">{{ item.module }}</span>
                  </el-checkbox>
                </el-checkbox-group>
                <div v-if="filteredHooks.length === 0" class="module-empty">暂无钩子模板</div>
              </div>
            </div>

            <!-- 配置模板 -->
            <div class="module-card">
              <div class="module-card-header">
                <Icon icon="mdi:file-document-outline" :size="20" />
                <span class="module-card-title">配置模板</span>
                <el-tag size="small" type="info" effect="plain">可选</el-tag>
                <el-link type="primary" :underline="false" size="small" class="select-all-btn" @click="toggleSelectAll('templates')">
                  {{ isAllSelected('templates') ? '取消全选' : '全选' }}
                </el-link>
                <el-input v-model="templateSearch" placeholder="搜索..." clearable size="small" style="width: 120px; margin-left: 8px;" />
              </div>
              <div class="module-card-body">
                <el-checkbox-group v-model="createForm.template_ids">
                  <el-checkbox 
                    v-for="item in filteredTemplates" 
                    :key="item.id" 
                    :label="item.id"
                    class="module-checkbox"
                  >
                    {{ item.name }}
                  </el-checkbox>
                </el-checkbox-group>
                <div v-if="filteredTemplates.length === 0" class="module-empty">暂无配置模板</div>
              </div>
            </div>

            <!-- 存储管理 -->
            <div class="module-card">
              <div class="module-card-header">
                <Icon icon="mdi:database" :size="20" />
                <span class="module-card-title">存储管理</span>
                <el-tag size="small" type="info" effect="plain">可选</el-tag>
                <el-link type="primary" :underline="false" size="small" class="select-all-btn" @click="toggleSelectAll('files')">
                  {{ isAllSelected('files') ? '取消全选' : '全选' }}
                </el-link>
                <el-input v-model="fileSearch" placeholder="搜索..." clearable size="small" style="width: 120px; margin-left: 8px;" />
              </div>
              <div class="module-card-body">
                <el-checkbox-group v-model="createForm.file_ids">
                  <el-checkbox 
                    v-for="item in filteredFiles" 
                    :key="item.id" 
                    :label="item.id"
                    class="module-checkbox"
                  >
                    {{ item.name }} <span class="module-checkbox-desc">{{ formatSize(item.size) }}</span>
                  </el-checkbox>
                </el-checkbox-group>
                <div v-if="filteredFiles.length === 0" class="module-empty">暂无存储文件</div>
              </div>
            </div>

            <!-- 工作流 -->
            <div class="module-card">
              <div class="module-card-header">
                <Icon icon="mdi:play-circle-outline" :size="20" />
                <span class="module-card-title">工作流</span>
                <el-tag size="small" type="info" effect="plain">可选</el-tag>
                <el-link type="primary" :underline="false" size="small" class="select-all-btn" @click="toggleSelectAll('workflows')">
                  {{ isAllSelected('workflows') ? '取消全选' : '全选' }}
                </el-link>
                <el-input v-model="workflowSearch" placeholder="搜索..." clearable size="small" style="width: 120px; margin-left: 8px;" />
              </div>
              <div class="module-card-body">
                <el-checkbox-group v-model="createForm.workflow_ids">
                  <el-checkbox 
                    v-for="item in filteredWorkflows" 
                    :key="item.id" 
                    :label="item.id"
                    class="module-checkbox"
                  >
                    {{ item.name }}
                  </el-checkbox>
                </el-checkbox-group>
                <div v-if="filteredWorkflows.length === 0" class="module-empty">暂无工作流</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 查看方案对话框 -->
    <el-dialog v-model="viewDialogVisible" title="查看方案" width="700px" destroy-on-close>
      <div v-if="viewPackageData" class="view-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="方案名称">{{ viewPackageData.name }}</el-descriptions-item>
          <el-descriptions-item label="分类">{{ viewPackageData.category || '-' }}</el-descriptions-item>
          <el-descriptions-item label="版本">{{ viewPackageData.version || '-' }}</el-descriptions-item>
          <el-descriptions-item label="作者">{{ viewPackageData.author || '-' }}</el-descriptions-item>
          <el-descriptions-item label="描述" :span="2">{{ viewPackageData.description || '暂无描述' }}</el-descriptions-item>
        </el-descriptions>
        <h4 class="view-section-title">包含内容</h4>
        <div class="view-stats-grid">
          <div class="view-stat-item">
            <Icon icon="mdi:view-column-outline" :size="18" />
            <span>{{ viewPackageData.stage_count || 0 }} 个阶段</span>
          </div>
          <div class="view-stat-item">
            <Icon icon="mdi:code-json" :size="18" />
            <span>{{ viewPackageData.variable_count || 0 }} 个变量</span>
          </div>
          <div class="view-stat-item">
            <Icon icon="mdi:hook" :size="18" />
            <span>{{ viewPackageData.hook_count || 0 }} 个钩子</span>
          </div>
          <div class="view-stat-item">
            <Icon icon="mdi:file-document-outline" :size="18" />
            <span>{{ viewPackageData.template_count || 0 }} 个模板</span>
          </div>
          <div class="view-stat-item">
            <Icon icon="mdi:database" :size="18" />
            <span>{{ viewPackageData.file_count || 0 }} 个文件</span>
          </div>
          <div class="view-stat-item">
            <Icon icon="mdi:play-circle-outline" :size="18" />
            <span>{{ viewPackageData.workflow_count || 0 }} 个工作流</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="viewDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="editPackage(viewPackageData!); viewDialogVisible = false">
          <Icon icon="mdi:pencil" :size="14" /> 编辑
        </el-button>
      </template>
    </el-dialog>

    <!-- 导入方案对话框 -->
    <el-dialog v-model="importDialogVisible" title="导入方案" width="600px" destroy-on-close>
      <div class="import-area">
        <el-upload
          ref="importUploadRef"
          drag
          :auto-upload="false"
          :limit="1"
          :on-change="handleImportFileChange"
          accept=".yaml,.yml"
        >
          <Icon icon="mdi:cloud-upload-outline" :size="48" style="color: var(--el-color-primary)" />
          <div class="el-upload__text">拖拽 YAML 文件到此处，或 <em>点击选择</em></div>
          <template #tip>
            <div class="el-upload__tip">支持 .yaml 或 .yml 格式的 orbit-pack 文件</div>
          </template>
        </el-upload>
        <el-input
          v-model="importYaml"
          type="textarea"
          :rows="8"
          placeholder="或直接粘贴 YAML 内容..."
          class="import-textarea"
        />
      </div>
      <template #footer>
        <el-button @click="importDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleImport" :loading="importing">导入</el-button>
      </template>
    </el-dialog>

    <!-- 冲突检测对话框 -->
    <el-dialog v-model="conflictDialogVisible" title="应用方案 - 冲突检测" width="700px" destroy-on-close>
      <div v-if="conflictData" class="conflict-dialog">
        <!-- 导入摘要 -->
        <div class="conflict-summary">
          <h4>应用内容摘要</h4>
          <div class="summary-grid">
            <span v-if="conflictData.summary.stage_count"><Icon icon="mdi:view-column-outline" :size="14" /> {{ conflictData.summary.stage_count }} 阶段</span>
            <span v-if="conflictData.summary.variable_count"><Icon icon="mdi:code-json" :size="14" /> {{ conflictData.summary.variable_count }} 变量</span>
            <span v-if="conflictData.summary.hook_count"><Icon icon="mdi:hook" :size="14" /> {{ conflictData.summary.hook_count }} 钩子</span>
            <span v-if="conflictData.summary.template_count"><Icon icon="mdi:file-document-outline" :size="14" /> {{ conflictData.summary.template_count }} 模板</span>
            <span v-if="conflictData.summary.file_count"><Icon icon="mdi:database" :size="14" /> {{ conflictData.summary.file_count }} 文件</span>
            <span v-if="conflictData.summary.workflow_count"><Icon icon="mdi:play-circle-outline" :size="14" /> {{ conflictData.summary.workflow_count }} 工作流</span>
          </div>
        </div>

        <!-- 无冲突 -->
        <div v-if="!conflictData.has_conflicts" class="conflict-free">
          <Icon icon="mdi:check-circle" :size="48" style="color: var(--el-color-success)" />
          <p>未检测到名称冲突，可以直接应用</p>
        </div>

        <!-- 有冲突 -->
        <div v-else class="conflict-list">
          <div class="conflict-header">
            <h4>检测到 {{ conflictData.conflicts.length }} 个名称冲突</h4>
            <div class="conflict-actions-all">
              <el-button size="small" @click="handleAllDecisions('skip')">全部跳过</el-button>
              <el-button size="small" type="danger" @click="handleAllDecisions('overwrite')">全部覆盖</el-button>
            </div>
          </div>
          <div class="conflict-items">
            <div v-for="(item, index) in conflictData.conflicts" :key="index" class="conflict-item">
              <div class="conflict-item-info">
                <el-tag size="small" :type="getConflictTypeTag(item.type)">{{ getConflictTypeName(item.type) }}</el-tag>
                <span class="conflict-name">{{ item.name }}</span>
                <span class="conflict-source">已有来源: {{ item.existing_source || '未知' }}</span>
              </div>
              <div class="conflict-item-actions">
                <el-button 
                  size="small" 
                  :type="getDecision(item.type, item.name) === 'skip' ? 'info' : ''"
                  @click="setDecision(item.type, item.name, 'skip')"
                >跳过</el-button>
                <el-button 
                  size="small" 
                  :type="getDecision(item.type, item.name) === 'overwrite' ? 'danger' : ''"
                  @click="setDecision(item.type, item.name, 'overwrite')"
                >覆盖</el-button>
              </div>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="conflictDialogVisible = false">取消</el-button>
        <el-button 
          type="primary" 
          @click="handleApplyWithDecisions" 
          :loading="applying"
          :disabled="conflictData?.has_conflicts && !hasAllDecisions"
        >
          {{ conflictData?.has_conflicts ? '确认应用' : '开始应用' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { UploadFile } from 'element-plus'
import * as yaml from 'js-yaml'
import {
  getSolutionLibrariesApi,
  createSolutionLibraryApi,
  updateSolutionLibraryApi,
  deleteSolutionLibraryApi,
  exportSolutionLibraryApi,
  importSolutionLibraryApi,
  applySolutionLibraryApi,
  type SolutionLibrary,
  type OrbitPack,
} from '@/api/solutionLibrary'
import { getStageTemplatesApi, type StageTemplate } from '@/api/stageTemplate'
import { getGlobalVariablesApi, type GlobalVariable } from '@/api/globalVariable'
import { getHookTemplatesApi, type HookTemplate } from '@/api/hookTemplate'
import { getWorkflowTemplatesApi, type WorkflowTemplate } from '@/api/workflowTemplate'
import { getStorageFilesApi, type StorageFile } from '@/api/storage'
import { getWorkflowsApi } from '@/api/workflow'

const loading = ref(false)
const searchText = ref('')
const selectedCategory = ref('')
const activeTab = ref('custom')
const packages = ref<SolutionLibrary[]>([])

const createDialogVisible = ref(false)
const importDialogVisible = ref(false)
const submitting = ref(false)
const importing = ref(false)
const applying = ref(false)
const createFormRef = ref()
const importYaml = ref('')
const importUploadRef = ref()

// 冲突检测相关
const conflictDialogVisible = ref(false)
const conflictPack = ref<OrbitPack | null>(null)
const conflictData = ref<any>(null)
const conflictDecisions = ref<Record<string, Record<string, string>>>({})
const applySolutionId = ref<number | null>(null)

// 查看/编辑相关
const viewDialogVisible = ref(false)
const viewPackageData = ref<SolutionLibrary | null>(null)
const editingPackageId = ref<number | null>(null)

// 模块搜索关键词
const stageSearch = ref('')
const variableSearch = ref('')
const hookSearch = ref('')
const templateSearch = ref('')
const fileSearch = ref('')
const workflowSearch = ref('')

// 可选数据
const availableStages = ref<StageTemplate[]>([])
const availableVariables = ref<GlobalVariable[]>([])
const availableHooks = ref<HookTemplate[]>([])
const availableTemplates = ref<WorkflowTemplate[]>([])
const availableFiles = ref<StorageFile[]>([])
const availableWorkflows = ref<any[]>([])

// 过滤后的列表（本地搜索，不调接口）
const filteredStages = computed(() => {
  if (!stageSearch.value) return availableStages.value
  const kw = stageSearch.value.toLowerCase()
  return availableStages.value.filter(s => s.name.toLowerCase().includes(kw))
})

const filteredVariables = computed(() => {
  if (!variableSearch.value) return availableVariables.value
  const kw = variableSearch.value.toLowerCase()
  return availableVariables.value.filter(v =>
    v.key.toLowerCase().includes(kw) || (v.description || '').toLowerCase().includes(kw)
  )
})

const filteredHooks = computed(() => {
  if (!hookSearch.value) return availableHooks.value
  const kw = hookSearch.value.toLowerCase()
  return availableHooks.value.filter(h =>
    h.name.toLowerCase().includes(kw) || (h.module || '').toLowerCase().includes(kw)
  )
})

const filteredTemplates = computed(() => {
  if (!templateSearch.value) return availableTemplates.value
  const kw = templateSearch.value.toLowerCase()
  return availableTemplates.value.filter(t => t.name.toLowerCase().includes(kw))
})

const filteredFiles = computed(() => {
  if (!fileSearch.value) return availableFiles.value
  const kw = fileSearch.value.toLowerCase()
  return availableFiles.value.filter(f => f.name.toLowerCase().includes(kw))
})

const filteredWorkflows = computed(() => {
  if (!workflowSearch.value) return availableWorkflows.value
  const kw = workflowSearch.value.toLowerCase()
  return availableWorkflows.value.filter(w => w.name.toLowerCase().includes(kw))
})

// 全选功能
const moduleConfig: Record<string, { available: any; formKey: string }> = {
  stages: { available: availableStages, formKey: 'stage_ids' },
  variables: { available: availableVariables, formKey: 'variable_ids' },
  hooks: { available: availableHooks, formKey: 'hook_ids' },
  templates: { available: availableTemplates, formKey: 'template_ids' },
  files: { available: availableFiles, formKey: 'file_ids' },
  workflows: { available: availableWorkflows, formKey: 'workflow_ids' },
}

function isAllSelected(module: string): boolean {
  const config = moduleConfig[module]
  if (!config || config.available.value.length === 0) return false
  const selected = createForm.value[config.formKey as keyof typeof createForm.value] as number[]
  return config.available.value.every((item: any) => selected.includes(item.id))
}

function toggleSelectAll(module: string) {
  const config = moduleConfig[module]
  if (!config) return
  const allIds = config.available.value.map((item: any) => item.id)
  if (isAllSelected(module)) {
    // 取消全选
    ;(createForm.value as any)[config.formKey] = []
  } else {
    // 全选
    ;(createForm.value as any)[config.formKey] = [...allIds]
  }
}

const createForm = ref({
  name: '',
  description: '',
  category: '',
  version: '',
  author: '',
  stage_ids: [] as number[],
  variable_ids: [] as number[],
  hook_ids: [] as number[],
  template_ids: [] as number[],
  file_ids: [] as number[],
  workflow_ids: [] as number[],
})

const createRules = {
  name: [{ required: true, message: '请输入方案名称', trigger: 'blur' }],
}

const filteredPackages = computed(() => {
  let result = packages.value
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

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i]
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

async function loadModuleData() {
  try {
    const [stages, vars, hooks, templates, files, workflows] = await Promise.all([
      getStageTemplatesApi().catch(() => []),
      getGlobalVariablesApi().catch(() => []),
      getHookTemplatesApi().catch(() => []),
      getWorkflowTemplatesApi().catch(() => []),
      getStorageFilesApi().catch(() => []),
      getWorkflowsApi().catch(() => []),
    ])
    availableStages.value = stages
    availableVariables.value = vars
    availableHooks.value = hooks
    availableTemplates.value = templates
    availableFiles.value = files
    availableWorkflows.value = workflows
  } catch (e) {
    console.error('加载模块数据失败', e)
  }
}

function showCreateDialog() {
  createForm.value = {
    name: '',
    description: '',
    category: '',
    version: '',
    author: '',
    stage_ids: [],
    variable_ids: [],
    hook_ids: [],
    template_ids: [],
    file_ids: [],
    workflow_ids: [],
  }
  editingPackageId.value = null
  // 重置搜索
  stageSearch.value = ''
  variableSearch.value = ''
  hookSearch.value = ''
  templateSearch.value = ''
  fileSearch.value = ''
  workflowSearch.value = ''
  createDialogVisible.value = true
  loadModuleData()
}

function showImportDialog() {
  importYaml.value = ''
  importDialogVisible.value = true
}

function viewPackage(pkg: SolutionLibrary) {
  viewPackageData.value = pkg
  viewDialogVisible.value = true
}

// 解析 JSON 字符串为数组，兼容已经是数组的情况
function parseJSONIds(value: any): number[] {
  if (Array.isArray(value)) return value
  if (typeof value === 'string' && value) {
    try {
      const parsed = JSON.parse(value)
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  return []
}

async function editPackage(pkg: SolutionLibrary) {
  editingPackageId.value = pkg.id
  // 重置搜索
  stageSearch.value = ''
  variableSearch.value = ''
  hookSearch.value = ''
  templateSearch.value = ''
  fileSearch.value = ''
  workflowSearch.value = ''
  createDialogVisible.value = true
  // 先加载可选数据，再填充表单（保证勾选框能正确回显）
  await loadModuleData()
  createForm.value = {
    name: pkg.name,
    description: pkg.description || '',
    category: pkg.category || '',
    version: pkg.version || '',
    author: pkg.author || '',
    stage_ids: parseJSONIds(pkg.stage_ids),
    variable_ids: parseJSONIds(pkg.variable_ids),
    hook_ids: parseJSONIds(pkg.hook_ids),
    template_ids: parseJSONIds(pkg.template_ids),
    file_ids: parseJSONIds(pkg.file_ids),
    workflow_ids: parseJSONIds(pkg.workflow_ids),
  }
}

function handleImportFileChange(file: UploadFile) {
  if (file.raw) {
    const reader = new FileReader()
    reader.onload = (e) => {
      importYaml.value = e.target?.result as string || ''
    }
    reader.readAsText(file.raw)
  }
}

async function handleCreate() {
  try {
    await createFormRef.value?.validate()
  } catch { return }

  if (createForm.value.stage_ids.length === 0) {
    ElMessage.warning('请至少选择一个阶段')
    return
  }

  submitting.value = true
  try {
    if (editingPackageId.value) {
      await updateSolutionLibraryApi(editingPackageId.value, createForm.value)
      ElMessage.success('更新成功')
    } else {
      await createSolutionLibraryApi(createForm.value)
      ElMessage.success('创建成功')
    }
    createDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e?.message || (editingPackageId.value ? '更新失败' : '创建失败'))
  } finally {
    submitting.value = false
  }
}

async function handleImport() {
  if (!importYaml.value.trim()) {
    ElMessage.warning('请上传或粘贴 YAML 内容')
    return
  }

  importing.value = true
  try {
    const pack = yaml.load(importYaml.value) as OrbitPack
    if (!pack || !pack.metadata?.name) {
      ElMessage.error('YAML 格式错误：缺少 metadata.name')
      return
    }
    // 直接导入，只检查方案名称是否重复
    await importSolutionLibraryApi({ pack })
    ElMessage.success('导入成功')
    importDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error('导入失败: ' + (e?.message || '未知错误'))
  } finally {
    importing.value = false
  }
}

// 冲突对话框辅助函数
function getConflictTypeName(type: string): string {
  const names: Record<string, string> = {
    stages: '阶段', variables: '变量', hooks: '钩子',
    templates: '模板', files: '文件', workflows: '工作流', solution: '方案',
  }
  return names[type] || type
}

function getConflictTypeTag(type: string): string {
  const tags: Record<string, string> = {
    stages: '', variables: 'success', hooks: 'warning',
    templates: 'info', files: 'info', workflows: 'danger', solution: 'danger',
  }
  return tags[type] || ''
}

function getDecision(type: string, name: string): string {
  return conflictDecisions.value[type]?.[name] || ''
}

function setDecision(type: string, name: string, decision: string) {
  if (!conflictDecisions.value[type]) {
    conflictDecisions.value[type] = {}
  }
  conflictDecisions.value[type][name] = decision
}

function handleAllDecisions(decision: string) {
  if (!conflictData.value?.conflicts) return
  for (const item of conflictData.value.conflicts) {
    setDecision(item.type, item.name, decision)
  }
}

const hasAllDecisions = computed(() => {
  if (!conflictData.value?.conflicts) return true
  return conflictData.value.conflicts.every(
    (item: any) => conflictDecisions.value[item.type]?.[item.name]
  )
})

async function handleApply(pkg: SolutionLibrary) {
  applying.value = true
  try {
    const result = await applySolutionLibraryApi(pkg.id) as any
    if (result?.has_conflicts) {
      // 有冲突，显示冲突对话框
      applySolutionId.value = pkg.id
      conflictData.value = result
      conflictDecisions.value = {}
      conflictDialogVisible.value = true
    } else {
      // 无冲突，后端已直接应用成功
      ElMessage.success(result?.message || '应用成功')
      loadData()
    }
  } catch (e: any) {
    ElMessage.error('应用失败: ' + (e?.message || '未知错误'))
  } finally {
    applying.value = false
  }
}

async function handleApplyWithDecisions() {
  if (!applySolutionId.value) return
  applying.value = true
  try {
    await applySolutionLibraryApi(
      applySolutionId.value,
      conflictData.value?.has_conflicts ? conflictDecisions.value : undefined,
    )
    ElMessage.success('应用成功')
    conflictDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error('应用失败: ' + (e?.message || '未知错误'))
  } finally {
    applying.value = false
  }
}

async function exportPackage(pkg: SolutionLibrary) {
  try {
    const pack = await exportSolutionLibraryApi(pkg.id)
    const yamlStr = yaml.dump(pack, { indent: 2, lineWidth: -1 })
    // 下载为文件
    const blob = new Blob([yamlStr], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${pkg.name}.yaml`
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('已导出 YAML 文件')
  } catch (e: any) {
    ElMessage.error('导出失败: ' + (e?.message || '未知错误'))
  }
}

async function deletePackage(pkg: SolutionLibrary) {
  try {
    await ElMessageBox.confirm(
      `确定删除方案「${pkg.name}」？`,
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

.package-tabs {
  margin-bottom: 16px;
}

/* 创建方案全屏样式 */
.create-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.create-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.create-title {
  font-size: 16px;
  font-weight: 600;
}

.create-content {
  max-width: 1200px;
  margin: 0 auto;
}

.create-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.section-hint {
  font-weight: normal;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.module-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}

.module-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.module-card-title {
  font-weight: 500;
  font-size: 14px;
  flex: 1;
}

.select-all-btn {
  font-size: 12px;
  margin-left: 8px;
}

.module-card-body {
  padding: 12px;
  max-height: 200px;
  overflow-y: auto;
}

.module-checkbox {
  display: flex;
  margin-bottom: 6px;
}

.module-checkbox-desc {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-left: 4px;
}

.module-empty {
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding: 20px;
}

/* 导入样式 */
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

/* 查看对话框样式 */
.view-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.view-section-title {
  font-size: 15px;
  font-weight: 600;
  margin: 8px 0 0 0;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.view-stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.view-stat-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

/* 冲突对话框样式 */
.conflict-dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.conflict-summary h4 {
  margin: 0 0 8px 0;
  font-size: 14px;
  font-weight: 600;
}

.summary-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.summary-grid span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.conflict-free {
  text-align: center;
  padding: 24px;
  color: var(--el-text-color-secondary);
}

.conflict-free p {
  margin: 8px 0 0 0;
}

.conflict-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.conflict-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-color-danger);
}

.conflict-actions-all {
  display: flex;
  gap: 8px;
}

.conflict-items {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  max-height: 300px;
  overflow-y: auto;
}

.conflict-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.conflict-item:last-child {
  border-bottom: none;
}

.conflict-item-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.conflict-name {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.conflict-source {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: auto;
}

.conflict-item-actions {
  display: flex;
  gap: 4px;
  margin-left: 12px;
}
</style>
