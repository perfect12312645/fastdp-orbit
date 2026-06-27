package modules

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
)

// 定义错误码常量
const (
	ErrorCodeInvalidParams = 500
)

type FilePullModule struct{}

func NewFilePullModule() Module {
	return &FilePullModule{}
}

func init() {
	Register("file_pull", NewFilePullModule)
}

func (m *FilePullModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 参数校验
	url, urlok := req.Parameters["url"]
	md5sum, md5sumok := req.Parameters["md5"]
	dest, destok := req.Parameters["dest"]

	// 检查必填参数
	if !urlok || url == "" {
		return utils.ErrorResponse(req, ErrorCodeInvalidParams, "未传入必须参数: url"), nil
	}
	if !destok || dest == "" || !filepath.IsAbs(dest) {
		return utils.ErrorResponse(req, ErrorCodeInvalidParams,
			fmt.Sprintf("目标路径dest必须输入且需为绝对路径，当前输入为:%s", dest)), nil
	}

	// 自动判断类型：如果 dest 以 / 结尾或已存在为目录，则为 dir 模式
	destType := "file"
	if strings.HasSuffix(dest, "/") {
		destType = "dir"
		dest = strings.TrimRight(dest, "/")
	} else if info, err := os.Stat(dest); err == nil && info.IsDir() {
		destType = "dir"
	}

	// 2. 确定最终目标路径
	var finalDest string
	switch destType {
	case "file":
		// 检查文件状态
		if f, err := os.Stat(dest); err == nil && f.IsDir() {
			return utils.ErrorResponse(req, ErrorCodeInvalidParams,
				fmt.Sprintf("目标位置为目录，请手动处理:%s", dest)), nil
		}
		finalDest = dest

	case "dir":
		// 从URL提取文件名
		fileName := path.Base(strings.TrimRight(url, "/"))
		finalDest = filepath.Join(dest, fileName)
	}

	// 3. MD5 校验（如果提供了 md5 参数）
	if md5sumok && md5sum != "" {
		if _, err := os.Stat(finalDest); err == nil {
			// 文件已存在，计算当前MD5
			currentMd5, err := utils.FileMD5(finalDest)
			if err != nil {
				logger.Warn("文件存在但计算MD5失败，将重新拉取",
					zap.String("dest", finalDest),
					zap.Error(err))
			} else if currentMd5 == md5sum {
				// MD5匹配，无需拉取
				logger.Info("文件已存在且MD5匹配，无需拉取",
					zap.String("dest", finalDest),
					zap.String("md5", currentMd5))
				return utils.SuccessResponseWithNoChange(req,
					fmt.Sprintf("文件已存在且MD5匹配，无需拉取: %s", finalDest)), nil
			} else {
				// MD5不匹配，需要重新拉取
				logger.Info("文件已存在但MD5不匹配，将重新拉取",
					zap.String("dest", finalDest),
					zap.String("expected_md5", md5sum),
					zap.String("current_md5", currentMd5))
			}
		} else if !os.IsNotExist(err) {
			logger.Warn("检查文件状态异常，将继续拉取",
				zap.String("dest", finalDest),
				zap.Error(err))
		}
	}

	// 4. 执行文件拉取
	return pullFile(url, finalDest, md5sum, req)
}

func pullFile(url, dest, md5sum string, req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 创建 HTTP 客户端（支持 HTTPS 跳过证书验证）
	// 使用请求中的超时时间，0 或未设置则永不超时
	client := &http.Client{
		Timeout: time.Duration(req.GetTimeout()) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 支持自签名证书的 HTTPS
			},
		},
	}

	// 2. 下载文件
	resp, err := client.Get(url)
	if err != nil {
		return utils.ErrorResponse(req, ErrorCodeInvalidParams,
			fmt.Sprintf("文件拉取异常: %s", err.Error())), nil
	}
	defer resp.Body.Close()

	// 检查HTTP状态
	if resp.StatusCode != http.StatusOK {
		return utils.ErrorResponse(req, ErrorCodeInvalidParams,
			fmt.Sprintf("拉取文件失败，状态码: %s", resp.Status)), nil
	}

	// 3. 创建目标目录
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return utils.ErrorResponse(req, ErrorCodeInvalidParams,
			fmt.Sprintf("创建目录失败: %s", err.Error())), nil
	}

	// 4. 创建临时文件
	tempFile, err := os.CreateTemp(filepath.Dir(dest), ".file_pull_*.tmp")
	if err != nil {
		return utils.ErrorResponse(req, ErrorCodeInvalidParams,
			fmt.Sprintf("创建临时文件失败: %s", err.Error())), nil
	}
	tempPath := tempFile.Name()
	defer func() {
		// 清理临时文件
		if _, err := os.Stat(tempPath); err == nil {
			_ = os.Remove(tempPath)
		}
	}()

	// 5. 写入临时文件
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		_ = tempFile.Close()
		return utils.ErrorResponse(req, ErrorCodeInvalidParams,
			fmt.Sprintf("写入文件失败: %s", err.Error())), nil
	}
	_ = tempFile.Close()

	// 6. 校验 MD5（如果提供了 md5 参数）
	if md5sum != "" {
		destMd5, err := utils.FileMD5(tempPath)
		if err != nil {
			return utils.ErrorResponse(req, ErrorCodeInvalidParams,
				fmt.Sprintf("计算文件md5失败:%s", err.Error())), nil
		}

		if md5sum != destMd5 {
			return utils.ErrorResponse(req, ErrorCodeInvalidParams,
				fmt.Sprintf("md5值不匹配: 期望 %s, 实际 %s", md5sum, destMd5)), nil
		}
	}

	// 7. 原子重命名
	if err := os.Rename(tempPath, dest); err != nil {
		return utils.ErrorResponse(req, ErrorCodeInvalidParams,
			fmt.Sprintf("重命名文件失败: %s", err.Error())), nil
	}

	// 8. 返回成功
	successMsg := fmt.Sprintf("文件%s已拉取至%s", url, dest)
	logger.Info("文件拉取成功", zap.String("url", url), zap.String("dest", dest))
	return utils.SuccessResponse(req, successMsg), nil
}
