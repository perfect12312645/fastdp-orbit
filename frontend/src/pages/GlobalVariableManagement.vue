<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>全局变量</h2>
        <p class="page-subtitle">管理可复用的变量，编排工作流时直接引用</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateDialog">
          <Icon icon="mdi:plus" :size="16" /> 创建变量
        </el-button>
      </div>
    </div>

    <div class="page-content">
        <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input v-model="searchText" placeholder="搜索变量名或描述" clearable style="width: 240px;">
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
          <el-select v-model="filterGroup" placeholder="按业务分组筛选" clearable style="width: 160px;">
            <el-option v-for="g in groupOptions" :key="g" :label="g" :value="g" />
          </el-select>
          <el-select v-model="selectedSource" placeholder="按来源筛选" clearable style="width: 160px;">
            <el-option v-for="g in packageGroups" :key="g" :label="g || '(默认)'" :value="g" />
          </el-select>
        </div>
        <div class="table-toolbar-right">
          <el-select v-model="sortField" style="width: 140px" size="default" @change="handleSort">
            <el-option label="按名称排序" value="key" />
            <el-option label="按更新时间" value="updated_at" />
          </el-select>
          <el-button text @click="sortDesc = !sortDesc; loadData()">
            <Icon :icon="sortDesc ? 'mdi:sort-descending' : 'mdi:sort-ascending'" :size="18" />
          </el-button>
          <span class="total-text">共 {{ filteredVars.length }} 个变量</span>
        </div>
      </div>

      <el-table
        :data="paginatedData"
        v-loading="loading"
        stripe
        style="width: 100%"
        :default-sort="{ prop: 'key', order: 'ascending' }"
        @sort-change="handleTableSort"
      >
        <el-table-column prop="key" label="变量名" width="200" sortable="custom">
          <template #default="{ row }">
            <span class="var-key">{{ row.key }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="getTypeTag(row.type)" effect="plain">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="value" label="值" min-width="160">
          <template #default="{ row }">
            <code class="var-value">{{ row.value || '-' }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="group" label="分组" width="140">
          <template #default="{ row }">
            {{ row.group || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="160" show-overflow-tooltip />
        <el-table-column prop="updated_at" label="更新时间" width="170" sortable="custom">
          <template #default="{ row }">
            {{ formatDateTime(row.updated_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="editVariable(row)">
              <Icon icon="mdi:pencil" :size="14" /> 编辑
            </el-button>
            <el-button type="danger" link size="small" @click="deleteVariable(row)">
              <Icon icon="mdi:delete-outline" :size="14" /> 删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper" v-if="filteredVars.length > pageSize">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="filteredVars.length"
          layout="total, sizes, prev, pager, next"
          @size-change="currentPage = 1"
        />
      </div>
    </div>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="showDialog"
      :title="editingId ? '编辑变量' : '创建变量'"
      width="520px"
      destroy-on-close
    >
      <el-form :model="formData" label-width="60px" ref="formRef" :rules="formRules">
        <el-form-item label="变量名" prop="key">
          <el-input v-model="formData.key" placeholder="如：docker_version" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="formData.type" style="width: 100%">
            <el-option label="String" value="string" />
            <el-option label="Number" value="number" />
            <el-option label="Bool" value="bool" />
          </el-select>
        </el-form-item>
        <el-form-item label="值" prop="value">
          <el-input v-model="formData.value" placeholder="变量值" />
        </el-form-item>
        <el-form-item label="分组">
          <el-input v-model="formData.group" placeholder="如：网络配置、系统设置" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="formData.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
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
import {
  getGlobalVariablesApi,
  createGlobalVariableApi,
  updateGlobalVariableApi,
  deleteGlobalVariableApi,
  type GlobalVariable,
} from '@/api/globalVariable'
import { formatDateTime } from '@/utils/format'

const loading = ref(false)
const searchText = ref('')
const filterGroup = ref('')
const selectedSource = ref('')
const variables = ref<GlobalVariable[]>([])

const sortField = ref('key')
const sortDesc = ref(false)

const currentPage = ref(1)
const pageSize = ref(20)

const showDialog = ref(false)
const editingId = ref(0)
const submitting = ref(false)
const formRef = ref()
const formData = ref({ key: '', type: 'string', value: '', description: '', group: '' })
const formRules = {
  key: [{ required: true, message: '请输入变量名', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  value: [{ required: true, message: '请输入变量值', trigger: 'blur' }],
}

const filteredVars = computed(() => {
  let result = [...variables.value]
  if (filterGroup.value) {
    result = result.filter((v) => v.group === filterGroup.value)
  }
  if (selectedSource.value) {
    result = result.filter((v) => v.source === selectedSource.value)
  }
  if (searchText.value) {
    const kw = searchText.value.toLowerCase()
    result = result.filter(
      (v) => v.key.toLowerCase().includes(kw) || (v.description || '').toLowerCase().includes(kw)
    )
  }
  result.sort((a, b) => {
    const field = sortField.value as keyof GlobalVariable
    const av = String(a[field] ?? '')
    const bv = String(b[field] ?? '')
    const cmp = av.localeCompare(bv)
    return sortDesc.value ? -cmp : cmp
  })
  return result
})

const packageGroups = computed(() => {
  const groups = new Set(variables.value.map(v => v.source).filter(Boolean))
  return Array.from(groups).sort()
})

const paginatedData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredVars.value.slice(start, start + pageSize.value)
})

const groupOptions = computed(() => {
  const groups = new Set<string>()
  variables.value.forEach((v) => {
    if (v.group) groups.add(v.group)
  })
  return Array.from(groups).sort()
})

function getTypeTag(type: string) {
  const map: Record<string, string> = {
    string: '',
    number: 'warning',
    bool: 'success',
  }
  return (map[type] || 'info') as any
}

function handleTableSort({ prop, order }: { prop: string; order: string }) {
  if (prop) sortField.value = prop
  sortDesc.value = order === 'descending'
  currentPage.value = 1
}

function handleSort() {
  currentPage.value = 1
}

async function loadData() {
  loading.value = true
  try {
    variables.value = await getGlobalVariablesApi()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  editingId.value = 0
  formData.value = { key: '', type: 'string', value: '', description: '', group: '' }
  showDialog.value = true
}

function editVariable(v: GlobalVariable) {
  editingId.value = v.id
  formData.value = { key: v.key, type: v.type, value: v.value, description: v.description, group: v.group }
  showDialog.value = true
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
      await updateGlobalVariableApi(editingId.value, formData.value)
      ElMessage.success('保存成功')
    } else {
      await createGlobalVariableApi(formData.value)
      ElMessage.success('创建成功')
    }
    showDialog.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function deleteVariable(v: GlobalVariable) {
  try {
    await ElMessageBox.confirm(
      `确定要删除变量「${v.key}」吗？`,
      '删除确认',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteGlobalVariableApi(v.id)
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

.table-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.total-text {
  font-size: 13px;
  color: var(--text-color-secondary);
}

.var-key {
  font-family: monospace;
  font-weight: 600;
  font-size: 13px;
}

.var-value {
  background: var(--el-fill-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  color: var(--el-color-success);
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
