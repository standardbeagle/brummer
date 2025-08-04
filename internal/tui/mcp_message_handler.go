package tui

import (
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
		// Handle MCP connection status change
		if model.mcpDebugController != nil {
			model.mcpDebugController.HandleConnection(m)
			if model.currentView() == ViewMCPConnections {
				model.mcpDebugController.UpdateConnectionsList()
			}
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
