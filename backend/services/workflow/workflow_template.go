package workflow

import (
	"fmt"

	"fastdp-orbit/backend/models/workflow"
)

// ==================== WorkflowTemplate CRUD ====================

// ListWorkflowTemplates 获取工作流模板文件（支持按分组过滤）
func (s *Service) ListWorkflowTemplates(packageGroup string) ([]workflow.WorkflowTemplate, error) {
	var templates []workflow.WorkflowTemplate
	query := s.db.Order("created_at DESC")
	if packageGroup != "" {
		query = query.Where("package_group = ?", packageGroup)
	}
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// GetWorkflowTemplate 获取工作流模板文件详情
func (s *Service) GetWorkflowTemplate(id uint) (*workflow.WorkflowTemplate, error) {
	var t workflow.WorkflowTemplate
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateWorkflowTemplate 创建工作流模板文件
func (s *Service) CreateWorkflowTemplate(t *workflow.WorkflowTemplate) error {
	if t.Name == "" {
		return fmt.Errorf("模板名称不能为空")
	}
	// 检查名称唯一性（同分组内）
	var count int64
	s.db.Model(&workflow.WorkflowTemplate{}).Where("name = ? AND package_group = ?", t.Name, t.Source).Count(&count)
	if count > 0 {
		return fmt.Errorf("模板名称「%s」在当前分组已存在", t.Name)
	}
	return s.db.Create(t).Error
}

// UpdateWorkflowTemplate 更新工作流模板文件
func (s *Service) UpdateWorkflowTemplate(id uint, t *workflow.WorkflowTemplate) error {
	if t.Name == "" {
		return fmt.Errorf("模板名称不能为空")
	}
	// 检查名称唯一性（同分组内，排除自身）
	var count int64
	s.db.Model(&workflow.WorkflowTemplate{}).Where("name = ? AND package_group = ? AND id != ?", t.Name, t.Source, id).Count(&count)
	if count > 0 {
		return fmt.Errorf("模板名称「%s」在当前分组已存在", t.Name)
	}
	return s.db.Model(&workflow.WorkflowTemplate{}).Where("id = ?", id).Updates(t).Error
}

// DeleteWorkflowTemplate 删除工作流模板文件
func (s *Service) DeleteWorkflowTemplate(id uint) error {
	return s.db.Delete(&workflow.WorkflowTemplate{}, id).Error
}

// GetWorkflowTemplateByName 根据名称获取工作流模板文件
func (s *Service) GetWorkflowTemplateByName(name string) (*workflow.WorkflowTemplate, error) {
	var t workflow.WorkflowTemplate
	if err := s.db.Where("name = ?", name).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}
