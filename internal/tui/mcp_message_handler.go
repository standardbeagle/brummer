package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// MCPMessageHandler handles MCP-related messages
type MCPMessageHandler struct{}

// NewMCPMessageHandler creates a new MCP message handler
func NewMCPMessageHandler() MessageHandler {
	return &MCPMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *MCPMessageHandler) CanHandle(msg tea.Msg) bool {
	switch msg.(type) {
	case mcpConnectionMsg, mcpRequestMsg, mcpResponseMsg, mcpErrorMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes MCP-related messages
func (h *MCPMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case mcpConnectionMsg:
		// Handle MCP connection status change (Redux/Elm pattern)
		if model.masterData != nil && model.masterData.MCPDebugState != nil {
			connectedTime := time.Unix(0, m.connectedAt)
			// Update state immutably
			model.masterData.MCPDebugState = model.masterData.MCPDebugState.AddConnection(
				m.sessionId, m.clientInfo, m.connected, connectedTime, m.connectionType, m.method,
			)
		}
		cmds = append(cmds, model.waitForUpdates())

	case mcpRequestMsg:
		// Handle MCP request - placeholder for future implementation
		cmds = append(cmds, model.waitForUpdates())

	case mcpResponseMsg:
		// Handle MCP response - placeholder for future implementation
		cmds = append(cmds, model.waitForUpdates())

	case mcpErrorMsg:
		// Handle MCP error - placeholder for future implementation
		cmds = append(cmds, model.waitForUpdates())
	}

	return model, tea.Batch(cmds...)
}
