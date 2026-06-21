package views

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"fastdp-orbit/backend/proto/agent"
	"fastdp-orbit/backend/server/cache"
	"fastdp-orbit/backend/server/grpc"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/connectivity"
)

// MachineCache 机器缓存
var MachineCache *cache.MachineCache

// AgentConnPool Agent连接池
var AgentConnPool *grpc.AgentConnPool

// MachineInfo 返回给前端的机器信息
type MachineInfo struct {
	ID              uint          `json:"id"`
	IP              string        `json:"ip"`
	Port            int           `json:"port"`
	Status          string        `json:"status"`
	Hostname        string        `json:"hostname"`
	Virtualization  string        `json:"virtualization"`
	UptimeSeconds   int64         `json:"uptime_seconds"`
	SystemTime      string        `json:"system_time"`
	HardwareTime    string        `json:"hardware_time"`
	OSName          string        `json:"os_name"`
	OSVersion       string        `json:"os_version"`
	Kernel          string        `json:"kernel"`
	Arch            string        `json:"arch"`
	CPUModel        string        `json:"cpu_model"`
	CPUCores        int32         `json:"cpu_cores"`
	MemoryKB        int64         `json:"memory_kb"`
	SwapKB          int64         `json:"swap_kb"`
	Gateway         string        `json:"gateway"`
	FirewallStatus  string        `json:"firewall_status"`
	FirewallEnabled string        `json:"firewall_enabled"`
	Timezone        string        `json:"timezone"`
	Disks           []DiskInfo    `json:"disks"`
	Networks        []NetworkInfo `json:"networks"`
	GPUs            []GPUInfo     `json:"gpus"`
}

type DiskInfo struct {
	Device  string `json:"device"`
	Type    string `json:"type"`
	TotalGB int64  `json:"total_gb"`
}

type NetworkInfo struct {
	Name   string `json:"name"`
	IP     string `json:"ip"`
	MAC    string `json:"mac"`
	Speed  int64  `json:"speed"`
	Status string `json:"status"`
}

type GPUInfo struct {
	Name          string `json:"name"`
	Count         int32  `json:"count"`
	DriverVersion string `json:"driver_version"`
}

// ListMachines 获取所有机器信息（从缓存读取，毫秒级响应）
func ListMachines(c *gin.Context) {
	if MachineCache == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "缓存未初始化"})
		return
	}

	agents := MachineCache.ListAgents()
	if len(agents) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []MachineInfo{}})
		return
	}

	results := make([]MachineInfo, 0, len(agents))
	for _, agentKey := range agents {
		parts := strings.SplitN(agentKey, ":", 2)
		if len(parts) != 2 {
			continue
		}
		ip := parts[0]
		port, _ := strconv.Atoi(parts[1])

		info := MachineInfo{IP: ip, Port: port}
		if snap, ok := MachineCache.GetSnapshot(ip, port); ok {
			fillFromSnapshot(&info, snap)
		}
		results = append(results, info)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": results})
}

// SyncHardware 手动触发gRPC刷新所有机器的硬件信息
func SyncHardware(c *gin.Context) {
	if MachineCache == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "缓存未初始化"})
		return
	}
	if AgentConnPool == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "连接池未初始化"})
		return
	}

	agents := MachineCache.ListAgents()
	if len(agents) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []MachineInfo{}})
		return
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make([]MachineInfo, 0, len(agents))
	)

	for _, agentKey := range agents {
		parts := strings.SplitN(agentKey, ":", 2)
		if len(parts) != 2 {
			continue
		}
		ip := parts[0]
		port, _ := strconv.Atoi(parts[1])

		wg.Add(1)
		go func(ip string, port int) {
			defer wg.Done()
			info := fetchAndSyncAgentInfo(ip, port)
			mu.Lock()
			results = append(results, info)
			mu.Unlock()
		}(ip, port)
	}

	wg.Wait()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": results})
}

// fillFromSnapshot 从缓存快照填充信息
func fillFromSnapshot(info *MachineInfo, snap *cache.MachineSnapshot) {
	info.ID = snap.ID
	info.Hostname = snap.Hostname
	info.Virtualization = snap.Virtualization
	info.OSName = snap.OSName
	info.OSVersion = snap.OSVersion
	info.Kernel = snap.Kernel
	info.Arch = snap.Arch
	info.CPUModel = snap.CPUModel
	info.CPUCores = snap.CPUCores
	info.MemoryKB = snap.MemoryKB
	info.SwapKB = snap.SwapKB
	info.Gateway = snap.Gateway
	info.FirewallStatus = snap.FirewallStatus
	info.FirewallEnabled = snap.FirewallEnabled
	info.Timezone = snap.Timezone
	info.UptimeSeconds = snap.UptimeSeconds
	info.SystemTime = snap.SystemTime
	info.HardwareTime = snap.HardwareTime
	info.Status = snap.Status

	for _, d := range snap.Disks {
		info.Disks = append(info.Disks, DiskInfo{Device: d.Device, Type: d.Type, TotalGB: d.TotalGB})
	}
	for _, n := range snap.Networks {
		info.Networks = append(info.Networks, NetworkInfo{Name: n.Name, IP: n.IP, MAC: n.MAC, Speed: n.Speed, Status: n.Status})
	}
	for _, g := range snap.GPUs {
		info.GPUs = append(info.GPUs, GPUInfo{Name: g.Name, Count: g.Count, DriverVersion: g.DriverVersion})
	}
}

// fetchAndSyncAgentInfo 通过gRPC获取Agent信息并更新缓存和DB
func fetchAndSyncAgentInfo(ip string, port int) MachineInfo {
	addr := fmt.Sprintf("%s:%d", ip, port)
	info := MachineInfo{IP: ip, Port: port}

	if AgentConnPool == nil {
		info.Status = "offline"
		return info
	}

	conn, err := AgentConnPool.GetConn(addr)
	if err != nil {
		info.Status = "offline"
		MachineCache.SetOffline(ip, port)
		if snap, ok := MachineCache.GetSnapshot(ip, port); ok {
			fillFromSnapshot(&info, snap)
		}
		return info
	}

	state := conn.GetState()
	if state != connectivity.Ready && state != connectivity.Idle {
		info.Status = "offline"
		MachineCache.SetOffline(ip, port)
		if snap, ok := MachineCache.GetSnapshot(ip, port); ok {
			fillFromSnapshot(&info, snap)
		}
		return info
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := agent.NewAgentServiceClient(conn)
	resp, err := client.GetSystemInfo(ctx, &agent.SystemInfoRequest{MachineId: addr})

	if err != nil {
		AgentConnPool.RemoveConn(addr)
		MachineCache.SetOffline(ip, port)
		if snap, ok := MachineCache.GetSnapshot(ip, port); ok {
			fillFromSnapshot(&info, snap)
		}
		return info
	}

	if !resp.Success {
		info.Status = "offline"
		if snap, ok := MachineCache.GetSnapshot(ip, port); ok {
			fillFromSnapshot(&info, snap)
		}
		return info
	}

	// 成功 → 更新缓存中的静态信息，再从缓存读取填充响应
	info.Status = "online"
	MachineCache.UpdateStaticInfo(ip, port, resp.EnvInfo)
	if snap, ok := MachineCache.GetSnapshot(ip, port); ok {
		fillFromSnapshot(&info, snap)
	}

	return info
}

// DeleteMachine 删除机器记录（DB+缓存），Agent下次心跳时会自动退出
func DeleteMachine(c *gin.Context) {
	ip := c.Param("ip")
	portStr := c.Param("port")
	if ip == "" || portStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少IP或端口"})
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "端口格式错误"})
		return
	}

	if MachineCache == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "缓存未初始化"})
		return
	}

	// 检查机器是否存在
	if !MachineCache.HasMachine(ip, port) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "机器不存在或已删除"})
		return
	}

	// 从缓存移除
	MachineCache.Remove(ip, port)

	// 从DB移除
	MachineCache.DeleteFromDB(ip, port)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    "机器记录已删除，Agent 下次心跳时会自动退出；若需立即停用，请登录机器执行 systemctl stop orbit-agent",
	})
}

// ExecOnMachine 远程执行命令
func ExecOnMachine(c *gin.Context) {
	ip := c.Param("ip")
	portStr := c.Param("port")
	if ip == "" || portStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少IP或端口"})
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "端口格式错误"})
		return
	}

	var req struct {
		Command string `json:"command"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误，需要command"})
		return
	}

	// TODO: 通过gRPC调用Agent执行命令
	_ = ip
	_ = port
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
}
