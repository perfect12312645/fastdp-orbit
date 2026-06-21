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
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input v-model="searchText" placeholder="搜索工作流..." clearable style="width: 240px;">
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
        </div>
        <div class="table-toolbar-right">
          <span class="total-text">共 {{ filteredData.length }} 个工作流</span>
        </div>
      </div>

      <el-table :data="paginatedData" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="180">
          <template #default="{ row }">
            <span class="link-text" @click="openCanvas(row)">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="阶段数" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.stage_groups?.length ? 'primary' : 'info'" effect="plain">
              {{ row.stage_groups?.length || 0 }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="openCanvas(row)">
              <Icon icon="mdi:edit-outline" :size="14" /> 编排
            </el-button>
            <el-button type="success" link size="small" @click="executeWorkflow(row)">
              <Icon icon="mdi:play" :size="14" /> 执行
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

    <!-- 创建对话框 -->
    <el-dialog v-model="showCreate" title="创建工作流" width="480px" destroy-on-close>
      <el-form :model="createForm" label-width="80px" ref="createFormRef" :rules="createRules">
        <el-form-item label="名称" prop="name">
          <el-input v-model="createForm.name" placeholder="如：Docker 批量部署" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="creating">
          创建并编排
        </el-button>
      </template>
    </el-dialog>

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
  createWorkflowApi,
  deleteWorkflowApi,
  executeWorkflowApi,
  getExecutionsApi,
} from '@/api/workflow'
import type { Workflow, WorkflowExecution } from '@/types/workflow'

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

const showCreate = ref(false)
const creating = ref(false)
const createFormRef = ref()
const createForm = ref({ name: '', description: '' })
const createRules = {
  name: [{ required: true, message: '请输入工作流名称', trigger: 'blur' }],
}

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
  createForm.value = { name: '', description: '' }
  showCreate.value = true
}

async function handleCreate() {
  try {
    await createFormRef.value?.validate()
  } catch {
    return
  }

  creating.value = true
  try {
    const wf = await createWorkflowApi({
      name: createForm.value.name,
      description: createForm.value.description,
      config: '',
      stage_groups: [],
      variables: [],
      hooks: [],
    })
    showCreate.value = false
    ElMessage.success('创建成功')
    router.push(`/workflow/${wf.id}/canvas`)
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  } finally {
    creating.value = false
  }
}

function openCanvas(row: Workflow) {
  router.push(`/workflow/${row.id}/canvas`)
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

.link-text {
  color: var(--el-color-primary);
  cursor: pointer;
  font-weight: 500;
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
