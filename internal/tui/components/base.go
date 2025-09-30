package components

import tea "github.com/charmbracelet/bubbletea"

// Component is the interface that all TUI components must implement.
// It extends tea.Model with additional lifecycle methods for focus and sizing.
type Component interface {
	tea.Model

	// SetSize updates the component's dimensions
	SetSize(width, height int)

	// Focus gives keyboard focus to the component
	Focus() tea.Cmd

	// Blur removes keyboard focus from the component
	Blur() tea.Cmd
}

// BaseComponent provides common functionality for all components.
// Components should embed this struct to get default implementations.
type BaseComponent struct {
	width   int
	height  int
	focused bool
	visible bool
}

// NewBaseComponent creates a base component with default visibility
func NewBaseComponent() BaseComponent {
	return BaseComponent{
		visible: true,
	}
}

// SetSize updates the component dimensions
func (b *BaseComponent) SetSize(width, height int) {
	b.width = width
	b.height = height
}

// Focus enables keyboard focus
func (b *BaseComponent) Focus() tea.Cmd {
	b.focused = true
	return nil
}

// Blur disables keyboard focus
func (b *BaseComponent) Blur() tea.Cmd {
	b.focused = false
	return nil
}

// Width returns the current width
func (b *BaseComponent) Width() int {
	return b.width
}

// Height returns the current height
func (b *BaseComponent) Height() int {
	return b.height
}

// IsFocused returns whether the component has focus
func (b *BaseComponent) IsFocused() bool {
	return b.focused
}

// IsVisible returns whether the component is visible
func (b *BaseComponent) IsVisible() bool {
	return b.visible
}

// SetVisible controls component visibility
func (b *BaseComponent) SetVisible(visible bool) {
	b.visible = visible
}