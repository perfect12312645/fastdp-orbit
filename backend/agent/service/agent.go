package service

import (
	"context"
	"fastdp-orbit/backend/agent/grpc"
	"fastdp-orbit/backend/agent/handler"
	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/proto/agent"
	"fastdp-orbit/backend/proto/server"
	"fmt"
	"net"
	"os"
	"time"

	"go.uber.org/zap"
	gogrpc "google.golang.org/grpc"
)

// Agent represents the agent service
type Agent struct {
	cfg     *config.AgentConfig
	handler *handler.Handler
	stopCh  chan struct{}
	grpcSrv *gogrpc.Server
}

// NewAgent creates a new agent instance
func NewAgent(cfg *config.AgentConfig) *Agent {
	return &Agent{
		cfg:     cfg,
		handler: handler.NewHandler(),
		stopCh:  make(chan struct{}),
	}
}

// Start starts the agent service
func (a *Agent) Start() error {
	// 连接server
	conn, err := grpc.GetClientConn(a.cfg)
	if err != nil {
		logger.Error("连接server失败", zap.Error(err))
		return err
	} else {
		client := server.NewServerServiceClient(conn)

		// 注册到server
		if err := a.register(context.Background(), client); err != nil {
			logger.Error("注册到server失败", zap.Error(err))
			return err
		}

		// 启动心跳
		go a.heartbeat(context.Background(), client)
	}

	// 启动本地gRPC服务（供server调用agent）
	go a.startGRPCServer()

	// 打印启动信息
	logger.Info("Agent启动完成", zap.String("listen", a.cfg.GetAgentListenAddr()))

	return nil
}

// Stop stops the agent service
func (a *Agent) Stop() {
	close(a.stopCh)
	if a.grpcSrv != nil {
		a.grpcSrv.GracefulStop()
	}
	grpc.CloseConn()
}

// register registers the agent to the server
func (a *Agent) register(ctx context.Context, client server.ServerServiceClient) error {
	logger.Info("正在注册到server...")

	// 获取系统信息
	sysInfo, err := a.handler.GetSystemInfo(ctx, &agent.SystemInfoRequest{
		MachineId: "local",
	})
	if err != nil {
		return err
	}

	// 转换为server.proto中的SystemEnvInfo
	envInfo := &server.SystemEnvInfo{}
	if sysInfo.EnvInfo != nil {
		envInfo.Hostname = sysInfo.EnvInfo.Hostname
		envInfo.Virtualization = sysInfo.EnvInfo.Virtualization
		envInfo.UptimeSeconds = sysInfo.EnvInfo.UptimeSeconds
		envInfo.Gateway = sysInfo.EnvInfo.Gateway
		envInfo.Timezone = sysInfo.EnvInfo.Timezone
		envInfo.SystemTime = sysInfo.EnvInfo.SystemTime
		envInfo.HardwareTime = sysInfo.EnvInfo.HardwareTime

		// 转换OS信息
		if sysInfo.EnvInfo.Os != nil {
			envInfo.Os = &server.SystemInfo{
				Name:    sysInfo.EnvInfo.Os.Name,
				Version: sysInfo.EnvInfo.Os.Version,
				Kernel:  sysInfo.EnvInfo.Os.Kernel,
				Arch:    sysInfo.EnvInfo.Os.Arch,
			}
		}

		// 转换CPU信息
		if sysInfo.EnvInfo.Cpu != nil {
			envInfo.Cpu = &server.CpuInfo{
				Model: sysInfo.EnvInfo.Cpu.Model,
				Cores: sysInfo.EnvInfo.Cpu.Cores,
			}
		}

		// 转换内存信息
		if sysInfo.EnvInfo.Memory != nil {
			envInfo.Memory = &server.MemoryInfo{
				TotalKb: sysInfo.EnvInfo.Memory.TotalKb,
			}
		}

		// 转换磁盘信息
		for _, disk := range sysInfo.EnvInfo.Disks {
			envInfo.Disks = append(envInfo.Disks, &server.DiskInfo{
				Device:  disk.Device,
				Type:    disk.Type,
				TotalGb: disk.TotalGb,
			})
		}

		// 转换网卡信息
		for _, net := range sysInfo.EnvInfo.Networks {
			envInfo.Networks = append(envInfo.Networks, &server.NetworkInfo{
				Name:   net.Name,
				Ip:     net.Ip,
				Mac:    net.Mac,
				Speed:  net.Speed,
				Status: net.Status,
			})
		}

		// 转换GPU信息
		for _, gpu := range sysInfo.EnvInfo.Gpus {
			envInfo.Gpus = append(envInfo.Gpus, &server.GpuInfo{
				Name:          gpu.Name,
				Count:         gpu.Count,
				DriverVersion: gpu.DriverVersion,
			})
		}

		// 转换防火墙信息
		if sysInfo.EnvInfo.Firewall != nil {
			envInfo.Firewall = &server.FirewallInfo{
				Status:  sysInfo.EnvInfo.Firewall.Status,
				Enabled: sysInfo.EnvInfo.Firewall.Enabled,
			}
		}

		// 转换Swap信息
		if sysInfo.EnvInfo.Swap != nil {
			envInfo.Swap = &server.SwapInfo{
				TotalKb: sysInfo.EnvInfo.Swap.TotalKb,
			}
		}
	}

	resp, err := client.Register(ctx, &server.RegisterRequest{
		Ip:      a.cfg.OrbitAgent.Host,
		Port:    int32(a.cfg.OrbitAgent.Port),
		Token:   a.cfg.OrbitAgent.Token,
		EnvInfo: envInfo,
	})
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("注册被server拒绝: %s", resp.Message)
	}

	logger.Info("注册成功")
	return nil
}

// heartbeat sends heartbeat to server
func (a *Agent) heartbeat(ctx context.Context, client server.ServerServiceClient) {
	interval := time.Duration(a.cfg.OrbitAgent.Heartbeat) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			resp, err := client.Heartbeat(ctx, &server.HeartbeatRequest{
				Ip:            a.cfg.OrbitAgent.Host,
				Port:          int32(a.cfg.OrbitAgent.Port),
				UptimeSeconds: a.handler.GetUptime(),
				SystemTime:    a.handler.GetSystemTime(),
				HardwareTime:  a.handler.GetHardwareTime(),
			})
			if err != nil {
				logger.Error("心跳发送失败", zap.Error(err))
			} else if resp.GetAgentExit() {
				logger.Warn("收到Server退出指令，Agent即将退出")
				os.Exit(100)
			} else {
				logger.Debug("心跳发送成功")
			}
		case <-a.stopCh:
			return
		}
	}
}

// startGRPCServer starts the local gRPC server
func (a *Agent) startGRPCServer() {
	listenAddr := a.cfg.GetAgentListenAddr()
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Fatal("agent gRPC服务启动失败：监听端口失败", zap.String("address", listenAddr), zap.Error(err))
	}

	a.grpcSrv = gogrpc.NewServer()
	agent.RegisterAgentServiceServer(a.grpcSrv, a.handler)

	logger.Info("agent gRPC 服务启动成功", zap.String("listen_addr", listenAddr))

	if err := a.grpcSrv.Serve(lis); err != nil {
		logger.Fatal("gRPC 服务崩溃", zap.String("address", listenAddr), zap.Error(err))
	}
}
