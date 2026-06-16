package workflow

import (
	"fastdp-orbit/backend/models/workflow"
	"fmt"

	"gorm.io/gorm"
)

// Service 工作流业务逻辑
type Service struct {
	db *gorm.DB
}

// NewService 创建工作流服务
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ListWorkflows 获取所有工作流（不含关联数据，列表用）
func (s *Service) ListWorkflows() ([]workflow.Workflow, error) {
	var wfs []workflow.Workflow
	if err := s.db.Order("created_at DESC").Find(&wfs).Error; err != nil {
		return nil, err
	}
	return wfs, nil
}

// GetWorkflow 获取工作流详情（含 stages 和 tasks）
func (s *Service) GetWorkflow(id uint) (*workflow.Workflow, error) {
	var wf workflow.Workflow
	if err := s.db.Preload("Stages.Tasks").First(&wf, id).Error; err != nil {
		return nil, err
	}
	return &wf, nil
}

// CreateWorkflow 创建工作流（含 stages 和 tasks，事务）
func (s *Service) CreateWorkflow(wf *workflow.Workflow) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 创建 workflow
		if err := tx.Create(wf).Error; err != nil {
			return err
		}

		// 创建 stages 和 tasks
		for i := range wf.Stages {
			wf.Stages[i].WorkflowID = wf.ID
			if err := tx.Create(&wf.Stages[i]).Error; err != nil {
				return err
			}
			for j := range wf.Stages[i].Tasks {
				wf.Stages[i].Tasks[j].StageID = wf.Stages[i].ID
				if err := tx.Create(&wf.Stages[i].Tasks[j]).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// UpdateWorkflow 更新工作流（事务替换 stages + tasks）
func (s *Service) UpdateWorkflow(id uint, wf *workflow.Workflow) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查存在
		var existing workflow.Workflow
		if err := tx.First(&existing, id).Error; err != nil {
			return err
		}

		// 更新基本字段
		existing.Name = wf.Name
		existing.Description = wf.Description
		existing.Config = wf.Config
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}

		// 删除旧的 stages（级联删除 tasks）
		var oldStageIDs []uint
		tx.Model(&workflow.WorkflowStage{}).Where("workflow_id = ?", id).Pluck("id", &oldStageIDs)
		if len(oldStageIDs) > 0 {
			tx.Where("stage_id IN ?", oldStageIDs).Delete(&workflow.WorkflowTask{})
			tx.Where("id IN ?", oldStageIDs).Delete(&workflow.WorkflowStage{})
		}

		// 创建新的 stages 和 tasks
		for i := range wf.Stages {
			wf.Stages[i].WorkflowID = id
			wf.Stages[i].ID = 0 // 重置 ID
			if err := tx.Create(&wf.Stages[i]).Error; err != nil {
				return err
			}
			for j := range wf.Stages[i].Tasks {
				wf.Stages[i].Tasks[j].StageID = wf.Stages[i].ID
				wf.Stages[i].Tasks[j].ID = 0
				if err := tx.Create(&wf.Stages[i].Tasks[j]).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DeleteWorkflow 删除工作流（级联）
func (s *Service) DeleteWorkflow(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除 task_executions
		var stageExecIDs []uint
		tx.Model(&workflow.WorkflowStageExecution{}).
			Where("execution_id IN (SELECT id FROM workflow_executions WHERE workflow_id = ?)", id).
			Pluck("id", &stageExecIDs)
		if len(stageExecIDs) > 0 {
			tx.Where("stage_execution_id IN ?", stageExecIDs).Delete(&workflow.WorkflowTaskExecution{})
		}
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowStageExecution{})
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowExecution{})

		// 删除 tasks
		var stageIDs []uint
		tx.Model(&workflow.WorkflowStage{}).Where("workflow_id = ?", id).Pluck("id", &stageIDs)
		if len(stageIDs) > 0 {
			tx.Where("stage_id IN ?", stageIDs).Delete(&workflow.WorkflowTask{})
		}
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowStage{})

		// 删除 workflow
		return tx.Delete(&workflow.Workflow{}, id).Error
	})
}

// ListExecutions 获取工作流的执行历史
func (s *Service) ListExecutions(workflowID uint) ([]workflow.WorkflowExecution, error) {
	var execs []workflow.WorkflowExecution
	if err := s.db.Where("workflow_id = ?", workflowID).
		Order("created_at DESC").Find(&execs).Error; err != nil {
		return nil, err
	}
	return execs, nil
}

// GetExecution 获取执行详情（含各 stage/task 状态）
func (s *Service) GetExecution(executionID uint) (*workflow.WorkflowExecution, error) {
	var exec workflow.WorkflowExecution
	if err := s.db.
		Preload("StageExecutions.Stage").
		Preload("StageExecutions.TaskExecutions.Task").
		First(&exec, executionID).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

// ValidateWorkflow 校验工作流定义
func (s *Service) ValidateWorkflow(wf *workflow.Workflow) error {
	if wf.Name == "" {
		return fmt.Errorf("工作流名称不能为空")
	}
	if len(wf.Stages) == 0 {
		return fmt.Errorf("工作流至少需要一个阶段")
	}
	for i, stage := range wf.Stages {
		if stage.Name == "" {
			return fmt.Errorf("阶段 %d 名称不能为空", i+1)
		}
		if len(stage.Tasks) == 0 {
			return fmt.Errorf("阶段 [%s] 至少需要一个任务", stage.Name)
		}
		for j, task := range stage.Tasks {
			if task.Host == "" {
				return fmt.Errorf("阶段 [%s] 任务 %d 目标主机不能为空", stage.Name, j+1)
			}
			if task.Module == "" {
				return fmt.Errorf("阶段 [%s] 任务 %d 模块类型不能为空", stage.Name, j+1)
			}
		}
	}
	return nil
}
