package modules

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// PackageManagerModule 多系统兼容的包管理模块
type PackageManagerModule struct{}

func NewPackageManagerModule() Module {
	return &PackageManagerModule{}
}

func init() {
	Register("package", NewPackageManagerModule)
}

// 支持的操作类型
const (
	actionPackageInstall      = "install" // 安装包
	actionPackageRemove       = "remove"  // 卸载包
	actionPackageUpdate       = "update"  // 更新包（仅更新指定包）
	actionPackageCheck        = "check"   // 检查包状态
	actionPackageLocalInstall = "localinstall"
)

// 错误码
const (
	PackageErrInvalidParams = 900 // 参数错误
	PackageErrUnsupportedOS = 901 // 不支持的系统
	PackageErrCmdFailed     = 902 // 命令执行失败
)

// 系统类型与包管理器映射
type osPackageManager struct {
	OSName          string   // 系统名称（如ubuntu、centos、rocky、kylin）
	PMType          string   // 包管理器类型（apt、yum、dnf）
	LocalInstallCmd []string // 本地安装命令（rpm/dpkg）
	//FixDepsCmd      []string // 修复依赖命令（如apt -f install）
	InstallCmd []string // 安装命令模板
	RemoveCmd  []string // 卸载命令模板
	UpdateCmd  []string // 更新指定包命令
	CheckCmd   []string // 检查包是否安装命令

}

// 系统包管理器配置（根据实际测试补充版本适配）
var osPMSettings = []osPackageManager{
	// Ubuntu（所有版本，用apt）
	{
		OSName:          "ubuntu",
		PMType:          "apt",
		InstallCmd:      []string{"apt", "install", "-y"},
		LocalInstallCmd: []string{"dpkg", "-i"}, // DEB包本地安装命令
		//FixDepsCmd:      []string{"apt", "install", "-y", "-f"}, // 修复依赖
		RemoveCmd: []string{"apt", "remove", "-y"},
		UpdateCmd: []string{"apt", "install", "-y"},
		CheckCmd:  []string{"dpkg", "-s"}, // 检查DEB包是否安装
	},
	// CentOS 7.x（RPM包）
	{
		OSName:          "centos",
		PMType:          "yum",
		InstallCmd:      []string{"yum", "install", "-y"},
		LocalInstallCmd: []string{"yum", "localinstall", "-y"}, // 复用yum命令
		RemoveCmd:       []string{"yum", "remove", "-y"},
		UpdateCmd:       []string{"yum", "update", "-y"},
		CheckCmd:        []string{"rpm", "-q"},
	},
	// Rocky/Kylin（RPM包）
	{
		OSName:          "rocky",
		PMType:          "dnf",
		InstallCmd:      []string{"dnf", "install", "-y"},
		LocalInstallCmd: []string{"dnf", "localinstall", "-y"}, // dnf统一用install
		RemoveCmd:       []string{"dnf", "remove", "-y"},
		UpdateCmd:       []string{"dnf", "update", "-y"},
		CheckCmd:        []string{"rpm", "-q"},
	},
	{
		OSName:          "kylin",
		PMType:          "dnf",
		InstallCmd:      []string{"dnf", "install", "-y"},
		LocalInstallCmd: []string{"yum", "localinstall", "-y"},
		RemoveCmd:       []string{"dnf", "remove", "-y"},
		UpdateCmd:       []string{"dnf", "update", "-y"},
		CheckCmd:        []string{"rpm", "-q"},
	},
	{
		OSName:          "red hat",
		PMType:          "dnf",
		InstallCmd:      []string{"dnf", "install", "-y"},
		LocalInstallCmd: []string{"dnf", "localinstall", "-y"},
		RemoveCmd:       []string{"dnf", "remove", "-y"},
		UpdateCmd:       []string{"dnf", "update", "-y"},
		CheckCmd:        []string{"rpm", "-q"},
	},
}

// Run 实现模块核心逻辑
func (m *PackageManagerModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 关键日志：记录操作开始
	logger.Info("开始包管理操作",
		zap.String("machine", req.MachineId),
		zap.String("template", req.TaskId))

	// 1. 解析参数
	action, actionOk := req.Parameters["action"]
	nameParam, nameOk := req.Parameters["name"]

	// 2. 参数校验
	if !actionOk {
		return utils.ErrorResponse(req, PackageErrInvalidParams, "未指定操作（action，支持：install/remove/update/check/localinstall）"), nil
	}
	if !nameOk || nameParam == "" {
		return utils.ErrorResponse(req, PackageErrInvalidParams, "未指定包名称或路径（name参数，多包用逗号分隔）"), nil
	}
	// 处理包列表：无论单包还是多包，均拆分为列表（支持逗号分隔）
	packageList := splitPackageList(nameParam)
	logger.Debug("解析请求参数",
		zap.String("action", action),
		zap.Strings("packages", packageList),
		zap.Int("count", len(packageList)))

	// 3. 识别系统类型和对应的包管理器
	osInfo, err := utils.GetLinuxDistribution()
	if err != nil || osInfo == "unknown" {
		logger.Error("不支持的操作系统", zap.String("os", osInfo), zap.Error(err))
		return utils.ErrorResponse(req, PackageErrUnsupportedOS, "不支持的系统"+osInfo), nil
	}

	pm, err := getPackageManager(osInfo)
	if err != nil {
		logger.Warn("未识别系统包管理器",
			zap.String("os", osInfo),
			zap.Error(err))
	} else {
		logger.Info("识别系统包管理器",
			zap.String("os", osInfo),
			zap.String("pm", pm.PMType))
	}
	var result pkgActionResult
	switch action {
	case actionPackageInstall:
		result, err = batchInstallPackages(pm, packageList)
	case actionPackageRemove:
		result, err = batchRemovePackages(pm, packageList)
	case actionPackageUpdate:
		result, err = batchUpdatePackages(pm, packageList)
	case actionPackageLocalInstall:
		result, err = batchLocalInstallPackages(pm, packageList)
	case actionPackageCheck:
		result, err = batchCheckPackages(pm, packageList)
	default:
		return utils.ErrorResponse(req, PackageErrInvalidParams, "不支持的操作类型："+action), nil
	}

	if err != nil {
		logger.Error("包管理操作失败",
			zap.String("action", action),
			zap.Error(err))
		return utils.ErrorResponse(req, PackageErrCmdFailed, err.Error()), nil
	}

	logger.Info("包管理操作完成",
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

// getPackageManager 根据系统信息获取对应的包管理器配置
func getPackageManager(os string) (osPackageManager, error) {
	lowerOS := strings.ToLower(os)

	// 优先匹配系统名称
	for _, pm := range osPMSettings {
		if strings.HasPrefix(lowerOS, pm.OSName) {
			return pm, nil
		}
	}
	return osPMSettings[1], fmt.Errorf("未知系统：%s，将使用yum", os)
}

// ------------------------------
// 包管理操作执行
// ------------------------------

// pkgActionResult 操作结果
type pkgActionResult struct {
	Changed bool   // 是否产生变更
	Message string // 简要信息
	Detail  string // 详细输出
}

func batchInstallPackages(pm osPackageManager, packages []string) (pkgActionResult, error) {
	// 筛选需要安装的包（未安装的）
	toInstall := []string{}
	skipped := []string{}
	for _, pkg := range packages {
		if isPackageInstalled(pm, pkg) {
			skipped = append(skipped, pkg)
		} else {
			toInstall = append(toInstall, pkg)
		}
	}

	// 全部已安装，直接返回
	if len(toInstall) == 0 {
		return pkgActionResult{
			Changed: false,
			Message: fmt.Sprintf("所有包均已安装（共%d个）", len(packages)),
			Detail:  fmt.Sprintf("跳过的包：%s", strings.Join(skipped, ", ")),
		}, nil
	}

	// 执行批量安装命令
	cmdArgs := append(pm.InstallCmd, toInstall...)
	output, err := runCommand(cmdArgs)
	if err != nil {
		return pkgActionResult{}, fmt.Errorf("安装失败：%w，输出：%s", err, output)
	}

	return pkgActionResult{
		Changed: true,
		Message: fmt.Sprintf("安装完成（成功%d个，跳过%d个）", len(toInstall), len(skipped)),
		Detail:  fmt.Sprintf("安装的包：%s", strings.Join(toInstall, ", ")),
	}, nil
}

func batchRemovePackages(pm osPackageManager, packages []string) (pkgActionResult, error) {
	// 筛选需要卸载的包（已安装的）
	toRemove := []string{}
	skipped := []string{}
	for _, pkg := range packages {
		if isPackageInstalled(pm, pkg) {
			toRemove = append(toRemove, pkg)
		} else {
			skipped = append(skipped, pkg)
		}
	}

	// 全部未安装，直接返回
	if len(toRemove) == 0 {
		return pkgActionResult{
			Changed: false,
			Message: fmt.Sprintf("所有包均未安装（共%d个）", len(packages)),
			Detail:  fmt.Sprintf("跳过的包：%s", strings.Join(skipped, ", ")),
		}, nil
	}

	// 执行批量卸载命令
	cmdArgs := append(pm.RemoveCmd, toRemove...)
	output, err := runCommand(cmdArgs)
	if err != nil {
		return pkgActionResult{}, fmt.Errorf("卸载失败：%w，输出：%s", err, output)
	}

	return pkgActionResult{
		Changed: true,
		Message: fmt.Sprintf("卸载完成（成功%d个，跳过%d个）", len(toRemove), len(skipped)),
		Detail:  fmt.Sprintf("卸载的包：%s", strings.Join(toRemove, ", ")),
	}, nil
}

func batchUpdatePackages(pm osPackageManager, packages []string) (pkgActionResult, error) {
	// 筛选需要更新的包（已安装的）
	toUpdate := []string{}
	notInstalled := []string{}
	for _, pkg := range packages {
		if isPackageInstalled(pm, pkg) {
			toUpdate = append(toUpdate, pkg)
		} else {
			notInstalled = append(notInstalled, pkg)
		}
	}

	// 全部未安装，返回错误
	if len(toUpdate) == 0 {
		return pkgActionResult{}, fmt.Errorf("所有包均未安装，无法更新（%s）", strings.Join(notInstalled, ", "))
	}

	// 执行批量更新命令
	cmdArgs := append(pm.UpdateCmd, toUpdate...)
	output, err := runCommand(cmdArgs)
	if err != nil {
		return pkgActionResult{}, fmt.Errorf("更新失败：%w，输出：%s", err, output)
	}

	// 判断是否有实际更新
	changed := !strings.Contains(output, "Nothing to do") &&
		!strings.Contains(output, "无需任何处理") &&
		!strings.Contains(output, "No packages marked for update")

	return pkgActionResult{
		Changed: changed,
		Message: fmt.Sprintf("更新完成（处理%d个，未安装%d个）", len(toUpdate), len(notInstalled)),
		Detail: fmt.Sprintf("更新的包：%s\n未安装的包：%s\n",
			strings.Join(toUpdate, ", "),
			strings.Join(notInstalled, ", ")),
	}, nil
}

// 修改后的batchCheckPackages：直接调用isPackageInstalled，不再依赖checkPackageStatus
func batchCheckPackages(pm osPackageManager, packages []string) (pkgActionResult, error) {
	details := []string{}
	for _, pkg := range packages {
		// 直接调用isPackageInstalled检查状态
		isInstalled := isPackageInstalled(pm, pkg)
		status := "未安装"
		if isInstalled {
			status = "已安装"
		}
		details = append(details, fmt.Sprintf("包「%s」状态：%s", pkg, status))
	}

	return pkgActionResult{
		Changed: false,
		Message: fmt.Sprintf("检查完成，共%d个包", len(packages)),
		Detail:  strings.Join(details, "\n"),
	}, nil
}

func batchLocalInstallPackages(pm osPackageManager, pkgPaths []string) (pkgActionResult, error) {

	missingFiles := []string{}

	for _, path := range pkgPaths {
		cleanPath := filepath.Clean(path)
		// 检查文件是否存在
		if !utils.FileExists(cleanPath) {
			missingFiles = append(missingFiles, cleanPath)
			continue
		}
	}

	// 检查错误
	if len(missingFiles) > 0 {
		return pkgActionResult{}, fmt.Errorf("本地文件不存在：%s", strings.Join(missingFiles, ", "))
	}
	// 筛选需要安装的包（未安装的）
	toInstall := []string{}
	skipped := []string{}
	for _, path := range pkgPaths {
		// 关键问题：本地包路径不能直接用于isPackageInstalled检查，需要先解析包名
		// 修复：添加包名解析逻辑
		pkgName, err := getPackageNameFromFile(pm, path)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s（解析失败：%v）", path, err))
			continue
		}

		if isPackageInstalled(pm, pkgName) {
			skipped = append(skipped, fmt.Sprintf("%s（包名：%s，已安装）", path, pkgName))
		} else {
			toInstall = append(toInstall, path) // 传递文件路径
		}
	}

	// 全部已安装，直接返回
	if len(toInstall) == 0 {
		return pkgActionResult{
			Changed: false,
			Message: fmt.Sprintf("所有本地包均已安装（共%d个）", len(pkgPaths)),
			Detail:  fmt.Sprintf("跳过的包：%s", strings.Join(skipped, "; ")),
		}, nil
	}

	// 执行批量本地安装命令
	cmdArgs := append(pm.LocalInstallCmd, toInstall...)
	output, err := runCommand(cmdArgs)
	if err != nil {

		return pkgActionResult{}, fmt.Errorf("安装失败: %v, 输出: %s", err, output)

	}
	return pkgActionResult{
		Changed: true,
		Message: fmt.Sprintf("本地安装完成（成功%d个，跳过%d个）", len(toInstall), len(skipped)),
		Detail: fmt.Sprintf("安装的包路径：%s\n跳过的包：%s",
			strings.Join(toInstall, ", "),
			strings.Join(skipped, "; ")),
	}, nil
}

// splitPackageList 拆分包列表（逗号分隔，自动过滤空项）
func splitPackageList(nameParam string) []string {
	list := strings.Split(nameParam, ",")
	filtered := make([]string, 0, len(list))
	for _, item := range list {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

// ------------------------------
// 辅助函数
// ------------------------------

// isPackageInstalled 检查包是否已安装
func isPackageInstalled(pm osPackageManager, pkgName string) bool {
	checkCmdArgs := append(pm.CheckCmd, pkgName)
	_, err := exec.Command(checkCmdArgs[0], checkCmdArgs[1:]...).CombinedOutput()

	// 记录详细检查结果
	status := "未安装"
	if err == nil {
		status = "已安装"
	}
	logger.Debug("包安装状态检查",
		zap.String("package", pkgName),
		zap.String("status", status),
		zap.String("cmd", strings.Join(checkCmdArgs, " ")),
		zap.Error(err))

	return err == nil
}

// runCommand 执行命令并返回输出
func runCommand(args []string) (string, error) {
	logger.Info("执行系统命令", zap.String("command", strings.Join(args, " ")))

	output, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	outputStr := string(output)

	// 始终记录命令输出（Debug级别）
	logger.Info("命令执行结果",
		zap.String("command", strings.Join(args, " ")),
		zap.String("output", outputStr),
		zap.Error(err))

	if err != nil {
		return outputStr, fmt.Errorf("命令执行失败：%w", err)
	}
	return outputStr, nil
}
func getPackageNameFromFile(pm osPackageManager, filePath string) (string, error) {
	var cmd *exec.Cmd
	switch pm.PMType {
	case "apt": // DEB包
		// 正确写法：先获取 control 内容，再提取 Package 字段
		cmd = exec.Command("bash", "-c",
			fmt.Sprintf("dpkg-deb -I %s control | grep ^Package: | cut -d' ' -f2", filePath),
		)
	case "yum", "dnf": // RPM包
		cmd = exec.Command("rpm", "-qp", "--queryformat", "%{NAME}", filePath)
	default:
		return "", fmt.Errorf("不支持的包管理器类型：%s", pm.PMType)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("解析包名失败: %w, 输出: %s", err, string(output))
	}

	pkgName := strings.TrimSpace(string(output))
	if pkgName == "" {
		return "", fmt.Errorf("未获取到有效包名")
	}
	return pkgName, nil
}
