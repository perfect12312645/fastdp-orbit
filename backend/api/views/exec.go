package views

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type ExecRequest struct {
	MachineID uint   `json:"machine_id" binding:"required"`
	Command   string `json:"command" binding:"required"`
}

func Exec(c *gin.Context) {
	var req ExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement command execution
	c.JSON(http.StatusOK, gin.H{
		"output": "Command executed successfully",
		"status": 0,
	})
}
