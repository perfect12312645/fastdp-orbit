package workflow

import (
	"fmt"

	"fastdp-orbit/backend/models/workflow"
)

// ==================== HookTemplate CRUD ====================

// ListHookTemplates 获取钩子模板（支持按分组过滤）
func (s *Service) ListHookTemplates(packageGroup string) ([]workflow.HookTemplate, error) {
	var templates []workflow.HookTemplate
	query := s.db.Order("created_at DESC")
	if packageGroup != "" {
		query = query.Where("package_group = ?", packageGroup)
	}
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// GetHookTemplate 获取钩子模板详情
func (s *Service) GetHookTemplate(id uint) (*workflow.HookTemplate, error) {
	var t workflow.HookTemplate
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateHookTemplate 创建钩子模板
func (s *Service) CreateHookTemplate(t *workflow.HookTemplate) error {
	if t.Name == "" {
		return fmt.Errorf("钩子名称不能为空")
	}
	if t.Module == "" {
		return fmt.Errorf("模块类型不能为空")
	}
	// 检查名称唯一性（同分组内）
	var count int64
	s.db.Model(&workflow.HookTemplate{}).Where("name = ? AND package_group = ?", t.Name, t.Source).Count(&count)
	if count > 0 {
		return fmt.Errorf("钩子名称「%s」在当前分组已存在", t.Name)
	}
	return s.db.Create(t).Error
}

// UpdateHookTemplate 更新钩子模板
func (s *Service) UpdateHookTemplate(id uint, t *workflow.HookTemplate) error {
	if t.Name == "" {
		return fmt.Errorf("钩子名称不能为空")
	}
	if t.Module == "" {
		return fmt.Errorf("模块类型不能为空")
	}
	// 检查名称唯一性（同分组内，排除自身）
	var count int64
	s.db.Model(&workflow.HookTemplate{}).Where("name = ? AND package_group = ? AND id != ?", t.Name, t.Source, id).Count(&count)
	if count > 0 {
		return fmt.Errorf("钩子名称「%s」在当前分组已存在", t.Name)
	}
	return s.db.Model(&workflow.HookTemplate{}).Where("id = ?", id).Updates(t).Error
}

// DeleteHookTemplate 删除钩子模板
func (s *Service) DeleteHookTemplate(id uint) error {
	return s.db.Delete(&workflow.HookTemplate{}, id).Error
}

// GetHookTemplateByName 根据名称获取钩子模板
func (s *Service) GetHookTemplateByName(name string) (*workflow.HookTemplate, error) {
	var t workflow.HookTemplate
	if err := s.db.Where("name = ?", name).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetHookTemplatesByNames 根据名称列表批量获取钩子模板
func (s *Service) GetHookTemplatesByNames(names []string) ([]workflow.HookTemplate, error) {
	var templates []workflow.HookTemplate
	if err := s.db.Where("name IN ?", names).Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}
