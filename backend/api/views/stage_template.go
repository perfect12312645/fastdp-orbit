package views

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"fastdp-orbit/backend/models/workflow"
	"fastdp-orbit/backend/pkg/errs"

	"github.com/gin-gonic/gin"
)

// friendlyParamError 将 Gin 参数校验错误转为用户友好的中文提示
func friendlyParamError(err error) string {
	msg := err.Error()
	// 常见字段的中文映射（不需要解析 validator error 结构体）
	switch {
	case strings.Contains(msg, "MachineGroupID"):
		return "请选择机器分组"
	case strings.Contains(msg, "Name"):
		return "请输入名称"
	}
	// fallback: 去掉 Go 内部错误信息细节
	if idx := strings.Index(msg, "Error:"); idx >= 0 {
		msg = strings.TrimSpace(msg[:idx])
	}
	return msg
}

// ==================== 请求结构 ====================

type CreateStageTemplateRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	MachineGroupID uint   `json:"machine_group_id" binding:"required"`
	Tasks          string `json:"tasks"`
	ChangeNote     string `json:"change_note"`
}

type UpdateStageTemplateRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	MachineGroupID uint   `json:"machine_group_id" binding:"required"`
	Tasks          string `json:"tasks"`
	ChangeNote     string `json:"change_note" binding:"required"`
}

type RollbackStageTemplateRequest struct {
	Version string `json:"version" binding:"required"`
}

// respondError 统一错误响应：区分业务错误和系统错误
func respondError(c *gin.Context, err error) {
	var bizErr *errs.BizError
	if errors.As(err, &bizErr) {
		c.JSON(bizErr.HTTPStatus(), gin.H{"code": bizErr.Code, "message": bizErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "系统内部错误"})
}

// ==================== Handlers ====================

// ListStageTemplates 获取所有阶段模板
func ListStageTemplates(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	templates, err := WorkflowService.ListStageTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "系统内部错误"})
		return
	}
	if templates == nil {
		templates = []workflow.StageTemplate{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": templates})
}

// CreateStageTemplate 创建阶段模板
func CreateStageTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	var req CreateStageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": errs.CodeParamInvalid, "message": friendlyParamError(err)})
		return
	}

	t := &workflow.StageTemplate{
		Name:           req.Name,
		Description:    req.Description,
		MachineGroupID: req.MachineGroupID,
		Tasks:          req.Tasks,
	}
	if err := WorkflowService.CreateStageTemplate(t); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": t})
}

// GetStageTemplate 获取阶段模板详情
func GetStageTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": errs.CodeIDFormat, "message": "ID格式错误"})
		return
	}

	t, err := WorkflowService.GetStageTemplate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": errs.CodeStageTemplateNotFound, "message": "阶段模板不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": t})
}

// UpdateStageTemplate 更新阶段模板（强制生成新版本）
func UpdateStageTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": errs.CodeIDFormat, "message": "ID格式错误"})
		return
	}

	var req UpdateStageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": errs.CodeParamInvalid, "message": friendlyParamError(err)})
		return
	}

	t := &workflow.StageTemplate{
		Name:           req.Name,
		Description:    req.Description,
		MachineGroupID: req.MachineGroupID,
		Tasks:          req.Tasks,
	}
	if err := WorkflowService.UpdateStageTemplate(uint(id), t, req.ChangeNote); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DeleteStageTemplate 删除阶段模板
func DeleteStageTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": errs.CodeIDFormat, "message": "ID格式错误"})
		return
	}

	if err := WorkflowService.DeleteStageTemplate(uint(id)); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// ==================== 版本管理 ====================

// ListStageTemplateVersions 获取阶段模板的版本历史
func ListStageTemplateVersions(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": errs.CodeIDFormat, "message": "ID格式错误"})
		return
	}

	versions, err := WorkflowService.ListStageTemplateVersions(uint(id))
	if err != nil {
		respondError(c, err)
		return
	}
	if versions == nil {
		versions = []workflow.StageTemplateVersion{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": versions})
}

// RollbackStageTemplate 回滚到指定版本
func RollbackStageTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": errs.CodeIDFormat, "message": "ID格式错误"})
		return
	}

	var req RollbackStageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": errs.CodeParamInvalid, "message": friendlyParamError(err)})
		return
	}

	if err := WorkflowService.RollbackStageTemplate(uint(id), req.Version); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
