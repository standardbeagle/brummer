package system

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/standardbeagle/brummer/internal/tui/navigation"
)

// SystemMessageMsg represents a system message event
type SystemMessageMsg struct {
	Level   string
	Context string
	Message string
}

// View type alias for unread indicators
type View = navigation.View

// UnreadIndicator tracks unread notifications for a view
type UnreadIndicator struct {
	Count    int
	Severity string
	Icon     string
}

// Controller manages system messages and panel state
type Controller struct {
	messages    []Message
	maxMessages int
	expanded    bool
	viewport    viewport.Model

	// Unread indicators management
	unreadIndicators map[View]UnreadIndicator
	currentView      View

	// Dimensions from parent
	width        int
	height       int
	headerHeight int
	footerHeight int
}

// NewController creates a new system message controller
func NewController(maxMessages int) *Controller {
	if maxMessages <= 0 {
		maxMessages = 100
	}

	return &Controller{
		messages:         make([]Message, 0, maxMessages),
		maxMessages:      maxMessages,
		viewport:         viewport.New(0, 0),
		unreadIndicators: make(map[View]UnreadIndicator),
	}
}

// AddMessage adds a new system message
func (c *Controller) AddMessage(level, context, message string) {
	msg := NewMessage(level, context, message)

	// Add to the beginning of the list (most recent first)
	c.messages = append([]Message{msg}, c.messages...)

	// Keep only the last maxMessages
	if len(c.messages) > c.maxMessages {
		c.messages = c.messages[:c.maxMessages]
	}

	// Update the viewport content
	c.updateViewport()
}

// Clear removes all system messages
func (c *Controller) Clear() {
	c.messages = []Message{}
	c.expanded = false
	c.viewport.SetContent("")
}

// ToggleExpanded toggles the expanded state of the panel
func (c *Controller) ToggleExpanded() {
	if len(c.messages) > 0 {
		c.expanded = !c.expanded
		c.updateViewport()
	}
}

// IsExpanded returns whether the panel is expanded
func (c *Controller) IsExpanded() bool {
	return c.expanded
}

// HasMessages returns whether there are any messages
func (c *Controller) HasMessages() bool {
	return len(c.messages) > 0
}

// GetMessageCount returns the number of messages
func (c *Controller) GetMessageCount() int {
	return len(c.messages)
}

// GetMessages returns all messages
func (c *Controller) GetMessages() []Message {
	return c.messages
}

// UpdateSize updates the controller dimensions
func (c *Controller) UpdateSize(width, height, headerHeight, footerHeight int) {
	c.width = width
	c.height = height
	c.headerHeight = headerHeight
	c.footerHeight = footerHeight
	c.updateViewport()
}

// updateViewport updates the viewport with current content
func (c *Controller) updateViewport() {
	// Calculate viewport height
	height := c.height - c.headerHeight - c.footerHeight
	if !c.expanded {
		height = 5 // Show only 5 lines when not expanded
	}

	c.viewport.Width = c.width
	c.viewport.Height = height

	// Format messages for display
	content := c.formatMessagesForDisplay()
	c.viewport.SetContent(content)
}

// formatMessagesForDisplay formats messages for the panel
func (c *Controller) formatMessagesForDisplay() string {
	if len(c.messages) == 0 {
		return "No system messages"
	}

	var b strings.Builder

	// Determine how many messages to show
	messagesToShow := c.messages
	if !c.expanded && len(c.messages) > 5 {
		messagesToShow = c.messages[:5]
	}

	// Format each message
	for i, msg := range messagesToShow {
		if i > 0 {
			b.WriteString("\n")
		}

		// Format timestamp
		timestamp := msg.Timestamp.Format("15:04:05")

		// Get icon based on level
		icon := GetIcon(msg.Level)

		// Build message line
		msgLine := fmt.Sprintf("[%s] %s %s: %s",
			timestamp,
			icon,
			msg.Context,
			msg.Message,
		)

		b.WriteString(msgLine)
	}

	// Add count if not showing all messages
	if !c.expanded && len(c.messages) > 5 {
		b.WriteString(fmt.Sprintf("\n... and %d more messages (press 'e' to expand, 'm' to clear)", len(c.messages)-5))
	} else if len(c.messages) > 0 {
		// Add clear hint when showing all messages
		b.WriteString("\n(Press 'm' to clear messages)")
	}

	return b.String()
}

// GetViewport returns the viewport for rendering
func (c *Controller) GetViewport() *viewport.Model {
	return &c.viewport
}

// SetCurrentView sets the current view for unread indicator management
func (c *Controller) SetCurrentView(view View) {
	c.currentView = view
}

// UpdateUnreadIndicator updates the unread indicator for a specific view
func (c *Controller) UpdateUnreadIndicator(view View, severity string, increment int) {
	if view == c.currentView {
		// Don't mark as unread if we're currently viewing this tab
		return
	}

	indicator := c.unreadIndicators[view]
	indicator.Count += increment

	// Update severity and icon based on priority
	if c.shouldUpdateSeverity(indicator.Severity, severity) {
		indicator.Severity = severity
		indicator.Icon = GetIcon(severity)
	}

	c.unreadIndicators[view] = indicator
}

// ClearUnreadIndicator clears the unread indicator for a specific view
func (c *Controller) ClearUnreadIndicator(view View) {
	delete(c.unreadIndicators, view)
}

// shouldUpdateSeverity determines if the new severity is higher priority
func (c *Controller) shouldUpdateSeverity(current, new string) bool {
	priority := map[string]int{
		"error":   4,
		"warning": 3,
		"success": 2,
		"info":    1,
	}
	return priority[new] > priority[current]
}

// GetUnreadIndicators returns all unread indicators
func (c *Controller) GetUnreadIndicators() map[View]UnreadIndicator {
	return c.unreadIndicators
}

// OverlaySystemPanel overlays the system panel on top of the main content
func (c *Controller) OverlaySystemPanel(mainContent string) string {
	// Split main content into lines
	lines := strings.Split(mainContent, "\n")

	// Calculate panel height (5 messages + title + border = 8 lines)
	panelHeight := 8
	if c.GetMessageCount() < 5 {
		panelHeight = c.GetMessageCount() + 3 // messages + title + border
	}

	// Position panel at bottom, but above help (2 lines)
	startLine := len(lines) - panelHeight - 2
	if startLine < 0 {
		startLine = 0
	}

	// Replace the lines with the panel
	panelContent := c.formatMessagesForDisplay()
	panelLines := strings.Split(panelContent, "\n")

	// Add border and title
	panelWithBorder := []string{
		"┌─ System Messages ─────────────────────────────────────────────────┐",
	}
	for _, line := range panelLines {
		// Pad line to fit within border
		paddedLine := fmt.Sprintf("│ %-70s │", line)
		if len(line) > 70 {
			paddedLine = fmt.Sprintf("│ %.67s... │", line)
		}
		panelWithBorder = append(panelWithBorder, paddedLine)
	}
	panelWithBorder = append(panelWithBorder, "└───────────────────────────────────────────────────────────────────┘")

	// Replace the appropriate lines
	result := make([]string, len(lines))
	copy(result, lines)

	for i, panelLine := range panelWithBorder {
		lineIndex := startLine + i
		if lineIndex < len(result) {
			result[lineIndex] = panelLine
		}
	}

	return strings.Join(result, "\n")
}
