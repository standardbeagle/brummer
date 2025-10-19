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
	// Handle both system messages and status messages
	_, ok1 := msg.(messages.SystemMessageAddedMsg)
	_, ok2 := msg.(StatusMsg)
	return ok1 || ok2
}

// HandleMessage processes system messages
func (h *SystemMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.SystemMessageAddedMsg:
		// Add system message to the system component
		if model.systemComponent != nil {
			// Convert system message to service format and add via component
			cmd := model.systemComponent.AddSystemMessage(
				msg.Level,
				msg.Content,
				msg.Source,
			)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			// Debug log to verify system messages are being received
			if strings.Contains(msg.Content, "MCP") {
				model.logStore.Add("system-debug", "TUI", fmt.Sprintf("Received MCP system message: %s", msg.Content), false)
			}
		}

	case StatusMsg:
		// Handle status messages (like "Press Ctrl+C again to quit")
		if model.logStore != nil {
			model.logStore.Add("status", "System", msg.Message, false)
		}

		// For critical status messages like quit confirmation, also add to system component
		if model.systemComponent != nil && strings.Contains(msg.Message, "quit") {
			cmd := model.systemComponent.AddSystemMessage(
				"info",
				msg.Message,
				"QuitHandler",
			)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	default:
		return model, nil
	}

	cmds = append(cmds, model.waitForUpdates())
	return model, tea.Batch(cmds...)
}

