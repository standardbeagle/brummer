package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NotificationComponent displays toast-style notifications.
//
// Design principles:
// - Non-blocking overlay display
// - Auto-dismiss after timeout
// - Type-based styling (info, success, warning, error)
// - Queue management for multiple notifications
// - Positioned at top or bottom
type NotificationComponent struct {
	BaseComponent

	notifications []Notification
	position      NotificationPosition
	maxVisible    int

	// Styles
	infoStyle    lipgloss.Style
	successStyle lipgloss.Style
	warningStyle lipgloss.Style
	errorStyle   lipgloss.Style
}

// NotificationType represents the notification severity
type NotificationType string

const (
	NotificationInfo    NotificationType = "info"
	NotificationSuccess NotificationType = "success"
	NotificationWarning NotificationType = "warning"
	NotificationError   NotificationType = "error"
)

// NotificationPosition represents where notifications appear
type NotificationPosition string

const (
	PositionTop    NotificationPosition = "top"
	PositionBottom NotificationPosition = "bottom"
)

// Notification represents a single notification
type Notification struct {
	ID        string
	Type      NotificationType
	Message   string
	CreatedAt time.Time
	Duration  time.Duration
}

// NotificationExpiredMsg signals a notification should be removed
type NotificationExpiredMsg struct {
	ID string
}

// NewNotificationComponent creates a notification system
func NewNotificationComponent(position NotificationPosition) *NotificationComponent {
	c := &NotificationComponent{
		BaseComponent: NewBaseComponent(),
		notifications: []Notification{},
		position:      position,
		maxVisible:    3,
		infoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // Bright Blue
			Background(lipgloss.Color("236")). // Dark Gray
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")),
		successStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Bright Green
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")),
		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Bright Yellow
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("11")),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Bright Red
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")),
	}

	return c
}

// Init initializes the component
func (c *NotificationComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *NotificationComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil

	case NotificationExpiredMsg:
		c.RemoveNotification(msg.ID)
		return c, nil
	}

	return c, nil
}

// View renders visible notifications
func (c *NotificationComponent) View() string {
	if !c.IsVisible() || len(c.notifications) == 0 {
		return ""
	}

	var lines []string

	// Get visible notifications
	visible := c.notifications
	if len(visible) > c.maxVisible {
		visible = visible[:c.maxVisible]
	}

	// Render each notification
	for _, notification := range visible {
		rendered := c.renderNotification(notification)
		lines = append(lines, rendered)
	}

	return strings.Join(lines, "\n")
}

// renderNotification renders a single notification
func (c *NotificationComponent) renderNotification(notification Notification) string {
	var icon string
	var style lipgloss.Style

	switch notification.Type {
	case NotificationInfo:
		icon = "ℹ"
		style = c.infoStyle
	case NotificationSuccess:
		icon = "✓"
		style = c.successStyle
	case NotificationWarning:
		icon = "⚠"
		style = c.warningStyle
	case NotificationError:
		icon = "✗"
		style = c.errorStyle
	default:
		icon = "•"
		style = c.infoStyle
	}

	content := fmt.Sprintf("%s %s", icon, notification.Message)
	return style.Render(content)
}

// SetSize updates component dimensions
func (c *NotificationComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)

	// Update notification widths
	maxWidth := width - 4 // Account for padding/borders
	if maxWidth > 60 {
		maxWidth = 60 // Max notification width
	}

	c.infoStyle = c.infoStyle.Width(maxWidth)
	c.successStyle = c.successStyle.Width(maxWidth)
	c.warningStyle = c.warningStyle.Width(maxWidth)
	c.errorStyle = c.errorStyle.Width(maxWidth)
}

// AddNotification adds a new notification with auto-dismiss
func (c *NotificationComponent) AddNotification(notifType NotificationType, message string, duration time.Duration) tea.Cmd {
	id := fmt.Sprintf("%d", time.Now().UnixNano())

	notification := Notification{
		ID:        id,
		Type:      notifType,
		Message:   message,
		CreatedAt: time.Now(),
		Duration:  duration,
	}

	c.notifications = append([]Notification{notification}, c.notifications...)

	// Return command for auto-dismiss
	if duration > 0 {
		return func() tea.Msg {
			time.Sleep(duration)
			return NotificationExpiredMsg{ID: id}
		}
	}

	return nil
}

// RemoveNotification removes a notification by ID
func (c *NotificationComponent) RemoveNotification(id string) {
	for i, notification := range c.notifications {
		if notification.ID == id {
			c.notifications = append(c.notifications[:i], c.notifications[i+1:]...)
			break
		}
	}
}

// ClearAll removes all notifications
func (c *NotificationComponent) ClearAll() {
	c.notifications = []Notification{}
}

// Count returns the number of active notifications
func (c *NotificationComponent) Count() int {
	return len(c.notifications)
}

// SetMaxVisible sets how many notifications to show simultaneously
func (c *NotificationComponent) SetMaxVisible(max int) {
	c.maxVisible = max
}

// Info shows an info notification
func (c *NotificationComponent) Info(message string) tea.Cmd {
	return c.AddNotification(NotificationInfo, message, 3*time.Second)
}

// Success shows a success notification
func (c *NotificationComponent) Success(message string) tea.Cmd {
	return c.AddNotification(NotificationSuccess, message, 3*time.Second)
}

// Warning shows a warning notification
func (c *NotificationComponent) Warning(message string) tea.Cmd {
	return c.AddNotification(NotificationWarning, message, 5*time.Second)
}

// Error shows an error notification
func (c *NotificationComponent) Error(message string) tea.Cmd {
	return c.AddNotification(NotificationError, message, 7*time.Second)
}
