package modules

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// ImageModule 镜像管理模块，支持镜像的加载、推送、移除和拉取
type ImageModule struct{}

// NewImageModule 创建镜像管理模块实例
func NewImageModule() Module {
	return &ImageModule{}
}

// 模块注册
func init() {
	Register("image", NewImageModule)
}

// 操作动作常量
const (
	imageActionLoad   = "load"   // 加载镜像
	imageActionPush   = "push"   // 推送镜像
	imageActionRemove = "remove" // 移除镜像
	imageActionPull   = "pull"   // 拉取镜像
)

// 镜像操作结果
type imageActionResult struct {
	Changed bool   // 是否产生变更
	Message string // 简要信息
	Detail  string // 详细输出
}

// Run 实现模块核心逻辑
func (m *ImageModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	logger.Info("开始镜像管理操作",
		zap.String("machine", req.MachineId),
		zap.String("task", req.TaskId))

	// 解析参数
	action, actionOk := req.Parameters["action"]
	path, pathOk := req.Parameters["path"]
	imageTag, tagOk := req.Parameters["tag"]

	// 参数验证
	if !actionOk {
		return utils.ErrorResponse(req, 1001, "未指定操作类型（action，支持：load/push/remove/pull）"), nil
	}
	if !tagOk || imageTag == "" {
		return utils.ErrorResponse(req, 1002, "推送/移除/拉取镜像需指定镜像标签（tag参数）"), nil
	}
	// 根据操作类型验证必要参数
	switch action {
	case imageActionLoad:
		if !pathOk || path == "" {
			return utils.ErrorResponse(req, 1002, "加载镜像需指定文件路径（path参数）"), nil
		}
		// load操作允许不指定tag，从文件名提取
	case imageActionPush, imageActionRemove, imageActionPull:

	default:
		return utils.ErrorResponse(req, 1003, "不支持的操作类型："+action), nil
	}

	// 执行对应操作
	var result imageActionResult
	var err error

	switch action {
	case imageActionLoad:
		result, err = m.loadImage(path, imageTag)
	case imageActionPush:
		result, err = m.pushImage(imageTag)
	case imageActionRemove:
		result, err = m.removeImage(imageTag)
	case imageActionPull:
		result, err = m.pullImage(imageTag)
	}

	if err != nil {
		logger.Error("镜像操作失败",
			zap.String("action", action),
			zap.Error(err))
		return utils.ErrorResponse(req, 1004, err.Error()), nil
	}

	logger.Info("镜像操作完成",
		zap.String("action", action),
		zap.Bool("changed", result.Changed),
		zap.String("message", result.Message))

	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    result.Message,
		Changed:   result.Changed,
	}, nil
}

// ------------------------------
// pullImage 拉取镜像（带幂等性校验）
// ------------------------------
func (m *ImageModule) pullImage(tag string) (imageActionResult, error) {
	// 1. 检查本地是否存在完整的目标镜像（标签匹配+完整性校验）
	isValid, digests, err := m.isImageValid(tag)
	if err != nil {
		return imageActionResult{}, fmt.Errorf("校验本地镜像状态失败: %w", err)
	}

	// 2. 已存在完整镜像，直接返回
	if isValid {
		return imageActionResult{
			Changed: false,
			Message: fmt.Sprintf("镜像%s已存在且完整，无需拉取", tag),
			Detail:  fmt.Sprintf("镜像详情: %s", digests),
		}, nil
	}

	// 3. 执行拉取命令
	output, err := runDockerCommand("pull", tag)
	if err != nil {
		// 特殊处理认证错误
		if strings.Contains(output, "authentication required") || strings.Contains(output, "no basic auth credentials") {
			return imageActionResult{}, fmt.Errorf("镜像仓库认证失败，请先执行login操作")
		}
		return imageActionResult{}, fmt.Errorf("拉取镜像失败: %w, 输出: %s", err, output)
	}

	// 4. 拉取后二次校验完整性
	postCheckValid, _, postErr := m.isImageValid(tag)
	if !postCheckValid || postErr != nil {
		return imageActionResult{}, fmt.Errorf("拉取成功但镜像校验失败，可能镜像不完整: %v", postErr)
	}

	return imageActionResult{
		Changed: true,
		Message: fmt.Sprintf("镜像%s拉取成功", tag),
		Detail:  output,
	}, nil
}

// ------------------------------
// loadImage 加载镜像（带幂等性校验）
// ------------------------------
func (m *ImageModule) loadImage(path string, tag string) (imageActionResult, error) {
	// 1. 检查文件是否存在
	if !utils.FileExists(path) {
		return imageActionResult{}, fmt.Errorf("镜像文件不存在: %s", path)
	}

	// 2. 确定目标镜像标签（优先使用指定tag，否则从路径提取）
	imageTag := tag

	// 3. 检查本地是否已存在完整的目标镜像
	isValid, digests, err := m.isImageValid(imageTag)
	if err != nil {
		return imageActionResult{}, fmt.Errorf("校验本地镜像状态失败: %w", err)
	}

	// 4. 已存在完整镜像，直接返回
	if isValid {
		return imageActionResult{
			Changed: false,
			Message: fmt.Sprintf("镜像%s已存在且完整，无需加载", imageTag),
			Detail:  fmt.Sprintf("镜像详情: %s", digests),
		}, nil
	}

	// 5. 执行加载命令
	output, err := runDockerCommand("load", "-i", path)
	if err != nil {
		return imageActionResult{}, fmt.Errorf("加载镜像失败: %w, 输出: %s", err, output)
	}
	loadTag := extractLoadedImageFromOutput(output)
	if loadTag != tag {
		return imageActionResult{}, fmt.Errorf("加载成功但镜像名与预期不符，预期tag:%s,load后tag:%s", tag, loadTag)
	}
	// 7. 加载后二次校验完整性
	postCheckValid, _, postErr := m.isImageValid(imageTag)
	if !postCheckValid || postErr != nil {
		return imageActionResult{}, fmt.Errorf("加载成功但镜像校验失败，可能镜像文件损坏: %v", postErr)
	}

	return imageActionResult{
		Changed: true,
		Message: fmt.Sprintf("镜像%s加载成功", imageTag),
		Detail:  output,
	}, nil
}

// ------------------------------
// 原有功能保持不变
// ------------------------------
func (m *ImageModule) pushImage(tag string) (imageActionResult, error) {
	// 检查镜像是否存在且完整有效（核心修改：用isImageValid替代isImageExists）
	isValid, digests, err := m.isImageValid(tag)
	if err != nil {
		return imageActionResult{}, fmt.Errorf("校验镜像有效性失败: %w", err)
	}
	if !isValid {
		return imageActionResult{}, fmt.Errorf("镜像%s不存在或不完整，无法推送（详情：%s）", tag, digests)
	}

	// 执行推送命令
	output, err := runDockerCommand("push", tag)
	if err != nil {
		return imageActionResult{}, fmt.Errorf("推送镜像失败: %w, 输出: %s", err, output)
	}

	// 判断是否有实际推送
	changed := strings.Contains(output, "Pushed")

	return imageActionResult{
		Changed: changed,
		Message: fmt.Sprintf("镜像%s推送完成", tag),
		Detail:  output,
	}, nil
}
func (m *ImageModule) removeImage(tag string) (imageActionResult, error) {
	// 检查镜像是否存在（基础检查）
	if !m.isImageExists(tag) {
		return imageActionResult{
			Changed: false,
			Message: fmt.Sprintf("镜像%s不存在，无需移除", tag),
			Detail:  "",
		}, nil
	}

	// 执行移除命令
	output, err := runDockerCommand("rmi", tag)
	if err != nil {
		if strings.Contains(output, "is using its referenced image") {
			return imageActionResult{}, fmt.Errorf("镜像%s正在被容器使用，无法移除", tag)
		}
		return imageActionResult{}, fmt.Errorf("移除镜像失败: %w, 输出: %s", err, output)
	}

	return imageActionResult{
		Changed: true,
		Message: fmt.Sprintf("镜像%s移除成功", tag),
		Detail:  output,
	}, nil
}

// ------------------------------
// 新增：镜像完整性校验核心函数
// ------------------------------

// isImageValid 检查镜像是否存在且完整（标签匹配+元数据校验）
func (m *ImageModule) isImageValid(tag string) (bool, string, error) {
	// 1. 执行inspect命令获取镜像元数据（镜像存在且完整的核心校验）
	output, err := runDockerCommand("inspect", tag)
	if err != nil {
		// 区分"镜像不存在"和"其他错误"
		if strings.Contains(output, "No such object") ||
			strings.Contains(output, "Error") ||
			strings.Contains(err.Error(), "exit status 1") {
			return false, "", nil // 镜像不存在，非错误
		}
		// 其他错误（如元数据损坏）视为校验失败
		return false, output, fmt.Errorf("inspect命令执行失败: %w", err)
	}
	var inspectResult []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &inspectResult); err != nil {
		return false, "", fmt.Errorf("解析镜像元数据失败: %w", err)
	}
	if len(inspectResult) == 0 {
		return false, "", errors.New("镜像元数据为空")
	}
	// 提取RepoDigests字段（可能为nil或空切片）
	repoDigests, ok := inspectResult[0]["RepoDigests"].([]interface{})
	if !ok {
		return false, "", errors.New("镜像元数据缺少RepoDigests字段")
	}

	// 转换RepoDigests为字符串（用逗号分隔）
	repoDigestsStr := strings.Join(convertToStringSlice(repoDigests), ", ")

	// 3. 校验核心字段（Id）
	if _, ok := inspectResult[0]["Id"].(string); !ok {
		return false, repoDigestsStr, errors.New("镜像元数据不完整，缺少ID字段")
	}
	// 4. 检查是否为虚悬镜像（dangling image）
	repoTags, ok := inspectResult[0]["RepoTags"].([]interface{})
	if !ok {
		return false, repoDigestsStr, errors.New("镜像元数据缺少RepoTags字段")
	}
	isDangling := len(repoTags) == 0 && len(repoDigests) == 0
	if isDangling {
		return false, repoDigestsStr, errors.New("镜像为虚悬镜像（dangling），视为无效")
	}

	return true, repoDigestsStr, nil
}

// ------------------------------
// 辅助函数
// ------------------------------

// isImageExists 基础检查：判断镜像是否存在（用于push/remove）
func (m *ImageModule) isImageExists(tag string) bool {
	output, err := runDockerCommand("images", "--format", "{{.Repository}}:{{.Tag}}", tag)
	if err != nil {
		logger.Debug("检查镜像存在性时出错",
			zap.String("tag", tag),
			zap.Error(err))
		return false
	}

	cleanOutput := strings.TrimSpace(output)
	return cleanOutput == tag || strings.Contains(cleanOutput, tag)
}

// runDockerCommand 执行docker命令并返回输出
func runDockerCommand(args ...string) (string, error) {
	cmdArgs := append([]string{"docker"}, args...)
	logger.Debug("执行Docker命令", zap.String("command", strings.Join(cmdArgs, " ")))

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	logger.Debug("命令执行结果",
		zap.String("command", strings.Join(cmdArgs, " ")),
		zap.String("output", outputStr),
		zap.Error(err))

	if err != nil {
		return outputStr, fmt.Errorf("命令执行失败: %w", err)
	}
	return outputStr, nil
}

// extractLoadedImageFromOutput 从load命令输出中提取加载的镜像名
func extractLoadedImageFromOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Loaded image: ") {
			return strings.TrimPrefix(line, "Loaded image: ")
		}
	}
	return ""
}
func convertToStringSlice(ifaceSlice []interface{}) []string {
	strSlice := make([]string, 0, len(ifaceSlice))
	for _, item := range ifaceSlice {
		if str, ok := item.(string); ok {
			strSlice = append(strSlice, str)
		}
	}
	return strSlice
}
