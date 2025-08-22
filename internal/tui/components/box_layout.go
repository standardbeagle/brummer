package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Direction represents the layout direction
type Direction int

const (
	Vertical Direction = iota
	Horizontal
)

// BoxLayout arranges components in a vertical or horizontal box
type BoxLayout struct {
	BaseComponent
	direction Direction
	children  []Component
	weights   []float64 // Proportional sizing (0 = fixed size)
	spacing   int
}

// NewBoxLayout creates a new box layout
func NewBoxLayout(direction Direction) *BoxLayout {
	return &BoxLayout{
		BaseComponent: BaseComponent{visible: true},
		direction:     direction,
		children:      []Component{},
		weights:       []float64{},
	}
}

// AddChild adds a component with a weight (0 = use component's natural size)
func (b *BoxLayout) AddChild(child Component, weight float64) {
	b.children = append(b.children, child)
	b.weights = append(b.weights, weight)
}

// Update propagates updates to children
func (b *BoxLayout) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle resize
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.SetSize(msg.Width, msg.Height)
		b.layoutChildren()
	}
	
	// Update focused child or all children for certain messages
	for i, child := range b.children {
		if child.Focused() || !isInputMessage(msg) {
			newChild, cmd := child.Update(msg)
			b.children[i] = newChild
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	
	return b, tea.Batch(cmds...)
}

// View renders the layout
func (b *BoxLayout) View() string {
	if !b.visible || len(b.children) == 0 {
		return ""
	}
	
	views := make([]string, 0, len(b.children))
	for _, child := range b.children {
		if v, ok := child.(Visible); ok && !v.IsVisible() {
			continue
		}
		views = append(views, child.View())
	}
	
	if b.direction == Vertical {
		return lipgloss.JoinVertical(lipgloss.Left, views...)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
}

// layoutChildren calculates and sets child sizes
func (b *BoxLayout) layoutChildren() {
	if len(b.children) == 0 {
		return
	}
	
	// Calculate flexible space
	totalWeight := 0.0
	fixedSize := 0
	
	for i, weight := range b.weights {
		if weight > 0 {
			totalWeight += weight
		} else if sizable, ok := b.children[i].(Sizable); ok {
			// Use natural size for fixed components
			w, h := sizable.GetSize()
			if b.direction == Vertical {
				fixedSize += h
			} else {
				fixedSize += w
			}
		}
	}
	
	// Calculate available space for flexible components
	var availableSpace int
	if b.direction == Vertical {
		availableSpace = b.height - fixedSize - (len(b.children)-1)*b.spacing
	} else {
		availableSpace = b.width - fixedSize - (len(b.children)-1)*b.spacing
	}
	
	// Set sizes for all children
	for i, child := range b.children {
		if sizable, ok := child.(Sizable); ok {
			if b.weights[i] > 0 {
				// Flexible sizing
				proportion := b.weights[i] / totalWeight
				if b.direction == Vertical {
					height := int(float64(availableSpace) * proportion)
					sizable.SetSize(b.width, height)
				} else {
					width := int(float64(availableSpace) * proportion)
					sizable.SetSize(width, b.height)
				}
			} else {
				// Fixed sizing - keep current size but adjust perpendicular dimension
				w, h := sizable.GetSize()
				if b.direction == Vertical {
					sizable.SetSize(b.width, h)
				} else {
					sizable.SetSize(w, b.height)
				}
			}
		}
	}
}

// isInputMessage returns true for messages that should only go to focused component
func isInputMessage(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		return true
	}
	return false
}