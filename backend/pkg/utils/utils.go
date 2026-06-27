package utils

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"os"
	"strings"
	"time"

	"fastdp-orbit/backend/proto/agent"
)

// GenerateID generates a unique ID
func GenerateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// MD5 calculates MD5 hash
func MD5(data string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

// Contains checks if a slice contains an element
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Map converts a slice to a map
func Map(slice []string) map[string]bool {
	result := make(map[string]bool)
	for _, item := range slice {
		result[item] = true
	}
	return result
}

// ==================== Agent 模块通用辅助函数 ====================

// ErrorResponse 构造失败响应（模块内统一用法，避免重复代码）
func ErrorResponse(req *agent.ExecRequest, code int32, message string) *agent.ExecResponse {
	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   false,
		Error: &agent.ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

// SuccessResponse 构造成功响应
func SuccessResponse(req *agent.ExecRequest, stdout string) *agent.ExecResponse {
	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    stdout,
	}
}

// FileExists 判断文件/目录是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SuccessResponseWithNoChange 构造成功响应（Changed=false，与 SuccessResponse 等价，兼容旧模块调用）
func SuccessResponseWithNoChange(req *agent.ExecRequest, stdout string) *agent.ExecResponse {
	return &agent.ExecResponse{
		MachineId: req.MachineId,
		TaskId:    req.TaskId,
		Success:   true,
		Stdout:    stdout,
	}
}

// FileMD5 计算文件的 MD5 哈希值
func FileMD5(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	// 逐块读取文件内容
	buf := make([]byte, 4096)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// GetLinuxDistribution 读取 /etc/os-release 获取 Linux 发行版名称（小写，如 ubuntu、centos、debian）
func GetLinuxDistribution() (string, error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "ID=") {
			id := strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			return strings.ToLower(id), nil
		}
	}
	return "", fmt.Errorf("无法识别 Linux 发行版")
}
