package workflow

import (
	"encoding/json"
	"fmt"

	"fastdp-orbit/backend/models/machine"
	"fastdp-orbit/backend/models/workflow"
)

// ==================== SolutionLibrary CRUD ====================

// ListSolutionLibrarys 获取模板包列表
func (s *Service) ListSolutionLibrarys(category string) ([]workflow.SolutionLibrary, error) {
	var packages []workflow.SolutionLibrary
	query := s.db.Order("created_at DESC")
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if err := query.Find(&packages).Error; err != nil {
		return nil, err
	}
	return packages, nil
}

// GetSolutionLibrary 获取模板包详情
func (s *Service) GetSolutionLibrary(id uint) (*workflow.SolutionLibrary, error) {
	var pkg workflow.SolutionLibrary
	if err := s.db.First(&pkg, id).Error; err != nil {
		return nil, err
	}
	return &pkg, nil
}

// CreateSolutionLibrary 创建模板包
func (s *Service) CreateSolutionLibrary(pkg *workflow.SolutionLibrary) error {
	if pkg.Name == "" {
		return fmt.Errorf("模板包名称不能为空")
	}
	// 检查名称唯一性
	var count int64
	s.db.Model(&workflow.SolutionLibrary{}).Where("name = ?", pkg.Name).Count(&count)
	if count > 0 {
		return fmt.Errorf("模板包名称「%s」已存在", pkg.Name)
	}
	return s.db.Create(pkg).Error
}

// DeleteSolutionLibrary 删除模板包（同时删除关联的所有内容）
func (s *Service) DeleteSolutionLibrary(id uint) error {
	var pkg workflow.SolutionLibrary
	if err := s.db.First(&pkg, id).Error; err != nil {
		return fmt.Errorf("模板包不存在")
	}

	// 删除该分组下的所有内容
	group := pkg.Name
	s.db.Where("package_group = ?", group).Delete(&workflow.StageTemplate{})
	s.db.Where("package_group = ?", group).Delete(&workflow.GlobalVariable{})
	s.db.Where("package_group = ?", group).Delete(&workflow.HookTemplate{})
	s.db.Where("package_group = ?", group).Delete(&workflow.WorkflowTemplate{})

	// 删除模板包本身
	return s.db.Delete(&pkg).Error
}

// ExportSolutionLibrary 导出模板包为 orbit-pack YAML
func (s *Service) ExportSolutionLibrary(id uint) (*workflow.OrbitPack, error) {
	var pkg workflow.SolutionLibrary
	if err := s.db.First(&pkg, id).Error; err != nil {
		return nil, fmt.Errorf("模板包不存在")
	}

	group := pkg.Name
	pack := &workflow.OrbitPack{
		APIVersion: "orbit/v1",
		Kind:       "SolutionLibrary",
	}
	pack.Metadata.Name = pkg.Name
	pack.Metadata.Description = pkg.Description
	pack.Metadata.Category = pkg.Category
	pack.Metadata.Version = pkg.Version
	pack.Metadata.Author = pkg.Author

	// 导出全局变量
	var vars []workflow.GlobalVariable
	s.db.Where("package_group = ?", group).Find(&vars)
	for _, v := range vars {
		pack.GlobalVariables = append(pack.GlobalVariables, workflow.PackGlobalVariable{
			Key: v.Key, Type: v.Type, Value: v.Value,
			Description: v.Description, Group: v.Group,
		})
	}

	// 导出钩子模板
	var hooks []workflow.HookTemplate
	s.db.Where("package_group = ?", group).Find(&hooks)
	for _, h := range hooks {
		pack.Hooks = append(pack.Hooks, workflow.PackHook{
			Name: h.Name, Description: h.Description, Module: h.Module,
			Params: h.Params, Timeout: h.Timeout, IgnoreErrors: h.IgnoreErrors,
			Retries: h.Retries, Delay: h.Delay,
		})
	}

	// 导出模板文件
	var templates []workflow.WorkflowTemplate
	s.db.Where("package_group = ?", group).Find(&templates)
	for _, t := range templates {
		pack.WorkflowTemplates = append(pack.WorkflowTemplates, workflow.PackWorkflowTemplate{
			Name: t.Name, Description: t.Description, Content: t.Content,
		})
	}

	// 导出阶段模板
	var stages []workflow.StageTemplate
	s.db.Where("package_group = ?", group).Find(&stages)
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
				packStage.Tasks = append(packStage.Tasks, workflow.PackTask{
					Ref: t.Ref, Name: t.Name, Module: t.Module, Order: t.Order,
					Params: t.Params, When: t.When, HookIDs: t.HookIDs,
					Loop: t.Loop, Timeout: t.Timeout, IgnoreErrors: t.IgnoreErrors,
					Retries: t.Retries, Delay: t.Delay, Register: t.Register,
				})
			}
		}
		pack.Stages = append(pack.Stages, packStage)
	}

	return pack, nil
}

// ImportSolutionLibrary 导入 orbit-pack YAML 为模板包
func (s *Service) ImportSolutionLibrary(pack *workflow.OrbitPack) (*workflow.SolutionLibrary, error) {
	if pack.Metadata.Name == "" {
		return nil, fmt.Errorf("模板包名称不能为空")
	}

	group := pack.Metadata.Name

	// 创建模板包记录
	pkg := &workflow.SolutionLibrary{
		Name:        pack.Metadata.Name,
		Description: pack.Metadata.Description,
		Category:    pack.Metadata.Category,
		Version:     pack.Metadata.Version,
		Author:      pack.Metadata.Author,
	}

	// 检查名称唯一性
	var count int64
	s.db.Model(&workflow.SolutionLibrary{}).Where("name = ?", pkg.Name).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("模板包名称「%s」已存在", pkg.Name)
	}

	// 导入全局变量
	for _, v := range pack.GlobalVariables {
		gv := workflow.GlobalVariable{
			Key: v.Key, Type: v.Type, Value: v.Value,
			Description: v.Description, Group: v.Group,
			Source: group,
		}
		s.db.Create(&gv)
		pkg.VariableCount++
	}

	// 导入钩子模板
	for _, h := range pack.Hooks {
		ht := workflow.HookTemplate{
			Name: h.Name, Description: h.Description, Module: h.Module,
			Params: h.Params, Timeout: h.Timeout, IgnoreErrors: h.IgnoreErrors,
			Retries: h.Retries, Delay: h.Delay,
			Source: group,
		}
		s.db.Create(&ht)
		pkg.HookCount++
	}

	// 导入模板文件
	for _, t := range pack.WorkflowTemplates {
		wt := workflow.WorkflowTemplate{
			Name: t.Name, Description: t.Description, Content: t.Content,
			Source: group,
		}
		s.db.Create(&wt)
		pkg.TemplateCount++
	}

	// 导入阶段模板
	for _, st := range pack.Stages {
		// 解析机器分组
		var machineGroupID uint
		if st.MachineGroup != "" {
			var mg machine.MachineGroup
			if err := s.db.Where("name = ?", st.MachineGroup).First(&mg).Error; err == nil {
				machineGroupID = mg.ID
			}
		}
		// 构建任务JSON
		var tasks []workflow.StageTask
		for _, t := range st.Tasks {
			tasks = append(tasks, workflow.StageTask{
				Ref: t.Ref, Name: t.Name, Module: t.Module, Order: t.Order,
				Params: t.Params, When: t.When, HookIDs: t.HookIDs,
				Loop: t.Loop, Timeout: t.Timeout, IgnoreErrors: t.IgnoreErrors,
				Retries: t.Retries, Delay: t.Delay, Register: t.Register,
			})
		}
		tasksJSON, _ := json.Marshal(tasks)
		stage := workflow.StageTemplate{
			Name:           st.Name,
			Description:    st.Description,
			MachineGroupID: machineGroupID,
			Tasks:          string(tasksJSON),
			Version:        "imported",
			Source:         group,
		}
		s.db.Create(&stage)
		pkg.StageCount++
	}

	// 保存模板包记录
	if err := s.db.Create(pkg).Error; err != nil {
		return nil, err
	}

	return pkg, nil
}
