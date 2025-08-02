package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
)

// ErrorsViewController manages the errors view state and rendering
type ErrorsViewController struct {
	errorsViewport viewport.Model
	
	// Dependencies injected from parent Model
	logStore       *logs.Store
	width          int
	height         int
	headerHeight   int
	footerHeight   int
}

// NewErrorsViewController creates a new errors view controller
func NewErrorsViewController(logStore *logs.Store) *ErrorsViewController {
	return &ErrorsViewController{
		errorsViewport: viewport.New(0, 0),
		logStore:       logStore,
	}
}

// UpdateSize updates the viewport dimensions
func (v *ErrorsViewController) UpdateSize(width, height, headerHeight, footerHeight int) {
	v.width = width
	v.height = height
	v.headerHeight = headerHeight
	v.footerHeight = footerHeight
	
	v.errorsViewport.Width = width
	v.errorsViewport.Height = height - headerHeight - footerHeight
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