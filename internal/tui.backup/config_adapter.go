package tui

import (
	"os"
	"path/filepath"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/config"
)

// configAdapter adapts config.Config to aicoder.Config interface
type configAdapter struct {
	cfg *config.Config
}

func (c *configAdapter) GetAICoderConfig() aicoder.AICoderConfig {
	if c.cfg == nil || c.cfg.AICoders == nil {
		// Return defaults if no config
		return aicoder.AICoderConfig{
			MaxConcurrent:    3,
			WorkspaceBaseDir: filepath.Join(os.Getenv("HOME"), ".brummer", "ai-coders"),
			DefaultProvider:  "claude",
			TimeoutMinutes:   30,
		}
	}

	aiCfg := c.cfg.AICoders

	// Helper function to safely dereference pointers with defaults
	intVal := func(ptr *int, def int) int {
		if ptr != nil {
			return *ptr
		}
		return def
	}
	stringVal := func(ptr *string, def string) string {
		if ptr != nil {
			return *ptr
		}
		return def
	}

	return aicoder.AICoderConfig{
		MaxConcurrent:    intVal(aiCfg.MaxConcurrent, 3),
		WorkspaceBaseDir: stringVal(aiCfg.WorkspaceBaseDir, filepath.Join(os.Getenv("HOME"), ".brummer", "ai-coders")),
		DefaultProvider:  stringVal(aiCfg.DefaultProvider, "claude"),
		TimeoutMinutes:   intVal(aiCfg.TimeoutMinutes, 30),
	}
}

func (c *configAdapter) GetProviderConfigs() map[string]*aicoder.ProviderConfig {
	result := make(map[string]*aicoder.ProviderConfig)

	if c.cfg == nil || c.cfg.AICoders == nil || c.cfg.AICoders.Providers == nil {
		return result
	}

	// Helper function to safely dereference string pointers
	stringVal := func(ptr *string, def string) string {
		if ptr != nil {
			return *ptr
		}
		return def
	}

	// Convert from config.ProviderConfig to aicoder.ProviderConfig
	for name, provider := range c.cfg.AICoders.Providers {
		if provider == nil {
			continue
		}

		aiProvider := &aicoder.ProviderConfig{}

		// Convert CLI tool config if present
		if provider.CLITool != nil {
			aiProvider.CLITool = &aicoder.CLIToolConfig{
				Command:     stringVal(provider.CLITool.Command, ""),
				BaseArgs:    provider.CLITool.BaseArgs,
				FlagMapping: provider.CLITool.FlagMapping,
				WorkingDir:  stringVal(provider.CLITool.WorkingDir, ""),
				Environment: provider.CLITool.Environment,
			}
		}

		// Copy other fields with pointer dereferencing
		if provider.Model != nil {
			aiProvider.Model = *provider.Model
		}
		if provider.APIKeyEnv != nil {
			aiProvider.APIKeyEnv = *provider.APIKeyEnv
		}
		if provider.MaxTokens != nil {
			aiProvider.MaxTokens = *provider.MaxTokens
		}
		if provider.Temperature != nil {
			aiProvider.Temperature = *provider.Temperature
		}

		result[name] = aiProvider
	}

	return result
}
