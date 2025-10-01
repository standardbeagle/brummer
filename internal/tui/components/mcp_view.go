package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// MCPViewComponent displays MCP server status and statistics.
//
// Design principles:
// - Display-only (server control via CLI flags)
// - Shows connection status
// - Lists available MCP tools
// - Shows request statistics
// - Keyboard scrolling
type MCPViewComponent struct {
	BaseComponent

	viewport viewport.Model
	status   MCPStatus

	// Styles
	headerStyle  lipgloss.Style
	activeStyle  lipgloss.Style
	inactiveStyle lipgloss.Style
	toolStyle    lipgloss.Style
	normalStyle  lipgloss.Style
}

// MCPStatus holds display information for MCP server
type MCPStatus struct {
	Active        bool
	Port          int
	Address       string
	ToolsCount    int
	RequestCount  int
	SessionCount  int
	AvailableTools []string
}

// NewMCPViewComponent creates an MCP status view
func NewMCPViewComponent() *MCPViewComponent {
	c := &MCPViewComponent{
		BaseComponent: NewBaseComponent(),
		viewport:      viewport.New(80, 24),
		status: MCPStatus{
			Active:  false, // Will be updated via messages
			Port:    7777,
			Address: "localhost:7777",
			AvailableTools: []string{
				"scripts/list", "scripts/run", "scripts/stop", "scripts/status",
				"logs/stream", "logs/search",
				"proxy/requests",
				"telemetry/sessions", "telemetry/events",
				"browser/open", "browser/navigate", "browser/refresh", "browser/screenshot",
				"repl/execute",
			},
		},
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		activeStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("10")),             // Bright Green
		inactiveStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),              // Bright Red
		toolStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("14")),             // Bright Cyan
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),             // White
	}

	c.updateViewport()
	return c
}

// Init initializes the component
func (c *MCPViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *MCPViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// View renders the MCP status display
func (c *MCPViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	return c.viewport.View()
}

// SetSize updates component dimensions
func (c *MCPViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.viewport.Width = width
	c.viewport.Height = height
	c.updateViewport()
}

// updateViewport updates the viewport content with formatted MCP status
func (c *MCPViewComponent) updateViewport() {
	var lines []string

	// Header
	header := c.headerStyle.Render("MCP Server Status")
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", c.viewport.Width))
	lines = append(lines, "")

	// Connection Status
	statusLabel := "Status:     "
	var statusValue string
	if c.status.Active {
		statusValue = c.activeStyle.Render("● Active")
	} else {
		statusValue = c.inactiveStyle.Render("● Inactive")
	}
	lines = append(lines, statusLabel+statusValue)

	// Server Details
	lines = append(lines, c.normalStyle.Render(fmt.Sprintf("Address:    %s", c.status.Address)))
	lines = append(lines, c.normalStyle.Render(fmt.Sprintf("Port:       %d", c.status.Port)))
	lines = append(lines, "")

	// Statistics
	lines = append(lines, c.headerStyle.Render("Statistics"))
	lines = append(lines, strings.Repeat("─", c.viewport.Width))
	lines = append(lines, c.normalStyle.Render(fmt.Sprintf("Available Tools:  %d", len(c.status.AvailableTools))))
	lines = append(lines, c.normalStyle.Render(fmt.Sprintf("Total Requests:   %d", c.status.RequestCount)))
	lines = append(lines, c.normalStyle.Render(fmt.Sprintf("Active Sessions:  %d", c.status.SessionCount)))
	lines = append(lines, "")

	// Available Tools
	lines = append(lines, c.headerStyle.Render("Available Tools"))
	lines = append(lines, strings.Repeat("─", c.viewport.Width))

	// Group tools by category
	categories := map[string][]string{
		"Scripts":   []string{},
		"Logs":      []string{},
		"Proxy":     []string{},
		"Telemetry": []string{},
		"Browser":   []string{},
		"REPL":      []string{},
	}

	// Map tool prefixes to categories
	categoryMap := map[string]string{
		"scripts":   "Scripts",
		"logs":      "Logs",
		"proxy":     "Proxy",
		"telemetry": "Telemetry",
		"browser":   "Browser",
		"repl":      "REPL",
	}

	for _, tool := range c.status.AvailableTools {
		parts := strings.Split(tool, "/")
		if len(parts) >= 2 {
			prefix := strings.ToLower(parts[0])
			if category, ok := categoryMap[prefix]; ok {
				categories[category] = append(categories[category], tool)
			}
		}
	}

	// Display by category
	categoryOrder := []string{"Scripts", "Logs", "Proxy", "Telemetry", "Browser", "REPL"}
	for _, category := range categoryOrder {
		tools := categories[category]
		if len(tools) > 0 {
			lines = append(lines, "")
			lines = append(lines, c.normalStyle.Render("▸ "+category))
			for _, tool := range tools {
				lines = append(lines, c.toolStyle.Render("  • "+tool))
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, c.normalStyle.Render("Use MCP client (Claude Code, Cursor, etc.) to invoke tools"))

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)
}