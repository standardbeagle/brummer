package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HeaderRenderer provides header content
type HeaderRenderer interface {
	RenderHeader() string
	GetHeaderHeight() int
}

// Header component wraps header rendering
type Header struct {
	BaseComponent
	renderer HeaderRenderer
	style    lipgloss.Style
}

// NewHeader creates a new header component
func NewHeader(renderer HeaderRenderer) *Header {
	h := &Header{
		BaseComponent: BaseComponent{visible: true},
		renderer:      renderer,
		style:         lipgloss.NewStyle(),
	}

	// Set fixed height based on renderer
	if renderer != nil {
		h.height = renderer.GetHeaderHeight()
	}

	return h
}

// Update handles header updates
func (h *Header) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Headers typically don't handle input
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		// Height stays fixed
	}
	return h, nil
}

// View renders the header
func (h *Header) View() string {
	if !h.visible || h.renderer == nil {
		return ""
	}

	content := h.renderer.RenderHeader()

	// Apply width constraint
	if h.width > 0 {
		return h.style.Width(h.width).Render(content)
	}

	return content
}

// Footer component wraps footer rendering
type Footer struct {
	BaseComponent
	renderer FooterRenderer
	style    lipgloss.Style
}

// FooterRenderer provides footer content
type FooterRenderer interface {
	RenderFooter() string
	GetFooterHeight() int
}

// NewFooter creates a new footer component
func NewFooter(renderer FooterRenderer) *Footer {
	f := &Footer{
		BaseComponent: BaseComponent{visible: true},
		renderer:      renderer,
		style:         lipgloss.NewStyle(),
	}

	// Set fixed height based on renderer
	if renderer != nil {
		f.height = renderer.GetFooterHeight()
	}

	return f
}

// Update handles footer updates
func (f *Footer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Footers typically don't handle input
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		// Height stays fixed
	}
	return f, nil
}

// View renders the footer
func (f *Footer) View() string {
	if !f.visible || f.renderer == nil {
		return ""
	}

	content := f.renderer.RenderFooter()

	// Apply width constraint
	if f.width > 0 {
		return f.style.Width(f.width).Render(content)
	}

	return content
}
