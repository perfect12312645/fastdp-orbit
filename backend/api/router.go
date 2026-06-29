package api

import (
	"fastdp-orbit/backend/api/middleware"
	"fastdp-orbit/backend/api/views"
	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/server/cache"
	servergrpc "fastdp-orbit/backend/server/grpc"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.ServerConfig, mc *cache.MachineCache, pool *servergrpc.AgentConnPool) *gin.Engine {
	// 设置依赖注入
	views.MachineCache = mc
	views.AgentConnPool = pool
	views.ServerConfig = cfg
	if cfg.OrbitServer.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(
		ginzap.Ginzap(logger.GetLogger(), time.RFC3339, true),
		ginzap.RecoveryWithZap(logger.GetLogger(), true),
		middleware.CORS(),
	)

	// Health checkx
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1
	api := router.Group("/api/v1")
	{
		// Dashboard
		api.GET("/dashboard/stats", views.GetDashboardStats)

		// Machine management
		machines := api.Group("/machines")
		{
			machines.GET("", views.ListMachines)
			machines.GET("/sync-hardware", views.SyncHardware)
			machines.DELETE("/:ip/:port", views.DeleteMachine)
			machines.POST("/:ip/:port", views.ExecOnMachine)
		}

		// Machine groups
		machineGroups := api.Group("/machine-groups")
		{
			machineGroups.GET("", views.ListMachineGroups)
			machineGroups.POST("", views.CreateMachineGroup)
			machineGroups.GET("/:id", views.GetMachineGroup)
			machineGroups.PUT("/:id", views.UpdateMachineGroup)
			machineGroups.DELETE("/:id", views.DeleteMachineGroup)
		}

		// Workflow/Orchestration
		workflows := api.Group("/workflows")
		{
			workflows.GET("", views.ListWorkflows)
			workflows.POST("", views.CreateWorkflow)
			workflows.GET("/:id", views.GetWorkflow)
			workflows.PUT("/:id", views.UpdateWorkflow)
			workflows.DELETE("/:id", views.DeleteWorkflow)
			workflows.POST("/:id/execute", views.ExecuteWorkflow)
			workflows.GET("/:id/executions", views.ListExecutions)
			workflows.GET("/:id/executions/:eid", views.GetExecution)

			// 执行控制
			workflows.POST("/:id/executions/:eid/pause", views.PauseWorkflow)
			workflows.POST("/:id/executions/:eid/resume", views.ResumeWorkflow)
			workflows.POST("/:id/executions/:eid/cancel", views.CancelWorkflow)
			workflows.POST("/:id/executions/:eid/retry", views.RetryExecution)
			workflows.POST("/:id/executions/:eid/stages/:sid/retry", views.RetryStage)
		}

		// Stage Templates（阶段模板管理）
		stageTemplates := api.Group("/stage-templates")
		{
			stageTemplates.GET("", views.ListStageTemplates)
			stageTemplates.POST("", views.CreateStageTemplate)
			stageTemplates.GET("/:id", views.GetStageTemplate)
			stageTemplates.PUT("/:id", views.UpdateStageTemplate)
			stageTemplates.DELETE("/:id", views.DeleteStageTemplate)

			// 版本管理
			stageTemplates.GET("/:id/versions", views.ListStageTemplateVersions)
			stageTemplates.POST("/:id/rollback", views.RollbackStageTemplate)

			// 单阶段执行
			stageTemplates.POST("/:id/execute", views.ExecuteSingleStage)

			// 执行历史
			stageTemplates.GET("/:id/executions", views.ListStageExecutions)
		}

		// Global Variables（全局变量管理）
		globalVars := api.Group("/global-variables")
		{
			globalVars.GET("", views.ListGlobalVariables)
			globalVars.POST("", views.CreateGlobalVariable)
			globalVars.GET("/:id", views.GetGlobalVariable)
			globalVars.PUT("/:id", views.UpdateGlobalVariable)
			globalVars.DELETE("/:id", views.DeleteGlobalVariable)
		}

		// Hook Templates（钩子模板管理）
		hookTemplates := api.Group("/hook-templates")
		{
			hookTemplates.GET("", views.ListHookTemplates)
			hookTemplates.POST("", views.CreateHookTemplate)
			hookTemplates.GET("/:id", views.GetHookTemplate)
			hookTemplates.PUT("/:id", views.UpdateHookTemplate)
			hookTemplates.DELETE("/:id", views.DeleteHookTemplate)
		}

		// Workflow Templates（工作流模板文件管理）
		workflowTemplates := api.Group("/workflow-templates")
		{
			workflowTemplates.GET("", views.ListWorkflowTemplates)
			workflowTemplates.POST("", views.CreateWorkflowTemplate)
			workflowTemplates.POST("/preview", views.PreviewTemplate)
			workflowTemplates.GET("/:id", views.GetWorkflowTemplate)
			workflowTemplates.PUT("/:id", views.UpdateWorkflowTemplate)
			workflowTemplates.DELETE("/:id", views.DeleteWorkflowTemplate)
		}

		// Solution Library（方案库管理）
		solutionLibraries := api.Group("/solution-libraries")
		{
			solutionLibraries.GET("", views.ListSolutionLibrarys)
			solutionLibraries.POST("", views.CreateSolutionLibrary)
			solutionLibraries.GET("/:id", views.GetSolutionLibrary)
			solutionLibraries.DELETE("/:id", views.DeleteSolutionLibrary)
			solutionLibraries.GET("/:id/export", views.ExportSolutionLibrary)
			solutionLibraries.POST("/import", views.ImportSolutionLibrary)
		}

		// Storage（文件存储管理）
		storage := api.Group("/storage")
		{
			storage.GET("/files", views.ListStorageFiles)
			storage.GET("/files/:id", views.GetStorageFile)
			storage.DELETE("/files/:id", views.DeleteStorageFile)
			storage.POST("/upload", views.UploadChunk)
			storage.GET("/resume-info", views.GetResumeInfo)
		}

		// SSE for execution real-time updates
		api.GET("/executions/:id/stream", views.HandleSSE)

		// Install commands
		api.GET("/install/command", views.GetInstallCommand)
	}

	// Static files
	router.Static("/static", "./static")
	router.Static("/materials", "./materials")

	// Vue 静态资源
	router.Static("/assets", "./dist/assets")
	router.StaticFile("/vite.svg", "./dist/vite.svg")

	// 文件下载路由（独立于 /api/v1，支持 wget -c 续传）
	router.GET("/download/*path", views.DownloadFile)

	// Vue 路由 history 模式支持（放在最后）
	router.NoRoute(func(c *gin.Context) {
		// API 请求返回 404
		if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:5] == "/api/" {
			c.JSON(404, gin.H{"error": "接口不存在"})
			return
		}
		// 其他请求返回 index.html
		c.File("./dist/index.html")
	})

	return router
}
