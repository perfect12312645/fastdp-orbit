package views

import (
	"fmt"
	"net/http"

	"fastdp-orbit/backend/config"

	"github.com/gin-gonic/gin"
)

// ServerConfig Server配置（用于生成安装命令）
var ServerConfig *config.ServerConfig

// GetInstallCommand 获取Agent安装命令
func GetInstallCommand(c *gin.Context) {
	if ServerConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "配置未初始化"})
		return
	}

	// 只取IP部分（去掉端口）
	ip := ServerConfig.OrbitServer.Address
	port := fmt.Sprintf("%d", ServerConfig.OrbitServer.Port)
	token := ServerConfig.GRPC.Token
	scheme := "https"
	if !ServerConfig.OrbitServer.TLS.Enabled {
		scheme = "http"
	}
	cmd := fmt.Sprintf("curl -kfSL %s://%s:%s/static/install-agent.sh | bash -x -s %s %s %s",
		scheme, ip, port, ip, port, token)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"command": cmd,
			"server":  ip + ":" + port,
			"token":   token,
		},
	})
}
