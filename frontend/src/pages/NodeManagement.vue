<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>节点管理</h2>
        <p class="page-subtitle">管理集群中的所有计算节点</p>
      </div>
      <div class="header-actions">
        <el-button type="success" @click="handleGetInstallCommand">
          <Icon icon="mdi:terminal" :size="16" /> 获取安装命令
        </el-button>
        <el-button type="warning" :loading="syncLoading" @click="handleSyncHardware">
          <Icon icon="mdi:sync" :size="16" /> 同步硬件信息
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
            placeholder="搜索主机名或IP"
            clearable
            style="width: 240px;"
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <Icon icon="mdi:magnify" :size="16" />
            </template>
          </el-input>
          <el-select
            v-model="statusFilter"
            placeholder="节点状态"
            clearable
            style="width: 130px;"
            @change="handleSearch"
          >
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
          </el-select>
        </div>
        <div class="table-toolbar-right">
          <span class="total-text">共 {{ filteredData.length }} 台节点</span>
        </div>
      </div>

      <!-- 数据表格 -->
      <el-table
        v-loading="loading"
        :data="paginatedData"
        border
        stripe
        style="width: 100%"
        row-key="ip"
      >
        <el-table-column prop="hostname" label="主机名" min-width="130">
          <template #default="{ row }">
            <div class="node-name clickable" @click="showDetail(row)">
              <span class="status-dot" :class="row.status === 'online' ? 'status-online' : 'status-offline'"></span>
              {{ row.hostname }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP地址" min-width="140">
          <template #default="{ row }">
            <code class="ip-code">{{ row.ip }}</code>
          </template>
        </el-table-column>
        <el-table-column label="操作系统" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ row.os_name }} {{ row.os_version }}</template>
        </el-table-column>
        <el-table-column label="CPU" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ row.cpu_model }}</span>
            <span class="core-badge">{{ row.cpu_cores }}核</span>
          </template>
        </el-table-column>
        <el-table-column label="内存" width="100" align="center">
          <template #default="{ row }">{{ formatMemoryKB(row.memory_kb) }}</template>
        </el-table-column>
        <el-table-column label="磁盘" width="100" align="center">
          <template #default="{ row }">{{ formatDiskTotal(row.disks) }}</template>
        </el-table-column>
        <el-table-column label="GPU" width="80" align="center">
          <template #default="{ row }">
            <template v-if="row.gpus && row.gpus.length > 0">
              <el-tag type="primary" effect="plain" size="small">
                {{ row.gpus.length }}
              </el-tag>
            </template>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="运行时间" width="120" align="center">
          <template #default="{ row }">{{ formatUptime(row.uptime_seconds) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag
              :type="row.status === 'online' ? 'success' : 'danger'"
              size="small"
              effect="light"
              round
            >
              {{ row.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="danger" link size="small" @click="handleDelete(row)">
              <Icon icon="mdi:delete-outline" :size="14" /> 删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 前端分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="filteredData.length"
          layout="total, sizes, prev, pager, next"
          @size-change="currentPage = 1"
        />
      </div>
    </div>

    <!-- 安装命令弹窗 -->
    <el-dialog
      v-model="installDialogVisible"
      title="Agent 安装命令"
      width="600px"
      destroy-on-close
    >
      <div class="install-command-content">
        <p class="install-tip">在目标服务器上执行以下命令安装 Agent：</p>
        <el-input
          v-model="installCommand"
          type="textarea"
          :rows="4"
          readonly
          class="install-command-input"
        />
        <div class="install-actions">
          <el-button type="primary" @click="copyInstallCommand">
            <Icon icon="mdi:content-copy" :size="16" /> 复制命令
          </el-button>
        </div>
        <el-divider />
        <p class="install-note">
          <Icon icon="mdi:information-outline" :size="14" />
          注意：请确保目标服务器可以访问 Server 的 HTTP 端口
        </p>
      </div>
      <template #footer>
        <el-button @click="installDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 节点详情弹框 -->
    <el-dialog
      v-model="detailVisible"
      :title="detailMachine ? `${detailMachine.hostname} 详细信息` : '节点详情'"
      width="680px"
      destroy-on-close
    >
      <template v-if="detailMachine">
        <div class="detail-grid">
          <div class="detail-section">
            <h4>基本信息</h4>
            <div class="detail-item"><span class="label">主机名</span><span>{{ detailMachine.hostname }}</span></div>
            <div class="detail-item"><span class="label">IP地址</span><code class="ip-code">{{ detailMachine.ip }}:{{ detailMachine.port }}</code></div>
            <div class="detail-item"><span class="label">虚拟化</span><span>{{ detailMachine.virtualization || '-' }}</span></div>
            <div class="detail-item"><span class="label">时区</span><span>{{ detailMachine.timezone }}</span></div>
            <div class="detail-item"><span class="label">网关</span><span>{{ detailMachine.gateway }}</span></div>
          </div>
          <div class="detail-section">
            <h4>系统信息</h4>
            <div class="detail-item"><span class="label">操作系统</span><span>{{ detailMachine.os_name }} {{ detailMachine.os_version }}</span></div>
            <div class="detail-item"><span class="label">内核</span><span>{{ detailMachine.kernel }}</span></div>
            <div class="detail-item"><span class="label">架构</span><span>{{ detailMachine.arch }}</span></div>
            <div class="detail-item"><span class="label">系统时间</span><span>{{ detailMachine.system_time }}</span></div>
            <div class="detail-item"><span class="label">硬件时间</span><span>{{ detailMachine.hardware_time }}</span></div>
            <div class="detail-item"><span class="label">运行时间</span><span>{{ formatUptime(detailMachine.uptime_seconds) }}</span></div>
          </div>
          <div class="detail-section">
            <h4>硬件信息</h4>
            <div class="detail-item"><span class="label">CPU</span><span>{{ detailMachine.cpu_model }} ({{ detailMachine.cpu_cores }}核)</span></div>
            <div class="detail-item"><span class="label">内存</span><span>{{ formatMemoryKB(detailMachine.memory_kb) }}</span></div>
            <div class="detail-item"><span class="label">交换分区</span><span>{{ formatMemoryKB(detailMachine.swap_kb) }}</span></div>
            <div class="detail-item"><span class="label">防火墙</span><span>{{ detailMachine.firewall_status }} ({{ detailMachine.firewall_enabled }})</span></div>
          </div>
          <div class="detail-section" v-if="detailMachine.disks && detailMachine.disks.length > 0">
            <h4>磁盘</h4>
            <div v-for="disk in detailMachine.disks" :key="disk.device" class="detail-item">
              <span class="label">{{ disk.device }}</span>
              <span>{{ disk.type }} - {{ disk.total_gb }} GB</span>
            </div>
          </div>
          <div class="detail-section" v-if="detailMachine.networks && detailMachine.networks.length > 0">
            <h4>网络</h4>
            <div v-for="net in detailMachine.networks" :key="net.name" class="detail-item">
              <span class="label">{{ net.name }}</span>
              <span>{{ net.ip }} ({{ net.mac }}) - {{ net.status }}</span>
            </div>
          </div>
          <div class="detail-section" v-if="detailMachine.gpus && detailMachine.gpus.length > 0">
            <h4>GPU</h4>
            <div v-for="gpu in detailMachine.gpus" :key="gpu.name" class="detail-item">
              <span class="label">{{ gpu.name }}</span>
              <span>驱动: {{ gpu.driver_version }} x{{ gpu.count }}</span>
            </div>
          </div>
        </div>
      </template>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getMachinesApi, syncHardwareApi, deleteMachineApi, type MachineInfo } from '@/api/machine'
import { getInstallCommandApi } from '@/api/install'

const loading = ref(false)
const syncLoading = ref(false)
const machines = ref<MachineInfo[]>([])
const searchText = ref('')
const statusFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(20)

const installDialogVisible = ref(false)
const installCommand = ref('')

const detailVisible = ref(false)
const detailMachine = ref<MachineInfo | null>(null)

const filteredData = computed(() => {
  return machines.value.filter((m) => {
    const matchText = !searchText.value ||
      m.hostname.toLowerCase().includes(searchText.value.toLowerCase()) ||
      m.ip.includes(searchText.value)
    const matchStatus = !statusFilter.value || m.status === statusFilter.value
    return matchText && matchStatus
  })
})

const paginatedData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredData.value.slice(start, start + pageSize.value)
})

async function loadData() {
  loading.value = true
  try {
    machines.value = await getMachinesApi()
  } catch {
    ElMessage.error('获取机器列表失败')
  } finally {
    loading.value = false
  }
}

async function handleSyncHardware() {
  syncLoading.value = true
  try {
    machines.value = await syncHardwareApi()
    ElMessage.success('硬件信息已同步')
  } catch {
    ElMessage.error('同步硬件信息失败')
  } finally {
    syncLoading.value = false
  }
}

function handleSearch() {
  currentPage.value = 1
}

function showDetail(row: MachineInfo) {
  detailMachine.value = row
  detailVisible.value = true
}

async function handleGetInstallCommand() {
  try {
    const res = await getInstallCommandApi()
    installCommand.value = res.command
    installDialogVisible.value = true
  } catch {
    ElMessage.error('获取安装命令失败，请检查网络连接')
  }
}

function copyInstallCommand() {
  navigator.clipboard.writeText(installCommand.value).then(() => {
    ElMessage.success('命令已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败，请手动复制')
  })
}

async function handleDelete(row: MachineInfo) {
  try {
    await ElMessageBox.confirm(
      `确定要删除节点 ${row.hostname}（${row.ip}）吗？\n\n删除后 Agent 下次心跳时会自动退出；若需立即停用，请登录机器执行 systemctl stop orbit-agent`,
      '删除确认',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
    )
    const msg = await deleteMachineApi(row.ip, row.port)
    ElMessage.success(msg || '删除成功')
    loadData()
  } catch {
    // 用户取消
  }
}

function formatMemoryKB(kb: number): string {
  if (!kb) return '-'
  const gb = kb / 1024 / 1024
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  return `${(kb / 1024).toFixed(0)} MB`
}

function formatDiskTotal(disks: { total_gb: number }[]): string {
  if (!disks || disks.length === 0) return '-'
  const total = disks.reduce((sum, d) => sum + d.total_gb, 0)
  return `${total} GB`
}

function formatUptime(seconds: number): string {
  if (!seconds) return '-'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}天${hours}小时`
  if (hours > 0) return `${hours}小时${mins}分`
  return `${mins}分钟`
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

.node-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: var(--font-weight-medium);
}

.node-name.clickable {
  cursor: pointer;
  color: var(--el-color-primary);
}

.node-name.clickable:hover {
  text-decoration: underline;
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

.ip-code {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--text-color-primary);
}

.core-badge {
  margin-left: 6px;
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.text-muted {
  color: var(--text-color-secondary);
}

/* Detail dialog */
.detail-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-section h4 {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color-primary);
  margin-bottom: 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.detail-item {
  display: flex;
  align-items: center;
  padding: 5px 0;
  font-size: 13px;
  line-height: 1.6;
}

.detail-item .label {
  width: 100px;
  flex-shrink: 0;
  color: var(--text-color-secondary);
}

/* Install dialog */
.install-command-content {
  padding: 10px 0;
}

.install-tip {
  margin-bottom: 12px;
  color: var(--text-color-primary);
}

.install-command-input :deep(.el-textarea__inner) {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
  background-color: var(--el-fill-color-lighter);
}

.install-actions {
  display: flex;
  justify-content: flex-end;
}

.install-note {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-color-secondary);
}
</style>
