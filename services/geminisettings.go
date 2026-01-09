package services

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	geminiSettingsDir      = ".gemini"
	geminiSettingsFileName = ".env"
	geminiBackupFileName   = ".env.code-switch.bak"
	geminiAuthTokenValue   = "code-switch"
)

type GeminiSettingsService struct {
	relayAddr string
}

func NewGeminiSettingsService(relayAddr string) *GeminiSettingsService {
	return &GeminiSettingsService{relayAddr: relayAddr}
}

func (gss *GeminiSettingsService) ProxyStatus() (ClaudeProxyStatus, error) {
	status := ClaudeProxyStatus{Enabled: false, BaseURL: gss.baseURL()}
	settingsPath, _, err := gss.paths()
	if err != nil {
		return status, err
	}
	file, err := os.Open(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status, nil
		}
		return status, err
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// 移除引号
			val = strings.Trim(val, `"'`)
			env[key] = val
		}
	}

	baseURL := gss.baseURL()
	enabled := strings.EqualFold(env["GEMINI_API_KEY"], geminiAuthTokenValue) &&
		strings.EqualFold(env["GOOGLE_GEMINI_BASE_URL"], baseURL)
	status.Enabled = enabled
	return status, nil
}

func (gss *GeminiSettingsService) EnableProxy() error {
	settingsPath, backupPath, err := gss.paths()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(settingsPath); err == nil {
		content, readErr := os.ReadFile(settingsPath)
		if readErr != nil {
			return readErr
		}
		if err := os.WriteFile(backupPath, content, 0o600); err != nil {
			return err
		}
	}

	// 为简单起见且由于有备份，我们直接覆盖为代理配置
	// 这样可以确保配置纯净且符合代理要求
	content := fmt.Sprintf("GEMINI_API_KEY=%s\nGOOGLE_GEMINI_BASE_URL=%s\nGEMINI_MODEL=gemini-3-pro-preview\n", geminiAuthTokenValue, gss.baseURL())
	return os.WriteFile(settingsPath, []byte(content), 0o600)
}

func (gss *GeminiSettingsService) DisableProxy() error {
	settingsPath, backupPath, err := gss.paths()
	if err != nil {
		return err
	}
	if err := os.Remove(settingsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Rename(backupPath, settingsPath); err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return nil
}

func (gss *GeminiSettingsService) paths() (settingsPath string, backupPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, geminiSettingsDir)
	return filepath.Join(dir, geminiSettingsFileName), filepath.Join(dir, geminiBackupFileName), nil
}

func (gss *GeminiSettingsService) baseURL() string {
	addr := strings.TrimSpace(gss.relayAddr)
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
