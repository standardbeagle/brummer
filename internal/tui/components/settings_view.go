package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// SettingsViewComponentNew is a stateless BubbleTea component for displaying settings
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type SettingsViewComponentNew struct {
	BaseComponent

	// UI configuration only (no state)
	width  int
	height int

	// List model for display (minimal, stateless)
	list   list.Model

	// Styles
	headerStyle     lipgloss.Style
	sectionStyle    lipgloss.Style
	infoStyle       lipgloss.Style
	valueStyle      lipgloss.Style
	errorStyle      lipgloss.Style
	loadingStyle    lipgloss.Style
	successStyle    lipgloss.Style
	copyableStyle   lipgloss.Style
}

// NewSettingsView creates a new stateless settings view component
func NewSettingsView() *SettingsViewComponentNew {
	settingsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	settingsList.Title = "Settings"
	settingsList.SetShowStatusBar(false)

	return &SettingsViewComponentNew{
		BaseComponent: NewBaseComponent(),
		width:         80,
		height:        24,
		list:          settingsList,

		// Initialize styles
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")), // Bright Blue
		sectionStyle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13")), // Bright Magenta
		infoStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("15")),            // White
		valueStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("14")),            // Bright Cyan
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),           // Bright Red
		loadingStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("11")),            // Bright Yellow
		successStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")),            // Bright Green
		copyableStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),             // Gray
	}
}

// ViewWithState renders the settings view using external state (Redux/Elm compliant)
func (c *SettingsViewComponentNew) ViewWithState(masterData *state.MasterData) string {
	if !c.IsVisible() {
		return ""
	}

	if masterData == nil || masterData.SettingsState == nil {
		return c.renderErrorState("Settings state not available")
	}

	settingsState := masterData.SettingsState

	// Update list with current state
	c.updateListFromState(settingsState)

	// Render the list
	return c.list.View()
}

// updateListFromState updates the settings list using current state
func (c *SettingsViewComponentNew) updateListFromState(settingsState *state.SettingsState) {
	if settingsState.GetLoading() {
		// Show loading state
		c.list.SetItems([]list.Item{
			SettingsSectionItemWrapper{TitleText: "⏳ Loading Settings..."},
		})
		return
	}

	if error := settingsState.GetError(); error != "" {
		// Show error state
		c.list.SetItems([]list.Item{
			SettingsSectionItemWrapper{TitleText: "❌ Error Loading Settings"},
			InfoDisplayItemWrapper{
				TitleText:       "Error Details",
				DescriptionText: error,
			},
		})
		return
	}

	settingsInfo := settingsState.GetSettingsInfo()
	if settingsInfo == nil {
		// Show empty state
		c.list.SetItems([]list.Item{
			SettingsSectionItemWrapper{TitleText: "📋 No Settings Available"},
			InfoDisplayItemWrapper{
				TitleText:       "Status",
				DescriptionText: "Settings information not available. Try refreshing.",
			},
		})
		return
	}

	items := make([]list.Item, 0)

	// Server Information section
	items = append(items, SettingsSectionItemWrapper{TitleText: "🔗 Server Information"})

	// MCP Server info
	mcpInfo := settingsState.GetMCPServerInfo()
	if mcpInfo != nil {
		items = append(items, MCPServerInfoItemWrapper{
			Port:   mcpInfo.Port,
			Status: mcpInfo.Status,
		})

		items = append(items, InfoDisplayItemWrapper{
			TitleText:       "🔗 MCP Endpoint",
			DescriptionText: fmt.Sprintf("%s (JSON-RPC 2.0 - all tools, resources & prompts)", mcpInfo.URL),
			Value:           mcpInfo.URL,
			Copyable:        true,
		})
	}

	// Proxy info
	proxyInfo := settingsState.GetProxyServerInfo()
	if proxyInfo != nil && proxyInfo.Running {
		items = append(items, ProxyInfoItemWrapper{
			PacURL: proxyInfo.PacURL,
			Mode:   proxyInfo.Mode,
			Port:   proxyInfo.Port,
		})
	}

	// Package manager section
	items = append(items, SettingsSectionItemWrapper{TitleText: "📦 Package Manager"})
	pkgMgrInfo := settingsState.GetPackageManagers()
	if pkgMgrInfo != nil {
		for _, mgr := range pkgMgrInfo.Available {
			items = append(items, PackageManagerItemWrapper{
				Name:      mgr,
				IsCurrent: mgr == pkgMgrInfo.Current,
			})
		}
	}

	// Working directory section
	items = append(items, SettingsSectionItemWrapper{TitleText: "📁 Working Directory"})
	workingDir := settingsState.GetWorkingDirectory()
	if workingDir != "" {
		items = append(items, InfoDisplayItemWrapper{
			TitleText:       "Current Directory",
			DescriptionText: workingDir,
			Value:           workingDir,
			Copyable:        false,
		})
	}

	// MCP Configuration status
	items = append(items, SettingsSectionItemWrapper{TitleText: "⚙️ Configuration"})
	mcpStatus := "❌ Not Configured"
	if settingsState.IsMCPConfigured() {
		mcpStatus = "✅ Configured"
	}
	items = append(items, InfoDisplayItemWrapper{
		TitleText:       "Claude Desktop MCP",
		DescriptionText: fmt.Sprintf("MCP configuration status: %s", mcpStatus),
		Value:           mcpStatus,
		Copyable:        false,
	})

	// Last refresh info
	lastUpdate := settingsState.GetLastUpdateTime()
	if !lastUpdate.IsZero() {
		items = append(items, InfoDisplayItemWrapper{
			TitleText:       "Last Updated",
			DescriptionText: lastUpdate.Format("2006-01-02 15:04:05"),
			Value:           "",
			Copyable:        false,
		})
	}

	c.list.SetItems(items)
}

// renderErrorState renders an error state when settings data is not available
func (c *SettingsViewComponentNew) renderErrorState(message string) string {
	var content strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Padding(0, 1).
		MarginBottom(1)

	content.WriteString(headerStyle.Render("⚙️  Settings"))
	content.WriteString("\n")

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Margin(1)

	content.WriteString(errorStyle.Render("⚠️  "+message))
	content.WriteString("\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Margin(1)

	content.WriteString(helpStyle.Render("Settings information will be available when the application is fully initialized"))

	return content.String()
}

// SetSize updates component dimensions
func (c *SettingsViewComponentNew) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.list.SetSize(width, height-6)
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *SettingsViewComponentNew) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.width = width
	c.height = height
	c.list.SetSize(width, contentHeight)
}

// View renders the component (legacy method for compatibility)
func (c *SettingsViewComponentNew) View() string {
	if !c.IsVisible() {
		return ""
	}
	return c.list.View()
}

// Update handles messages (minimal - stateless component)
func (c *SettingsViewComponentNew) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil

	case tea.KeyMsg:
		// Forward keyboard input to list for selection
		c.list, cmd = c.list.Update(msg)
		return c, cmd
	}

	return c, nil
}

// Init initializes the component
func (c *SettingsViewComponentNew) Init() tea.Cmd {
	return nil
}

// Wrapper types for list.Item interface compatibility

// SettingsSectionItemWrapper represents a section header in the settings list
type SettingsSectionItemWrapper struct {
	TitleText string
}

func (i SettingsSectionItemWrapper) FilterValue() string { return "" }
func (i SettingsSectionItemWrapper) Title() string       { return i.TitleText }
func (i SettingsSectionItemWrapper) Description() string { return "" }

// MCPServerInfoItemWrapper represents MCP server information
type MCPServerInfoItemWrapper struct {
	Port   int
	Status string
}

func (i MCPServerInfoItemWrapper) FilterValue() string { return "mcp server" }
func (i MCPServerInfoItemWrapper) Title() string       { return fmt.Sprintf("MCP Server: %s", i.Status) }
func (i MCPServerInfoItemWrapper) Description() string { return fmt.Sprintf("Port: %d", i.Port) }

// InfoDisplayItemWrapper represents a display-only information item
type InfoDisplayItemWrapper struct {
	TitleText       string
	DescriptionText string
	Value           string
	Copyable        bool
}

func (i InfoDisplayItemWrapper) FilterValue() string { return i.TitleText }
func (i InfoDisplayItemWrapper) Title() string       { return i.TitleText }
func (i InfoDisplayItemWrapper) Description() string { return i.DescriptionText }

// ProxyInfoItemWrapper represents proxy server information
type ProxyInfoItemWrapper struct {
	PacURL string
	Mode   string
	Port   int
}

func (i ProxyInfoItemWrapper) FilterValue() string { return "proxy" }
func (i ProxyInfoItemWrapper) Title() string       { return "🌐 Proxy Server: Running" }
func (i ProxyInfoItemWrapper) Description() string {
	return fmt.Sprintf("Mode: %s | Port: %d | PAC URL: %s", i.Mode, i.Port, i.PacURL)
}

// PackageManagerItemWrapper represents a package manager option
type PackageManagerItemWrapper struct {
	Name      string
	IsCurrent bool
}

func (i PackageManagerItemWrapper) FilterValue() string { return i.Name }
func (i PackageManagerItemWrapper) Title() string {
	if i.IsCurrent {
		return fmt.Sprintf("✅ %s (current)", i.Name)
	}
	return fmt.Sprintf("   %s", i.Name)
}
func (i PackageManagerItemWrapper) Description() string { return "" }