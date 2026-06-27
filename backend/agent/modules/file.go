package modules

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// FileModule 文件管理模块，类似 Ansible file 模块
type FileModule struct{}

// NewFileModule 创建文件模块实例
func NewFileModule() Module {
	return &FileModule{}
}

// 支持的操作类型
const (
	actionFileCreate  = "create"  // 创建文件 / 目录
	actionFileDelete  = "delete"  // 删除文件 / 目录
	actionFileTouch   = "touch"   // 创建空文件（类似 touch 命令）
	actionFileSymlink = "symlink" // 创建符号链接
)

// 文件类型
const (
	fileTypeFile    = "file"      // 普通文件
	fileTypeDir     = "directory" // 目录
	fileTypeSymlink = "symlink"   // 符号链接
)

// 错误码
const (
	FileErrInvalidParams = 400 // 参数错误
	FileErrOperationFail = 401 // 操作失败
)

// 文件操作参数
type fileParams struct {
	Path    string // 目标路径（必填）
	Action  string // 操作类型（必填）
	Type    string // 文件类型（create/symlink 时必填）
	Mode    string // 权限模式（如 "0644"，可选）
	Owner   string // 所有者（用户名或 UID，可选）
	Group   string // 所属组（组名或 GID，可选）
	Src     string // 符号链接源路径（symlink 时必填）
	Recurse bool   // 创建目录时是否递归（仅 dir 类型）
	Force   bool   // 删除时是否强制（如非空目录）
	Backup  bool   // 是否备份文件（删除 / 覆盖前）
}

// Run 实现 Module 接口的 Run 方法
func (m *FileModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 关键日志：记录操作开始
	logger.Info("开始文件管理操作",
		zap.String("machine", req.MachineId),
		zap.String("TaskId", req.TaskId),
	)

	// 1. 解析参数
	params, err := parseFileParams(req.Parameters)
	if err != nil {
		logger.Error("参数解析失败", zap.Error(err))
		return utils.ErrorResponse(req, FileErrInvalidParams, err.Error()), nil
	}

	// 关键日志：记录解析后的参数
	logger.Debug("文件操作参数解析完成",
		zap.String("path", params.Path),
		zap.String("action", params.Action),
		zap.String("type", params.Type))

	// 2. 执行文件操作
	result, err := executeFileAction(params)
	if err != nil {
		logger.Error("文件操作执行失败",
			zap.String("path", params.Path),
			zap.String("action", params.Action),
			zap.Error(err))
		return utils.ErrorResponse(req, FileErrOperationFail, err.Error()), nil
	}

	// 关键日志：记录操作结果
	logger.Info("文件管理操作完成",
		zap.String("path", params.Path),
		zap.String("action", params.Action),
		zap.Bool("changed", result.Changed))

	// 3. 返回结果
	if result.Changed {
		return utils.SuccessResponse(req, result.Message), nil
	}
	return utils.SuccessResponseWithNoChange(req, result.Message), nil
}

// 解析文件操作参数
func parseFileParams(params map[string]string) (fileParams, error) {
	path, ok := params["path"]
	if !ok || path == "" {
		return fileParams{}, errors.New("未指定目标路径（path）")
	}

	action, ok := params["action"]
	if !ok || action == "" {
		return fileParams{}, errors.New("未指定操作类型（action：create/delete/touch/symlink）")
	}

	// 校验操作类型合法性
	supportedActions := map[string]bool{
		actionFileCreate:  true,
		actionFileDelete:  true,
		actionFileTouch:   true,
		actionFileSymlink: true,
	}
	if !supportedActions[action] {
		return fileParams{}, fmt.Errorf("不支持的操作类型：%s，支持：create/delete/touch/symlink", action)
	}

	// 解析文件类型（create/symlink 需要）
	fileType := params["type"]
	if (action == actionFileCreate || action == actionFileSymlink) && fileType == "" {
		return fileParams{}, errors.New("创建操作需指定文件类型（type：file/directory/symlink）")
	}
	if fileType != "" {
		supportedTypes := map[string]bool{
			fileTypeFile:    true,
			fileTypeDir:     true,
			fileTypeSymlink: true,
		}
		if !supportedTypes[fileType] {
			return fileParams{}, fmt.Errorf("不支持的文件类型：% s，支持：file/directory/symlink", fileType)
		}
	}

	// 解析递归创建参数（默认 false）
	recurse := false
	if params["recurse"] == "true" {
		recurse = true
	}

	// 解析强制删除参数（默认 false）
	force := false
	if params["force"] == "true" {
		force = true
	}

	// 解析备份参数（默认 false）
	backup := false
	if params["backup"] == "true" {
		backup = true
	}

	// 符号链接必须指定源路径
	if action == actionFileSymlink && params["src"] == "" {
		return fileParams{}, errors.New("创建符号链接需指定源路径（src）")
	}

	return fileParams{
		Path:    path,
		Action:  action,
		Type:    fileType,
		Mode:    params["mode"],  // 如 "0644"
		Owner:   params["owner"], // 用户名或 UID
		Group:   params["group"], // 组名或 GID
		Src:     params["src"],   // 符号链接源
		Recurse: recurse,
		Force:   force,
		Backup:  backup,
	}, nil
}

// 文件操作结果
type fileActionResult struct {
	Changed bool   // 是否有状态变更
	Message string // 简要信息
	Detail  string // 详细信息
}

// 执行具体的文件操作
func executeFileAction(params fileParams) (fileActionResult, error) {
	switch params.Action {
	case actionFileCreate:
		return createFileOrDir(params)
	case actionFileDelete:
		return deleteFileOrDir(params)
	case actionFileTouch:
		return touchFile(params)
	case actionFileSymlink:
		return createSymlink(params)
	default:
		return fileActionResult{}, fmt.Errorf("不支持的操作：% s", params.Action)
	}
}

// 创建文件或目录
func createFileOrDir(params fileParams) (fileActionResult, error) {
	pathExists, err := checkPathExists(params.Path)
	if err != nil {
		return fileActionResult{}, fmt.Errorf("检查路径存在性失败：% w", err)
	}

	// 路径已存在时检查是否需要调整属性
	if pathExists && (params.Mode != "" || params.Owner != "" || params.Group != "") {
		return adjustFileAttributes(params)
	}

	// 路径不存在，执行创建操作
	var createDetail string
	switch params.Type {
	case fileTypeFile:
		// 创建普通文件
		if _, err = os.Create(params.Path); err != nil {
			return fileActionResult{}, fmt.Errorf("创建文件失败：% w", err)
		}
		createDetail = "创建普通文件"

	case fileTypeDir:
		// 创建目录（支持递归）
		if err := os.MkdirAll(params.Path, 0755); err != nil { // 临时权限，后续会调整
			return fileActionResult{}, fmt.Errorf("创建目录失败：% w", err)
		}
		createDetail = fmt.Sprintf("创建目录（递归：% v）", params.Recurse)

	default:
		return fileActionResult{}, fmt.Errorf("不支持的创建类型：% s", params.Type)
	}

	// 应用权限、所有者等属性
	attrDetail, err := applyFileAttributes(params)
	if err != nil {
		return fileActionResult{}, fmt.Errorf("设置文件属性失败：% w", err)
	}

	return fileActionResult{
		Changed: true,
		Message: fmt.Sprintf("成功创建 % s：% s", params.Type, params.Path),
		Detail:  fmt.Sprintf("% s；% s", createDetail, attrDetail),
	}, nil
}

// 调整已存在文件的属性（权限、所有者等）
func adjustFileAttributes(params fileParams) (fileActionResult, error) {
	// 检查当前属性是否符合预期
	currentMode, currentOwner, currentGroup, err := getFileAttributes(params.Path)
	if err != nil {
		return fileActionResult{}, fmt.Errorf("获取文件属性失败：% w", err)
	}

	// 检查是否需要调整
	needsAdjust := false
	if params.Mode != "" && currentMode != params.Mode {
		needsAdjust = true
	}
	if params.Owner != "" && currentOwner != params.Owner {
		needsAdjust = true
	}
	if params.Group != "" && currentGroup != params.Group {
		needsAdjust = true
	}

	if !needsAdjust {
		return fileActionResult{
			Changed: false,
			Message: fmt.Sprintf("% s 已存在且属性符合预期，无需操作", params.Type),
			Detail: fmt.Sprintf("路径：% s，当前权限：% s，所有者：% s，所属组：% s",
				params.Path, currentMode, currentOwner, currentGroup),
		}, nil
	}

	// 调整属性
	attrDetail, err := applyFileAttributes(params)
	if err != nil {
		return fileActionResult{}, fmt.Errorf("调整文件属性失败：% w", err)
	}

	return fileActionResult{
		Changed: true,
		Message: fmt.Sprintf("已调整 % s 属性：% s", params.Type, params.Path),
		Detail:  attrDetail,
	}, nil
}

// 删除文件或目录
func deleteFileOrDir(params fileParams) (fileActionResult, error) {
	pathExists, err := checkPathExists(params.Path)
	if err != nil {
		return fileActionResult{}, fmt.Errorf("检查路径存在性失败：% w", err)
	}

	if !pathExists {
		return fileActionResult{
			Changed: false,
			Message: "路径不存在，无需删除",
			Detail:  fmt.Sprintf("路径：% s", params.Path),
		}, nil
	}

	// 备份文件（如果需要）
	backupDetail := ""
	if params.Backup {
		backupPath, err := backupFile(params.Path)
		if err != nil {
			return fileActionResult{}, fmt.Errorf("备份文件失败：% w", err)
		}
		backupDetail = fmt.Sprintf("已备份至：% s；", backupPath)
	}

	// 执行删除
	if params.Force {
		// 强制删除（支持非空目录）
		if err := os.RemoveAll(params.Path); err != nil {
			return fileActionResult{}, fmt.Errorf("强制删除失败：% w", err)
		}
	} else {
		// 普通删除（目录必须为空）
		if err := os.Remove(params.Path); err != nil {
			return fileActionResult{}, fmt.Errorf("删除失败（可能目录非空，需开启 force）：% w", err)
		}
	}

	return fileActionResult{
		Changed: true,
		Message: fmt.Sprintf("已删除路径：% s", params.Path),
		Detail:  fmt.Sprintf("% s 删除操作完成", backupDetail),
	}, nil
}

// 创建空文件（类似 touch 命令）
func touchFile(params fileParams) (fileActionResult, error) {
	pathExists, err := checkPathExists(params.Path)
	if err != nil {
		return fileActionResult{}, fmt.Errorf("检查路径存在性失败：% w", err)
	}

	var message, detail string
	changed := false

	if pathExists {
		// 更新文件时间戳
		now := time.Now()
		if err := os.Chtimes(params.Path, now, now); err != nil {
			return fileActionResult{}, fmt.Errorf("更新文件时间戳失败：% w", err)
		}
		message = fmt.Sprintf("文件已存在，更新时间戳：% s", params.Path)
		detail = "更新访问和修改时间为当前时间"
		changed = true
	} else {
		// 创建新文件
		if _, err = os.Create(params.Path); err != nil {
			return fileActionResult{}, fmt.Errorf("创建文件失败：% w", err)
		}
		message = fmt.Sprintf("创建空文件：% s", params.Path)
		detail = "创建新文件并设置默认权限"
		changed = true
	}

	// 应用权限（如果指定）
	if params.Mode != "" {
		mode, err := parseFileMode(params.Mode)
		if err != nil {
			return fileActionResult{}, fmt.Errorf("解析权限模式失败：% w", err)
		}
		if err := os.Chmod(params.Path, mode); err != nil {
			return fileActionResult{}, fmt.Errorf("设置文件权限失败：% w", err)
		}
		detail += fmt.Sprintf("；设置权限为：% s", params.Mode)
	}

	return fileActionResult{
		Changed: changed,
		Message: message,
		Detail:  detail,
	}, nil
}

// 创建符号链接
func createSymlink(params fileParams) (fileActionResult, error) {
	// 检查源路径是否存在
	srcExists, err := checkPathExists(params.Src)
	if err != nil {
		return fileActionResult{}, fmt.Errorf("检查源路径存在性失败：% w", err)
	}
	if !srcExists {
		return fileActionResult{}, fmt.Errorf("符号链接源路径不存在：% s", params.Src)
	}

	// 检查目标路径是否已存在
	dstExists, err := checkPathExists(params.Path)
	if err != nil {
		return fileActionResult{}, fmt.Errorf("检查目标路径存在性失败：% w", err)
	}

	if dstExists {
		// 检查是否已是相同的符号链接
		currentLink, err := os.Readlink(params.Path)
		if err != nil {
			return fileActionResult{}, fmt.Errorf("读取现有链接失败，可能已存在目录或文件：% w", err)
		}
		if currentLink == params.Src {
			return fileActionResult{
				Changed: false,
				Message: fmt.Sprintf("符号链接已存在且指向正确目标：% s -> % s", params.Path, params.Src),
				Detail:  "",
			}, nil
		}

		// 存在不同的链接，删除后重新创建（强制模式）
		if err := os.Remove(params.Path); err != nil {
			return fileActionResult{}, fmt.Errorf("删除现有链接失败：% w", err)
		}
	}

	// 创建符号链接
	if err := os.Symlink(params.Src, params.Path); err != nil {
		return fileActionResult{}, fmt.Errorf("创建符号链接失败：% w", err)
	}

	return fileActionResult{
		Changed: true,
		Message: fmt.Sprintf("创建符号链接：% s -> % s", params.Path, params.Src),
		Detail:  "符号链接创建成功",
	}, nil
}

//------------------------------
// 辅助函数
//------------------------------

// 检查路径是否存在
func checkPathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err // 其他错误（如权限不足）
}

// 获取文件属性（权限、所有者、所属组）
func getFileAttributes(path string) (mode string, owner string, group string, err error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", "", "", err
	}

	// 权限模式（转换为八进制字符串）
	mode = fmt.Sprintf("%04o", fileInfo.Mode().Perm())

	// 所有者和组（仅支持 Unix 系统）
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		owner = strconv.FormatUint(uint64(stat.Uid), 10)
		group = strconv.FormatUint(uint64(stat.Gid), 10)
		// 可选：转换为用户名 / 组名（需要额外调用 user.LookupId）
	} else {
		owner = "unknown"
		group = "unknown"
	}

	return mode, owner, group, nil
}

// 解析文件权限模式（如 "0644" -> os.FileMode (0644)）
func parseFileMode(modeStr string) (os.FileMode, error) {
	if modeStr == "" {
		return 0, nil
	}
	// 移除前缀 "0"（如果有），然后解析为八进制
	mode, err := strconv.ParseUint(strings.TrimPrefix(modeStr, "0"), 8, 32)
	if err != nil {
		return 0, fmt.Errorf("无效的权限模式：% s，需为八进制（如 0644）", modeStr)
	}
	return os.FileMode(mode), nil
}

// 应用文件属性（权限、所有者、所属组）
func applyFileAttributes(params fileParams) (string, error) {
	var details []string

	// 设置权限
	if params.Mode != "" {
		mode, err := parseFileMode(params.Mode)
		if err != nil {
			return "", err
		}
		if err := os.Chmod(params.Path, mode); err != nil {
			return "", fmt.Errorf(" 设置权限失败：% w", err)
		}
		details = append(details, fmt.Sprintf("权限设置为：% s", params.Mode))
	}

	// 设置所有者和组（简化实现，实际可能需要解析用户名到 UID）
	// 注意：生产环境需通过 user.Lookup 和 group.Lookup 转换名称为 ID
	if params.Owner != "" || params.Group != "" {
		// 此处仅为示例，实际需根据系统调用实现（如 syscall.Chown）
		ownerDetail, err := setFileOwnerAndGroup(params.Path, params.Owner, params.Group)
		if err != nil {
			return "", fmt.Errorf("设置所有者/组失败：%w", err)
		}
		details = append(details, ownerDetail)
	}

	return strings.Join(details, "；"), nil
}

// 设置文件所有者和组
func setFileOwnerAndGroup(path, uidStr, gidStr string) (string, error) {
	uid := -1
	var err error
	// 转换 UID 字符串为整数
	if uidStr != "" {
		uid, err = strconv.Atoi(uidStr)
		if err != nil {
			return "", fmt.Errorf("无效的 UID: %s，必须是整数", uidStr)
		}
	}

	// 转换 GID 字符串为整数（允许空字符串，此时不修改组）
	gid := -1 // -1 表示不修改 GID
	if gidStr != "" {
		gid, err = strconv.Atoi(gidStr)
		if err != nil {
			return "", fmt.Errorf("无效的 GID: %s，必须是整数", gidStr)
		}
	}

	// 调用系统调用设置所有者和组
	// syscall.Chown 中，-1 表示保持原有值不变
	if err := syscall.Chown(path, uid, gid); err != nil {
		return "", fmt.Errorf("系统调用失败: %w（路径: %s, UID: %d, GID: %d）",
			err, path, uid, gid)
	}
	var detailParts []string
	if uid != -1 {
		detailParts = append(detailParts, fmt.Sprintf("所有者 UID: %d", uid))
	}
	if gid != -1 {
		detailParts = append(detailParts, fmt.Sprintf("所属组 GID: %d", gid))
	}
	if len(detailParts) == 0 {
		return "未修改所有者和组（uid和gid均为-1）", nil
	}
	return "已设置：" + strings.Join(detailParts, "，"), nil
}

// 模块注册
func init() {
	Register("file", NewFileModule)
}
