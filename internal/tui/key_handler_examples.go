package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Example: How to properly register view-specific key handlers
// This file demonstrates best practices for key handler registration

// RegisterViewSpecificHandlers shows how to register handlers for different views
// that may use the same keys for different purposes
func RegisterViewSpecificHandlers(registry *KeyHandlerRegistry) {
	// LOGS VIEW: 'c' = clear logs, 'f' = filter, 'x' = export
	registry.Register(NewViewSpecificKeyHandler(ViewLogs, func(msg tea.KeyMsg, model *Model) tea.Cmd {
		switch msg.String() {
		case "c":
			return func() tea.Msg {
				model.logStore.ClearLogs()
				return StatusMsg{Message: "Logs cleared"}
			}
		case "f":
			return func() tea.Msg {
				// Open filter dialog
				return StatusMsg{Message: "Opening filter dialog"}
			}
		case "x":
			return func() tea.Msg {
				// Export logs
				return StatusMsg{Message: "Exporting logs..."}
			}
		}
		return nil
	}))

	// ERRORS VIEW: 'c' = copy error, 'd' = dismiss error, 'x' = clear all
	registry.Register(NewViewSpecificKeyHandler(ViewErrors, func(msg tea.KeyMsg, model *Model) tea.Cmd {
		switch msg.String() {
		case "c":
			return func() tea.Msg {
				// Copy error to clipboard
				return StatusMsg{Message: "Error copied to clipboard"}
			}
		case "d":
			return func() tea.Msg {
				// Dismiss current error
				return StatusMsg{Message: "Error dismissed"}
			}
		case "x":
			return func() tea.Msg {
				model.logStore.ClearErrors()
				return StatusMsg{Message: "All errors cleared"}
			}
		}
		return nil
	}))

	// PROCESSES VIEW: 's' = stop, 'r' = restart, 'l' = view logs
	registry.Register(NewViewSpecificKeyHandler(ViewProcesses, func(msg tea.KeyMsg, model *Model) tea.Cmd {
		switch msg.String() {
		case "s":
			return func() tea.Msg {
				// Stop selected process
				return StatusMsg{Message: "Stopping processes..."}
			}
		case "r":
			return func() tea.Msg {
				// Restart selected process
				return StatusMsg{Message: "Restarting processes..."}
			}
		case "l":
			return func() tea.Msg {
				// Switch to logs view for selected process
				model.switchToView(ViewLogs)
				return StatusMsg{Message: "Viewing process logs"}
			}
		}
		return nil
	}))

	// WEB VIEW: 'f' = filter requests, 'c' = clear, 'i' = inspect
	registry.Register(NewViewSpecificKeyHandler(ViewWeb, func(msg tea.KeyMsg, model *Model) tea.Cmd {
		switch msg.String() {
		case "f":
			return func() tea.Msg {
				// Cycle through filters
				return StatusMsg{Message: "Cycling web request filters"}
			}
		case "c":
			return func() tea.Msg {
				// Clear captured requests
				return StatusMsg{Message: "Web requests cleared"}
			}
		case "i":
			return func() tea.Msg {
				// Inspect selected request
				return StatusMsg{Message: "Inspecting request details"}
			}
		}
		return nil
	}))
}

// RegisterModalHandlers shows how to register handlers for modal/overlay contexts
// These have higher priority and override view-specific handlers
func RegisterModalHandlers(registry *KeyHandlerRegistry) {
	// Command Palette Handler (Priority 50 - very high)
	registry.Register(&CommandPaletteHandler{})

	// Dialog Handler (Priority 75 - high)
	registry.Register(&DialogHandler{})

	// File Browser Handler (Priority 80 - high)
	registry.Register(&FileBrowserHandler{})
}

// CommandPaletteHandler handles keys when command palette is open
type CommandPaletteHandler struct{}

func (h *CommandPaletteHandler) Name() string     { return "CommandPalette" }
func (h *CommandPaletteHandler) Priority() int    { return 50 }

func (h *CommandPaletteHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	// Only handle if command palette is open
	return ctx.Context == "command_palette"
}

func (h *CommandPaletteHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		// Close command palette
		return func() tea.Msg {
			model.masterData.CommandWindowState = model.masterData.CommandWindowState.HideCommandWindow()
			return nil
		}
	case tea.KeyEnter:
		// Execute selected command
		return func() tea.Msg {
			// Execute command logic
			return StatusMsg{Message: "Executing command..."}
		}
	case tea.KeyUp, tea.KeyDown:
		// Navigate command list
		// Let the command window component handle it
		return nil
	}
	return nil
}

// DialogHandler handles keys when any dialog is open
type DialogHandler struct{}

func (h *DialogHandler) Name() string     { return "Dialog" }
func (h *DialogHandler) Priority() int    { return 75 }

func (h *DialogHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	return ctx.Context == "modal" || ctx.Context == "dialog"
}

func (h *DialogHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		// Close dialog
		return func() tea.Msg {
			return StatusMsg{Message: "Dialog closed"}
		}
	case tea.KeyEnter:
		// Confirm dialog action
		return func() tea.Msg {
			return StatusMsg{Message: "Dialog confirmed"}
		}
	}
	return nil
}

// FileBrowserHandler handles keys when file browser is open
type FileBrowserHandler struct{}

func (h *FileBrowserHandler) Name() string     { return "FileBrowser" }
func (h *FileBrowserHandler) Priority() int    { return 80 }

func (h *FileBrowserHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	return ctx.Context == "file_browser"
}

func (h *FileBrowserHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	switch key.String() {
	case "enter":
		// Select file/directory
		return func() tea.Msg {
			return StatusMsg{Message: "File selected"}
		}
	case "esc":
		// Close file browser
		return func() tea.Msg {
			return StatusMsg{Message: "File browser closed"}
		}
	case "n":
		// New file
		return func() tea.Msg {
			return StatusMsg{Message: "Creating new file..."}
		}
	case "d":
		// Delete file
		return func() tea.Msg {
			return StatusMsg{Message: "Delete file confirmation..."}
		}
	}
	return nil
}


// Priority Guidelines (from documentation):
// 0-99:     Emergency/System (force quit, panic recovery)
// 100-199:  Global Navigation (number keys, Tab, view switching)
// 200-299:  View-Specific Actions (Enter in process view, arrows in logs)
// 300-399:  Overlays/Modals (command palette, dialogs)
// 400-499:  Context Menus (right-click menus, dropdowns)
// 500-599:  Input Fields (text input, form fields)
// 1000+:    Fallback/Default (quit handler, help)

// Best Practices:
// 1. Always check the context (view, modal state, etc.) in CanHandle
// 2. Use priority to ensure proper precedence (modals > views > global)
// 3. Return nil from Handle if you don't want to trigger an update
// 4. Use descriptive names for debugging
// 5. Keep handlers focused - one handler per logical group of keys
// 6. Document what keys each handler processes