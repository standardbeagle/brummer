package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// SettingsMessageHandler handles settings-related messages
type SettingsMessageHandler struct{}

// NewSettingsMessageHandler creates a new settings message handler
func NewSettingsMessageHandler() *SettingsMessageHandler {
	return &SettingsMessageHandler{}
}

// CanHandle checks if the handler can process the given message
func (h *SettingsMessageHandler) CanHandle(msg tea.Msg) bool {
	switch msg.(type) {
	case messages.SettingsRefreshedMsg:
		return true
	case messages.SettingsMCPInstalledMsg:
		return true
	case messages.SettingsErrorMsg:
		return true
	case messages.SettingsUpdateMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes the message and returns updated model and commands
func (h *SettingsMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.SettingsRefreshedMsg:
		// Update settings state using Redux/Elm pattern
		if model.masterData != nil && model.masterData.SettingsState != nil {
			// Update state immutably with refreshed settings info
			model.masterData.SettingsState = model.masterData.SettingsState.UpdateSettingsInfo(msg.SettingsInfo)
		}

	case messages.SettingsMCPInstalledMsg:
		// MCP configuration was installed - could trigger a refresh
		// This will be handled by the main model requesting a settings refresh
		// For now, just mark that settings need refresh
		if model.masterData != nil && model.masterData.SettingsState != nil {
			model.masterData.SettingsState = model.masterData.SettingsState.MarkForRefresh()
		}

	case messages.SettingsErrorMsg:
		// Handle settings errors - update state with error
		if model.masterData != nil && model.masterData.SettingsState != nil {
			model.masterData.SettingsState = model.masterData.SettingsState.SetError(fmt.Errorf("settings error: %s", msg.Error))
		}

	case messages.SettingsUpdateMsg:
		// Handle settings update requests
		var cmd tea.Cmd
		if msg.Refresh {
			// Request settings refresh - this would be handled by ServiceBridge
			// For now, just return nil as the service will emit SettingsRefreshedMsg
			cmd = nil
		}
		if msg.InstallMCP {
			// Request MCP installation - this would be handled by ServiceBridge
			// For now, just return nil as the service will emit SettingsMCPInstalledMsg
			cmd = nil
		}
		return model, cmd
	}

	return model, nil
}
