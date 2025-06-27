package test

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoroutineLeaks verifies no goroutines are leaked
func TestGoroutineLeaks(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	baseline := runtime.NumGoroutine()
	
	// Run test multiple times to detect leaks
	for i := 0; i < 5; i++ {
		func() {
			// Create services
			baseEB := events.NewEventBus()
			improvedEB := events.NewImprovedEventBus(1000)
			ls := logs.NewStore(1000)
			pm, _ := process.NewManager("test", baseEB, false)
			ps := proxy.NewServer(0, baseEB)
			
			// For this test, we'll use the regular StreamableServer
			// since ImprovedStreamableServer is just a demonstration
			server := mcp.NewStreamableServer(0, pm, ls, ps, baseEB)
			err := server.Start()
			require.NoError(t, err)
			
			// Simulate some activity
			baseEB.Publish(events.Event{Type: events.ProcessStarted})
			baseEB.Publish(events.Event{Type: events.LogLine})
			baseEB.Publish(events.Event{Type: events.ErrorDetected})
			
			time.Sleep(50 * time.Millisecond)
			
			// Stop everything
			err = server.Stop()
			assert.NoError(t, err)
			
			err = improvedEB.Shutdown(1 * time.Second)
			assert.NoError(t, err)
			
			pm.Cleanup()
			ps.Stop()
		}()
		
		// Allow goroutines to clean up
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
	}
	
	// Check final goroutine count
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	final := runtime.NumGoroutine()
	
	// Allow some variance but detect significant leaks
	leak := final - baseline
	assert.LessOrEqual(t, leak, 5, "Detected goroutine leak: baseline=%d, final=%d, leaked=%d", 
		baseline, final, leak)
}

// TestConcurrentLogWrites tests concurrent access to log store
func TestConcurrentLogWrites(t *testing.T) {
	ls := logs.NewStore(10000)
	
	const (
		writers = 50
		writes  = 100
	)
	
	var wg sync.WaitGroup
	errors := atomic.Int32{}
	
	// Start concurrent writers
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < writes; j++ {
				ls.Add(
					fmt.Sprintf("proc-%d", id),
					"TEST",
					fmt.Sprintf("Message %d from writer %d", j, id),
					j%10 == 0, // Some are errors
				)
				
				// Also test URL detection
				if j%5 == 0 {
					ls.Add(
						fmt.Sprintf("proc-%d", id),
						"URL",
						fmt.Sprintf("Server at http://localhost:%d/test", 8000+id),
						false,
					)
				}
			}
		}(i)
	}
	
	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for j := 0; j < 50; j++ {
				// Test various read operations
				logs := ls.GetAll()
				_ = len(logs)
				
				errors := ls.GetErrors()
				_ = len(errors)
				
				// Get last N logs
				all := ls.GetAll()
				if len(all) > 100 {
					recent := all[len(all)-100:]
					_ = len(recent)
				}
				
				// Check URLs in logs
				for _, log := range all {
					if strings.Contains(log.Text, "http://") {
						// Found URL
						break
					}
				}
				
				time.Sleep(time.Microsecond)
			}
		}()
	}
	
	wg.Wait()
	
	// Verify data integrity
	allLogs := ls.GetAll()
	assert.GreaterOrEqual(t, len(allLogs), writers*writes)
	
	// Check no data corruption
	for _, log := range allLogs {
		assert.NotEmpty(t, log.ProcessID)
		assert.NotEmpty(t, log.Text)
		assert.NotZero(t, log.Timestamp)
	}
}

// TestMemoryExhaustion tests behavior under memory pressure
func TestMemoryExhaustion(t *testing.T) {
	t.Skip("Memory exhaustion test - enable manually")
	
	// Create log store with small limit
	ls := logs.NewStore(100)
	
	// Try to overwhelm it
	for i := 0; i < 10000; i++ {
		ls.Add("stress", "TEST", 
			strings.Repeat("X", 1000), // Large message
			false,
		)
	}
	
	// Should still be functional
	logs := ls.GetAll()
	assert.LessOrEqual(t, len(logs), 110) // Allow some buffer
	
	// Should be able to get errors
	errors := ls.GetErrors()
	assert.NotNil(t, errors)
}

// TestPanicRecovery tests panic handling in event handlers
func TestPanicRecovery(t *testing.T) {
	eb := events.NewImprovedEventBus(100)
	defer eb.Shutdown(1 * time.Second)
	
	var (
		beforePanic atomic.Bool
		afterPanic  atomic.Bool
		normalHandler atomic.Bool
	)
	
	// Handler that panics
	eb.Subscribe(events.TestFailed, func(e events.Event) {
		beforePanic.Store(true)
		panic("test panic")
		afterPanic.Store(true) // Should not execute
	}, &events.HandlerOptions{
		RecoverPanic: true,
	})
	
	// Normal handler should still run
	eb.Subscribe(events.TestFailed, func(e events.Event) {
		normalHandler.Store(true)
	}, nil)
	
	// Publish event
	eb.Publish(events.Event{Type: events.TestFailed})
	
	time.Sleep(50 * time.Millisecond)
	
	assert.True(t, beforePanic.Load())
	assert.False(t, afterPanic.Load())
	assert.True(t, normalHandler.Load())
	
	// System should still be functional
	metrics := eb.GetMetrics()
	assert.Equal(t, uint64(1), metrics["failed_handlers"])
}

// TestRapidConnectionChurn tests handling of rapid connect/disconnect
func TestRapidConnectionChurn(t *testing.T) {
	cm := mcp.NewConnectionManager()
	defer cm.Stop()
	
	const (
		instances = 20
		cycles    = 50
	)
	
	var wg sync.WaitGroup
	errors := atomic.Int32{}
	
	// Simulate rapid instance registration/connection
	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < cycles; j++ {
				// Register
				instance := &discovery.Instance{
					ID:   fmt.Sprintf("instance-%d", id),
					Name: fmt.Sprintf("Test %d", id),
					Port: 8000 + id,
				}
				
				if err := cm.RegisterInstance(instance); err != nil {
					errors.Add(1)
					continue
				}
				
				// Connect session
				sessionID := fmt.Sprintf("session-%d-%d", id, j)
				if err := cm.ConnectSession(sessionID, instance.ID); err != nil {
					errors.Add(1)
				}
				
				// Brief activity
				time.Sleep(time.Microsecond)
				
				// Disconnect
				cm.DisconnectSession(sessionID)
				
				// Occasionally list instances
				if j%10 == 0 {
					instances := cm.ListInstances()
					_ = len(instances)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Should handle churn without errors
	assert.Equal(t, int32(0), errors.Load())
	
	// Final state should be consistent
	finalInstances := cm.ListInstances()
	for _, inst := range finalInstances {
		assert.NotEmpty(t, inst.InstanceID)
		assert.GreaterOrEqual(t, inst.Port, 8000)
	}
}

// TestDeadlockDetection tests for potential deadlocks
func TestDeadlockDetection(t *testing.T) {
	// Set deadlock detector
	done := make(chan struct{})
	go func() {
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-done:
			timer.Stop()
		case <-timer.C:
			buf := make([]byte, 1<<20)
			runtime.Stack(buf, true)
			t.Errorf("Deadlock detected! Stack:\n%s", buf)
		}
	}()
	
	// Run operations that might deadlock
	eb := events.NewImprovedEventBus(100)
	ls := logs.NewStore(1000)
	
	// Nested event publishing (potential deadlock scenario)
	eb.Subscribe(events.ProcessStarted, func(e events.Event) {
		// Publishing from within handler
		eb.Publish(events.Event{Type: events.LogLine})
	}, nil)
	
	eb.Subscribe(events.LogLine, func(e events.Event) {
		// Another level of nesting
		eb.Publish(events.Event{Type: events.ErrorDetected})
	}, nil)
	
	// Start the chain
	eb.Publish(events.Event{Type: events.ProcessStarted})
	
	// Concurrent operations on log store
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				ls.Add(fmt.Sprintf("proc-%d", id), "TEST", "Message", false)
				ls.GetAll()
				ls.GetErrors()
			}
		}(i)
	}
	
	wg.Wait()
	eb.Shutdown(1 * time.Second)
	
	// Signal completion
	close(done)
}

// BenchmarkConcurrentEventProcessing benchmarks event handling under load
func BenchmarkConcurrentEventProcessing(b *testing.B) {
	eb := events.NewImprovedEventBus(10000)
	defer eb.Shutdown(5 * time.Second)
	
	// Add multiple handlers
	for i := 0; i < 10; i++ {
		eb.Subscribe(events.LogLine, func(e events.Event) {
			// Simulate some processing
			_ = e.Data
		}, nil)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			eb.Publish(events.Event{
				Type: events.LogLine,
				Data: map[string]interface{}{
					"line": "test log message",
				},
			})
		}
	})
}

// BenchmarkLogStoreOperations benchmarks concurrent log operations
func BenchmarkLogStoreOperations(b *testing.B) {
	ls := logs.NewStore(10000)
	
	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		ls.Add(fmt.Sprintf("proc-%d", i%10), "TEST", 
			fmt.Sprintf("Message %d", i), i%10 == 0)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				ls.Add("bench", "TEST", "Message", false)
			case 1:
				ls.GetAll()
			case 2:
				all := ls.GetAll()
				if len(all) > 100 {
					_ = all[len(all)-100:]
				}
			case 3:
				ls.GetErrors()
			}
			i++
		}
	})
}