package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// TemplateModule 处理模板内容写入（复用BlockinfileModule的工具函数）
type TemplateModule struct{}

// NewTemplateModule 创建模板模块实例
func NewTemplateModule() Module {
	return &TemplateModule{}
}

// 注册模块
func init() {
	Register("template", NewTemplateModule)
}

// 错误码
const (
	TemplateErrInvalidParams = 900 // 参数错误
	TemplateErrNotAbsolute   = 901 // 非绝对路径
	TemplateErrIsDir         = 902 // 目标是目录
	TemplateErrIOFailed      = 903 // IO操作失败
)

// Run 实现模板内容写入逻辑（支持覆盖/追加模式）
func (m *TemplateModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 解析参数
	content, contentOk := req.Parameters["content"]
	targetPath, pathOk := req.Parameters["dest"]
	appendMode := req.Parameters["append"] == "true" // 追加模式开关（默认false=覆盖）

	// 校验必填参数
	if !contentOk || content == "" {
		return utils.ErrorResponse(req, TemplateErrInvalidParams, "未指定内容（参数content）"), nil
	}
	if !pathOk || targetPath == "" {
		return utils.ErrorResponse(req, TemplateErrInvalidParams, "未指定目标路径（参数dest）"), nil
	}

	// 2. 校验目标路径是否为绝对路径
	if !filepath.IsAbs(targetPath) {
		return utils.ErrorResponse(req, TemplateErrNotAbsolute, "目标路径必须为绝对路径"), nil
	}

	// 3. 检查目标路径是否已存在且为目录
	if exists, isDir, err := checkPathStatus(targetPath); err != nil {
		logger.Error("检查目标路径失败", zap.String("dest", targetPath), zap.Error(err))
		return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("检查路径失败: %v", err)), nil
	} else if exists && isDir {
		return utils.ErrorResponse(req, TemplateErrIsDir, "目标路径已存在且为目录"), nil
	}

	// 4. 根据模式执行操作（覆盖/追加）
	if appendMode {
		return m.handleAppendMode(req, targetPath, content)
	}
	return m.handleOverwriteMode(req, targetPath, content)
}

// ------------------------------
// 覆盖模式（默认）：原子替换文件，MD5比对
// ------------------------------
func (m *TemplateModule) handleOverwriteMode(req *agent.ExecRequest, targetPath, content string) (*agent.ExecResponse, error) {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("创建目录失败: %v", err)), nil
	}
	// 创建临时文件
	tempFile, err := createTempFile(targetPath)
	if err != nil {
		logger.Error("创建临时文件失败", zap.String("dest", targetPath), zap.Error(err))
		return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("创建临时文件失败: %v", err)), nil
	}
	defer os.Remove(tempFile)

	// 写入内容到临时文件
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		logger.Error("写入临时文件失败", zap.String("temp", tempFile), zap.Error(err))
		return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("写入内容失败: %v", err)), nil
	}

	// 计算临时文件MD5
	tempMD5, err := utils.FileMD5(tempFile)
	if err != nil {
		logger.Error("计算临时文件MD5失败", zap.String("temp", tempFile), zap.Error(err))
		return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("计算MD5失败: %v", err)), nil
	}

	// 若目标文件已存在，比对MD5
	if exists, _, _ := checkPathStatus(targetPath); exists {
		targetMD5, err := utils.FileMD5(targetPath)
		if err != nil {
			logger.Error("计算目标文件MD5失败", zap.String("dest", targetPath), zap.Error(err))
			return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("读取目标文件失败: %v", err)), nil
		}

		if tempMD5 == targetMD5 {
			logger.Info("文件内容未变更，跳过写入", zap.String("dest", targetPath))
			return utils.SuccessResponseWithNoChange(
				req,
				"文件内容未变更",
			), nil
		}
	}

	// 原子性替换目标文件
	if err := os.Rename(tempFile, targetPath); err != nil {
		logger.Error("替换目标文件失败", zap.String("temp", tempFile), zap.String("target", targetPath), zap.Error(err))
		return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("写入文件失败: %v", err)), nil
	}

	logger.Info("模板内容覆盖写入成功", zap.String("dest", targetPath))
	return utils.SuccessResponse(
		req,
		"文件覆盖写入成功",
	), nil
}

// ------------------------------
// 追加模式：复用BlockinfileModule的标记和工具函数
// ------------------------------
func (m *TemplateModule) handleAppendMode(req *agent.ExecRequest, targetPath, content string) (*agent.ExecResponse, error) {
	// 1. 读取文件内容（复用blockinfile的读取函数）
	lines, err := readFileLines(targetPath)
	if err != nil {
		// 文件不存在时创建空文件（复用blockinfile的创建函数）
		if os.IsNotExist(err) {
			if err := createEmptyFile(targetPath); err != nil {
				logger.Error("创建空文件失败", zap.String("dest", targetPath), zap.Error(err))
				return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("创建文件失败: %v", err)), nil
			}
			lines = []string{}
		} else {
			logger.Error("读取文件失败", zap.String("dest", targetPath), zap.Error(err))
			return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("读取文件失败: %v", err)), nil
		}
	}

	// 2. 检查是否已存在blockinfile的标记块（直接使用其常量和查找函数）
	beginIdx, endIdx := findBlockIndices(lines)
	if beginIdx != -1 && endIdx != -1 {
		logger.Info("追加模式：标记块已存在，不做改动", zap.String("dest", targetPath))
		return utils.SuccessResponseWithNoChange(
			req,
			"标记块已存在",
		), nil
	}

	// 3. 构建带标记的内容块（使用blockinfile的固定标记）
	contentLines := strings.Split(content, "\n")
	newBlock := []string{MarkerBegin} // 直接使用BlockinfileModule的开始标记
	newBlock = append(newBlock, contentLines...)
	newBlock = append(newBlock, MarkerEnd) // 直接使用BlockinfileModule的结束标记

	// 4. 追加到文件末尾
	newLines := append(lines, newBlock...)

	// 5. 原子性写入（复用blockinfile的临时文件写入函数）
	if err := writeFileWithTemp(targetPath, newLines); err != nil {
		logger.Error("追加模式写入失败", zap.String("dest", targetPath), zap.Error(err))
		return utils.ErrorResponse(req, TemplateErrIOFailed, fmt.Sprintf("追加内容失败: %v", err)), nil
	}

	logger.Info("模板内容追加成功", zap.String("dest", targetPath))
	return utils.SuccessResponse(
		req,
		"文件追加成功",
	), nil
}

// ------------------------------
// 本地辅助函数（仅本模块专用）
// ------------------------------

// checkPathStatus 检查路径是否存在及是否为目录
func checkPathStatus(path string) (exists bool, isDir bool, err error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	return true, fileInfo.IsDir(), nil
}

// createTempFile 在目标路径同目录创建临时文件
func createTempFile(targetPath string) (string, error) {
	dir := filepath.Dir(targetPath)
	filename := filepath.Base(targetPath)
	tempFile, err := os.CreateTemp(dir, fmt.Sprintf(".%s.tmp-*", filename))
	if err != nil {
		return "", err
	}
	tempFile.Close()
	return tempFile.Name(), nil
}

// ------------------------------
// 直接复用BlockinfileModule的工具函数和常量
// （注意：这些函数和常量在blockinfile.go中需保持可访问性）
// ------------------------------

/* 以下函数和常量实际定义在BlockinfileModule中，此处仅作引用说明：

// 来自BlockinfileModule的标记常量
const (
	MarkerBegin = "# BEGIN FASTDP-OPS MANAGED BLOCK"
	MarkerEnd   = "# END FASTDP-OPS MANAGED BLOCK"
)

// 来自BlockinfileModule的工具函数
func readFileLines(filePath string) ([]string, error) { ... }
func createEmptyFile(filePath string) error { ... }
func writeFileWithTemp(filePath string, lines []string) error { ... }
func findBlockIndices(lines []string) (beginIdx, endIdx int) { ... }

*/
