package migrations

import (
	"fastdp-orbit/backend/models/common"
	"fastdp-orbit/backend/models/machine"
	"fastdp-orbit/backend/models/storage"
	"fastdp-orbit/backend/models/workflow"

	"gorm.io/gorm"
)

// InitialMigration creates initial database tables
func InitialMigration(db *gorm.DB) error {
	// Auto migrate models
	err := db.AutoMigrate(
		// Machine
		&machine.Machine{},
		&machine.MachineDisk{},
		&machine.MachineNetwork{},
		&machine.MachineGPU{},
		&machine.MachineGroup{},
		&machine.MachineGroupMember{},
		// Common
		&common.Template{},
		&common.Cluster{},
		&common.AuditLog{},
		// Workflow - 定义层
		&workflow.Workflow{},
		&workflow.WorkflowStageGroup{},
		&workflow.WorkflowStage{},
		&workflow.WorkflowTask{},
		&workflow.WorkflowHook{},
		// Workflow - 执行层
		&workflow.WorkflowExecution{},
		&workflow.WorkflowStageGroupExecution{},
		&workflow.WorkflowStageExecution{},
		&workflow.WorkflowTaskExecution{},
		// Stage Templates
		&workflow.StageTemplate{},
		&workflow.StageTemplateVersion{},
		// Global Variables
		&workflow.GlobalVariable{},
		// Hook Templates
		&workflow.HookTemplate{},
		// Workflow Templates（模板文件）
		&workflow.WorkflowTemplate{},
		// Storage（文件存储）
		&storage.StorageFile{},
		// SolutionLibrary（方案库）
		&workflow.SolutionLibrary{},
		// StageExecution（单阶段执行记录）
		&workflow.StageExecution{},
		&workflow.StageTaskExecution{},
	)
	if err != nil {
		return err
	}

	return nil
}
