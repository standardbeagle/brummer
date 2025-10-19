package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// ErrorsViewComponent is a stateless BubbleTea component for displaying errors
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type ErrorsViewComponent struct {
	BaseComponent

	// UI configuration only (no state)
	width  int
	height int

	// List model for error list (minimal, stateless)
	list list.Model

	// Styles
	headerStyle      lipgloss.Style
	errorStyle       lipgloss.Style
	processStyle     lipgloss.Style
	timestampStyle   lipgloss.Style
	typeStyle        lipgloss.Style
	emptyStyle       lipgloss.Style
	selectedStyle    lipgloss.Style
	descriptionStyle lipgloss.Style
}

// NewErrorsViewComponent creates a new stateless errors view component
func NewErrorsViewComponent() *ErrorsViewComponent {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("196"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("196"))

	errorsList := list.New([]list.Item{}, delegate, 0, 0)
	errorsList.Title = "Errors"
	errorsList.SetShowStatusBar(false)
	errorsList.SetShowHelp(false)
	errorsList.SetFilteringEnabled(true)
	errorsList.SetShowPagination(false)

	return &ErrorsViewComponent{
		BaseComponent: NewBaseComponent(),
		width:         80,
		height:        24,
		list:          errorsList,

		// Initialize styles
		headerStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		errorStyle:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")),
		processStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		timestampStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		typeStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		emptyStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true).Align(lipgloss.Center, lipgloss.Center),
		selectedStyle:    lipgloss.NewStyle().Background(lipgloss.Color("240")).Bold(true),
		descriptionStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	}
}

// Init initializes the component
func (c *ErrorsViewComponent) Init() tea.Cmd {
	return nil
}

// ViewWithState renders the component using external state (stateless/Elm-compliant)
func (c *ErrorsViewComponent) ViewWithState(masterData *state.MasterData, logStore *logs.Store) string {
	if !c.IsVisible() || masterData == nil || masterData.ErrorsState == nil {
		return ""
	}

	errorsState := masterData.ErrorsState

	// Get errors from cache or log store
	var errorContexts []logs.ErrorContext
	if errorsState.IsCacheValid(5*time.Second) {
		errorContexts = errorsState.GetErrorContexts()
	} else if logStore != nil {
		errorContexts = logStore.GetErrorContexts()
		// Update state with fresh data
		newState := errorsState.UpdateErrorContexts(errorContexts)
		masterData.ErrorsState = newState
	}

	if len(errorContexts) == 0 {
		return c.renderEmptyState()
	}

	// Render based on view mode
	if errorsState.GetShowDetailView() && errorsState.GetSelectedError() != nil {
		return c.renderDetailView(errorsState.GetSelectedError())
	}

	return c.renderListView(errorContexts, errorsState)
}

// renderEmptyState renders the empty state when no errors exist
func (c *ErrorsViewComponent) renderEmptyState() string {
	content := []string{
		"No errors detected.",
		"",
		"System running smoothly! ✓",
	}

	emptyText := strings.Join(content, "\n")
	return c.emptyStyle.Width(c.width).Height(c.height).Render(emptyText)
}

// renderListView renders the error list view
func (c *ErrorsViewComponent) renderListView(errorContexts []logs.ErrorContext, errorsState *state.ErrorsState) string {
	var content strings.Builder

	// Header
	content.WriteString(c.headerStyle.Render("📋 Recent Errors") + "\n")
	content.WriteString(strings.Repeat("─", c.width) + "\n\n")

	// Create list items
	items := c.createErrorListItems(errorContexts, errorsState)

	// Update viewport
	c.list.SetItems(items)
	c.list.SetSize(c.width, c.height-4) // Reserve space for header and footer

	// Render list
	content.WriteString(c.list.View())

	// Footer
	content.WriteString(strings.Repeat("─", c.width) + "\n")
	footer := c.renderFooter(errorsState)
	content.WriteString(footer)

	return content.String()
}

// renderDetailView renders the detailed view of a single error
func (c *ErrorsViewComponent) renderDetailView(selectedError *logs.ErrorContext) string {
	var content strings.Builder

	// Header with navigation
	content.WriteString(c.headerStyle.Render("🔍 Error Details") + "\n")
	content.WriteString(strings.Repeat("─", c.width) + "\n\n")

	// Format the error
	formattedError := c.formatErrorContext(selectedError)
	content.WriteString(formattedError)

	return content.String()
}

// renderFooter creates the footer section
func (c *ErrorsViewComponent) renderFooter(errorsState *state.ErrorsState) string {
	var parts []string

	if errorsState.GetErrorCount() > 0 {
		parts = append(parts, fmt.Sprintf("Total: %d error(s)", errorsState.GetErrorCount()))
	}

	if errorsState.GetAutoRefresh() {
		parts = append(parts, "🔄 Auto-refresh")
	}

	if len(parts) > 0 {
		return strings.Join(parts, " • ") + "\nEnter=Details | c=Clear | f=Filter | r=Refresh"
	}

	return "c=Clear | f=Filter | r=Refresh"
}

// createErrorListItems creates list items from error contexts
func (c *ErrorsViewComponent) createErrorListItems(errorContexts []logs.ErrorContext, errorsState *state.ErrorsState) []list.Item {
	var items []list.Item

	for _, ec := range errorContexts {
		item := ErrorItemStateless{
			ErrorContext: &ec,
			Timestamp:    ec.Timestamp.Format("15:04:05"),
			ProcessName:  ec.ProcessName,
			IsSelected:   ec.ID == errorsState.GetSelectedErrorID(),
		}
		items = append(items, item)
	}

	return items
}

// formatErrorContext formats an error context for display
func (c *ErrorsViewComponent) formatErrorContext(ec *logs.ErrorContext) string {
	var content strings.Builder

	content.WriteString(c.errorStyle.Render(fmt.Sprintf("❌ %s", ec.Type)) + "\n")
	content.WriteString(fmt.Sprintf("%s %s\n\n",
		c.timestampStyle.Render(ec.Timestamp.Format("2006-01-02 15:04:05")),
		c.processStyle.Render(fmt.Sprintf("[%s]", ec.ProcessName))))

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

// SetSize updates component dimensions
func (c *ErrorsViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.width = width
	c.height = height
	c.list.SetSize(width, height-4)
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *ErrorsViewComponent) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.width = width
	c.height = height
	c.list.SetSize(width, contentHeight)
}

// GetErrorStats returns error statistics
func (c *ErrorsViewComponent) GetErrorStats(errorContexts []logs.ErrorContext) (total, critical, warning int) {
	total = len(errorContexts)
	for _, ec := range errorContexts {
		switch ec.Type {
		case "FATAL", "ERROR", "CRITICAL":
			critical++
		case "WARN", "WARNING":
			warning++
		}
	}
	return
}

// ErrorItemStateless represents an error in the list (stateless version)
type ErrorItemStateless struct {
	ErrorContext *logs.ErrorContext
	Timestamp    string
	ProcessName  string
	IsSelected   bool
}

func (i ErrorItemStateless) Title() string {
	if i.ErrorContext == nil {
		return ""
	}
	return fmt.Sprintf("%s [%s] %s", i.Timestamp, i.ProcessName, i.ErrorContext.Type)
}

func (i ErrorItemStateless) Description() string {
	if i.ErrorContext == nil {
		return ""
	}

	message := i.ErrorContext.Message
	if len(message) > 80 {
		message = message[:77] + "..."
	}
	return message
}

func (i ErrorItemStateless) FilterValue() string {
	if i.ErrorContext == nil {
		return ""
	}
	return i.ErrorContext.Message
}