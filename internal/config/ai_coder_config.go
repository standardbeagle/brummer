package config

// AICoderConfig holds all configuration for AI coder functionality
type AICoderConfig struct {
	Enabled             *bool                      `toml:"enabled,omitempty"`
	MaxConcurrent       *int                       `toml:"max_concurrent,omitempty"`
	WorkspaceBaseDir    *string                    `toml:"workspace_base_dir,omitempty"`
	DefaultProvider     *string                    `toml:"default_provider,omitempty"`
	TimeoutMinutes      *int                       `toml:"timeout_minutes,omitempty"`
	AutoCleanup         *bool                      `toml:"auto_cleanup,omitempty"`
	CleanupAfterHours   *int                       `toml:"cleanup_after_hours,omitempty"`
	Providers           map[string]*ProviderConfig `toml:"providers,omitempty"`
	ResourceLimits      *ResourceLimits            `toml:"resource_limits,omitempty"`
	WorkspaceSettings   *WorkspaceSettings         `toml:"workspace,omitempty"`
	LoggingConfig       *LoggingConfig             `toml:"logging,omitempty"`
}

// ProviderConfig holds configuration for a specific AI provider
type ProviderConfig struct {
	APIKeyEnv       *string            `toml:"api_key_env,omitempty"`
	Model           *string            `toml:"model,omitempty"`
	BaseURL         *string            `toml:"base_url,omitempty"`
	MaxTokens       *int               `toml:"max_tokens,omitempty"`
	Temperature     *float64           `toml:"temperature,omitempty"`
	RequestTimeout  *int               `toml:"request_timeout_seconds,omitempty"`
	RateLimit       *RateLimitConfig   `toml:"rate_limit,omitempty"`
	CustomHeaders   map[string]string  `toml:"custom_headers,omitempty"`
	
	// CLI Tool specific configuration
	CLITool         *CLIToolConfig     `toml:"cli_tool,omitempty"`
}

// CLIToolConfig represents configuration for CLI-based AI tools
type CLIToolConfig struct {
	Command     *string            `toml:"command,omitempty"`     // e.g., "aider"
	BaseArgs    []string           `toml:"base_args,omitempty"`   // e.g., ["--yes", "--no-auto-commits"]
	FlagMapping map[string]string  `toml:"flag_mapping,omitempty"` // e.g., {"model": "--model", "max_tokens": "--max-tokens"}
	WorkingDir  *string            `toml:"working_dir,omitempty"`
	Environment map[string]string  `toml:"environment,omitempty"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute *int `toml:"requests_per_minute,omitempty"`
	TokensPerMinute   *int `toml:"tokens_per_minute,omitempty"`
}

// ResourceLimits holds resource limit configuration
type ResourceLimits struct {
	MaxMemoryMB        *int `toml:"max_memory_mb,omitempty"`
	MaxDiskSpaceMB     *int `toml:"max_disk_space_mb,omitempty"`
	MaxCPUPercent      *int `toml:"max_cpu_percent,omitempty"`
	MaxProcesses       *int `toml:"max_processes,omitempty"`
	MaxFilesPerCoder   *int `toml:"max_files_per_coder,omitempty"`
}

// WorkspaceSettings holds workspace configuration
type WorkspaceSettings struct {
	Template           *string   `toml:"template,omitempty"`
	GitIgnoreRules     []string  `toml:"gitignore_rules,omitempty"`
	AllowedExtensions  []string  `toml:"allowed_extensions,omitempty"`
	ForbiddenPaths     []string  `toml:"forbidden_paths,omitempty"`
	MaxFileSize        *int      `toml:"max_file_size_mb,omitempty"`
	BackupEnabled      *bool     `toml:"backup_enabled,omitempty"`
}

// LoggingConfig holds logging configuration for AI coders
type LoggingConfig struct {
	Level           *string `toml:"level,omitempty"`
	OutputFile      *string `toml:"output_file,omitempty"`
	RotateSize      *int    `toml:"rotate_size_mb,omitempty"`
	KeepRotations   *int    `toml:"keep_rotations,omitempty"`
	IncludeAIOutput *bool   `toml:"include_ai_output,omitempty"`
}

// Helper methods to get values with defaults

func (c *AICoderConfig) GetEnabled() bool {
	if c == nil || c.Enabled == nil {
		return true // default enabled
	}
	return *c.Enabled
}

func (c *AICoderConfig) GetMaxConcurrent() int {
	if c == nil || c.MaxConcurrent == nil {
		return 3 // default
	}
	return *c.MaxConcurrent
}

func (c *AICoderConfig) GetWorkspaceBaseDir() string {
	if c == nil || c.WorkspaceBaseDir == nil {
		return "~/.brummer/ai-coders" // default
	}
	return *c.WorkspaceBaseDir
}

func (c *AICoderConfig) GetDefaultProvider() string {
	if c == nil || c.DefaultProvider == nil {
		return "claude" // default to claude
	}
	return *c.DefaultProvider
}

func (c *AICoderConfig) GetTimeoutMinutes() int {
	if c == nil || c.TimeoutMinutes == nil {
		return 30 // default
	}
	return *c.TimeoutMinutes
}

func (c *AICoderConfig) GetAutoCleanup() bool {
	if c == nil || c.AutoCleanup == nil {
		return true // default
	}
	return *c.AutoCleanup
}

func (c *AICoderConfig) GetCleanupAfterHours() int {
	if c == nil || c.CleanupAfterHours == nil {
		return 24 // default
	}
	return *c.CleanupAfterHours
}

func (c *AICoderConfig) GetResourceLimits() ResourceLimits {
	if c == nil || c.ResourceLimits == nil {
		return ResourceLimits{
			MaxMemoryMB:      intPtr(512),
			MaxDiskSpaceMB:   intPtr(1024),
			MaxCPUPercent:    intPtr(50),
			MaxProcesses:     intPtr(5),
			MaxFilesPerCoder: intPtr(100),
		}
	}
	return *c.ResourceLimits
}

func (c *AICoderConfig) GetWorkspaceSettings() WorkspaceSettings {
	if c == nil || c.WorkspaceSettings == nil {
		return WorkspaceSettings{
			Template:          stringPtr("basic"),
			GitIgnoreRules:    []string{"node_modules/", ".env", "*.log"},
			AllowedExtensions: []string{".go", ".js", ".ts", ".py", ".md", ".json", ".yaml", ".toml"},
			ForbiddenPaths:    []string{"/etc", "/var", "/sys", "/proc"},
			MaxFileSize:       intPtr(10),
			BackupEnabled:     boolPtr(true),
		}
	}
	return *c.WorkspaceSettings
}

func (c *AICoderConfig) GetLoggingConfig() LoggingConfig {
	if c == nil || c.LoggingConfig == nil {
		return LoggingConfig{
			Level:           stringPtr("info"),
			OutputFile:      stringPtr("ai-coders.log"),
			RotateSize:      intPtr(50),
			KeepRotations:   intPtr(5),
			IncludeAIOutput: boolPtr(false),
		}
	}
	return *c.LoggingConfig
}

// Provider config helpers

func (p *ProviderConfig) GetAPIKeyEnv() string {
	if p == nil || p.APIKeyEnv == nil {
		return ""
	}
	return *p.APIKeyEnv
}

func (p *ProviderConfig) GetModel() string {
	if p == nil || p.Model == nil {
		return ""
	}
	return *p.Model
}

func (p *ProviderConfig) GetBaseURL() string {
	if p == nil || p.BaseURL == nil {
		return ""
	}
	return *p.BaseURL
}

func (p *ProviderConfig) GetMaxTokens() int {
	if p == nil || p.MaxTokens == nil {
		return 4096 // default
	}
	return *p.MaxTokens
}

func (p *ProviderConfig) GetTemperature() float64 {
	if p == nil || p.Temperature == nil {
		return 0.7 // default
	}
	return *p.Temperature
}

func (p *ProviderConfig) GetRequestTimeout() int {
	if p == nil || p.RequestTimeout == nil {
		return 30 // default 30 seconds
	}
	return *p.RequestTimeout
}

func (p *ProviderConfig) GetRateLimit() RateLimitConfig {
	if p == nil || p.RateLimit == nil {
		return RateLimitConfig{
			RequestsPerMinute: intPtr(50),
			TokensPerMinute:   intPtr(150000),
		}
	}
	return *p.RateLimit
}

// Resource limit helpers

func (r *ResourceLimits) GetMaxMemoryMB() int {
	if r == nil || r.MaxMemoryMB == nil {
		return 512 // default
	}
	return *r.MaxMemoryMB
}

func (r *ResourceLimits) GetMaxDiskSpaceMB() int {
	if r == nil || r.MaxDiskSpaceMB == nil {
		return 1024 // default
	}
	return *r.MaxDiskSpaceMB
}

func (r *ResourceLimits) GetMaxCPUPercent() int {
	if r == nil || r.MaxCPUPercent == nil {
		return 50 // default
	}
	return *r.MaxCPUPercent
}

func (r *ResourceLimits) GetMaxProcesses() int {
	if r == nil || r.MaxProcesses == nil {
		return 5 // default
	}
	return *r.MaxProcesses
}

func (r *ResourceLimits) GetMaxFilesPerCoder() int {
	if r == nil || r.MaxFilesPerCoder == nil {
		return 100 // default
	}
	return *r.MaxFilesPerCoder
}

// Workspace settings helpers

func (w *WorkspaceSettings) GetTemplate() string {
	if w == nil || w.Template == nil {
		return "basic" // default
	}
	return *w.Template
}

func (w *WorkspaceSettings) GetMaxFileSize() int {
	if w == nil || w.MaxFileSize == nil {
		return 10 // default 10MB
	}
	return *w.MaxFileSize
}

func (w *WorkspaceSettings) GetBackupEnabled() bool {
	if w == nil || w.BackupEnabled == nil {
		return true // default
	}
	return *w.BackupEnabled
}

// Logging config helpers

func (l *LoggingConfig) GetLevel() string {
	if l == nil || l.Level == nil {
		return "info" // default
	}
	return *l.Level
}

func (l *LoggingConfig) GetOutputFile() string {
	if l == nil || l.OutputFile == nil {
		return "ai-coders.log" // default
	}
	return *l.OutputFile
}

func (l *LoggingConfig) GetRotateSize() int {
	if l == nil || l.RotateSize == nil {
		return 50 // default 50MB
	}
	return *l.RotateSize
}

func (l *LoggingConfig) GetKeepRotations() int {
	if l == nil || l.KeepRotations == nil {
		return 5 // default
	}
	return *l.KeepRotations
}

func (l *LoggingConfig) GetIncludeAIOutput() bool {
	if l == nil || l.IncludeAIOutput == nil {
		return false // default
	}
	return *l.IncludeAIOutput
}

// Rate limit helpers

func (r *RateLimitConfig) GetRequestsPerMinute() int {
	if r == nil || r.RequestsPerMinute == nil {
		return 50 // default
	}
	return *r.RequestsPerMinute
}

func (r *RateLimitConfig) GetTokensPerMinute() int {
	if r == nil || r.TokensPerMinute == nil {
		return 150000 // default
	}
	return *r.TokensPerMinute
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func float64Ptr(f float64) *float64 {
	return &f
}

// GetDefaultProviderConfigs returns default provider configurations
func GetDefaultProviderConfigs() map[string]*ProviderConfig {
	return map[string]*ProviderConfig{
		"claude": {
			APIKeyEnv:      stringPtr("ANTHROPIC_API_KEY"),
			Model:          stringPtr("claude-3-5-sonnet-20241022"),
			MaxTokens:      intPtr(4096),
			Temperature:    float64Ptr(0.7),
			RequestTimeout: intPtr(30),
			RateLimit: &RateLimitConfig{
				RequestsPerMinute: intPtr(50),
				TokensPerMinute:   intPtr(150000),
			},
		},
		"openai": {
			APIKeyEnv:      stringPtr("OPENAI_API_KEY"),
			Model:          stringPtr("gpt-4"),
			MaxTokens:      intPtr(4096),
			Temperature:    float64Ptr(0.7),
			RequestTimeout: intPtr(30),
			RateLimit: &RateLimitConfig{
				RequestsPerMinute: intPtr(60),
				TokensPerMinute:   intPtr(200000),
			},
		},
		"gemini": {
			APIKeyEnv:      stringPtr("GEMINI_API_KEY"),
			Model:          stringPtr("gemini-1.5-pro"),
			MaxTokens:      intPtr(8192),
			Temperature:    float64Ptr(0.7),
			RequestTimeout: intPtr(30),
			RateLimit: &RateLimitConfig{
				RequestsPerMinute: intPtr(15),
				TokensPerMinute:   intPtr(1000000),
			},
		},
		"terminal": {
			Model:          stringPtr("bash"),
			MaxTokens:      intPtr(1000000),
			Temperature:    float64Ptr(0.0),
			RequestTimeout: intPtr(300), // Longer timeout for shell commands
			RateLimit: &RateLimitConfig{
				RequestsPerMinute: intPtr(30),
				TokensPerMinute:   intPtr(1000000),
			},
		},
		"local": {
			BaseURL:        stringPtr("http://localhost:11434"),
			Model:          stringPtr("codellama"),
			MaxTokens:      intPtr(2048),
			Temperature:    float64Ptr(0.7),
			RequestTimeout: intPtr(60),
		},
		"aider": {
			CLITool: &CLIToolConfig{
				Command:  stringPtr("aider"),
				BaseArgs: []string{"--yes"},
				FlagMapping: map[string]string{
					"model":   "--model",
					"message": "--message",
				},
				WorkingDir: stringPtr("."),
			},
			MaxTokens:      intPtr(4096),
			Temperature:    float64Ptr(0.7),
			RequestTimeout: intPtr(300), // Longer timeout for interactive tools
		},
		"mock": {
			Model:          stringPtr("mock-model"),
			MaxTokens:      intPtr(1000),
			Temperature:    float64Ptr(0.5),
			RequestTimeout: intPtr(5),
		},
	}
}