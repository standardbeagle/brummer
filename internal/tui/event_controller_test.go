package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/system"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
)

func TestEventController_SystemMessages(t *testing.T) {
	// Setup - use real components instead of mocks
	updateChan := make(chan tea.Msg, 10)
	eventBus := events.NewEventBus()

	// Create a full test model with real components
	model := createTestModelWithDefaults()

	// Verify we're using the same eventBus
	t.Logf("Test eventBus: %p", eventBus)
	t.Logf("Model eventBus: %p", model.eventBus)

	// Replace model's eventBus with our test eventBus to ensure they're the same
	model.eventBus = eventBus
	model.updateChan = updateChan

	// Re-setup event subscriptions with the new eventBus
	model.eventController.eventBus = eventBus
	model.eventController.updateChan = updateChan
	model.setupEventSubscriptions()

	// EventController is now initialized as part of the model
	// Just verify the event flow works

	// Test system message events
	tests := []struct {
		name     string
		level    string
		message  string
		expected string
	}{
		{
			name:     "info_message",
			level:    "info",
			message:  "Test info message",
			expected: "Test info message",
		},
		{
			name:     "error_message",
			level:    "error",
			message:  "Test error message",
			expected: "Test error message",
		},
		{
			name:     "warning_message",
			level:    "warn",
			message:  "Test warning message",
			expected: "Test warning message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Publish event using the correct API
			t.Logf("Publishing system.message event with level=%s, message=%s", tt.level, tt.message)
			eventBus.Publish(events.Event{
				Type: events.EventType("system.message"),
				Data: map[string]interface{}{
					"level":   tt.level,
					"message": tt.message,
					"context": "Test",
				},
			})

			// Wait for message to be processed (increased timeout for goroutine execution)
			select {
			case msg := <-updateChan:
				t.Logf("Received message: %T", msg)
				sysMsg, ok := msg.(system.SystemMessageMsg)
				assert.True(t, ok, "Should receive SystemMessageMsg")
				assert.Equal(t, tt.level, sysMsg.Level)
				assert.Equal(t, tt.expected, sysMsg.Message)
			case <-time.After(500 * time.Millisecond):
				// Check if there are any messages in the channel
				select {
				case msg := <-updateChan:
					t.Logf("Late message received: %T", msg)
				default:
					t.Logf("No messages in channel")
				}
				t.Fatal("Timeout waiting for system message")
			}
		})
	}
}

func TestEventController_ProcessEvents(t *testing.T) {
	// Setup - use real components instead of mocks
	updateChan := make(chan tea.Msg, 10)
	eventBus := events.NewEventBus()

	// Create a full test model with real components
	model := createTestModelWithDefaults()
	model.eventBus = eventBus
	model.updateChan = updateChan

	// Re-setup event subscriptions and update channel for event controller
	model.eventController.eventBus = eventBus
	model.eventController.updateChan = updateChan
	model.setupEventSubscriptions()

	// Test process lifecycle events (using actual event constants)
	processEvents := []events.EventType{
		events.ProcessStarted,
		events.ProcessExited,
		events.LogLine,
		events.ErrorDetected,
	}

	for _, event := range processEvents {
		t.Run(string(event), func(t *testing.T) {
			// Publish event using the correct API
			eventBus.Publish(events.Event{
				Type: event,
				Data: map[string]interface{}{
					"processId": "test-123",
					"name":      "test-process",
				},
			})

			// Wait for update message - different events send different message types
			select {
			case msg := <-updateChan:
				switch event {
				case events.ProcessStarted, events.ProcessExited:
					_, ok := msg.(processUpdateMsg)
					assert.True(t, ok, "Should receive processUpdateMsg for %s", event)
				case events.LogLine:
					_, ok := msg.(logUpdateMsg)
					assert.True(t, ok, "Should receive logUpdateMsg for %s", event)
				case events.ErrorDetected:
					_, ok := msg.(errorUpdateMsg)
					assert.True(t, ok, "Should receive errorUpdateMsg for %s", event)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Timeout waiting for update message from %s", event)
			}
		})
	}
}

func TestEventController_LogEvents(t *testing.T) {
	// Setup - use real components instead of mocks
	updateChan := make(chan tea.Msg, 10)
	eventBus := events.NewEventBus()

	// Create a full test model with real components
	model := createTestModelWithDefaults()
	model.eventBus = eventBus
	model.updateChan = updateChan

	// Re-setup event subscriptions and update channel for event controller
	model.eventController.eventBus = eventBus
	model.eventController.updateChan = updateChan
	model.setupEventSubscriptions()

	// Test log event
	eventBus.Publish(events.Event{
		Type: events.LogLine,
		Data: map[string]interface{}{
			"processId": "test-456",
			"message":   "Test log message",
			"isError":   false,
		},
	})

	// Wait for update
	select {
	case msg := <-updateChan:
		_, ok := msg.(logUpdateMsg)
		assert.True(t, ok, "Should receive logUpdateMsg")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for log update")
	}
}

func TestEventController_URLDetection(t *testing.T) {
	// Skip this test - url.detected event type doesn't exist in the current event system
	// URL detection happens differently in the current implementation
	t.Skip("URL detection uses a different mechanism - this event type doesn't exist")
}

func TestEventController_StartupMessage(t *testing.T) {
	// Setup - use real components instead of mocks
	updateChan := make(chan tea.Msg, 10)
	eventBus := events.NewEventBus()

	// Create a full test model with real components
	model := createTestModelWithDefaults()
	model.eventBus = eventBus
	model.updateChan = updateChan

	// Re-setup event subscriptions and update channel for event controller
	model.eventController.eventBus = eventBus
	model.eventController.updateChan = updateChan
	model.setupEventSubscriptions()

	// Send startup message
	model.eventController.SendStartupMessage()

	// Wait for message
	select {
	case msg := <-updateChan:
		sysMsg, ok := msg.(system.SystemMessageMsg)
		assert.True(t, ok, "Should receive SystemMessageMsg")
		assert.Equal(t, "info", sysMsg.Level)
		assert.Contains(t, sysMsg.Message, "Brummer started")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for startup message")
	}
}

func TestEventController_EmptyMessage(t *testing.T) {
	// Setup - use real components instead of mocks
	updateChan := make(chan tea.Msg, 10)
	eventBus := events.NewEventBus()

	// Create a full test model with real components
	model := createTestModelWithDefaults()
	model.eventBus = eventBus
	model.updateChan = updateChan

	// Re-setup event subscriptions and update channel for event controller
	model.eventController.eventBus = eventBus
	model.eventController.updateChan = updateChan
	model.setupEventSubscriptions()

	// Publish event with empty message (should be ignored)
	eventBus.Publish(events.Event{
		Type: events.EventType("system.message"),
		Data: map[string]interface{}{
			"level":   "info",
			"message": "",
		},
	})

	// Should not receive any message
	select {
	case <-updateChan:
		t.Fatal("Should not receive message for empty content")
	case <-time.After(50 * time.Millisecond):
		// Expected - no message sent
	}
}

func TestEventController_EventFlow(t *testing.T) {
	// This test verifies that events flow properly through the system
	// without needing to test internal implementation details

	// Setup - use real components
	model := createTestModelWithDefaults()
	updateChan := make(chan tea.Msg, 10)
	model.updateChan = updateChan

	// Also update the event controller's update channel to match
	model.eventController.updateChan = updateChan

	// Test that process events flow through the system
	model.eventBus.Publish(events.Event{
		Type: events.EventType("process.started"),
		Data: map[string]interface{}{
			"processId": "test-flow-123",
			"name":      "test-flow",
		},
	})

	// Verify we get a process update message
	msg := waitForMessage[processUpdateMsg](t, updateChan, 100*time.Millisecond)
	assert.NotNil(t, msg, "Should receive process update message")
}
