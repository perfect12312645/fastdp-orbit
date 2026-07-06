package views

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"fastdp-orbit/backend/models/workflow"
	workflowsvc "fastdp-orbit/backend/services/workflow"

	"github.com/gin-gonic/gin"
)

// ==================== 请求结构 ====================

// ImportSolutionLibraryRequest 导入方案请求
type ImportSolutionLibraryRequest struct {
	Pack workflow.OrbitPack `json:"pack" binding:"required"`
}

// ApplySolutionLibraryRequest 应用方案请求
type ApplySolutionLibraryRequest struct {
	Decisions           map[string]map[string]string `json:"decisions,omitempty"`             // type -> name -> "skip"|"overwrite"
	VariableValues      map[string]string            `json:"variable_values,omitempty"`       // key -> value (用户修改后的变量值)
	MachineGroupMachines map[string][]uint            `json:"machine_group_machines,omitempty"` // 分组名 -> 机器ID列表
}

// ConflictResponse 冲突检测响应
type ConflictResponse struct {
	HasConflicts bool                        `json:"has_conflicts"`
	Conflicts    []workflowsvc.ConflictItem  `json:"conflicts"`
	Summary      ImportSummary               `json:"summary"`
}

// ImportSummary 导入内容摘要
type ImportSummary struct {
	StageCount    int `json:"stage_count"`
	VariableCount int `json:"variable_count"`
	HookCount     int `json:"hook_count"`
	TemplateCount int `json:"template_count"`
	FileCount     int `json:"file_count"`
	WorkflowCount int `json:"workflow_count"`
}

type CreateSolutionLibraryRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	Version      string `json:"version"`
	Author       string `json:"author"`
	StageIDs     []uint `json:"stage_ids"`
	VariableIDs  []uint `json:"variable_ids"`
	HookIDs      []uint `json:"hook_ids"`
	TemplateIDs  []uint `json:"template_ids"`
	FileIDs      []uint `json:"file_ids"`
	WorkflowIDs  []uint `json:"workflow_ids"`
}

type UpdateSolutionLibraryRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	Version      string `json:"version"`
	Author       string `json:"author"`
	StageIDs     []uint `json:"stage_ids"`
	VariableIDs  []uint `json:"variable_ids"`
	HookIDs      []uint `json:"hook_ids"`
	TemplateIDs  []uint `json:"template_ids"`
	FileIDs      []uint `json:"file_ids"`
	WorkflowIDs  []uint `json:"workflow_ids"`
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

// CreateSolutionLibrary 创建方案
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

	if len(req.StageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "请至少选择一个阶段"})
		return
	}

	// 将ID数组转为JSON字符串
	stageIDsJSON, _ := json.Marshal(req.StageIDs)
	variableIDsJSON, _ := json.Marshal(req.VariableIDs)
	hookIDsJSON, _ := json.Marshal(req.HookIDs)
	templateIDsJSON, _ := json.Marshal(req.TemplateIDs)
	fileIDsJSON, _ := json.Marshal(req.FileIDs)
	workflowIDsJSON, _ := json.Marshal(req.WorkflowIDs)

	solution := &workflow.SolutionLibrary{
		Name:         req.Name,
		Description:  req.Description,
		Category:     req.Category,
		Version:      req.Version,
		Author:       req.Author,
		StageIDs:     string(stageIDsJSON),
		VariableIDs:  string(variableIDsJSON),
		HookIDs:      string(hookIDsJSON),
		TemplateIDs:  string(templateIDsJSON),
		FileIDs:      string(fileIDsJSON),
		WorkflowIDs:  string(workflowIDsJSON),
	}
	if err := WorkflowService.CreateSolutionLibrary(solution); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": solution})
}

// UpdateSolutionLibrary 更新方案
func UpdateSolutionLibrary(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	var req UpdateSolutionLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	if len(req.StageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "请至少选择一个阶段"})
		return
	}

	// 将ID数组转为JSON字符串
	stageIDsJSON, _ := json.Marshal(req.StageIDs)
	variableIDsJSON, _ := json.Marshal(req.VariableIDs)
	hookIDsJSON, _ := json.Marshal(req.HookIDs)
	templateIDsJSON, _ := json.Marshal(req.TemplateIDs)
	fileIDsJSON, _ := json.Marshal(req.FileIDs)
	workflowIDsJSON, _ := json.Marshal(req.WorkflowIDs)

	solution := &workflow.SolutionLibrary{
		Name:         req.Name,
		Description:  req.Description,
		Category:     req.Category,
		Version:      req.Version,
		Author:       req.Author,
		StageIDs:     string(stageIDsJSON),
		VariableIDs:  string(variableIDsJSON),
		HookIDs:      string(hookIDsJSON),
		TemplateIDs:  string(templateIDsJSON),
		FileIDs:      string(fileIDsJSON),
		WorkflowIDs:  string(workflowIDsJSON),
	}
	if err := WorkflowService.UpdateSolutionLibrary(uint(id), solution); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DeleteSolutionLibrary 删除方案
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

// ImportSolutionLibrary 导入 orbit-pack YAML（支持冲突检测模式）
func ImportSolutionLibrary(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	var req ImportSolutionLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	pack := &req.Pack

	// 导入时只检查方案名是否已存在
	pkg, err := WorkflowService.ImportSolutionLibrary(pack)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "导入失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": pkg})
}

// ApplySolutionLibrary 应用方案（检测冲突，由用户决策后执行）
func ApplySolutionLibrary(c *gin.Context) {
	if WorkflowService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	var req ApplySolutionLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	solutionID := uint(id)

	// 如果没有提供决策，进行冲突检测并返回（即使无冲突也要让用户配置变量和机器分组）
	if req.Decisions == nil {
		conflicts, svcSummary, err := WorkflowService.CheckApplyConflicts(solutionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 0, "message": "success",
			"data": ConflictResponse{
				HasConflicts: len(conflicts) > 0,
				Conflicts:    conflicts,
				Summary: ImportSummary{
					StageCount:    svcSummary.StageCount,
					VariableCount: svcSummary.VariableCount,
					HookCount:     svcSummary.HookCount,
					TemplateCount: svcSummary.TemplateCount,
					FileCount:     svcSummary.FileCount,
					WorkflowCount: svcSummary.WorkflowCount,
				},
			},
		})
		return
	}

	// 有决策，执行应用
	err = WorkflowService.ApplySolutionLibraryWithDecisions(solutionID, req.Decisions, req.VariableValues, req.MachineGroupMachines)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "应用失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DownloadFileProxy 文件下载代理，解决前端跨域问题
func DownloadFileProxy(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	resp, err := http.Get(req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "下载失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"code": -1, "message": "远程服务器返回 " + resp.Status})
		return
	}

	// 从URL提取文件名
	parts := strings.Split(req.URL, "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		filename = "download"
	}

	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Content-Length", resp.Header.Get("Content-Length"))
	io.Copy(c.Writer, resp.Body)
}
