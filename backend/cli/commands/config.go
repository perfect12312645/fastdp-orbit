package commands

import (
	"fmt"
	"os"
	"strings"

	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/cli/output"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理CLI配置",
	Long:  "管理orbitctl的配置文件，包括Server地址、TLS设置等",
}

var configSetServerCmd = &cobra.Command{
	Use:   "set-server <address>",
	Short: "设置Server地址",
	Long:  "设置orbitctl连接的Server地址，格式为 host:port",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		address := args[0]

		// 验证地址格式
		if !strings.Contains(address, ":") {
			return fmt.Errorf("地址格式错误，应为 host:port")
		}

		// 加载现有配置
		configPath, _ := cmd.Flags().GetString("config")
		cfg, err := config.LoadCLIConfig(configPath)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}

		// 更新Server地址
		cfg.Server.Address = "https://" + address

		// 保存配置
		if err := config.SaveCLIConfig(cfg, configPath); err != nil {
			return fmt.Errorf("保存配置失败: %v", err)
		}

		output.PrintSuccess(fmt.Sprintf("Server地址已设置为: %s", cfg.Server.Address))
		output.PrintWarning("注意: 默认使用HTTPS协议，如需HTTP请直接编辑配置文件")
		return nil
	},
}

var configGetServerCmd = &cobra.Command{
	Use:   "get-server",
	Short: "获取当前Server地址",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		cfg, err := config.LoadCLIConfig(configPath)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}

		if cfg.Server.Address == "" {
			output.PrintWarning("未设置Server地址")
			output.PrintInfo("使用 'orbitctl config set-server <address>' 设置")
			return nil
		}

		output.PrintInfo(cfg.Server.Address)
		return nil
	},
}

var configSetTLSCmd = &cobra.Command{
	Use:   "set-tls-insecure <true|false>",
	Short: "设置是否跳过TLS验证",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]
		var insecure bool
		switch strings.ToLower(value) {
		case "true", "yes", "1":
			insecure = true
		case "false", "no", "0":
			insecure = false
		default:
			return fmt.Errorf("无效的值: %s，应为 true 或 false", value)
		}

		configPath, _ := cmd.Flags().GetString("config")
		cfg, err := config.LoadCLIConfig(configPath)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}

		cfg.TLS.InsecureSkipTLSVerify = insecure

		if err := config.SaveCLIConfig(cfg, configPath); err != nil {
			return fmt.Errorf("保存配置失败: %v", err)
		}

		output.PrintSuccess(fmt.Sprintf("TLS验证跳过设置为: %v", insecure))
		return nil
	},
}

var configSetTLSCACertCmd = &cobra.Command{
	Use:   "set-tls-ca-cert <path>",
	Short: "设置CA证书路径",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		caCert := args[0]

		// 验证文件是否存在
		if _, err := os.Stat(caCert); os.IsNotExist(err) {
			return fmt.Errorf("CA证书文件不存在: %s", caCert)
		}

		configPath, _ := cmd.Flags().GetString("config")
		cfg, err := config.LoadCLIConfig(configPath)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}

		cfg.TLS.CACert = caCert

		if err := config.SaveCLIConfig(cfg, configPath); err != nil {
			return fmt.Errorf("保存配置失败: %v", err)
		}

		output.PrintSuccess(fmt.Sprintf("CA证书路径已设置为: %s", caCert))
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "显示配置文件路径",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = config.GetCLIConfigPath()
		}

		// 检查文件是否存在
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			output.PrintWarning(fmt.Sprintf("配置文件不存在: %s", configPath))
			output.PrintInfo("运行 'orbitctl config set-server <address>' 创建配置")
			return nil
		}

		output.PrintInfo(configPath)
		return nil
	},
}

func init() {
	// 添加子命令
	configCmd.AddCommand(configSetServerCmd)
	configCmd.AddCommand(configGetServerCmd)
	configCmd.AddCommand(configSetTLSCmd)
	configCmd.AddCommand(configSetTLSCACertCmd)
	configCmd.AddCommand(configPathCmd)

	// 注册到根命令
	rootCmd.AddCommand(configCmd)
}

// GetConfigPath 获取配置文件路径（供其他命令使用）
func GetConfigPath(cmd *cobra.Command) string {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		return configPath
	}
	return config.GetCLIConfigPath()
}

// LoadConfigFromCmd 从命令加载配置（供其他命令使用）
func LoadConfigFromCmd(cmd *cobra.Command) (*config.CLIConfig, error) {
	configPath, _ := cmd.Flags().GetString("config")
	return config.LoadCLIConfig(configPath)
}

// EnsureServerConfig 确保Server地址已配置（供其他命令使用）
func EnsureServerConfig(cfg *config.CLIConfig) error {
	if cfg.Server.Address == "" {
		return fmt.Errorf("未设置Server地址，请先运行: orbitctl config set-server <address>")
	}
	return nil
}

// EnsureAuth 确保已登录（供需要认证的命令使用）
func EnsureAuth(cfg *config.CLIConfig) error {
	if cfg.Auth.Token == "" {
		return fmt.Errorf("未登录，请先运行: orbitctl login <username>")
	}
	return nil
}
