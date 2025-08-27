package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SystemPanelRenderer provides system panel content
type SystemPanelRenderer interface {
	RenderPanel() string
	IsExpanded() bool
	HasMessages() bool
	HandleKey(key string) tea.Cmd
}

// SystemPanel is an overlay component for system messages
type SystemPanel struct {
	Window
	renderer SystemPanelRenderer

	// Positioning
	position  string // "top", "bottom", "center"
	slideFrom string // Animation direction
}

// NewSystemPanel creates a new system panel overlay
func NewSystemPanel(renderer SystemPanelRenderer) *SystemPanel {
	// Create a simple content wrapper for the renderer
	content := &systemPanelContent{renderer: renderer}

	panel := &SystemPanel{
		Window:   *NewWindow("System Messages", content),
		renderer: renderer,
		position: "bottom",
	}

	// Configure as modal when expanded
	panel.SetModal(false) // Not modal by default, can be changed when expanded

	// Use a distinctive border style
	panel.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("220"))

	return panel
}

// Update handles system panel specific updates
func (s *SystemPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check if we should show/hide based on renderer state
	if s.renderer != nil {
		if s.renderer.HasMessages() && !s.visible {
			s.Show()
		} else if !s.renderer.HasMessages() && s.visible {
			s.Hide()
		}

		// Update expanded state
		if s.renderer.IsExpanded() {
			s.state = WindowExpanded
			s.SetModal(true) // Modal when expanded
		} else {
			s.state = WindowNormal
			s.SetModal(false)
		}
	}

	// Handle escape to close
	if s.visible {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				if s.renderer != nil {
					return s, s.renderer.HandleKey("esc")
				}
			}
		}
	}

	// Delegate to Window
	return s.Window.Update(msg)
}

// View renders the system panel with positioning
func (s *SystemPanel) View() string {
	if !s.visible || s.renderer == nil {
		return ""
	}

	// Get the panel content
	content := s.renderer.RenderPanel()

	// Apply styling based on state
	if s.state == WindowExpanded {
		// Full screen with border
		return s.borderStyle.
			Width(s.width - 2).
			Height(s.height - 2).
			Render(content)
	}

	// Normal state - position at bottom
	// Calculate height based on content
	lines := lipgloss.Height(content)
	if lines > s.height/3 {
		lines = s.height / 3 // Max 1/3 of screen
	}

	// Style for bottom panel
	styled := lipgloss.NewStyle().
		Width(s.width).
		Height(lines).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("240")).
		Render(content)

	// Position at bottom
	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Left,
		lipgloss.Bottom,
		styled,
	)
}

// systemPanelContent wraps the renderer as a component
type systemPanelContent struct {
	BaseComponent
	renderer SystemPanelRenderer
}

func (c *systemPanelContent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// System panel handles its own updates through the renderer
	return c, nil
}

func (c *systemPanelContent) View() string {
	if c.renderer != nil {
		return c.renderer.RenderPanel()
	}
	return ""
}
