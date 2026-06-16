package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"fastdp-orbit/backend/cli/cliutil"
	"fastdp-orbit/backend/cli/output"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "获取Agent安装命令",
	Long:  "从Server获取Agent的安装命令，需要先配置Server地址",
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

		// 解析Server地址
		serverURL, err := url.Parse(cfg.Server.Address)
		if err != nil {
			return fmt.Errorf("Server地址格式错误: %v", err)
		}

		// 获取IP和端口
		host := serverURL.Hostname()
		port := serverURL.Port()
		if port == "" {
			if serverURL.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}

		// 构建API URL
		apiURL := fmt.Sprintf("%s://%s:%s/api/v1/install/command", serverURL.Scheme, host, port)

		// 检查TLS配置
		if serverURL.Scheme == "https" && !cfg.TLS.InsecureSkipTLSVerify && cfg.TLS.CACert == "" {
			return fmt.Errorf("Server使用HTTPS，但未配置CA证书且未跳过TLS验证，请执行：\n  orbitctl config set-tls-insecure true\n或\n  orbitctl config set-tls-ca-cert <ca证书路径>")
		}

		// 创建HTTP客户端
		client, err := cliutil.NewHTTPClient(cfg)
		if err != nil {
			return fmt.Errorf("创建HTTP客户端失败: %v", err)
		}

		// 从Server获取安装命令
		resp, err := client.Get(apiURL)
		if err != nil {
			return fmt.Errorf("连接Server失败: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("读取响应失败: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("解析响应失败: %v", err)
		}

		// 检查业务码
		if code, ok := result["code"].(float64); ok && code != 0 {
			msg, _ := result["message"].(string)
			return fmt.Errorf("Server返回错误: %s", msg)
		}

		// 提取data中的command
		data, ok := result["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("响应中缺少data字段")
		}

		cmdStr, ok := data["command"].(string)
		if !ok {
			return fmt.Errorf("响应中缺少command字段")
		}

		output.PrintSuccess("Orbit Agent 安装命令")
		fmt.Println()
		fmt.Println(cmdStr)
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
