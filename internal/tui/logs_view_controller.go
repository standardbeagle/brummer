package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
)

// LogsViewController manages the logs view state and rendering
type LogsViewController struct {
	logsViewport     viewport.Model
	searchInput      textinput.Model
	searchResults    []logs.LogEntry
	showHighPriority bool
	showPattern      string // Regex pattern for /show command
	hidePattern      string // Regex pattern for /hide command
	logsAutoScroll   bool
	logsAtBottom     bool

	// Dependencies injected from parent Model
	logStore        *logs.Store
	selectedProcess string
	width           int
	height          int
	headerHeight    int
	footerHeight    int
}

// NewLogsViewController creates a new logs view controller
func NewLogsViewController(logStore *logs.Store) *LogsViewController {
	searchInput := textinput.New()
	searchInput.Placeholder = "Commands: /show <pattern> | /hide <pattern>"
	searchInput.Focus()

	return &LogsViewController{
		logsViewport:   viewport.New(0, 0),
		searchInput:    searchInput,
		logStore:       logStore,
		logsAutoScroll: true, // Start with auto-scroll enabled
	}
}

// UpdateSize updates the viewport dimensions with pre-calculated content height
func (v *LogsViewController) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	v.width = width
	v.height = height
	v.headerHeight = headerHeight
	v.footerHeight = footerHeight

	v.logsViewport.Width = width
	v.logsViewport.Height = contentHeight
}

// SetSelectedProcess sets the currently selected process for filtering
func (v *LogsViewController) SetSelectedProcess(processID string) {
	v.selectedProcess = processID
}

// SetShowPattern sets the show filter pattern
func (v *LogsViewController) SetShowPattern(pattern string) {
	v.showPattern = pattern
}

// SetHidePattern sets the hide filter pattern
func (v *LogsViewController) SetHidePattern(pattern string) {
	v.hidePattern = pattern
}

// SetShowHighPriority sets whether to show only high priority logs
func (v *LogsViewController) SetShowHighPriority(show bool) {
	v.showHighPriority = show
}

// ToggleHighPriority toggles the high priority filter
func (v *LogsViewController) ToggleHighPriority() {
	v.showHighPriority = !v.showHighPriority
}

// GetShowPatternPtr returns a pointer to the show pattern
func (v *LogsViewController) GetShowPatternPtr() *string {
	return &v.showPattern
}

// GetHidePatternPtr returns a pointer to the hide pattern
func (v *LogsViewController) GetHidePatternPtr() *string {
	return &v.hidePattern
}

// GetSearchResultsPtr returns a pointer to the search results
func (v *LogsViewController) GetSearchResultsPtr() *[]logs.LogEntry {
	return &v.searchResults
}

// GetShowPattern returns the current show pattern
func (v *LogsViewController) GetShowPattern() string {
	return v.showPattern
}

// GetHidePattern returns the current hide pattern
func (v *LogsViewController) GetHidePattern() string {
	return v.hidePattern
}

// GetSearchResults returns the current search results
func (v *LogsViewController) GetSearchResults() []logs.LogEntry {
	return v.searchResults
}

// ToggleAutoScroll toggles auto-scroll behavior
func (v *LogsViewController) ToggleAutoScroll() {
	v.logsAutoScroll = !v.logsAutoScroll
}

// IsAutoScrollEnabled returns whether auto-scroll is enabled
func (v *LogsViewController) IsAutoScrollEnabled() bool {
	return v.logsAutoScroll
}

// GetLogsViewport returns the logs viewport for direct manipulation
func (v *LogsViewController) GetLogsViewport() *viewport.Model {
	return &v.logsViewport
}

// UpdateLogsView refreshes the logs view with current data
func (v *LogsViewController) UpdateLogsView() {
	var collapsedEntries []logs.CollapsedLogEntry

	if v.selectedProcess != "" {
		// Show logs for specific process
		collapsedEntries = v.logStore.GetByProcessCollapsed(v.selectedProcess)
	} else {
		// Show all logs
		collapsedEntries = v.logStore.GetAllCollapsed()
	}

	// Apply filters
	collapsedEntries = v.applyFilters(collapsedEntries)

	// Format for display
	content := v.formatLogsForDisplay(collapsedEntries)

	// Update viewport
	v.logsViewport.SetContent(content)

	// Auto-scroll to bottom if enabled
	if v.logsAutoScroll {
		v.logsViewport.GotoBottom()
		v.logsAtBottom = true
	}
}

// Render renders the logs view
func (v *LogsViewController) Render() string {
	// Just return the viewport directly without any header
	return v.logsViewport.View()
}

// convertToCollapsedEntries - now handled by logStore directly

// areLogsIdentical - now handled by logStore directly

// applyFilters applies show/hide patterns and priority filters
func (v *LogsViewController) applyFilters(entries []logs.CollapsedLogEntry) []logs.CollapsedLogEntry {
	var filtered []logs.CollapsedLogEntry

	for _, entry := range entries {
		// Apply high priority filter
		if v.showHighPriority && entry.LogEntry.Priority <= 50 {
			continue
		}

		// Apply show pattern
		if v.showPattern != "" {
			if matched, _ := regexp.MatchString(v.showPattern, entry.LogEntry.Content); !matched {
				continue
			}
		}

		// Apply hide pattern
		if v.hidePattern != "" {
			if matched, _ := regexp.MatchString(v.hidePattern, entry.LogEntry.Content); matched {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// formatLogsForDisplay formats collapsed log entries for display
func (v *LogsViewController) formatLogsForDisplay(entries []logs.CollapsedLogEntry) string {
	var content strings.Builder

	for _, entry := range entries {
		// Format timestamp
		timestamp := entry.LogEntry.Timestamp.Format("15:04:05")

		// Format process name
		processName := entry.LogEntry.ProcessName
		if len(processName) > 12 {
			processName = processName[:12]
		}

		// Get log style
		logStyle := v.getLogStyle(entry.LogEntry)

		// Format content
		cleanContent := v.cleanLogContent(entry.LogEntry.Content)

		// Add collapse indicator if needed
		var collapseIndicator string
		if entry.IsCollapsed {
			collapseIndicator = fmt.Sprintf(" (×%d)", entry.Count)
		}

		// Build log line
		logLine := fmt.Sprintf("[%s] %s: %s%s",
			timestamp,
			processName,
			cleanContent,
			collapseIndicator,
		)

		content.WriteString(logStyle.Render(logLine) + "\n")
	}

	return content.String()
}

// cleanLogContent cleans log content for display
func (v *LogsViewController) cleanLogContent(content string) string {
	// Keep the original content with ANSI codes
	cleaned := content

	// Handle different line ending styles - ensure proper line endings
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n") // Windows line endings -> Unix
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")   // Lone CR -> newline (for terminal resets)

	// Don't trim or limit - preserve the original formatting
	// The terminal/viewport will handle wrapping and display

	return cleaned
}

// getLogStyle returns the appropriate style for a log entry
func (v *LogsViewController) getLogStyle(log logs.LogEntry) lipgloss.Style {
	base := lipgloss.NewStyle()

	switch log.Level {
	case logs.LevelError, logs.LevelCritical:
		return base.Foreground(lipgloss.Color("196"))
	case logs.LevelWarn:
		return base.Foreground(lipgloss.Color("214"))
	case logs.LevelDebug:
		return base.Foreground(lipgloss.Color("245"))
	default:
		if log.Priority > 50 {
			return base.Bold(true)
		}
		return base
	}
}
