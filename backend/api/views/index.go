package views

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func Index(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "FastDP Orbit",
		"version": "1.0.0",
		"status":  "running",
	})
}
