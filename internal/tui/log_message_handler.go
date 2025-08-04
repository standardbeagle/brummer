package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// LogMessageHandler handles log-related messages
type LogMessageHandler struct{}

// NewLogMessageHandler creates a new log message handler
func NewLogMessageHandler() MessageHandler {
	return &LogMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *LogMessageHandler) CanHandle(msg tea.Msg) bool {
	switch msg.(type) {
	case logUpdateMsg, logsClearedMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes log-related messages
func (h *LogMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.(type) {
	case logUpdateMsg:
		// Always update logs, regardless of current view
		model.updateLogsView()
		// If we're currently viewing logs, we need to refresh the viewport
		if model.currentView() == ViewLogs || model.currentView() == ViewURLs {
			// Force a viewport update to show new content
			model.logsViewController.GetLogsViewport().GotoBottom()
		}
		cmds = append(cmds, model.waitForUpdates())

	case logsClearedMsg:
		// Handle logs cleared - update logs view
		model.updateLogsView()
		cmds = append(cmds, model.waitForUpdates())
	}

	return model, tea.Batch(cmds...)
}
