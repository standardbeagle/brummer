package notifications

import (
	"fmt"
	"sync"
	
	"github.com/standardbeagle/brummer/pkg/events"
)

// Publisher provides a simplified interface for publishing system notifications
type Publisher struct {
	eventBus *events.EventBus
}

// NewPublisher creates a new notification publisher
func NewPublisher(eventBus *events.EventBus) *Publisher {
	return &Publisher{
		eventBus: eventBus,
	}
}

// Error publishes an error notification
func (p *Publisher) Error(context, message string) {
	if p.eventBus != nil {
		p.eventBus.PublishErrorNotification(context, message)
	}
}

// Errorf publishes a formatted error notification
func (p *Publisher) Errorf(context, format string, args ...interface{}) {
	if p.eventBus != nil {
		p.eventBus.PublishErrorNotification(context, fmt.Sprintf(format, args...))
	}
}

// Warning publishes a warning notification
func (p *Publisher) Warning(context, message string) {
	if p.eventBus != nil {
		p.eventBus.PublishWarningNotification(context, message)
	}
}

// Warningf publishes a formatted warning notification
func (p *Publisher) Warningf(context, format string, args ...interface{}) {
	if p.eventBus != nil {
		p.eventBus.PublishWarningNotification(context, fmt.Sprintf(format, args...))
	}
}

// Info publishes an info notification
func (p *Publisher) Info(context, message string) {
	if p.eventBus != nil {
		p.eventBus.PublishInfoNotification(context, message)
	}
}

// Infof publishes a formatted info notification
func (p *Publisher) Infof(context, format string, args ...interface{}) {
	if p.eventBus != nil {
		p.eventBus.PublishInfoNotification(context, fmt.Sprintf(format, args...))
	}
}

// Success publishes a success notification
func (p *Publisher) Success(context, message string) {
	if p.eventBus != nil {
		p.eventBus.PublishSuccessNotification(context, message)
	}
}

// Successf publishes a formatted success notification
func (p *Publisher) Successf(context, format string, args ...interface{}) {
	if p.eventBus != nil {
		p.eventBus.PublishSuccessNotification(context, fmt.Sprintf(format, args...))
	}
}

// WithDetails publishes a notification with additional details
func (p *Publisher) WithDetails(level events.SystemNotificationLevel, context, message, details string) {
	if p.eventBus != nil {
		p.eventBus.PublishSystemNotificationWithDetails(level, context, message, details)
	}
}

// ErrorWithDetails publishes an error notification with details
func (p *Publisher) ErrorWithDetails(context, message, details string) {
	p.WithDetails(events.LevelError, context, message, details)
}

// Global singleton instance for easy access
var (
	globalPublisher *Publisher
	globalMutex     sync.RWMutex
)

// SetGlobalPublisher sets the global notification publisher in a thread-safe manner
func SetGlobalPublisher(eventBus *events.EventBus) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalPublisher = NewPublisher(eventBus)
}

// GetGlobalPublisher returns the global notification publisher in a thread-safe manner
func GetGlobalPublisher() *Publisher {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalPublisher
}

// Convenience global functions that use the global publisher

// NotifyError publishes a global error notification
func NotifyError(context, message string) {
	if globalPublisher != nil {
		globalPublisher.Error(context, message)
	}
}

// NotifyErrorf publishes a formatted global error notification
func NotifyErrorf(context, format string, args ...interface{}) {
	if globalPublisher != nil {
		globalPublisher.Errorf(context, format, args...)
	}
}

// NotifyWarning publishes a global warning notification
func NotifyWarning(context, message string) {
	if globalPublisher != nil {
		globalPublisher.Warning(context, message)
	}
}

// NotifyWarningf publishes a formatted global warning notification
func NotifyWarningf(context, format string, args ...interface{}) {
	if globalPublisher != nil {
		globalPublisher.Warningf(context, format, args...)
	}
}

// NotifyInfo publishes a global info notification
func NotifyInfo(context, message string) {
	if globalPublisher != nil {
		globalPublisher.Info(context, message)
	}
}

// NotifyInfof publishes a formatted global info notification
func NotifyInfof(context, format string, args ...interface{}) {
	if globalPublisher != nil {
		globalPublisher.Infof(context, format, args...)
	}
}

// NotifySuccess publishes a global success notification
func NotifySuccess(context, message string) {
	if globalPublisher != nil {
		globalPublisher.Success(context, message)
	}
}

// NotifySuccessf publishes a formatted global success notification
func NotifySuccessf(context, format string, args ...interface{}) {
	if globalPublisher != nil {
		globalPublisher.Successf(context, format, args...)
	}
}