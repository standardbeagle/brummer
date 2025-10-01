package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// IntegratedShell is the top-level component that routes messages to child components.
// It manages component lifecycle, focus, and message routing.
//
// Architecture:
// - Receives all messages from BubbleTea
// - Routes messages to appropriate components based on type
// - Manages which component is currently active/focused
// - Handles component switching and layout
type IntegratedShell struct {
	width  int
	height int

	// View Components (nil until initialized)
	logsView     Component
	processView  Component
	urlsView     Component
	errorsView   Component
	webView      Component
	aicodersView Component
	mcpView      Component
	settingsView Component

	// Chrome Components (optional)
	tabsComponent         Component
	headerComponent       Component
	footerComponent       Component
	notificationComponent Component

	// Overlay Components (optional)
	commandInput Component // Slash command input (activated with "/")
	helpOverlay  Component

	// Command system
	commandRegistry *CommandRegistry
	commandExecutor *CommandExecutor

	// State
	currentView string // "logs", "processes", "urls", "errors", "web", "aicoders", "mcp", "settings", etc.
}

// NewIntegratedShell creates a new shell with no components
func NewIntegratedShell() *IntegratedShell {
	return &IntegratedShell{
		currentView:     "logs",
		commandRegistry: NewCommandRegistry(),
	}
}

// GetCommandRegistry returns the command registry for registering service commands
func (s *IntegratedShell) GetCommandRegistry() *CommandRegistry {
	return s.commandRegistry
}

// SetCommandExecutor sets the command executor for handling CommandExecuteMsg
func (s *IntegratedShell) SetCommandExecutor(executor *CommandExecutor) {
	s.commandExecutor = executor
}

// SetLogsView sets the logs component
func (s *IntegratedShell) SetLogsView(component Component) {
	s.logsView = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// SetProcessView sets the process component
func (s *IntegratedShell) SetProcessView(component Component) {
	s.processView = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// SetURLsView sets the URLs component
func (s *IntegratedShell) SetURLsView(component Component) {
	s.urlsView = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// SetErrorsView sets the errors component
func (s *IntegratedShell) SetErrorsView(component Component) {
	s.errorsView = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// SetWebView sets the web/proxy component
func (s *IntegratedShell) SetWebView(component Component) {
	s.webView = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// SetAICodersView sets the AI coders component
func (s *IntegratedShell) SetAICodersView(component Component) {
	s.aicodersView = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// SetMCPView sets the MCP status component
func (s *IntegratedShell) SetMCPView(component Component) {
	s.mcpView = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// SetSettingsView sets the settings component
func (s *IntegratedShell) SetSettingsView(component Component) {
	s.settingsView = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// SetCommandInput sets the command input overlay component
func (s *IntegratedShell) SetCommandInput(component Component) {
	s.commandInput = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// SetHelpOverlay sets the help overlay component
func (s *IntegratedShell) SetHelpOverlay(component Component) {
	s.helpOverlay = component
	if s.width > 0 && s.height > 0 {
		component.SetSize(s.width, s.height)
	}
}

// Init initializes the shell
func (s *IntegratedShell) Init() tea.Cmd {
	var cmds []tea.Cmd

	if s.logsView != nil {
		cmds = append(cmds, s.logsView.Init())
	}
	if s.processView != nil {
		cmds = append(cmds, s.processView.Init())
	}
	if s.urlsView != nil {
		cmds = append(cmds, s.urlsView.Init())
	}
	if s.errorsView != nil {
		cmds = append(cmds, s.errorsView.Init())
	}
	if s.webView != nil {
		cmds = append(cmds, s.webView.Init())
	}
	if s.aicodersView != nil {
		cmds = append(cmds, s.aicodersView.Init())
	}
	if s.mcpView != nil {
		cmds = append(cmds, s.mcpView.Init())
	}
	if s.settingsView != nil {
		cmds = append(cmds, s.settingsView.Init())
	}

	// Initialize overlays
	if s.commandInput != nil {
		cmds = append(cmds, s.commandInput.Init())
	}
	if s.helpOverlay != nil {
		cmds = append(cmds, s.helpOverlay.Init())
	}

	return tea.Batch(cmds...)
}

// Update handles messages and routes them to components
func (s *IntegratedShell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update shell size
		s.width = msg.Width
		s.height = msg.Height

		// Propagate size to all components
		if s.logsView != nil {
			s.logsView.SetSize(msg.Width, msg.Height)
		}
		if s.processView != nil {
			s.processView.SetSize(msg.Width, msg.Height)
		}
		if s.urlsView != nil {
			s.urlsView.SetSize(msg.Width, msg.Height)
		}
		if s.errorsView != nil {
			s.errorsView.SetSize(msg.Width, msg.Height)
		}
		if s.webView != nil {
			s.webView.SetSize(msg.Width, msg.Height)
		}
		if s.aicodersView != nil {
			s.aicodersView.SetSize(msg.Width, msg.Height)
		}
		if s.mcpView != nil {
			s.mcpView.SetSize(msg.Width, msg.Height)
		}
		if s.settingsView != nil {
			s.settingsView.SetSize(msg.Width, msg.Height)
		}

		// Propagate size to overlays
		if s.commandInput != nil {
			s.commandInput.SetSize(msg.Width, msg.Height)
		}
		if s.helpOverlay != nil {
			s.helpOverlay.SetSize(msg.Width, msg.Height)
		}

	case messages.LogAddedMsg:
		// Route log messages to logs component
		if s.logsView != nil {
			_, cmd := s.logsView.Update(msg)
			cmds = append(cmds, cmd)
		}
		// Also route to errors view (filters IsError internally)
		if s.errorsView != nil {
			_, cmd := s.errorsView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case messages.ProcessStartedMsg:
		// Route to process component
		if s.processView != nil {
			_, cmd := s.processView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case messages.ProcessStoppedMsg:
		// Route to process component
		if s.processView != nil {
			_, cmd := s.processView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case messages.ProcessStateChangedMsg:
		// Route to process component
		if s.processView != nil {
			_, cmd := s.processView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case messages.URLDetectedMsg:
		// Route to URLs component
		if s.urlsView != nil {
			_, cmd := s.urlsView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case messages.ProxyRequestMsg:
		// Route to web component
		if s.webView != nil {
			_, cmd := s.webView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case messages.CommandExecuteMsg:
		// Command execution - route to executor
		if s.commandExecutor != nil {
			cmd := s.commandExecutor.Execute(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			// No executor configured, pass message through for parent to handle
			return s, func() tea.Msg { return msg }
		}

	case tea.KeyMsg:
		// Check if overlays are active and should handle the key first
		if s.helpOverlay != nil {
			// Type assert to check if help overlay is active
			if helpComponent, ok := s.helpOverlay.(*HelpOverlayComponent); ok && helpComponent.IsActive() {
				_, cmd := s.helpOverlay.Update(msg)
				cmds = append(cmds, cmd)
				return s, tea.Batch(cmds...)
			}
		}
		if s.commandInput != nil {
			// Type assert to check if command input is active
			if cmdComponent, ok := s.commandInput.(*CommandInputComponent); ok && cmdComponent.IsActive() {
				_, cmd := s.commandInput.Update(msg)
				cmds = append(cmds, cmd)
				return s, tea.Batch(cmds...)
			}
		}

		// Handle global keys (component switching, overlay activation, etc.)
		switch msg.String() {
		case "ctrl+c", "q":
			// Quit handled by parent
		case "/":
			// Open command input
			if s.commandInput != nil {
				if cmdComponent, ok := s.commandInput.(*CommandInputComponent); ok {
					cmdComponent.Open()
				}
			}
			return s, nil
		case "?":
			// Open help overlay
			if s.helpOverlay != nil {
				if helpComponent, ok := s.helpOverlay.(*HelpOverlayComponent); ok {
					helpComponent.Open()
				}
			}
			return s, nil
		case "1":
			// Switch to logs view
			s.currentView = "logs"
			return s, nil
		case "2":
			// Switch to processes view
			if s.processView != nil {
				s.currentView = "processes"
			}
			return s, nil
		case "3":
			// Switch to URLs view
			if s.urlsView != nil {
				s.currentView = "urls"
			}
			return s, nil
		case "4":
			// Switch to Errors view
			if s.errorsView != nil {
				s.currentView = "errors"
			}
			return s, nil
		case "5":
			// Switch to Web view
			if s.webView != nil {
				s.currentView = "web"
			}
			return s, nil
		case "6":
			// Switch to AI Coders view
			if s.aicodersView != nil {
				s.currentView = "aicoders"
			}
			return s, nil
		case "7":
			// Switch to MCP view
			if s.mcpView != nil {
				s.currentView = "mcp"
			}
			return s, nil
		case "8":
			// Switch to Settings view
			if s.settingsView != nil {
				s.currentView = "settings"
			}
			return s, nil
		default:
			// Forward to active component
			switch s.currentView {
			case "logs":
				if s.logsView != nil {
					_, cmd := s.logsView.Update(msg)
					cmds = append(cmds, cmd)
				}
			case "processes":
				if s.processView != nil {
					_, cmd := s.processView.Update(msg)
					cmds = append(cmds, cmd)
				}
			case "urls":
				if s.urlsView != nil {
					_, cmd := s.urlsView.Update(msg)
					cmds = append(cmds, cmd)
				}
			case "errors":
				if s.errorsView != nil {
					_, cmd := s.errorsView.Update(msg)
					cmds = append(cmds, cmd)
				}
			case "web":
				if s.webView != nil {
					_, cmd := s.webView.Update(msg)
					cmds = append(cmds, cmd)
				}
			case "aicoders":
				if s.aicodersView != nil {
					_, cmd := s.aicodersView.Update(msg)
					cmds = append(cmds, cmd)
				}
			case "mcp":
				if s.mcpView != nil {
					_, cmd := s.mcpView.Update(msg)
					cmds = append(cmds, cmd)
				}
			case "settings":
				if s.settingsView != nil {
					_, cmd := s.settingsView.Update(msg)
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	return s, tea.Batch(cmds...)
}

// View renders the current component
func (s *IntegratedShell) View() string {
	// Render base view
	var baseView string
	switch s.currentView {
	case "logs":
		if s.logsView != nil {
			baseView = s.logsView.View()
		} else {
			baseView = "Logs view not initialized"
		}

	case "processes":
		if s.processView != nil {
			baseView = s.processView.View()
		} else {
			baseView = "Process view not initialized"
		}

	case "urls":
		if s.urlsView != nil {
			baseView = s.urlsView.View()
		} else {
			baseView = "URLs view not initialized"
		}

	case "errors":
		if s.errorsView != nil {
			baseView = s.errorsView.View()
		} else {
			baseView = "Errors view not initialized"
		}

	case "web":
		if s.webView != nil {
			baseView = s.webView.View()
		} else {
			baseView = "Web view not initialized"
		}

	case "aicoders":
		if s.aicodersView != nil {
			baseView = s.aicodersView.View()
		} else {
			baseView = "AI Coders view not initialized"
		}

	case "mcp":
		if s.mcpView != nil {
			baseView = s.mcpView.View()
		} else {
			baseView = "MCP view not initialized"
		}

	case "settings":
		if s.settingsView != nil {
			baseView = s.settingsView.View()
		} else {
			baseView = "Settings view not initialized"
		}

	default:
		baseView = "Unknown view: " + s.currentView
	}

	// Render overlays on top (command input and help have higher priority)
	// Command input renders on top when active
	if s.commandInput != nil {
		if cmdComponent, ok := s.commandInput.(*CommandInputComponent); ok && cmdComponent.IsActive() {
			overlayView := s.commandInput.View()
			if overlayView != "" {
				// Overlay is already centered by the component
				return overlayView
			}
		}
	}

	// Help overlay renders on top
	if s.helpOverlay != nil {
		if helpComponent, ok := s.helpOverlay.(*HelpOverlayComponent); ok && helpComponent.IsActive() {
			overlayView := s.helpOverlay.View()
			if overlayView != "" {
				// Center the overlay on the screen
				return s.centerOverlay(baseView, overlayView)
			}
		}
	}

	return baseView
}

// centerOverlay centers an overlay on top of the base view
func (s *IntegratedShell) centerOverlay(baseView, overlayView string) string {
	// For now, just render overlay on top of base
	// In a more sophisticated implementation, we could:
	// - Parse base view into lines
	// - Calculate overlay dimensions
	// - Position overlay in center
	// - Combine base and overlay with proper positioning

	// Simple implementation: just show the overlay
	// The overlay components handle their own centering via lipgloss
	return overlayView
}