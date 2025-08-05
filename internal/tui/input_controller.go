package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/system"
)

// InputController handles all keyboard input and routing
type InputController struct {
	// Dependencies
	model       *Model // Will be refactored to use interface in future iteration
	keys        keyMap
	viewConfigs map[View]ViewConfig
	debugMode   bool
}

// NewInputController creates a new input controller
func NewInputController(model *Model, keys keyMap, viewConfigs map[View]ViewConfig, debugMode bool) *InputController {
	return &InputController{
		model:       model,
		keys:        keys,
		viewConfigs: viewConfigs,
		debugMode:   debugMode,
	}
}

// HandleKeyMsg processes keyboard input and returns whether it was handled
func (ic *InputController) HandleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// Handle script selector view
	if ic.model.currentView() == ViewScriptSelector && ic.model.scriptSelectorController != nil {
		handled, cmd := ic.model.scriptSelectorController.HandleKeyMsg(msg)
		if handled {
			return ic.model, cmd, true
		}
	}

	// Handle command window first
	if ic.model.commandWindowController.IsShowingCommandWindow() {
		return ic.handleCommandWindow(msg)
	}

	// Handle "/" key for Brummer commands - check if we should intercept it
	if msg.String() == "/" || (msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == '/') {
		// Check if we should intercept the slash command
		shouldIntercept := true
		if ic.model.currentView() == ViewAICoders && ic.model.aiCoderController != nil {
			shouldIntercept = ic.model.aiCoderController.ShouldInterceptSlashCommand()
		}

		if shouldIntercept {
			ic.model.showCommandWindow()
			return ic.model, nil, true
		}
		// If not intercepting, fall through to PTY handling
	}

	// Check if we're in AI Coders view - route input to the controller
	// The controller will handle both focused and unfocused states
	if ic.model.currentView() == ViewAICoders && ic.model.aiCoderController != nil {
		// Route all input to the AI Coder controller when in AI Coders view
		// It will handle Enter to focus, keys when focused, etc.
		var cmd tea.Cmd
		cmd = ic.model.aiCoderController.Update(msg)
		// Only mark as handled if we're focused or it's a key the PTY view handles
		if ic.model.aiCoderController.IsTerminalFocused() {
			return ic.model, cmd, true
		}
		// Check if it's a key the PTY view handles when unfocused
		switch msg.String() {
		case "enter", "f11", "ctrl+h", "ctrl+n", "ctrl+shift+p", "ctrl+d", "f12", "pgup", "pgdown":
			return ic.model, cmd, true
		}
		// For other keys, continue to global key handling
	}

	// Handle global keys
	if model, cmd, handled := ic.handleGlobalKeys(msg); handled {
		return model, cmd, true
	}

	// Handle view-specific keys
	switch {
	case key.Matches(msg, ic.keys.ClearErrors):
		if ic.model.currentView() == ViewErrors {
			ic.model.handleClearErrors()
			return ic.model, nil, true
		}

	case key.Matches(msg, ic.keys.Enter):
		cmd := ic.handleEnter()
		return ic.model, cmd, true

	case key.Matches(msg, ic.keys.RunDialog):
		if !ic.model.commandWindowController.IsShowingRunDialog() {
			ic.model.showRunDialog()
		}
		return ic.model, nil, true
	}

	return ic.model, nil, false
}

// handleGlobalKeys handles keys that work across all views
func (ic *InputController) handleGlobalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, ic.keys.Quit):
		// Check if there are running processes
		runningProcesses := 0
		for _, proc := range ic.model.processMgr.GetAllProcesses() {
			if proc.GetStatus() == process.StatusRunning {
				runningProcesses++
			}
		}

		if runningProcesses > 0 {
			return ic.model, tea.Sequence(
				tea.Printf("Stopping %d running processes...\n", runningProcesses),
				func() tea.Msg {
					_ = ic.model.processMgr.Cleanup() // Ignore cleanup errors during shutdown
					return tea.Msg(nil)
				},
				tea.Printf("%s", renderExitScreen()),
				tea.Quit,
			), true
		} else {
			return ic.model, tea.Sequence(
				tea.Printf("%s", renderExitScreen()),
				tea.Quit,
			), true
		}

	case key.Matches(msg, ic.keys.Tab):
		ic.model.cycleView()
		return ic.model, nil, true

	case msg.String() == "shift+tab":
		ic.model.cyclePrevView()
		return ic.model, nil, true

	case msg.String() == "left":
		ic.model.cyclePrevView()
		return ic.model, nil, true

	case msg.String() == "right":
		ic.model.cycleView()
		return ic.model, nil, true

	case key.Matches(msg, ic.keys.ClearScreen):
		ic.model.handleClearScreen()
		return ic.model, nil, true

	case key.Matches(msg, ic.keys.Back):
		if ic.model.currentView() == ViewFilters {
			ic.model.navController.SwitchTo(ViewLogs)
		} else if ic.model.currentView() == ViewLogs || ic.model.currentView() == ViewErrors || ic.model.currentView() == ViewURLs {
			ic.model.navController.SwitchTo(ViewProcesses)
		}
		return ic.model, nil, true

	case key.Matches(msg, ic.keys.Priority):
		if ic.model.currentView() == ViewLogs {
			// Toggle high priority in LogsViewController
			ic.model.logsViewController.ToggleHighPriority()
			ic.model.updateLogsView()
		}
		return ic.model, nil, true

	case key.Matches(msg, ic.keys.RestartAll):
		if ic.model.currentView() == ViewProcesses {
			ic.model.logStore.Add("system", "System", "Restarting all running processes...", false)
			return ic.model, ic.model.handleRestartAll(), true
		}
		return ic.model, nil, true

	case key.Matches(msg, ic.keys.CopyError):
		return ic.model, ic.model.handleCopyError(), true

	case key.Matches(msg, ic.keys.ClearLogs):
		if ic.model.currentView() == ViewLogs {
			ic.model.handleClearLogs()
		}
		return ic.model, nil, true

	case key.Matches(msg, ic.keys.ToggleError):
		if ic.model.systemController != nil && ic.model.systemController.HasMessages() {
			// Toggle system panel via layout controller
			if ic.model.layoutController != nil {
				current := ic.model.layoutController.GetSystemPanelHeight() > 0
				ic.model.layoutController.SetSystemPanelOpen(!current)
			}
		}
		return ic.model, nil, true

	case key.Matches(msg, ic.keys.ClearMessages):
		// Clear system messages - create new controller instance
		if ic.model.systemController != nil && ic.model.systemController.HasMessages() {
			// For now, just create a new controller instance to clear messages
			ic.model.systemController = system.NewController(100)
			// Force immediate re-render
			return ic.model, tea.ClearScreen, true
		}
		return ic.model, nil, true
	}

	// Handle number keys for view switching
	for viewType, cfg := range ic.viewConfigs {
		if msg.String() == cfg.KeyBinding {
			// Skip MCP connections view if not in debug mode
			if viewType == ViewMCPConnections && !ic.debugMode {
				continue
			}
			ic.model.switchToView(viewType)
			return ic.model, nil, true
		}
	}

	return ic.model, nil, false // Key not handled
}

// handleEnter handles the Enter key based on current view
func (ic *InputController) handleEnter() tea.Cmd {
	switch ic.model.currentView() {
	case ViewProcesses:
		if i, ok := ic.model.processViewController.GetProcessesList().SelectedItem().(processItem); ok {
			ic.model.selectedProcess = i.process.ID
			ic.model.navController.SwitchTo(ViewLogs)
			ic.model.updateLogsView()
		}
	}

	return nil
}

// handleCommandWindow handles input when command window is open
func (ic *InputController) handleCommandWindow(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	commandAutocomplete := ic.model.commandWindowController.GetCommandAutocomplete()

	switch msg.String() {
	case "esc":
		ic.model.commandWindowController.HideCommandWindow()
		return ic.model, nil, true

	case "backspace":
		if commandAutocomplete.Value() == "" {
			ic.model.commandWindowController.HideCommandWindow()
			return ic.model, nil, true
		}

	case "enter":
		// If there are suggestions available, apply the selected one first
		if len(commandAutocomplete.GetSuggestions()) > 0 {
			// Simulate a tab key press to apply the selected suggestion
			tabMsg := tea.KeyMsg{Type: tea.KeyTab}
			*commandAutocomplete, _ = commandAutocomplete.Update(tabMsg)
		}

		// Validate the command first
		if valid, errMsg := commandAutocomplete.ValidateInput(); !valid {
			// Set error message in the autocomplete component
			commandAutocomplete.SetError(errMsg)
			return ic.model, nil, true
		}

		// Execute the command
		value := commandAutocomplete.Value()
		ic.model.commandWindowController.HideCommandWindow()

		// Handle the command through slash command handler
		ic.model.handleSlashCommand(value)
		return ic.model, nil, true
	}

	// Update the autocomplete component
	newAuto, cmd := commandAutocomplete.Update(msg)
	*commandAutocomplete = newAuto

	return ic.model, cmd, true
}
