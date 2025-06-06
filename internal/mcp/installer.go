package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Tool represents a development tool that supports MCP
type Tool struct {
	Name           string
	ConfigPath     string
	ConfigFormat   string
	Supported      bool
	InstallCommand string
}

// MCPConfig represents the configuration for different tools
type MCPConfig struct {
	Name    string                 `json:"name"`
	Command string                 `json:"command"`
	Args    []string               `json:"args,omitempty"`
	Env     map[string]string      `json:"env,omitempty"`
	Schema  map[string]interface{} `json:"schema,omitempty"`
}

// GetSupportedTools returns a list of tools that support MCP
func GetSupportedTools() []Tool {
	homeDir, _ := os.UserHomeDir()
	
	tools := []Tool{
		{
			Name:         "Claude Desktop",
			ConfigPath:   getClaudeConfigPath(homeDir),
			ConfigFormat: "claude_desktop",
			Supported:    true,
		},
		{
			Name:           "Claude Code",
			ConfigPath:     filepath.Join(homeDir, ".claude", "claude_code_config.json"),
			ConfigFormat:   "claude_code",
			Supported:      true,
			InstallCommand: "claude mcp add",
		},
		{
			Name:         "Cursor",
			ConfigPath:   getCursorConfigPath(homeDir),
			ConfigFormat: "cursor",
			Supported:    true,
		},
		{
			Name:           "VSCode (with MCP extension)",
			ConfigPath:     getVSCodeConfigPath(homeDir),
			ConfigFormat:   "vscode",
			Supported:      true,
			InstallCommand: "code --add-mcp",
		},
		{
			Name:         "Cline",
			ConfigPath:   filepath.Join(homeDir, ".cline", "mcp_config.json"),
			ConfigFormat: "cline",
			Supported:    true,
		},
		{
			Name:         "Windsurf",
			ConfigPath:   filepath.Join(homeDir, ".windsurf", "mcp_servers.json"),
			ConfigFormat: "windsurf",
			Supported:    true,
		},
		{
			Name:         "Roo Code",
			ConfigPath:   filepath.Join(homeDir, ".roo", "mcp_config.json"),
			ConfigFormat: "standard",
			Supported:    false, // Unclear if supported
		},
		{
			Name:         "Augment",
			ConfigPath:   filepath.Join(homeDir, ".augment", "mcp_config.json"),
			ConfigFormat: "standard",
			Supported:    false, // Unclear if supported
		},
		{
			Name:         "Cody",
			ConfigPath:   filepath.Join(homeDir, ".cody", "mcp_config.json"),
			ConfigFormat: "standard",
			Supported:    false, // Unclear if supported
		},
	}
	
	return tools
}

func getClaudeConfigPath(homeDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json")
	default: // linux
		return filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json")
	}
}

func getCursorConfigPath(homeDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Cursor", "User", "mcp_servers.json")
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Cursor", "User", "mcp_servers.json")
	default: // linux
		return filepath.Join(homeDir, ".config", "Cursor", "User", "mcp_servers.json")
	}
}

func getVSCodeConfigPath(homeDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "settings.json")
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Code", "User", "settings.json")
	default: // linux
		return filepath.Join(homeDir, ".config", "Code", "User", "settings.json")
	}
}

// GenerateBrummerConfig creates the MCP configuration for brummer
func GenerateBrummerConfig(execPath string, port int) MCPConfig {
	return MCPConfig{
		Name:    "brummer",
		Command: execPath,
		Args:    []string{"--port", fmt.Sprintf("%d", port), "--no-tui"},
		Env: map[string]string{
			"BRUMMER_MODE": "mcp",
		},
		Schema: map[string]interface{}{
			"name":        "brummer",
			"description": "🐝 Brummer package script manager - intelligent log management for npm/yarn/pnpm/bun scripts",
			"version":     "1.0.0",
			"capabilities": []string{
				"logs",
				"processes", 
				"scripts",
				"execute",
				"search",
				"filters",
				"events",
			},
		},
	}
}

// InstallForTool installs the MCP configuration for a specific tool
func InstallForTool(tool Tool, config MCPConfig) error {
	// Ensure config directory exists
	configDir := filepath.Dir(tool.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	switch tool.ConfigFormat {
	case "claude_desktop":
		return installClaudeDesktopConfig(tool.ConfigPath, config)
	case "claude_code":
		return installClaudeCodeConfig(config)
	case "cursor":
		return installCursorConfig(tool.ConfigPath, config)
	case "vscode":
		return installVSCodeConfig(config)
	case "cline":
		return installClineConfig(tool.ConfigPath, config)
	case "windsurf":
		return installWindsurfConfig(tool.ConfigPath, config)
	default:
		return installStandardConfig(tool.ConfigPath, config)
	}
}

func installClaudeDesktopConfig(configPath string, config MCPConfig) error {
	// Read existing config or create new
	existingData := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &existingData)
	}
	
	// Ensure mcpServers exists
	if existingData["mcpServers"] == nil {
		existingData["mcpServers"] = make(map[string]interface{})
	}
	
	servers := existingData["mcpServers"].(map[string]interface{})
	
	// Add brummer config
	servers["brummer"] = map[string]interface{}{
		"command": config.Command,
		"args":    config.Args,
		"env":     config.Env,
	}
	
	// Write back
	data, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

func installClaudeCodeConfig(config MCPConfig) error {
	// Use the claude mcp add command
	args := []string{"mcp", "add", config.Name, config.Command}
	args = append(args, config.Args...)
	
	cmd := exec.Command("claude", args...)
	
	// Set environment variables if any
	if len(config.Env) > 0 {
		env := os.Environ()
		for k, v := range config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("claude mcp add failed: %w\nOutput: %s", err, string(output))
	}
	
	return nil
}

func installCursorConfig(configPath string, config MCPConfig) error {
	// Similar format to Claude Desktop
	return installClaudeDesktopConfig(configPath, config)
}

func installVSCodeConfig(config MCPConfig) error {
	// Create the JSON for VSCode --add-mcp command
	mcpDef := map[string]interface{}{
		"name":    config.Name,
		"command": config.Command,
		"args":    config.Args,
		"env":     config.Env,
	}
	
	jsonData, err := json.Marshal(mcpDef)
	if err != nil {
		return fmt.Errorf("failed to marshal MCP definition: %w", err)
	}
	
	// Use the code --add-mcp command
	cmd := exec.Command("code", "--add-mcp", string(jsonData))
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("code --add-mcp failed: %w\nOutput: %s", err, string(output))
	}
	
	return nil
}

func installClineConfig(configPath string, config MCPConfig) error {
	// Cline uses a simple array format
	var configs []MCPConfig
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &configs)
	}
	
	// Remove existing brummer config
	filtered := []MCPConfig{}
	for _, c := range configs {
		if c.Name != "brummer" {
			filtered = append(filtered, c)
		}
	}
	
	// Add new config
	filtered = append(filtered, config)
	
	// Write back
	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

func installWindsurfConfig(configPath string, config MCPConfig) error {
	// Similar to Cline
	return installClineConfig(configPath, config)
}

func installStandardConfig(configPath string, config MCPConfig) error {
	// Generic format - just write the config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

// GetInstalledTools checks which tools have brummer installed
func GetInstalledTools() []string {
	installed := []string{}
	tools := GetSupportedTools()
	
	for _, tool := range tools {
		if hasBrummerInstalled(tool) {
			installed = append(installed, tool.Name)
		}
	}
	
	return installed
}

func hasBrummerInstalled(tool Tool) bool {
	// Special handling for tools with CLI commands
	switch tool.ConfigFormat {
	case "claude_code":
		return hasClaudeCodeMCPInstalled()
	case "vscode":
		return hasVSCodeMCPInstalled(tool.ConfigPath)
	}
	
	data, err := os.ReadFile(tool.ConfigPath)
	if err != nil {
		return false
	}
	
	return json.Valid(data) && contains(string(data), "brummer")
}

func hasClaudeCodeMCPInstalled() bool {
	cmd := exec.Command("claude", "mcp", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	return contains(string(output), "brummer")
}

func hasVSCodeMCPInstalled(configPath string) bool {
	// VSCode doesn't seem to have a list command, so check settings.json
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}
	
	// Check both possible locations
	if mcpServers, ok := settings["mcp.servers"].(map[string]interface{}); ok {
		_, exists := mcpServers["brummer"]
		return exists
	}
	
	// Also check if it's just a string match for robustness
	return contains(string(data), "brummer")
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}