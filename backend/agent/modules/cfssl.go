package modules

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// CfsslModule 证书管理模块
type CfsslModule struct{}

func NewCfsslModule() Module {
	return &CfsslModule{}
}

func init() {
	Register("cfssl", NewCfsslModule)
}

// 支持的操作类型
const (
	actionGenerateCA   = "generate_ca"   // 生成CA证书
	actionGenerateCert = "generate_cert" // 生成普通证书
)

// 错误码
const (
	CfsslErrInvalidParams = 910 // 参数错误
	CfsslErrCmdFailed     = 911 // 命令执行失败
	CfsslErrFileExists    = 912 // 文件已存在
)

// Run 实现模块核心逻辑
func (m *CfsslModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	logger.Info("开始证书管理操作",
		zap.String("machine", req.MachineId),
		zap.String("task", req.TaskId))

	// 1. 解析参数
	action, actionOk := req.Parameters["action"]
	if !actionOk {
		return utils.ErrorResponse(req, CfsslErrInvalidParams, "未指定操作（action，支持：generate_ca/generate_cert）"), nil
	}

	switch action {
	case actionGenerateCA:
		return m.generateCA(req)
	case actionGenerateCert:
		return m.generateCert(req)
	default:
		return utils.ErrorResponse(req, CfsslErrInvalidParams, "不支持的操作类型："+action), nil
	}
}

// generateCA 生成CA证书
func (m *CfsslModule) generateCA(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 解析必要参数
	csrPath, ok1 := req.Parameters["csr_path"]
	outputDir, ok2 := req.Parameters["output_dir"]
	basename, ok3 := req.Parameters["basename"]

	if !ok1 || !ok2 || !ok3 {
		return utils.ErrorResponse(req, CfsslErrInvalidParams,
			"缺少必要参数：csr_path, output_dir, basename"), nil
	}

	// 检查证书文件是否已存在（幂等性检查）
	certPath := filepath.Join(outputDir, basename+".pem")
	keyPath := filepath.Join(outputDir, basename+"-key.pem")

	if m.certificateExists(certPath, keyPath) {
		msg := fmt.Sprintf("CA证书已存在：%s 和 %s", certPath, keyPath)
		logger.Info(msg, zap.String("task", req.TaskId))
		return utils.SuccessResponseWithNoChange(req, msg), nil
	}

	// 创建输出目录（如果不存在）
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return utils.ErrorResponse(req, CfsslErrCmdFailed,
			"创建目录失败: "+err.Error()), nil
	}

	// 执行命令：cfssl gencert -initca | cfssljson -bare
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		"cfssl gencert -initca %s | cfssljson -bare %s",
		csrPath, filepath.Join(outputDir, basename)))

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		logger.Error("生成CA证书失败",
			zap.String("command", cmd.String()),
			zap.String("output", outputStr),
			zap.Error(err))

		return utils.ErrorResponse(req, CfsslErrCmdFailed,
			fmt.Sprintf("命令执行失败: %v, 输出: %s", err, outputStr)), nil
	}

	// 验证生成的证书文件
	if !m.certificateExists(certPath, keyPath) {
		return utils.ErrorResponse(req, CfsslErrCmdFailed,
			"证书文件未生成: "+certPath+" 或 "+keyPath), nil
	}

	msg := fmt.Sprintf("成功生成CA证书: %s 和 %s", certPath, keyPath)
	logger.Info(msg, zap.String("task", req.TaskId))

	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    outputStr,
		Changed:   true,
	}, nil
}

// generateCert 生成普通证书（修复版本：使用管道调用cfssljson）
func (m *CfsslModule) generateCert(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 解析必要参数
	csrPath, ok1 := req.Parameters["csr_path"]
	outputDir, ok2 := req.Parameters["output_dir"]
	basename, ok3 := req.Parameters["basename"]
	caCert, ok4 := req.Parameters["ca_cert"]
	caKey, ok5 := req.Parameters["ca_key"]
	configFile, ok6 := req.Parameters["config_file"]
	profile, ok7 := req.Parameters["profile"]

	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 || !ok7 {
		return utils.ErrorResponse(req, CfsslErrInvalidParams,
			"缺少必要参数：csr_path, output_dir, basename, ca_cert, ca_key, config_file, profile"), nil
	}

	// 检查证书文件是否已存在（幂等性检查）
	certPath := filepath.Join(outputDir, basename+".pem")
	keyPath := filepath.Join(outputDir, basename+"-key.pem")

	if m.certificateExists(certPath, keyPath) {
		msg := fmt.Sprintf("证书已存在：%s 和 %s", certPath, keyPath)
		logger.Info(msg, zap.String("task", req.TaskId))
		return utils.SuccessResponseWithNoChange(req, msg), nil
	}

	// 创建输出目录（如果不存在）
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return utils.ErrorResponse(req, CfsslErrCmdFailed,
			"创建目录失败: "+err.Error()), nil
	}

	// 构建完整命令：使用bash -c支持管道，格式与示例一致
	// 示例：cfssl gencert -ca=xxx -ca-key=xxx -config=xxx -profile=xxx csr.json | cfssljson -bare 输出前缀
	fullOutputPath := filepath.Join(outputDir, basename)
	cmdStr := fmt.Sprintf(
		"cfssl gencert -ca=%s -ca-key=%s -config=%s -profile=%s %s | cfssljson -bare %s",
		caCert, caKey, configFile, profile, csrPath, fullOutputPath)

	// 执行命令（使用bash -c解析完整命令串，支持管道和换行）
	cmd := exec.Command("bash", "-c", cmdStr)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		logger.Error("生成证书失败",
			zap.String("command", cmdStr),
			zap.String("output", outputStr),
			zap.Error(err))

		return utils.ErrorResponse(req, CfsslErrCmdFailed,
			fmt.Sprintf("命令执行失败: %v, 输出: %s", err, outputStr)), nil
	}

	// 验证生成的证书文件
	if !m.certificateExists(certPath, keyPath) {
		return utils.ErrorResponse(req, CfsslErrCmdFailed,
			"证书文件未生成: "+certPath+" 或 "+keyPath), nil
	}

	msg := fmt.Sprintf("成功生成证书: %s 和 %s", certPath, keyPath)
	logger.Info(msg, zap.String("task", req.TaskId))

	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    outputStr,
		Changed:   true,
	}, nil
}

// certificateExists 检查证书文件是否已存在
func (m *CfsslModule) certificateExists(certPath, keyPath string) bool {
	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	return certErr == nil && keyErr == nil
}
