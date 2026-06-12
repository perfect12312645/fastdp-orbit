<template>
  <div class="page-container">
    <div class="page-header">
      <div>
        <h2>仪表盘</h2>
        <p class="page-subtitle">集群运行概览与实时监控</p>
      </div>
      <div class="page-actions">
        <el-button @click="loadData" :icon="Refresh">刷新数据</el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card stat-card--primary" v-for="(card, idx) in statCards" :key="idx">
        <div class="stat-card-left">
          <div class="stat-icon-wrap" :style="{ background: card.bg }">
            <Icon :icon="card.icon" :size="24" :style="{ color: card.color }" />
          </div>
        </div>
        <div class="stat-card-right">
          <div class="stat-value animate-count">{{ card.value }}</div>
          <div class="stat-label">{{ card.label }}</div>
          <div class="stat-footer">
            <span :class="card.subClass">{{ card.sub }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="charts-row">
      <el-card class="content-card chart-card chart-card--wide">
        <template #header>
          <div class="card-header">
            <span class="card-title">
              <Icon icon="mdi:chart-line" :size="18" /> GPU利用率趋势
            </span>
            <div class="card-actions">
              <el-radio-group v-model="timeRange" size="small">
                <el-radio-button value="24h">24小时</el-radio-button>
                <el-radio-button value="7d">7天</el-radio-button>
                <el-radio-button value="30d">30天</el-radio-button>
              </el-radio-group>
            </div>
          </div>
        </template>
        <v-chart class="chart" :option="lineChartOption" autoresize />
      </el-card>

      <el-card class="content-card chart-card">
        <template #header>
          <span class="card-title">
            <Icon icon="mdi:chart-donut" :size="18" /> 资源分布
          </span>
        </template>
        <v-chart class="chart" :option="pieChartOption" autoresize />
      </el-card>
    </div>

    <!-- 第二行图表 -->
    <div class="charts-row">
      <el-card class="content-card chart-card">
        <template #header>
          <span class="card-title">
            <Icon icon="mdi:chart-bar" :size="18" /> 节点GPU数量对比
          </span>
        </template>
        <v-chart class="chart" :option="barChartOption" autoresize />
      </el-card>

      <el-card class="content-card chart-card chart-card--wide">
        <template #header>
          <div class="card-header">
            <span class="card-title">
              <Icon icon="mdi:clock-outline" :size="18" /> 最近活动
            </span>
            <el-button type="primary" link>查看全部</el-button>
          </div>
        </template>
        <el-table :data="recentActivity" style="width: 100%" size="small">
          <el-table-column prop="time" label="时间" width="160" />
          <el-table-column prop="user" label="操作人" width="100" />
          <el-table-column prop="action" label="操作" />
          <el-table-column prop="target" label="目标" />
          <el-table-column prop="result" label="结果" width="80" align="center">
            <template #default="{ row }">
              <el-tag
                :type="row.result === '成功' ? 'success' : 'danger'"
                size="small"
                effect="light"
                round
              >
                {{ row.result }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { Icon } from '@iconify/vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart, BarChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
} from 'echarts/components'
import { getDashboardStatsApi } from '@/api/dashboard'
import type { DashboardStats } from '@/api/types'

use([
  CanvasRenderer,
  LineChart,
  PieChart,
  BarChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
])

const timeRange = ref('24h')

const stats = reactive<DashboardStats>({
  totalNodes: 0,
  onlineNodes: 0,
  totalGpus: 0,
  activeGpus: 0,
  totalServices: 0,
  runningServices: 0,
})

const recentActivity = ref([
  { time: '2024-01-15 14:30', user: 'admin', action: '添加节点', target: 'node-gpu-03', result: '成功' },
  { time: '2024-01-15 13:20', user: 'admin', action: '更新GPU驱动', target: 'node-gpu-01', result: '成功' },
  { time: '2024-01-15 12:10', user: 'user01', action: '部署模型服务', target: 'llama-7b', result: '成功' },
  { time: '2024-01-15 11:00', user: 'admin', action: '删除节点', target: 'node-gpu-05', result: '成功' },
  { time: '2024-01-15 10:30', user: 'user02', action: '重启服务', target: 'chat-service', result: '失败' },
])

const clusterHealth = computed(() => {
  if (stats.totalNodes === 0) return '-'
  return `${((stats.onlineNodes / stats.totalNodes) * 100).toFixed(0)}%`
})

const statCards = computed(() => [
  {
    icon: 'mdi:server-network-outline',
    label: '节点总数',
    value: stats.totalNodes,
    sub: `在线 ${stats.onlineNodes} 台`,
    subClass: 'text-success',
    color: '#165DFF',
    bg: 'rgba(22, 93, 255, 0.1)',
  },
  {
    icon: 'mdi:chip-outline',
    label: 'GPU总数',
    value: stats.totalGpus,
    sub: `使用中 ${stats.activeGpus} 块`,
    subClass: 'text-primary',
    color: '#00B42A',
    bg: 'rgba(0, 180, 42, 0.1)',
  },
  {
    icon: 'mdi:robot-outline',
    label: '模型服务',
    value: stats.totalServices,
    sub: `运行中 ${stats.runningServices} 个`,
    subClass: 'text-warning',
    color: '#FF7D00',
    bg: 'rgba(255, 125, 0, 0.1)',
  },
  {
    icon: 'mdi:heart-pulse-outline',
    label: '集群健康度',
    value: clusterHealth.value,
    sub: '运行正常',
    subClass: 'text-success',
    color: '#722ED1',
    bg: 'rgba(114, 46, 209, 0.1)',
  },
])

/** GPU利用率趋势 - 科技风折线图 */
const lineChartOption = computed(() => {
  const hours = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}:00`)
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(15, 23, 42, 0.9)',
      borderColor: 'rgba(22, 93, 255, 0.3)',
      textStyle: { color: '#fff', fontSize: 12 },
    },
    grid: { left: 40, right: 20, top: 20, bottom: 30 },
    xAxis: {
      type: 'category',
      data: hours,
      axisLine: { lineStyle: { color: '#E5E7EB' } },
      axisLabel: { color: '#86909C', fontSize: 11 },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      max: 100,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: '#86909C', fontSize: 11 },
      splitLine: { lineStyle: { color: '#F2F3F5', type: 'dashed' } },
    },
    series: [
      {
        name: 'GPU利用率',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        data: hours.map(() => Math.floor(40 + Math.random() * 40)),
        lineStyle: { color: '#165DFF', width: 2.5 },
        itemStyle: { color: '#165DFF' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(22, 93, 255, 0.25)' },
              { offset: 1, color: 'rgba(22, 93, 255, 0.02)' },
            ],
          },
        },
      },
    ],
  }
})

/** 资源分布 - 饼图 */
const pieChartOption = computed(() => ({
  tooltip: {
    trigger: 'item',
    backgroundColor: 'rgba(15, 23, 42, 0.9)',
    borderColor: 'rgba(22, 93, 255, 0.3)',
    textStyle: { color: '#fff', fontSize: 12 },
  },
  legend: {
    bottom: 10,
    itemWidth: 10,
    itemHeight: 10,
    textStyle: { color: '#86909C', fontSize: 12 },
  },
  series: [
    {
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['50%', '45%'],
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { show: false },
      emphasis: {
        label: { show: true, fontSize: 14, fontWeight: 'bold' },
      },
      data: [
        { value: stats.activeGpus || 12, name: '已使用', itemStyle: { color: '#165DFF' } },
        { value: (stats.totalGpus - stats.activeGpus) || 8, name: '空闲', itemStyle: { color: '#00B42A' } },
        { value: 2, name: '异常', itemStyle: { color: '#F53F3F' } },
        { value: 1, name: '离线', itemStyle: { color: '#C9CDD4' } },
      ],
    },
  ],
}))

/** 节点GPU数量 - 柱状图 */
const barChartOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
    backgroundColor: 'rgba(15, 23, 42, 0.9)',
    borderColor: 'rgba(22, 93, 255, 0.3)',
    textStyle: { color: '#fff', fontSize: 12 },
  },
  grid: { left: 50, right: 20, top: 20, bottom: 40 },
  xAxis: {
    type: 'category',
    data: ['node-01', 'node-02', 'node-03', 'node-04', 'node-05', 'node-06'],
    axisLine: { lineStyle: { color: '#E5E7EB' } },
    axisLabel: { color: '#86909C', fontSize: 11, rotate: 15 },
    axisTick: { show: false },
  },
  yAxis: {
    type: 'value',
    axisLine: { show: false },
    axisTick: { show: false },
    axisLabel: { color: '#86909C', fontSize: 11 },
    splitLine: { lineStyle: { color: '#F2F3F5', type: 'dashed' } },
  },
  series: [
    {
      type: 'bar',
      barWidth: 24,
      data: [8, 4, 8, 4, 2, 8],
      itemStyle: {
        borderRadius: [6, 6, 0, 0],
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: '#4080FF' },
            { offset: 1, color: '#165DFF' },
          ],
        },
      },
    },
  ],
}))

async function loadData() {
  try {
    const res = await getDashboardStatsApi()
    Object.assign(stats, res.data)
  } catch {
    // 错误已在request拦截器中处理
  }
}

onMounted(() => {
  loadData()
})
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

/* ============ 统计卡片 ============ */
.stat-card {
  background: var(--el-bg-color);
  border-radius: var(--card-radius);
  padding: var(--spacing-lg);
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-md);
  box-shadow: var(--shadow-card);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
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

.stat-card--primary::before { background: var(--gradient-primary); }
.stat-card:nth-child(2)::before { background: var(--gradient-success); }
.stat-card:nth-child(3)::before { background: var(--gradient-warning); }
.stat-card:nth-child(4)::before { background: var(--gradient-purple); }

.stat-card:hover {
  transform: translateY(-4px);
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

.stat-card-right {
  flex: 1;
  min-width: 0;
}

.stat-card .stat-value {
  font-size: var(--font-size-3xl);
  font-weight: var(--font-weight-bold);
  color: var(--text-color-heading);
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.stat-card .stat-label {
  font-size: var(--font-size-sm);
  color: var(--text-color-secondary);
  margin-top: 4px;
}

.stat-card .stat-footer {
  margin-top: var(--spacing-sm);
  padding-top: var(--spacing-sm);
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: var(--font-size-xs);
  color: var(--text-color-secondary);
}

/* ============ 图表区域 ============ */
.charts-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--card-gap);
  margin-bottom: var(--card-gap);
}

.charts-row:last-child {
  margin-bottom: 0;
}

.chart-card {
  border-radius: var(--card-radius) !important;
}

.chart-card--wide {
  grid-column: span 1;
}

.chart {
  height: 300px;
  width: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color-heading);
}

.card-title .iconify {
  color: var(--el-color-primary);
}

/* ============ 响应式 ============ */
@media (max-width: 1400px) {
  .charts-row {
    grid-template-columns: 1fr;
  }
}
</style>
