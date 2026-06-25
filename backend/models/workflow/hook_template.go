package workflow

import (
	"time"

	"gorm.io/gorm"
)

// HookTemplate 可复用的钩子模板（全局管理，可被多个工作流的任务引用）
type HookTemplate struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name" gorm:"size:100;not null;uniqueIndex"`
	Description  string         `json:"description" gorm:"size:500"`
	Module       string         `json:"module" gorm:"size:50;not null;default:shell"`
	Params       string         `json:"params" gorm:"type:text"`
	Timeout      int            `json:"timeout" gorm:"default:0"`
	IgnoreErrors bool           `json:"ignore_errors" gorm:"default:false"`
	Retries      int            `json:"retries" gorm:"default:0"`
	Delay        int            `json:"delay" gorm:"default:0"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (HookTemplate) TableName() string { return "hook_templates" }
