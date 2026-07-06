package modules

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// LineinfileModule 实现单行文本管理功能（匹配、替换、插入、删除）
type LineinfileModule struct{}

// NewLineinfileModule 创建模块实例
func NewLineinfileModule() Module {
	return &LineinfileModule{}
}

// 操作动作常量（替换原State）
const (
	lineActionInsert  = "insert"  // 插入：在匹配行前/后插入新行（仅当匹配行存在时）
	lineActionReplace = "replace" // 替换：替换匹配行（仅当匹配行存在时）
	lineActionDelete  = "delete"  // 删除：删除匹配行（仅当匹配行存在时）
)

// 参数结构定义（按需求调整）
type lineParams struct {
	FilePath     string // 目标文件路径（必填）
	Regexp       string // 匹配行的正则表达式（必填）
	Line         string // 目标行内容（insert/replace时必填）
	Action       string // 操作动作：insert/replace/delete（必填）
	Backrefs     bool   // 是否启用正则反向引用（仅replace时生效）
	InsertBefore bool   // 插入位置：true=匹配行前，false=匹配行后（默认false）
}

// Run 实现模块核心逻辑
func (m *LineinfileModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 解析并验证参数
	params, err := parseLineParams(req.Parameters)
	if err != nil {
		return utils.ErrorResponse(req, 3001, fmt.Sprintf("参数解析失败: %v", err)), nil
	}

	// 2. 执行操作
	change, _, err := m.executeAction(params)
	if err != nil {
		logger.Error("单行操作失败",
			zap.String("file", params.FilePath),
			zap.String("action", params.Action),
			zap.Error(err))
		return utils.ErrorResponse(req, 3002, err.Error()), nil
	}

	// 3. 返回结果
	if change {
		return utils.SuccessResponse(req, fmt.Sprintf("文件%s操作成功", params.FilePath)), nil
	}
	return utils.SuccessResponseWithNoChange(req, fmt.Sprintf("文件%s无变更", params.FilePath)), nil
}

// 解析并验证参数
func parseLineParams(params map[string]string) (lineParams, error) {
	// 必选参数校验
	filePath, ok := params["path"]
	if !ok || filePath == "" || !path.IsAbs(filePath) {
		return lineParams{}, fmt.Errorf("未指定目标文件路径或不为绝对路径: %s", filePath)
	}

	regexpStr, ok := params["regexp"]
	if !ok || regexpStr == "" {
		return lineParams{}, errors.New("未指定匹配正则表达式（regexp）")
	}
	// 验证正则语法
	if _, err := regexp.Compile(regexpStr); err != nil {
		return lineParams{}, fmt.Errorf("正则表达式无效: %w", err)
	}

	// 动作参数校验（必填）
	action := params["action"]
	if action == "" {
		return lineParams{}, errors.New("未指定操作动作（action: insert/replace/delete）")
	}
	if action != lineActionInsert && action != lineActionReplace && action != lineActionDelete {
		return lineParams{}, errors.New("action参数必须为'insert'/'replace'/'delete'")
	}

	// insert/replace需要line参数
	lineContent := params["line"]
	if (action == lineActionInsert || action == lineActionReplace) && lineContent == "" {
		return lineParams{}, errors.New("action为insert/replace时必须指定line参数")
	}

	// 反向引用仅在replace时生效
	backrefs := false
	if params["backrefs"] == "true" {
		if action != lineActionReplace {
			return lineParams{}, errors.New("backrefs仅在action=replace时有效")
		}
		backrefs = true
	}

	// 插入位置参数（默认插入到匹配行后）
	insertBefore := false
	if params["insertbefore"] == "true" {
		insertBefore = true
	}
	logger.Info("lineinfile模块传入参数：",
		zap.String("path", filePath),
		zap.String("regexp", regexpStr),
		zap.String("line", lineContent),
		zap.String("action", action),
		zap.Bool("backrefs", backrefs),
		zap.Bool("insertbefore", insertBefore),
	)
	return lineParams{
		FilePath:     filePath,
		Regexp:       regexpStr,
		Line:         lineContent,
		Action:       action,
		Backrefs:     backrefs,
		InsertBefore: insertBefore,
	}, nil
}

// 执行具体操作
func (m *LineinfileModule) executeAction(params lineParams) (changed bool, detail string, err error) {
	// 检查文件是否存在（文件不存在则所有操作都不执行）
	fileExists := utils.FileExists(params.FilePath)

	if !fileExists {
		return false, "文件不存在，不执行任何操作", nil
	}

	// 读取文件内容
	lines, err := readFileLines(params.FilePath)
	if err != nil {
		return false, "", fmt.Errorf("读取文件失败: %w", err)
	}

	// 编译正则表达式
	re, err := regexp.Compile(params.Regexp)
	if err != nil {
		return false, "", fmt.Errorf("正则编译失败: %w", err)
	}

	// 查找匹配行（只处理第一个匹配行，避免多匹配场景的歧义）
	matchedLines := findMatchedLines(lines, re)
	lineExists := len(matchedLines) > 0
	if !lineExists {
		return false, "未找到匹配行，不执行任何操作", nil
	}
	firstMatchedIdx := matchedLines[0] // 只处理第一个匹配行

	// 根据动作执行操作
	switch params.Action {
	case lineActionReplace:
		return m.handleReplace(lines, params, re, firstMatchedIdx)
	case lineActionInsert:
		return m.handleInsert(lines, params, firstMatchedIdx)
	case lineActionDelete:
		return m.handleDelete(lines, params, firstMatchedIdx)
	default:
		return false, "", fmt.Errorf("不支持的操作动作: %s", params.Action)
	}
}

// 处理replace动作：替换匹配行（仅当匹配行存在时）
func (m *LineinfileModule) handleReplace(
	lines []string,
	params lineParams,
	re *regexp.Regexp,
	matchedIdx int,
) (bool, string, error) {
	// 计算目标行内容（支持反向引用）
	targetLine := params.Line
	if params.Backrefs {
		originalLine := lines[matchedIdx]
		targetLine = re.ReplaceAllString(originalLine, params.Line)
	}

	// 检查内容是否一致（一致则不修改）
	if lines[matchedIdx] == targetLine {
		return false, "匹配行内容与目标行一致，无需替换", nil
	}

	// 执行替换
	lines[matchedIdx] = targetLine
	return m.writeChanges(lines, params)
}

// 处理insert动作：在匹配行前/后插入新行（仅当匹配行存在时）
func (m *LineinfileModule) handleInsert(
	lines []string,
	params lineParams,
	matchedIdx int,
) (bool, string, error) {
	// 计算插入位置（前/后）
	insertIdx := matchedIdx + 1 // 默认插入到匹配行后
	positionDesc := "匹配行后"
	if params.InsertBefore {
		insertIdx = matchedIdx // 插入到匹配行前
		positionDesc = "匹配行前"
	}

	// 检查插入行是否已存在（避免重复插入）
	if insertIdx < len(lines) && lines[insertIdx] == params.Line {
		return false, fmt.Sprintf("目标行已存在于%s，无需插入", positionDesc), nil
	}

	// 执行插入
	lines = insertLine(lines, insertIdx, params.Line)
	return m.writeChanges(lines, params)
}

// 处理delete动作：删除匹配行（仅当匹配行存在时）
func (m *LineinfileModule) handleDelete(
	lines []string,
	params lineParams,
	matchedIdx int,
) (bool, string, error) {
	// 执行删除
	lines = removeLine(lines, matchedIdx)
	return m.writeChanges(lines, params)
}

// ------------------------------
// 工具函数（精简版，移除备份相关逻辑）
// ------------------------------

// 查找匹配行的索引（返回所有匹配行，取第一个处理）
func findMatchedLines(lines []string, re *regexp.Regexp) []int {
	var matched []int
	for i, line := range lines {
		if re.MatchString(line) {
			matched = append(matched, i)
		}
	}
	return matched
}

// 插入行到指定位置
func insertLine(lines []string, idx int, line string) []string {
	if idx >= len(lines) {
		return append(lines, line)
	}
	// 分割并插入
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:idx]...)
	newLines = append(newLines, line)
	newLines = append(newLines, lines[idx:]...)
	return newLines
}

// 从指定位置移除行
func removeLine(lines []string, idx int) []string {
	if idx < 0 || idx >= len(lines) {
		return lines
	}
	return append(lines[:idx], lines[idx+1:]...)
}

// 写入文件变更（原子写入，无备份）
func (m *LineinfileModule) writeChanges(newLines []string, params lineParams) (bool, string, error) {
	// 原子写入新内容（通过临时文件避免部分写入损坏）
	if err := lineWriteFileWithTemp(params.FilePath, newLines); err != nil {
		return false, "", fmt.Errorf("写入失败: %w", err)
	}

	return true, "内容已更新", nil
}

// 原子写入文件（通过临时文件）
func lineWriteFileWithTemp(filePath string, lines []string) error {
	// 创建临时文件
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), ".lineinfile.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // 清理临时文件

	// 写入内容
	writer := bufio.NewWriter(tempFile)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			tempFile.Close()
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		tempFile.Close()
		return err
	}
	tempFile.Close()

	// 原子替换原文件
	return os.Rename(tempPath, filePath)
}

// 模块注册
func init() {
	Register("lineinfile", NewLineinfileModule)
}
