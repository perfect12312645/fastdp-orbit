package grpc

import (
	"context"
	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/models/machine"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/proto/server"
	"fastdp-orbit/backend/server/cache"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// ServerGRPCServer server端的gRPC服务实现
type ServerGRPCServer struct {
	server.UnimplementedServerServiceServer
	cfg   *config.ServerConfig
	cache *cache.MachineCache
}

// NewServerGRPCServer 创建server端gRPC服务
func NewServerGRPCServer(cfg *config.ServerConfig, cache *cache.MachineCache) *ServerGRPCServer {
	return &ServerGRPCServer{cfg: cfg, cache: cache}
}

// Register agent注册
func (s *ServerGRPCServer) Register(ctx context.Context, req *server.RegisterRequest) (*server.RegisterResponse, error) {
	logger.Info("收到agent注册请求",
		zap.String("ip", req.Ip),
		zap.Int32("port", req.Port),
	)

	// 校验Token
	if s.cfg.GRPC.Token != "" && req.Token != s.cfg.GRPC.Token {
		logger.Warn("agent注册失败: Token不匹配",
			zap.String("ip", req.Ip),
		)
		return &server.RegisterResponse{
			Success: false,
			Message: "Token不匹配，注册被拒绝",
		}, nil
	}

	// 构建内存快照
	snap := &cache.MachineSnapshot{
		IP:   req.Ip,
		Port: int(req.Port),
	}

	if req.EnvInfo != nil {
		snap.Hostname = req.EnvInfo.Hostname
		snap.Virtualization = req.EnvInfo.Virtualization
		snap.OSName = req.EnvInfo.Os.GetName()
		snap.OSVersion = req.EnvInfo.Os.GetVersion()
		snap.Kernel = req.EnvInfo.Os.GetKernel()
		snap.Arch = req.EnvInfo.Os.GetArch()
		snap.CPUModel = req.EnvInfo.Cpu.GetModel()
		snap.CPUCores = req.EnvInfo.Cpu.GetCores()
		snap.MemoryKB = req.EnvInfo.Memory.GetTotalKb()
		snap.SwapKB = req.EnvInfo.Swap.GetTotalKb()
		snap.Gateway = req.EnvInfo.Gateway
		snap.Timezone = req.EnvInfo.Timezone

		if req.EnvInfo.Firewall != nil {
			snap.FirewallStatus = req.EnvInfo.Firewall.Status
			snap.FirewallEnabled = req.EnvInfo.Firewall.Enabled
		}

		// 磁盘信息
		for _, d := range req.EnvInfo.Disks {
			snap.Disks = append(snap.Disks, machine.MachineDisk{
				Device:  d.Device,
				Type:    d.Type,
				TotalGB: d.TotalGb,
			})
		}

		// 网卡信息
		for _, n := range req.EnvInfo.Networks {
			snap.Networks = append(snap.Networks, machine.MachineNetwork{
				Name:   n.Name,
				IP:     n.Ip,
				MAC:    n.Mac,
				Speed:  n.Speed,
				Status: n.Status,
			})
		}

		// GPU信息
		for _, g := range req.EnvInfo.Gpus {
			snap.GPUs = append(snap.GPUs, machine.MachineGPU{
				Name:          g.Name,
				Count:         g.Count,
				DriverVersion: g.DriverVersion,
			})
		}

		// 动态信息（仅内存）
		snap.UptimeSeconds = req.EnvInfo.UptimeSeconds
		snap.SystemTime = req.EnvInfo.SystemTime
		snap.HardwareTime = req.EnvInfo.HardwareTime
	}

	// 写入缓存和数据库
	if err := s.cache.Register(snap); err != nil {
		logger.Error("注册写入失败", zap.Error(err), zap.String("ip", req.Ip))
		return &server.RegisterResponse{
			Success: false,
			Message: "注册写入失败",
		}, nil
	}

	return &server.RegisterResponse{
		Success: true,
		Message: "注册成功",
	}, nil
}

// Heartbeat agent心跳
func (s *ServerGRPCServer) Heartbeat(ctx context.Context, req *server.HeartbeatRequest) (*server.HeartbeatResponse, error) {
	logger.Debug("收到心跳", zap.String("ip", req.Ip), zap.Int32("port", req.Port))

	// 检查机器是否在缓存中（已注册）
	if !s.cache.HasMachine(req.Ip, int(req.Port)) {
		logger.Warn("心跳收到已删除或未注册的机器，通知退出", zap.String("ip", req.Ip), zap.Int32("port", req.Port))
		return &server.HeartbeatResponse{
			Success:   true,
			AgentExit: true,
		}, nil
	}

	// 更新缓存（含动态信息）
	s.cache.Heartbeat(req.Ip, int(req.Port), req.UptimeSeconds, req.SystemTime, req.HardwareTime)
	return &server.HeartbeatResponse{
		Success: true,
	}, nil
}

// CopyFile Agent请求文件，Server流式返回文件块
func (s *ServerGRPCServer) CopyFile(req *server.CopyFileRequest, stream server.ServerService_CopyFileServer) error {
	logger.Info("收到文件传输请求", zap.String("source_path", req.SourcePath))

	file, err := os.Open(req.SourcePath)
	if err != nil {
		logger.Error("打开文件失败", zap.Error(err), zap.String("path", req.SourcePath))
		return err
	}
	defer file.Close()

	buffer := make([]byte, 64*1024)

	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Error("读取文件失败", zap.Error(err))
			return err
		}

		chunk := &server.FileChunk{
			Data: buffer[:n],
		}

		if err := stream.Send(chunk); err != nil {
			logger.Error("发送文件块失败", zap.Error(err))
			return err
		}
	}

	logger.Info("文件传输完成", zap.String("path", req.SourcePath))
	return nil
}

// ListDirectory Agent获取目录列表
func (s *ServerGRPCServer) ListDirectory(ctx context.Context, req *server.ListDirectoryRequest) (*server.ListDirectoryResponse, error) {
	logger.Debug("获取目录列表", zap.String("path", req.Path))

	files := make([]*server.FileInfo, 0)

	err := filepath.Walk(req.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !req.Recursive {
			relPath, _ := filepath.Rel(req.Path, path)
			if relPath != "." && filepath.Dir(relPath) != "." {
				return filepath.SkipDir
			}
		}

		relPath, _ := filepath.Rel(req.Path, path)
		if relPath == "." {
			return nil
		}

		fileInfo := &server.FileInfo{
			Path:         relPath,
			IsDir:        info.IsDir(),
			Size:         info.Size(),
			ModifiedTime: info.ModTime().Unix(),
			Permissions:  info.Mode().String(),
		}

		files = append(files, fileInfo)
		return nil
	})

	if err != nil {
		logger.Error("获取目录列表失败", zap.Error(err), zap.String("path", req.Path))
		return nil, err
	}

	return &server.ListDirectoryResponse{
		Files: files,
	}, nil
}

// RegisterGRPCServer 注册gRPC服务到server
func RegisterGRPCServer(serverGrpc *grpc.Server, cfg *config.ServerConfig, db *gorm.DB, mc *cache.MachineCache) *ServerGRPCServer {
	svc := NewServerGRPCServer(cfg, mc)
	server.RegisterServerServiceServer(serverGrpc, svc)
	return svc
}
