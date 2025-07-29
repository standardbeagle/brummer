# Task: Configuration System Extensions for AI Coders
**Generated from Master Planning**: 2025-01-28
**Context Package**: `/requests/agentic-ai-coders/context/`
**Next Phase**: [subtasks-execute.md](../subtasks-execute.md)

## Task Sizing Assessment
**File Count**: 3 files - Within target range (3-7 files)
**Estimated Time**: 15 minutes - Within target (15-30min)
**Token Estimate**: 60k tokens - Well within target (<150k)
**Complexity Level**: 1 (Simple) - Configuration extension with established patterns
**Parallelization Benefit**: MEDIUM - Independent from core implementation details
**Atomicity Assessment**: ✅ ATOMIC - Complete configuration support for AI coders
**Boundary Analysis**: ✅ CLEAR - Extends existing config system with minimal changes

## Persona Assignment
**Persona**: Software Engineer (Configuration/DevOps)
**Expertise Required**: TOML configuration, Go struct tags, validation patterns
**Worktree**: `~/work/worktrees/agentic-ai-coders/03-config-system/`

## Context Summary
**Risk Level**: LOW (well-established configuration patterns)
**Integration Points**: Core AI coder service, existing configuration system
**Architecture Pattern**: Configuration Extension Pattern (from existing config)
**Similar Reference**: `internal/config/config.go` - TOML configuration patterns

### Codebase Context (from master analysis)
**Files in Scope**:
```yaml
read_files:   [internal/config/config.go, cmd/main.go]
modify_files: [internal/config/config.go]
create_files: [
  /internal/config/ai_coder_config.go,
  /internal/config/ai_coder_validation.go
]
# Total: 3 files (1 modify, 2 create) - minimal configuration extension
```

**Existing Patterns to Follow**:
- `internal/config/config.go` - TOML configuration structure and validation
- BurntSushi/toml package usage for configuration parsing
- Environment variable integration for sensitive values

**Dependencies Context**:
- `github.com/BurntSushi/toml v1.5.0` - TOML configuration parsing
- Standard library `os` package for environment variables
- Validation patterns for configuration sanity checks

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /internal/config/config.go              # Add AI coder config section
  - /internal/config/ai_coder_config.go     # AI coder configuration types
  - /internal/config/ai_coder_validation.go # Configuration validation logic

direct_dependencies: []                      # No other files require changes
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /cmd/main.go                           # Review for config initialization
  - /.brum.toml                            # Example configuration file
  - /internal/aicoder/manager.go           # Will consume configuration (future)

check_documentation:
  - /docs/configuration.md                 # Configuration documentation updates
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/tui/                         # TUI uses config but doesn't modify it
  - /internal/mcp/                         # MCP uses config but doesn't modify it  
  - /internal/process/                     # Process manager separate config area
  - /internal/proxy/                       # Proxy configuration separate
  - /internal/logs/                        # Logging configuration separate

ignore_search_patterns:
  - "**/testdata/**"                       # Test data files
  - "**/vendor/**"                         # Third-party dependencies
  - "**/*.example.*"                       # Example configuration files
```

**Boundary Analysis Results**:
- **Usage Count**: 1 existing file to extend
- **Scope Assessment**: LIMITED scope - isolated configuration extension
- **Impact Radius**: 1 file modified, 2 new files, clear boundaries

### External Context Sources (from master research)
**Primary Documentation**:
- [TOML Specification](https://toml.io/en/) - Configuration syntax and validation
- [Go Struct Tags](https://golang.org/ref/spec#Tag) - TOML struct tag usage
- [BurntSushi TOML](https://github.com/BurntSushi/toml) - Go TOML library documentation

**Standards Applied**:
- TOML naming conventions - snake_case for keys
- Environment variable naming - UPPER_CASE with underscores
- Configuration validation - early validation with helpful error messages

**Reference Implementation**:
- Existing config structure in `internal/config/config.go`
- Configuration validation patterns
- Environment variable integration for API keys

## Task Requirements
**Objective**: Extend existing configuration system with comprehensive AI coder settings

**Success Criteria**:
- [ ] AI coder configuration section added to main config struct
- [ ] Provider-specific configuration support (Claude, OpenAI, local models)
- [ ] Environment variable integration for API keys and sensitive values
- [ ] Configuration validation with clear error messages
- [ ] Default values for all configuration options
- [ ] TOML example configuration for user documentation
- [ ] Integration with existing configuration loading system

**Configuration Areas to Support**:
1. **Global AI Coder Settings** - max concurrent, workspace directory, timeouts
2. **Provider Configuration** - API keys, models, endpoints, limits
3. **Workspace Settings** - directory structure, security settings, cleanup
4. **Resource Limits** - memory, CPU, file system quotas
5. **Logging Configuration** - AI coder specific logging levels and outputs

**Validation Commands**:
```bash
# Configuration Integration Verification
grep -q "ai_coders" internal/config/config.go         # Config section exists
go build ./internal/config                            # Config package compiles
echo '[ai_coders]' | brum --validate-config          # TOML validation works
go test ./internal/config -v                          # Validation tests pass
```

## Implementation Specifications

### Main Configuration Integration
```go
// Addition to internal/config/config.go
type Config struct {
    // Existing configuration fields...
    Process  ProcessConfig  `toml:"process"`
    MCP      MCPConfig      `toml:"mcp"`
    Proxy    ProxyConfig    `toml:"proxy"`
    
    // Add AI coder configuration
    AICoders AICoderConfig  `toml:"ai_coders"`
}

// Update LoadConfig function to validate AI coder config
func LoadConfig(configPath string) (*Config, error) {
    // Existing loading logic...
    
    // Validate AI coder configuration
    if err := config.AICoders.Validate(); err != nil {
        return nil, fmt.Errorf("invalid ai_coders configuration: %w", err)
    }
    
    return config, nil
}
```

### AI Coder Configuration Types
```go
// internal/config/ai_coder_config.go
type AICoderConfig struct {
    Enabled             bool                      `toml:"enabled"`
    MaxConcurrent       int                       `toml:"max_concurrent"`
    WorkspaceBaseDir    string                    `toml:"workspace_base_dir"`
    DefaultProvider     string                    `toml:"default_provider"`
    TimeoutMinutes      int                       `toml:"timeout_minutes"`
    AutoCleanup         bool                      `toml:"auto_cleanup"`
    CleanupAfterHours   int                       `toml:"cleanup_after_hours"`
    Providers           map[string]ProviderConfig `toml:"providers"`
    ResourceLimits      ResourceLimits            `toml:"resource_limits"`
    WorkspaceSettings   WorkspaceSettings         `toml:"workspace"`
    LoggingConfig       LoggingConfig             `toml:"logging"`
}

type ProviderConfig struct {
    APIKeyEnv       string            `toml:"api_key_env"`
    Model           string            `toml:"model"`
    BaseURL         string            `toml:"base_url"`
    MaxTokens       int               `toml:"max_tokens"`
    Temperature     float64           `toml:"temperature"`
    RequestTimeout  int               `toml:"request_timeout_seconds"`
    RateLimit       RateLimitConfig   `toml:"rate_limit"`
    CustomHeaders   map[string]string `toml:"custom_headers"`
}

type RateLimitConfig struct {
    RequestsPerMinute int `toml:"requests_per_minute"`
    TokensPerMinute   int `toml:"tokens_per_minute"`
}

type ResourceLimits struct {
    MaxMemoryMB        int `toml:"max_memory_mb"`
    MaxDiskSpaceMB     int `toml:"max_disk_space_mb"`
    MaxCPUPercent      int `toml:"max_cpu_percent"`
    MaxProcesses       int `toml:"max_processes"`
    MaxFilesPerCoder   int `toml:"max_files_per_coder"`
}

type WorkspaceSettings struct {
    Template           string   `toml:"template"`
    GitIgnoreRules     []string `toml:"gitignore_rules"`
    AllowedExtensions  []string `toml:"allowed_extensions"`
    ForbiddenPaths     []string `toml:"forbidden_paths"`
    MaxFileSize        int      `toml:"max_file_size_mb"`
    BackupEnabled      bool     `toml:"backup_enabled"`
}

type LoggingConfig struct {
    Level           string `toml:"level"`
    OutputFile      string `toml:"output_file"`
    RotateSize      int    `toml:"rotate_size_mb"`
    KeepRotations   int    `toml:"keep_rotations"`
    IncludeAIOutput bool   `toml:"include_ai_output"`
}

// Default configuration values
func DefaultAICoderConfig() AICoderConfig {
    return AICoderConfig{
        Enabled:           true,
        MaxConcurrent:     3,
        WorkspaceBaseDir:  "~/.brummer/ai-coders",
        DefaultProvider:   "claude",
        TimeoutMinutes:    30,
        AutoCleanup:       true,
        CleanupAfterHours: 24,
        Providers: map[string]ProviderConfig{
            "claude": {
                APIKeyEnv:      "ANTHROPIC_API_KEY",
                Model:          "claude-3-5-sonnet-20241022",
                MaxTokens:      4096,
                Temperature:    0.7,
                RequestTimeout: 30,
                RateLimit: RateLimitConfig{
                    RequestsPerMinute: 50,
                    TokensPerMinute:   150000,
                },
            },
            "openai": {
                APIKeyEnv:      "OPENAI_API_KEY", 
                Model:          "gpt-4",
                MaxTokens:      4096,
                Temperature:    0.7,
                RequestTimeout: 30,
                RateLimit: RateLimitConfig{
                    RequestsPerMinute: 60,
                    TokensPerMinute:   200000,
                },
            },
            "local": {
                BaseURL:        "http://localhost:11434",
                Model:          "codellama",
                MaxTokens:      2048,
                Temperature:    0.7,
                RequestTimeout: 60,
            },
        },
        ResourceLimits: ResourceLimits{
            MaxMemoryMB:      512,
            MaxDiskSpaceMB:   1024,
            MaxCPUPercent:    50,
            MaxProcesses:     5,
            MaxFilesPerCoder: 100,
        },
        WorkspaceSettings: WorkspaceSettings{
            Template:          "basic",
            GitIgnoreRules:    []string{"node_modules/", ".env", "*.log"},
            AllowedExtensions: []string{".go", ".js", ".ts", ".py", ".md", ".json", ".yaml", ".toml"},
            ForbiddenPaths:    []string{"/etc", "/var", "/sys", "/proc"},
            MaxFileSize:       10,
            BackupEnabled:     true,
        },
        LoggingConfig: LoggingConfig{
            Level:           "info",
            OutputFile:      "ai-coders.log",
            RotateSize:      50,
            KeepRotations:   5,
            IncludeAIOutput: false,
        },
    }
}
```

### Configuration Validation
```go
// internal/config/ai_coder_validation.go
import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

func (c *AICoderConfig) Validate() error {
    var errs []string
    
    // Validate basic settings
    if c.MaxConcurrent <= 0 {
        errs = append(errs, "max_concurrent must be positive")
    }
    if c.MaxConcurrent > 10 {
        errs = append(errs, "max_concurrent should not exceed 10 for system stability")
    }
    
    if c.WorkspaceBaseDir == "" {
        errs = append(errs, "workspace_base_dir is required")
    }
    
    if c.TimeoutMinutes <= 0 {
        errs = append(errs, "timeout_minutes must be positive")
    }
    
    // Validate default provider exists
    if c.DefaultProvider != "" {
        if _, exists := c.Providers[c.DefaultProvider]; !exists {
            errs = append(errs, fmt.Sprintf("default_provider '%s' not found in providers configuration", c.DefaultProvider))
        }
    }
    
    // Validate providers
    if len(c.Providers) == 0 {
        errs = append(errs, "at least one provider must be configured")
    }
    
    for name, provider := range c.Providers {
        if err := provider.Validate(name); err != nil {
            errs = append(errs, fmt.Sprintf("provider '%s': %v", name, err))
        }
    }
    
    // Validate resource limits
    if err := c.ResourceLimits.Validate(); err != nil {
        errs = append(errs, fmt.Sprintf("resource_limits: %v", err))
    }
    
    // Validate workspace settings
    if err := c.WorkspaceSettings.Validate(); err != nil {
        errs = append(errs, fmt.Sprintf("workspace: %v", err))
    }
    
    if len(errs) > 0 {
        return errors.New(strings.Join(errs, "; "))
    }
    
    return nil
}

func (p *ProviderConfig) Validate(providerName string) error {
    var errs []string
    
    // Validate API key configuration (except for local providers)
    if providerName != "local" && p.APIKeyEnv == "" {
        errs = append(errs, "api_key_env is required for non-local providers")
    }
    
    if p.APIKeyEnv != "" {
        if os.Getenv(p.APIKeyEnv) == "" {
            errs = append(errs, fmt.Sprintf("environment variable %s is not set", p.APIKeyEnv))
        }
    }
    
    // Validate model
    if p.Model == "" {
        errs = append(errs, "model is required")
    }
    
    // Validate limits
    if p.MaxTokens <= 0 {
        errs = append(errs, "max_tokens must be positive")
    }
    if p.MaxTokens > 50000 {
        errs = append(errs, "max_tokens exceeds reasonable limit (50000)")
    }
    
    if p.Temperature < 0 || p.Temperature > 2 {
        errs = append(errs, "temperature must be between 0 and 2")
    }
    
    if p.RequestTimeout <= 0 {
        errs = append(errs, "request_timeout_seconds must be positive")
    }
    
    // Validate rate limits
    if p.RateLimit.RequestsPerMinute <= 0 {
        errs = append(errs, "rate_limit.requests_per_minute must be positive")
    }
    if p.RateLimit.TokensPerMinute <= 0 {
        errs = append(errs, "rate_limit.tokens_per_minute must be positive")
    }
    
    if len(errs) > 0 {
        return errors.New(strings.Join(errs, "; "))
    }
    
    return nil
}

func (r *ResourceLimits) Validate() error {
    var errs []string
    
    if r.MaxMemoryMB <= 0 {
        errs = append(errs, "max_memory_mb must be positive")
    }
    if r.MaxMemoryMB > 8192 {
        errs = append(errs, "max_memory_mb exceeds reasonable limit (8GB)")
    }
    
    if r.MaxDiskSpaceMB <= 0 {
        errs = append(errs, "max_disk_space_mb must be positive")
    }
    
    if r.MaxCPUPercent <= 0 || r.MaxCPUPercent > 100 {
        errs = append(errs, "max_cpu_percent must be between 1 and 100")
    }
    
    if r.MaxProcesses <= 0 {
        errs = append(errs, "max_processes must be positive")
    }
    
    if r.MaxFilesPerCoder <= 0 {
        errs = append(errs, "max_files_per_coder must be positive")
    }
    
    if len(errs) > 0 {
        return errors.New(strings.Join(errs, "; "))
    }
    
    return nil
}

func (w *WorkspaceSettings) Validate() error {
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
    
    if w.MaxFileSize <= 0 {
        errs = append(errs, "max_file_size_mb must be positive")
    }
    if w.MaxFileSize > 100 {
        errs = append(errs, "max_file_size_mb should not exceed 100MB for performance")
    }
    
    if len(errs) > 0 {
        return errors.New(strings.Join(errs, "; "))
    }
    
    return nil
}

// Helper function to check if configuration directory exists and is writable
func (c *AICoderConfig) ValidateWorkspaceDirectory() error {
    // Expand tilde in path
    workspaceDir := c.WorkspaceBaseDir
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
```

## Example Configuration File
```toml
# .brum.toml - AI Coder Configuration Example
[ai_coders]
enabled = true
max_concurrent = 3
workspace_base_dir = "~/.brummer/ai-coders"
default_provider = "claude"
timeout_minutes = 30
auto_cleanup = true
cleanup_after_hours = 24

[ai_coders.providers.claude]
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-3-5-sonnet-20241022"
max_tokens = 4096
temperature = 0.7
request_timeout_seconds = 30

[ai_coders.providers.claude.rate_limit]
requests_per_minute = 50
tokens_per_minute = 150000

[ai_coders.providers.openai]
api_key_env = "OPENAI_API_KEY"
model = "gpt-4"
max_tokens = 4096
temperature = 0.7
request_timeout_seconds = 30

[ai_coders.providers.openai.rate_limit]
requests_per_minute = 60
tokens_per_minute = 200000

[ai_coders.providers.local]
base_url = "http://localhost:11434"
model = "codellama"
max_tokens = 2048
temperature = 0.7
request_timeout_seconds = 60

[ai_coders.resource_limits]
max_memory_mb = 512
max_disk_space_mb = 1024
max_cpu_percent = 50
max_processes = 5
max_files_per_coder = 100

[ai_coders.workspace]
template = "basic"
gitignore_rules = ["node_modules/", ".env", "*.log"]
allowed_extensions = [".go", ".js", ".ts", ".py", ".md", ".json", ".yaml", ".toml"]
forbidden_paths = ["/etc", "/var", "/sys", "/proc"]
max_file_size_mb = 10
backup_enabled = true

[ai_coders.logging]
level = "info"
output_file = "ai-coders.log"
rotate_size_mb = 50
keep_rotations = 5
include_ai_output = false
```

## Risk Mitigation (from master analysis)
**Low-Risk Mitigations**:
- Configuration validation - Comprehensive validation with clear error messages - Testing: Validation test suite with invalid configurations
- Environment variable security - API keys stored in environment variables only - Security: No secrets in configuration files
- Default value safety - All options have safe, tested default values - Fallback: System works with minimal configuration

**Context Validation**:
- [ ] TOML configuration patterns from `internal/config/config.go` successfully applied
- [ ] Environment variable integration consistent with existing patterns  
- [ ] Validation follows established error handling patterns

## Integration with Other Tasks
**Dependencies**: None (parallel with core service)
**Integration Points**: 
- Task 01 (Core Service) will consume this configuration
- Task 04 (TUI Integration) will display configuration status
- Task 05 (Process Integration) will use resource limits

**Shared Context**: Configuration becomes the single source of truth for AI coder behavior

## Execution Notes
- **Start Pattern**: Use existing configuration structure in `internal/config/config.go` as template
- **Key Context**: Focus on comprehensive validation and helpful error messages
- **Security Focus**: Ensure API keys are only accessed via environment variables
- **Review Focus**: Configuration validation logic and default value appropriateness

This task creates a robust, secure, and user-friendly configuration system that supports all AI coder functionality while maintaining consistency with Brummer's existing configuration patterns.