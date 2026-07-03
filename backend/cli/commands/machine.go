package commands

import (
	"fmt"
	"strconv"
	"strings"

	"fastdp-orbit/backend/cli/cliutil"
	"fastdp-orbit/backend/cli/output"

	"github.com/spf13/cobra"
)

// MachineInfo CLI 层机器信息（与 API 返回字段对齐）
type MachineInfo struct {
	ID              uint   `json:"id"`
	IP              string `json:"ip"`
	Port            int    `json:"port"`
	Status          string `json:"status"`
	Hostname        string `json:"hostname"`
	Virtualization  string `json:"virtualization"`
	UptimeSeconds   int64  `json:"uptime_seconds"`
	SystemTime      string `json:"system_time"`
	HardwareTime    string `json:"hardware_time"`
	OSName          string `json:"os_name"`
	OSVersion       string `json:"os_version"`
	Kernel          string `json:"kernel"`
	Arch            string `json:"arch"`
	CPUModel        string `json:"cpu_model"`
	CPUCores        int32  `json:"cpu_cores"`
	MemoryKB        int64  `json:"memory_kb"`
	SwapKB          int64  `json:"swap_kb"`
	Gateway         string `json:"gateway"`
	FirewallStatus  string `json:"firewall_status"`
	FirewallEnabled string `json:"firewall_enabled"`
	Timezone        string `json:"timezone"`
}

var machineCmd = &cobra.Command{
	Use:   "machine",
	Short: "管理机器",
}

// machineListCmd 列出所有已注册的机器
var machineListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有机器",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfigFromCmd(cmd)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}
		if err := EnsureServerConfig(cfg); err != nil {
			return err
		}
		if err := EnsureAuth(cfg); err != nil {
			return err
		}

		client, err := cliutil.NewHTTPClient(cfg)
		if err != nil {
			return fmt.Errorf("创建 HTTP 客户端失败: %v", err)
		}

		req, err := cliutil.NewRequest(cfg, "GET", "/api/v1/machines", nil)
		if err != nil {
			return fmt.Errorf("创建请求失败: %v", err)
		}

		var machines []MachineInfo
		if err := cliutil.Do(client, req, &machines); err != nil {
			return fmt.Errorf("获取机器列表失败: %v", err)
		}

		if len(machines) == 0 {
			output.PrintWarning("暂无已注册的机器")
			return nil
		}

		rows := make([][]string, 0, len(machines))
		for _, m := range machines {
			hostname := m.Hostname
			if hostname == "" {
				hostname = "-"
			}
			osInfo := m.OSName
			if m.OSVersion != "" {
				osInfo += " " + m.OSVersion
			}
			if osInfo == "" {
				osInfo = "-"
			}
			cpuModel := m.CPUModel
			if cpuModel == "" {
				cpuModel = "-"
			} else if len(cpuModel) > 30 {
				cpuModel = cpuModel[:30] + "..."
			}
			cpuCores := ""
			if m.CPUCores > 0 {
				cpuCores = fmt.Sprintf("%d核", m.CPUCores)
			} else {
				cpuCores = "-"
			}
			memStr := "-"
			if m.MemoryKB > 0 {
				memGB := float64(m.MemoryKB) / 1024 / 1024
				memStr = fmt.Sprintf("%.1fGB", memGB)
			}

			status := m.Status
			if status == "" {
				status = "unknown"
			}

			rows = append(rows, []string{
				m.IP,
				strconv.Itoa(m.Port),
				hostname,
				status,
				osInfo,
				cpuCores,
				memStr,
			})
		}

		output.PrintTable(
			[]string{"IP", "端口", "主机名", "状态", "操作系统", "CPU", "内存"},
			rows,
		)
		fmt.Printf("\n总计: %d 台机器\n", len(machines))
		return nil
	},
}

// machineRemoveCmd 删除机器
var machineRemoveCmd = &cobra.Command{
	Use:   "remove <ip:port>",
	Short: "删除机器",
	Long: `从 Server 中删除机器记录。
机器 Agent 下次心跳时会自动退出；若需立即停用，请登录机器执行 systemctl stop orbit-agent。

用法:
  orbitctl machine remove 192.168.1.100:9090`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := args[0]

		// 解析 ip:port
		parts := strings.Split(addr, ":")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("格式错误，应为 ip:port，例如: 192.168.1.100:9090")
		}
		ip := parts[0]
		port := parts[1]
		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf("端口格式错误: %s", port)
		}

		cfg, err := LoadConfigFromCmd(cmd)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}
		if err := EnsureServerConfig(cfg); err != nil {
			return err
		}
		if err := EnsureAuth(cfg); err != nil {
			return err
		}

		client, err := cliutil.NewHTTPClient(cfg)
		if err != nil {
			return fmt.Errorf("创建 HTTP 客户端失败: %v", err)
		}

		req, err := cliutil.NewRequest(cfg, "DELETE", fmt.Sprintf("/api/v1/machines/%s/%s", ip, port), nil)
		if err != nil {
			return fmt.Errorf("创建请求失败: %v", err)
		}

		var result string
		if err := cliutil.Do(client, req, &result); err != nil {
			return fmt.Errorf("删除机器失败: %v", err)
		}

		output.PrintSuccess(fmt.Sprintf("机器 %s:%s 已删除", ip, port))
		if result != "" {
			fmt.Println(result)
		}
		return nil
	},
}

// machineExecCmd 预留，暂不实现
var machineExecCmd = &cobra.Command{
	Use:   "exec",
	Short: "在机器上执行命令（暂未实现）",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("该命令尚未实现")
	},
}

func init() {
	machineCmd.AddCommand(machineListCmd)
	machineCmd.AddCommand(machineRemoveCmd)
	machineCmd.AddCommand(machineExecCmd)
}
