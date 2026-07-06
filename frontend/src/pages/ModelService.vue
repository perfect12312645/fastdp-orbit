<template>
  <div class="page-container">
    <div class="page-header">
      <h2>大模型服务</h2>
      <el-button type="primary"><el-icon><Plus /></el-icon> 部署服务</el-button>
    </div>
    <div class="page-content">
      <div class="table-toolbar">
        <div class="table-toolbar-left">
          <el-input placeholder="搜索服务名称" :prefix-icon="Search" clearable style="width: 200px;" />
          <el-select placeholder="服务状态" clearable style="width: 140px;">
            <el-option label="运行中" value="running" />
            <el-option label="已停止" value="stopped" />
            <el-option label="部署中" value="deploying" />
          </el-select>
          <el-button type="primary"><el-icon><Search /></el-icon> 搜索</el-button>
        </div>
      </div>

      <el-empty v-if="!loading && tableData.length === 0" description="暂无模型服务" />
      <el-table v-else v-loading="loading" :data="tableData" border stripe style="width: 100%">
        <el-table-column prop="name" label="服务名称" min-width="160" show-overflow-tooltip />
        <el-table-column prop="modelName" label="模型" min-width="140" show-overflow-tooltip />
        <el-table-column prop="endpoint" label="访问端点" min-width="220" show-overflow-tooltip />
        <el-table-column prop="gpuCount" label="GPU数量" width="100" align="center" />
        <el-table-column prop="replicas" label="副本数" width="90" align="center" />
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'running' ? 'success' : row.status === 'deploying' ? 'warning' : 'info'" size="small">
              {{ row.status === 'running' ? '运行中' : row.status === 'deploying' ? '部署中' : '已停止' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right" align="center">
          <template #default>
            <el-button type="primary" link size="small">编辑</el-button>
            <el-button type="primary" link size="small">详情</el-button>
            <el-button type="danger" link size="small">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Search, Plus } from '@element-plus/icons-vue'

const loading = ref(false)
const tableData = ref([])
</script>

<style scoped></style>
