package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/standardbeagle/brummer/internal/parser"
)

type Config struct {
	PreferredPackageManager *parser.PackageManager `json:"preferredPackageManager,omitempty"`
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".brummer")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return &Config{}, nil // Return empty config on error
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil // Return empty config if file doesn't exist
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
