package modules

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// SelinuxModule 简化版SELinux模块（适配k8s部署场景）
type SelinuxModule struct{}

// NewSelinuxModule 创建SELinux模块实例
func NewSelinuxModule() Module {
	return &SelinuxModule{}
}

// 注册模块
func init() {
	Register("selinux", NewSelinuxModule)
}

// 支持的操作类型（仅保留三个核心功能）
const (
	actionGetStatus  = "get_status"     // 查看当前状态（getenforce）
	actionSetPermiss = "set_permissive" // 临时关闭（setenforce 0）
	actionDisabled   = "disabled"       // 永久禁用（修改配置文件）
)

// 错误码
const (
	SelinuxErrInvalidParams = 700 // 参数错误
	SelinuxErrCmdFailed     = 701 // 命令执行失败
)

// Run 实现模块核心逻辑
func (m *SelinuxModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 检查系统是否为Ubuntu或不支持SELinux
	if !isSelinuxSupported() {
		return utils.SuccessResponseWithNoChange(
			req,
			"系统不支持SELinux,无需操作",
		), nil
	}

	// 2. 参数校验
	action, actionOk := req.Parameters["action"]
	if !actionOk {
		return utils.ErrorResponse(
			req,
			SelinuxErrInvalidParams,
			"未传入操作类型（参数action，支持：get_status/set_permissive/disabled）",
		), nil
	}

	supportedActions := map[string]bool{
		actionGetStatus:  true,
		actionSetPermiss: true,
		actionDisabled:   true,
	}
	if !supportedActions[action] {
		return utils.ErrorResponse(
			req,
			SelinuxErrInvalidParams,
			fmt.Sprintf("不支持的操作类型: %s，支持：get_status/set_permissive/disabled", action),
		), nil
	}

	// 3. 执行对应操作
	output, isChange, err := execSelinuxAction(action)
	if err != nil {
		return utils.ErrorResponse(
			req,
			SelinuxErrCmdFailed,
			fmt.Sprintf("操作失败: %v，输出：%s", err, output),
		), nil
	}

	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    output,
		Changed:   isChange,
	}, nil
}

// ------------------------------
// 核心操作实现
// ------------------------------

// execSelinuxAction 执行具体的SELinux操作
func execSelinuxAction(action string) (string, bool, error) {
	switch action {
	case actionGetStatus:
		// 查看当前状态（getenforce）
		return getSelinuxStatus()

	case actionSetPermiss:
		// 临时关闭（setenforce 0），先检查当前状态
		currentMode, _, err := getSelinuxStatus()
		if err != nil {
			return "", false, fmt.Errorf("获取当前状态失败: %w", err)
		}
		if strings.TrimSpace(currentMode) == "Permissive" || strings.TrimSpace(currentMode) == "Disabled" {
			return "SELinux已处于Permissive或Disabled模式，无需重复操作", false, nil
		}

		// 执行setenforce 0
		cmd := exec.Command("setenforce", "0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return string(output), false, fmt.Errorf("setenforce 0执行失败: %w", err)
		}
		return "SELinux已临时关闭（Permissive模式）", true, nil

	case actionDisabled:
		// 永久禁用（修改/etc/selinux/config）
		configPath := "/etc/selinux/config"
		if !utils.FileExists(configPath) {
			return "", false, fmt.Errorf("配置文件不存在: %s", configPath)
		}

		// 检查当前配置是否已为disabled
		currentVal, err := getSelinuxConfigValue(configPath)
		if err != nil {
			return "", false, fmt.Errorf("读取配置文件失败: %w", err)
		}
		if currentVal == "disabled" {
			return "SELinux已永久禁用，无需重复操作", false, nil
		}

		// 修改配置文件（SELINUX=disabled）
		if err := setSelinuxConfigValue(configPath, "disabled"); err != nil {
			return "", false, fmt.Errorf("修改配置文件失败: %w", err)
		}
		return "SELinux已配置为永久禁用（需重启生效）", true, nil
	}

	return "", false, fmt.Errorf("未知操作: %s", action)
}

// ------------------------------
// 辅助函数
// ------------------------------

// isSelinuxSupported 检查系统是否支持SELinux（存在核心工具和配置）
func isSelinuxSupported() bool {
	// 检查getenforce命令是否存在
	if _, err := exec.LookPath("getenforce"); err != nil {
		return false
	}
	// 检查配置文件是否存在
	if !utils.FileExists("/etc/selinux/config") {
		return false
	}
	return true
}

// getSelinuxStatus 获取当前SELinux状态（getenforce输出）
func getSelinuxStatus() (string, bool, error) {
	cmd := exec.Command("getenforce")
	output, err := cmd.CombinedOutput()
	return string(output), false, err
}

// getSelinuxConfigValue 读取/etc/selinux/config中的SELINUX值
func getSelinuxConfigValue(configPath string) (string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		// 匹配SELINUX=xxx（忽略注释行）
		if !strings.HasPrefix(line, "#") && strings.HasPrefix(line, "SELINUX=") {
			return strings.TrimPrefix(line, "SELINUX="), nil
		}
	}
	return "", fmt.Errorf("未找到SELINUX配置项")
}

// setSelinuxConfigValue 修改/etc/selinux/config中的SELINUX值为target
func setSelinuxConfigValue(configPath, target string) error {
	// 读取原文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// 替换SELINUX=xxx为SELINUX=target（保留注释行）
	lines := strings.Split(string(content), "\n")
	newLines := make([]string, 0, len(lines))
	updated := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmedLine, "#") && strings.HasPrefix(trimmedLine, "SELINUX=") {
			// 替换配置值
			newLines = append(newLines, "SELINUX="+target)
			updated = true
		} else {
			// 保留其他行
			newLines = append(newLines, line)
		}
	}

	// 如果没有找到配置项，在文件末尾添加
	if !updated {
		newLines = append(newLines, "SELINUX="+target)
	}

	// 写回文件（覆盖原文件）
	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644)
}
