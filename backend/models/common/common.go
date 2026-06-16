package common

import (
	"time"

	"gorm.io/gorm"
)

// Template represents a deployment template
type Template struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`
	Description string         `json:"description" gorm:"size:500"`
	Category    string         `json:"category" gorm:"size:50"`
	Content     string         `json:"content" gorm:"type:text"`
	Variables   string         `json:"variables" gorm:"type:text"`
	IsPublic    bool           `json:"is_public" gorm:"default:true"`
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Cluster represents a Kubernetes cluster
type Cluster struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`
	Description string         `json:"description" gorm:"size:500"`
	Version     string         `json:"version" gorm:"size:20"`
	Status      string         `json:"status" gorm:"size:20;default:creating"`
	Config      string         `json:"config" gorm:"type:text"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// AuditLog represents system audit logs
type AuditLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"index"`
	Action     string    `json:"action" gorm:"size:50"`
	Resource   string    `json:"resource" gorm:"size:100"`
	ResourceID uint      `json:"resource_id"`
	Details    string    `json:"details" gorm:"type:text"`
	IP         string    `json:"ip" gorm:"size:45"`
	CreatedAt  time.Time `json:"created_at"`
}
