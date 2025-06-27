package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImprovedEventBus_ConcurrentOperations(t *testing.T) {
	eb := NewImprovedEventBus(1000)
	defer eb.Shutdown(5 * time.Second)
	
	const (
		numPublishers  = 10
		numSubscribers = 20
		eventsPerPub   = 100
	)
	
	var (
		receivedEvents atomic.Int64
		wg             sync.WaitGroup
	)
	
	// Create subscribers
	for i := 0; i < numSubscribers; i++ {
		_, err := eb.Subscribe(ProcessStarted, func(e Event) {
			receivedEvents.Add(1)
			// Simulate some work
			time.Sleep(time.Microsecond)
		}, nil)
		require.NoError(t, err)
	}
	
	// Start publishers
	wg.Add(numPublishers)
	for i := 0; i < numPublishers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerPub; j++ {
				eb.Publish(Event{
					Type: ProcessStarted,
					Data: map[string]interface{}{
						"publisher": id,
						"event":     j,
					},
				})
			}
		}(i)
	}
	
	// Wait for publishers
	wg.Wait()
	
	// Wait for handlers to complete
	time.Sleep(100 * time.Millisecond)
	
	// Verify all events were received
	expected := int64(numPublishers * eventsPerPub * numSubscribers)
	assert.Equal(t, expected, receivedEvents.Load())
}

func TestImprovedEventBus_HandlerPanic(t *testing.T) {
	eb := NewImprovedEventBus(100)
	defer eb.Shutdown(5 * time.Second)
	
	var (
		normalHandlerCalled atomic.Bool
		panicHandlerCalled  atomic.Bool
	)
	
	// Handler that panics
	_, err := eb.Subscribe(ErrorDetected, func(e Event) {
		panicHandlerCalled.Store(true)
		panic("test panic")
	}, &HandlerOptions{
		RecoverPanic: true,
		NonBlocking:  true,
	})
	require.NoError(t, err)
	
	// Normal handler
	_, err = eb.Subscribe(ErrorDetected, func(e Event) {
		normalHandlerCalled.Store(true)
	}, nil)
	require.NoError(t, err)
	
	// Publish event
	eb.Publish(Event{Type: ErrorDetected})
	
	// Wait for handlers
	time.Sleep(50 * time.Millisecond)
	
	// Both handlers should have been called
	assert.True(t, panicHandlerCalled.Load())
	assert.True(t, normalHandlerCalled.Load())
	
	// Check metrics
	metrics := eb.GetMetrics()
	assert.Equal(t, uint64(1), metrics["failed_handlers"])
}

func TestImprovedEventBus_HandlerTimeout(t *testing.T) {
	eb := NewImprovedEventBus(100)
	defer eb.Shutdown(5 * time.Second)
	
	var timedOut atomic.Bool
	
	// Handler that times out
	_, err := eb.Subscribe(ProcessStarted, func(e Event) {
		select {
		case <-time.After(1 * time.Second):
			// Should not reach here
		case <-context.Background().Done():
			timedOut.Store(true)
		}
	}, &HandlerOptions{
		Timeout:     50 * time.Millisecond,
		NonBlocking: true,
	})
	require.NoError(t, err)
	
	// Publish event
	eb.Publish(Event{Type: ProcessStarted})
	
	// Wait for timeout
	time.Sleep(100 * time.Millisecond)
	
	// Check metrics
	metrics := eb.GetMetrics()
	assert.Equal(t, uint64(1), metrics["timed_out_handlers"])
}

func TestImprovedEventBus_Unsubscribe(t *testing.T) {
	eb := NewImprovedEventBus(100)
	defer eb.Shutdown(5 * time.Second)
	
	var (
		handler1Called atomic.Int32
		handler2Called atomic.Int32
	)
	
	// Subscribe two handlers
	id1, err := eb.Subscribe(LogLine, func(e Event) {
		handler1Called.Add(1)
	}, nil)
	require.NoError(t, err)
	
	id2, err := eb.Subscribe(LogLine, func(e Event) {
		handler2Called.Add(1)
	}, nil)
	require.NoError(t, err)
	
	// Publish first event
	eb.Publish(Event{Type: LogLine})
	time.Sleep(50 * time.Millisecond)
	
	assert.Equal(t, int32(1), handler1Called.Load())
	assert.Equal(t, int32(1), handler2Called.Load())
	
	// Unsubscribe first handler
	err = eb.Unsubscribe(LogLine, id1)
	require.NoError(t, err)
	
	// Publish second event
	eb.Publish(Event{Type: LogLine})
	time.Sleep(50 * time.Millisecond)
	
	// Only handler2 should be called
	assert.Equal(t, int32(1), handler1Called.Load())
	assert.Equal(t, int32(2), handler2Called.Load())
	
	// Test error cases
	err = eb.Unsubscribe(LogLine, "invalid-id")
	assert.Error(t, err)
	
	err = eb.Unsubscribe(ProcessExited, id2)
	assert.Error(t, err)
}

func TestImprovedEventBus_MaxHandlers(t *testing.T) {
	eb := NewImprovedEventBus(5)
	defer eb.Shutdown(5 * time.Second)
	
	// Subscribe up to limit
	for i := 0; i < 5; i++ {
		_, err := eb.Subscribe(ProcessStarted, func(e Event) {}, nil)
		require.NoError(t, err)
	}
	
	// Try to exceed limit
	_, err := eb.Subscribe(ProcessStarted, func(e Event) {}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of handlers")
}

func TestImprovedEventBus_Shutdown(t *testing.T) {
	eb := NewImprovedEventBus(100)
	
	var (
		slowHandlerStarted  atomic.Bool
		slowHandlerFinished atomic.Bool
	)
	
	// Subscribe slow handler
	_, err := eb.Subscribe(ProcessExited, func(e Event) {
		slowHandlerStarted.Store(true)
		time.Sleep(100 * time.Millisecond)
		slowHandlerFinished.Store(true)
	}, nil)
	require.NoError(t, err)
	
	// Publish event
	eb.Publish(Event{Type: ProcessExited})
	
	// Wait for handler to start
	time.Sleep(20 * time.Millisecond)
	assert.True(t, slowHandlerStarted.Load())
	
	// Shutdown with sufficient timeout
	err = eb.Shutdown(200 * time.Millisecond)
	require.NoError(t, err)
	assert.True(t, slowHandlerFinished.Load())
}

func TestImprovedEventBus_ShutdownTimeout(t *testing.T) {
	eb := NewImprovedEventBus(100)
	
	// Subscribe handler that blocks forever
	_, err := eb.Subscribe(ProcessExited, func(e Event) {
		<-make(chan struct{}) // Block forever
	}, nil)
	require.NoError(t, err)
	
	// Publish event
	eb.Publish(Event{Type: ProcessExited})
	
	// Shutdown with short timeout
	err = eb.Shutdown(50 * time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown timed out")
}

func TestImprovedEventBus_SynchronousHandler(t *testing.T) {
	eb := NewImprovedEventBus(100)
	defer eb.Shutdown(5 * time.Second)
	
	var callOrder []int
	var mu sync.Mutex
	
	// Subscribe synchronous handler
	_, err := eb.Subscribe(BuildEvent, func(e Event) {
		mu.Lock()
		callOrder = append(callOrder, 1)
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}, &HandlerOptions{
		NonBlocking: false,
	})
	require.NoError(t, err)
	
	// Subscribe another synchronous handler
	_, err = eb.Subscribe(BuildEvent, func(e Event) {
		mu.Lock()
		callOrder = append(callOrder, 2)
		mu.Unlock()
	}, &HandlerOptions{
		NonBlocking: false,
	})
	require.NoError(t, err)
	
	// Publish event
	start := time.Now()
	eb.Publish(Event{Type: BuildEvent})
	duration := time.Since(start)
	
	// Should take at least 50ms (handlers run sequentially)
	assert.GreaterOrEqual(t, duration, 50*time.Millisecond)
	
	// Check order
	assert.Equal(t, []int{1, 2}, callOrder)
}

func TestImprovedEventBus_RetryLogic(t *testing.T) {
	eb := NewImprovedEventBus(100)
	defer eb.Shutdown(5 * time.Second)
	
	var attempts atomic.Int32
	
	// Handler that fails first 2 times
	_, err := eb.Subscribe(TestFailed, func(e Event) {
		count := attempts.Add(1)
		if count < 3 {
			panic(errors.New("simulated failure"))
		}
	}, &HandlerOptions{
		MaxRetries:   3,
		RetryDelay:   10 * time.Millisecond,
		RecoverPanic: true,
	})
	require.NoError(t, err)
	
	// Publish event
	eb.Publish(Event{Type: TestFailed})
	
	// Wait for retries
	time.Sleep(100 * time.Millisecond)
	
	// Should have succeeded on 3rd attempt
	assert.Equal(t, int32(3), attempts.Load())
}

func TestImprovedEventBus_UnsubscribeAll(t *testing.T) {
	eb := NewImprovedEventBus(100)
	defer eb.Shutdown(5 * time.Second)
	
	var callCount atomic.Int32
	
	// Subscribe multiple handlers
	for i := 0; i < 5; i++ {
		_, err := eb.Subscribe(MCPActivity, func(e Event) {
			callCount.Add(1)
		}, nil)
		require.NoError(t, err)
	}
	
	// Publish and verify
	eb.Publish(Event{Type: MCPActivity})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(5), callCount.Load())
	
	// Unsubscribe all
	eb.UnsubscribeAll(MCPActivity)
	
	// Reset counter
	callCount.Store(0)
	
	// Publish again
	eb.Publish(Event{Type: MCPActivity})
	time.Sleep(50 * time.Millisecond)
	
	// No handlers should be called
	assert.Equal(t, int32(0), callCount.Load())
}

// Benchmark concurrent publish/subscribe
func BenchmarkImprovedEventBus_ConcurrentPubSub(b *testing.B) {
	eb := NewImprovedEventBus(1000)
	defer eb.Shutdown(5 * time.Second)
	
	// Subscribe handlers
	for i := 0; i < 10; i++ {
		eb.Subscribe(ProcessStarted, func(e Event) {
			// Minimal work
			_ = e.ID
		}, nil)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			eb.Publish(Event{Type: ProcessStarted})
		}
	})
}