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
	logsView Component

	// State
	currentView string // "logs", "processes", "urls", etc.
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

// Init initializes the shell
func (s *IntegratedShell) Init() tea.Cmd {
	var cmds []tea.Cmd

	if s.logsView != nil {
		cmds = append(cmds, s.logsView.Init())
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

	case messages.LogAddedMsg:
		// Route log messages to logs component
		if s.logsView != nil {
			_, cmd := s.logsView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case messages.ProcessStartedMsg:
		// Future: Route to process component
		// For now, just log it via LogAddedMsg if needed

	case messages.ProcessStoppedMsg:
		// Future: Route to process component

	case messages.URLDetectedMsg:
		// Future: Route to URLs component

	case tea.KeyMsg:
		// Handle global keys (component switching, etc.)
		switch msg.String() {
		case "ctrl+c", "q":
			// Quit handled by parent
		default:
			// Forward to active component
			if s.currentView == "logs" && s.logsView != nil {
				_, cmd := s.logsView.Update(msg)
				cmds = append(cmds, cmd)
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

	default:
		return "Unknown view: " + s.currentView
	}
}