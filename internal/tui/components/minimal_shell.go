package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// MinimalShell is a clean, simple component system shell for testing
type MinimalShell struct {
	width       int
	height      int
	currentView string
	quit        bool

	// Notification tracking
	notifications map[string]int
}

// Quit returns whether the shell should quit
func (m *MinimalShell) Quit() bool {
	return m.quit
}

// NewMinimalShell creates a minimal working shell
func NewMinimalShell() *MinimalShell {
	// Force color output for terminal UI
	lipgloss.SetColorProfile(termenv.ANSI256)

	shell := &MinimalShell{
		width:         80,
		height:        24,
		currentView:   "processes",
		notifications: make(map[string]int),
	}

	// Add some demo notifications
	shell.notifications["logs"] = 5
	shell.notifications["errors"] = 2
	shell.notifications["urls"] = 1

	return shell
}

// Init initializes the shell
func (m *MinimalShell) Init() tea.Cmd {
	// Send initial window size
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: 80, Height: 24}
	}
}

// Update handles all messages
func (m *MinimalShell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Always quit on Ctrl+C
			m.quit = true
			return m, tea.Quit
		case "q":
			// Quit unless in input field
			m.quit = true
			return m, tea.Quit
		case "1":
			if m.currentView != "processes" {
				delete(m.notifications, "processes")
			}
			m.currentView = "processes"
		case "2":
			if m.currentView != "logs" {
				delete(m.notifications, "logs")
			}
			m.currentView = "logs"
		case "3":
			if m.currentView != "errors" {
				delete(m.notifications, "errors")
			}
			m.currentView = "errors"
		case "4":
			if m.currentView != "urls" {
				delete(m.notifications, "urls")
			}
			m.currentView = "urls"
		case "5":
			if m.currentView != "web" {
				delete(m.notifications, "web")
			}
			m.currentView = "web"
		case "right", "alt+right":
			// Cycle to next view with arrow keys
			views := []string{"processes", "logs", "errors", "urls", "web"}
			for i, v := range views {
				if v == m.currentView {
					nextView := views[(i+1)%len(views)]
					// Clear notifications when switching
					delete(m.notifications, nextView)
					m.currentView = nextView
					break
				}
			}
		case "left", "alt+left":
			// Cycle to previous view with arrow keys
			views := []string{"processes", "logs", "errors", "urls", "web"}
			for i, v := range views {
				if v == m.currentView {
					prevView := views[(i-1+len(views))%len(views)]
					// Clear notifications when switching
					delete(m.notifications, prevView)
					m.currentView = prevView
					break
				}
			}
		}
		return m, nil
	}

	return m, nil
}

// View renders the shell
func (m *MinimalShell) View() string {
	if m.quit {
		return "Goodbye!\n"
	}

	// Header
	header := m.renderHeader()

	// Main content area
	content := m.renderContent()

	// Footer
	footer := m.renderFooter()

	// Calculate heights
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	contentHeight := m.height - headerHeight - footerHeight - 1 // -1 for safety

	// Ensure content fills the space
	// IMPORTANT: Width must be terminal width - 2 for the border characters
	contentStyle := lipgloss.NewStyle().
		Width(m.width - 2). // Subtract 2 for left and right border
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	styledContent := contentStyle.Render(content)

	// Join all parts
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		styledContent,
		footer,
	)
}

func (m *MinimalShell) renderHeader() string {
	// Title line
	title := "🎯 Brummer Process Manager"

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Background(lipgloss.Color("235")).
		Width(m.width).
		Align(lipgloss.Center)

	// Tab configuration with icons
	tabConfigs := []struct {
		key  string
		name string
		view string
		icon string
	}{
		{"1", "Processes", "processes", "🏃"},
		{"2", "Logs", "logs", "📄"},
		{"3", "Errors", "errors", "❌"},
		{"4", "URLs", "urls", "🔗"},
		{"5", "Web", "web", "🌐"},
	}

	var tabLine []string
	for _, cfg := range tabConfigs {
		// Build tab label with icon - simplified format
		label := fmt.Sprintf("%s %s.%s", cfg.icon, cfg.key, cfg.name)

		// Add notification badge if present
		if count, hasNotif := m.notifications[cfg.view]; hasNotif && count > 0 {
			// Choose notification emoji based on view type
			notifEmoji := "🔵" // Default info
			if cfg.view == "errors" {
				notifEmoji = "🔴" // Red for errors
			} else if cfg.view == "urls" {
				notifEmoji = "🟡" // Yellow for URLs (warning-like)
			}

			// Add count badge
			if count > 99 {
				label += fmt.Sprintf(" %s 99+", notifEmoji)
			} else {
				label += fmt.Sprintf(" %s %d", notifEmoji, count)
			}
		}

		// Add selection indicator or reserve space
		var indicator string
		if cfg.view == m.currentView {
			indicator = "▶"
		} else {
			indicator = " " // Reserve space for alignment
		}

		// Combine indicator and label
		fullLabel := indicator + label

		// Apply style based on active state
		style := lipgloss.NewStyle()
		if cfg.view == m.currentView {
			style = style.Bold(true).
				Foreground(lipgloss.Color("205")).
				Background(lipgloss.Color("236"))
		} else {
			// Inactive tab
			style = style.Foreground(lipgloss.Color("241"))
		}
		tabLine = append(tabLine, style.Render(fullLabel))
	}

	// Join tabs with separator
	separator := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" │ ")
	tabsStr := strings.Join(tabLine, separator)

	// Left-align tabs with padding
	tabsStyle := lipgloss.NewStyle().
		Width(m.width).
		PaddingLeft(2)

	// Join title and tabs
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		tabsStyle.Render(tabsStr),
	)
}

func (m *MinimalShell) renderContent() string {
	// Render empty panel based on current view
	var content string

	switch m.currentView {
	case "processes":
		content = "Process View (Empty)\n\nNo processes running"
	case "logs":
		content = "Logs View (Empty)\n\nNo logs available"
	case "errors":
		content = "Errors View (Empty)\n\nNo errors detected"
	case "urls":
		content = "URLs View (Empty)\n\nNo URLs discovered"
	case "web":
		content = "Web View (Empty)\n\nNo web content"
	default:
		content = "Unknown view: " + m.currentView
	}

	// Add some debug info
	content += fmt.Sprintf("\n\nWindow: %dx%d", m.width, m.height)
	content += fmt.Sprintf("\nCurrent View: %s", m.currentView)

	return content
}

func (m *MinimalShell) renderFooter() string {
	help := "1-5: Switch views | ←→: Navigate | Ctrl+C: Quit"

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Width(m.width).
		Align(lipgloss.Center).
		Padding(0, 1)

	return footerStyle.Render(help)
}
