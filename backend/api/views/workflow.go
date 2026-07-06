package views

import (
	"net/http"
	"strconv"
	"time"

	"fastdp-orbit/backend/engine/orchestrator"
	"fastdp-orbit/backend/models/workflow"
	workflowsvc "fastdp-orbit/backend/services/workflow"

	"github.com/gin-gonic/gin"
)

// WorkflowService 工作流服务（由 main 初始化注入）
var WorkflowService *workflowsvc.Service

// Orchestrator 工作流执行引擎（由 main 初始化注入）
var Orchestrator *orchestrator.Orchestrator

// ==================== 请求/响应结构 ====================

type CreateWorkflowRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Description string                  `json:"description"`
	StageGroups []CreateStageGroupInput `json:"stage_groups"`
	Hooks       []CreateHookInput       `json:"hooks"`
}

type UpdateWorkflowRequest struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	StageGroups []CreateStageGroupInput `json:"stage_groups"`
	Hooks       []CreateHookInput       `json:"hooks"`
}

type CreateStageGroupInput struct {
	Name        string             `json:"name" binding:"required"`
	Description string             `json:"description"`
	Order       int                `json:"order"`
	Mode        string             `json:"mode"` // sequential/parallel
	Stages      []CreateStageInput `json:"stages" binding:"required,min=1"`
}

type CreateStageInput struct {
	Name            string            `json:"name" binding:"required"`
	Description     string            `json:"description"`
	Order           int               `json:"order"`
	MachineGroupID  uint              `json:"machine_group_id"`
	TemplateVersion string            `json:"template_version"`
	Tasks           []CreateTaskInput `json:"tasks" binding:"required,min=1"`
}

type CreateTaskInput struct {
	Ref          int    `json:"ref"`
	Name         string `json:"name" binding:"required"`
	Module       string `json:"module" binding:"required"`
	Params       string `json:"params"`
	Order        int    `json:"order"`
	When         string `json:"when"`
	Hooks      string `json:"hooks"`
	Loop         string `json:"loop"`
	Timeout      int    `json:"timeout"`
	IgnoreErrors bool   `json:"ignore_errors"`
	Retries      int    `json:"retries"`
	Delay        int    `json:"delay"`
	Register     string `json:"register"`
}

type CreateHookInput struct {
	Name         string `json:"name" binding:"required"`
	Module       string `json:"module" binding:"required"`
	Params       string `json:"params"`
	Timeout      int    `json:"timeout"`
	IgnoreErrors bool   `json:"ignore_errors"`
	Retries      int    `json:"retries"`
	Delay        int    `json:"delay"`
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
	}

	// 构建 StageGroups
	for _, g := range req.StageGroups {
		group := workflow.WorkflowStageGroup{
			Name:        g.Name,
			Description: g.Description,
			Order:       g.Order,
			Mode:        g.Mode,
		}
		for _, s := range g.Stages {
			stage := workflow.WorkflowStage{
				Name:            s.Name,
				Description:     s.Description,
				Order:           s.Order,
				MachineGroupID:  s.MachineGroupID,
				TemplateVersion: s.TemplateVersion,
			}
			for _, t := range s.Tasks {
				stage.Tasks = append(stage.Tasks, workflow.WorkflowTask{
					Ref:          t.Ref,
					Name:         t.Name,
					Module:       t.Module,
					Params:       t.Params,
					Order:        t.Order,
					When:         t.When,
					Hooks:      t.Hooks,
					Loop:         t.Loop,
					Timeout:      t.Timeout,
					IgnoreErrors: t.IgnoreErrors,
					Retries:      t.Retries,
					Delay:        t.Delay,
					Register:     t.Register,
				})
			}
			group.Stages = append(group.Stages, stage)
		}
		wf.StageGroups = append(wf.StageGroups, group)
	}

	// 构建 Hooks
	for _, h := range req.Hooks {
		wf.Hooks = append(wf.Hooks, workflow.WorkflowHook{
			Name:         h.Name,
			Module:       h.Module,
			Params:       h.Params,
			Timeout:      h.Timeout,
			IgnoreErrors: h.IgnoreErrors,
			Retries:      h.Retries,
			Delay:        h.Delay,
		})
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

	// 检查是否有运行中的执行
	running, err := WorkflowService.HasRunningExecutions(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "系统内部错误"})
		return
	}
	if running {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "工作流正在执行中，无法编辑"})
		return
	}

	wf := &workflow.Workflow{
		Name:        req.Name,
		Description: req.Description,
	}

	// 构建 StageGroups
	for _, g := range req.StageGroups {
		group := workflow.WorkflowStageGroup{
			Name:        g.Name,
			Description: g.Description,
			Order:       g.Order,
			Mode:        g.Mode,
		}
		for _, s := range g.Stages {
			stage := workflow.WorkflowStage{
				Name:            s.Name,
				Description:     s.Description,
				Order:           s.Order,
				MachineGroupID:  s.MachineGroupID,
				TemplateVersion: s.TemplateVersion,
			}
			for _, t := range s.Tasks {
				stage.Tasks = append(stage.Tasks, workflow.WorkflowTask{
					Ref:          t.Ref,
					Name:         t.Name,
					Module:       t.Module,
					Params:       t.Params,
					Order:        t.Order,
					When:         t.When,
					Hooks:      t.Hooks,
					Loop:         t.Loop,
					Timeout:      t.Timeout,
					IgnoreErrors: t.IgnoreErrors,
					Retries:      t.Retries,
					Delay:        t.Delay,
					Register:     t.Register,
				})
			}
			group.Stages = append(group.Stages, stage)
		}
		wf.StageGroups = append(wf.StageGroups, group)
	}

	// 构建 Hooks
	for _, h := range req.Hooks {
		wf.Hooks = append(wf.Hooks, workflow.WorkflowHook{
			Name:         h.Name,
			Module:       h.Module,
			Params:       h.Params,
			Timeout:      h.Timeout,
			IgnoreErrors: h.IgnoreErrors,
			Retries:      h.Retries,
			Delay:        h.Delay,
		})
	}

	// 校验
	if err := WorkflowService.ValidateWorkflow(wf); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	if err := WorkflowService.UpdateWorkflow(uint(id), wf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "更新失败: " + err.Error()})
		return
	}

	// 返回更新后的工作流（含 updated_at）
	updatedWf, err := WorkflowService.GetWorkflow(uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": updatedWf})
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

// ExecuteWorkflow 触发工作流执行（创建新的执行记录并启动）
func ExecuteWorkflow(c *gin.Context) {
	if Orchestrator == nil || WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	// 验证工作流存在
	wf, err := WorkflowService.GetWorkflow(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "工作流不存在"})
		return
	}

	// 创建执行记录
	exec := &workflow.WorkflowExecution{
		WorkflowID: wf.ID,
		Status:     "running",
		Trigger:    "user",
		StartedAt:  time.Now(),
	}
	if err := Orchestrator.CreateAndExecute(exec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "执行已启动", "data": gin.H{
		"execution_id": exec.ID,
	}})
}

// PauseWorkflow 暂停工作流执行
func PauseWorkflow(c *gin.Context) {
	if Orchestrator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	eid, err := strconv.ParseUint(c.Param("eid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "执行ID格式错误"})
		return
	}

	if err := Orchestrator.Pause(uint(eid)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已暂停"})
}

// ResumeWorkflow 继续执行暂停的工作流
func ResumeWorkflow(c *gin.Context) {
	if Orchestrator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	eid, err := strconv.ParseUint(c.Param("eid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "执行ID格式错误"})
		return
	}

	if err := Orchestrator.Resume(uint(eid)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已继续执行"})
}

// CancelWorkflow 终止工作流执行
func CancelWorkflow(c *gin.Context) {
	if Orchestrator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	eid, err := strconv.ParseUint(c.Param("eid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "执行ID格式错误"})
		return
	}

	if err := Orchestrator.Cancel(uint(eid)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已终止"})
}

// RetryStage 重试失败的 stage
func RetryStage(c *gin.Context) {
	if Orchestrator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	eid, err := strconv.ParseUint(c.Param("eid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "执行ID格式错误"})
		return
	}

	sid, err := strconv.ParseUint(c.Param("sid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "Stage执行ID格式错误"})
		return
	}

	if err := Orchestrator.RetryStage(uint(eid), uint(sid)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "重试已启动"})
}

// RetryExecution 重新执行整个工作流（从失败处开始）
func RetryExecution(c *gin.Context) {
	if Orchestrator == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	eid, err := strconv.ParseUint(c.Param("eid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "执行ID格式错误"})
		return
	}

	if err := Orchestrator.RetryExecution(uint(eid)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "重新执行已启动"})
}

// GetExecution 获取执行详情
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

// DeleteExecution 删除执行记录
func DeleteExecution(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	execID, err := strconv.ParseUint(c.Param("eid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := WorkflowService.DeleteExecution(uint(execID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
