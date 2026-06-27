package workflow

import (
	"time"

	"gorm.io/gorm"
)

// WorkflowTemplate 工作流模板文件（可复用的配置模板，供 Stage Template 编排时选取）
type WorkflowTemplate struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`
	Description string         `json:"description" gorm:"size:500"`
	Content     string         `json:"content" gorm:"type:text"`   // Go template 语法内容
	Variables   string         `json:"variables" gorm:"type:text"` // 变量说明文档（JSON格式）
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (WorkflowTemplate) TableName() string { return "workflow_templates" }
