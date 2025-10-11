package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/system"
)

// SystemMessageHandler handles system message events
type SystemMessageHandler struct{}

// NewSystemMessageHandler creates a new system message handler
func NewSystemMessageHandler() MessageHandler {
	return &SystemMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *SystemMessageHandler) CanHandle(msg tea.Msg) bool {
	_, ok := msg.(system.SystemMessageMsg)
	return ok
}

// HandleMessage processes system messages
func (h *SystemMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	sysMsg, ok := msg.(system.SystemMessageMsg)
	if !ok {
		return model, nil
	}

	// Add system message to the system controller
	if model.systemController != nil {
		model.systemController.AddMessage(sysMsg.Level, sysMsg.Context, sysMsg.Message)
		model.systemController.UpdateSize(model.width, model.height, model.headerHeight, model.footerHeight)

		// Debug log to verify system messages are being received
		if strings.Contains(sysMsg.Message, "MCP") {
			model.logStore.Add("system-debug", "TUI", fmt.Sprintf("Received MCP system message: %s", sysMsg.Message), false)
		}

		// Forward to debug forwarder if enabled
		// TODO(Feature 005): Re-implement debug forwarding for service-based architecture
		// The debug forwarder was part of the old controller-based system
	}

	cmds = append(cmds, model.waitForUpdates())
	return model, tea.Batch(cmds...)
}
