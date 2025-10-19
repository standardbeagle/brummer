package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/messages"
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

	// Handle command execution messages
	if cmdMsg, ok := msg.(messages.CommandExecuteMsg); ok {
		// Close command window if open
		if model.masterData.CommandWindowState.IsShowingCommandWindow() {
			model.masterData.CommandWindowState = model.masterData.CommandWindowState.HideCommandWindow()
		}

		// Execute the command using the model's command handling
		commandStr := "/" + cmdMsg.CommandName
		if len(cmdMsg.Arguments) > 0 {
			commandStr += " " + strings.Join(cmdMsg.Arguments, " ")
		}
		cmd := model.handleSlashCommand(commandStr)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		return model, tea.Batch(cmds...)
	}

	// Handle command window (slash command palette)
	if model.masterData.CommandWindowState.IsShowingCommandWindow() {
		// Handle escape key to close command window
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Back) {
			model.masterData.CommandWindowState = model.masterData.CommandWindowState.HideCommandWindow()
			return model, nil
		}

		// Update the command autocomplete component through the view
		if model.commandWindowView != nil {
			newModel, cmd := model.commandWindowView.Update(msg)
			if newCmdModel, ok := newModel.(*components.CommandWindowViewComponentNew); ok {
				model.commandWindowView = newCmdModel
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		return model, tea.Batch(cmds...)
	}

	// Handle run dialog updates
	if model.masterData.CommandWindowState.IsShowingRunDialog() {
		// Handle escape key to close dialog
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Back) {
			model.masterData.CommandWindowState = model.masterData.CommandWindowState.HideRunDialog()
			return model, nil
		}

		// Handle enter key to run command
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Enter) {
			cmds = append(cmds, model.handleRunCommand())
			return model, tea.Batch(cmds...)
		}

		// Update the commands list through the view
		if model.commandWindowView != nil {
			newModel, cmd := model.commandWindowView.Update(msg)
			if newCmdModel, ok := newModel.(*components.CommandWindowViewComponentNew); ok {
				model.commandWindowView = newCmdModel
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		return model, tea.Batch(cmds...)
	}

	// Handle custom command dialog
	if model.masterData.CommandWindowState.IsShowingCustomCommand() {
		// Handle escape key to close dialog
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Back) {
			model.masterData.CommandWindowState = model.masterData.CommandWindowState.HideCustomCommandDialog()
			return model, nil
		}

		// Handle enter key to run command
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, model.keys.Enter) {
			command := strings.TrimSpace(model.masterData.CommandWindowState.GetCustomCommandText())
			if command != "" {
				model.masterData.CommandWindowState = model.masterData.CommandWindowState.HideCustomCommandDialog()
				// Parse the command and arguments
				parts := strings.Fields(command)
				if len(parts) > 0 {
					cmdName := parts[0]
					cmdArgs := parts[1:]

					// Start the custom command using ProcessManager
					cmd := func() tea.Msg {
						process, err := model.processMgr.StartCommand(
							cmdName,
							cmdName,
							cmdArgs,
						)
						if err != nil {
							model.logStore.Add("system", "System",
								fmt.Sprintf("❌ Failed to start custom command '%s': %v", cmdName, err), true)
							return nil
						}

						model.logStore.Add("system", "System",
							fmt.Sprintf("✅ Started custom command '%s' (process ID: %s)", cmdName, process.ID), false)
						return nil
					}

					cmds = append(cmds, cmd)
					model.switchToView(ViewProcesses)
					model.updateProcessList()
					return model, tea.Batch(cmds...)
				}
			}
			return model, nil
		}

		// Update the text input through the view
		if model.commandWindowView != nil {
			newModel, cmd := model.commandWindowView.Update(msg)
			if newCmdModel, ok := newModel.(*components.CommandWindowViewComponentNew); ok {
				model.commandWindowView = newCmdModel
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		return model, tea.Batch(cmds...)
	}

	// If no dialogs are active, don't handle the message
	return model, nil
}
