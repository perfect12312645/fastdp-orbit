package migrations

import (
	"fastdp-orbit/backend/models/common"
	"fastdp-orbit/backend/models/machine"
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
		// Common
		&common.Template{},
		&common.Cluster{},
		&common.AuditLog{},
		// Workflow - 定义层
		&workflow.Workflow{},
		&workflow.WorkflowStage{},
		&workflow.WorkflowTask{},
		// Workflow - 执行层
		&workflow.WorkflowExecution{},
		&workflow.WorkflowStageExecution{},
		&workflow.WorkflowTaskExecution{},
	)
	if err != nil {
		return err
	}

	return nil
}
