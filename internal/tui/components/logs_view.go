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
	"github.com/standardbeagle/brummer/internal/tui/commands"
)

// LogsView is a BubbleTea component for displaying process logs
// Migrated from logs_view_controller.go (T044-T048)
// Architecture: Component → Commands → Service (async log operations)
type LogsView struct {
	logsViewport     viewport.Model
	searchInput      textinput.Model
	searchResults    []logs.LogEntry
	showHighPriority bool
	showPattern      string
	hidePattern      string
	logsAutoScroll   bool
	logsAtBottom     bool

	// Service access (now through commands)
	logStore        *logs.Store
	selectedProcess string
	width           int
	height          int

	// Command pattern support
	commandID       string
	loading         bool
	lastError       error
}

// NewLogsView creates a new logs view component
func NewLogsView(logStore *logs.Store) *LogsView {
	searchInput := textinput.New()
	searchInput.Placeholder = "Commands: /show <pattern> | /hide <pattern>"
	searchInput.Focus()

	return &LogsView{
		logsViewport:   viewport.New(0, 0),
		searchInput:    searchInput,
		logStore:       logStore,
		logsAutoScroll: true,
	}
}

// Init initializes the component (BubbleTea interface)
func (v *LogsView) Init() tea.Cmd {
	return nil
}

// Update handles messages (BubbleTea interface)
func (v *LogsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	v.logsViewport, cmd = v.logsViewport.Update(msg)
	return v, cmd
}

// View renders the component (BubbleTea interface)
func (v *LogsView) View() string {
	return v.Render()
}

// SetSize updates the component's dimensions (Component interface)
func (v *LogsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	contentHeight := height - 6
	v.logsViewport.Width = width
	v.logsViewport.Height = contentHeight
}

// Focus gives keyboard focus to the component (Component interface)
func (v *LogsView) Focus() tea.Cmd {
	return nil
}

// Blur removes keyboard focus from the component (Component interface)
func (v *LogsView) Blur() tea.Cmd {
	return nil
}

// GetSize returns current dimensions (Component interface)
func (v *LogsView) GetSize() (width, height int) {
	return v.width, v.height
}

// UpdateSize updates the dimensions with pre-calculated content height
func (v *LogsView) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	v.width = width
	v.height = height
	v.logsViewport.Width = width
	v.logsViewport.Height = contentHeight
}

// SetSelectedProcess sets the currently selected process for filtering
func (v *LogsView) SetSelectedProcess(processID string) {
	v.selectedProcess = processID
}

// SetShowPattern sets the show filter pattern
func (v *LogsView) SetShowPattern(pattern string) {
	v.showPattern = pattern
}

// SetHidePattern sets the hide filter pattern
func (v *LogsView) SetHidePattern(pattern string) {
	v.hidePattern = pattern
}

// SetShowHighPriority sets whether to show only high priority logs
func (v *LogsView) SetShowHighPriority(show bool) {
	v.showHighPriority = show
}

// ToggleHighPriority toggles the high priority filter
func (v *LogsView) ToggleHighPriority() {
	v.showHighPriority = !v.showHighPriority
}

// GetShowPatternPtr returns a pointer to the show pattern
func (v *LogsView) GetShowPatternPtr() *string {
	return &v.showPattern
}

// GetHidePatternPtr returns a pointer to the hide pattern
func (v *LogsView) GetHidePatternPtr() *string {
	return &v.hidePattern
}

// GetSearchResultsPtr returns a pointer to the search results
func (v *LogsView) GetSearchResultsPtr() *[]logs.LogEntry {
	return &v.searchResults
}

// GetShowPattern returns the current show pattern
func (v *LogsView) GetShowPattern() string {
	return v.showPattern
}

// GetHidePattern returns the current hide pattern
func (v *LogsView) GetHidePattern() string {
	return v.hidePattern
}

// GetSearchResults returns the current search results
func (v *LogsView) GetSearchResults() []logs.LogEntry {
	return v.searchResults
}

// ToggleAutoScroll toggles auto-scroll behavior
func (v *LogsView) ToggleAutoScroll() {
	v.logsAutoScroll = !v.logsAutoScroll
}

// IsAutoScrollEnabled returns whether auto-scroll is enabled
func (v *LogsView) IsAutoScrollEnabled() bool {
	return v.logsAutoScroll
}

// GetLogsViewport returns the logs viewport for direct manipulation
func (v *LogsView) GetLogsViewport() *viewport.Model {
	return &v.logsViewport
}

// UpdateLogsViewCommand returns a command to refresh the logs view asynchronously
func (v *LogsView) UpdateLogsViewCommand() tea.Cmd {
	if v.logStore == nil {
		return nil
	}

	// Generate unique command ID
	v.commandID = fmt.Sprintf("logs-update-%d", time.Now().UnixNano())

	context := commands.UIOperationContext(0) // Version will be set by the model

	if v.selectedProcess != "" {
		return func() tea.Msg {
			cmd := commands.NewGetProcessLogsCommand(v.commandID, v.logStore, v.selectedProcess, context)
			return cmd.Run()
		}
	} else {
		return func() tea.Msg {
			cmd := commands.NewGetAllLogsCommand(v.commandID, v.logStore, context)
			return cmd.Run()
		}
	}
}

// UpdateLogsView refreshes the logs view with current data (legacy sync method)
func (v *LogsView) UpdateLogsView() {
	var collapsedEntries []logs.CollapsedLogEntry

	// Direct service call - no controller indirection (T046)
	if v.selectedProcess != "" {
		collapsedEntries = v.logStore.GetByProcessCollapsed(v.selectedProcess)
	} else {
		collapsedEntries = v.logStore.GetAllCollapsed()
	}

	collapsedEntries = v.applyFilters(collapsedEntries)
	content := v.formatLogsForDisplay(collapsedEntries)
	v.logsViewport.SetContent(content)

	if v.logsAutoScroll {
		v.logsViewport.GotoBottom()
	}
}

// HandleLogResult handles the result of a log command
func (v *LogsView) HandleLogResult(msg commands.LogResultMsg) {
	if msg.ID != v.commandID {
		return // Not our command
	}

	v.loading = false
	v.lastError = msg.Error

	if msg.IsSuccess() {
		// Process the result
		if logData, ok := msg.Result.(commands.LogData); ok {
			collapsedEntries := v.applyFilters(logData.Entries)
			content := v.formatLogsForDisplay(collapsedEntries)
			v.logsViewport.SetContent(content)

			if v.logsAutoScroll {
				v.logsViewport.GotoBottom()
			}
		}
	}
}

// SearchLogsCommand returns a command to search logs asynchronously
func (v *LogsView) SearchLogsCommand(query string) tea.Cmd {
	if v.logStore == nil || query == "" {
		return nil
	}

	v.commandID = fmt.Sprintf("logs-search-%d", time.Now().UnixNano())

	context := commands.UIOperationContext(0)

	return func() tea.Msg {
		cmd := commands.NewSearchLogsCommand(v.commandID, v.logStore, query, context)
		return cmd.Run()
	}
}

// GetHighPriorityLogsCommand returns a command to get high priority logs asynchronously
func (v *LogsView) GetHighPriorityLogsCommand(threshold int) tea.Cmd {
	if v.logStore == nil {
		return nil
	}

	v.commandID = fmt.Sprintf("logs-highprio-%d", time.Now().UnixNano())

	context := commands.UIOperationContext(0)

	return func() tea.Msg {
		cmd := commands.NewGetHighPriorityLogsCommand(v.commandID, v.logStore, threshold, context)
		return cmd.Run()
	}
}

// ClearLogsCommand returns a command to clear logs asynchronously
func (v *LogsView) ClearLogsCommand(processName string) tea.Cmd {
	if v.logStore == nil {
		return nil
	}

	v.commandID = fmt.Sprintf("logs-clear-%d", time.Now().UnixNano())

	context := commands.UIOperationContext(0)

	return func() tea.Msg {
		cmd := commands.NewClearLogsCommand(v.commandID, v.logStore, processName, context)
		return cmd.Run()
	}
}

// IsLoading returns whether the component is currently loading data
func (v *LogsView) IsLoading() bool {
	return v.loading
}

// SetLoading sets the loading state of the component
func (v *LogsView) SetLoading(loading bool) {
	v.loading = loading
}

// GetLastError returns the last error encountered
func (v *LogsView) GetLastError() error {
	return v.lastError
}

// ClearError clears the last error
func (v *LogsView) ClearError() {
	v.lastError = nil
}

// Render renders the logs view
func (v *LogsView) Render() string {
	return v.logsViewport.View()
}

// applyFilters applies show/hide patterns to logs
func (v *LogsView) applyFilters(entries []logs.CollapsedLogEntry) []logs.CollapsedLogEntry {
	if v.showPattern == "" && v.hidePattern == "" && !v.showHighPriority {
		return entries
	}

	var filtered []logs.CollapsedLogEntry
	var showRe, hideRe *regexp.Regexp
	var err error

	if v.showPattern != "" {
		showRe, err = regexp.Compile(v.showPattern)
		if err != nil {
			showRe = nil
		}
	}

	if v.hidePattern != "" {
		hideRe, err = regexp.Compile(v.hidePattern)
		if err != nil {
			hideRe = nil
		}
	}

	for _, entry := range entries {
		if v.showHighPriority && entry.Priority == 0 {
			continue
		}

		if showRe != nil && !showRe.MatchString(entry.Content) {
			continue
		}

		if hideRe != nil && hideRe.MatchString(entry.Content) {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// formatLogsForDisplay formats collapsed log entries for display
func (v *LogsView) formatLogsForDisplay(entries []logs.CollapsedLogEntry) string {
	var builder strings.Builder

	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)

	for _, entry := range entries {
		ts := timestampStyle.Render(entry.Timestamp.Format("15:04:05"))

		var content string
		if entry.IsError {
			content = errorStyle.Render(entry.Content)
		} else {
			content = normalStyle.Render(entry.Content)
		}

		processName := processStyle.Render(entry.ProcessName)

		if entry.Count > 1 {
			countText := countStyle.Render(fmt.Sprintf(" [%d times]", entry.Count))
			builder.WriteString(fmt.Sprintf("%s [%s] %s%s\n", ts, processName, content, countText))
		} else {
			builder.WriteString(fmt.Sprintf("%s [%s] %s\n", ts, processName, content))
		}
	}

	if builder.Len() == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true).
			Align(lipgloss.Center, lipgloss.Center)
		return emptyStyle.Render("No logs to display")
	}

	return builder.String()
}
