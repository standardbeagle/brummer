package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ViewSpecificHandler handles view-specific keyboard interactions
type ViewSpecificHandler struct{}

// NewViewSpecificHandler creates a new view-specific handler
func NewViewSpecificHandler() *ViewSpecificHandler {
	return &ViewSpecificHandler{}
}

// CanHandle checks if this handler can process the message
func (h *ViewSpecificHandler) CanHandle(msg tea.Msg) bool {
	// This handler is deprecated - no longer handles specific messages
	return false
}

// HandleMessage processes view-specific messages
func (h *ViewSpecificHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	// This handler is deprecated - no longer processes messages
	return model, nil
}

// handleProcessViewKeys handles Process view specific keyboard interactions (Redux/Elm compliant)
func (h *ViewSpecificHandler) handleProcessViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// This handler is deprecated - process management now handled by new key handlers
	// For now, just handle basic key events to prevent crashes

	// Add a basic process selection update when arrow keys are used
	if model.masterData != nil && model.masterData.ProcessesState != nil {
		// This would need more complex implementation to track selection state
		// For now, just ensure no crashes
	}

	return model, tea.Batch(cmds...)
}