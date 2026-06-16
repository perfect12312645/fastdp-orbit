package grpc

import (
	"fastdp-orbit/backend/pkg/logger"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// AgentConnPool Agent gRPC 连接池
type AgentConnPool struct {
	mu    sync.RWMutex
	conns map[string]*grpc.ClientConn // key: "ip:port"
}

// NewAgentConnPool 创建连接池
func NewAgentConnPool() *AgentConnPool {
	return &AgentConnPool{
		conns: make(map[string]*grpc.ClientConn),
	}
}

// GetConn 获取或创建连接（复用）
func (p *AgentConnPool) GetConn(addr string) (*grpc.ClientConn, error) {
	// 读锁检查
	p.mu.RLock()
	if conn, ok := p.conns[addr]; ok {
		state := conn.GetState()
		if state != connectivity.Shutdown {
			p.mu.RUnlock()
			return conn, nil
		}
		// 连接已关闭，需要重建
		p.mu.RUnlock()
		p.mu.Lock()
		defer p.mu.Unlock()
		// 双重检查
		if conn, ok := p.conns[addr]; ok && conn.GetState() != connectivity.Shutdown {
			return conn, nil
		}
		// 关闭旧连接
		conn.Close()
		delete(p.conns, addr)
	} else {
		p.mu.RUnlock()
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	// 创建新连接
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("创建Agent连接失败", zap.String("addr", addr), zap.Error(err))
		return nil, err
	}

	p.conns[addr] = conn
	logger.Debug("创建Agent连接", zap.String("addr", addr))
	return conn, nil
}

// RemoveConn 移除连接（Agent离线时调用）
func (p *AgentConnPool) RemoveConn(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.conns[addr]; ok {
		conn.Close()
		delete(p.conns, addr)
		logger.Debug("移除Agent连接", zap.String("addr", addr))
	}
}

// Close 关闭所有连接
func (p *AgentConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, conn := range p.conns {
		conn.Close()
		delete(p.conns, addr)
	}
	logger.Info("Agent连接池已关闭")
}
