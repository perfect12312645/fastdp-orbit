package handler

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"fastdp-orbit/backend/agent/modules"
	"fastdp-orbit/backend/pkg/logger"
	"fastdp-orbit/backend/proto/agent"

	"go.uber.org/zap"
)

// Handler handles agent tasks
type Handler struct {
	agent.UnimplementedAgentServiceServer
}

// NewHandler creates a new handler
func NewHandler() *Handler {
	return &Handler{}
}

// Exec executes a single module task on the machine
func (h *Handler) Exec(ctx context.Context, req *agent.ExecRequest) (*agent.ExecResponse, error) {
	module := req.Module
	if module == "" {
		return &agent.ExecResponse{
			MachineId: req.MachineId,
			TaskId:    req.TaskId,
			Success:   false,
			Error: &agent.ErrorDetail{
				Code:    1002,
				Message: "module is required",
			},
		}, nil
	}

	logger.Info("执行模块任务",
		zap.String("task_id", req.TaskId),
		zap.String("machine_id", req.MachineId),
		zap.String("module", module),
		zap.Any("parameters", req.Parameters),
	)

	startTime := time.Now()

	// 获取模块
	mod, err := modules.GetModule(module)
	if err != nil {
		return &agent.ExecResponse{
			MachineId: req.MachineId,
			TaskId:    req.TaskId,
			Success:   false,
			Error: &agent.ErrorDetail{
				Code:    1001,
				Message: fmt.Sprintf("获取模块失败: %s", err.Error()),
			},
		}, nil
	}

	// 执行模块
	resp, err := mod.Run(req)
	if resp != nil {
		resp.DurationMs = time.Since(startTime).Milliseconds()
	}

	return resp, err
}

// BatchExec executes multiple module tasks and streams results
func (h *Handler) BatchExec(req *agent.BatchExecRequest, stream agent.AgentService_BatchExecServer) error {
	logger.Info("批量执行模块任务",
		zap.String("batch_id", req.BatchId),
		zap.Int("task_count", len(req.Tasks)),
		zap.Bool("ordered", req.Ordered),
	)

	if req.Ordered {
		// 顺序执行
		for _, task := range req.Tasks {
			resp, err := h.Exec(stream.Context(), task)
			if err != nil {
				return err
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	} else {
		// 并发执行，完成一个发送一个
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, task := range req.Tasks {
			wg.Add(1)
			go func(t *agent.ExecRequest) {
				defer wg.Done()
				resp, err := h.Exec(stream.Context(), t)
				if err != nil {
					logger.Error("批量任务执行失败", zap.Error(err), zap.String("task_id", t.TaskId))
					return
				}
				mu.Lock()
				stream.Send(resp)
				mu.Unlock()
			}(task)
		}

		wg.Wait()
	}

	return nil
}

// GetSystemInfo gets system information
func (h *Handler) GetSystemInfo(ctx context.Context, req *agent.SystemInfoRequest) (*agent.SystemInfoResponse, error) {
	envInfo := &agent.SystemEnvInfo{}

	// 获取所有信息（如果指定了info_types，则只获取指定类型）
	if len(req.InfoTypes) == 0 {
		// 获取所有信息
		envInfo.Hostname, _ = os.Hostname()
		envInfo.Os = h.getOSInfo()
		envInfo.Cpu = h.getCPUInfo()
		envInfo.Memory = h.getMemoryInfo()
		envInfo.Disks = h.getDiskInfo()
		envInfo.Networks = h.getNetworkInfo()
		envInfo.Gpus = h.getGPUInfo()
		envInfo.Gateway = h.getGateway()
		envInfo.Firewall = h.getFirewallInfo()
		envInfo.Swap = h.getSwapInfo()
		envInfo.Timezone = h.getTimezone()
		envInfo.SystemTime = h.GetSystemTime()
		envInfo.HardwareTime = h.GetHardwareTime()
		envInfo.Virtualization = h.getVirtualization()
		envInfo.UptimeSeconds = h.GetUptime()
	} else {
		// 根据info_types获取指定信息
		for _, infoType := range req.InfoTypes {
			switch infoType {
			case "os":
				if envInfo.Os == nil {
					envInfo.Os = h.getOSInfo()
				}
			case "cpu":
				if envInfo.Cpu == nil {
					envInfo.Cpu = h.getCPUInfo()
				}
			case "memory":
				if envInfo.Memory == nil {
					envInfo.Memory = h.getMemoryInfo()
				}
			case "disk":
				if len(envInfo.Disks) == 0 {
					envInfo.Disks = h.getDiskInfo()
				}
			case "network":
				if len(envInfo.Networks) == 0 {
					envInfo.Networks = h.getNetworkInfo()
				}
			case "gpu":
				if len(envInfo.Gpus) == 0 {
					envInfo.Gpus = h.getGPUInfo()
				}
			case "firewall":
				if envInfo.Firewall == nil {
					envInfo.Firewall = h.getFirewallInfo()
				}
			case "swap":
				if envInfo.Swap == nil {
					envInfo.Swap = h.getSwapInfo()
				}
			case "timezone":
				envInfo.Timezone = h.getTimezone()
			}
		}
		// 主机名总是返回
		if envInfo.Hostname == "" {
			envInfo.Hostname, _ = os.Hostname()
		}
	}

	return &agent.SystemInfoResponse{
		MachineId: req.MachineId,
		Success:   true,
		EnvInfo:   envInfo,
	}, nil
}

// ==================== 信息获取辅助函数 ====================

// getOSInfo 获取操作系统信息
func (h *Handler) getOSInfo() *agent.SystemInfo {
	info := &agent.SystemInfo{
		Name:    "",
		Version: "",
		Kernel:  "",
		Arch:    runtime.GOARCH,
	}

	// 尝试读取/etc/os-release获取更详细信息
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := strings.Trim(parts[1], "\"")
				switch key {
				case "NAME":
					info.Name = value
				case "VERSION":
					info.Version = value
				case "PRETTY_NAME":
					if info.Version == "" {
						info.Version = value
					}
				}
			}
		}
	}

	// 获取内核版本
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		info.Kernel = strings.TrimSpace(string(out))
	}

	return info
}

// getCPUInfo 获取CPU信息
func (h *Handler) getCPUInfo() *agent.CpuInfo {
	info := &agent.CpuInfo{}

	// 读取/proc/cpuinfo获取CPU信息（无locale问题）
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			// 统计processor行获取核数
			if strings.HasPrefix(line, "processor") {
				info.Cores++
			}
			// 获取CPU型号（找到第一个即可）
			if info.Model == "" && strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.Model = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return info
}

// getMemoryInfo 获取内存信息
func (h *Handler) getMemoryInfo() *agent.MemoryInfo {
	info := &agent.MemoryInfo{}

	// 读取/proc/meminfo（单位已是kB）
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				valueStr := strings.TrimSpace(parts[1])
				valueStr = strings.TrimSuffix(valueStr, " kB")
				var value int64
				fmt.Sscanf(valueStr, "%d", &value)

				if key == "MemTotal" {
					info.TotalKb = value
					break
				}
			}
		}
	}

	return info
}

// getDiskInfo 获取磁盘信息
func (h *Handler) getDiskInfo() []*agent.DiskInfo {
	disks := make([]*agent.DiskInfo, 0)

	cmd := exec.Command("lsblk", "-d", "-n", "-o", "NAME,TYPE,SIZE")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("lsblk exec err: %v\n", err)
		return disks
	}

	outputStr := strings.TrimSpace(string(out))
	if outputStr == "" {
		return disks
	}
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		devName := fields[0]
		devType := fields[1]
		sizeStr := fields[2]

		// 跳过光驱rom设备
		if devType == "rom" {
			continue
		}

		disk := &agent.DiskInfo{
			Device: devName, // 补全设备路径
			Type:   devType,
		}

		var sizeFloat float64
		var unit string
		// 校验解析是否成功
		n, err := fmt.Sscanf(sizeStr, "%f%s", &sizeFloat, &unit)
		if n != 2 || err != nil {
			disk.TotalGb = 0
			disks = append(disks, disk)
			continue
		}

		unit = strings.ToUpper(unit)
		switch unit {
		case "M", "MB":
			disk.TotalGb = int64(sizeFloat/1024 + 0.5) // 四舍五入
		case "G", "GB":
			disk.TotalGb = int64(sizeFloat + 0.5)
		case "T", "TB":
			disk.TotalGb = int64(sizeFloat*1024 + 0.5)
		case "P", "PB":
			disk.TotalGb = int64(sizeFloat*1024*1024 + 0.5)
		default:
			disk.TotalGb = 0
		}

		disks = append(disks, disk)
	}

	return disks
}

// getNetworkInfo 获取网卡信息
func (h *Handler) getNetworkInfo() []*agent.NetworkInfo {
	networks := make([]*agent.NetworkInfo, 0)

	ifaces, err := net.Interfaces()
	if err != nil {
		return networks
	}

	// 虚拟网卡前缀列表
	virtualPrefixes := []string{"docker", "flannel", "calico", "vxlan", "cni", "kube-ipvs", "veth", "bridge"}

	for _, iface := range ifaces {
		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 跳过虚拟网卡
		isVirtual := false
		for _, prefix := range virtualPrefixes {
			if strings.HasPrefix(iface.Name, prefix) {
				isVirtual = true
				break
			}
		}
		if isVirtual {
			continue
		}

		// 判断状态
		status := "down"
		if iface.Flags&net.FlagUp != 0 {
			status = "up"
		}

		info := &agent.NetworkInfo{
			Name:   iface.Name,
			Mac:    iface.HardwareAddr.String(),
			Status: status,
			Speed:  getInterfaceSpeed(iface.Name),
		}

		// UP的网卡才获取IP
		if status == "up" {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
					info.Ip = ipNet.String()
					break
				}
			}
		}

		networks = append(networks, info)
	}

	return networks
}

// getInterfaceSpeed 获取网卡速率（从/sys/class/net/<name>/speed读取）
func getInterfaceSpeed(name string) int64 {
	data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/speed", name))
	if err != nil {
		return 0
	}
	var speed int64
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &speed)
	return speed
}

// getGPUInfo 获取GPU信息
func (h *Handler) getGPUInfo() []*agent.GpuInfo {
	gpus := make([]*agent.GpuInfo, 0)

	// 优先使用nvidia-smi（驱动已安装）
	if out, err := exec.Command("nvidia-smi", "--query-gpu=name,driver_version", "--format=csv,noheader").Output(); err == nil {
		// 获取驱动版本
		driverVersion := ""
		if versionOut, err := exec.Command("nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader").Output(); err == nil {
			driverVersion = strings.TrimSpace(string(versionOut))
		}

		gpuCount := make(map[string]int32)
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			// 解析 "name, version" 格式
			parts := strings.SplitN(line, ",", 2)
			if len(parts) >= 1 {
				name := strings.TrimSpace(parts[0])
				if name != "" {
					gpuCount[name]++
				}
			}
			// 从第一行获取驱动版本
			if driverVersion == "" && len(parts) >= 2 {
				driverVersion = strings.TrimSpace(parts[1])
			}
		}
		for name, count := range gpuCount {
			gpus = append(gpus, &agent.GpuInfo{
				Name:          name,
				Count:         count,
				DriverVersion: driverVersion,
			})
		}
		return gpus
	}

	// 降级使用lspci（驱动未安装）
	if out, err := exec.Command("lspci", "-nn", "-d", "10de:").Output(); err == nil {
		gpuCount := make(map[string]int32)
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			// 提取冒号后的描述
			if idx := strings.Index(line, ": "); idx != -1 {
				desc := strings.TrimSpace(line[idx+2:])
				// 去掉(rev xx)后缀
				if revIdx := strings.Index(desc, " (rev "); revIdx != -1 {
					desc = desc[:revIdx]
				}
				// 简化名称
				desc = simplifyGPUName(desc)
				gpuCount[desc]++
			}
		}
		for name, count := range gpuCount {
			gpus = append(gpus, &agent.GpuInfo{
				Name:  name,
				Count: count,
			})
		}
	}

	return gpus
}

// simplifyGPUName 简化GPU名称
// "NVIDIA Corporation GA100 [A100 PCIe 40GB] (rev a1)" -> "NVIDIA A100 PCIe 40GB"
func simplifyGPUName(name string) string {
	// 去掉 "Corporation"
	name = strings.ReplaceAll(name, "Corporation", "")
	name = strings.TrimSpace(name)

	// 提取方括号中的内容（如果有）
	if start := strings.Index(name, "["); start != -1 {
		if end := strings.Index(name[start:], "]"); end != -1 {
			short := name[start+1 : start+end]
			// 加上NVIDIA前缀
			if strings.HasPrefix(strings.ToUpper(name), "NVIDIA") {
				return "NVIDIA " + short
			}
			return short
		}
	}

	// 没有方括号，尝试去掉芯片型号（如GA100, AD102等）
	if idx := strings.Index(name, " "); idx != -1 {
		parts := strings.Fields(name)
		if len(parts) >= 2 {
			// 跳过可能的芯片型号（通常是大写字母+数字组合）
			for i, part := range parts {
				if i == 0 {
					continue
				}
				// 如果是芯片型号（如GA100, AD102），跳过
				if len(part) >= 3 && part[:1] == strings.ToUpper(part[:1]) &&
					strings.ContainsAny(part, "0123456789") {
					continue
				}
				// 找到描述部分
				return strings.Join(parts[:i], " ") + " " + strings.Join(parts[i:], " ")
			}
		}
	}

	return strings.TrimSpace(name)
}

// getGateway 获取默认网关
func (h *Handler) getGateway() string {
	if out, err := exec.Command("ip", "route", "show", "default").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 {
			fields := strings.Fields(lines[0])
			for i, field := range fields {
				if field == "via" && i+1 < len(fields) {
					return fields[i+1]
				}
			}
		}
	}
	return ""
}

// getFirewallInfo 获取防火墙状态
func (h *Handler) getFirewallInfo() *agent.FirewallInfo {
	info := &agent.FirewallInfo{
		Status:  "inactive",
		Enabled: "disabled",
	}

	// 检查firewalld
	if _, err := exec.Command("bash", "-c", "systemctl list-unit-files --type=service | grep -q firewalld").Output(); err == nil {
		// firewalld存在
		if out, err := exec.Command("systemctl", "is-active", "firewalld").Output(); err == nil {
			info.Status = strings.TrimSpace(string(out))
		}
		if out, err := exec.Command("systemctl", "is-enabled", "firewalld").Output(); err == nil {
			info.Enabled = strings.TrimSpace(string(out))
		}
		return info
	}

	// 检查ufw
	if _, err := exec.Command("bash", "-c", "systemctl list-unit-files --type=service | grep -q ufw").Output(); err == nil {
		// ufw存在
		if out, err := exec.Command("systemctl", "is-active", "ufw").Output(); err == nil {
			info.Status = strings.TrimSpace(string(out))
		}
		if out, err := exec.Command("systemctl", "is-enabled", "ufw").Output(); err == nil {
			info.Enabled = strings.TrimSpace(string(out))
		}
		return info
	}

	return info
}

// getSwapInfo 获取Swap信息
func (h *Handler) getSwapInfo() *agent.SwapInfo {
	info := &agent.SwapInfo{}

	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				valueStr := strings.TrimSpace(parts[1])
				valueStr = strings.TrimSuffix(valueStr, " kB")
				var value int64
				fmt.Sscanf(valueStr, "%d", &value)

				if key == "SwapTotal" {
					info.TotalKb = value
					break
				}
			}
		}
	}

	return info
}

// getTimezone 获取时区
func (h *Handler) getTimezone() string {
	// 优先使用 timedatectl 获取标准时区名
	if out, err := exec.Command("timedatectl", "show", "-p", "Timezone", "--value").Output(); err == nil {
		tz := strings.TrimSpace(string(out))
		if tz != "" {
			return tz
		}
	}
	// 回退到 time.LoadLocation
	loc, _ := time.LoadLocation("Local")
	return loc.String()
}

// GetSystemTime 获取系统时间
func (h *Handler) GetSystemTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// GetHardwareTime 获取硬件时间
func (h *Handler) GetHardwareTime() string {
	if out, err := exec.Command("hwclock", "-r").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// getVirtualization 获取虚拟化类型
func (h *Handler) getVirtualization() string {
	// 检查是否在Docker容器中
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}

	// 检查systemd-detect-virt
	if out, err := exec.Command("systemd-detect-virt").Output(); err == nil {
		virt := strings.TrimSpace(string(out))
		if virt != "none" {
			return virt
		}
	}

	// 检查/proc/cpuinfo中的hypervisor
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		if strings.Contains(string(data), "hypervisor") {
			return "kvm"
		}
	}

	return "none"
}

// GetUptime 获取运行时间（秒）
func (h *Handler) GetUptime() int64 {
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		var uptime float64
		fmt.Sscanf(strings.Fields(string(data))[0], "%f", &uptime)
		return int64(uptime)
	}
	return 0
}

// 确保Handler实现了AgentServiceServer接口
var _ agent.AgentServiceServer = (*Handler)(nil)
