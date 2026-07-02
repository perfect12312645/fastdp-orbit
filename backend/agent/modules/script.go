package modules

import (
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
	"fmt"
	"os"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

// ScriptModule 脚本执行模块（通过临时文件执行，支持 heredoc 和大脚本）
type ScriptModule struct{}

func NewScriptModule() Module {
	return &ScriptModule{}
}

func init() {
	Register("script", NewScriptModule)
}

const (
	ScriptErrInvalidParams = 900 // 参数错误
	ScriptErrWriteFailed   = 901 // 写临时文件失败
	ScriptErrExecFailed    = 902 // 执行失败
)

// Run 执行脚本：优先使用 script_file（已有脚本路径），否则写临时文件执行 script_content
func (m *ScriptModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	scriptContent := req.Parameters["script"]
	scriptFile := req.Parameters["script_file"]

	// 记录接收到的参数
	logger.Info("开始执行脚本", zap.String("task_id", req.TaskId), zap.String("machine_id", req.MachineId))
	logger.Info("脚本参数", zap.String("script_content_length", fmt.Sprintf("%d", len(scriptContent))), zap.String("script_file", scriptFile))

	// 至少指定一个
	if scriptContent == "" && scriptFile == "" {
		return utils.ErrorResponse(req, ScriptErrInvalidParams, "必须指定 script（脚本内容）或 script_file（脚本路径）"), nil
	}

	// 如果指定了 script_file，直接执行
	if scriptFile != "" {
		if !utils.FileExists(scriptFile) {
			return utils.ErrorResponse(req, ScriptErrInvalidParams, fmt.Sprintf("脚本文件不存在: %s", scriptFile)), nil
		}
		logger.Info("直接执行脚本文件", zap.String("script_file", scriptFile))
		return executeScriptFile(req, scriptFile)
	}

	// 写临时文件 → 执行 → 清理
	logger.Info("准备写入临时脚本文件", zap.String("script_content_length", fmt.Sprintf("%d", len(scriptContent))))
	return executeScriptContent(req, scriptContent)
}

// executeScriptContent 写入临时文件并执行
func executeScriptContent(req *agent.ExecRequest, content string) (*agent.ExecResponse, error) {
	tmpFile := fmt.Sprintf("/tmp/orbit_script_%d_%d.sh", time.Now().UnixNano(), os.Getpid())

	// 写入脚本内容
	if err := os.WriteFile(tmpFile, []byte(content), 0755); err != nil {
		logger.Error("写临时脚本文件失败", zap.Error(err), zap.String("tmp_file", tmpFile))
		return utils.ErrorResponse(req, ScriptErrWriteFailed, fmt.Sprintf("写临时脚本文件失败: %v", err)), nil
	}
	defer os.Remove(tmpFile)

	// 验证文件是否创建成功
	if _, err := os.Stat(tmpFile); err != nil {
		logger.Error("临时脚本文件创建失败", zap.Error(err), zap.String("tmp_file", tmpFile))
		return utils.ErrorResponse(req, ScriptErrWriteFailed, fmt.Sprintf("临时脚本文件创建失败: %v", err)), nil
	}
	logger.Info("临时脚本文件创建成功", zap.String("tmp_file", tmpFile))

	return runBash(req, tmpFile)
}

// executeScriptFile 直接执行指定路径的脚本文件
func executeScriptFile(req *agent.ExecRequest, scriptFile string) (*agent.ExecResponse, error) {
	return runBash(req, scriptFile)
}

// runBash 执行 bash 脚本文件
func runBash(req *agent.ExecRequest, scriptFile string) (*agent.ExecResponse, error) {
	cmd := exec.Command("bash", scriptFile)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		logger.Error("脚本执行失败", zap.Error(err), zap.String("output", outputStr))
		exitCode := -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return &agent.ExecResponse{
			MachineId: req.MachineId,
			TaskId:    req.TaskId,
			Success:   false,
			Stdout:    outputStr,
			Changed:   false,
			Error: &agent.ErrorDetail{
				Code:    int32(exitCode),
				Message: "脚本执行失败",
				Trace:   err.Error(),
			},
		}, nil
	}

	logger.Info("脚本执行成功", zap.String("output", outputStr))

	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    outputStr,
		Changed:   true,
	}, nil
}
