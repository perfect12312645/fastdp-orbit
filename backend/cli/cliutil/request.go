package cliutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"fastdp-orbit/backend/config"
)

// NewRequest 创建带认证的 HTTP 请求
// 如果配置中有 token，自动添加 Authorization: Bearer 头
func NewRequest(cfg *config.CLIConfig, method, apiPath string, body any) (*http.Request, error) {
	address := strings.TrimRight(cfg.Server.Address, "/")
	url := address + "/" + strings.TrimLeft(apiPath, "/")

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 自动注入 token
	if cfg.Auth.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Auth.Token)
	}

	return req, nil
}

// Do 发送 HTTP 请求并解析 JSON 响应
// result 为 nil 时仅检查 HTTP 状态码
func Do(client *http.Client, req *http.Request, result any) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 尝试解析统一响应格式 {code, message, data}
	var apiResp struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Code != 0 {
		if apiResp.Message != "" {
			return fmt.Errorf("服务端返回错误: %s", apiResp.Message)
		}
		return fmt.Errorf("服务端返回错误码: %d", apiResp.Code)
	}

	if result != nil {
		// 优先从 data 字段解析，否则直接解析整个响应
		if apiResp.Data != nil {
			if err := json.Unmarshal(apiResp.Data, result); err != nil {
				return fmt.Errorf("解析响应数据失败: %w", err)
			}
		} else {
			if err := json.Unmarshal(body, result); err != nil {
				return fmt.Errorf("解析响应失败: %w", err)
			}
		}
	}

	return nil
}
