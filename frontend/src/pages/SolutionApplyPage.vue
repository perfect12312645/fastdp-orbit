<template>
  <el-dialog v-model="visible" fullscreen :close-on-click-modal="false" destroy-on-close @close="handleClose" class="apply-dialog">
    <template #header>
      <div class="apply-header">
        <Icon icon="mdi:play-circle-outline" :size="20" />
        <span class="apply-title">应用方案 - {{ solution?.name || '...' }}</span>
      </div>
    </template>

    <div class="apply-body" v-loading="loading">
      <el-steps :active="currentStep" finish-status="success" class="apply-steps">
        <el-step title="冲突检测" :status="stepStatus(0)" />
        <el-step title="机器分组" :status="stepStatus(1)" />
        <el-step title="全局变量" :status="stepStatus(2)" />
        <el-step title="依赖文件" :status="stepStatus(3)" />
      </el-steps>

      <div class="apply-scroll">
        <div class="apply-content">
        <!-- Tab 1: Conflict Detection -->
      <div v-show="currentStep === 0" class="tab-content">
        <div class="tab-header">
          <h3>冲突检测</h3>
          <div class="tab-actions" v-if="conflictData?.conflicts?.length">
            <el-button size="small" @click="setAllDecisions('skip')">全部跳过</el-button>
            <el-button size="small" type="danger" @click="setAllDecisions('overwrite')">全部覆盖</el-button>
          </div>
        </div>

        <div v-if="!conflictData?.conflicts?.length" class="empty-state">
          <Icon icon="mdi:check-circle" :size="48" style="color: var(--el-color-success)" />
          <p>未检测到名称冲突</p>
        </div>

        <div v-else class="conflict-table-wrap">
          <el-table :data="conflictData.conflicts" size="small" border max-height="400">
            <el-table-column label="类型" prop="type" width="100">
              <template #default="{ row }">
                <el-tag :type="conflictTypeTag(row.type)" size="small">{{ conflictTypeName(row.type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="名称" prop="name" min-width="180" />
            <el-table-column label="已有来源" prop="existing_source" min-width="140" />
            <el-table-column label="操作" width="180">
              <template #default="{ row }">
                <el-button
                  size="small"
                  :type="getDecision(row.type, row.name) === 'skip' ? 'info' : ''"
                  @click="setDecision(row.type, row.name, 'skip')"
                >跳过</el-button>
                <el-button
                  size="small"
                  :type="getDecision(row.type, row.name) === 'overwrite' ? 'danger' : ''"
                  @click="setDecision(row.type, row.name, 'overwrite')"
                >覆盖</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <!-- Tab 2: Machine Groups -->
      <div v-show="currentStep === 1" class="tab-content">
        <div class="tab-header">
          <h3>机器分组</h3>
        </div>
        <el-alert
          title="机器分组不是必填项，但在执行工作流前需要确保各分组已关联正确的机器"
          type="warning"
          :closable="false"
          show-icon
          class="mb-16"
        />

        <div v-if="!machineGroups.length" class="empty-state">
          <Icon icon="mdi:server" :size="48" style="color: var(--el-text-color-secondary)" />
          <p>当前方案未引用机器分组</p>
        </div>

        <el-table v-else :data="machineGroups" size="small" border>
          <el-table-column label="分组名称" prop="name" width="180" />
          <el-table-column label="关联机器" min-width="350">
            <template #default="{ row }">
              <el-select
                v-model="row.machineIds"
                placeholder="选择机器（多选）"
                multiple
                clearable
                filterable
                style="width: 100%"
              >
                <el-option
                  v-for="m in allMachines"
                  :key="m.id"
                  :label="`${m.hostname || m.ip} (${m.ip}:${m.port})`"
                  :value="m.id"
                />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="已选机器数" width="120">
            <template #default="{ row }">
              {{ row.machineIds?.length || 0 }} 台
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- Tab 3: Global Variables -->
      <div v-show="currentStep === 2" class="tab-content">
        <div class="tab-header">
          <h3>全局变量</h3>
          <div class="tab-actions">
            <el-select v-model="variableGroupFilter" placeholder="全部分组" clearable size="small" style="width: 140px;">
              <el-option v-for="g in variableGroups" :key="g" :label="g" :value="g" />
            </el-select>
            <el-input v-model="variableSearch" placeholder="搜索变量名..." clearable size="small" prefix-icon="Search" style="width: 200px;" />
          </div>
        </div>

        <div v-if="!filteredVariables.length" class="empty-state">
          <Icon icon="mdi:code-json" :size="48" style="color: var(--el-text-color-secondary)" />
          <p>暂无全局变量</p>
        </div>

        <div v-else>
          <div class="variable-list">
            <div v-for="(v, i) in pagedVariables" :key="i" class="variable-item">
              <div class="variable-key">
                <span class="variable-name">{{ v.key }}</span>
                <el-tag v-if="v.group" size="small">{{ v.group }}</el-tag>
              </div>
              <el-input
                v-model="variableValues[v.key]"
                :placeholder="v.value || '输入值...'"
                size="small"
                class="variable-input"
              />
              <div class="variable-desc">{{ v.description || '-' }}</div>
            </div>
          </div>
          <div class="variable-pagination">
            <el-pagination
              v-if="filteredVariables.length > variablePageSize"
              layout="prev, pager, next"
              :total="filteredVariables.length"
              :page-size="variablePageSize"
              v-model:current-page="variablePage"
              small
            />
          </div>
        </div>
      </div>
    </div>
    <!-- END .apply-content -->

      <!-- Tab 4: Dependencies/Files -->
      <div v-show="currentStep === 3" class="tab-content">
        <div class="tab-header">
          <h3>依赖文件</h3>
          <div class="tab-actions">
            <el-button size="small" @click="downloadAllFiles" :loading="downloadingAll">
              <Icon icon="mdi:download" :size="14" /> 批量下载全部
            </el-button>
          </div>
        </div>

        <div v-if="!files.length" class="empty-state">
          <Icon icon="mdi:file-outline" :size="48" style="color: var(--el-text-color-secondary)" />
          <p>当前方案无依赖文件</p>
        </div>

        <el-table v-else :data="files" size="small" border>
          <el-table-column label="文件名" prop="name" min-width="200" />
          <el-table-column label="大小" width="100">
            <template #default="{ row }">{{ formatFileSize(row.size) }}</template>
          </el-table-column>
          <el-table-column label="MD5" prop="md5" min-width="160" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="fileStatusType(row._status)" size="small">{{ fileStatusText(row._status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button size="small" @click="downloadFile(row)" :disabled="row._status === 'downloaded'">
                {{ row.download_url ? '从公网下载' : '下载' }}
              </el-button>
              <el-upload
                :auto-upload="false"
                :show-file-list="false"
                :on-change="(f: any) => handleUpload(row, f)"
              >
                <el-button size="small">上传</el-button>
              </el-upload>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
    <!-- END .apply-scroll -->

    <!-- footer buttons, bottom-anchored via flex -->
    <div class="apply-footer">
      <div class="footer-buttons">
        <el-button v-if="currentStep > 0" @click="currentStep--">上一步</el-button>
        <el-button @click="handleClose">取消</el-button>
        <el-button v-if="currentStep < 3" type="primary" @click="currentStep++" :disabled="currentStep === 0 && !allDecisionsMade">下一步</el-button>
        <el-button
          v-else
          type="primary"
          @click="handleApply"
          :loading="applying"
          :disabled="!allDecisionsMade"
        >
          确认应用
        </el-button>
      </div>
    </div>
  </div>
</el-dialog>  
</template>
<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import { applySolutionLibraryApi, downloadFileAPI, type ConflictResponse } from '@/api/solutionLibrary'

interface MachineGroupItem {
  name: string
  machineIds: number[]
}

interface FileItem {
  name: string
  size: number
  md5: string
  download_url?: string
  _status: 'pending' | 'downloaded' | 'uploaded'
}

const props = defineProps<{
  solutionId: number
}>()

const emit = defineEmits<{
  close: []
  applied: []
}>()

const visible = ref(true)
const loading = ref(true)
const applying = ref(false)
const currentStep = ref(0)
const solution = ref<any>(null)
const conflictData = ref<ConflictResponse | null>(null)
const decisions = ref<Record<string, Record<string, string>>>({})

// Machine groups
const machineGroups = ref<MachineGroupItem[]>([])
const allMachines = ref<any[]>([])

// Variables
const variableSearch = ref('')
const variableGroupFilter = ref('')
const variableValues = ref<Record<string, string>>({})
const allVariables = ref<any[]>([])
const variablePage = ref(1)
const variablePageSize = ref(20)

// Reset page when filters change
watch([variableSearch, variableGroupFilter], () => {
  variablePage.value = 1
})

// Files
const files = ref<FileItem[]>([])
const downloadingAll = ref(false)

const variableGroups = computed(() => {
  const groups = new Set(allVariables.value.map((v: any) => v.group).filter(Boolean))
  return Array.from(groups).sort()
})

const filteredVariables = computed(() => {
  let result = allVariables.value
  if (variableGroupFilter.value) {
    result = result.filter((v: any) => v.group === variableGroupFilter.value)
  }
  if (variableSearch.value) {
    const kw = variableSearch.value.toLowerCase()
    result = result.filter((v: any) => v.key.toLowerCase().includes(kw))
  }
  return result
})

const pagedVariables = computed(() => {
  const start = (variablePage.value - 1) * variablePageSize.value
  return filteredVariables.value.slice(start, start + variablePageSize.value)
})

const allDecisionsMade = computed(() => {
  if (!conflictData.value?.conflicts?.length) return true
  return conflictData.value.conflicts.every(
    (item: any) => decisions.value[item.type]?.[item.name]
  )
})

onMounted(async () => {
  try {
    const res = await applySolutionLibraryApi(props.solutionId) as any
    conflictData.value = res as ConflictResponse
    await loadSolutionDetail()
  } catch (e: any) {
    ElMessage.error('加载失败: ' + (e?.message || '未知错误'))
    closeDialog()
  } finally {
    loading.value = false
  }
})

async function loadSolutionDetail() {
  try {
    const { getSolutionLibraryApi } = await import('@/api/solutionLibrary')
    solution.value = await getSolutionLibraryApi(props.solutionId)

    if (!solution.value?.pack_data) return

    const pack = JSON.parse(solution.value.pack_data)

    // Parse machine groups from stages and top-level machineGroups field
    const mgNames = new Set<string>()
    // 从顶层 machineGroups 字段读取
    for (const mg of pack.machineGroups || []) {
      if (mg.name) mgNames.add(mg.name)
    }
    // 从阶段模板中引用的机器分组读取
    for (const st of pack.stages || []) {
      if (st.machineGroup) mgNames.add(st.machineGroup)
    }
    // 从工作流阶段中引用的机器分组读取
    for (const wf of pack.workflows || []) {
      for (const sg of wf.stageGroups || []) {
        for (const st of sg.stages || []) {
          if (st.machineGroup) mgNames.add(st.machineGroup)
        }
      }
    }

    machineGroups.value = Array.from(mgNames).map(name => ({
      name,
      machineIds: []
    }))

    // 加载全量机器列表，预选同名分组的已有机器
    const { getMachinesApi } = await import('@/api/machine')
    allMachines.value = await getMachinesApi()
    const { getMachineGroupsApi } = await import('@/api/machineGroup')
    const groups = await getMachineGroupsApi()
    for (const mg of machineGroups.value) {
      const match = groups.find((g: any) => g.name === mg.name)
      if (match && match.machines) {
        mg.machineIds = match.machines.map((m: any) => m.id)
      }
    }

    // Parse variables
    allVariables.value = pack.globalVariables || []
    for (const v of allVariables.value) {
      variableValues.value[v.key] = v.value || ''
    }

    // Parse files
    files.value = (pack.materials || []).map((m: any) => ({
      name: m.name,
      size: m.size,
      md5: m.md5,
      download_url: m.download_url,
      _status: 'pending'
    }))
  } catch (e) {
    console.error('Failed to load solution detail', e)
  }
}

function conflictTypeName(type: string): string {
  const names: Record<string, string> = {
    stages: '阶段', variables: '变量', hooks: '钩子',
    templates: '模板', files: '文件', workflows: '工作流'
  }
  return names[type] || type
}

function conflictTypeTag(type: string): string {
  const tags: Record<string, string> = {
    stages: '', variables: 'success', hooks: 'warning',
    templates: 'info', files: 'info', workflows: 'danger'
  }
  return tags[type] || ''
}

function getDecision(type: string, name: string): string {
  return decisions.value[type]?.[name] || ''
}

function setDecision(type: string, name: string, decision: string) {
  if (!decisions.value[type]) {
    decisions.value[type] = {}
  }
  decisions.value[type][name] = decision
}

function setAllDecisions(decision: string) {
  if (!conflictData.value?.conflicts) return
  for (const item of conflictData.value.conflicts) {
    setDecision(item.type, item.name, decision)
  }
}

function stepStatus(index: number) {
  if (currentStep.value > index) return 'success'
  if (currentStep.value === index) return 'process'
  return 'wait'
}

function formatFileSize(bytes: number): string {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

function fileStatusType(status: string) {
  return status === 'downloaded' ? 'success' : status === 'uploaded' ? 'warning' : 'info'
}

function fileStatusText(status: string) {
  return status === 'downloaded' ? '已下载' : status === 'uploaded' ? '已上传' : '未下载'
}

function downloadFile(file: FileItem) {
  const url = file.download_url
  if (!url) {
    ElMessage.warning('该文件无下载链接')
    return
  }
  downloadFileAPI(url).then((blob) => {
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = file.name
    a.click()
    URL.revokeObjectURL(a.href)
    file._status = 'downloaded'
  }).catch(() => {
    ElMessage.error(`下载 ${file.name} 失败`)
  })
}

function downloadAllFiles() {
  downloadingAll.value = true
  const pendings = files.value.filter(f => f.download_url && f._status !== 'downloaded')
  let done = 0
  for (const f of pendings) {
    downloadFileAPI(f.download_url!).then((blob) => {
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = f.name
      a.click()
      URL.revokeObjectURL(a.href)
      f._status = 'downloaded'
    }).catch(() => {
      console.error(`Download failed: ${f.name}`)
    }).finally(() => {
      done++
      if (done >= pendings.length) {
        downloadingAll.value = false
        ElMessage.success(`已下载 ${done} 个文件`)
      }
    })
  }
  if (pendings.length === 0) {
    downloadingAll.value = false
  }
}

function handleUpload(file: FileItem, uploadFile: any) {
  if (uploadFile.raw) {
    file._status = 'uploaded'
    ElMessage.success(`文件 ${file.name} 已上传`)
  }
}

async function handleApply() {
  applying.value = true
  try {
    // 收集机器分组的多选机器
    const mgMachines: Record<string, number[]> = {}
    for (const mg of machineGroups.value) {
      if (mg.machineIds?.length) {
        mgMachines[mg.name] = mg.machineIds
      }
    }

    await applySolutionLibraryApi(
      props.solutionId,
      decisions.value,
      variableValues.value,
      mgMachines
    )
    ElMessage.success('应用成功')
    emit('applied')
    closeDialog()
  } catch (e: any) {
    ElMessage.error('应用失败: ' + (e?.message || '未知错误'))
  } finally {
    applying.value = false
  }
}

function closeDialog() {
  visible.value = false
}

function handleClose() {
  emit('close')
}
</script>

<style scoped>
.apply-dialog :deep(.el-dialog__body) {
  padding: 0;
  display: flex;
  flex-direction: column;
}

.apply-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.apply-title {
  font-size: 16px;
  font-weight: 600;
}

.apply-body {
  height: calc(100vh - 56px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 8px;
}

.apply-steps {
  margin-bottom: 24px;
  flex-shrink: 0;
}

.apply-scroll {
  flex: 1;
  overflow-y: auto;
  padding-bottom: 16px;
}

.tab-content {
  max-width: 900px;
  margin: 0 auto;
}

.tab-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.tab-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.tab-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.empty-state {
  text-align: center;
  padding: 60px 0;
  color: var(--el-text-color-secondary);
}

.empty-state p {
  margin-top: 12px;
}

.conflict-table-wrap {
  border-radius: 6px;
  overflow: hidden;
}

.mb-16 {
  margin-bottom: 16px;
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

.variable-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.variable-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
}

.variable-key {
  width: 180px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.variable-name {
  font-weight: 600;
  font-size: 13px;
  font-family: monospace;
}

.variable-input {
  flex: 1;
}

.variable-desc {
  width: 180px;
  flex-shrink: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.variable-pagination {
  display: flex;
  justify-content: center;
  margin-top: 16px;
  padding-top: 12px;
}

.apply-footer {
  margin-top: auto;
  flex-shrink: 0;
  padding: 12px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}

.footer-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
