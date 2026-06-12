<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>节点管理</h2>
        <p class="page-subtitle">管理集群中的所有计算节点</p>
      </div>
      <el-button type="primary" @click="handleAdd">
        <Icon icon="mdi:plus" :size="16" /> 添加节点
      </el-button>
    </div>

    <div class="page-content">
      <!-- 搜索与工具栏 -->
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input
            v-model="queryForm.name"
            placeholder="搜索节点名称"
            clearable
            style="width: 200px;"
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
          <el-input
            v-model="queryForm.ip"
            placeholder="搜索IP地址"
            clearable
            style="width: 200px;"
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
          <el-select
            v-model="queryForm.status"
            placeholder="节点状态"
            clearable
            style="width: 140px;"
            @change="handleSearch"
          >
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
            <el-option label="维护中" value="maintenance" />
            <el-option label="异常" value="error" />
          </el-select>
          <el-button type="primary" @click="handleSearch">
            <Icon icon="mdi:magnify" :size="16" /> 搜索
          </el-button>
          <el-button @click="handleReset">
            <Icon icon="mdi:refresh" :size="16" /> 重置
          </el-button>
        </div>
        <div class="table-toolbar-right">
          <el-button
            type="danger"
            :disabled="selectedIds.length === 0"
            @click="handleBatchDelete"
          >
            <Icon icon="mdi:delete-outline" :size="16" /> 批量删除
          </el-button>
          <el-button @click="handleExport">
            <Icon icon="mdi:download-outline" :size="16" /> 导出
          </el-button>
        </div>
      </div>

      <!-- 数据表格 -->
      <el-table
        v-loading="loading"
        :data="tableData"
        border
        stripe
        @selection-change="handleSelectionChange"
        @sort-change="handleSortChange"
        style="width: 100%"
        row-key="id"
      >
        <el-table-column type="selection" width="50" align="center" />
        <el-table-column prop="name" label="节点名称" min-width="140" sortable="custom" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="node-name">
              <Icon icon="mdi:server-network-outline" :size="16" class="node-icon" />
              {{ row.name }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP地址" min-width="140" sortable="custom" show-overflow-tooltip>
          <template #default="{ row }">
            <code class="ip-code">{{ row.ip }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="os" label="操作系统" min-width="120" show-overflow-tooltip />
        <el-table-column prop="cpuCores" label="CPU核心数" width="110" align="center" sortable="custom" />
        <el-table-column prop="memoryMb" label="内存大小" width="110" align="center" sortable="custom">
          <template #default="{ row }">{{ formatMemory(row.memoryMb) }}</template>
        </el-table-column>
        <el-table-column prop="diskGb" label="磁盘大小" width="110" align="center" sortable="custom">
          <template #default="{ row }">{{ row.diskGb ? `${row.diskGb} GB` : '-' }}</template>
        </el-table-column>
        <el-table-column prop="gpuCount" label="GPU数量" width="100" align="center" sortable="custom">
          <template #default="{ row }">
            <el-tag type="primary" effect="plain" size="small" round>
              <Icon icon="mdi:chip-outline" :size="12" /> {{ row.gpuCount }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getNodeStatusType(row.status)" size="small" effect="light" round>
              <span class="status-dot" :class="getNodeStatusType(row.status)"></span>
              {{ getNodeStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updatedAt" label="最后更新时间" width="180" sortable="custom">
          <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right" align="center">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleEdit(row)">
              <Icon icon="mdi:pencil-outline" :size="14" /> 编辑
            </el-button>
            <el-button type="primary" link size="small" @click="handleDetail(row)">
              <Icon icon="mdi:eye-outline" :size="14" /> 详情
            </el-button>
            <el-popconfirm
              title="确定要删除该节点吗？"
              confirm-button-text="确定"
              cancel-button-text="取消"
              @confirm="handleDelete(row)"
            >
              <template #reference>
                <el-button type="danger" link size="small">
                  <Icon icon="mdi:delete-outline" :size="14" /> 删除
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="queryForm.page"
          v-model:page-size="queryForm.pageSize"
          :page-sizes="PaginationConfig.pageSizes"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSearch"
          @current-change="handleSearch"
        />
      </div>
    </div>

    <!-- 新增/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '添加节点' : '编辑节点'"
      width="600px"
      destroy-on-close
      @close="resetForm"
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="100px"
        label-position="right"
      >
        <el-form-item label="节点名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入节点名称" />
        </el-form-item>
        <el-form-item label="IP地址" prop="ip">
          <el-input v-model="formData.ip" placeholder="请输入IP地址" />
        </el-form-item>
        <el-form-item label="操作系统" prop="os">
          <el-input v-model="formData.os" placeholder="如 Ubuntu 22.04" />
        </el-form-item>
        <el-form-item label="CPU核心数" prop="cpuCores">
          <el-input-number v-model="formData.cpuCores" :min="1" :max="1024" style="width: 100%;" />
        </el-form-item>
        <el-form-item label="内存(MB)" prop="memoryMb">
          <el-input-number v-model="formData.memoryMb" :min="1" :max="1048576" style="width: 100%;" />
        </el-form-item>
        <el-form-item label="磁盘(GB)" prop="diskGb">
          <el-input-number v-model="formData.diskGb" :min="1" :max="10485760" style="width: 100%;" />
        </el-form-item>
        <el-form-item label="所属集群" prop="clusterName">
          <el-input v-model="formData.clusterName" placeholder="请输入集群名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取 消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确 定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { getNodeListApi, createNodeApi, updateNodeApi, deleteNodeApi, batchDeleteNodeApi } from '@/api/node'
import type { NodeInfo, NodeFormData } from '@/api/types'
import { PaginationConfig, NodeStatusLabel, NodeStatusType } from '@/constants'
import { formatDateTime, formatMemory } from '@/utils/format'
import { exportToExcel } from '@/utils/export'

function getNodeStatusLabel(status: string) {
  return NodeStatusLabel[status as keyof typeof NodeStatusLabel] || status
}

function getNodeStatusType(status: string) {
  return NodeStatusType[status as keyof typeof NodeStatusType] || 'info'
}

const queryForm = reactive({
  page: PaginationConfig.defaultPage,
  pageSize: PaginationConfig.defaultPageSize,
  name: '',
  ip: '',
  status: '',
})

const tableData = ref<NodeInfo[]>([])
const total = ref(0)
const loading = ref(false)
const selectedIds = ref<number[]>([])

const dialogVisible = ref(false)
const dialogType = ref<'add' | 'edit'>('add')
const submitLoading = ref(false)
const formRef = ref<FormInstance>()

const formData = reactive<NodeFormData & { id?: number }>({
  name: '',
  ip: '',
  os: '',
  cpuCores: 8,
  memoryMb: 32768,
  diskGb: 512,
  clusterName: '',
})

const formRules: FormRules = {
  name: [
    { required: true, message: '请输入节点名称', trigger: 'blur' },
    { min: 2, max: 50, message: '名称长度应在2-50个字符之间', trigger: 'blur' },
  ],
  ip: [
    { required: true, message: '请输入IP地址', trigger: 'blur' },
    { pattern: /^(\d{1,3}\.){3}\d{1,3}$/, message: 'IP地址格式不正确', trigger: 'blur' },
  ],
  os: [{ required: true, message: '请输入操作系统', trigger: 'blur' }],
  cpuCores: [{ required: true, message: '请输入CPU核心数', trigger: 'blur' }],
  memoryMb: [{ required: true, message: '请输入内存大小', trigger: 'blur' }],
  diskGb: [{ required: true, message: '请输入磁盘大小', trigger: 'blur' }],
}

async function loadData() {
  loading.value = true
  try {
    const res = await getNodeListApi(queryForm)
    tableData.value = res.data.list
    total.value = res.data.total
  } catch { /* handled */ } finally {
    loading.value = false
  }
}

function handleSearch() {
  queryForm.page = 1
  loadData()
}

function handleReset() {
  queryForm.name = ''
  queryForm.ip = ''
  queryForm.status = ''
  queryForm.page = 1
  loadData()
}

function handleSortChange({ prop, order }: { prop: string; order: string }) {
  ;(queryForm as Record<string, unknown>).sortField = prop
  ;(queryForm as Record<string, unknown>).sortOrder = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
  loadData()
}

function handleSelectionChange(selection: NodeInfo[]) {
  selectedIds.value = selection.map((item) => item.id)
}

function handleAdd() {
  dialogType.value = 'add'
  dialogVisible.value = true
}

function handleEdit(row: NodeInfo) {
  dialogType.value = 'edit'
  Object.assign(formData, {
    id: row.id, name: row.name, ip: row.ip, os: row.os,
    cpuCores: row.cpuCores, memoryMb: row.memoryMb, diskGb: row.diskGb, clusterName: row.clusterName,
  })
  dialogVisible.value = true
}

function handleDetail(row: NodeInfo) {
  ElMessage.info(`查看节点详情: ${row.name}`)
}

async function handleSubmit() {
  if (!formRef.value) return
  try { await formRef.value.validate() } catch { return }
  submitLoading.value = true
  try {
    if (dialogType.value === 'add') {
      await createNodeApi(formData)
      ElMessage.success('节点创建成功')
    } else {
      await updateNodeApi(formData as NodeFormData & { id: number })
      ElMessage.success('节点更新成功')
    }
    dialogVisible.value = false
    loadData()
  } catch { /* handled */ } finally {
    submitLoading.value = false
  }
}

async function handleDelete(row: NodeInfo) {
  try {
    await deleteNodeApi(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch { /* handled */ }
}

async function handleBatchDelete() {
  try {
    await ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 个节点吗？`, '批量删除确认', {
      confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning',
    })
    await batchDeleteNodeApi(selectedIds.value)
    ElMessage.success('批量删除成功')
    selectedIds.value = []
    loadData()
  } catch { /* cancelled or failed */ }
}

function handleExport() {
  exportToExcel({
    filename: '节点列表',
    columns: {
      name: '节点名称', ip: 'IP地址', os: '操作系统', cpuCores: 'CPU核心数',
      memoryMb: '内存(MB)', diskGb: '磁盘(GB)', gpuCount: 'GPU数量', status: '状态', updatedAt: '最后更新时间',
    },
    data: tableData.value,
  })
}

function resetForm() {
  Object.assign(formData, {
    id: undefined, name: '', ip: '', os: '', cpuCores: 8, memoryMb: 32768, diskGb: 512, clusterName: '',
  })
  formRef.value?.clearValidate()
}

onMounted(() => loadData())
</script>

<style scoped>
.page-subtitle {
  font-size: var(--font-size-sm);
  color: var(--text-color-secondary);
  margin-top: 4px;
}

.node-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: var(--font-weight-medium);
}

.node-icon {
  color: var(--el-color-primary);
}

.ip-code {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--text-color-primary);
}
</style>
