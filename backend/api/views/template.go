package views

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func ListTemplates(c *gin.Context) {
	// TODO: Implement list templates
	c.JSON(http.StatusOK, gin.H{"templates": []interface{}{}})
}

func CreateTemplate(c *gin.Context) {
	// TODO: Implement create template
	c.JSON(http.StatusCreated, gin.H{"message": "template created"})
}

func GetTemplate(c *gin.Context) {
	// TODO: Implement get template
	c.JSON(http.StatusOK, gin.H{"template": nil})
}

func UpdateTemplate(c *gin.Context) {
	// TODO: Implement update template
	c.JSON(http.StatusOK, gin.H{"message": "template updated"})
}

func DeleteTemplate(c *gin.Context) {
	// TODO: Implement delete template
	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}
