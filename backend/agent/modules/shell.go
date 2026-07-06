package modules

import (
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// ShellModule 执行单条 shell 命令的模块
type ShellModule struct{}

func NewShellModule() Module {
	return &ShellModule{}
}

func init() {
	Register("shell", NewShellModule)
}

func (m *ShellModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	cmd := req.Parameters["command"]
	if cmd == "" {
		return utils.ErrorResponse(req, 1000, "必须指定 command 参数"), nil
	}

	cmd = strings.ReplaceAll(cmd, `\"`, `"`)

	output, err := exec.Command("/bin/bash", "-c", cmd).CombinedOutput()
	outputStr := string(output)

	if err != nil {
		logger.Error("命令执行失败", zap.Error(err), zap.String("output", outputStr))
		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return utils.ErrorResponse(req, int32(exitCode), "命令执行失败"), nil
	}

	logger.Info("命令执行成功", zap.String("output", outputStr))

	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    outputStr,
		Changed:   true,
	}, nil
}
