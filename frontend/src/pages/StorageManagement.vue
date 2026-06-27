<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>文件存储</h2>
        <p class="page-subtitle">上传文件供 file_pull 模块拉取，支持大文件分片上传和断点续传</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="showUploadDialog">
          <Icon icon="mdi:upload" :size="16" /> 上传文件
        </el-button>
      </div>
    </div>

    <div class="page-content">
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input v-model="searchText" placeholder="搜索文件名" clearable style="width: 240px;" @input="debounceSearch">
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
        </div>
        <div class="table-toolbar-right">
          <span class="total-text">共 {{ files.length }} 个文件</span>
        </div>
      </div>

      <el-table :data="files" v-loading="loading" stripe>
        <el-table-column label="文件名" prop="name" min-width="200" show-overflow-tooltip />
        <el-table-column label="大小" width="120">
          <template #default="{ row }">
            {{ formatSize(row.size) }}
          </template>
        </el-table-column>
        <el-table-column label="MD5" width="260">
          <template #default="{ row }">
            <span v-if="row.md5" class="md5-text">{{ row.md5 }}</span>
            <span v-else class="md5-pending">计算中...</span>
          </template>
        </el-table-column>
        <el-table-column label="上传时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="copyDownloadUrl(row)">
              <Icon icon="mdi:link" :size="14" /> 复制链接
            </el-button>
            <el-button type="success" link size="small" @click="copyWgetCommand(row)">
              <Icon icon="mdi:console" :size="14" /> wget
            </el-button>
            <el-button type="danger" link size="small" @click="deleteFile(row)">
              <Icon icon="mdi:delete-outline" :size="14" />
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 上传对话框 -->
    <el-dialog v-model="uploadDialogVisible" title="上传文件" width="600px" destroy-on-close>
      <div class="upload-area">
        <el-upload
          ref="uploadRef"
          drag
          :auto-upload="false"
          :limit="1"
          :on-change="handleFileChange"
          :on-exceed="handleExceed"
          :before-upload="beforeUpload"
        >
          <Icon icon="mdi:cloud-upload-outline" :size="48" style="color: var(--el-color-primary)" />
          <div class="el-upload__text">拖拽文件到此处，或 <em>点击选择</em></div>
          <template #tip>
            <div class="el-upload__tip">支持大文件分片上传，中断后可自动续传</div>
          </template>
        </el-upload>

        <!-- 上传进度 -->
        <div v-if="uploading" class="upload-progress">
          <div class="progress-info">
            <span>{{ uploadFileName }}</span>
            <span>{{ uploadStatusText }}</span>
          </div>
          <el-progress :percentage="uploadPercent" :status="uploadPercent === 100 ? 'success' : ''" />
          <div class="progress-detail">
            分块 {{ currentChunk }} / {{ totalChunks }}
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="uploadDialogVisible = false" :disabled="uploading">取消</el-button>
        <el-button type="primary" @click="startUpload" :loading="uploading" :disabled="!selectedFile">
          {{ uploading ? '上传中...' : '开始上传' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { UploadInstance, UploadFile, UploadRawFile } from 'element-plus'
import { Icon } from '@iconify/vue'
import {
  getStorageFilesApi,
  deleteStorageFileApi,
  uploadChunkApi,
  getResumeInfoApi,
  getDownloadUrl,
  getWgetCommand,
  type StorageFile,
} from '@/api/storage'

// ==================== 数据 ====================
const loading = ref(false)
const files = ref<StorageFile[]>([])
const searchText = ref('')

// ==================== 上传相关 ====================
const uploadDialogVisible = ref(false)
const uploadRef = ref<UploadInstance>()
const selectedFile = ref<File | null>(null)
const uploading = ref(false)
const uploadPercent = ref(0)
const uploadFileName = ref('')
const uploadStatusText = ref('')
const currentChunk = ref(0)
const totalChunks = ref(0)

const CHUNK_SIZE = 5 * 1024 * 1024 // 5MB

// ==================== 加载数据 ====================
async function loadFiles() {
  loading.value = true
  try {
    files.value = await getStorageFilesApi(searchText.value || undefined)
  } catch {
    files.value = []
  } finally {
    loading.value = false
  }
}

let searchTimer: ReturnType<typeof setTimeout>
function debounceSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => loadFiles(), 300)
}

// ==================== 上传逻辑 ====================
function showUploadDialog() {
  selectedFile.value = null
  uploading.value = false
  uploadPercent.value = 0
  uploadFileName.value = ''
  uploadStatusText.value = ''
  currentChunk.value = 0
  totalChunks.value = 0
  uploadDialogVisible.value = true
}

function handleFileChange(file: UploadFile) {
  selectedFile.value = file.raw || null
}

function handleExceed() {
  ElMessage.warning('只能选择一个文件')
}

function beforeUpload(file: UploadRawFile) {
  selectedFile.value = file
  return false // 阻止自动上传
}

async function startUpload() {
  if (!selectedFile.value) return

  const file = selectedFile.value
  uploadFileName.value = file.name
  uploading.value = true
  uploadPercent.value = 0

  // 计算总分块数
  totalChunks.value = Math.ceil(file.size / CHUNK_SIZE)
  if (totalChunks.value === 0) totalChunks.value = 1

  // 检查是否有续传信息
  try {
    const resumeInfo = await getResumeInfoApi(file.name)
    if (resumeInfo.file_exists && resumeInfo.uploaded_chunks > 0) {
      currentChunk.value = resumeInfo.uploaded_chunks
      uploadStatusText.value = `续传中，已上传 ${resumeInfo.uploaded_chunks} 块`
      ElMessage.info(`检测到已上传部分，从第 ${resumeInfo.uploaded_chunks + 1} 块继续`)
    } else {
      currentChunk.value = 0
      uploadStatusText.value = '上传中...'
    }
  } catch {
    currentChunk.value = 0
    uploadStatusText.value = '上传中...'
  }

  // 分块上传
  try {
    while (currentChunk.value < totalChunks.value) {
      const start = currentChunk.value * CHUNK_SIZE
      const end = Math.min(start + CHUNK_SIZE, file.size)
      const chunk = file.slice(start, end)

      const result = await uploadChunkApi(
        file.name,
        currentChunk.value,
        totalChunks.value,
        chunk,
        (percent) => {
          // 单块内进度
        }
      )

      currentChunk.value++
      uploadPercent.value = Math.round((currentChunk.value / totalChunks.value) * 100)

      if (result.status === 'completed') {
        uploadStatusText.value = '上传完成！'
        uploadPercent.value = 100
        ElMessage.success('文件上传成功')
        loadFiles()
        setTimeout(() => {
          uploadDialogVisible.value = false
        }, 1500)
        return
      }
    }
  } catch (e: any) {
    ElMessage.error('上传失败: ' + (e.message || '未知错误'))
    uploadStatusText.value = '上传失败'
  } finally {
    uploading.value = false
  }
}

// ==================== 文件操作 ====================
function copyDownloadUrl(file: StorageFile) {
  const url = getDownloadUrl(file.path)
  navigator.clipboard.writeText(url).then(() => {
    ElMessage.success('下载链接已复制')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

function copyWgetCommand(file: StorageFile) {
  const cmd = getWgetCommand(file.path)
  navigator.clipboard.writeText(cmd).then(() => {
    ElMessage.success('wget 命令已复制')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

async function deleteFile(file: StorageFile) {
  try {
    await ElMessageBox.confirm(`确定删除文件「${file.name}」？`, '确认删除', {
      type: 'warning',
    })
    await deleteStorageFileApi(file.id)
    ElMessage.success('删除成功')
    loadFiles()
  } catch {
    // 取消
  }
}

// ==================== 工具函数 ====================
function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i]
}

function formatDateTime(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// ==================== 初始化 ====================
onMounted(() => {
  loadFiles()
})
</script>

<style scoped>
.upload-area {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.upload-progress {
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 14px;
}

.progress-detail {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.md5-text {
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.md5-pending {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
