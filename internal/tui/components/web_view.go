package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// WebViewComponentRefactored represents a refactored web view using proper Bubble Tea components
type WebViewComponentRefactored struct {
	BaseComponent
	width  int
	height int
	list   list.Model

	// Lipgloss styles
	headerStyle     lipgloss.Style
	statsStyle      lipgloss.Style
	selectedStyle   lipgloss.Style
	successStyle    lipgloss.Style
	redirectStyle   lipgloss.Style
	clientStyle     lipgloss.Style
	serverStyle     lipgloss.Style
	normalStyle     lipgloss.Style
	errorStyle      lipgloss.Style
	panelStyle      lipgloss.Style
	detailHeaderStyle lipgloss.Style
	detailStyle     lipgloss.Style
}

// Helper functions for icons and formatting

// getStatusIcon returns an icon and color for HTTP status codes
func getStatusIcon(status int) (string, lipgloss.Color) {
	switch {
	case status >= 200 && status < 300:
		return "✅", lipgloss.Color("10") // Green
	case status >= 300 && status < 400:
		return "↩️", lipgloss.Color("11") // Yellow
	case status >= 400 && status < 500:
		return "❌", lipgloss.Color("196") // Red
	case status >= 500:
		return "🚨", lipgloss.Color("196") // Red
	default:
		return "❓", lipgloss.Color("245") // Gray
	}
}

// getRequestTypeIcon returns an icon for different request types based on URL and content type
func getRequestTypeIcon(url, contentType string) string {
	// Check file extension first
	if strings.Contains(url, ".js") || strings.HasSuffix(url, "/js") {
		return "📜" // JavaScript
	}
	if strings.Contains(url, ".css") || strings.HasSuffix(url, "/css") {
		return "🎨" // CSS
	}
	if strings.Contains(url, ".png") || strings.Contains(url, ".jpg") || strings.Contains(url, ".jpeg") ||
		strings.Contains(url, ".gif") || strings.Contains(url, ".svg") || strings.Contains(url, ".webp") {
		return "🖼️" // Image
	}
	if strings.Contains(url, ".html") || strings.HasSuffix(url, "/") {
		return "📄" // HTML page
	}
	if strings.Contains(url, ".json") || strings.Contains(url, "/api/") {
		return "🔌" // API
	}
	if strings.Contains(url, ".woff") || strings.Contains(url, ".ttf") {
		return "🔤" // Font
	}
	if strings.Contains(url, ".pdf") {
		return "📋" // PDF
	}

	// Check content type if available
	if strings.Contains(contentType, "application/json") {
		return "🔌" // API
	}
	if strings.Contains(contentType, "text/html") {
		return "📄" // HTML page
	}
	if strings.Contains(contentType, "text/css") {
		return "🎨" // CSS
	}
	if strings.Contains(contentType, "application/javascript") {
		return "📜" // JavaScript
	}
	if strings.Contains(contentType, "image/") {
		return "🖼️" // Image
	}

	return "📎" // Generic/Other
}

// formatDuration formats duration in a compact way
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "0ms"
	} else if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1000000)
	} else if d < 10*time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
}

// cleanURL removes protocol, domain, and port from URLs for cleaner display
// Since all proxy requests will have the same domain/port, we show only the path
func cleanURL(fullURL string) string {
	if fullURL == "" {
		return ""
	}

	// Parse URL to extract path
	// Handle common patterns:
	// http://localhost:3000/path -> /path
	// http://localhost:20888/path -> /path
	// https://example.com/path -> /path

	// Find the start of the path (after //)
	protocolIndex := strings.Index(fullURL, "://")
	if protocolIndex == -1 {
		// No protocol found, might already be a path
		return fullURL
	}

	// Find the domain part (://)
	domainStart := protocolIndex + 3
	if domainStart >= len(fullURL) {
		return fullURL
	}

	// Find the first / after the domain
	pathStart := strings.Index(fullURL[domainStart:], "/")
	if pathStart == -1 {
		// No path, return root
		return "/"
	}

	// Return the path part
	path := fullURL[domainStart+pathStart:]
	if path == "" {
		return "/"
	}

	return path
}

// CompactDelegate creates a custom delegate for single-line items
type CompactDelegate struct {
	list.DefaultDelegate
}

func (d CompactDelegate) Height() int {
	return 1 // Force single line height
}

func (d CompactDelegate) Spacing() int {
	return 0 // No spacing between items
}

func (d CompactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	// Handle key messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up"))):
			if m.Index() > 0 {
				m.Select(m.Index() - 1)
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down"))):
			if m.Index() < len(m.Items())-1 {
				m.Select(m.Index() + 1)
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Enter key shows details (handled by parent)
			return m.NewStatusMessage("Selected: " + m.SelectedItem().(WebRequestItemWrapper).URL)
		case key.Matches(msg, key.NewBinding(key.WithKeys("q"))):
			// Let parent handle quit
			return nil
		}
	}

	// Use default handling for other keys
	return d.DefaultDelegate.Update(msg, m)
}

// NewWebViewComponentRefactored creates a new web view component using Bubble Tea list
func NewWebViewComponentRefactored() *WebViewComponentRefactored {
	// Create compact delegate for single-line display
	delegate := &CompactDelegate{
		DefaultDelegate: list.NewDefaultDelegate(),
	}
	delegate.ShowDescription = false // Don't render description lines at all
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("33")).Bold(true).Underline(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("33")).Bold(true)

	// Create the list
	requestsList := list.New([]list.Item{}, delegate, 0, 0)
	requestsList.Title = "HTTP Requests"
	requestsList.SetShowStatusBar(false)
	requestsList.SetFilteringEnabled(false)
	requestsList.SetShowPagination(false)
	requestsList.SetShowHelp(false)

	// Configure for compact display
	requestsList.DisableQuitKeybindings() // Allow 'q' to be used by parent

	return &WebViewComponentRefactored{
		BaseComponent: NewBaseComponent(),
		width:         80,
		height:        24,
		list:          requestsList,

		// Initialize styles
		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")), // Bright Blue

		statsStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")), // Gray

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")). // Bright Blue
			Bold(true).
			Underline(true),

		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")), // Bright Green

		redirectStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")), // Bright Yellow

		clientStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")), // Bright Magenta

		serverStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")), // Bright Red

		normalStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")), // White

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Bright Red

		panelStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")), // Dark Gray

		detailHeaderStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")), // Bright Blue

		detailStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")), // White
	}
}

// SetSize updates the component dimensions
func (c *WebViewComponentRefactored) SetSize(width, height int) {
	c.width = width
	c.height = height

	// Update list dimensions
	listWidth := width
	listHeight := height - 6 // Account for header and stats
	c.list.SetSize(listWidth, listHeight)
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *WebViewComponentRefactored) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.width = width
	c.height = height

	// Update list dimensions
	c.list.SetSize(width, contentHeight-6) // Account for header and stats
}

// ViewWithState renders the web view using external state (Redux/Elm compliant)
func (c *WebViewComponentRefactored) ViewWithState(masterData *state.MasterData) string {
	if !c.IsVisible() {
		return ""
	}

	if masterData == nil || masterData.WebViewState == nil {
		return c.renderErrorState("Web view state not available")
	}

	webViewState := masterData.WebViewState

	// Update list with current state
	c.updateListFromState(webViewState)

	// Check if we should use side-by-side layout
	useSidePanel := c.width >= 120 && webViewState.GetSelectedRequest() != nil

	if useSidePanel {
		return c.renderSideBySideView(webViewState)
	}

	// Full-width list view
	return c.renderFullWidthView(webViewState)
}

// updateListFromState updates the list items from the current state
func (c *WebViewComponentRefactored) updateListFromState(webViewState *state.WebViewState) {
	requests := webViewState.GetFilteredRequests()
	items := make([]list.Item, len(requests))

	for i, req := range requests {
		items[i] = WebRequestItemWrapper{
			RequestInfo: req,
			IsSelected:  i == webViewState.GetSelectedIndex(),
		}
	}

	c.list.SetItems(items)

	// Set selected index
	if webViewState.GetSelectedIndex() >= 0 && webViewState.GetSelectedIndex() < len(items) {
		c.list.Select(webViewState.GetSelectedIndex())
	}
}

// renderFullWidthView renders the full-width list view
func (c *WebViewComponentRefactored) renderFullWidthView(webViewState *state.WebViewState) string {
	// Create main layout with lipgloss
	header := c.renderHeader(webViewState)
	stats := c.renderStats(webViewState)
	listView := c.list.View()
	help := c.renderHelp()

	// Use lipgloss Join to stack components vertically
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		stats,
		listView,
		help,
	)

	return content
}

// renderSideBySideView renders list on left and details on right
func (c *WebViewComponentRefactored) renderSideBySideView(webViewState *state.WebViewState) string {
	// Calculate widths (60% list, 40% details)
	listWidth := (c.width * 60) / 100
	detailWidth := c.width - listWidth - 3 // 3 for borders and spacing

	if detailWidth < 30 {
		detailWidth = 30
		listWidth = c.width - detailWidth - 3
	}

	// Left panel: Request list
	listHeight := c.height - 6 // Account for header and stats
	c.list.SetSize(listWidth, listHeight)

	leftPanelStyle := lipgloss.NewStyle().
		Width(listWidth).
		Height(c.height)

	leftPanelContent := lipgloss.JoinVertical(lipgloss.Left,
		c.renderHeader(webViewState),
		c.renderStats(webViewState),
		c.list.View(),
	)

	leftPanel := leftPanelStyle.Render(leftPanelContent)

	// Right panel: Details
	detailPanel := c.renderDetailPanel(webViewState, detailWidth)

	// Use lipgloss JoinHorizontal for side-by-side layout
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, detailPanel)
}

// renderDetailPanel renders the detail panel
func (c *WebViewComponentRefactored) renderDetailPanel(webViewState *state.WebViewState, width int) string {
	selectedRequest := webViewState.GetSelectedRequest()
	if selectedRequest == nil {
		return c.panelStyle.Width(width).Height(c.height).Render(
			lipgloss.JoinVertical(lipgloss.Center,
				"\n",
				c.statsStyle.Render("No request selected"),
				c.statsStyle.Render("Use ↑↓ to select"),
			),
		)
	}

	// Build detail content
	var details []string
	details = append(details, fmt.Sprintf("Method:    %s", selectedRequest.Method))
	details = append(details, fmt.Sprintf("URL:       %s", selectedRequest.URL))
	details = append(details, fmt.Sprintf("Path:      %s", selectedRequest.Path))
	details = append(details, fmt.Sprintf("Status:    %d", selectedRequest.StatusCode))
	// Format duration properly for detail panel
	durationStr := ""
	if selectedRequest.Duration > 0 {
		if selectedRequest.Duration < time.Millisecond {
			durationStr = fmt.Sprintf("%.0fμs", float64(selectedRequest.Duration.Nanoseconds())/1000)
		} else if selectedRequest.Duration < time.Second {
			durationStr = fmt.Sprintf("%.0fms", float64(selectedRequest.Duration.Nanoseconds())/1000000)
		} else {
			durationStr = fmt.Sprintf("%.1fs", selectedRequest.Duration.Seconds())
		}
	} else {
		durationStr = "<1ms"
	}
	details = append(details, fmt.Sprintf("Duration:  %s", durationStr))
	details = append(details, fmt.Sprintf("Process:   %s", selectedRequest.ProcessName))
	details = append(details, fmt.Sprintf("Timestamp: %s", selectedRequest.Timestamp.Format("2006-01-02 15:04:05")))
	details = append(details, fmt.Sprintf("Size:      %d bytes", selectedRequest.Size))

	if selectedRequest.Error != "" {
		details = append(details, "")
		details = append(details, c.errorStyle.Render(fmt.Sprintf("Error: %s", selectedRequest.Error)))
	}

	// Authentication details
	if selectedRequest.HasAuth {
		details = append(details, "")
		details = append(details, "Authentication:")
		details = append(details, fmt.Sprintf("  Type: %s", selectedRequest.AuthType))
		if selectedRequest.JWTError != "" {
			details = append(details, fmt.Sprintf("  JWT Error: %s", selectedRequest.JWTError))
		}
	}

	// Performance metrics details
	if selectedRequest.HasTelemetry && len(selectedRequest.PerformanceMetrics) > 0 {
		details = append(details, "")
		details = append(details, "Performance:")

		// Display key performance metrics
		if ttfb, ok := selectedRequest.PerformanceMetrics["responseTime"].(float64); ok {
			details = append(details, fmt.Sprintf("  TTFB: %.1fms", ttfb))
		}
		if domLoad, ok := selectedRequest.PerformanceMetrics["domContentLoadedTime"].(float64); ok {
			details = append(details, fmt.Sprintf("  DOM Loaded: %.1fms", domLoad))
		}
		if loadTime, ok := selectedRequest.PerformanceMetrics["loadCompleteTime"].(float64); ok && loadTime > 0 {
			details = append(details, fmt.Sprintf("  Load Complete: %.1fms", loadTime))
		}
		if firstPaint, ok := selectedRequest.PerformanceMetrics["first_paint"].(float64); ok {
			details = append(details, fmt.Sprintf("  First Paint: %.1fms", firstPaint))
		}
		if firstContentfulPaint, ok := selectedRequest.PerformanceMetrics["first_contentful_paint"].(float64); ok {
			details = append(details, fmt.Sprintf("  First Contentful Paint: %.1fms", firstContentfulPaint))
		}

		// Show memory usage if available
		if memUsed, ok := selectedRequest.PerformanceMetrics["usedJSHeapSize"].(float64); ok {
			details = append(details, fmt.Sprintf("  Memory Used: %.1fMB", memUsed/1024/1024))
		}
	}

	// Session details
	if selectedRequest.HasTelemetry && selectedRequest.SessionID != "" {
		details = append(details, "")
		details = append(details, fmt.Sprintf("Session: %s", selectedRequest.SessionID))
	}

	// Request Headers
	if len(selectedRequest.RequestHeaders) > 0 {
		details = append(details, "")
		details = append(details, c.detailHeaderStyle.Render("Request Headers:"))
		// Sort header keys for stable display order (prevent bouncing)
		reqHeaderKeys := make([]string, 0, len(selectedRequest.RequestHeaders))
		for key := range selectedRequest.RequestHeaders {
			reqHeaderKeys = append(reqHeaderKeys, key)
		}
		sort.Strings(reqHeaderKeys)
		for _, key := range reqHeaderKeys {
			headerLine := fmt.Sprintf("  %s: %s", key, selectedRequest.RequestHeaders[key])
			details = append(details, headerLine)
		}
	}

	// Response Headers
	if len(selectedRequest.ResponseHeaders) > 0 {
		details = append(details, "")
		details = append(details, c.detailHeaderStyle.Render("Response Headers:"))
		// Sort header keys for stable display order (prevent bouncing)
		respHeaderKeys := make([]string, 0, len(selectedRequest.ResponseHeaders))
		for key := range selectedRequest.ResponseHeaders {
			respHeaderKeys = append(respHeaderKeys, key)
		}
		sort.Strings(respHeaderKeys)
		for _, key := range respHeaderKeys {
			headerLine := fmt.Sprintf("  %s: %s", key, selectedRequest.ResponseHeaders[key])
			details = append(details, headerLine)
		}
	}

	// Request Body
	if selectedRequest.RequestBody != "" {
		details = append(details, "")
		details = append(details, c.detailHeaderStyle.Render("Request Body:"))
		// Split body into lines for better display
		bodyLines := strings.Split(selectedRequest.RequestBody, "\n")
		for _, line := range bodyLines {
			details = append(details, "  "+line)
		}
	}

	// Response Body
	if selectedRequest.ResponseBody != "" {
		details = append(details, "")
		details = append(details, c.detailHeaderStyle.Render("Response Body:"))
		// Split body into lines for better display
		bodyLines := strings.Split(selectedRequest.ResponseBody, "\n")
		for _, line := range bodyLines {
			details = append(details, "  "+line)
		}
	}

	// Format detail lines with proper width
	var formattedDetails []string
	for _, line := range details {
		if len(line) > width-4 {
			// Simple truncation - could be improved with word wrapping
			line = line[:width-7] + "..."
		}
		formattedDetails = append(formattedDetails, line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		c.detailHeaderStyle.Render("Request Details"),
		strings.Repeat("─", width-4),
		"",
		lipgloss.JoinVertical(lipgloss.Left, formattedDetails...),
	)

	return c.panelStyle.Width(width).Height(c.height).Render(content)
}

// renderHeader renders the header with filter information
func (c *WebViewComponentRefactored) renderHeader(webViewState *state.WebViewState) string {
	title := "HTTP Requests"
	filter := webViewState.GetFilter()
	if filter != "all" {
		title += fmt.Sprintf(" (filtered: %s)", filter)
	}

	// Use lipgloss for header styling
	return c.headerStyle.Render(title)
}

// renderStats renders statistics
func (c *WebViewComponentRefactored) renderStats(webViewState *state.WebViewState) string {
	total := webViewState.GetTotalRequests()
	filtered := len(webViewState.GetFilteredRequests())

	var stats string
	if total == filtered {
		stats = fmt.Sprintf("%d requests", total)
	} else {
		stats = fmt.Sprintf("%d of %d requests (filtered)", filtered, total)
	}

	return c.statsStyle.Render(stats)
}

// renderHelp renders help text
func (c *WebViewComponentRefactored) renderHelp() string {
	help := "↑↓ Navigate • Enter Full Detail • a Send to AI Coder • f Filter • c Clear • Space Auto-scroll • q Quit"
	return c.statsStyle.Render(help)
}

// renderErrorState renders the error state
func (c *WebViewComponentRefactored) renderErrorState(message string) string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		c.headerStyle.Render("HTTP Requests"),
		strings.Repeat("─", c.width),
		c.errorStyle.Render(fmt.Sprintf("Error: %s", message)),
	)

	return content
}

// WebRequestItemWrapper wraps RequestInfo for list.Item interface
type WebRequestItemWrapper struct {
	state.RequestInfo
	IsSelected bool
}

func (i WebRequestItemWrapper) FilterValue() string {
	return fmt.Sprintf("%s %s %s %d", i.Method, i.URL, i.ProcessName, i.StatusCode)
}

func (i WebRequestItemWrapper) Title() string {
	// Get status icon
	statusIcon, _ := getStatusIcon(i.StatusCode)

	// Get request type icon
	typeIcon := getRequestTypeIcon(i.URL, "") // Content type not available in wrapper

	// Add telemetry indicator
	telemetryIcon := ""
	if i.HasTelemetry {
		telemetryIcon = "📊" // Telemetry available
	}

	// Format duration
	duration := formatDuration(i.Duration)

	// Build compact format: [status] [type] [method] [path] [duration] [telemetry]
	// Example: ✅ 📄 GET /api/users 125ms 📊

	// Clean URL to remove protocol, domain, and port
	cleanURL := cleanURL(i.URL)

	// Truncate URL for compact display
	maxURLLength := 25
	url := cleanURL
	if len(url) > maxURLLength {
		url = url[:maxURLLength-3] + "..."
	}

	// Build the line
	parts := []string{
		statusIcon,
		typeIcon,
		i.Method,
		url,
		duration,
	}

	// Add telemetry if available
	if telemetryIcon != "" {
		parts = append(parts, telemetryIcon)
	}

	// Join parts with spacing
	title := strings.Join(parts, " ")

	// Apply status color to the status icon
	// Note: In a real implementation, we'd need to handle colored text in the title
	// For now, we'll use a simple format

	return title
}

func (i WebRequestItemWrapper) Description() string {
	// Description is hidden for compact display
	return ""
}