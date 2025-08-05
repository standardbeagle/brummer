package system

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PanelRenderer handles rendering of system message panels
type PanelRenderer struct {
	controller *Controller
	styles     PanelStyles
}

// PanelStyles contains the styling configuration for panels
type PanelStyles struct {
	Border         lipgloss.Style
	Title          lipgloss.Style
	Message        lipgloss.Style
	ErrorStyle     lipgloss.Style
	WarningStyle   lipgloss.Style
	SuccessStyle   lipgloss.Style
	InfoStyle      lipgloss.Style
	TimestampStyle lipgloss.Style
}

// NewPanelRenderer creates a new panel renderer with default styles
func NewPanelRenderer(controller *Controller) *PanelRenderer {
	return &PanelRenderer{
		controller: controller,
		styles:     DefaultPanelStyles(),
	}
}

// DefaultPanelStyles returns the default styling configuration
func DefaultPanelStyles() PanelStyles {
	return PanelStyles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241")),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")),
		Message: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		ErrorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),
		WarningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")),
		SuccessStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")),
		InfoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
		TimestampStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

// RenderPanel renders the system message panel at the bottom of the screen
func (r *PanelRenderer) RenderPanel() string {
	// Don't render if no messages
	if !r.controller.HasMessages() {
		return ""
	}

	// Calculate dimensions
	width := r.controller.width
	height := 7 // Default compact height

	// Create title
	title := "System Messages"
	if r.controller.IsExpanded() {
		title = fmt.Sprintf("All System Messages (%d)", r.controller.GetMessageCount())
		// Use full available height when expanded
		height = r.controller.height - r.controller.headerHeight - r.controller.footerHeight
	}

	// Get viewport content
	content := r.controller.GetViewport().View()

	// Apply border styling
	panel := r.styles.Border.
		Width(width - 2).
		Height(height - 2).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			r.styles.Title.Render(title),
			content,
		))

	return panel
}

// RenderOverlay renders the system panel as an overlay (for compact mode)
func (r *PanelRenderer) RenderOverlay() string {
	if !r.controller.HasMessages() {
		return ""
	}

	// Calculate panel dimensions
	panelWidth := r.controller.width - 4
	if panelWidth > 100 {
		panelWidth = 100 // Cap width for readability
	}

	// Format messages directly for overlay
	var content strings.Builder
	messages := r.controller.GetMessages()
	messagesToShow := messages
	if len(messages) > 5 {
		messagesToShow = messages[:5]
	}

	for i, msg := range messagesToShow {
		if i > 0 {
			content.WriteString("\n")
		}

		// Format message with appropriate styling
		timestamp := r.styles.TimestampStyle.Render(msg.Timestamp.Format("15:04:05"))
		icon := GetIcon(msg.Level)

		// Apply level-specific styling to the message
		var styledMessage string
		switch msg.Level {
		case LevelError:
			styledMessage = r.styles.ErrorStyle.Render(fmt.Sprintf("%s %s: %s", icon, msg.Context, msg.Message))
		case LevelWarning:
			styledMessage = r.styles.WarningStyle.Render(fmt.Sprintf("%s %s: %s", icon, msg.Context, msg.Message))
		case LevelSuccess:
			styledMessage = r.styles.SuccessStyle.Render(fmt.Sprintf("%s %s: %s", icon, msg.Context, msg.Message))
		case LevelInfo:
			styledMessage = r.styles.InfoStyle.Render(fmt.Sprintf("%s %s: %s", icon, msg.Context, msg.Message))
		default:
			styledMessage = r.styles.Message.Render(fmt.Sprintf("%s %s: %s", icon, msg.Context, msg.Message))
		}

		content.WriteString(fmt.Sprintf("[%s] %s", timestamp, styledMessage))
	}

	// Add more messages indicator
	if len(messages) > 5 {
		moreMsg := fmt.Sprintf("\n... and %d more messages", len(messages)-5)
		content.WriteString(r.styles.InfoStyle.Render(moreMsg))
	}

	// Create the panel with title
	title := "System Messages"
	if r.controller.IsExpanded() {
		title = fmt.Sprintf("All System Messages (%d)", len(messages))
	}

	// Style the panel
	panel := r.styles.Border.
		Width(panelWidth).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			r.styles.Title.Render(title),
			"",
			content.String(),
		))

	// Use lipgloss to position panel at bottom
	return lipgloss.Place(
		r.controller.width,
		r.controller.height,
		lipgloss.Left,
		lipgloss.Bottom,
		panel,
		lipgloss.WithWhitespaceChars(" "),
	)
}

// RenderCompactNotification renders a single-line notification for the status bar
func (r *PanelRenderer) RenderCompactNotification() string {
	if !r.controller.HasMessages() {
		return ""
	}

	// Get the most recent message
	messages := r.controller.GetMessages()
	if len(messages) == 0 {
		return ""
	}

	msg := messages[0]
	icon := GetIcon(msg.Level)

	// Format as compact notification
	notification := fmt.Sprintf(" %s %s: %s", icon, msg.Context, msg.Message)

	// Apply level-specific styling
	switch msg.Level {
	case LevelError:
		return r.styles.ErrorStyle.Render(notification)
	case LevelWarning:
		return r.styles.WarningStyle.Render(notification)
	case LevelSuccess:
		return r.styles.SuccessStyle.Render(notification)
	case LevelInfo:
		return r.styles.InfoStyle.Render(notification)
	default:
		return r.styles.Message.Render(notification)
	}
}
