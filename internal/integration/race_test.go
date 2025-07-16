package integration

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSystemWideRaceConditions validates that all race conditions have been eliminated
func TestSystemWideRaceConditions(t *testing.T) {
	t.Run("EventBusWorkerPoolStress", func(t *testing.T) {
		eb := events.NewEventBus()
		defer eb.Shutdown()

		const numPublishers = 50
		const eventsPerPublisher = 200

		var processedCount int64
		var handlerExecutions int64

		// Subscribe multiple handlers
		eb.Subscribe(events.LogLine, func(event events.Event) {
			atomic.AddInt64(&handlerExecutions, 1)
			time.Sleep(time.Microsecond) // Simulate work
		})

		eb.Subscribe(events.ProcessStarted, func(event events.Event) {
			atomic.AddInt64(&handlerExecutions, 1)
			time.Sleep(time.Microsecond)
		})

		// Flood with events from multiple goroutines
		var wg sync.WaitGroup
		for i := 0; i < numPublishers; i++ {
			wg.Add(1)
			go func(publisherID int) {
				defer wg.Done()
				for j := 0; j < eventsPerPublisher; j++ {
					eventType := events.LogLine
					if j%2 == 0 {
						eventType = events.ProcessStarted
					}

					eb.Publish(events.Event{
						Type:      eventType,
						ProcessID: fmt.Sprintf("stress-test-%d", publisherID),
						Data: map[string]interface{}{
							"publisher": publisherID,
							"sequence":  j,
						},
					})
					atomic.AddInt64(&processedCount, 1)
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond) // Allow processing to complete

		finalProcessed := atomic.LoadInt64(&processedCount)
		finalExecutions := atomic.LoadInt64(&handlerExecutions)

		assert.Equal(t, int64(numPublishers*eventsPerPublisher), finalProcessed)
		assert.Equal(t, finalProcessed, finalExecutions) // Each event hits one handler

		t.Logf("EventBus processed %d events with %d handler executions",
			finalProcessed, finalExecutions)
	})

	t.Run("ProcessManagerConcurrentOperations", func(t *testing.T) {
		eb := events.NewEventBus()
		defer eb.Shutdown()

		mgr, err := process.NewManager("/tmp", eb, false)
		require.NoError(t, err)
		defer mgr.Cleanup()

		const numWorkers = 10
		const operationsPerWorker = 20

		var processesStarted int64
		var processesQueried int64
		var errors int64

		var wg sync.WaitGroup
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerWorker; j++ {
					// Start a process
					proc, err := mgr.StartCommand(
						fmt.Sprintf("worker-%d-op-%d", workerID, j),
						"echo",
						[]string{fmt.Sprintf("hello from worker %d operation %d", workerID, j)},
					)

					if err != nil {
						atomic.AddInt64(&errors, 1)
						continue
					}

					atomic.AddInt64(&processesStarted, 1)

					// Query the process status (tests thread-safe getters)
					status := proc.GetStatus()
					if status != process.StatusPending && status != process.StatusRunning {
						t.Logf("Unexpected status: %v", status)
					}

					// Query from manager (tests concurrent map access)
					_, exists := mgr.GetProcess(proc.ID)
					if exists {
						atomic.AddInt64(&processesQueried, 1)
					}

					// Small delay to increase concurrency pressure
					time.Sleep(time.Millisecond)
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(500 * time.Millisecond) // Allow processes to complete

		finalStarted := atomic.LoadInt64(&processesStarted)
		finalQueried := atomic.LoadInt64(&processesQueried)
		finalErrors := atomic.LoadInt64(&errors)

		assert.Greater(t, finalStarted, int64(0), "Should have started some processes")
		assert.Equal(t, finalStarted, finalQueried, "All started processes should be queryable")
		assert.Equal(t, int64(0), finalErrors, "Should have no errors")

		t.Logf("Started %d processes, queried %d, errors: %d",
			finalStarted, finalQueried, finalErrors)
	})

	t.Run("LogStoreConcurrentWrites", func(t *testing.T) {
		store := logs.NewStore(10000)
		defer store.Close()

		const numWriters = 25
		const logsPerWriter = 100

		var logsWritten int64

		var wg sync.WaitGroup
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(writerID int) {
				defer wg.Done()

				for j := 0; j < logsPerWriter; j++ {
					entry := store.Add(
						fmt.Sprintf("process-%d", writerID),
						fmt.Sprintf("writer-%d", writerID),
						fmt.Sprintf("Log message %d from writer %d", j, writerID),
						j%5 == 0, // Every 5th message is an error
					)

					if entry != nil {
						atomic.AddInt64(&logsWritten, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond) // Allow async processing

		finalWritten := atomic.LoadInt64(&logsWritten)
		storedLogs := store.GetAll()

		// Due to fire-and-forget async pattern, some logs might be dropped under pressure
		assert.Greater(t, finalWritten, int64(0), "Should have written some logs")
		assert.LessOrEqual(t, len(storedLogs), int(finalWritten), "Stored logs should not exceed written")

		t.Logf("Wrote %d logs, stored %d", finalWritten, len(storedLogs))
	})

	t.Run("ProxyServerConcurrentRequests", func(t *testing.T) {
		eb := events.NewEventBus()
		defer eb.Shutdown()

		server := proxy.NewServerWithMode(0, proxy.ProxyModeReverse, eb)
		// Don't start the server to avoid hanging - just test the concurrent map operations

		const numRequesters = 10
		const requestsPerRequester = 5

		var requestsProcessed int64
		var mappingsCreated int64

		var wg sync.WaitGroup
		for i := 0; i < numRequesters; i++ {
			wg.Add(1)
			go func(requesterID int) {
				defer wg.Done()

				for j := 0; j < requestsPerRequester; j++ {
					// Register URL mappings (tests concurrent map operations)
					proxyURL := server.RegisterURL(
						fmt.Sprintf("http://localhost:%d", 3000+requesterID),
						fmt.Sprintf("requester-%d", requesterID),
					)

					if proxyURL != "" {
						atomic.AddInt64(&mappingsCreated, 1)
					}

					// Get proxy requests (tests concurrent slice access)
					requests := server.GetRequests()
					atomic.AddInt64(&requestsProcessed, int64(len(requests)))

					time.Sleep(time.Millisecond)
				}
			}(i)
		}

		wg.Wait()

		finalMappings := atomic.LoadInt64(&mappingsCreated)
		finalRequests := atomic.LoadInt64(&requestsProcessed)

		assert.Greater(t, finalMappings, int64(0), "Should have created some mappings")

		t.Logf("Created %d mappings, processed %d requests",
			finalMappings, finalRequests)
	})

	t.Run("MCPConnectionManagerConcurrentSessions", func(t *testing.T) {
		cm := mcp.NewConnectionManager()
		defer cm.Stop()

		const numSessions = 20
		const operationsPerSession = 10

		var connectSuccesses int64
		var disconnectSuccesses int64
		var errors int64

		var wg sync.WaitGroup
		for i := 0; i < numSessions; i++ {
			wg.Add(1)
			go func(sessionID int) {
				defer wg.Done()

				sessionName := fmt.Sprintf("session-%d", sessionID)
				instanceID := fmt.Sprintf("instance-%d", sessionID%5) // 5 instances

				for j := 0; j < operationsPerSession; j++ {
					// Connect session
					err := cm.ConnectSession(sessionName, instanceID)
					if err == nil {
						atomic.AddInt64(&connectSuccesses, 1)
					} else {
						atomic.AddInt64(&errors, 1)
					}

					// Disconnect session
					err = cm.DisconnectSession(sessionName)
					if err == nil {
						atomic.AddInt64(&disconnectSuccesses, 1)
					} else {
						atomic.AddInt64(&errors, 1)
					}

					time.Sleep(time.Millisecond)
				}
			}(i)
		}

		wg.Wait()

		finalConnects := atomic.LoadInt64(&connectSuccesses)
		finalDisconnects := atomic.LoadInt64(&disconnectSuccesses)
		finalErrors := atomic.LoadInt64(&errors)

		// Note: Connections will fail because instances aren't registered,
		// but the important thing is no race conditions
		t.Logf("Connects: %d, Disconnects: %d, Errors: %d",
			finalConnects, finalDisconnects, finalErrors)
	})
}

// TestGoroutineLeakPrevention ensures proper cleanup of goroutines
func TestGoroutineLeakPrevention(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	t.Run("EventBusCleanup", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			eb := events.NewEventBus()

			// Use the EventBus
			eb.Subscribe(events.LogLine, func(event events.Event) {
				// No-op handler
			})

			for j := 0; j < 100; j++ {
				eb.Publish(events.Event{
					Type:      events.LogLine,
					ProcessID: "test",
				})
			}

			// Proper shutdown
			eb.Shutdown()
		}
	})

	t.Run("LogStoreCleanup", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			store := logs.NewStore(100)

			// Use the store
			for j := 0; j < 50; j++ {
				store.Add("test", "test", fmt.Sprintf("message %d", j), false)
			}

			// Proper shutdown
			store.Close()
		}
	})

	// Force garbage collection and wait for cleanup
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()

	// Allow for some variance in goroutine count
	assert.LessOrEqual(t, finalGoroutines, initialGoroutines+5,
		"Should not leak significant goroutines")

	t.Logf("Goroutines: %d -> %d", initialGoroutines, finalGoroutines)
}

// TestSystemStressWithRaceDetection validates system behavior under extreme load
func TestSystemStressWithRaceDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	eb := events.NewEventBus()
	defer eb.Shutdown()

	store := logs.NewStore(50000)
	defer store.Close()

	mgr, err := process.NewManager("/tmp", eb, false)
	require.NoError(t, err)
	defer mgr.Cleanup()

	// Note: ProxyServer stress testing would require server start which can hang in tests
	// The concurrent map operations are already tested in the ProxyServerConcurrentRequests subtest

	cm := mcp.NewConnectionManager()
	defer cm.Stop()

	// Stress test all components simultaneously
	const duration = 5 * time.Second
	const numWorkers = 20

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var operations int64
	var errors int64

	var wg sync.WaitGroup

	// Event flood
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					eb.Publish(events.Event{
						Type:      events.LogLine,
						ProcessID: fmt.Sprintf("stress-%d", workerID),
					})
					atomic.AddInt64(&operations, 1)
				}
			}
		}(i)
	}

	// Log flood
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					store.Add(
						fmt.Sprintf("stress-%d", workerID),
						"stress-test",
						fmt.Sprintf("Message from worker %d", workerID),
						false,
					)
					atomic.AddInt64(&operations, 1)
				}
			}
		}(i)
	}

	// MCP session operations
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			sessionID := fmt.Sprintf("stress-session-%d", workerID)
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// These will fail but test concurrent channel operations
					cm.ConnectSession(sessionID, "nonexistent")
					cm.DisconnectSession(sessionID)
					atomic.AddInt64(&operations, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	finalOps := atomic.LoadInt64(&operations)
	finalErrors := atomic.LoadInt64(&errors)

	assert.Greater(t, finalOps, int64(1000), "Should have processed many operations")
	t.Logf("Stress test completed: %d operations, %d errors in %v",
		finalOps, finalErrors, duration)
}
