package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// AppConfig represents the configuration for a single application.
type AppConfig struct {
	Name string            `json:"name"`
	Port int               `json:"port"`
	Env  map[string]string `json:"env"`
}

// Config represents the global configuration for VPSMyth.
type Config struct {
	Apps []AppConfig `json:"apps"`
}

// LoadConfig reads and parses a JSON configuration file from the given path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &cfg, nil
}

// GetAppConfig retrieves the configuration for a specific application by its name.
func (c *Config) GetAppConfig(name string) (*AppConfig, bool) {
	for _, app := range c.Apps {
		if app.Name == name {
			return &app, true
		}
	}
	return nil, false
}
