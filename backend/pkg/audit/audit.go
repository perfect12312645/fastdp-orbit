package audit

import (
	"fastdp-orbit/backend/database"
	"fastdp-orbit/backend/models/common"

	"github.com/gin-gonic/gin"
)

// Log 记录审计日志
// action: create / update / delete / execute / login / logout
// resource: 资源类型名（如 workflow, stage_template, machine_group 等）
func Log(c *gin.Context, action string, resource string, resourceID uint, details string) {
	userID, _ := c.Get("userID")
	uid, _ := userID.(uint)

	entry := common.AuditLog{
		UserID:     uid,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IP:         c.ClientIP(),
	}

	database.GetDB().Create(&entry)
}
