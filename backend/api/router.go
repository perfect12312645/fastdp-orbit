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
		// Machine management
		machines := api.Group("/machines")
		{
			machines.GET("", views.ListMachines)
			machines.GET("/sync-hardware", views.SyncHardware)
			machines.DELETE("/:ip/:port", views.DeleteMachine)
			machines.POST("/:ip/:port", views.ExecOnMachine)
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
		}

		// Templates
		templates := api.Group("/templates")
		{
			templates.GET("", views.ListTemplates)
			templates.POST("", views.CreateTemplate)
			templates.GET("/:id", views.GetTemplate)
			templates.PUT("/:id", views.UpdateTemplate)
			templates.DELETE("/:id", views.DeleteTemplate)
		}

		// Cluster management
		clusters := api.Group("/clusters")
		{
			clusters.GET("", views.ListClusters)
			clusters.POST("", views.CreateCluster)
			clusters.GET("/:id", views.GetCluster)
			clusters.POST("/:id/init", views.InitCluster)
			clusters.POST("/:id/join", views.JoinCluster)
			clusters.GET("/:id/nodes", views.ListClusterNodes)
		}

		// Monitoring
		monitor := api.Group("/monitor")
		{
			monitor.GET("/overview", views.GetOverview)
			monitor.GET("/nodes", views.ListNodes)
			monitor.GET("/nodes/:id", views.GetNodeMetrics)
			monitor.GET("/pods", views.ListPods)
			monitor.GET("/events", views.ListEvents)
		}

		// GPU management
		gpu := api.Group("/gpu")
		{
			gpu.GET("/nodes", views.ListGPUNodes)
			gpu.GET("/tasks", views.ListGPUTasks)
			gpu.POST("/tasks", views.CreateGPUTask)
			gpu.GET("/models", views.ListModels)
			gpu.POST("/models/deploy", views.DeployModel)
		}

		// WebSocket for real-time updates
		api.GET("/ws", views.HandleWebSocket)

		// Install commands
		api.GET("/install/command", views.GetInstallCommand)
	}

	// Static files
	router.Static("/static", "./static")
	router.Static("/materials", "./materials")

	// Vue 静态资源
	router.Static("/assets", "./dist/assets")
	router.StaticFile("/favicon.ico", "./dist/favicon.ico")

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
