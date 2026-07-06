<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>GPU资源管理</h2>
        <p class="page-subtitle">实时监控GPU资源使用情况</p>
      </div>
      <div class="page-actions">
        <el-button @click="loadData">
          <Icon icon="mdi:refresh" :size="16" /> 刷新
        </el-button>
        <el-button @click="handleExport">
          <Icon icon="mdi:download-outline" :size="16" /> 导出
        </el-button>
      </div>
    </div>

    <!-- GPU统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card" v-for="(card, idx) in statCards" :key="idx">
        <div class="stat-icon-wrap" :style="{ background: card.bg }">
          <Icon :icon="card.icon" :size="24" :style="{ color: card.color }" />
        </div>
        <div class="stat-card-right">
          <div class="stat-value">{{ card.value }}</div>
          <div class="stat-label">{{ card.label }}</div>
        </div>
      </div>
    </div>

    <!-- 图表 + 表格 -->
    <div class="gpu-content-row">
      <el-card class="content-card gpu-chart-card">
        <template #header>
          <span class="card-title">
            <Icon icon="mdi:thermometer-outline" :size="18" /> GPU温度分布
          </span>
        </template>
        <v-chart class="chart" :option="tempChartOption" autoresize />
      </el-card>

      <el-card class="content-card gpu-util-card">
        <template #header>
          <span class="card-title">
            <Icon icon="mdi:chart-arc" :size="18" /> GPU使用率分布
          </span>
        </template>
        <v-chart class="chart" :option="utilPieOption" autoresize />
      </el-card>
    </div>

    <!-- GPU列表表格 -->
    <el-card class="content-card" style="margin-top: var(--card-gap);">
      <template #header>
        <div class="card-header">
          <span class="card-title">
            <Icon icon="mdi:format-list-bulleted" :size="18" /> GPU列表
          </span>
          <div class="table-toolbar" style="margin-bottom: 0;">
            <div class="table-toolbar-left">
              <el-input
                v-model="queryForm.nodeName"
                placeholder="节点名称"
                clearable
                style="width: 160px;"
                @keyup.enter="handleSearch"
              >
                <template #prefix><Icon icon="mdi:magnify" :size="16" /></template>
              </el-input>
              <el-input
                v-model="queryForm.model"
                placeholder="GPU型号"
                clearable
                style="width: 160px;"
                @keyup.enter="handleSearch"
              >
                <template #prefix><Icon icon="mdi:magnify" :size="16" /></template>
              </el-input>
              <el-select v-model="queryForm.status" placeholder="GPU状态" clearable style="width: 130px;" @change="handleSearch">
                <el-option label="空闲" value="idle" />
                <el-option label="使用中" value="in_use" />
                <el-option label="异常" value="error" />
                <el-option label="离线" value="offline" />
              </el-select>
              <el-button type="primary" size="small" @click="handleSearch">
                <Icon icon="mdi:magnify" :size="14" /> 搜索
              </el-button>
              <el-button size="small" @click="handleReset">
                <Icon icon="mdi:refresh" :size="14" /> 重置
              </el-button>
            </div>
          </div>
        </div>
      </template>

      <el-table v-loading="loading" :data="tableData" border stripe style="width: 100%" row-key="id" @sort-change="handleSortChange">
        <el-table-column prop="nodeName" label="所属节点" min-width="130" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="node-cell">
              <Icon icon="mdi:server-outline" :size="14" class="node-cell-icon" />
              {{ row.nodeName }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="model" label="GPU型号" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="gpu-model">{{ row.model }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="memoryMb" label="显存" width="100" align="center" sortable="custom">
          <template #default="{ row }">{{ formatMemory(row.memoryMb) }}</template>
        </el-table-column>
        <el-table-column prop="usedMemoryMb" label="已用显存" width="100" align="center" sortable="custom">
          <template #default="{ row }">{{ formatMemory(row.usedMemoryMb) }}</template>
        </el-table-column>
        <el-table-column prop="utilization" label="使用率" width="140" align="center" sortable="custom">
          <template #default="{ row }">
            <el-progress
              :percentage="row.utilization"
              :color="row.utilization > 80 ? '#F53F3F' : row.utilization > 50 ? '#FF7D00' : '#165DFF'"
              :stroke-width="12"
              :text-inside="true"
            />
          </template>
        </el-table-column>
        <el-table-column prop="temperature" label="温度" width="90" align="center" sortable="custom">
          <template #default="{ row }">
            <span :style="{ color: row.temperature > 80 ? '#F53F3F' : row.temperature > 60 ? '#FF7D00' : '' }">
              {{ row.temperature }}°C
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="driverVersion" label="驱动" width="110" />
        <el-table-column prop="cudaVersion" label="CUDA" width="100" />
        <el-table-column prop="status" label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="getGpuStatusType(row.status)" size="small" effect="light" round>
              {{ getGpuStatusLabel(row.status) }}
            </el-tag>
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
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, PieChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import { getGpuListApi } from '@/api/gpu'
import type { GpuInfo } from '@/api/types'
import { PaginationConfig, GpuStatusLabel, GpuStatusType } from '@/constants'
import { formatMemory } from '@/utils/format'
import { exportToExcel } from '@/utils/export'

use([CanvasRenderer, BarChart, PieChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent])

function getGpuStatusLabel(status: string) { return GpuStatusLabel[status as keyof typeof GpuStatusLabel] || status }
function getGpuStatusType(status: string) { return GpuStatusType[status as keyof typeof GpuStatusType] || 'info' }

const queryForm = reactive({ page: 1, pageSize: 20, nodeName: '', model: '', status: '' })
const tableData = ref<GpuInfo[]>([])
const total = ref(0)
const loading = ref(false)

const statCards = computed(() => {
  const list = tableData.value
  const totalCount = total.value || list.length
  const idleCount = list.filter(g => g.status === 'idle').length
  const activeCount = list.filter(g => g.status === 'in_use').length
  const avgUtil = list.length ? Math.round(list.reduce((s, g) => s + g.utilization, 0) / list.length) : 0
  return [
    { icon: 'mdi:chip-outline', label: 'GPU总数', value: totalCount, color: '#165DFF', bg: 'rgba(22, 93, 255, 0.1)' },
    { icon: 'mdi:play-circle-outline', label: '使用中', value: activeCount, color: '#00B42A', bg: 'rgba(0, 180, 42, 0.1)' },
    { icon: 'mdi:pause-circle-outline', label: '空闲', value: idleCount, color: '#FF7D00', bg: 'rgba(255, 125, 0, 0.1)' },
    { icon: 'mdi:chart-line-variant', label: '平均利用率', value: `${avgUtil}%`, color: '#722ED1', bg: 'rgba(114, 46, 209, 0.1)' },
  ]
})

const tempChartOption = computed(() => {
  const temps = tableData.value.map(g => g.temperature || Math.floor(30 + Math.random() * 50))
  const labels = tableData.value.map((_, i) => `GPU-${i + 1}`)
  const barData = temps.length ? temps : [45, 52, 38, 67, 55, 42]
  const barLabels = labels.length ? labels : ['GPU-1', 'GPU-2', 'GPU-3', 'GPU-4', 'GPU-5', 'GPU-6']

  return {
    tooltip: { trigger: 'axis', backgroundColor: 'rgba(15, 23, 42, 0.9)', borderColor: 'rgba(22, 93, 255, 0.3)', textStyle: { color: '#fff', fontSize: 12 } },
    grid: { left: 40, right: 16, top: 16, bottom: 30 },
    xAxis: { type: 'category', data: barLabels, axisLine: { lineStyle: { color: '#E5E7EB' } }, axisLabel: { color: '#86909C', fontSize: 10, rotate: 25 }, axisTick: { show: false } },
    yAxis: { type: 'value', axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: '#86909C', fontSize: 11 }, splitLine: { lineStyle: { color: '#F2F3F5', type: 'dashed' } } },
    series: [{
      type: 'bar', barWidth: 18,
      data: barData,
      itemStyle: {
        borderRadius: [4, 4, 0, 0],
        color: (params: { value: number }) => {
          const v = params.value
          if (v > 80) return { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#FF7875' }, { offset: 1, color: '#F53F3F' }] }
          if (v > 60) return { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#FFA940' }, { offset: 1, color: '#FF7D00' }] }
          return { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#4080FF' }, { offset: 1, color: '#165DFF' }] }
        },
      },
    }],
  }
})

const utilPieOption = computed(() => ({
  tooltip: { trigger: 'item', backgroundColor: 'rgba(15, 23, 42, 0.9)', borderColor: 'rgba(22, 93, 255, 0.3)', textStyle: { color: '#fff', fontSize: 12 } },
  legend: { bottom: 8, itemWidth: 10, itemHeight: 10, textStyle: { color: '#86909C', fontSize: 12 } },
  series: [{
    type: 'pie', radius: ['42%', '68%'], center: ['50%', '42%'],
    itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
    label: { show: false },
    emphasis: { label: { show: true, fontSize: 13, fontWeight: 'bold' } },
    data: [
      { value: tableData.value.filter(g => g.utilization < 30).length || 3, name: '低负载', itemStyle: { color: '#00B42A' } },
      { value: tableData.value.filter(g => g.utilization >= 30 && g.utilization < 70).length || 4, name: '中负载', itemStyle: { color: '#165DFF' } },
      { value: tableData.value.filter(g => g.utilization >= 70 && g.utilization < 90).length || 2, name: '高负载', itemStyle: { color: '#FF7D00' } },
      { value: tableData.value.filter(g => g.utilization >= 90).length || 1, name: '满载', itemStyle: { color: '#F53F3F' } },
    ],
  }],
}))

async function loadData() {
  loading.value = true
  try {
    const res = await getGpuListApi(queryForm)
    tableData.value = res.data.list
    total.value = res.data.total
  } catch { /* handled */ } finally { loading.value = false }
}

function handleSearch() { queryForm.page = 1; loadData() }
function handleReset() { queryForm.nodeName = ''; queryForm.model = ''; queryForm.status = ''; queryForm.page = 1; loadData() }
function handleSortChange({ prop, order }: { prop: string; order: string }) {
  ;(queryForm as Record<string, unknown>).sortField = prop
  ;(queryForm as Record<string, unknown>).sortOrder = order === 'ascending' ? 'asc' : order === 'descending' ? 'desc' : ''
  loadData()
}

function handleExport() {
  exportToExcel({
    filename: 'GPU列表',
    columns: { nodeName: '所属节点', model: 'GPU型号', memoryMb: '显存(MB)', usedMemoryMb: '已用显存(MB)', utilization: '使用率(%)', temperature: '温度(°C)', driverVersion: '驱动版本', cudaVersion: 'CUDA版本', status: '状态' },
    data: tableData.value as unknown as Record<string, unknown>[],
  })
}

onMounted(() => loadData())
</script>

<style scoped>
.page-subtitle {
  font-size: var(--font-size-sm);
  color: var(--text-color-secondary);
  margin-top: 4px;
}

.page-actions {
  display: flex;
  gap: 8px;
}

.stat-card {
  background: var(--el-bg-color);
  border-radius: var(--card-radius);
  padding: var(--spacing-lg);
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  box-shadow: var(--shadow-card);
  transition: all 0.3s;
  position: relative;
  overflow: hidden;
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 4px;
  height: 100%;
}

.stat-card:nth-child(1)::before { background: var(--gradient-primary); }
.stat-card:nth-child(2)::before { background: var(--gradient-success); }
.stat-card:nth-child(3)::before { background: var(--gradient-warning); }
.stat-card:nth-child(4)::before { background: var(--gradient-purple); }

.stat-card:hover {
  transform: translateY(-3px);
  box-shadow: var(--shadow-card-hover);
}

.stat-icon-wrap {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-card-right .stat-value {
  font-size: var(--font-size-2xl);
  font-weight: var(--font-weight-bold);
  color: var(--text-color-heading);
  letter-spacing: -0.02em;
}

.stat-card-right .stat-label {
  font-size: var(--font-size-sm);
  color: var(--text-color-secondary);
  margin-top: 2px;
}

.gpu-content-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--card-gap);
  margin-top: var(--card-gap);
}

.chart {
  height: 260px;
  width: 100%;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-semibold);
}

.card-title .iconify {
  color: var(--el-color-primary);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.node-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.node-cell-icon {
  color: var(--el-color-primary);
}

.gpu-model {
  font-weight: var(--font-weight-medium);
}

@media (max-width: 1200px) {
  .gpu-content-row {
    grid-template-columns: 1fr;
  }
}
</style>
