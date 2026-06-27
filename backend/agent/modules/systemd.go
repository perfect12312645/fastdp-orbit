package modules

import (
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SystemdModule 实现Module接口，用于管理systemd服务
type SystemdModule struct{}

// NewSystemdModule 创建systemd模块实例
func NewSystemdModule() Module {
	return &SystemdModule{}
}

// 注册模块（唯一名称"systemd"）
func init() {
	Register("systemd", NewSystemdModule)
}

// 支持的systemd操作类型
const (
	actionStart   = "start"
	actionStop    = "stop"
	actionRestart = "restart"
	actionReload  = "reload"
	actionStatus  = "status"
	actionEnable  = "enable"
	actionDisable = "disable"
)
const (
	verifyRetryCount = 3               // 校验重试次数
	verifyInterval   = 2 * time.Second // 重试间隔
)

// 系统d模块错误码（添加前缀避免冲突）
const (
	SystemdErrInvalidParams = 600 // 参数错误
	SystemdErrCmdFailed     = 601 // 命令执行失败
)

// Run 实现Module接口的Run方法
func (m *SystemdModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 参数解析与校验
	serviceName, nameOk := req.Parameters["name"] // 服务名称（必填）
	action, actionOk := req.Parameters["action"]  // 操作类型（必填）

	// 校验必填参数
	if action != actionReload && !nameOk {
		return utils.ErrorResponse(req, SystemdErrInvalidParams, "未传入服务名称（参数name）"), nil
	}
	if !actionOk {
		return utils.ErrorResponse(req, SystemdErrInvalidParams, "未传入操作类型（参数action）"), nil
	}

	// 校验操作类型是否支持
	supportedActions := map[string]bool{
		actionStart:   true,
		actionStop:    true,
		actionRestart: true,
		actionReload:  true,
		actionStatus:  true,
		actionEnable:  true,
		actionDisable: true,
	}
	if !supportedActions[action] {
		return utils.ErrorResponse(req, SystemdErrInvalidParams,
			fmt.Sprintf("不支持的操作类型: %s，支持的操作：start/stop/restart/reload/status/enable/disable", action)), nil
	}

	// 2. 执行对应systemd命令
	output, _, err := execSystemdCommand(serviceName, action)
	if err != nil {
		// 命令执行失败（如权限不足、服务不存在）
		return utils.ErrorResponse(req, SystemdErrCmdFailed,
			fmt.Sprintf("操作失败: %s，错误：%v，输出：%s", action, err, output)), nil
	}

	// 4. 返回成功响应
	return utils.SuccessResponse(
		req,
		output,
	), nil

}

// execSystemdCommand 执行systemctl命令并返回输出（增加状态预检查）
func execSystemdCommand(serviceName, action string) (string, bool, error) {
	// 1. 针对reload（daemon-reload）的特殊处理
	if action == actionReload {
		// 执行daemon-reload（全局配置重载，无需服务名称）
		cmd := exec.Command("systemctl", "daemon-reload")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		logger.Info("执行systemd命令",
			zap.String("action", "daemon-reload"),
			zap.String("output", outputStr))

		if err != nil {
			return outputStr, false, fmt.Errorf("daemon-reload执行失败: %w", err)
		}
		// daemon-reload每次执行都算变更
		return "daemon-reload执行成功，已重新加载systemd配置", true, nil
	}

	// 1. 先检查服务是否存在
	exists, err := checkServiceExists(serviceName)
	if err != nil {
		return "", false, fmt.Errorf("检查服务是否存在失败: %w", err)
	}
	if !exists {
		return "", false, fmt.Errorf("服务不存在: %s", serviceName)
	}

	// 2. 针对不同action，检查当前状态是否已符合预期（若符合则不执行命令）
	switch action {
	case actionStart:
		// 检查服务是否已启动（active）
		if isActive, err := isServiceActive(serviceName); err != nil {
			return "", false, fmt.Errorf("检查服务活性失败: %w", err)
		} else if isActive {
			return "服务已处于启动状态，无需重复执行start", false, nil // 已符合预期，不执行命令
		}

	case actionStop:
		// 检查服务是否已停止（inactive）
		if isActive, err := isServiceActive(serviceName); err != nil {
			return "", false, fmt.Errorf("检查服务活性失败: %w", err)
		} else if !isActive {
			return "服务已处于停止状态，无需重复执行stop", false, nil // 已符合预期，不执行命令
		}

	case actionEnable:
		// 检查服务是否已启用（enabled）
		if isEnabled, err := isServiceEnabled(serviceName); err != nil {
			return "", false, fmt.Errorf("检查服务启用状态失败: %w", err)
		} else if isEnabled {
			return "服务已处于启用状态，无需重复执行enable", false, nil // 已符合预期，不执行命令
		}

	case actionDisable:
		// 检查服务是否已禁用（disabled）
		if isEnabled, err := isServiceEnabled(serviceName); err != nil {
			return "", false, fmt.Errorf("检查服务启用状态失败: %w", err)
		} else if !isEnabled {
			return "服务已处于禁用状态，无需重复执行disable", false, nil // 已符合预期，不执行命令
		}

	// 对于restart/reload/status，暂不做预检查（restart通常需要强制执行，status是查询操作）
	case actionRestart, actionReload, actionStatus:
		break
	}

	// 3. 状态不符合预期，执行实际命令
	cmdArgs := []string{action, serviceName}
	cmd := exec.Command("systemctl", cmdArgs...)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	logger.Info("执行systemd命令",
		zap.String("service", serviceName),
		zap.String("action", action),
		zap.String("output", outputStr))

	if err != nil {
		return outputStr, false, fmt.Errorf("systemctl命令执行失败: %w", err)
	}
	switch action {
	case actionStart, actionRestart:
		// 校验服务是否真正启动
		if err := verifyServiceState(serviceName, true, verifyRetryCount, verifyInterval); err != nil {
			return outputStr, false, fmt.Errorf("启动校验失败: %w，命令输出: %s", err, outputStr)
		}
	case actionStop:
		// 校验服务是否真正停止
		if err := verifyServiceState(serviceName, false, verifyRetryCount, verifyInterval); err != nil {
			return outputStr, false, fmt.Errorf("停止校验失败: %w，命令输出: %s", err, outputStr)
		}
	}
	if action == actionStatus {
		return outputStr, false, nil
	}
	return outputStr, true, nil

}

func verifyServiceState(serviceName string, expectedActive bool, retry int, interval time.Duration) error {
	for i := 0; i <= retry; i++ {
		isActive, err := isServiceActive(serviceName)
		if err != nil {
			logger.Warn("服务状态校验失败",
				zap.String("service", serviceName),
				zap.Error(err))
			if i == retry { // 最后一次重试失败
				return fmt.Errorf("校验服务状态失败: %w", err)
			}
			time.Sleep(interval)
			continue
		}

		// 状态符合预期
		if isActive == expectedActive {
			logger.Info("服务状态校验通过",
				zap.String("service", serviceName),
				zap.Bool("expectedActive", expectedActive),
				zap.Int("retryCount", i))
			return nil
		}

		// 状态不符合预期，继续重试
		if i < retry {
			logger.Warn("服务状态未达预期，将重试",
				zap.String("service", serviceName),
				zap.Bool("currentActive", isActive),
				zap.Bool("expectedActive", expectedActive),
				zap.Int("remainingRetry", retry-i))
			time.Sleep(interval)
		}
	}

	// 所有重试后仍不符合预期
	return fmt.Errorf("经过%d次重试，服务状态仍未达到预期（期望活跃状态: %v）", retry, expectedActive)
}

// checkServiceExists 检查服务是否存在
func checkServiceExists(serviceName string) (bool, error) {
	// 使用 systemctl cat 命令检查服务单元文件
	cmd := exec.Command("systemctl", "cat", serviceName)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// 处理命令执行错误
	if err != nil {
		if strings.Contains(strings.ToLower(outputStr), "no files found") ||
			strings.Contains(strings.ToLower(outputStr), "is not loaded") {
			return false, nil // 服务不存在
		}
		// 其他错误（如权限不足、命令不存在等）
		return false, fmt.Errorf("failed to check service: %v, output: %s", err, outputStr)
	}

	// 命令成功执行，服务存在
	return true, nil
}

// isServiceActive 检查服务是否处于活跃状态（active）
func isServiceActive(serviceName string) (bool, error) {
	// systemctl is-active返回"active"表示活跃，其他为非活跃（inactive/failed等）
	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	// 命令执行成功且输出为"active"，则返回true
	if err == nil && outputStr == "active" {
		return true, nil
	}

	// 命令执行失败（如状态为inactive）或输出非"active"，返回false
	return false, nil
}

// isServiceEnabled 检查服务是否已启用（开机启动）
func isServiceEnabled(serviceName string) (bool, error) {
	// systemctl is-enabled返回"enabled"表示已启用
	cmd := exec.Command("systemctl", "is-enabled", serviceName)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	// 命令执行成功且输出为"enabled"，则返回true
	if err == nil && outputStr == "enabled" {
		return true, nil
	}

	// 其他情况（disabled/masked等）返回false
	return false, nil
}
