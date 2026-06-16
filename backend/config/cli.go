package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// CLIConfig orbitctl CLI配置
type CLIConfig struct {
	Server CLIServerConfig `mapstructure:"server"`
	TLS    CLITLSConfig    `mapstructure:"tls"`
	Auth   CLIAuthConfig   `mapstructure:"auth"`
}

// CLIServerConfig Server连接配置
type CLIServerConfig struct {
	Address string `mapstructure:"address"` // Server地址 (https://host:port)
}

// CLITLSConfig TLS配置
type CLITLSConfig struct {
	InsecureSkipTLSVerify bool   `mapstructure:"insecure-skip-tls-verify"` // 是否跳过TLS验证
	CACert                string `mapstructure:"ca-cert"`                  // CA证书路径
}

// CLIAuthConfig 认证配置（预留JWT）
type CLIAuthConfig struct {
	AccessToken  string `mapstructure:"access-token"`
	RefreshToken string `mapstructure:"refresh-token"`
	ExpiresAt    string `mapstructure:"expires-at"`
	User         string `mapstructure:"user"`
}

// DefaultCLIConfig 返回默认CLI配置
func DefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		Server: CLIServerConfig{
			Address: "",
		},
		TLS: CLITLSConfig{
			InsecureSkipTLSVerify: false,
			CACert:                "",
		},
		Auth: CLIAuthConfig{},
	}
}

// GetCLIConfigDir 获取CLI配置目录
// 优先级: 用户家目录 > /etc/fastdp-orbit
func GetCLIConfigDir() string {
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".fastdp-orbit")
	}
	return DefaultConfigDir
}

// GetCLIConfigPath 获取CLI配置文件路径
func GetCLIConfigPath() string {
	return filepath.Join(GetCLIConfigDir(), "config.toml")
}

// LoadCLIConfig 加载CLI配置
func LoadCLIConfig(configPath string) (*CLIConfig, error) {
	v := viper.New()

	// 设置默认值
	v.SetDefault("server.address", "")
	v.SetDefault("tls.insecure-skip-tls-verify", false)
	v.SetDefault("tls.ca-cert", "")

	if configPath != "" {
		// 使用指定路径
		dir := filepath.Dir(configPath)
		fileName := filepath.Base(configPath)
		ext := filepath.Ext(fileName)
		if ext == "" {
			v.SetConfigName("config")
			v.SetConfigType("toml")
			v.AddConfigPath(dir)
		} else {
			v.SetConfigName(fileName[:len(fileName)-len(ext)])
			v.SetConfigType(ext[1:])
			v.AddConfigPath(dir)
		}
	} else {
		// 搜索默认路径
		v.SetConfigName("config")
		v.SetConfigType("toml")

		// 用户家目录优先
		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(filepath.Join(home, ".fastdp-orbit"))
		}
		// 系统配置目录
		v.AddConfigPath(DefaultConfigDir)
	}

	// 读取配置
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在，返回默认配置
			return DefaultCLIConfig(), nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg CLIConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}

// SaveCLIConfig 保存CLI配置
func SaveCLIConfig(cfg *CLIConfig, configPath string) error {
	if configPath == "" {
		configPath = GetCLIConfigPath()
	}

	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	v := viper.New()
	v.Set("server.address", cfg.Server.Address)
	v.Set("tls.insecure-skip-tls-verify", cfg.TLS.InsecureSkipTLSVerify)
	v.Set("tls.ca-cert", cfg.TLS.CACert)
	v.Set("auth.access-token", cfg.Auth.AccessToken)
	v.Set("auth.refresh-token", cfg.Auth.RefreshToken)
	v.Set("auth.expires-at", cfg.Auth.ExpiresAt)
	v.Set("auth.user", cfg.Auth.User)

	v.SetConfigFile(configPath)
	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("保存配置文件失败: %w", err)
	}

	// 设置文件权限（仅用户可读写）
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("设置配置文件权限失败: %w", err)
	}

	return nil
}
