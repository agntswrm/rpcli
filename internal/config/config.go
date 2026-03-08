package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIKey string `json:"api_key,omitempty"`
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "rpcli"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return &Config{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &Config{}, nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{}, nil
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ResolveAPIKey returns the API key with priority: flag > env > config file.
func ResolveAPIKey(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("RUNPOD_API_KEY"); env != "" {
		return env
	}
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.APIKey
}

// MaskKey returns a masked version of the API key for display.
func MaskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

// Show returns config info as a map for display.
func Show(flagKey string) map[string]any {
	key := ResolveAPIKey(flagKey)
	source := "none"
	if flagKey != "" {
		source = "flag"
	} else if os.Getenv("RUNPOD_API_KEY") != "" {
		source = "env"
	} else if key != "" {
		source = "config_file"
	}

	path, _ := configPath()

	result := map[string]any{
		"api_key":     MaskKey(key),
		"key_source":  source,
		"config_file": path,
	}

	if key == "" {
		result["api_key"] = ""
		result["status"] = "not_configured"
	} else {
		result["status"] = "configured"
	}

	return result
}

// SetKey saves the API key to the config file.
func SetKey(key string) error {
	if key == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	cfg, _ := Load()
	cfg.APIKey = key
	return Save(cfg)
}
