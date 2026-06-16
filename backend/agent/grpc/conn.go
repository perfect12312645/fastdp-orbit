package grpc

import (
	"context"
	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/tlsutil"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
)

var (
	globalConn *grpc.ClientConn
	connMux    sync.Mutex
	serverAddr string
)

// GetClientConn 获取gRPC连接（自动初始化/重建）
func GetClientConn(cfg *config.AgentConfig) (*grpc.ClientConn, error) {
	serverAddr = cfg.GetServerAddr()
	logger.Info("初始化 server 地址", zap.String("address", serverAddr))

	connMux.Lock()
	defer connMux.Unlock()

	if globalConn != nil {
		state := globalConn.GetState()
		if state == connectivity.Ready || state == connectivity.Idle {
			return globalConn, nil
		}
		if state == connectivity.Shutdown {
			logger.Warn("旧连接已关闭，准备重建连接")
			if err := globalConn.Close(); err != nil {
				logger.Warn("关闭旧连接失败", zap.Error(err))
			}
			globalConn = nil
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if !globalConn.WaitForStateChange(ctx, state) {
				logger.Warn("连接状态无变化，准备重建连接", zap.String("state", state.String()))
				globalConn = nil
			} else if globalConn.GetState() == connectivity.Ready {
				return globalConn, nil
			}
		}
	}

	logger.Info("开始创建新连接", zap.String("address", serverAddr))

	// 创建连接选项
	var opts []grpc.DialOption

	// 配置keepalive
	kp := keepalive.ClientParameters{
		Time:                time.Duration(cfg.OrbitAgent.Heartbeat) * time.Second,
		Timeout:             5 * time.Second,
		PermitWithoutStream: true,
	}
	opts = append(opts, grpc.WithKeepaliveParams(kp))

	// 配置双向TLS
	if cfg.OrbitAgent.TLS.Enabled {
		creds, err := tlsutil.LoadClientTLSCredentials(cfg.OrbitAgent.TLS)
		if err != nil {
			logger.Error("加载客户端TLS配置失败", zap.Error(err))
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
		logger.Info("已启用双向TLS", zap.String("ca_file", cfg.OrbitAgent.TLS.CAFile))
	} else {
		// 使用不安全的传输凭证
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// 配置自定义拨号器
	opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 5*time.Second)
	}))

	newConn, err := grpc.NewClient(serverAddr, opts...)
	if err != nil {
		logger.Error("创建新连接失败", zap.Error(err))
		return nil, err
	}

	globalConn = newConn
	logger.Info("新连接创建成功", zap.String("state", globalConn.GetState().String()))
	return globalConn, nil
}

// CloseConn 关闭全局连接
func CloseConn() error {
	connMux.Lock()
	defer connMux.Unlock()
	if globalConn != nil {
		logger.Info("开始关闭全局 gRPC 连接")
		err := globalConn.Close()
		globalConn = nil
		return err
	}
	return nil
}
