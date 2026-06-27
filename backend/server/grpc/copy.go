package grpc

import (
	"bufio"
	"context"
	"errors"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/proto/server"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// 定义块大小为64KB（gRPC流式传输推荐大小）
const chunkSize = 64 * 1024

type CopyServer struct {
	server.UnimplementedServerServiceServer
}

// CopyFile 实现服务端流式传输：读取本地文件并分块发送给agent
func (m *CopyServer) CopyFile(req *server.CopyFileRequest, stream server.ServerService_CopyFileServer) error {
	// 1. 参数校验
	if req.SourcePath == "" {
		err := errors.New("源文件路径不能为空")
		logger.Error("CopyFile请求参数错误", zap.Error(err))
		return err
	}
	// 检查路径是否存在
	fileInfo, err := os.Stat(req.SourcePath)
	if err != nil {
		err = fmt.Errorf("获取文件信息失败: %w", err)
		logger.Error("文件传输失败", zap.Error(err))
		return err
	}

	// 如果是目录，返回错误（目录应使用ListDirectory服务）
	if fileInfo.IsDir() {
		err = fmt.Errorf("源路径是目录，请使用ListDirectory服务: %s", req.SourcePath)
		logger.Error("文件传输失败", zap.Error(err))
		return err
	}
	logger.Info("开始处理文件传输请求",
		zap.String("source_path", req.SourcePath))

	file, err := os.Open(req.SourcePath)
	if err != nil {
		err = fmt.Errorf("打开源文件失败: %w", err)
		logger.Error("文件传输失败", zap.Error(err))
		return err
	}
	defer file.Close() // 确保文件最终关闭
	// 4. 分块读取文件并发送
	reader := bufio.NewReader(file)
	buffer := make([]byte, chunkSize)
	var totalSent int64

	for {
		// 读取一块数据（最后一块可能小于chunkSize）
		n, err := reader.Read(buffer)
		if err != nil {
			// 正常结束（读取到文件末尾）
			if errors.Is(err, os.ErrClosed) || errors.Is(err, io.EOF) {
				break
			}
			// 异常错误
			err = fmt.Errorf("读取文件内容失败: %w", err)
			logger.Error("文件传输中断", zap.Error(err))
			return err
		}

		// 发送文件块
		if err := stream.Send(&server.FileChunk{
			Data: buffer[:n], // 只发送实际读取的字节数
		}); err != nil {
			err = fmt.Errorf("发送文件块失败: %w", err)
			logger.Error("文件块发送失败", zap.Error(err))
			return err
		}

		totalSent += int64(n)

		// 打印进度日志（每发送1MB打印一次）
		if totalSent > 0 && totalSent%(1024*1024) == 0 {
			logger.Info("文件传输中",
				zap.Int64("sent", totalSent),
				zap.String("total", fmt.Sprintf("%d bytes", fileInfo.Size())))
		}
	}

	logger.Info("文件传输完成",
		zap.String("source_path", req.SourcePath),
		zap.Int64("total_sent", totalSent))

	return nil
}

// ListDirectory 实现目录列表服务（返回相对路径）
func (m *CopyServer) ListDirectory(ctx context.Context, req *server.ListDirectoryRequest) (*server.ListDirectoryResponse, error) {
	// 1. 参数校验
	if req.Path == "" {
		err := errors.New("目录路径不能为空")
		logger.Error("ListDirectory请求参数错误", zap.Error(err))
		return nil, err
	}

	logger.Info("开始处理目录列表请求",
		zap.String("path", req.Path),
		zap.Bool("recursive", req.Recursive))

	// 2. 检查路径是否存在且是目录
	rootInfo, err := os.Stat(req.Path)
	if err != nil {
		err = fmt.Errorf("获取路径信息失败: %w", err)
		logger.Error("目录列表失败", zap.String("path", req.Path), zap.Error(err))
		return nil, err
	}
	if !rootInfo.IsDir() {
		err = fmt.Errorf("路径不是目录: %s", req.Path)
		logger.Error("目录列表失败", zap.String("path", req.Path), zap.Error(err))
		return nil, err
	}

	// 3. 遍历目录并返回相对路径
	files, err := listFilesRelative(req.Path, req.Recursive)
	if err != nil {
		logger.Error("遍历目录失败", zap.String("path", req.Path), zap.Error(err))
		return nil, err
	}

	// 4. 转换为protobuf格式
	pbFiles := make([]*server.FileInfo, 0, len(files))
	for _, relPath := range files {
		// 获取文件信息
		fullPath := filepath.Join(req.Path, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			logger.Warn("获取文件信息失败", zap.String("path", fullPath), zap.Error(err))
			continue
		}

		pbFiles = append(pbFiles, &server.FileInfo{
			Path:  relPath,
			IsDir: info.IsDir(),
		})
	}

	logger.Info("目录列表完成",
		zap.String("path", req.Path),
		zap.Int("file_count", len(pbFiles)))

	return &server.ListDirectoryResponse{Files: pbFiles}, nil
}

// listFilesRelative 返回目录中所有文件的相对路径
func listFilesRelative(root string, recursive bool) ([]string, error) {
	var files []string

	// 确保root是绝对路径
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// 遍历目录
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		// 跳过根目录本身
		if relPath == "." {
			return nil
		}

		// 添加到结果列表
		files = append(files, relPath)

		// 如果不递归且是目录，跳过子目录
		if !recursive && info.IsDir() {
			return filepath.SkipDir
		}

		return nil
	})

	return files, err
}
