package cliutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"fastdp-orbit/backend/config"
)

// NewHTTPClient 创建HTTP客户端（根据CLI配置处理TLS）
func NewHTTPClient(cfg *config.CLIConfig) (*http.Client, error) {
	client := &http.Client{}

	if cfg.TLS.InsecureSkipTLSVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		return client, nil
	}

	if cfg.TLS.CACert != "" {
		caCert, err := os.ReadFile(cfg.TLS.CACert)
		if err != nil {
			return nil, fmt.Errorf("读取CA证书失败: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("CA证书解析失败")
		}

		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}
	}

	return client, nil
}
