package workflow

import (
	"time"

	"gorm.io/gorm"
)

// StageExecution 单阶段执行记录（独立于工作流执行）
type StageExecution struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	StageTemplateID uint           `json:"stage_template_id" gorm:"index;not null"`
	StageName       string         `json:"stage_name" gorm:"size:200;not null"`
	MachineGroupID  uint           `json:"machine_group_id" gorm:"not null"`
	MachineGroupName string        `json:"machine_group_name" gorm:"size:100"`
	Status          string         `json:"status" gorm:"size:20;default:pending"` // pending/running/success/failed/cancelled
	Error           string         `json:"error" gorm:"type:text"`
	Trigger         string         `json:"trigger" gorm:"size:20;default:manual"` // manual
	StartedAt       *time.Time     `json:"started_at"`
	FinishedAt      *time.Time     `json:"finished_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	TaskExecutions []StageTaskExecution `json:"task_executions" gorm:"foreignKey:StageExecutionID"`
}

func (StageExecution) TableName() string { return "stage_executions" }

// StageTaskExecution 单阶段任务执行记录
type StageTaskExecution struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	StageExecutionID uint       `json:"stage_execution_id" gorm:"index;not null"`
	TaskRef          int        `json:"task_ref" gorm:"not null"`
	TaskName         string     `json:"task_name" gorm:"size:200"`
	TaskModule       string     `json:"task_module" gorm:"size:50"`
	Host             string     `json:"host" gorm:"size:100;not null"`
	Status           string     `json:"status" gorm:"size:20;default:pending"`
	Output           string     `json:"output" gorm:"type:text"`
	Error            string     `json:"error" gorm:"type:text"`
	ErrorCode        int32      `json:"error_code"`
	Trace            string     `json:"trace" gorm:"type:text"`
	Changed          bool       `json:"changed"`
	HookStatus       string     `json:"hook_status" gorm:"size:20;default:none"`
	HookError        string     `json:"hook_error" gorm:"type:text"`
	DurationMs       int64      `json:"duration_ms"`
	StartedAt        *time.Time `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (StageTaskExecution) TableName() string { return "stage_task_executions" }
