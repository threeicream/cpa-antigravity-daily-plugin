package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type pluginConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

var pluginConfigState struct {
	sync.RWMutex
	value pluginConfig
}

func configure(raw []byte) error {
	var req lifecycleRequest
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			return fmt.Errorf("decode plugin lifecycle request: %w", err)
		}
	}

	cfg := pluginConfig{}
	if len(req.ConfigYAML) > 0 {
		if err := yaml.Unmarshal(req.ConfigYAML, &cfg); err != nil {
			return fmt.Errorf("decode plugin config: %w", err)
		}
	}
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)

	pluginConfigState.Lock()
	pluginConfigState.value = cfg
	pluginConfigState.Unlock()
	return nil
}

func loadedPluginConfig() pluginConfig {
	pluginConfigState.RLock()
	cfg := pluginConfigState.value
	pluginConfigState.RUnlock()
	return cfg
}

func requirePluginConfig() (pluginConfig, error) {
	cfg := loadedPluginConfig()
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return pluginConfig{}, fmt.Errorf("plugin config requires client_id and client_secret")
	}
	return cfg, nil
}
