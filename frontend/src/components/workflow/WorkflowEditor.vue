<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑工作流' : workflow ? '查看工作流' : '创建工作流'"
    width="960px"
    destroy-on-close
    top="3vh"
  >
    <div class="editor-container" v-loading="saving">
      <!-- 左侧：阶段列表 -->
      <div class="stage-panel">
        <div class="panel-header">
          <span>阶段列表</span>
          <el-button type="primary" size="small" @click="addStage">
            <Icon icon="mdi:plus" :size="14" /> 添加
          </el-button>
        </div>
        <div class="stage-list">
          <div
            v-for="(stage, si) in form.stages"
            :key="si"
            class="stage-item"
            :class="{ active: selectedStageIndex === si }"
            @click="selectedStageIndex = si"
          >
            <div class="stage-info">
              <span class="stage-order">{{ si + 1 }}</span>
              <span class="stage-name">{{ stage.name || '未命名阶段' }}</span>
            </div>
            <el-button
              type="danger"
              link
              size="small"
              @click.stop="removeStage(si)"
            >
              <Icon icon="mdi:close" :size="14" />
            </el-button>
          </div>
          <div v-if="form.stages.length === 0" class="empty-tip">
            点击「添加」开始编排
          </div>
        </div>
      </div>

      <!-- 右侧：工作流信息 + 阶段详情 -->
      <div class="detail-panel">
        <!-- 工作流基本信息 -->
        <div class="section">
          <div class="section-title">工作流信息</div>
          <el-form label-width="80px" size="default">
            <el-form-item label="名称">
              <el-input v-model="form.name" placeholder="如：Docker 批量部署" />
            </el-form-item>
            <el-form-item label="描述">
              <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选" />
            </el-form-item>
          </el-form>
        </div>

        <el-divider />

        <!-- 阶段详情 -->
        <template v-if="selectedStage">
          <div class="section">
            <div class="section-title">阶段信息</div>
            <el-form label-width="80px" size="default">
              <el-form-item label="名称">
                <el-input v-model="selectedStage.name" placeholder="如：安装 Docker" />
              </el-form-item>
              <el-form-item label="描述">
                <el-input v-model="selectedStage.description" type="textarea" :rows="1" placeholder="可选" />
              </el-form-item>
              <el-form-item label="目标机器">
                <el-select
                  v-model="selectedStage.machine_group_id"
                  placeholder="选择机器分组"
                  filterable
                  style="width: 100%"
                  :loading="machineGroupLoading"
                >
                  <el-option
                    v-for="g in machineGroups"
                    :key="g.id"
                    :label="g.name"
                    :value="g.id"
                  >
                    <span>{{ g.name }}</span>
                    <span style="color: var(--el-text-color-secondary); margin-left: 8px; font-size: 12px">
                      {{ g.machines?.length || 0 }} 台机器
                    </span>
                  </el-option>
                </el-select>
              </el-form-item>
            </el-form>
          </div>

          <!-- 任务列表 -->
          <div class="section">
            <div class="section-header">
              <div class="section-title">任务列表</div>
              <el-button type="primary" size="small" @click="addTask">
                <Icon icon="mdi:plus" :size="14" /> 添加任务
              </el-button>
            </div>
            <div class="task-list">
              <div v-for="(task, ti) in selectedStage.tasks" :key="ti" class="task-card">
                <div class="task-header">
                  <span class="task-index">{{ ti + 1 }}</span>
                  <el-input v-model="task.name" placeholder="任务名称" class="task-name-input" />
                  <el-button
                    type="danger"
                    link
                    size="small"
                    @click="removeTask(ti)"
                  >
                    <Icon icon="mdi:delete" :size="14" />
                  </el-button>
                </div>
                <div class="task-body">
                  <div class="task-row">
                    <el-select v-model="task.module" placeholder="模块" style="width: 130px">
                      <el-option label="Shell" value="shell" />
                      <el-option label="Systemd" value="systemd" />
                      <el-option label="Package" value="package" />
                      <el-option label="File" value="file" />
                      <el-option label="Template" value="template" />
                      <el-option label="Repo" value="repo" />
                      <el-option label="Blockinfile" value="blockinfile" />
                      <el-option label="Modprobe" value="modprobe" />
                    </el-select>
                    <el-input-number v-model="task.timeout" :min="0" :max="3600" placeholder="超时(秒)" style="width: 120px" />
                    <el-input-number v-model="task.retries" :min="0" :max="10" placeholder="重试次数" style="width: 120px" />
                    <el-input-number v-model="task.delay" :min="0" :max="60" placeholder="重试间隔(秒)" style="width: 140px" />
                  </div>
                  <el-input
                    v-model="task.params"
                    type="textarea"
                    :rows="2"
                    placeholder='参数 JSON，如: {"command": "yum install -y docker-ce"}'
                    class="task-command"
                  />
                  <el-input v-model="task.when" placeholder='条件，如: {{.machine.os_name}} !contains ubuntu' style="width: 100%" />
                  <div class="task-row">
                    <el-input v-model="task.hook_ids" placeholder='后置钩子 ref，如: [1,3]' style="flex: 1" />
                    <el-input v-model="task.register" placeholder="注册变量名" style="width: 150px" />
                    <el-checkbox v-model="task.ignore_errors">忽略错误</el-checkbox>
                  </div>
                </div>
              </div>
              <div v-if="selectedStage.tasks.length === 0" class="empty-tip">
                点击「添加任务」配置此阶段的操作
              </div>
            </div>
          </div>
        </template>
        <div v-else class="empty-stage">
          <Icon icon="mdi:cursor-default-click" :size="48" />
          <p>选择或创建一个阶段</p>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="saveWorkflow" :loading="saving">
        {{ isEdit ? '保存' : '创建' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import { createWorkflowApi, updateWorkflowApi } from '@/api/workflow'
import { getMachineGroupsApi, type MachineGroup } from '@/api/machineGroup'
import type { Workflow, WorkflowStage, WorkflowTask } from '@/types/workflow'

const props = defineProps<{
  modelValue: boolean
  workflow?: Workflow | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'saved'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const isEdit = computed(() => !!props.workflow)
const saving = ref(false)
const selectedStageIndex = ref(0)

const machineGroups = ref<MachineGroup[]>([])
const machineGroupLoading = ref(false)

async function loadMachineGroups() {
  machineGroupLoading.value = true
  try {
    machineGroups.value = await getMachineGroupsApi()
  } catch (e) {
    console.error(e)
  } finally {
    machineGroupLoading.value = false
  }
}

onMounted(loadMachineGroups)

interface StageForm extends WorkflowStage {
  tasks: WorkflowTask[]
}

const form = ref<{
  name: string
  description: string
  stages: StageForm[]
}>({
  name: '',
  description: '',
  stages: [],
})

const selectedStage = computed(() => form.value.stages[selectedStageIndex.value] || null)

watch(
  () => props.workflow,
  (wf) => {
    if (wf) {
      form.value = {
        name: wf.name,
        description: wf.description,
        stages: (wf.stage_groups || []).flatMap((g) =>
          (g.stages || []).map((s) => ({
            ...s,
            machine_group_id: s.machine_group_id || 0,
            tasks: (s.tasks || []).map((t) => ({
              ...t,
              ignore_errors: t.ignore_errors ?? false,
              retries: t.retries ?? 0,
              delay: t.delay ?? 0,
              register: t.register || '',
            })),
          }))
        ),
      }
      selectedStageIndex.value = 0
    } else {
      form.value = { name: '', description: '', stages: [] }
      selectedStageIndex.value = 0
    }
  },
  { immediate: true }
)

let refCounter = 0

function addStage() {
  form.value.stages.push({
    name: '',
    description: '',
    order: form.value.stages.length + 1,
    machine_group_id: 0,
    tasks: [],
  })
  selectedStageIndex.value = form.value.stages.length - 1
}

function removeStage(index: number) {
  form.value.stages.splice(index, 1)
  if (selectedStageIndex.value >= form.value.stages.length) {
    selectedStageIndex.value = Math.max(0, form.value.stages.length - 1)
  }
}

function addTask() {
  if (!selectedStage.value) return
  refCounter++
  selectedStage.value.tasks.push({
    ref: refCounter,
    name: '',
    module: 'shell',
    params: '',
    order: selectedStage.value.tasks.length + 1,
    when: '',
    hook_ids: '',
    loop: '',
    timeout: 0,
    ignore_errors: false,
    retries: 0,
    delay: 0,
    register: '',
  })
}

function removeTask(index: number) {
  if (!selectedStage.value) return
  selectedStage.value.tasks.splice(index, 1)
}

async function saveWorkflow() {
  if (!form.value.name) {
    ElMessage.warning('请输入工作流名称')
    return
  }
  if (form.value.stages.length === 0) {
    ElMessage.warning('至少需要一个阶段')
    return
  }
  for (const stage of form.value.stages) {
    if (!stage.name) {
      ElMessage.warning('阶段名称不能为空')
      return
    }
    if (!stage.machine_group_id) {
      ElMessage.warning(`阶段「${stage.name}」请选择目标机器分组`)
      return
    }
    if (stage.tasks.length === 0) {
      ElMessage.warning(`阶段「${stage.name}」至少需要一个任务`)
      return
    }
    for (const task of stage.tasks) {
      if (!task.name) {
        ElMessage.warning('任务名称不能为空')
        return
      }
      if (!task.module) {
        ElMessage.warning(`任务「${task.name}」的模块类型不能为空`)
        return
      }
    }
  }

  // 重新编号 order + ref
  let globalRef = 1
  form.value.stages.forEach((s, i) => {
    s.order = i + 1
    s.tasks.forEach((t, j) => {
      t.order = j + 1
      t.ref = globalRef++
    })
  })

  saving.value = true
  try {
    const payload = {
      name: form.value.name,
      description: form.value.description,
      config: '',
      stage_groups: [
        {
          name: '默认阶段组',
          description: '',
          order: 1,
          mode: 'sequential' as const,
          stages: form.value.stages.map((s) => ({
            name: s.name,
            description: s.description,
            order: s.order,
            machine_group_id: s.machine_group_id || 0,
            tasks: s.tasks,
          })),
        },
      ],
      variables: [],
      hooks: [],
    }
    if (isEdit.value && props.workflow) {
      await updateWorkflowApi(props.workflow.id, payload)
      ElMessage.success('保存成功')
    } else {
      await createWorkflowApi(payload)
      ElMessage.success('创建成功')
    }
    visible.value = false
    emit('saved')
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.editor-container {
  display: flex;
  gap: 16px;
  min-height: 550px;
  max-height: 75vh;
}

.stage-panel {
  width: 200px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-weight: 600;
  font-size: 14px;
}

.stage-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.stage-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  margin-bottom: 4px;
  transition: all 0.15s;
}

.stage-item:hover {
  background: var(--el-fill-color-light);
}

.stage-item.active {
  background: rgba(22, 93, 255, 0.08);
  color: var(--el-color-primary);
}

.stage-info {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
}

.stage-order {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--el-fill-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}

.stage-name {
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-panel {
  flex: 1;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow-y: auto;
  padding: 16px;
}

.section {
  margin-bottom: 16px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 12px;
}

.section-header .section-title {
  margin-bottom: 0;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.task-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 12px;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.task-index {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: var(--el-color-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}

.task-name-input {
  flex: 1;
}

.task-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.task-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.task-command {
  flex: 1;
}

.empty-tip {
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding: 20px;
}

.empty-stage {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--el-text-color-secondary);
}

.empty-stage p {
  margin-top: 12px;
}
</style>
