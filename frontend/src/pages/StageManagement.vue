<template>
  <div class="page-container">
    <!-- 列表视图 -->
    <template v-if="viewMode === 'list'">
      <div class="page-header">
        <div>
          <h2>阶段管理</h2>
          <p class="page-subtitle">预编排阶段模板，供工作流画布拖拽使用</p>
        </div>
        <div class="header-actions">
          <el-button type="primary" @click="showCreateEditor">
            <Icon icon="mdi:plus" :size="16" /> 创建阶段
          </el-button>
        </div>
      </div>

      <div class="page-content">
        <div class="table-toolbar">
          <div class="table-toolbar-left">
            <el-input v-model="searchText" placeholder="搜索阶段名称" clearable style="width: 240px;">
              <template #prefix>
                <Icon icon="mdi:magnify" :size="16" />
              </template>
            </el-input>
            <el-select v-model="selectedSource" placeholder="按来源筛选" clearable style="width: 160px;">
              <el-option v-for="g in stageSources" :key="g" :label="g || '(默认)'" :value="g" />
            </el-select>
          </div>
          <div class="table-toolbar-right">
            <span class="total-text">共 {{ filteredStages.length }} 个阶段</span>
          </div>
        </div>

        <div class="stage-cards" v-loading="loading">
          <div
            v-for="stage in paginatedStages"
            :key="stage.id"
            class="stage-card"
          >
            <div class="stage-card-header">
              <div class="stage-card-title">
                <Icon icon="mdi:view-column-outline" :size="20" />
                <span>{{ stage.name }}</span>
                <el-tag size="small" type="info" effect="plain" class="version-tag">{{ stage.version }}</el-tag>
              </div>
              <div class="stage-card-actions">
                <el-button type="success" link size="small" @click="showExecuteDialog(stage)">
                  <Icon icon="mdi:play" :size="14" /> 执行
                </el-button>
                <el-button type="info" link size="small" @click="showExecutionHistory(stage)">
                  <Icon icon="mdi:history" :size="14" /> 执行记录
                </el-button>
                <el-button type="primary" link size="small" @click="showVersionHistory(stage)">
                  <Icon icon="mdi:source-commit" :size="14" /> 版本
                </el-button>
                <el-button type="primary" link size="small" @click="editStage(stage)">
                  <Icon icon="mdi:pencil" :size="14" /> 编辑
                </el-button>
                <el-button type="danger" link size="small" @click="deleteStage(stage)">
                  <Icon icon="mdi:delete-outline" :size="14" />
                </el-button>
              </div>
            </div>
            <p v-if="stage.description" class="stage-card-desc">{{ stage.description }}</p>
            <div class="stage-card-info">
              <el-tag size="small" type="info" effect="plain">
                <Icon icon="mdi:server-network" :size="12" /> {{ getMachineGroupName(stage.machine_group_id) || '未指定分组' }}
              </el-tag>
              <el-tag size="small" type="primary" effect="plain">
                {{ getTasks(stage).length }} 个任务
              </el-tag>
            </div>
            <div class="stage-card-tasks" v-if="getTasks(stage).length > 0">
              <div v-for="(task, ti) in getTasks(stage).slice(0, 3)" :key="ti" class="task-mini">
                <span class="task-mini-index">{{ task.ref }}</span>
                <span class="task-mini-name">{{ task.name || '未命名' }}</span>
                <el-tag size="small" effect="plain" class="task-mini-module">{{ task.module }}</el-tag>
              </div>
              <div v-if="getTasks(stage).length > 3" class="task-more">
                +{{ getTasks(stage).length - 3 }} 个任务
              </div>
            </div>
          </div>

          <div v-if="filteredStages.length === 0 && !loading" class="stage-empty">
            <Icon icon="mdi:view-column-outline" :size="48" />
            <p>暂无阶段模板</p>
            <el-button type="primary" @click="showCreateEditor">创建阶段</el-button>
          </div>
        </div>

        <div class="pagination-wrapper" v-if="filteredStages.length > pageSize">
          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="[10, 20, 50]"
            :total="filteredStages.length"
            layout="total, sizes, prev, pager, next"
            @size-change="currentPage = 1"
          />
        </div>
      </div>
    </template>

    <!-- 全屏编辑视图 -->
    <template v-if="viewMode === 'edit'">
      <div class="fullscreen-view">
        <div class="fullscreen-header">
          <div class="fullscreen-header-left">
            <el-button text @click="exitFullscreen">
              <Icon icon="mdi:arrow-left" :size="20" /> 返回列表
            </el-button>
            <el-divider direction="vertical" />
            <span class="fullscreen-title">{{ editingId ? '编辑阶段' : '创建阶段' }}</span>
            <el-tag v-if="editingId && currentStage" size="small" type="info" effect="plain">
              当前版本: {{ currentStage.version }}
            </el-tag>
          </div>
          <div class="fullscreen-header-right">
            <el-button @click="exitFullscreen">取消</el-button>
            <el-button type="primary" @click="handleSave" :loading="submitting">
              <Icon icon="mdi:content-save" :size="16" />
              {{ editingId ? '保存为新版本' : '创建阶段' }}
            </el-button>
          </div>
        </div>

        <div class="fullscreen-content">
          <el-form :model="formData" label-width="80px" ref="formRef" :rules="formRules">
            <div class="form-section form-section-compact">
              <h3 class="form-section-title">基本信息</h3>
              <el-row :gutter="16">
                <el-col :span="12">
                  <el-form-item label="阶段名称" prop="name">
                    <el-input v-model="formData.name" placeholder="如：安装 Docker" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="目标分组" prop="machine_group_id">
                    <el-select
                      v-if="machineGroups.length > 0"
                      v-model="formData.machine_group_id"
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
                    <div v-else class="empty-group-hint">
                      <span>暂无机器分组，请先</span>
                      <router-link to="/node" class="group-nav-link">创建机器分组</router-link>
                    </div>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="描述" class="form-item-compact">
                <el-input v-model="formData.description" type="textarea" :rows="1" placeholder="可选" />
              </el-form-item>
            </div>

            <div class="form-section">
              <div class="form-section-header">
                <div class="form-section-title-row">
                  <h3 class="form-section-title">任务列表</h3>
                  <el-button type="primary" size="small" @click="addTask">
                    <Icon icon="mdi:plus" :size="14" /> 添加任务
                  </el-button>
                </div>
                <el-radio-group v-model="editMode" size="small">
                  <el-radio-button value="form">
                    <Icon icon="mdi:form-textbox" :size="14" /> 表单
                  </el-radio-button>
                  <el-radio-button value="yaml">
                    <Icon icon="mdi:code-json" :size="14" /> YAML
                  </el-radio-button>
                </el-radio-group>
              </div>

              <!-- 表单模式 -->
              <div v-if="editMode === 'form'" class="task-list">
                <div v-for="(task, ti) in formData.tasks" :key="ti" class="task-card">
                  <div class="task-card-header">
                    <div class="task-card-title">
                      <span class="task-index">{{ task.ref }}</span>
                      <el-input v-model="task.name" placeholder="任务名称" class="task-name-input" />
                    </div>
                    <div class="task-card-actions">
                      <el-button
                        v-if="ti > 0"
                        type="info"
                        link
                        size="default"
                        @click="moveTask(ti, -1)"
                      >
                        <Icon icon="mdi:arrow-up-bold" :size="16" /> 上移
                      </el-button>
                      <el-button
                        v-if="ti < formData.tasks.length - 1"
                        type="info"
                        link
                        size="default"
                        @click="moveTask(ti, 1)"
                      >
                        <Icon icon="mdi:arrow-down-bold" :size="16" /> 下移
                      </el-button>
                      <el-button
                        type="danger"
                        link
                        size="default"
                        @click="removeTask(ti)"
                      >
                        <Icon icon="mdi:delete" :size="16" /> 删除
                      </el-button>
                    </div>
                  </div>
                  <div class="task-card-body">
                    <el-row :gutter="12">
                      <el-col :span="8">
                        <el-form-item label="模块类型" class="task-field">
                           <el-select v-model="task.module" placeholder="选择模块" style="width: 100%" @change="resetTaskParams(task)">
                             <el-option label="Shell" value="shell">
                               <div class="module-option">
                                 <span class="module-option-name">Shell</span>
                                 <span class="module-option-desc">执行Shell命令</span>
                               </div>
                             </el-option>
                             <el-option label="Script" value="script">
                               <div class="module-option">
                                 <span class="module-option-name">Script</span>
                                 <span class="module-option-desc">执行脚本内容</span>
                               </div>
                             </el-option>
                             <el-option label="Systemd" value="systemd">
                               <div class="module-option">
                                 <span class="module-option-name">Systemd</span>
                                 <span class="module-option-desc">管理服务（启动/停止/重启）</span>
                               </div>
                             </el-option>
                             <el-option label="Package" value="package">
                               <div class="module-option">
                                 <span class="module-option-name">Package</span>
                                 <span class="module-option-desc">安装/卸载软件包</span>
                               </div>
                             </el-option>
                             <el-option label="File" value="file">
                               <div class="module-option">
                                 <span class="module-option-name">File</span>
                                 <span class="module-option-desc">文件/目录操作</span>
                               </div>
                             </el-option>
                             <el-option label="Template" value="template">
                               <div class="module-option">
                                 <span class="module-option-name">Template</span>
                                 <span class="module-option-desc">渲染模板并写入文件</span>
                               </div>
                             </el-option>
                             <el-option label="Copy" value="copy">
                               <div class="module-option">
                                 <span class="module-option-name">Copy</span>
                                 <span class="module-option-desc">从Server分发文件到Agent</span>
                               </div>
                             </el-option>
                             <el-option label="File Pull" value="file_pull">
                               <div class="module-option">
                                 <span class="module-option-name">File Pull</span>
                                 <span class="module-option-desc">从URL拉取文件到Agent</span>
                               </div>
                             </el-option>
                             <el-option label="Unarchive" value="unarchive">
                               <div class="module-option">
                                 <span class="module-option-name">Unarchive</span>
                                 <span class="module-option-desc">解压文件</span>
                               </div>
                             </el-option>
                             <el-option label="Repo" value="repo">
                               <div class="module-option">
                                 <span class="module-option-name">Repo</span>
                                 <span class="module-option-desc">管理YUM/APT仓库</span>
                               </div>
                             </el-option>
                             <el-option label="Blockinfile" value="blockinfile">
                               <div class="module-option">
                                 <span class="module-option-name">Blockinfile</span>
                                 <span class="module-option-desc">在文件中插入/更新文本块</span>
                               </div>
                             </el-option>
                             <el-option label="Lineinfile" value="lineinfile">
                               <div class="module-option">
                                 <span class="module-option-name">Lineinfile</span>
                                 <span class="module-option-desc">在文件中插入/替换/删除行</span>
                               </div>
                             </el-option>
                             <el-option label="Cfssl" value="cfssl">
                               <div class="module-option">
                                 <span class="module-option-name">Cfssl</span>
                                 <span class="module-option-desc">生成TLS证书</span>
                               </div>
                             </el-option>
                             <el-option label="Image" value="image">
                               <div class="module-option">
                                 <span class="module-option-name">Image</span>
                                 <span class="module-option-desc">管理容器镜像</span>
                               </div>
                             </el-option>
                             <el-option label="Modprobe" value="modprobe">
                               <div class="module-option">
                                 <span class="module-option-name">Modprobe</span>
                                 <span class="module-option-desc">加载/卸载内核模块</span>
                               </div>
                             </el-option>
                           </el-select>
                        </el-form-item>
                      </el-col>
                      <el-col :span="4">
                        <el-form-item class="task-field">
                          <template #label>
                            <el-tooltip content="0 表示不限制超时" placement="top">
                              <span style="cursor: help">超时(秒) <Icon icon="mdi:information-outline" :size="12" style="vertical-align: middle" /></span>
                            </el-tooltip>
                          </template>
                          <el-input-number v-model="task.timeout" :min="0" :max="3600" style="width: 100%" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="4">
                        <el-form-item label="重试次数" class="task-field">
                          <el-input-number v-model="task.retries" :min="0" :max="10" style="width: 100%" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="4">
                        <el-form-item label="重试间隔(秒)" class="task-field">
                          <el-input-number v-model="task.delay" :min="0" :max="60" style="width: 100%" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="4">
                        <el-form-item label="引用ID" class="task-field">
                          <el-input-number v-model="task.ref" :min="1" style="width: 100%" />
                        </el-form-item>
                      </el-col>
                    </el-row>

                    <div class="params-section">
                      <el-form-item label="执行参数" class="task-field">
                        <template #label>
                          <span>执行参数</span>
                        </template>
                        <div class="params-kv-list">
                          <div v-for="(key, pi) in Object.keys(task.params)" :key="pi" class="params-kv-row">
                            <span class="params-kv-key">{{ key }}</span>
                            <!-- template 模块的 src 参数：下拉选择模板文件 -->
                            <el-select
                              v-if="task.module === 'template' && key === 'src'"
                              v-model="task.params[key]"
                              placeholder="选择模板文件"
                              clearable
                              filterable
                              class="params-kv-value"
                            >
                              <el-option
                                v-for="tpl in workflowTemplates"
                                :key="tpl.id"
                                :label="tpl.name"
                                :value="tpl.name"
                              >
                                <span>{{ tpl.name }}</span>
                                <span style="color: #909399; margin-left: 8px; font-size: 12px;">{{ tpl.description }}</span>
                              </el-option>
                            </el-select>
                            <!-- 其他参数：普通输入框 / 多行输入框 -->
                            <el-input
                              v-else-if="!isMultilineParam(task.module, key)"
                              v-model="task.params[key]"
                              :placeholder="getParamPlaceholder(task.module, key)"
                              class="params-kv-value"
                              @focus="trackFocus"
                            />
                            <el-input
                              v-else
                              v-model="task.params[key]"
                              type="textarea"
                              :rows="4"
                              :placeholder="getParamPlaceholder(task.module, key)"
                              class="params-kv-value"
                              @focus="trackFocus"
                            />
                            <div class="params-kv-actions">
                              <el-dropdown v-if="task.loop && task.loop.length > 0" trigger="click" @command="(cmd: string) => { task.params[key] += cmd }">
                                <el-button size="small" type="success" link class="loop-var-btn">
                                  <Icon icon="mdi:code-braces" :size="14" />循环变量
                                </el-button>
                                <template #dropdown>
                                  <el-dropdown-menu>
                                    <el-dropdown-item v-if="task.loop_mode === 'simple'" :command="'{{.item}}'" v-text="'{{.item}}'" />
                                    <template v-if="task.loop_mode === 'object'">
                                      <el-dropdown-item :command="'{{.item}}'" v-text="'{{.item}}（整个对象）'" />
                                      <el-dropdown-item v-for="k in task.loop_keys" :key="k" :command="`{{.item.${k}}}`" v-text="`{{.item.${k}}}`" />
                                    </template>
                                  </el-dropdown-menu>
                                </template>
                              </el-dropdown>
                              <!-- template 模块的 src 参数不显示插入变量按钮 -->
                              <VariablePicker
                                v-if="!(task.module === 'template' && key === 'src')"
                                button-type="primary"
                                :registered-vars="getRegisteredVarsBefore(formData.tasks, ti)"
                                :machine-groups="machineGroupsForPicker"
                                @select="(expr: string) => { task.params[key] = insertAtCursor(task.params[key], expr) }"
                              />
                            </div>
                          </div>
                          <div v-if="Object.keys(task.params).length === 0" class="params-empty">
                            请先选择模块类型
                          </div>
                        </div>
                      </el-form-item>
                    </div>

                    <el-form-item label="执行条件" class="task-field">
                      <div class="when-builder">
                        <div class="when-row">
                          <VariablePicker
                            button-type="primary"
                            :registered-vars="getRegisteredVarsBefore(formData.tasks, ti)"
                            :machine-groups="machineGroupsForPicker"
                            @select="(expr: string) => { getWhenClause(task, ti).left = insertAtCursor(getWhenClause(task, ti).left, expr); updateWhenFromClause(task, ti) }"
                          />
                          <el-input
                            :model-value="getWhenClause(task, ti).left"
                            @update:model-value="(v: string) => { getWhenClause(task, ti).left = v; updateWhenFromClause(task, ti) }"
                            placeholder="选择或输入变量"
                            class="when-left"
                            size="small"
                            @focus="trackFocus"
                          />
                          <el-select
                            :model-value="getWhenClause(task, ti).operator"
                            @update:model-value="(v: string) => { getWhenClause(task, ti).operator = v; updateWhenFromClause(task, ti) }"
                            size="small"
                            class="when-operator"
                          >
                            <el-option
                              v-for="op in WHEN_OPERATORS"
                              :key="op.value"
                              :label="op.label"
                              :value="op.value"
                            />
                          </el-select>
                          <el-input
                            :model-value="getWhenClause(task, ti).right"
                            @update:model-value="(v: string) => { getWhenClause(task, ti).right = v; updateWhenFromClause(task, ti) }"
                            placeholder="值"
                            class="when-right"
                            size="small"
                            @focus="trackFocus"
                          />
                          <VariablePicker
                            button-type="primary"
                            :registered-vars="getRegisteredVarsBefore(formData.tasks, ti)"
                            :machine-groups="machineGroupsForPicker"
                            @select="(expr: string) => { getWhenClause(task, ti).right = insertAtCursor(getWhenClause(task, ti).right, expr); updateWhenFromClause(task, ti) }"
                          />
                        </div>
                      </div>
                    </el-form-item>

                    <el-form-item label="循环列表" class="task-field">
                      <div class="loop-section">
                        <el-radio-group v-model="task.loop_mode" size="small" class="loop-mode-switch">
                          <el-radio-button value="simple">简单列表</el-radio-button>
                          <el-radio-button value="object">对象列表</el-radio-button>
                        </el-radio-group>

                        <!-- 简单列表模式 -->
                        <div v-if="task.loop_mode === 'simple'" class="loop-input-wrapper">
                          <el-tag
                            v-for="(item, idx) in task.loop_array"
                            :key="idx"
                            closable
                            @close="task.loop_array.splice(idx, 1)"
                            class="loop-tag"
                          >
                            {{ item }}
                          </el-tag>
                          <el-input
                            v-if="loopInputVisible[ti]"
                            ref="loopInputRef"
                            v-model="loopInputValue[ti]"
                            size="small"
                            class="loop-input"
                            placeholder="输入后按 Enter 添加"
                            @keyup.enter="addLoopItem(task, ti)"
                            @blur="addLoopItem(task, ti)"
                          />
                          <el-button v-else size="small" @click="showLoopInput(ti)">+ 添加项</el-button>
                        </div>

                        <!-- 对象列表模式 -->
                        <div v-else class="loop-object-section">
                          <div class="loop-keys-row">
                            <span class="loop-keys-label">列名（英文逗号分隔）：</span>
                          <el-input
                            :model-value="(task.loop_keys || []).join(',')"
                            @update:model-value="(v: string) => { task.loop_keys = v.split(',').map((s: string) => s.trim()) }"
                            placeholder="如: name,host,port"
                            size="small"
                            class="loop-keys-input"
                          />
                          </div>
                          <div v-if="task.loop_keys.length > 0" class="loop-table-wrapper">
                            <table class="loop-table">
                              <thead>
                                <tr>
                                  <th v-for="key in task.loop_keys" :key="key">{{ key }}</th>
                                  <th class="loop-table-action"></th>
                                </tr>
                              </thead>
                              <tbody>
                                <tr v-for="(row, ri) in task.loop_rows" :key="ri">
                                  <td v-for="key in task.loop_keys" :key="key">
                                    <el-input v-model="row[key]" size="small" />
                                  </td>
                                  <td class="loop-table-action">
                                    <el-button link type="danger" size="small" @click="task.loop_rows.splice(ri, 1)">删</el-button>
                                  </td>
                                </tr>
                              </tbody>
                            </table>
                            <el-button size="small" @click="addLoopRow(task)">+ 添加行</el-button>
                          </div>
                        </div>
                      </div>
                    </el-form-item>

                    <el-row :gutter="16">
                      <el-col :span="8">
                        <el-form-item label="后置钩子" class="task-field">
                          <el-select-v2
                            v-model="task.hooks_array"
                            :options="hookTemplateOptions"
                            multiple
                            placeholder="选择钩子模板"
                            style="width: 100%"
                            filterable
                          />
                        </el-form-item>
                      </el-col>
                      <el-col :span="8">
                        <el-form-item label="注册变量" class="task-field">
                          <el-input v-model="task.register" placeholder="变量名" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="8">
                        <el-form-item label="忽略错误" class="task-field">
                          <el-checkbox v-model="task.ignore_errors">即使本任务失败也继续执行后续任务</el-checkbox>
                        </el-form-item>
                      </el-col>
                    </el-row>
                  </div>
                </div>
                <div v-if="formData.tasks.length === 0" class="empty-tip">
                  点击「添加任务」配置此阶段的操作
                </div>
              </div>

              <!-- YAML 模式 -->
              <div v-else class="yaml-editor">
                <div class="yaml-editor-wrapper">
                  <codemirror
                    v-model="yamlContent"
                    :style="{ height: '100%', width: '100%' }"
                    :extensions="codemirrorExtensions"
                    :tab-size="2"
                    :indent-with-tab="true"
                    placeholder="在此编辑 YAML 格式的完整配置（基本信息 + 任务列表）"
                  />
                </div>
                <div class="yaml-actions">
                  <el-button size="small" @click="formatYaml">
                    <Icon icon="mdi:format-align-left" :size="14" /> 格式化
                  </el-button>
                  <el-button size="small" type="warning" @click="yamlToForm">
                    <Icon icon="mdi:transfer-right" :size="14" /> 应用到表单
                  </el-button>
                </div>
              </div>
            </div>
          </el-form>
        </div>
      </div>
    </template>

    <!-- 全屏版本历史视图 -->
    <template v-if="viewMode === 'versions'">
      <div class="fullscreen-view">
        <div class="fullscreen-header">
          <div class="fullscreen-header-left">
            <el-button text @click="exitFullscreen">
              <Icon icon="mdi:arrow-left" :size="20" /> 返回列表
            </el-button>
            <el-divider direction="vertical" />
            <span class="fullscreen-title">版本历史 - {{ versionStage?.name }}</span>
            <el-tag size="small" type="info" effect="plain">
              当前版本: {{ versionStage?.version }}
            </el-tag>
          </div>
        </div>

        <div class="fullscreen-content version-layout">
          <div class="version-sidebar" v-loading="versionLoading">
            <div class="version-list">
              <div
                v-for="v in versionList"
                :key="v.id"
                class="version-item"
                :class="{ 'is-current': v.version === versionStage?.version, 'is-selected': selectedVersion?.id === v.id }"
                @click="selectVersion(v)"
              >
                <div class="version-item-header">
                  <el-tag size="small" :type="v.version === versionStage?.version ? 'primary' : 'info'" effect="plain">
                    {{ v.version }}
                  </el-tag>
                  <span v-if="v.version === versionStage?.version" class="current-badge">当前</span>
                </div>
                <p class="version-note">{{ v.change_note || '无描述' }}</p>
                <span class="version-time">{{ formatTime(v.created_at) }}</span>
              </div>
            </div>
          </div>

          <div class="version-detail">
            <template v-if="selectedVersion">
              <div class="version-detail-header">
                <h3>版本 {{ selectedVersion.version }} 详情</h3>
                <el-button
                  v-if="selectedVersion.version !== versionStage?.version"
                  type="warning"
                  size="small"
                  @click="handleUpdateToVersion(selectedVersion)"
                >
                  <Icon icon="mdi:backup-restore" :size="14" /> 更新到此版本
                </el-button>
              </div>

              <div class="version-detail-content">
                <el-descriptions :column="2" border size="small">
                  <el-descriptions-item label="版本号">{{ selectedVersion.version }}</el-descriptions-item>
                  <el-descriptions-item label="修改描述">{{ selectedVersion.change_note || '无描述' }}</el-descriptions-item>
                  <el-descriptions-item label="阶段名称">{{ selectedVersion.name }}</el-descriptions-item>
                  <el-descriptions-item label="目标分组">{{ getMachineGroupName(selectedVersion.machine_group_id) }}</el-descriptions-item>
                  <el-descriptions-item label="创建时间" :span="2">{{ formatTime(selectedVersion.created_at) }}</el-descriptions-item>
                </el-descriptions>

                <h4 class="detail-section-title">任务列表</h4>
                <div class="version-task-list">
                  <div v-for="(task, ti) in getVersionTasks(selectedVersion)" :key="ti" class="version-task-item">
                    <div class="version-task-header">
                      <span class="task-index">#{{ task.ref }}</span>
                      <span class="version-task-name">{{ task.name || '未命名' }}</span>
                      <el-tag size="small" effect="plain">{{ task.module }}</el-tag>
                    </div>
                    <div class="version-task-body">
                      <div v-if="task.params" class="version-task-params">
                        <span class="params-label">参数:</span>
                        <code>{{ task.params }}</code>
                      </div>
                      <div v-if="task.when" class="version-task-when">
                        <span class="params-label">条件:</span>
                        <code>{{ task.when }}</code>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </template>
            <div v-else class="version-detail-empty">
              <Icon icon="mdi:information-outline" :size="48" />
              <p>选择左侧版本查看详情</p>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 保存确认对话框（仅编辑已有阶段时弹出） -->
    <el-dialog
      v-model="saveDialogVisible"
      title="保存为新版本"
      width="500px"
      destroy-on-close
    >
      <div class="save-dialog-content">
        <div class="save-version-info">
          <div class="save-version-row">
            <span class="save-label">当前版本:</span>
            <el-tag size="small" type="info" effect="plain">
              {{ currentStage?.version }}
            </el-tag>
          </div>
          <div class="save-version-row">
            <span class="save-label">保存后:</span>
            <el-tag size="small" type="success" effect="plain">
              将生成新版本
            </el-tag>
          </div>
        </div>
        <el-form :model="saveFormData" label-width="100px" ref="saveFormRef" :rules="saveFormRules">
          <el-form-item label="修改描述" prop="change_note">
            <el-input
              v-model="saveFormData.change_note"
              type="textarea"
              :rows="3"
              placeholder="请描述本次修改内容，如：添加了安装 Docker 任务"
            />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="saveDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmSave" :loading="submitting">
          确认保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 执行阶段对话框 -->
    <el-dialog v-model="executeDialogVisible" title="执行阶段" width="480px" destroy-on-close>
      <div v-if="executingStage">
        <p style="margin-bottom: 16px; font-size: 14px; color: var(--el-text-color-secondary);">
          即将执行阶段：<strong>{{ executingStage.name }}</strong>
          <el-tag size="small" type="info" effect="plain" style="margin-left: 8px;">
            {{ getMachineGroupName(executingStage.machine_group_id) || '未指定分组' }}
          </el-tag>
        </p>
        <el-collapse v-model="overrideCollapse">
          <el-collapse-item title="覆盖机器分组（可选）" name="override">
            <el-select
              v-model="executeGroupId"
              placeholder="选择其他分组"
              clearable
              style="width: 100%"
            >
              <el-option
                v-for="g in machineGroups"
                :key="g.id"
                :label="`${g.name} (${g.machines?.length || 0} 台)`"
                :value="g.id"
              />
            </el-select>
            <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 4px;">
              不选择则使用上方显示的默认分组
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>
      <template #footer>
        <el-button @click="executeDialogVisible = false">取消</el-button>
        <el-button type="success" @click="handleExecute" :loading="executing">
          <Icon icon="mdi:play" :size="16" /> 执行
        </el-button>
      </template>
    </el-dialog>

    <!-- 执行日志查看器 -->
    <ExecutionLogViewer
      ref="executionLogViewer"
      :execution-id="currentExecutionId"
      :stage-name="executingStage?.name || ''"
      :machine-group-name="getMachineGroupName(executingStage?.machine_group_id || 0)"
    />

    <!-- 执行历史对话框 -->
    <el-dialog v-model="executionHistoryVisible" title="执行记录" width="700px" destroy-on-close>
      <div v-if="executionHistoryStage">
        <p style="margin-bottom: 16px; color: var(--el-text-color-secondary); font-size: 14px;">
          阶段：<strong>{{ executionHistoryStage.name }}</strong>
        </p>
        <el-table :data="executionHistory" v-loading="executionHistoryLoading" stripe>
          <el-table-column label="执行ID" prop="id" width="80" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" size="small">
                {{ getStatusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="触发方式" width="100" prop="trigger" />
          <el-table-column label="错误" min-width="150" show-overflow-tooltip>
            <template #default="{ row }">
              <span v-if="row.error" style="color: var(--el-color-danger);">{{ row.error }}</span>
              <span v-else style="color: var(--el-text-color-secondary);">-</span>
            </template>
          </el-table-column>
          <el-table-column label="开始时间" width="170">
            <template #default="{ row }">
              {{ formatDateTime(row.started_at) }}
            </template>
          </el-table-column>
          <el-table-column label="结束时间" width="170">
            <template #default="{ row }">
              {{ row.finished_at ? formatDateTime(row.finished_at) : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link size="small" @click="viewExecutionLog(row.id)">
                <Icon icon="mdi:console" :size="14" /> 日志
              </el-button>
              <el-button
                v-if="row.status === 'running' || row.status === 'paused'"
                type="danger"
                link
                size="small"
                @click="cancelExecution(row.id)"
              >
                <Icon icon="mdi:stop" :size="14" /> 终止
              </el-button>
              <el-button
                v-if="row.status !== 'running'"
                type="danger"
                link
                size="small"
                @click="deleteExecutionRecord(row.id)"
              >
                <Icon icon="mdi:delete-outline" :size="14" />
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div v-if="executionHistory.length === 0 && !executionHistoryLoading" style="text-align: center; padding: 40px; color: var(--el-text-color-secondary);">
          暂无执行记录
        </div>
      </div>
      <template #footer>
        <el-button @click="executionHistoryVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 重新打开日志按钮 -->
    <transition name="el-fade-in">
      <div
        v-if="currentExecutionId && !logViewerVisible"
        class="reopen-log-btn"
        @click="reopenLogViewer"
      >
        <Icon icon="mdi:console" :size="20" />
        <span>查看执行日志</span>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as yaml from 'js-yaml'
import { Codemirror } from 'vue-codemirror'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView } from '@codemirror/view'
import { getMachineGroupsApi, type MachineGroup } from '@/api/machineGroup'
import ExecutionLogViewer from '@/components/ExecutionLogViewer.vue'
import {
  getStageTemplatesApi,
  createStageTemplateApi,
  updateStageTemplateApi,
  deleteStageTemplateApi,
  listStageTemplateVersionsApi,
  rollbackStageTemplateApi,
  executeStageTemplateApi,
  type StageTemplate,
  type StageTemplateVersion,
} from '@/api/stageTemplate'
import {
  getHookTemplatesApi,
  type HookTemplate,
} from '@/api/hookTemplate'
import {
  getWorkflowTemplatesApi,
  type WorkflowTemplate,
} from '@/api/workflowTemplate'
import { HandledError } from '@/utils/request'
import VariablePicker from '@/components/VariablePicker.vue'

// CodeMirror 配置
const codemirrorExtensions = [oneDark, EditorView.lineWrapping]

interface StageTask {
  ref: number
  name: string
  module: string
  params: Record<string, string>
  order: number
  timeout: number
  retries: number
  delay: number
  when: string
  hooks: string
  hooks_array: string[]
  register: string
  ignore_errors: boolean
  loop: string
  loop_array: string[]
  loop_mode: 'simple' | 'object'
  loop_keys: string[]
  loop_rows: Record<string, string>[]
}

// 每种模块支持的参数 key 和占位说明
const MODULE_PARAMS: Record<string, Record<string, string>> = {
  shell: { command: '执行的命令 [必填]' },
  script: { script: '脚本内容（与script_file二选一）', script_file: '脚本文件路径（与script二选一）' },
  systemd: { name: '服务名称 [必填，reload操作可不填]', action: '操作类型 [必填]: start/stop/restart/reload/status/enable/disable' },
  package: { action: '操作类型 [必填]: install/remove/update/check/localinstall', name: '包名（多包逗号分隔，localinstall时填文件路径）[必填]' },
  file: { path: '目标路径（绝对路径）[必填]', action: '操作类型 [必填]: create/delete/touch/symlink', type: '文件类型（create/symlink时必填）: file/directory', src: '符号链接源路径（symlink时必填）', mode: '权限模式（可选），如 0644', owner: '所有者UID（可选）', group: '所属组GID（可选）', recurse: '递归创建目录（可选）: true/false', force: '强制删除非空目录（可选）: true/false', backup: '操作前备份（可选）: true/false' },
  file_pull: { url: '文件URL（支持http/https）[必填]', dest: '目标路径（绝对路径，以/结尾则自动提取文件名）[必填]', md5: '文件MD5（可选，用于校验）' },
  template: { src: '选择模板文件（与content二选一，引擎层渲染为content）', content: 'Go template模板内容（与src二选一，直接填写时使用）', dest: '目标路径（绝对路径）[必填]', append: '追加模式（可选）: true/false，默认false覆盖' },
  repo: { action: '操作类型 [必填]: add/remove/test/backup/restore/makecache', name: '仓库名称（add/remove时必填）', url: '仓库URL（add/test时必填）' },
  blockinfile: { action: '操作类型 [必填]: ensure/delete', path: '目标文件路径 [必填]', content: '文本块内容（ensure时必填，支持换行）', backup: '操作前备份（可选）: true/false' },
  lineinfile: { path: '目标文件路径（绝对路径）[必填]', regexp: '匹配行的正则表达式 [必填]', line: '目标行内容（insert/replace时必填）', action: '操作类型 [必填]: insert/replace/delete', backrefs: '启用正则反向引用（仅replace时有效）: true/false', insertbefore: '插入到匹配行前（可选）: true/false，默认false插入到匹配行后' },
  modprobe: { module: '内核模块名（与loop二选一）', loop: '模块列表（逗号分隔，与module二选一）', action: '操作类型（可选）: load/remove，默认load', options: '模块加载选项（可选）' },
  cfssl: { action: '操作类型 [必填]: generate_ca/generate_cert', csr_path: 'CSR配置文件路径 [必填]', output_dir: '证书输出目录 [必填]', basename: '输出文件名前缀 [必填]', ca_cert: 'CA证书路径（generate_cert时必填）', ca_key: 'CA私钥路径（generate_cert时必填）', config_file: 'cfssl配置文件（generate_cert时必填）', profile: '配置profile名称（generate_cert时必填）' },
  image: { action: '操作类型 [必填]: load/push/remove/pull', tag: '镜像标签（如nginx:latest）[必填]', path: '镜像文件路径（load时必填，绝对路径）' },
  unarchive: { src: '压缩文件路径（绝对路径）[必填]', dest: '目标目录（绝对路径）[必填]', strip_components: '去除路径层级数（可选），如1' },
  copy: { src: 'Server端源文件路径（绝对路径）[必填]', dest: 'Agent端目标路径（绝对路径）[必填]', type: '类型（可选）: file/dir', recursive: '递归复制（可选）: true/false', mode: '文件权限（可选），如 0644' },
}

// 需要多行输入的参数（textarea）
const MULTILINE_PARAMS: Record<string, string[]> = {
  blockinfile: ['content'],
  script: ['script'],
  file: ['command'],
  template: ['content'],
}

function isMultilineParam(module: string, key: string): boolean {
  return MULTILINE_PARAMS[module]?.includes(key) ?? false
}

function getParamPlaceholder(module: string, key: string): string {
  return MODULE_PARAMS[module]?.[key] || ''
}

function resetTaskParams(task: StageTask) {
  const keys = Object.keys(MODULE_PARAMS[task.module] || {})
  const newParams: Record<string, string> = {}
  for (const k of keys) newParams[k] = ''
  task.params = newParams
}

// 获取当前 task 之前的所有已注册变量名
function getRegisteredVarsBefore(tasks: StageTask[], currentIndex: number): string[] {
  const vars: string[] = []
  for (let i = 0; i < currentIndex; i++) {
    if (tasks[i].register && tasks[i].register.trim()) {
      vars.push(tasks[i].register.trim())
    }
  }
  return vars
}

// 机器分组数据（用于 VariablePicker 的 Groups 引用）
const machineGroupsForPicker = computed(() => {
  return machineGroups.value.map(g => ({
    name: g.name,
    count: g.machines?.length || 0,
  }))
})

// 跟踪最后聚焦的 input 元素（用于在光标位置插入变量）
const lastFocusedInput = ref<HTMLInputElement | HTMLTextAreaElement | null>(null)

function trackFocus(e: FocusEvent) {
  const el = e.target as HTMLInputElement | HTMLTextAreaElement
  if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
    lastFocusedInput.value = el
  }
}

// 在 el-input 光标位置插入文本
function insertAtCursor(value: string, insertText: string): string {
  const el = lastFocusedInput.value
  if (el && el.selectionStart !== null && el.selectionStart !== undefined) {
    const start = el.selectionStart
    const end = el.selectionEnd ?? start
    const before = value.slice(0, start)
    const after = value.slice(end)
    const newValue = before + insertText + after
    // 恢复光标位置
    nextTick(() => {
      el.focus()
      el.selectionStart = el.selectionEnd = start + insertText.length
    })
    return newValue
  }
  // 降级：追加到末尾
  return value + insertText
}

const loading = ref(false)
const searchText = ref('')
const selectedSource = ref('')
const stages = ref<StageTemplate[]>([])
const machineGroups = ref<MachineGroup[]>([])
const machineGroupLoading = ref(false)
const hookTemplates = ref<HookTemplate[]>([])
const workflowTemplates = ref<WorkflowTemplate[]>([])
const currentPage = ref(1)
const pageSize = ref(10)

// 视图模式
const viewMode = ref<'list' | 'edit' | 'versions'>('list')

// 全屏编辑
const editingId = ref(0)
const currentStage = ref<StageTemplate | null>(null)
const formData = ref({
  name: '',
  description: '',
  machine_group_id: 0,
  tasks: [] as StageTask[],
})
const formRules = {
  name: [{ required: true, message: '请输入阶段名称', trigger: 'blur' }],
  machine_group_id: [{ required: true, message: '请选择目标分组', trigger: 'change' }],
}

// 编辑模式：form / yaml
const editMode = ref<'form' | 'yaml'>('form')
const yamlContent = ref('')

// 保存对话框（仅编辑已有阶段）
const saveDialogVisible = ref(false)
const saveFormData = ref({ change_note: '' })
const saveFormRules = {
  change_note: [{ required: true, message: '请描述本次修改内容', trigger: 'blur' }],
}
const saveFormRef = ref()

const submitting = ref(false)
const formRef = ref()

// 执行对话框
const executeDialogVisible = ref(false)
const executingStage = ref<StageTemplate | null>(null)
const executeGroupId = ref(0)
const executing = ref(false)
const overrideCollapse = ref<string[]>([])
const executionLogViewer = ref<InstanceType<typeof ExecutionLogViewer> | null>(null)
const currentExecutionId = ref(0)
const logViewerVisible = ref(false)

// 执行历史
const executionHistoryVisible = ref(false)
const executionHistoryStage = ref<StageTemplate | null>(null)
const executionHistory = ref<any[]>([])
const executionHistoryLoading = ref(false)

function reopenLogViewer() {
  if (executionLogViewer.value && currentExecutionId.value) {
    executionLogViewer.value.open(currentExecutionId.value)
    logViewerVisible.value = true
  }
}

// 执行历史
async function showExecutionHistory(stage: StageTemplate) {
  executionHistoryStage.value = stage
  executionHistoryVisible.value = true
  executionHistoryLoading.value = true
  try {
    // 查询该阶段的执行记录（通过 workflow_executions 表中 workflow_id=0 的记录）
    const { getExecutionHistoryApi } = await import('@/api/stageTemplate')
    executionHistory.value = await getExecutionHistoryApi(stage.id)
  } catch {
    executionHistory.value = []
  } finally {
    executionHistoryLoading.value = false
  }
}

function viewExecutionLog(executionId: number) {
  executionHistoryVisible.value = false
  currentExecutionId.value = executionId
  if (executionLogViewer.value) {
    executionLogViewer.value.open(executionId)
    logViewerVisible.value = true
  }
}

async function cancelExecution(executionId: number) {
  try {
    await ElMessageBox.confirm('确认终止该执行？', '终止确认', {
      confirmButtonText: '终止',
      cancelButtonText: '取消',
      type: 'warning',
    })
    const { cancelStageExecutionApi } = await import('@/api/stageTemplate')
    await cancelStageExecutionApi(executionId)
    ElMessage.success('已发送终止请求')
    // 刷新执行记录
    if (executionHistoryStage.value) {
      showExecutionHistory(executionHistoryStage.value)
    }
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e?.message || '终止失败')
  }
}

async function deleteExecutionRecord(executionId: number) {
  try {
    await ElMessageBox.confirm('确认删除该执行记录？', '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    const { deleteStageExecutionApi } = await import('@/api/stageTemplate')
    await deleteStageExecutionApi(executionId)
    ElMessage.success('删除成功')
    // 刷新执行记录
    if (executionHistoryStage.value) {
      showExecutionHistory(executionHistoryStage.value)
    }
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e?.message || '删除失败')
  }
}

// 版本历史
const versionStage = ref<StageTemplate | null>(null)
const versionLoading = ref(false)
const versionList = ref<StageTemplateVersion[]>([])
const selectedVersion = ref<StageTemplateVersion | null>(null)

// 执行条件结构化编辑
const WHEN_OPERATORS = [
  { label: '包含', value: 'contains' },
  { label: '不包含', value: '!contains' },
  { label: '等于', value: '==' },
  { label: '不等于', value: '!=' },
]

// 状态相关函数
function getStatusType(status: string) {
  const map: Record<string, string> = {
    running: 'warning',
    success: 'success',
    failed: 'danger',
    cancelled: 'info',
    pending: 'info',
  }
  return (map[status] || 'info') as any
}

function getStatusLabel(status: string) {
  const map: Record<string, string> = {
    running: '执行中',
    success: '成功',
    failed: '失败',
    cancelled: '已终止',
    pending: '等待中',
  }
  return map[status] || status
}

function formatDateTime(dateStr: string) {
  if (!dateStr) return '-'
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

interface WhenClause {
  left: string
  operator: string
  right: string
}

function parseWhen(when: string): WhenClause {
  if (!when || !when.trim()) {
    return { left: '', operator: 'contains', right: '' }
  }
  // 尝试解析: {{ .Machine.os_name }} contains 'ubuntu'
  // 或旧格式: {{.machine.os_name}} contains ubuntu
  const m = when.match(/\{\{\s*[\.\$](.+?)\s*\}\}\s*(contains|!contains|==|!=)\s*'?([^']*)'?/)
  if (m) {
    const varName = m[1]
    const left = `{{ .${varName} }}`
    return { left, operator: m[2], right: m[3] || '' }
  }
  // 无法解析时，把整个值放到 right
  return { left: '', operator: 'contains', right: when }
}

function assembleWhen(clause: WhenClause): string {
  if (!clause.left && !clause.right) return ''
  if (!clause.left) return ''
  return `${clause.left} ${clause.operator} ${clause.right}`
}

const whenClauses = ref<Record<number, WhenClause>>({})

function getWhenClause(task: StageTask, ti: number): WhenClause {
  if (!(ti in whenClauses.value)) {
    whenClauses.value[ti] = parseWhen(task.when)
  }
  return whenClauses.value[ti]
}

function updateWhenFromClause(task: StageTask, ti: number) {
  task.when = assembleWhen(whenClauses.value[ti])
}

// 循环列表输入
const loopInputVisible = ref<Record<number, boolean>>({})
const loopInputValue = ref<Record<number, string>>({})
const loopInputRef = ref<InstanceType<typeof import('element-plus')['ElInput']>>()

function showLoopInput(ti: number) {
  loopInputVisible.value[ti] = true
  loopInputValue.value[ti] = ''
  nextTick(() => {
    const input = loopInputRef.value
    if (input) input.focus()
  })
}

function addLoopItem(task: StageTask, ti: number) {
  const val = (loopInputValue.value[ti] || '').trim()
  if (val) {
    if (!task.loop_array) task.loop_array = []
    task.loop_array.push(val)
  }
  loopInputVisible.value[ti] = false
  loopInputValue.value[ti] = ''
}

function addLoopRow(task: StageTask) {
  const row: Record<string, string> = {}
  for (const key of task.loop_keys) {
    row[key] = ''
  }
  task.loop_rows.push(row)
}

const filteredStages = computed(() => {
  let result = stages.value
  if (selectedSource.value) {
    result = result.filter(s => s.source === selectedSource.value)
  }
  if (searchText.value) {
    const kw = searchText.value.toLowerCase()
    result = result.filter(
      (s) => s.name.toLowerCase().includes(kw) || (s.description || '').toLowerCase().includes(kw)
    )
  }
  return result
})

const stageSources = computed(() => {
  const groups = new Set(stages.value.map(s => s.source).filter(Boolean))
  return Array.from(groups).sort()
})

const hookTemplateOptions = computed(() =>
  hookTemplates.value.map(ht => ({
    value: ht.name,
    label: `${ht.name} (${ht.module})`,
  }))
)

const paginatedStages = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredStages.value.slice(start, start + pageSize.value)
})

watch([searchText, selectedSource], () => {
  currentPage.value = 1
})

function normalizeTasks(rawTasks: any[]): StageTask[] {
  return rawTasks.map((t: any, i: number) => {
    let params: Record<string, string> = {}
    if (typeof t.params === 'object' && t.params !== null) {
      params = {}
      for (const [k, v] of Object.entries(t.params)) {
        params[k] = String(v ?? '')
      }
    } else if (typeof t.params === 'string' && t.params.trim()) {
      try {
        const parsed = JSON.parse(t.params)
        if (typeof parsed === 'object' && parsed !== null) {
          params = {}
          for (const [k, v] of Object.entries(parsed)) {
            params[k] = String(v ?? '')
          }
        } else {
          params = { command: t.params }
        }
      } catch {
        params = { command: t.params }
      }
    } else {
      params = { command: '' }
    }

    // 解析 hooks JSON 数组 -> hooks_array（名称列表）
    let hooks_array: string[] = []
    if (t.hooks) {
      try {
        const parsed = JSON.parse(t.hooks)
        if (Array.isArray(parsed)) hooks_array = parsed.map(String)
      } catch {
        hooks_array = []
      }
    }

    // 解析 loop JSON 数组 -> loop_array / loop_rows + loop_mode
    let loop_array: string[] = []
    let loop_mode: 'simple' | 'object' = 'simple'
    let loop_keys: string[] = []
    let loop_rows: Record<string, string>[] = []
    if (t.loop) {
      try {
        const parsed = JSON.parse(t.loop)
        if (Array.isArray(parsed) && parsed.length > 0) {
          if (typeof parsed[0] === 'object' && parsed[0] !== null) {
            loop_mode = 'object'
            loop_keys = Object.keys(parsed[0])
            loop_rows = parsed.map((item: any) => {
              const row: Record<string, string> = {}
              for (const k of loop_keys) {
                row[k] = String(item[k] ?? '')
              }
              return row
            })
          } else {
            loop_array = parsed.map(String)
          }
        }
      } catch {
        loop_array = []
      }
    }

    return {
      ...t,
      params,
      order: t.order || i + 1,
      hooks_array,
      loop_array,
      loop_mode,
      loop_keys,
      loop_rows,
    }
  })
}

function getTasks(stage: StageTemplate): StageTask[] {
  try {
    const raw = JSON.parse(stage.tasks || '[]') as any[]
    return normalizeTasks(raw)
  } catch {
    return []
  }
}

function getVersionTasks(version: StageTemplateVersion): StageTask[] {
  try {
    const raw = JSON.parse(version.tasks || '[]') as any[]
    return normalizeTasks(raw)
  } catch {
    return []
  }
}

function formatTime(t: string): string {
  if (!t) return ''
  return new Date(t).toLocaleString()
}

async function loadData() {
  loading.value = true
  try {
    const [stagesData, hooksData, templatesData] = await Promise.all([
      getStageTemplatesApi(),
      getHookTemplatesApi().catch(() => [] as HookTemplate[]),
      getWorkflowTemplatesApi().catch(() => [] as WorkflowTemplate[]),
    ])
    stages.value = stagesData
    hookTemplates.value = hooksData
    workflowTemplates.value = templatesData
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

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

function getMachineGroupName(id: number): string {
  const g = machineGroups.value.find((g) => g.id === id)
  return g?.name || ''
}

// ==================== 全屏视图控制 ====================

function exitFullscreen() {
  viewMode.value = 'list'
  editingId.value = 0
  currentStage.value = null
  versionStage.value = null
  selectedVersion.value = null
  editMode.value = 'form'
  yamlContent.value = ''
}

// 创建阶段：直接进入全屏编辑
function showCreateEditor() {
  editingId.value = 0
  currentStage.value = null
  formData.value = { name: '', description: '', machine_group_id: 0, tasks: [] }
  editMode.value = 'form'
  yamlContent.value = ''
  viewMode.value = 'edit'
  loadMachineGroups()
}

// 编辑阶段：进入全屏编辑
function editStage(stage: StageTemplate) {
  editingId.value = stage.id
  currentStage.value = stage
  formData.value = {
    name: stage.name,
    description: stage.description,
    machine_group_id: stage.machine_group_id,
    tasks: getTasks(stage),
  }
  editMode.value = 'form'
  yamlContent.value = ''
  viewMode.value = 'edit'
  loadMachineGroups()
}

// ==================== 任务管理 ====================

function addTask() {
  const maxRef = formData.value.tasks.reduce((max: number, t: StageTask) => Math.max(max, t.ref), 0)
  formData.value.tasks.push({
    ref: maxRef + 1,
    name: '',
    module: 'shell',
    params: { command: '' },
    order: formData.value.tasks.length + 1,
    timeout: 0,
    retries: 0,
    delay: 0,
    when: '',
    hooks: '',
    hooks_array: [],
    register: '',
    ignore_errors: false,
    loop: '',
    loop_array: [],
    loop_mode: 'simple',
    loop_keys: [],
    loop_rows: [],
  })
}

function removeTask(index: number) {
  formData.value.tasks.splice(index, 1)
  formData.value.tasks.forEach((t, i) => { t.order = i + 1 })
}

function moveTask(index: number, direction: -1 | 1) {
  const tasks = formData.value.tasks
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= tasks.length) return
  const temp = tasks[index]
  tasks[index] = tasks[newIndex]
  tasks[newIndex] = temp
  tasks.forEach((t, i) => { t.order = i + 1 })
}

function validateTasks(tasks: StageTask[]): boolean {
  if (tasks.length === 0) {
    ElMessage.warning('至少需要一个任务')
    return false
  }
  const refSet = new Set<number>()
  for (const task of tasks) {
    if (!task.ref || task.ref <= 0) {
      ElMessage.warning('任务引用ID必须大于0')
      return false
    }
    if (refSet.has(task.ref)) {
      ElMessage.warning(`任务引用ID ${task.ref} 重复`)
      return false
    }
    refSet.add(task.ref)
    if (!task.name) {
      ElMessage.warning('任务名称不能为空')
      return false
    }
    if (!task.module) {
      ElMessage.warning(`任务「${task.name}」的模块类型不能为空`)
      return false
    }
  }
  return true
}

// ==================== YAML 转换 ====================

function formToYaml() {
  try {
    const groupName = machineGroups.value.find(g => g.id === formData.value.machine_group_id)?.name || ''
    const obj: Record<string, any> = {
      name: formData.value.name,
      machine_group: groupName,
    }
    if (formData.value.description) obj.description = formData.value.description
    obj.tasks = formData.value.tasks.map(t => {
      const task: Record<string, any> = {
        ref: t.ref,
        name: t.name,
        module: t.module,
        order: t.order,
      }
      if (t.timeout) task.timeout = t.timeout
      if (t.retries) task.retries = t.retries
      if (t.delay) task.delay = t.delay
      if (t.params) task.params = t.params
      if (t.when) task.when = t.when
      if (t.hooks) task.hooks = t.hooks
      if (t.loop) task.loop = t.loop
      if (t.register) task.register = t.register
      if (t.ignore_errors) task.ignore_errors = t.ignore_errors
      return task
    })
    yamlContent.value = yaml.dump(obj, { indent: 2, lineWidth: -1 })
  } catch {
    yamlContent.value = '{}'
  }
}

function yamlToForm() {
  try {
    const parsed = yaml.load(yamlContent.value) as Record<string, any>
    if (!parsed || typeof parsed !== 'object') {
      ElMessage.error('YAML 格式错误：应为对象')
      return
    }
    if (parsed.name !== undefined) formData.value.name = String(parsed.name)
    if (parsed.description !== undefined) formData.value.description = String(parsed.description)
    // 支持 machine_group (name) 和 machine_group_id (兼容旧格式)
    if (parsed.machine_group !== undefined) {
      const group = machineGroups.value.find(g => g.name === parsed.machine_group)
      if (group) {
        formData.value.machine_group_id = group.id
      } else {
        ElMessage.warning(`机器分组「${parsed.machine_group}」不存在，请先创建`)
      }
    } else if (parsed.machine_group_id !== undefined) {
      formData.value.machine_group_id = Number(parsed.machine_group_id)
    }
    if (Array.isArray(parsed.tasks)) {
      formData.value.tasks = normalizeTasks(parsed.tasks)
    }
    editMode.value = 'form'
    ElMessage.success('已应用到表单')
  } catch (e: any) {
    ElMessage.error('YAML 解析失败: ' + (e.message || ''))
  }
}

function formatYaml() {
  try {
    const parsed = yaml.load(yamlContent.value)
    yamlContent.value = yaml.dump(parsed, { indent: 2, lineWidth: -1 })
  } catch (e: any) {
    ElMessage.error('YAML 格式化失败: ' + (e.message || ''))
  }
}

// 切换到 YAML 模式时自动转换
watch(editMode, (mode) => {
  if (mode === 'yaml') {
    formToYaml()
  }
})

// 同步 hooks_array <-> hooks, loop_array/loop_rows <-> loop
watch(() => formData.value.tasks, (tasks) => {
  for (const task of tasks) {
    if (task.hooks_array && task.hooks_array.length > 0) {
      task.hooks = JSON.stringify(task.hooks_array)
    } else {
      task.hooks = ''
    }
    if (task.loop_mode === 'object') {
      if (Array.isArray(task.loop_array)) task.loop_array.splice(0)
      else task.loop_array = []
      if (task.loop_rows && task.loop_rows.length > 0) {
        task.loop = JSON.stringify(task.loop_rows)
      } else {
        task.loop = ''
      }
    } else {
      if (Array.isArray(task.loop_keys)) task.loop_keys.splice(0)
      else task.loop_keys = []
      if (Array.isArray(task.loop_rows)) task.loop_rows.splice(0)
      else task.loop_rows = []
      if (task.loop_array && task.loop_array.length > 0) {
        task.loop = JSON.stringify(task.loop_array)
      } else {
        task.loop = ''
      }
    }
  }
}, { deep: true })

// ==================== 保存 ====================

async function handleSave() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  // 如果当前是 YAML 模式，先应用到表单
  if (editMode.value === 'yaml') {
    yamlToForm()
    if (editMode.value === 'yaml') return // 解析失败则中断
  }
  if (!validateTasks(formData.value.tasks)) return

  // 编辑已有阶段：弹出修改描述对话框
  if (editingId.value) {
    saveDialogVisible.value = true
    saveFormData.value.change_note = ''
  } else {
    // 创建新阶段：直接提交
    doCreate()
  }
}

async function doCreate() {
  submitting.value = true
  try {
    await createStageTemplateApi({
      name: formData.value.name,
      description: formData.value.description,
      machine_group_id: formData.value.machine_group_id,
      tasks: JSON.stringify(formData.value.tasks.map(t => ({ ...t, params: JSON.stringify(t.params) }))),
    })
    ElMessage.success('创建成功')
    exitFullscreen()
    loadData()
  } catch (e: any) {
    if (!(e instanceof HandledError)) ElMessage.error(e?.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

async function confirmSave() {
  try {
    await saveFormRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    await updateStageTemplateApi(editingId.value, {
      name: formData.value.name,
      description: formData.value.description,
      machine_group_id: formData.value.machine_group_id,
      tasks: JSON.stringify(formData.value.tasks.map(t => ({ ...t, params: JSON.stringify(t.params) }))),
      change_note: saveFormData.value.change_note,
    })
    ElMessage.success('保存成功，已生成新版本')
    saveDialogVisible.value = false
    exitFullscreen()
    loadData()
  } catch (e: any) {
    if (!(e instanceof HandledError)) ElMessage.error(e?.message || '保存失败')
  } finally {
    submitting.value = false
  }
}

// ==================== 执行 ====================

function showExecuteDialog(stage: StageTemplate) {
  executingStage.value = stage
  executeGroupId.value = 0
  executeDialogVisible.value = true
}

async function handleExecute() {
  if (!executingStage.value) return
  executing.value = true
  try {
    const result = await executeStageTemplateApi(executingStage.value.id, executeGroupId.value || undefined)
    ElMessage.success(`执行已启动，执行ID: ${result.execution_id}`)
    currentExecutionId.value = result.execution_id
    executeDialogVisible.value = false

    // 打开执行日志查看器
    if (executionLogViewer.value) {
      executionLogViewer.value.open(result.execution_id)
    }
  } catch (e: any) {
    if (!(e instanceof HandledError)) ElMessage.error(e?.message || '执行失败')
  } finally {
    executing.value = false
  }
}

// ==================== 删除 ====================

async function deleteStage(stage: StageTemplate) {
  try {
    await ElMessageBox.confirm(
      `确定要删除阶段「${stage.name}」吗？所有版本历史将一并删除。`,
      '删除确认',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteStageTemplateApi(stage.id)
    ElMessage.success('删除成功')
    loadData()
  } catch {
    // 用户取消
  }
}

// ==================== 版本管理 ====================

async function showVersionHistory(stage: StageTemplate) {
  versionStage.value = stage
  selectedVersion.value = null
  viewMode.value = 'versions'
  versionLoading.value = true
  try {
    versionList.value = await listStageTemplateVersionsApi(stage.id)
  } catch (e) {
    console.error(e)
    versionList.value = []
  } finally {
    versionLoading.value = false
  }
}

function selectVersion(version: StageTemplateVersion) {
  selectedVersion.value = version
}

async function handleUpdateToVersion(version: StageTemplateVersion) {
  try {
    await ElMessageBox.confirm(
      `确定要更新到版本 ${version.version} 吗？`,
      '更新确认',
      { confirmButtonText: '确定更新', cancelButtonText: '取消', type: 'warning' }
    )
    await rollbackStageTemplateApi(versionStage.value!.id, version.version)
    ElMessage.success(`已更新到 ${version.version}`)
    exitFullscreen()
    loadData()
  } catch {
    // 用户取消
  }
}

onMounted(() => {
  loadData()
  loadMachineGroups()
})
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

.stage-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.stage-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 16px;
  background: var(--el-bg-color);
  transition: box-shadow 0.2s;
}

.stage-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.stage-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.stage-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color-primary);
}

.version-tag {
  font-size: 11px;
  font-weight: 600;
}

.stage-card-actions {
  display: flex;
  gap: 4px;
}

.stage-card-desc {
  font-size: 13px;
  color: var(--text-color-secondary);
  margin-bottom: 12px;
}

.stage-card-info {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.stage-card-tasks {
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 12px;
}

.task-mini {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  font-size: 13px;
}

.task-mini-index {
  width: 18px;
  height: 18px;
  border-radius: 4px;
  background: var(--el-fill-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}

.task-mini-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-mini-module {
  flex-shrink: 0;
}

.task-more {
  text-align: center;
  font-size: 12px;
  color: var(--text-color-secondary);
  padding-top: 4px;
}

.stage-empty {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 0;
  color: var(--text-color-secondary);
}

/* 全屏视图样式 */
.fullscreen-view {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--el-bg-color);
  z-index: 1000;
  display: flex;
  flex-direction: column;
}

.fullscreen-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.fullscreen-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.fullscreen-header-right {
  display: flex;
  gap: 8px;
}

.fullscreen-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color-primary);
}

.fullscreen-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px;
}

.form-section {
  margin-bottom: 16px;
}

.form-section-compact .el-form-item {
  margin-bottom: 12px;
}

.form-section-compact .form-item-compact {
  margin-bottom: 8px;
}

.form-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color-primary);
  margin-bottom: 12px;
}

.form-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.form-section-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.form-section-title-row .form-section-title {
  margin-bottom: 0;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: calc(100vh - 300px);
  overflow-y: auto;
  padding-bottom: 16px;
}

.task-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 16px;
  background: var(--el-bg-color);
}

.task-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.task-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  padding-left: 90px;
}

.task-card-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.task-card-actions :deep(.el-button) {
  font-size: 13px;
  font-weight: 500;
}

.task-index {
  width: 24px;
  height: 24px;
  border-radius: 4px;
  background: var(--el-color-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}

.task-name-input {
  flex: 1;
}

.task-card-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.task-field {
  margin-bottom: 8px;
}

.task-field :deep(.el-form-item__label) {
  font-size: 12px;
  color: var(--text-color-secondary);
  font-weight: 500;
  line-height: 28px;
  padding-right: 8px;
}

.task-field :deep(.el-form-item) {
  margin-bottom: 0;
}

.empty-tip {
  text-align: center;
  color: var(--text-color-secondary);
  font-size: 13px;
  padding: 40px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
}

.empty-group-hint {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: var(--el-fill-color-lighter);
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

/* 模块选项样式 */
.module-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 2px 0;
}

.module-option-name {
  font-weight: 500;
  min-width: 80px;
}

.module-option-desc {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.group-nav-link {
  color: var(--el-color-primary);
  text-decoration: none;
  font-weight: 500;
}
.group-nav-link:hover {
  text-decoration: underline;
}

/* YAML 编辑器样式 */
.yaml-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.yaml-editor-wrapper {
  height: calc(100vh - 300px);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  overflow: hidden;
}

.yaml-editor-wrapper :deep(.cm-editor) {
  height: 100%;
}

.yaml-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

/* 版本历史视图样式 */
.version-layout {
  display: flex;
  gap: 24px;
  padding: 24px;
}

.version-sidebar {
  width: 320px;
  flex-shrink: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}

.version-list {
  max-height: calc(100vh - 120px);
  overflow-y: auto;
}

.version-item {
  padding: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  cursor: pointer;
  transition: background 0.2s;
}

.version-item:hover {
  background: var(--el-fill-color-light);
}

.version-item.is-current {
  background: var(--el-color-primary-light-9);
}

.version-item.is-selected {
  background: var(--el-color-primary-light-7);
}

.version-item-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.current-badge {
  font-size: 12px;
  color: var(--el-color-primary);
  font-weight: 600;
}

.version-note {
  font-size: 13px;
  color: var(--text-color-secondary);
  margin: 0 0 4px 0;
}

.version-time {
  font-size: 12px;
  color: var(--text-color-placeholder);
}

.version-detail {
  flex: 1;
  min-width: 0;
}

.version-detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.version-detail-header h3 {
  margin: 0;
  font-size: 18px;
  color: var(--text-color-primary);
}

.version-detail-content {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 24px;
}

.detail-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color-primary);
  margin: 24px 0 12px 0;
}

.version-task-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.version-task-item {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 12px;
}

.version-task-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.version-task-name {
  flex: 1;
  font-weight: 500;
}

.version-task-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}

.version-task-params,
.version-task-when {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.params-label {
  color: var(--text-color-secondary);
  flex-shrink: 0;
}

.version-task-params code,
.version-task-when code {
  background: var(--el-fill-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  word-break: break-all;
}

.version-detail-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-color-secondary);
}

/* 保存对话框样式 */
.save-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.save-version-info {
  background: var(--el-fill-color-light);
  border-radius: 8px;
  padding: 16px;
}

.save-version-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0;
}

.save-label {
  font-size: 14px;
  color: var(--text-color-secondary);
  min-width: 80px;
}

/* 执行参数 key-value 表单 */
.params-section {
  margin-bottom: 8px;
}

.params-kv-list {
  width: 100%;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  overflow: hidden;
}

.params-kv-row {
  display: flex;
  align-items: center;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.params-kv-row:last-child {
  border-bottom: none;
}

.params-kv-key {
  min-width: 120px;
  max-width: 160px;
  padding: 8px 12px;
  background: var(--el-fill-color-lighter);
  font-family: monospace;
  font-size: 13px;
  color: var(--el-color-primary);
  font-weight: 500;
  border-right: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}

.params-kv-value {
  flex: 1;
}

.params-kv-value :deep(.el-input__wrapper) {
  box-shadow: none !important;
  border-radius: 0;
}

.params-kv-value :deep(.el-input__inner) {
  font-family: monospace;
  font-size: 13px;
}

.params-kv-actions {
  flex-shrink: 0;
  padding: 2px 4px;
  display: flex;
  align-items: center;
}

.params-empty {
  padding: 16px;
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

/* 执行条件结构化编辑 */
.when-builder {
  width: 100%;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
}

.when-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.when-left {
  flex: 1;
}

.when-left :deep(.el-input__wrapper) {
  box-shadow: none !important;
  border-radius: 4px;
  background: var(--el-bg-color);
}

.when-left :deep(.el-input__inner) {
  font-family: monospace;
  font-size: 13px;
}

.when-operator {
  width: 100px;
  flex-shrink: 0;
}

.when-right {
  flex: 1;
}

.when-right :deep(.el-input__wrapper) {
  box-shadow: none !important;
  border-radius: 4px;
  background: var(--el-bg-color);
}

.when-right :deep(.el-input__inner) {
  font-family: monospace;
  font-size: 13px;
}

.loop-input-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  width: 100%;
}

.loop-tag {
  margin: 0;
}

.loop-input {
  width: 180px;
}

.loop-section {
  width: 100%;
}

.loop-mode-switch {
  margin-bottom: 8px;
}

.loop-object-section {
  width: 100%;
}

.loop-keys-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.loop-keys-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.loop-keys-input {
  flex: 1;
}

.loop-table-wrapper {
  width: 100%;
  overflow-x: auto;
}

.loop-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
  margin-bottom: 8px;
}

.loop-table th,
.loop-table td {
  border: 1px solid var(--el-border-color-lighter);
  padding: 4px 6px;
  text-align: left;
}

.loop-table th {
  background: var(--el-fill-color-lighter);
  font-weight: 500;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.loop-table-action {
  width: 40px;
  text-align: center !important;
}

.loop-var-btn {
  margin-left: 4px;
}

.reopen-log-btn {
  position: fixed;
  bottom: 24px;
  right: 24px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: var(--el-color-primary);
  color: #fff;
  border-radius: 8px;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.2s;
  z-index: 100;
}

.reopen-log-btn:hover {
  background: var(--el-color-primary-dark-2);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
}
</style>
