package views

import (
	"net/http"
	"strconv"

	workflowsvc "fastdp-orbit/backend/services/workflow"

	"github.com/gin-gonic/gin"
)

// StageExecutionService 单阶段执行服务（由 main 初始化注入）
var StageExecutionService *workflowsvc.StageExecutionService

// ==================== 请求结构 ====================

type ExecuteStageRequest struct {
	MachineGroupID uint `json:"machine_group_id"` // 可选，覆盖阶段模板默认分组
}

// ==================== Handlers ====================

// ExecuteSingleStage 单独执行一个阶段
func ExecuteSingleStage(c *gin.Context) {
	if StageExecutionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "执行服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	var req ExecuteStageRequest
	_ = c.ShouldBindJSON(&req) // body 可选

	exec, err := StageExecutionService.ExecuteStage(uint(id), req.MachineGroupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "执行已启动", "data": gin.H{
		"execution_id": exec.ID,
	}})
}

// ListStageExecutions 获取阶段的执行历史
func ListStageExecutions(c *gin.Context) {
	if StageExecutionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	executions, err := StageExecutionService.ListStageExecutions(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": executions})
}

// GetStageExecution 获取执行详情
func GetStageExecution(c *gin.Context) {
	if StageExecutionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	exec, err := StageExecutionService.GetStageExecution(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "执行记录不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": exec})
}

// DeleteStageExecution 删除执行记录
func DeleteStageExecution(c *gin.Context) {
	if StageExecutionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := StageExecutionService.DeleteStageExecution(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// CancelStageExecution 取消执行
func CancelStageExecution(c *gin.Context) {
	if StageExecutionService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := StageExecutionService.CancelStageExecution(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
