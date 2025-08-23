package components

import (
	"fmt"
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SafeShell is a shell implementation that uses proper width calculations
// It demonstrates best practices for handling emoji and Unicode width
type SafeShell struct {
	width       int
	height      int
	currentView ViewType
	views       map[ViewType]*ViewInfo
	viewOrder   []ViewType
	quit        bool
	
	// Width calculator for responsive design
	widthCalc *TabWidthCalculator
	
	// Styles (reused from EnhancedShell)
	titleStyle        lipgloss.Style
	activeTabStyle    lipgloss.Style
	inactiveTabStyle  lipgloss.Style
	notifBadgeStyle   lipgloss.Style
	errorBadgeStyle   lipgloss.Style
	warnBadgeStyle    lipgloss.Style
	infoBadgeStyle    lipgloss.Style
	separatorStyle    lipgloss.Style
}

// NewSafeShell creates a shell with proper width handling
func NewSafeShell() *SafeShell {
	// ALWAYS initialize colors first
	MustInitColors()
	
	shell := &SafeShell{
		width:       80,
		height:      24,
		currentView: ViewProcesses,
		views:       make(map[ViewType]*ViewInfo),
		viewOrder:   []ViewType{},
	}
	
	// Initialize views
	shell.initializeViews()
	
	// Initialize styles
	shell.initializeStyles()
	
	// Create width calculator
	shell.widthCalc = NewTabWidthCalculator(shell.width, len(shell.viewOrder))
	
	// Add demo notifications
	shell.addDemoNotifications()
	
	return shell
}

// initializeViews sets up all views (same as EnhancedShell)
func (s *SafeShell) initializeViews() {
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
	
	s.viewOrder = []ViewType{
		ViewProcesses,
		ViewLogs,
		ViewErrors,
		ViewURLs,
		ViewWeb,
	}
}

// initializeStyles (same as before)
func (s *SafeShell) initializeStyles() {
	s.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)
	
	s.activeTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)
	
	s.inactiveTabStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)
	
	s.errorBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Background(lipgloss.Color("52"))
	
	s.warnBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		Background(lipgloss.Color("94"))
	
	s.infoBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("17"))
	
	s.notifBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("82")).
		Background(lipgloss.Color("22"))
	
	s.separatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
}

// addDemoNotifications adds test notifications
func (s *SafeShell) addDemoNotifications() {
	s.views[ViewErrors].HasNotif = true
	s.views[ViewErrors].NotifCount = 3
	s.views[ViewErrors].NotifSeverity = "error"
	s.views[ViewErrors].NotifIcon = "🔴"
	
	s.views[ViewLogs].HasNotif = true
	s.views[ViewLogs].NotifCount = 12
	s.views[ViewLogs].NotifSeverity = "info"
	s.views[ViewLogs].NotifIcon = "🔵"
}

// Init initializes the shell
func (s *SafeShell) Init() tea.Cmd {
	return func() tea.Msg {
		return tea.WindowSizeMsg{
			Width:  s.width,
			Height: s.height,
		}
	}
}

// Update handles messages
func (s *SafeShell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		// Update width calculator when terminal size changes
		s.widthCalc = NewTabWidthCalculator(s.width, len(s.viewOrder))
		return s, nil
		
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			s.quit = true
			return s, tea.Quit
			
		case "1", "2", "3", "4", "5":
			for _, view := range s.viewOrder {
				if s.views[view].KeyBinding == msg.String() {
					if s.currentView != view && s.views[view].HasNotif {
						s.views[view].HasNotif = false
						s.views[view].NotifCount = 0
					}
					s.currentView = view
					break
				}
			}
		}
	}
	
	return s, nil
}

// View renders the shell with PROPER width calculations
func (s *SafeShell) View() string {
	// Render header with proper width handling
	header := s.renderHeaderSafe()
	
	// Calculate actual header height using lipgloss
	headerHeight := lipgloss.Height(header)
	
	// Render content
	content := s.renderContent()
	
	// Calculate available height for content
	footerHeight := 1
	// Use CalculateAvailableWidth helper
	contentHeight := s.height - headerHeight - footerHeight - 2 // -2 for border
	contentWidth := CalculateAvailableWidth(s.width, true) // true = has border
	
	// Apply border with calculated dimensions
	contentStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
	
	contentBox := contentStyle.Render(content)
	
	// Render footer
	footer := s.renderFooterSafe()
	
	// Combine all parts
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		contentBox,
		footer,
	)
}

// renderHeaderSafe renders the header with proper width calculations
func (s *SafeShell) renderHeaderSafe() string {
	// Title with emoji
	titleText := "🎯 Brummer Process Manager"
	
	// IMPORTANT: Use SafeLipglossWidth for styled text
	title := s.titleStyle.Render(titleText)
	
	// Create title line with proper width
	titleLine := lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(title)
	
	// Render tabs with safe width calculation
	tabs := s.renderTabsSafe()
	
	// Center tabs properly using safe width
	tabLine := lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(tabs)
	
	return lipgloss.JoinVertical(lipgloss.Left, titleLine, tabLine)
}

// renderTabsSafe renders tabs with proper width validation
func (s *SafeShell) renderTabsSafe() string {
	// Calculate available width for tabs
	availableWidth := s.widthCalc.AvailablePerTab()
	
	var tabs []string
	var tabWidths []int
	
	// First pass: calculate actual width needed for each tab
	for _, viewType := range s.viewOrder {
		view := s.views[viewType]
		
		// Calculate the width this tab will need
		tabWidth := s.calculateTabWidth(view, s.widthCalc.Mode)
		tabWidths = append(tabWidths, tabWidth)
		
		// Render the tab
		tab := s.renderTabSafe(viewType, view, s.widthCalc.Mode, availableWidth)
		tabs = append(tabs, tab)
	}
	
	// Check if tabs will overflow
	if !s.widthCalc.CanFitFullTabs(tabWidths) {
		// Switch to more compact mode if needed
		if s.widthCalc.Mode == "full" {
			s.widthCalc.Mode = "normal"
		} else if s.widthCalc.Mode == "normal" {
			s.widthCalc.Mode = "compact"
		}
		
		// Re-render with compact mode
		tabs = nil
		for _, viewType := range s.viewOrder {
			view := s.views[viewType]
			tab := s.renderTabSafe(viewType, view, s.widthCalc.Mode, availableWidth)
			tabs = append(tabs, tab)
		}
	}
	
	// Join tabs with separator
	separator := s.separatorStyle.Render(" │ ")
	return strings.Join(tabs, separator)
}

// calculateTabWidth calculates the display width a tab will need
func (s *SafeShell) calculateTabWidth(view *ViewInfo, mode string) int {
	icon := ""
	name := ""
	
	switch mode {
	case "minimal":
		// Just key + optional dot
		if view.HasNotif {
			return 2 // "1•"
		}
		return 1 // "1"
		
	case "compact":
		icon = view.Icon
		// icon(2) + space(1) + key(1) = 4
		width := EmojiWidth(icon) + 1 + 1
		if view.HasNotif {
			width += 1 + EmojiWidth(view.NotifIcon) // space + emoji
		}
		return width
		
	case "normal":
		icon = view.Icon
		name = s.getAbbreviation(view.Title)
		
	case "full":
		icon = view.Icon
		name = view.Title
	}
	
	// Use the helper function
	return CalculateTabWidth(icon, view.KeyBinding, name, view.NotifIcon, view.NotifCount)
}

// renderTabSafe renders a single tab with width constraints
func (s *SafeShell) renderTabSafe(viewType ViewType, view *ViewInfo, mode string, maxWidth int) string {
	var label string
	
	// Build label based on mode
	switch mode {
	case "minimal":
		label = view.KeyBinding
		if view.HasNotif {
			label += "•"
		}
		
	case "compact":
		label = fmt.Sprintf("%s %s", view.Icon, view.KeyBinding)
		if view.HasNotif {
			label += fmt.Sprintf(" %s", view.NotifIcon)
		}
		
	case "normal":
		abbrev := s.getAbbreviation(view.Title)
		label = fmt.Sprintf("%s %s.%s", view.Icon, view.KeyBinding, abbrev)
		if view.HasNotif && view.NotifCount > 0 {
			label += fmt.Sprintf(" %s %d", view.NotifIcon, view.NotifCount)
		}
		
	default: // "full"
		label = fmt.Sprintf("%s %s.%s", view.Icon, view.KeyBinding, view.Title)
		if view.HasNotif && view.NotifCount > 0 {
			if view.NotifCount > 99 {
				label += fmt.Sprintf(" %s 99+", view.NotifIcon)
			} else {
				label += fmt.Sprintf(" %s %d", view.NotifIcon, view.NotifCount)
			}
		}
	}
	
	// SAFETY CHECK: Truncate if too wide
	if SafeStringWidth(label) > maxWidth && maxWidth > 0 {
		label = SafeTruncateWithEllipsis(label, maxWidth)
	}
	
	// Apply style
	if viewType == s.currentView {
		// Add active indicator
		indicator := "▶ "
		if mode == "minimal" {
			indicator = ">"
		}
		// Check combined width
		combined := indicator + label
		if SafeStringWidth(combined) > maxWidth && maxWidth > 0 {
			// Truncate label to make room for indicator
			labelMaxWidth := maxWidth - SafeStringWidth(indicator)
			label = SafeTruncateWithEllipsis(label, labelMaxWidth)
		}
		return s.activeTabStyle.Render(indicator + label)
	}
	
	return s.inactiveTabStyle.Render(label)
}

// renderFooterSafe renders footer with proper width
func (s *SafeShell) renderFooterSafe() string {
	help := "1-5: Switch tabs | Tab: Next | q: Quit"
	
	// Truncate help text if terminal is too narrow
	if SafeStringWidth(help) > s.width {
		help = SafeTruncateWithEllipsis(help, s.width-4) // Leave some padding
	}
	
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Width(s.width).
		Align(lipgloss.Center).
		Padding(0, 1)
	
	return footerStyle.Render(help)
}

// renderContent renders the main content area
func (s *SafeShell) renderContent() string {
	view := s.views[s.currentView]
	
	// Title with icon
	title := fmt.Sprintf("%s %s", view.Icon, view.Title)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
	
	// Content
	content := fmt.Sprintf("Content for %s view\n\nWindow: %dx%d\nMode: %s",
		view.Title, s.width, s.height, s.widthCalc.Mode)
	
	// Add width debug info
	debugInfo := fmt.Sprintf("\n\n--- Width Calculations ---\n")
	debugInfo += fmt.Sprintf("Terminal width: %d\n", s.width)
	debugInfo += fmt.Sprintf("Available for content: %d (after borders)\n", CalculateAvailableWidth(s.width, true))
	debugInfo += fmt.Sprintf("Tab render mode: %s\n", s.widthCalc.Mode)
	debugInfo += fmt.Sprintf("Available per tab: %d cells\n", s.widthCalc.AvailablePerTab())
	
	// Show emoji widths
	debugInfo += fmt.Sprintf("\nEmoji widths:\n")
	for _, v := range s.viewOrder {
		info := s.views[v]
		debugInfo += fmt.Sprintf("  %s = %d cells\n", info.Icon, EmojiWidth(info.Icon))
	}
	
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		content,
		debugInfo,
	)
}

// getAbbreviation returns abbreviated form of title
func (s *SafeShell) getAbbreviation(title string) string {
	abbreviations := map[string]string{
		"Processes": "Proc",
		"Logs":      "Log",
		"Errors":    "Err",
		"URLs":      "URL",
		"Web":       "Web",
	}
	
	if abbr, ok := abbreviations[title]; ok {
		return abbr
	}
	
	// Use SafeTruncate for other strings
	return SafeTruncate(title, 4)
}

// Quit returns whether the shell should quit
func (s *SafeShell) Quit() bool {
	return s.quit
}