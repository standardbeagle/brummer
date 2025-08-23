package components

import (
	"fmt"
	"strings"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ViewType represents different views in the enhanced shell
type ViewType string

const (
	ViewProcesses ViewType = "processes"
	ViewLogs      ViewType = "logs"
	ViewErrors    ViewType = "errors"
	ViewURLs      ViewType = "urls"
	ViewWeb       ViewType = "web"
	ViewAICoders  ViewType = "ai"
	ViewSettings  ViewType = "settings"
)

// ViewInfo contains information about each view
type ViewInfo struct {
	Title       string
	Icon        string
	KeyBinding  string
	HasNotif    bool
	NotifCount  int
	NotifIcon   string
	NotifSeverity string
}

// NotificationUpdate message for updating tab notifications
type NotificationUpdate struct {
	View     ViewType
	Count    int
	Severity string // "error", "warning", "info", "success"
}

// EnhancedShell is an improved shell with proper tab structure and notifications
type EnhancedShell struct {
	width       int
	height      int
	currentView ViewType
	views       map[ViewType]*ViewInfo
	viewOrder   []ViewType
	quit        bool
	
	// Styles
	titleStyle        lipgloss.Style
	activeTabStyle    lipgloss.Style
	inactiveTabStyle  lipgloss.Style
	notifBadgeStyle   lipgloss.Style
	errorBadgeStyle   lipgloss.Style
	warnBadgeStyle    lipgloss.Style
	infoBadgeStyle    lipgloss.Style
	separatorStyle    lipgloss.Style
}

// NewEnhancedShell creates an enhanced shell with notifications
func NewEnhancedShell() *EnhancedShell {
	// Force color output
	lipgloss.SetColorProfile(termenv.ANSI256)
	
	shell := &EnhancedShell{
		width:       80,
		height:      24,
		currentView: ViewProcesses,
		views:       make(map[ViewType]*ViewInfo),
		viewOrder:   []ViewType{},
	}
	
	// Initialize views with icons
	shell.initializeViews()
	
	// Initialize styles
	shell.initializeStyles()
	
	// Add some demo notifications
	shell.addDemoNotifications()
	
	return shell
}

// initializeViews sets up all views with their icons and key bindings
func (s *EnhancedShell) initializeViews() {
	s.views[ViewProcesses] = &ViewInfo{
		Title:      "Processes",
		Icon:       "🏃",
		KeyBinding: "1",
	}
	
	s.views[ViewLogs] = &ViewInfo{
		Title:      "Logs",
		Icon:       "📄",
		KeyBinding: "2",
	}
	
	s.views[ViewErrors] = &ViewInfo{
		Title:      "Errors",
		Icon:       "❌",
		KeyBinding: "3",
	}
	
	s.views[ViewURLs] = &ViewInfo{
		Title:      "URLs",
		Icon:       "🔗",
		KeyBinding: "4",
	}
	
	s.views[ViewWeb] = &ViewInfo{
		Title:      "Web",
		Icon:       "🌐",
		KeyBinding: "5",
	}
	
	s.views[ViewAICoders] = &ViewInfo{
		Title:      "AI Coders",
		Icon:       "🤖",
		KeyBinding: "6",
	}
	
	s.views[ViewSettings] = &ViewInfo{
		Title:      "Settings",
		Icon:       "⚙️",
		KeyBinding: "7",
	}
	
	// Set view order
	s.viewOrder = []ViewType{
		ViewProcesses,
		ViewLogs,
		ViewErrors,
		ViewURLs,
		ViewWeb,
		ViewAICoders,
		ViewSettings,
	}
}

// initializeStyles sets up all lipgloss styles
func (s *EnhancedShell) initializeStyles() {
	// Title style
	s.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")). // Bright yellow
		Background(lipgloss.Color("235")).
		Padding(0, 1)
	
	// Active tab style
	s.activeTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)
	
	// Inactive tab style
	s.inactiveTabStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)
	
	// Notification badge styles
	s.errorBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")). // Red
		Background(lipgloss.Color("52"))    // Dark red background
	
	s.warnBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")). // Orange
		Background(lipgloss.Color("94"))    // Dark orange background
	
	s.infoBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).  // Blue
		Background(lipgloss.Color("17"))    // Dark blue background
	
	s.notifBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("82")).  // Green
		Background(lipgloss.Color("22"))    // Dark green background
	
	// Separator style
	s.separatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
}

// addDemoNotifications adds some example notifications for demonstration
func (s *EnhancedShell) addDemoNotifications() {
	// Add error notification to Errors tab
	s.views[ViewErrors].HasNotif = true
	s.views[ViewErrors].NotifCount = 3
	s.views[ViewErrors].NotifSeverity = "error"
	s.views[ViewErrors].NotifIcon = "🔴"
	
	// Add info notification to Logs tab
	s.views[ViewLogs].HasNotif = true
	s.views[ViewLogs].NotifCount = 12
	s.views[ViewLogs].NotifSeverity = "info"
	s.views[ViewLogs].NotifIcon = "🔵"
	
	// Add warning notification to URLs tab
	s.views[ViewURLs].HasNotif = true
	s.views[ViewURLs].NotifCount = 1
	s.views[ViewURLs].NotifSeverity = "warning"
	s.views[ViewURLs].NotifIcon = "🟡"
}

// Init initializes the shell
func (s *EnhancedShell) Init() tea.Cmd {
	return tea.Batch(
		// Send initial window size
		func() tea.Msg {
			return tea.WindowSizeMsg{
				Width:  s.width,
				Height: s.height,
			}
		},
		// Start a ticker for demo notification updates
		tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
			return NotificationUpdate{
				View:     ViewProcesses,
				Count:    1,
				Severity: "success",
			}
		}),
	)
}

// Update handles messages
func (s *EnhancedShell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil
		
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			s.quit = true
			return s, tea.Quit
			
		// Number keys for view switching
		case "1", "2", "3", "4", "5", "6", "7":
			for _, view := range s.viewOrder {
				if s.views[view].KeyBinding == msg.String() {
					// Clear notifications when switching to a view
					if s.currentView != view && s.views[view].HasNotif {
						s.views[view].HasNotif = false
						s.views[view].NotifCount = 0
					}
					s.currentView = view
					break
				}
			}
			
		// Tab navigation
		case "tab":
			currentIndex := 0
			for i, v := range s.viewOrder {
				if v == s.currentView {
					currentIndex = i
					break
				}
			}
			nextIndex := (currentIndex + 1) % len(s.viewOrder)
			nextView := s.viewOrder[nextIndex]
			
			// Clear notifications when switching
			if s.views[nextView].HasNotif {
				s.views[nextView].HasNotif = false
				s.views[nextView].NotifCount = 0
			}
			s.currentView = nextView
			
		case "shift+tab":
			currentIndex := 0
			for i, v := range s.viewOrder {
				if v == s.currentView {
					currentIndex = i
					break
				}
			}
			prevIndex := currentIndex - 1
			if prevIndex < 0 {
				prevIndex = len(s.viewOrder) - 1
			}
			prevView := s.viewOrder[prevIndex]
			
			// Clear notifications when switching
			if s.views[prevView].HasNotif {
				s.views[prevView].HasNotif = false
				s.views[prevView].NotifCount = 0
			}
			s.currentView = prevView
		}
		
	case NotificationUpdate:
		// Update notification for the specified view
		if view, ok := s.views[msg.View]; ok {
			// Don't show notification if we're currently viewing this tab
			if s.currentView != msg.View {
				view.HasNotif = true
				view.NotifCount += msg.Count
				
				// Set notification icon based on severity
				switch msg.Severity {
				case "error":
					view.NotifSeverity = "error"
					view.NotifIcon = "🔴"
				case "warning":
					view.NotifSeverity = "warning"
					view.NotifIcon = "🟡"
				case "success":
					view.NotifSeverity = "success"
					view.NotifIcon = "🟢"
				default:
					view.NotifSeverity = "info"
					view.NotifIcon = "🔵"
				}
			}
		}
		
		// Continue ticker for demo
		return s, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
			// Rotate through different views for demo
			views := []ViewType{ViewProcesses, ViewLogs, ViewErrors, ViewURLs}
			view := views[int(t.Unix())%len(views)]
			severities := []string{"error", "warning", "info", "success"}
			severity := severities[int(t.Unix()/3)%len(severities)]
			
			return NotificationUpdate{
				View:     view,
				Count:    1,
				Severity: severity,
			}
		})
	}
	
	return s, nil
}

// View renders the shell
func (s *EnhancedShell) View() string {
	// Render header with tabs
	header := s.renderHeader()
	
	// Render content
	content := s.renderContent()
	
	// Calculate content height
	headerLines := strings.Count(header, "\n") + 1
	footerLines := 1
	contentHeight := s.height - headerLines - footerLines - 2 // -2 for borders
	
	// Apply border to content
	contentStyle := lipgloss.NewStyle().
		Width(s.width - 2). // -2 for border
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
	
	contentBox := contentStyle.Render(content)
	
	// Render footer
	footer := s.renderFooter()
	
	// Combine all parts
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		contentBox,
		footer,
	)
}

// renderHeader renders the title and tabs with notifications
func (s *EnhancedShell) renderHeader() string {
	// Title
	title := s.titleStyle.Render("🎯 Brummer Process Manager")
	titleLine := lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title)
	
	// Render tabs based on width
	tabs := s.renderTabs()
	tabLine := lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(tabs)
	
	return lipgloss.JoinVertical(lipgloss.Left, titleLine, tabLine)
}

// renderTabs renders the tab bar with responsive sizing
func (s *EnhancedShell) renderTabs() string {
	mode := s.getTabRenderMode()
	var tabs []string
	
	for _, viewType := range s.viewOrder {
		view := s.views[viewType]
		tab := s.renderTab(viewType, view, mode)
		tabs = append(tabs, tab)
	}
	
	// Join tabs with separators
	tabsJoined := strings.Join(tabs, s.separatorStyle.Render(" │ "))
	
	return tabsJoined
}

// getTabRenderMode determines how to render tabs based on width
func (s *EnhancedShell) getTabRenderMode() string {
	if s.width < 50 {
		return "minimal" // Just numbers and notification dots
	} else if s.width < 80 {
		return "compact" // Icons and numbers
	} else if s.width < 120 {
		return "normal" // Icons, numbers, short names
	}
	return "full" // Everything including full names
}

// renderTab renders a single tab with notifications
func (s *EnhancedShell) renderTab(viewType ViewType, view *ViewInfo, mode string) string {
	var label string
	
	// Build label based on mode
	switch mode {
	case "minimal":
		// Just number with notification dot
		label = view.KeyBinding
		if view.HasNotif {
			label += "•"
		}
		
	case "compact":
		// Icon and number
		label = fmt.Sprintf("%s %s", view.Icon, view.KeyBinding)
		
	case "normal":
		// Icon, number, abbreviated name
		abbrev := s.getAbbreviation(view.Title)
		label = fmt.Sprintf("%s %s.%s", view.Icon, view.KeyBinding, abbrev)
		
	default: // "full"
		// Icon, number, full name
		label = fmt.Sprintf("%s %s.%s", view.Icon, view.KeyBinding, view.Title)
	}
	
	// Add notification badge if present
	if view.HasNotif && mode != "minimal" {
		badge := s.renderNotificationBadge(view)
		label = label + " " + badge
	}
	
	// Apply appropriate style
	if viewType == s.currentView {
		// Active tab with arrow indicator
		indicator := "▶ "
		if mode == "minimal" {
			indicator = ">"
		}
		return s.activeTabStyle.Render(indicator + label)
	}
	
	return s.inactiveTabStyle.Render(label)
}

// renderNotificationBadge renders the notification indicator
func (s *EnhancedShell) renderNotificationBadge(view *ViewInfo) string {
	// Choose style based on severity
	var badgeStyle lipgloss.Style
	switch view.NotifSeverity {
	case "error":
		badgeStyle = s.errorBadgeStyle
	case "warning":
		badgeStyle = s.warnBadgeStyle
	case "info":
		badgeStyle = s.infoBadgeStyle
	default:
		badgeStyle = s.notifBadgeStyle
	}
	
	// Format badge content
	var badge string
	if view.NotifCount > 99 {
		badge = fmt.Sprintf("%s 99+", view.NotifIcon)
	} else if view.NotifCount > 0 {
		badge = fmt.Sprintf("%s %d", view.NotifIcon, view.NotifCount)
	} else {
		badge = view.NotifIcon
	}
	
	return badgeStyle.Render(badge)
}

// getAbbreviation returns abbreviated form of title
func (s *EnhancedShell) getAbbreviation(title string) string {
	abbreviations := map[string]string{
		"Processes": "Proc",
		"Logs":      "Log",
		"Errors":    "Err",
		"URLs":      "URL",
		"Web":       "Web",
		"AI Coders": "AI",
		"Settings":  "Set",
	}
	
	if abbr, ok := abbreviations[title]; ok {
		return abbr
	}
	
	// Default: first 3 characters
	if len(title) > 3 {
		return title[:3]
	}
	return title
}

// renderContent renders the main content area
func (s *EnhancedShell) renderContent() string {
	view := s.views[s.currentView]
	
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
	
	title := fmt.Sprintf("%s %s", view.Icon, view.Title)
	
	// Content based on current view
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))
	
	var content string
	switch s.currentView {
	case ViewProcesses:
		content = "Process list would appear here\n\n• npm run dev (running)\n• npm test (stopped)\n• build process (running)"
	case ViewLogs:
		content = "Log output would stream here\n\n[2024-01-15 10:30:45] Starting development server...\n[2024-01-15 10:30:46] Server listening on port 3000"
	case ViewErrors:
		content = "Error messages would appear here\n\n❌ TypeError: Cannot read property 'foo' of undefined\n   at app.js:42:15"
	case ViewURLs:
		content = "Discovered URLs would be listed here\n\n🔗 http://localhost:3000\n🔗 http://localhost:3000/api\n🔗 http://localhost:8080/health"
	case ViewWeb:
		content = "Web/proxy requests would appear here\n\nGET /api/users 200 OK (42ms)\nPOST /api/login 401 Unauthorized (15ms)"
	case ViewAICoders:
		content = "AI Coder sessions would be managed here\n\n🤖 Session 1: Refactoring user service\n🤖 Session 2: Adding authentication"
	case ViewSettings:
		content = "Settings and configuration\n\n⚙️ Theme: Dark\n⚙️ Auto-start: Enabled\n⚙️ Port: 7777"
	default:
		content = "Unknown view"
	}
	
	// Add debug info
	debugInfo := fmt.Sprintf("\n\n---\nWindow: %dx%d | Mode: %s | Notifications: %d",
		s.width, s.height, s.getTabRenderMode(), s.getTotalNotifications())
	
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		contentStyle.Render(content),
		debugInfo,
	)
}

// renderFooter renders the footer with help text
func (s *EnhancedShell) renderFooter() string {
	help := "1-7: Switch views | ←→: Navigate | Tab: Focus | Ctrl+C: Quit"
	
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Width(s.width).
		Align(lipgloss.Center).
		Padding(0, 1)
	
	return footerStyle.Render(help)
}

// getTotalNotifications counts all notifications
func (s *EnhancedShell) getTotalNotifications() int {
	total := 0
	for _, view := range s.views {
		if view.HasNotif {
			total += view.NotifCount
		}
	}
	return total
}

// Quit returns whether the shell should quit
func (s *EnhancedShell) Quit() bool {
	return s.quit
}

// UpdateNotification updates notification for a specific view (external API)
func (s *EnhancedShell) UpdateNotification(view ViewType, count int, severity string) {
	if v, ok := s.views[view]; ok {
		// Don't show notification if we're currently viewing this tab
		if s.currentView != view {
			v.HasNotif = true
			v.NotifCount += count
			
			// Set notification icon based on severity
			switch severity {
			case "error":
				v.NotifSeverity = "error"
				v.NotifIcon = "🔴"
			case "warning":
				v.NotifSeverity = "warning"
				v.NotifIcon = "🟡"
			case "success":
				v.NotifSeverity = "success"
				v.NotifIcon = "🟢"
			default:
				v.NotifSeverity = "info"
				v.NotifIcon = "🔵"
			}
		}
	}
}

// ClearNotifications clears notifications for a specific view
func (s *EnhancedShell) ClearNotifications(view ViewType) {
	if v, ok := s.views[view]; ok {
		v.HasNotif = false
		v.NotifCount = 0
		v.NotifIcon = ""
		v.NotifSeverity = ""
	}
}