package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/parser"
)

type Config struct {
	PreferredPackageManager *parser.PackageManager `toml:"preferred_package_manager,omitempty"`

	// MCP Settings
	MCPPort *int  `toml:"mcp_port,omitempty"`
	NoMCP   *bool `toml:"no_mcp,omitempty"`

	// Network Robustness Settings
	UseRobustNetworking *bool `toml:"use_robust_networking,omitempty"`

	// Proxy Settings
	ProxyPort     *int    `toml:"proxy_port,omitempty"`
	ProxyMode     *string `toml:"proxy_mode,omitempty"`
	ProxyURL      *string `toml:"proxy_url,omitempty"`
	StandardProxy *bool   `toml:"standard_proxy,omitempty"`
	NoProxy       *bool   `toml:"no_proxy,omitempty"`

	// AI Coder Settings
	AICoders *AICoderConfig `toml:"ai_coders,omitempty"`
}

// ConfigWithSources tracks where each config value comes from
type ConfigWithSources struct {
	Config
	Sources map[string]string // field name -> source file path
}

// getConfigPaths returns all potential config file paths in override order
func getConfigPaths() ([]string, error) {
	var paths []string

	// Start from current directory and walk up to root
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Walk up the directory tree
	dir := currentDir
	for {
		configPath := filepath.Join(dir, ".brum.toml")
		paths = append(paths, configPath)

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	// Add home directory config as fallback
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	paths = append(paths, filepath.Join(homeDir, ".brum.toml"))

	return paths, nil
}

// Load loads the configuration from disk with override chain
func Load() (*Config, error) {
	paths, err := getConfigPaths()
	if err != nil {
		return &Config{}, nil // Return empty config on error
	}

	// Start with empty config
	cfg := &Config{}

	// Load configs in reverse order (home -> root -> ... -> current)
	// so that more specific configs override general ones
	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue // Skip non-existent files
		}

		var fileCfg Config
		if _, err := toml.DecodeFile(path, &fileCfg); err != nil {
			continue // Skip invalid files
		}

		// Merge config - more specific values override general ones
		if fileCfg.PreferredPackageManager != nil {
			cfg.PreferredPackageManager = fileCfg.PreferredPackageManager
		}
		if fileCfg.MCPPort != nil {
			cfg.MCPPort = fileCfg.MCPPort
		}
		if fileCfg.NoMCP != nil {
			cfg.NoMCP = fileCfg.NoMCP
		}
		if fileCfg.UseRobustNetworking != nil {
			cfg.UseRobustNetworking = fileCfg.UseRobustNetworking
		}
		if fileCfg.ProxyPort != nil {
			cfg.ProxyPort = fileCfg.ProxyPort
		}
		if fileCfg.ProxyMode != nil {
			cfg.ProxyMode = fileCfg.ProxyMode
		}
		if fileCfg.ProxyURL != nil {
			cfg.ProxyURL = fileCfg.ProxyURL
		}
		if fileCfg.StandardProxy != nil {
			cfg.StandardProxy = fileCfg.StandardProxy
		}
		if fileCfg.NoProxy != nil {
			cfg.NoProxy = fileCfg.NoProxy
		}
		if fileCfg.AICoders != nil {
			cfg.AICoders = fileCfg.AICoders
		}
	}

	return cfg, nil
}

// LoadWithSources loads the configuration with source tracking
func LoadWithSources() (*ConfigWithSources, error) {
	paths, err := getConfigPaths()
	if err != nil {
		return &ConfigWithSources{Config: Config{}, Sources: make(map[string]string)}, nil
	}

	cfg := &ConfigWithSources{
		Config:  Config{},
		Sources: make(map[string]string),
	}

	// Load configs in reverse order (home -> root -> ... -> current)
	// so that more specific configs override general ones
	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue // Skip non-existent files
		}

		var fileCfg Config
		if _, err := toml.DecodeFile(path, &fileCfg); err != nil {
			continue // Skip invalid files
		}

		// Merge config and track sources - more specific values override general ones
		if fileCfg.PreferredPackageManager != nil {
			cfg.PreferredPackageManager = fileCfg.PreferredPackageManager
			cfg.Sources["preferred_package_manager"] = path
		}
		if fileCfg.MCPPort != nil {
			cfg.MCPPort = fileCfg.MCPPort
			cfg.Sources["mcp_port"] = path
		}
		if fileCfg.NoMCP != nil {
			cfg.NoMCP = fileCfg.NoMCP
			cfg.Sources["no_mcp"] = path
		}
		if fileCfg.UseRobustNetworking != nil {
			cfg.UseRobustNetworking = fileCfg.UseRobustNetworking
			cfg.Sources["use_robust_networking"] = path
		}
		if fileCfg.ProxyPort != nil {
			cfg.ProxyPort = fileCfg.ProxyPort
			cfg.Sources["proxy_port"] = path
		}
		if fileCfg.ProxyMode != nil {
			cfg.ProxyMode = fileCfg.ProxyMode
			cfg.Sources["proxy_mode"] = path
		}
		if fileCfg.ProxyURL != nil {
			cfg.ProxyURL = fileCfg.ProxyURL
			cfg.Sources["proxy_url"] = path
		}
		if fileCfg.StandardProxy != nil {
			cfg.StandardProxy = fileCfg.StandardProxy
			cfg.Sources["standard_proxy"] = path
		}
		if fileCfg.NoProxy != nil {
			cfg.NoProxy = fileCfg.NoProxy
			cfg.Sources["no_proxy"] = path
		}
		if fileCfg.AICoders != nil {
			cfg.AICoders = fileCfg.AICoders
			cfg.Sources["ai_coders"] = path
		}
	}

	return cfg, nil
}

// Save saves the configuration to current directory .brum.toml
func (c *Config) Save() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	path := filepath.Join(currentDir, ".brum.toml")

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return toml.NewEncoder(file).Encode(c)
}

// SaveToHome saves the configuration to home directory .brum.toml
func (c *Config) SaveToHome() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := filepath.Join(homeDir, ".brum.toml")

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return toml.NewEncoder(file).Encode(c)
}

// Helper functions to get config values with defaults

func (c *Config) GetMCPPort() int {
	if c.MCPPort != nil {
		return *c.MCPPort
	}
	return 7777 // default
}

func (c *Config) GetNoMCP() bool {
	if c.NoMCP != nil {
		return *c.NoMCP
	}
	return false // default
}

func (c *Config) GetUseRobustNetworking() bool {
	if c.UseRobustNetworking != nil {
		return *c.UseRobustNetworking
	}
	return false // default - disabled for safety
}

func (c *Config) GetProxyPort() int {
	if c.ProxyPort != nil {
		return *c.ProxyPort
	}
	return 19888 // default
}

func (c *Config) GetProxyMode() string {
	if c.ProxyMode != nil {
		return *c.ProxyMode
	}
	return "reverse" // default
}

func (c *Config) GetProxyURL() string {
	if c.ProxyURL != nil {
		return *c.ProxyURL
	}
	return "" // default
}

func (c *Config) GetStandardProxy() bool {
	if c.StandardProxy != nil {
		return *c.StandardProxy
	}
	return false // default
}

func (c *Config) GetNoProxy() bool {
	if c.NoProxy != nil {
		return *c.NoProxy
	}
	return false // default
}

func (c *Config) GetAICoderConfig() aicoder.AICoderConfig {
	aiConfig := c.AICoders
	if aiConfig == nil {
		aiConfig = &AICoderConfig{}
	}
	
	// Convert to simplified config for aicoder package
	return aicoder.AICoderConfig{
		MaxConcurrent:    aiConfig.GetMaxConcurrent(),
		WorkspaceBaseDir: aiConfig.GetWorkspaceBaseDir(),
		DefaultProvider:  aiConfig.GetDefaultProvider(),
		TimeoutMinutes:   aiConfig.GetTimeoutMinutes(),
	}
}

// DisplaySettingsWithSources returns a TOML-formatted string with source comments
func (c *ConfigWithSources) DisplaySettingsWithSources() string {
	var lines []string

	// Helper to shorten path for display
	shortenPath := func(path string) string {
		if strings.HasPrefix(path, os.Getenv("HOME")) {
			return strings.Replace(path, os.Getenv("HOME"), "~", 1)
		}
		wd, _ := os.Getwd()
		if strings.HasPrefix(path, wd) {
			rel, _ := filepath.Rel(wd, path)
			if !strings.HasPrefix(rel, "..") {
				return "./" + rel
			}
		}
		return path
	}

	lines = append(lines, "# Brummer Configuration")
	lines = append(lines, "# Generated by: brum --settings")
	lines = append(lines, "")

	// Package Manager
	if c.PreferredPackageManager != nil {
		if source, ok := c.Sources["preferred_package_manager"]; ok {
			lines = append(lines, fmt.Sprintf("# Source: %s", shortenPath(source)))
		}
		lines = append(lines, fmt.Sprintf("preferred_package_manager = \"%s\"", *c.PreferredPackageManager))
		lines = append(lines, "")
	}

	// MCP Settings
	lines = append(lines, "# MCP (Model Context Protocol) Settings")
	if c.MCPPort != nil {
		if source, ok := c.Sources["mcp_port"]; ok {
			lines = append(lines, fmt.Sprintf("# Source: %s", shortenPath(source)))
		}
		lines = append(lines, fmt.Sprintf("mcp_port = %d", *c.MCPPort))
	} else {
		lines = append(lines, "# mcp_port = 7777  # default")
	}

	if c.NoMCP != nil {
		if source, ok := c.Sources["no_mcp"]; ok {
			lines = append(lines, fmt.Sprintf("# Source: %s", shortenPath(source)))
		}
		lines = append(lines, fmt.Sprintf("no_mcp = %t", *c.NoMCP))
	} else {
		lines = append(lines, "# no_mcp = false  # default")
	}
	lines = append(lines, "")

	// Proxy Settings
	lines = append(lines, "# Proxy Server Settings")
	if c.ProxyPort != nil {
		if source, ok := c.Sources["proxy_port"]; ok {
			lines = append(lines, fmt.Sprintf("# Source: %s", shortenPath(source)))
		}
		lines = append(lines, fmt.Sprintf("proxy_port = %d", *c.ProxyPort))
	} else {
		lines = append(lines, "# proxy_port = 19888  # default")
	}

	if c.ProxyMode != nil {
		if source, ok := c.Sources["proxy_mode"]; ok {
			lines = append(lines, fmt.Sprintf("# Source: %s", shortenPath(source)))
		}
		lines = append(lines, fmt.Sprintf("proxy_mode = \"%s\"", *c.ProxyMode))
	} else {
		lines = append(lines, "# proxy_mode = \"reverse\"  # default")
	}

	if c.ProxyURL != nil && *c.ProxyURL != "" {
		if source, ok := c.Sources["proxy_url"]; ok {
			lines = append(lines, fmt.Sprintf("# Source: %s", shortenPath(source)))
		}
		lines = append(lines, fmt.Sprintf("proxy_url = \"%s\"", *c.ProxyURL))
	} else {
		lines = append(lines, "# proxy_url = \"\"  # default (auto-detect)")
	}

	if c.StandardProxy != nil {
		if source, ok := c.Sources["standard_proxy"]; ok {
			lines = append(lines, fmt.Sprintf("# Source: %s", shortenPath(source)))
		}
		lines = append(lines, fmt.Sprintf("standard_proxy = %t", *c.StandardProxy))
	} else {
		lines = append(lines, "# standard_proxy = false  # default")
	}

	if c.NoProxy != nil {
		if source, ok := c.Sources["no_proxy"]; ok {
			lines = append(lines, fmt.Sprintf("# Source: %s", shortenPath(source)))
		}
		lines = append(lines, fmt.Sprintf("no_proxy = %t", *c.NoProxy))
	} else {
		lines = append(lines, "# no_proxy = false  # default")
	}

	return strings.Join(lines, "\n")
}
