package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// ContentLayout manages the main content area layout using Lipgloss
type ContentLayout struct {
	width        int
	height       int
	headerHeight int
	footerHeight int

	// Styles
	containerStyle lipgloss.Style
	contentStyle   lipgloss.Style
}

// NewContentLayout creates a new content layout manager
func NewContentLayout() *ContentLayout {
	cl := &ContentLayout{}
	cl.initStyles()
	return cl
}

// initStyles initializes the Lipgloss styles
func (cl *ContentLayout) initStyles() {
	cl.containerStyle = lipgloss.NewStyle()
	cl.contentStyle = lipgloss.NewStyle()
}

// UpdateDimensions updates the layout dimensions
func (cl *ContentLayout) UpdateDimensions(width, height, headerHeight, footerHeight int) {
	cl.width = width
	cl.height = height
	cl.headerHeight = headerHeight
	cl.footerHeight = footerHeight
}

// RenderContent renders content with proper layout
func (cl *ContentLayout) RenderContent(content string) string {
	// Calculate available content height
	// Subtract 1 to account for proper spacing
	contentHeight := cl.height - cl.headerHeight - cl.footerHeight - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Apply content style with proper dimensions
	// Must set both Width and Height to constrain content properly
	styledContent := cl.contentStyle.
		Width(cl.width).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(content)

	return styledContent
}

// RenderWithOverlay renders content with an overlay component
func (cl *ContentLayout) RenderWithOverlay(mainContent, overlay string, overlayWidth, overlayHeight int) string {
	// Calculate content area
	contentHeight := cl.height - cl.headerHeight - cl.footerHeight

	// Use Lipgloss Place to position the overlay
	return lipgloss.Place(
		cl.width,
		contentHeight,
		lipgloss.Right,
		lipgloss.Bottom,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceBackground(lipgloss.NoColor{}),
	)
}

// RenderFullScreenOverlay renders a full-screen overlay
func (cl *ContentLayout) RenderFullScreenOverlay(overlay string) string {
	// Calculate content area
	contentHeight := cl.height - cl.headerHeight - cl.footerHeight

	// Style the overlay to fill the content area
	overlayStyle := lipgloss.NewStyle().
		Width(cl.width).
		Height(contentHeight)

	return overlayStyle.Render(overlay)
}

// RenderSplitView renders a split view layout
func (cl *ContentLayout) RenderSplitView(left, right string, splitRatio float64) string {
	// Calculate widths
	leftWidth := int(float64(cl.width) * splitRatio)
	rightWidth := cl.width - leftWidth

	// Ensure minimum widths
	if leftWidth < 20 {
		leftWidth = 20
		rightWidth = cl.width - leftWidth
	}
	if rightWidth < 20 {
		rightWidth = 20
		leftWidth = cl.width - rightWidth
	}

	// Calculate content height
	contentHeight := cl.height - cl.headerHeight - cl.footerHeight

	// Style each side
	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(contentHeight).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderLeft(false).
		BorderTop(false).
		BorderBottom(false)

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight)

	// Render sides
	leftContent := leftStyle.Render(left)
	rightContent := rightStyle.Render(right)

	// Join horizontally
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftContent,
		rightContent,
	)
}

// RenderCentered renders content centered in the available space
func (cl *ContentLayout) RenderCentered(content string) string {
	contentHeight := cl.height - cl.headerHeight - cl.footerHeight

	centeredStyle := lipgloss.NewStyle().
		Width(cl.width).
		Height(contentHeight).
		Align(lipgloss.Center, lipgloss.Center)

	return centeredStyle.Render(content)
}
