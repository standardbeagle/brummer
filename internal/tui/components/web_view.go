package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
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

// NewWebViewComponentRefactored creates a new web view component using Bubble Tea list
func NewWebViewComponentRefactored() *WebViewComponentRefactored {
	// Create list delegate with custom styling
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("33")).Bold(true).Underline(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("33")).Bold(true)

	// Create the list
	requestsList := list.New([]list.Item{}, delegate, 0, 0)
	requestsList.Title = "HTTP Requests"
	requestsList.SetShowStatusBar(false)
	requestsList.SetFilteringEnabled(false)
	requestsList.SetShowPagination(false)
	requestsList.SetShowHelp(false)

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
		if loadTime, ok := selectedRequest.PerformanceMetrics["loadCompleteTime"].(float64); ok {
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
	// Use ► indicator for selected items
	indicator := "  "
	if i.IsSelected {
		indicator = "► "
	}

	// Format duration - always show duration even for very fast requests
	duration := ""
	if i.Duration > 0 {
		if i.Duration < time.Millisecond {
			duration = fmt.Sprintf(" %.0fμs", float64(i.Duration.Nanoseconds())/1000)
		} else if i.Duration < time.Second {
			duration = fmt.Sprintf(" %.0fms", float64(i.Duration.Nanoseconds())/1000000)
		} else {
			duration = fmt.Sprintf(" %.1fs", i.Duration.Seconds())
		}
	} else {
		// For 0 duration requests, show as "<1ms" to indicate very fast local requests
		duration = " <1ms"
	}

	return fmt.Sprintf("%s%-8s %-12s", indicator, i.Method, i.ProcessName) + duration
}

func (i WebRequestItemWrapper) Description() string {
	// Truncate URL if too long
	maxURLLength := 50
	url := i.URL
	if len(url) > maxURLLength {
		url = url[:maxURLLength-3] + "..."
	}

	// Handle error status
	statusText := fmt.Sprintf("%d", i.StatusCode)
	if i.Error != "" {
		statusText = "Error"
	}

	return fmt.Sprintf("%-50s %-8s", url, statusText)
}