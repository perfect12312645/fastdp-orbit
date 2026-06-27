package storage

import (
	"crypto/md5"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fastdp-orbit/backend/models/storage"

	"gorm.io/gorm"
)

// UploadStatus 分块上传状态
type UploadStatus struct {
	TotalChunks    int       // 总分块数
	ReceivedChunks int       // 已接收分块数
	FilePath       string    // 文件存储路径
	FileName       string    // 原始文件名
	LastUpdated    time.Time // 最后更新时间
}

// ResumeInfo 断点续传信息
type ResumeInfo struct {
	FileName       string `json:"file_name"`
	FileExists     bool   `json:"file_exists"`
	UploadedBytes  int64  `json:"uploaded_bytes"`
	UploadedChunks int    `json:"uploaded_chunks"`
	TotalChunks    int    `json:"total_chunks"`
}

// 全局上传状态缓存
var uploadStatusCache sync.Map

// Service 存储服务
type Service struct {
	db         *gorm.DB
	storageDir string // 文件存储根目录
	chunkSize  int64  // 分块大小（字节）
}

// NewService 创建存储服务
func NewService(db *gorm.DB, storageDir string) *Service {
	if storageDir == "" {
		storageDir = "./storage"
	}
	// 确保存储目录存在
	os.MkdirAll(storageDir, 0755)
	return &Service{
		db:         db,
		storageDir: storageDir,
		chunkSize:  5 * 1024 * 1024, // 默认 5MB
	}
}

// GetStorageDir 获取存储目录
func (s *Service) GetStorageDir() string {
	return s.storageDir
}

// GetChunkSize 获取分块大小
func (s *Service) GetChunkSize() int64 {
	return s.chunkSize
}

// UploadChunk 处理分块上传
func (s *Service) UploadChunk(fileName string, chunkIndex int, totalChunks int, chunkData io.Reader) (*UploadStatus, error) {
	// 1. 校验参数
	if fileName == "" {
		return nil, fmt.Errorf("文件名不能为空")
	}
	if chunkIndex < 0 || totalChunks <= 0 {
		return nil, fmt.Errorf("分块参数无效")
	}

	// 2. 拼接文件路径（直接存在 storage 根目录下）
	filePath := filepath.Join(s.storageDir, fileName)

	// 3. 获取或创建上传状态
	var status *UploadStatus
	if chunkIndex == 0 {
		// 新上传：初始化状态
		status = &UploadStatus{
			TotalChunks:    totalChunks,
			ReceivedChunks: 0,
			FilePath:       filePath,
			FileName:       fileName,
			LastUpdated:    time.Now(),
		}
		uploadStatusCache.Store(fileName, status)
	} else {
		// 续传：从缓存获取状态
		statusInterface, ok := uploadStatusCache.Load(fileName)
		if !ok {
			// 缓存丢失，尝试从磁盘恢复
			if info, err := os.Stat(filePath); err == nil {
				uploadedChunks := int((info.Size() + s.chunkSize - 1) / s.chunkSize)
				status = &UploadStatus{
					TotalChunks:    totalChunks,
					ReceivedChunks: uploadedChunks,
					FilePath:       filePath,
					FileName:       fileName,
					LastUpdated:    time.Now(),
				}
				uploadStatusCache.Store(fileName, status)
			} else {
				return nil, fmt.Errorf("上传状态不存在，无法续传")
			}
		} else {
			status = statusInterface.(*UploadStatus)
		}
	}

	// 4. 校验分块索引
	if chunkIndex != status.ReceivedChunks {
		return nil, fmt.Errorf("分块索引错误，期望%d，收到%d", status.ReceivedChunks, chunkIndex)
	}

	// 5. 写入分块数据
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer f.Close()

	_, err = io.Copy(f, chunkData)
	if err != nil {
		return nil, fmt.Errorf("写入分块数据失败: %v", err)
	}

	// 6. 更新状态
	status.ReceivedChunks++
	status.LastUpdated = time.Now()

	// 7. 检查是否上传完成
	if status.ReceivedChunks == totalChunks {
		// 上传完成：清理缓存
		uploadStatusCache.Delete(fileName)

		// 计算文件大小
		fileInfo, _ := os.Stat(filePath)
		fileSize := int64(0)
		if fileInfo != nil {
			fileSize = fileInfo.Size()
		}

		// 检测 MIME 类型
		mimeType := mime.TypeByExtension(filepath.Ext(fileName))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// 保存文件元数据到数据库
		storageFile := &storage.StorageFile{
			Name:     fileName,
			Path:     fileName,
			Size:     fileSize,
			MimeType: mimeType,
		}

		// 检查是否已存在（软删除的记录）
		var existing storage.StorageFile
		if err := s.db.Unscoped().Where("path = ?", fileName).First(&existing).Error; err == nil {
			// 已存在，更新
			existing.DeletedAt = gorm.DeletedAt{}
			existing.Size = fileSize
			existing.MimeType = mimeType
			existing.MD5 = "" // 重新计算
			s.db.Unscoped().Save(&existing)
			storageFile = &existing
		} else {
			// 不存在，创建
			s.db.Create(storageFile)
		}

		// 异步计算 MD5
		go s.calculateMD5(storageFile.ID, filePath)

		return status, nil
	}

	// 分块上传成功（未完成）
	return status, nil
}

// GetResumeInfo 获取续传信息
func (s *Service) GetResumeInfo(fileName string) (*ResumeInfo, error) {
	if fileName == "" {
		return nil, fmt.Errorf("文件名不能为空")
	}

	filePath := filepath.Join(s.storageDir, fileName)
	info := &ResumeInfo{
		FileName: fileName,
	}

	// 从缓存获取状态
	if statusInterface, ok := uploadStatusCache.Load(fileName); ok {
		status := statusInterface.(*UploadStatus)
		info.FileExists = true
		info.UploadedChunks = status.ReceivedChunks
		info.TotalChunks = status.TotalChunks
		info.UploadedBytes = int64(status.ReceivedChunks) * s.chunkSize
		return info, nil
	}

	// 从磁盘获取状态
	if fileStat, err := os.Stat(filePath); err == nil {
		info.FileExists = true
		info.UploadedBytes = fileStat.Size()
		info.UploadedChunks = int((fileStat.Size() + s.chunkSize - 1) / s.chunkSize)
	}

	return info, nil
}

// ListFiles 列出所有文件
func (s *Service) ListFiles(keyword string) ([]storage.StorageFile, error) {
	var files []storage.StorageFile
	query := s.db.Order("created_at DESC")
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if err := query.Find(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}

// GetFile 获取文件详情
func (s *Service) GetFile(id uint) (*storage.StorageFile, error) {
	var file storage.StorageFile
	if err := s.db.First(&file, id).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

// GetFileByPath 根据路径获取文件
func (s *Service) GetFileByPath(path string) (*storage.StorageFile, error) {
	var file storage.StorageFile
	if err := s.db.Where("path = ?", path).First(&file).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

// DeleteFile 删除文件
func (s *Service) DeleteFile(id uint) error {
	var file storage.StorageFile
	if err := s.db.First(&file, id).Error; err != nil {
		return fmt.Errorf("文件不存在")
	}

	// 删除磁盘文件
	filePath := filepath.Join(s.storageDir, file.Path)
	os.Remove(filePath)

	// 删除数据库记录
	return s.db.Delete(&file).Error
}

// GetFilePath 获取文件的完整磁盘路径
func (s *Service) GetFilePath(relPath string) (string, error) {
	// 清理路径，防止路径穿越
	cleanPath := filepath.Clean(relPath)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("非法路径")
	}

	fullPath := filepath.Join(s.storageDir, cleanPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件不存在")
	}

	return fullPath, nil
}

// calculateMD5 异步计算文件 MD5
func (s *Service) calculateMD5(fileID uint, filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return
	}

	md5Str := fmt.Sprintf("%x", h.Sum(nil))

	// 更新数据库
	s.db.Model(&storage.StorageFile{}).Where("id = ?", fileID).Update("md5", md5Str)
}

// FormatSize 格式化文件大小
func FormatSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unitIndex := 0
	floatSize := float64(size)

	for floatSize >= 1024 && unitIndex < len(units)-1 {
		floatSize /= 1024
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", floatSize, units[unitIndex])
}
