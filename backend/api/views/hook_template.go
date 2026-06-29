package views

import (
	"net/http"
	"strconv"

	"fastdp-orbit/backend/models/workflow"

	"github.com/gin-gonic/gin"
)

// ==================== 请求结构 ====================

type CreateHookTemplateRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	Module       string `json:"module" binding:"required"`
	Params       string `json:"params"`
	Timeout      int    `json:"timeout"`
	IgnoreErrors bool   `json:"ignore_errors"`
	Retries      int    `json:"retries"`
	Delay        int    `json:"delay"`
}

type UpdateHookTemplateRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	Module       string `json:"module" binding:"required"`
	Params       string `json:"params"`
	Timeout      int    `json:"timeout"`
	IgnoreErrors bool   `json:"ignore_errors"`
	Retries      int    `json:"retries"`
	Delay        int    `json:"delay"`
}

// ==================== Handlers ====================

// ListHookTemplates 获取所有钩子模板
func ListHookTemplates(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	packageGroup := c.Query("source")
	templates, err := WorkflowService.ListHookTemplates(packageGroup)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "系统内部错误"})
		return
	}
	if templates == nil {
		templates = []workflow.HookTemplate{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": templates})
}

// CreateHookTemplate 创建钩子模板
func CreateHookTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	var req CreateHookTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	t := &workflow.HookTemplate{
		Name:         req.Name,
		Description:  req.Description,
		Module:       req.Module,
		Params:       req.Params,
		Timeout:      req.Timeout,
		IgnoreErrors: req.IgnoreErrors,
		Retries:      req.Retries,
		Delay:        req.Delay,
	}
	if err := WorkflowService.CreateHookTemplate(t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": t})
}

// GetHookTemplate 获取钩子模板详情
func GetHookTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	t, err := WorkflowService.GetHookTemplate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "钩子模板不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": t})
}

// UpdateHookTemplate 更新钩子模板
func UpdateHookTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	var req UpdateHookTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	t := &workflow.HookTemplate{
		Name:         req.Name,
		Description:  req.Description,
		Module:       req.Module,
		Params:       req.Params,
		Timeout:      req.Timeout,
		IgnoreErrors: req.IgnoreErrors,
		Retries:      req.Retries,
		Delay:        req.Delay,
	}
	if err := WorkflowService.UpdateHookTemplate(uint(id), t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DeleteHookTemplate 删除钩子模板
func DeleteHookTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := WorkflowService.DeleteHookTemplate(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
