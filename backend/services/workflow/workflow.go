package workflow

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fastdp-orbit/backend/models/workflow"
	"fastdp-orbit/backend/pkg/errs"
	"fmt"

	"gorm.io/gorm"
)

// generateVersionString 生成随机版本字符串（8位hex）
func generateVersionString() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Service 工作流业务逻辑
type Service struct {
	db *gorm.DB
}

// NewService 创建工作流服务
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// DB 获取数据库实例
func (s *Service) DB() *gorm.DB {
	return s.db
}

// ListWorkflows 获取所有工作流（不含关联数据，列表用）
func (s *Service) ListWorkflows() ([]workflow.Workflow, error) {
	var wfs []workflow.Workflow
	if err := s.db.Order("created_at DESC").Find(&wfs).Error; err != nil {
		return nil, err
	}
	return wfs, nil
}

// GetWorkflow 获取工作流详情（含 stage_groups、stages、tasks、hooks）
func (s *Service) GetWorkflow(id uint) (*workflow.Workflow, error) {
	var wf workflow.Workflow
	if err := s.db.
		Preload("StageGroups.Stages.Tasks").
		Preload("Hooks").
		First(&wf, id).Error; err != nil {
		return nil, err
	}
	return &wf, nil
}

// CreateWorkflow 创建工作流（含 stage_groups、stages、tasks、variables、hooks，事务）
func (s *Service) CreateWorkflow(wf *workflow.Workflow) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 创建 workflow（跳过自动创建子关联）
		if err := tx.Omit("StageGroups", "Hooks").Create(wf).Error; err != nil {
			return err
		}

		// 创建 stage_groups
		for i := range wf.StageGroups {
			wf.StageGroups[i].WorkflowID = wf.ID
			if err := tx.Omit("Stages").Create(&wf.StageGroups[i]).Error; err != nil {
				return err
			}
			// 创建 stages
			for j := range wf.StageGroups[i].Stages {
				wf.StageGroups[i].Stages[j].StageGroupID = wf.StageGroups[i].ID
				if err := tx.Omit("Tasks").Create(&wf.StageGroups[i].Stages[j]).Error; err != nil {
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

		// 删除旧的 hooks
		tx.Where("workflow_id = ?", id).Delete(&workflow.WorkflowHook{})

		// 创建新的 stage_groups
		for i := range wf.StageGroups {
			wf.StageGroups[i].WorkflowID = id
			wf.StageGroups[i].ID = 0
			if err := tx.Omit("Stages").Create(&wf.StageGroups[i]).Error; err != nil {
				return err
			}
			for j := range wf.StageGroups[i].Stages {
				wf.StageGroups[i].Stages[j].StageGroupID = wf.StageGroups[i].ID
				wf.StageGroups[i].Stages[j].ID = 0
				if err := tx.Omit("Tasks").Create(&wf.StageGroups[i].Stages[j]).Error; err != nil {
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

// DeleteExecution 删除执行记录（级联删除关联的 stage/task 记录）
func (s *Service) DeleteExecution(executionID uint) error {
	var exec workflow.WorkflowExecution
	if err := s.db.First(&exec, executionID).Error; err != nil {
		return fmt.Errorf("执行记录不存在")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除 task_executions
		var stageExecIDs []uint
		tx.Model(&workflow.WorkflowStageExecution{}).
			Where("stage_group_execution_id IN (SELECT id FROM workflow_stage_group_executions WHERE execution_id = ?)", executionID).
			Pluck("id", &stageExecIDs)
		if len(stageExecIDs) > 0 {
			tx.Where("stage_execution_id IN ?", stageExecIDs).Delete(&workflow.WorkflowTaskExecution{})
		}
		// 删除 stage_executions
		tx.Where("stage_group_execution_id IN (SELECT id FROM workflow_stage_group_executions WHERE execution_id = ?)", executionID).
			Delete(&workflow.WorkflowStageExecution{})
		// 删除 stage_group_executions
		tx.Where("execution_id = ?", executionID).Delete(&workflow.WorkflowStageGroupExecution{})
		// 删除 execution
		return tx.Delete(&exec).Error
	})
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

// HasRunningExecutions 检查工作流是否有运行中的执行
func (s *Service) HasRunningExecutions(workflowID uint) (bool, error) {
	var count int64
	if err := s.db.Model(&workflow.WorkflowExecution{}).
		Where("workflow_id = ? AND status IN ?", workflowID, []string{"running", "paused"}).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
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
	hookNameSet := make(map[string]bool)
	for _, hook := range wf.Hooks {
		if hook.Name == "" {
			return fmt.Errorf("钩子名称不能为空")
		}
		if hook.Module == "" {
			return fmt.Errorf("钩子 [%s] 模块类型不能为空", hook.Name)
		}
		if hookNameSet[hook.Name] {
			return fmt.Errorf("钩子名称 [%s] 重复", hook.Name)
		}
		hookNameSet[hook.Name] = true
	}

	return nil
}

// ==================== StageTemplate CRUD ====================

// ValidateStageTemplate 校验阶段模板定义
func ValidateStageTemplate(name string, tasksJSON string) error {
	if name == "" {
		return errs.NewBadRequest(errs.CodeValidateFailed, "阶段名称不能为空")
	}

	if tasksJSON == "" {
		return nil // 允许空任务（草稿状态）
	}

	var tasks []workflow.StageTask
	if err := json.Unmarshal([]byte(tasksJSON), &tasks); err != nil {
		return errs.NewBadRequest(errs.CodeValidateFailed, fmt.Sprintf("任务数据格式错误: %v", err))
	}

	if len(tasks) == 0 {
		return nil
	}

	refSet := make(map[int]bool)
	orderSet := make(map[int]bool)

	for i, task := range tasks {
		if task.Ref <= 0 {
			return errs.NewBadRequest(errs.CodeValidateFailed, fmt.Sprintf("任务 %d 引用ID必须大于0", i+1))
		}
		if refSet[task.Ref] {
			return errs.NewBadRequest(errs.CodeValidateFailed, fmt.Sprintf("任务引用ID %d 重复", task.Ref))
		}
		refSet[task.Ref] = true

		if task.Name == "" {
			return errs.NewBadRequest(errs.CodeValidateFailed, fmt.Sprintf("任务 %d 名称不能为空", i+1))
		}
		if task.Module == "" {
			return errs.NewBadRequest(errs.CodeValidateFailed, fmt.Sprintf("任务 [%s] 模块类型不能为空", task.Name))
		}
		if task.Order <= 0 {
			return errs.NewBadRequest(errs.CodeValidateFailed, fmt.Sprintf("任务 [%s] 执行顺序必须大于0", task.Name))
		}
		if orderSet[task.Order] {
			return errs.NewBadRequest(errs.CodeValidateFailed, fmt.Sprintf("任务执行顺序 %d 重复", task.Order))
		}
		orderSet[task.Order] = true
	}

	return nil
}

// ListStageTemplates 获取阶段模板（支持按分组过滤）
func (s *Service) ListStageTemplates(packageGroup string) ([]workflow.StageTemplate, error) {
	var templates []workflow.StageTemplate
	query := s.db.Order("created_at DESC")
	if packageGroup != "" {
		query = query.Where("package_group = ?", packageGroup)
	}
	if err := query.Find(&templates).Error; err != nil {
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

// CreateStageTemplate 创建阶段模板（初始版本）
func (s *Service) CreateStageTemplate(t *workflow.StageTemplate) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查 name 唯一性（同分组内，已软删除的不算）
		var count int64
		if err := tx.Model(&workflow.StageTemplate{}).
			Where("name = ? AND package_group = ?", t.Name, t.Source).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errs.NewConflict(errs.CodeStageTemplateNameDuplicate,
				fmt.Sprintf("阶段名称「%s」在当前分组已存在", t.Name))
		}

		// 校验任务
		if err := ValidateStageTemplate(t.Name, t.Tasks); err != nil {
			return err
		}

		t.Version = generateVersionString()
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		// 创建初始版本记录
		version := workflow.StageTemplateVersion{
			TemplateID:     t.ID,
			Version:        t.Version,
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
		// 获取当前模板（仅用于校验存在性和名称唯一性）
		var existing workflow.StageTemplate
		if err := tx.First(&existing, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errs.NewNotFound(errs.CodeStageTemplateNotFound, "阶段模板不存在")
			}
			return err
		}

		// 如果 name 有变化，检查新 name 唯一性（同分组内，排除自身和已软删除的）
		if t.Name != existing.Name {
			var count int64
			if err := tx.Model(&workflow.StageTemplate{}).
				Where("name = ? AND package_group = ? AND id != ?", t.Name, t.Source, id).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errs.NewConflict(errs.CodeStageTemplateNameDuplicate,
					fmt.Sprintf("阶段名称「%s」在当前分组已存在", t.Name))
			}
		}

		// 校验任务
		if err := ValidateStageTemplate(t.Name, t.Tasks); err != nil {
			return err
		}

		// 生成新版本字符串
		newVersion := generateVersionString()

		// 将新内容保存为历史版本（旧版本已经在历史表中，无需重复保存）
		versionRecord := workflow.StageTemplateVersion{
			TemplateID:     id,
			Version:        newVersion,
			Name:           t.Name,
			Description:    t.Description,
			MachineGroupID: t.MachineGroupID,
			Tasks:          t.Tasks,
			ChangeNote:     changeNote,
		}
		if err := tx.Create(&versionRecord).Error; err != nil {
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
		Order("created_at DESC").Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// RollbackStageTemplate 回滚到指定版本（直接复制目标版本内容到主表）
func (s *Service) RollbackStageTemplate(templateID uint, targetVersion string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 获取目标版本内容
		var target workflow.StageTemplateVersion
		if err := tx.Where("template_id = ? AND version = ?", templateID, targetVersion).
			First(&target).Error; err != nil {
			return errs.NewNotFound(errs.CodeStageTemplateVersionMiss,
				fmt.Sprintf("版本 %s 不存在", targetVersion))
		}

		// 直接将目标版本内容更新到主表，版本字符串保持不变
		// 当前版本已在历史表中，无需重复保存
		return tx.Model(&workflow.StageTemplate{}).Where("id = ?", templateID).Updates(
			map[string]interface{}{
				"name":             target.Name,
				"description":      target.Description,
				"machine_group_id": target.MachineGroupID,
				"tasks":            target.Tasks,
				"version":          target.Version,
			},
		).Error
	})
}

// ==================== GlobalVariable ====================

// ListGlobalVariables 获取全局变量（支持按分组过滤）
func (s *Service) ListGlobalVariables(packageGroup string) ([]workflow.GlobalVariable, error) {
	var vars []workflow.GlobalVariable
	query := s.db.Order("`group` ASC, `key` ASC")
	if packageGroup != "" {
		query = query.Where("package_group = ?", packageGroup)
	}
	if err := query.Find(&vars).Error; err != nil {
		return nil, err
	}
	return vars, nil
}

// GetGlobalVariable 获取全局变量详情
func (s *Service) GetGlobalVariable(id uint) (*workflow.GlobalVariable, error) {
	var v workflow.GlobalVariable
	if err := s.db.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// CreateGlobalVariable 创建全局变量
func (s *Service) CreateGlobalVariable(v *workflow.GlobalVariable) error {
	// 检查变量名唯一性（同分组内）
	var count int64
	if err := s.db.Model(&workflow.GlobalVariable{}).Where("key = ? AND package_group = ?", v.Key, v.Source).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("变量名「%s」在当前分组已存在", v.Key)
	}
	if v.Value == "" {
		return fmt.Errorf("变量值不能为空")
	}
	return s.db.Create(v).Error
}

// UpdateGlobalVariable 更新全局变量
func (s *Service) UpdateGlobalVariable(id uint, v *workflow.GlobalVariable) error {
	// 检查变量名唯一性（同分组内，排除自身）
	var count int64
	if err := s.db.Model(&workflow.GlobalVariable{}).Where("key = ? AND package_group = ? AND id != ?", v.Key, v.Source, id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("变量名「%s」在当前分组已存在", v.Key)
	}
	if v.Value == "" {
		return fmt.Errorf("变量值不能为空")
	}
	return s.db.Model(&workflow.GlobalVariable{}).Where("id = ?", id).Updates(v).Error
}

// DeleteGlobalVariable 删除全局变量
func (s *Service) DeleteGlobalVariable(id uint) error {
	return s.db.Delete(&workflow.GlobalVariable{}, id).Error
}
