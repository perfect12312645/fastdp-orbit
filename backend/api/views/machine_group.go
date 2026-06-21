package views

import (
	"net/http"

	"fastdp-orbit/backend/models/machine"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MachineGroupDB 机器分组数据库（由 main 初始化注入）
var MachineGroupDB *gorm.DB

// CreateMachineGroupInput 创建机器分组请求
type CreateMachineGroupInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	MachineIDs  []uint `json:"machine_ids"` // 关联的机器ID列表
}

// ListMachineGroups 获取所有机器分组
func ListMachineGroups(c *gin.Context) {
	if MachineGroupDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "数据库未初始化"})
		return
	}

	var groups []machine.MachineGroup
	if err := MachineGroupDB.Preload("Machines").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	if groups == nil {
		groups = []machine.MachineGroup{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": groups})
}

// CreateMachineGroup 创建机器分组
func CreateMachineGroup(c *gin.Context) {
	if MachineGroupDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "数据库未初始化"})
		return
	}

	var req CreateMachineGroupInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	// 校验名称唯一性
	var count int64
	MachineGroupDB.Model(&machine.MachineGroup{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "分组名称已存在"})
		return
	}

	group := machine.MachineGroup{
		Name:        req.Name,
		Description: req.Description,
	}

	// 关联机器
	if len(req.MachineIDs) > 0 {
		var machines []machine.Machine
		if err := MachineGroupDB.Where("id IN ?", req.MachineIDs).Find(&machines).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "查询机器失败: " + err.Error()})
			return
		}
		group.Machines = machines
	}

	if err := MachineGroupDB.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": group})
}

// DeleteMachineGroup 删除机器分组
func DeleteMachineGroup(c *gin.Context) {
	if MachineGroupDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "数据库未初始化"})
		return
	}

	id := c.Param("id")
	var group machine.MachineGroup
	if err := MachineGroupDB.First(&group, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "分组不存在"})
		return
	}

	// 解除关联（多对多中间表）
	MachineGroupDB.Model(&group).Association("Machines").Clear()

	if err := MachineGroupDB.Unscoped().Delete(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// GetMachineGroup 获取机器分组详情
func GetMachineGroup(c *gin.Context) {
	if MachineGroupDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "数据库未初始化"})
		return
	}

	id := c.Param("id")
	var group machine.MachineGroup
	if err := MachineGroupDB.Preload("Machines").First(&group, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "分组不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": group})
}

// UpdateMachineGroup 更新机器分组
func UpdateMachineGroup(c *gin.Context) {
	if MachineGroupDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "数据库未初始化"})
		return
	}

	id := c.Param("id")
	var group machine.MachineGroup
	if err := MachineGroupDB.First(&group, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "分组不存在"})
		return
	}

	var req CreateMachineGroupInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	// 校验名称唯一性（排除自身）
	var count int64
	MachineGroupDB.Model(&machine.MachineGroup{}).Where("name = ? AND id != ?", req.Name, group.ID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "分组名称已存在"})
		return
	}

	group.Name = req.Name
	group.Description = req.Description

	// 更新关联机器
	if req.MachineIDs != nil {
		var machines []machine.Machine
		if err := MachineGroupDB.Where("id IN ?", req.MachineIDs).Find(&machines).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "查询机器失败: " + err.Error()})
			return
		}
		MachineGroupDB.Model(&group).Association("Machines").Replace(machines)
	}

	if err := MachineGroupDB.Save(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "更新失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": group})
}
