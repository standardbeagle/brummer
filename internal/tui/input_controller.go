package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/navigation"
	"github.com/standardbeagle/brummer/internal/tui/system"
)

// InputController handles all keyboard input and routing
type InputController struct {
	// Dependencies
	model       *Model // Will be refactored to use interface in future iteration
	keys        keyMap
	viewConfigs map[View]ViewConfig
	debugMode   bool

	// Hierarchical key handling system
	keyHandler *components.HierarchicalKeyHandler

	// Double-press quit tracking
	lastQuitPress time.Time
	quitTimeout   time.Duration
}

// NewInputController creates a new input controller
func NewInputController(model *Model, keys keyMap, viewConfigs map[View]ViewConfig, debugMode bool) *InputController {
	ic := &InputController{
		model:       model,
		keys:        keys,
		viewConfigs: viewConfigs,
		debugMode:   debugMode,
		quitTimeout: 2 * time.Second, // 2 second window for double Ctrl+C
		keyHandler:  components.NewHierarchicalKeyHandler(),
	}

	// Initialize hierarchical key handler context
	ic.setupHierarchicalKeyHandlers()

	return ic
}

// HandleKeyMsg processes keyboard input using the hierarchical key handler
func (ic *InputController) HandleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// Handle script selector view (special case - highest priority)
	if ic.model.currentView() == ViewScriptSelector && ic.model.scriptSelector != nil {
		handled, cmd := ic.model.scriptSelector.HandleKeyMsg(msg)
		if handled {
			return ic.model, cmd, true
		}
	}

	// Handle command window dialogs (highest priority after script selector)
	if ic.model.commandWindowView != nil {
		if ic.model.commandWindowView.IsShowingCommandWindow() {
			return ic.handleCommandWindow(msg)
		}
		if ic.model.commandWindowView.IsShowingRunDialog() {
			return ic.handleRunDialog(msg)
		}
		if ic.model.commandWindowView.IsShowingCustomCommand() {
			return ic.handleCustomCommandDialog(msg)
		}
	}

	// Update hierarchical context based on current state
	ic.UpdateHierarchicalContext()

	// Try the hierarchical key handler first
	newModel, cmd := ic.keyHandler.HandleKey(msg, ic.model)

	// Check if the key was handled by examining if a command was returned
	// The hierarchical handler returns (originalModel, nil) if not handled
	// and (handledModel, cmd) where cmd may be nil but the key was still handled
	if cmd != nil {
		// A command was returned, so the key was definitely handled
		ic.model = newModel.(*Model)
		return ic.model, cmd, true
	} else if newModel != ic.model {
		// Model changed, so the key was handled
		ic.model = newModel.(*Model)
		return ic.model, cmd, true
	} else {
		// Check if the key was handled by seeing if any handler could process it
		// We need to check if the key was handled even if no command was returned
		// and the model reference is the same (e.g., for navigation keys)

		// Check if this is a global key that should be handled
		keyStr := msg.String()
		globalKeys := []string{"q", "ctrl+c", "left", "right", "up", "down", "tab", "shift+tab", "?"}

		// Also check for number keys
		if len(msg.Runes) == 1 && msg.Runes[0] >= '1' && msg.Runes[0] <= '9' {
			// This is a number key (1-9)
			ic.model = newModel.(*Model)
			return ic.model, cmd, true
		}

		for _, globalKey := range globalKeys {
			if keyStr == globalKey {
				// This is a global key that should be handled
				// Update the model reference and mark as handled
				ic.model = newModel.(*Model)
				return ic.model, cmd, true
			}
		}
	}

	// Handle special cases not covered by hierarchical handler

	// Handle "/" key for Brummer commands - check if we should intercept it
	if msg.String() == "/" || (msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == '/') {
		// Check if we should intercept the slash command
		shouldIntercept := true
		if ic.model.currentView() == ViewAICoders && ic.model.aicodersView != nil {
			// TODO(Feature 005): Component should determine if slash should be intercepted
			// For now, always intercept "/" at start of line (standard behavior)
			shouldIntercept = true
		}

		if shouldIntercept {
			ic.model.showCommandWindow()
			return ic.model, nil, true
		}
		// If not intercepting, fall through to PTY handling
	}

	// Check if we're in AI Coders view - route input to the component
	// The component will handle both list and PTY modes
	if ic.model.currentView() == ViewAICoders && ic.model.aicodersView != nil {
		// Route all input to the AI Coder component when in AI Coders view
		// It will handle navigation in list mode and input forwarding in PTY mode
		updatedComponent, cmd := ic.model.aicodersView.Update(msg)
		ic.model.aicodersView = updatedComponent.(*components.AICodersViewComponent)
		// TODO(Feature 005): Determine when input is handled vs fall-through
		// For now, handle all keys when in AI Coders view
		return ic.model, cmd, true
	}

	// Handle view-specific keys not covered by hierarchical handler
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
		if !ic.model.commandWindowView.IsShowingRunDialog() {
			ic.model.showRunDialog()
		}
		return ic.model, nil, true

		// Note: Command window dialogs are handled at the top of HandleKeyMsg
		// for proper input routing when they are visible
	}

	return ic.model, nil, false
}

// handleGlobalKeys handles keys that work across all views
func (ic *InputController) handleGlobalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if key.Matches(msg, ic.keys.Quit) {
		return ic.handleQuit()
	}

	switch {
	case key.Matches(msg, ic.keys.Tab):
		ic.model.cycleView()
		return ic.model, nil, true

	case msg.String() == "shift+tab":
		ic.model.cyclePrevView()
		return ic.model, nil, true

	case key.Matches(msg, ic.keys.Left):
		ic.model.cyclePrevView()
		return ic.model, nil, true

	case key.Matches(msg, ic.keys.Right):
		ic.model.cycleView()
		return ic.model, nil, true

	// Fallback for string-based detection
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
			ic.model.logsView.ToggleHighPriority()
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
			// TODO: Implement system panel toggle when TabsComponent has the appropriate methods
			// Toggle system panel via layout controller
			// if ic.model.tabsComponent != nil {
			// 	current := ic.model.tabsComponent.GetSystemPanelHeight() > 0
			// 	ic.model.tabsComponent.SetSystemPanelOpen(!current)
			// }
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
		if i, ok := ic.model.processesView.GetProcessesList().SelectedItem().(processItem); ok {
			ic.model.selectedProcess = i.process.ID
			ic.model.navController.SwitchTo(ViewLogs)
			ic.model.updateLogsView()
		}
	}

	return nil
}

// handleCommandWindow handles input when command window is open
func (ic *InputController) handleCommandWindow(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	commandAutocomplete := ic.model.commandWindowView.GetCommandAutocomplete()

	switch msg.String() {
	case "esc":
		ic.model.commandWindowView.HideCommandWindow()
		return ic.model, nil, true

	case "backspace":
		if commandAutocomplete.Value() == "" {
			ic.model.commandWindowView.HideCommandWindow()
			return ic.model, nil, true
		}

	case "enter":
		// Validate the command first
		if valid, errMsg := commandAutocomplete.ValidateInput(); !valid {
			// Set error message in the autocomplete component
			commandAutocomplete.SetError(errMsg)
			return ic.model, nil, true
		}

		// Execute the command
		value := commandAutocomplete.Value()
		ic.model.commandWindowView.HideCommandWindow()

		// Handle the command through slash command handler
		ic.model.handleSlashCommand(value)
		return ic.model, nil, true
	}

	// Update the autocomplete component
	_, cmd := commandAutocomplete.Update(msg)

	return ic.model, cmd, true
}

// handleRunDialog handles input when run dialog is open
func (ic *InputController) handleRunDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		ic.model.commandWindowView.HideRunDialog()
		return ic.model, nil, true

	case "enter":
		cmd := ic.model.handleRunCommand()
		return ic.model, cmd, true
	}

	// Update the commands list
	commandsList := ic.model.commandWindowView.GetCommandsList()
	_, cmd := commandsList.Update(msg)

	return ic.model, cmd, true
}

// handleCustomCommandDialog handles input when custom command dialog is open
func (ic *InputController) handleCustomCommandDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	customCommandInput := ic.model.commandWindowView.GetCustomCommandInput()

	switch msg.String() {
	case "esc":
		ic.model.commandWindowView.HideCustomCommandDialog()
		return ic.model, nil, true

	case "enter":
		commandText := customCommandInput.Value()
		if commandText == "" {
			return ic.model, nil, true
		}

		ic.model.commandWindowView.HideCustomCommandDialog()

		// Execute the custom command
		errorHandler := NewStandardErrorHandler(ic.model.logStore, ic.model.updateChan)
		SafeGoroutine(
			fmt.Sprintf("start custom command '%s'", commandText),
			func() error {
				_, err := ic.model.processMgr.StartCommand("custom", commandText, nil)
				return err
			},
			func(err error) {
				ctx := ProcessStartContext("custom", "Custom Command Dialog", ic.model.logStore, ic.model.updateChan)
				errorHandler.HandleError(err, ctx)
			},
		)

		ic.model.navController.SwitchTo(ViewProcesses)
		ic.model.updateProcessList()
		return ic.model, ic.model.waitForUpdates(), true
	}

	// Update the custom command input
	_, cmd := customCommandInput.Update(msg)

	return ic.model, cmd, true
}

// handleQuit handles the quit functionality
func (ic *InputController) handleQuit() (tea.Model, tea.Cmd, bool) {
	now := time.Now()

	// Check if this is a 'q' key press (immediate quit)
	// We differentiate by checking if this is not a Ctrl+C or if enough time has passed
	if !ic.lastQuitPress.IsZero() && now.Sub(ic.lastQuitPress) <= ic.quitTimeout {
		// This is the second quit press within timeout - actually quit
		return ic.performQuit()
	} else {
		// Check if this was a 'q' key press (not Ctrl+C)
		// For now, we'll treat any quit as needing confirmation to be safe
		ic.lastQuitPress = now
		ic.model.logStore.Add("system", "System", "Press quit key again within 2 seconds to exit", false)
		return ic.model, nil, true
	}
}

// performQuit performs the actual quit operation
func (ic *InputController) performQuit() (tea.Model, tea.Cmd, bool) {
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
}

// setupHierarchicalKeyHandlers initializes the hierarchical key handler with global handlers
func (ic *InputController) setupHierarchicalKeyHandlers() {
	// Create global key handler for navigation and system-wide commands
	globalHandler := components.NewComponentKeyHandler("global", components.PriorityGlobal, ic.model)

	// Add global key bindings
	globalHandler.AddKeys(map[string]func(tea.KeyMsg) (tea.Model, tea.Cmd){
		"1": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(ViewProcesses)
			return ic.model, nil
		},
		"2": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(ViewLogs)
			return ic.model, nil
		},
		"3": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(ViewErrors)
			return ic.model, nil
		},
		"4": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(ViewURLs)
			return ic.model, nil
		},
		"5": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(ViewWeb)
			return ic.model, nil
		},
		"6": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(navigation.ViewTests)
			return ic.model, nil
		},
		"7": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(ViewAICoders)
			return ic.model, nil
		},
		"8": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(ViewSettings)
			return ic.model, nil
		},
		"9": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(ViewMCPConnections)
			return ic.model, nil
		},
		"tab": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.cycleView()
			return ic.model, nil
		},
		"shift+tab": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.cyclePrevView()
			return ic.model, nil
		},
		"left": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.cyclePrevView()
			return ic.model, nil
		},
		"right": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.cycleView()
			return ic.model, nil
		},
		"?": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			return ic.showHelpOverlay()
		},
		"q": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			return ic.handleQuitWithHierarchical()
		},
		"ctrl+c": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			return ic.handleQuitWithHierarchical()
		},
		"t": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			ic.model.switchToView(navigation.ViewTests)
			return ic.model, nil
		},
	})

	// Set up the initial context
	ctx := &components.KeyHandlerContext{
		GlobalHandlers: []components.KeyHandler{globalHandler},
	}

	ic.keyHandler.SetContext(ctx)
}

// handleQuitWithHierarchical handles quit using the hierarchical system
func (ic *InputController) handleQuitWithHierarchical() (tea.Model, tea.Cmd) {
	_, cmd, _ := ic.handleQuit()
	return ic.model, cmd
}

// showHelpOverlay shows the help overlay
func (ic *InputController) showHelpOverlay() (tea.Model, tea.Cmd) {
	// For now, add a help message to logs
	ic.model.logStore.Add("system", "System", "Help: Use number keys 1-9 to switch views, t for test view, / for commands, q/Ctrl+C to quit", false)
	return ic.model, nil
}

// UpdateHierarchicalContext updates the hierarchical key handler context based on current state
func (ic *InputController) UpdateHierarchicalContext() {
	// Get current context or create a new one
	ctx := ic.keyHandler.GetContext()
	if ctx == nil {
		ctx = &components.KeyHandlerContext{}
	}

	// For now, we only use global handlers
	// Overlays and focused components will be handled by the existing system
	// This provides a clean integration path without disrupting current functionality

	ic.keyHandler.SetContext(ctx)
}
