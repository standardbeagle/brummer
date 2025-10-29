package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// AICoderMouseMessageHandler handles mouse events specifically for the AI Coder
// This handler has high priority to ensure mouse wheel events reach the AI Coder content
type AICoderMouseMessageHandler struct{}

// NewAICoderMouseMessageHandler creates a new AI Coder mouse message handler
func NewAICoderMouseMessageHandler() MessageHandler {
	return &AICoderMouseMessageHandler{}
}

// CanHandle returns true only for mouse messages when in AI Coder view with active session
func (h *AICoderMouseMessageHandler) CanHandle(msg tea.Msg) bool {
	// Only handle mouse messages
	mouseMsg, ok := msg.(tea.MouseMsg)
	if !ok {
		return false
	}

	// Only handle mouse wheel events
	if mouseMsg.Type != tea.MouseWheelUp && mouseMsg.Type != tea.MouseWheelDown {
		return false
	}

	return true // Let HandleMessage do the session validation
}

// HandleMessage routes mouse wheel events to the AI Coder component when active
func (h *AICoderMouseMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	// Only process if we're in AI Coder view
	if model.currentView() != ViewAICoders {
		return model, nil
	}

	// Check if we have an active AI Coder session
	if model.masterData == nil || model.masterData.AICoderState == nil {
		return model, nil
	}

	aiState := model.masterData.AICoderState
	focusedSession := aiState.FocusedSession

	// Only handle mouse events if there's a focused session
	if focusedSession == "" {
		return model, nil
	}

	// Route the mouse event directly to the AI Coder component
	if model.aicodersView != nil {
		cmd := model.aicodersView.Update(msg)
		if cmd != nil {
			return model, cmd
		}
	}

	return model, nil
}