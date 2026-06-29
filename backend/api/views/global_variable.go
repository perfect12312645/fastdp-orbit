package views

import (
	"net/http"
	"strconv"

	"fastdp-orbit/backend/models/workflow"

	"github.com/gin-gonic/gin"
)

// ==================== 请求结构 ====================

type CreateGlobalVariableRequest struct {
	Key         string `json:"key" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Value       string `json:"value"`
	Description string `json:"description"`
	Group       string `json:"group"`
}

type UpdateGlobalVariableRequest struct {
	Key         string `json:"key" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Value       string `json:"value"`
	Description string `json:"description"`
	Group       string `json:"group"`
}

// ==================== Handlers ====================

// ListGlobalVariables 获取所有全局变量
func ListGlobalVariables(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	packageGroup := c.Query("source")
	vars, err := WorkflowService.ListGlobalVariables(packageGroup)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	if vars == nil {
		vars = []workflow.GlobalVariable{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": vars})
}

// GetGlobalVariable 获取全局变量详情
func GetGlobalVariable(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	v, err := WorkflowService.GetGlobalVariable(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "全局变量不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": v})
}

// CreateGlobalVariable 创建全局变量
func CreateGlobalVariable(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	var req CreateGlobalVariableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	v := &workflow.GlobalVariable{
		Key:         req.Key,
		Type:        req.Type,
		Value:       req.Value,
		Description: req.Description,
		Group:       req.Group,
	}
	if err := WorkflowService.CreateGlobalVariable(v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": v})
}

// UpdateGlobalVariable 更新全局变量
func UpdateGlobalVariable(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	var req UpdateGlobalVariableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	v := &workflow.GlobalVariable{
		Key:         req.Key,
		Type:        req.Type,
		Value:       req.Value,
		Description: req.Description,
		Group:       req.Group,
	}
	if err := WorkflowService.UpdateGlobalVariable(uint(id), v); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DeleteGlobalVariable 删除全局变量
func DeleteGlobalVariable(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := WorkflowService.DeleteGlobalVariable(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
