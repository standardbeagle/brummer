package components

// DEPRECATED: Placeholder AIViewComponent kept for reference only.
// Not used in Component mode; the active AI UI is AIPTYComponent.
// Keeping this file to avoid accidental re-wiring of the old placeholder.

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AIViewComponent represents the AI Coders view
type AIViewComponent struct {
	BaseComponent

	// Styles
	titleStyle lipgloss.Style
	helpStyle  lipgloss.Style
}

// NewAIViewComponent creates a new AI view component
func NewAIViewComponent() *AIViewComponent {
	a := &AIViewComponent{}

	// Initialize styles
	a.initStyles()

	// Set visible by default
	a.visible = true

	return a
}

// initStyles initializes the component styles
func (a *AIViewComponent) initStyles() {
	a.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("213"))

	a.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
}

// Update handles messages
func (a *AIViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle size messages
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
	case tea.KeyMsg:
		// Handle key events when focused
		if a.focused {
			switch msg.String() {
			case "?":
				// Show help
				return a, nil
			}
		}
	}

	return a, nil
}

// View renders the component
func (a *AIViewComponent) View() string {
	if !a.visible {
		return ""
	}

	// Calculate content area
	contentHeight := a.height - 4 // Leave room for borders and padding
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Build content
	var content string

	// Title
	title := a.titleStyle.Render("🤖 AI Coders")
	content += title + "\n\n"

	// Placeholder content
	content += a.helpStyle.Render("AI Coders feature coming soon!\n\n")
	content += a.helpStyle.Render("This will provide tmux-style interactive AI coding sessions.\n")
	content += a.helpStyle.Render("• Split panes for multiple AI sessions\n")
	content += a.helpStyle.Render("• Context-aware code completion\n")
	content += a.helpStyle.Render("• Integrated with your development workflow\n")

	// Add instructions
	content += "\n" + a.helpStyle.Render("Press '/' to open command palette")

	// Apply sizing
	style := lipgloss.NewStyle().
		Width(a.width).
		Height(contentHeight).
		Padding(1)

	return style.Render(content)
}

// Focus sets focus to this component
func (a *AIViewComponent) Focus() tea.Cmd {
	a.focused = true
	return nil
}

// Blur removes focus from this component
func (a *AIViewComponent) Blur() tea.Cmd {
	a.focused = false
	return nil
}

// SetSize sets the component size
func (a *AIViewComponent) SetSize(width, height int) {
	a.width = width
	a.height = height
}

// String returns a string representation
func (a *AIViewComponent) String() string {
	return fmt.Sprintf("AIViewComponent{width:%d,height:%d,focused:%v,visible:%v}",
		a.width, a.height, a.focused, a.visible)
}
