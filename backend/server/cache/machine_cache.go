package cache

import (
	"errors"
	"fastdp-orbit/backend/models/machine"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/proto/agent"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MachineSnapshot 内存中的机器快照
type MachineSnapshot struct {
	// 静态信息（注册时写入DB，内存也保留一份用于返回）
	ID              uint
	IP              string
	Port            int
	Hostname        string
	Virtualization  string
	OSName          string
	OSVersion       string
	Kernel          string
	Arch            string
	CPUModel        string
	CPUCores        int32
	MemoryKB        int64
	SwapKB          int64
	Gateway         string
	FirewallStatus  string
	FirewallEnabled string
	Timezone        string
	Disks           []machine.MachineDisk
	Networks        []machine.MachineNetwork
	GPUs            []machine.MachineGPU

	// 动态信息（仅内存）
	LastSeenAt    time.Time
	UptimeSeconds int64
	SystemTime    string
	HardwareTime  string

	// 状态
	Status        string
	StatusChanged bool
}

// MachineCache 机器内存缓存
type MachineCache struct {
	mu       sync.RWMutex
	machines map[string]*MachineSnapshot // key: "ip:port"
	db       *gorm.DB
}

// NewMachineCache 创建机器缓存
func NewMachineCache(db *gorm.DB) *MachineCache {
	return &MachineCache{
		machines: make(map[string]*MachineSnapshot),
		db:       db,
	}
}

func machineKey(ip string, port int) string {
	return fmt.Sprintf("%s:%d", ip, port)
}

// LoadFromDB 从数据库加载已注册机器到缓存（Server重启后恢复）
func (c *MachineCache) LoadFromDB() error {
	var machines []machine.Machine
	if err := c.db.Preload("Disks").Preload("Networks").Preload("GPUs").Find(&machines).Error; err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, m := range machines {
		c.machines[machineKey(m.IP, m.Port)] = &MachineSnapshot{
			ID:              m.ID,
			IP:              m.IP,
			Port:            m.Port,
			Hostname:        m.Hostname,
			Virtualization:  m.Virtualization,
			OSName:          m.OSName,
			OSVersion:       m.OSVersion,
			Kernel:          m.Kernel,
			Arch:            m.Arch,
			CPUModel:        m.CPUModel,
			CPUCores:        m.CPUCores,
			MemoryKB:        m.MemoryKB,
			SwapKB:          m.SwapKB,
			Gateway:         m.Gateway,
			FirewallStatus:  m.FirewallStatus,
			FirewallEnabled: m.FirewallEnabled,
			Timezone:        m.Timezone,
			Disks:           m.Disks,
			Networks:        m.Networks,
			GPUs:            m.GPUs,
			Status:          "offline",
			StatusChanged:   false,
		}
	}

	if len(machines) > 0 {
		logger.Info("从数据库加载机器到缓存", zap.Int("count", len(machines)))
	}
	return nil
}

// AgentStatus 机器状态信息（供API返回）
type AgentStatus struct {
	IP         string
	Port       int
	Status     string
	LastSeenAt time.Time
}

// ListAgents 获取所有已注册机器的标识列表（ip:port格式）
func (c *MachineCache) ListAgents() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]string, 0, len(c.machines))
	for key := range c.machines {
		result = append(result, key)
	}
	return result
}

// GetSnapshot 获取机器快照（供API返回缓存中的信息）
func (c *MachineCache) GetSnapshot(ip string, port int) (*MachineSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snap, ok := c.machines[machineKey(ip, port)]
	if !ok {
		return nil, false
	}
	// 返回副本
	return new(*snap), true
}

// HasMachine 检查机器是否在缓存中
func (c *MachineCache) HasMachine(ip string, port int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.machines[machineKey(ip, port)]
	return ok
}

// SetOffline 标记机器离线（供API调用）
func (c *MachineCache) SetOffline(ip string, port int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	snap, ok := c.machines[machineKey(ip, port)]
	if !ok {
		return
	}

	if snap.Status != "offline" {
		snap.Status = "offline"
		snap.StatusChanged = true
		logger.Warn("API检测：机器离线", zap.String("ip", ip), zap.Int("port", port))
	}
}

// Remove 从缓存中移除机器
func (c *MachineCache) Remove(ip string, port int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := machineKey(ip, port)
	delete(c.machines, key)
	logger.Info("机器已从缓存移除", zap.String("ip", ip), zap.Int("port", port))
}

// DeleteFromDB 从数据库中删除机器记录及关联数据
func (c *MachineCache) DeleteFromDB(ip string, port int) {
	var m machine.Machine
	if err := c.db.Unscoped().Where("ip = ? AND port = ?", ip, port).First(&m).Error; err != nil {
		return
	}

	// 删除关联数据
	c.db.Where("machine_id = ?", m.ID).Delete(&machine.MachineDisk{})
	c.db.Where("machine_id = ?", m.ID).Delete(&machine.MachineNetwork{})
	c.db.Where("machine_id = ?", m.ID).Delete(&machine.MachineGPU{})

	// 硬删除机器记录（不能用软删除，否则IP的uniqueIndex会冲突导致重新注册失败）
	c.db.Unscoped().Delete(&m)
	logger.Info("机器已从数据库删除", zap.String("ip", ip), zap.Int("port", port))
}

// UpdateStaticInfo 通过gRPC获取的最新信息更新缓存中的静态字段
func (c *MachineCache) UpdateStaticInfo(ip string, port int, env *agent.SystemEnvInfo) {
	if env == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := machineKey(ip, port)
	snap, ok := c.machines[key]
	if !ok {
		return
	}

	snap.Hostname = env.Hostname
	snap.Virtualization = env.Virtualization
	snap.OSName = env.Os.GetName()
	snap.OSVersion = env.Os.GetVersion()
	snap.Kernel = env.Os.GetKernel()
	snap.Arch = env.Os.GetArch()
	snap.CPUModel = env.Cpu.GetModel()
	snap.CPUCores = env.Cpu.GetCores()
	snap.MemoryKB = env.Memory.GetTotalKb()
	snap.SwapKB = env.Swap.GetTotalKb()
	snap.Gateway = env.Gateway
	snap.Timezone = env.Timezone

	if env.Firewall != nil {
		snap.FirewallStatus = env.Firewall.Status
		snap.FirewallEnabled = env.Firewall.Enabled
	}

	// 更新磁盘
	snap.Disks = make([]machine.MachineDisk, 0, len(env.Disks))
	for _, d := range env.Disks {
		snap.Disks = append(snap.Disks, machine.MachineDisk{
			Device:  d.Device,
			Type:    d.Type,
			TotalGB: d.TotalGb,
		})
	}

	// 更新网卡
	snap.Networks = make([]machine.MachineNetwork, 0, len(env.Networks))
	for _, n := range env.Networks {
		snap.Networks = append(snap.Networks, machine.MachineNetwork{
			Name:   n.Name,
			IP:     n.Ip,
			MAC:    n.Mac,
			Speed:  n.Speed,
			Status: n.Status,
		})
	}

	// 更新GPU
	snap.GPUs = make([]machine.MachineGPU, 0, len(env.Gpus))
	for _, g := range env.Gpus {
		snap.GPUs = append(snap.GPUs, machine.MachineGPU{
			Name:          g.Name,
			Count:         g.Count,
			DriverVersion: g.DriverVersion,
		})
	}

	// 同步到DB
	c.syncStaticToDB(snap)

	logger.Info("缓存静态信息已更新", zap.String("ip", ip), zap.Int("port", port))
}

// syncStaticToDB 将缓存中的静态信息同步到数据库
func (c *MachineCache) syncStaticToDB(snap *MachineSnapshot) {
	var existing machine.Machine
	if err := c.db.Where("ip = ? AND port = ?", snap.IP, snap.Port).First(&existing).Error; err != nil {
		return
	}

	existing.Hostname = snap.Hostname
	existing.Virtualization = snap.Virtualization
	existing.OSName = snap.OSName
	existing.OSVersion = snap.OSVersion
	existing.Kernel = snap.Kernel
	existing.Arch = snap.Arch
	existing.CPUModel = snap.CPUModel
	existing.CPUCores = snap.CPUCores
	existing.MemoryKB = snap.MemoryKB
	existing.SwapKB = snap.SwapKB
	existing.Gateway = snap.Gateway
	existing.FirewallStatus = snap.FirewallStatus
	existing.FirewallEnabled = snap.FirewallEnabled
	existing.Timezone = snap.Timezone

	c.db.Save(&existing)
	c.db.Model(&existing).Association("Disks").Replace(snap.Disks)
	c.db.Model(&existing).Association("Networks").Replace(snap.Networks)
	c.db.Model(&existing).Association("GPUs").Replace(snap.GPUs)
}

// Register 注册时写入缓存和数据库
func (c *MachineCache) Register(snap *MachineSnapshot) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := machineKey(snap.IP, snap.Port)

	// 写入数据库
	dbMachine := &machine.Machine{
		IP:              snap.IP,
		Port:            snap.Port,
		Status:          "online",
		Hostname:        snap.Hostname,
		Virtualization:  snap.Virtualization,
		OSName:          snap.OSName,
		OSVersion:       snap.OSVersion,
		Kernel:          snap.Kernel,
		Arch:            snap.Arch,
		CPUModel:        snap.CPUModel,
		CPUCores:        snap.CPUCores,
		MemoryKB:        snap.MemoryKB,
		SwapKB:          snap.SwapKB,
		Gateway:         snap.Gateway,
		FirewallStatus:  snap.FirewallStatus,
		FirewallEnabled: snap.FirewallEnabled,
		Timezone:        snap.Timezone,
	}

	// 先检查是否已存在（静默查询，record not found是正常流程）
	var existing machine.Machine
	err := c.db.Session(&gorm.Session{Logger: nil}).
		Where("ip = ? AND port = ?", snap.IP, snap.Port).First(&existing).Error
	if err == nil {
		// 已存在，更新
		dbMachine.ID = existing.ID
		if err := c.db.Save(dbMachine).Error; err != nil {
			logger.Error("更新机器信息失败", zap.Error(err), zap.String("ip:port", key))
			return err
		}
		// 更新关联
		if len(snap.Disks) > 0 {
			c.db.Model(&existing).Association("Disks").Replace(snap.Disks)
		}
		if len(snap.Networks) > 0 {
			c.db.Model(&existing).Association("Networks").Replace(snap.Networks)
		}
		if len(snap.GPUs) > 0 {
			c.db.Model(&existing).Association("GPUs").Replace(snap.GPUs)
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// 新增
		if err := c.db.Create(dbMachine).Error; err != nil {
			logger.Error("写入机器信息失败", zap.Error(err), zap.String("ip:port", key))
			return err
		}
		// 写入关联
		for i := range snap.Disks {
			snap.Disks[i].MachineID = dbMachine.ID
		}
		for i := range snap.Networks {
			snap.Networks[i].MachineID = dbMachine.ID
		}
		for i := range snap.GPUs {
			snap.GPUs[i].MachineID = dbMachine.ID
		}
		if len(snap.Disks) > 0 {
			c.db.Create(&snap.Disks)
		}
		if len(snap.Networks) > 0 {
			c.db.Create(&snap.Networks)
		}
		if len(snap.GPUs) > 0 {
			c.db.Create(&snap.GPUs)
		}
	} else {
		logger.Error("查询机器信息失败", zap.Error(err), zap.String("ip:port", key))
		return err
	}

	// 写入缓存
	snap.ID = dbMachine.ID
	snap.LastSeenAt = time.Now()
	snap.Status = "online"
	snap.StatusChanged = false
	c.machines[key] = snap

	logger.Info("机器注册成功", zap.String("ip", snap.IP), zap.Int("port", snap.Port))
	return nil
}

// Heartbeat 更新心跳信息（含动态信息）
func (c *MachineCache) Heartbeat(ip string, port int, uptimeSeconds int64, systemTime, hardwareTime string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := machineKey(ip, port)
	snap, ok := c.machines[key]
	if !ok {
		logger.Warn("心跳收到未注册的机器", zap.String("ip:port", key))
		return
	}

	snap.LastSeenAt = time.Now()
	snap.UptimeSeconds = uptimeSeconds
	snap.SystemTime = systemTime
	snap.HardwareTime = hardwareTime

	// 如果之前是offline，标记状态变化
	if snap.Status == "offline" {
		snap.Status = "online"
		snap.StatusChanged = true
	}
}

// CheckOffline 检测离线机器
func (c *MachineCache) CheckOffline(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, snap := range c.machines {
		if snap.Status == "online" && time.Since(snap.LastSeenAt) > timeout {
			snap.Status = "offline"
			snap.StatusChanged = true
			logger.Warn("机器离线", zap.String("ip", snap.IP), zap.Duration("lastSeen", time.Since(snap.LastSeenAt)))
		}
	}
}

// SyncStatus 将状态变化同步到数据库
func (c *MachineCache) SyncStatus() {
	c.mu.Lock()
	changed := make([]*MachineSnapshot, 0)
	for _, snap := range c.machines {
		if snap.StatusChanged {
			changed = append(changed, snap)
			snap.StatusChanged = false
		}
	}
	c.mu.Unlock()

	// 异步写DB
	for _, snap := range changed {
		if err := c.db.Model(&machine.Machine{}).
			Where("ip = ? AND port = ?", snap.IP, snap.Port).
			Update("status", snap.Status).Error; err != nil {
			logger.Error("同步机器状态失败", zap.Error(err), zap.String("ip", snap.IP))
		}
	}
}

// StartOfflineChecker 启动离线检测定时器
func (c *MachineCache) StartOfflineChecker(offlineTimeout time.Duration) {
	checkInterval := offlineTimeout / 2
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()
		for range ticker.C {
			c.CheckOffline(offlineTimeout)
			c.SyncStatus()
		}
	}()
}
