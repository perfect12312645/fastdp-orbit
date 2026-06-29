package workflow

import (
	"time"

	"gorm.io/gorm"
)

// StageTemplate 阶段模板（独立管理，供工作流画布拖拽使用）
type StageTemplate struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
	Name           string         `json:"name" gorm:"size:100;not null"`
	Description    string         `json:"description" gorm:"size:500"`
	MachineGroupID uint           `json:"machine_group_id" gorm:"index"`
	Tasks          string         `json:"tasks" gorm:"type:text"` // JSON array of StageTask
	Version        string         `json:"version" gorm:"size:20;not null;default:'init'"`
	Source         string         `json:"source" gorm:"size:100;index;default:''"` // 来源（模板包名）
}

// StageTask 存储在 StageTemplate.Tasks JSON 中的任务结构
type StageTask struct {
	Ref          int    `json:"ref"`
	Name         string `json:"name"`
	Module       string `json:"module"`
	Params       string `json:"params"`
	Order        int    `json:"order"`
	When         string `json:"when"`
	HookIDs      string `json:"hook_ids"`
	Loop         string `json:"loop"`
	Timeout      int    `json:"timeout"`
	IgnoreErrors bool   `json:"ignore_errors"`
	Retries      int    `json:"retries"`
	Delay        int    `json:"delay"`
	Register     string `json:"register"`
}

func (StageTemplate) TableName() string { return "stage_templates" }
