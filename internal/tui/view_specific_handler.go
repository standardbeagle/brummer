package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewSpecificHandler handles view-specific interactions and keyboard commands
type ViewSpecificHandler struct{}

// NewViewSpecificHandler creates a new view-specific handler
func NewViewSpecificHandler() MessageHandler {
	return &ViewSpecificHandler{}
}

// CanHandle checks if this handler can process the message
func (h *ViewSpecificHandler) CanHandle(msg tea.Msg) bool {
	// Handle keyboard messages for view-specific interactions
	_, ok := msg.(tea.KeyMsg)
	return ok
}

// HandleMessage processes view-specific keyboard interactions
func (h *ViewSpecificHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return model, nil
	}

	switch model.currentView() {
	case ViewWeb:
		return h.handleWebViewKeys(keyMsg, model)
	case ViewProcesses:
		return h.handleProcessViewKeys(keyMsg, model)
	case ViewLogs, ViewURLs:
		return h.handleLogsViewKeys(keyMsg, model)
	case ViewErrors:
		return h.handleErrorsViewKeys(keyMsg, model)
	case ViewSettings:
		return h.handleSettingsViewKeys(keyMsg, model)
	// ViewFileBrowser not defined - removed
	case ViewMCPConnections:
		return h.handleMCPViewKeys(keyMsg, model)
	case ViewAICoders:
		return h.handleAICodersKeys(keyMsg, model)
	}

	return model, tea.Batch(cmds...)
}

// handleWebViewKeys handles Web view specific keyboard interactions
func (h *ViewSpecificHandler) handleWebViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "f":
		// Cycle through filters: all -> pages -> api -> images -> other -> all
		currentFilter := model.webView.GetWebFilter()
		switch currentFilter {
		case "all":
			model.webView.SetWebFilter("pages")
		case "pages":
			model.webView.SetWebFilter("api")
		case "api":
			model.webView.SetWebFilter("images")
		case "images":
			model.webView.SetWebFilter("other")
		case "other":
			model.webView.SetWebFilter("all")
		default:
			model.webView.SetWebFilter("all")
		}

		// Reset selection to first item if available
		model.webView.SelectFirst()
		return model, nil

	case "up", "k":
		// Navigate up in request list
		model.webView.NavigateUp()
		model.webView.SetWebAutoScroll(false) // Disable auto-scroll when manually navigating
		return model, nil

	case "down", "j":
		// Navigate down in request list
		model.webView.NavigateDown()
		model.webView.SetWebAutoScroll(false) // Disable auto-scroll when manually navigating
		return model, nil

	case "enter":
		// Select request for detail view - handled by component
		return model, nil

	case "pgup":
		// Page up in web list, disable auto-scroll
		model.webView.SetWebAutoScroll(false)
		model.webView.PageUp()
		return model, nil

	case "pgdown":
		// Page down in web list
		model.webView.PageDown()
		return model, nil

	case "end":
		// End key re-enables auto-scroll and goes to bottom
		model.webView.SetWebAutoScroll(true)
		model.webView.SelectLast()
		return model, nil

	case "home":
		// Home key goes to top and disables auto-scroll
		model.webView.SetWebAutoScroll(false)
		model.webView.SelectFirst()
		return model, nil
	}

	return model, nil
}

// handleProcessViewKeys handles Process view specific keyboard interactions
func (h *ViewSpecificHandler) handleProcessViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, model.keys.Stop):
		if i, ok := model.processesView.GetProcessesList().SelectedItem().(processItem); ok && !i.isHeader && i.process != nil {
			if err := model.processMgr.StopProcess(i.process.ID); err != nil {
				msg := fmt.Sprintf("Failed to stop process %s: %v", i.process.Name, err)
				model.logStore.Add("system", "System", msg, true)
				model.systemController.AddMessage("error", "Process Control", msg)
			} else {
				msg := fmt.Sprintf("Stopping process: %s", i.process.Name)
				model.logStore.Add("system", "System", msg, false)
				model.systemController.AddMessage("info", "Process Control", msg)
			}
			cmds = append(cmds, model.waitForUpdates())
		} else {
			msg := "No process selected to stop"
			model.logStore.Add("system", "System", msg, true)
			model.systemController.AddMessage("error", "Process Control", msg)
		}
		return model, tea.Batch(cmds...)

	case key.Matches(msg, model.keys.Restart):
		if i, ok := model.processesView.GetProcessesList().SelectedItem().(processItem); ok && !i.isHeader && i.process != nil {
			cmds = append(cmds, model.handleRestartProcess(i.process))
			msg := fmt.Sprintf("Restarting process: %s", i.process.Name)
			model.logStore.Add("system", "System", msg, false)
			model.systemController.AddMessage("info", "Process Control", msg)
		} else {
			msg := "No process selected to restart"
			model.logStore.Add("system", "System", msg, true)
			model.systemController.AddMessage("error", "Process Control", msg)
		}
		return model, tea.Batch(cmds...)
	}

	// Update the controller's list for other keys
	newList, cmd := model.processesView.GetProcessesList().Update(msg)
	*model.processesView.GetProcessesList() = newList
	cmds = append(cmds, cmd)

	return model, tea.Batch(cmds...)
}

// handleLogsViewKeys handles Logs/URLs view keyboard interactions
func (h *ViewSpecificHandler) handleLogsViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	if model.currentView() == ViewLogs {
		switch {
		case key.Matches(msg, model.keys.Up):
			// Disable auto-scroll when user scrolls up
			if model.logsView.IsAutoScrollEnabled() {
				model.logsView.ToggleAutoScroll()
			}
			model.logsView.GetLogsViewport().LineUp(1)
			return model, nil

		case key.Matches(msg, model.keys.Down):
			model.logsView.GetLogsViewport().LineDown(1)
			// Check if we're at the bottom
			if model.logsView.GetLogsViewport().AtBottom() {
				if !model.logsView.IsAutoScrollEnabled() {
					model.logsView.ToggleAutoScroll()
				}
			}
			return model, nil

		case msg.String() == "pgup":
			if model.logsView.IsAutoScrollEnabled() {
				model.logsView.ToggleAutoScroll()
			}
			model.logsView.GetLogsViewport().ViewUp()
			return model, nil

		case msg.String() == "pgdown":
			model.logsView.GetLogsViewport().ViewDown()
			if model.logsView.GetLogsViewport().AtBottom() {
				if !model.logsView.IsAutoScrollEnabled() {
					model.logsView.ToggleAutoScroll()
				}
			}
			return model, nil

		case msg.String() == "end":
			// End key re-enables auto-scroll and goes to bottom
			if !model.logsView.IsAutoScrollEnabled() {
				model.logsView.ToggleAutoScroll()
			}
			model.logsView.GetLogsViewport().GotoBottom()
			return model, nil

		case msg.String() == "home":
			// Home key goes to top and disables auto-scroll
			if model.logsView.IsAutoScrollEnabled() {
				model.logsView.ToggleAutoScroll()
			}
			model.logsView.GetLogsViewport().GotoTop()
			return model, nil
		}
	}

	return model, nil
}

// Placeholder handlers for other views
func (h *ViewSpecificHandler) handleErrorsViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	// Update the errors list
	if model.errorsView != nil {
		newList, cmd := model.errorsView.GetErrorsList().Update(msg)
		*model.errorsView.GetErrorsList() = newList
		return model, cmd
	}
	return model, nil
}

func (h *ViewSpecificHandler) handleSettingsViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	// Update the settings list
	if model.settingsView != nil {
		newList, cmd := model.settingsView.GetSettingsList().Update(msg)
		*model.settingsView.GetSettingsList() = newList
		return model, cmd
	}
	return model, nil
}

// handleFileBrowserKeys removed - ViewFileBrowser not defined

func (h *ViewSpecificHandler) handleMCPViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	// MCP view specific handling - placeholder
	return model, nil
}

func (h *ViewSpecificHandler) handleAICodersKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	// AI coders view specific handling - placeholder
	return model, nil
}
