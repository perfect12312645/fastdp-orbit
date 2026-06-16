<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>工作流管理</h2>
        <p class="page-subtitle">编排和管理自动化运维流程</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateDialog">
          <Icon icon="mdi:plus" :size="16" /> 创建工作流
        </el-button>
      </div>
    </div>

    <div class="page-content">
      <div class="toolbar">
        <el-input v-model="searchText" placeholder="搜索工作流..." clearable class="search-input">
          <template #prefix>
            <Icon icon="mdi:magnify" :size="16" />
          </template>
        </el-input>
      </div>

      <el-table :data="paginatedData" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="名称" min-width="180">
          <template #default="{ row }">
            <span class="link-text" @click="viewWorkflow(row)">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="阶段数" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.stages?.length || 0 }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="executeWorkflow(row)">
              <Icon icon="mdi:play" :size="14" /> 执行
            </el-button>
            <el-button type="primary" link size="small" @click="editWorkflow(row)">
              <Icon icon="mdi:pencil" :size="14" /> 编辑
            </el-button>
            <el-button type="primary" link size="small" @click="viewExecutions(row)">
              <Icon icon="mdi:history" :size="14" /> 历史
            </el-button>
            <el-button type="danger" link size="small" @click="deleteWorkflow(row)">
              <Icon icon="mdi:delete" :size="14" /> 删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper" v-if="filteredData.length > pageSize">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="filteredData.length"
          layout="prev, pager, next"
          background
        />
      </div>
    </div>

    <!-- 创建/编辑对话框 -->
    <WorkflowEditor
      v-model="showEditor"
      :workflow="editingWorkflow"
      @saved="onWorkflowSaved"
    />

    <!-- 执行历史对话框 -->
    <el-dialog v-model="showExecutions" title="执行历史" width="700px" destroy-on-close>
      <el-table :data="executions" v-loading="loadingExecutions" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">{{ getStatusLabel(row.status) }}</el-tag>
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
            <el-button type="primary" link size="small" @click="viewExecutionDetail(row)">
              <Icon icon="mdi:eye" :size="14" /> 详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDateTime } from '@/utils/format'
import {
  getWorkflowsApi,
  deleteWorkflowApi,
  executeWorkflowApi,
  getExecutionsApi,
} from '@/api/workflow'
import type { Workflow, WorkflowExecution } from '@/types/workflow'
import WorkflowEditor from '@/components/workflow/WorkflowEditor.vue'

const router = useRouter()

const loading = ref(false)
const searchText = ref('')
const currentPage = ref(1)
const pageSize = 20
const workflows = ref<Workflow[]>([])

const filteredData = computed(() => {
  if (!searchText.value) return workflows.value
  const kw = searchText.value.toLowerCase()
  return workflows.value.filter(
    (w) => w.name.toLowerCase().includes(kw) || w.description.toLowerCase().includes(kw)
  )
})

const paginatedData = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  return filteredData.value.slice(start, start + pageSize)
})

// 编辑器
const showEditor = ref(false)
const editingWorkflow = ref<Workflow | null>(null)

// 执行历史
const showExecutions = ref(false)
const loadingExecutions = ref(false)
const executions = ref<WorkflowExecution[]>([])
const currentWorkflowId = ref(0)

async function loadData() {
  loading.value = true
  try {
    workflows.value = await getWorkflowsApi()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  editingWorkflow.value = null
  showEditor.value = true
}

function editWorkflow(row: Workflow) {
  editingWorkflow.value = row
  showEditor.value = true
}

function viewWorkflow(row: Workflow) {
  editingWorkflow.value = row
  showEditor.value = true
}

async function executeWorkflow(row: Workflow) {
  try {
    await ElMessageBox.confirm(`确认执行工作流「${row.name}」？`, '执行确认', {
      confirmButtonText: '执行',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await executeWorkflowApi(row.id)
    ElMessage.success('工作流已触发执行')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('执行失败')
  }
}

async function deleteWorkflow(row: Workflow) {
  try {
    await ElMessageBox.confirm(`确认删除工作流「${row.name}」？此操作不可恢复。`, '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'error',
    })
    await deleteWorkflowApi(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

async function viewExecutions(row: Workflow) {
  currentWorkflowId.value = row.id
  showExecutions.value = true
  loadingExecutions.value = true
  try {
    executions.value = await getExecutionsApi(row.id)
  } catch (e) {
    console.error(e)
  } finally {
    loadingExecutions.value = false
  }
}

function viewExecutionDetail(row: WorkflowExecution) {
  router.push(`/workflow/${currentWorkflowId.value}/executions/${row.id}`)
}

function onWorkflowSaved() {
  loadData()
}

function getStatusType(status: string) {
  const map: Record<string, string> = {
    running: 'warning',
    success: 'success',
    failed: 'danger',
    paused: 'info',
    cancelled: 'info',
  }
  return (map[status] || 'info') as any
}

function getStatusLabel(status: string) {
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

onMounted(loadData)
</script>

<style scoped>
.page-container {
  padding: 24px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
}

.page-subtitle {
  margin: 4px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.page-content {
  flex: 1;
  background: var(--el-bg-color);
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.toolbar {
  margin-bottom: 16px;
}

.search-input {
  width: 280px;
}

.link-text {
  color: var(--el-color-primary);
  cursor: pointer;
}

.link-text:hover {
  text-decoration: underline;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
