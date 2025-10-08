package components

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/filebrowser"
)

// SettingsView is a BubbleTea component for displaying settings
// Migrated from settings_controller.go (T062-T066)
// Architecture: Component with multiple service dependencies
type SettingsView struct {
	config    *config.Config
	mcpServer interface {
		IsRunning() bool
		GetPort() int
	}
	processMgr   *process.Manager
	workingDir   string
	fileBrowser  *filebrowser.Controller
	settingsList list.Model
	logStore     *logs.Store
	proxyServer  *proxy.Server
	width        int
	height       int
}

// NewSettingsView creates a new settings view component
func NewSettingsView(cfg *config.Config, mcpServer interface {
	IsRunning() bool
	GetPort() int
}, processMgr *process.Manager, workingDir string, fileBrowserController *filebrowser.Controller, logStore *logs.Store, proxyServer *proxy.Server) *SettingsView {
	settingsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	settingsList.Title = "Package Manager Settings"
	settingsList.SetShowStatusBar(false)

	return &SettingsView{
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

// Init initializes the component (BubbleTea interface)
func (v *SettingsView) Init() tea.Cmd {
	return nil
}

// Update handles messages (BubbleTea interface)
func (v *SettingsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	v.settingsList, cmd = v.settingsList.Update(msg)
	return v, cmd
}

// View renders the component (BubbleTea interface)
func (v *SettingsView) View() string {
	return v.Render()
}

// SetSize updates the component's dimensions (Component interface)
func (v *SettingsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.settingsList.SetSize(width, height-6)
}

// Focus gives keyboard focus to the component (Component interface)
func (v *SettingsView) Focus() tea.Cmd {
	return nil
}

// Blur removes keyboard focus from the component (Component interface)
func (v *SettingsView) Blur() tea.Cmd {
	return nil
}

// GetSize returns current dimensions (Component interface)
func (v *SettingsView) GetSize() (width, height int) {
	return v.width, v.height
}

// UpdateSize updates dimensions with pre-calculated content height
func (v *SettingsView) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	v.width = width
	v.height = height
	v.settingsList.SetSize(width, contentHeight)
}

// GetSettingsList returns the settings list
func (v *SettingsView) GetSettingsList() *list.Model {
	return &v.settingsList
}

// UpdateSettingsList updates the settings list with all configuration options
func (v *SettingsView) UpdateSettingsList() {
	installedMgrs := v.processMgr.GetInstalledPackageManagers()
	currentMgr := v.processMgr.GetCurrentPackageManager()

	items := make([]list.Item, 0)

	// Server Information section
	items = append(items, SettingsSectionItem{TitleText: "🔗 Server Information"})

	// MCP Server info
	mcpStatus := "🔴 Not Running"
	actualPort := 7777
	if v.mcpServer != nil && v.mcpServer.IsRunning() {
		mcpStatus = "🟢 Running"
		actualPort = v.mcpServer.GetPort()
	}

	items = append(items, MCPServerInfoItem{
		Port:   actualPort,
		Status: mcpStatus,
	})

	mcpURL := fmt.Sprintf("http://localhost:%d/mcp", actualPort)
	items = append(items, InfoDisplayItem{
		TitleText:       "🔗 MCP Endpoint",
		DescriptionText: fmt.Sprintf("%s (JSON-RPC 2.0 - all tools, resources & prompts)", mcpURL),
		Value:           mcpURL,
		Copyable:        true,
	})

	// Proxy info
	if v.proxyServer != nil && v.proxyServer.IsRunning() && v.proxyServer.GetMode() == proxy.ProxyModeFull {
		items = append(items, ProxyInfoItem{
			PacURL: v.proxyServer.GetPACURL(),
			Mode:   v.proxyServer.GetMode(),
			Port:   v.proxyServer.GetPort(),
		})
	}

	// Package manager section
	items = append(items, SettingsSectionItem{TitleText: "📦 Package Manager"})
	for _, mgr := range installedMgrs {
		items = append(items, PackageManagerItem{
			Name:      string(mgr.Manager),
			IsCurrent: mgr.Manager == currentMgr,
		})
	}

	// Working directory section
	items = append(items, SettingsSectionItem{TitleText: "📁 Working Directory"})
	items = append(items, InfoDisplayItem{
		TitleText:       "Current Directory",
		DescriptionText: v.workingDir,
		Value:           v.workingDir,
		Copyable:        false,
	})

	v.settingsList.SetItems(items)
}

// Render renders the settings view
func (v *SettingsView) Render() string {
	return v.settingsList.View()
}

// InstallMCP installs MCP configuration to Claude desktop config file
func (v *SettingsView) InstallMCP() {
	if v.fileBrowser != nil {
		v.fileBrowser.Show()
		v.UpdateSettingsList()
	}
}

// InstallMCPToFile installs MCP configuration to a specific file
func (v *SettingsView) InstallMCPToFile(filePath string) {
	execPath, err := os.Executable()
	if err != nil {
		if v.logStore != nil {
			v.logStore.Add("system", "System", fmt.Sprintf("Error getting executable path: %v", err), true)
		}
		return
	}

	config := mcp.GenerateBrummerConfig(execPath, 7777)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if v.logStore != nil {
			v.logStore.Add("system", "System", fmt.Sprintf("Error reading file %s: %v", filePath, err), true)
		}
		return
	}

	var existingData map[string]interface{}
	if err := json.Unmarshal(data, &existingData); err != nil {
		if v.logStore != nil {
			v.logStore.Add("system", "System", fmt.Sprintf("Error parsing JSON in %s: %v", filePath, err), true)
		}
		return
	}

	if existingData["mcpServers"] == nil {
		existingData["mcpServers"] = make(map[string]interface{})
	}

	servers := existingData["mcpServers"].(map[string]interface{})
	servers["brummer"] = map[string]interface{}{
		"command": config.Command,
		"args":    config.Args,
		"env":     config.Env,
	}

	updatedData, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		if v.logStore != nil {
			v.logStore.Add("system", "System", fmt.Sprintf("Error encoding JSON: %v", err), true)
		}
		return
	}

	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		if v.logStore != nil {
			v.logStore.Add("system", "System", fmt.Sprintf("Error writing file %s: %v", filePath, err), true)
		}
		return
	}

	if v.logStore != nil {
		v.logStore.Add("system", "System", fmt.Sprintf("✅ MCP configuration added to %s", filePath), false)
	}
}

// SettingsSectionItem represents a section header in the settings list
type SettingsSectionItem struct {
	TitleText string
}

func (i SettingsSectionItem) FilterValue() string { return "" }
func (i SettingsSectionItem) Title() string       { return i.TitleText }
func (i SettingsSectionItem) Description() string { return "" }

// MCPServerInfoItem represents MCP server information
type MCPServerInfoItem struct {
	Port   int
	Status string
}

func (i MCPServerInfoItem) FilterValue() string { return "mcp server" }
func (i MCPServerInfoItem) Title() string       { return fmt.Sprintf("MCP Server: %s", i.Status) }
func (i MCPServerInfoItem) Description() string { return fmt.Sprintf("Port: %d", i.Port) }

// InfoDisplayItem represents a display-only information item
type InfoDisplayItem struct {
	TitleText       string
	DescriptionText string
	Value           string
	Copyable        bool
}

func (i InfoDisplayItem) FilterValue() string { return i.TitleText }
func (i InfoDisplayItem) Title() string       { return i.TitleText }
func (i InfoDisplayItem) Description() string { return i.DescriptionText }

// ProxyInfoItem represents proxy server information
type ProxyInfoItem struct {
	PacURL string
	Mode   proxy.ProxyMode
	Port   int
}

func (i ProxyInfoItem) FilterValue() string { return "proxy" }
func (i ProxyInfoItem) Title() string       { return "🌐 Proxy Server: Running" }
func (i ProxyInfoItem) Description() string {
	return fmt.Sprintf("Mode: %s | Port: %d | PAC URL: %s", i.Mode, i.Port, i.PacURL)
}

// PackageManagerItem represents a package manager option
type PackageManagerItem struct {
	Name      string
	IsCurrent bool
}

func (i PackageManagerItem) FilterValue() string { return i.Name }
func (i PackageManagerItem) Title() string {
	if i.IsCurrent {
		return fmt.Sprintf("✅ %s (current)", i.Name)
	}
	return fmt.Sprintf("   %s", i.Name)
}
func (i PackageManagerItem) Description() string { return "" }
