package components

import (
	"fmt"
	"strings"
	
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
)

// LogsViewComponent adapts the logs view to the component system
type LogsViewComponent struct {
	BaseComponent
	
	// Component state
	logEntries   []*logs.LogEntry
	filteredLogs []string
	viewport     viewport.Model
	filterInput  textinput.Model
	
	// Filter state
	currentFilter string
	processFilter string
	levelFilter   string
	following     bool
	
	// Display options
	showTimestamps bool
	showProcessID  bool
	wrapLines      bool
	
	// Styles
	titleStyle    lipgloss.Style
	helpStyle     lipgloss.Style
	filterStyle   lipgloss.Style
	logStyle      lipgloss.Style
	errorStyle    lipgloss.Style
	warnStyle     lipgloss.Style
	infoStyle     lipgloss.Style
	debugStyle    lipgloss.Style
	
	// Log store reference
	logStore     *logs.Store
}

// NewLogsViewComponent creates a new logs view component
func NewLogsViewComponent(logStore *logs.Store) *LogsViewComponent {
	l := &LogsViewComponent{
		logStore:       logStore,
		logEntries:     []*logs.LogEntry{},
		filteredLogs:   []string{},
		following:      true,
		showTimestamps: true,
		showProcessID:  true,
		wrapLines:      true,
	}
	
	// Initialize styles
	l.initStyles()
	
	// Initialize viewport
	l.viewport = viewport.New(80, 20)
	l.viewport.SetContent("")
	
	// Initialize filter input
	l.filterInput = textinput.New()
	l.filterInput.Placeholder = "Filter logs..."
	l.filterInput.CharLimit = 100
	l.filterInput.Width = 40
	
	// Set visible by default
	l.visible = true
	
	return l
}

// initStyles initializes the component styles
func (l *LogsViewComponent) initStyles() {
	l.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
	
	l.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	l.filterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205"))
	
	l.logStyle = lipgloss.NewStyle()
	
	l.errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
	
	l.warnStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))
	
	l.infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86"))
	
	l.debugStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
}

// Update handles messages
func (l *LogsViewComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle component-specific messages
	switch msg := msg.(type) {
	case LogsUpdateMsg:
		// Update log entries
		if msg.Append {
			l.logEntries = append(l.logEntries, msg.Entries...)
		} else {
			l.logEntries = msg.Entries
		}
		l.applyFilter()
		l.updateViewport()
		
		// Auto-scroll if following
		if l.following {
			l.viewport.GotoBottom()
		}
		return l, nil
		
	case LogsFilterMsg:
		// Apply filter
		l.currentFilter = msg.Filter
		l.processFilter = msg.ProcessID
		l.levelFilter = msg.Level
		l.applyFilter()
		l.updateViewport()
		return l, nil
		
	case LogsClearMsg:
		// Clear logs
		if msg.ProcessID != "" {
			// Clear logs for specific process
			newEntries := []*logs.LogEntry{}
			for _, entry := range l.logEntries {
				if entry.ProcessID != msg.ProcessID {
					newEntries = append(newEntries, entry)
				}
			}
			l.logEntries = newEntries
		} else {
			// Clear all logs
			l.logEntries = []*logs.LogEntry{}
		}
		l.applyFilter()
		l.updateViewport()
		return l, nil
		
	case LogsFollowMsg:
		// Toggle following
		l.following = msg.Follow
		if l.following {
			l.viewport.GotoBottom()
		}
		return l, nil
		
	case LogsScrollMsg:
		// Handle scrolling
		switch msg.Direction {
		case ScrollUp:
			l.viewport.LineUp(msg.Amount)
			l.following = false
		case ScrollDown:
			l.viewport.LineDown(msg.Amount)
			// Check if we're at the bottom
			if l.viewport.AtBottom() {
				l.following = true
			}
		case ScrollPageUp:
			l.viewport.ViewUp()
			l.following = false
		case ScrollPageDown:
			l.viewport.ViewDown()
			if l.viewport.AtBottom() {
				l.following = true
			}
		case ScrollHome:
			l.viewport.GotoTop()
			l.following = false
		case ScrollEnd:
			l.viewport.GotoBottom()
			l.following = true
		}
		return l, nil
		
	case tea.WindowSizeMsg:
		l.width = msg.Width
		l.height = msg.Height
		headerHeight := 3
		footerHeight := 2
		l.viewport.Width = msg.Width
		l.viewport.Height = msg.Height - headerHeight - footerHeight
		l.updateViewport()
		
	case tea.KeyMsg:
		// Handle key presses
		if l.filterInput.Focused() {
			switch msg.String() {
			case "enter":
				// Apply filter
				l.currentFilter = l.filterInput.Value()
				l.applyFilter()
				l.updateViewport()
				l.filterInput.Blur()
				return l, nil
				
			case "esc":
				// Cancel filter
				l.filterInput.SetValue(l.currentFilter)
				l.filterInput.Blur()
				return l, nil
			}
			
			// Update filter input
			var cmd tea.Cmd
			l.filterInput, cmd = l.filterInput.Update(msg)
			return l, cmd
		}
		
		// Handle viewport navigation
		switch msg.String() {
		case "/":
			// Focus filter input
			l.filterInput.Focus()
			return l, textinput.Blink
			
		case "c":
			// Clear logs
			return l, func() tea.Msg {
				return LogsClearMsg{ProcessID: l.processFilter}
			}
			
		case "C":
			// Clear all logs
			return l, func() tea.Msg {
				return LogsClearMsg{}
			}
			
		case "f":
			// Toggle following
			l.following = !l.following
			if l.following {
				l.viewport.GotoBottom()
			}
			return l, nil
			
		case "t":
			// Toggle timestamps
			l.showTimestamps = !l.showTimestamps
			l.updateViewport()
			return l, nil
			
		case "p":
			// Toggle process IDs
			l.showProcessID = !l.showProcessID
			l.updateViewport()
			return l, nil
			
		case "w":
			// Toggle line wrapping
			l.wrapLines = !l.wrapLines
			l.updateViewport()
			return l, nil
			
		case "g":
			// Go to top
			l.viewport.GotoTop()
			l.following = false
			return l, nil
			
		case "G":
			// Go to bottom
			l.viewport.GotoBottom()
			l.following = true
			return l, nil
			
		case "k", "up":
			l.viewport.LineUp(1)
			l.following = false
			
		case "j", "down":
			l.viewport.LineDown(1)
			if l.viewport.AtBottom() {
				l.following = true
			}
			
		case "u", "pgup":
			l.viewport.ViewUp()
			l.following = false
			
		case "d", "pgdown":
			l.viewport.ViewDown()
			if l.viewport.AtBottom() {
				l.following = true
			}
		}
	}
	
	// Update viewport
	if !l.filterInput.Focused() {
		var cmd tea.Cmd
		l.viewport, cmd = l.viewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	return l, tea.Batch(cmds...)
}

// View renders the component
func (l *LogsViewComponent) View() string {
	if !l.visible {
		return ""
	}
	
	var s strings.Builder
	
	// Header
	title := "📄 Logs"
	if l.processFilter != "" {
		title += fmt.Sprintf(" (Process: %s)", l.processFilter)
	}
	if l.currentFilter != "" {
		title += fmt.Sprintf(" [Filter: %s]", l.currentFilter)
	}
	if l.following {
		title += " [Following]"
	}
	s.WriteString(l.titleStyle.Render(title) + "\n")
	
	// Filter input or help
	if l.filterInput.Focused() {
		s.WriteString(l.filterStyle.Render("Filter: ") + l.filterInput.View() + "\n")
	} else {
		help := []string{
			"/: filter",
			"f: follow",
			"c: clear",
			"t: timestamps",
			"p: process IDs",
			"w: wrap",
		}
		s.WriteString(l.helpStyle.Render(strings.Join(help, " | ")) + "\n")
	}
	
	// Viewport
	s.WriteString(l.viewport.View())
	
	// Footer
	scrollPercent := l.viewport.ScrollPercent()
	footer := fmt.Sprintf("%d logs | %.0f%%", len(l.filteredLogs), scrollPercent*100)
	s.WriteString("\n" + l.helpStyle.Render(footer))
	
	return s.String()
}

// applyFilter filters the log entries based on current filters
func (l *LogsViewComponent) applyFilter() {
	l.filteredLogs = []string{}
	
	for _, entry := range l.logEntries {
		// Apply process filter
		if l.processFilter != "" && entry.ProcessID != l.processFilter {
			continue
		}
		
		// Apply level filter
		if l.levelFilter != "" && !l.matchesLevel(entry, l.levelFilter) {
			continue
		}
		
		// Apply text filter
		if l.currentFilter != "" {
			filterLower := strings.ToLower(l.currentFilter)
			contentLower := strings.ToLower(entry.Content)
			if !strings.Contains(contentLower, filterLower) {
				continue
			}
		}
		
		// Format log entry
		formatted := l.formatLogEntry(entry)
		l.filteredLogs = append(l.filteredLogs, formatted)
	}
}

// matchesLevel checks if a log entry matches the level filter
func (l *LogsViewComponent) matchesLevel(entry *logs.LogEntry, level string) bool {
	content := strings.ToLower(entry.Content)
	levelLower := strings.ToLower(level)
	
	switch levelLower {
	case "error":
		return strings.Contains(content, "error") || strings.Contains(content, "err") ||
			strings.Contains(content, "fatal") || strings.Contains(content, "panic")
	case "warn", "warning":
		return strings.Contains(content, "warn") || strings.Contains(content, "warning")
	case "info":
		return strings.Contains(content, "info")
	case "debug":
		return strings.Contains(content, "debug") || strings.Contains(content, "trace")
	default:
		return true
	}
}

// formatLogEntry formats a log entry for display
func (l *LogsViewComponent) formatLogEntry(entry *logs.LogEntry) string {
	var parts []string
	
	// Add timestamp
	if l.showTimestamps {
		timestamp := entry.Timestamp.Format("15:04:05.000")
		parts = append(parts, l.helpStyle.Render(timestamp))
	}
	
	// Add process ID
	if l.showProcessID && entry.ProcessID != "" {
		parts = append(parts, l.infoStyle.Render(fmt.Sprintf("[%s]", entry.ProcessID)))
	}
	
	// Style content based on level
	content := entry.Content
	contentLower := strings.ToLower(content)
	
	var styledContent string
	if strings.Contains(contentLower, "error") || strings.Contains(contentLower, "err") ||
		strings.Contains(contentLower, "fatal") || strings.Contains(contentLower, "panic") {
		styledContent = l.errorStyle.Render(content)
	} else if strings.Contains(contentLower, "warn") || strings.Contains(contentLower, "warning") {
		styledContent = l.warnStyle.Render(content)
	} else if strings.Contains(contentLower, "info") {
		styledContent = l.infoStyle.Render(content)
	} else if strings.Contains(contentLower, "debug") || strings.Contains(contentLower, "trace") {
		styledContent = l.debugStyle.Render(content)
	} else {
		styledContent = l.logStyle.Render(content)
	}
	
	// Wrap lines if needed
	if l.wrapLines && len(content) > l.viewport.Width {
		// Simple line wrapping
		wrapped := []string{}
		words := strings.Fields(content)
		currentLine := ""
		
		for _, word := range words {
			if len(currentLine)+len(word)+1 > l.viewport.Width {
				wrapped = append(wrapped, currentLine)
				currentLine = word
			} else {
				if currentLine != "" {
					currentLine += " "
				}
				currentLine += word
			}
		}
		if currentLine != "" {
			wrapped = append(wrapped, currentLine)
		}
		
		styledContent = strings.Join(wrapped, "\n    ")
	}
	
	parts = append(parts, styledContent)
	
	return strings.Join(parts, " ")
}

// updateViewport updates the viewport content
func (l *LogsViewComponent) updateViewport() {
	content := strings.Join(l.filteredLogs, "\n")
	l.viewport.SetContent(content)
}

// Focus sets focus to this component
func (l *LogsViewComponent) Focus() tea.Cmd {
	l.focused = true
	return nil
}

// Blur removes focus from this component
func (l *LogsViewComponent) Blur() tea.Cmd {
	l.focused = false
	l.filterInput.Blur()
	return nil
}