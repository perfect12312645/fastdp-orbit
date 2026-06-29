package workflow

import "time"

// GlobalVariable 全局变量（独立管理，可被多个工作流引用）
type GlobalVariable struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Key         string    `json:"key" gorm:"size:100;not null"`
	Type        string    `json:"type" gorm:"size:20;not null"`            // string/number/bool
	Value       string    `json:"value" gorm:"type:text"`                  // 变量值（默认值）
	Description string    `json:"description" gorm:"size:500"`             // 变量描述
	Group       string    `json:"group" gorm:"size:100"`                   // 变量分组，如 "网络配置"、"系统设置"
	Source      string    `json:"source" gorm:"size:100;index;default:''"` // 来源（模板包名）
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (GlobalVariable) TableName() string { return "global_variables" }
