package components

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNotificationComponent_NewComponent(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(80, 24)

	if component.Count() != 0 {
		t.Errorf("Expected 0 initial notifications, got %d", component.Count())
	}

	view := component.View()
	if view != "" {
		t.Error("View should be empty with no notifications")
	}
}

func TestNotificationComponent_AddNotification(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(80, 24)

	component.AddNotification(NotificationInfo, "Test message", 0)

	if component.Count() != 1 {
		t.Errorf("Expected 1 notification, got %d", component.Count())
	}

	view := component.View()
	if !strings.Contains(view, "Test message") {
		t.Error("View should contain notification message")
	}
}

func TestNotificationComponent_NotificationTypes(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(200, 24) // Wide viewport
	component.SetMaxVisible(4) // Show all notifications

	component.AddNotification(NotificationInfo, "Info message", 0)
	component.AddNotification(NotificationSuccess, "Success message", 0)
	component.AddNotification(NotificationWarning, "Warning message", 0)
	component.AddNotification(NotificationError, "Error message", 0)

	if component.Count() != 4 {
		t.Errorf("Expected 4 notifications, got %d", component.Count())
	}

	view := component.View()
	// Check for messages (icons may be styled/invisible in test)
	if !strings.Contains(view, "Info message") {
		t.Error("View should contain info message")
	}
	if !strings.Contains(view, "Success message") {
		t.Error("View should contain success message")
	}
	if !strings.Contains(view, "Warning message") {
		t.Error("View should contain warning message")
	}
	if !strings.Contains(view, "Error message") {
		t.Error("View should contain error message")
	}
}

func TestNotificationComponent_ConvenienceMethods(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(200, 24)
	component.SetMaxVisible(4) // Show all notifications

	component.Info("Info test")
	component.Success("Success test")
	component.Warning("Warning test")
	component.Error("Error test")

	if component.Count() != 4 {
		t.Errorf("Expected 4 notifications, got %d", component.Count())
	}

	view := component.View()
	// Check for all messages since we increased maxVisible
	if !strings.Contains(view, "Info test") {
		t.Error("View should contain info message")
	}
	if !strings.Contains(view, "Success test") {
		t.Error("View should contain success message")
	}
	if !strings.Contains(view, "Warning test") {
		t.Error("View should contain warning message")
	}
	if !strings.Contains(view, "Error test") {
		t.Error("View should contain error message")
	}
}

func TestNotificationComponent_RemoveNotification(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(80, 24)

	cmd := component.AddNotification(NotificationInfo, "Test message", 0)
	if cmd != nil {
		t.Error("Should not return command for notification with 0 duration")
	}

	if component.Count() != 1 {
		t.Errorf("Expected 1 notification, got %d", component.Count())
	}

	// Remove the notification
	if len(component.notifications) > 0 {
		component.RemoveNotification(component.notifications[0].ID)
	}

	if component.Count() != 0 {
		t.Errorf("Expected 0 notifications after removal, got %d", component.Count())
	}
}

func TestNotificationComponent_ClearAll(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(80, 24)

	component.AddNotification(NotificationInfo, "Message 1", 0)
	component.AddNotification(NotificationInfo, "Message 2", 0)
	component.AddNotification(NotificationInfo, "Message 3", 0)

	if component.Count() != 3 {
		t.Errorf("Expected 3 notifications, got %d", component.Count())
	}

	component.ClearAll()

	if component.Count() != 0 {
		t.Errorf("Expected 0 notifications after clear, got %d", component.Count())
	}

	view := component.View()
	if view != "" {
		t.Error("View should be empty after clearing all notifications")
	}
}

func TestNotificationComponent_MaxVisible(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(80, 24)
	component.SetMaxVisible(2)

	component.AddNotification(NotificationInfo, "Message 1", 0)
	component.AddNotification(NotificationInfo, "Message 2", 0)
	component.AddNotification(NotificationInfo, "Message 3", 0)

	if component.Count() != 3 {
		t.Errorf("Expected 3 total notifications, got %d", component.Count())
	}

	view := component.View()
	// Should only show 2 notifications
	lines := strings.Split(view, "\n")
	visibleCount := 0
	for _, line := range lines {
		if strings.Contains(line, "Message") {
			visibleCount++
		}
	}

	if visibleCount > 2 {
		t.Errorf("Expected max 2 visible notifications, got %d", visibleCount)
	}
}

func TestNotificationComponent_AutoDismiss(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(80, 24)

	// Add notification with auto-dismiss
	cmd := component.AddNotification(NotificationInfo, "Auto-dismiss test", 10*time.Millisecond)

	if cmd == nil {
		t.Error("Should return command for auto-dismiss notification")
	}

	if component.Count() != 1 {
		t.Errorf("Expected 1 notification, got %d", component.Count())
	}

	// Execute the dismiss command (simulating time passage)
	if cmd != nil {
		msg := cmd()
		if expiredMsg, ok := msg.(NotificationExpiredMsg); ok {
			component.Update(expiredMsg)
		}
	}

	// After processing expired message, count should be 0
	if component.Count() != 0 {
		t.Errorf("Expected 0 notifications after auto-dismiss, got %d", component.Count())
	}
}

func TestNotificationComponent_SetSize(t *testing.T) {
	component := NewNotificationComponent(PositionTop)

	component.SetSize(100, 24)

	if component.Width() != 100 {
		t.Errorf("Expected width 100, got %d", component.Width())
	}
	if component.Height() != 24 {
		t.Errorf("Expected height 24, got %d", component.Height())
	}
}

func TestNotificationComponent_Visibility(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(80, 24)

	component.AddNotification(NotificationInfo, "Test message", 0)

	// Default is visible
	if !component.IsVisible() {
		t.Error("Component should be visible by default")
	}

	view := component.View()
	if !strings.Contains(view, "Test message") {
		t.Error("View should show notification when visible")
	}

	// Test hiding
	component.SetVisible(false)
	if component.IsVisible() {
		t.Error("Component should be hidden")
	}

	// View should be empty when hidden
	view = component.View()
	if view != "" {
		t.Error("Hidden component should return empty view")
	}
}

func TestNotificationComponent_WindowSizeMsg(t *testing.T) {
	component := NewNotificationComponent(PositionTop)
	component.SetSize(80, 24)

	component.AddNotification(NotificationInfo, "Test message", 0)

	// Simulate window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 30}
	component.Update(msg)

	if component.Width() != 120 {
		t.Errorf("Expected width 120 after resize, got %d", component.Width())
	}
	if component.Height() != 30 {
		t.Errorf("Expected height 30 after resize, got %d", component.Height())
	}

	// Verify view still renders
	view := component.View()
	if view == "" {
		t.Error("View should not be empty after resize")
	}
}

func TestNotificationComponent_Position(t *testing.T) {
	top := NewNotificationComponent(PositionTop)
	bottom := NewNotificationComponent(PositionBottom)

	if top.position != PositionTop {
		t.Error("Top component should have PositionTop")
	}
	if bottom.position != PositionBottom {
		t.Error("Bottom component should have PositionBottom")
	}
}
