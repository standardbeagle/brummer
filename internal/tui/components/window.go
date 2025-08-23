package components

import (
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WindowState represents the state of a window
type WindowState int

const (
	WindowHidden WindowState = iota
	WindowMinimized
	WindowNormal
	WindowExpanded
)

// Window is a container that can be shown/hidden, minimized, or expanded
type Window struct {
	BaseComponent
	
	// Window properties
	title       string
	state       WindowState
	modal       bool  // If true, captures all input when visible
	resizable   bool
	movable     bool
	
	// Content
	content     Component
	
	// Styling
	borderStyle lipgloss.Style
	titleStyle  lipgloss.Style
	
	// Animation (future enhancement)
	animationProgress float64
}

// NewWindow creates a new window component
func NewWindow(title string, content Component) *Window {
	return &Window{
		BaseComponent: BaseComponent{visible: false}, // Start hidden
		title:        title,
		state:        WindowHidden,
		content:      content,
		borderStyle:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()),
		titleStyle:   lipgloss.NewStyle().Bold(true),
	}
}

// SetModal sets whether the window is modal
func (w *Window) SetModal(modal bool) *Window {
	w.modal = modal
	return w
}

// SetResizable sets whether the window can be resized
func (w *Window) SetResizable(resizable bool) *Window {
	w.resizable = resizable
	return w
}

// Show shows the window
func (w *Window) Show() {
	w.visible = true
	w.state = WindowNormal
}

// Hide hides the window
func (w *Window) Hide() {
	w.visible = false
	w.state = WindowHidden
}

// Toggle toggles the window visibility
func (w *Window) Toggle() {
	if w.visible {
		w.Hide()
	} else {
		w.Show()
	}
}

// Minimize minimizes the window
func (w *Window) Minimize() {
	w.state = WindowMinimized
}

// Expand expands the window to full size
func (w *Window) Expand() {
	w.state = WindowExpanded
	w.visible = true
}

// IsModal returns whether the window is modal
func (w *Window) IsModal() bool {
	return w.modal && w.visible
}

// Init implements tea.Model interface
func (w *Window) Init() tea.Cmd {
	return nil
}

// Update handles window-specific messages
func (w *Window) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window-specific keys when visible
	if w.visible {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				if !w.modal {
					w.Hide()
					return w, nil
				}
			}
		}
	}
	
	// If modal and visible, consume all input
	if w.IsModal() && w.content != nil {
		newContent, cmd := w.content.Update(msg)
		if c, ok := newContent.(Component); ok { w.content = c }
		return w, cmd
	}
	
	// If not modal but visible, pass through to content
	if w.visible && w.content != nil {
		newContent, cmd := w.content.Update(msg)
		if c, ok := newContent.(Component); ok { w.content = c }
		return w, cmd
	}
	
	return w, nil
}

// View renders the window
func (w *Window) View() string {
	if !w.visible {
		return ""
	}
	
	switch w.state {
	case WindowMinimized:
		return w.renderMinimized()
	case WindowExpanded:
		return w.renderExpanded()
	default:
		return w.renderNormal()
	}
}

// renderMinimized renders a minimized window (just title bar)
func (w *Window) renderMinimized() string {
	title := w.titleStyle.Render(w.title)
	return w.borderStyle.Width(len(w.title) + 4).Height(1).Render(title)
}

// renderNormal renders a normal window
func (w *Window) renderNormal() string {
	if w.content == nil {
		return ""
	}
	
	// Get content
	content := w.content.View()
	
	// Add title if present
	if w.title != "" {
		titleLine := w.titleStyle.Render(w.title)
		content = lipgloss.JoinVertical(lipgloss.Left, titleLine, content)
	}
	
	// Apply border and size constraints
	styled := w.borderStyle.Render(content)
	
	// Apply size if set
	if w.width > 0 && w.height > 0 {
		styled = lipgloss.NewStyle().
			Width(w.width).
			Height(w.height).
			Render(styled)
	}
	
	return styled
}

// renderExpanded renders an expanded (full-screen) window
func (w *Window) renderExpanded() string {
	if w.content == nil {
		return ""
	}
	
	// In expanded mode, fill the entire space
	content := w.content.View()
	
	// Add title bar
	if w.title != "" {
		titleBar := w.renderTitleBar(true)
		content = lipgloss.JoinVertical(lipgloss.Left, titleBar, content)
	}
	
	return content
}

// renderTitleBar renders the window title bar
func (w *Window) renderTitleBar(expanded bool) string {
	var controls []string
	
	if expanded {
		controls = append(controls, "[-]") // Minimize
		controls = append(controls, "[□]")  // Restore
	} else {
		controls = append(controls, "[□]") // Maximize
	}
	controls = append(controls, "[x]") // Close
	
	controlStr := strings.Join(controls, " ")
	
	// Calculate padding for title centering
	titleWidth := w.width - len(controlStr) - 4
	if titleWidth < 0 {
		titleWidth = len(w.title)
	}
	
	title := lipgloss.NewStyle().
		Width(titleWidth).
		Align(lipgloss.Center).
		Render(w.title)
	
	return lipgloss.JoinHorizontal(lipgloss.Top, title, " ", controlStr)
}

// SetSize sets the window size
func (w *Window) SetSize(width, height int) {
	w.BaseComponent.SetSize(width, height)
	if w.content != nil {
		if sizable, ok := w.content.(Sizable); ok {
			// Account for borders and title
			contentWidth := width - 2
			contentHeight := height - 2
			if w.title != "" {
				contentHeight -= 2
			}
			sizable.SetSize(contentWidth, contentHeight)
		}
	}
}