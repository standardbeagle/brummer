package services

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// ClearService is a stateless service for clearing tab content
// It provides clear commands for Errors, Logs, and Web tabs
//
// IMMUTABLE: This service is stateless. It contains no mutable state.
// NO MUTEXES: Thread-safety is achieved through statelessness.
type ClearService struct {
	// No state - this is a stateless service
}

// NewClearService creates a new clear service
func NewClearService() *ClearService {
	return &ClearService{}
}

// ClearErrors returns a command to clear all errors
func (s *ClearService) ClearErrors() tea.Cmd {
	return func() tea.Msg {
		return messages.ClearErrorsMsg{}
	}
}

// ClearLogs returns a command to clear all logs
func (s *ClearService) ClearLogs() tea.Cmd {
	return func() tea.Msg {
		return messages.ClearLogsMsg{}
	}
}

// ClearWeb returns a command to clear all web events
func (s *ClearService) ClearWeb() tea.Cmd {
	return func() tea.Msg {
		return messages.ClearWebMsg{}
	}
}

// ShowClearConfirmation returns a command to show a clear confirmation dialog
func (s *ClearService) ShowClearConfirmation(tab string) tea.Cmd {
	return func() tea.Msg {
		return messages.ShowClearConfirmationMsg{
			Tab: tab,
		}
	}
}

// GetConfig returns the service configuration (empty for this service)
func (s *ClearService) GetConfig() map[string]interface{} {
	// This service has no configuration
	return make(map[string]interface{})
}