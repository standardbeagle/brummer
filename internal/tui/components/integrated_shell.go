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

	// Components (nil until initialized)
	logsView    Component
	processView Component
	urlsView    Component
	errorsView  Component
	webView     Component

	// State
	currentView string // "logs", "processes", "urls", "errors", "web", etc.
}

// NewIntegratedShell creates a new shell with no components
func NewIntegratedShell() *IntegratedShell {
	return &IntegratedShell{
		currentView: "logs",
	}
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

	case tea.KeyMsg:
		// Handle global keys (component switching, etc.)
		switch msg.String() {
		case "ctrl+c", "q":
			// Quit handled by parent
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
			}
		}
	}

	return s, tea.Batch(cmds...)
}

// View renders the current component
func (s *IntegratedShell) View() string {
	switch s.currentView {
	case "logs":
		if s.logsView != nil {
			return s.logsView.View()
		}
		return "Logs view not initialized"

	case "processes":
		if s.processView != nil {
			return s.processView.View()
		}
		return "Process view not initialized"

	case "urls":
		if s.urlsView != nil {
			return s.urlsView.View()
		}
		return "URLs view not initialized"

	case "errors":
		if s.errorsView != nil {
			return s.errorsView.View()
		}
		return "Errors view not initialized"

	case "web":
		if s.webView != nil {
			return s.webView.View()
		}
		return "Web view not initialized"

	default:
		return "Unknown view: " + s.currentView
	}
}