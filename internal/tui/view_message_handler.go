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

		// Layout calculations now use components.LayoutManager (T021)
		// Header and footer heights are constants
		headerHeight := 3
		footerHeight := 1

		// Calculate content height once for all controllers
		contentHeight := model.calculateContentHeight()

		// Update all view controllers with new size including content height
		if model.processesView != nil {
			model.processesView.UpdateSize(m.Width, m.Height, headerHeight, footerHeight, contentHeight)
		}
		if model.logsView != nil {
			model.logsView.UpdateSize(m.Width, m.Height, headerHeight, footerHeight, contentHeight)
		}
		if model.webView != nil {
			model.webView.UpdateSize(m.Width, m.Height, headerHeight, footerHeight, contentHeight)
		}
		if model.settingsView != nil {
			model.settingsView.UpdateSize(m.Width, m.Height, headerHeight, footerHeight, contentHeight)
		}

		// Update overlay controllers
		if model.commandWindowView != nil {
			model.commandWindowView.UpdateSize(m.Width, m.Height)
		}
		// Note: ScriptSelectorController gets size directly via getTerminalSize()
		// to ensure proper sizing for overlay rendering

	case tea.MouseMsg:
		// Handle mouse events (placeholder for future mouse support)
		// Currently not implemented

	case urlDetectedMsg:
		// Handle URL detection - update web view
		if model.webView != nil {
			// URL detection is handled internally by the web view component during updates
		}

	case refreshMsg:
		// Handle refresh request
		// Could trigger data refreshes in various controllers
		if model.currentView() == ViewProcesses && model.processesView != nil {
			// Refresh process list
			model.updateProcessList()
			cmds = append(cmds, model.waitForUpdates())
		}
	}

	return model, tea.Batch(cmds...)
}
