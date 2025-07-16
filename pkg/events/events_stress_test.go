package events

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBusStressTest(t *testing.T) {
	eb := NewEventBus()
	defer eb.Shutdown()

	const (
		numPublishers      = 100
		eventsPerPublisher = 100
		totalEvents        = numPublishers * eventsPerPublisher
	)

	// Track processed events
	var processedCount int64
	var handlerExecutions int64

	// Subscribe handlers for different event types
	eb.Subscribe(LogLine, func(event Event) {
		atomic.AddInt64(&handlerExecutions, 1)
		// Simulate some work
		time.Sleep(time.Microsecond)
	})

	eb.Subscribe(ProcessStarted, func(event Event) {
		atomic.AddInt64(&handlerExecutions, 1)
		time.Sleep(time.Microsecond)
	})

	// Track goroutine count before stress test
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("Initial goroutines: %d", initialGoroutines)

	// Start stress test
	start := time.Now()
	var wg sync.WaitGroup

	// Launch publishers
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()
			for j := 0; j < eventsPerPublisher; j++ {
				eventType := LogLine
				if j%2 == 0 {
					eventType = ProcessStarted
				}

				eb.Publish(Event{
					Type:      eventType,
					ProcessID: "stress-test",
					Data: map[string]interface{}{
						"publisher": publisherID,
						"sequence":  j,
					},
				})
				atomic.AddInt64(&processedCount, 1)
			}
		}(i)
	}

	// Wait for all publishers to finish
	wg.Wait()
	duration := time.Since(start)

	// Allow time for all events to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify results
	finalProcessed := atomic.LoadInt64(&processedCount)
	finalExecutions := atomic.LoadInt64(&handlerExecutions)
	finalGoroutines := runtime.NumGoroutine()

	t.Logf("Published %d events in %v", finalProcessed, duration)
	t.Logf("Handler executions: %d", finalExecutions)
	t.Logf("Events per second: %.0f", float64(finalProcessed)/duration.Seconds())
	t.Logf("Final goroutines: %d (vs initial %d)", finalGoroutines, initialGoroutines)

	// Verify all events were published
	if finalProcessed != totalEvents {
		t.Errorf("Expected %d events published, got %d", totalEvents, finalProcessed)
	}

	// Verify handler executions (should equal total events since we alternate event types)
	// Each event matches exactly one handler (LogLine or ProcessStarted)
	expectedExecutions := int64(totalEvents)
	if finalExecutions != expectedExecutions {
		t.Errorf("Expected %d handler executions, got %d", expectedExecutions, finalExecutions)
	}

	// Verify goroutine count is bounded (should not exceed initial + worker pool size significantly)
	config := DefaultWorkerPoolConfig()
	maxExpectedGoroutines := initialGoroutines + config.WorkerCount + 10 // +10 for test overhead
	if finalGoroutines > maxExpectedGoroutines {
		t.Errorf("Too many goroutines: %d (expected max ~%d)", finalGoroutines, maxExpectedGoroutines)
	}

	// Performance benchmark - should process events quickly
	eventsPerSecond := float64(finalProcessed) / duration.Seconds()
	if eventsPerSecond < 10000 { // Expect at least 10k events/sec
		t.Errorf("Performance too slow: %.0f events/sec (expected >10000)", eventsPerSecond)
	}
}

func TestEventBusGoroutineCount(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	// Create and destroy multiple EventBus instances
	for i := 0; i < 10; i++ {
		eb := NewEventBus()

		// Publish some events
		for j := 0; j < 50; j++ {
			eb.Publish(Event{
				Type:      LogLine,
				ProcessID: "test",
				Data:      map[string]interface{}{"iteration": i, "event": j},
			})
		}

		// Shutdown and verify cleanup
		eb.Shutdown()
	}

	// Force garbage collection and give time for cleanup
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	t.Logf("Goroutines before: %d, after: %d", initialGoroutines, finalGoroutines)

	// Should not leak goroutines
	if finalGoroutines > initialGoroutines+5 { // Allow small variance
		t.Errorf("Goroutine leak detected: %d -> %d", initialGoroutines, finalGoroutines)
	}
}

func TestEventBusPoolFullScenario(t *testing.T) {
	// Create small worker pool to test fallback behavior
	config := WorkerPoolConfig{
		WorkerCount: 2,
		BufferSize:  5,
	}
	eb := NewEventBusWithConfig(config)
	defer eb.Shutdown()

	// Subscribe slow handler to fill up the pool
	var processedCount int64
	eb.Subscribe(LogLine, func(event Event) {
		atomic.AddInt64(&processedCount, 1)
		time.Sleep(10 * time.Millisecond) // Slow handler
	})

	// Publish many events quickly to trigger fallback
	const numEvents = 50
	for i := 0; i < numEvents; i++ {
		eb.Publish(Event{
			Type:      LogLine,
			ProcessID: "pool-full-test",
			Data:      map[string]interface{}{"event": i},
		})
	}

	// Wait for processing to complete
	time.Sleep(200 * time.Millisecond)

	// Verify all events were processed (either via pool or fallback)
	processed := atomic.LoadInt64(&processedCount)
	if processed != numEvents {
		t.Errorf("Expected %d events processed, got %d", numEvents, processed)
	}

	t.Logf("Successfully processed %d events with small pool (fallback working)", processed)
}
