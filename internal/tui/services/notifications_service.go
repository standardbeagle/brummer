package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
)

// Error constants for Notification Service
var (
	ErrInvalidNotificationDuration    = errors.New("INVALID_NOTIFICATION_DURATION")
	ErrNotificationNotFound           = errors.New("NOTIFICATION_NOT_FOUND")
	ErrNotificationLimitExceeded      = errors.New("NOTIFICATION_LIMIT_EXCEEDED")
	ErrInvalidNotificationLevel       = errors.New("INVALID_NOTIFICATION_LEVEL")
	ErrNotificationServiceUnavailable = errors.New("NOTIFICATION_SERVICE_UNAVAILABLE")
)

// Notification represents a toast notification with display properties
type Notification struct {
	ID       string                    `json:"id"`
	Title    string                    `json:"title"`
	Message  string                    `json:"message"`
	Level    messages.NotificationLevel `json:"level"`
	Duration *time.Duration            `json:"duration,omitempty"` // nil for persistent, otherwise auto-dismiss
	Created  time.Time                 `json:"created"`
	Expires  *time.Time                `json:"expires,omitempty"`
}

// NotificationsConfig contains configuration for the notifications service
type NotificationsConfig struct {
	MaxVisible     int           `toml:"max_visible"`
	DefaultDuration time.Duration `toml:"default_duration"`
	Position       string        `toml:"position"` // "top-right", "bottom-right", "top-left", "bottom-left"
}

// DefaultNotificationsConfig returns default configuration for the notifications service
func DefaultNotificationsConfig() NotificationsConfig {
	return NotificationsConfig{
		MaxVisible:     3,
		DefaultDuration: 5 * time.Second,
		Position:       "top-right",
	}
}

// NotificationsService manages toast notification display and lifecycle
// Integrates with ServiceBridge to provide notification events and UI updates
type NotificationsService struct {
	// Notification storage
	activeNotifications sync.Map // map[string]*Notification

	// Configuration
	config    atomic.Value // NotificationsConfig
	maxVisible atomic.Int32 // Maximum simultaneous notifications
	position  atomic.Value // string (notification position)

	// Core components
	eventBus *events.EventBus

	// Lifecycle management
	ctx            context.Context
	cancel         context.CancelFunc
	cleanupWg      sync.WaitGroup
	shutdownCh     chan struct{}
	shutdownInitiated atomic.Bool

	// Metrics
	totalShown   atomic.Uint64
	totalCleared atomic.Uint64
	totalExpired atomic.Uint64
}

// NewNotificationsService creates a new notifications service
func NewNotificationsService(eventBus *events.EventBus) *NotificationsService {
	ctx, cancel := context.WithCancel(context.Background())

	ns := &NotificationsService{
		eventBus:    eventBus,
		ctx:         ctx,
		cancel:      cancel,
		shutdownCh:  make(chan struct{}),
	}

	// Initialize atomic values with defaults
	ns.config.Store(DefaultNotificationsConfig())
	ns.maxVisible.Store(3)
	ns.position.Store("top-right")

	// Start cleanup goroutine
	ns.cleanupWg.Add(1)
	go ns.cleanupExpiredNotifications()

	return ns
}

// ShowNotification displays a notification with default duration
func (ns *NotificationsService) ShowNotification(ctx context.Context, notification messages.Notification) error {
	// Validate notification
	if err := ns.validateNotification(notification); err != nil {
		return err
	}

	// Validate duration field if provided
	if notification.Duration != nil {
		if *notification.Duration <= 0 {
			return ErrInvalidNotificationDuration
		}
		// Use the provided duration
		return ns.showNotificationWithConfig(ctx, notification, *notification.Duration)
	}

	// Set default duration if not provided
	duration := ns.config.Load().(NotificationsConfig).DefaultDuration
	return ns.showNotificationWithConfig(ctx, notification, duration)
}

// ShowNotificationWithDuration displays a notification with custom duration
func (ns *NotificationsService) ShowNotificationWithDuration(ctx context.Context, notification messages.Notification, duration time.Duration) error {
	// Validate notification
	if err := ns.validateNotification(notification); err != nil {
		return err
	}

	// Validate duration
	if duration <= 0 {
		return ErrInvalidNotificationDuration
	}

	return ns.showNotificationWithConfig(ctx, notification, duration)
}

// showNotificationWithConfig is the internal implementation for showing notifications
func (ns *NotificationsService) showNotificationWithConfig(ctx context.Context, notification messages.Notification, duration time.Duration) error {
	// Check if service is shutting down
	if ns.shutdownInitiated.Load() {
		return ErrNotificationServiceUnavailable
	}

	// Check notification limit
	if err := ns.enforceNotificationLimit(); err != nil {
		return err
	}

	// Generate ID if not provided
	notificationID := notification.ID
	if notificationID == "" {
		notificationID = uuid.New().String()
	}

	// Create notification record
	now := time.Now()
	var expires *time.Time
	if duration > 0 {
		expTime := now.Add(duration)
		expires = &expTime
	}

	notif := &Notification{
		ID:       notificationID,
		Title:    notification.Title,
		Message:  notification.Message,
		Level:    notification.Level,
		Duration: &duration,
		Created:  now,
		Expires:  expires,
	}

	// Store notification
	ns.activeNotifications.Store(notificationID, notif)

	// Update metrics
	ns.totalShown.Add(1)

	// Publish event
	ns.publishNotificationShownEvent(notif)

	return nil
}

// ClearNotification removes a specific notification by ID
func (ns *NotificationsService) ClearNotification(ctx context.Context, notificationID string) error {
	// Check if notification exists
	_, exists := ns.activeNotifications.LoadAndDelete(notificationID)
	if !exists {
		return ErrNotificationNotFound
	}

	// Update metrics
	ns.totalCleared.Add(1)

	// Publish event
	ns.publishNotificationClearedEvent(notificationID, "manual")

	return nil
}

// ClearAllNotifications removes all active notifications
func (ns *NotificationsService) ClearAllNotifications(ctx context.Context) error {
	var clearedCount int

	// Collect all notification IDs
	ns.activeNotifications.Range(func(key, value interface{}) bool {
		ns.activeNotifications.Delete(key)
		clearedCount++
		return true
	})

	// Update metrics
	ns.totalCleared.Add(uint64(clearedCount))

	// Publish event
	ns.publishAllNotificationsClearedEvent(clearedCount, "manual")

	return nil
}

// GetActiveNotifications returns a list of currently active notifications
func (ns *NotificationsService) GetActiveNotifications() []messages.Notification {
	var notifications []messages.Notification

	ns.activeNotifications.Range(func(key, value interface{}) bool {
		notif := value.(*Notification)

		// Convert to message format
		var duration *time.Duration
		if notif.Duration != nil {
			duration = notif.Duration
		}

		notifications = append(notifications, messages.Notification{
			ID:       notif.ID,
			Title:    notif.Title,
			Message:  notif.Message,
			Level:    notif.Level,
			Duration: duration,
		})
		return true
	})

	return notifications
}

// SetPosition updates the notification display position
func (ns *NotificationsService) SetPosition(position string) error {
	oldPosition := ns.position.Load().(string)
	ns.position.Store(position)

	// Publish event
	ns.publishNotificationPositionChangedEvent(oldPosition, position)

	return nil
}

// SetMaxVisible updates the maximum number of simultaneous notifications
func (ns *NotificationsService) SetMaxVisible(maxVisible int) error {
	if maxVisible <= 0 {
		return fmt.Errorf("max visible must be greater than 0")
	}

	ns.maxVisible.Store(int32(maxVisible))

	// Update config
	config := ns.config.Load().(NotificationsConfig)
	config.MaxVisible = maxVisible
	ns.config.Store(config)

	return nil
}

// GetPosition returns the current notification position
func (ns *NotificationsService) GetPosition() string {
	return ns.position.Load().(string)
}

// GetMaxVisible returns the current maximum visible notifications
func (ns *NotificationsService) GetMaxVisible() int {
	return int(ns.maxVisible.Load())
}

// GetServiceName returns the service name for ServiceBridge integration
func (ns *NotificationsService) GetServiceName() string {
	return "notifications"
}

// GetServiceStatus returns the current status of the notifications service
func (ns *NotificationsService) GetServiceStatus() ServiceStatus {
	status := "active"
	details := fmt.Sprintf("max: %d, active: %d", ns.GetMaxVisible(), ns.getActiveNotificationCount())

	if ns.shutdownInitiated.Load() {
		status = "shutting down"
	}

	return ServiceStatus{
		Service: ns.GetServiceName(),
		Status:  status,
		Details: details,
	}
}

// Stop gracefully shuts down the notifications service
func (ns *NotificationsService) Stop() error {
	// Check if already shutdown
	if ns.shutdownInitiated.Swap(true) {
		return nil // Already shutdown, idempotent
	}

	// Cancel context
	ns.cancel()

	// Close shutdown channel
	close(ns.shutdownCh)

	// Clear all notifications
	ns.ClearAllNotifications(context.Background())

	// Wait for cleanup goroutine
	done := make(chan struct{})
	go func() {
		ns.cleanupWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for notifications service shutdown")
	}
}

// SubscribeToEvents subscribes the notifications service to relevant system events
func (ns *NotificationsService) SubscribeToEvents() {
	if ns.eventBus == nil {
		return
	}

	// Notifications service doesn't need to subscribe to external events
	// It only publishes events based on notification actions
}

// Helper methods

func (ns *NotificationsService) validateNotification(notification messages.Notification) error {
	if notification.Title == "" && notification.Message == "" {
		return fmt.Errorf("notification must have either title or message")
	}

	// Validate notification level
	validLevels := map[messages.NotificationLevel]bool{
		messages.InfoNotification:     true,
		messages.SuccessNotification:  true,
		messages.WarningNotification:  true,
		messages.ErrorNotification:    true,
	}

	if !validLevels[notification.Level] {
		return ErrInvalidNotificationLevel
	}

	return nil
}

func (ns *NotificationsService) enforceNotificationLimit() error {
	maxVisible := int(ns.maxVisible.Load())
	currentCount := ns.getActiveNotificationCount()

	if currentCount >= maxVisible {
		// Remove oldest notification to make room
		var oldestID string
		var oldestTime time.Time

		ns.activeNotifications.Range(func(key, value interface{}) bool {
			notif := value.(*Notification)
			if oldestID == "" || notif.Created.Before(oldestTime) {
				oldestID = notif.ID
				oldestTime = notif.Created
			}
			return true
		})

		if oldestID != "" {
			ns.activeNotifications.Delete(oldestID)
			ns.publishNotificationClearedEvent(oldestID, "limit_exceeded")
		}
	}

	return nil
}

func (ns *NotificationsService) getActiveNotificationCount() int {
	count := 0
	ns.activeNotifications.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// cleanupExpiredNotifications runs in a background goroutine to remove expired notifications
func (ns *NotificationsService) cleanupExpiredNotifications() {
	defer ns.cleanupWg.Done()

	ticker := time.NewTicker(1 * time.Second) // Check every second
	defer ticker.Stop()

	for {
		select {
		case <-ns.ctx.Done():
			return
		case <-ns.shutdownCh:
			return
		case <-ticker.C:
			ns.removeExpiredNotifications()
		}
	}
}

func (ns *NotificationsService) removeExpiredNotifications() {
	now := time.Now()
	var expiredIDs []string

	// Find expired notifications
	ns.activeNotifications.Range(func(key, value interface{}) bool {
		notif := value.(*Notification)
		if notif.Expires != nil && now.After(*notif.Expires) {
			expiredIDs = append(expiredIDs, notif.ID)
		}
		return true
	})

	// Remove expired notifications
	for _, id := range expiredIDs {
		if _, exists := ns.activeNotifications.LoadAndDelete(id); exists {
			ns.totalExpired.Add(1)
			ns.publishNotificationClearedEvent(id, "timeout")
		}
	}
}

// Event publishing methods

func (ns *NotificationsService) publishNotificationShownEvent(notification *Notification) {
	if ns.eventBus == nil {
		return
	}

	duration := ns.config.Load().(NotificationsConfig).DefaultDuration
	if notification.Duration != nil {
		duration = *notification.Duration
	}

	ns.eventBus.Publish(events.Event{
		Type: "notification.shown",
		Data: map[string]interface{}{
			"notification_id": notification.ID,
			"title":          notification.Title,
			"message":        notification.Message,
			"level":          string(notification.Level),
			"duration":       duration,
			"position":       ns.GetPosition(),
			"time":           notification.Created,
		},
	})
}

func (ns *NotificationsService) publishNotificationClearedEvent(notificationID, trigger string) {
	if ns.eventBus == nil {
		return
	}

	ns.eventBus.Publish(events.Event{
		Type: "notification.cleared",
		Data: map[string]interface{}{
			"notification_id": notificationID,
			"trigger":         trigger,
			"time":            time.Now(),
		},
	})
}

func (ns *NotificationsService) publishAllNotificationsClearedEvent(clearedCount int, trigger string) {
	if ns.eventBus == nil {
		return
	}

	ns.eventBus.Publish(events.Event{
		Type: "notifications.all_cleared",
		Data: map[string]interface{}{
			"cleared_count": clearedCount,
			"trigger":       trigger,
			"time":          time.Now(),
		},
	})
}

func (ns *NotificationsService) publishNotificationPositionChangedEvent(oldPosition, newPosition string) {
	if ns.eventBus == nil {
		return
	}

	ns.eventBus.Publish(events.Event{
		Type: "notification.position_changed",
		Data: map[string]interface{}{
			"old_position": oldPosition,
			"new_position": newPosition,
			"time":         time.Now(),
		},
	})
}