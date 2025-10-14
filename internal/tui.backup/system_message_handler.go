package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// SystemMessageHandler handles system message events
type SystemMessageHandler struct{}

// NewSystemMessageHandler creates a new system message handler
func NewSystemMessageHandler() MessageHandler {
	return &SystemMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *SystemMessageHandler) CanHandle(msg tea.Msg) bool {
	_, ok := msg.(messages.SystemMessageAddedMsg)
	return ok
}

// HandleMessage processes system messages
func (h *SystemMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	sysMsg, ok := msg.(messages.SystemMessageAddedMsg)
	if !ok {
		return model, nil
	}

	// Add system message to the system component
	if model.systemComponent != nil {
		// Convert system message to service format and add via component
		cmd := model.systemComponent.AddSystemMessage(
			sysMsg.Level,
			sysMsg.Content,
			sysMsg.Source,
		)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Debug log to verify system messages are being received
		if strings.Contains(sysMsg.Content, "MCP") {
			model.logStore.Add("system-debug", "TUI", fmt.Sprintf("Received MCP system message: %s", sysMsg.Content), false)
		}
	}

	cmds = append(cmds, model.waitForUpdates())
	return model, tea.Batch(cmds...)
}

