package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
)

// TabsComponent manages the tab bar rendering using Lipgloss
type TabsComponent struct {
	// Dependencies
	processMgr              *process.Manager
	notificationsController NotificationsControllerInterface
	systemController        SystemControllerInterface

	// View configuration
	views     []View
	debugMode bool

	// Current state
	activeView View
	width      int

	// Styles
	titleStyle     lipgloss.Style
	activeStyle    lipgloss.Style
	inactiveStyle  lipgloss.Style
	separatorStyle lipgloss.Style
	tabBarStyle    lipgloss.Style
}

// NewTabsComponent creates a new tabs component
func NewTabsComponent(processMgr *process.Manager, notificationsController NotificationsControllerInterface, systemController SystemControllerInterface, debugMode bool) *TabsComponent {
	tc := &TabsComponent{
		processMgr:              processMgr,
		notificationsController: notificationsController,
		systemController:        systemController,
		debugMode:               debugMode,
	}

	// Initialize views
	tc.views = []View{ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewWeb, ViewAICoders, ViewSettings}
	if debugMode {
		tc.views = append(tc.views, ViewMCPConnections)
	}

	// Initialize styles
	tc.initStyles()

	return tc
}

// initStyles initializes all the Lipgloss styles
func (tc *TabsComponent) initStyles() {
	// Title style
	tc.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226"))

	// Tab styles
	tc.activeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	tc.inactiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)

	tc.separatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	// Tab bar container style with background
	tc.tabBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("0")) // Black background for visibility
	// Width will be set dynamically in Render()
	// No margins to keep header compact
}

// SetActiveView updates the active view
func (tc *TabsComponent) SetActiveView(view View) {
	tc.activeView = view
}

// SetWidth updates the component width
func (tc *TabsComponent) SetWidth(width int) {
	tc.width = width
}

// Render returns the rendered tab bar
func (tc *TabsComponent) Render() string {
	// Build title section
	title := tc.renderTitle()

	// Build tabs section
	tabs := tc.renderTabs()

	// Create the header layout using Lipgloss
	headerContainer := lipgloss.NewStyle().
		Width(tc.width)

	// Join title and tabs directly without extra margins
	fullHeader := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		tabs, // Render tabs directly without the tabBarStyle wrapper
	)

	// Apply header container style with bottom border
	styledHeader := headerContainer.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(lipgloss.Color("240")).
		Render(fullHeader)

	return styledHeader
}

// renderTitle renders the title section with process info and notifications
func (tc *TabsComponent) renderTitle() string {
	// Get process count information
	processes := tc.processMgr.GetAllProcesses()
	runningCount := 0
	for _, proc := range processes {
		if proc.GetStatus() == process.StatusRunning {
			runningCount++
		}
	}

	// Build title based on available width
	var baseTitle string
	var processInfo string

	if tc.width < 40 {
		// Ultra narrow - just emoji and counts
		baseTitle = "🐝"
		if len(processes) > 0 {
			processInfo = fmt.Sprintf(" %d/%d", runningCount, len(processes))
		}
	} else if tc.width < 60 {
		// Narrow - short title
		baseTitle = "🐝 Brummer"
		if len(processes) > 0 {
			processInfo = fmt.Sprintf(" (%d/%d)", runningCount, len(processes))
		}
	} else if tc.width < 100 {
		// Medium - abbreviated subtitle
		baseTitle = "🐝 Brummer - Dev Buddy"
		if len(processes) > 0 {
			if runningCount > 0 {
				processInfo = fmt.Sprintf(" (%d proc, %d run)", len(processes), runningCount)
			} else {
				processInfo = fmt.Sprintf(" (%d proc)", len(processes))
			}
		}
	} else {
		// Full width - complete title
		baseTitle = "🐝 Brummer - Development Buddy"
		if len(processes) > 0 {
			if runningCount > 0 {
				processInfo = fmt.Sprintf(" (%d processes, %d running)", len(processes), runningCount)
			} else {
				processInfo = fmt.Sprintf(" (%d processes)", len(processes))
			}
		}
	}

	// Add notification if active (skip for ultra narrow)
	notification := ""
	if tc.width >= 40 && tc.notificationsController != nil && tc.notificationsController.IsActive() {
		notificationStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true).
			MarginLeft(1)

		notificationText := tc.notificationsController.GetMessage()
		// Truncate notification for narrow screens
		if tc.width < 80 && len(notificationText) > 20 {
			notificationText = notificationText[:17] + "..."
		}
		notification = notificationStyle.Render(notificationText)
	}

	// Combine title parts
	titleText := baseTitle + processInfo

	// Create a container that uses the full width
	titleContainer := lipgloss.NewStyle().
		Width(tc.width)

	// Render title and notification
	titleSection := tc.titleStyle.Render(titleText)
	if notification != "" {
		titleSection = lipgloss.JoinHorizontal(
			lipgloss.Top,
			titleSection,
			notification,
		)
	}

	// Ensure the title is visible by not using the container if width is 0
	if tc.width <= 0 {
		return titleSection
	}

	return titleContainer.Render(titleSection)
}

// getTabRenderMode determines how to render tabs based on available width
func (tc *TabsComponent) getTabRenderMode() string {
	if tc.width < 40 {
		return "minimal" // Just key bindings: 1 2 3...
	} else if tc.width < 60 {
		return "abbreviated" // Short names: 1.Proc 2.Log...
	} else if tc.width < 100 {
		return "compact" // No icons: 1.Processes 2.Logs...
	}
	return "full" // Full with icons
}

// renderTabs renders the tab bar
func (tc *TabsComponent) renderTabs() string {
	mode := tc.getTabRenderMode()
	var tabs []string

	for _, viewType := range tc.views {
		if cfg, ok := viewConfigs[viewType]; ok {
			tab := tc.renderTabResponsive(viewType, cfg, mode)
			tabs = append(tabs, tab)
		}
	}

	// Join tabs with proper spacing
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		tabs...,
	)
}

// getAbbreviation returns abbreviated form of the title
func (tc *TabsComponent) getAbbreviation(title string) string {
	abbreviations := map[string]string{
		"Processes":       "Proc",
		"Logs":            "Log",
		"Errors":          "Err",
		"URLs":            "URL",
		"Web View":        "Web",
		"AI Coders":       "AI",
		"Settings":        "Set",
		"MCP Connections": "MCP",
	}

	if abbr, ok := abbreviations[title]; ok {
		return abbr
	}
	// Default: take first 3-4 chars
	if len(title) > 4 {
		return title[:4]
	}
	return title
}

// renderTabResponsive renders a tab based on the current render mode
func (tc *TabsComponent) renderTabResponsive(viewType View, cfg ViewConfig, mode string) string {
	var label string

	// Build label based on mode
	switch mode {
	case "minimal":
		// Just the key binding
		label = cfg.KeyBinding
	case "abbreviated":
		// Key + abbreviated name
		abbr := tc.getAbbreviation(cfg.Title)
		label = fmt.Sprintf("%s.%s", cfg.KeyBinding, abbr)
	case "compact":
		// Key + full name, no icon
		label = fmt.Sprintf("%s.%s", cfg.KeyBinding, cfg.Title)
	default: // "full"
		// Icon + key + full name
		label = fmt.Sprintf("%s.%s", cfg.KeyBinding, cfg.Title)
		if cfg.Icon != "" {
			label = cfg.Icon + " " + label
		}
	}

	// Get unread indicator (only show in non-minimal modes)
	indicatorIcon := ""
	if mode != "minimal" && tc.systemController != nil {
		indicators := tc.systemController.GetUnreadIndicators()
		if indicator, exists := indicators[viewType]; exists && indicator.Count > 0 {
			// Use simpler indicator for narrow screens
			if mode == "abbreviated" {
				indicatorIcon = "*"
			} else {
				indicatorIcon = " " + indicator.Icon
			}
		}
	}

	// Render the tab with appropriate style
	var tabContent string
	if viewType == tc.activeView {
		// Active tab with selection indicator (simpler for narrow screens)
		activeIndicator := "▶ "
		if mode == "minimal" {
			activeIndicator = ">"
		}
		tabContent = tc.activeStyle.Render(activeIndicator + label + indicatorIcon)
	} else {
		// Inactive tab
		tabContent = tc.inactiveStyle.Render(label + indicatorIcon)
	}

	// Add separator (simpler for narrow screens)
	if viewType != tc.views[len(tc.views)-1] {
		separatorChar := " │ "
		if mode == "minimal" {
			separatorChar = " "
		} else if mode == "abbreviated" {
			separatorChar = "|"
		}
		separator := tc.separatorStyle.Render(separatorChar)
		tabContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			tabContent,
			separator,
		)
	}

	return tabContent
}

// renderTab renders a single tab (legacy method, keeping for compatibility)
func (tc *TabsComponent) renderTab(viewType View, cfg ViewConfig) string {
	return tc.renderTabResponsive(viewType, cfg, "full")
}

// GetHeight returns the height of the rendered tab bar
func (tc *TabsComponent) GetHeight() int {
	// Title (1) + tabs (1) + border (1) = 3
	return 3
}
