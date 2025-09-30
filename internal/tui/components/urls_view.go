package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// URLsViewComponent displays detected URLs in a table format.
// It receives URLDetectedMsg messages to update the display.
//
// Design principles:
// - No direct service access (all updates via URLDetectedMsg)
// - Table view with columns: URL, Process, Type, Proxy Status
// - Supports scrolling through URLs
// - Highlights URLs with proxy access
type URLsViewComponent struct {
	BaseComponent

	viewport viewport.Model
	urls     []*URLInfo // List of detected URLs

	// Styles
	headerStyle lipgloss.Style
	urlStyle    lipgloss.Style
	proxyStyle  lipgloss.Style
	normalStyle lipgloss.Style
}

// URLInfo holds display information for a detected URL
type URLInfo struct {
	URL         string
	ProcessID   string
	Type        string // "http", "https", "ws", "wss"
	HasProxy    bool
	ProxyURL    string
	ProxyLabel  string
}

// NewURLsViewComponent creates a URLs view
func NewURLsViewComponent() *URLsViewComponent {
	c := &URLsViewComponent{
		BaseComponent: NewBaseComponent(),
		viewport:      viewport.New(80, 24),
		urls:          []*URLInfo{},
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		urlStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("14")),             // Bright Cyan
		proxyStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("10")),             // Bright Green
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),             // White
	}

	c.updateViewport()
	return c
}

// Init initializes the component
func (c *URLsViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *URLsViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.URLDetectedMsg:
		// NEW URL! Event-driven update

		// Check for duplicates
		isDuplicate := false
		for _, existing := range c.urls {
			if existing.URL == msg.URL && existing.ProcessID == msg.ProcessID {
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			c.urls = append(c.urls, &URLInfo{
				URL:        msg.URL,
				ProcessID:  msg.ProcessID,
				Type:       msg.Type,
				HasProxy:   msg.HasProxy,
				ProxyURL:   msg.ProxyURL,
				ProxyLabel: msg.ProxyLabel,
			})
			c.updateViewport()
		}
		return c, nil

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

// View renders the URLs table
func (c *URLsViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	return c.viewport.View()
}

// SetSize updates component dimensions
func (c *URLsViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.viewport.Width = width
	c.viewport.Height = height
	c.updateViewport()
}

// updateViewport updates the viewport content with formatted URLs table
func (c *URLsViewComponent) updateViewport() {
	var lines []string

	// Header
	header := c.headerStyle.Render(fmt.Sprintf("%-40s %-15s %-8s %s",
		"URL", "PROCESS", "TYPE", "PROXY"))
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", c.viewport.Width))

	// URL rows
	if len(c.urls) == 0 {
		lines = append(lines, c.normalStyle.Render("No URLs detected"))
	} else {
		for _, info := range c.urls {
			lines = append(lines, c.formatURLRow(info))
		}
	}

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)
}

// formatURLRow formats a single URL row
func (c *URLsViewComponent) formatURLRow(info *URLInfo) string {
	// Truncate long URLs
	url := info.URL
	if len(url) > 40 {
		url = url[:37] + "..."
	}

	// Truncate long process IDs
	processID := info.ProcessID
	if len(processID) > 15 {
		processID = processID[:12] + "..."
	}

	// Format type
	urlType := info.Type
	if len(urlType) > 8 {
		urlType = urlType[:8]
	}

	// Format proxy status
	proxyStatus := "-"
	if info.HasProxy {
		if info.ProxyLabel != "" {
			proxyStatus = info.ProxyLabel
		} else if info.ProxyURL != "" {
			proxyStatus = info.ProxyURL
		} else {
			proxyStatus = "✓"
		}
	}

	// Style the row
	styledURL := c.urlStyle.Render(fmt.Sprintf("%-40s", url))
	styledProxy := proxyStatus
	if info.HasProxy {
		styledProxy = c.proxyStyle.Render(proxyStatus)
	}

	return fmt.Sprintf("%s %-15s %-8s %s",
		styledURL,
		processID,
		urlType,
		styledProxy,
	)
}