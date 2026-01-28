package services

import (
	"strings"
)

type OpenAISettingsService struct {
	relayAddr string
}

func NewOpenAISettingsService(relayAddr string) *OpenAISettingsService {
	return &OpenAISettingsService{relayAddr: relayAddr}
}

func (oss *OpenAISettingsService) ProxyStatus() (ClaudeProxyStatus, error) {
	// 目前传统 API 转发暂不支持自动配置本地 CLI（如无官方 OpenAI CLI 广泛使用）
	// 仅返回转发地址作为参考
	return ClaudeProxyStatus{Enabled: false, BaseURL: oss.baseURL()}, nil
}

func (oss *OpenAISettingsService) EnableProxy() error {
	// 占位实现
	return nil
}

func (oss *OpenAISettingsService) DisableProxy() error {
	// 占位实现
	return nil
}

func (oss *OpenAISettingsService) baseURL() string {
	addr := strings.TrimSpace(oss.relayAddr)
	if addr == "" {
		addr = ":18100"
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	return host
}
