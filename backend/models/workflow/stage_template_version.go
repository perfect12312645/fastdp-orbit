package workflow

import (
	"time"

	"gorm.io/gorm"
)

// StageTemplateVersion 阶段模板版本历史（每次修改保存都生成新版本）
type StageTemplateVersion struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	TemplateID     uint           `json:"template_id" gorm:"uniqueIndex:uk_template_version;not null"`
	Version        string         `json:"version" gorm:"size:20;uniqueIndex:uk_template_version;not null"`
	Name           string         `json:"name" gorm:"size:100;not null"`
	Description    string         `json:"description" gorm:"size:500"`
	MachineGroupID uint           `json:"machine_group_id" gorm:"index"`
	Tasks          string         `json:"tasks" gorm:"type:text"`
	ChangeNote     string         `json:"change_note" gorm:"size:500"`
	CreatedAt      time.Time      `json:"created_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (StageTemplateVersion) TableName() string { return "stage_template_versions" }
