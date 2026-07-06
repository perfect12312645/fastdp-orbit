package modules

import (
	"fmt"
	"sync"

	"fastdp-orbit/backend/proto/agent"
)

// Module 模块接口，所有模块实现此接口
type Module interface {
	// Run 执行模块逻辑
	Run(req *agent.ExecRequest) (*agent.ExecResponse, error)
}

var (
	registry = make(map[string]func() Module)
	mu       sync.RWMutex
)

// Register 注册模块（在 init 函数中调用，每个模块自己注册）
func Register(name string, factory func() Module) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = factory
}

// GetModule 通过模块名获取实例
func GetModule(name string) (Module, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("未知模块: %s", name)
	}
	return factory(), nil
}

// ListModules 列出所有已注册的模块
func ListModules() []string {
	mu.RLock()
	defer mu.RUnlock()
	modules := make([]string, 0, len(registry))
	for name := range registry {
		modules = append(modules, name)
	}
	return modules
}
