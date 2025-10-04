package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// LogsViewComponent displays logs in real-time using a viewport.
// It receives LogAddedMsg messages and appends them to the display.
//
// Design principles:
// - No direct access to LogStore (read-only initial load is acceptable)
// - All updates via LogAddedMsg messages
// - Viewport auto-scrolls to bottom on new logs
// - Supports scrolling with arrow keys and page up/down
type LogsViewComponent struct {
	BaseComponent

	viewport viewport.Model
	logs     []string

	// Read-only reference for initial load
	logStore *logs.Store

	// Styles
	errorStyle  lipgloss.Style
	normalStyle lipgloss.Style
}

// NewLogsViewComponent creates a logs view with initial logs from store
func NewLogsViewComponent(logStore *logs.Store) *LogsViewComponent {
	c := &LogsViewComponent{
		BaseComponent: NewBaseComponent(),
		logStore:      logStore,
		viewport:      viewport.New(80, 24),
		logs:          []string{},
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")),  // Red
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")), // White
	}

	// Load existing logs (one-time read)
	// Note: LogStore.GetAll() returns a copy, so this is safe
	if logStore != nil {
		entries := logStore.GetAll()
		c.logs = make([]string, 0, len(entries))
		for i := range entries {
			c.logs = append(c.logs, c.formatEntry(&entries[i]))
		}
		c.updateViewport()
	}

	return c
}

// Init initializes the component
func (c *LogsViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *LogsViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.LogAddedMsg:
		// NEW LOG! This is the event-driven magic!
		c.logs = append(c.logs, c.formatEntry(msg.Entry))
		c.updateViewport()
		return c, nil

	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height
		c.updateViewport()
		return c, nil

	case tea.KeyMsg:
		// Forward keyboard input to viewport for scrolling
		c.viewport, cmd = c.viewport.Update(msg)
		return c, cmd
	}

	return c, nil
}

// View renders the logs viewport
func (c *LogsViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	return c.viewport.View()
}

// SetSize updates component dimensions
func (c *LogsViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.viewport.Width = width
	c.viewport.Height = height
	c.updateViewport()
}

// formatEntry formats a log entry for display
func (c *LogsViewComponent) formatEntry(entry *logs.LogEntry) string {
	timestamp := entry.Timestamp.Format("15:04:05")
	processName := entry.ProcessName

	// Truncate long process names
	if len(processName) > 12 {
		processName = processName[:12]
	}

	style := c.normalStyle
	if entry.IsError {
		style = c.errorStyle
	}

	// Format: [HH:MM:SS] ProcessName: content
	line := fmt.Sprintf("[%s] %-12s: %s",
		timestamp,
		processName,
		entry.Content,
	)

	return style.Render(line)
}

// updateViewport updates the viewport content and scrolls to bottom
func (c *LogsViewComponent) updateViewport() {
	content := strings.Join(c.logs, "\n")
	c.viewport.SetContent(content)

	// Auto-scroll to bottom if not manually scrolled
	// Check if user is near the bottom (within last 5 lines)
	if c.viewport.YOffset >= c.viewport.TotalLineCount()-c.viewport.Height-5 {
		c.viewport.GotoBottom()
	}
}
