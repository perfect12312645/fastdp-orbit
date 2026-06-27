package modules

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// RepoManagerModule 仓库管理模块
type RepoManagerModule struct{}

func NewRepoManagerModule() Module {
	return &RepoManagerModule{}
}

func init() {
	Register("repo", NewRepoManagerModule)
}

// 支持的操作类型
const (
	actionRepoAdd       = "add"
	actionRepoRemove    = "remove"
	actionRepoTest      = "test"
	actionRepoMakecache = "makecache"
	actionRepoBackup    = "backup" // 新增备份操作
	actionRepoRestore   = "restore"
)

// 错误码
const (
	RepoErrInvalidParams = 910
	RepoErrUnsupportedOS = 911
	RepoErrCmdFailed     = 912
	RepoErrRepoExists    = 913
	RepoErrRepoNotExists = 914
	RepoErrRepoTestFail  = 915
)

// 仓库文件配置
const (
	// 基础目录：包含sources.list文件
	aptBaseDir = "/etc/apt/"
	// 额外源目录：包含各类.list文件
	aptSourcesDir    = "/etc/apt/sources.list.d/"
	customRepoPrefix = "fastdp-ops-" // 自定义仓库前缀（用于区分）
	yumRepoDir       = "/etc/yum.repos.d/"
	backupSuffix     = "backup"
)

// Run 实现仓库管理逻辑
func (m *RepoManagerModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 解析参数
	action, actionOk := req.Parameters["action"]
	repoName, nameOk := req.Parameters["name"]
	repoURL, urlOk := req.Parameters["url"]

	// 2. 参数校验
	if !actionOk {
		return utils.ErrorResponse(req, RepoErrInvalidParams, "未指定操作（action，支持：add/remove/test/backup/restore/makecache）"), nil
	}

	if action != actionRepoBackup && action != actionRepoRestore && action != actionRepoMakecache && action != actionRepoTest && !nameOk {
		return utils.ErrorResponse(req, RepoErrInvalidParams, "未指定仓库名称（name）"), nil
	}

	// add操作需要URL
	if (action == actionRepoAdd || action == actionRepoTest) && !urlOk {
		return utils.ErrorResponse(req, RepoErrInvalidParams, "未指定仓库URL（url）"), nil
	}

	logger.Info("开始仓库管理操作",
		zap.String("action", action),
		zap.String("name", repoName),
		zap.String("url", repoURL))

	// 3. 识别系统类型
	osInfo, err := utils.GetLinuxDistribution()
	if err != nil {
		return utils.ErrorResponse(req, RepoErrUnsupportedOS, "获取系统信息失败: "+err.Error()), nil
	}
	isDebian := strings.Contains(strings.ToLower(osInfo), "ubuntu")
	isRPM := strings.Contains(strings.ToLower(osInfo), "centos") ||
		strings.Contains(strings.ToLower(osInfo), "red hat") ||
		strings.Contains(strings.ToLower(osInfo), "rocky") ||
		strings.Contains(strings.ToLower(osInfo), "kylin")

	if !isDebian && !isRPM {
		return utils.ErrorResponse(req, RepoErrUnsupportedOS, "不支持的操作系统: "+osInfo), nil
	}

	// 4. 执行对应操作
	var result *repoActionResult

	switch action {
	case actionRepoAdd:
		if isDebian {
			result, err = manageDebRepo(actionRepoAdd, repoName, repoURL)
		} else {
			result, err = manageYumRepo(actionRepoAdd, repoName, repoURL)
		}
	case actionRepoRemove:
		if isDebian {
			result, err = manageDebRepo(actionRepoRemove, repoName, "")
		} else {
			result, err = manageYumRepo(actionRepoRemove, repoName, "")
		}
	case actionRepoBackup: // 新增备份操作
		if isDebian {
			result, err = backupDebRepo()
		} else {
			result, err = backupYumRepo()
		}
	case actionRepoRestore:
		if isDebian {
			result, err = restoreDebRepo()
		} else {
			result, err = restoreYumRepo()
		}
	case actionRepoTest:
		if isDebian {
			result, err = testDebRepo(repoURL)
		} else {
			result, err = testYumRepo(repoURL)
		}
	case actionRepoMakecache:
		if isDebian {
			result, err = makeAptCache()
		} else {
			result, err = makeYumCache()
		}
	default:
		return utils.ErrorResponse(req, RepoErrInvalidParams, "不支持的操作: "+action), nil
	}
	if err != nil {
		logger.Error("仓库操作失败",
			zap.String("action", action),
			zap.String("name", repoName),
			zap.Error(err))
		return utils.ErrorResponse(req, RepoErrCmdFailed, err.Error()), nil
	}
	logger.Info("仓库操作成功",
		zap.String("action", action),
		zap.String("name", repoName),
		zap.Bool("change", result.Changed),
		zap.String("message", result.Message),
	)
	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    result.Message,
		Changed:   result.Changed,
	}, nil
}

// ------------------------------
// Debian系仓库管理
// ------------------------------

// manageDebRepo 管理Debian系仓库（处理两个目录下的.list文件）
func manageDebRepo(action, repoName, repoURL string) (*repoActionResult, error) {
	// 自定义仓库文件路径
	customRepoFile := filepath.Join(aptSourcesDir, customRepoPrefix+repoName+".list")

	switch action {
	case actionRepoAdd:
		// 检查自定义仓库是否已存在
		if exists, err := isCustomRepoExists(customRepoFile, repoURL); err != nil {
			return nil, fmt.Errorf("检查仓库存在性失败: %w", err)
		} else if exists {
			return &repoActionResult{
				Changed: false,
				Message: fmt.Sprintf("自定义仓库%s已存在，无需添加", repoName),
			}, nil
		}

		// 写入自定义仓库配置
		content := fmt.Sprintf("deb [trusted=yes] %s ./\n", repoURL)
		if err := os.WriteFile(customRepoFile, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("写入仓库文件失败: %w", err)
		}

		return &repoActionResult{
			Changed: true,
			Message: fmt.Sprintf("自定义仓库%s添加成功", repoName),
			Detail:  content,
		}, nil

	case actionRepoRemove:
		// 检查自定义仓库是否存在
		if _, err := os.Stat(customRepoFile); os.IsNotExist(err) {
			return &repoActionResult{
				Changed: false,
				Message: fmt.Sprintf("自定义仓库%s不存在，无需移除", repoName),
			}, nil
		}

		// 删除自定义仓库文件
		if err := os.Remove(customRepoFile); err != nil {
			return nil, fmt.Errorf("删除仓库文件失败: %w", err)
		}

		return &repoActionResult{
			Changed: true,
			Message: fmt.Sprintf("自定义仓库%s已移除", repoName),
		}, nil

	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}
func backupDebRepo() (*repoActionResult, error) {
	// 创建时间戳备份目录
	backupDir := filepath.Join(aptBaseDir, customRepoPrefix+backupSuffix)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 备份主源文件
	if err := backupRepoFile(filepath.Join(aptBaseDir, "sources.list"), backupDir); err != nil {
		return nil, err
	}

	// 备份额外源目录
	sourcesBackupDir := filepath.Join(aptSourcesDir, customRepoPrefix+backupSuffix)
	if err := os.MkdirAll(sourcesBackupDir, 0755); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %w", err)
	}

	files, err := os.ReadDir(aptSourcesDir)
	if err != nil {
		return nil, fmt.Errorf("读取源目录失败: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		src := filepath.Join(aptSourcesDir, file.Name())
		if err := backupRepoFile(src, sourcesBackupDir); err != nil {
			return nil, err
		}
	}

	return &repoActionResult{
		Changed: true,
		Message: fmt.Sprintf("APT源配置已备份到: %s", backupDir),
		Detail:  backupDir,
	}, nil
}
func restoreDebRepo() (*repoActionResult, error) {

	src := filepath.Join(aptBaseDir, customRepoPrefix+backupSuffix, "sources.list")
	dst := filepath.Join(aptBaseDir, "sources.list")
	if _, err := os.Stat(src); err == nil {
		if err := restoreRepoFile(src, dst); err != nil {
			return nil, err
		}
	}

	// 恢复额外源目录
	sourcesBackupDir := filepath.Join(aptSourcesDir, customRepoPrefix+backupSuffix)
	sourcesFiles, err := os.ReadDir(sourcesBackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &repoActionResult{
				Changed: false,
				Message: "未找到APT源备份目录，无需恢复",
				Detail:  sourcesBackupDir,
			}, nil
		}
		return nil, fmt.Errorf("读取备份源目录失败: %w", err)
	}

	for _, file := range sourcesFiles {
		src := filepath.Join(sourcesBackupDir, file.Name())
		dst := filepath.Join(aptSourcesDir, file.Name())
		if err := restoreRepoFile(src, dst); err != nil {
			return nil, err
		}
	}

	// 保留备份目录（不移除）
	return &repoActionResult{
		Changed: true,
		Message: fmt.Sprintf("APT源配置已从备份恢复: %s", aptSourcesDir),
		Detail:  aptSourcesDir,
	}, nil
}
func backupRepoFile(src, backupDir string) error {
	// 确保源文件存在
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil // 文件不存在，跳过
	}

	// 创建备份
	fileName := filepath.Base(src)
	dst := filepath.Join(backupDir, fileName)

	err := os.Rename(src, dst)
	if err != nil {
		return err
	}
	return nil
}
func restoreRepoFile(src, dst string) error {
	// 确保备份文件存在
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", src)
	}

	err := os.Rename(src, dst)
	if err != nil {
		return err
	}
	return nil
}

// isCustomRepoExists 检查自定义仓库文件是否存在且内容匹配
func isCustomRepoExists(repoFile, repoURL string) (bool, error) {
	// 检查文件是否存在
	if _, err := os.Stat(repoFile); os.IsNotExist(err) {
		return false, nil
	}

	// 检查文件内容是否匹配
	content, err := os.ReadFile(repoFile)
	if err != nil {
		return false, fmt.Errorf("读取仓库文件%s失败: %w", repoFile, err)
	}

	expectedContent := fmt.Sprintf("deb [trusted=yes] %s ./\n", repoURL)
	return string(content) == expectedContent, nil
}

// ------------------------------
// YUM系仓库管理
// ------------------------------

func manageYumRepo(action, repoName, repoURL string) (*repoActionResult, error) {
	repoFile := filepath.Join(yumRepoDir, "fastdp-ops-"+repoName+".repo")

	switch action {
	case actionRepoAdd:
		// 检查是否已存在
		if repoExists, _ := yumRepoExists(repoName); repoExists {
			return &repoActionResult{
				Changed: false,
				Message: "仓库已存在，无需添加",
			}, nil
		}

		// 写入新仓库配置
		content := fmt.Sprintf(`[%s]
name=FastDP OPS Repository - %s
baseurl=%s
enabled=1
gpgcheck=0
`, repoName, repoName, repoURL)

		if err := os.WriteFile(repoFile, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("写入仓库文件失败: %w", err)
		}

		return &repoActionResult{
			Changed: true,
			Message: "YUM仓库添加成功",
			Detail:  content,
		}, nil

	case actionRepoRemove:
		// 检查是否存在
		if _, err := os.Stat(repoFile); os.IsNotExist(err) {
			return &repoActionResult{
				Changed: false,
				Message: "仓库不存在，无需移除",
			}, nil
		}

		// 直接删除
		if err := os.Remove(repoFile); err != nil {
			return nil, fmt.Errorf("删除仓库文件失败: %w", err)
		}

		return &repoActionResult{
			Changed: true,
			Message: "仓库已移除",
		}, nil
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}
func backupYumRepo() (*repoActionResult, error) {

	backupDir := filepath.Join(yumRepoDir, customRepoPrefix+backupSuffix)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 备份所有.repo文件
	files, err := os.ReadDir(yumRepoDir)
	if err != nil {
		return nil, fmt.Errorf("读取仓库目录失败: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".repo") {
			continue
		}

		src := filepath.Join(yumRepoDir, file.Name())
		if err := backupRepoFile(src, backupDir); err != nil {
			return nil, err
		}
	}

	return &repoActionResult{
		Changed: true,
		Message: fmt.Sprintf("YUM仓库配置已备份到: %s", backupDir),
		Detail:  backupDir,
	}, nil
}
func restoreYumRepo() (*repoActionResult, error) {
	// 获取最新的备份目录
	backupRoot := filepath.Join(yumRepoDir, customRepoPrefix+backupSuffix)

	// 恢复所有.repo文件
	files, err := os.ReadDir(backupRoot)
	if err != nil {
		return nil, fmt.Errorf("读取备份文件失败: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".repo") {
			continue
		}

		src := filepath.Join(backupRoot, file.Name())
		dst := filepath.Join(yumRepoDir, file.Name())
		if err := restoreRepoFile(src, dst); err != nil {
			return nil, err
		}
	}

	// 保留备份目录（不移除）
	return &repoActionResult{
		Changed: true,
		Message: fmt.Sprintf("YUM仓库配置已从备份恢复: %s", backupRoot),
		Detail:  backupRoot,
	}, nil
}

func yumRepoExists(repoName string) (bool, error) {
	// 使用yum repolist检查更可靠
	cmd := exec.Command("yum", "repolist")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("执行yum repolist失败: %w", err)
	}

	return strings.Contains(string(output), repoName), nil
}

// ------------------------------
// 仓库测试
// ------------------------------

func testDebRepo(repoURL string) (*repoActionResult, error) {
	// 简单测试：尝试获取Release文件
	testURL := strings.TrimSuffix(repoURL, "/") + "/Release"
	cmd := exec.Command("curl", "-sI", testURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("测试APT仓库失败: %w", err)
	}

	if !strings.Contains(string(output), "200 OK") {
		return nil, fmt.Errorf("仓库不可达: %s", string(output))
	}

	return &repoActionResult{
		Changed: false,
		Message: "APT仓库测试成功",
		Detail:  string(output),
	}, nil
}

func testYumRepo(repoURL string) (*repoActionResult, error) {
	// 使用curl测试baseurl是否可达
	testURL := strings.TrimSuffix(repoURL, "/") + "/repodata/repomd.xml"
	cmd := exec.Command("curl", "-sI", testURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("测试YUM仓库失败: %w", err)
	}

	if !strings.Contains(string(output), "200 OK") {
		return nil, fmt.Errorf("仓库不可达: %s", string(output))
	}

	return &repoActionResult{
		Changed: false,
		Message: "YUM仓库测试成功",
		Detail:  string(output),
	}, nil
}

// ------------------------------
// 缓存更新
// ------------------------------

func makeAptCache() (*repoActionResult, error) {
	cmd := exec.Command("apt", "update")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("APT缓存更新失败: %w\n输出: %s", err, string(output))
	}

	return &repoActionResult{
		Changed: true,
		Message: "APT缓存更新成功",
		Detail:  string(output),
	}, nil
}

func makeYumCache() (*repoActionResult, error) {
	cmd := exec.Command("yum", "makecache")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("YUM缓存更新失败: %w\n输出: %s", err, string(output))
	}

	return &repoActionResult{
		Changed: true,
		Message: "YUM缓存更新成功",
		Detail:  string(output),
	}, nil
}

// ------------------------------
// 辅助结构和方法
// ------------------------------

type repoActionResult struct {
	Changed bool   // 是否产生变更
	Message string // 简要信息
	Detail  string // 详细输出
}
