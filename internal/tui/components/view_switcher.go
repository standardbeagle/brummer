package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ViewSwitcher manages switching between different view components
type ViewSwitcher struct {
	BaseComponent
	views       map[string]Component
	currentView string
	current     Component
}

// NewViewSwitcher creates a new view switcher
func NewViewSwitcher() *ViewSwitcher {
	return &ViewSwitcher{
		BaseComponent: BaseComponent{visible: true},
		views:        make(map[string]Component),
	}
}

// AddView adds a view with a name
func (v *ViewSwitcher) AddView(name string, component Component) {
	v.views[name] = component
	if v.currentView == "" {
		v.SwitchTo(name)
	}
}

// SwitchTo switches to a named view
func (v *ViewSwitcher) SwitchTo(name string) {
	if component, ok := v.views[name]; ok {
		// Blur previous view
		if v.current != nil {
			v.current.Blur()
		}
		
		// Switch to new view
		v.currentView = name
		v.current = component
		
		// Size the new view
		if sizable, ok := v.current.(Sizable); ok {
			sizable.SetSize(v.width, v.height)
		}
		
		// Focus the new view if we have focus
		if v.focused {
			v.current.Focus()
		}
	}
}

// CurrentView returns the name of the current view
func (v *ViewSwitcher) CurrentView() string {
	return v.currentView
}

// Update delegates to the current view
func (v *ViewSwitcher) Update(msg tea.Msg) (Component, tea.Cmd) {
	// Handle resize
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetSize(msg.Width, msg.Height)
		if sizable, ok := v.current.(Sizable); ok {
			sizable.SetSize(msg.Width, msg.Height)
		}
	}
	
	// Update current view
	if v.current != nil {
		newCurrent, cmd := v.current.Update(msg)
		v.views[v.currentView] = newCurrent
		v.current = newCurrent
		return v, cmd
	}
	
	return v, nil
}

// View renders the current view
func (v *ViewSwitcher) View() string {
	if !v.visible || v.current == nil {
		return ""
	}
	return v.current.View()
}

// Focus passes focus to the current view
func (v *ViewSwitcher) Focus() tea.Cmd {
	v.focused = true
	if v.current != nil {
		return v.current.Focus()
	}
	return nil
}

// Blur removes focus from the current view
func (v *ViewSwitcher) Blur() tea.Cmd {
	v.focused = false
	if v.current != nil {
		return v.current.Blur()
	}
	return nil
}

// SetSize sets the size and propagates to current view
func (v *ViewSwitcher) SetSize(width, height int) {
	v.BaseComponent.SetSize(width, height)
	if v.current != nil {
		if sizable, ok := v.current.(Sizable); ok {
			sizable.SetSize(width, height)
		}
	}
}