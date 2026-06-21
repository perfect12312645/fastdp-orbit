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

// GetWorkflow 获取工作流详情（含 stage_groups、stages、tasks、variables、hooks）
func (s *Service) GetWorkflow(id uint) (*workflow.Workflow, error) {
	var wf workflow.Workflow
	if err := s.db.
		Preload("StageGroups.Stages.Tasks").
		Preload("Variables").
		Preload("Hooks").
		First(&wf, id).Error; err != nil {
		return nil, err
	}
	return &wf, nil
}

// CreateWorkflow 创建工作流（含 stage_groups、stages、tasks、variables、hooks，事务）
func (s *Service) CreateWorkflow(wf *workflow.Workflow) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 创建 workflow
		if err := tx.Create(wf).Error; err != nil {
			return err
		}

		// 创建 stage_groups
		for i := range wf.StageGroups {
			wf.StageGroups[i].WorkflowID = wf.ID
			if err := tx.Create(&wf.StageGroups[i]).Error; err != nil {
				return err
			}
			// 创建 stages
			for j := range wf.StageGroups[i].Stages {
				wf.StageGroups[i].Stages[j].StageGroupID = wf.StageGroups[i].ID
				if err := tx.Create(&wf.StageGroups[i].Stages[j]).Error; err != nil {
					return err
				}
				// 创建 tasks
				for k := range wf.StageGroups[i].Stages[j].Tasks {
					wf.StageGroups[i].Stages[j].Tasks[k].StageID = wf.StageGroups[i].Stages[j].ID
					if err := tx.Create(&wf.StageGroups[i].Stages[j].Tasks[k]).Error; err != nil {
						return err
					}
				}
			}
		}

		// 创建 variables
		for i := range wf.Variables {
			wf.Variables[i].WorkflowID = wf.ID
			if err := tx.Create(&wf.Variables[i]).Error; err != nil {
				return err
			}
		}

		// 创建 hooks
		for i := range wf.Hooks {
			wf.Hooks[i].WorkflowID = wf.ID
			if err := tx.Create(&wf.Hooks[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// UpdateWorkflow 更新工作流（事务替换 stage_groups + stages + tasks + variables + hooks）
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

		// 删除旧的 stage_groups（级联删除 stages 和 tasks）
		var oldGroupIDs []uint
		tx.Model(&workflow.WorkflowStageGroup{}).Where("workflow_id = ?", id).Pluck("id", &oldGroupIDs)
		if len(oldGroupIDs) > 0 {
			// 删除 tasks（通过 stage_id 关联）
			var oldStageIDs []uint
			tx.Model(&workflow.WorkflowStage{}).Where("stage_group_id IN ?", oldGroupIDs).Pluck("id", &oldStageIDs)
			if len(oldStageIDs) > 0 {
				tx.Where("stage_id IN ?", oldStageIDs).Delete(&workflow.WorkflowTask{})
			}
			// 删除 stages
			tx.Where("stage_group_id IN ?", oldGroupIDs).Delete(&workflow.WorkflowStage{})
			// 删除 stage_groups
			tx.Where("id IN ?", oldGroupIDs).Delete(&workflow.WorkflowStageGroup{})
		}

		// 删除旧的 variables
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowVariable{})

		// 删除旧的 hooks
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowHook{})

		// 创建新的 stage_groups
		for i := range wf.StageGroups {
			wf.StageGroups[i].WorkflowID = id
			wf.StageGroups[i].ID = 0
			if err := tx.Create(&wf.StageGroups[i]).Error; err != nil {
				return err
			}
			for j := range wf.StageGroups[i].Stages {
				wf.StageGroups[i].Stages[j].StageGroupID = wf.StageGroups[i].ID
				wf.StageGroups[i].Stages[j].ID = 0
				if err := tx.Create(&wf.StageGroups[i].Stages[j]).Error; err != nil {
					return err
				}
				for k := range wf.StageGroups[i].Stages[j].Tasks {
					wf.StageGroups[i].Stages[j].Tasks[k].StageID = wf.StageGroups[i].Stages[j].ID
					wf.StageGroups[i].Stages[j].Tasks[k].ID = 0
					if err := tx.Create(&wf.StageGroups[i].Stages[j].Tasks[k]).Error; err != nil {
						return err
					}
				}
			}
		}

		// 创建新的 variables
		for i := range wf.Variables {
			wf.Variables[i].WorkflowID = id
			wf.Variables[i].ID = 0
			if err := tx.Create(&wf.Variables[i]).Error; err != nil {
				return err
			}
		}

		// 创建新的 hooks
		for i := range wf.Hooks {
			wf.Hooks[i].WorkflowID = id
			wf.Hooks[i].ID = 0
			if err := tx.Create(&wf.Hooks[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// DeleteWorkflow 删除工作流（级联）
func (s *Service) DeleteWorkflow(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除 task_executions（通过 stage_executions 关联）
		var stageExecIDs []uint
		tx.Model(&workflow.WorkflowStageExecution{}).
			Where("stage_group_execution_id IN (SELECT id FROM workflow_stage_group_executions WHERE execution_id IN (SELECT id FROM workflow_executions WHERE workflow_id = ?))", id).
			Pluck("id", &stageExecIDs)
		if len(stageExecIDs) > 0 {
			tx.Where("stage_execution_id IN ?", stageExecIDs).Delete(&workflow.WorkflowTaskExecution{})
		}

		// 删除 stage_executions（通过 stage_group_executions 关联）
		var stageGroupExecIDs []uint
		tx.Model(&workflow.WorkflowStageGroupExecution{}).
			Where("execution_id IN (SELECT id FROM workflow_executions WHERE workflow_id = ?)", id).
			Pluck("id", &stageGroupExecIDs)
		if len(stageGroupExecIDs) > 0 {
			tx.Where("stage_group_execution_id IN ?", stageGroupExecIDs).Delete(&workflow.WorkflowStageExecution{})
		}

		// 删除 stage_group_executions
		tx.Where("execution_id IN (SELECT id FROM workflow_executions WHERE workflow_id = ?)", id).
			Delete(&workflow.WorkflowStageGroupExecution{})

		// 删除 executions
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowExecution{})

		// 删除 tasks（通过 stages 关联）
		var groupIDs []uint
		tx.Model(&workflow.WorkflowStageGroup{}).Where("workflow_id = ?", id).Pluck("id", &groupIDs)
		if len(groupIDs) > 0 {
			var stageIDs []uint
			tx.Model(&workflow.WorkflowStage{}).Where("stage_group_id IN ?", groupIDs).Pluck("id", &stageIDs)
			if len(stageIDs) > 0 {
				tx.Where("stage_id IN ?", stageIDs).Delete(&workflow.WorkflowTask{})
			}
			// 删除 stages
			tx.Where("stage_group_id IN ?", groupIDs).Delete(&workflow.WorkflowStage{})
		}

		// 删除 stage_groups
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowStageGroup{})

		// 删除 variables
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowVariable{})

		// 删除 hooks
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowHook{})

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

// GetExecution 获取执行详情（含各 group/stage/task 状态）
func (s *Service) GetExecution(executionID uint) (*workflow.WorkflowExecution, error) {
	var exec workflow.WorkflowExecution
	if err := s.db.
		Preload("StageGroupExecutions.Group").
		Preload("StageGroupExecutions.StageExecutions.Stage").
		Preload("StageGroupExecutions.StageExecutions.TaskExecutions.Task").
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
	if len(wf.StageGroups) == 0 {
		return nil // 允许空阶段组（草稿状态）
	}

	// 收集所有 Task Ref 用于唯一性校验
	refSet := make(map[int]bool)
	// 收集同层级 Order 用于唯一性校验
	groupOrderSet := make(map[int]bool)

	for i, group := range wf.StageGroups {
		if group.Name == "" {
			return fmt.Errorf("阶段组 %d 名称不能为空", i+1)
		}
		if group.Order <= 0 {
			return fmt.Errorf("阶段组 [%s] 执行顺序必须大于0", group.Name)
		}
		if groupOrderSet[group.Order] {
			return fmt.Errorf("阶段组执行顺序 %d 重复", group.Order)
		}
		groupOrderSet[group.Order] = true
		if len(group.Stages) == 0 {
			return fmt.Errorf("阶段组 [%s] 至少需要一个阶段", group.Name)
		}

		stageOrderSet := make(map[int]bool)
		for j, stage := range group.Stages {
			if stage.Name == "" {
				return fmt.Errorf("阶段组 [%s] 阶段 %d 名称不能为空", group.Name, j+1)
			}
			if stage.Order <= 0 {
				return fmt.Errorf("阶段 [%s] 执行顺序必须大于0", stage.Name)
			}
			if stageOrderSet[stage.Order] {
				return fmt.Errorf("阶段 [%s] 执行顺序 %d 重复", stage.Name, stage.Order)
			}
			stageOrderSet[stage.Order] = true
			if stage.MachineGroupID == 0 {
				return fmt.Errorf("阶段 [%s] 必须选择一个机器分组", stage.Name)
			}
			if len(stage.Tasks) == 0 {
				return fmt.Errorf("阶段 [%s] 至少需要一个任务", stage.Name)
			}

			taskOrderSet := make(map[int]bool)
			for k, task := range stage.Tasks {
				if task.Name == "" {
					return fmt.Errorf("阶段 [%s] 任务 %d 名称不能为空", stage.Name, k+1)
				}
				if task.Module == "" {
					return fmt.Errorf("阶段 [%s] 任务 [%s] 模块类型不能为空", stage.Name, task.Name)
				}
				if task.Order <= 0 {
					return fmt.Errorf("阶段 [%s] 任务 [%s] 执行顺序必须大于0", stage.Name, task.Name)
				}
				if taskOrderSet[task.Order] {
					return fmt.Errorf("阶段 [%s] 任务执行顺序 %d 重复", stage.Name, task.Order)
				}
				taskOrderSet[task.Order] = true
				if task.Ref == 0 {
					return fmt.Errorf("阶段 [%s] 任务 %d 引用ID不能为空", stage.Name, k+1)
				}
				if refSet[task.Ref] {
					return fmt.Errorf("任务引用ID %d 重复", task.Ref)
				}
				refSet[task.Ref] = true
			}
		}
	}

	// 校验 Hook
	hookRefSet := make(map[int]bool)
	for _, hook := range wf.Hooks {
		if hook.Name == "" {
			return fmt.Errorf("钩子引用ID %d 名称不能为空", hook.Ref)
		}
		if hook.Module == "" {
			return fmt.Errorf("钩子 [%s] 模块类型不能为空", hook.Name)
		}
		if hook.Ref == 0 {
			return fmt.Errorf("钩子 [%s] 引用ID不能为空", hook.Name)
		}
		if hookRefSet[hook.Ref] {
			return fmt.Errorf("钩子引用ID %d 重复", hook.Ref)
		}
		hookRefSet[hook.Ref] = true
	}

	return nil
}

// ==================== StageTemplate CRUD ====================

// ListStageTemplates 获取所有阶段模板
func (s *Service) ListStageTemplates() ([]workflow.StageTemplate, error) {
	var templates []workflow.StageTemplate
	if err := s.db.Order("created_at DESC").Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// GetStageTemplate 获取阶段模板详情
func (s *Service) GetStageTemplate(id uint) (*workflow.StageTemplate, error) {
	var t workflow.StageTemplate
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateStageTemplate 创建阶段模板（初始版本 v1）
func (s *Service) CreateStageTemplate(t *workflow.StageTemplate) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		t.Version = 1
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		// 创建初始版本记录
		version := workflow.StageTemplateVersion{
			TemplateID:     t.ID,
			Version:        1,
			Name:           t.Name,
			Description:    t.Description,
			MachineGroupID: t.MachineGroupID,
			Tasks:          t.Tasks,
			ChangeNote:     "初始版本",
		}
		return tx.Create(&version).Error
	})
}

// UpdateStageTemplate 更新阶段模板（强制生成新版本）
func (s *Service) UpdateStageTemplate(id uint, t *workflow.StageTemplate, changeNote string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 获取当前模板
		var existing workflow.StageTemplate
		if err := tx.First(&existing, id).Error; err != nil {
			return err
		}

		// 将当前内容保存为历史版本
		newVersion := existing.Version + 1
		version := workflow.StageTemplateVersion{
			TemplateID:     id,
			Version:        existing.Version,
			Name:           existing.Name,
			Description:    existing.Description,
			MachineGroupID: existing.MachineGroupID,
			Tasks:          existing.Tasks,
			ChangeNote:     changeNote,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}

		// 更新主表
		t.Version = newVersion
		return tx.Model(&workflow.StageTemplate{}).Where("id = ?", id).Updates(t).Error
	})
}

// DeleteStageTemplate 删除阶段模板（级联删除版本历史）
func (s *Service) DeleteStageTemplate(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除版本历史
		if err := tx.Where("template_id = ?", id).Delete(&workflow.StageTemplateVersion{}).Error; err != nil {
			return err
		}
		// 删除主表
		return tx.Delete(&workflow.StageTemplate{}, id).Error
	})
}

// ==================== StageTemplate Version ====================

// ListStageTemplateVersions 获取阶段模板的版本历史
func (s *Service) ListStageTemplateVersions(templateID uint) ([]workflow.StageTemplateVersion, error) {
	var versions []workflow.StageTemplateVersion
	if err := s.db.Where("template_id = ?", templateID).
		Order("version DESC").Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// RollbackStageTemplate 回滚到指定版本（基于旧版本内容创建新版本）
func (s *Service) RollbackStageTemplate(templateID uint, targetVersion int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 获取目标版本内容
		var target workflow.StageTemplateVersion
		if err := tx.Where("template_id = ? AND version = ?", templateID, targetVersion).
			First(&target).Error; err != nil {
			return fmt.Errorf("版本 %d 不存在", targetVersion)
		}

		// 获取当前模板
		var current workflow.StageTemplate
		if err := tx.First(&current, templateID).Error; err != nil {
			return err
		}

		// 将当前内容保存为历史版本
		currentVersion := workflow.StageTemplateVersion{
			TemplateID:     templateID,
			Version:        current.Version,
			Name:           current.Name,
			Description:    current.Description,
			MachineGroupID: current.MachineGroupID,
			Tasks:          current.Tasks,
			ChangeNote:     fmt.Sprintf("回滚前版本（回滚到 v%d）", targetVersion),
		}
		if err := tx.Create(&currentVersion).Error; err != nil {
			return err
		}

		// 更新主表为目标版本内容，版本号递增
		newVersion := current.Version + 1
		return tx.Model(&workflow.StageTemplate{}).Where("id = ?", templateID).Updates(
			map[string]interface{}{
				"name":             target.Name,
				"description":      target.Description,
				"machine_group_id": target.MachineGroupID,
				"tasks":            target.Tasks,
				"version":          newVersion,
			},
		).Error
	})
}
