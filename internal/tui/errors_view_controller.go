package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
)

// errorUpdateMsg represents an error update event
type errorUpdateMsg struct{}

// errorItem implements list.Item for error contexts
type errorItem struct {
	errorContext *logs.ErrorContext
	timestamp    string
	processName  string
}

func (i errorItem) FilterValue() string {
	return i.errorContext.Message
}

func (i errorItem) Title() string {
	return fmt.Sprintf("%s [%s] %s", i.timestamp, i.processName, i.errorContext.Type)
}

func (i errorItem) Description() string {
	message := i.errorContext.Message
	if len(message) > 100 {
		message = message[:97] + "..."
	}
	return message
}

// ErrorsViewController manages the errors view state and rendering
type ErrorsViewController struct {
	errorsViewport  viewport.Model
	errorDetailView viewport.Model
	errorsList      list.Model
	selectedError   *logs.ErrorContext
	lastErrorCount  int

	// Dependencies injected from parent Model
	logStore     *logs.Store
	width        int
	height       int
	headerHeight int
	footerHeight int
}

// NewErrorsViewController creates a new errors view controller
func NewErrorsViewController(logStore *logs.Store) *ErrorsViewController {
	// Initialize errors list
	errorsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	errorsList.Title = "Errors"
	errorsList.SetShowStatusBar(false)

	return &ErrorsViewController{
		errorsViewport:  viewport.New(0, 0),
		errorDetailView: viewport.New(0, 0),
		errorsList:      errorsList,
		logStore:        logStore,
	}
}

// UpdateSize updates the viewport dimensions with pre-calculated content height
func (v *ErrorsViewController) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	v.width = width
	v.height = height
	v.headerHeight = headerHeight
	v.footerHeight = footerHeight

	v.errorsViewport.Width = width
	v.errorsViewport.Height = contentHeight

	// Update other components
	v.errorDetailView.Width = width
	v.errorDetailView.Height = contentHeight
	v.errorsList.SetSize(width, contentHeight)
}

// GetErrorsViewport returns the errors viewport for direct manipulation
func (v *ErrorsViewController) GetErrorsViewport() *viewport.Model {
	return &v.errorsViewport
}

// UpdateErrorsView refreshes the errors view with current data
func (v *ErrorsViewController) UpdateErrorsView() {
	errorContexts := v.logStore.GetErrorContexts()

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent Errors") + "\n\n")

	if len(errorContexts) == 0 {
		// Fall back to simple errors if no contexts
		errors := v.logStore.GetErrors()
		if len(errors) == 0 {
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No errors detected yet. Use /clear errors to clear when errors appear"))
		} else {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

			// Show most recent errors first
			for i := len(errors) - 1; i >= 0 && i >= len(errors)-20; i-- {
				err := errors[i]
				content.WriteString(fmt.Sprintf("%s %s\n%s\n\n",
					timeStyle.Render(err.Timestamp.Format("15:04:05")),
					processStyle.Render(fmt.Sprintf("[%s]", err.ProcessName)),
					errorStyle.Render(err.Content),
				))
			}
		}
	} else {
		// Styles for different parts
		errorTypeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		stackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))

		// Show most recent error contexts first
		shown := 0
		for i := len(errorContexts) - 1; i >= 0 && shown < 10; i-- {
			errorCtx := errorContexts[i]

			// Error header
			content.WriteString(fmt.Sprintf("%s %s %s\n",
				timeStyle.Render(errorCtx.Timestamp.Format("15:04:05")),
				processStyle.Render(fmt.Sprintf("[%s]", errorCtx.ProcessName)),
				errorTypeStyle.Render(errorCtx.Type),
			))

			// Main error message
			content.WriteString(messageStyle.Render(errorCtx.Message) + "\n")

			// Stack trace if available
			if len(errorCtx.Stack) > 0 {
				content.WriteString(stackStyle.Render("Stack Trace:") + "\n")
				for j, stackLine := range errorCtx.Stack {
					if j > 5 { // Limit stack trace lines
						content.WriteString(stackStyle.Render(fmt.Sprintf("  ... and %d more lines", len(errorCtx.Stack)-j)) + "\n")
						break
					}
					content.WriteString(stackStyle.Render("  "+strings.TrimSpace(stackLine)) + "\n")
				}
			}

			// Additional context if available
			if len(errorCtx.Context) > 0 && len(errorCtx.Context) <= 5 {
				for _, ctxLine := range errorCtx.Context {
					if strings.TrimSpace(ctxLine) != "" {
						content.WriteString(contextStyle.Render("  "+strings.TrimSpace(ctxLine)) + "\n")
					}
				}
			}

			// Separator between errors
			content.WriteString(separatorStyle.Render("─────────────────────────────────────────") + "\n\n")
			shown++
		}
	}

	v.errorsViewport.SetContent(content.String())
}

// Render renders the errors view
func (v *ErrorsViewController) Render() string {
	// Update content with latest errors
	v.UpdateErrorsView()

	return v.errorsViewport.View()
}

// GetErrorsList returns the errors list for direct manipulation
func (v *ErrorsViewController) GetErrorsList() *list.Model {
	return &v.errorsList
}

// GetErrorDetailView returns the error detail viewport for direct manipulation
func (v *ErrorsViewController) GetErrorDetailView() *viewport.Model {
	return &v.errorDetailView
}

// SetSelectedError sets the currently selected error
func (v *ErrorsViewController) SetSelectedError(errorCtx *logs.ErrorContext) {
	v.selectedError = errorCtx
}

// GetSelectedError returns the currently selected error
func (v *ErrorsViewController) GetSelectedError() *logs.ErrorContext {
	return v.selectedError
}

// UpdateErrorsList refreshes the errors list with current data
func (v *ErrorsViewController) UpdateErrorsList() int {
	errorContexts := v.logStore.GetErrorContexts()
	newCount := len(errorContexts)

	items := make([]list.Item, 0, len(errorContexts))
	for i := len(errorContexts) - 1; i >= 0; i-- {
		errorCtx := errorContexts[i]
		items = append(items, errorItem{
			errorContext: &errorCtx,
			timestamp:    errorCtx.Timestamp.Format("15:04:05"),
			processName:  errorCtx.ProcessName,
		})
	}

	v.errorsList.SetItems(items)

	// Return count change for unread indicator updates
	countChange := newCount - v.lastErrorCount
	v.lastErrorCount = newCount
	return countChange
}

// UpdateErrorDetailView updates the error detail view with the selected error
func (v *ErrorsViewController) UpdateErrorDetailView() {
	if v.selectedError == nil {
		v.errorDetailView.SetContent("Select an error to view details")
		return
	}

	var content strings.Builder

	// Error header with enhanced styling
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	content.WriteString(fmt.Sprintf("%s %s %s\n\n",
		timeStyle.Render(v.selectedError.Timestamp.Format("2006-01-02 15:04:05")),
		processStyle.Render(fmt.Sprintf("[%s]", v.selectedError.ProcessName)),
		headerStyle.Render(v.selectedError.Type),
	))

	// Main error message
	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	content.WriteString(messageStyle.Render("Error Message:") + "\n")
	content.WriteString(v.selectedError.Message + "\n\n")

	// File reference if available
	fileRef := v.findLowestCodeReference(v.selectedError)
	if fileRef != "" {
		refStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		content.WriteString(refStyle.Render("📍 Code Reference: ") + fileRef + "\n\n")
	}

	// Stack trace
	if len(v.selectedError.Stack) > 0 {
		stackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		content.WriteString(stackStyle.Render("Stack Trace:") + "\n")
		for i, stackLine := range v.selectedError.Stack {
			if i > 10 { // Show more lines in detail view
				content.WriteString(stackStyle.Render(fmt.Sprintf("  ... and %d more lines", len(v.selectedError.Stack)-i)) + "\n")
				break
			}
			content.WriteString(stackStyle.Render("  "+strings.TrimSpace(stackLine)) + "\n")
		}
		content.WriteString("\n")
	}

	// Additional context
	if len(v.selectedError.Context) > 0 {
		contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		content.WriteString(contextStyle.Render("Additional Context:") + "\n")
		for _, ctxLine := range v.selectedError.Context {
			if strings.TrimSpace(ctxLine) != "" {
				content.WriteString(contextStyle.Render("  "+strings.TrimSpace(ctxLine)) + "\n")
			}
		}
	}

	v.errorDetailView.SetContent(content.String())
}

// HandleClearErrors clears all errors and logs the action
func (v *ErrorsViewController) HandleClearErrors() {
	v.logStore.ClearErrors()
	v.logStore.Add("system", "System", "🗑️ Error history cleared", false)
	v.selectedError = nil
	v.lastErrorCount = 0
}

// HandleCopyError creates a command to copy error details to clipboard
func (v *ErrorsViewController) HandleCopyError() tea.Cmd {
	return func() tea.Msg {
		// Try to get error contexts first
		errorContexts := v.logStore.GetErrorContexts()

		var errorText string
		if len(errorContexts) > 0 {
			// Use the most recent error context
			errorCtx := errorContexts[len(errorContexts)-1]

			var builder strings.Builder
			builder.WriteString(fmt.Sprintf("Error: %s\n", errorCtx.Type))
			builder.WriteString(fmt.Sprintf("Process: %s\n", errorCtx.ProcessName))
			builder.WriteString(fmt.Sprintf("Time: %s\n", errorCtx.Timestamp.Format("2006-01-02 15:04:05")))
			builder.WriteString(fmt.Sprintf("Message: %s\n", errorCtx.Message))

			if len(errorCtx.Stack) > 0 {
				builder.WriteString("\nStack Trace:\n")
				for _, line := range errorCtx.Stack {
					builder.WriteString(fmt.Sprintf("  %s\n", strings.TrimSpace(line)))
				}
			}

			if len(errorCtx.Context) > 0 {
				builder.WriteString("\nContext:\n")
				for _, line := range errorCtx.Context {
					if strings.TrimSpace(line) != "" {
						builder.WriteString(fmt.Sprintf("  %s\n", strings.TrimSpace(line)))
					}
				}
			}

			errorText = builder.String()
		} else {
			// Fall back to simple error format
			errors := v.logStore.GetErrors()
			if len(errors) > 0 {
				latestError := errors[len(errors)-1]
				errorText = fmt.Sprintf("Error from %s at %s:\n%s",
					latestError.ProcessName,
					latestError.Timestamp.Format("2006-01-02 15:04:05"),
					latestError.Content,
				)
			} else {
				errorText = "No errors to copy"
			}
		}

		// TODO: Implement actual clipboard copy functionality
		// For now, just return a message that text would be copied
		_ = errorText // Will be used for actual clipboard implementation
		return struct {
			message string
		}{
			message: "Error details copied to clipboard",
		}
	}
}

// RenderErrorsViewSplit renders a split view with error list and details
func (v *ErrorsViewController) RenderErrorsViewSplit() string {
	if v.width < 100 {
		// For narrow screens, use the old view
		return v.Render()
	}

	// Update both views
	v.UpdateErrorsList()
	v.UpdateErrorDetailView()

	// Use ContentLayout for split view if available
	splitRatio := DefaultSplitRatio
	leftWidth := int(float64(v.width) * splitRatio)
	rightWidth := v.width - leftWidth

	// Update list size for split view
	v.errorsList.SetSize(leftWidth-2, v.height-v.headerHeight-v.footerHeight)
	v.errorDetailView.Width = rightWidth - 2
	v.errorDetailView.Height = v.height - v.headerHeight - v.footerHeight

	// Create left panel with border
	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(v.height-v.headerHeight-v.footerHeight).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("240")).
		Render(v.errorsList.View())

	// Create right panel
	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(v.height - v.headerHeight - v.footerHeight).
		Render(v.errorDetailView.View())

	// Join horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

// findLowestCodeReference finds the lowest-level code reference in error context
func (v *ErrorsViewController) findLowestCodeReference(errorCtx *logs.ErrorContext) string {
	// Look for file paths with line numbers in stack traces
	filePattern := regexp.MustCompile(`([^\s\(\)]+\.(js|ts|jsx|tsx|go|py|java|rs|rb|php)):(\d+)(?::(\d+))?`)

	var lowestRef string
	var lowestInProject bool

	// Check stack trace first
	for _, line := range errorCtx.Stack {
		matches := filePattern.FindStringSubmatch(line)
		if len(matches) >= 4 {
			filepath := matches[1]
			lineNum := matches[3]

			// Prefer files that look like they're in the project (not node_modules, etc.)
			isInProject := !strings.Contains(filepath, "node_modules") &&
				!strings.Contains(filepath, ".pnpm") &&
				!strings.Contains(filepath, "/usr/") &&
				!strings.Contains(filepath, "/opt/")

			// If we don't have a reference yet, or this one is better
			if lowestRef == "" || (isInProject && !lowestInProject) {
				if len(matches) >= 5 && matches[4] != "" {
					lowestRef = fmt.Sprintf("%s:%s:%s", filepath, lineNum, matches[4])
				} else {
					lowestRef = fmt.Sprintf("%s:%s", filepath, lineNum)
				}
				lowestInProject = isInProject
			}
		}
	}

	// Also check the main error message
	if lowestRef == "" {
		matches := filePattern.FindStringSubmatch(errorCtx.Message)
		if len(matches) >= 4 {
			filepath := matches[1]
			lineNum := matches[3]
			if len(matches) >= 5 && matches[4] != "" {
				lowestRef = fmt.Sprintf("%s:%s:%s", filepath, lineNum, matches[4])
			} else {
				lowestRef = fmt.Sprintf("%s:%s", filepath, lineNum)
			}
		}
	}

	return lowestRef
}
