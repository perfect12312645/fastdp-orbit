<template>
  <el-popover
    ref="popoverRef"
    placement="bottom-start"
    :width="380"
    trigger="click"
    :show-arrow="false"
    @show="onShow"
  >
    <template #reference>
      <el-button size="small" :type="buttonType" @click.stop>
        <Icon icon="mdi:code-braces" :size="14" />
        <span v-if="!hideLabel" style="margin-left: 4px">{{ buttonText }}</span>
      </el-button>
    </template>

    <div class="var-picker">
      <el-input
        v-model="searchText"
        placeholder="搜索变量..."
        clearable
        size="small"
        class="var-picker-search"
      >
        <template #prefix>
          <Icon icon="mdi:magnify" :size="14" />
        </template>
      </el-input>

      <div class="var-picker-list">
        <template v-if="filteredGlobalVars.length > 0">
          <div class="var-picker-group">
            <div class="var-picker-group-title">
              <Icon icon="mdi:code-json" :size="14" /> 全局变量
            </div>
            <div
              v-for="v in filteredGlobalVars"
              :key="'gv_' + v.key"
              class="var-picker-item"
              @click="selectVariable(`{{ .GlobalVariable.${v.key} }}`)"
            >
              <span class="var-picker-item-name">{{ v.key }}</span>
              <span class="var-picker-item-type">{{ v.type }}</span>
              <span class="var-picker-item-value" v-if="v.value">{{ truncate(v.value, 30) }}</span>
            </div>
          </div>
        </template>

        <template v-if="filteredMachineVars.length > 0">
          <div class="var-picker-group">
            <div class="var-picker-group-title">
              <Icon icon="mdi:server" :size="14" /> 机器属性
            </div>
            <div
              v-for="v in filteredMachineVars"
              :key="'mv_' + v.key"
              class="var-picker-item"
              @click="selectVariable(machineVarExpr(v.key))"
            >
              <span class="var-picker-item-name">{{ v.key }}</span>
              <span class="var-picker-item-desc">{{ v.label }}</span>
            </div>
            <div class="var-picker-item var-picker-item-loop" @click="selectVariable(gpuLoopExpr())">
              <span class="var-picker-item-name">gpus</span>
              <span class="var-picker-item-desc">GPU列表（循环）</span>
            </div>
            <div class="var-picker-item var-picker-item-loop" @click="selectVariable(diskLoopExpr())">
              <span class="var-picker-item-name">disks</span>
              <span class="var-picker-item-desc">磁盘列表（循环）</span>
            </div>
            <div class="var-picker-item var-picker-item-loop" @click="selectVariable(networkLoopExpr())">
              <span class="var-picker-item-name">networks</span>
              <span class="var-picker-item-desc">网卡列表（循环）</span>
            </div>
          </div>
        </template>

        <template v-if="filteredGroupVars.length > 0">
          <div class="var-picker-group">
            <div class="var-picker-group-title">
              <Icon icon="mdi:server-network" :size="14" /> 当前分组
            </div>
            <div
              v-for="v in filteredGroupVars"
              :key="'gv2_' + v.key"
              class="var-picker-item"
              @click="selectVariable(`{{ .Group.${v.key} }}`)"
            >
              <span class="var-picker-item-name">{{ v.key }}</span>
              <span class="var-picker-item-desc">{{ v.label }}</span>
            </div>
          </div>
        </template>

        <template v-if="filteredServerVars.length > 0">
          <div class="var-picker-group">
            <div class="var-picker-group-title">
              <Icon icon="mdi:server" :size="14" /> 平台配置
            </div>
            <div
              v-for="v in filteredServerVars"
              :key="'sv_' + v.key"
              class="var-picker-item"
              @click="selectVariable(serverVarExpr(v.key))"
            >
              <span class="var-picker-item-name">{{ v.key }}</span>
              <span class="var-picker-item-desc">{{ v.label }}</span>
            </div>
          </div>
        </template>

        <template v-if="filteredGroupsVars.length > 0">
          <div class="var-picker-group">
            <div class="var-picker-group-title">
              <Icon icon="mdi:server-network" :size="14" /> 所有分组（循环引用）
            </div>
            <div
              v-for="g in filteredGroupsVars"
              :key="'groups_' + g.name"
              class="var-picker-item"
              @click="selectVariable(`{{ range $index, $value := .Groups.${g.name} }}\\n  {{ $value.ip }}\\n{{ end }}`)"
            >
              <span class="var-picker-item-name">{{ g.name }}</span>
              <span class="var-picker-item-desc">{{ g.count }} 台机器</span>
            </div>
          </div>
        </template>

        <template v-if="filteredRegisteredVars.length > 0">
          <div class="var-picker-group">
            <div class="var-picker-group-title">
              <Icon icon="mdi:variable" :size="14" /> 注册参数
            </div>
            <div
              v-for="v in filteredRegisteredVars"
              :key="'rv_' + v"
              class="var-picker-item"
              @click="selectVariable(`{{ .Register.${v}.stdout }}`)"
            >
              <span class="var-picker-item-name">{{ v }}</span>
              <span class="var-picker-item-desc">stdout / changed</span>
            </div>
          </div>
        </template>

        <div v-if="noResults" class="var-picker-empty">
          无匹配变量
        </div>
      </div>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { getGlobalVariablesApi, type GlobalVariable } from '@/api/globalVariable'

const props = withDefaults(defineProps<{
  buttonType?: string
  hideLabel?: boolean
  buttonText?: string
  registeredVars?: string[]
  machineGroups?: Array<{ name: string; count: number }>
}>(), {
  buttonType: 'default',
  hideLabel: false,
  buttonText: '插入变量',
  registeredVars: () => [],
  machineGroups: () => [],
})

const emit = defineEmits<{
  (e: 'select', expression: string): void
}>()

const popoverRef = ref()
const searchText = ref('')
const globalVars = ref<GlobalVariable[]>([])

interface MachineVar {
  key: string
  label: string
}

const machineVars: MachineVar[] = [
  { key: 'ip', label: 'IP 地址' },
  { key: 'hostname', label: '主机名' },
  { key: 'os_name', label: '操作系统' },
  { key: 'os_version', label: '系统版本' },
  { key: 'arch', label: '架构' },
  { key: 'kernel', label: '内核版本' },
  { key: 'cpu_model', label: 'CPU 型号' },
  { key: 'cpu_cores', label: 'CPU 核数' },
  { key: 'memory_kb', label: '内存(KB)' },
  { key: 'swap_kb', label: 'Swap(KB)' },
  { key: 'gateway', label: '网关' },
  { key: 'virtualization', label: '虚拟化类型' },
  { key: 'timezone', label: '时区' },
]

interface GroupVar {
  key: string
  label: string
}

const groupVars: GroupVar[] = [
  { key: 'name', label: '分组名称' },
]

const serverVars: GroupVar[] = [
  { key: 'ip', label: 'Server IP' },
  { key: 'port', label: 'Server 端口' },
  { key: 'protocol', label: '协议(http/https)' },
]

const filteredGlobalVars = computed(() => {
  const kw = searchText.value.toLowerCase()
  return globalVars.value.filter(
    (v) => v.key.toLowerCase().includes(kw) || (v.description || '').toLowerCase().includes(kw)
  )
})

const filteredMachineVars = computed(() => {
  const kw = searchText.value.toLowerCase()
  return machineVars.filter(
    (v) => v.key.toLowerCase().includes(kw) || v.label.toLowerCase().includes(kw)
  )
})

const filteredGroupVars = computed(() => {
  const kw = searchText.value.toLowerCase()
  return groupVars.filter(
    (v) => v.key.toLowerCase().includes(kw) || v.label.toLowerCase().includes(kw)
  )
})

const filteredServerVars = computed(() => {
  const kw = searchText.value.toLowerCase()
  return serverVars.filter(
    (v) => v.key.toLowerCase().includes(kw) || v.label.toLowerCase().includes(kw)
  )
})

const filteredRegisteredVars = computed(() => {
  const kw = searchText.value.toLowerCase()
  return props.registeredVars.filter(
    (v) => v.toLowerCase().includes(kw)
  )
})

const filteredGroupsVars = computed(() => {
  const kw = searchText.value.toLowerCase()
  return props.machineGroups.filter(
    (g) => g.name.toLowerCase().includes(kw)
  )
})

const noResults = computed(() => {
  return (
    filteredGlobalVars.value.length === 0 &&
    filteredMachineVars.value.length === 0 &&
    filteredGroupVars.value.length === 0 &&
    filteredServerVars.value.length === 0 &&
    filteredGroupsVars.value.length === 0 &&
    filteredRegisteredVars.value.length === 0
  )
})

function truncate(s: string, max: number): string {
  return s.length > max ? s.slice(0, max) + '...' : s
}

// 表达式生成函数
function machineVarExpr(key: string) {
  return `{{ .Machine.${key} }}`
}

function gpuLoopExpr() {
  return `{{ range $gpuIdx, $gpu := .Machine.gpus }}
GPU{{$gpuIdx}}: {{$gpu.name}} 数量:{{$gpu.count}} 驱动:{{$gpu.driver_version}}
{{ end }}`
}

function diskLoopExpr() {
  return `{{ range $diskIdx, $disk := .Machine.disks }}
磁盘{{$diskIdx}}: {{$disk.device}} {{$disk.total_gb}}GB
{{ end }}`
}

function networkLoopExpr() {
  return `{{ range $netIdx, $net := .Machine.networks }}
网卡{{$netIdx}}: {{$net.ip}} {{$net.mac}}
{{ end }}`
}

function serverVarExpr(key: string) {
  return `{{ .Server.${key} }}`
}

function selectVariable(expression: string) {
  emit('select', expression)
  searchText.value = ''
  popoverRef.value?.hide()
}

function onShow() {
  searchText.value = ''
}

onMounted(async () => {
  try {
    globalVars.value = await getGlobalVariablesApi()
  } catch {
    globalVars.value = []
  }
})
</script>

<style scoped>
.var-picker {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.var-picker-search :deep(.el-input__wrapper) {
  box-shadow: none;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
}

.var-picker-list {
  max-height: 320px;
  overflow-y: auto;
}

.var-picker-group {
  margin-bottom: 8px;
}

.var-picker-group-title {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
  margin-bottom: 4px;
}

.var-picker-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.15s;
}

.var-picker-item:hover {
  background: var(--el-fill-color-light);
}

.var-picker-item-name {
  font-family: monospace;
  font-weight: 500;
  color: var(--el-color-primary);
}

.var-picker-item-type {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
  padding: 1px 6px;
  border-radius: 4px;
}

.var-picker-item-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: auto;
}

.var-picker-item-loop {
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  margin-top: 2px;
}

.var-picker-item-loop:hover {
  background: var(--el-fill-color-light);
}

.var-picker-item-value {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  margin-left: auto;
  font-family: monospace;
}

.var-picker-empty {
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding: 16px 0;
}
</style>
