package modules

import (
	"context"
	"errors"
	"fastdp-orbit/backend/agent/grpc"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/pkg/utils"
	"fastdp-orbit/backend/proto/agent"
	"fastdp-orbit/backend/proto/server"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type CopyModule struct{}

func NewCopyModule() Module {
	return &CopyModule{}
}

func init() {
	Register("copy", NewCopyModule)
}

// 新增参数常量定义
const (
	defaultCopyType   = "file" // 默认拷贝类型：文件
	defaultRecursive  = false  // 默认不递归
	copyTypeFile      = "file" // 文件拷贝类型
	copyTypeDirectory = "dir"  // 目录拷贝类型
)

// Run 实现文件拉取逻辑：带MD5校验、存在性判断和权限检查
func (m *CopyModule) Run(req *agent.ExecRequest) (*agent.ExecResponse, error) {
	// 1. 解析参数
	srcPath, ok := req.Parameters["src"]
	if !ok || srcPath == "" || !filepath.IsAbs(srcPath) {
		return utils.ErrorResponse(req, 3001, fmt.Sprintf("源文件位置src必须输入且需为绝对路径，当前输入为:%s", srcPath)), nil
	}

	destPath, ok := req.Parameters["dest"]
	if !ok || destPath == "" || !filepath.IsAbs(destPath) {
		return utils.ErrorResponse(req, 3002, fmt.Sprintf("目标文件位置dest必须输入且需为绝对路径，当前输入为:%s", destPath)), nil
	}
	// 解析拷贝类型（默认文件）
	copyType := req.Parameters["type"]
	if copyType == "" {
		copyType = defaultCopyType
	}
	if copyType != copyTypeFile && copyType != copyTypeDirectory {
		return utils.ErrorResponse(req, 3003, fmt.Sprintf("不支持的拷贝类型type：%s，支持file/dir", copyType)), nil
	}
	recursive := defaultRecursive
	if recursiveStr, ok := req.Parameters["recursive"]; ok && recursiveStr != "" {
		var err error
		recursive, err = strconv.ParseBool(recursiveStr)
		if err != nil {
			return utils.ErrorResponse(req, 3004, fmt.Sprintf("递归参数格式错误（应为true/false）：%v", err)), nil
		}
	}
	// 解析文件权限（mode）：八进制转uint32，未输入则为0（不处理）
	var fileMode uint32 = 0
	if modeStr, ok := req.Parameters["mode"]; ok && modeStr != "" {
		mode, err := strconv.ParseUint(modeStr, 8, 32) // 权限为八进制（如0644）
		if err != nil {
			return utils.ErrorResponse(req, 3004, fmt.Sprintf("权限格式错误（应为八进制，如0644）：%v", err)), nil
		}
		fileMode = uint32(mode)
	}

	// 解析目标文件MD5
	targetMD5 := req.Parameters["md5"]

	// 如果 dest 是目录，自动拼接 src 的文件名
	destInfo, destStatErr := os.Stat(destPath)
	if destStatErr == nil && destInfo.IsDir() {
		destPath = filepath.Join(destPath, filepath.Base(srcPath))
	}

	switch copyType {
	case copyTypeFile:
		// 保持原有文件拷贝逻辑
		return m.copyFile(req, srcPath, destPath, targetMD5, fileMode)
	case copyTypeDirectory:
		// 新增目录拷贝逻辑
		return m.copyDirectory(req, srcPath, destPath, recursive, fileMode)
	default:
		return utils.ErrorResponse(req, 3003, fmt.Sprintf("不支持的拷贝类型：%s", copyType)), nil
	}

}

func (m *CopyModule) copyFile(req *agent.ExecRequest, srcPath, destPath, targetMD5 string, fileMode uint32) (*agent.ExecResponse, error) {
	// 2. 检查目标文件是否已存在且符合条件（MD5匹配 + 权限检查）
	if targetMD5 != "" {
		exists, sameMD5, currentMode, err := m.checkExistingFile(destPath, targetMD5)
		if err != nil {
			return utils.ErrorResponse(req, 3009, fmt.Sprintf("检查目标文件失败：%v", err)), nil
		}

		if exists && sameMD5 {
			// 用户未指定权限（fileMode=0），直接返回无需传输
			if fileMode == 0 {
				detail := "文件已存在且MD5匹配，无需传输（未指定权限，不做权限检查）"
				logger.Info(detail)
				return utils.SuccessResponse(req, "文件无需传输"), nil
			}

			// 用户指定了权限，处理权限判断与调整
			permChanged, err := m.adjustFilePermission(destPath, currentMode, fileMode)
			if err != nil {
				return utils.ErrorResponse(req, 3010, fmt.Sprintf("调整文件权限失败：%v", err)), nil
			}
			if !permChanged {
				// 返回成功（说明无需传输，可能仅修改了权限）
				detail := fmt.Sprintf("文件已存在且MD5匹配，无需传输。权限状态：%s",
					map[bool]string{true: "已更新", false: "未变更"}[permChanged])
				logger.Info(detail)
				return utils.SuccessResponse(req, "文件无需传输"), nil
			}
			// 返回成功（说明无需传输，可能仅修改了权限）
			detail := fmt.Sprintf("文件已存在且MD5匹配，无需传输。权限状态：%s",
				map[bool]string{true: "已更新", false: "未变更"}[permChanged])
			logger.Info(detail)
			return utils.SuccessResponse(req, "文件无需传输"), nil
		}
	}

		// 3. 目标文件不存在或不匹配，执行文件传输
		conn, err := grpc.GetExistingConn()
		if err != nil {
			logger.Error("获取gRPC连接失败", zap.Error(err))
			return utils.ErrorResponse(req, 3005, "连接服务器失败"), nil
		}

	transferClient := server.NewServerServiceClient(conn)
	// 使用请求中的超时时间，0 或未设置则永不超时
	var ctx context.Context
	var cancel context.CancelFunc
	if req.GetTimeout() > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(req.GetTimeout())*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	stream, err := transferClient.CopyFile(ctx, &server.CopyFileRequest{
		SourcePath: srcPath,
	})
	if err != nil {
		logger.Error("发起文件拉取请求失败", zap.String("task_id", req.TaskId), zap.Error(err))
		return utils.ErrorResponse(req, 3006, fmt.Sprintf("拉取文件请求失败：%v", err)), nil
	}

	// 接收文件流并写入本地
	receivedSize, err := m.receiveFileStream(stream, destPath)
	if err != nil {
		logger.Error("接收文件流失败", zap.String("target", destPath), zap.Error(err))
		return utils.ErrorResponse(req, 3007, fmt.Sprintf("接收文件失败：%v", err)), nil
	}

	// 设置文件权限（如果用户指定了mode）
	if fileMode != 0 {
		if err := os.Chmod(destPath, os.FileMode(fileMode)); err != nil {
			logger.Warn("设置文件权限失败", zap.String("path", destPath), zap.Uint32("mode", fileMode), zap.Error(err))
			// 权限设置失败不阻断主流程
		}
	}

	// 返回成功结果
	detail := fmt.Sprintf("文件拉取完成，路径：%s，大小：%d字节，权限：%#o", destPath, receivedSize, fileMode)
	logger.Info(detail, zap.String("task_id", req.TaskId))
	return utils.SuccessResponse(req, "文件拉取成功"), nil
}

func (m *CopyModule) copyDirectory(req *agent.ExecRequest, srcDir, destDir string, recursive bool, dirMode uint32) (*agent.ExecResponse, error) {
	// 1. 获取gRPC连接
	conn, err := grpc.GetExistingConn()
	if err != nil {
		logger.Error("获取gRPC连接失败", zap.Error(err))
		return utils.ErrorResponse(req, 3005, "连接服务器失败"), nil
	}

	// 2. 创建文件传输客户端
	transferClient := server.NewServerServiceClient(conn)
	// 使用请求中的超时时间，0 或未设置则永不超时
	var ctx context.Context
	var cancel context.CancelFunc
	if req.GetTimeout() > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(req.GetTimeout())*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	// 3. 获取目录列表
	listReq := &server.ListDirectoryRequest{
		Path:      srcDir,
		Recursive: recursive,
	}
	listResp, err := transferClient.ListDirectory(ctx, listReq)
	if err != nil {
		logger.Error("获取目录列表失败", zap.String("src", srcDir), zap.Error(err))
		return utils.ErrorResponse(req, 3018, fmt.Sprintf("获取目录列表失败: %v", err)), nil
	}

	// 4. 创建目标目录（如果不存在）
	if destInfo, err := os.Stat(destDir); err == nil {
		if !destInfo.IsDir() {
			return utils.ErrorResponse(req, 3014, fmt.Sprintf("目标路径已存在且不是目录: %s", destDir)), nil
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(destDir, os.FileMode(dirMode|0755)); err != nil {
			return utils.ErrorResponse(req, 3016, fmt.Sprintf("创建目标目录失败: %v", err)), nil
		}
	} else {
		return utils.ErrorResponse(req, 3015, fmt.Sprintf("获取目标目录信息失败: %v", err)), nil
	}

	// 5. 设置目标目录权限（如果指定）
	if dirMode != 0 {
		if err := os.Chmod(destDir, os.FileMode(dirMode)); err != nil {
			logger.Warn("设置目录权限失败", zap.String("path", destDir), zap.Uint32("mode", dirMode), zap.Error(err))
		}
	}

	// 6. 处理目录和文件
	var totalFiles, successFiles, skippedFiles int
	var errors []string

	for _, fileInfo := range listResp.Files {
		// 构建本地完整路径
		localPath := filepath.Join(destDir, fileInfo.Path)

		if fileInfo.IsDir {
			// 处理目录
			if err := os.MkdirAll(localPath, os.FileMode(dirMode|0755)); err != nil {
				errors = append(errors, fmt.Sprintf("创建目录失败 (%s): %v", fileInfo.Path, err))
				continue
			}
			// 设置目录权限
			if dirMode != 0 {
				if err := os.Chmod(localPath, os.FileMode(dirMode)); err != nil {
					logger.Warn("设置目录权限失败", zap.String("path", localPath), zap.Error(err))
				}
			}
		} else {
			// 处理文件
			totalFiles++

			// 创建父目录（如果不存在）
			parentDir := filepath.Dir(localPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				errors = append(errors, fmt.Sprintf("创建父目录失败 (%s): %v", parentDir, err))
				continue
			}

			// 对比本地文件 mtime 与 server 端 mtime
			localInfo, statErr := os.Stat(localPath)
			if statErr == nil && !localInfo.IsDir() {
				// fileInfo.ModifiedTime 是 server 返回的 unix 时间戳
				if localInfo.ModTime().Unix() >= fileInfo.ModifiedTime {
					skippedFiles++
					logger.Debug("文件已存在且mtime不旧于server，跳过拷贝", zap.String("path", fileInfo.Path))
					continue
				}
			}

			// 复制文件（使用绝对路径）
			resp, err := m.copyFile(req, filepath.Join(srcDir, fileInfo.Path), localPath, "", 0)
			if err != nil {
				errors = append(errors, fmt.Sprintf("文件复制失败 (%s): %v", fileInfo.Path, err))
				continue
			}

			if resp.Success {
				successFiles++
			} else if resp.Error != nil {
				errors = append(errors, fmt.Sprintf("文件复制失败 (%s): %s", fileInfo.Path, resp.Error.Message))
			}
		}
	}

	// 7. 生成结果信息
	detail := fmt.Sprintf("目录复制完成: %s → %s\n文件总数: %d, 成功: %d, 跳过: %d, 失败: %d",
		srcDir, destDir, totalFiles, successFiles, skippedFiles, len(errors))

	if len(errors) > 0 {
		detail += "\n错误详情:\n" + strings.Join(errors, "\n")
		logger.Warn(detail, zap.String("task_id", req.TaskId))
		return utils.SuccessResponse(req, "目录复制部分成功"), nil
	}

	logger.Info(detail, zap.String("task_id", req.TaskId))
	return utils.SuccessResponse(req, "目录复制成功"), nil
}

// 检查目标文件是否存在、MD5是否匹配，并返回当前权限
func (m *CopyModule) checkExistingFile(destPath, targetMD5 string) (exists bool, sameMD5 bool, currentMode uint32, err error) {
	fileInfo, err := os.Stat(destPath)
	if os.IsNotExist(err) {
		return false, false, 0, nil // 文件不存在
	}
	if fileInfo.IsDir() {
		return false, false, 0, fmt.Errorf("目标位置已存在为目录,请手动查看")
	}
	if err != nil {
		return false, false, 0, fmt.Errorf("获取文件信息失败：%w", err) // 其他错误（如权限不足）
	}

	// 计算现有文件的MD5
	currentMD5, err := utils.FileMD5(destPath)
	if err != nil {
		return true, false, 0, fmt.Errorf("计算文件MD5失败：%w", err)
	}

	// 比较MD5
	sameMD5 = (currentMD5 == targetMD5)
	// 获取当前文件权限（仅保留权限位，去除其他信息）
	currentMode = uint32(fileInfo.Mode() & os.ModePerm)
	return true, sameMD5, currentMode, nil
}

// 调整文件权限（仅当用户指定了mode且与当前权限不一致时）
func (m *CopyModule) adjustFilePermission(destPath string, currentMode, targetMode uint32) (changed bool, err error) {
	if currentMode == targetMode {
		return false, nil // 权限一致，无需修改
	}

	// 修改权限
	if err := os.Chmod(destPath, os.FileMode(targetMode)); err != nil {
		return false, fmt.Errorf("修改权限失败：%w", err)
	}
	return true, nil
}

// receiveFileStream 接收流式文件内容并写入目标路径
func (m *CopyModule) receiveFileStream(stream server.ServerService_CopyFileClient, destPath string) (int64, error) {
	// 创建目标目录（如果不存在）
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("创建目标目录失败：%w", err)
	}

	// 创建临时文件（避免传输中断导致目标文件损坏）
	tempFile, err := os.CreateTemp(destDir, ".copy-*.tmp")
	if err != nil {
		return 0, fmt.Errorf("创建临时文件失败：%w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		// 无论成功失败，都清理临时文件（成功时会先重命名）
		if _, err := os.Stat(tempPath); err == nil {
			_ = os.Remove(tempPath)
		}
	}()

	// 循环接收文件块并写入临时文件
	var totalReceived int64
	for {
		chunk, err := stream.Recv()
		if err != nil {
			// 流正常结束
			if err == io.EOF {
				break
			}
			// 超时或取消
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return totalReceived, fmt.Errorf("接收超时：%w", err)
			}
			// 其他错误
			return totalReceived, fmt.Errorf("接收文件块失败：%w", err)
		}

		// 写入文件块
		n, err := tempFile.Write(chunk.Data)
		if err != nil {
			_ = tempFile.Close()
			return totalReceived, fmt.Errorf("写入文件块失败：%w", err)
		}
		totalReceived += int64(n)

		// 打印进度日志（每接收1MB打印一次）
		if totalReceived > 0 && totalReceived%(1024*1024) == 0 {
			logger.Info("文件接收中",
				zap.String("dest", destPath),
				zap.Int64("received", totalReceived))
		}
	}

	// 关闭临时文件（确保数据刷盘）
	if err := tempFile.Close(); err != nil {
		return totalReceived, fmt.Errorf("关闭临时文件失败：%w", err)
	}

	// 原子重命名临时文件为目标文件（避免部分写入的文件被使用）
	if err := os.Rename(tempPath, destPath); err != nil {
		return totalReceived, fmt.Errorf("重命名文件失败：%w", err)
	}

	return totalReceived, nil
}
