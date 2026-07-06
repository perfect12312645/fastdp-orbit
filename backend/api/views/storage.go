package views

import (
	"fmt"
	"net/http"
	"strconv"

	"fastdp-orbit/backend/models/storage"
	storagesvc "fastdp-orbit/backend/services/storage"

	"github.com/gin-gonic/gin"
)

// StorageService 存储服务实例（在 router.go 中初始化）
var StorageService *storagesvc.Service

// ==================== 请求结构 ====================

// ListFilesRequest 文件列表请求
type ListFilesRequest struct {
	Keyword string `form:"keyword"`
}

// ==================== Handlers ====================

// UploadChunk 处理分块上传
func UploadChunk(c *gin.Context) {
	if StorageService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "存储服务未初始化"})
		return
	}

	// 解析表单参数
	fileName := c.PostForm("file_name")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "文件名不能为空"})
		return
	}

	chunkIndex, err := strconv.Atoi(c.PostForm("chunk_index"))
	if err != nil || chunkIndex < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "分块索引无效"})
		return
	}

	totalChunks, err := strconv.Atoi(c.PostForm("total_chunks"))
	if err != nil || totalChunks <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "总分块数无效"})
		return
	}

	// 获取上传文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "获取上传文件失败: " + err.Error()})
		return
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "打开文件失败"})
		return
	}
	defer src.Close()

	// 调用服务
	status, err := StorageService.UploadChunk(fileName, chunkIndex, totalChunks, src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	// 判断是否上传完成
	if status.ReceivedChunks == totalChunks {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "文件上传完成",
			"data": gin.H{
				"filename": fileName,
				"status":   "completed",
			},
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": fmt.Sprintf("分块 %d/%d 上传成功", status.ReceivedChunks, totalChunks),
			"data": gin.H{
				"next_chunk": status.ReceivedChunks,
				"status":     "uploading",
			},
		})
	}
}

// GetResumeInfo 获取续传信息
func GetResumeInfo(c *gin.Context) {
	if StorageService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "存储服务未初始化"})
		return
	}

	fileName := c.Query("file_name")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "文件名不能为空"})
		return
	}

	info, err := StorageService.GetResumeInfo(fileName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": info})
}

// ListStorageFiles 列出存储文件
func ListStorageFiles(c *gin.Context) {
	if StorageService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "存储服务未初始化"})
		return
	}

	keyword := c.Query("keyword")
	files, err := StorageService.ListFiles(keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})
		return
	}

	if files == nil {
		files = []storage.StorageFile{}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": files})
}

// GetStorageFile 获取文件详情
func GetStorageFile(c *gin.Context) {
	if StorageService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "存储服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	file, err := StorageService.GetFile(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "文件不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": file})
}

// DeleteStorageFile 删除文件
func DeleteStorageFile(c *gin.Context) {
	if StorageService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "存储服务未初始化"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "ID格式错误"})
		return
	}

	if err := StorageService.DeleteFile(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DownloadFile 文件下载（不在 /api/v1 下，独立路由）
func DownloadFile(c *gin.Context) {
	if StorageService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "存储服务未初始化"})
		return
	}

	// 获取文件路径（通配符匹配）
	relPath := c.Param("path")
	if relPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未指定文件路径"})
		return
	}

	// 去掉开头的 /
	relPath = trimLeadingSlash(relPath)

	// 获取文件完整路径
	fullPath, err := StorageService.GetFilePath(relPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 输出文件（c.File 内部会设置 Content-Length、Content-Type，支持 Range 请求）
	c.File(fullPath)
}

// trimLeadingSlash 去掉路径开头的 /
func trimLeadingSlash(path string) string {
	if len(path) > 0 && path[0] == '/' {
		return path[1:]
	}
	return path
}
