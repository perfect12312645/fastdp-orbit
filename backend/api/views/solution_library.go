package views

import (
	"net/http"
	"strconv"

	"fastdp-orbit/backend/models/workflow"

	"github.com/gin-gonic/gin"
)

// ==================== 请求结构 ====================

type CreateSolutionLibraryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Version     string `json:"version"`
	Author      string `json:"author"`
}

// ==================== Handlers ====================

// ListSolutionLibrarys 获取模板包列表
func ListSolutionLibrarys(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	category := c.Query("category")
	packages, err := WorkflowService.ListSolutionLibrarys(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}
	if packages == nil {
		packages = []workflow.SolutionLibrary{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": packages})
}

// GetSolutionLibrary 获取模板包详情
func GetSolutionLibrary(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	pkg, err := WorkflowService.GetSolutionLibrary(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "模板包不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": pkg})
}

// CreateSolutionLibrary 创建模板包
func CreateSolutionLibrary(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	var req CreateSolutionLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	pkg := &workflow.SolutionLibrary{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Version:     req.Version,
		Author:      req.Author,
	}
	if err := WorkflowService.CreateSolutionLibrary(pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": pkg})
}

// DeleteSolutionLibrary 删除模板包（同时删除关联内容）
func DeleteSolutionLibrary(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := WorkflowService.DeleteSolutionLibrary(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "删除失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// ExportSolutionLibrary 导出模板包为 orbit-pack YAML
func ExportSolutionLibrary(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	pack, err := WorkflowService.ExportSolutionLibrary(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "导出失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": pack})
}

// ImportSolutionLibrary 导入 orbit-pack YAML
func ImportSolutionLibrary(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	var pack workflow.OrbitPack
	if err := c.ShouldBindJSON(&pack); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	pkg, err := WorkflowService.ImportSolutionLibrary(&pack)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "导入失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": pkg})
}
