package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HeaderComponent displays global status indicators across the top.
//
// Design principles:
// - Shows system-wide status (MCP, Proxy, AI Coders)
// - Compact, single-line display
// - Color-coded status indicators
// - Updates via status messages
type HeaderComponent struct {
	BaseComponent

	title       string
	mcpStatus   StatusInfo
	proxyStatus StatusInfo
	aiStatus    StatusInfo
	processCount int

	// Styles
	titleStyle     lipgloss.Style
	activeStyle    lipgloss.Style
	inactiveStyle  lipgloss.Style
	warningStyle   lipgloss.Style
	separatorStyle lipgloss.Style
}

// StatusInfo holds status display information
type StatusInfo struct {
	Active  bool
	Label   string
	Details string // Optional details (e.g., "port 7777")
}

// NewHeaderComponent creates a header status component
func NewHeaderComponent(title string) *HeaderComponent {
	c := &HeaderComponent{
		BaseComponent: NewBaseComponent(),
		title:         title,
		mcpStatus: StatusInfo{
			Active: false,
			Label:  "MCP",
		},
		proxyStatus: StatusInfo{
			Active: false,
			Label:  "Proxy",
		},
		aiStatus: StatusInfo{
			Active: false,
			Label:  "AI Coders",
		},
		processCount: 0,
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")). // Bright Blue
			Padding(0, 1),
		activeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Bright Green
			Bold(true),
		inactiveStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")), // Dark Gray
		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Bright Yellow
			Bold(true),
		separatorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			SetString(" │ "),
	}

	return c
}

// Init initializes the component
func (c *HeaderComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *HeaderComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil
	}

	return c, nil
}

// View renders the header
func (c *HeaderComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	var parts []string

	// Title
	parts = append(parts, c.titleStyle.Render(c.title))

	// Process count
	if c.processCount > 0 {
		count := c.activeStyle.Render(fmt.Sprintf("⚙️  %d", c.processCount))
		parts = append(parts, count)
	}

	// MCP Status
	mcpText := c.renderStatus(c.mcpStatus)
	parts = append(parts, mcpText)

	// Proxy Status
	proxyText := c.renderStatus(c.proxyStatus)
	parts = append(parts, proxyText)

	// AI Coders Status
	aiText := c.renderStatus(c.aiStatus)
	parts = append(parts, aiText)

	// Join with separators
	return strings.Join(parts, c.separatorStyle.String())
}

// renderStatus renders a single status indicator
func (c *HeaderComponent) renderStatus(status StatusInfo) string {
	var icon string
	var style lipgloss.Style

	if status.Active {
		icon = "●"
		style = c.activeStyle
	} else {
		icon = "○"
		style = c.inactiveStyle
	}

	text := fmt.Sprintf("%s %s", icon, status.Label)
	if status.Details != "" {
		text = fmt.Sprintf("%s (%s)", text, status.Details)
	}

	return style.Render(text)
}

// SetSize updates component dimensions
func (c *HeaderComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
}

// SetMCPStatus updates MCP server status
func (c *HeaderComponent) SetMCPStatus(active bool, details string) {
	c.mcpStatus.Active = active
	c.mcpStatus.Details = details
}

// SetProxyStatus updates proxy server status
func (c *HeaderComponent) SetProxyStatus(active bool, details string) {
	c.proxyStatus.Active = active
	c.proxyStatus.Details = details
}

// SetAICodersStatus updates AI coders status
func (c *HeaderComponent) SetAICodersStatus(active bool, details string) {
	c.aiStatus.Active = active
	c.aiStatus.Details = details
}

// SetProcessCount updates the process count
func (c *HeaderComponent) SetProcessCount(count int) {
	c.processCount = count
}

// SetTitle updates the header title
func (c *HeaderComponent) SetTitle(title string) {
	c.title = title
}
