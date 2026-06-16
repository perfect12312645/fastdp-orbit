package modules

import (
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/proto/agent"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// ShellModule 执行shell命令的模块
type ShellModule struct{}

// NewShellModule 创建Shell模块实例
func NewShellModule() Module {
	return &ShellModule{}
}

// Run 执行shell命令
func (m *ShellModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 获取命令和参数
	scriptContent := req.Parameters["script"]
	fullCmd := req.Parameters["command"]

	var executeContent string
	if scriptContent != "" {
		executeContent = scriptContent
		logger.Info("执行多行脚本", zap.String("script", executeContent))
	} else if fullCmd != "" {
		executeContent = strings.ReplaceAll(fullCmd, `\"`, `"`)
		logger.Info("执行单条命令", zap.String("command", executeContent))
	} else {
		logger.Error("未指定command或script参数")
		return &agent.ExecResponse{
			MachineId: req.MachineId,
			TaskId:    req.TaskId,
			Success:   false,
			Error: &agent.ErrorDetail{
				Code:    1000,
				Message: "必须指定command（单条命令）或script（多行脚本）参数",
			},
		}, nil
	}

	// 使用shell执行命令（支持管道等特性）
	cmd := exec.Command("/bin/bash", "-c", executeContent)

	// 捕获输出
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// 处理执行结果
	if err != nil {
		logger.Error("命令执行失败", zap.Error(err), zap.String("output", outputStr))

		// 获取退出码
		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}

		return &agent.ExecResponse{
			MachineId: req.MachineId,
			TaskId:    req.TaskId,
			Success:   false,
			Stdout:    outputStr,
			Error: &agent.ErrorDetail{
				Code:    int32(exitCode),
				Message: "命令执行失败",
				Trace:   err.Error(),
			},
		}, nil
	}

	logger.Info("命令执行成功", zap.String("output", outputStr))

	// 返回成功响应
	changedDetails := ""
	if fullCmd != "" {
		changedDetails = "执行命令: " + fullCmd
	} else {
		changedDetails = "执行脚本"
	}

	return &agent.ExecResponse{
		MachineId:      req.MachineId,
		TaskId:         req.TaskId,
		Success:        true,
		Stdout:         outputStr,
		Changed:        true,
		ChangedDetails: changedDetails,
	}, nil
}

func init() {
	Register("shell", NewShellModule)
}
