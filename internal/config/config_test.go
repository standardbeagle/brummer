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