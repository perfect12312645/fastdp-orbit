package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"fastdp-orbit/backend/agent/service"
	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/version"

	"go.uber.org/zap"
)

func main() {
	// 解析命令行参数
	var (
		configFile string
		serverAddr string
		port       int
		host       string
	)

	flag.StringVar(&configFile, "config", "", "配置文件路径 (默认: /etc/fastdp-orbit/agent.toml)")
	flag.StringVar(&serverAddr, "server", "", "Server gRPC地址 (host:port)")
	flag.IntVar(&port, "port", 0, "Agent监听端口 (覆盖配置文件)")
	flag.StringVar(&host, "host", "", "Agent监听地址 + 上报给Server的IP (覆盖配置文件)")
	flag.Usage = printUsage
	flag.Parse()

	// 处理 --help
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printUsage()
		os.Exit(0)
	}

	flags := &config.AgentFlags{
		Config: configFile,
		Server: serverAddr,
		Port:   port,
		Host:   host,
	}

	cfg, err := config.LoadAgentWithFlags(flags)
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

	// 打印启动信息
	fmt.Println("╔════════════════════════════════════════════════╗")
	fmt.Printf("║          Orbit Agent %-24s║\n", version.Version)
	fmt.Println("╚════════════════════════════════════════════════╝")
	fmt.Printf("  Listen:    %s\n", cfg.GetAgentListenAddr())
	fmt.Printf("  Server:    %s\n", cfg.GetServerAddr())
	fmt.Printf("  Heartbeat: %ds\n", cfg.OrbitAgent.Heartbeat)
	fmt.Println("──────────────────────────────────────────────────")

	logger.Info("Agent启动中",
		zap.String("version", version.Version),
		zap.String("listen", cfg.GetAgentListenAddr()),
		zap.String("server", cfg.GetServerAddr()),
	)

	// 创建并启动agent服务
	agent := service.NewAgent(cfg)
	if err := agent.Start(); err != nil {
		logger.Fatal("Agent启动失败", zap.Error(err))
	}

	// 等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("收到关闭信号，Agent正在关闭...")
	agent.Stop()
	logger.Info("Agent已关闭")
}

// printUsage 打印帮助信息
func printUsage() {
	fmt.Fprintf(os.Stderr, `Orbit Agent

版本: %s

用法:
  orbit-agent [选项]

选项:
  --config string       配置文件路径 (默认: /etc/fastdp-orbit/agent.toml)
  --server string       Server gRPC地址 (host:port)
  --port int            Agent监听端口 (覆盖配置文件)
  --host string         Agent监听地址 + 上报给Server的IP (覆盖配置文件)
  -h, --help            显示帮助信息

示例:
  # 使用默认配置
  orbit-agent

  # 指定配置文件
  orbit-agent --config /path/to/agent.toml

  # 指定Server地址和监听IP
  orbit-agent --server 10.0.0.1:9090 --host 192.168.1.100

  # 覆盖端口
  orbit-agent --server 10.0.0.1:9090 --host 192.168.1.100 --port 8701

配置文件优先级: 命令行参数 > 配置文件 > 默认值
`, version.Version)
}
