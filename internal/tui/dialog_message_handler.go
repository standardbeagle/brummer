package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/commands"
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
	if model.commandWindowView.IsShowingRunDialog() {
		// Handle escape key to close dialog
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Back) {
			model.commandWindowView.HideRunDialog()
			return model, nil
		}

		// Handle enter key to run command
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Enter) {
			cmds = append(cmds, model.handleRunCommand())
			return model, tea.Batch(cmds...)
		}

		// Update the commands list
		commandsList := model.commandWindowView.GetCommandsList()
		newList, cmd := commandsList.Update(msg)
		*commandsList = newList
		cmds = append(cmds, cmd)

		return model, tea.Batch(cmds...)
	}

	// Handle custom command dialog
	if model.commandWindowView.IsShowingCustomCommand() {
		// Handle escape key to close dialog
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Back) {
			model.commandWindowView.HideCustomCommandDialog()
			return model, nil
		}

		// Handle enter key to run command
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Enter) {
			command := strings.TrimSpace(model.commandWindowView.GetCustomCommandInput().Value())
			if command != "" {
				model.commandWindowView.HideCustomCommandDialog()
				// Parse the command and arguments
				parts := strings.Fields(command)
				if len(parts) > 0 {
					cmdName := parts[0]
					// Create async command for custom command execution
					safeCommands := commands.NewSafeOperationCommands()
					asyncCmd := safeCommands.ExecuteWithPanicRecovery(
						fmt.Sprintf("custom-command-%s", cmdName),
						fmt.Sprintf("start custom command '%s'", cmdName),
						false, // shouldFail = false
						model,
					)
					cmds = append(cmds, asyncCmd)
					model.switchToView(ViewProcesses)
					model.updateProcessList()
					return model, model.waitForUpdates()
				}
			}
			return model, nil
		}

		// Update the text input
		customInput := model.commandWindowView.GetCustomCommandInput()
		newInput, cmd := customInput.Update(msg)
		*customInput = newInput
		cmds = append(cmds, cmd)

		return model, tea.Batch(cmds...)
	}

	// If no dialogs are active, don't handle the message
	return model, nil
}
