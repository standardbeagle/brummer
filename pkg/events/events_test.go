package events

import (
	"sync"
	"testing"
	"time"
)

func TestEventBus(t *testing.T) {
	bus := NewEventBus()

	t.Run("Subscribe and Publish", func(t *testing.T) {
		var received Event
		var wg sync.WaitGroup
		wg.Add(1)

		// Subscribe to events
		bus.Subscribe(EventProcessStarted, func(e Event) {
			received = e
			wg.Done()
		})

		// Publish event
		event := Event{
			Type:      EventProcessStarted,
			ProcessID: "test-process",
			Data: map[string]interface{}{
				"command": "echo hello",
				"pid":     12345,
			},
		}
		bus.Publish(event)

		// Wait for event
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Event received
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for event")
		}

		// Verify event data
		if received.Type != EventProcessStarted {
			t.Errorf("Expected type %v, got %v", EventProcessStarted, received.Type)
		}
		if received.ProcessID != "test-process" {
			t.Errorf("Expected process ID 'test-process', got %s", received.ProcessID)
		}
		if received.Data["command"] != "echo hello" {
			t.Errorf("Expected command 'echo hello', got %v", received.Data["command"])
		}
	})

	t.Run("Multiple Subscribers", func(t *testing.T) {
		var received1, received2 Event
		var wg sync.WaitGroup
		wg.Add(2)

		// Multiple subscribers
		bus.Subscribe(EventLogLine, func(e Event) {
			received1 = e
			wg.Done()
		})
		bus.Subscribe(EventLogLine, func(e Event) {
			received2 = e
			wg.Done()
		})

		// Publish event
		event := Event{
			Type:      EventLogLine,
			ProcessID: "logger",
			Data: map[string]interface{}{
				"line":    "Test log message",
				"isError": false,
			},
		}
		bus.Publish(event)

		// Wait for both events
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Both events received
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for events")
		}

		// Both should have received the same event
		if received1.ProcessID != "logger" || received2.ProcessID != "logger" {
			t.Error("Not all subscribers received the event")
		}
	})

	t.Run("Different Event Types", func(t *testing.T) {
		var processEvents, logEvents int
		var wg sync.WaitGroup
		wg.Add(3) // We'll send 2 process events and 1 log event

		bus.Subscribe(EventProcessStarted, func(e Event) {
			processEvents++
			wg.Done()
		})
		bus.Subscribe(EventProcessExited, func(e Event) {
			processEvents++
			wg.Done()
		})
		bus.Subscribe(EventLogLine, func(e Event) {
			logEvents++
			wg.Done()
		})

		// Publish different types
		bus.Publish(Event{Type: EventProcessStarted, ProcessID: "proc1"})
		bus.Publish(Event{Type: EventProcessExited, ProcessID: "proc1"})
		bus.Publish(Event{Type: EventLogLine, ProcessID: "proc1"})

		// Wait for all events
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// All events received
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for events")
		}

		if processEvents != 2 {
			t.Errorf("Expected 2 process events, got %d", processEvents)
		}
		if logEvents != 1 {
			t.Errorf("Expected 1 log event, got %d", logEvents)
		}
	})

	t.Run("Concurrent Publishing", func(t *testing.T) {
		var eventCount int
		var mutex sync.Mutex
		var wg sync.WaitGroup

		// Subscribe to count events
		bus.Subscribe(EventBuildEvent, func(e Event) {
			mutex.Lock()
			eventCount++
			mutex.Unlock()
			wg.Done()
		})

		// Publish many events concurrently
		numEvents := 100
		wg.Add(numEvents)

		for i := 0; i < numEvents; i++ {
			go func(id int) {
				bus.Publish(Event{
					Type:      EventBuildEvent,
					ProcessID: "builder",
					Data: map[string]interface{}{
						"id": id,
					},
				})
			}(i)
		}

		// Wait for all events
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// All events received
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent events")
		}

		mutex.Lock()
		finalCount := eventCount
		mutex.Unlock()

		if finalCount != numEvents {
			t.Errorf("Expected %d events, got %d", numEvents, finalCount)
		}
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		var eventCount int
		var wg sync.WaitGroup

		handler := func(e Event) {
			eventCount++
			wg.Done()
		}

		// Subscribe
		bus.Subscribe(EventTestResult, handler)

		// Send first event
		wg.Add(1)
		bus.Publish(Event{Type: EventTestResult, ProcessID: "test1"})
		wg.Wait()

		// Unsubscribe
		bus.Unsubscribe(EventTestResult, handler)

		// Send second event (should not be received)
		bus.Publish(Event{Type: EventTestResult, ProcessID: "test2"})

		// Give some time for potential delivery
		time.Sleep(100 * time.Millisecond)

		if eventCount != 1 {
			t.Errorf("Expected 1 event (before unsubscribe), got %d", eventCount)
		}
	})

	t.Run("Event Type String Representation", func(t *testing.T) {
		tests := []struct {
			eventType EventType
			expected  string
		}{
			{EventProcessStarted, "process.started"},
			{EventProcessExited, "process.exited"},
			{EventLogLine, "log.line"},
			{EventErrorDetected, "error.detected"},
			{EventBuildEvent, "build.event"},
			{EventTestResult, "test.result"},
		}

		for _, tt := range tests {
			if string(tt.eventType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.eventType))
			}
		}
	})

	t.Run("Event Data Integrity", func(t *testing.T) {
		var receivedEvent Event
		var wg sync.WaitGroup
		wg.Add(1)

		bus.Subscribe(EventErrorDetected, func(e Event) {
			receivedEvent = e
			wg.Done()
		})

		// Create complex event data
		originalData := map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Test error",
				"file":    "/path/to/file.go",
				"line":    42,
			},
			"context": []string{"context1", "context2"},
			"metadata": map[string]interface{}{
				"timestamp": time.Now().Unix(),
				"level":     "error",
			},
		}

		event := Event{
			Type:      EventErrorDetected,
			ProcessID: "error-process",
			Data:      originalData,
		}

		bus.Publish(event)
		wg.Wait()

		// Verify data integrity
		errorData := receivedEvent.Data["error"].(map[string]interface{})
		if errorData["message"] != "Test error" {
			t.Error("Event data was corrupted during transmission")
		}

		context := receivedEvent.Data["context"].([]string)
		if len(context) != 2 || context[0] != "context1" {
			t.Error("Array data was corrupted")
		}
	})
}

func TestEventBusMemoryLeaks(t *testing.T) {
	bus := NewEventBus()

	t.Run("No Memory Leak on Subscribe/Unsubscribe", func(t *testing.T) {
		initialHandlerCount := len(bus.handlers)

		// Add many handlers
		var handlers []Handler
		for i := 0; i < 100; i++ {
			handler := func(e Event) {}
			handlers = append(handlers, handler)
			bus.Subscribe(EventLogLine, handler)
		}

		// Verify handlers were added
		if len(bus.handlers[EventLogLine]) != 100 {
			t.Errorf("Expected 100 handlers, got %d", len(bus.handlers[EventLogLine]))
		}

		// Remove all handlers
		for _, handler := range handlers {
			bus.Unsubscribe(EventLogLine, handler)
		}

		// Should be back to initial state
		currentHandlerCount := len(bus.handlers)
		if currentHandlerCount != initialHandlerCount {
			t.Errorf("Memory leak: expected %d handler maps, got %d", 
				initialHandlerCount, currentHandlerCount)
		}
	})
}

func BenchmarkEventBus(b *testing.B) {
	bus := NewEventBus()
	
	// Subscribe a simple handler
	bus.Subscribe(EventLogLine, func(e Event) {
		// Minimal processing
		_ = e.ProcessID
	})

	event := Event{
		Type:      EventLogLine,
		ProcessID: "bench-process",
		Data: map[string]interface{}{
			"line":    "Benchmark log line",
			"isError": false,
		},
	}

	b.ResetTimer()
	
	b.Run("Publish", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bus.Publish(event)
		}
	})

	b.Run("Subscribe", func(b *testing.B) {
		handler := func(e Event) {}
		for i := 0; i < b.N; i++ {
			bus.Subscribe(EventLogLine, handler)
		}
	})

	b.Run("Concurrent", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				bus.Publish(event)
			}
		})
	})
}