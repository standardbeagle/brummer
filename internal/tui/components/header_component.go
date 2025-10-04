package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SimpleHeaderComponent displays a clean, simple header
type SimpleHeaderComponent struct {
	BaseComponent

	title string
	width int
}

// NewSimpleHeaderComponent creates a simple header component
func NewSimpleHeaderComponent(title string) *SimpleHeaderComponent {
	return &SimpleHeaderComponent{
		BaseComponent: NewBaseComponent(),
		title:         title,
	}
}

// Init initializes the component
func (c *SimpleHeaderComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *SimpleHeaderComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil
	}

	return c, nil
}

// View renders a simple, visible header
func (c *SimpleHeaderComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	// Create a simple, visible header with fixed height
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")). // Bright Blue
		Background(lipgloss.Color("236")). // Dark Gray background
		Padding(0, 2).
		MaxWidth(c.width) // Constrain width

	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MaxWidth(c.width).
		SetString("─")

	header := headerStyle.Render(fmt.Sprintf("🐝 %s - Development Buddy", c.title))
	divider := dividerStyle.Render("")

	// Apply fixed height constraints - header always takes exactly 2 lines
	headerContainer := lipgloss.NewStyle().
		Height(2).  // Exactly 2 lines (header + divider)
		Width(c.width)

	return headerContainer.Render(fmt.Sprintf("%s\n%s", header, divider))
}

// SetSize updates component dimensions
func (c *SimpleHeaderComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.width = width
}
