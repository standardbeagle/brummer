package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventBusCreation tests creating a new event bus
func TestEventBusCreation(t *testing.T) {
	bus := NewEventBus()
	require.NotNil(t, bus)
	assert.NotNil(t, bus.handlers)
}

// TestEventSubscription tests subscribing to events
func TestEventSubscription(t *testing.T) {
	bus := NewEventBus()
	
	var receivedEvents []Event
	var mu sync.Mutex
	
	handler := func(event Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	}
	
	// Subscribe to ProcessStarted events
	bus.Subscribe(ProcessStarted, handler)
	
	// Publish an event
	testEvent := Event{
		Type:      ProcessStarted,
		ProcessID: "test-process",
		Data: map[string]interface{}{
			"command": "echo hello",
			"pid":     12345,
		},
	}
	
	bus.Publish(testEvent)
	
	// Wait for async handler execution
	time.Sleep(10 * time.Millisecond)
	
	// Verify event was received
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, receivedEvents, 1)
	assert.Equal(t, ProcessStarted, receivedEvents[0].Type)
	assert.Equal(t, "test-process", receivedEvents[0].ProcessID)
	assert.Equal(t, "echo hello", receivedEvents[0].Data["command"])
	assert.Equal(t, 12345, receivedEvents[0].Data["pid"])
	assert.NotEmpty(t, receivedEvents[0].ID)        // ID should be auto-generated
	assert.False(t, receivedEvents[0].Timestamp.IsZero()) // Timestamp should be set
}

// TestMultipleSubscribers tests multiple handlers for the same event type
func TestMultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	
	var handler1Events []Event
	var handler2Events []Event
	var mu1, mu2 sync.Mutex
	
	handler1 := func(event Event) {
		mu1.Lock()
		handler1Events = append(handler1Events, event)
		mu1.Unlock()
	}
	
	handler2 := func(event Event) {
		mu2.Lock()
		handler2Events = append(handler2Events, event)
		mu2.Unlock()
	}
	
	// Both handlers subscribe to the same event type
	bus.Subscribe(LogLine, handler1)
	bus.Subscribe(LogLine, handler2)
	
	// Publish an event
	testEvent := Event{
		Type:      LogLine,
		ProcessID: "test-process",
		Data: map[string]interface{}{
			"line":    "Test log line",
			"isError": false,
		},
	}
	
	bus.Publish(testEvent)
	
	// Wait for async handler execution
	time.Sleep(10 * time.Millisecond)
	
	// Both handlers should have received the event
	mu1.Lock()
	defer mu1.Unlock()
	mu2.Lock()
	defer mu2.Unlock()
	
	require.Len(t, handler1Events, 1)
	require.Len(t, handler2Events, 1)
	
	assert.Equal(t, LogLine, handler1Events[0].Type)
	assert.Equal(t, LogLine, handler2Events[0].Type)
	assert.Equal(t, "Test log line", handler1Events[0].Data["line"])
	assert.Equal(t, "Test log line", handler2Events[0].Data["line"])
}

// TestMultipleEventTypes tests subscribing to different event types
func TestMultipleEventTypes(t *testing.T) {
	bus := NewEventBus()
	
	var processEvents []Event
	var logEvents []Event
	var errorEvents []Event
	var muProcess, muLog, muError sync.Mutex
	
	bus.Subscribe(ProcessStarted, func(event Event) {
		muProcess.Lock()
		processEvents = append(processEvents, event)
		muProcess.Unlock()
	})
	
	bus.Subscribe(LogLine, func(event Event) {
		muLog.Lock()
		logEvents = append(logEvents, event)
		muLog.Unlock()
	})
	
	bus.Subscribe(ErrorDetected, func(event Event) {
		muError.Lock()
		errorEvents = append(errorEvents, event)
		muError.Unlock()
	})
	
	// Publish different types of events
	bus.Publish(Event{Type: ProcessStarted, ProcessID: "proc1", Data: map[string]interface{}{"command": "echo"}})
	bus.Publish(Event{Type: LogLine, ProcessID: "proc1", Data: map[string]interface{}{"line": "output"}})
	bus.Publish(Event{Type: ErrorDetected, ProcessID: "proc1", Data: map[string]interface{}{"error": "test error"}})
	bus.Publish(Event{Type: LogLine, ProcessID: "proc1", Data: map[string]interface{}{"line": "more output"}})
	
	// Wait for async handler execution
	time.Sleep(10 * time.Millisecond)
	
	// Verify each handler only received its event type
	muProcess.Lock()
	defer muProcess.Unlock()
	muLog.Lock()
	defer muLog.Unlock()
	muError.Lock()
	defer muError.Unlock()
	
	assert.Len(t, processEvents, 1)
	assert.Len(t, logEvents, 2)
	assert.Len(t, errorEvents, 1)
	
	assert.Equal(t, ProcessStarted, processEvents[0].Type)
	assert.Equal(t, LogLine, logEvents[0].Type)
	assert.Equal(t, LogLine, logEvents[1].Type)
	assert.Equal(t, ErrorDetected, errorEvents[0].Type)
}

// TestEventMetadata tests automatic ID and timestamp generation
func TestEventMetadata(t *testing.T) {
	bus := NewEventBus()
	
	var receivedEvent Event
	var received bool
	var mu sync.Mutex
	
	bus.Subscribe(BuildEvent, func(event Event) {
		mu.Lock()
		receivedEvent = event
		received = true
		mu.Unlock()
	})
	
	// Publish event without ID or timestamp
	originalEvent := Event{
		Type:      BuildEvent,
		ProcessID: "build-process",
		Data:      map[string]interface{}{"buildID": 123},
	}
	
	publishTime := time.Now()
	bus.Publish(originalEvent)
	
	// Wait for async handler execution
	time.Sleep(10 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	require.True(t, received)
	
	// Verify metadata was automatically added
	assert.NotEmpty(t, receivedEvent.ID)
	assert.False(t, receivedEvent.Timestamp.IsZero())
	assert.True(t, receivedEvent.Timestamp.After(publishTime.Add(-1*time.Second)))
	assert.True(t, receivedEvent.Timestamp.Before(publishTime.Add(1*time.Second)))
	
	// Original data should be preserved
	assert.Equal(t, BuildEvent, receivedEvent.Type)
	assert.Equal(t, "build-process", receivedEvent.ProcessID)
	assert.Equal(t, 123, receivedEvent.Data["buildID"])
}

// TestConcurrentPublishing tests thread safety with concurrent publishing
func TestConcurrentPublishing(t *testing.T) {
	bus := NewEventBus()
	
	var receivedEvents []Event
	var mu sync.Mutex
	
	bus.Subscribe(TestPassed, func(event Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})
	
	// Publish events concurrently from multiple goroutines
	var wg sync.WaitGroup
	numPublishers := 10
	eventsPerPublisher := 5
	
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()
			
			for j := 0; j < eventsPerPublisher; j++ {
				bus.Publish(Event{
					Type:      TestPassed,
					ProcessID: "test-process",
					Data: map[string]interface{}{
						"publisherID": publisherID,
						"eventID":     j,
					},
				})
			}
		}(i)
	}
	
	wg.Wait()
	
	// Wait for all async handlers to complete
	time.Sleep(50 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	// Should have received all published events
	expectedCount := numPublishers * eventsPerPublisher
	assert.Len(t, receivedEvents, expectedCount)
	
	// Verify all events have unique IDs
	idSet := make(map[string]bool)
	for _, event := range receivedEvents {
		assert.False(t, idSet[event.ID], "Duplicate event ID found: %s", event.ID)
		idSet[event.ID] = true
		assert.Equal(t, TestPassed, event.Type)
	}
}

// TestConcurrentSubscription tests thread safety with concurrent subscription
func TestConcurrentSubscription(t *testing.T) {
	bus := NewEventBus()
	
	var totalReceived int64
	var mu sync.Mutex
	
	// Add subscribers concurrently
	var wg sync.WaitGroup
	numSubscribers := 5
	
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(subscriberID int) {
			defer wg.Done()
			
			bus.Subscribe(TestFailed, func(event Event) {
				mu.Lock()
				totalReceived++
				mu.Unlock()
			})
		}(i)
	}
	
	wg.Wait()
	
	// Publish a single event
	bus.Publish(Event{
		Type:      TestFailed,
		ProcessID: "test-process",
		Data:      map[string]interface{}{"test": "concurrent subscription"},
	})
	
	// Wait for all handlers
	time.Sleep(20 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	// All subscribers should have received the event
	assert.Equal(t, int64(numSubscribers), totalReceived)
}

// TestEventTypeConstants tests all defined event type constants
func TestEventTypeConstants(t *testing.T) {
	eventTypes := []EventType{
		ProcessStarted,
		ProcessExited,
		LogLine,
		ErrorDetected,
		BuildEvent,
		TestFailed,
		TestPassed,
		MCPActivity,
		MCPConnected,
		MCPDisconnected,
	}
	
	bus := NewEventBus()
	var receivedTypes []EventType
	var mu sync.Mutex
	
	// Subscribe to all event types
	for _, eventType := range eventTypes {
		bus.Subscribe(eventType, func(event Event) {
			mu.Lock()
			receivedTypes = append(receivedTypes, event.Type)
			mu.Unlock()
		})
	}
	
	// Publish events of all types
	for i, eventType := range eventTypes {
		bus.Publish(Event{
			Type:      eventType,
			ProcessID: "test-process",
			Data:      map[string]interface{}{"index": i},
		})
	}
	
	// Wait for all handlers
	time.Sleep(20 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	// Should have received all event types
	assert.Len(t, receivedTypes, len(eventTypes))
	
	// Verify all types were received
	receivedSet := make(map[EventType]bool)
	for _, eventType := range receivedTypes {
		receivedSet[eventType] = true
	}
	
	for _, expectedType := range eventTypes {
		assert.True(t, receivedSet[expectedType], "Event type %s was not received", expectedType)
	}
}

// TestEmptyEventHandling tests handling of events with minimal data
func TestEmptyEventHandling(t *testing.T) {
	bus := NewEventBus()
	
	var receivedEvent Event
	var received bool
	var mu sync.Mutex
	
	bus.Subscribe(MCPActivity, func(event Event) {
		mu.Lock()
		receivedEvent = event
		received = true
		mu.Unlock()
	})
	
	// Publish event with minimal data
	bus.Publish(Event{
		Type: MCPActivity,
		// ProcessID is empty
		// Data is nil
	})
	
	time.Sleep(10 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	require.True(t, received)
	assert.Equal(t, MCPActivity, receivedEvent.Type)
	assert.Empty(t, receivedEvent.ProcessID)
	assert.Nil(t, receivedEvent.Data)
	assert.NotEmpty(t, receivedEvent.ID)
	assert.False(t, receivedEvent.Timestamp.IsZero())
}