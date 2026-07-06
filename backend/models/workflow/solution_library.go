package workflow

import (
	"time"
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
	// 原始数据（导入时存储 OrbitPack JSON，应用后清空）
	PackData string `json:"pack_data" gorm:"type:text"` // OrbitPack JSON
	// 关联ID（JSON数组格式存储，应用后才有值）
	StageIDs    string `json:"stage_ids" gorm:"type:text"`    // [1,2,3]
	VariableIDs string `json:"variable_ids" gorm:"type:text"` // [1,2]
	HookIDs     string `json:"hook_ids" gorm:"type:text"`     // [1]
	TemplateIDs string `json:"template_ids" gorm:"type:text"` // [1,2]
	FileIDs     string `json:"file_ids" gorm:"type:text"`     // [1]
	WorkflowIDs string `json:"workflow_ids" gorm:"type:text"` // [1]
	// 统计字段
	StageCount    int       `json:"stage_count"`
	VariableCount int       `json:"variable_count"`
	HookCount     int       `json:"hook_count"`
	TemplateCount int       `json:"template_count"`
	FileCount     int       `json:"file_count"`
	WorkflowCount int       `json:"workflow_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (SolutionLibrary) TableName() string { return "solution_library" }

// OrbitPack orbit-pack YAML 格式定义
type OrbitPack struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
	Metadata   struct {
		Name        string `yaml:"name" json:"name"`
		Description string `yaml:"description" json:"description"`
		Category    string `yaml:"category" json:"category"`
		Version     string `yaml:"version" json:"version"`
		Author      string `yaml:"author" json:"author"`
	} `yaml:"metadata" json:"metadata"`
	Materials         []PackMaterial         `yaml:"materials,omitempty" json:"materials,omitempty"`
	GlobalVariables   []PackGlobalVariable   `yaml:"global_variables,omitempty" json:"globalVariables,omitempty"`
	Hooks             []PackHook             `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	WorkflowTemplates []PackWorkflowTemplate `yaml:"workflow_templates,omitempty" json:"workflowTemplates,omitempty"`
	MachineGroups     []PackMachineGroup      `yaml:"machine_groups,omitempty" json:"machineGroups,omitempty"`
	Stages            []PackStage            `yaml:"stages,omitempty" json:"stages,omitempty"`
	Workflows         []PackWorkflow         `yaml:"workflows,omitempty" json:"workflows,omitempty"`
}

// PackMachineGroup 打包的机器分组
type PackMachineGroup struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// PackMaterial 物料清单
type PackMaterial struct {
	Name        string `yaml:"name" json:"name"`
	Size        int64  `yaml:"size" json:"size"`
	MD5         string `yaml:"md5" json:"md5"`
	DownloadURL string `yaml:"download_url,omitempty" json:"downloadUrl,omitempty"`
}

// PackGlobalVariable 打包的全局变量
type PackGlobalVariable struct {
	Key         string `yaml:"key" json:"key"`
	Type        string `yaml:"type" json:"type"`
	Value       string `yaml:"value" json:"value"`
	Description string `yaml:"description" json:"description"`
	Group       string `yaml:"group" json:"group"`
}

// PackHook 打包的钩子模板
type PackHook struct {
	Name         string `yaml:"name" json:"name"`
	Description  string `yaml:"description,omitempty" json:"description,omitempty"`
	Module       string `yaml:"module" json:"module"`
	Params       string `yaml:"params,omitempty" json:"params,omitempty"`
	Timeout      int    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	IgnoreErrors bool   `yaml:"ignore_errors,omitempty" json:"ignoreErrors,omitempty"`
	Retries      int    `yaml:"retries,omitempty" json:"retries,omitempty"`
	Delay        int    `yaml:"delay,omitempty" json:"delay,omitempty"`
}

// PackWorkflowTemplate 打包的工作流模板文件
type PackWorkflowTemplate struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Content     string `yaml:"content" json:"content"`
}

// PackStage 打包的阶段模板
type PackStage struct {
	Name         string     `yaml:"name" json:"name"`
	Description  string     `yaml:"description,omitempty" json:"description,omitempty"`
	MachineGroup string     `yaml:"machine_group,omitempty" json:"machineGroup,omitempty"` // 使用名称而非ID
	Tasks        []PackTask `yaml:"tasks" json:"tasks"`
}

// PackTask 打包的任务
type PackTask struct {
	Ref          int      `yaml:"ref" json:"ref"`
	Name         string   `yaml:"name" json:"name"`
	Module       string   `yaml:"module" json:"module"`
	Order        int      `yaml:"order" json:"order"`
	Params       string   `yaml:"params,omitempty" json:"params,omitempty"`
	When         string   `yaml:"when,omitempty" json:"when,omitempty"`
	Hooks        []string `yaml:"hooks,omitempty" json:"hooks,omitempty"` // 使用名称而非ID
	Loop         string   `yaml:"loop,omitempty" json:"loop,omitempty"`
	Timeout      int      `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	IgnoreErrors bool     `yaml:"ignore_errors,omitempty" json:"ignoreErrors,omitempty"`
	Retries      int      `yaml:"retries,omitempty" json:"retries,omitempty"`
	Delay        int      `yaml:"delay,omitempty" json:"delay,omitempty"`
	Register     string   `yaml:"register,omitempty" json:"register,omitempty"`
}

type PackWorkflow struct {
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description,omitempty" json:"description,omitempty"`
	StageGroups []PackStageGroup `yaml:"stage_groups,omitempty" json:"stageGroups,omitempty"`
	Hooks       []PackHookGroup  `yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

type PackHookGroup struct {
	Name         string `yaml:"name" json:"name"`
	Module       string `yaml:"module" json:"module"`
	Params       string `yaml:"params,omitempty" json:"params,omitempty"`
	Timeout      int    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	IgnoreErrors bool   `yaml:"ignore_errors,omitempty" json:"ignoreErrors,omitempty"`
	Retries      int    `yaml:"retries,omitempty" json:"retries,omitempty"`
	Delay        int    `yaml:"delay,omitempty" json:"delay,omitempty"`
}

type PackStageGroup struct {
	Name        string              `yaml:"name" json:"name"`
	Description string              `yaml:"description,omitempty" json:"description,omitempty"`
	Order       int                 `yaml:"order" json:"order"`
	Mode        string              `yaml:"mode" json:"mode"`
	Stages      []PackWorkflowStage `yaml:"stages,omitempty" json:"stages,omitempty"`
}

type PackWorkflowStage struct {
	Name         string             `yaml:"name" json:"name"`
	Description  string             `yaml:"description,omitempty" json:"description,omitempty"`
	Order        int                `yaml:"order" json:"order"`
	MachineGroup string             `yaml:"machine_group,omitempty" json:"machineGroup,omitempty"` // 使用名称而非ID
	Tasks        []PackWorkflowTask `yaml:"tasks" json:"tasks"`
}

type PackWorkflowTask struct {
	Ref          int      `yaml:"ref" json:"ref"`
	Name         string   `yaml:"name" json:"name"`
	Module       string   `yaml:"module" json:"module"`
	Params       string   `yaml:"params,omitempty" json:"params,omitempty"`
	Order        int      `yaml:"order" json:"order"`
	When         string   `yaml:"when,omitempty" json:"when,omitempty"`
	Hooks        []string `yaml:"hooks,omitempty" json:"hooks,omitempty"` // 使用名称而非ID
	Loop         string   `yaml:"loop,omitempty" json:"loop,omitempty"`
	Timeout      int      `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	IgnoreErrors bool     `yaml:"ignore_errors,omitempty" json:"ignoreErrors,omitempty"`
	Retries      int      `yaml:"retries,omitempty" json:"retries,omitempty"`
	Delay        int      `yaml:"delay,omitempty" json:"delay,omitempty"`
	Register     string   `yaml:"register,omitempty" json:"register,omitempty"`
}
