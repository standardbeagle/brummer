package tui

import (
	tea "github.com/charmbracelet/bubbletea"
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
	case aiCoderSessionMsg, aiCoderOutputMsg, aiCoderErrorMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes AI coder-related messages
func (h *AICoderMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case aiCoderSessionMsg:
		// Handle AI coder session events
		if model.aiCoderController != nil {
			// Use the Update method which handles all AI coder messages
			if cmd := model.aiCoderController.Update(m); cmd != nil {
				return model, cmd
			}
		}

	case aiCoderOutputMsg:
		// Handle AI coder output
		if model.aiCoderController != nil {
			// Use the Update method which handles all AI coder messages
			if cmd := model.aiCoderController.Update(m); cmd != nil {
				return model, cmd
			}
		}

	case aiCoderErrorMsg:
		// Handle AI coder error
		if model.aiCoderController != nil {
			// Use the Update method which handles all AI coder messages
			if cmd := model.aiCoderController.Update(m); cmd != nil {
				return model, cmd
			}
		}
	}

	return model, nil
}
