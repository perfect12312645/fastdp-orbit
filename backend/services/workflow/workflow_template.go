package workflow

import (
	"fmt"

	"fastdp-orbit/backend/models/workflow"
)

// ==================== WorkflowTemplate CRUD ====================

// ListWorkflowTemplates 获取所有工作流模板文件
func (s *Service) ListWorkflowTemplates() ([]workflow.WorkflowTemplate, error) {
	var templates []workflow.WorkflowTemplate
	if err := s.db.Order("created_at DESC").Find(&templates).Error; err != nil {
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
	// 检查名称唯一性
	var count int64
	s.db.Model(&workflow.WorkflowTemplate{}).Where("name = ?", t.Name).Count(&count)
	if count > 0 {
		return fmt.Errorf("模板名称「%s」已存在", t.Name)
	}
	return s.db.Create(t).Error
}

// UpdateWorkflowTemplate 更新工作流模板文件
func (s *Service) UpdateWorkflowTemplate(id uint, t *workflow.WorkflowTemplate) error {
	if t.Name == "" {
		return fmt.Errorf("模板名称不能为空")
	}
	// 检查名称唯一性（排除自身）
	var count int64
	s.db.Model(&workflow.WorkflowTemplate{}).Where("name = ? AND id != ?", t.Name, id).Count(&count)
	if count > 0 {
		return fmt.Errorf("模板名称「%s」已存在", t.Name)
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
