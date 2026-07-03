package commands

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"

	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/models/common"

	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var resetPwdConfig string

var resetPwdCmd = &cobra.Command{
	Use:   "reset-password [username]",
	Short: "重置用户密码",
	Long: `重置指定用户的密码，生成随机密码并输出到终端。
需要服务器配置文件来确定数据库路径。

用法:
  orbitctl reset-password admin
  orbitctl reset-password --config /path/to/server.toml admin`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]

		// 加载服务端配置获取数据库路径
		cfg, err := config.LoadServerWithFlags(&config.ServerFlags{
			Config: resetPwdConfig,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
			fmt.Fprintf(os.Stderr, "可通过 --config 参数指定配置文件路径\n")
			os.Exit(1)
		}
		dbPath := cfg.Database.Path
		if dbPath == "" {
			dbPath = "/opt/fastdp-orbit/data/fastdp-orbit.db"
		}

		// 检查数据库文件是否存在（SQLite 不会自动创建空文件）
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "数据库文件不存在: %s\n", dbPath)
			fmt.Fprintf(os.Stderr, "请确认服务器配置文件路径正确，已退出。\n")
			os.Exit(1)
		}

		// 直连 SQLite
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "连接数据库失败: %v\n", err)
			os.Exit(1)
		}

		var user common.User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			fmt.Fprintf(os.Stderr, "用户 '%s' 不存在\n", username)
			os.Exit(1)
		}

		// 生成随机密码（12位，大小写+数字）
		const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
		pwd := make([]byte, 12)
		for i := range pwd {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			pwd[i] = chars[n.Int64()]
		}
		newPwd := string(pwd)

		hashed, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
		if err != nil {
			fmt.Fprintf(os.Stderr, "密码加密失败: %v\n", err)
			os.Exit(1)
		}

		db.Model(&user).Updates(map[string]any{
			"password":        string(hashed),
			"must_change_pwd": true,
		})

		fmt.Println("╔══════════════════════════════════════════╗")
		fmt.Println("║          密码重置成功                     ║")
		fmt.Println("╠══════════════════════════════════════════╣")
		fmt.Printf("║  用户名:  %-26s║\n", user.Username)
		fmt.Printf("║  新密码:  %-26s║\n", newPwd)
		fmt.Println("╠══════════════════════════════════════════╣")
		fmt.Println("║  请在登录后立即修改密码                    ║")
		fmt.Println("╚══════════════════════════════════════════╝")
	},
}

func init() {
	resetPwdCmd.Flags().StringVarP(&resetPwdConfig, "config", "c", "", "服务器配置文件路径（默认: /etc/fastdp-orbit/server.toml）")
	rootCmd.AddCommand(resetPwdCmd)
}
