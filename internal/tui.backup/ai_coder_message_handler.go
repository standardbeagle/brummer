package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/components"
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
	// Route messages to the AI Coders view component (Feature 005)
	if model.aicodersView != nil {
		switch m := msg.(type) {
		case aiCoderSessionMsg:
			// Handle AI coder session events via component
			updatedComponent, cmd := model.aicodersView.Update(m)
			model.aicodersView = updatedComponent.(*components.AICodersViewComponent)
			return model, cmd

		case aiCoderOutputMsg:
			// Handle AI coder output via component
			updatedComponent, cmd := model.aicodersView.Update(m)
			model.aicodersView = updatedComponent.(*components.AICodersViewComponent)
			return model, cmd

		case aiCoderErrorMsg:
			// Handle AI coder error via component
			updatedComponent, cmd := model.aicodersView.Update(m)
			model.aicodersView = updatedComponent.(*components.AICodersViewComponent)
			return model, cmd
		}
	}

	return model, nil
}
