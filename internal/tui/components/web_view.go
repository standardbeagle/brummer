package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// WebViewComponent displays HTTP proxy requests in a table format.
// It receives ProxyRequestMsg messages to update the display.
//
// Design principles:
// - Event-driven via ProxyRequestMsg
// - Shows method, URL, status code, duration
// - Color-coded by HTTP status
// - Supports keyboard scrolling
type WebViewComponent struct {
	BaseComponent

	viewport viewport.Model
	requests []*RequestInfo

	// Read-only reference for initial load
	proxyServer *proxy.Server

	// Styles
	headerStyle   lipgloss.Style
	successStyle  lipgloss.Style // 2xx
	redirectStyle lipgloss.Style // 3xx
	clientStyle   lipgloss.Style // 4xx
	serverStyle   lipgloss.Style // 5xx
	normalStyle   lipgloss.Style
}

// RequestInfo holds display information for a proxy request
type RequestInfo struct {
	ID          string
	Method      string
	URL         string
	Path        string
	StatusCode  int
	Duration    time.Duration
	ProcessName string
	Timestamp   time.Time
}

// NewWebViewComponent creates a web view with initial requests from proxy server
func NewWebViewComponent(proxyServer *proxy.Server) *WebViewComponent {
	c := &WebViewComponent{
		BaseComponent: NewBaseComponent(),
		proxyServer:   proxyServer,
		viewport:      viewport.New(80, 24),
		requests:      []*RequestInfo{},
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		successStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")),             // Bright Green
		redirectStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("14")),             // Bright Cyan
		clientStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("11")),             // Bright Yellow
		serverStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")),              // Bright Red
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),             // White
	}

	// Load existing requests
	if proxyServer != nil {
		existingRequests := proxyServer.GetRequests()
		for i := range existingRequests {
			req := &existingRequests[i]
			c.requests = append(c.requests, &RequestInfo{
				ID:          req.ID,
				Method:      req.Method,
				URL:         req.URL,
				Path:        req.Path,
				StatusCode:  req.StatusCode,
				Duration:    req.Duration,
				ProcessName: req.ProcessName,
				Timestamp:   req.StartTime,
			})
		}
		c.updateViewport()
	}

	return c
}

// Init initializes the component
func (c *WebViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *WebViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.ProxyRequestMsg:
		// NEW REQUEST! Event-driven update
		c.requests = append(c.requests, &RequestInfo{
			ID:          msg.RequestID,
			Method:      msg.Method,
			URL:         msg.URL,
			Path:        msg.Path,
			StatusCode:  msg.StatusCode,
			Duration:    msg.Duration,
			ProcessName: msg.ProcessName,
			Timestamp:   msg.Timestamp,
		})
		c.updateViewport()
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

// View renders the request table
func (c *WebViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	return c.viewport.View()
}

// SetSize updates component dimensions
func (c *WebViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.viewport.Width = width
	c.viewport.Height = height
	c.updateViewport()
}

// updateViewport updates the viewport content with formatted request table
func (c *WebViewComponent) updateViewport() {
	var lines []string

	// Header
	header := c.headerStyle.Render(fmt.Sprintf("%-8s %-6s %-40s %-10s %s",
		"METHOD", "STATUS", "URL", "DURATION", "PROCESS"))
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", c.viewport.Width))

	// Request rows
	if len(c.requests) == 0 {
		lines = append(lines, c.normalStyle.Render("No HTTP requests captured"))
	} else {
		// Show most recent requests first
		for i := len(c.requests) - 1; i >= 0; i-- {
			lines = append(lines, c.formatRequestRow(c.requests[i]))
		}
	}

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)

	// Auto-scroll to top (most recent)
	c.viewport.GotoTop()
}

// formatRequestRow formats a single request row
func (c *WebViewComponent) formatRequestRow(info *RequestInfo) string {
	// Format method
	method := info.Method
	if len(method) > 8 {
		method = method[:8]
	}

	// Format status with color
	statusStr := fmt.Sprintf("%d", info.StatusCode)
	if info.StatusCode == 0 {
		statusStr = "-"
	}

	var statusStyle lipgloss.Style
	switch {
	case info.StatusCode >= 200 && info.StatusCode < 300:
		statusStyle = c.successStyle
	case info.StatusCode >= 300 && info.StatusCode < 400:
		statusStyle = c.redirectStyle
	case info.StatusCode >= 400 && info.StatusCode < 500:
		statusStyle = c.clientStyle
	case info.StatusCode >= 500:
		statusStyle = c.serverStyle
	default:
		statusStyle = c.normalStyle
	}

	styledStatus := statusStyle.Render(fmt.Sprintf("%-6s", statusStr))

	// Format URL (truncate if too long)
	url := info.URL
	if len(url) > 40 {
		url = url[:37] + "..."
	}

	// Format duration
	duration := "-"
	if info.Duration > 0 {
		if info.Duration < time.Millisecond {
			duration = fmt.Sprintf("%dμs", info.Duration.Microseconds())
		} else if info.Duration < time.Second {
			duration = fmt.Sprintf("%dms", info.Duration.Milliseconds())
		} else {
			duration = fmt.Sprintf("%.1fs", info.Duration.Seconds())
		}
	}

	// Format process name
	processName := info.ProcessName
	if len(processName) > 15 {
		processName = processName[:12] + "..."
	}

	return fmt.Sprintf("%-8s %s %-40s %-10s %s",
		method,
		styledStatus,
		url,
		duration,
		processName,
	)
}