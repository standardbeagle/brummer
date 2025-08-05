package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/proxy"
)

// proxyRequestItem implements list.Item for proxy requests
type proxyRequestItem struct {
	Request proxy.Request
}

func (i proxyRequestItem) FilterValue() string {
	return i.Request.URL + " " + i.Request.Method
}

func (i proxyRequestItem) Title() string {
	// Basic title - actual rendering with truncation is handled in delegate
	return fmt.Sprintf("%s %d %s %s",
		i.Request.StartTime.Format("15:04:05"),
		i.Request.StatusCode,
		i.Request.Method,
		i.Request.URL)
}

func (i proxyRequestItem) Description() string {
	if i.Request.Error != "" {
		return "Error: " + i.Request.Error
	}
	if i.Request.Size > 0 {
		return fmt.Sprintf("Size: %s", formatBytes(i.Request.Size))
	}
	return fmt.Sprintf("Duration: %dms", i.Request.Duration.Milliseconds())
}

// proxyRequestDelegate implements list.ItemDelegate for proxy requests
type proxyRequestDelegate struct{}

func (d proxyRequestDelegate) Height() int                               { return 1 }
func (d proxyRequestDelegate) Spacing() int                              { return 0 }
func (d proxyRequestDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d proxyRequestDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if item, ok := listItem.(proxyRequestItem); ok {
		// Calculate available width for URL based on list width
		listWidth := m.Width()

		// For very narrow terminals, use a compact format
		if listWidth < 50 {
			// Compact format: "HH:MM STATUS URL"
			url := item.Request.URL

			// Calculate actual space needed: time(5) + space(1) + status(3) + space(1) = 10 chars
			timeStr := item.Request.StartTime.Format("15:04")
			statusStr := fmt.Sprintf("%d", item.Request.StatusCode)
			reservedSpace := len(timeStr) + 1 + len(statusStr) + 1

			maxURLLength := listWidth - reservedSpace
			if maxURLLength < 3 {
				// If we can't fit even "...", just show status
				line := fmt.Sprintf("%s %s", timeStr, statusStr)
				var str string
				if index == m.Index() {
					str = lipgloss.NewStyle().Background(lipgloss.Color("240")).Render(line)
				} else {
					str = line
				}
				fmt.Fprint(w, str)
				return
			}

			if len(url) > maxURLLength {
				if maxURLLength <= 3 {
					url = "..."
				} else {
					url = url[:maxURLLength-3] + "..."
				}
			}

			line := fmt.Sprintf("%s %s %s", timeStr, statusStr, url)

			var str string
			if index == m.Index() {
				str = lipgloss.NewStyle().Background(lipgloss.Color("240")).Render(line)
			} else {
				str = line
			}
			fmt.Fprint(w, str)
			return
		}

		// Standard format for wider terminals
		// Fixed parts: time(8) + space + status(3) + space + method(7 max) + space + indicators(6 max) + padding(4)
		timeWidth := 8       // "15:04:05"
		statusWidth := 3     // "200"
		methodWidth := 7     // "DELETE" (longest common method)
		indicatorsWidth := 6 // " ❌ 🔐 📊" (worst case)
		spacesWidth := 4     // spaces between elements
		paddingWidth := 4    // general padding/margins

		fixedWidth := timeWidth + statusWidth + methodWidth + indicatorsWidth + spacesWidth + paddingWidth

		// Available width for URL with safety checks
		maxURLLength := listWidth - fixedWidth
		if maxURLLength < 10 {
			maxURLLength = 10 // Reasonable minimum for readability
		}

		url := item.Request.URL
		if len(url) > maxURLLength {
			if maxURLLength <= 3 {
				url = "..." // Fallback for extremely narrow cases
			} else {
				url = url[:maxURLLength-3] + "..."
			}
		}

		// Build the line
		line := fmt.Sprintf("%s %d %s %s",
			item.Request.StartTime.Format("15:04:05"),
			item.Request.StatusCode,
			item.Request.Method,
			url)

		// Add indicators
		if item.Request.Error != "" {
			line += " ❌"
		}
		if item.Request.HasAuth {
			line += " 🔐"
		}
		if item.Request.HasTelemetry {
			line += " 📊"
		}

		var str string
		if index == m.Index() {
			// Selected item - highlighted
			str = lipgloss.NewStyle().Background(lipgloss.Color("240")).Render(line)
		} else {
			// Normal item
			str = line
		}
		fmt.Fprint(w, str)
	}
}

// WebViewController manages the web view state and rendering
type WebViewController struct {
	webRequestsList   list.Model
	webDetailViewport viewport.Model
	webFilter         string
	webAutoScroll     bool
	selectedRequest   *proxy.Request
	lastWebCount      int

	// Dependencies injected from parent Model
	proxyServer  *proxy.Server
	width        int
	height       int
	headerHeight int
	footerHeight int
}

// NewWebViewController creates a new web view controller
func NewWebViewController(proxyServer *proxy.Server) *WebViewController {
	webRequestsList := list.New([]list.Item{}, proxyRequestDelegate{}, 0, 0)
	webRequestsList.Title = "Web Proxy Requests"
	webRequestsList.SetShowStatusBar(false)
	webRequestsList.SetShowTitle(false)
	webRequestsList.SetShowHelp(false)
	webRequestsList.SetShowPagination(false)
	webRequestsList.DisableQuitKeybindings()

	return &WebViewController{
		webRequestsList:   webRequestsList,
		webDetailViewport: viewport.New(0, 0),
		webFilter:         "all",
		webAutoScroll:     true,
		proxyServer:       proxyServer,
	}
}

// UpdateSize updates the viewport and list dimensions
func (v *WebViewController) UpdateSize(width, height, headerHeight, footerHeight int) {
	v.width = width
	v.height = height
	v.headerHeight = headerHeight
	v.footerHeight = footerHeight

	// Update sizes based on current layout
	if width < 100 {
		// Narrow view - full width list
		contentHeight := height - headerHeight - footerHeight - 3 // Account for filter headers
		v.webRequestsList.SetSize(width, contentHeight)
	} else {
		// Split view
		listWidth := int(float64(width) * 0.4)
		detailWidth := width - listWidth - 3
		contentHeight := height - headerHeight - footerHeight - 1

		v.webRequestsList.SetSize(listWidth-2, contentHeight-2)
		v.webDetailViewport.Width = detailWidth - 2
		v.webDetailViewport.Height = contentHeight - 2
	}
}

// SetWebFilter sets the current web filter
func (v *WebViewController) SetWebFilter(filter string) {
	v.webFilter = filter
}

// GetWebFilter returns the current web filter
func (v *WebViewController) GetWebFilter() string {
	return v.webFilter
}

// ToggleWebAutoScroll toggles auto-scroll behavior
func (v *WebViewController) ToggleWebAutoScroll() {
	v.webAutoScroll = !v.webAutoScroll
}

// IsWebAutoScrollEnabled returns whether auto-scroll is enabled
func (v *WebViewController) IsWebAutoScrollEnabled() bool {
	return v.webAutoScroll
}

// SetWebAutoScroll sets auto-scroll behavior
func (v *WebViewController) SetWebAutoScroll(enabled bool) {
	v.webAutoScroll = enabled
}

// GetWebRequestsList returns the web requests list for direct manipulation
func (v *WebViewController) GetWebRequestsList() *list.Model {
	return &v.webRequestsList
}

// GetSelectedRequest returns the currently selected request
func (v *WebViewController) GetSelectedRequest() *proxy.Request {
	return v.selectedRequest
}

// SetSelectedRequest sets the currently selected request
func (v *WebViewController) SetSelectedRequest(request *proxy.Request) {
	v.selectedRequest = request
}

// UpdateWebView updates the web view with latest proxy data
func (v *WebViewController) UpdateWebView() (unreadCount int, hasError bool) {
	if v.proxyServer == nil {
		return 0, false
	}

	// Check for new requests
	requests := v.proxyServer.GetRequests()
	newCount := len(requests)

	// Calculate unread count and check for errors
	unreadCount = 0
	hasError = false
	if newCount > v.lastWebCount {
		unreadCount = newCount - v.lastWebCount
		// Check if any of the new requests are errors
		for i := v.lastWebCount; i < newCount; i++ {
			if requests[i].IsError {
				hasError = true
				break
			}
		}
	}
	v.lastWebCount = newCount

	// Update the web requests list with latest proxy requests
	filtered := v.getFilteredRequests()
	v.updateWebRequestsList(filtered)

	// Auto-scroll to bottom if enabled
	if v.webAutoScroll && len(v.webRequestsList.Items()) > 0 {
		v.webRequestsList.Select(len(v.webRequestsList.Items()) - 1)
		v.updateSelectedRequestFromList()
	}

	return unreadCount, hasError
}

// Render renders the web view
func (v *WebViewController) Render() string {
	if v.width < 100 {
		// For narrow screens, use the simple view
		return v.renderNarrow()
	}

	// Check if proxy server is running - if not, show appropriate message
	if v.proxyServer == nil || !v.proxyServer.IsRunning() {
		return "\n🔴 Proxy server not running\n\nThe web proxy is currently disabled.\nTo enable it, check your configuration or start it manually."
	}

	// Calculate heights
	contentHeight := v.height - v.headerHeight - v.footerHeight

	// Split view: requests list on left, detail on right
	// Use a more conservative split for better readability
	listWidth := int(float64(v.width) * 0.4) // 40% for list
	detailWidth := v.width - listWidth - 3   // Rest for detail

	// Ensure minimum widths
	if listWidth < 40 {
		listWidth = 40
	}
	if detailWidth < 40 {
		detailWidth = 40
	}

	// Update list and detail viewport sizes
	v.webRequestsList.SetSize(listWidth-2, contentHeight-2) // Account for borders
	v.webDetailViewport.Width = detailWidth - 2
	v.webDetailViewport.Height = contentHeight - 2

	// Get filtered requests and update list
	requests := v.getFilteredRequests()
	v.updateWebRequestsList(requests)

	// Update selected request from list
	v.updateSelectedRequestFromList()

	// Render detail panel
	detailContent := v.renderRequestDetail()
	v.webDetailViewport.SetContent(detailContent)

	// Get list content
	listContent := v.renderRequestsList(requests, listWidth)

	// Create bordered views
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	// Apply borders with proper sizing
	listView := borderStyle.
		Width(listWidth - 2). // Account for border characters
		Height(contentHeight - 2).
		Render(listContent)

	detailView := borderStyle.
		Width(detailWidth - 2).
		Height(contentHeight - 2).
		Render(v.webDetailViewport.View())

	// Combine bordered views horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, listView, " ", detailView)
}

// renderRequestsList renders the requests list
func (v *WebViewController) renderRequestsList(requests []proxy.Request, width int) string {
	var content strings.Builder

	// Header with filter info and auto-scroll indicator
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	title := "Web Proxy Requests"
	if !v.webAutoScroll {
		scrollStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
			Bold(true)
		scrollIndicator := scrollStyle.Render("⏸ PAUSED")
		title += " " + scrollIndicator
	}
	content.WriteString(headerStyle.Render(title) + "\n")

	// Filter buttons
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeFilterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	filters := []string{"all", "pages", "api", "images", "other"}
	var filterParts []string
	for _, filter := range filters {
		if filter == v.webFilter {
			filterParts = append(filterParts, activeFilterStyle.Render("["+filter+"]"))
		} else {
			filterParts = append(filterParts, filterStyle.Render(filter))
		}
	}
	filterLine := "Filter: " + strings.Join(filterParts, " ") + " (f to cycle)"
	if !v.webAutoScroll {
		filterLine += " ⏸"
	}
	content.WriteString(filterLine + "\n\n")

	// Proxy status
	if v.proxyServer != nil && v.proxyServer.IsRunning() {
		modeStr := "Full Proxy"
		if v.proxyServer.GetMode() == proxy.ProxyModeReverse {
			modeStr = "Reverse Proxy"
		}
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("🟢 "+modeStr) + "\n\n")
	}

	if len(requests) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No matching requests"))
		return content.String()
	}

	// Requests table header
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	content.WriteString(headerStyle.Render("Time     St Method  URL") + "\n")
	// Use lipgloss border style instead of manual line drawing
	separatorStyle := lipgloss.NewStyle().
		Width(width - 4).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(lipgloss.Color("240"))
	content.WriteString(separatorStyle.Render("") + "\n")

	// Show recent requests (limit for performance)
	startIdx := 0
	if len(requests) > 100 {
		startIdx = len(requests) - 100
	}

	for i := startIdx; i < len(requests); i++ {
		req := requests[i]

		// Highlight selected request
		isSelected := v.selectedRequest != nil && req.ID == v.selectedRequest.ID

		// Color code status
		var statusColor string
		switch {
		case req.StatusCode >= 200 && req.StatusCode < 300:
			statusColor = "82" // Green
		case req.StatusCode >= 300 && req.StatusCode < 400:
			statusColor = "220" // Yellow
		case req.StatusCode >= 400 && req.StatusCode < 500:
			statusColor = "208" // Orange
		case req.StatusCode >= 500:
			statusColor = "196" // Red
		default:
			statusColor = "245" // Gray
		}

		// Color code method
		var methodColor string
		switch req.Method {
		case "GET":
			methodColor = "82"
		case "POST":
			methodColor = "220"
		case "PUT", "PATCH":
			methodColor = "208"
		case "DELETE":
			methodColor = "196"
		default:
			methodColor = "245"
		}

		// Truncate URL for display
		urlStr := req.URL
		maxURLLen := width - 25
		if len(urlStr) > maxURLLen {
			urlStr = urlStr[:maxURLLen-3] + "..."
		}

		// Format line
		line := fmt.Sprintf("%s %s %s %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(req.StartTime.Format("15:04:05")),
			lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Bold(true).Render(fmt.Sprintf("%3d", req.StatusCode)),
			lipgloss.NewStyle().Foreground(lipgloss.Color(methodColor)).Render(fmt.Sprintf("%-6s", req.Method)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(urlStr),
		)

		// Add indicators
		if req.Error != "" {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(" ❌")
		}
		if req.HasAuth {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(" 🔐")
		}
		if req.HasTelemetry {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(" 📊")
		}

		// Highlight if selected
		if isSelected {
			line = lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(line)
		}

		content.WriteString(line + "\n")
	}

	// Navigation help
	content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑/↓ navigate, Enter select, f filter"))
	content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Indicators: ❌ error, 🔐 auth, 📊 telemetry"))

	return content.String()
}

// renderRequestDetail renders the detail view for the selected request
func (v *WebViewController) renderRequestDetail() string {
	if v.selectedRequest == nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Select a request to view details")
	}

	req := *v.selectedRequest
	var content strings.Builder

	// Request header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	content.WriteString(headerStyle.Render("Request Details") + "\n\n")

	// Basic info
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	content.WriteString(labelStyle.Render("Method: ") + valueStyle.Render(req.Method) + "\n")
	content.WriteString(labelStyle.Render("URL: ") + valueStyle.Render(req.URL) + "\n")
	content.WriteString(labelStyle.Render("Status: ") + v.formatStatus(req.StatusCode) + "\n")
	content.WriteString(labelStyle.Render("Duration: ") + valueStyle.Render(fmt.Sprintf("%.0fms", req.Duration.Seconds()*1000)) + "\n")
	content.WriteString(labelStyle.Render("Time: ") + valueStyle.Render(req.StartTime.Format("15:04:05")) + "\n")
	content.WriteString(labelStyle.Render("Process: ") + valueStyle.Render(req.ProcessName) + "\n")

	if req.Size > 0 {
		content.WriteString(labelStyle.Render("Size: ") + valueStyle.Render(formatBytes(req.Size)) + "\n")
	}

	if req.Error != "" {
		content.WriteString(labelStyle.Render("Error: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(req.Error) + "\n")
	}

	// Authentication section
	if req.HasAuth {
		content.WriteString("\n" + headerStyle.Render("🔐 Authentication") + "\n\n")
		content.WriteString(labelStyle.Render("Type: ") + valueStyle.Render(req.AuthType) + "\n")

		if req.JWTError != "" {
			content.WriteString(labelStyle.Render("JWT Error: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(req.JWTError) + "\n")
		} else if req.JWTClaims != nil && len(req.JWTClaims) > 0 {
			content.WriteString(labelStyle.Render("JWT Claims:") + "\n")

			// Display common JWT claims
			claimOrder := []string{"sub", "iss", "aud", "exp", "iat", "nbf", "jti", "email", "name", "role", "scope"}
			displayedClaims := make(map[string]bool)

			for _, claim := range claimOrder {
				if value, exists := req.JWTClaims[claim]; exists {
					displayedClaims[claim] = true
					content.WriteString(fmt.Sprintf("  %s: %v\n", claim, value))
				}
			}

			// Display any remaining claims not in the ordered list
			for claim, value := range req.JWTClaims {
				if !displayedClaims[claim] {
					content.WriteString(fmt.Sprintf("  %s: %v\n", claim, value))
				}
			}
		}
	}

	// Telemetry section
	if req.HasTelemetry && req.Telemetry != nil {
		content.WriteString("\n" + headerStyle.Render("📊 Telemetry") + "\n\n")
		content.WriteString(v.renderTelemetryDetails(req.Telemetry))
	}

	return content.String()
}

// renderTelemetryDetails renders detailed telemetry information
func (v *WebViewController) renderTelemetryDetails(session *proxy.PageSession) string {
	if session == nil || len(session.Events) == 0 {
		return "No telemetry data available"
	}

	var content strings.Builder
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Summary stats
	eventCounts := make(map[proxy.TelemetryEventType]int)
	for _, event := range session.Events {
		eventCounts[event.Type]++
	}

	// Show event counts
	content.WriteString(labelStyle.Render("Events: "))
	var eventParts []string
	for eventType, count := range eventCounts {
		eventParts = append(eventParts, fmt.Sprintf("%s (%d)", v.formatTelemetryEvent(eventType), count))
	}
	content.WriteString(valueStyle.Render(strings.Join(eventParts, ", ")) + "\n\n")

	// Show detailed events (limit to recent ones)
	maxEvents := 10
	startIdx := 0
	if len(session.Events) > maxEvents {
		startIdx = len(session.Events) - maxEvents
	}

	for i := startIdx; i < len(session.Events); i++ {
		event := session.Events[i]
		timestamp := "15:04:05.000" // TODO: Fix timestamp formatting
		eventTitle := v.formatTelemetryEvent(event.Type)

		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(timestamp) + " ")
		content.WriteString(eventTitle + "\n")

		// Show event-specific data
		if len(event.Data) > 0 {
			for key, value := range event.Data {
				// Limit displayed data to avoid overwhelming output
				valueStr := fmt.Sprintf("%v", value)
				if len(valueStr) > 100 {
					valueStr = valueStr[:97] + "..."
				}
				content.WriteString(fmt.Sprintf("  %s: %s\n", key, valueStr))
			}
		}
		content.WriteString("\n")
	}

	return content.String()
}

// formatTelemetryEvent formats telemetry event types for display
func (v *WebViewController) formatTelemetryEvent(eventType proxy.TelemetryEventType) string {
	switch eventType {
	case proxy.TelemetryPageLoad:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("Page Load")
	case proxy.TelemetryDOMState:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("DOM State")
	case proxy.TelemetryJSError:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("JS Error")
	case proxy.TelemetryConsoleOutput:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("Console")
	case proxy.TelemetryUserInteraction:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("129")).Render("User Action")
	case proxy.TelemetryResourceTiming:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Render("Resource Timing")
	case proxy.TelemetryMemoryUsage:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("Memory")
	case proxy.TelemetryUnhandledReject:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Promise Reject")
	case proxy.TelemetryPerformance:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("Performance Metrics")
	default:
		return string(eventType)
	}
}

// formatStatus formats HTTP status codes with appropriate colors
func (v *WebViewController) formatStatus(status int) string {
	var style lipgloss.Style
	switch {
	case status >= 200 && status < 300:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	case status >= 300 && status < 400:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	case status >= 400 && status < 500:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	case status >= 500:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	default:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	}
	return style.Render(fmt.Sprintf("%d", status))
}

// renderNarrow renders a narrow view for small screens
func (v *WebViewController) renderNarrow() string {
	var content strings.Builder

	// Compact header: combine status + filter on one line, help + indicators on another
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeFilterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	// Line 1: Status + Filter
	var statusAndFilter strings.Builder
	if v.proxyServer != nil && v.proxyServer.IsRunning() {
		modeStr := "Full Proxy"
		if v.proxyServer.GetMode() == proxy.ProxyModeReverse {
			modeStr = "Reverse Proxy"
		}
		statusAndFilter.WriteString(statusStyle.Render(fmt.Sprintf("🟢 %s", modeStr)))
	} else {
		statusAndFilter.WriteString(statusStyle.Render("🔴 Proxy not running"))
		content.WriteString(statusAndFilter.String() + "\n")
		return content.String()
	}

	// Add filter to same line
	filters := []string{"all", "pages", "api", "images", "other"}
	var filterParts []string
	for _, filter := range filters {
		if filter == v.webFilter {
			filterParts = append(filterParts, activeFilterStyle.Render("["+filter+"]"))
		} else {
			filterParts = append(filterParts, filterStyle.Render(filter))
		}
	}
	filterText := " | Filter: " + strings.Join(filterParts, " ") + " (f)"
	if !v.webAutoScroll {
		filterText += " ⏸"
	}
	statusAndFilter.WriteString(filterText)
	content.WriteString(statusAndFilter.String() + "\n")

	// Line 2: Help + Indicators (compact)
	content.WriteString("↑/↓ navigate, Enter select | Indicators: ❌🔐📊\n")

	// Line 3: Separator
	// Use lipgloss border style instead of manual line drawing
	separatorStyle := lipgloss.NewStyle().
		Width(v.width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(lipgloss.Color("240"))
	content.WriteString(separatorStyle.Render("") + "\n")

	// Calculate list height correctly
	totalContentHeight := v.height - v.headerHeight - v.footerHeight
	filterHeaderLines := 3 // status+filter + help+indicators + separator (compact)
	listHeight := totalContentHeight - filterHeaderLines

	// Setup list size and update with filtered requests
	v.webRequestsList.SetSize(v.width, listHeight)
	requests := v.getFilteredRequests()
	v.updateWebRequestsList(requests)
	v.updateSelectedRequestFromList()

	// Add the list view - show helpful message if empty
	if len(v.webRequestsList.Items()) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true).
			Padding(1, 0)
		emptyMsg := emptyStyle.Render("No requests captured yet. Make some HTTP requests to see them here.")
		content.WriteString(emptyMsg)
	} else {
		content.WriteString(v.webRequestsList.View())
	}

	return content.String()
}

// getFilteredRequests returns requests filtered by the current filter
func (v *WebViewController) getFilteredRequests() []proxy.Request {
	if v.proxyServer == nil {
		return []proxy.Request{}
	}

	allRequests := v.proxyServer.GetRequests()

	if v.webFilter == "all" {
		return allRequests
	}

	// Apply filter logic
	var filtered []proxy.Request
	for _, req := range allRequests {
		switch v.webFilter {
		case "pages":
			if v.isPageRequest(req) {
				filtered = append(filtered, req)
			}
		case "api":
			if v.isAPIRequest(req) {
				filtered = append(filtered, req)
			}
		case "images":
			if v.isImageRequest(req) {
				filtered = append(filtered, req)
			}
		case "other":
			if !v.isPageRequest(req) && !v.isAPIRequest(req) && !v.isImageRequest(req) {
				filtered = append(filtered, req)
			}
		}
	}

	return filtered
}

// Helper methods for request classification
func (v *WebViewController) isPageRequest(req proxy.Request) bool {
	// XHR requests are never pages
	if req.IsXHR {
		return false
	}
	return strings.Contains(req.Path, ".html") || req.Path == "/" || (!strings.Contains(req.Path, ".") && !strings.Contains(req.Path, "/api/"))
}

func (v *WebViewController) isAPIRequest(req proxy.Request) bool {
	// Check content type for response (if available)
	contentType := ""
	if req.Telemetry != nil && len(req.Telemetry.Events) > 0 {
		// Look for response headers in telemetry
		for _, event := range req.Telemetry.Events {
			if event.Type == "response" {
				if headers, ok := event.Data["headers"].(map[string]interface{}); ok {
					if ct, ok := headers["content-type"].(string); ok {
						contentType = ct
					}
				}
			}
		}
	}

	// Exclude HTML responses from API category
	if strings.Contains(contentType, "text/html") {
		return false
	}

	return strings.Contains(req.Path, "/api/") || strings.Contains(req.Path, "/graphql") ||
		req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" || req.Method == "PATCH"
}

func (v *WebViewController) isImageRequest(req proxy.Request) bool {
	return strings.HasSuffix(req.Path, ".jpg") || strings.HasSuffix(req.Path, ".jpeg") ||
		strings.HasSuffix(req.Path, ".png") || strings.HasSuffix(req.Path, ".gif") ||
		strings.HasSuffix(req.Path, ".webp") || strings.HasSuffix(req.Path, ".svg") ||
		strings.HasSuffix(req.Path, ".ico")
}

// updateWebRequestsList updates the list with new request items
func (v *WebViewController) updateWebRequestsList(requests []proxy.Request) {
	// Convert requests to list items
	items := make([]list.Item, len(requests))
	for i, req := range requests {
		items[i] = proxyRequestItem{Request: req}
	}

	// Store current selection index before updating
	currentIndex := v.webRequestsList.Index()

	// Set the items in the list
	v.webRequestsList.SetItems(items)

	// Handle selection after items are updated
	if len(items) == 0 {
		// No items to select
		return
	}

	if v.webAutoScroll {
		// Auto-scroll: select last item
		v.webRequestsList.Select(len(items) - 1)
	} else {
		// Manual mode: try to maintain current selection or clamp to valid range
		if currentIndex >= len(items) {
			// If current index is out of bounds, select last item
			v.webRequestsList.Select(len(items) - 1)
		} else if currentIndex >= 0 {
			// Keep current selection if valid
			v.webRequestsList.Select(currentIndex)
		} else {
			// Default to first item
			v.webRequestsList.Select(0)
		}
	}
}

// updateSelectedRequestFromList updates the selected request from the list selection
func (v *WebViewController) updateSelectedRequestFromList() {
	if len(v.webRequestsList.Items()) == 0 {
		v.selectedRequest = nil
		return
	}

	selectedItem := v.webRequestsList.SelectedItem()
	if selectedItem == nil {
		v.selectedRequest = nil
		return
	}

	if proxyItem, ok := selectedItem.(proxyRequestItem); ok {
		v.selectedRequest = &proxyItem.Request
	}
}

// renderTelemetrySummary renders a one-line summary of telemetry data
func (v *WebViewController) renderTelemetrySummary(session *proxy.PageSession) string {
	if session == nil || len(session.Events) == 0 {
		return ""
	}

	// Extract key metrics from telemetry
	var loadTime, domReady float64
	var jsErrors, consoleLogs int
	var hasMemoryData, hasInteractions bool

	for _, event := range session.Events {
		switch event.Type {
		case proxy.TelemetryPageLoad:
			if timing, ok := event.Data["timing"].(map[string]interface{}); ok {
				if domComplete, ok := timing["domComplete"].(float64); ok {
					domReady = domComplete
				}
				if loadEventEnd, ok := timing["loadEventEnd"].(float64); ok {
					loadTime = loadEventEnd
				}
			}
		case proxy.TelemetryJSError, proxy.TelemetryUnhandledReject:
			jsErrors++
		case proxy.TelemetryConsoleOutput:
			consoleLogs++
		case proxy.TelemetryMemoryUsage:
			hasMemoryData = true
		case proxy.TelemetryUserInteraction:
			hasInteractions = true
		}
	}

	// Build summary line
	parts := []string{}
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))

	// Add timing info
	if domReady > 0 {
		parts = append(parts, fmt.Sprintf("DOM: %.0fms", domReady))
	}
	if loadTime > 0 {
		parts = append(parts, fmt.Sprintf("Load: %.0fms", loadTime))
	}

	// Add error count
	if jsErrors > 0 {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		parts = append(parts, errorStyle.Render(fmt.Sprintf("%d errors", jsErrors)))
	}

	// Add console log count
	if consoleLogs > 0 {
		parts = append(parts, fmt.Sprintf("%d logs", consoleLogs))
	}

	// Add feature indicators
	if hasMemoryData {
		parts = append(parts, "📊 memory")
	}
	if hasInteractions {
		parts = append(parts, "👆 interactions")
	}

	if len(parts) == 0 {
		return detailStyle.Render("No performance data")
	}

	return detailStyle.Render(strings.Join(parts, " | "))
}

// formatBytes is defined in model.go - no need to redefine
