package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
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
				return messages.ClearLogsMsg{}
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
		case "up", "k":
			return func() tea.Msg {
				return messages.ScrollLogsUpMsg{}
			}
		case "down", "j":
			return func() tea.Msg {
				return messages.ScrollLogsDownMsg{}
			}
		case " ", "space":
			return func() tea.Msg {
				return messages.ToggleAutoScrollMsg{}
			}
		}
		return nil
	}))

	// ERRORS VIEW: 'c' = clear errors, 'd' = dismiss error, 'x' = clear all (alternative), 'a' = send to AI Coder
	registry.Register(NewViewSpecificKeyHandler(ViewErrors, func(msg tea.KeyMsg, model *Model) tea.Cmd {
		switch msg.String() {
		case "c":
			return func() tea.Msg {
				return messages.ClearErrorsMsg{}
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
		case "a":
			// Send selected error to AI Coder
			if model.masterData != nil && model.masterData.ErrorsState != nil {
				selectedError := model.masterData.ErrorsState.GetSelectedError()
				if selectedError != nil {
					// Format error for AI Coder
					content := fmt.Sprintf("Error Type: %s\n", selectedError.Type)
					content += fmt.Sprintf("Process: %s\n", selectedError.ProcessName)
					content += fmt.Sprintf("Timestamp: %s\n\n", selectedError.Timestamp.Format("2006-01-02 15:04:05"))
					content += fmt.Sprintf("Message:\n%s\n\n", selectedError.Message)

					if len(selectedError.Stack) > 0 {
						content += "Stack Trace:\n"
						for _, line := range selectedError.Stack {
							content += fmt.Sprintf("  %s\n", line)
						}
						content += "\n"
					}

					if len(selectedError.Context) > 0 {
						content += "Context:\n"
						for _, line := range selectedError.Context {
							content += fmt.Sprintf("  %s\n", line)
						}
					}

					title := fmt.Sprintf("%s in %s", selectedError.Type, selectedError.ProcessName)

					return func() tea.Msg {
						return messages.SendToAICoderMsg{
							ContentType: "error",
							Content:     content,
							Title:       title,
							SessionID:   "", // Empty = use active session
						}
					}
				}
			}
			return func() tea.Msg {
				return StatusMsg{Message: "No error selected"}
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

	// WEB VIEW: 'f' = filter requests, 'c' = clear, arrow keys = navigate, Enter = details, 'a' = send to AI Coder
	registry.Register(NewViewSpecificKeyHandler(ViewWeb, func(msg tea.KeyMsg, model *Model) tea.Cmd {
		switch msg.String() {
		case "f":
			return func() tea.Msg {
				// Cycle through filters - implement proper filter cycling
				if model.masterData != nil && model.masterData.WebViewState != nil {
					currentFilter := model.masterData.WebViewState.GetFilter()
					var nextFilter string
					switch currentFilter {
					case "all":
						nextFilter = "2xx"
					case "2xx":
						nextFilter = "3xx"
					case "3xx":
						nextFilter = "4xx"
					case "4xx":
						nextFilter = "5xx"
					case "5xx":
						nextFilter = "error"
					case "error":
						nextFilter = "all"
					default:
						nextFilter = "all"
					}
					model.masterData.WebViewState = model.masterData.WebViewState.SetFilter(nextFilter)
					return StatusMsg{Message: fmt.Sprintf("Filter: %s", nextFilter)}
				}
				return StatusMsg{Message: "Cycling web request filters"}
			}
		case "c":
			return func() tea.Msg {
				return messages.ClearWebMsg{}
			}
		case "up", "k":
			return func() tea.Msg {
				return messages.NavigateWebUpMsg{}
			}
		case "down", "j":
			return func() tea.Msg {
				return messages.NavigateWebDownMsg{}
			}
		case "enter", " ":
			return func() tea.Msg {
				return messages.ToggleWebDetailMsg{}
			}
		case "i":
			return func() tea.Msg {
				// Inspect selected request - same as Enter
				return messages.ToggleWebDetailMsg{}
			}
		case "a":
			// Send selected proxy request to AI Coder
			if model.masterData != nil && model.masterData.WebViewState != nil {
				selectedRequest := model.masterData.WebViewState.GetSelectedRequest()
				if selectedRequest != nil {
					// Format proxy request for AI Coder
					content := fmt.Sprintf("HTTP %s %s\n", selectedRequest.Method, selectedRequest.URL)
					content += fmt.Sprintf("Status: %d\n", selectedRequest.StatusCode)
					content += fmt.Sprintf("Duration: %v\n", selectedRequest.Duration)
					content += fmt.Sprintf("Process: %s\n", selectedRequest.ProcessName)
					content += fmt.Sprintf("Timestamp: %s\n", selectedRequest.Timestamp.Format("2006-01-02 15:04:05"))

					if selectedRequest.Size > 0 {
						content += fmt.Sprintf("Size: %d bytes\n", selectedRequest.Size)
					}

					if selectedRequest.Error != "" {
						content += fmt.Sprintf("\nError: %s\n", selectedRequest.Error)
					}

					if selectedRequest.HasAuth {
						content += fmt.Sprintf("\nAuth Type: %s\n", selectedRequest.AuthType)
						if selectedRequest.JWTError != "" {
							content += fmt.Sprintf("JWT Error: %s\n", selectedRequest.JWTError)
						}
					}

					if len(selectedRequest.PerformanceMetrics) > 0 {
						content += "\nPerformance Metrics:\n"
						for key, value := range selectedRequest.PerformanceMetrics {
							content += fmt.Sprintf("  %s: %v\n", key, value)
						}
					}

					title := fmt.Sprintf("%s %s (%d)", selectedRequest.Method, selectedRequest.Path, selectedRequest.StatusCode)

					return func() tea.Msg {
						return messages.SendToAICoderMsg{
							ContentType: "proxy_request",
							Content:     content,
							Title:       title,
							SessionID:   "", // Empty = use active session
						}
					}
				}
			}
			return func() tea.Msg {
				return StatusMsg{Message: "No proxy request selected"}
			}
		}
		return nil
	}))

	// CLEAR CONFIRMATION HANDLER (T026) - Works in Errors/Logs/Web views
	registry.Register(NewViewSpecificKeyHandler(ViewErrors, func(msg tea.KeyMsg, model *Model) tea.Cmd {
		switch msg.String() {
		case "C":
			return func() tea.Msg {
				return messages.ShowClearConfirmationMsg{Tab: "errors"}
			}
		}
		return nil
	}))

	registry.Register(NewViewSpecificKeyHandler(ViewLogs, func(msg tea.KeyMsg, model *Model) tea.Cmd {
		switch msg.String() {
		case "C":
			return func() tea.Msg {
				return messages.ShowClearConfirmationMsg{Tab: "logs"}
			}
		}
		return nil
	}))

	registry.Register(NewViewSpecificKeyHandler(ViewWeb, func(msg tea.KeyMsg, model *Model) tea.Cmd {
		switch msg.String() {
		case "C":
			return func() tea.Msg {
				return messages.ShowClearConfirmationMsg{Tab: "web"}
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