package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validate validates the AI coder configuration
func (c *AICoderConfig) Validate() error {
	if c == nil {
		return nil // nil config is valid, will use defaults
	}

	var errs []string

	// Validate basic settings
	if c.MaxConcurrent != nil && *c.MaxConcurrent <= 0 {
		errs = append(errs, "max_concurrent must be positive")
	}
	if c.MaxConcurrent != nil && *c.MaxConcurrent > 10 {
		errs = append(errs, "max_concurrent should not exceed 10 for system stability")
	}

	if c.WorkspaceBaseDir != nil && *c.WorkspaceBaseDir == "" {
		errs = append(errs, "workspace_base_dir cannot be empty")
	}

	if c.TimeoutMinutes != nil && *c.TimeoutMinutes <= 0 {
		errs = append(errs, "timeout_minutes must be positive")
	}

	if c.CleanupAfterHours != nil && *c.CleanupAfterHours <= 0 {
		errs = append(errs, "cleanup_after_hours must be positive")
	}

	// Validate default provider exists
	if c.DefaultProvider != nil {
		if c.Providers != nil {
			if _, exists := c.Providers[*c.DefaultProvider]; !exists {
				// Check if it's in default providers
				defaultProviders := GetDefaultProviderConfigs()
				if _, exists := defaultProviders[*c.DefaultProvider]; !exists {
					errs = append(errs, fmt.Sprintf("default_provider '%s' not found in providers configuration", *c.DefaultProvider))
				}
			}
		}
	}

	// Validate providers
	if c.Providers != nil {
		for name, provider := range c.Providers {
			if err := provider.Validate(name); err != nil {
				errs = append(errs, fmt.Sprintf("provider '%s': %v", name, err))
			}
		}
	}

	// Validate resource limits
	if c.ResourceLimits != nil {
		if err := c.ResourceLimits.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("resource_limits: %v", err))
		}
	}

	// Validate workspace settings
	if c.WorkspaceSettings != nil {
		if err := c.WorkspaceSettings.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("workspace: %v", err))
		}
	}

	// Validate logging config
	if c.LoggingConfig != nil {
		if err := c.LoggingConfig.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("logging: %v", err))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// Validate validates a provider configuration
func (p *ProviderConfig) Validate(providerName string) error {
	if p == nil {
		return nil // nil provider config is valid, will use defaults
	}

	var errs []string

	// Validate API key configuration (except for local and mock providers)
	if providerName != "local" && providerName != "mock" {
		if p.APIKeyEnv != nil && *p.APIKeyEnv == "" {
			errs = append(errs, "api_key_env cannot be empty for non-local providers")
		}

		// Check if environment variable is set
		if p.APIKeyEnv != nil && os.Getenv(*p.APIKeyEnv) == "" {
			// This is a warning, not an error - the env var might be set later
			// errs = append(errs, fmt.Sprintf("environment variable %s is not set", *p.APIKeyEnv))
		}
	}

	// Validate model
	if p.Model != nil && *p.Model == "" {
		errs = append(errs, "model cannot be empty")
	}

	// Validate limits
	if p.MaxTokens != nil {
		if *p.MaxTokens <= 0 {
			errs = append(errs, "max_tokens must be positive")
		}
		if *p.MaxTokens > 50000 {
			errs = append(errs, "max_tokens exceeds reasonable limit (50000)")
		}
	}

	if p.Temperature != nil {
		if *p.Temperature < 0 || *p.Temperature > 2 {
			errs = append(errs, "temperature must be between 0 and 2")
		}
	}

	if p.RequestTimeout != nil && *p.RequestTimeout <= 0 {
		errs = append(errs, "request_timeout_seconds must be positive")
	}

	// Validate rate limits
	if p.RateLimit != nil {
		if err := p.RateLimit.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("rate_limit: %v", err))
		}
	}

	// Validate base URL for local providers
	if providerName == "local" && p.BaseURL != nil && *p.BaseURL == "" {
		errs = append(errs, "base_url is required for local providers")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// Validate validates rate limit configuration
func (r *RateLimitConfig) Validate() error {
	if r == nil {
		return nil
	}

	var errs []string

	if r.RequestsPerMinute != nil && *r.RequestsPerMinute <= 0 {
		errs = append(errs, "requests_per_minute must be positive")
	}

	if r.TokensPerMinute != nil && *r.TokensPerMinute <= 0 {
		errs = append(errs, "tokens_per_minute must be positive")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// Validate validates resource limits
func (r *ResourceLimits) Validate() error {
	if r == nil {
		return nil
	}

	var errs []string

	if r.MaxMemoryMB != nil {
		if *r.MaxMemoryMB <= 0 {
			errs = append(errs, "max_memory_mb must be positive")
		}
		if *r.MaxMemoryMB > 8192 {
			errs = append(errs, "max_memory_mb exceeds reasonable limit (8GB)")
		}
	}

	if r.MaxDiskSpaceMB != nil && *r.MaxDiskSpaceMB <= 0 {
		errs = append(errs, "max_disk_space_mb must be positive")
	}

	if r.MaxCPUPercent != nil {
		if *r.MaxCPUPercent <= 0 || *r.MaxCPUPercent > 100 {
			errs = append(errs, "max_cpu_percent must be between 1 and 100")
		}
	}

	if r.MaxProcesses != nil && *r.MaxProcesses <= 0 {
		errs = append(errs, "max_processes must be positive")
	}

	if r.MaxFilesPerCoder != nil && *r.MaxFilesPerCoder <= 0 {
		errs = append(errs, "max_files_per_coder must be positive")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// Validate validates workspace settings
func (w *WorkspaceSettings) Validate() error {
	if w == nil {
		return nil
	}

	var errs []string

	// Validate allowed extensions format
	for _, ext := range w.AllowedExtensions {
		if !strings.HasPrefix(ext, ".") {
			errs = append(errs, fmt.Sprintf("allowed extension '%s' must start with '.'", ext))
		}
	}

	// Validate forbidden paths are absolute
	for _, path := range w.ForbiddenPaths {
		if !filepath.IsAbs(path) {
			errs = append(errs, fmt.Sprintf("forbidden path '%s' must be absolute", path))
		}
	}

	if w.MaxFileSize != nil {
		if *w.MaxFileSize <= 0 {
			errs = append(errs, "max_file_size_mb must be positive")
		}
		if *w.MaxFileSize > 100 {
			errs = append(errs, "max_file_size_mb should not exceed 100MB for performance")
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// Validate validates logging configuration
func (l *LoggingConfig) Validate() error {
	if l == nil {
		return nil
	}

	var errs []string

	// Validate log level
	if l.Level != nil {
		validLevels := map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		}
		if !validLevels[*l.Level] {
			errs = append(errs, fmt.Sprintf("invalid log level '%s' (must be debug, info, warn, or error)", *l.Level))
		}
	}

	if l.RotateSize != nil && *l.RotateSize <= 0 {
		errs = append(errs, "rotate_size_mb must be positive")
	}

	if l.KeepRotations != nil && *l.KeepRotations < 0 {
		errs = append(errs, "keep_rotations must be non-negative")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// ValidateWorkspaceDirectory validates and ensures the workspace directory exists
func (c *AICoderConfig) ValidateWorkspaceDirectory() error {
	workspaceDir := c.GetWorkspaceBaseDir()

	// Expand tilde in path
	if strings.HasPrefix(workspaceDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		workspaceDir = filepath.Join(homeDir, workspaceDir[2:])
	}

	// Check if directory exists, create if it doesn't
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			return fmt.Errorf("cannot create workspace directory %s: %w", workspaceDir, err)
		}
	}

	// Test write permissions
	testFile := filepath.Join(workspaceDir, ".brummer-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("workspace directory %s is not writable: %w", workspaceDir, err)
	}
	os.Remove(testFile)

	return nil
}

// GetExpandedWorkspaceDir returns the workspace directory with tilde expanded
func (c *AICoderConfig) GetExpandedWorkspaceDir() (string, error) {
	workspaceDir := c.GetWorkspaceBaseDir()

	// Expand tilde in path
	if strings.HasPrefix(workspaceDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		workspaceDir = filepath.Join(homeDir, workspaceDir[2:])
	}

	return workspaceDir, nil
}
