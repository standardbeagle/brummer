package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
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
	case LogUpdateMsg, logsClearedMsg, messages.LogAddedMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes log-related messages
func (h *LogMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.(type) {
	case LogUpdateMsg, messages.LogAddedMsg:
		// Always update logs, regardless of current view
		// LogAddedMsg is sent by ServiceBridge when a new log entry is added
		model.updateLogsView()
		// If we're currently viewing logs and auto-scroll is enabled, mark as at bottom
		if (model.currentView() == ViewLogs || model.currentView() == ViewURLs) && model.masterData != nil && model.masterData.LogsState != nil {
			if model.masterData.LogsState.GetAutoScroll() {
				model.masterData.LogsState = model.masterData.LogsState.SetAtBottom(true)
			}
		}
		cmds = append(cmds, model.waitForUpdates())

	case logsClearedMsg:
		// Handle logs cleared - update logs view
		model.updateLogsView()
		cmds = append(cmds, model.waitForUpdates())
	}

	return model, tea.Batch(cmds...)
}
