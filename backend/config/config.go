package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// 默认配置目录
const DefaultConfigDir = "/etc/fastdp-orbit"

// ==================== 通用配置 ====================

// Config 通用配置（server和agent共用）
type Config struct {
	Mode string    `mapstructure:"mode"` // debug, release
	Log  LogConfig `mapstructure:"log"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`       // 日志级别: debug, info, warn, error
	Output     string `mapstructure:"output"`      // 输出方式: stdout, file, both
	Path       string `mapstructure:"path"`        // 日志文件路径
	MaxSize    int    `mapstructure:"max_size"`    // 单个文件最大MB
	MaxBackups int    `mapstructure:"max_backups"` // 最多保留备份数
	MaxAge     int    `mapstructure:"max_age"`     // 保留天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩旧日志
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`   // 是否启用TLS
	CertFile string `mapstructure:"cert_file"` // 证书文件路径
	KeyFile  string `mapstructure:"key_file"`  // 私钥文件路径
	CAFile   string `mapstructure:"ca_file"`   // CA证书文件路径（双向TLS需要）
}

// ==================== Server 配置 ====================

// ServerConfig server配置
type ServerConfig struct {
	OrbitServer OrbitServerConfig `mapstructure:"orbit-server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	GRPC        GRPCConfig        `mapstructure:"grpc"`
	Log         LogConfig         `mapstructure:"log"`
}

// OrbitServerConfig orbit-server配置
type OrbitServerConfig struct {
	Address string    `mapstructure:"address"`
	Port    int       `mapstructure:"port"`
	Mode    string    `mapstructure:"mode"` // debug, release
	TLS     TLSConfig `mapstructure:"tls"`  // HTTP TLS配置
}

// GRPCConfig gRPC配置
type GRPCConfig struct {
	Address        string    `mapstructure:"address"`         // gRPC监听地址
	Port           int       `mapstructure:"port"`            // gRPC监听端口
	DefaultTimeout int       `mapstructure:"default_timeout"` // 默认超时时间(秒)
	Token          string    `mapstructure:"token"`           // 集群准入Token
	TLS            TLSConfig `mapstructure:"tls"`             // gRPC TLS配置
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string `mapstructure:"type"` // sqlite, mysql, postgres
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Path     string `mapstructure:"path"` // sqlite路径
}

// ==================== Agent 配置 ====================

// AgentConfig agent配置
type AgentConfig struct {
	OrbitAgent OrbitAgentConfig `mapstructure:"orbit-agent"`
	Log        LogConfig        `mapstructure:"log"`
}

// OrbitAgentConfig orbit-agent配置
type OrbitAgentConfig struct {
	Host          string    `mapstructure:"host"`           // Agent监听地址 + 上报给Server的IP（不能是0.0.0.0或127.0.0.1）
	Port          int       `mapstructure:"port"`           // Agent gRPC端口
	RpcServerHost string    `mapstructure:"rpcserver_host"` // Server的gRPC地址
	RpcServerPort int       `mapstructure:"rpcserver_port"` // Server的gRPC端口
	Token         string    `mapstructure:"token"`          // 集群准入Token
	Heartbeat     int       `mapstructure:"heartbeat"`      // 心跳间隔(秒)
	TLS           TLSConfig `mapstructure:"tls"`            // gRPC TLS配置
}

// ==================== Flags ====================

// ServerFlags command line flags for server
type ServerFlags struct {
	Config string
	Port   int
	Mode   string
}

// AgentFlags command line flags for agent
type AgentFlags struct {
	Config string
	Server string // --server host:port
	Port   int    // --port agent监听端口
	Host   string // --host Agent监听地址 + 上报给Server的IP
}

// ParseServerFlags parses command line flags for server
func ParseServerFlags() *ServerFlags {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	flags := &ServerFlags{}

	fs.StringVar(&flags.Config, "config", "", "配置文件路径 (默认: /etc/fastdp-orbit/server.toml)")
	fs.IntVar(&flags.Port, "port", 0, "服务监听端口 (覆盖配置文件)")
	fs.StringVar(&flags.Mode, "mode", "", "运行模式: debug/release (覆盖配置文件)")

	fs.Parse(os.Args[1:])
	return flags
}

// ParseAgentFlags parses command line flags for agent
func ParseAgentFlags() *AgentFlags {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)
	flags := &AgentFlags{}

	fs.StringVar(&flags.Config, "config", "", "配置文件路径 (默认: /etc/fastdp-orbit/agent.toml)")
	fs.StringVar(&flags.Server, "server", "", "Server gRPC地址 (host:port)")
	fs.IntVar(&flags.Port, "port", 0, "Agent监听端口 (覆盖配置文件)")
	fs.StringVar(&flags.Host, "host", "", "Agent监听地址 + 上报给Server的IP (覆盖配置文件)")

	fs.Parse(os.Args[1:])
	return flags
}

// ==================== 配置验证 ====================

// validatePort 验证端口范围
func validatePort(port int, fieldName string) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s 端口 %d 不在有效范围内 (1-65535)", fieldName, port)
	}
	return nil
}

// validateRequired 验证必填字段
func validateRequired(value string, fieldName string) error {
	if value == "" {
		return fmt.Errorf("%s 不能为空", fieldName)
	}
	return nil
}

// validateTLSConfig 验证TLS配置
func validateTLSConfig(cfg TLSConfig, componentName string, requireCA bool) error {
	if !cfg.Enabled {
		return nil
	}

	// 验证证书文件路径
	if err := validateRequired(cfg.CertFile, componentName+" TLS 证书文件(cert_file)"); err != nil {
		return err
	}

	// 验证私钥文件路径
	if err := validateRequired(cfg.KeyFile, componentName+" TLS 私钥文件(key_file)"); err != nil {
		return err
	}

	// 验证CA文件路径（双向TLS需要）
	if requireCA {
		if err := validateRequired(cfg.CAFile, componentName+" TLS CA证书文件(ca_file)"); err != nil {
			return err
		}
	}

	return nil
}

// validateServerConfig 验证server配置
func validateServerConfig(cfg *ServerConfig) error {
	// 验证HTTP监听地址
	if err := validateRequired(cfg.OrbitServer.Address, "Server HTTP监听地址"); err != nil {
		return err
	}
	if cfg.OrbitServer.Address == "0.0.0.0" || isLoopbackAddress(cfg.OrbitServer.Address) {
		return fmt.Errorf("Server HTTP监听地址不能是 %s，请使用真实IP", cfg.OrbitServer.Address)
	}
	// 验证HTTP端口
	if err := validatePort(cfg.OrbitServer.Port, "HTTP"); err != nil {
		return err
	}
	// 验证gRPC监听地址
	if err := validateRequired(cfg.GRPC.Address, "Server gRPC监听地址"); err != nil {
		return err
	}
	if cfg.GRPC.Address == "0.0.0.0" || isLoopbackAddress(cfg.GRPC.Address) {
		return fmt.Errorf("Server gRPC监听地址不能是 %s，请使用真实IP", cfg.GRPC.Address)
	}
	// 验证gRPC端口
	if err := validatePort(cfg.GRPC.Port, "gRPC"); err != nil {
		return err
	}
	// 验证数据库端口（非sqlite时）
	if cfg.Database.Type != "sqlite" {
		if err := validatePort(cfg.Database.Port, "数据库端口"); err != nil {
			return err
		}
		if err := validateRequired(cfg.Database.Host, "数据库主机"); err != nil {
			return err
		}
	}
	// 验证HTTP TLS配置（单向TLS，不需要CA证书）
	if err := validateTLSConfig(cfg.OrbitServer.TLS, "Server HTTP", false); err != nil {
		return err
	}
	// 验证gRPC TLS配置（双向TLS，需要CA证书）
	if err := validateTLSConfig(cfg.GRPC.TLS, "Server gRPC", true); err != nil {
		return err
	}
	return nil
}

// validateAgentConfig 验证agent配置
func validateAgentConfig(cfg *AgentConfig) error {
	// 验证Agent监听地址（必填，不能是0.0.0.0或127.0.0.1）
	if err := validateRequired(cfg.OrbitAgent.Host, "Agent监听地址"); err != nil {
		return err
	}
	if cfg.OrbitAgent.Host == "0.0.0.0" || isLoopbackAddress(cfg.OrbitAgent.Host) {
		return fmt.Errorf("Agent监听地址不能是 %s，请使用真实IP", cfg.OrbitAgent.Host)
	}
	// 验证Agent监听端口
	if err := validatePort(cfg.OrbitAgent.Port, "Agent监听端口"); err != nil {
		return err
	}
	// 验证Server gRPC地址（必填，不能是0.0.0.0）
	if err := validateRequired(cfg.OrbitAgent.RpcServerHost, "Server gRPC 地址"); err != nil {
		return err
	}
	if cfg.OrbitAgent.RpcServerHost == "0.0.0.0" {
		return fmt.Errorf("Server gRPC地址不能是 0.0.0.0，请使用真实IP")
	}
	// 验证Server gRPC端口
	if err := validatePort(cfg.OrbitAgent.RpcServerPort, "Server gRPC 端口"); err != nil {
		return err
	}
	// 验证心跳间隔
	if cfg.OrbitAgent.Heartbeat < 1 {
		return fmt.Errorf("心跳间隔必须大于0秒")
	}
	// 验证gRPC TLS配置（双向TLS，需要CA证书）
	if err := validateTLSConfig(cfg.OrbitAgent.TLS, "Agent gRPC", true); err != nil {
		return err
	}
	return nil
}

// isLoopbackAddress 判断是否是回环地址
func isLoopbackAddress(ip string) bool {
	return strings.HasPrefix(ip, "127.") || ip == "::1"
}

// ==================== Load Functions ====================

// LoadServerWithFlags 使用指定的flags加载server配置
func LoadServerWithFlags(flags *ServerFlags) (*ServerConfig, error) {
	return loadServerConfig(flags.Config, flags)
}

// LoadAgentWithFlags 使用指定的flags加载agent配置
func LoadAgentWithFlags(flags *AgentFlags) (*AgentConfig, error) {
	return loadAgentConfig(flags.Config, flags)
}

// LoadCLI 加载CLI配置（不需要完整配置，只返回基础配置）
func LoadCLI() *Config {
	fs := flag.NewFlagSet("cli", flag.ExitOnError)
	configPath := fs.String("config", "", "配置文件路径")
	fs.Parse(os.Args[1:])

	// CLI模式不需要完整配置，只返回基础配置
	_ = configPath
	return &Config{
		Mode: "debug",
		Log: LogConfig{
			Level:  "info",
			Output: "stdout",
		},
	}
}

// loadServerConfig 加载server配置
func loadServerConfig(configPath string, flags *ServerFlags) (*ServerConfig, error) {
	v := viper.New()

	// 默认配置
	v.SetDefault("orbit-server.address", "0.0.0.0")
	v.SetDefault("orbit-server.port", 8080)
	v.SetDefault("orbit-server.mode", "debug")
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.path", "./data/fastdp-orbit.db")
	v.SetDefault("grpc.address", "0.0.0.0")
	v.SetDefault("grpc.port", 9090)
	v.SetDefault("grpc.default_timeout", 60)
	v.SetDefault("log.level", "info")

	// 设置配置文件路径
	if configPath != "" {
		dir := filepath.Dir(configPath)
		fileName := filepath.Base(configPath)
		ext := filepath.Ext(fileName)
		v.SetConfigName(fileName[:len(fileName)-len(ext)])
		v.SetConfigType(ext[1:])
		v.AddConfigPath(dir)
	} else {
		v.SetConfigName("server")
		v.SetConfigType("toml")
		v.AddConfigPath(DefaultConfigDir)
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			if configPath != "" {
				return nil, fmt.Errorf("配置文件读取失败: %w", err)
			}
		}
	}

	var cfg ServerConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("配置解析失败: %w", err)
	}

	// 命令行参数覆盖配置文件
	if flags.Port > 0 {
		cfg.OrbitServer.Port = flags.Port
	}
	if flags.Mode != "" {
		cfg.OrbitServer.Mode = flags.Mode
	}

	// 验证配置
	if err := validateServerConfig(&cfg); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 自动生成Token（首次启动或使用默认值时）
	if cfg.GRPC.Token == "" || cfg.GRPC.Token == "fastdp-orbit-token-change-me" {
		token, err := GenerateToken(16) // 32位十六进制字符串
		if err != nil {
			return nil, fmt.Errorf("生成Token失败: %w", err)
		}
		cfg.GRPC.Token = token

		// 写回配置文件
		if configPath != "" {
			v.Set("grpc.token", token)
			if err := v.WriteConfigAs(configPath); err != nil {
				fmt.Printf("警告：无法写入配置文件，Token未持久化: %v\n", err)
			}
		} else {
			// 尝试写入默认配置路径
			defaultPath := filepath.Join(DefaultConfigDir, "server.toml")
			v.Set("grpc.token", token)
			if err := v.WriteConfigAs(defaultPath); err != nil {
				fmt.Printf("警告：无法写入配置文件，Token未持久化: %v\n", err)
			}
		}
	}

	// 确保sqlite数据目录存在
	if cfg.Database.Type == "sqlite" && cfg.Database.Path != "" {
		dir := filepath.Dir(cfg.Database.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建数据目录失败: %w", err)
		}
	}

	return &cfg, nil
}

// loadAgentConfig 加载agent配置
func loadAgentConfig(configPath string, flags *AgentFlags) (*AgentConfig, error) {
	v := viper.New()

	// 默认配置
	v.SetDefault("orbit-agent.host", "")
	v.SetDefault("orbit-agent.port", 8700)
	v.SetDefault("orbit-agent.rpcserver_host", "") // 必填，不设默认值
	v.SetDefault("orbit-agent.rpcserver_port", 9090)
	v.SetDefault("orbit-agent.heartbeat", 30)
	v.SetDefault("log.level", "info")

	// 设置配置文件路径
	if configPath != "" {
		dir := filepath.Dir(configPath)
		fileName := filepath.Base(configPath)
		ext := filepath.Ext(fileName)
		v.SetConfigName(fileName[:len(fileName)-len(ext)])
		v.SetConfigType(ext[1:])
		v.AddConfigPath(dir)
	} else {
		v.SetConfigName("agent")
		v.SetConfigType("toml")
		v.AddConfigPath(DefaultConfigDir)
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			if configPath != "" {
				return nil, fmt.Errorf("配置文件读取失败: %w", err)
			}
		}
	}

	var cfg AgentConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("配置解析失败: %w", err)
	}

	// 命令行参数覆盖配置文件
	if flags.Server != "" {
		host, portStr, err := net.SplitHostPort(flags.Server)
		if err != nil {
			return nil, fmt.Errorf("--server 参数格式错误，应为 host:port: %w", err)
		}
		cfg.OrbitAgent.RpcServerHost = host
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("--server 端口号格式错误: %w", err)
		}
		cfg.OrbitAgent.RpcServerPort = port
	}
	if flags.Port > 0 {
		cfg.OrbitAgent.Port = flags.Port
	}
	if flags.Host != "" {
		cfg.OrbitAgent.Host = flags.Host
	}

	// 验证配置
	if err := validateAgentConfig(&cfg); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &cfg, nil
}

// ==================== 工具函数 ====================

// GenerateToken 生成随机Token
func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetServerAddr 获取server gRPC完整地址
func (c *AgentConfig) GetServerAddr() string {
	return c.OrbitAgent.RpcServerHost + ":" + strconv.Itoa(c.OrbitAgent.RpcServerPort)
}

// GetAgentListenAddr 获取agent监听地址
func (c *AgentConfig) GetAgentListenAddr() string {
	return c.OrbitAgent.Host + ":" + strconv.Itoa(c.OrbitAgent.Port)
}

// GetServerListenAddr 获取server监听地址
func (c *ServerConfig) GetServerListenAddr() string {
	return c.OrbitServer.Address + ":" + strconv.Itoa(c.OrbitServer.Port)
}

// GetGRPCListenAddr 获取gRPC监听地址
func (c *ServerConfig) GetGRPCListenAddr() string {
	return c.GRPC.Address + ":" + strconv.Itoa(c.GRPC.Port)
}
