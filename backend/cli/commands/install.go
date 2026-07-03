package commands

import (
	"fmt"
	"net/url"

	"fastdp-orbit/backend/cli/cliutil"
	"fastdp-orbit/backend/cli/output"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "获取Agent安装命令",
	Long:  "从Server获取Agent的安装命令，需要先配置Server地址并登录",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 加载CLI配置
		cfg, err := LoadConfigFromCmd(cmd)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}

		// 检查Server地址是否已配置
		if err := EnsureServerConfig(cfg); err != nil {
			return err
		}

		// 检查是否已登录
		if cfg.Auth.Token == "" {
			return fmt.Errorf("未登录，请先执行: orbitctl login <username>")
		}

		// 检查TLS配置
		serverURL, err := url.Parse(cfg.Server.Address)
		if err != nil {
			return fmt.Errorf("Server地址格式错误: %v", err)
		}
		if serverURL.Scheme == "https" && !cfg.TLS.InsecureSkipTLSVerify && cfg.TLS.CACert == "" {
			return fmt.Errorf("Server使用HTTPS，但未配置CA证书且未跳过TLS验证，请执行：\n  orbitctl config set-tls-insecure true\n或\n  orbitctl config set-tls-ca-cert <ca证书路径>")
		}

		// 创建HTTP客户端
		client, err := cliutil.NewHTTPClient(cfg)
		if err != nil {
			return fmt.Errorf("创建HTTP客户端失败: %v", err)
		}

		// 携带 token 发起请求（NewRequest 自动注入 Authorization header）
		req, err := cliutil.NewRequest(cfg, "GET", "/api/v1/install/command", nil)
		if err != nil {
			return fmt.Errorf("创建请求失败: %v", err)
		}

		var result struct {
			Command string `json:"command"`
		}
		if err := cliutil.Do(client, req, &result); err != nil {
			return fmt.Errorf("获取安装命令失败: %v", err)
		}

		if result.Command == "" {
			return fmt.Errorf("服务端返回的安装命令为空")
		}

		output.PrintSuccess("Orbit Agent 安装命令")
		fmt.Println()
		fmt.Println(result.Command)
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
