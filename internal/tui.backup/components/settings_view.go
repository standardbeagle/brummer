package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// SettingsView is a BubbleTea component for displaying settings
// Refactored to use ServiceBridge message-based architecture
type SettingsView struct {
	settingsList list.Model
	width        int
	height       int
	// Cached settings information from ServiceBridge
	cachedSettingsInfo *messages.SettingsInfo
	lastUpdateTime     time.Time
}

// NewSettingsView creates a new settings view component
func NewSettingsView() *SettingsView {
	settingsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	settingsList.Title = "Settings"
	settingsList.SetShowStatusBar(false)

	return &SettingsView{
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

	switch msg := msg.(type) {
	case messages.SettingsRefreshedMsg:
		// Update cached settings information
		v.cachedSettingsInfo = msg.SettingsInfo
		v.lastUpdateTime = msg.Timestamp
		v.updateSettingsListFromCache()

	case messages.SettingsMCPInstalledMsg:
		// MCP configuration was installed - could trigger a refresh
		// This will be handled by the main model requesting a settings refresh

	case messages.SettingsErrorMsg:
		// Handle settings errors - could display error state
		// For now, just log or could update UI to show error
	}

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

// updateSettingsListFromCache updates the settings list using cached service information
func (v *SettingsView) updateSettingsListFromCache() {
	if v.cachedSettingsInfo == nil {
		// Show loading state
		v.settingsList.SetItems([]list.Item{
			SettingsSectionItem{TitleText: "⏳ Loading Settings..."},
		})
		return
	}

	items := make([]list.Item, 0)

	// Server Information section
	items = append(items, SettingsSectionItem{TitleText: "🔗 Server Information"})

	// MCP Server info
	items = append(items, MCPServerInfoItem{
		Port:   v.cachedSettingsInfo.ServerInfo.MCPServer.Port,
		Status: v.cachedSettingsInfo.ServerInfo.MCPServer.Status,
	})

	items = append(items, InfoDisplayItem{
		TitleText:       "🔗 MCP Endpoint",
		DescriptionText: fmt.Sprintf("%s (JSON-RPC 2.0 - all tools, resources & prompts)", v.cachedSettingsInfo.ServerInfo.MCPServer.URL),
		Value:           v.cachedSettingsInfo.ServerInfo.MCPServer.URL,
		Copyable:        true,
	})

	// Proxy info
	if v.cachedSettingsInfo.ServerInfo.ProxyServer.Running {
		items = append(items, ProxyInfoItem{
			PacURL: v.cachedSettingsInfo.ServerInfo.ProxyServer.PacURL,
			Mode:   v.cachedSettingsInfo.ServerInfo.ProxyServer.Mode,
			Port:   v.cachedSettingsInfo.ServerInfo.ProxyServer.Port,
		})
	}

	// Package manager section
	items = append(items, SettingsSectionItem{TitleText: "📦 Package Manager"})
	for _, mgr := range v.cachedSettingsInfo.PackageMgr.Available {
		items = append(items, PackageManagerItem{
			Name:      mgr,
			IsCurrent: mgr == v.cachedSettingsInfo.PackageMgr.Current,
		})
	}

	// Working directory section
	items = append(items, SettingsSectionItem{TitleText: "📁 Working Directory"})
	items = append(items, InfoDisplayItem{
		TitleText:       "Current Directory",
		DescriptionText: v.cachedSettingsInfo.WorkingDir,
		Value:           v.cachedSettingsInfo.WorkingDir,
		Copyable:        false,
	})

	// MCP Configuration status
	items = append(items, SettingsSectionItem{TitleText: "⚙️ Configuration"})
	mcpStatus := "❌ Not Configured"
	if v.cachedSettingsInfo.MCPConfigured {
		mcpStatus = "✅ Configured"
	}
	items = append(items, InfoDisplayItem{
		TitleText:       "Claude Desktop MCP",
		DescriptionText: fmt.Sprintf("MCP configuration status: %s", mcpStatus),
		Value:           mcpStatus,
		Copyable:        false,
	})

	// Last refresh info
	items = append(items, InfoDisplayItem{
		TitleText:       "Last Updated",
		DescriptionText: v.lastUpdateTime.Format("2006-01-02 15:04:05"),
		Value:           "",
		Copyable:        false,
	})

	v.settingsList.SetItems(items)
}

// Render renders the settings view
func (v *SettingsView) Render() string {
	return v.settingsList.View()
}

// RequestSettingsRefresh returns a command to request settings refresh
func (v *SettingsView) RequestSettingsRefresh() tea.Cmd {
	return func() tea.Msg {
		return messages.SettingsUpdateMsg{
			Refresh: true,
		}
	}
}

// RequestMCPInstall returns a command to request MCP installation
func (v *SettingsView) RequestMCPInstall() tea.Cmd {
	return func() tea.Msg {
		return messages.SettingsUpdateMsg{
			InstallMCP: true,
		}
	}
}

// InitializeWithLoadingState sets up the initial loading state
func (v *SettingsView) InitializeWithLoadingState() {
	v.settingsList.SetItems([]list.Item{
		SettingsSectionItem{TitleText: "⏳ Loading Settings..."},
	})
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
	Mode   string
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
