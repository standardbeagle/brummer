package notifications

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Controller manages notification state and display
type Controller struct {
	message          string
	notificationTime time.Time
	duration         time.Duration
}

// NewController creates a new notifications controller
func NewController() *Controller {
	return &Controller{
		duration: 3 * time.Second, // Default notification duration
	}
}

// SetDuration sets the default notification duration
func (c *Controller) SetDuration(d time.Duration) {
	c.duration = d
}

// Show displays a notification message
func (c *Controller) Show(message string) tea.Cmd {
	c.message = message
	c.notificationTime = time.Now()

	// Return a command to clear the notification after duration
	return tea.Tick(c.duration, func(t time.Time) tea.Msg {
		return clearNotificationMsg{clearTime: c.notificationTime}
	})
}

// ShowWithDuration displays a notification with a custom duration
func (c *Controller) ShowWithDuration(message string, duration time.Duration) tea.Cmd {
	c.message = message
	c.notificationTime = time.Now()

	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return clearNotificationMsg{clearTime: c.notificationTime}
	})
}

// Clear clears the current notification
func (c *Controller) Clear() {
	c.message = ""
	c.notificationTime = time.Time{}
}

// HandleMsg handles notification-related messages
func (c *Controller) HandleMsg(msg tea.Msg) tea.Cmd {
	if clearMsg, ok := msg.(clearNotificationMsg); ok {
		// Only clear if this is for the current notification
		if clearMsg.clearTime.Equal(c.notificationTime) {
			c.Clear()
		}
	}
	return nil
}

// GetMessage returns the current notification message
func (c *Controller) GetMessage() string {
	return c.message
}

// IsActive returns whether a notification is currently displayed
func (c *Controller) IsActive() bool {
	if c.message == "" {
		return false
	}

	// Also check if notification has expired
	if time.Since(c.notificationTime) > c.duration {
		c.Clear()
		return false
	}

	return true
}

// clearNotificationMsg is sent to clear a notification
type clearNotificationMsg struct {
	clearTime time.Time
}

// Export the message type for use in the main update loop
type ClearNotificationMsg = clearNotificationMsg
