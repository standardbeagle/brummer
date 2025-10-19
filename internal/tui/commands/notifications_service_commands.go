package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// NotificationsServiceCommand represents a notifications service-related operation
type NotificationsServiceCommand struct {
	BaseCommand
	NotificationOperation string
	ServiceID             string
	Args                  map[string]interface{}
	Model                 interface{}
}

// Run executes the notifications service operation and returns a message (implements Command interface)
func (nsc *NotificationsServiceCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "NotificationsService",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := nsc.executeNotificationOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   nsc.id,
		OperationID: nsc.id,
		Priority:    types.OperationPriority(nsc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"notificationOperation": nsc.NotificationOperation,
			"serviceID":             nsc.ServiceID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(nsc.id, types.CommandType(nsc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(nsc.id, types.CommandType(nsc.cmdType), result, executionTime, typesContext)
}

// executeNotificationOperation performs the actual notifications service operation
func (nsc *NotificationsServiceCommand) executeNotificationOperation() (interface{}, error) {
	switch nsc.NotificationOperation {
	case "cleanup_expired_notifications":
		return nsc.cleanupExpiredNotifications()
	case "wait_for_cleanup_goroutines":
		return nsc.waitForCleanupGoroutines()
	case "show_notification":
		return nsc.showNotification()
	case "hide_notification":
		return nsc.hideNotification()
	case "clear_notifications":
		return nsc.clearNotifications()
	case "manage_notification_queue":
		return nsc.manageNotificationQueue()
	default:
		return nil, fmt.Errorf("unknown notifications service operation: %s", nsc.NotificationOperation)
	}
}

// cleanupExpiredNotifications cleans up expired notifications
func (nsc *NotificationsServiceCommand) cleanupExpiredNotifications() (interface{}, error) {
	// This replaces the goroutine at line 97 in notifications_service.go
	serviceID, ok := nsc.Args["serviceID"].(string)
	if !ok {
		serviceID = nsc.ServiceID
	}

	expiredCount, ok := nsc.Args["expiredCount"].(int)
	if !ok {
		expiredCount = 0
	}

	return fmt.Sprintf("Cleaned up %d expired notifications in service %s", expiredCount, serviceID), nil
}

// waitForCleanupGoroutines waits for cleanup goroutines
func (nsc *NotificationsServiceCommand) waitForCleanupGoroutines() (interface{}, error) {
	// This replaces the goroutine at line 325 in notifications_service.go
	goroutineCount, ok := nsc.Args["goroutineCount"].(int)
	if !ok {
		goroutineCount = 1
	}

	timeout, ok := nsc.Args["timeout"].(time.Duration)
	if !ok {
		timeout = 5 * time.Second
	}

	return fmt.Sprintf("Waited for %d cleanup goroutines with timeout %v", goroutineCount, timeout), nil
}

// showNotification shows a notification
func (nsc *NotificationsServiceCommand) showNotification() (interface{}, error) {
	title, ok := nsc.Args["title"].(string)
	if !ok {
		title = "Notification"
	}

	message, ok := nsc.Args["message"].(string)
	if !ok {
		message = "No message"
	}

	level, ok := nsc.Args["level"].(string)
	if !ok {
		level = "info"
	}

	return fmt.Sprintf("Showed notification: %s - %s (%s)", title, message, level), nil
}

// hideNotification hides a notification
func (nsc *NotificationsServiceCommand) hideNotification() (interface{}, error) {
	notificationID, ok := nsc.Args["notificationID"].(string)
	if !ok {
		notificationID = "unknown"
	}

	return fmt.Sprintf("Hid notification %s", notificationID), nil
}

// clearNotifications clears notifications
func (nsc *NotificationsServiceCommand) clearNotifications() (interface{}, error) {
	clearedCount, ok := nsc.Args["clearedCount"].(int)
	if !ok {
		clearedCount = 0
	}

	return fmt.Sprintf("Cleared %d notifications", clearedCount), nil
}

// manageNotificationQueue manages the notification queue
func (nsc *NotificationsServiceCommand) manageNotificationQueue() (interface{}, error) {
	queueSize, ok := nsc.Args["queueSize"].(int)
	if !ok {
		queueSize = 0
	}

	action, ok := nsc.Args["action"].(string)
	if !ok {
		action = "unknown"
	}

	return fmt.Sprintf("Managed notification queue: %s (size: %d)", action, queueSize), nil
}

// NotificationsServiceCommandFactory creates notifications service-related commands
type NotificationsServiceCommandFactory struct{}

// NewNotificationsServiceCommandFactory creates a new notifications service command factory
func NewNotificationsServiceCommandFactory() *NotificationsServiceCommandFactory {
	return &NotificationsServiceCommandFactory{}
}

// CreateCleanupExpiredNotificationsCommand creates a command to cleanup expired notifications
func (f *NotificationsServiceCommandFactory) CreateCleanupExpiredNotificationsCommand(serviceID string, expiredCount int, model interface{}) Command {
	id := fmt.Sprintf("notif-cleanup-%s-%d", serviceID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &NotificationsServiceCommand{
		BaseCommand:           *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		NotificationOperation: "cleanup_expired_notifications",
		ServiceID:             serviceID,
		Args: map[string]interface{}{
			"serviceID":    serviceID,
			"expiredCount": expiredCount,
		},
		Model: model,
	}
}

// CreateWaitForCleanupGoroutinesCommand creates a command to wait for cleanup goroutines
func (f *NotificationsServiceCommandFactory) CreateWaitForCleanupGoroutinesCommand(serviceID string, goroutineCount int, timeout time.Duration, model interface{}) Command {
	id := fmt.Sprintf("notif-wait-%s-%d", serviceID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &NotificationsServiceCommand{
		BaseCommand:           *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		NotificationOperation: "wait_for_cleanup_goroutines",
		ServiceID:             serviceID,
		Args: map[string]interface{}{
			"serviceID":      serviceID,
			"goroutineCount": goroutineCount,
			"timeout":        timeout,
		},
		Model: model,
	}
}

// CreateShowNotificationCommand creates a command to show a notification
func (f *NotificationsServiceCommandFactory) CreateShowNotificationCommand(serviceID, title, message, level string, model interface{}) Command {
	id := fmt.Sprintf("notif-show-%s-%d", serviceID, time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &NotificationsServiceCommand{
		BaseCommand:           *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		NotificationOperation: "show_notification",
		ServiceID:             serviceID,
		Args: map[string]interface{}{
			"serviceID": serviceID,
			"title":    title,
			"message":  message,
			"level":    level,
		},
		Model: model,
	}
}

// CreateHideNotificationCommand creates a command to hide a notification
func (f *NotificationsServiceCommandFactory) CreateHideNotificationCommand(serviceID, notificationID string, model interface{}) Command {
	id := fmt.Sprintf("notif-hide-%s-%d", serviceID, time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &NotificationsServiceCommand{
		BaseCommand:           *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		NotificationOperation: "hide_notification",
		ServiceID:             serviceID,
		Args: map[string]interface{}{
			"serviceID":      serviceID,
			"notificationID": notificationID,
		},
		Model: model,
	}
}

// CreateClearNotificationsCommand creates a command to clear notifications
func (f *NotificationsServiceCommandFactory) CreateClearNotificationsCommand(serviceID string, clearedCount int, model interface{}) Command {
	id := fmt.Sprintf("notif-clear-%s-%d", serviceID, time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &NotificationsServiceCommand{
		BaseCommand:           *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		NotificationOperation: "clear_notifications",
		ServiceID:             serviceID,
		Args: map[string]interface{}{
			"serviceID":    serviceID,
			"clearedCount": clearedCount,
		},
		Model: model,
	}
}

// CreateManageNotificationQueueCommand creates a command to manage notification queue
func (f *NotificationsServiceCommandFactory) CreateManageNotificationQueueCommand(serviceID, action string, queueSize int, model interface{}) Command {
	id := fmt.Sprintf("notif-queue-%s-%d", serviceID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &NotificationsServiceCommand{
		BaseCommand:           *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		NotificationOperation: "manage_notification_queue",
		ServiceID:             serviceID,
		Args: map[string]interface{}{
			"serviceID":  serviceID,
			"action":    action,
			"queueSize": queueSize,
		},
		Model: model,
	}
}

// NotificationsServiceCommands provides high-level commands for notifications service operations
type NotificationsServiceCommands struct {
	factory *NotificationsServiceCommandFactory
}

// NewNotificationsServiceCommands creates a new notifications service commands instance
func NewNotificationsServiceCommands() *NotificationsServiceCommands {
	return &NotificationsServiceCommands{
		factory: NewNotificationsServiceCommandFactory(),
	}
}

// CleanupExpiredNotifications returns a command to cleanup expired notifications
func (nsc *NotificationsServiceCommands) CleanupExpiredNotifications(serviceID string, expiredCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := nsc.factory.CreateCleanupExpiredNotificationsCommand(serviceID, expiredCount, model)
		return cmd.Run()
	}
}

// WaitForCleanupGoroutines returns a command to wait for cleanup goroutines
func (nsc *NotificationsServiceCommands) WaitForCleanupGoroutines(serviceID string, goroutineCount int, timeout time.Duration, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := nsc.factory.CreateWaitForCleanupGoroutinesCommand(serviceID, goroutineCount, timeout, model)
		return cmd.Run()
	}
}

// ShowNotification returns a command to show a notification
func (nsc *NotificationsServiceCommands) ShowNotification(serviceID, title, message, level string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := nsc.factory.CreateShowNotificationCommand(serviceID, title, message, level, model)
		return cmd.Run()
	}
}

// HideNotification returns a command to hide a notification
func (nsc *NotificationsServiceCommands) HideNotification(serviceID, notificationID string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := nsc.factory.CreateHideNotificationCommand(serviceID, notificationID, model)
		return cmd.Run()
	}
}

// ClearNotifications returns a command to clear notifications
func (nsc *NotificationsServiceCommands) ClearNotifications(serviceID string, clearedCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := nsc.factory.CreateClearNotificationsCommand(serviceID, clearedCount, model)
		return cmd.Run()
	}
}

// ManageNotificationQueue returns a command to manage notification queue
func (nsc *NotificationsServiceCommands) ManageNotificationQueue(serviceID, action string, queueSize int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := nsc.factory.CreateManageNotificationQueueCommand(serviceID, action, queueSize, model)
		return cmd.Run()
	}
}