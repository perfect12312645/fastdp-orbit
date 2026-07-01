package workflow

import (
	"encoding/json"
	"fmt"

	"fastdp-orbit/backend/models/machine"
	"fastdp-orbit/backend/models/storage"
	"fastdp-orbit/backend/models/workflow"

	"gorm.io/gorm"
)

// ==================== SolutionLibrary CRUD ====================

// ListSolutionLibrarys 获取方案列表
func (s *Service) ListSolutionLibrarys(category string) ([]workflow.SolutionLibrary, error) {
	var solutions []workflow.SolutionLibrary
	query := s.db.Order("created_at DESC")
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if err := query.Find(&solutions).Error; err != nil {
		return nil, err
	}
	return solutions, nil
}

// GetSolutionLibrary 获取方案详情
func (s *Service) GetSolutionLibrary(id uint) (*workflow.SolutionLibrary, error) {
	var solution workflow.SolutionLibrary
	if err := s.db.First(&solution, id).Error; err != nil {
		return nil, err
	}
	return &solution, nil
}

// CreateSolutionLibrary 创建方案（存储关联ID）
func (s *Service) CreateSolutionLibrary(solution *workflow.SolutionLibrary) error {
	if solution.Name == "" {
		return fmt.Errorf("方案名称不能为空")
	}
	// 检查名称唯一性
	var count int64
	s.db.Model(&workflow.SolutionLibrary{}).Where("name = ?", solution.Name).Count(&count)
	if count > 0 {
		return fmt.Errorf("方案名称「%s」已存在", solution.Name)
	}
	// 更新统计字段
	s.updateCounts(solution)
	return s.db.Create(solution).Error
}

// UpdateSolutionLibrary 更新方案
func (s *Service) UpdateSolutionLibrary(id uint, solution *workflow.SolutionLibrary) error {
	if solution.Name == "" {
		return fmt.Errorf("方案名称不能为空")
	}
	// 检查名称唯一性（排除自身）
	var count int64
	s.db.Model(&workflow.SolutionLibrary{}).Where("name = ? AND id != ?", solution.Name, id).Count(&count)
	if count > 0 {
		return fmt.Errorf("方案名称「%s」已存在", solution.Name)
	}
	// 更新统计字段
	s.updateCounts(solution)
	return s.db.Model(&workflow.SolutionLibrary{}).Where("id = ?", id).Updates(solution).Error
}

// DeleteSolutionLibrary 删除方案
func (s *Service) DeleteSolutionLibrary(id uint) error {
	var solution workflow.SolutionLibrary
	if err := s.db.First(&solution, id).Error; err != nil {
		return fmt.Errorf("方案不存在")
	}
	return s.db.Delete(&solution).Error
}

// ExportSolutionLibrary 导出方案为 orbit-pack YAML
func (s *Service) ExportSolutionLibrary(id uint) (*workflow.OrbitPack, error) {
	var solution workflow.SolutionLibrary
	if err := s.db.First(&solution, id).Error; err != nil {
		return nil, fmt.Errorf("方案不存在")
	}

	// 如果方案未应用（有 pack_data），直接解析返回原始数据
	if solution.PackData != "" {
		var pack workflow.OrbitPack
		if err := json.Unmarshal([]byte(solution.PackData), &pack); err != nil {
			return nil, fmt.Errorf("解析方案数据失败: %v", err)
		}
		return &pack, nil
	}

	// 已应用的方案，根据存储的ID查询数据导出
	pack := &workflow.OrbitPack{
		APIVersion: "orbit/v1",
		Kind:       "SolutionLibrary",
	}
	pack.Metadata.Name = solution.Name
	pack.Metadata.Description = solution.Description
	pack.Metadata.Category = solution.Category
	pack.Metadata.Version = solution.Version
	pack.Metadata.Author = solution.Author

	// 解析关联ID
	stageIDs := parseIDs(solution.StageIDs)
	variableIDs := parseIDs(solution.VariableIDs)
	hookIDs := parseIDs(solution.HookIDs)
	templateIDs := parseIDs(solution.TemplateIDs)
	fileIDs := parseIDs(solution.FileIDs)
	workflowIDs := parseIDs(solution.WorkflowIDs)

	// 导出全局变量
	if len(variableIDs) > 0 {
		var vars []workflow.GlobalVariable
		s.db.Where("id IN ?", variableIDs).Find(&vars)
		for _, v := range vars {
			pack.GlobalVariables = append(pack.GlobalVariables, workflow.PackGlobalVariable{
				Key: v.Key, Type: v.Type, Value: v.Value,
				Description: v.Description, Group: v.Group,
			})
		}
	}

	// 导出钩子模板
	if len(hookIDs) > 0 {
		var hooks []workflow.HookTemplate
		s.db.Where("id IN ?", hookIDs).Find(&hooks)
		for _, h := range hooks {
			pack.Hooks = append(pack.Hooks, workflow.PackHook{
				Name: h.Name, Description: h.Description, Module: h.Module,
				Params: h.Params, Timeout: h.Timeout, IgnoreErrors: h.IgnoreErrors,
				Retries: h.Retries, Delay: h.Delay,
			})
		}
	}

	// 导出模板文件
	if len(templateIDs) > 0 {
		var templates []workflow.WorkflowTemplate
		s.db.Where("id IN ?", templateIDs).Find(&templates)
		for _, t := range templates {
			pack.WorkflowTemplates = append(pack.WorkflowTemplates, workflow.PackWorkflowTemplate{
				Name: t.Name, Description: t.Description, Content: t.Content,
			})
		}
	}

	// 导出阶段模板
	if len(stageIDs) > 0 {
		var stages []workflow.StageTemplate
		s.db.Where("id IN ?", stageIDs).Find(&stages)
		for _, st := range stages {
			packStage := workflow.PackStage{
				Name:        st.Name,
				Description: st.Description,
			}
			// 获取机器分组名称
			if st.MachineGroupID > 0 {
				var mg machine.MachineGroup
				if err := s.db.First(&mg, st.MachineGroupID).Error; err == nil {
					packStage.MachineGroup = mg.Name
				}
			}
			// 解析任务
			var tasks []workflow.StageTask
			if err := json.Unmarshal([]byte(st.Tasks), &tasks); err == nil {
				for _, t := range tasks {
					// hooks 字段直接存储钩子名称数组，如 ["test","rollback"]
					var hookNames []string
					if t.Hooks != "" && t.Hooks != "null" {
						json.Unmarshal([]byte(t.Hooks), &hookNames)
					}

					packStage.Tasks = append(packStage.Tasks, workflow.PackTask{
						Ref: t.Ref, Name: t.Name, Module: t.Module, Order: t.Order,
						Params: t.Params, When: t.When, Hooks: hookNames,
						Loop: t.Loop, Timeout: t.Timeout, IgnoreErrors: t.IgnoreErrors,
						Retries: t.Retries, Delay: t.Delay, Register: t.Register,
					})
				}
			}
			pack.Stages = append(pack.Stages, packStage)
		}
	}

	// 导出物料信息（从存储文件）
	if len(fileIDs) > 0 {
		var files []storage.StorageFile
		s.db.Where("id IN ?", fileIDs).Find(&files)
		for _, f := range files {
			pack.Materials = append(pack.Materials, workflow.PackMaterial{
				Name: f.Name,
				Size: f.Size,
				MD5:  f.MD5,
			})
		}
	}
	// 导出工作流信息
	if len(workflowIDs) > 0 {
		var workflows []workflow.Workflow
		s.db.Preload("Hooks").Preload("StageGroups.Stages.Tasks").Where("id IN ?", workflowIDs).Find(&workflows)
		for _, wf := range workflows {
			var packHooks []workflow.PackHookGroup
			for _, h := range wf.Hooks {
				packHooks = append(packHooks, workflow.PackHookGroup{
					Name: h.Name, Module: h.Module, Params: h.Params,
					Timeout: h.Timeout, IgnoreErrors: h.IgnoreErrors,
					Retries: h.Retries, Delay: h.Delay,
				})
			}

			// 转换阶段组
			var packStageGroups []workflow.PackStageGroup
			for _, sg := range wf.StageGroups {
				var packStages []workflow.PackWorkflowStage
				for _, st := range sg.Stages {
					// 解析机器分组名称
					var machineGroupName string
					if st.MachineGroupID > 0 {
						var mg machine.MachineGroup
						if err := s.db.First(&mg, st.MachineGroupID).Error; err == nil {
							machineGroupName = mg.Name
						}
					}
					// 转换任务
					var packTasks []workflow.PackWorkflowTask
					for _, t := range st.Tasks {
						var hookNames []string
						if t.Hooks != "" && t.Hooks != "null" {
							json.Unmarshal([]byte(t.Hooks), &hookNames)
						}
						packTasks = append(packTasks, workflow.PackWorkflowTask{
							Ref: t.Ref, Name: t.Name, Module: t.Module, Order: t.Order,
							Params: t.Params, When: t.When, Hooks: hookNames,
							Loop: t.Loop, Timeout: t.Timeout, IgnoreErrors: t.IgnoreErrors,
							Retries: t.Retries, Delay: t.Delay, Register: t.Register,
						})
					}
					packStages = append(packStages, workflow.PackWorkflowStage{
						Name: st.Name, Description: st.Description, Order: st.Order,
						MachineGroup: machineGroupName, Tasks: packTasks,
					})
				}
				packStageGroups = append(packStageGroups, workflow.PackStageGroup{
					Name: sg.Name, Description: sg.Description, Order: sg.Order,
					Mode: sg.Mode, Stages: packStages,
				})
			}

			pack.Workflows = append(pack.Workflows, workflow.PackWorkflow{
				Name: wf.Name, Description: wf.Description,
				StageGroups: packStageGroups, Hooks: packHooks,
			})
		}
	}

	// 收集所有引用的机器分组名称（包含阶段模板和工作流阶段中的分组）
	machineGroupNames := make(map[string]bool)
	for _, st := range pack.Stages {
		if st.MachineGroup != "" {
			machineGroupNames[st.MachineGroup] = true
		}
	}
	for _, pw := range pack.Workflows {
		for _, psg := range pw.StageGroups {
			for _, pws := range psg.Stages {
				if pws.MachineGroup != "" {
					machineGroupNames[pws.MachineGroup] = true
				}
			}
		}
	}
	for name := range machineGroupNames {
		pmg := workflow.PackMachineGroup{Name: name}
		var mg machine.MachineGroup
		if err := s.db.Where("name = ?", name).First(&mg).Error; err == nil {
			pmg.Description = mg.Description
		}
		pack.MachineGroups = append(pack.MachineGroups, pmg)
	}

	return pack, nil
}

// ImportSolutionLibrary 导入 orbit-pack YAML 为方案（只存储原始数据，不创建关联记录）
func (s *Service) ImportSolutionLibrary(pack *workflow.OrbitPack) (*workflow.SolutionLibrary, error) {
	if pack.Metadata.Name == "" {
		return nil, fmt.Errorf("方案名称不能为空")
	}

	// 检查名称唯一性
	var count int64
	s.db.Model(&workflow.SolutionLibrary{}).Where("name = ?", pack.Metadata.Name).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("方案名称「%s」已存在", pack.Metadata.Name)
	}

	// 序列化 pack 为 JSON
	packJSON, err := json.Marshal(pack)
	if err != nil {
		return nil, fmt.Errorf("序列化方案数据失败: %v", err)
	}

	// 创建方案记录，只存储元数据和原始 pack 数据
	solution := &workflow.SolutionLibrary{
		Name:        pack.Metadata.Name,
		Description: pack.Metadata.Description,
		Category:    pack.Metadata.Category,
		Version:     pack.Metadata.Version,
		Author:      pack.Metadata.Author,
		PackData:    string(packJSON),
		// 统计字段从 pack 中计算
		StageCount:    len(pack.Stages),
		VariableCount: len(pack.GlobalVariables),
		HookCount:     len(pack.Hooks),
		TemplateCount: len(pack.WorkflowTemplates),
		WorkflowCount: len(pack.Workflows),
	}

	// 保存方案记录
	if err := s.db.Create(solution).Error; err != nil {
		return nil, err
	}

	return solution, nil
}

// updateCounts 更新统计字段
func (s *Service) updateCounts(solution *workflow.SolutionLibrary) {
	solution.StageCount = len(parseIDs(solution.StageIDs))
	solution.VariableCount = len(parseIDs(solution.VariableIDs))
	solution.HookCount = len(parseIDs(solution.HookIDs))
	solution.TemplateCount = len(parseIDs(solution.TemplateIDs))
	solution.FileCount = len(parseIDs(solution.FileIDs))
	solution.WorkflowCount = len(parseIDs(solution.WorkflowIDs))
}

// parseIDs 解析JSON数组为uint切片
func parseIDs(jsonStr string) []uint {
	if jsonStr == "" || jsonStr == "null" {
		return nil
	}
	var ids []uint
	json.Unmarshal([]byte(jsonStr), &ids)
	return ids
}

// toJSON 将uint切片转为JSON字符串
func toJSON(ids []uint) string {
	if len(ids) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

// ==================== 冲突检测 ====================

// ConflictItem 冲突项
type ConflictItem struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	ExistingSource string `json:"existing_source"`
}

// ==================== 应用方案 ====================

// ImportSummary 导入摘要
type ImportSummary struct {
	StageCount    int `json:"stage_count"`
	VariableCount int `json:"variable_count"`
	HookCount     int `json:"hook_count"`
	TemplateCount int `json:"template_count"`
	FileCount     int `json:"file_count"`
	WorkflowCount int `json:"workflow_count"`
}

// CheckApplyConflicts 检测方案应用时的冲突（基于 pack_data）
func (s *Service) CheckApplyConflicts(solutionID uint) ([]ConflictItem, ImportSummary, error) {
	var solution workflow.SolutionLibrary
	if err := s.db.First(&solution, solutionID).Error; err != nil {
		return nil, ImportSummary{}, fmt.Errorf("方案不存在")
	}

	if solution.PackData == "" {
		return nil, ImportSummary{}, fmt.Errorf("方案未包含导入数据，请重新导入")
	}

	var pack workflow.OrbitPack
	if err := json.Unmarshal([]byte(solution.PackData), &pack); err != nil {
		return nil, ImportSummary{}, fmt.Errorf("解析方案数据失败: %v", err)
	}

	var conflicts []ConflictItem
	summary := ImportSummary{}

	// 检测阶段模板冲突
	summary.StageCount = len(pack.Stages)
	for _, st := range pack.Stages {
		var count int64
		s.db.Model(&workflow.StageTemplate{}).Where("name = ?", st.Name).Count(&count)
		if count > 0 {
			var existing workflow.StageTemplate
			s.db.Where("name = ?", st.Name).First(&existing)
			conflicts = append(conflicts, ConflictItem{
				Type: "stages", Name: st.Name, ExistingSource: existing.Source,
			})
		}
	}

	// 检测全局变量冲突
	summary.VariableCount = len(pack.GlobalVariables)
	for _, v := range pack.GlobalVariables {
		var count int64
		s.db.Model(&workflow.GlobalVariable{}).Where("`key` = ?", v.Key).Count(&count)
		if count > 0 {
			var existing workflow.GlobalVariable
			s.db.Where("`key` = ?", v.Key).First(&existing)
			conflicts = append(conflicts, ConflictItem{
				Type: "variables", Name: v.Key, ExistingSource: existing.Source,
			})
		}
	}

	// 检测钩子模板冲突
	summary.HookCount = len(pack.Hooks)
	for _, h := range pack.Hooks {
		var count int64
		s.db.Model(&workflow.HookTemplate{}).Where("name = ?", h.Name).Count(&count)
		if count > 0 {
			var existing workflow.HookTemplate
			s.db.Where("name = ?", h.Name).First(&existing)
			conflicts = append(conflicts, ConflictItem{
				Type: "hooks", Name: h.Name, ExistingSource: existing.Source,
			})
		}
	}

	// 检测配置模板冲突
	summary.TemplateCount = len(pack.WorkflowTemplates)
	for _, t := range pack.WorkflowTemplates {
		var count int64
		s.db.Model(&workflow.WorkflowTemplate{}).Where("name = ?", t.Name).Count(&count)
		if count > 0 {
			var existing workflow.WorkflowTemplate
			s.db.Where("name = ?", t.Name).First(&existing)
			conflicts = append(conflicts, ConflictItem{
				Type: "templates", Name: t.Name, ExistingSource: existing.Source,
			})
		}
	}

	// 检测工作流冲突
	summary.WorkflowCount = len(pack.Workflows)
	for _, wf := range pack.Workflows {
		var count int64
		s.db.Model(&workflow.Workflow{}).Where("name = ?", wf.Name).Count(&count)
		if count > 0 {
			conflicts = append(conflicts, ConflictItem{
				Type: "workflows", Name: wf.Name, ExistingSource: "系统",
			})
		}
	}

	return conflicts, summary, nil
}

// ApplySolutionLibraryWithDecisions 根据用户决策应用方案（从 pack_data 创建记录）
func (s *Service) ApplySolutionLibraryWithDecisions(solutionID uint, decisions map[string]map[string]string, variableValues map[string]string, machineGroupMachines map[string][]uint) error {
	var solution workflow.SolutionLibrary
	if err := s.db.First(&solution, solutionID).Error; err != nil {
		return fmt.Errorf("方案不存在")
	}

	if solution.PackData == "" {
		return fmt.Errorf("方案未包含导入数据，请重新导入")
	}

	var pack workflow.OrbitPack
	if err := json.Unmarshal([]byte(solution.PackData), &pack); err != nil {
		return fmt.Errorf("解析方案数据失败: %v", err)
	}

	// 先处理机器分组（创建/更新 + 关联机器），后续阶段和工作流按名称查找
	if machineGroupMachines != nil {
		for _, pmg := range pack.MachineGroups {
			machineIDs := machineGroupMachines[pmg.Name]
			var mg machine.MachineGroup
			err := s.db.Where("name = ?", pmg.Name).First(&mg).Error
			if err != nil {
				// 不存在则创建
				mg = machine.MachineGroup{
					Name:        pmg.Name,
					Description: pmg.Description,
				}
				s.db.Create(&mg)
			}
			// 更新关联的机器
			if len(machineIDs) > 0 {
				var machines []machine.Machine
				s.db.Where("id IN ?", machineIDs).Find(&machines)
				s.db.Model(&mg).Association("Machines").Replace(machines)
			}
		}
	}

	// 导入全局变量
	var variableIDs []uint
	for _, v := range pack.GlobalVariables {
		// 检查冲突处理
		if decisions["variables"] != nil && decisions["variables"][v.Key] == "skip" {
			// 查找已有的变量ID
			var existing workflow.GlobalVariable
			if err := s.db.Where("`key` = ?", v.Key).First(&existing).Error; err == nil {
				variableIDs = append(variableIDs, existing.ID)
				continue
			}
		}
		if decisions["variables"] != nil && decisions["variables"][v.Key] == "overwrite" {
			s.db.Unscoped().Where("`key` = ?", v.Key).Delete(&workflow.GlobalVariable{})
		}
		// 使用用户修改后的值（如果有）
		val := v.Value
		if variableValues != nil && variableValues[v.Key] != "" {
			val = variableValues[v.Key]
		}
		gv := workflow.GlobalVariable{
			Key: v.Key, Type: v.Type, Value: val,
			Description: v.Description, Group: v.Group,
			Source: pack.Metadata.Name,
		}
		s.db.Create(&gv)
		variableIDs = append(variableIDs, gv.ID)
	}

	// 导入钩子模板
	var hookIDs []uint
	for _, h := range pack.Hooks {
		if decisions["hooks"] != nil && decisions["hooks"][h.Name] == "skip" {
			var existing workflow.HookTemplate
			if err := s.db.Where("name = ?", h.Name).First(&existing).Error; err == nil {
				hookIDs = append(hookIDs, existing.ID)
				continue
			}
		}
		if decisions["hooks"] != nil && decisions["hooks"][h.Name] == "overwrite" {
			s.db.Unscoped().Where("name = ?", h.Name).Delete(&workflow.HookTemplate{})
		}
		ht := workflow.HookTemplate{
			Name: h.Name, Description: h.Description, Module: h.Module,
			Params: h.Params, Timeout: h.Timeout, IgnoreErrors: h.IgnoreErrors,
			Retries: h.Retries, Delay: h.Delay,
			Source: pack.Metadata.Name,
		}
		s.db.Create(&ht)
		hookIDs = append(hookIDs, ht.ID)
	}

	// 导入模板文件
	var templateIDs []uint
	for _, t := range pack.WorkflowTemplates {
		if decisions["templates"] != nil && decisions["templates"][t.Name] == "skip" {
			var existing workflow.WorkflowTemplate
			if err := s.db.Where("name = ?", t.Name).First(&existing).Error; err == nil {
				templateIDs = append(templateIDs, existing.ID)
				continue
			}
		}
		if decisions["templates"] != nil && decisions["templates"][t.Name] == "overwrite" {
			s.db.Unscoped().Where("name = ?", t.Name).Delete(&workflow.WorkflowTemplate{})
		}
		wt := workflow.WorkflowTemplate{
			Name: t.Name, Description: t.Description, Content: t.Content,
			Source: pack.Metadata.Name,
		}
		s.db.Create(&wt)
		templateIDs = append(templateIDs, wt.ID)
	}

	// 导入阶段模板
	var stageIDs []uint
	for _, st := range pack.Stages {
		if decisions["stages"] != nil && decisions["stages"][st.Name] == "skip" {
			var existing workflow.StageTemplate
			if err := s.db.Where("name = ?", st.Name).First(&existing).Error; err == nil {
				stageIDs = append(stageIDs, existing.ID)
				continue
			}
		}
		if decisions["stages"] != nil && decisions["stages"][st.Name] == "overwrite" {
			s.db.Unscoped().Where("name = ?", st.Name).Delete(&workflow.StageTemplate{})
		}
		var machineGroupID uint
		// 机器分组已在前面处理完毕，直接按名称查找
		if st.MachineGroup != "" {
			var mg machine.MachineGroup
			if err := s.db.Where("name = ?", st.MachineGroup).First(&mg).Error; err == nil {
				machineGroupID = mg.ID
			}
		}
		var tasks []workflow.StageTask
		for _, t := range st.Tasks {
			var hooksJSON string
			if len(t.Hooks) > 0 {
				b, _ := json.Marshal(t.Hooks)
				hooksJSON = string(b)
			}
			tasks = append(tasks, workflow.StageTask{
				Ref: t.Ref, Name: t.Name, Module: t.Module, Order: t.Order,
				Params: t.Params, When: t.When, Hooks: hooksJSON,
				Loop: t.Loop, Timeout: t.Timeout, IgnoreErrors: t.IgnoreErrors,
				Retries: t.Retries, Delay: t.Delay, Register: t.Register,
			})
		}
		tasksJSON, _ := json.Marshal(tasks)
		ver := generateVersionString()
		stage := workflow.StageTemplate{
			Name:           st.Name,
			Description:    st.Description,
			MachineGroupID: machineGroupID,
			Tasks:          string(tasksJSON),
			Version:        ver,
			Source:         pack.Metadata.Name,
		}
		s.db.Create(&stage)
		// 创建版本记录（用于版本回滚）
		version := workflow.StageTemplateVersion{
			TemplateID:     stage.ID,
			Version:        ver,
			Name:           st.Name,
			Description:    st.Description,
			MachineGroupID: machineGroupID,
			Tasks:          string(tasksJSON),
			ChangeNote:     "方案导入初始版本",
		}
		s.db.Create(&version)
		stageIDs = append(stageIDs, stage.ID)
	}

	// 导入工作流
	var workflowIDs []uint
	for _, pw := range pack.Workflows {
		if decisions["workflows"] != nil && decisions["workflows"][pw.Name] == "skip" {
			var existing workflow.Workflow
			if err := s.db.Where("name = ?", pw.Name).First(&existing).Error; err == nil {
				workflowIDs = append(workflowIDs, existing.ID)
				continue
			}
		}
		if decisions["workflows"] != nil && decisions["workflows"][pw.Name] == "overwrite" {
			var oldWf workflow.Workflow
			if err := s.db.Where("name = ?", pw.Name).First(&oldWf).Error; err == nil {
				s.db.Transaction(func(tx *gorm.DB) error {
					// 删除执行记录（task → stage → stage_group → execution）
					var stageExecIDs []uint
					tx.Model(&workflow.WorkflowStageExecution{}).
						Where("stage_group_execution_id IN (SELECT id FROM workflow_stage_group_executions WHERE execution_id IN (SELECT id FROM workflow_executions WHERE workflow_id = ?))", oldWf.ID).
						Pluck("id", &stageExecIDs)
					if len(stageExecIDs) > 0 {
						tx.Where("stage_execution_id IN ?", stageExecIDs).Delete(&workflow.WorkflowTaskExecution{})
					}
					var stageGroupExecIDs []uint
					tx.Model(&workflow.WorkflowStageGroupExecution{}).
						Where("execution_id IN (SELECT id FROM workflow_executions WHERE workflow_id = ?)", oldWf.ID).
						Pluck("id", &stageGroupExecIDs)
					if len(stageGroupExecIDs) > 0 {
						tx.Where("stage_group_execution_id IN ?", stageGroupExecIDs).Delete(&workflow.WorkflowStageExecution{})
					}
					tx.Where("execution_id IN (SELECT id FROM workflow_executions WHERE workflow_id = ?)", oldWf.ID).
						Delete(&workflow.WorkflowStageGroupExecution{})
					tx.Where("workflow_id = ?", oldWf.ID).Delete(&workflow.WorkflowExecution{})

					// 删除 tasks（通过 stages 关联）
					var groupIDs []uint
					tx.Model(&workflow.WorkflowStageGroup{}).Where("workflow_id = ?", oldWf.ID).Pluck("id", &groupIDs)
					if len(groupIDs) > 0 {
						var sIDs []uint
						tx.Model(&workflow.WorkflowStage{}).Where("stage_group_id IN ?", groupIDs).Pluck("id", &sIDs)
						if len(sIDs) > 0 {
							tx.Where("stage_id IN ?", sIDs).Delete(&workflow.WorkflowTask{})
						}
						tx.Where("stage_group_id IN ?", groupIDs).Delete(&workflow.WorkflowStage{})
					}
					tx.Where("workflow_id = ?", oldWf.ID).Delete(&workflow.WorkflowStageGroup{})
					tx.Where("workflow_id = ?", oldWf.ID).Delete(&workflow.WorkflowHook{})
					tx.Delete(&oldWf)
					return nil
				})
			}
		}
		wf := workflow.Workflow{
			Name:        pw.Name,
			Description: pw.Description,
			CreatedBy:   "import",
		}
		s.db.Create(&wf)

		// 导入工作流钩子
		for _, ph := range pw.Hooks {
			wh := workflow.WorkflowHook{
				WorkflowID:   wf.ID,
				Name:         ph.Name,
				Module:       ph.Module,
				Params:       ph.Params,
				Timeout:      ph.Timeout,
				IgnoreErrors: ph.IgnoreErrors,
				Retries:      ph.Retries,
				Delay:        ph.Delay,
			}
			s.db.Create(&wh)
		}

		// 导入阶段组
		for _, psg := range pw.StageGroups {
			sg := workflow.WorkflowStageGroup{
				WorkflowID:  wf.ID,
				Name:        psg.Name,
				Description: psg.Description,
				Order:       psg.Order,
				Mode:        psg.Mode,
			}
			s.db.Create(&sg)

			for _, pws := range psg.Stages {
				var machineGroupID uint
				// 机器分组已在前面处理完毕，直接按名称查找
				if pws.MachineGroup != "" {
					var mg machine.MachineGroup
					if err := s.db.Where("name = ?", pws.MachineGroup).First(&mg).Error; err == nil {
						machineGroupID = mg.ID
					}
				}
				st := workflow.WorkflowStage{
					StageGroupID:   sg.ID,
					Name:           pws.Name,
					Description:    pws.Description,
					Order:          pws.Order,
					MachineGroupID: machineGroupID,
				}
				s.db.Create(&st)

				for _, pwt := range pws.Tasks {
					var hooksJSON string
					if len(pwt.Hooks) > 0 {
						b, _ := json.Marshal(pwt.Hooks)
						hooksJSON = string(b)
					}
					task := workflow.WorkflowTask{
						StageID:      st.ID,
						Ref:          pwt.Ref,
						Name:         pwt.Name,
						Module:       pwt.Module,
						Params:       pwt.Params,
						Order:        pwt.Order,
						When:         pwt.When,
						Hooks:        hooksJSON,
						Loop:         pwt.Loop,
						Timeout:      pwt.Timeout,
						IgnoreErrors: pwt.IgnoreErrors,
						Retries:      pwt.Retries,
						Delay:        pwt.Delay,
						Register:     pwt.Register,
					}
					s.db.Create(&task)
				}
			}
		}
		workflowIDs = append(workflowIDs, wf.ID)
	}

	// 更新方案记录：写入关联ID，清空 pack_data
	solution.StageIDs = toJSON(stageIDs)
	solution.VariableIDs = toJSON(variableIDs)
	solution.HookIDs = toJSON(hookIDs)
	solution.TemplateIDs = toJSON(templateIDs)
	solution.WorkflowIDs = toJSON(workflowIDs)
	solution.PackData = "" // 应用后清空原始数据

	// 更新统计
	s.updateCounts(&solution)

	// 保存更新
	return s.db.Save(&solution).Error
}
