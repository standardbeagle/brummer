package contracts_test

import (
	"context"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// These tests validate the Notification Service API contract
// They should fail initially and pass after implementation

func TestNotificationsService_ShowNotification(t *testing.T) {
	// Test basic notification display functionality

	ctx := context.Background()
	notification := messages.Notification{
		ID:      "test-001",
		Title:   "Test Notification",
		Message: "This is a test notification",
		Level:   messages.InfoNotification,
	}

	// Create service
	service := NewNotificationsService()
	err := service.ShowNotification(ctx, notification)

	if err != nil {
		t.Fatalf("Expected ShowNotification to work, got error: %v", err)
	}

	// Verify notification is displayed
	notifications := service.GetActiveNotifications()
	found := false
	for _, n := range notifications {
		if n.ID == notification.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected notification to be in active notifications list")
	}
}

func TestNotificationsService_ShowNotificationWithDuration(t *testing.T) {
	// Test notification display with custom duration

	ctx := context.Background()
	duration := 5 * time.Second
	notification := messages.Notification{
		ID:       "test-002",
		Title:    "Temporary Notification",
		Message:  "This notification should disappear after 5 seconds",
		Level:    messages.WarningNotification,
		Duration: &duration,
	}

	// Create service
	service := NewNotificationsService()
	err := service.ShowNotificationWithDuration(ctx, notification, duration)

	if err != nil {
		t.Fatalf("Expected ShowNotificationWithDuration to work, got error: %v", err)
	}

	// Verify notification is displayed with correct duration
	notifications := service.GetActiveNotifications()
	found := false
	for _, n := range notifications {
		if n.ID == notification.ID && n.Duration != nil && *n.Duration == duration {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected notification to be displayed with correct duration")
	}

	// Wait for notification to expire (in real implementation, this would use time.Sleep)
	// For now, just verify the duration is set correctly
}

func TestNotificationsService_ClearNotification(t *testing.T) {
	// Test clearing a specific notification

	ctx := context.Background()
	notificationID := "test-003"

	// First, show a notification
	service := NewNotificationsService()
	notification := messages.Notification{
		ID:      notificationID,
		Title:   "To Be Cleared",
		Message: "This notification will be cleared",
		Level:   messages.InfoNotification,
	}
	err1 := service.ShowNotification(ctx, notification)
	if err1 != nil {
		t.Fatalf("Expected ShowNotification to work, got error: %v", err1)
	}

	// Verify notification is displayed before clearing
	notificationsBefore := service.GetActiveNotifications()
	foundBefore := false
	for _, n := range notificationsBefore {
		if n.ID == notificationID {
			foundBefore = true
			break
		}
	}
	if !foundBefore {
		t.Fatalf("Expected notification to be displayed before clearing")
	}

	// Clear the notification
	err2 := service.ClearNotification(ctx, notificationID)
	if err2 != nil {
		t.Fatalf("Expected ClearNotification to work, got error: %v", err2)
	}

	// Verify notification is cleared
	notificationsAfter := service.GetActiveNotifications()
	foundAfter := false
	for _, n := range notificationsAfter {
		if n.ID == notificationID {
			foundAfter = true
			break
		}
	}
	if foundAfter {
		t.Fatalf("Expected notification to be cleared from active notifications list")
	}
}

func TestNotificationsService_ClearAllNotifications(t *testing.T) {
	// Test clearing all notifications

	ctx := context.Background()

	// First, show multiple notifications
	service := NewNotificationsService()
	notifications := []messages.Notification{
		{ID: "test-004", Title: "First", Message: "First notification", Level: messages.InfoNotification},
		{ID: "test-005", Title: "Second", Message: "Second notification", Level: messages.WarningNotification},
		{ID: "test-006", Title: "Third", Message: "Third notification", Level: messages.ErrorNotification},
	}

	for _, notif := range notifications {
		err := service.ShowNotification(ctx, notif)
		if err != nil {
			t.Fatalf("Expected ShowNotification to work, got error: %v", err)
		}
	}

	// Verify notifications are displayed before clearing
	notificationsBefore := service.GetActiveNotifications()
	if len(notificationsBefore) != len(notifications) {
		t.Fatalf("Expected %d notifications before clearing, got %d", len(notifications), len(notificationsBefore))
	}

	// Clear all notifications
	err := service.ClearAllNotifications(ctx)
	if err != nil {
		t.Fatalf("Expected ClearAllNotifications to work, got error: %v", err)
	}

	// Verify all notifications are cleared
	notificationsAfter := service.GetActiveNotifications()
	if len(notificationsAfter) != 0 {
		t.Fatalf("Expected no notifications after clearing all, got %d", len(notificationsAfter))
	}
}

func TestNotificationsService_TooManyNotifications(t *testing.T) {
	// Test handling of too many notifications (should enforce limits)

	ctx := context.Background()
	maxNotifications := 10 // This should be configurable in implementation

	// Attempt to show more notifications than allowed
	service := NewNotificationsService()
	service.SetMaxVisible(maxNotifications) // Configure the limit

	// Show max allowed notifications
	for i := 0; i < maxNotifications; i++ {
		notification := messages.Notification{
			ID:      "test-limit-" + string(rune('a'+i)),
			Title:   "Notification " + string(rune('A'+i)),
			Message: "This is notification " + string(rune('a'+i)),
			Level:   messages.InfoNotification,
		}
		err := service.ShowNotification(ctx, notification)
		if err != nil {
			t.Fatalf("Expected ShowNotification to work for notification %d, got error: %v", i, err)
		}
	}

	// Verify we have max notifications
	notifications := service.GetActiveNotifications()
	if len(notifications) != maxNotifications {
		t.Fatalf("Expected %d notifications, got %d", maxNotifications, len(notifications))
	}

	// Try to add one more notification - should fail or replace oldest
	extraNotification := messages.Notification{
		ID:      "test-extra",
		Title:   "Extra Notification",
		Message: "This should trigger limit handling",
		Level:   messages.WarningNotification,
	}
	_ = service.ShowNotification(ctx, extraNotification) // err ignored for limit test

	// Implementation should either fail with error or replace oldest notification
	notificationsAfter := service.GetActiveNotifications()
	if len(notificationsAfter) > maxNotifications {
		t.Fatalf("Expected notification limit of %d to be enforced, but got %d notifications",
			maxNotifications, len(notificationsAfter))
	}

	// If we get here without error, the notification limit is being enforced
	// In actual implementation, we'd check specific behavior (oldest removed, etc.)
}

func TestNotificationsService_InvalidDuration(t *testing.T) {
	// Test handling of invalid notification duration

	ctx := context.Background()

	// Test with negative duration
	negativeDuration := -5 * time.Second
	notification := messages.Notification{
		ID:       "test-invalid-duration",
		Title:    "Invalid Duration",
		Message:  "This notification has invalid duration",
		Level:    messages.InfoNotification,
		Duration: &negativeDuration,
	}

	// Attempt to show notification with invalid duration - should fail
	service := NewNotificationsService()
	err := service.ShowNotification(ctx, notification)

	if err == nil {
		t.Fatalf("Expected ShowNotification to fail with negative duration")
	}
	if err != services.ErrInvalidNotificationDuration {
		t.Fatalf("Expected invalid duration error, got '%v'", err)
	}

	// Test with zero duration
	zeroDuration := 0 * time.Second
	notification.Duration = &zeroDuration

	err2 := service.ShowNotification(ctx, notification)
	if err2 == nil {
		t.Fatalf("Expected ShowNotification to fail with zero duration")
	}
	if err2 != services.ErrInvalidNotificationDuration {
		t.Fatalf("Expected invalid duration error, got '%v'", err2)
	}
}

func TestNotificationsService_NotificationNotFound(t *testing.T) {
	// Test clearing a notification that doesn't exist

	ctx := context.Background()
	nonExistentID := "non-existent-notification"

	// Attempt to clear non-existent notification - should fail
	service := NewNotificationsService()
	err := service.ClearNotification(ctx, nonExistentID)

	if err == nil {
		t.Fatalf("Expected ClearNotification to fail for non-existent notification")
	}
	if err != services.ErrNotificationNotFound {
		t.Fatalf("Expected notification not found error, got '%v'", err)
	}
}

func TestNotificationsService_Performance(t *testing.T) {
	// Test notification service performance meets <5ms target

	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()

	// Test notification service performance
	service := NewNotificationsService()

	// Measure notification creation and display time
	start := time.Now()
	for i := 0; i < 1000; i++ {
		notification := messages.Notification{
			ID:      "perf-test-" + string(rune(i)),
			Title:   "Performance Test",
			Message: "This is a performance test notification",
			Level:   messages.InfoNotification,
		}
		err := service.ShowNotification(ctx, notification)
		if err != nil {
			t.Fatalf("Expected ShowNotification to work during performance test, got error: %v", err)
		}

		// Clear it to avoid hitting limits
		err2 := service.ClearNotification(ctx, notification.ID)
		if err2 != nil {
			t.Fatalf("Expected ClearNotification to work during performance test, got error: %v", err2)
		}
	}
	duration := time.Since(start)

	avgTime := duration / 1000
	if avgTime >= 5*time.Millisecond {
		t.Fatalf("Average notification operation time %v is >= 5ms target", avgTime)
	}

	// Test retrieval performance
	start2 := time.Now()
	for i := 0; i < 1000; i++ {
		notifications := service.GetActiveNotifications()
		_ = notifications // Just to ensure the call is made
	}
	duration2 := time.Since(start2)

	avgTime2 := duration2 / 1000
	if avgTime2 >= 1*time.Millisecond {
		t.Fatalf("Average GetActiveNotifications time %v is >= 1ms target", avgTime2)
	}
}

// NewNotificationsService creates a new notification service instance
func NewNotificationsService() *services.NotificationsService {
	eventBus := events.NewEventBus()
	return services.NewNotificationsService(eventBus)
}