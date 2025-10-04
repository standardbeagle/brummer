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

	// Selection and detail view
	selectedIndex int
	selectedRequest *RequestInfo
	showDetail bool
	filter string
	autoScroll bool

	// Layout
	width int
	height int

	// Styles
	headerStyle   lipgloss.Style
	successStyle  lipgloss.Style // 2xx
	redirectStyle lipgloss.Style // 3xx
	clientStyle   lipgloss.Style // 4xx
	serverStyle   lipgloss.Style // 5xx
	normalStyle   lipgloss.Style
	detailStyle   lipgloss.Style
	labelStyle    lipgloss.Style
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
	Size        int64
	Error       string

	// Authentication
	HasAuth   bool
	AuthType  string
	JWTError  string
	JWTClaims map[string]interface{}

	// Telemetry
	HasTelemetry bool
	SessionID    string
	// Note: Full telemetry data would require more complex types for this component
}

// NewWebViewComponent creates a web view with initial requests from proxy server
func NewWebViewComponent(proxyServer *proxy.Server) *WebViewComponent {
	c := &WebViewComponent{
		BaseComponent: NewBaseComponent(),
		proxyServer:   proxyServer,
		viewport:      viewport.New(80, 24),
		requests:      []*RequestInfo{},

		// Selection and detail view
		selectedIndex:  -1,
		selectedRequest: nil,
		showDetail:    false,
		filter:        "all",
		autoScroll:    true,

		// Layout
		width:  80,
		height: 24,

		// Styles
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		successStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")),             // Bright Green
		redirectStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("14")),             // Bright Cyan
		clientStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("11")),             // Bright Yellow
		serverStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")),              // Bright Red
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),             // White
		detailStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),            // Bright White
		labelStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true), // Gray
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
				Size:        req.Size,
				Error:       req.Error,

				// Authentication
				HasAuth:   req.HasAuth,
				AuthType:  req.AuthType,
				JWTError:  req.JWTError,
				JWTClaims: req.JWTClaims,

				// Telemetry
				HasTelemetry: req.HasTelemetry,
				SessionID:    req.SessionID,
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
		newRequest := &RequestInfo{
			ID:          msg.RequestID,
			Method:      msg.Method,
			URL:         msg.URL,
			Path:        msg.Path,
			StatusCode:  msg.StatusCode,
			Duration:    msg.Duration,
			ProcessName: msg.ProcessName,
			Timestamp:   msg.Timestamp,
			// Additional fields would be populated from proxy server directly
		}
		c.requests = append(c.requests, newRequest)

		// Auto-scroll to newest request if enabled
		if c.autoScroll {
			filteredRequests := c.getFilteredRequests()
			c.selectedIndex = len(filteredRequests) - 1
			c.selectedRequest = newRequest
		}

		return c, nil

	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height
		c.updateViewport()
		return c, nil

	case tea.KeyMsg:
		// Handle keyboard navigation
		switch msg.Type {
		case tea.KeyUp:
			c.moveSelection(-1)
		case tea.KeyDown:
			c.moveSelection(1)
		case tea.KeyEnter:
			c.toggleDetailView()
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "f":
				c.cycleFilter()
			case " ":
				c.toggleAutoScroll()
			}
		default:
			// Forward other keyboard input to viewport for scrolling
			c.viewport, cmd = c.viewport.Update(msg)
		}
		return c, cmd
	}

	return c, nil
}

// View renders the request table or detail view
func (c *WebViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	// Render based on detail view state and available width
	if c.width >= 100 && c.showDetail && c.selectedRequest != nil {
		// Split view: list on left, detail on right
		return c.renderSplitView()
	} else {
		// Single view: either list or detail
		if c.showDetail && c.selectedRequest != nil {
			return c.renderDetailView()
		} else {
			return c.renderListView()
		}
	}
}

// SetSize updates component dimensions
func (c *WebViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.width = width
	c.height = height
	c.viewport.Width = width
	c.viewport.Height = height
	c.updateViewport()
}

// updateViewport updates the viewport content when in single view mode
func (c *WebViewComponent) updateViewport() {
	// This method is now only used for single view mode
	// Split view and detail view are handled by their respective render methods
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

	// Build the base line
	line := fmt.Sprintf("%-8s %s %-40s %-10s %s",
		method,
		styledStatus,
		url,
		duration,
		processName,
	)

	// Add indicators for additional data
	if info.Error != "" {
		line += " ❌"
	}
	if info.HasAuth {
		line += " 🔐"
	}
	if info.HasTelemetry {
		line += " 📊"
	}

	return line
}

// Helper methods for navigation and interaction

// moveSelection moves the selection up or down
func (c *WebViewComponent) moveSelection(direction int) {
	filteredRequests := c.getFilteredRequests()
	if len(filteredRequests) == 0 {
		return
	}

	c.selectedIndex += direction

	// Wrap around
	if c.selectedIndex < 0 {
		c.selectedIndex = len(filteredRequests) - 1
	} else if c.selectedIndex >= len(filteredRequests) {
		c.selectedIndex = 0
	}

	// Update selected request
	c.selectedRequest = filteredRequests[c.selectedIndex]
}

// toggleDetailView toggles the detail view
func (c *WebViewComponent) toggleDetailView() {
	if c.selectedRequest != nil {
		c.showDetail = !c.showDetail
	}
}

// cycleFilter cycles through the available filters
func (c *WebViewComponent) cycleFilter() {
	filters := []string{"all", "pages", "api", "images", "other"}
	for i, filter := range filters {
		if filter == c.filter {
			c.filter = filters[(i+1)%len(filters)]
			c.selectedIndex = -1
			c.selectedRequest = nil
			c.showDetail = false
			return
		}
	}
	c.filter = "all"
}

// toggleAutoScroll toggles auto-scroll behavior
func (c *WebViewComponent) toggleAutoScroll() {
	c.autoScroll = !c.autoScroll
}

// getFilteredRequests returns requests filtered by the current filter
func (c *WebViewComponent) getFilteredRequests() []*RequestInfo {
	if c.filter == "all" {
		return c.requests
	}

	var filtered []*RequestInfo
	for _, req := range c.requests {
		switch c.filter {
		case "pages":
			if c.isPageRequest(req) {
				filtered = append(filtered, req)
			}
		case "api":
			if c.isAPIRequest(req) {
				filtered = append(filtered, req)
			}
		case "images":
			if c.isImageRequest(req) {
				filtered = append(filtered, req)
			}
		case "other":
			if !c.isPageRequest(req) && !c.isAPIRequest(req) && !c.isImageRequest(req) {
				filtered = append(filtered, req)
			}
		}
	}
	return filtered
}

// Request classification helpers
func (c *WebViewComponent) isPageRequest(req *RequestInfo) bool {
	return strings.Contains(req.Path, ".html") || req.Path == "/" ||
		(!strings.Contains(req.Path, ".") && !strings.Contains(req.Path, "/api/"))
}

func (c *WebViewComponent) isAPIRequest(req *RequestInfo) bool {
	return strings.Contains(req.Path, "/api/") || strings.Contains(req.Path, "/graphql") ||
		req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" || req.Method == "PATCH"
}

func (c *WebViewComponent) isImageRequest(req *RequestInfo) bool {
	return strings.HasSuffix(req.Path, ".jpg") || strings.HasSuffix(req.Path, ".jpeg") ||
		strings.HasSuffix(req.Path, ".png") || strings.HasSuffix(req.Path, ".gif") ||
		strings.HasSuffix(req.Path, ".webp") || strings.HasSuffix(req.Path, ".svg") ||
		strings.HasSuffix(req.Path, ".ico")
}

// Rendering methods

// renderListView renders the list view with filters and status
func (c *WebViewComponent) renderListView() string {
	var content strings.Builder

	// Header with filter info
	content.WriteString(c.renderHeader())

	// Calculate available height for content
	headerLines := 2 // header title + separator
	footerLines := 3 // navigation help lines
	contentHeight := c.height - headerLines - footerLines

	if contentHeight < 1 {
		contentHeight = 1
	}

	// Requests table
	filteredRequests := c.getFilteredRequests()
	if len(filteredRequests) == 0 {
		// Show empty message in the middle of available space
		emptyLines := contentHeight - 1
		if emptyLines > 0 {
			content.WriteString(strings.Repeat("\n", emptyLines/2))
		}
		content.WriteString(c.normalStyle.Render("No matching requests"))
	} else {
		// Table header
		content.WriteString(c.headerStyle.Render(fmt.Sprintf("%-8s %-6s %-40s %-10s %s",
			"METHOD", "STATUS", "URL", "DURATION", "PROCESS")) + "\n")
		content.WriteString(strings.Repeat("─", c.width) + "\n")

		// Calculate how many request rows we can show
		tableHeaderLines := 2 // method/status header + separator
		availableRequestLines := contentHeight - tableHeaderLines

		if availableRequestLines < 1 {
			availableRequestLines = 1
		}

		// Show most recent requests that fit in available space
		startIdx := len(filteredRequests) - availableRequestLines
		if startIdx < 0 {
			startIdx = 0
		}

		// Render request rows
		for i := startIdx; i < len(filteredRequests); i++ {
			req := filteredRequests[i]
			line := c.formatRequestRow(req)

			// Highlight if this is the selected request
			if i == c.selectedIndex {
				line = lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(line)
			}
			content.WriteString(line + "\n")
		}

		// If we have fewer requests than available lines, add spacing
		renderedLines := len(filteredRequests) - startIdx
		remainingLines := availableRequestLines - renderedLines
		if remainingLines > 0 {
			for i := 0; i < remainingLines; i++ {
				content.WriteString("\n")
			}
		}
	}

	// Add spacing before footer
	content.WriteString(strings.Repeat("\n", footerLines))

	// Navigation help (always at bottom)
	content.WriteString(c.normalStyle.Render("↑/↓ navigate, Enter details, f filter, Space auto-scroll"))
	content.WriteString("\n" + c.normalStyle.Render("Filter: "+c.filter+" | Auto-scroll: "+func() string {
		if c.autoScroll { return "ON" } else { return "OFF" }
	}()))
	content.WriteString("\n" + c.normalStyle.Render("Indicators: ❌ error, 🔐 auth, 📊 telemetry"))

	return content.String()
}

// renderDetailView renders the detailed view for selected request
func (c *WebViewComponent) renderDetailView() string {
	if c.selectedRequest == nil {
		return c.normalStyle.Render("No request selected")
	}

	var content strings.Builder
	req := c.selectedRequest

	// Header
	content.WriteString(c.headerStyle.Render("Request Details") + "\n\n")

	// Basic information
	content.WriteString(c.labelStyle.Render("Method: ") + c.detailStyle.Render(req.Method) + "\n")
	content.WriteString(c.labelStyle.Render("URL: ") + c.detailStyle.Render(req.URL) + "\n")
	content.WriteString(c.labelStyle.Render("Status: ") + c.formatStatus(req.StatusCode) + "\n")
	content.WriteString(c.labelStyle.Render("Duration: ") + c.detailStyle.Render(c.formatDuration(req.Duration)) + "\n")
	content.WriteString(c.labelStyle.Render("Time: ") + c.detailStyle.Render(req.Timestamp.Format("15:04:05")) + "\n")
	content.WriteString(c.labelStyle.Render("Process: ") + c.detailStyle.Render(req.ProcessName) + "\n")

	if req.Size > 0 {
		content.WriteString(c.labelStyle.Render("Size: ") + c.detailStyle.Render(c.formatBytes(req.Size)) + "\n")
	}

	if req.Error != "" {
		content.WriteString(c.labelStyle.Render("Error: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(req.Error) + "\n")
	}

	// Authentication section
	if req.HasAuth {
		content.WriteString("\n" + c.headerStyle.Render("🔐 Authentication") + "\n\n")
		content.WriteString(c.labelStyle.Render("Type: ") + c.detailStyle.Render(req.AuthType) + "\n")

		if req.JWTError != "" {
			content.WriteString(c.labelStyle.Render("JWT Error: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(req.JWTError) + "\n")
		} else if req.JWTClaims != nil && len(req.JWTClaims) > 0 {
			content.WriteString(c.labelStyle.Render("JWT Claims:") + "\n")
			c.renderJWTClaims(&content, req.JWTClaims)
		}
	}

	// Telemetry section
	if req.HasTelemetry {
		content.WriteString("\n" + c.headerStyle.Render("📊 Telemetry") + "\n\n")
		content.WriteString(c.detailStyle.Render("Telemetry data available for this request") + "\n")
		content.WriteString(c.labelStyle.Render("Session ID: ") + c.detailStyle.Render(req.SessionID) + "\n")
	}

	// Help
	content.WriteString("\n" + c.normalStyle.Render("Press Enter to return to list"))

	return content.String()
}

// renderDetailViewWithFooter renders the detailed view for selected request with proper height management
func (c *WebViewComponent) renderDetailViewWithFooter() string {
	if c.selectedRequest == nil {
		return c.normalStyle.Render("No request selected")
	}

	var content strings.Builder
	req := c.selectedRequest

	// Header
	content.WriteString(c.headerStyle.Render("Request Details") + "\n\n")

	// Basic information
	content.WriteString(c.labelStyle.Render("Method: ") + c.detailStyle.Render(req.Method) + "\n")
	content.WriteString(c.labelStyle.Render("URL: ") + c.detailStyle.Render(req.URL) + "\n")
	content.WriteString(c.labelStyle.Render("Status: ") + c.formatStatus(req.StatusCode) + "\n")
	content.WriteString(c.labelStyle.Render("Duration: ") + c.detailStyle.Render(c.formatDuration(req.Duration)) + "\n")
	content.WriteString(c.labelStyle.Render("Time: ") + c.detailStyle.Render(req.Timestamp.Format("15:04:05")) + "\n")
	content.WriteString(c.labelStyle.Render("Process: ") + c.detailStyle.Render(req.ProcessName) + "\n")

	if req.Size > 0 {
		content.WriteString(c.labelStyle.Render("Size: ") + c.detailStyle.Render(c.formatBytes(req.Size)) + "\n")
	}

	if req.Error != "" {
		content.WriteString(c.labelStyle.Render("Error: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(req.Error) + "\n")
	}

	// Authentication section
	if req.HasAuth {
		content.WriteString("\n" + c.headerStyle.Render("🔐 Authentication") + "\n\n")
		content.WriteString(c.labelStyle.Render("Type: ") + c.detailStyle.Render(req.AuthType) + "\n")

		if req.JWTError != "" {
			content.WriteString(c.labelStyle.Render("JWT Error: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(req.JWTError) + "\n")
		} else if req.JWTClaims != nil && len(req.JWTClaims) > 0 {
			content.WriteString(c.labelStyle.Render("JWT Claims:") + "\n")
			c.renderJWTClaims(&content, req.JWTClaims)
		}
	}

	// Telemetry section
	if req.HasTelemetry {
		content.WriteString("\n" + c.headerStyle.Render("📊 Telemetry") + "\n\n")
		content.WriteString(c.detailStyle.Render("Telemetry data available for this request") + "\n")
		content.WriteString(c.labelStyle.Render("Session ID: ") + c.detailStyle.Render(req.SessionID) + "\n")
	}

	// Calculate how many lines we've rendered so far
	renderedLines := strings.Count(content.String(), "\n") + 1
	availableHeight := c.height

	// Add spacing to push help text to bottom
	footerLines := 2 // help text + spacing
	remainingSpace := availableHeight - renderedLines - footerLines

	if remainingSpace > 0 {
		content.WriteString(strings.Repeat("\n", remainingSpace))
	}

	// Help text (always at bottom)
	content.WriteString("\n" + c.normalStyle.Render("Press Enter to return to list"))

	return content.String()
}

// renderSplitView renders the split view with list and detail side by side
func (c *WebViewComponent) renderSplitView() string {
	// Calculate split
	listWidth := int(float64(c.width) * 0.4)
	detailWidth := c.width - listWidth - 3 // Account for separator

	// Get filtered requests
	filteredRequests := c.getFilteredRequests()

	// Build list content with proper height management
	var listContent strings.Builder

	// Header for list side
	listContent.WriteString(c.headerStyle.Render("Web Requests") + "\n")
	listContent.WriteString(strings.Repeat("─", listWidth) + "\n")

	// Calculate available height for list content (account for header)
	listHeaderLines := 2
	availableListHeight := c.height - listHeaderLines - 1 // -1 for bottom margin

	if availableListHeight < 1 {
		availableListHeight = 1
	}

	if len(filteredRequests) == 0 {
		// Center empty message
		emptyLines := availableListHeight - 1
		if emptyLines > 0 {
			listContent.WriteString(strings.Repeat("\n", emptyLines/2))
		}
		listContent.WriteString(c.normalStyle.Render("No requests"))
	} else {
		// Show most recent requests that fit in available space
		startIdx := len(filteredRequests) - availableListHeight
		if startIdx < 0 {
			startIdx = 0
		}

		// Render request rows
		for i := startIdx; i < len(filteredRequests); i++ {
			req := filteredRequests[i]
			line := c.formatCompactRequestRow(req, listWidth)
			if i == c.selectedIndex {
				line = lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(line)
			}
			listContent.WriteString(line + "\n")
		}

		// Fill remaining space if we have fewer requests than available lines
		renderedLines := len(filteredRequests) - startIdx
		remainingLines := availableListHeight - renderedLines
		if remainingLines > 0 {
			for i := 0; i < remainingLines; i++ {
				listContent.WriteString("\n")
			}
		}
	}

	// Build detail content
	detailContent := c.renderDetailViewWithFooter()

	// Apply borders
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	listView := borderStyle.Width(listWidth).Height(c.height).Render(listContent.String())
	detailView := borderStyle.Width(detailWidth).Height(c.height).Render(detailContent)

	// Combine views
	return lipgloss.JoinHorizontal(lipgloss.Top, listView, " ", detailView)
}

// renderHeader renders the header with filter and status information
func (c *WebViewComponent) renderHeader() string {
	var header strings.Builder

	// Title and filter
	header.WriteString(c.headerStyle.Render("Web Proxy Requests") + " ")

	// Filter indicator
	activeFilterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	header.WriteString(filterStyle.Render("Filter: ") + activeFilterStyle.Render("["+c.filter+"]") + " ")

	// Auto-scroll indicator
	if !c.autoScroll {
		pausedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Background(lipgloss.Color("235"))
		header.WriteString(pausedStyle.Render("⏸ PAUSED"))
	}

	header.WriteString("\n" + strings.Repeat("─", c.width) + "\n")
	return header.String()
}

// formatCompactRequestRow formats a request row for compact display
func (c *WebViewComponent) formatCompactRequestRow(req *RequestInfo, maxWidth int) string {
	// Format time
	timeStr := req.Timestamp.Format("15:04:05")

	// Format status with color
	statusStr := fmt.Sprintf("%d", req.StatusCode)
	var statusStyle lipgloss.Style
	switch {
	case req.StatusCode >= 200 && req.StatusCode < 300:
		statusStyle = c.successStyle
	case req.StatusCode >= 300 && req.StatusCode < 400:
		statusStyle = c.redirectStyle
	case req.StatusCode >= 400 && req.StatusCode < 500:
		statusStyle = c.clientStyle
	case req.StatusCode >= 500:
		statusStyle = c.serverStyle
	default:
		statusStyle = c.normalStyle
	}

	// Truncate URL
	maxURLLength := maxWidth - len(timeStr) - len(statusStr) - 3 // spaces and padding
	url := req.URL
	if len(url) > maxURLLength && maxURLLength > 3 {
		url = url[:maxURLLength-3] + "..."
	}

	return fmt.Sprintf("%s %s %s",
		timeStr,
		statusStyle.Render(statusStr),
		url,
	)
}

// formatStatus formats status codes with color
func (c *WebViewComponent) formatStatus(status int) string {
	var style lipgloss.Style
	switch {
	case status >= 200 && status < 300:
		style = c.successStyle
	case status >= 300 && status < 400:
		style = c.redirectStyle
	case status >= 400 && status < 500:
		style = c.clientStyle
	case status >= 500:
		style = c.serverStyle
	default:
		style = c.normalStyle
	}
	return style.Render(fmt.Sprintf("%d", status))
}

// formatDuration formats duration for display
func (c *WebViewComponent) formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	} else if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
}

// renderJWTClaims renders JWT claims in a readable format
func (c *WebViewComponent) renderJWTClaims(content *strings.Builder, claims map[string]interface{}) {
	// Display common JWT claims in order
	claimOrder := []string{"sub", "iss", "aud", "exp", "iat", "nbf", "jti", "email", "name", "role", "scope"}
	displayedClaims := make(map[string]bool)

	for _, claim := range claimOrder {
		if value, exists := claims[claim]; exists {
			displayedClaims[claim] = true
			content.WriteString(fmt.Sprintf("  %s: %v\n", claim, value))
		}
	}

	// Display any remaining claims not in the ordered list
	for claim, value := range claims {
		if !displayedClaims[claim] {
			content.WriteString(fmt.Sprintf("  %s: %v\n", claim, value))
		}
	}
}

// formatBytes formats byte sizes for display
func (c *WebViewComponent) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}