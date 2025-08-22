package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BorderWindow is a component that properly handles borders with content sizing
// It automatically adjusts content width/height to account for borders
type BorderWindow struct {
	BaseComponent
	
	// Content to display
	content Component
	
	// Title (optional)
	title string
	
	// Border style
	borderStyle  lipgloss.Border
	borderColor  lipgloss.Color
	
	// Title style
	titleStyle   lipgloss.Style
	
	// Padding inside border
	paddingTop    int
	paddingBottom int
	paddingLeft   int
	paddingRight  int
}

// NewBorderWindow creates a new border window with content
func NewBorderWindow(content Component) *BorderWindow {
	return &BorderWindow{
		BaseComponent: BaseComponent{
			visible: true,
			width:   80,
			height:  24,
		},
		content:      content,
		borderStyle:  lipgloss.NormalBorder(),
		borderColor:  lipgloss.Color("240"),
		titleStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")),
	}
}

// SetTitle sets the window title
func (b *BorderWindow) SetTitle(title string) {
	b.title = title
}

// SetBorderStyle sets the border style (Normal, Rounded, Double, etc)
func (b *BorderWindow) SetBorderStyle(style lipgloss.Border) {
	b.borderStyle = style
}

// SetBorderColor sets the border color
func (b *BorderWindow) SetBorderColor(color lipgloss.Color) {
	b.borderColor = color
}

// SetPadding sets internal padding
func (b *BorderWindow) SetPadding(top, right, bottom, left int) {
	b.paddingTop = top
	b.paddingRight = right
	b.paddingBottom = bottom
	b.paddingLeft = left
}

// SetContent changes the content component
func (b *BorderWindow) SetContent(content Component) {
	b.content = content
	b.updateContentSize()
}

// SetSize sets the outer size (including border)
func (b *BorderWindow) SetSize(width, height int) {
	b.width = width
	b.height = height
	b.updateContentSize()
}

// updateContentSize updates the content size accounting for borders and padding
func (b *BorderWindow) updateContentSize() {
	if b.content == nil {
		return
	}
	
	// Calculate available space for content
	// Borders take 2 chars width (left + right) and 2 lines height (top + bottom)
	contentWidth := b.width - 2 - b.paddingLeft - b.paddingRight
	contentHeight := b.height - 2 - b.paddingTop - b.paddingBottom
	
	// If we have a title, it takes an additional line
	if b.title != "" {
		contentHeight--
	}
	
	// Ensure positive dimensions
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}
	
	// Update content size if it's sizable
	if sizable, ok := b.content.(Sizable); ok {
		sizable.SetSize(contentWidth, contentHeight)
	}
}

// Update handles messages
func (b *BorderWindow) Update(msg tea.Msg) (Component, tea.Cmd) {
	// Handle window sizing
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.SetSize(msg.Width, msg.Height)
	}
	
	// Forward to content
	if b.content != nil {
		newContent, cmd := b.content.Update(msg)
		b.content = newContent
		return b, cmd
	}
	
	return b, nil
}

// View renders the window with border
func (b *BorderWindow) View() string {
	if !b.visible || b.content == nil {
		return ""
	}
	
	// Get content view
	contentView := b.content.View()
	
	// Apply padding if set
	if b.paddingTop > 0 || b.paddingBottom > 0 || b.paddingLeft > 0 || b.paddingRight > 0 {
		paddingStyle := lipgloss.NewStyle().
			PaddingTop(b.paddingTop).
			PaddingBottom(b.paddingBottom).
			PaddingLeft(b.paddingLeft).
			PaddingRight(b.paddingRight)
		contentView = paddingStyle.Render(contentView)
	}
	
	// Create border style with proper width
	// The width here is the INNER width (content area)
	innerWidth := b.width - 2  // Subtract border chars
	innerHeight := b.height - 2 // Subtract border lines
	
	style := lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		Border(b.borderStyle).
		BorderForeground(b.borderColor)
	
	// If we have a title, render it in the top border
	if b.title != "" {
		// For now, render title separately
		// TODO: Integrate title into border top line
		titleLine := b.titleStyle.
			Width(innerWidth).
			Align(lipgloss.Center).
			Render(b.title)
		
		// Combine title and content
		contentWithTitle := lipgloss.JoinVertical(lipgloss.Left, titleLine, contentView)
		return style.Render(contentWithTitle)
	}
	
	return style.Render(contentView)
}

// Focus passes focus to content
func (b *BorderWindow) Focus() tea.Cmd {
	b.focused = true
	if b.content != nil {
		return b.content.Focus()
	}
	return nil
}

// Blur removes focus from content
func (b *BorderWindow) Blur() tea.Cmd {
	b.focused = false
	if b.content != nil {
		return b.content.Blur()
	}
	return nil
}

// Focused returns whether the window is focused
func (b *BorderWindow) Focused() bool {
	return b.focused
}