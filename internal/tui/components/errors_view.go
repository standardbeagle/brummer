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

// ErrorsViewComponent displays aggregated error contexts in a list format.
// It receives LogAddedMsg messages and filters for errors only.
//
// Design principles:
// - Event-driven via LogAddedMsg (filters IsError=true)
// - Aggregates errors using LogStore.GetErrorContexts()
// - Shows error type, message, and count
// - Supports keyboard scrolling and selection
type ErrorsViewComponent struct {
	BaseComponent

	viewport viewport.Model
	errors   []*ErrorInfo // Aggregated error information

	// Read-only reference for error aggregation
	logStore *logs.Store

	// Selection state
	selectedIndex int

	// Styles
	headerStyle   lipgloss.Style
	errorStyle    lipgloss.Style
	warningStyle  lipgloss.Style
	criticalStyle lipgloss.Style
	normalStyle   lipgloss.Style
	selectedStyle lipgloss.Style
}

// ErrorInfo holds display information for an error context
type ErrorInfo struct {
	ID          string
	Type        string
	Message     string
	ProcessName string
	Timestamp   string
	Severity    string
	Count       int // Number of similar errors
	Context     *logs.ErrorContext
}

// NewErrorsViewComponent creates an errors view with initial error contexts
func NewErrorsViewComponent(logStore *logs.Store) *ErrorsViewComponent {
	c := &ErrorsViewComponent{
		BaseComponent: NewBaseComponent(),
		logStore:      logStore,
		viewport:      viewport.New(80, 24),
		errors:        []*ErrorInfo{},
		selectedIndex: 0,
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")), // Bright Blue
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")),             // Bright Red
		warningStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("11")),            // Bright Yellow
		criticalStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),  // Bold Red
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),            // White
		selectedStyle: lipgloss.NewStyle().Background(lipgloss.Color("8")),             // Gray background
	}

	// Load existing errors
	c.refreshErrors()

	return c
}

// Init initializes the component
func (c *ErrorsViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *ErrorsViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.LogAddedMsg:
		// NEW ERROR LOG! Filter and aggregate
		if msg.Entry.IsError {
			// Refresh error aggregation from LogStore
			c.refreshErrors()
			c.updateViewport()
		}
		return c, nil

	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height
		c.updateViewport()
		return c, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if c.selectedIndex > 0 {
				c.selectedIndex--
				c.updateViewport()
			}
			return c, nil
		case "down", "j":
			if c.selectedIndex < len(c.errors)-1 {
				c.selectedIndex++
				c.updateViewport()
			}
			return c, nil
		default:
			// Forward other keys to viewport for scrolling
			c.viewport, cmd = c.viewport.Update(msg)
			return c, cmd
		}
	}

	return c, nil
}

// View renders the errors list
func (c *ErrorsViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	return c.viewport.View()
}

// SetSize updates component dimensions
func (c *ErrorsViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.viewport.Width = width
	c.viewport.Height = height
	c.updateViewport()
}

// refreshErrors reloads error contexts from LogStore and aggregates them
func (c *ErrorsViewComponent) refreshErrors() {
	if c.logStore == nil {
		return
	}

	// Get error contexts from LogStore (already aggregated)
	errorContexts := c.logStore.GetErrorContexts()

	// Convert to ErrorInfo
	c.errors = make([]*ErrorInfo, 0, len(errorContexts))
	for i := range errorContexts {
		ctx := &errorContexts[i]
		c.errors = append(c.errors, &ErrorInfo{
			ID:          ctx.ID,
			Type:        ctx.Type,
			Message:     ctx.Message,
			ProcessName: ctx.ProcessName,
			Timestamp:   ctx.Timestamp.Format("15:04:05"),
			Severity:    ctx.Severity,
			Count:       1, // LogStore already does aggregation
			Context:     ctx,
		})
	}

	// Ensure selected index is valid
	if c.selectedIndex >= len(c.errors) {
		c.selectedIndex = len(c.errors) - 1
	}
	if c.selectedIndex < 0 {
		c.selectedIndex = 0
	}
}

// updateViewport updates the viewport content with formatted errors list
func (c *ErrorsViewComponent) updateViewport() {
	var lines []string

	// Header
	header := c.headerStyle.Render(fmt.Sprintf("%-20s %-15s %-50s",
		"ERROR TYPE", "PROCESS", "MESSAGE"))
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", c.viewport.Width))

	// Error rows
	if len(c.errors) == 0 {
		lines = append(lines, c.normalStyle.Render("No errors detected"))
	} else {
		for i, info := range c.errors {
			line := c.formatErrorRow(info, i == c.selectedIndex)
			lines = append(lines, line)
		}

		// Show selected error details below the list
		if c.selectedIndex < len(c.errors) && c.selectedIndex >= 0 {
			lines = append(lines, "")
			lines = append(lines, c.headerStyle.Render("Error Details:"))
			lines = append(lines, c.formatErrorDetails(c.errors[c.selectedIndex]))
		}
	}

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)
}

// formatErrorRow formats a single error row
func (c *ErrorsViewComponent) formatErrorRow(info *ErrorInfo, selected bool) string {
	// Truncate long strings
	errorType := info.Type
	if len(errorType) > 20 {
		errorType = errorType[:17] + "..."
	}

	processName := info.ProcessName
	if len(processName) > 15 {
		processName = processName[:12] + "..."
	}

	message := info.Message
	if len(message) > 50 {
		message = message[:47] + "..."
	}

	// Choose style based on severity
	var style lipgloss.Style
	switch info.Severity {
	case "critical":
		style = c.criticalStyle
	case "error":
		style = c.errorStyle
	case "warning":
		style = c.warningStyle
	default:
		style = c.normalStyle
	}

	row := fmt.Sprintf("%-20s %-15s %-50s",
		errorType,
		processName,
		message,
	)

	// Apply selection highlight
	if selected {
		row = c.selectedStyle.Render("→ " + row)
	} else {
		row = "  " + style.Render(row)
	}

	return row
}

// formatErrorDetails formats detailed error information
func (c *ErrorsViewComponent) formatErrorDetails(info *ErrorInfo) string {
	var lines []string

	// Basic info
	lines = append(lines, c.normalStyle.Render(fmt.Sprintf("Type: %s", info.Type)))
	lines = append(lines, c.normalStyle.Render(fmt.Sprintf("Process: %s", info.ProcessName)))
	lines = append(lines, c.normalStyle.Render(fmt.Sprintf("Time: %s", info.Timestamp)))
	lines = append(lines, c.normalStyle.Render(fmt.Sprintf("Severity: %s", info.Severity)))
	lines = append(lines, "")

	// Message
	lines = append(lines, c.errorStyle.Render("Message:"))
	lines = append(lines, c.normalStyle.Render(info.Message))
	lines = append(lines, "")

	// Stack trace (if available)
	if info.Context != nil && len(info.Context.Stack) > 0 {
		lines = append(lines, c.normalStyle.Render("Stack Trace:"))
		maxStackLines := 10
		for i, stackLine := range info.Context.Stack {
			if i >= maxStackLines {
				lines = append(lines, c.normalStyle.Render(fmt.Sprintf("  ... (%d more lines)", len(info.Context.Stack)-maxStackLines)))
				break
			}
			lines = append(lines, c.normalStyle.Render(fmt.Sprintf("  %s", stackLine)))
		}
	}

	return strings.Join(lines, "\n")
}
