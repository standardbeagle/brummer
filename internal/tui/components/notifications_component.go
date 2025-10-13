package components

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// NotificationsComponent manages toast notifications and display state
// This component migrates functionality from the old NotificationController
// to use the new service-based architecture
type NotificationsComponent struct {
	BaseComponent

	// Service integration
	notificationsService *services.NotificationsService

	// Component state
	activeNotifications []messages.Notification
	position            string
	maxVisible          int
	displayQueue        []messages.Notification

	// Event subscription
	subscriptionCh chan events.Event

	// Performance tracking
	lastNotificationTime time.Time
	notificationCount    int

	// Rendering
	styles map[messages.NotificationLevel]lipgloss.Style
}

// NewNotificationsComponent creates a new notifications component
func NewNotificationsComponent(notificationsService *services.NotificationsService) *NotificationsComponent {
	if notificationsService == nil {
		notificationsService = services.NewNotificationsService(events.NewEventBus())
	}

	component := &NotificationsComponent{
		BaseComponent:        NewBaseComponent(),
		notificationsService: notificationsService,
		subscriptionCh:       make(chan events.Event, 100),
		activeNotifications:  make([]messages.Notification, 0),
		displayQueue:         make([]messages.Notification, 0),
		lastNotificationTime: time.Now(),
		notificationCount:    0,
		styles:               make(map[messages.NotificationLevel]lipgloss.Style),
	}

	// Initialize component state from service
	component.syncFromService()

	// Initialize styles
	component.initializeStyles()

	// Subscribe to notifications service events
	component.subscribeToEvents()

	return component
}

// Init initializes the component
func (nc *NotificationsComponent) Init() tea.Cmd {
	// Get initial state from service
	return func() tea.Msg {
		// Get initial active notifications
		notifications := nc.notificationsService.GetActiveNotifications()
		nc.activeNotifications = notifications
		nc.displayQueue = notifications

		// Get initial position
		position := nc.notificationsService.GetPosition()
		nc.position = position

		// Get initial max visible
		maxVisible := nc.notificationsService.GetMaxVisible()
		nc.maxVisible = maxVisible

		return messages.ComponentReadyMsg{
			ComponentID: "notifications",
			Time:        time.Now(),
		}
	}
}

// Update handles incoming messages and notification events
func (nc *NotificationsComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.NotificationShownMsg:
		return nc.handleNotificationShown(msg)

	case messages.NotificationClearedMsg:
		return nc.handleNotificationCleared(msg)

	case messages.AllNotificationsClearedMsg:
		return nc.handleAllNotificationsCleared(msg)

	case messages.NotificationPositionChangedMsg:
		return nc.handleNotificationPositionChanged(msg)

	case tea.KeyMsg:
		return nc.handleKeyMsg(msg)

	case events.Event:
		return nc.handleServiceEvent(msg)

	default:
		return nc, nil
	}
}

// View renders the notifications component
func (nc *NotificationsComponent) View() string {
	if !nc.IsVisible() || len(nc.displayQueue) == 0 {
		return ""
	}

	// Limit to max visible notifications
	visibleNotifications := nc.displayQueue
	if len(visibleNotifications) > nc.maxVisible {
		visibleNotifications = visibleNotifications[:nc.maxVisible]
	}

	var notificationStrings []string

	// Render each notification
	for _, notification := range visibleNotifications {
		rendered := nc.renderNotification(notification)
		notificationStrings = append(notificationStrings, rendered)
	}

	// Position notifications
	content := lipgloss.JoinVertical(lipgloss.Left, notificationStrings...)

	// Apply positioning based on current position setting
	switch nc.position {
	case "top-right":
		return nc.positionTopRight(content)
	case "top-left":
		return nc.positionTopLeft(content)
	case "bottom-right":
		return nc.positionBottomRight(content)
	case "bottom-left":
		return nc.positionBottomLeft(content)
	default:
		return nc.positionTopRight(content)
	}
}

// handleNotificationShown processes new notification displays
func (nc *NotificationsComponent) handleNotificationShown(msg messages.NotificationShownMsg) (tea.Model, tea.Cmd) {
	notification := messages.Notification{
		ID:       msg.NotificationID,
		Title:    "", // Extract from event data if needed
		Message:  msg.Message,
		Level:    msg.Level,
		Duration: &msg.Duration,
	}

	// Add to active notifications
	nc.activeNotifications = append([]messages.Notification{notification}, nc.activeNotifications...)
	nc.displayQueue = nc.activeNotifications

	// Update tracking
	nc.lastNotificationTime = msg.Time
	nc.notificationCount++

	// Limit to max visible
	if len(nc.displayQueue) > nc.maxVisible {
		nc.displayQueue = nc.displayQueue[:nc.maxVisible]
	}

	return nc, func() tea.Msg {
		return messages.NotificationShownMsg{
			NotificationID: msg.NotificationID,
			Message:        msg.Message,
			Level:          msg.Level,
			Duration:       msg.Duration,
			Position:       msg.Position,
			Time:           msg.Time,
		}
	}
}

// handleNotificationCleared processes notification clearing
func (nc *NotificationsComponent) handleNotificationCleared(msg messages.NotificationClearedMsg) (tea.Model, tea.Cmd) {
	// Remove from active notifications
	for i, notification := range nc.activeNotifications {
		if notification.ID == msg.NotificationID {
			nc.activeNotifications = append(nc.activeNotifications[:i], nc.activeNotifications[i+1:]...)
			break
		}
	}

	// Update display queue
	nc.displayQueue = nc.activeNotifications

	// Limit to max visible
	if len(nc.displayQueue) > nc.maxVisible {
		nc.displayQueue = nc.displayQueue[:nc.maxVisible]
	}

	return nc, func() tea.Msg {
		return messages.NotificationClearedMsg{
			NotificationID: msg.NotificationID,
			Trigger:        msg.Trigger,
			Time:           msg.Time,
		}
	}
}

// handleAllNotificationsCleared processes all notifications clearing
func (nc *NotificationsComponent) handleAllNotificationsCleared(msg messages.AllNotificationsClearedMsg) (tea.Model, tea.Cmd) {
	nc.activeNotifications = make([]messages.Notification, 0)
	nc.displayQueue = make([]messages.Notification, 0)

	return nc, func() tea.Msg {
		return messages.AllNotificationsClearedMsg{
			ClearedCount: msg.ClearedCount,
			Trigger:      msg.Trigger,
			Time:         msg.Time,
		}
	}
}

// handleNotificationPositionChanged processes position changes
func (nc *NotificationsComponent) handleNotificationPositionChanged(msg messages.NotificationPositionChangedMsg) (tea.Model, tea.Cmd) {
	nc.position = msg.NewPosition

	return nc, func() tea.Msg {
		return messages.NotificationPositionChangedMsg{
			OldPosition: msg.OldPosition,
			NewPosition: msg.NewPosition,
			Time:        msg.Time,
		}
	}
}

// handleKeyMsg processes keyboard input for notification operations
func (nc *NotificationsComponent) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		// Clear all notifications on escape
		return nc.clearAllNotifications()

	default:
		// Check for specific key strings
		keyStr := msg.String()
		if keyStr == "esc" {
			return nc.clearAllNotifications()
		}

		return nc, nil
	}
}

// handleServiceEvent handles events from the notifications service
func (nc *NotificationsComponent) handleServiceEvent(event events.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case "notification.shown":
		return nc.handleNotificationShownEvent(event)
	case "notification.cleared":
		return nc.handleNotificationClearedEvent(event)
	case "notifications.all_cleared":
		return nc.handleAllNotificationsClearedEvent(event)
	case "notification.position_changed":
		return nc.handleNotificationPositionChangedEvent(event)
	default:
		return nc, nil
	}
}

// handleNotificationShownEvent handles notification shown events from service
func (nc *NotificationsComponent) handleNotificationShownEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	notificationID, _ := data["notification_id"].(string)
	message, _ := data["message"].(string)
	levelStr, _ := data["level"].(string)
	duration, _ := data["duration"].(time.Duration)
	position, _ := data["position"].(string)

	notification := messages.Notification{
		ID:       notificationID,
		Title:    "", // Extract from event data if needed
		Message:  message,
		Level:    messages.NotificationLevel(levelStr),
		Duration: &duration,
	}

	// Add to active notifications
	nc.activeNotifications = append([]messages.Notification{notification}, nc.activeNotifications...)
	nc.displayQueue = nc.activeNotifications

	// Update tracking
	nc.lastNotificationTime = event.Timestamp
	nc.notificationCount++

	// Update position if provided
	if position != "" {
		nc.position = position
	}

	// Limit to max visible
	if len(nc.displayQueue) > nc.maxVisible {
		nc.displayQueue = nc.displayQueue[:nc.maxVisible]
	}

	return nc, func() tea.Msg {
		return messages.NotificationShownMsg{
			NotificationID: notificationID,
			Message:        message,
			Level:          messages.NotificationLevel(levelStr),
			Duration:       duration,
			Position:       position,
			Time:           event.Timestamp,
		}
	}
}

// handleNotificationClearedEvent handles notification cleared events from service
func (nc *NotificationsComponent) handleNotificationClearedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	notificationID, _ := data["notification_id"].(string)
	trigger, _ := data["trigger"].(string)

	// Remove from active notifications
	for i, notification := range nc.activeNotifications {
		if notification.ID == notificationID {
			nc.activeNotifications = append(nc.activeNotifications[:i], nc.activeNotifications[i+1:]...)
			break
		}
	}

	// Update display queue
	nc.displayQueue = nc.activeNotifications

	return nc, func() tea.Msg {
		return messages.NotificationClearedMsg{
			NotificationID: notificationID,
			Trigger:        trigger,
			Time:           event.Timestamp,
		}
	}
}

// handleAllNotificationsClearedEvent handles all notifications cleared events from service
func (nc *NotificationsComponent) handleAllNotificationsClearedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	clearedCount, _ := data["cleared_count"].(int)
	trigger, _ := data["trigger"].(string)

	nc.activeNotifications = make([]messages.Notification, 0)
	nc.displayQueue = make([]messages.Notification, 0)

	return nc, func() tea.Msg {
		return messages.AllNotificationsClearedMsg{
			ClearedCount: clearedCount,
			Trigger:      trigger,
			Time:         event.Timestamp,
		}
	}
}

// handleNotificationPositionChangedEvent handles position changed events from service
func (nc *NotificationsComponent) handleNotificationPositionChangedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	newPosition, _ := data["new_position"].(string)

	nc.position = newPosition

	return nc, func() tea.Msg {
		return messages.NotificationPositionChangedMsg{
			OldPosition: nc.position, // Previous position
			NewPosition: newPosition,
			Time:        event.Timestamp,
		}
	}
}

// syncFromService synchronizes component state from the service
func (nc *NotificationsComponent) syncFromService() {
	// Get active notifications
	nc.activeNotifications = nc.notificationsService.GetActiveNotifications()
	nc.displayQueue = nc.activeNotifications

	// Get position
	nc.position = nc.notificationsService.GetPosition()

	// Get max visible
	nc.maxVisible = nc.notificationsService.GetMaxVisible()
}

// subscribeToEvents subscribes to notifications service events
func (nc *NotificationsComponent) subscribeToEvents() {
	// Subscribe to notifications service events through the event bus
	// This would typically use the event bus subscription mechanism
	// For now, we'll simulate this with direct channel subscription
}

// initializeStyles sets up the notification styles
func (nc *NotificationsComponent) initializeStyles() {
	nc.styles[messages.InfoNotification] = lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).  // Bright Blue
		Background(lipgloss.Color("236")). // Dark Gray
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12"))

	nc.styles[messages.SuccessNotification] = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // Bright Green
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("10"))

	nc.styles[messages.WarningNotification] = lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")). // Bright Yellow
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11"))

	nc.styles[messages.ErrorNotification] = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")). // Bright Red
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("9"))
}

// renderNotification renders a single notification
func (nc *NotificationsComponent) renderNotification(notification messages.Notification) string {
	style, exists := nc.styles[notification.Level]
	if !exists {
		style = nc.styles[messages.InfoNotification] // Default to info style
	}

	// Create icon based on level
	var icon string
	switch notification.Level {
	case messages.InfoNotification:
		icon = "ℹ"
	case messages.SuccessNotification:
		icon = "✓"
	case messages.WarningNotification:
		icon = "⚠"
	case messages.ErrorNotification:
		icon = "✗"
	default:
		icon = "•"
	}

	// Format message
	if notification.Title != "" && notification.Message != "" {
		content := icon + " " + notification.Title + ": " + notification.Message
		return style.Render(content)
	} else if notification.Title != "" {
		content := icon + " " + notification.Title
		return style.Render(content)
	} else {
		content := icon + " " + notification.Message
		return style.Render(content)
	}
}

// Position helpers

func (nc *NotificationsComponent) positionTopRight(content string) string {
	return lipgloss.Place(
		nc.Width(), nc.Height(),
		lipgloss.Right, lipgloss.Top,
		content,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}

func (nc *NotificationsComponent) positionTopLeft(content string) string {
	return lipgloss.Place(
		nc.Width(), nc.Height(),
		lipgloss.Left, lipgloss.Top,
		content,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}

func (nc *NotificationsComponent) positionBottomRight(content string) string {
	return lipgloss.Place(
		nc.Width(), nc.Height(),
		lipgloss.Right, lipgloss.Bottom,
		content,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}

func (nc *NotificationsComponent) positionBottomLeft(content string) string {
	return lipgloss.Place(
		nc.Width(), nc.Height(),
		lipgloss.Left, lipgloss.Bottom,
		content,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}

// Public API methods

// ShowNotification displays a new notification
func (nc *NotificationsComponent) ShowNotification(title, message string, level messages.NotificationLevel) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		notification := messages.Notification{
			Title:   title,
			Message: message,
			Level:   level,
		}

		err := nc.notificationsService.ShowNotification(ctx, notification)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "NotificationsComponent",
				Message: "Failed to show notification: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.NotificationShownMsg{
			NotificationID: notification.ID,
			Message:        message,
			Level:          level,
			Duration:       5 * time.Second, // Default duration
			Position:       nc.position,
			Time:           time.Now(),
		}
	}
}

// ShowNotificationWithDuration displays a notification with custom duration
func (nc *NotificationsComponent) ShowNotificationWithDuration(title, message string, level messages.NotificationLevel, duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		notification := messages.Notification{
			Title:    title,
			Message:  message,
			Level:    level,
			Duration: &duration,
		}

		err := nc.notificationsService.ShowNotificationWithDuration(ctx, notification, duration)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "NotificationsComponent",
				Message: "Failed to show notification: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.NotificationShownMsg{
			NotificationID: notification.ID,
			Message:        message,
			Level:          level,
			Duration:       duration,
			Position:       nc.position,
			Time:           time.Now(),
		}
	}
}

// ClearNotification removes a specific notification
func (nc *NotificationsComponent) ClearNotification(notificationID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := nc.notificationsService.ClearNotification(ctx, notificationID)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "NotificationsComponent",
				Message: "Failed to clear notification: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.NotificationClearedMsg{
			NotificationID: notificationID,
			Trigger:        "manual",
			Time:           time.Now(),
		}
	}
}

// ClearAllNotifications removes all active notifications
func (nc *NotificationsComponent) ClearAllNotifications() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := nc.notificationsService.ClearAllNotifications(ctx)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "NotificationsComponent",
				Message: "Failed to clear all notifications: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.AllNotificationsClearedMsg{
			ClearedCount: len(nc.activeNotifications),
			Trigger:      "manual",
			Time:         time.Now(),
		}
	}
}

// clearAllNotifications is an internal helper
func (nc *NotificationsComponent) clearAllNotifications() (tea.Model, tea.Cmd) {
	return nc, nc.ClearAllNotifications()
}

// SetPosition updates the notification display position
func (nc *NotificationsComponent) SetPosition(position string) tea.Cmd {
	return func() tea.Msg {
		err := nc.notificationsService.SetPosition(position)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "NotificationsComponent",
				Message: "Failed to set position: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.NotificationPositionChangedMsg{
			OldPosition: nc.position,
			NewPosition: position,
			Time:        time.Now(),
		}
	}
}

// SetMaxVisible updates the maximum number of visible notifications
func (nc *NotificationsComponent) SetMaxVisible(maxVisible int) tea.Cmd {
	return func() tea.Msg {
		err := nc.notificationsService.SetMaxVisible(maxVisible)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "NotificationsComponent",
				Message: "Failed to set max visible: " + err.Error(),
				Time:    time.Now(),
			}
		}

		// Update local state
		nc.maxVisible = maxVisible

		// Update display queue
		if len(nc.displayQueue) > maxVisible {
			nc.displayQueue = nc.displayQueue[:maxVisible]
		}

		return nil
	}
}

// GetActiveNotifications returns current active notifications
func (nc *NotificationsComponent) GetActiveNotifications() []messages.Notification {
	return nc.activeNotifications
}

// GetNotificationCount returns the current notification count
func (nc *NotificationsComponent) GetNotificationCount() int {
	return len(nc.activeNotifications)
}

// GetPosition returns the current position
func (nc *NotificationsComponent) GetPosition() string {
	return nc.position
}

// GetMaxVisible returns the current max visible setting
func (nc *NotificationsComponent) GetMaxVisible() int {
	return nc.maxVisible
}

// GetNotificationStats returns notification processing statistics
func (nc *NotificationsComponent) GetNotificationStats() (int, time.Time) {
	return nc.notificationCount, nc.lastNotificationTime
}

// Focus gives keyboard focus to the notifications component
func (nc *NotificationsComponent) Focus() tea.Cmd {
	nc.BaseComponent.Focus()
	return nil
}

// Blur removes keyboard focus from the notifications component
func (nc *NotificationsComponent) Blur() tea.Cmd {
	nc.BaseComponent.Blur()
	return nil
}

// Cleanup performs cleanup when component is destroyed
func (nc *NotificationsComponent) Cleanup() error {
	return nc.notificationsService.Stop()
}