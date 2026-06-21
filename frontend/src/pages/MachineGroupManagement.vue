<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>机器分组</h2>
        <p class="page-subtitle">管理机器分组，用于工作流中按分组批量执行任务</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateDialog">
          <Icon icon="mdi:plus" :size="16" /> 创建分组
        </el-button>
        <el-button @click="loadData" :loading="loading">
          <Icon icon="mdi:refresh" :size="16" /> 刷新
        </el-button>
      </div>
    </div>

    <div class="page-content">
      <!-- 搜索工具栏 -->
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input
            v-model="searchText"
            placeholder="搜索分组名称"
            clearable
            style="width: 240px;"
          >
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
        </div>
        <div class="table-toolbar-right">
          <span class="total-text">共 {{ filteredData.length }} 个分组</span>
        </div>
      </div>

      <!-- 数据表格 -->
      <el-table
        v-loading="loading"
        :data="filteredData"
        border
        stripe
        style="width: 100%"
        row-key="id"
      >
        <el-table-column prop="name" label="分组名称" min-width="160">
          <template #default="{ row }">
            <span class="group-name">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="包含机器" width="260">
          <template #default="{ row }">
            <div class="machine-tags" v-if="row.machines?.length">
              <el-tag
                v-for="m in row.machines.slice(0, 3)"
                :key="m.id"
                size="small"
                type="info"
                effect="plain"
              >
                {{ m.hostname || m.ip }}
              </el-tag>
              <el-tag v-if="row.machines.length > 3" size="small" type="info" effect="plain">
                +{{ row.machines.length - 3 }}
              </el-tag>
            </div>
            <span v-else class="text-muted">无机器</span>
          </template>
        </el-table-column>
        <el-table-column label="机器数" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.machines?.length ? 'primary' : 'info'" effect="plain">
              {{ row.machines?.length || 0 }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="showEditDialog(row)">
              <Icon icon="mdi:pencil" :size="14" /> 编辑
            </el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">
              <Icon icon="mdi:delete-outline" :size="14" /> 删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEditing ? '编辑分组' : '创建分组'"
      width="600px"
      destroy-on-close
    >
      <el-form :model="formData" label-width="80px" ref="formRef" :rules="formRules">
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="如：k8s_master, web_servers" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="formData.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="选择机器">
          <el-select
            v-model="formData.machine_ids"
            multiple
            filterable
            placeholder="选择要加入分组的机器"
            style="width: 100%"
            :loading="machineLoading"
          >
            <el-option
              v-for="m in allMachines"
              :key="m.id"
              :label="`${m.hostname || m.ip} (${m.ip}:${m.port})`"
              :value="m.id"
            >
              <div class="machine-option">
                <span class="status-dot" :class="m.status === 'online' ? 'status-online' : 'status-offline'"></span>
                <span>{{ m.hostname || m.ip }}</span>
                <span class="machine-ip">{{ m.ip }}:{{ m.port }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          {{ isEditing ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getMachineGroupsApi,
  createMachineGroupApi,
  updateMachineGroupApi,
  deleteMachineGroupApi,
  type MachineGroup,
} from '@/api/machineGroup'
import { getMachinesApi, type MachineInfo } from '@/api/machine'

const loading = ref(false)
const searchText = ref('')
const groups = ref<MachineGroup[]>([])

const allMachines = ref<MachineInfo[]>([])
const machineLoading = ref(false)

// Dialog
const dialogVisible = ref(false)
const isEditing = ref(false)
const editingId = ref(0)
const submitting = ref(false)
const formRef = ref()

const formData = ref({
  name: '',
  description: '',
  machine_ids: [] as number[],
})

const formRules = {
  name: [{ required: true, message: '请输入分组名称', trigger: 'blur' }],
}

const filteredData = computed(() => {
  if (!searchText.value) return groups.value
  const kw = searchText.value.toLowerCase()
  return groups.value.filter(
    (g) => g.name.toLowerCase().includes(kw) || (g.description || '').toLowerCase().includes(kw)
  )
})

async function loadData() {
  loading.value = true
  try {
    groups.value = await getMachineGroupsApi()
  } catch {
    ElMessage.error('获取分组列表失败')
  } finally {
    loading.value = false
  }
}

async function loadMachines() {
  machineLoading.value = true
  try {
    const machines = await getMachinesApi()
    allMachines.value = machines.sort((a, b) => {
      const ipA = a.ip.split('.').map(Number)
      const ipB = b.ip.split('.').map(Number)
      for (let i = 0; i < 4; i++) {
        if (ipA[i] !== ipB[i]) {
          return ipA[i] - ipB[i]
        }
      }
      return 0
    })
  } catch {
    console.error('获取机器列表失败')
  } finally {
    machineLoading.value = false
  }
}

function showCreateDialog() {
  isEditing.value = false
  editingId.value = 0
  formData.value = { name: '', description: '', machine_ids: [] }
  dialogVisible.value = true
  loadMachines()
}

function showEditDialog(row: MachineGroup) {
  isEditing.value = true
  editingId.value = row.id
  formData.value = {
    name: row.name,
    description: row.description || '',
    machine_ids: (row.machines || []).map((m) => m.id),
  }
  dialogVisible.value = true
  loadMachines()
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    const payload = {
      name: formData.value.name,
      description: formData.value.description,
      machine_ids: formData.value.machine_ids,
    }
    if (isEditing.value) {
      await updateMachineGroupApi(editingId.value, payload)
      ElMessage.success('更新成功')
    } else {
      await createMachineGroupApi(payload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: MachineGroup) {
  try {
    await ElMessageBox.confirm(
      `确定要删除分组「${row.name}」吗？\n分组内的机器不会被删除。`,
      '删除确认',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteMachineGroupApi(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch {
    // 用户取消
  }
}

onMounted(() => loadData())
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

.group-name {
  font-weight: var(--font-weight-medium);
}

.machine-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.text-muted {
  color: var(--text-color-secondary);
}

.machine-option {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.machine-option .machine-ip {
  margin-left: auto;
  color: var(--text-color-secondary);
  font-size: 12px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
  flex-shrink: 0;
}

.status-online {
  background-color: var(--el-color-success);
  box-shadow: 0 0 6px var(--el-color-success-light-5);
}

.status-offline {
  background-color: var(--el-color-danger);
  box-shadow: 0 0 6px var(--el-color-danger-light-5);
}
</style>
