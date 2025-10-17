package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/commands"
)

// ErrorsView is a BubbleTea component for displaying errors
// Migrated from errors_view_controller.go (T050-T054)
// Architecture: Component → Commands → Service (async error operations)
type ErrorsView struct {
	errorsViewport  viewport.Model
	errorDetailView viewport.Model
	errorsList      list.Model
	selectedError   *logs.ErrorContext
	lastErrorCount  int

	// Service access (now through commands)
	logStore *logs.Store
	width    int
	height   int

	// Command pattern support
	commandID     string
	loading       bool
	lastError     error
	errorContexts []logs.ErrorContext // Cache for error contexts
}

// ErrorItem implements list.Item for error contexts
type ErrorItem struct {
	ErrorContext *logs.ErrorContext
	Timestamp    string
	ProcessName  string
}

func (i ErrorItem) FilterValue() string {
	return i.ErrorContext.Message
}

func (i ErrorItem) Title() string {
	return fmt.Sprintf("%s [%s] %s", i.Timestamp, i.ProcessName, i.ErrorContext.Type)
}

func (i ErrorItem) Description() string {
	message := i.ErrorContext.Message
	if len(message) > 100 {
		message = message[:97] + "..."
	}
	return message
}

// NewErrorsView creates a new errors view component
func NewErrorsView(logStore *logs.Store) *ErrorsView {
	errorsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	errorsList.Title = "Errors"
	errorsList.SetShowStatusBar(false)

	return &ErrorsView{
		errorsViewport:  viewport.New(0, 0),
		errorDetailView: viewport.New(0, 0),
		errorsList:      errorsList,
		logStore:        logStore,
	}
}

// Init initializes the component (BubbleTea interface)
func (v *ErrorsView) Init() tea.Cmd {
	return nil
}

// Update handles messages (BubbleTea interface)
func (v *ErrorsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	v.errorsList, cmd = v.errorsList.Update(msg)
	return v, cmd
}

// View renders the component (BubbleTea interface)
func (v *ErrorsView) View() string {
	return v.Render()
}

// SetSize updates the component's dimensions (Component interface)
func (v *ErrorsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	contentHeight := height - 6
	v.errorsViewport.Width = width
	v.errorsViewport.Height = contentHeight
	v.errorDetailView.Width = width
	v.errorDetailView.Height = contentHeight
	v.errorsList.SetSize(width, contentHeight)
}

// Focus gives keyboard focus to the component (Component interface)
func (v *ErrorsView) Focus() tea.Cmd {
	return nil
}

// Blur removes keyboard focus from the component (Component interface)
func (v *ErrorsView) Blur() tea.Cmd {
	return nil
}

// GetSize returns current dimensions (Component interface)
func (v *ErrorsView) GetSize() (width, height int) {
	return v.width, v.height
}

// UpdateSize updates dimensions with pre-calculated content height
func (v *ErrorsView) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	v.width = width
	v.height = height
	v.errorsViewport.Width = width
	v.errorsViewport.Height = contentHeight
	v.errorDetailView.Width = width
	v.errorDetailView.Height = contentHeight
	v.errorsList.SetSize(width, contentHeight)
}

// GetErrorsViewport returns the errors viewport for direct manipulation
func (v *ErrorsView) GetErrorsViewport() *viewport.Model {
	return &v.errorsViewport
}

// UpdateErrorsView refreshes the errors view with current data
func (v *ErrorsView) UpdateErrorsView() {
	// Direct service call - no controller indirection (T052)
	errorContexts := v.logStore.GetErrorContexts()

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent Errors") + "\n\n")

	if len(errorContexts) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true).
			Render("No errors detected. System running smoothly! ✓"))
	} else {
		for i, ec := range errorContexts {
			if i > 0 {
				content.WriteString("\n" + strings.Repeat("─", v.width) + "\n")
			}
			content.WriteString(v.formatErrorContext(&ec))
		}
	}

	v.errorsViewport.SetContent(content.String())
}

// UpdateErrorsList updates the list of errors
func (v *ErrorsView) UpdateErrorsList() {
	// Use cached error contexts
	errorContexts := v.GetErrorContexts()

	var items []list.Item
	for _, ec := range errorContexts {
		ecCopy := ec
		items = append(items, ErrorItem{
			ErrorContext: &ecCopy,
			Timestamp:    ec.Timestamp.Format("15:04:05"),
			ProcessName:  ec.ProcessName,
		})
	}

	v.errorsList.SetItems(items)
}

// Render renders the errors view
func (v *ErrorsView) Render() string {
	return v.errorsViewport.View()
}

// GetErrorsList returns the errors list for direct manipulation
func (v *ErrorsView) GetErrorsList() *list.Model {
	return &v.errorsList
}

// SetSelectedError sets the currently selected error
func (v *ErrorsView) SetSelectedError(ec *logs.ErrorContext) {
	v.selectedError = ec
	if ec != nil {
		content := v.formatErrorContext(ec)
		v.errorDetailView.SetContent(content)
	}
}

// GetSelectedError returns the currently selected error
func (v *ErrorsView) GetSelectedError() *logs.ErrorContext {
	return v.selectedError
}

// formatErrorContext formats an error context for display
func (v *ErrorsView) formatErrorContext(ec *logs.ErrorContext) string {
	var content strings.Builder

	errorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	_ = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // typeStyle

	content.WriteString(errorStyle.Render(fmt.Sprintf("❌ %s", ec.Type)) + "\n")
	content.WriteString(fmt.Sprintf("%s %s\n\n",
		timestampStyle.Render(ec.Timestamp.Format("2006-01-02 15:04:05")),
		processStyle.Render(fmt.Sprintf("[%s]", ec.ProcessName))))

	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Error Message:") + "\n")
	content.WriteString(ec.Message + "\n\n")

	if len(ec.Stack) > 0 {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Stack Trace:") + "\n")
		for _, line := range ec.Stack {
			content.WriteString("  " + line + "\n")
		}
		content.WriteString("\n")
	}

	if len(ec.Context) > 0 {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Context:") + "\n")
		for _, line := range ec.Context {
			if strings.TrimSpace(line) != "" {
				content.WriteString("  " + line + "\n")
			}
		}
	}

	return content.String()
}

// ClearError clears a specific error by ID
func (v *ErrorsView) ClearError(errorID string) {
	// v.logStore.ClearError(errorID) // Method not available in logs.Store
	v.UpdateErrorsList()
	v.UpdateErrorsView()
}

// ClearAllErrors clears all errors
func (v *ErrorsView) ClearAllErrors() {
	v.logStore.ClearErrors()
	v.selectedError = nil
	v.UpdateErrorsList()
	v.UpdateErrorsView()
}

// GetErrorDetailView returns the error detail viewport for direct manipulation
func (v *ErrorsView) GetErrorDetailView() *viewport.Model {
	return &v.errorDetailView
}

// UpdateErrorDetailView updates the detail view for selected error
func (v *ErrorsView) UpdateErrorDetailView() {
	if v.selectedError != nil {
		content := v.formatErrorContext(v.selectedError)
		v.errorDetailView.SetContent(content)
	}
}

// HandleCopyError handles copying error to clipboard (returns command)
func (v *ErrorsView) HandleCopyError() tea.Cmd {
	return nil // Placeholder for clipboard functionality
}

// HandleClearErrors clears all errors
func (v *ErrorsView) HandleClearErrors() {
	v.ClearAllErrors()
}

// UpdateErrorsCommand returns a command to refresh the errors view asynchronously
func (v *ErrorsView) UpdateErrorsCommand() tea.Cmd {
	if v.logStore == nil {
		return nil
	}

	// Generate unique command ID
	v.commandID = fmt.Sprintf("errors-update-%d", time.Now().UnixNano())

	context := commands.UIOperationContext(0) // Version will be set by the model

	return func() tea.Msg {
		cmd := commands.NewGetErrorsCommand(v.commandID, v.logStore, context)
		return cmd.Run()
	}
}

// UpdateErrorsViewWithCache refreshes the errors view using cached data
func (v *ErrorsView) UpdateErrorsViewWithCache() {
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent Errors") + "\n\n")

	if len(v.errorContexts) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true).
			Render("No errors detected. System running smoothly! ✓"))
	} else {
		for i, ec := range v.errorContexts {
			if i > 0 {
				content.WriteString("\n" + strings.Repeat("─", v.width) + "\n")
			}
			content.WriteString(v.formatErrorContext(&ec))
		}
	}

	v.errorsViewport.SetContent(content.String())
}

// UpdateErrorsListWithCache updates the list of errors using cached data
func (v *ErrorsView) UpdateErrorsListWithCache() {
	var items []list.Item
	for _, ec := range v.errorContexts {
		ecCopy := ec
		items = append(items, ErrorItem{
			ErrorContext: &ecCopy,
			Timestamp:    ec.Timestamp.Format("15:04:05"),
			ProcessName:  ec.ProcessName,
		})
	}

	v.errorsList.SetItems(items)
}

// HandleErrorResult handles the result of an error command
func (v *ErrorsView) HandleErrorResult(msg commands.LogResultMsg) {
	if msg.ID != v.commandID {
		return // Not our command
	}

	v.loading = false
	v.lastError = msg.Error

	if msg.IsSuccess() {
		// Process the result
		if errorData, ok := msg.Result.(commands.ErrorData); ok {
			// Convert log entries to error contexts
			v.errorContexts = v.convertLogEntriesToErrorContexts(errorData.Entries)
		}
	}
}

// GetErrorContexts returns the cached error contexts (falls back to sync method if cache is empty)
func (v *ErrorsView) GetErrorContexts() []logs.ErrorContext {
	if v.errorContexts == nil && v.logStore != nil {
		// Fallback to sync method
		v.errorContexts = v.logStore.GetErrorContexts()
	}
	return v.errorContexts
}

// convertLogEntriesToErrorContexts converts log entries to error contexts for compatibility
func (v *ErrorsView) convertLogEntriesToErrorContexts(entries []logs.LogEntry) []logs.ErrorContext {
	// This is a simplified conversion - in a full implementation,
	// we would use the log store's error grouping functionality
	return v.logStore.GetErrorContexts()
}

// IsLoading returns whether the component is currently loading data
func (v *ErrorsView) IsLoading() bool {
	return v.loading
}

// SetLoading sets the loading state of the component
func (v *ErrorsView) SetLoading(loading bool) {
	v.loading = loading
}

// GetLastError returns the last error encountered
func (v *ErrorsView) GetLastError() error {
	return v.lastError
}

// ClearCommandError clears the last command error
func (v *ErrorsView) ClearCommandError() {
	v.lastError = nil
}

// ClearCache clears the cached error contexts
func (v *ErrorsView) ClearCache() {
	v.errorContexts = nil
}
