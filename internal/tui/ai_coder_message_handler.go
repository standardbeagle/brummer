package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// AICoderMessageHandler handles AI coder-related messages
type AICoderMessageHandler struct{}

// NewAICoderMessageHandler creates a new AI coder message handler
func NewAICoderMessageHandler() MessageHandler {
	return &AICoderMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *AICoderMessageHandler) CanHandle(msg tea.Msg) bool {
	switch msg.(type) {
	case messages.AICoderSessionCreatedMsg, messages.AICoderSessionUpdatedMsg, messages.AICoderSessionEndedMsg:
		return true
	case messages.AICoderPTYOutputMsg, messages.AICoderErrorMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes AI coder-related messages
func (h *AICoderMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	// Update AI coder state in master data directly (Redux/Elm pattern)
	if model.masterData != nil && model.masterData.AICoderState != nil {
		switch m := msg.(type) {
		case messages.AICoderSessionCreatedMsg:
			// Add new session to AI coder state
			newState := model.masterData.AICoderState.AddSession(m.SessionID, m.Name)
			model.masterData.AICoderState = newState
			return model, nil

		case messages.AICoderSessionUpdatedMsg:
			// Update existing session status in AI coder state
			newState := model.masterData.AICoderState.UpdateSessionStatus(m.SessionID, string(m.Status))
			model.masterData.AICoderState = newState
			return model, nil

		case messages.AICoderSessionEndedMsg:
			// Remove session from AI coder state
			newState := model.masterData.AICoderState.RemoveSession(m.SessionID)
			model.masterData.AICoderState = newState
			return model, nil

		case messages.AICoderPTYOutputMsg:
			// Write PTY output to log store for visibility in logs tab
			if len(m.Data) > 0 && model.logStore != nil {
				outputStr := string(m.Data)
				model.logStore.Add("ai-coder", m.SessionID, outputStr, false)
			}

			// Add PTY output to AI coder state
			newState := model.masterData.AICoderState.AddPTYOutput(m.SessionID, string(m.Data))
			model.masterData.AICoderState = newState
			return model, nil

		case messages.AICoderErrorMsg:
			// Write AI coder errors to log store
			if model.logStore != nil && m.Error != nil {
				model.logStore.Add("ai-coder", m.SessionID, fmt.Sprintf("AI Coder Error: %s", m.Error.Error()), true)
			}

			// Add error to AI coder state
			newState := model.masterData.AICoderState.AddPTYOutput(m.SessionID, fmt.Sprintf("ERROR: %s", m.Error.Error()))
			model.masterData.AICoderState = newState
			return model, nil
		}
	}

	return model, nil
}
