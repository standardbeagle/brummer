package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// NotificationsMessageHandler handles all messages by updating the notifications component
// This ensures notifications see all messages for proper display and state tracking
type NotificationsMessageHandler struct{}

// NewNotificationsMessageHandler creates a new notifications message handler
func NewNotificationsMessageHandler() MessageHandler {
	return &NotificationsMessageHandler{}
}

// CanHandle always returns true since notifications should see all messages
// Priority is set to 0 (highest) so it runs before other handlers
func (h *NotificationsMessageHandler) CanHandle(msg tea.Msg) bool {
	return true
}

// HandleMessage updates the notifications component and passes through
// The handler never blocks other handlers from processing the message
func (h *NotificationsMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update notifications component if present
	if model.notificationsComponent != nil {
		_, cmd := model.notificationsComponent.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// ALWAYS return model unchanged and let other handlers process the message
	// This is a "pass-through" handler that just observes messages
	return model, tea.Batch(cmds...)
}
