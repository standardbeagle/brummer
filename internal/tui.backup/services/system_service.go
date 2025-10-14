package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
)

// Sentinel errors
var (
	ErrMessageTooLarge  = errors.New("MESSAGE_TOO_LARGE")
	ErrBufferFull       = errors.New("BUFFER_FULL")
	ErrInvalidTimestamp = errors.New("INVALID_TIMESTAMP")
	ErrServiceNotActive = errors.New("SERVICE_NOT_ACTIVE")
	ErrInvalidOffset    = errors.New("INVALID_OFFSET")
	ErrInvalidLimit     = errors.New("INVALID_LIMIT")
)

// SystemMessage represents a system message with metadata
type SystemMessage struct {
	ID        string                    `json:"id"`
	Level     messages.LogLevel         `json:"level"`
	Content   string                    `json:"content"`
	Timestamp time.Time                 `json:"timestamp"`
	Read      bool                      `json:"read"`
	Source    string                    `json:"source"`
}

// SystemPanelState represents the current state of the system panel
type SystemPanelState struct {
	Expanded bool      `json:"expanded"`
	Time     time.Time `json:"time"`
}

// SystemService manages system messages and panel state
type SystemService struct {
	// Message storage
	messages    []SystemMessage // slice for message storage with size limit
	bufferLimit int             // maximum number of messages to store

	// Thread-safe counters
	unreadCount atomic.Int32 // atomic.Int32 for thread-safe unread tracking

	// Panel state
	isExpanded atomic.Bool // atomic.Bool for panel state

	// Thread safety
	mutex sync.Mutex // mutex for protecting the messages slice during operations

	// Event system
	eventBus *events.EventBus

	// Service state
	shutdown atomic.Bool
}

// NewSystemService creates a new system service with the provided event bus
func NewSystemService(eventBus *events.EventBus) *SystemService {
	if eventBus == nil {
		eventBus = events.NewEventBus()
	}

	service := &SystemService{
		messages:    make([]SystemMessage, 0, 100), // Initial capacity
		bufferLimit: 1000,                           // Maximum 1000 messages
		eventBus:    eventBus,
	}

	// Initialize atomic values
	service.unreadCount.Store(0)
	service.isExpanded.Store(false) // Start with collapsed panel

	return service
}

// AddSystemMessage adds a new system message with validation
func (s *SystemService) AddSystemMessage(ctx context.Context, message SystemMessage) (string, error) {
	// Check if service is shutdown
	if s.shutdown.Load() {
		return "", ErrServiceNotActive
	}

	// Validate message content size (max 10KB)
	if len(message.Content) > 10000 {
		return "", ErrMessageTooLarge
	}

	// Generate unique ID if not provided
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// Set default source if not provided
	if message.Source == "" {
		message.Source = "system"
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check buffer capacity and remove oldest messages if needed
	if len(s.messages) >= s.bufferLimit {
		// Remove oldest message(s) to make space
		// Remove the oldest unread message if we're dropping messages
		oldestUnreadIndex := -1
		for i, msg := range s.messages {
			if !msg.Read {
				oldestUnreadIndex = i
				break
			}
		}

		if oldestUnreadIndex != -1 {
			// We're dropping an unread message
			s.messages = s.messages[1:] // Remove first (oldest) message
		} else {
			// All messages are read, safely remove the oldest
			s.messages = s.messages[1:]
		}
	}

	// Add new message
	s.messages = append(s.messages, message)

	// Update unread count if message is not read
	if !message.Read {
		s.unreadCount.Add(1)
	}

	// Sort messages by timestamp (newest first) for display
	sort.Slice(s.messages, func(i, j int) bool {
		return s.messages[i].Timestamp.After(s.messages[j].Timestamp)
	})

	// Publish event
	unreadCount := int(s.unreadCount.Load())
	s.eventBus.Publish(events.Event{
		Type:      events.SystemMessage,
		ProcessID: "system-service",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message_id":   message.ID,
			"level":        string(message.Level),
			"content":      message.Content,
			"source":       message.Source,
			"unread_count": unreadCount,
			"time":         message.Timestamp,
		},
	})

	return message.ID, nil
}

// GetUnreadMessageCount returns the current number of unread messages
func (s *SystemService) GetUnreadMessageCount(ctx context.Context) (int, error) {
	if s.shutdown.Load() {
		return 0, ErrServiceNotActive
	}

	return int(s.unreadCount.Load()), nil
}

// MarkMessagesAsRead marks all messages up to the specified timestamp as read
func (s *SystemService) MarkMessagesAsRead(ctx context.Context, upToTime time.Time) error {
	if s.shutdown.Load() {
		return ErrServiceNotActive
	}

	// Validate timestamp
	if upToTime.IsZero() {
		return ErrInvalidTimestamp
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	readCount := 0

	// Mark messages as read
	for i := range s.messages {
		if !s.messages[i].Read && s.messages[i].Timestamp.Before(upToTime) || s.messages[i].Timestamp.Equal(upToTime) {
			s.messages[i].Read = true
			readCount++
		}
	}

	// Update unread count atomically
	if readCount > 0 {
		s.unreadCount.Add(int32(-readCount))
	}

	// Publish event
	unreadCount := int(s.unreadCount.Load())
	s.eventBus.Publish(events.Event{
		Type:      events.SystemMessage,
		ProcessID: "system-service",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"event_type":   "messages_read",
			"up_to_time":   upToTime,
			"read_count":   readCount,
			"unread_count": unreadCount,
			"time":         time.Now(),
		},
	})

	return nil
}

// ToggleSystemPanel toggles the expanded state of the system panel
func (s *SystemService) ToggleSystemPanel(ctx context.Context) (bool, error) {
	if s.shutdown.Load() {
		return false, ErrServiceNotActive
	}

	// Toggle state atomically
	newState := !s.isExpanded.Swap(!s.isExpanded.Load())

	// Publish event
	s.eventBus.Publish(events.Event{
		Type:      events.SystemMessage,
		ProcessID: "system-service",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"event_type": "panel_toggled",
			"expanded":   newState,
			"trigger":    "user_action",
			"time":       time.Now(),
		},
	})

	return newState, nil
}

// GetMessages retrieves messages with pagination and ordering
func (s *SystemService) GetMessages(ctx context.Context, offset, limit int) ([]SystemMessage, error) {
	if s.shutdown.Load() {
		return nil, ErrServiceNotActive
	}

	// Validate pagination parameters
	if offset < 0 {
		return nil, ErrInvalidOffset
	}
	if limit <= 0 || limit > 100 {
		return nil, ErrInvalidLimit
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Ensure offset is within bounds
	if offset >= len(s.messages) {
		return []SystemMessage{}, nil
	}

	// Calculate end index
	end := offset + limit
	if end > len(s.messages) {
		end = len(s.messages)
	}

	// Return slice (messages are already sorted by timestamp, newest first)
	// Return a copy to avoid race conditions
	result := make([]SystemMessage, end-offset)
	copy(result, s.messages[offset:end])

	return result, nil
}

// GetPanelState returns the current panel state
func (s *SystemService) GetPanelState(ctx context.Context) (*SystemPanelState, error) {
	if s.shutdown.Load() {
		return nil, ErrServiceNotActive
	}

	return &SystemPanelState{
		Expanded: s.isExpanded.Load(),
		Time:     time.Now(),
	}, nil
}

// ClearMessages clears all messages from the buffer
func (s *SystemService) ClearMessages(ctx context.Context) error {
	if s.shutdown.Load() {
		return ErrServiceNotActive
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	clearedCount := len(s.messages)

	// Clear messages
	s.messages = make([]SystemMessage, 0, 100)

	// Reset unread count
	s.unreadCount.Store(0)

	// Publish event
	s.eventBus.Publish(events.Event{
		Type:      events.SystemMessage,
		ProcessID: "system-service",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"event_type":   "buffer_cleared",
			"cleared_count": clearedCount,
			"trigger":      "user_action",
			"time":         time.Now(),
		},
	})

	return nil
}

// GetBufferLimit returns the maximum number of messages that can be stored
func (s *SystemService) GetBufferLimit() int {
	return s.bufferLimit
}

// GetBufferUsage returns the current number of messages stored
func (s *SystemService) GetBufferUsage(ctx context.Context) (int, error) {
	if s.shutdown.Load() {
		return 0, ErrServiceNotActive
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	return len(s.messages), nil
}

// Shutdown gracefully shuts down the system service
func (s *SystemService) Shutdown(ctx context.Context) error {
	if s.shutdown.Swap(true) {
		return nil // Already shutdown, idempotent
	}

	// Clear all messages
	if err := s.ClearMessages(ctx); err != nil {
		// Log error but continue shutdown
		fmt.Printf("Error clearing messages during shutdown: %v\n", err)
	}

	return nil
}

// EventBus returns the EventBus for subscribing to service events
func (s *SystemService) EventBus() *events.EventBus {
	return s.eventBus
}

// GetServiceName returns the service name for ServiceBridge integration
func (s *SystemService) GetServiceName() string {
	return "system"
}

// GetServiceStatus returns the current status of the system service
func (s *SystemService) GetServiceStatus() ServiceStatus {
	status := "active"
	details := fmt.Sprintf("messages: %d/%d, unread: %d", s.getMessageCount(), s.bufferLimit, s.unreadCount.Load())
	if s.shutdown.Load() {
		status = "shutting down"
	}

	return ServiceStatus{
		Service: s.GetServiceName(),
		Status:  status,
		Details: details,
	}
}

// SubscribeToEvents subscribes the system service to relevant system events
func (s *SystemService) SubscribeToEvents() {
	if s.eventBus == nil {
		return
	}

	// System service can subscribe to system-wide events that need to be shown as messages
	// For now, it doesn't subscribe to specific events
}

// Stop gracefully shuts down the system service
func (s *SystemService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.Shutdown(ctx)
}

// getMessageCount returns the current number of messages stored
func (s *SystemService) getMessageCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.messages)
}