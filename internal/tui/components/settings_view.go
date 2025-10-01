package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// SettingsViewComponent displays configuration settings (read-only for now).
//
// Design principles:
// - Display-only (no editing yet)
// - Grouped by category
// - Shows setting descriptions
// - Keyboard scrolling
type SettingsViewComponent struct {
	BaseComponent

	viewport viewport.Model
	settings []SettingInfo

	// Styles
	headerStyle   lipgloss.Style
	categoryStyle lipgloss.Style
	keyStyle      lipgloss.Style
	valueStyle    lipgloss.Style
	normalStyle   lipgloss.Style
}

// SettingInfo holds display information for a setting
type SettingInfo struct {
	Category    string
	Key         string
	Value       string
	Description string
}

// NewSettingsViewComponent creates a settings view with configuration
func NewSettingsViewComponent() *SettingsViewComponent {
	c := &SettingsViewComponent{
		BaseComponent: NewBaseComponent(),
		viewport:      viewport.New(80, 24),
		settings:      []SettingInfo{},
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		categoryStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")),  // Bright Cyan
		keyStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("10")),             // Bright Green
		valueStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),             // Bright Yellow
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),             // White
	}

	// Initialize with default settings
	c.loadDefaultSettings()
	c.updateViewport()

	return c
}

// loadDefaultSettings loads default configuration settings
func (c *SettingsViewComponent) loadDefaultSettings() {
	c.settings = []SettingInfo{
		{
			Category:    "MCP Server",
			Key:         "mcp.port",
			Value:       "7777",
			Description: "Port for MCP server",
		},
		{
			Category:    "MCP Server",
			Key:         "mcp.enabled",
			Value:       "true",
			Description: "Enable MCP server",
		},
		{
			Category:    "Proxy Server",
			Key:         "proxy.enabled",
			Value:       "true",
			Description: "Enable HTTP proxy",
		},
		{
			Category:    "Proxy Server",
			Key:         "proxy.port",
			Value:       "auto",
			Description: "Proxy port (auto-detect)",
		},
		{
			Category:    "Process Manager",
			Key:         "process.max",
			Value:       "10",
			Description: "Maximum concurrent processes",
		},
		{
			Category:    "Logs",
			Key:         "logs.max_entries",
			Value:       "1000",
			Description: "Maximum log entries to keep",
		},
		{
			Category:    "UI",
			Key:         "ui.refresh_rate",
			Value:       "16ms",
			Description: "UI refresh rate",
		},
	}
}

// Init initializes the component
func (c *SettingsViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *SettingsViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height
		c.updateViewport()
		return c, nil

	case tea.KeyMsg:
		// Forward keyboard input to viewport for scrolling
		c.viewport, cmd = c.viewport.Update(msg)
		return c, cmd
	}

	return c, nil
}

// View renders the settings display
func (c *SettingsViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	return c.viewport.View()
}

// SetSize updates component dimensions
func (c *SettingsViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.viewport.Width = width
	c.viewport.Height = height
	c.updateViewport()
}

// updateViewport updates the viewport content with formatted settings
func (c *SettingsViewComponent) updateViewport() {
	var lines []string

	// Header
	header := c.headerStyle.Render("Configuration Settings")
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", c.viewport.Width))
	lines = append(lines, "")

	// Group by category
	currentCategory := ""
	for _, setting := range c.settings {
		if setting.Category != currentCategory {
			currentCategory = setting.Category
			if len(lines) > 3 {
				lines = append(lines, "")
			}
			lines = append(lines, c.categoryStyle.Render("▸ "+currentCategory))
		}

		// Setting line
		keyPart := c.keyStyle.Render(fmt.Sprintf("  %-20s", setting.Key))
		valuePart := c.valueStyle.Render(fmt.Sprintf("%-15s", setting.Value))
		descPart := c.normalStyle.Render(setting.Description)

		lines = append(lines, fmt.Sprintf("%s %s  %s", keyPart, valuePart, descPart))
	}

	lines = append(lines, "")
	lines = append(lines, c.normalStyle.Render("Note: Settings are read-only in this version"))

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)
}