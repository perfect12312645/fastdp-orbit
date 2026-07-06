package commands

import (
	"fmt"

	"fastdp-orbit/backend/cli/output"
	"fastdp-orbit/backend/config"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "退出登录",
	Long:  "清除本地保存的认证令牌，退出当前登录状态。不会影响其他会话。",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfigFromCmd(cmd)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}

		if cfg.Auth.Token == "" {
			output.PrintWarning("当前未登录")
			return nil
		}

		// 清除 token
		cfg.Auth.Token = ""
		configPath, _ := cmd.Flags().GetString("config")
		if err := config.SaveCLIConfig(cfg, configPath); err != nil {
			return fmt.Errorf("保存配置失败: %v", err)
		}

		output.PrintSuccess("已退出登录")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
