package common

import (
	"time"

	"gorm.io/gorm"
)

// User 系统用户
type User struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Username    string         `json:"username" gorm:"size:50;uniqueIndex;not null"`
	Password    string         `json:"-" gorm:"size:255;not null"` // bcrypt hashed, never expose in JSON
	Nickname    string         `json:"nickname" gorm:"size:100"`
	Email       string         `json:"email" gorm:"size:200"`
	Avatar      string         `json:"avatar" gorm:"size:500"`
	Role          string         `json:"role" gorm:"size:20;default:admin"`
	MustChangePwd bool           `json:"must_change_pwd" gorm:"default:false"`
	LastLoginAt   *time.Time     `json:"last_login_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
