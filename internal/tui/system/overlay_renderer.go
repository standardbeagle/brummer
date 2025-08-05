package system

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// OverlayRenderer handles rendering system messages as overlays using Lipgloss
type OverlayRenderer struct {
	controller *Controller

	// Styles
	borderStyle  lipgloss.Style
	titleStyle   lipgloss.Style
	messageStyle lipgloss.Style
	errorStyle   lipgloss.Style
	warnStyle    lipgloss.Style
	infoStyle    lipgloss.Style
}

// NewOverlayRenderer creates a new overlay renderer
func NewOverlayRenderer(controller *Controller) *OverlayRenderer {
	r := &OverlayRenderer{
		controller: controller,
	}
	r.initStyles()
	return r
}

// initStyles initializes all Lipgloss styles
func (r *OverlayRenderer) initStyles() {
	// Border style for the overlay panel with background
	r.borderStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Background(lipgloss.Color("235")). // Dark background to ensure visibility
		Padding(1, 2)

	// Title style
	r.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		MarginBottom(1)

	// Message styles by level
	r.errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	r.warnStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	r.infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	r.messageStyle = lipgloss.NewStyle()
}

// RenderOverlay renders the system messages as an overlay on the main content
func (r *OverlayRenderer) RenderOverlay(mainContent string, width, height int) string {
	if !r.controller.HasMessages() {
		return mainContent
	}

	// Create the panel content
	panel := r.renderPanel(width)
	panelLines := strings.Split(panel, "\n")
	panelHeight := len(panelLines)

	// Split main content into lines
	mainLines := strings.Split(mainContent, "\n")

	// Calculate position for bottom-right placement
	panelWidth := r.getPanelWidth(width)
	startRow := height - panelHeight - 2 // Leave some margin from bottom
	if startRow < 0 {
		startRow = 0
	}
	startCol := width - panelWidth - 2 // Leave some margin from right
	if startCol < 0 {
		startCol = 0
	}

	// Ensure we have enough lines in main content
	for len(mainLines) < height {
		mainLines = append(mainLines, strings.Repeat(" ", width))
	}

	// Overlay the panel onto the main content
	for i, panelLine := range panelLines {
		targetRow := startRow + i
		if targetRow >= 0 && targetRow < len(mainLines) {
			line := mainLines[targetRow]
			// Ensure line is long enough
			if len(line) < startCol {
				line = line + strings.Repeat(" ", startCol-len(line))
			}

			// Replace the portion of the line with the panel content
			if startCol > 0 {
				// Keep the left part of the original line
				beforePanel := line
				if len(beforePanel) > startCol {
					beforePanel = beforePanel[:startCol]
				} else {
					beforePanel = beforePanel + strings.Repeat(" ", startCol-len(beforePanel))
				}
				mainLines[targetRow] = beforePanel + panelLine
			} else {
				mainLines[targetRow] = panelLine
			}
		}
	}

	return strings.Join(mainLines, "\n")
}

// renderPanel renders the system message panel
func (r *OverlayRenderer) renderPanel(maxWidth int) string {
	// Get messages to display
	messages := r.controller.GetMessages()
	if len(messages) == 0 {
		return ""
	}

	// Title
	title := r.titleStyle.Render("System Messages")

	// Render messages
	var messageLines []string
	for _, msg := range messages {
		messageLines = append(messageLines, r.renderMessage(msg))
	}

	// Join all messages
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		strings.Join(messageLines, "\n"),
	)

	// Apply border and padding
	panelWidth := r.getPanelWidth(maxWidth)
	panel := r.borderStyle.
		Width(panelWidth - 4). // Account for border and padding
		Render(content)

	return panel
}

// renderMessage renders a single message with appropriate styling
func (r *OverlayRenderer) renderMessage(msg Message) string {
	// Choose style based on level
	var style lipgloss.Style
	switch msg.Level {
	case "ERROR":
		style = r.errorStyle
	case "WARN":
		style = r.warnStyle
	case "INFO":
		style = r.infoStyle
	default:
		style = r.messageStyle
	}

	// Format message with timestamp and context
	timestamp := msg.Timestamp.Format("15:04:05")
	prefix := fmt.Sprintf("[%s] %s:", timestamp, msg.Context)

	// Combine prefix and message
	fullMessage := fmt.Sprintf("%s %s", prefix, msg.Message)

	// Truncate if too long
	maxLength := 60
	if len(fullMessage) > maxLength {
		fullMessage = fullMessage[:maxLength-3] + "..."
	}

	return style.Render(fullMessage)
}

// getPanelWidth calculates the appropriate panel width
func (r *OverlayRenderer) getPanelWidth(screenWidth int) int {
	// Use 1/3 of screen width, with min/max bounds
	panelWidth := screenWidth / 3

	minWidth := 40
	maxWidth := 80

	if panelWidth < minWidth {
		panelWidth = minWidth
	}
	if panelWidth > maxWidth {
		panelWidth = maxWidth
	}

	// Don't exceed screen width
	if panelWidth > screenWidth-4 {
		panelWidth = screenWidth - 4
	}

	return panelWidth
}

// RenderFullScreen renders the system messages in full screen mode
func (r *OverlayRenderer) RenderFullScreen(width, height int) string {
	if !r.controller.HasMessages() {
		noMessagesStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Align(lipgloss.Center, lipgloss.Center).
			Width(width).
			Height(height)
		return noMessagesStyle.Render("No system messages")
	}

	// Get all messages
	messages := r.controller.GetMessages()

	// Title
	title := r.titleStyle.Render(fmt.Sprintf("System Messages (%d total)", len(messages)))

	// Render messages with scrolling handled by viewport in controller
	var messageLines []string
	for _, msg := range messages {
		messageLines = append(messageLines, r.renderFullMessage(msg))
	}

	// Create container for full screen
	containerStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1, 2)

	// Create separator with lipgloss border
	separatorStyle := lipgloss.NewStyle().
		Width(width - 4).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(lipgloss.Color("240"))

	// Join content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		separatorStyle.Render(""),
		strings.Join(messageLines, "\n"),
	)

	return containerStyle.Render(content)
}

// renderFullMessage renders a message with full details for full screen view
func (r *OverlayRenderer) renderFullMessage(msg Message) string {
	// Choose style based on level
	var levelStyle lipgloss.Style
	switch msg.Level {
	case "ERROR":
		levelStyle = r.errorStyle
	case "WARN":
		levelStyle = r.warnStyle
	case "INFO":
		levelStyle = r.infoStyle
	default:
		levelStyle = r.messageStyle
	}

	// Format message parts
	timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")
	level := levelStyle.Render(fmt.Sprintf("[%-5s]", msg.Level))
	context := lipgloss.NewStyle().Bold(true).Render(msg.Context)

	// Combine parts
	header := fmt.Sprintf("%s %s %s", timestamp, level, context)
	message := "  " + msg.Message

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		message,
	)
}
