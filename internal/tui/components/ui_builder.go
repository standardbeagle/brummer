package components

import (
	"github.com/standardbeagle/brummer/internal/tui/navigation"
)

// UIBuilder helps construct the component tree
type UIBuilder struct {
	headerRenderer HeaderRenderer
	footerRenderer FooterRenderer
	views          map[navigation.View]ViewRenderer
	overlays       []Component
}

// NewUIBuilder creates a new UI builder
func NewUIBuilder() *UIBuilder {
	return &UIBuilder{
		views:    make(map[navigation.View]ViewRenderer),
		overlays: []Component{},
	}
}

// SetHeader sets the header renderer
func (b *UIBuilder) SetHeader(renderer HeaderRenderer) *UIBuilder {
	b.headerRenderer = renderer
	return b
}

// SetFooter sets the footer renderer
func (b *UIBuilder) SetFooter(renderer FooterRenderer) *UIBuilder {
	b.footerRenderer = renderer
	return b
}

// AddView adds a view renderer
func (b *UIBuilder) AddView(view navigation.View, renderer ViewRenderer) *UIBuilder {
	b.views[view] = renderer
	return b
}

// AddOverlay adds an overlay component
func (b *UIBuilder) AddOverlay(overlay Component) *UIBuilder {
	b.overlays = append(b.overlays, overlay)
	return b
}

// Build constructs the complete UI tree
func (b *UIBuilder) Build() Component {
	// Create the main view switcher
	viewSwitcher := NewViewSwitcher()
	for view, renderer := range b.views {
		adapter := NewViewAdapter(string(view), renderer)
		viewSwitcher.AddView(string(view), adapter)
	}
	
	// Create the main layout
	mainLayout := NewBoxLayout(Vertical)
	
	// Add header if present
	if b.headerRenderer != nil {
		header := NewHeader(b.headerRenderer)
		mainLayout.AddChild(header, 0) // Fixed size
	}
	
	// Add main content (view switcher)
	mainLayout.AddChild(viewSwitcher, 1.0) // Flexible size
	
	// Add footer if present
	if b.footerRenderer != nil {
		footer := NewFooter(b.footerRenderer)
		mainLayout.AddChild(footer, 0) // Fixed size
	}
	
	// If there are overlays, wrap in a stack layout
	if len(b.overlays) > 0 {
		stack := NewStackLayout()
		stack.AddLayer(mainLayout) // Base layer
		
		// Add overlay layers
		for _, overlay := range b.overlays {
			stack.AddLayer(overlay)
		}
		
		return stack
	}
	
	return mainLayout
}

// GetViewSwitcher extracts the view switcher from the built tree
// This is useful for switching views programmatically
func ExtractViewSwitcher(root Component) *ViewSwitcher {
	switch c := root.(type) {
	case *ViewSwitcher:
		return c
	case *BoxLayout:
		// Search children
		for _, child := range c.children {
			if vs := ExtractViewSwitcher(child); vs != nil {
				return vs
			}
		}
	case *StackLayout:
		// Search layers
		for _, layer := range c.layers {
			if vs := ExtractViewSwitcher(layer); vs != nil {
				return vs
			}
		}
	}
	return nil
}