package workflow

import (
	"time"

	"fastdp-orbit/backend/models/machine"

	"gorm.io/gorm"
)

// ==================== 定义层 ====================

// Workflow 工作流定义
type Workflow struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`
	Description string         `json:"description" gorm:"size:500"`
	CreatedBy   string         `json:"created_by" gorm:"size:50"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	StageGroups []WorkflowStageGroup `json:"stage_groups" gorm:"foreignKey:WorkflowID"`
	Hooks       []WorkflowHook       `json:"hooks" gorm:"foreignKey:WorkflowID"`
}

// WorkflowStageGroup 阶段组（画布中的列，按顺序从左到右执行）
type WorkflowStageGroup struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	WorkflowID  uint   `json:"workflow_id" gorm:"index;not null"`
	Name        string `json:"name" gorm:"size:100;not null"`
	Description string `json:"description" gorm:"size:500"`
	Order       int    `json:"order" gorm:"not null"` // 列顺序，从左到右

	// 组内执行模式
	Mode string `json:"mode" gorm:"size:20;default:sequential"` // sequential/parallel（组内阶段是顺序执行还是并行执行）

	// 关联
	Stages []WorkflowStage `json:"stages" gorm:"foreignKey:StageGroupID"`
}

// WorkflowStage 阶段（画布节点，内部包含多个任务）
type WorkflowStage struct {
	ID               uint   `json:"id" gorm:"primaryKey"`
	StageGroupID     uint   `json:"stage_group_id" gorm:"index;not null"`
	Name             string `json:"name" gorm:"size:100;not null"`
	Description      string `json:"description" gorm:"size:500"`
	Order            int    `json:"order" gorm:"not null"` // 组内顺序，从上到下
	MachineGroupID   uint   `json:"machine_group_id" gorm:"index"` // 关联机器分组，该阶段内所有任务批量执行的目标机器
	TemplateVersion  string `json:"template_version" gorm:"size:20"` // 来源阶段模板版本号

	// 关联（运行时加载，不持久化）
	MachineGroup *machine.MachineGroup `json:"machine_group,omitempty" gorm:"-"`
	Tasks        []WorkflowTask        `json:"tasks" gorm:"foreignKey:StageID"`
}

// WorkflowTask 任务（阶段内的单个操作，对应一个 Agent 模块调用）
type WorkflowTask struct {
	ID      uint `json:"id" gorm:"primaryKey"`
	StageID uint `json:"stage_id" gorm:"index;not null"`

	Ref    int    `json:"ref" gorm:"not null"`                     // 工作流内唯一引用ID（用于YAML引用和错误定位）
	Name   string `json:"name" gorm:"size:200;not null"`
	Module string `json:"module" gorm:"size:50;not null;default:shell"` // 模块类型: shell/systemd/file/template/package/repo/blockinfile/modprobe 等
	Params string `json:"params" gorm:"type:text"`                     // JSON: 模块参数（对应 YAML 中的 params）
	Order  int    `json:"order" gorm:"not null"`

	// 条件执行（Go 模板表达式，如 "{{.machine.os_name}} == ubuntu"）
	When string `json:"when" gorm:"size:500"`
	// 后置钩子（引用 WorkflowHook 的 ID 列表，JSON 数组如 [1,3,5]）
	HookIDs string `json:"hook_ids" gorm:"type:text"`
	// 循环执行（JSON 数组，如 '["item1","item2"]'），使用 {{.item}} 引用当前项
	Loop string `json:"loop" gorm:"type:text"`
	// 超时（秒），0表示不超时
	Timeout int `json:"timeout" gorm:"default:0"`

	// 失败时跳过，继续执行后续任务
	IgnoreErrors bool `json:"ignore_errors" gorm:"default:false"`
	// 重试次数，0表示不重试
	Retries int `json:"retries" gorm:"default:0"`
	// 重试间隔（秒）
	Delay int `json:"delay" gorm:"default:0"`
	// 将任务输出注册为变量名，供后续任务的 when 或 params 中引用
	Register string `json:"register" gorm:"size:100"`
}

// WorkflowHook 工作流钩子快照（保存时从 HookTemplate 复制过来）
type WorkflowHook struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	WorkflowID uint   `json:"workflow_id" gorm:"index;not null"`
	Name       string `json:"name" gorm:"size:100;not null"`
	Module     string `json:"module" gorm:"size:50;not null"`
	Params     string `json:"params" gorm:"type:text"`
	Timeout    int    `json:"timeout" gorm:"default:0"`

	// 失败时跳过，继续执行后续任务
	IgnoreErrors bool `json:"ignore_errors" gorm:"default:false"`
	// 重试次数，0表示不重试
	Retries int `json:"retries" gorm:"default:0"`
	// 重试间隔（秒）
	Delay int `json:"delay" gorm:"default:0"`
}

// ==================== 执行层 ====================

// WorkflowExecution 一次工作流执行实例
type WorkflowExecution struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	WorkflowID uint           `json:"workflow_id" gorm:"index;not null"`
	Workflow   *Workflow      `json:"workflow,omitempty" gorm:"foreignKey:WorkflowID"`
	Status     string         `json:"status" gorm:"size:20;default:running"` // running/success/failed/paused/cancelled
	Trigger    string         `json:"trigger" gorm:"size:50"`                // user/system
	Error      string         `json:"error" gorm:"type:text"`                // 失败原因
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	StageGroupExecutions []WorkflowStageGroupExecution `json:"stage_group_executions" gorm:"foreignKey:ExecutionID"`
}

// WorkflowStageGroupExecution 阶段组执行记录
type WorkflowStageGroupExecution struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	ExecutionID uint           `json:"execution_id" gorm:"index;not null"`
	GroupID     uint           `json:"group_id" gorm:"index;not null"`
	Group       *WorkflowStageGroup `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	Status      string         `json:"status" gorm:"size:20;default:pending"` // pending/running/success/failed/skipped
	Error       string         `json:"error" gorm:"type:text"`
	StartedAt   *time.Time     `json:"started_at"`
	FinishedAt  *time.Time     `json:"finished_at"`

	// 关联
	StageExecutions []WorkflowStageExecution `json:"stage_executions" gorm:"foreignKey:StageGroupExecutionID"`
}

// WorkflowStageExecution 阶段执行记录
type WorkflowStageExecution struct {
	ID                    uint           `json:"id" gorm:"primaryKey"`
	StageGroupExecutionID uint           `json:"stage_group_execution_id" gorm:"index;not null"`
	StageID               uint           `json:"stage_id" gorm:"index;not null"`
	Stage                 *WorkflowStage `json:"stage,omitempty" gorm:"foreignKey:StageID"`
	Status                string         `json:"status" gorm:"size:20;default:pending"` // pending/running/success/failed/skipped
	Error                 string         `json:"error" gorm:"type:text"`
	StartedAt             *time.Time     `json:"started_at"`
	FinishedAt            *time.Time     `json:"finished_at"`

	// 关联
	TaskExecutions []WorkflowTaskExecution `json:"task_executions" gorm:"foreignKey:StageExecutionID"`
}

// WorkflowTaskExecution 任务执行记录
type WorkflowTaskExecution struct {
	ID                uint                   `json:"id" gorm:"primaryKey"`
	StageExecutionID  uint                   `json:"stage_execution_id" gorm:"index;not null"`
	TaskID            uint                   `json:"task_id" gorm:"index;not null"`
	Task              *WorkflowTask          `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Host              string                 `json:"host" gorm:"size:100"`                       // 目标机器 ip:port
	Status            string                 `json:"status" gorm:"size:20;default:pending"`       // pending/running/success/failed/skipped
	Output            string                 `json:"output" gorm:"type:text"`                     // 标准输出
	Stderr            string                 `json:"stderr" gorm:"type:text"`                     // 标准错误（非致命警告）
	Error             string                 `json:"error" gorm:"type:text"`                      // 错误信息
	ErrorCode         int32                  `json:"error_code"`                                  // 错误码
	Changed           bool                   `json:"changed"`                                     // 是否产生变更
	HookStatus        string                 `json:"hook_status" gorm:"size:20;default:none"`     // 钩子执行状态: none/running/success/failed
	HookError         string                 `json:"hook_error" gorm:"type:text"`                 // 钩子失败原因
	DurationMs        int64                  `json:"duration_ms"`                                 // 执行耗时（毫秒）
	StartedAt         *time.Time             `json:"started_at"`
	FinishedAt        *time.Time             `json:"finished_at"`
}

// TableName 指定表名
func (Workflow) TableName() string                    { return "workflows" }
func (WorkflowStageGroup) TableName() string          { return "workflow_stage_groups" }
func (WorkflowStage) TableName() string               { return "workflow_stages" }
func (WorkflowTask) TableName() string                { return "workflow_tasks" }
func (WorkflowHook) TableName() string                { return "workflow_hooks" }
func (WorkflowExecution) TableName() string           { return "workflow_executions" }
func (WorkflowStageGroupExecution) TableName() string { return "workflow_stage_group_executions" }
func (WorkflowStageExecution) TableName() string      { return "workflow_stage_executions" }
func (WorkflowTaskExecution) TableName() string       { return "workflow_task_executions" }
