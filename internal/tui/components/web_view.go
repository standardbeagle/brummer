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

// WebViewComponentNew is a stateless BubbleTea component for displaying HTTP proxy requests
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type WebViewComponentNew struct {
	BaseComponent

	// UI configuration only (no state)
	viewport viewport.Model

	// Styles
	headerStyle   lipgloss.Style
	successStyle  lipgloss.Style // 2xx
	redirectStyle lipgloss.Style // 3xx
	clientStyle   lipgloss.Style // 4xx
	serverStyle   lipgloss.Style // 5xx
	normalStyle   lipgloss.Style
	detailStyle   lipgloss.Style
	labelStyle    lipgloss.Style
	errorStyle    lipgloss.Style
	statsStyle    lipgloss.Style
}

// NewWebViewComponentNew creates a new stateless web view component
func NewWebViewComponentNew() *WebViewComponentNew {
	vp := viewport.New(80, 24)

	return &WebViewComponentNew{
		BaseComponent: NewBaseComponent(),
		viewport:      vp,

		// Initialize styles
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // Bright Blue
		successStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")),             // Bright Green
		redirectStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),             // Bright Yellow
		clientStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("13")),             // Bright Magenta
		serverStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")),              // Bright Red
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),             // White
		detailStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Background(lipgloss.Color("235")).Padding(1).Border(lipgloss.RoundedBorder()),
		labelStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8")),   // Dark Gray
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true), // Bright Red
		statsStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),            // Gray
	}
}

// ViewWithState renders the web view using external state (Redux/Elm compliant)
func (c *WebViewComponentNew) ViewWithState(masterData *state.MasterData) string {
	if !c.IsVisible() {
		return ""
	}

	if masterData == nil || masterData.WebViewState == nil {
		return c.renderErrorState("Web view state not available")
	}

	webViewState := masterData.WebViewState

	if webViewState.GetLoading() {
		return c.renderLoadingState()
	}

	if error := webViewState.GetError(); error != "" {
		return c.renderErrorState(error)
	}

	// Update viewport with current state
	c.updateViewportFromState(webViewState)

	// Get filtered requests
	filteredRequests := webViewState.GetFilteredRequests()

	if len(filteredRequests) == 0 {
		return c.renderEmptyState(webViewState)
	}

	// Render based on whether detail view is shown
	if webViewState.IsShowingDetail() && webViewState.GetSelectedRequest() != nil {
		return c.renderDetailView(webViewState)
	}

	return c.renderListView(webViewState, filteredRequests)
}

// renderListView renders the main request list view
func (c *WebViewComponentNew) renderListView(webViewState *state.WebViewState, requests []state.RequestInfo) string {
	var content strings.Builder

	// Header with filter and stats
	content.WriteString(c.renderHeader(webViewState))

	// Request table
	tableContent := c.renderRequestTable(requests, webViewState.GetSelectedIndex())
	content.WriteString(tableContent)

	// Set viewport content
	c.viewport.SetContent(content.String())

	return c.viewport.View()
}

// renderDetailView renders the detailed view of a selected request
func (c *WebViewComponentNew) renderDetailView(webViewState *state.WebViewState) string {
	selectedRequest := webViewState.GetSelectedRequest()
	if selectedRequest == nil {
		return c.renderListView(webViewState, webViewState.GetFilteredRequests())
	}

	var content strings.Builder

	// Header
	content.WriteString(c.renderHeader(webViewState))

	// Request details
	details := []string{
		fmt.Sprintf("Method:     %s", selectedRequest.Method),
		fmt.Sprintf("URL:        %s", selectedRequest.URL),
		fmt.Sprintf("Path:       %s", selectedRequest.Path),
		fmt.Sprintf("Status:     %d", selectedRequest.StatusCode),
		fmt.Sprintf("Duration:   %v", selectedRequest.Duration),
		fmt.Sprintf("Process:    %s", selectedRequest.ProcessName),
		fmt.Sprintf("Timestamp:  %s", selectedRequest.Timestamp.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("Size:       %d bytes", selectedRequest.Size),
	}

	if selectedRequest.Error != "" {
		details = append(details, fmt.Sprintf("Error:      %s", selectedRequest.Error))
	}

	// Authentication details
	if selectedRequest.HasAuth {
		details = append(details, fmt.Sprintf("Auth Type:  %s", selectedRequest.AuthType))
		if selectedRequest.JWTError != "" {
			details = append(details, fmt.Sprintf("JWT Error:  %s", selectedRequest.JWTError))
		}
	}

	// Telemetry details
	if selectedRequest.HasTelemetry {
		details = append(details, fmt.Sprintf("Session ID: %s", selectedRequest.SessionID))
	}

	// Render details
	for _, line := range details {
		content.WriteString(c.labelStyle.Render(line))
		content.WriteString("\n")
	}

	// Add note about how to exit detail view
	content.WriteString("\n")
	content.WriteString(c.statsStyle.Render("Press Enter to return to list view"))

	return c.detailStyle.Render(content.String())
}

// renderHeader renders the header with current filter and statistics
func (c *WebViewComponentNew) renderHeader(webViewState *state.WebViewState) string {
	var parts []string

	// Title and filter
	title := "HTTP Requests"
	filter := webViewState.GetFilter()
	if filter != "all" {
		title += fmt.Sprintf(" (%s)", filter)
	}

	if webViewState.IsAutoScroll() {
		title += " [Auto-scroll]"
	}

	parts = append(parts, c.headerStyle.Render(title))

	// Statistics
	totalRequests := webViewState.GetTotalRequests()
	filteredRequests := len(webViewState.GetFilteredRequests())
	requestsByStatus := webViewState.GetRequestsByStatus()

	stats := fmt.Sprintf("Total: %d | Filtered: %d", totalRequests, filteredRequests)
	if len(requestsByStatus) > 0 {
		var statusParts []string
		for status, count := range requestsByStatus {
			statusParts = append(statusParts, fmt.Sprintf("%s: %d", status, count))
		}
		stats += fmt.Sprintf(" | %s", strings.Join(statusParts, ", "))
	}

	parts = append(parts, c.statsStyle.Render(stats))

	// Help text
	help := "↑↓ Navigate • Enter Details • f Filter • Space Auto-scroll • q Quit"
	parts = append(parts, c.statsStyle.Render(help))

	return lipgloss.JoinVertical(lipgloss.Left, parts...) + "\n"
}

// renderRequestTable renders the request table
func (c *WebViewComponentNew) renderRequestTable(requests []state.RequestInfo, selectedIndex int) string {
	if len(requests) == 0 {
		return c.statsStyle.Render("No requests to display")
	}

	// Table header
	header := fmt.Sprintf("%-8s %-12s %-40s %-8s %-10s",
		"Method", "Process", "URL", "Status", "Duration")

	var content strings.Builder
	content.WriteString(c.labelStyle.Render(header))
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", 80))
	content.WriteString("\n")

	// Table rows
	for i, req := range requests {
		row := c.formatRequestRow(req, i == selectedIndex)
		content.WriteString(row)
		content.WriteString("\n")
	}

	return content.String()
}

// formatRequestRow formats a single request row
func (c *WebViewComponentNew) formatRequestRow(req state.RequestInfo, isSelected bool) string {
	// Truncate URL if too long
	maxURLLength := 40
	url := req.URL
	if len(url) > maxURLLength {
		url = url[:maxURLLength-3] + "..."
	}

	// Truncate process name if too long
	maxProcessLength := 12
	process := req.ProcessName
	if len(process) > maxProcessLength {
		process = process[:maxProcessLength-3] + "..."
	}

	// Choose style based on status code
	var style lipgloss.Style
	switch {
	case req.StatusCode >= 200 && req.StatusCode < 300:
		style = c.successStyle
	case req.StatusCode >= 300 && req.StatusCode < 400:
		style = c.redirectStyle
	case req.StatusCode >= 400 && req.StatusCode < 500:
		style = c.clientStyle
	case req.StatusCode >= 500:
		style = c.serverStyle
	default:
		style = c.normalStyle
	}

	// Highlight selected row
	if isSelected {
		style = style.Background(lipgloss.Color("240")).Bold(true)
	}

	// Format duration
	duration := ""
	if req.Duration > 0 {
		if req.Duration < time.Millisecond {
			duration = fmt.Sprintf("%.0fμs", float64(req.Duration.Nanoseconds())/1000)
		} else if req.Duration < time.Second {
			duration = fmt.Sprintf("%.0fms", float64(req.Duration.Nanoseconds())/1000000)
		} else {
			duration = fmt.Sprintf("%.1fs", req.Duration.Seconds())
		}
	}

	// Handle error status
	statusText := fmt.Sprintf("%d", req.StatusCode)
	if req.Error != "" {
		statusText = "Error"
		style = c.errorStyle
	}

	row := fmt.Sprintf("%-8s %-12s %-40s %-8s %-10s",
		req.Method,
		process,
		url,
		statusText,
		duration,
	)

	return style.Render(row)
}

// renderEmptyState renders the empty state when no requests are available
func (c *WebViewComponentNew) renderEmptyState(webViewState *state.WebViewState) string {
	var content strings.Builder

	content.WriteString(c.renderHeader(webViewState))
	content.WriteString("\n")

	message := "No HTTP requests captured yet"
	if webViewState.GetFilter() != "all" {
		message = fmt.Sprintf("No requests match filter: %s", webViewState.GetFilter())
	}

	content.WriteString(c.statsStyle.Render(message))
	content.WriteString("\n\n")
	content.WriteString(c.statsStyle.Render("Requests will appear here when applications make HTTP calls"))

	// Set viewport content
	c.viewport.SetContent(content.String())

	return c.viewport.View()
}

// renderLoadingState renders the loading state
func (c *WebViewComponentNew) renderLoadingState() string {
	content := c.headerStyle.Render("HTTP Requests") + "\n"
	content += strings.Repeat("─", 80) + "\n"
	content += c.statsStyle.Render("Loading request data...")

	c.viewport.SetContent(content)
	return c.viewport.View()
}

// renderErrorState renders the error state
func (c *WebViewComponentNew) renderErrorState(message string) string {
	content := c.headerStyle.Render("HTTP Requests") + "\n"
	content += strings.Repeat("─", 80) + "\n"
	content += c.errorStyle.Render(fmt.Sprintf("Error: %s", message))

	c.viewport.SetContent(content)
	return c.viewport.View()
}

// updateViewportFromState updates the viewport based on current state
func (c *WebViewComponentNew) updateViewportFromState(webViewState *state.WebViewState) {
	width, height := webViewState.GetSize()
	c.viewport.Width = width
	c.viewport.Height = height - 4 // Account for header

	// Update viewport offset from state
	offset := webViewState.GetViewportOffset()
	if offset >= 0 && offset < c.viewport.TotalLineCount() {
		c.viewport.YOffset = offset
	}
}

// SetSize updates component dimensions
func (c *WebViewComponentNew) SetSize(width, height int) {
	c.viewport.Width = width
	c.viewport.Height = height - 4 // Account for header
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *WebViewComponentNew) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.viewport.Width = width
	c.viewport.Height = contentHeight
}

// View renders the component (legacy method for compatibility)
func (c *WebViewComponentNew) View() string {
	if !c.IsVisible() {
		return ""
	}
	return "" // Rendering is done via ViewWithState
}

// Update handles messages (minimal - stateless component)
func (c *WebViewComponentNew) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil
	}

	// Update viewport
	c.viewport, cmd = c.viewport.Update(msg)
	return c, cmd
}

// Init initializes the component
func (c *WebViewComponentNew) Init() tea.Cmd {
	return nil
}