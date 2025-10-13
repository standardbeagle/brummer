package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// NavigationMessageHandler handles navigation-related messages
type NavigationMessageHandler struct{}

// NewNavigationMessageHandler creates a new navigation message handler
func NewNavigationMessageHandler() MessageHandler {
	return &NavigationMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *NavigationMessageHandler) CanHandle(msg tea.Msg) bool {
	switch msg.(type) {
	case messages.ViewSwitchRequestedMsg, messages.ViewChangedMsg, messages.ViewOrderChangedMsg, messages.NumberKeyMappingChangedMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes navigation-related messages
func (h *NavigationMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Route navigation messages to the NavigationComponent
	if model.navigationComponent != nil {
		updatedComponent, cmd := model.navigationComponent.Update(msg)
		model.navigationComponent = updatedComponent.(*components.NavigationComponent)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Special handling for ViewChangedMsg to trigger re-render
		if _, ok := msg.(messages.ViewChangedMsg); ok {
			// Force an update to re-render with the new view
			cmds = append(cmds, func() tea.Msg {
				return nil // Trigger a re-render
			})
		}
	}

	return model, tea.Batch(cmds...)
}