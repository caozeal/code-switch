package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Provider struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	APIURL  string `json:"apiUrl"`
	APIKey  string `json:"apiKey"`
	Site    string `json:"officialSite"`
	Icon    string `json:"icon"`
	Tint    string `json:"tint"`
	Accent  string `json:"accent"`
	Enabled bool   `json:"enabled"`
}

type providerEnvelope struct {
	Providers []Provider `json:"providers"`
}

type ProviderService struct {
	mu sync.Mutex
}

func NewProviderService() *ProviderService {
	return &ProviderService{}
}

func (ps *ProviderService) Start() error { return nil }
func (ps *ProviderService) Stop() error  { return nil }

func providerFilePath(kind string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".code-switch")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	var filename string
	switch strings.ToLower(kind) {
	case "claude", "claude-code", "claude_code":
		filename = "claude-code.json"
	case "codex":
		filename = "codex.json"
	default:
		return "", fmt.Errorf("unknown provider type: %s", kind)
	}
	return filepath.Join(dir, filename), nil
}

func (ps *ProviderService) SaveProviders(kind string, providers []Provider) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	path, err := providerFilePath(kind)
	if err != nil {
		return err
	}

	existingProviders, err := ps.LoadProviders(kind)
	if err != nil {
		return err
	}
	nameByID := make(map[int]string, len(existingProviders))
	for _, p := range existingProviders {
		nameByID[p.ID] = p.Name
	}
	for _, p := range providers {
		if oldName, ok := nameByID[p.ID]; ok && oldName != p.Name {
			return fmt.Errorf("provider id %d 的 name 不可修改", p.ID)
		}
	}

	data, err := json.MarshalIndent(providerEnvelope{Providers: providers}, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (ps *ProviderService) LoadProviders(kind string) ([]Provider, error) {
	path, err := providerFilePath(kind)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var envelope providerEnvelope
	if len(data) == 0 {
		return []Provider{}, nil
	}

	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return envelope.Providers, nil
}
