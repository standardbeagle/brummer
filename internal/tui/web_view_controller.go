package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/proxy"
)

// ProxyServerInterface defines the interface for proxy server operations
type ProxyServerInterface interface {
	IsRunning() bool
	GetMode() proxy.ProxyMode
	GetRequests() []proxy.Request
	ClearRequests()
}

// WebViewController manages the web view state and rendering
type WebViewController struct {
	webRequestsList   list.Model
	webDetailViewport viewport.Model
	webFilter         string
	webAutoScroll     bool
	selectedRequest   *proxy.Request
	
	// Dependencies injected from parent Model
	proxyServer *proxy.Server
	width       int
	height      int
	headerHeight int
	footerHeight int
}

// NewWebViewController creates a new web view controller
func NewWebViewController(proxyServer *proxy.Server) *WebViewController {
	webRequestsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	webRequestsList.Title = "Web Proxy Requests"
	webRequestsList.SetShowStatusBar(false)

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

// UpdateWebRequestsList updates the web requests list with filtered data
func (v *WebViewController) UpdateWebRequestsList() {
	requests := v.getFilteredRequests()
	v.updateWebRequestsList(requests)
	v.updateSelectedRequestFromList()
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

	// Build filter header that will be shown above the bordered views
	var header strings.Builder
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeFilterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	// Filter tabs
	filters := []string{"all", "pages", "api", "images", "other"}
	var filterParts []string
	for _, filter := range filters {
		if filter == v.webFilter {
			filterParts = append(filterParts, activeFilterStyle.Render("["+filter+"]"))
		} else {
			filterParts = append(filterParts, filterStyle.Render(filter))
		}
	}

	// Filter line with pause indicator
	filterLine := "Filter: " + strings.Join(filterParts, " ") + " (f)"
	if !v.webAutoScroll {
		filterLine += " ⏸"
	}
	header.WriteString(filterLine + "\n")

	// Calculate heights accounting for the filter header
	filterHeaderHeight := 1 // 1 line for filter
	contentHeight := v.height - v.headerHeight - v.footerHeight - filterHeaderHeight

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

	// Get list content without filter header (we'll render it above)
	listContent := v.renderWebRequestsListSimple()

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

	// Combine header with bordered views
	borderedContent := lipgloss.JoinHorizontal(lipgloss.Top, listView, " ", detailView)
	return header.String() + borderedContent
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
	content.WriteString(strings.Repeat("─", v.width) + "\n")

	// Calculate list height correctly
	// Use shared header/footer heights for consistent layout
	// Our filter headers are WITHIN this content area, so subtract them
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

// renderWebRequestsListSimple renders just the list content without headers
func (v *WebViewController) renderWebRequestsListSimple() string {
	var content strings.Builder

	// Add the list view - check if empty and show helpful message
	itemCount := len(v.webRequestsList.Items())
	if itemCount == 0 {
		// Show helpful message when no requests are available
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true).
			MarginTop(2).
			MarginLeft(2)
		emptyMsg := emptyStyle.Render("No requests captured yet.\n\nMake some HTTP requests to see them here.")
		content.WriteString(emptyMsg)
	} else {
		// Just show the list without any headers
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
	return strings.Contains(req.ContentType, "text/html")
}

func (v *WebViewController) isAPIRequest(req proxy.Request) bool {
	return strings.Contains(req.ContentType, "application/json") || 
		   strings.Contains(req.Path, "/api/") ||
		   req.IsXHR
}

func (v *WebViewController) isImageRequest(req proxy.Request) bool {
	return strings.HasPrefix(req.ContentType, "image/")
}

// updateWebRequestsList updates the list with new request items
func (v *WebViewController) updateWebRequestsList(requests []proxy.Request) {
	// Convert requests to list items
	var items []list.Item
	for _, req := range requests {
		items = append(items, webRequestItem{request: req})
	}

	// Store current selection
	currentIndex := v.webRequestsList.Index()

	// Update list
	v.webRequestsList.SetItems(items)

	// Handle auto-scroll to bottom for new requests
	if v.webAutoScroll && len(items) > 0 {
		v.webRequestsList.Select(len(items) - 1)
	} else if currentIndex < len(items) {
		// Keep current selection if possible
		if currentIndex >= len(items) {
			v.webRequestsList.Select(len(items) - 1)
		} else {
			v.webRequestsList.Select(currentIndex)
		}
	} else {
		v.webRequestsList.Select(0)
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

	if reqItem, ok := selectedItem.(webRequestItem); ok {
		v.selectedRequest = &reqItem.request
	}
}

// renderRequestDetail renders the detail view for the selected request
func (v *WebViewController) renderRequestDetail() string {
	if v.selectedRequest == nil {
		return "Select a request to view details"
	}

	var content strings.Builder
	
	// Request header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	content.WriteString(headerStyle.Render(fmt.Sprintf("%s %s", v.selectedRequest.Method, v.selectedRequest.URL)) + "\n\n")
	
	// Status and timing info
	statusStyle := lipgloss.NewStyle().Bold(true)
	if v.selectedRequest.StatusCode >= 200 && v.selectedRequest.StatusCode < 300 {
		statusStyle = statusStyle.Foreground(lipgloss.Color("82"))
	} else if v.selectedRequest.StatusCode >= 400 {
		statusStyle = statusStyle.Foreground(lipgloss.Color("196"))
	}
	
	content.WriteString(fmt.Sprintf("Status: %s\n", statusStyle.Render(fmt.Sprintf("%d", v.selectedRequest.StatusCode))))
	content.WriteString(fmt.Sprintf("Duration: %v\n", v.selectedRequest.Duration))
	content.WriteString(fmt.Sprintf("Size: %d bytes\n", v.selectedRequest.Size))
	content.WriteString(fmt.Sprintf("Time: %s\n", v.selectedRequest.StartTime.Format("15:04:05")))
	
	if v.selectedRequest.ProcessName != "" {
		content.WriteString(fmt.Sprintf("Process: %s\n", v.selectedRequest.ProcessName))
	}
	
	if v.selectedRequest.Error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		content.WriteString(fmt.Sprintf("Error: %s\n", errorStyle.Render(v.selectedRequest.Error)))
	}
	
	return content.String()
}

// webRequestItem implements list.Item for web requests
type webRequestItem struct {
	request proxy.Request
}

func (i webRequestItem) FilterValue() string {
	return i.request.URL
}

func (i webRequestItem) Title() string {
	// Status indicator
	var statusIcon string
	switch {
	case i.request.StatusCode >= 200 && i.request.StatusCode < 300:
		statusIcon = "✓"
	case i.request.StatusCode >= 400:
		statusIcon = "❌"
	default:
		statusIcon = "⚠"
	}
	
	return fmt.Sprintf("%s %s %s", statusIcon, i.request.Method, i.request.Path)
}

func (i webRequestItem) Description() string {
	return fmt.Sprintf("%d • %v • %s", i.request.StatusCode, i.request.Duration, i.request.StartTime.Format("15:04:05"))
}