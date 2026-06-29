package workflow

import (
	"time"

	"gorm.io/gorm"
)

// SolutionLibrary 方案库（模板市场的基本单位，包含多个模块的组合）
type SolutionLibrary struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"size:100;not null;uniqueIndex"`
	Description string `json:"description" gorm:"size:500"`
	Category    string `json:"category" gorm:"size:50;index"` // k8s, database, monitoring, etc.
	Version     string `json:"version" gorm:"size:20"`
	Author      string `json:"author" gorm:"size:100"`
	Icon        string `json:"icon" gorm:"size:50"` // mdi icon name
	// 统计字段
	StageCount    int            `json:"stage_count"`
	VariableCount int            `json:"variable_count"`
	HookCount     int            `json:"hook_count"`
	TemplateCount int            `json:"template_count"`
	MaterialCount int            `json:"material_count"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (SolutionLibrary) TableName() string { return "solution_library" }

// OrbitPack orbit-pack YAML 格式定义
type OrbitPack struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Category    string `yaml:"category"`
		Version     string `yaml:"version"`
		Author      string `yaml:"author"`
	} `yaml:"metadata"`
	Materials         []PackMaterial         `yaml:"materials,omitempty"`
	GlobalVariables   []PackGlobalVariable   `yaml:"global_variables,omitempty"`
	Hooks             []PackHook             `yaml:"hooks,omitempty"`
	WorkflowTemplates []PackWorkflowTemplate `yaml:"workflow_templates,omitempty"`
	Stages            []PackStage            `yaml:"stages,omitempty"`
}

// PackMaterial 物料清单
type PackMaterial struct {
	Name        string `yaml:"name"`
	Size        int64  `yaml:"size"`
	MD5         string `yaml:"md5"`
	DownloadURL string `yaml:"download_url,omitempty"`
}

// PackGlobalVariable 打包的全局变量
type PackGlobalVariable struct {
	Key         string `yaml:"key"`
	Type        string `yaml:"type"`
	Value       string `yaml:"value"`
	Description string `yaml:"description"`
	Group       string `yaml:"group"`
}

// PackHook 打包的钩子模板
type PackHook struct {
	Name         string `yaml:"name"`
	Description  string `yaml:"description,omitempty"`
	Module       string `yaml:"module"`
	Params       string `yaml:"params,omitempty"`
	Timeout      int    `yaml:"timeout,omitempty"`
	IgnoreErrors bool   `yaml:"ignore_errors,omitempty"`
	Retries      int    `yaml:"retries,omitempty"`
	Delay        int    `yaml:"delay,omitempty"`
}

// PackWorkflowTemplate 打包的工作流模板文件
type PackWorkflowTemplate struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Content     string `yaml:"content"`
}

// PackStage 打包的阶段模板
type PackStage struct {
	Name         string     `yaml:"name"`
	Description  string     `yaml:"description,omitempty"`
	MachineGroup string     `yaml:"machine_group,omitempty"` // 使用名称而非ID
	Tasks        []PackTask `yaml:"tasks"`
}

// PackTask 打包的任务
type PackTask struct {
	Ref          int    `yaml:"ref"`
	Name         string `yaml:"name"`
	Module       string `yaml:"module"`
	Order        int    `yaml:"order"`
	Params       string `yaml:"params,omitempty"`
	When         string `yaml:"when,omitempty"`
	HookIDs      string `yaml:"hook_ids,omitempty"`
	Loop         string `yaml:"loop,omitempty"`
	Timeout      int    `yaml:"timeout,omitempty"`
	IgnoreErrors bool   `yaml:"ignore_errors,omitempty"`
	Retries      int    `yaml:"retries,omitempty"`
	Delay        int    `yaml:"delay,omitempty"`
	Register     string `yaml:"register,omitempty"`
}
