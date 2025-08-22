package components

import (
	"fmt"
	"io"
	"strings"
	"time"
	
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WebViewComponent provides web/proxy display functionality
type WebViewComponent struct {
	BaseComponent
	
	// View modes
	mode           WebViewMode
	currentURL     string
	isLoading      bool
	
	// Proxy request tracking
	requests       []ProxyRequestInfo
	selectedReqID  string
	
	// Request/Response display
	requestView    viewport.Model
	responseView   viewport.Model
	requestList    list.Model
	
	// Browser simulation
	browserContent string
	browserHistory []string
	historyIndex   int
	
	// Display options
	showHeaders    bool
	showBody       bool
	splitView      bool
	
	// Styles
	titleStyle     lipgloss.Style
	urlStyle       lipgloss.Style
	methodStyle    lipgloss.Style
	statusStyle    lipgloss.Style
	headerStyle    lipgloss.Style
	bodyStyle      lipgloss.Style
	loadingStyle   lipgloss.Style
	errorStyle     lipgloss.Style
}

// WebViewMode is defined in custom_messages.go

// NewWebViewComponent creates a new web view component
func NewWebViewComponent() *WebViewComponent {
	w := &WebViewComponent{
		mode:           WebViewModeBrowser,
		requests:       []ProxyRequestInfo{},
		browserHistory: []string{},
		historyIndex:   -1,
		showHeaders:    true,
		showBody:       true,
		splitView:      true,
	}
	
	// Initialize styles
	w.initStyles()
	
	// Initialize viewports
	w.requestView = viewport.New(40, 20)
	w.responseView = viewport.New(40, 20)
	
	// Initialize request list
	w.initializeRequestList()
	
	// Set visible by default
	w.visible = true
	
	return w
}

// initStyles initializes the component styles
func (w *WebViewComponent) initStyles() {
	w.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))
	
	w.urlStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Underline(true)
	
	w.methodStyle = lipgloss.NewStyle().
		Bold(true)
	
	w.statusStyle = lipgloss.NewStyle().
		Bold(true)
	
	w.headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	w.bodyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))
	
	w.loadingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Italic(true)
	
	w.errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
}

// initializeRequestList sets up the request list
func (w *WebViewComponent) initializeRequestList() {
	items := w.requestsToListItems()
	
	w.requestList = list.New(items, requestItemDelegate{}, 40, 20)
	w.requestList.Title = "HTTP Requests"
	w.requestList.SetShowStatusBar(true)
	w.requestList.SetFilteringEnabled(true)
	
	// Customize key bindings
	w.requestList.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	)
	w.requestList.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	)
}

// Update handles messages
func (w *WebViewComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle component-specific messages
	switch msg := msg.(type) {
	case WebNavigateMsg:
		// Navigate to URL
		w.currentURL = msg.URL
		w.isLoading = true
		w.browserHistory = append(w.browserHistory, msg.URL)
		w.historyIndex = len(w.browserHistory) - 1
		
		// Simulate loading
		return w, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return WebLoadCompleteMsg{
				URL:     msg.URL,
				Content: fmt.Sprintf("<html><body>Simulated content for %s</body></html>", msg.URL),
			}
		})
		
	case WebLoadCompleteMsg:
		// Page loaded
		w.isLoading = false
		w.browserContent = msg.Content
		return w, SendSystemNotification(NotificationInfo,
			"Page loaded", msg.URL)
		
	case WebBackMsg:
		// Navigate back
		if w.historyIndex > 0 {
			w.historyIndex--
			w.currentURL = w.browserHistory[w.historyIndex]
			return w, SendWebNavigate(w.currentURL)
		}
		return w, nil
		
	case WebForwardMsg:
		// Navigate forward
		if w.historyIndex < len(w.browserHistory)-1 {
			w.historyIndex++
			w.currentURL = w.browserHistory[w.historyIndex]
			return w, SendWebNavigate(w.currentURL)
		}
		return w, nil
		
	case WebRefreshMsg:
		// Refresh current page
		if w.currentURL != "" {
			return w, SendWebNavigate(w.currentURL)
		}
		return w, nil
		
	case ProxyRequestMsg:
		// New proxy request
		reqInfo := ProxyRequestInfo{
			ID:        msg.ID,
			Method:    msg.Method,
			URL:       msg.URL,
			Timestamp: msg.Timestamp,
			ProcessID: msg.ProcessID,
		}
		w.requests = append(w.requests, reqInfo)
		w.updateRequestList()
		
		// Select this request if in proxy mode
		if w.mode == WebViewModeProxy {
			w.selectedReqID = msg.ID
			w.updateRequestDetails(msg)
		}
		return w, nil
		
	case ProxyResponseMsg:
		// Update request with response
		for i, req := range w.requests {
			if req.ID == msg.RequestID {
				w.requests[i].StatusCode = msg.StatusCode
				w.requests[i].Duration = msg.Duration
				if msg.Error != nil {
					w.requests[i].Error = msg.Error.Error()
				}
				break
			}
		}
		w.updateRequestList()
		
		// Update response view if this is selected
		if w.selectedReqID == msg.RequestID {
			w.updateResponseDetails(msg)
		}
		return w, nil
		
	case ProxyRequestsUpdateMsg:
		// Update entire request list
		w.requests = msg.Requests
		w.updateRequestList()
		return w, nil
		
	case WebViewModeMsg:
		// Change view mode
		w.mode = msg.Mode
		return w, nil
		
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
		
		// Update viewport sizes
		if w.splitView {
			w.requestView.Width = msg.Width/2 - 2
			w.requestView.Height = msg.Height - 10
			w.responseView.Width = msg.Width/2 - 2
			w.responseView.Height = msg.Height - 10
		} else {
			w.requestView.Width = msg.Width - 4
			w.requestView.Height = msg.Height - 10
			w.responseView.Width = msg.Width - 4
			w.responseView.Height = msg.Height - 10
		}
		
		w.requestList.SetSize(msg.Width/3, msg.Height-4)
		
	case tea.KeyMsg:
		// Handle key presses based on mode
		switch w.mode {
		case WebViewModeBrowser:
			switch msg.String() {
			case "b", "alt+left":
				// Back
				return w, SendWebBack()
				
			case "f", "alt+right":
				// Forward
				return w, SendWebForward()
				
			case "r", "ctrl+r":
				// Refresh
				return w, SendWebRefresh()
				
			case "g":
				// Go to URL (would open input)
				return w, SendSystemNotification(NotificationInfo,
					"Go to URL", "Enter URL in command palette")
				
			case "i":
				// Toggle inspector mode
				w.mode = WebViewModeInspector
				return w, nil
			}
			
		case WebViewModeProxy:
			switch msg.String() {
			case "enter", " ":
				// Select request for details
				if item := w.requestList.SelectedItem(); item != nil {
					if ri, ok := item.(requestItem); ok {
						w.selectedReqID = ri.request.ID
						// Load request details
						return w, nil
					}
				}
				
			case "h":
				// Toggle headers
				w.showHeaders = !w.showHeaders
				return w, nil
				
			case "b":
				// Toggle body
				w.showBody = !w.showBody
				return w, nil
				
			case "s":
				// Toggle split view
				w.splitView = !w.splitView
				return w, nil
				
			case "c":
				// Clear requests
				w.requests = []ProxyRequestInfo{}
				w.updateRequestList()
				return w, nil
				
			case "m":
				// Switch to browser mode
				w.mode = WebViewModeBrowser
				return w, nil
			}
			
		case WebViewModeInspector:
			switch msg.String() {
			case "m":
				// Switch to browser mode
				w.mode = WebViewModeBrowser
				return w, nil
				
			case "p":
				// Switch to proxy mode
				w.mode = WebViewModeProxy
				return w, nil
			}
		}
		
		// Tab switches between modes
		if msg.String() == "tab" {
			w.mode = WebViewMode((int(w.mode) + 1) % 3)
			return w, nil
		}
	}
	
	// Update sub-components based on mode
	if w.visible {
		switch w.mode {
		case WebViewModeProxy:
			newList, cmd := w.requestList.Update(msg)
			w.requestList = newList
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			
		case WebViewModeBrowser, WebViewModeInspector:
			// Update viewports
			newReqView, cmd := w.requestView.Update(msg)
			w.requestView = newReqView
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			
			newRespView, cmd := w.responseView.Update(msg)
			w.responseView = newRespView
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	
	return w, tea.Batch(cmds...)
}

// View renders the component
func (w *WebViewComponent) View() string {
	if !w.visible {
		return ""
	}
	
	var b strings.Builder
	
	switch w.mode {
	case WebViewModeBrowser:
		w.renderBrowserView(&b)
	case WebViewModeProxy:
		w.renderProxyView(&b)
	case WebViewModeInspector:
		w.renderInspectorView(&b)
	}
	
	return b.String()
}

// renderBrowserView renders the browser simulation
func (w *WebViewComponent) renderBrowserView(b *strings.Builder) {
	// Title bar
	title := "🌐 Web Browser"
	if w.isLoading {
		title += " " + w.loadingStyle.Render("(Loading...)")
	}
	b.WriteString(w.titleStyle.Render(title) + "\n")
	
	// URL bar
	urlBar := fmt.Sprintf("📍 %s", w.currentURL)
	if w.currentURL == "" {
		urlBar = "📍 (no page loaded)"
	}
	b.WriteString(w.urlStyle.Render(urlBar) + "\n\n")
	
	// Content area
	if w.browserContent != "" {
		b.WriteString(w.bodyStyle.Render(w.browserContent))
	} else {
		b.WriteString(w.headerStyle.Render("No content to display"))
	}
	
	// Navigation hints
	b.WriteString("\n\n")
	hints := []string{
		"b: back",
		"f: forward",
		"r: refresh",
		"g: go to URL",
		"i: inspector",
		"tab: switch mode",
	}
	b.WriteString(w.headerStyle.Render(strings.Join(hints, " | ")))
}

// renderProxyView renders the proxy request inspector
func (w *WebViewComponent) renderProxyView(b *strings.Builder) {
	// Title
	b.WriteString(w.titleStyle.Render(fmt.Sprintf("🔍 Proxy Inspector (%d requests)", len(w.requests))) + "\n\n")
	
	if w.splitView && w.selectedReqID != "" {
		// Split view: list on left, details on right
		leftWidth := w.width / 3
		rightWidth := w.width - leftWidth - 2
		
		// Render list and details side by side
		listLines := strings.Split(w.requestList.View(), "\n")
		detailLines := strings.Split(w.renderRequestDetails(), "\n")
		
		maxLines := max(len(listLines), len(detailLines))
		
		for i := 0; i < maxLines; i++ {
			// Left side (list)
			if i < len(listLines) {
				line := listLines[i]
				if len(line) > leftWidth {
					line = line[:leftWidth]
				}
				b.WriteString(fmt.Sprintf("%-*s", leftWidth, line))
			} else {
				b.WriteString(strings.Repeat(" ", leftWidth))
			}
			
			b.WriteString(" │ ")
			
			// Right side (details)
			if i < len(detailLines) {
				line := detailLines[i]
				if len(line) > rightWidth {
					line = line[:rightWidth]
				}
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	} else {
		// Full view: just the list or details
		if w.selectedReqID != "" {
			b.WriteString(w.renderRequestDetails())
		} else {
			b.WriteString(w.requestList.View())
		}
	}
	
	// Footer hints
	b.WriteString("\n")
	hints := []string{
		"enter: view details",
		"h: toggle headers",
		"b: toggle body",
		"s: split view",
		"c: clear",
		"m: browser mode",
	}
	b.WriteString(w.headerStyle.Render(strings.Join(hints, " | ")))
}

// renderInspectorView renders the inspector mode
func (w *WebViewComponent) renderInspectorView(b *strings.Builder) {
	// Title
	b.WriteString(w.titleStyle.Render("🔬 Web Inspector") + "\n\n")
	
	// Show both browser state and recent requests
	b.WriteString(w.headerStyle.Render("Current Page:") + "\n")
	b.WriteString(w.urlStyle.Render(w.currentURL) + "\n\n")
	
	b.WriteString(w.headerStyle.Render("Recent Requests:") + "\n")
	count := min(5, len(w.requests))
	for i := len(w.requests) - count; i < len(w.requests); i++ {
		req := w.requests[i]
		status := w.getStatusStyle(req.StatusCode).Render(fmt.Sprintf("%d", req.StatusCode))
		method := w.getMethodStyle(req.Method).Render(req.Method)
		line := fmt.Sprintf("%s %s %s", method, status, req.URL)
		b.WriteString("  " + line + "\n")
	}
	
	// Hints
	b.WriteString("\n")
	hints := []string{
		"m: browser mode",
		"p: proxy mode",
		"tab: switch mode",
	}
	b.WriteString(w.headerStyle.Render(strings.Join(hints, " | ")))
}

// renderRequestDetails renders detailed request/response information
func (w *WebViewComponent) renderRequestDetails() string {
	// Find the selected request
	var selected *ProxyRequestInfo
	for _, req := range w.requests {
		if req.ID == w.selectedReqID {
			selected = &req
			break
		}
	}
	
	if selected == nil {
		return w.headerStyle.Render("No request selected")
	}
	
	var b strings.Builder
	
	// Request details
	b.WriteString(w.titleStyle.Render("Request Details") + "\n\n")
	
	method := w.getMethodStyle(selected.Method).Render(selected.Method)
	b.WriteString(fmt.Sprintf("%s %s\n", method, selected.URL))
	b.WriteString(w.headerStyle.Render(fmt.Sprintf("Time: %s\n", selected.Timestamp.Format("15:04:05"))))
	
	if selected.ProcessID != "" {
		b.WriteString(w.headerStyle.Render(fmt.Sprintf("Process: %s\n", selected.ProcessID)))
	}
	
	// Response details
	if selected.StatusCode > 0 {
		b.WriteString("\n" + w.titleStyle.Render("Response") + "\n")
		status := w.getStatusStyle(selected.StatusCode).Render(fmt.Sprintf("%d", selected.StatusCode))
		b.WriteString(fmt.Sprintf("Status: %s\n", status))
		b.WriteString(w.headerStyle.Render(fmt.Sprintf("Duration: %v\n", selected.Duration)))
		
		if selected.Size > 0 {
			b.WriteString(w.headerStyle.Render(fmt.Sprintf("Size: %d bytes\n", selected.Size)))
		}
	}
	
	if selected.Error != "" {
		b.WriteString("\n" + w.errorStyle.Render(fmt.Sprintf("Error: %s", selected.Error)))
	}
	
	return b.String()
}

// Helper methods

func (w *WebViewComponent) updateRequestList() {
	items := w.requestsToListItems()
	w.requestList.SetItems(items)
	
	// Update title with count
	w.requestList.Title = fmt.Sprintf("HTTP Requests (%d)", len(w.requests))
}

func (w *WebViewComponent) requestsToListItems() []list.Item {
	items := make([]list.Item, len(w.requests))
	for i, req := range w.requests {
		items[i] = requestItem{request: req}
	}
	return items
}

func (w *WebViewComponent) updateRequestDetails(msg ProxyRequestMsg) {
	// Format request details for viewport
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Method: %s\n", msg.Method))
	content.WriteString(fmt.Sprintf("URL: %s\n", msg.URL))
	content.WriteString(fmt.Sprintf("Time: %s\n", msg.Timestamp.Format("15:04:05.000")))
	
	if w.showHeaders && len(msg.Headers) > 0 {
		content.WriteString("\nHeaders:\n")
		for key, values := range msg.Headers {
			for _, value := range values {
				content.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
			}
		}
	}
	
	if w.showBody && len(msg.Body) > 0 {
		content.WriteString("\nBody:\n")
		content.WriteString(string(msg.Body))
	}
	
	w.requestView.SetContent(content.String())
}

func (w *WebViewComponent) updateResponseDetails(msg ProxyResponseMsg) {
	// Format response details for viewport
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Status: %d\n", msg.StatusCode))
	content.WriteString(fmt.Sprintf("Duration: %v\n", msg.Duration))
	
	if msg.Error != nil {
		content.WriteString(fmt.Sprintf("Error: %v\n", msg.Error))
	}
	
	if w.showHeaders && len(msg.Headers) > 0 {
		content.WriteString("\nHeaders:\n")
		for key, values := range msg.Headers {
			for _, value := range values {
				content.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
			}
		}
	}
	
	if w.showBody && len(msg.Body) > 0 {
		content.WriteString("\nBody:\n")
		content.WriteString(string(msg.Body))
	}
	
	w.responseView.SetContent(content.String())
}

func (w *WebViewComponent) getMethodStyle(method string) lipgloss.Style {
	color := "250"
	switch method {
	case "GET":
		color = "86"
	case "POST":
		color = "39"
	case "PUT":
		color = "226"
	case "DELETE":
		color = "196"
	case "PATCH":
		color = "205"
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
}

func (w *WebViewComponent) getStatusStyle(status int) lipgloss.Style {
	color := "250"
	if status >= 200 && status < 300 {
		color = "86" // Green for success
	} else if status >= 300 && status < 400 {
		color = "226" // Yellow for redirect
	} else if status >= 400 && status < 500 {
		color = "208" // Orange for client error
	} else if status >= 500 {
		color = "196" // Red for server error
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
}

// Focus sets focus to this component
func (w *WebViewComponent) Focus() tea.Cmd {
	w.focused = true
	return nil
}

// Blur removes focus from this component
func (w *WebViewComponent) Blur() tea.Cmd {
	w.focused = false
	return nil
}

// requestItem implements list.Item for proxy requests
type requestItem struct {
	request ProxyRequestInfo
}

func (i requestItem) Title() string {
	method := fmt.Sprintf("%-6s", i.request.Method)
	status := ""
	if i.request.StatusCode > 0 {
		status = fmt.Sprintf("[%d]", i.request.StatusCode)
	} else {
		status = "[...]"
	}
	
	// Truncate URL if too long
	url := i.request.URL
	if len(url) > 40 {
		url = url[:37] + "..."
	}
	
	return fmt.Sprintf("%s %s %s", method, status, url)
}

func (i requestItem) Description() string {
	parts := []string{}
	
	// Add timestamp
	parts = append(parts, i.request.Timestamp.Format("15:04:05"))
	
	// Add duration if available
	if i.request.Duration > 0 {
		parts = append(parts, fmt.Sprintf("%v", i.request.Duration))
	}
	
	// Add process if available
	if i.request.ProcessID != "" {
		parts = append(parts, i.request.ProcessID)
	}
	
	// Add error indicator
	if i.request.Error != "" {
		parts = append(parts, "ERROR")
	}
	
	return strings.Join(parts, " | ")
}

func (i requestItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s", i.request.Method, i.request.URL, i.request.ProcessID)
}

// requestItemDelegate implements list.ItemDelegate
type requestItemDelegate struct{}

func (d requestItemDelegate) Height() int { return 2 }

func (d requestItemDelegate) Spacing() int { return 1 }

func (d requestItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d requestItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Styles
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("170")).
		Bold(true)
	normalStyle := lipgloss.NewStyle().
		PaddingLeft(2)
	
	item, ok := listItem.(requestItem)
	if !ok {
		return
	}
	
	// Format the item
	str := fmt.Sprintf("%s\n  %s", item.Title(), item.Description())
	
	if index == m.Index() {
		fmt.Fprintf(w, "%s", selectedStyle.Render("> "+str))
	} else {
		fmt.Fprintf(w, "%s", normalStyle.Render(str))
	}
}

// Helper functions for web messages

func SendWebNavigate(url string) tea.Cmd {
	return func() tea.Msg {
		return WebNavigateMsg{URL: url}
	}
}

func SendWebBack() tea.Cmd {
	return func() tea.Msg {
		return WebBackMsg{}
	}
}

func SendWebForward() tea.Cmd {
	return func() tea.Msg {
		return WebForwardMsg{}
	}
}

func SendWebRefresh() tea.Cmd {
	return func() tea.Msg {
		return WebRefreshMsg{}
	}
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}