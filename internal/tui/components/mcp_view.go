package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// MCPViewComponentNew is a stateless BubbleTea component for displaying MCP server status
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type MCPViewComponentNew struct {
	BaseComponent

	// UI configuration only (no state)
	width   int
	height  int
	viewport viewport.Model

	// Styles
	headerStyle   lipgloss.Style
	activeStyle   lipgloss.Style
	inactiveStyle lipgloss.Style
	toolStyle     lipgloss.Style
	normalStyle   lipgloss.Style
	categoryStyle lipgloss.Style
	errorStyle    lipgloss.Style
	timestampStyle lipgloss.Style
}

// NewMCPViewComponent creates a new stateless MCP view component
func NewMCPViewComponent() *MCPViewComponentNew {
	return &MCPViewComponentNew{
		BaseComponent: NewBaseComponent(),
		width:         80,
		height:        24,
		viewport:      viewport.New(80, 24),

		// Initialize styles
		headerStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")), // Bright Blue
		activeStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("10")),            // Bright Green
		inactiveStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("9")),             // Bright Red
		toolStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("14")),            // Bright Cyan
		normalStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("15")),            // White
		categoryStyle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13")), // Bright Magenta
		errorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")),           // Bright Red
		timestampStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),          // Gray
	}
}

// ViewWithState renders the MCP view using external state (Redux/Elm compliant)
func (c *MCPViewComponentNew) ViewWithState(masterData *state.MasterData) string {
	if !c.IsVisible() {
		return ""
	}

	if masterData == nil || masterData.MCPState == nil {
		return c.renderErrorState("MCP state not available")
	}

	mcpState := masterData.MCPState

	// Generate content based on current state
	content := c.renderContent(mcpState)

	// Update viewport and render
	c.viewport.SetContent(content)
	return c.viewport.View()
}

// renderContent generates the main content based on MCP state
func (c *MCPViewComponentNew) renderContent(mcpState *state.MCPState) string {
	var content strings.Builder

	// Header
	content.WriteString(c.headerStyle.Render("MCP Server Status") + "\n")
	content.WriteString(strings.Repeat("─", c.width) + "\n\n")

	// Connection Status
	c.renderConnectionStatus(&content, mcpState)

	// Server Details
	c.renderServerDetails(&content, mcpState)

	// Statistics
	c.renderStatistics(&content, mcpState)

	// Available Tools
	if mcpState.GetShowDetails() {
		c.renderToolsSection(&content, mcpState)
	}

	// Footer help
	c.renderFooter(&content, mcpState)

	return content.String()
}

// renderConnectionStatus renders the connection status section
func (c *MCPViewComponentNew) renderConnectionStatus(content *strings.Builder, mcpState *state.MCPState) {
	statusLabel := "Status:     "
	var statusValue string

	if mcpState.GetLoading() {
		statusValue = c.normalStyle.Render("⟳ Loading...")
	} else if mcpState.GetActive() {
		statusValue = c.activeStyle.Render("● Active")
	} else {
		statusValue = c.inactiveStyle.Render("● Inactive")
	}

	content.WriteString(statusLabel + statusValue + "\n")

	// Show error if present
	if error := mcpState.GetError(); error != "" {
		content.WriteString(c.errorStyle.Render("Error:      "+error) + "\n")
	}
	content.WriteString("\n")
}

// renderServerDetails renders the server details section
func (c *MCPViewComponentNew) renderServerDetails(content *strings.Builder, mcpState *state.MCPState) {
	content.WriteString(c.normalStyle.Render(fmt.Sprintf("Address:    %s", mcpState.GetAddress())) + "\n")
	content.WriteString(c.normalStyle.Render(fmt.Sprintf("Port:       %d", mcpState.GetPort())) + "\n")

	// Show connection timestamp
	if !mcpState.GetActive() && mcpState.GetError() != "" {
		if lastConnected := mcpState.GetLastConnected(); !lastConnected.IsZero() {
			content.WriteString(c.timestampStyle.Render(fmt.Sprintf("Last seen:  %s", c.formatTimestamp(lastConnected))) + "\n")
		}
	}

	content.WriteString("\n")
}

// renderStatistics renders the statistics section
func (c *MCPViewComponentNew) renderStatistics(content *strings.Builder, mcpState *state.MCPState) {
	content.WriteString(c.headerStyle.Render("Statistics") + "\n")
	content.WriteString(strings.Repeat("─", c.width) + "\n")
	content.WriteString(c.normalStyle.Render(fmt.Sprintf("Available Tools:  %d", mcpState.GetToolsCount())) + "\n")
	content.WriteString(c.normalStyle.Render(fmt.Sprintf("Total Requests:   %d", mcpState.GetRequestCount())) + "\n")
	content.WriteString(c.normalStyle.Render(fmt.Sprintf("Active Sessions:  %d", mcpState.GetSessionCount())) + "\n")

	// Show last request timestamp
	if lastRequest := mcpState.GetLastRequest(); !lastRequest.IsZero() {
		content.WriteString(c.timestampStyle.Render(fmt.Sprintf("Last Request:    %s", c.formatTimestamp(lastRequest))) + "\n")
	}

	content.WriteString("\n")
}

// renderToolsSection renders the available tools section
func (c *MCPViewComponentNew) renderToolsSection(content *strings.Builder, mcpState *state.MCPState) {
	tools := mcpState.GetAvailableTools()
	if len(tools) == 0 {
		return
	}

	content.WriteString(c.headerStyle.Render("Available Tools") + "\n")
	content.WriteString(strings.Repeat("─", c.width) + "\n")

	// Get tools grouped by category
	categories := mcpState.GetToolsByCategory()

	// Display by category in a specific order
	categoryOrder := []string{"Scripts", "Logs", "Proxy", "Telemetry", "Browser", "REPL", "Other"}
	for _, category := range categoryOrder {
		categoryTools := categories[category]
		if len(categoryTools) > 0 {
			content.WriteString("\n")
			content.WriteString(c.categoryStyle.Render("▸ "+category) + "\n")
			for _, tool := range categoryTools {
				content.WriteString(c.toolStyle.Render("  • "+tool) + "\n")
			}
		}
	}

	content.WriteString("\n")
}

// renderFooter renders the footer help section
func (c *MCPViewComponentNew) renderFooter(content *strings.Builder, mcpState *state.MCPState) {
	content.WriteString(strings.Repeat("─", c.width) + "\n")

	helpLines := []string{
		"Use MCP client (Claude Code, Cursor, etc.) to invoke tools",
	}

	// Add refresh status if auto-refresh is enabled
	if mcpState.GetAutoRefresh() {
		helpLines = append(helpLines, fmt.Sprintf("Auto-refresh every %v", mcpState.GetRefreshInterval()))
	}

	if lastUpdate := mcpState.GetCacheTimestamp(); !lastUpdate.IsZero() {
		helpLines = append(helpLines, c.timestampStyle.Render("Last updated: "+c.formatTimestamp(lastUpdate)))
	}

	for i, line := range helpLines {
		content.WriteString(c.normalStyle.Render(line))
		if i < len(helpLines)-1 {
			content.WriteString(" • ")
		}
	}
}

// renderErrorState renders an error state when MCP data is not available
func (c *MCPViewComponentNew) renderErrorState(message string) string {
	var content strings.Builder

	content.WriteString(c.headerStyle.Render("MCP Server Status") + "\n")
	content.WriteString(strings.Repeat("─", c.width) + "\n\n")
	content.WriteString(c.errorStyle.Render("⚠️  "+message) + "\n\n")
	content.WriteString(c.normalStyle.Render("Check if MCP server is running with --mcp flag") + "\n")

	return content.String()
}

// formatTimestamp formats a timestamp for display
func (c *MCPViewComponentNew) formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	// Show relative time for recent timestamps, absolute for older ones
	now := time.Now()
	if diff := now.Sub(t); diff < 24*time.Hour {
		if diff < time.Minute {
			return "Just now"
		} else if diff < time.Hour {
			return fmt.Sprintf("%d min ago", int(diff.Minutes()))
		} else {
			return fmt.Sprintf("%d hour(s) ago", int(diff.Hours()))
		}
	}

	return t.Format("2006-01-02 15:04:05")
}

// SetSize updates component dimensions
func (c *MCPViewComponentNew) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.viewport.Width = width
	c.viewport.Height = height
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *MCPViewComponentNew) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.width = width
	c.height = height
	c.viewport.Width = width
	c.viewport.Height = contentHeight
}

// View renders the component (legacy method for compatibility)
func (c *MCPViewComponentNew) View() string {
	if !c.IsVisible() {
		return ""
	}
	return c.viewport.View()
}

// Update handles messages (minimal - stateless component)
func (c *MCPViewComponentNew) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil

	case tea.KeyMsg:
		// Forward keyboard input to viewport for scrolling
		c.viewport, cmd = c.viewport.Update(msg)
		return c, cmd
	}

	return c, nil
}

// Init initializes the component
func (c *MCPViewComponentNew) Init() tea.Cmd {
	return nil
}