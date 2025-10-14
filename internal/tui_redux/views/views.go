package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui_redux/model"
)

// Styles for the TUI
var (
	// Base styles
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#007ACC")).
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C757D")).
			MarginBottom(1)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C757D")).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F44747")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#28A745")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFC107")).
			Bold(true)

	// View-specific styles
	viewStyle = lipgloss.NewStyle().
			Margin(1).
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#ABB2BF"))

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#007ACC")).
			Bold(true).
			Underline(true)

	inactiveTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ABB2BF"))
)

// Render renders the complete TUI based on the current model state
func Render(m model.Model) string {
	// Get window dimensions (using reasonable defaults)
	width, height := 80, 24

	// Build the UI
	var content strings.Builder

	// Header
	content.WriteString(renderHeader(m))

	// Main content area based on current view
	content.WriteString(renderMainContent(m, width, height-6)) // Reserve space for header and footer

	// Footer
	content.WriteString(renderFooter(m))

	return content.String()
}

// renderHeader renders the top header with title and tabs
func renderHeader(m model.Model) string {
	title := titleStyle.Render("Brummer - Redux/Elm TUI")

	// Create view tabs
	tabs := make([]string, 7)
	views := []model.View{
		model.ViewProcesses,
		model.ViewLogs,
		model.ViewErrors,
		model.ViewURLs,
		model.ViewWeb,
		model.ViewAICoders,
		model.ViewSettings,
	}

	viewNames := []string{"1.Processes", "2.Logs", "3.Errors", "4.URLs", "5.Web", "6.AI", "7.Settings"}

	currentView := m.CurrentView()

	for i, view := range views {
		if view == currentView {
			tabs[i] = activeTabStyle.Render(viewNames[i])
		} else {
			tabs[i] = inactiveTabStyle.Render(viewNames[i])
		}
	}

	tabLine := strings.Join(tabs, " | ")

	return fmt.Sprintf("%s\n%s", title, tabLine)
}

// renderMainContent renders the main content area based on current view
func renderMainContent(m model.Model, width, height int) string {
	switch m.CurrentView() {
	case model.ViewProcesses:
		return renderProcessesView(m, width, height)
	case model.ViewLogs:
		return renderLogsView(m, width, height)
	case model.ViewErrors:
		return renderErrorsView(m, width, height)
	case model.ViewURLs:
		return renderURLsView(m, width, height)
	case model.ViewWeb:
		return renderWebView(m, width, height)
	case model.ViewAICoders:
		return renderAICodersView(m, width, height)
	case model.ViewSettings:
		return renderSettingsView(m, width, height)
	default:
		return viewStyle.Render("Unknown view")
	}
}

// renderFooter renders the bottom footer with status and help
func renderFooter(m model.Model) string {
	status := "Ready"
	if m.IsInitializing() {
		status = "Initializing..."
	}

	helpText := "[q]uit [1-7] Switch View [/] Command Palette"

	return fmt.Sprintf("%s | %s",
		footerStyle.Render(status),
		footerStyle.Render(helpText))
}

// View-specific render functions

func renderProcessesView(m model.Model, width, height int) string {
	content := "Processes View\n\n"

	processes := m.Processes()
	if len(processes) == 0 {
		content += "No processes running\n"
		content += "\nPress 'Enter' to start a new process"
	} else {
		content += fmt.Sprintf("Running Processes: %d\n\n", len(processes))
		for id, process := range processes {
			status := "Running"
			if !process.IsRunning() {
				status = "Stopped"
			}
			content += fmt.Sprintf("  [%s] %s (%s)\n", status, process.Name, id)
		}
	}

	return viewStyle.Render(content)
}

func renderLogsView(m model.Model, width, height int) string {
	content := "Logs View\n\n"

	logs := m.FilteredLogs()
	if len(logs) == 0 {
		content += "No logs available\n"
		content += "\nSelect a process to view its logs"
	} else {
		content += fmt.Sprintf("Log Entries: %d (Filter: '%s')\n\n", len(logs), m.LogFilter())

		// Show recent logs (limited by height)
		maxLines := height - 8 // Account for borders and padding
		start := 0
		if len(logs) > maxLines {
			start = len(logs) - maxLines
		}

		for i := start; i < len(logs); i++ {
			log := logs[i]
			content += fmt.Sprintf("[%s] %s\n",
				log.Timestamp.Format("15:04:05"),
				log.Line)
		}
	}

	return viewStyle.Render(content)
}

func renderErrorsView(m model.Model, width, height int) string {
	content := "Errors View\n\n"

	errors := m.Errors()
	if len(errors) == 0 {
		content += "No errors detected\n"
		content += "\nGreat! Everything is running smoothly."
	} else {
		content += fmt.Sprintf("Errors Found: %d\n\n", len(errors))
		for i, err := range errors {
			if i >= 10 { // Limit display
				content += fmt.Sprintf("... and %d more errors\n", len(errors)-10)
				break
			}
			content += fmt.Sprintf("Error: %s\n", err.Error)
			if err.Context != "" {
				content += fmt.Sprintf("   Context: %s\n", err.Context)
			}
			content += "\n"
		}
	}

	return viewStyle.Render(content)
}

func renderURLsView(m model.Model, width, height int) string {
	content := "Discovered URLs\n\n"

	// Get URLs from all processes
	allURLs := make([]model.URL, 0)
	// This would normally get URLs from the model
	// For now, show placeholder

	if len(allURLs) == 0 {
		content += "No URLs discovered yet\n"
		content += "\nURLs will appear as processes log them"
	} else {
		content += fmt.Sprintf("Total URLs: %d\n\n", len(allURLs))
		for i, url := range allURLs {
			if i >= 20 { // Limit display
				break
			}
			content += fmt.Sprintf("Link: %s", url.URL)
			if url.Label != "" {
				content += fmt.Sprintf(" (%s)", url.Label)
			}
			content += "\n"
		}
	}

	return viewStyle.Render(content)
}

func renderWebView(m model.Model, width, height int) string {
	content := "Web Interface\n\n"
	content += "Web view will show:\n"
	content += "- Process web interfaces\n"
	content += "- Local development servers\n"
	content += "- Browser automation tools\n\n"
	content += "No web interfaces detected yet"

	return viewStyle.Render(content)
}

func renderAICodersView(m model.Model, width, height int) string {
	content := "AI Coders\n\n"

	// Get AI sessions
	sessions := make([]*model.AISession, 0)
	// This would normally get sessions from the model

	if len(sessions) == 0 {
		content += "No AI sessions active\n\n"
		content += "Start an AI coding session:\n"
		content += "- Configure AI provider\n"
		content += "- Select coding model\n"
		content += "- Begin conversation"
	} else {
		content += fmt.Sprintf("Active Sessions: %d\n\n", len(sessions))
		for _, session := range sessions {
			content += fmt.Sprintf("AI: %s (%s)\n", session.Name, session.State)
		}
	}

	return viewStyle.Render(content)
}

func renderSettingsView(m model.Model, width, height int) string {
	content := "Settings\n\n"

	settings := m.Settings()

	content += fmt.Sprintf("Theme: %s\n", settings.Theme)
	content += fmt.Sprintf("Max Log Entries: %d\n", settings.MaxLogEntries)
	content += fmt.Sprintf("Auto Refresh: %t\n", settings.AutoRefresh)
	content += fmt.Sprintf("Debug Mode: %t\n", settings.Debug)
	content += fmt.Sprintf("Working Directory: %s\n", settings.WorkDir)

	return viewStyle.Render(content)
}