package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ViewMessageHandler handles view-related messages and keyboard input
type ViewMessageHandler struct{}

// NewViewMessageHandler creates a new view message handler
func NewViewMessageHandler() MessageHandler {
	return &ViewMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *ViewMessageHandler) CanHandle(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyMsg, tea.WindowSizeMsg, tea.MouseMsg, urlDetectedMsg, refreshMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes view-related messages
func (h *ViewMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.KeyMsg:
		// Delegate keyboard handling to input controller
		if model.inputController != nil {
			newModel, cmd, handled := model.inputController.HandleKeyMsg(m)
			if handled {
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				model = newModel.(*Model)
			}
		}

	case tea.WindowSizeMsg:
		// Update model dimensions
		model.width = m.Width
		model.height = m.Height

		// Update layout controller
		if model.layoutController != nil {
			model.layoutController.UpdateSize(m.Width, m.Height)
		}

		// Update all view controllers with new size
		if model.processViewController != nil {
			model.processViewController.UpdateSize(m.Width, m.Height, model.layoutController.GetHeaderHeight(), model.layoutController.GetFooterHeight())
		}
		if model.logsViewController != nil {
			model.logsViewController.UpdateSize(m.Width, m.Height, model.layoutController.GetHeaderHeight(), model.layoutController.GetFooterHeight())
		}
		if model.webViewController != nil {
			model.webViewController.UpdateSize(m.Width, m.Height, model.layoutController.GetHeaderHeight(), model.layoutController.GetFooterHeight())
		}
		if model.settingsController != nil {
			model.settingsController.UpdateSize(m.Width, m.Height, model.layoutController.GetHeaderHeight(), model.layoutController.GetFooterHeight())
		}

	case tea.MouseMsg:
		// Handle mouse events (placeholder for future mouse support)
		// Currently not implemented

	case urlDetectedMsg:
		// Handle URL detection - update web view
		if model.webViewController != nil {
			// URL detection is handled internally by the web view controller during updates
		}

	case refreshMsg:
		// Handle refresh request
		// Could trigger data refreshes in various controllers
		if model.currentView() == ViewProcesses && model.processViewController != nil {
			// Refresh process list
			model.updateProcessList()
			cmds = append(cmds, model.waitForUpdates())
		}
	}

	return model, tea.Batch(cmds...)
}
