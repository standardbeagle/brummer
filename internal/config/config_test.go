package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/standardbeagle/brummer/internal/parser"
)

func TestConfigLoadSave(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Test loading empty config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.PreferredPackageManager != nil {
		t.Error("Expected empty config to have nil PreferredPackageManager")
	}

	// Test saving config
	npm := parser.NPM
	cfg.PreferredPackageManager = &npm

	err = cfg.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ".brum.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Test loading saved config
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Load() after save failed: %v", err)
	}

	if cfg2.PreferredPackageManager == nil {
		t.Error("Expected loaded config to have PreferredPackageManager")
	} else if *cfg2.PreferredPackageManager != npm {
		t.Errorf("Expected %v, got %v", npm, *cfg2.PreferredPackageManager)
	}
}

func TestConfigOverrideChain(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "project", "subdir")
	os.MkdirAll(subDir, 0755)

	// Create config in parent directory
	parentConfig := filepath.Join(tmpDir, "project", ".brum.toml")
	parentContent := `preferred_package_manager = "yarn"
mcp_port = 8080
proxy_port = 20000`
	os.WriteFile(parentConfig, []byte(parentContent), 0644)

	// Create config in subdirectory
	subConfig := filepath.Join(subDir, ".brum.toml")
	subContent := `preferred_package_manager = "npm"
mcp_port = 9090
proxy_mode = "full"`
	os.WriteFile(subConfig, []byte(subContent), 0644)

	// Change to subdirectory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(subDir)

	// Load config - should get npm from local config overriding yarn from parent
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.PreferredPackageManager == nil {
		t.Error("Expected config to have PreferredPackageManager")
	} else if *cfg.PreferredPackageManager != parser.NPM {
		t.Errorf("Expected npm, got %v", *cfg.PreferredPackageManager)
	}

	// Test MCP port override (local overrides parent)
	if cfg.GetMCPPort() != 9090 {
		t.Errorf("Expected MCP port 9090, got %d", cfg.GetMCPPort())
	}

	// Test proxy port inheritance (from parent, not overridden locally)
	if cfg.GetProxyPort() != 20000 {
		t.Errorf("Expected proxy port 20000, got %d", cfg.GetProxyPort())
	}

	// Test proxy mode override (local overrides default)
	if cfg.GetProxyMode() != "full" {
		t.Errorf("Expected proxy mode 'full', got %s", cfg.GetProxyMode())
	}
}

func TestConfigDefaults(t *testing.T) {
	// Create empty config
	cfg := &Config{}

	// Test default values
	if cfg.GetMCPPort() != 7777 {
		t.Errorf("Expected default MCP port 7777, got %d", cfg.GetMCPPort())
	}

	if cfg.GetProxyPort() != 19888 {
		t.Errorf("Expected default proxy port 19888, got %d", cfg.GetProxyPort())
	}

	if cfg.GetProxyMode() != "reverse" {
		t.Errorf("Expected default proxy mode 'reverse', got %s", cfg.GetProxyMode())
	}

	if cfg.GetNoMCP() != false {
		t.Errorf("Expected default no_mcp false, got %t", cfg.GetNoMCP())
	}

	if cfg.GetNoProxy() != false {
		t.Errorf("Expected default no_proxy false, got %t", cfg.GetNoProxy())
	}
}

func TestAIConfigurationLoading(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Create AI configuration file
	aiConfigContent := `[ai_coders]
enabled = true
max_concurrent = 5
default_provider = "claude"
timeout_minutes = 45
auto_cleanup = true

[ai_coders.providers.claude]
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-3-5-sonnet-20241022"
max_tokens = 8192
temperature = 0.5

[ai_coders.providers.claude.rate_limit]
requests_per_minute = 100
tokens_per_minute = 200000

[ai_coders.providers.aider]
api_key_env = ""
model = ""

[ai_coders.providers.aider.cli_tool]
command = "aider"
base_args = ["--yes", "--no-auto-commits"]

[ai_coders.providers.aider.cli_tool.flag_mapping]
model = "--model"
message = "--message"

[ai_coders.resource_limits]
max_memory_mb = 1024
max_cpu_percent = 75

[ai_coders.workspace]
template = "advanced"
backup_enabled = true

[ai_coders.logging]
level = "debug"
include_ai_output = true`

	configPath := filepath.Join(tmpDir, ".brum.toml")
	err := os.WriteFile(configPath, []byte(aiConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write AI config: %v", err)
	}

	// Load configuration
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Test AI configuration loading
	if cfg.AICoders == nil {
		t.Fatal("Expected AICoders to be loaded")
	}

	// Test basic AI settings
	if cfg.AICoders.Enabled == nil || *cfg.AICoders.Enabled != true {
		t.Error("Expected AI coders to be enabled")
	}

	if cfg.AICoders.MaxConcurrent == nil || *cfg.AICoders.MaxConcurrent != 5 {
		t.Error("Expected max_concurrent to be 5")
	}

	if cfg.AICoders.DefaultProvider == nil || *cfg.AICoders.DefaultProvider != "claude" {
		t.Error("Expected default_provider to be claude")
	}

	if cfg.AICoders.TimeoutMinutes == nil || *cfg.AICoders.TimeoutMinutes != 45 {
		t.Error("Expected timeout_minutes to be 45")
	}

	// Test provider configuration
	if cfg.AICoders.Providers == nil {
		t.Fatal("Expected providers to be loaded")
	}

	claudeProvider, exists := cfg.AICoders.Providers["claude"]
	if !exists {
		t.Fatal("Expected claude provider to be loaded")
	}

	if claudeProvider.GetAPIKeyEnv() != "ANTHROPIC_API_KEY" {
		t.Error("Expected claude provider API key env to be ANTHROPIC_API_KEY")
	}

	if claudeProvider.GetModel() != "claude-3-5-sonnet-20241022" {
		t.Error("Expected claude provider model to be claude-3-5-sonnet-20241022")
	}

	if claudeProvider.GetMaxTokens() != 8192 {
		t.Error("Expected claude provider max_tokens to be 8192")
	}

	if claudeProvider.GetTemperature() != 0.5 {
		t.Error("Expected claude provider temperature to be 0.5")
	}

	// Test aider CLI tool configuration
	aiderProvider, exists := cfg.AICoders.Providers["aider"]
	if !exists {
		t.Fatal("Expected aider provider to be loaded")
	}

	if aiderProvider.CLITool == nil {
		t.Fatal("Expected aider CLI tool configuration")
	}

	if aiderProvider.CLITool.Command == nil || *aiderProvider.CLITool.Command != "aider" {
		t.Error("Expected aider CLI command to be 'aider'")
	}

	if len(aiderProvider.CLITool.BaseArgs) != 2 {
		t.Error("Expected aider base_args to have 2 elements")
	}

	if aiderProvider.CLITool.FlagMapping == nil {
		t.Fatal("Expected aider flag mapping")
	}

	if aiderProvider.CLITool.FlagMapping["model"] != "--model" {
		t.Error("Expected model flag mapping to be '--model'")
	}

	// Test resource limits
	if cfg.AICoders.ResourceLimits == nil {
		t.Fatal("Expected resource limits to be loaded")
	}

	if cfg.AICoders.ResourceLimits.GetMaxMemoryMB() != 1024 {
		t.Error("Expected max_memory_mb to be 1024")
	}

	if cfg.AICoders.ResourceLimits.GetMaxCPUPercent() != 75 {
		t.Error("Expected max_cpu_percent to be 75")
	}

	// Test workspace settings
	if cfg.AICoders.WorkspaceSettings == nil {
		t.Fatal("Expected workspace settings to be loaded")
	}

	if cfg.AICoders.WorkspaceSettings.GetTemplate() != "advanced" {
		t.Error("Expected workspace template to be 'advanced'")
	}

	if cfg.AICoders.WorkspaceSettings.GetBackupEnabled() != true {
		t.Error("Expected backup_enabled to be true")
	}

	// Test logging configuration
	if cfg.AICoders.LoggingConfig == nil {
		t.Fatal("Expected logging config to be loaded")
	}

	if cfg.AICoders.LoggingConfig.GetLevel() != "debug" {
		t.Error("Expected logging level to be 'debug'")
	}

	if cfg.AICoders.LoggingConfig.GetIncludeAIOutput() != true {
		t.Error("Expected include_ai_output to be true")
	}
}

func TestAIConfigurationDefaults(t *testing.T) {
	// Create empty config
	cfg := &Config{}

	// Test AI config defaults
	aiConfig := cfg.GetAICoderConfig()
	if aiConfig.MaxConcurrent != 3 {
		t.Errorf("Expected default max_concurrent 3, got %d", aiConfig.MaxConcurrent)
	}

	if aiConfig.DefaultProvider != "claude" {
		t.Errorf("Expected default provider 'claude', got %s", aiConfig.DefaultProvider)
	}

	if aiConfig.TimeoutMinutes != 30 {
		t.Errorf("Expected default timeout_minutes 30, got %d", aiConfig.TimeoutMinutes)
	}

	if aiConfig.WorkspaceBaseDir != "~/.brummer/ai-coders" {
		t.Errorf("Expected default workspace_base_dir '~/.brummer/ai-coders', got %s", aiConfig.WorkspaceBaseDir)
	}
}

func TestAIConfigurationOverrideChain(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "project", "subdir")
	os.MkdirAll(subDir, 0755)

	// Create parent AI config
	parentConfig := filepath.Join(tmpDir, "project", ".brum.toml")
	parentContent := `[ai_coders]
enabled = true
max_concurrent = 2
default_provider = "gemini"

[ai_coders.providers.gemini]
api_key_env = "GEMINI_API_KEY"
model = "gemini-1.5-pro"`
	os.WriteFile(parentConfig, []byte(parentContent), 0644)

	// Create subdirectory AI config (overriding some settings)
	subConfig := filepath.Join(subDir, ".brum.toml")
	subContent := `[ai_coders]
max_concurrent = 5
timeout_minutes = 60

[ai_coders.providers.claude]
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-3-5-sonnet-20241022"`
	os.WriteFile(subConfig, []byte(subContent), 0644)

	// Change to subdirectory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(subDir)

	// Load configuration
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Test that local config overrides parent config
	if cfg.AICoders == nil {
		t.Fatal("Expected AICoders to be loaded")
	}

	// max_concurrent should be from local config (5), not parent (2)
	if cfg.AICoders.MaxConcurrent == nil || *cfg.AICoders.MaxConcurrent != 5 {
		t.Error("Expected max_concurrent to be 5 from local config")
	}

	// timeout_minutes should be from local config
	if cfg.AICoders.TimeoutMinutes == nil || *cfg.AICoders.TimeoutMinutes != 60 {
		t.Error("Expected timeout_minutes to be 60 from local config")
	}

	// enabled should be inherited from parent config (not overridden locally)
	if cfg.AICoders.Enabled == nil || *cfg.AICoders.Enabled != true {
		t.Error("Expected enabled to be inherited from parent config")
	}

	// default_provider should be inherited from parent config
	if cfg.AICoders.DefaultProvider == nil || *cfg.AICoders.DefaultProvider != "gemini" {
		t.Error("Expected default_provider to be inherited from parent config")
	}

	// Providers should be merged (claude from local + gemini from parent)
	if cfg.AICoders.Providers == nil {
		t.Fatal("Expected providers to be loaded")
	}

	if len(cfg.AICoders.Providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(cfg.AICoders.Providers))
	}

	if _, exists := cfg.AICoders.Providers["claude"]; !exists {
		t.Error("Expected claude provider from local config")
	}

	if _, exists := cfg.AICoders.Providers["gemini"]; !exists {
		t.Error("Expected gemini provider from parent config")
	}
}

func TestConfigurationSourceTracking(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(subDir, 0755)

	// Create parent config
	parentConfig := filepath.Join(tmpDir, ".brum.toml")
	parentContent := `mcp_port = 8080
preferred_package_manager = "yarn"`
	os.WriteFile(parentConfig, []byte(parentContent), 0644)

	// Create subdirectory config
	subConfig := filepath.Join(subDir, ".brum.toml")
	subContent := `mcp_port = 9090
proxy_port = 20000`
	os.WriteFile(subConfig, []byte(subContent), 0644)

	// Change to subdirectory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(subDir)

	// Load configuration with source tracking
	cfg, err := LoadWithSources()
	if err != nil {
		t.Fatalf("LoadWithSources() failed: %v", err)
	}

	// Test source tracking
	if len(cfg.Sources) == 0 {
		t.Error("Expected sources to be tracked")
	}

	// MCP port should come from local config
	if source, ok := cfg.Sources["mcp_port"]; !ok || source != subConfig {
		t.Errorf("Expected mcp_port source to be %s, got %s", subConfig, source)
	}

	// Preferred package manager should come from parent config
	if source, ok := cfg.Sources["preferred_package_manager"]; !ok || source != parentConfig {
		t.Errorf("Expected preferred_package_manager source to be %s, got %s", parentConfig, source)
	}

	// Proxy port should come from local config
	if source, ok := cfg.Sources["proxy_port"]; !ok || source != subConfig {
		t.Errorf("Expected proxy_port source to be %s, got %s", subConfig, source)
	}
}

func TestInvalidConfigurationHandling(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Create invalid TOML file
	invalidConfig := filepath.Join(tmpDir, ".brum.toml")
	invalidContent := `mcp_port = "not_a_number"
preferred_package_manager = invalid_value
[invalid_section`
	err := os.WriteFile(invalidConfig, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Load configuration - should not fail, just skip invalid file
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not fail with invalid config: %v", err)
	}

	// Should get default values since invalid file was skipped
	if cfg.GetMCPPort() != 7777 {
		t.Errorf("Expected default MCP port when config is invalid, got %d", cfg.GetMCPPort())
	}
}
