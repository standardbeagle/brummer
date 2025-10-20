package components

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// LogsViewComponent is a stateless BubbleTea component for displaying process logs
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type LogsViewComponent struct {
	BaseComponent

	// UI configuration only (no state)
	width  int
	height int

	// Styles
	headerStyle      lipgloss.Style
	searchStyle      lipgloss.Style
	normalStyle      lipgloss.Style
	errorStyle       lipgloss.Style
	warningStyle     lipgloss.Style
	infoStyle        lipgloss.Style
	timestampStyle   lipgloss.Style
	processStyle     lipgloss.Style
	countStyle       lipgloss.Style
	emptyStyle       lipgloss.Style
	filterStyle      lipgloss.Style
	loadingStyle     lipgloss.Style
}

// NewLogsViewComponent creates a new stateless logs view component
func NewLogsViewComponent() *LogsViewComponent {
	return &LogsViewComponent{
		BaseComponent: NewBaseComponent(),
		width:         80,
		height:        24,

		// Initialize styles
		headerStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		searchStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		normalStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		errorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		warningStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		infoStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		timestampStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		processStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		countStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true),
		emptyStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true).Align(lipgloss.Center, lipgloss.Center),
		filterStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		loadingStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("247")).Italic(true),
	}
}

// Init initializes the component
func (c *LogsViewComponent) Init() tea.Cmd {
	return nil
}

// ViewWithState renders the component using external state (stateless/Elm-compliant)
//
// ⚠️  CRITICAL: This method MUST be pure and non-blocking!
// ❌ WRONG:  logs := logStore.FetchAllLogs()  // DON'T call blocking methods!
// WHY: View() is called frequently and MUST return immediately. Blocking operations
//      freeze the BubbleTea event loop, causing UI freezes and breaking keyboard input.
// ✅ CORRECT: Only read from masterData (cached state) and use logStore read-only methods.
//
// See: /docs/architecture/tui-deadlock-prevention.md for mandatory patterns
func (c *LogsViewComponent) ViewWithState(masterData *state.MasterData, logStore *logs.Store) string {
	if !c.IsVisible() || masterData == nil || masterData.LogsState == nil {
		return ""
	}

	logsState := masterData.LogsState

	// Build header
	header := c.renderHeader(logsState)

	// Build content area
	content := c.renderContent(logsState, logStore)

	// Build footer
	footer := c.renderFooter(logsState)

	// Combine all sections
	return strings.Join([]string{header, content, footer}, "\n")
}

// renderHeader creates the header section with search and filter info
func (c *LogsViewComponent) renderHeader(logsState *state.LogsState) string {
	var lines []string

	// Main header
	title := c.headerStyle.Render("📝 Logs")
	if logsState.GetSelectedProcess() != "" {
		title += " " + c.processStyle.Render(fmt.Sprintf("[%s]", logsState.GetSelectedProcess()))
	}
	lines = append(lines, title)
	lines = append(lines, strings.Repeat("─", c.width))

	// Filter information
	var filterInfo []string
	if logsState.GetShowHighPriority() {
		filterInfo = append(filterInfo, c.warningStyle.Render("High Priority"))
	}
	if logsState.GetShowPattern() != "" {
		filterInfo = append(filterInfo, c.filterStyle.Render(fmt.Sprintf("Show: %s", logsState.GetShowPattern())))
	}
	if logsState.GetHidePattern() != "" {
		filterInfo = append(filterInfo, c.filterStyle.Render(fmt.Sprintf("Hide: %s", logsState.GetHidePattern())))
	}

	if len(filterInfo) > 0 {
		lines = append(lines, strings.Join(filterInfo, " • "))
		lines = append(lines, strings.Repeat("─", c.width))
	}

	return strings.Join(lines, "\n")
}

// renderContent creates the main content area with logs
func (c *LogsViewComponent) renderContent(logsState *state.LogsState, logStore *logs.Store) string {
	if logsState.GetLoading() {
		return c.loadingStyle.Render("Loading logs...")
	}

	if logsState.GetError() != "" {
		return c.errorStyle.Render(fmt.Sprintf("❌ Error: %s", logsState.GetError()))
	}

	// Get logs from cache or log store
	var logEntries []logs.CollapsedLogEntry
	if logsState.IsCacheValid(logsState.GetSelectedProcess(), 5*time.Second) {
		logEntries = logsState.GetCachedLogs()
	} else {
		// Fetch from log store (this should be done via commands in the model)
		if logStore != nil {
			if logsState.GetSelectedProcess() != "" {
				logEntries = logStore.GetByProcessCollapsed(logsState.GetSelectedProcess())
			} else {
				logEntries = logStore.GetAllCollapsed()
			}
		}
	}

	// Apply filters
	filteredEntries := logsState.ApplyFilters(logEntries)

	// Format for display
	if len(filteredEntries) == 0 {
		// Check if logs were recently cleared (Phase E enhancement)
		if logsState.GetShowClearedMsg() {
			clearedTime := logsState.GetClearedAt()
			timeSinceClear := time.Since(clearedTime)

			// Show cleared message for 3 seconds
			if timeSinceClear < 3*time.Second {
				return c.normalStyle.Render("✅ Logs cleared")
			}
		}

		return c.emptyStyle.Render("No logs to display")
	}

	// If we have logs but were showing cleared message, hide it
	if logsState.GetShowClearedMsg() {
		// In a real implementation, this would be handled by the Update() method
		// when new logs arrive. For now, we'll just hide the message.
	}

	content := c.formatLogsForDisplay(filteredEntries)

	// Create viewport for scrolling
	vp := viewport.New(c.width, c.height-6) // Reserve space for header/footer
	vp.SetContent(content)

	// Apply scroll offset
	if logsState.GetScrollOffset() > 0 {
		vp.GotoTop()
		for i := 0; i < logsState.GetScrollOffset(); i++ {
			vp.LineDown(1)
		}
	} else if logsState.GetAtBottom() {
		vp.GotoBottom()
	}

	return vp.View()
}

// renderFooter creates the footer section with status and controls
func (c *LogsViewComponent) renderFooter(logsState *state.LogsState) string {
	var lines []string

	// Status line
	var status []string
	if logsState.GetAutoScroll() {
		status = append(status, c.infoStyle.Render("🔄 Auto-scroll"))
	} else {
		status = append(status, c.filterStyle.Render("⏸ Paused"))
	}

	if logsState.GetSearchQuery() != "" {
		status = append(status, c.searchStyle.Render(fmt.Sprintf("🔍 %s", logsState.GetSearchQuery())))
	}

	lines = append(lines, strings.Repeat("─", c.width))
	lines = append(lines, strings.Join(status, " • "))

	// Help line
	help := c.filterStyle.Render("↑↓/j/k=Scroll | Space=Toggle Auto-scroll | /=Search | c=Clear")
	lines = append(lines, help)

	return strings.Join(lines, "\n")
}

// formatLogsForDisplay formats collapsed log entries for display
func (c *LogsViewComponent) formatLogsForDisplay(entries []logs.CollapsedLogEntry) string {
	var builder strings.Builder

	for _, entry := range entries {
		ts := c.timestampStyle.Render(entry.Timestamp.Format("15:04:05"))

		var content string
		if entry.IsError {
			content = c.errorStyle.Render(entry.Content)
		} else if entry.Priority > 1 {
			content = c.warningStyle.Render(entry.Content)
		} else {
			content = c.normalStyle.Render(entry.Content)
		}

		processName := c.processStyle.Render(entry.ProcessName)

		if entry.Count > 1 {
			countText := c.countStyle.Render(fmt.Sprintf("[%d×]", entry.Count))
			builder.WriteString(fmt.Sprintf("%s [%s] %s %s\n", ts, processName, content, countText))
		} else {
			builder.WriteString(fmt.Sprintf("%s [%s] %s\n", ts, processName, content))
		}
	}

	return builder.String()
}

// SetSize updates component dimensions
func (c *LogsViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.width = width
	c.height = height
}

// CreateSearchInput creates a text input component for search functionality
func (c *LogsViewComponent) CreateSearchInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "Search logs or use /show <pattern> | /hide <pattern>"
	input.Prompt = c.searchStyle.Render("🔍 ")
	input.CursorStyle = c.searchStyle
	input.TextStyle = c.normalStyle
	return input
}

// ParseSearchCommand parses search/filter commands
func (c *LogsViewComponent) ParseSearchCommand(input string) (searchQuery, showPattern, hidePattern string, isHighPriority bool, isClear bool) {
	input = strings.TrimSpace(input)

	// Clear command
	if input == "c" || input == "clear" {
		isClear = true
		return
	}

	// High priority toggle
	if input == "h" || input == "high" {
		isHighPriority = true
		return
	}

	// Show pattern
	if strings.HasPrefix(input, "/show ") {
		showPattern = strings.TrimSpace(input[6:])
		return
	}

	// Hide pattern
	if strings.HasPrefix(input, "/hide ") {
		hidePattern = strings.TrimSpace(input[6:])
		return
	}

	// Regular search
	if input != "" {
		searchQuery = input
		return
	}

	return
}

// ValidatePattern checks if a regex pattern is valid
func (c *LogsViewComponent) ValidatePattern(pattern string) bool {
	if pattern == "" {
		return true
	}
	_, err := regexp.Compile(pattern)
	return err == nil
}

// GetFilterSummary returns a summary of active filters
func (c *LogsViewComponent) GetFilterSummary(logsState *state.LogsState) string {
	var filters []string

	if logsState.GetShowHighPriority() {
		filters = append(filters, "High Priority")
	}

	if logsState.GetShowPattern() != "" {
		filters = append(filters, fmt.Sprintf("Show: %s", logsState.GetShowPattern()))
	}

	if logsState.GetHidePattern() != "" {
		filters = append(filters, fmt.Sprintf("Hide: %s", logsState.GetHidePattern()))
	}

	if logsState.GetSelectedProcess() != "" {
		filters = append(filters, fmt.Sprintf("Process: %s", logsState.GetSelectedProcess()))
	}

	if len(filters) == 0 {
		return "No filters"
	}

	return strings.Join(filters, ", ")
}