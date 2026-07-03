package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"fastdp-orbit/backend/cli/cliutil"
	"fastdp-orbit/backend/cli/output"
	"fastdp-orbit/backend/config"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login [username]",
	Short: "登录到 Orbit Server",
	Long: `登录到 Orbit Server 并保存认证令牌。
令牌将保存在配置文件（~/.fastdp-orbit/config.toml）中。

用法:
  orbitctl login admin          # 交互式输入密码
  orbitctl login --password-stdin < user.pwd   # 从标准输入读取密码`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		passwordStdin, _ := cmd.Flags().GetBool("password-stdin")

		// 加载配置（需先设置 server address）
		cfg, err := LoadConfigFromCmd(cmd)
		if err != nil {
			return fmt.Errorf("加载配置失败: %v", err)
		}
		if err := EnsureServerConfig(cfg); err != nil {
			return err
		}

		// 获取密码
		var password string
		if passwordStdin {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				password = strings.TrimSpace(scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("读取密码失败: %v", err)
			}
		} else {
			fmt.Printf("密码: ")
			bytePwd, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("读取密码失败: %v", err)
			}
			password = strings.TrimSpace(string(bytePwd))
			fmt.Println()
		}

		if password == "" {
			return fmt.Errorf("密码不能为空")
		}

		// 创建 HTTP 客户端
		client, err := cliutil.NewHTTPClient(cfg)
		if err != nil {
			return fmt.Errorf("创建 HTTP 客户端失败: %v", err)
		}

		// 发送登录请求
		req, err := cliutil.NewRequest(cfg, "POST", "/api/v1/auth/login", map[string]string{
			"username": username,
			"password": password,
		})
		if err != nil {
			return fmt.Errorf("创建请求失败: %v", err)
		}

		var loginResp struct {
			Token string `json:"token"`
		}
		if err := cliutil.Do(client, req, &loginResp); err != nil {
			return fmt.Errorf("登录失败: %v", err)
		}

		if loginResp.Token == "" {
			return fmt.Errorf("登录成功但未获取到令牌")
		}

		// 保存 token 到配置
		cfg.Auth.Token = loginResp.Token
		configPath, _ := cmd.Flags().GetString("config")
		if err := config.SaveCLIConfig(cfg, configPath); err != nil {
			return fmt.Errorf("保存令牌失败: %v", err)
		}

		output.PrintSuccess("登录成功")
		return nil
	},
}

func init() {
	loginCmd.Flags().Bool("password-stdin", false, "从标准输入读取密码")
	rootCmd.AddCommand(loginCmd)
}
