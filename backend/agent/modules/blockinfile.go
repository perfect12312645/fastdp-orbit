package modules

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// BlockinfileModule 实现文本块管理功能（声明式操作）
type BlockinfileModule struct{}

// NewBlockinfileModule 创建模块实例
func NewBlockinfileModule() Module {
	return &BlockinfileModule{}
}

// 固定的标记常量（无需用户输入）
const (
	MarkerBegin = "# BEGIN FASTDP-ORBIT MANAGED BLOCK" // 开始标记
	MarkerEnd   = "# END FASTDP-ORBIT MANAGED BLOCK"   // 结束标记
)

// 操作类型常量（仅保留声明式操作）
const (
	actionEnsure = "ensure" // 声明式确保文本块存在且内容正确
	actionDelete = "delete" // 删除文本块
)

// 文本块参数结构
type blockParams struct {
	FilePath string   // 目标文件路径
	Content  []string // 文本块内容（每行一个元素）
	Backup   bool     // 是否备份原文件
}

// Run 实现模块核心逻辑
func (m *BlockinfileModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 解析参数
	action, ok := req.Parameters["action"]
	if !ok || action == "" {
		// 修正错误提示：仅支持 ensure/delete
		return utils.ErrorResponse(req, 2001, "未指定操作类型（action: ensure/delete）"), nil
	}

	params, err := parseBlockParams(req.Parameters)
	if err != nil {
		return utils.ErrorResponse(req, 2002, fmt.Sprintf("参数解析失败: %v", err)), nil
	}

	// 2. 执行操作
	result, err := m.executeAction(params, action)
	if err != nil {
		logger.Error("文本块操作失败",
			zap.String("file", params.FilePath),
			zap.String("action", action),
			zap.Error(err))
		return utils.ErrorResponse(req, 2003, err.Error()), nil
	}

	// 3. 返回结果
	if result {
		return utils.SuccessResponse(req, fmt.Sprintf("文件%s操作成功", params.FilePath)), nil
	}
	return utils.SuccessResponseWithNoChange(req, fmt.Sprintf("文件%s无变更", params.FilePath)), nil
}

// 解析参数
func parseBlockParams(params map[string]string) (blockParams, error) {
	filePath, ok := params["path"]
	if !ok || filePath == "" {
		return blockParams{}, errors.New("未指定目标文件路径（path）")
	}

	// 解析内容（支持换行符分隔）
	contentStr := params["content"]
	// 仅 ensure 操作需要 content，delete 操作可忽略
	if action := params["action"]; action == actionEnsure && contentStr == "" {
		return blockParams{}, errors.New("ensure 操作需指定文本块内容（content）")
	}

	content := strings.Split(contentStr, "\n")

	// 解析备份参数（默认不备份）
	backup := false
	if params["backup"] == "true" {
		backup = true
	}

	return blockParams{
		FilePath: filePath,
		Content:  content,
		Backup:   backup,
	}, nil
}

// 执行具体操作
func (m *BlockinfileModule) executeAction(params blockParams, action string) (bool, error) {
	// 检查文件是否存在
	if _, err := os.Stat(params.FilePath); os.IsNotExist(err) {
		// 对于 ensure 操作，文件不存在则创建；delete 操作文件不存在视为无变更
		if action == actionEnsure {
			if err := createEmptyFile(params.FilePath); err != nil {
				return false, fmt.Errorf("创建文件失败: %w", err)
			}
		} else {
			return false, nil
		}
	}

	// 读取文件内容
	lines, err := readFileLines(params.FilePath)
	if err != nil {
		return false, fmt.Errorf("读取文件失败: %w", err)
	}

	// 查找现有文本块位置
	beginIdx, endIdx := findBlockIndices(lines)
	blockExists := beginIdx != -1 && endIdx != -1

	// 根据操作类型处理
	switch action {
	case actionEnsure:
		return m.handleEnsure(lines, params, blockExists)
	case actionDelete:
		return m.handleDelete(lines, params, blockExists)
	default:
		return false, fmt.Errorf("不支持的操作: %s（支持 ensure/delete）", action)
	}
}

// 声明式确保文本块存在且内容正确
func (m *BlockinfileModule) handleEnsure(lines []string, params blockParams, blockExists bool) (bool, error) {
	// 构建目标文本块（固定标记+内容）
	targetBlock := []string{MarkerBegin}
	targetBlock = append(targetBlock, params.Content...)
	targetBlock = append(targetBlock, MarkerEnd)

	if blockExists {
		// 块存在：提取现有内容并对比
		beginIdx, endIdx := findBlockIndices(lines)
		existingContent := getBlockContent(lines, beginIdx, endIdx)

		// 内容完全一致 → 无变更
		if linesEqual(existingContent, params.Content) {
			return false, nil
		}

		// 内容不同 → 替换现有块
		newLines := append(append(lines[:beginIdx], targetBlock...), lines[endIdx+1:]...)
		return m.writeChanges(newLines, params)
	}

	// 块不存在 → 插入新块
	newLines := append(lines, targetBlock...)
	return m.writeChanges(newLines, params)
}

// 处理删除操作
func (m *BlockinfileModule) handleDelete(lines []string, params blockParams, blockExists bool) (bool, error) {
	if !blockExists {
		return false, nil
	}

	// 移除块内容（包括固定标记）
	beginIdx, endIdx := findBlockIndices(lines)
	newLines := append(lines[:beginIdx], lines[endIdx+1:]...)

	// 备份并写入文件
	return m.writeChanges(newLines, params)
}

// ------------------------------
// 工具函数
// ------------------------------

// 查找文本块的开始和结束索引
func findBlockIndices(lines []string) (beginIdx, endIdx int) {
	beginIdx = -1
	endIdx = -1

	for i, line := range lines {
		if strings.TrimSpace(line) == MarkerBegin {
			beginIdx = i
		}
		if strings.TrimSpace(line) == MarkerEnd && beginIdx != -1 {
			endIdx = i
			break // 找到第一个匹配的结束标记
		}
	}

	// 校验块完整性
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		return -1, -1
	}
	return beginIdx, endIdx
}

// 提取块内实际内容（排除标记行）
func getBlockContent(lines []string, beginIdx, endIdx int) []string {
	if beginIdx >= endIdx-1 {
		return []string{} // 空块（仅标记无内容）
	}
	return lines[beginIdx+1 : endIdx]
}

// 读取文件内容为行列表
func readFileLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// 备份并写入变更
func (m *BlockinfileModule) writeChanges(newLines []string, params blockParams) (bool, error) {
	// 备份原文件（如果启用备份）
	if params.Backup {
		if _, err := backupFile(params.FilePath); err != nil {
			return false, fmt.Errorf("备份文件失败: %w", err)
		}
	}

	// 使用临时文件原子写入新内容
	if err := writeFileWithTemp(params.FilePath, newLines); err != nil {
		return false, fmt.Errorf("写入新内容失败: %w", err)
	}

	return true, nil
}

// 创建空文件（含父目录）
func createEmptyFile(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// 备份文件（添加时间戳后缀）
func backupFile(filePath string) (string, error) {
	backupPath := fmt.Sprintf("%s.bak.%d", filePath, time.Now().Unix())
	if err := copyFile(filePath, backupPath); err != nil {
		return "", err
	}
	return backupPath, nil
}

// 复制文件
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = bufio.NewReader(srcFile).WriteTo(dstFile)
	return err
}

// 使用临时文件写入（避免部分写入导致文件损坏）
func writeFileWithTemp(filePath string, lines []string) error {
	// 创建临时文件
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), ".blockinfile.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // 确保失败时清理临时文件

	// 写入内容
	writer := bufio.NewWriter(tempFile)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			tempFile.Close()
			return err
		}
	}
	writer.Flush()
	tempFile.Close()

	// 原子替换原文件
	return os.Rename(tempPath, filePath)
}

// 检查两行列表是否相等
func linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 模块注册
func init() {
	Register("blockinfile", NewBlockinfileModule)
}
