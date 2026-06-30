package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fastdp-orbit/backend/api"
	"fastdp-orbit/backend/api/views"
	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/database"
	"fastdp-orbit/backend/engine/orchestrator"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/tlsutil"
	"fastdp-orbit/backend/pkg/version"
	"fastdp-orbit/backend/server/cache"
	servergrpc "fastdp-orbit/backend/server/grpc"
	storagesvc "fastdp-orbit/backend/services/storage"
	workflowsvc "fastdp-orbit/backend/services/workflow"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func main() {
	// 解析命令行参数
	var (
		configFile string
		port       int
		mode       string
	)

	flag.StringVar(&configFile, "config", "", "配置文件路径 (默认: /etc/fastdp-orbit/server.toml)")
	flag.IntVar(&port, "port", 0, "HTTP监听端口 (覆盖配置文件)")
	flag.StringVar(&mode, "mode", "", "运行模式: debug/release (覆盖配置文件)")
	flag.Usage = printUsage
	flag.Parse()

	// 处理 --help
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printUsage()
		os.Exit(0)
	}

	flags := &config.ServerFlags{
		Config: configFile,
		Port:   port,
		Mode:   mode,
	}

	cfg, err := config.LoadServerWithFlags(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger.InitWithConfig(&logger.LoggerConfig{
		Level:      cfg.Log.Level,
		Output:     cfg.Log.Output,
		Path:       cfg.Log.Path,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	})
	defer logger.Sync()

	// 初始化数据库
	db, err := database.Init(cfg.Database)
	if err != nil {
		logger.Fatal("数据库初始化失败", zap.Error(err))
	}

	// 创建机器缓存
	machineCache := cache.NewMachineCache(db)

	// 从数据库加载已有机器到缓存（重启后恢复）
	if err := machineCache.LoadFromDB(); err != nil {
		logger.Error("加载机器缓存失败", zap.Error(err))
	}

	// 启动离线检测（default_timeout秒无心跳标记离线，检测间隔为其一半）
	offlineTimeout := time.Duration(cfg.GRPC.DefaultTimeout) * time.Second
	machineCache.StartOfflineChecker(offlineTimeout)

	// 创建Agent连接池
	agentConnPool := servergrpc.NewAgentConnPool()
	defer agentConnPool.Close()

	// 创建工作流服务
	workflowService := workflowsvc.NewService(db)
	views.WorkflowService = workflowService

	// 创建存储服务
	storageService := storagesvc.NewService(db, "./storage")
	views.StorageService = storageService

	// 注入机器分组数据库
	views.MachineGroupDB = db

	Protocol := "http"
	if cfg.OrbitServer.TLS.Enabled {
		Protocol = "https"
	}
	// 创建执行引擎
	eng := orchestrator.NewOrchestrator(db, agentConnPool, cfg.GetServerListenAddr(), Protocol)
	views.Orchestrator = eng

	// 创建单阶段执行服务
	stageExecService := workflowsvc.NewStageExecutionService(db)
	stageExecService.SetExecuteTaskFunc(eng.ExecuteTaskForStage)
	stageExecService.SetEmitFuncs(
		func(executionID uint, taskRef int, taskName string, status string, host string, output string, errStr string, trace string, errorCode int32, changed bool, duration int64) {
			views.BroadcastTaskStatus(executionID, 0, taskRef, taskName, status, host, output, errStr, trace, errorCode, changed, duration)
		},
		func(executionID uint, status string) {
			views.BroadcastExecutionStatus(executionID, status, "")
		},
	)
	views.StageExecutionService = stageExecService

	// 设置 SSE 事件监听器
	eng.SetEventListener(&views.SSEListener{})

	// 打印启动信息
	fmt.Println("╔════════════════════════════════════════════════╗")
	fmt.Printf("║          Orbit Server %-24s║\n", version.Version)
	fmt.Println("╚════════════════════════════════════════════════╝")
	fmt.Printf("  Mode:      %s\n", cfg.OrbitServer.Mode)
	fmt.Printf("  HTTP:      %s\n", cfg.GetServerListenAddr())
	fmt.Printf("  gRPC:      %s\n", cfg.GetGRPCListenAddr())
	fmt.Printf("  Database:  %s\n", cfg.Database.Type)
	fmt.Printf("  TLS:       %v\n", cfg.GRPC.TLS.Enabled)
	fmt.Println("──────────────────────────────────────────────────")

	logger.Info("Server启动中",
		zap.String("version", version.Version),
		zap.String("http", cfg.GetServerListenAddr()),
		zap.String("grpc", cfg.GetGRPCListenAddr()),
		zap.Bool("tls_enabled", cfg.GRPC.TLS.Enabled),
	)

	// 启动gRPC服务（给agent连接）
	grpcSrv := startGRPCServer(cfg, db, machineCache)

	// 启动HTTP API服务
	router := api.SetupRouter(cfg, machineCache, agentConnPool)
	httpAddr := cfg.GetServerListenAddr()

	// 创建HTTP Server
	httpSrv := &http.Server{
		Addr:    httpAddr,
		Handler: router,
	}

	// 配置HTTP TLS
	if cfg.OrbitServer.TLS.Enabled {
		tlsConfig, err := tlsutil.LoadServerHTTPTLSCredentials(cfg.OrbitServer.TLS)
		if err != nil {
			logger.Fatal("加载HTTP TLS配置失败", zap.Error(err))
		}
		httpSrv.TLSConfig = tlsConfig
	}

	// 启动HTTP服务
	go func() {
		logger.Info("HTTP Server启动", zap.String("address", httpAddr))
		var err error
		if cfg.OrbitServer.TLS.Enabled {
			err = httpSrv.ListenAndServeTLS("", "")
		} else {
			err = httpSrv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP Server启动失败", zap.Error(err))
		}
	}()

	// 等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("收到关闭信号，开始优雅关闭...")

	// 统一设置关闭超时（预留时间处理存量请求）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭 gRPC
	logger.Info("正在关闭gRPC服务...")
	grpcSrv.GracefulStop()

	// 优雅关闭 HTTP
	logger.Info("正在关闭HTTP服务...")
	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Error("HTTP优雅关闭超时", zap.Error(err))
	}

	logger.Info("服务已全部关闭")
}

// printUsage 打印帮助信息
func printUsage() {
	fmt.Fprintf(os.Stderr, `Orbit Server

版本: %s

用法:
  orbit-server [选项]

选项:
  --config string   配置文件路径 (默认: /etc/fastdp-orbit/server.toml)
  --port int        HTTP监听端口 (覆盖配置文件)
  --mode string     运行模式: debug/release (覆盖配置文件)
  -h, --help        显示帮助信息

示例:
  # 使用默认配置
  orbit-server

  # 指定配置文件
  orbit-server --config /path/to/server.toml

  # 覆盖端口和模式
  orbit-server --port 8080 --mode release

配置文件优先级: 命令行参数 > 配置文件 > 默认值
`, version.Version)
}

// startGRPCServer 启动gRPC服务
func startGRPCServer(cfg *config.ServerConfig, db *gorm.DB, mc *cache.MachineCache) *grpc.Server {
	addr := cfg.GetGRPCListenAddr()
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal("gRPC server failed to listen", zap.String("address", addr), zap.Error(err))
	}

	// 创建gRPC服务器选项
	var opts []grpc.ServerOption

	// 配置双向TLS
	if cfg.GRPC.TLS.Enabled {
		creds, err := tlsutil.LoadServerTLSCredentials(cfg.GRPC.TLS)
		if err != nil {
			logger.Fatal("加载gRPC TLS配置失败", zap.Error(err))
		}
		opts = append(opts, grpc.Creds(creds))
	}

	s := grpc.NewServer(opts...)
	servergrpc.RegisterGRPCServer(s, cfg, db, mc)

	logger.Info("gRPC Server启动", zap.String("address", addr), zap.Bool("tls", cfg.GRPC.TLS.Enabled))

	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Fatal("gRPC server failed", zap.String("address", addr), zap.Error(err))
		}
	}()

	return s
}
