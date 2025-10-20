package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// KeyboardMessageHandler integrates KeyHandlerRegistry into the MessageRouter
// This consolidates keyboard handling into the same routing flow as other messages
type KeyboardMessageHandler struct {
	keyRegistry *KeyHandlerRegistry
}

// NewKeyboardMessageHandler creates a new keyboard message handler
func NewKeyboardMessageHandler(keyRegistry *KeyHandlerRegistry) MessageHandler {
	return &KeyboardMessageHandler{
		keyRegistry: keyRegistry,
	}
}

// CanHandle returns true only for keyboard messages
func (h *KeyboardMessageHandler) CanHandle(msg tea.Msg) bool {
	_, ok := msg.(tea.KeyMsg)
	return ok
}

// HandleMessage routes keyboard messages through the key handler registry
func (h *KeyboardMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return model, nil
	}

	// Build rich context for key handlers
	ctx := model.buildKeyContext()

	// Route through the key registry
	cmd := h.keyRegistry.HandleKey(keyMsg, ctx, model)

	return model, cmd
}
