package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ViewRenderer is a simplified interface for existing view controllers
type ViewRenderer interface {
	Render(width, height int) string
}

// ViewUpdater is an optional interface for views that handle updates
type ViewUpdater interface {
	Update(msg tea.Msg) tea.Cmd
}

// ViewAdapter wraps existing view controllers to implement Component interface
type ViewAdapter struct {
	BaseComponent
	view ViewRenderer
	name string
}

// NewViewAdapter creates an adapter for existing view controllers
func NewViewAdapter(name string, view ViewRenderer) *ViewAdapter {
	return &ViewAdapter{
		BaseComponent: BaseComponent{visible: true},
		view:         view,
		name:         name,
	}
}

// Update handles updates if the view supports it
func (v *ViewAdapter) Update(msg tea.Msg) (Component, tea.Cmd) {
	// Handle resize
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
	}
	
	// If view implements ViewUpdater, delegate update
	if updater, ok := v.view.(ViewUpdater); ok {
		cmd := updater.Update(msg)
		return v, cmd
	}
	
	return v, nil
}

// View renders using the wrapped view controller
func (v *ViewAdapter) View() string {
	if !v.visible {
		return ""
	}
	return v.view.Render(v.width, v.height)
}

// GetName returns the name of the wrapped view
func (v *ViewAdapter) GetName() string {
	return v.name
}