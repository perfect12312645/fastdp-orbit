package storage

import (
	"time"

	"gorm.io/gorm"
)

// StorageFile 存储文件元数据
type StorageFile struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name" gorm:"size:255;not null;index"`     // 原始文件名
	Path         string         `json:"path" gorm:"size:500;not null;unique"`    // 相对存储路径（唯一）
	Size         int64          `json:"size" gorm:"not null"`                    // 文件大小（字节）
	MD5          string         `json:"md5" gorm:"size:32"`                      // 文件MD5（上传完成后异步计算）
	MimeType     string         `json:"mime_type" gorm:"size:100"`               // MIME类型
	Source      string         `json:"source" gorm:"column:package_group;size:100;index;default:''"` // 来源（模板包名）
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (StorageFile) TableName() string { return "storage_files" }
