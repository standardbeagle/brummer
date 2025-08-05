package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/filebrowser"
)

// SettingsController manages settings view state and rendering
type SettingsController struct {
	// Dependencies
	config       *config.Config
	mcpServer    MCPServerInterface
	processMgr   *process.Manager
	workingDir   string
	fileBrowser  *filebrowser.Controller
	settingsList list.Model
	logStore     *logs.Store
	proxyServer  *proxy.Server

	// View state
	width        int
	height       int
	headerHeight int
	footerHeight int
}

// NewSettingsController creates a new settings controller
func NewSettingsController(cfg *config.Config, mcpServer MCPServerInterface, processMgr *process.Manager, workingDir string, fileBrowserController *filebrowser.Controller, logStore *logs.Store, proxyServer *proxy.Server) *SettingsController {
	settingsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	settingsList.Title = "Package Manager Settings"
	settingsList.SetShowStatusBar(false)

	return &SettingsController{
		config:       cfg,
		mcpServer:    mcpServer,
		processMgr:   processMgr,
		workingDir:   workingDir,
		fileBrowser:  fileBrowserController,
		logStore:     logStore,
		proxyServer:  proxyServer,
		settingsList: settingsList,
	}
}

// UpdateSize updates the viewport dimensions
func (s *SettingsController) UpdateSize(width, height, headerHeight, footerHeight int) {
	s.width = width
	s.height = height
	s.headerHeight = headerHeight
	s.footerHeight = footerHeight

	// Update settings list size
	contentHeight := height - headerHeight - footerHeight
	s.settingsList.SetSize(width, contentHeight)
}

// GetSettingsListPointer returns a pointer to the settings list (for backward compatibility)
func (s *SettingsController) GetSettingsListPointer() *list.Model {
	return &s.settingsList
}

// GetSettingsList returns the settings list
func (s *SettingsController) GetSettingsList() *list.Model {
	return &s.settingsList
}

// UpdateSettingsList updates the settings list with all configuration options
func (s *SettingsController) UpdateSettingsList() {
	// settingsList is now a value type, no need for nil check

	installedMgrs := s.processMgr.GetInstalledPackageManagers()
	currentMgr := s.processMgr.GetCurrentPackageManager()

	items := make([]list.Item, 0)

	// Server Information section (prominently displayed at top)
	items = append(items, settingsSectionItem{title: "🔗 Server Information"})

	// MCP Server info
	mcpStatus := "🔴 Not Running"
	if s.mcpServer != nil && s.mcpServer.IsRunning() {
		mcpStatus = "🟢 Running"
	}

	// Get actual port from MCP server if running
	actualPort := 7777 // default
	if s.mcpServer != nil && s.mcpServer.IsRunning() {
		actualPort = s.mcpServer.GetPort()
	}

	items = append(items, mcpServerInfoItem{
		port:   actualPort,
		status: mcpStatus,
	})

	// Add MCP endpoint information - always show URL for easy access
	mcpURL := fmt.Sprintf("http://localhost:%d/mcp", actualPort)
	items = append(items, infoDisplayItem{
		title:       "🔗 MCP Endpoint",
		description: fmt.Sprintf("%s (JSON-RPC 2.0 - all tools, resources & prompts)", mcpURL),
		value:       mcpURL,
		copyable:    true,
	})

	// Proxy Server info (if running and in full mode for PAC)
	if s.proxyServer != nil && s.proxyServer.IsRunning() && s.proxyServer.GetMode() == proxy.ProxyModeFull {
		items = append(items, proxyInfoItem{
			pacURL: s.proxyServer.GetPACURL(),
			mode:   s.proxyServer.GetMode(),
			port:   s.proxyServer.GetPort(),
		})
	}

	// Package Manager section
	items = append(items, settingsSectionItem{title: "📦 Package Managers"})
	for _, mgr := range installedMgrs {
		item := packageManagerSettingsItem{packageManagerItem{
			manager:  mgr,
			current:  mgr.Manager == currentMgr,
			fromJSON: s.processMgr.IsPackageManagerFromJSON(mgr.Manager),
		}}
		items = append(items, item)
	}

	// MCP Integration section
	items = append(items, settingsSectionItem{title: "🛠 MCP Integration"})
	mcpTools := mcp.GetSupportedTools()
	installedTools := mcp.GetInstalledTools()
	installedSet := make(map[string]bool)
	for _, tool := range installedTools {
		installedSet[tool] = true
	}

	for _, tool := range mcpTools {
		if tool.Supported {
			item := mcpInstallItem{
				tool:      tool,
				installed: installedSet[tool.Name],
			}
			items = append(items, item)
		}
	}

	// Add custom file browser option
	items = append(items, mcpFileBrowserItem{})

	s.settingsList.SetItems(items)
}

// UpdateFileBrowserList updates the file browser list
func (s *SettingsController) UpdateFileBrowserList() {
	// This method is delegated to the Model since it needs access to fileBrowserList
	// The Model will handle updating the list items from the file browser controller
}

// InstallMCPForTool installs an MCP tool
func (s *SettingsController) InstallMCPForTool(tool mcp.Tool) {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		if s.logStore != nil {
			s.logStore.Add("system", "System", fmt.Sprintf("Error getting executable path: %v", err), true)
		}
		return
	}

	// Generate config
	config := mcp.GenerateBrummerConfig(execPath, 7777)

	// Install
	if err := mcp.InstallForTool(tool, config); err != nil {
		if s.logStore != nil {
			s.logStore.Add("system", "System", fmt.Sprintf("Error installing MCP for %s: %v", tool.Name, err), true)
		}
	} else {
		if s.logStore != nil {
			s.logStore.Add("system", "System", fmt.Sprintf("Successfully configured %s with Brummer!", tool.Name), false)
		}
		s.UpdateSettingsList()
	}
}

// InstallMCPToFile installs MCP configuration to a specific file
func (s *SettingsController) InstallMCPToFile(filePath string) {
	s.installMCPToFile(filePath)
}

// installMCPToFile is the internal implementation
func (s *SettingsController) installMCPToFile(filePath string) {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		if s.logStore != nil {
			s.logStore.Add("system", "System", fmt.Sprintf("Error getting executable path: %v", err), true)
		}
		return
	}

	// Generate config
	config := mcp.GenerateBrummerConfig(execPath, 7777)

	// Read existing file
	data, err := os.ReadFile(filePath)
	if err != nil {
		if s.logStore != nil {
			s.logStore.Add("system", "System", fmt.Sprintf("Error reading file %s: %v", filePath, err), true)
		}
		return
	}

	var existingData map[string]interface{}
	if err := json.Unmarshal(data, &existingData); err != nil {
		if s.logStore != nil {
			s.logStore.Add("system", "System", fmt.Sprintf("Error parsing JSON in %s: %v", filePath, err), true)
		}
		return
	}

	// Try common MCP config formats
	if existingData["mcpServers"] == nil {
		existingData["mcpServers"] = make(map[string]interface{})
	}

	servers := existingData["mcpServers"].(map[string]interface{})
	servers["brummer"] = map[string]interface{}{
		"command": config.Command,
		"args":    config.Args,
		"env":     config.Env,
	}

	// Write back
	newData, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		if s.logStore != nil {
			s.logStore.Add("system", "System", fmt.Sprintf("Error marshaling JSON: %v", err), true)
		}
		return
	}

	if err := os.WriteFile(filePath, newData, 0644); err != nil {
		if s.logStore != nil {
			s.logStore.Add("system", "System", fmt.Sprintf("Error writing to %s: %v", filePath, err), true)
		}
		return
	}

	if s.logStore != nil {
		s.logStore.Add("system", "System", fmt.Sprintf("Successfully configured %s with Brummer!", filePath), false)
	}
}

// GetCLICommandFromConfig gets the CLI command for a specific task
// Note: This is a placeholder implementation since the config structure
// doesn't currently have Tools configuration
func (s *SettingsController) GetCLICommandFromConfig(configKey string, task string) (string, []string, error) {
	// TODO: Implement when Tools configuration is available in Config
	return "", nil, fmt.Errorf("tools configuration not implemented")
}

// getCLICommand gets the CLI command for a tool
// Note: This is a placeholder implementation
func (s *SettingsController) getCLICommand(configKey string) string {
	// TODO: Implement when Tools configuration is available in Config
	return ""
}

// Render renders the settings view
func (s *SettingsController) Render() string {
	var content strings.Builder

	// Header with branding and description
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1).
		Width(s.width)

	content.WriteString(headerStyle.Render("⚙️  Brummer Settings & Configuration"))
	content.WriteString("\n")

	// Subtitle with helpful information
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		MarginBottom(1)

	content.WriteString(subtitleStyle.Render("Configure your development environment and server settings • Press Enter to copy URLs"))
	content.WriteString("\n")

	// Calculate available height for the list
	localHeaderHeight := 4 // header + subtitle + margins within this view
	availableHeight := s.height - s.headerHeight - s.footerHeight - localHeaderHeight

	// Update list size and render
	s.settingsList.SetSize(s.width, availableHeight)
	content.WriteString(s.settingsList.View())

	return content.String()
}

// Settings item types are defined in model.go to avoid circular dependencies
// since they implement list.Item interface and are used in the settings list
