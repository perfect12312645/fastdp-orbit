package views

import (
	"net/http"
	"strconv"

	"fastdp-orbit/backend/engine/orchestrator"
	"fastdp-orbit/backend/models/machine"
	"fastdp-orbit/backend/models/workflow"

	"github.com/gin-gonic/gin"
)

// ==================== 请求结构 ====================

type CreateWorkflowTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Variables   string `json:"variables"`
}

type UpdateWorkflowTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Variables   string `json:"variables"`
}

// ==================== Handlers ====================

// ListWorkflowTemplates 获取所有工作流模板文件
func ListWorkflowTemplates(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	packageGroup := c.Query("source")
	templates, err := WorkflowService.ListWorkflowTemplates(packageGroup)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "系统内部错误"})
		return
	}
	if templates == nil {
		templates = []workflow.WorkflowTemplate{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": templates})
}

// CreateWorkflowTemplate 创建工作流模板文件
func CreateWorkflowTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	var req CreateWorkflowTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	t := &workflow.WorkflowTemplate{
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Variables:   req.Variables,
	}
	if err := WorkflowService.CreateWorkflowTemplate(t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": t})
}

// GetWorkflowTemplate 获取工作流模板文件详情
func GetWorkflowTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	t, err := WorkflowService.GetWorkflowTemplate(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "模板不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": t})
}

// UpdateWorkflowTemplate 更新工作流模板文件
func UpdateWorkflowTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	var req UpdateWorkflowTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	t := &workflow.WorkflowTemplate{
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Variables:   req.Variables,
	}
	if err := WorkflowService.UpdateWorkflowTemplate(uint(id), t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DeleteWorkflowTemplate 删除工作流模板文件
func DeleteWorkflowTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := WorkflowService.DeleteWorkflowTemplate(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// PreviewTemplateRequest 模板预览请求
type PreviewTemplateRequest struct {
	Content   string `json:"content" binding:"required"`
	MachineID uint   `json:"machine_id"` // 选择具体机器
}

// PreviewTemplate 渲染模板预览
func PreviewTemplate(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	var req PreviewTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	// 构建模板变量
	templateVars := buildPreviewVars(req.MachineID)

	// 复用 RenderTemplate 渲染（与 orchestrator 一致）
	rendered, err := orchestrator.RenderTemplate(req.Content, templateVars)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "渲染失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": rendered})
}

// buildPreviewVars 构建预览用的模板变量（选择具体机器）
func buildPreviewVars(machineID uint) map[string]interface{} {
	db := WorkflowService.DB()

	// 加载全局变量
	var globalVarList []workflow.GlobalVariable
	db.Find(&globalVarList)
	globalVars := make(map[string]interface{})
	for _, v := range globalVarList {
		globalVars[v.Key] = v.Value
	}

	// 加载所有机器分组（供 Groups 使用）
	var allGroups []machine.MachineGroup
	db.Preload("Machines").Find(&allGroups)
	groupsMap := orchestrator.BuildGroupsMap(allGroups)

	// 构建当前机器变量（默认占位）
	machineMap := map[string]interface{}{
		"ip":       "127.0.0.1",
		"hostname": "preview-host",
		"os_name":  "preview-os",
	}

	// 如果指定了机器，使用该机器的真实数据
	if machineID > 0 {
		var m machine.Machine
		if err := db.First(&m, machineID).Error; err == nil {
			machineMap = orchestrator.MachineToMap(&m)
		}
	}

	// Group 变量（默认占位）
	groupVars := map[string]interface{}{
		"name": "preview-group",
	}

	// Server 变量
	serverVars := map[string]interface{}{
		"ip":       "127.0.0.1",
		"port":     "8080",
		"protocol": "http",
	}
	if Orchestrator != nil {
		serverVars["ip"] = Orchestrator.GetServerIP()
		serverVars["port"] = Orchestrator.GetServerPort()
		serverVars["protocol"] = Orchestrator.GetProtocol()
	}

	return map[string]interface{}{
		"Machine":        machineMap,
		"GlobalVariable": globalVars,
		"Group":          groupVars,
		"Groups":         groupsMap,
		"Server":         serverVars,
	}
}
