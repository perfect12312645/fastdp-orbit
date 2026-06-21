package views

import (
	"net/http"
	"strconv"

	"fastdp-orbit/backend/models/workflow"

	"github.com/gin-gonic/gin"
)

// ==================== 请求结构 ====================

type CreateStageTemplateRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	MachineGroupID uint   `json:"machine_group_id"`
	Tasks          string `json:"tasks"`
	ChangeNote     string `json:"change_note"`
}

type UpdateStageTemplateRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	MachineGroupID uint   `json:"machine_group_id"`
	Tasks          string `json:"tasks"`
	ChangeNote     string `json:"change_note" binding:"required"`
}

type RollbackStageTemplateRequest struct {
	Version int `json:"version" binding:"required"`
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	t := &workflow.StageTemplate{
		Name:           req.Name,
		Description:    req.Description,
		MachineGroupID: req.MachineGroupID,
		Tasks:          req.Tasks,
	}
	if err := WorkflowService.CreateStageTemplate(t); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "创建失败: " + err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	t, err := WorkflowService.GetStageTemplate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "阶段模板不存在"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	var req UpdateStageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	t := &workflow.StageTemplate{
		Name:           req.Name,
		Description:    req.Description,
		MachineGroupID: req.MachineGroupID,
		Tasks:          req.Tasks,
	}
	if err := WorkflowService.UpdateStageTemplate(uint(id), t, req.ChangeNote); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "更新失败: " + err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := WorkflowService.DeleteStageTemplate(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "删除失败: " + err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	versions, err := WorkflowService.ListStageTemplateVersions(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	var req RollbackStageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	if err := WorkflowService.RollbackStageTemplate(uint(id), req.Version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "回滚失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
