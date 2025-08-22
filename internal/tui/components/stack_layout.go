package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// StackLayout overlays components on top of each other (z-order)
type StackLayout struct {
	BaseComponent
	layers []Component // Back to front rendering order
}

// NewStackLayout creates a new stack layout
func NewStackLayout() *StackLayout {
	return &StackLayout{
		BaseComponent: BaseComponent{visible: true},
		layers:        []Component{},
	}
}

// AddLayer adds a component as a new layer (on top)
func (s *StackLayout) AddLayer(component Component) {
	s.layers = append(s.layers, component)
	if sizable, ok := component.(Sizable); ok {
		sizable.SetSize(s.width, s.height)
	}
}

// RemoveLayer removes the top layer
func (s *StackLayout) RemoveTopLayer() Component {
	if len(s.layers) == 0 {
		return nil
	}
	layer := s.layers[len(s.layers)-1]
	s.layers = s.layers[:len(s.layers)-1]
	return layer
}

// Update propagates updates to layers
func (s *StackLayout) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle resize
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)
		// Resize all layers
		for _, layer := range s.layers {
			if sizable, ok := layer.(Sizable); ok {
				sizable.SetSize(msg.Width, msg.Height)
			}
		}
	}
	
	// Input goes to top visible layer first
	if isInputMessage(msg) {
		for i := len(s.layers) - 1; i >= 0; i-- {
			if v, ok := s.layers[i].(Visible); ok && !v.IsVisible() {
				continue
			}
			
			// Let this layer handle the input
			newLayer, cmd := s.layers[i].Update(msg)
			s.layers[i] = newLayer
			if cmd != nil {
				return s, cmd // Top layer consumed the input
			}
		}
	} else {
		// Non-input messages go to all layers
		for i, layer := range s.layers {
			newLayer, cmd := layer.Update(msg)
			s.layers[i] = newLayer
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	
	return s, tea.Batch(cmds...)
}

// View renders all visible layers with overlays
func (s *StackLayout) View() string {
	if !s.visible || len(s.layers) == 0 {
		return ""
	}
	
	// Start with the base layer
	var baseView string
	for _, layer := range s.layers {
		if v, ok := layer.(Visible); ok && !v.IsVisible() {
			continue
		}
		
		if baseView == "" {
			baseView = layer.View()
		} else {
			// Overlay subsequent layers
			overlay := layer.View()
			if overlay != "" {
				// Center the overlay by default
				// Components can override this by implementing their own positioning
				// For now, just replace the view entirely
				// TODO: Implement proper overlay positioning
				baseView = overlay
			}
		}
	}
	
	return baseView
}

// SetSize sets the size and propagates to all layers
func (s *StackLayout) SetSize(width, height int) {
	s.BaseComponent.SetSize(width, height)
	for _, layer := range s.layers {
		if sizable, ok := layer.(Sizable); ok {
			sizable.SetSize(width, height)
		}
	}
}