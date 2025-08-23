package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// FocusChain manages focus between multiple components
type FocusChain struct {
	BaseComponent
	components   []Component
	focusIndex   int
	wrapAround   bool
}

// NewFocusChain creates a new focus chain
func NewFocusChain(wrapAround bool) *FocusChain {
	return &FocusChain{
		BaseComponent: BaseComponent{visible: true},
		components:    []Component{},
		focusIndex:    0,
		wrapAround:    wrapAround,
	}
}

// Add adds a component to the focus chain
func (f *FocusChain) Add(component Component) {
	f.components = append(f.components, component)
	// Focus first component if chain has focus
	if len(f.components) == 1 && f.focused {
		component.Focus()
	}
}

// Update handles focus cycling and delegates to focused component
func (f *FocusChain) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(f.components) == 0 {
		return f, nil
	}
	
	// Handle resize for all components
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.SetSize(msg.Width, msg.Height)
		for _, comp := range f.components {
			if sizable, ok := comp.(Sizable); ok {
				sizable.SetSize(msg.Width, msg.Height)
			}
		}
	}
	
	// Handle focus navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			return f, f.focusNext()
		case "shift+tab":
			return f, f.focusPrevious()
		}
	}
	
	// Update all components, but only the focused one gets input messages
	var cmds []tea.Cmd
	for i, comp := range f.components {
		if i == f.focusIndex || !isInputMessage(msg) {
			newComp, cmd := comp.Update(msg)
			// Type assert back to Component
			if c, ok := newComp.(Component); ok {
				f.components[i] = c
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	
	return f, tea.Batch(cmds...)
}

// View renders the focus chain (usually delegates to a layout)
func (f *FocusChain) View() string {
	// This is typically not used directly - components are arranged by layouts
	// But we provide a simple vertical stack as default
	if !f.visible || len(f.components) == 0 {
		return ""
	}
	
	layout := NewBoxLayout(Vertical)
	for _, comp := range f.components {
		layout.AddChild(comp, 1.0) // Equal weight
	}
	layout.SetSize(f.width, f.height)
	return layout.View()
}

// focusNext moves focus to the next component
func (f *FocusChain) focusNext() tea.Cmd {
	if len(f.components) == 0 {
		return nil
	}
	
	// Blur current
	f.components[f.focusIndex].Blur()
	
	// Move to next
	f.focusIndex++
	if f.focusIndex >= len(f.components) {
		if f.wrapAround {
			f.focusIndex = 0
		} else {
			f.focusIndex = len(f.components) - 1
		}
	}
	
	// Focus new
	return f.components[f.focusIndex].Focus()
}

// focusPrevious moves focus to the previous component
func (f *FocusChain) focusPrevious() tea.Cmd {
	if len(f.components) == 0 {
		return nil
	}
	
	// Blur current
	f.components[f.focusIndex].Blur()
	
	// Move to previous
	f.focusIndex--
	if f.focusIndex < 0 {
		if f.wrapAround {
			f.focusIndex = len(f.components) - 1
		} else {
			f.focusIndex = 0
		}
	}
	
	// Focus new
	return f.components[f.focusIndex].Focus()
}

// Focus sets focus to the chain and its current component
func (f *FocusChain) Focus() tea.Cmd {
	f.focused = true
	if f.focusIndex < len(f.components) {
		return f.components[f.focusIndex].Focus()
	}
	return nil
}

// Blur removes focus from the chain and its current component
func (f *FocusChain) Blur() tea.Cmd {
	f.focused = false
	if f.focusIndex < len(f.components) {
		return f.components[f.focusIndex].Blur()
	}
	return nil
}

// FocusFirst focuses the first component
func (f *FocusChain) FocusFirst() tea.Cmd {
	if len(f.components) == 0 {
		return nil
	}
	
	if f.focusIndex < len(f.components) {
		f.components[f.focusIndex].Blur()
	}
	
	f.focusIndex = 0
	return f.components[f.focusIndex].Focus()
}

// FocusLast focuses the last component
func (f *FocusChain) FocusLast() tea.Cmd {
	if len(f.components) == 0 {
		return nil
	}
	
	if f.focusIndex < len(f.components) {
		f.components[f.focusIndex].Blur()
	}
	
	f.focusIndex = len(f.components) - 1
	return f.components[f.focusIndex].Focus()
}