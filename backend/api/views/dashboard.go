package views

import (
	"net/http"

	"fastdp-orbit/backend/models/workflow"

	"github.com/gin-gonic/gin"
)

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	TotalMachines  int64                  `json:"total_machines"`
	OnlineMachines int64                  `json:"online_machines"`
	TotalWorkflows int64                  `json:"total_workflows"`
	TotalStages    int64                  `json:"total_stages"`
	TotalVariables int64                  `json:"total_variables"`
	TotalHooks     int64                  `json:"total_hooks"`
	TotalFiles     int64                  `json:"total_files"`
	RecentExecs    []RecentExecution      `json:"recent_execs"`
	ExecStats      ExecutionStats         `json:"exec_stats"`
}

// RecentExecution 最近执行记录
type RecentExecution struct {
	ID         uint   `json:"id"`
	WorkflowName string `json:"workflow_name"`
	Status     string `json:"status"`
	Trigger    string `json:"trigger"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	Duration   int64  `json:"duration_ms"`
}

// ExecutionStats 执行统计
type ExecutionStats struct {
	Total    int64 `json:"total"`
	Success  int64 `json:"success"`
	Failed   int64 `json:"failed"`
	Running  int64 `json:"running"`
}

// GetDashboardStats 获取仪表盘统计数据
func GetDashboardStats(c *gin.Context) {
	db := WorkflowService.DB()

	var stats DashboardStats

	// 机器统计（从缓存获取）
	if MachineCache != nil {
		machines := MachineCache.List()
		stats.TotalMachines = int64(len(machines))
		for _, m := range machines {
			if m.Status == "online" {
				stats.OnlineMachines++
			}
		}
	}

	// 工作流数量
	db.Model(&workflow.Workflow{}).Count(&stats.TotalWorkflows)

	// 阶段模板数量
	db.Model(&workflow.StageTemplate{}).Count(&stats.TotalStages)

	// 全局变量数量
	db.Model(&workflow.GlobalVariable{}).Count(&stats.TotalVariables)

	// 钩子模板数量
	db.Model(&workflow.HookTemplate{}).Count(&stats.TotalHooks)

	// 存储文件数量
	db.Table("storage_files").Where("deleted_at IS NULL").Count(&stats.TotalFiles)

	// 最近执行记录
	var execs []workflow.WorkflowExecution
	db.Preload("Workflow").Order("created_at DESC").Limit(10).Find(&execs)
	for _, e := range execs {
		re := RecentExecution{
			ID:         e.ID,
			Status:     e.Status,
			Trigger:    e.Trigger,
			StartedAt:  e.StartedAt.Format("2006-01-02 15:04:05"),
		}
		if e.Workflow != nil {
			re.WorkflowName = e.Workflow.Name
		}
		if e.FinishedAt != nil && !e.FinishedAt.IsZero() {
			re.FinishedAt = e.FinishedAt.Format("2006-01-02 15:04:05")
			re.Duration = e.FinishedAt.Sub(e.StartedAt).Milliseconds()
		}
		stats.RecentExecs = append(stats.RecentExecs, re)
	}
	if stats.RecentExecs == nil {
		stats.RecentExecs = []RecentExecution{}
	}

	// 执行统计
	db.Model(&workflow.WorkflowExecution{}).Count(&stats.ExecStats.Total)
	db.Model(&workflow.WorkflowExecution{}).Where("status = ?", "success").Count(&stats.ExecStats.Success)
	db.Model(&workflow.WorkflowExecution{}).Where("status = ?", "failed").Count(&stats.ExecStats.Failed)
	db.Model(&workflow.WorkflowExecution{}).Where("status IN ?", []string{"running", "paused"}).Count(&stats.ExecStats.Running)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}
