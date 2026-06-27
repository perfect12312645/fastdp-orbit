package modules

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// ModprobeModule 实现内核模块加载功能（带幂等性）
type ModprobeModule struct{}

// NewModprobeModule 创建模块实例
func NewModprobeModule() Module {
	return &ModprobeModule{}
}

// 操作动作常量
const (
	actionLoad   = "load"   // 加载模块（默认）
	actionRemove = "remove" // 移除模块
)

// 参数结构定义
type modprobeParams struct {
	Module  string   // 单个模块名称
	Loop    []string // 模块列表（循环加载）
	Action  string   // 操作动作：load/remove
	Options string   // 模块加载选项（可选）
}

// Run 实现模块核心逻辑
func (m *ModprobeModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 解析并验证参数
	params, err := parseModprobeParams(req.Parameters)
	if err != nil {
		return utils.ErrorResponse(req, 4001, fmt.Sprintf("参数解析失败: %v", err)), nil
	}

	// 2. 执行操作（带幂等性检查）
	result, detail, err := m.executeAction(params)
	if err != nil {
		logger.Error("模块操作失败",
			zap.String("action", params.Action),
			zap.Error(err))
		return utils.ErrorResponse(req, 4002, err.Error()), nil
	}

	// 3. 返回结果
	if result {
		return utils.SuccessResponse(req, detail), nil
	}
	return utils.SuccessResponseWithNoChange(req, detail), nil
}

// 解析并验证参数
func parseModprobeParams(params map[string]string) (modprobeParams, error) {
	// 解析操作动作（默认加载）
	action := params["action"]
	if action == "" {
		action = actionLoad
	}
	if action != actionLoad && action != actionRemove {
		return modprobeParams{}, errors.New("action参数必须为'load'或'remove'")
	}

	// 解析模块列表（优先使用loop参数）
	var loop []string
	if loopStr, ok := params["loop"]; ok && loopStr != "" {
		loop = strings.Split(loopStr, ",")
		for i, item := range loop {
			loop[i] = strings.TrimSpace(item)
		}
	}

	// 解析单个模块（module与loop二选一）
	module := params["module"]
	if len(loop) == 0 && module == "" {
		return modprobeParams{}, errors.New("必须指定module参数或loop参数（模块列表）")
	}

	return modprobeParams{
		Module:  module,
		Loop:    loop,
		Action:  action,
		Options: params["options"],
	}, nil
}

// 执行具体操作（增加幂等性检查）
func (m *ModprobeModule) executeAction(params modprobeParams) (changed bool, detail string, err error) {
	// 确定要操作的模块列表
	var modules []string
	if len(params.Loop) > 0 {
		modules = params.Loop
	} else {
		modules = []string{params.Module}
	}

	var results []string
	var allSuccess bool = true

	// 遍历模块执行操作（带幂等性检查）
	for _, module := range modules {
		if module == "" {
			results = append(results, "跳过空模块名称")
			continue
		}

		// 检查模块当前状态（是否已加载）
		loaded, err := m.checkModuleLoaded(module)
		if err != nil {
			results = append(results, fmt.Sprintf("模块[%s]状态检查失败: %v", module, err))
			allSuccess = false
			continue
		}

		// 根据动作和当前状态决定是否执行操作
		switch params.Action {
		case actionLoad:
			if loaded {
				// 已加载，无需操作
				results = append(results, fmt.Sprintf("模块[%s]已加载，无需重复操作", module))
				continue
			}
			// 未加载，执行加载
			output, err := m.runModprobeCommand(module, params)
			if err != nil {
				results = append(results, fmt.Sprintf("模块[%s]加载失败: %v", module, err))
				allSuccess = false
				continue
			}
			results = append(results, fmt.Sprintf("模块[%s]加载成功: %s", module, output))
			changed = true

		case actionRemove:
			if !loaded {
				// 未加载，无需操作
				results = append(results, fmt.Sprintf("模块[%s]未加载，无需移除", module))
				continue
			}
			// 已加载，执行移除
			output, err := m.runModprobeCommand(module, params)
			if err != nil {
				results = append(results, fmt.Sprintf("模块[%s]移除失败: %v", module, err))
				allSuccess = false
				continue
			}
			results = append(results, fmt.Sprintf("模块[%s]移除成功: %s", module, output))
			changed = true
		}
	}

	// 汇总结果
	if !allSuccess {
		return changed, strings.Join(results, "; "), fmt.Errorf("部分模块操作失败")
	}
	return changed, strings.Join(results, "; "), nil
}

// 检查模块是否已加载
func (m *ModprobeModule) checkModuleLoaded(module string) (bool, error) {
	// 执行lsmod命令，获取已加载模块列表
	cmd := exec.Command("lsmod")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("执行lsmod失败: %v, 输出: %s", err, string(output))
	}

	// 解析输出（lsmod第一列是模块名）
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// 按空格分割，取第一个字段（模块名）
		parts := strings.Fields(line)
		if len(parts) > 0 && parts[0] == module {
			return true, nil
		}
	}

	// 检查扫描错误
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("解析lsmod输出失败: %v", err)
	}

	// 未找到模块
	return false, nil
}

// 执行modprobe命令
func (m *ModprobeModule) runModprobeCommand(module string, params modprobeParams) (string, error) {
	args := []string{}
	if params.Action == actionRemove {
		args = append(args, "-r") // 移除模块
	}

	// 添加模块选项
	if params.Options != "" {
		args = append(args, params.Options)
	}

	args = append(args, module)

	// 执行命令
	cmd := exec.Command("modprobe", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("命令执行失败: %v, 输出: %s", err, string(output))
	}

	return string(output), nil
}

// 模块注册
func init() {
	Register("modprobe", NewModprobeModule)
}
