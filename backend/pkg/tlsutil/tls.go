package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/pkg/logger"
	"os"

	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
)

// LoadServerTLSCredentials 加载服务端TLS凭证（双向TLS）
func LoadServerTLSCredentials(cfg config.TLSConfig) (credentials.TransportCredentials, error) {

	// 加载服务端证书和私钥
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		logger.Error("加载服务端证书失败", zap.Error(err))
		return nil, err
	}

	// 加载CA证书（用于验证客户端）
	certPool := x509.NewCertPool()
	if cfg.CAFile != "" {
		ca, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			logger.Error("加载CA证书失败", zap.Error(err))
			return nil, err
		}
		if !certPool.AppendCertsFromPEM(ca) {
			logger.Error("CA证书解析失败")
			return nil, err
		}
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS12,
	}

	return credentials.NewTLS(tlsConfig), nil
}

// LoadClientTLSCredentials 加载客户端TLS凭证（双向TLS）
func LoadClientTLSCredentials(cfg config.TLSConfig) (credentials.TransportCredentials, error) {
	// 加载客户端证书和私钥
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		logger.Error("加载客户端证书失败", zap.Error(err))
		return nil, err
	}

	// 加载CA证书（用于验证服务端）
	certPool := x509.NewCertPool()
	if cfg.CAFile != "" {
		ca, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			logger.Error("加载CA证书失败", zap.Error(err))
			return nil, err
		}
		if !certPool.AppendCertsFromPEM(ca) {
			logger.Error("CA证书解析失败")
			return nil, err
		}
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            certPool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // 跳过主机名验证（证书不绑定IP）
	}

	return credentials.NewTLS(tlsConfig), nil
}

// LoadServerHTTPTLSCredentials 加载服务端HTTP TLS凭证（单向TLS）
func LoadServerHTTPTLSCredentials(cfg config.TLSConfig) (*tls.Config, error) {

	// 加载证书和私钥
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		logger.Error("加载HTTP TLS证书失败", zap.Error(err))
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return tlsConfig, nil
}
