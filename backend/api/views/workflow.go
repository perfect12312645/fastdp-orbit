package views

import (
	"net/http"
	"strconv"

	"fastdp-orbit/backend/models/workflow"
	workflowsvc "fastdp-orbit/backend/services/workflow"

	"github.com/gin-gonic/gin"
)

// WorkflowService 工作流服务（由 main 初始化注入）
var WorkflowService *workflowsvc.Service

// ==================== 请求/响应结构 ====================

type CreateWorkflowRequest struct {
	Name        string             `json:"name" binding:"required"`
	Description string             `json:"description"`
	Config      string             `json:"config"`
	Stages      []CreateStageInput `json:"stages" binding:"required,min=1"`
}

type UpdateWorkflowRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Config      string             `json:"config"`
	Stages      []CreateStageInput `json:"stages"`
}

type CreateStageInput struct {
	Name            string            `json:"name" binding:"required"`
	Description     string            `json:"description"`
	Order           int               `json:"order"`
	RetryPolicy     string            `json:"retry_policy"`
	MaxRetries      int               `json:"max_retries"`
	ContinueOnError bool              `json:"continue_on_error"`
	Tasks           []CreateTaskInput `json:"tasks" binding:"required,min=1"`
}

type CreateTaskInput struct {
	Name    string `json:"name" binding:"required"`
	Module  string `json:"module" binding:"required"`
	Params  string `json:"params"`
	Host    string `json:"host" binding:"required"`
	Order   int    `json:"order"`
	When    string `json:"when"`
	Hooks   string `json:"hooks"`
	Loop    string `json:"loop"`
	Timeout int    `json:"timeout"`
}

// ==================== Handlers ====================

// ListWorkflows 获取所有工作流
func ListWorkflows(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	wfs, err := WorkflowService.ListWorkflows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	if wfs == nil {
		wfs = []workflow.Workflow{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": wfs})
}

// CreateWorkflow 创建工作流
func CreateWorkflow(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	// 构建 model
	wf := &workflow.Workflow{
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
	}
	for _, s := range req.Stages {
		stage := workflow.WorkflowStage{
			Name:            s.Name,
			Description:     s.Description,
			Order:           s.Order,
			RetryPolicy:     s.RetryPolicy,
			MaxRetries:      s.MaxRetries,
			ContinueOnError: s.ContinueOnError,
		}
		for _, t := range s.Tasks {
			stage.Tasks = append(stage.Tasks, workflow.WorkflowTask{
				Name:    t.Name,
				Module:  t.Module,
				Params:  t.Params,
				Host:    t.Host,
				Order:   t.Order,
				When:    t.When,
				Hooks:   t.Hooks,
				Loop:    t.Loop,
				Timeout: t.Timeout,
			})
		}
		wf.Stages = append(wf.Stages, stage)
	}

	// 校验
	if err := WorkflowService.ValidateWorkflow(wf); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	// 创建
	if err := WorkflowService.CreateWorkflow(wf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": wf})
}

// GetWorkflow 获取工作流详情
func GetWorkflow(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	wf, err := WorkflowService.GetWorkflow(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "工作流不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": wf})
}

// UpdateWorkflow 更新工作流
func UpdateWorkflow(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	var req UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	wf := &workflow.Workflow{
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
	}
	for _, s := range req.Stages {
		stage := workflow.WorkflowStage{
			Name:            s.Name,
			Description:     s.Description,
			Order:           s.Order,
			RetryPolicy:     s.RetryPolicy,
			MaxRetries:      s.MaxRetries,
			ContinueOnError: s.ContinueOnError,
		}
		for _, t := range s.Tasks {
			stage.Tasks = append(stage.Tasks, workflow.WorkflowTask{
				Name:    t.Name,
				Module:  t.Module,
				Params:  t.Params,
				Host:    t.Host,
				Order:   t.Order,
				When:    t.When,
				Hooks:   t.Hooks,
				Loop:    t.Loop,
				Timeout: t.Timeout,
			})
		}
		wf.Stages = append(wf.Stages, stage)
	}

	if err := WorkflowService.UpdateWorkflow(uint(id), wf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "更新失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DeleteWorkflow 删除工作流
func DeleteWorkflow(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := WorkflowService.DeleteWorkflow(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// ExecuteWorkflow 触发工作流执行
func ExecuteWorkflow(c *gin.Context) {
	// TODO: Phase 1.4 实现执行引擎集成
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "执行功能开发中"})
}

// GetWorkflowStatus 获取工作流执行状态
func GetWorkflowStatus(c *gin.Context) {
	// TODO: Phase 1.4 实现执行状态查询
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"status": "pending"}})
}

// ListExecutions 获取工作流执行历史
func ListExecutions(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	execs, err := WorkflowService.ListExecutions(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	if execs == nil {
		execs = []workflow.WorkflowExecution{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": execs})
}

// GetExecution 获取执行详情
func GetExecution(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	execID, err := strconv.ParseUint(c.Param("eid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	exec, err := WorkflowService.GetExecution(uint(execID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "执行记录不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": exec})
}
