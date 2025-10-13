package messages

import (
	"time"
)

// Notification Messages handles toast notification display and lifecycle communication
// between NotificationsService and NotificationsComponent

// NotificationShownMsg is sent when a new notification is displayed
type NotificationShownMsg struct {
	NotificationID string            `json:"notification_id"`
	Message        string            `json:"message"`
	Level          NotificationLevel `json:"level"`
	Duration       time.Duration     `json:"duration"`
	Position       string            `json:"position"`
	Time           time.Time         `json:"time"`
}

// NotificationClearedMsg is sent when a notification is cleared
type NotificationClearedMsg struct {
	NotificationID string    `json:"notification_id"`
	Trigger        string    `json:"trigger"` // "timeout", "manual", "system"
	Time           time.Time `json:"time"`
}

// AllNotificationsClearedMsg is sent when all notifications are cleared
type AllNotificationsClearedMsg struct {
	ClearedCount int       `json:"cleared_count"`
	Trigger      string    `json:"trigger"`
	Time         time.Time `json:"time"`
}

// NotificationPositionChangedMsg is sent when notification position changes
type NotificationPositionChangedMsg struct {
	OldPosition string    `json:"old_position"`
	NewPosition string    `json:"new_position"`
	Time        time.Time `json:"time"`
}

// Notification represents a toast notification
type Notification struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Message  string          `json:"message"`
	Level    NotificationLevel `json:"level"`
	Duration *time.Duration  `json:"duration,omitempty"` // nil for persistent, otherwise auto-dismiss
}

// NotificationLevel represents the severity level of notifications
type NotificationLevel string

const (
	InfoNotification     NotificationLevel = "info"
	SuccessNotification  NotificationLevel = "success"
	WarningNotification  NotificationLevel = "warning"
	ErrorNotification    NotificationLevel = "error"
)