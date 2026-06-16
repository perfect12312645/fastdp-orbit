package machine

import (
	"time"

	"gorm.io/gorm"
)

// Machine 机器主表（Agent注册时写入静态信息）
type Machine struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	IP              string         `json:"ip" gorm:"size:45;uniqueIndex;not null"`
	Port            int            `json:"port" gorm:"not null"`
	Status          string         `json:"status" gorm:"size:20;default:online"`
	Hostname        string         `json:"hostname" gorm:"size:100"`
	Virtualization  string         `json:"virtualization" gorm:"size:50"`
	OSName          string         `json:"os_name" gorm:"size:100"`
	OSVersion       string         `json:"os_version" gorm:"size:100"`
	Kernel          string         `json:"kernel" gorm:"size:100"`
	Arch            string         `json:"arch" gorm:"size:20"`
	CPUModel        string         `json:"cpu_model" gorm:"size:200"`
	CPUCores        int32          `json:"cpu_cores"`
	MemoryKB        int64          `json:"memory_kb"`
	SwapKB          int64          `json:"swap_kb"`
	Gateway         string         `json:"gateway" gorm:"size:45"`
	FirewallStatus  string         `json:"firewall_status" gorm:"size:20"`
	FirewallEnabled string         `json:"firewall_enabled" gorm:"size:20"`
	Timezone        string         `json:"timezone" gorm:"size:50"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`

	Disks    []MachineDisk    `json:"disks" gorm:"foreignKey:MachineID"`
	Networks []MachineNetwork `json:"networks" gorm:"foreignKey:MachineID"`
	GPUs     []MachineGPU     `json:"gpus" gorm:"foreignKey:MachineID"`
}

// MachineDisk 磁盘信息
type MachineDisk struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	MachineID uint   `json:"machine_id" gorm:"index"`
	Device    string `json:"device" gorm:"size:50"`
	Type      string `json:"type" gorm:"size:20"`
	TotalGB   int64  `json:"total_gb"`
}

// MachineNetwork 网卡信息
type MachineNetwork struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	MachineID uint   `json:"machine_id" gorm:"index"`
	Name      string `json:"name" gorm:"size:50"`
	IP        string `json:"ip" gorm:"size:45"`
	MAC       string `json:"mac" gorm:"size:20"`
	Speed     int64  `json:"speed"`
	Status    string `json:"status" gorm:"size:10"`
}

// MachineGPU GPU信息
type MachineGPU struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	MachineID     uint   `json:"machine_id" gorm:"index"`
	Name          string `json:"name" gorm:"size:200"`
	Count         int32  `json:"count"`
	DriverVersion string `json:"driver_version" gorm:"size:50"`
}
