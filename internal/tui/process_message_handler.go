package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ProcessMessageHandler handles process-related messages
type ProcessMessageHandler struct{}

// NewProcessMessageHandler creates a new process message handler
func NewProcessMessageHandler() MessageHandler {
	return &ProcessMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *ProcessMessageHandler) CanHandle(msg tea.Msg) bool {
	switch msg.(type) {
	case processUpdateMsg, processStoppedMsg, processStartedMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes process-related messages
func (h *ProcessMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.(type) {
	case processUpdateMsg:
		// Update process list
		model.updateProcessList()
		cmds = append(cmds, model.waitForUpdates())

	case processStoppedMsg:
		// Handle process stopped - update process list
		model.updateProcessList()
		cmds = append(cmds, model.waitForUpdates())

	case processStartedMsg:
		// Handle process started - update process list
		model.updateProcessList()
		cmds = append(cmds, model.waitForUpdates())
	}

	return model, tea.Batch(cmds...)
}
