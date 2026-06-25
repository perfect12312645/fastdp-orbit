package modules

import (
	"fastdp-orbit/backend/proto/agent"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// UnarchiveModule 文件解压模块（自动识别格式，带重复解压检查，强制绝对路径）
type UnarchiveModule struct{}

// NewUnarchiveModule 创建解压模块实例
func NewUnarchiveModule() Module {
	return &UnarchiveModule{}
}

// 注册模块
func init() {
	Register("unarchive", NewUnarchiveModule)
}

// 支持的压缩格式
const (
	formatTar    = "tar"             // 支持tar, tar.gz, tar.bz2, tgz等
	formatZip    = "zip"             // zip格式
	defaultPerm  = 0755              // 默认目录权限
	markerSuffix = ".unarchive.done" // 标记文件后缀
)

// 错误码
const (
	UnarchiveErrInvalidParams     = 800 // 参数错误
	UnarchiveErrFileNotExist      = 801 // 源文件不存在
	UnarchiveErrToolNotFound      = 802 // 解压工具不存在
	UnarchiveErrExtractFailed     = 803 // 解压失败
	UnarchiveErrDestNotWritable   = 804 // 目标目录不可写
	UnarchiveErrMarkerFailed      = 805 // 标记文件操作失败
	UnarchiveErrUnsupportedFormat = 806 // 不支持的文件格式
)

// Run 实现解压核心逻辑（自动识别格式）
func (m *UnarchiveModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 参数解析与校验（仅需src和dest）
	srcPath, srcOk := req.Parameters["src"]
	destPath, destOk := req.Parameters["dest"]
	stripStr, stripOk := req.Parameters["strip_components"]
	// 校验必填参数
	if !srcOk || srcPath == "" {
		return utils.ErrorResponse(
			req,
			UnarchiveErrInvalidParams,
			"未指定源文件路径（参数src）",
		), nil
	}
	if !destOk || destPath == "" {
		return utils.ErrorResponse(
			req,
			UnarchiveErrInvalidParams,
			"未指定目标目录（参数dest）",
		), nil
	}

	stripComponents := 0
	if stripOk && stripStr != "" {
		// 转换为整数并校验
		num, err := strconv.Atoi(stripStr)
		if err != nil || num < 0 {
			return utils.ErrorResponse(
				req,
				UnarchiveErrInvalidParams,
				"参数strip_components必须为非负整数（如0、1）",
			), nil
		}
		stripComponents = num
	}
	// 校验绝对路径
	if !filepath.IsAbs(srcPath) {
		return utils.ErrorResponse(
			req,
			UnarchiveErrInvalidParams,
			"源文件路径必须为绝对路径（参数src）",
		), nil
	}
	if !filepath.IsAbs(destPath) {
		return utils.ErrorResponse(
			req,
			UnarchiveErrInvalidParams,
			"目标目录路径必须为绝对路径（参数dest）",
		), nil
	}

	// 2. 自动识别文件格式
	format, err := getFileFormat(srcPath)
	if err != nil {
		return utils.ErrorResponse(
			req,
			UnarchiveErrUnsupportedFormat,
			fmt.Sprintf("无法识别文件格式: %v，支持tar(含.gz/.bz2)和zip", err),
		), nil
	}

	// 3. 生成标记文件路径
	markerFile := getMarkerPath(srcPath, destPath)

	// 4. 检查是否已解压（存在标记文件则直接跳过）
	if utils.FileExists(markerFile) {
		return utils.SuccessResponse(
			req,
			"文件已解压，跳过操作",
			fmt.Sprintf("目标目录已存在标记文件[%s]，确认已解压", markerFile),
		), nil
	}

	// 5. 前置检查（源文件、工具、目标目录）
	if !utils.FileExists(srcPath) {
		return utils.ErrorResponse(
			req,
			UnarchiveErrFileNotExist,
			fmt.Sprintf("源文件不存在: %s", srcPath),
		), nil
	}

	tool, err := getExtractTool(format)
	if err != nil {
		return utils.ErrorResponse(
			req,
			UnarchiveErrToolNotFound,
			fmt.Sprintf("解压工具不存在: %v", err),
		), nil
	}

	if err := ensureDestDir(destPath); err != nil {
		return utils.ErrorResponse(
			req,
			UnarchiveErrDestNotWritable,
			fmt.Sprintf("目标目录不可写或创建失败: %v", err),
		), nil
	}

	// 6. 执行解压操作
	output, isChange, err := extractFile(srcPath, destPath, format, tool, stripComponents)
	if err != nil {
		return utils.ErrorResponse(
			req,
			UnarchiveErrExtractFailed,
			fmt.Sprintf("解压失败: %v，输出：%s", err, output),
		), nil
	}

	// 7. 解压成功后创建标记文件
	if err := createMarkerFile(markerFile); err != nil {
		return utils.ErrorResponse(
			req,
			UnarchiveErrMarkerFailed,
			fmt.Sprintf("创建标记文件失败: %v", err),
		), nil
	}

	// 8. 返回成功响应
	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    output + fmt.Sprintf("\n文件[%s]（自动识别为%s格式）解压至[%s]成功，已创建标记文件", srcPath, format, destPath),
		Stderr:    "",
		Changed:   isChange,
		Error: &agent.ErrorDetail{
			Code:    0,
			Message: "",
			Trace:   "",
		},
	}, nil
}

// ------------------------------
// 核心操作实现
// ------------------------------

// getFileFormat 通过文件名自动识别压缩格式
func getFileFormat(srcPath string) (string, error) {
	// 转换为小写便于匹配
	base := strings.ToLower(filepath.Base(srcPath))

	// 优先检查多级扩展名（如 .tar.gz）
	switch {
	case strings.HasSuffix(base, ".tar.gz") || strings.HasSuffix(base, ".tgz"):
		return formatTar, nil
	case strings.HasSuffix(base, ".tar.bz2") || strings.HasSuffix(base, ".tbz2"):
		return formatTar, nil
	case strings.HasSuffix(base, ".tar.xz") || strings.HasSuffix(base, ".txz"):
		return formatTar, nil
	case strings.HasSuffix(base, ".zip"):
		return formatZip, nil
	case strings.HasSuffix(base, ".tar"):
		return formatTar, nil
	}
	ext := filepath.Ext(base)
	switch ext {
	case ".gz", ".bz2", ".xz":
		// 检查是否是压缩的tar文件（例如：file.tar.gz）
		if prevExt := filepath.Ext(strings.TrimSuffix(base, ext)); prevExt == ".tar" {
			return formatTar, nil
		}
	}
	return "", fmt.Errorf("无法识别文件格式: 文件扩展名 %q 不匹配支持的格式，支持tar(含.gz/.bz2/.xz)和zip", ext)
}

func extractFile(src, dest, format, tool string, stripComponents int) (string, bool, error) {
	var cmd *exec.Cmd

	switch format {
	case formatTar:
		args := []string{"-xf", src, "-C", dest}
		if stripComponents > 0 {
			args = append(args, fmt.Sprintf("--strip-components=%d", stripComponents))
		}
		cmd = exec.Command(tool, args...)
	case formatZip:
		args := []string{"-q", src, "-d", dest}
		cmd = exec.Command(tool, args...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), false, fmt.Errorf("命令执行失败: %w", err)
	}

	isChange, err := checkDestHasContent(dest)
	if err != nil {
		return string(output), false, fmt.Errorf("检查解压结果失败: %w", err)
	}

	return string(output), isChange, nil
}

// ------------------------------
// 标记文件相关函数
// ------------------------------

func getMarkerPath(srcPath, destPath string) string {
	srcName := filepath.Base(srcPath)
	markerName := "." + srcName + markerSuffix
	return filepath.Join(destPath, markerName)
}

func createMarkerFile(markerPath string) error {
	f, err := os.OpenFile(markerPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return f.Close()
}

// ------------------------------
// 辅助函数
// ------------------------------

func getExtractTool(format string) (string, error) {
	switch format {
	case formatTar:
		if _, err := exec.LookPath("tar"); err != nil {
			return "", fmt.Errorf("tar工具未安装")
		}
		return "tar", nil
	case formatZip:
		if _, err := exec.LookPath("unzip"); err != nil {
			return "", fmt.Errorf("unzip工具未安装")
		}
		return "unzip", nil
	}
	return "", fmt.Errorf("不支持的格式: %s", format)
}

func ensureDestDir(dest string) error {
	if err := os.MkdirAll(dest, os.FileMode(defaultPerm)); err != nil {
		return err
	}

	testFile := filepath.Join(dest, ".unarchive_test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("目录不可写: %w", err)
	}
	f.Close()
	return os.Remove(testFile)
}

func checkDestHasContent(dest string) (bool, error) {
	dir, err := os.Open(dest)
	if err != nil {
		return false, err
	}
	defer dir.Close()

	entries, err := dir.Readdir(1)
	if err != nil && err != os.ErrNotExist {
		return false, err
	}
	return len(entries) > 0, nil
}
