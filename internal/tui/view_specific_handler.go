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
		currentFilter := model.webViewController.GetWebFilter()
		switch currentFilter {
		case "all":
			model.webViewController.SetWebFilter("pages")
		case "pages":
			model.webViewController.SetWebFilter("api")
		case "api":
			model.webViewController.SetWebFilter("images")
		case "images":
			model.webViewController.SetWebFilter("other")
		case "other":
			model.webViewController.SetWebFilter("all")
		default:
			model.webViewController.SetWebFilter("all")
		}

		// Reset selection to first item if available
		if len(model.webViewController.GetWebRequestsList().Items()) > 0 {
			model.webViewController.GetWebRequestsList().Select(0)
		}
		return model, nil

	case "up", "k":
		// Navigate up in request list - delegate to list component
		*model.webViewController.GetWebRequestsList(), _ = model.webViewController.GetWebRequestsList().Update(msg)
		model.webViewController.SetWebAutoScroll(false) // Disable auto-scroll when manually navigating
		return model, nil

	case "down", "j":
		// Navigate down in request list - delegate to list component
		*model.webViewController.GetWebRequestsList(), _ = model.webViewController.GetWebRequestsList().Update(msg)
		model.webViewController.SetWebAutoScroll(false) // Disable auto-scroll when manually navigating
		return model, nil

	case "enter":
		// Select request for detail view - handled by controller
		return model, nil

	case "pgup":
		// Page up in web list, disable auto-scroll
		model.webViewController.SetWebAutoScroll(false)
		*model.webViewController.GetWebRequestsList(), _ = model.webViewController.GetWebRequestsList().Update(msg)
		return model, nil

	case "pgdown":
		// Page down in web list
		*model.webViewController.GetWebRequestsList(), _ = model.webViewController.GetWebRequestsList().Update(msg)
		return model, nil

	case "end":
		// End key re-enables auto-scroll and goes to bottom
		model.webViewController.SetWebAutoScroll(true)
		// Go to last item in list
		if len(model.webViewController.GetWebRequestsList().Items()) > 0 {
			model.webViewController.GetWebRequestsList().Select(len(model.webViewController.GetWebRequestsList().Items()) - 1)
		}
		return model, nil

	case "home":
		// Home key goes to top and disables auto-scroll
		model.webViewController.SetWebAutoScroll(false)
		model.webViewController.GetWebRequestsList().Select(0)
		return model, nil
	}

	return model, nil
}

// handleProcessViewKeys handles Process view specific keyboard interactions
func (h *ViewSpecificHandler) handleProcessViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, model.keys.Stop):
		if i, ok := model.processViewController.GetProcessesList().SelectedItem().(processItem); ok && !i.isHeader && i.process != nil {
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
		if i, ok := model.processViewController.GetProcessesList().SelectedItem().(processItem); ok && !i.isHeader && i.process != nil {
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
	newList, cmd := model.processViewController.GetProcessesList().Update(msg)
	*model.processViewController.GetProcessesList() = newList
	cmds = append(cmds, cmd)

	return model, tea.Batch(cmds...)
}

// handleLogsViewKeys handles Logs/URLs view keyboard interactions
func (h *ViewSpecificHandler) handleLogsViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	if model.currentView() == ViewLogs {
		switch {
		case key.Matches(msg, model.keys.Up):
			// Disable auto-scroll when user scrolls up
			if model.logsViewController.IsAutoScrollEnabled() {
				model.logsViewController.ToggleAutoScroll()
			}
			model.logsViewController.GetLogsViewport().LineUp(1)
			return model, nil

		case key.Matches(msg, model.keys.Down):
			model.logsViewController.GetLogsViewport().LineDown(1)
			// Check if we're at the bottom
			if model.logsViewController.GetLogsViewport().AtBottom() {
				if !model.logsViewController.IsAutoScrollEnabled() {
					model.logsViewController.ToggleAutoScroll()
				}
			}
			return model, nil

		case msg.String() == "pgup":
			if model.logsViewController.IsAutoScrollEnabled() {
				model.logsViewController.ToggleAutoScroll()
			}
			model.logsViewController.GetLogsViewport().ViewUp()
			return model, nil

		case msg.String() == "pgdown":
			model.logsViewController.GetLogsViewport().ViewDown()
			if model.logsViewController.GetLogsViewport().AtBottom() {
				if !model.logsViewController.IsAutoScrollEnabled() {
					model.logsViewController.ToggleAutoScroll()
				}
			}
			return model, nil

		case msg.String() == "end":
			// End key re-enables auto-scroll and goes to bottom
			if !model.logsViewController.IsAutoScrollEnabled() {
				model.logsViewController.ToggleAutoScroll()
			}
			model.logsViewController.GetLogsViewport().GotoBottom()
			return model, nil

		case msg.String() == "home":
			// Home key goes to top and disables auto-scroll
			if model.logsViewController.IsAutoScrollEnabled() {
				model.logsViewController.ToggleAutoScroll()
			}
			model.logsViewController.GetLogsViewport().GotoTop()
			return model, nil
		}
	}

	return model, nil
}

// Placeholder handlers for other views
func (h *ViewSpecificHandler) handleErrorsViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	// Update the errors list
	if model.errorsViewController != nil {
		newList, cmd := model.errorsViewController.GetErrorsList().Update(msg)
		*model.errorsViewController.GetErrorsList() = newList
		return model, cmd
	}
	return model, nil
}

func (h *ViewSpecificHandler) handleSettingsViewKeys(msg tea.KeyMsg, model *Model) (tea.Model, tea.Cmd) {
	// Update the settings list
	if model.settingsController != nil {
		newList, cmd := model.settingsController.GetSettingsList().Update(msg)
		*model.settingsController.GetSettingsList() = newList
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
