// Package components provides a composable UI component system for BubbleTea
package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Component is the base interface for all UI components in the system.
// Components handle their own updates, rendering, and focus management.
type Component interface {
	// Update handles messages and returns updated component and command
	Update(msg tea.Msg) (Component, tea.Cmd)
	
	// View renders the component to a string
	View() string
	
	// Focus management
	Focus() tea.Cmd
	Blur() tea.Cmd
	Focused() bool
}

// Sizable components can be resized
type Sizable interface {
	Component
	SetSize(width, height int)
	GetSize() (width, height int)
}

// Visible components can be shown/hidden
type Visible interface {
	Component
	SetVisible(bool)
	IsVisible() bool
}

// Container components can contain other components
type Container interface {
	Component
	AddChild(Component)
	RemoveChild(Component)
	Children() []Component
}

// BaseComponent provides common functionality for components
type BaseComponent struct {
	width    int
	height   int
	focused  bool
	visible  bool
}

// SetSize sets the component dimensions
func (b *BaseComponent) SetSize(width, height int) {
	b.width = width
	b.height = height
}

// GetSize returns the component dimensions
func (b *BaseComponent) GetSize() (int, int) {
	return b.width, b.height
}

// Focus sets focus to this component
func (b *BaseComponent) Focus() tea.Cmd {
	b.focused = true
	return nil
}

// Blur removes focus from this component
func (b *BaseComponent) Blur() tea.Cmd {
	b.focused = false
	return nil
}

// Focused returns whether the component has focus
func (b *BaseComponent) Focused() bool {
	return b.focused
}

// SetVisible sets the visibility of the component
func (b *BaseComponent) SetVisible(visible bool) {
	b.visible = visible
}

// IsVisible returns whether the component is visible
func (b *BaseComponent) IsVisible() bool {
	return b.visible
}

// Update provides a default no-op update
func (b *BaseComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	return b, nil
}

// View provides a default empty view
func (b *BaseComponent) View() string {
	return ""
}