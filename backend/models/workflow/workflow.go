package workflow

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 定义层 ====================

// Workflow 工作流定义
type Workflow struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`
	Description string         `json:"description" gorm:"size:500"`
	Config      string         `json:"config" gorm:"type:text"` // JSON: 全局变量等
	CreatedBy   string         `json:"created_by" gorm:"size:50"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Stages []WorkflowStage `json:"stages" gorm:"foreignKey:WorkflowID"`
}

// WorkflowStage 阶段（画布节点，内部包含多个任务）
type WorkflowStage struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	WorkflowID  uint   `json:"workflow_id" gorm:"index;not null"`
	Name        string `json:"name" gorm:"size:100;not null"`
	Description string `json:"description" gorm:"size:500"`
	Order       int    `json:"order" gorm:"not null"`

	// 阶段配置
	RetryPolicy     string `json:"retry_policy" gorm:"size:20;default:none"` // none/always/on_failure
	MaxRetries      int    `json:"max_retries" gorm:"default:3"`
	ContinueOnError bool   `json:"continue_on_error" gorm:"default:false"` // 失败后是否继续下一阶段

	// 关联
	Tasks []WorkflowTask `json:"tasks" gorm:"foreignKey:StageID"`
}

// WorkflowTask 任务（阶段内的单个操作，对应一个 Agent 模块调用）
type WorkflowTask struct {
	ID      uint `json:"id" gorm:"primaryKey"`
	StageID uint `json:"stage_id" gorm:"index;not null"`

	Name   string `json:"name" gorm:"size:200;not null"`
	Module string `json:"module" gorm:"size:50;not null;default:shell"` // 模块类型: shell/systemd/file/template/package/repo/blockinfile/modprobe 等
	Params string `json:"params" gorm:"type:text"`                     // JSON: 模块参数（对应 YAML 中的 params）
	Host   string `json:"host" gorm:"size:100;not null"`               // 目标机器 ip:port
	Order  int    `json:"order" gorm:"not null"`

	// 条件执行（Go 模板表达式，如 "{{.Machine.OSName}} !contains ubuntu"）
	When string `json:"when" gorm:"size:500"`
	// 后置钩子（逗号分隔，如 "restart_NetworkManager,restart_chronyd"）
	Hooks string `json:"hooks" gorm:"size:500"`
	// 循环执行（JSON 数组，如 '["item1","item2"]'），使用 {{.item}} 引用当前项
	Loop string `json:"loop" gorm:"type:text"`
	// 超时（秒），0表示不超时
	Timeout int `json:"timeout" gorm:"default:0"`
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
	StageExecutions []WorkflowStageExecution `json:"stage_executions" gorm:"foreignKey:ExecutionID"`
}

// WorkflowStageExecution 阶段执行记录
type WorkflowStageExecution struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	ExecutionID uint           `json:"execution_id" gorm:"index;not null"`
	StageID     uint           `json:"stage_id" gorm:"index;not null"`
	Stage       *WorkflowStage `json:"stage,omitempty" gorm:"foreignKey:StageID"`
	Status      string         `json:"status" gorm:"size:20;default:pending"` // pending/running/success/failed/skipped
	Error       string         `json:"error" gorm:"type:text"`
	StartedAt   *time.Time     `json:"started_at"`
	FinishedAt  *time.Time     `json:"finished_at"`

	// 关联
	TaskExecutions []WorkflowTaskExecution `json:"task_executions" gorm:"foreignKey:StageExecutionID"`
}

// WorkflowTaskExecution 任务执行记录
type WorkflowTaskExecution struct {
	ID                uint                   `json:"id" gorm:"primaryKey"`
	StageExecutionID  uint                   `json:"stage_execution_id" gorm:"index;not null"`
	TaskID            uint                   `json:"task_id" gorm:"index;not null"`
	Task              *WorkflowTask          `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Status            string                 `json:"status" gorm:"size:20;default:pending"` // pending/running/success/failed
	Output            string                 `json:"output" gorm:"type:text"`                // 标准输出
	Error             string                 `json:"error" gorm:"type:text"`                 // 错误信息
	DurationMs        int64                  `json:"duration_ms"`                            // 执行耗时（毫秒）
	StartedAt         *time.Time             `json:"started_at"`
	FinishedAt        *time.Time             `json:"finished_at"`
}

// TableName 指定表名
func (Workflow) TableName() string            { return "workflows" }
func (WorkflowStage) TableName() string        { return "workflow_stages" }
func (WorkflowTask) TableName() string         { return "workflow_tasks" }
func (WorkflowExecution) TableName() string    { return "workflow_executions" }
func (WorkflowStageExecution) TableName() string { return "workflow_stage_executions" }
func (WorkflowTaskExecution) TableName() string  { return "workflow_task_executions" }
