package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// DialogMessageHandler handles dialog-related messages and interactions
type DialogMessageHandler struct{}

// NewDialogMessageHandler creates a new dialog message handler
func NewDialogMessageHandler() MessageHandler {
	return &DialogMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *DialogMessageHandler) CanHandle(msg tea.Msg) bool {
	// This handler has special logic - it checks dialog state in HandleMessage
	// and only processes when dialogs are active
	return true
}

// HandleMessage processes dialog-related messages
func (h *DialogMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle run dialog updates
	if model.commandWindowController.IsShowingRunDialog() {
		// Handle escape key to close dialog
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Back) {
			model.commandWindowController.HideRunDialog()
			return model, nil
		}

		// Handle enter key to run command
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Enter) {
			cmds = append(cmds, model.handleRunCommand())
			return model, tea.Batch(cmds...)
		}

		// Update the commands list
		commandsList := model.commandWindowController.GetCommandsList()
		newList, cmd := commandsList.Update(msg)
		*commandsList = newList
		cmds = append(cmds, cmd)

		return model, tea.Batch(cmds...)
	}

	// Handle custom command dialog
	if model.commandWindowController.IsShowingCustomCommand() {
		// Handle escape key to close dialog
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Back) {
			model.commandWindowController.HideCustomCommandDialog()
			return model, nil
		}

		// Handle enter key to run command
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Enter) {
			command := strings.TrimSpace(model.commandWindowController.GetCustomCommandInput().Value())
			if command != "" {
				model.commandWindowController.HideCustomCommandDialog()
				// Parse the command and arguments
				parts := strings.Fields(command)
				if len(parts) > 0 {
					cmdName := parts[0]
					args := parts[1:]
					// Create error handler for consistent error handling
					errorHandler := NewStandardErrorHandler(model.logStore, model.updateChan)
					SafeGoroutine(
						fmt.Sprintf("start custom command '%s'", cmdName),
						func() error {
							_, err := model.processMgr.StartCommand(command, cmdName, args)
							return err
						},
						func(err error) {
							ctx := CommandExecutionContext(cmdName, "Dialog", model.logStore, model.updateChan)
							errorHandler.HandleError(err, ctx)
						},
					)
					model.navController.SwitchTo(ViewProcesses)
					model.updateProcessList()
					return model, model.waitForUpdates()
				}
			}
			return model, nil
		}

		// Update the text input
		customInput := model.commandWindowController.GetCustomCommandInput()
		newInput, cmd := customInput.Update(msg)
		*customInput = newInput
		cmds = append(cmds, cmd)

		return model, tea.Batch(cmds...)
	}

	// If no dialogs are active, don't handle the message
	return model, nil
}
