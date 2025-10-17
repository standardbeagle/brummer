package tui

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// TestSafePTYBufferRaceConditions tests the safe buffer for race conditions
func TestSafePTYBufferRaceConditions(t *testing.T) {
	// Enable race detection: go test -race -run TestSafePTYBufferRaceConditions

	t.Log("🧪 Testing SafePTYBuffer for race conditions...")

	buffer := NewSafePTYBuffer(1024)

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // writers + readers

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Writer goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(writerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					// Create unique message
					msg := &SafePTYMessage{
						SessionID: fmt.Sprintf("writer-%d", writerID),
						Data:      []byte(fmt.Sprintf("message-%d-%d", writerID, j)),
						Timestamp: time.Now(),
						Type:      SafePTYMessageOutput,
						Sequence:  uint64(writerID*numOperations + j),
					}

					// Try to write - this should be thread-safe
					buffer.TryWrite(msg)

					// Add some delay to increase chance of races
					if j%100 == 0 {
						time.Sleep(time.Microsecond)
					}
				}
			}
		}(i)
	}

	// Reader goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(readerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					// Try to read batch - this should be thread-safe
					_, count := buffer.TryReadBatch(10)

					// Process batch
					if count > 0 {
						// Simulate processing
						time.Sleep(time.Microsecond)
					}

					// Add some delay to increase chance of races
					if j%100 == 0 {
						time.Sleep(time.Microsecond)
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify buffer is still in a consistent state
	metrics := buffer.GetMetrics()
	t.Logf("Buffer metrics after stress test: %+v", metrics)

	// Buffer should be functional
	if metrics.BufferSize == 0 {
		t.Error("Buffer size should not be zero after operations")
	}
}

// TestLockFreePTYManagerRaceConditions tests the PTY manager for race conditions
func TestLockFreePTYManagerRaceConditions(t *testing.T) {
	// Enable race detection: go test -race -run TestLockFreePTYManagerRaceConditions

	t.Log("🧪 Testing LockFreePTYManager for race conditions...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	updateChan := make(chan tea.Msg, 1000)
	logStore := logs.NewStore(1000, nil)

	manager := NewLockFreePTYManager(ctx, updateChan, logStore)
	defer manager.Shutdown()

	const numSessions = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numSessions)

	// Start multiple sessions concurrently
	for i := 0; i < numSessions; i++ {
		go func(sessionID int) {
			defer wg.Done()

			// Create mock PTY session
			mockSession := &aicoder.PTYSession{
				ID:         fmt.Sprintf("test-session-%d", sessionID),
				Name:       fmt.Sprintf("Test Session %d", sessionID),
				OutputChan: make(chan []byte, 100),
				EventChan:  make(chan aicoder.PTYEvent, 10),
			}

			// Start session
			err := manager.StartSession(mockSession)
			if err != nil {
				t.Errorf("Failed to start session %d: %v", sessionID, err)
				return
			}

			// Simulate PTY output
			for j := 0; j < numOperations; j++ {
				select {
				case <-ctx.Done():
					return
				case mockSession.OutputChan <- []byte(fmt.Sprintf("output-%d-%d\n", sessionID, j)):
					// Output sent
				}

				// Add delay to simulate realistic PTY timing
				time.Sleep(time.Millisecond)
			}

			// Stop session
			err = manager.StopSession(mockSession.ID)
			if err != nil {
				t.Errorf("Failed to stop session %d: %v", sessionID, err)
			}
		}(i)
	}

	// Process messages from all sessions concurrently
	for i := 0; i < 50; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Process session messages
					cmd := manager.ProcessSessionMessages()
					if cmd != nil {
						msg := cmd()
						if msg != nil {
							// Process the batch message
							select {
							case updateChan <- msg:
							case <-ctx.Done():
								return
							}
						}
					}
					time.Sleep(time.Millisecond * 10)
				}
			}
		}()
	}

	// Wait for all sessions to complete
	wg.Wait()

	// Verify manager is in a consistent state
	metrics := manager.GetMetrics()
	t.Logf("Manager metrics after stress test: %+v", metrics)

	// All sessions should be stopped
	if metrics.ActiveSessions > 0 {
		t.Errorf("Expected no active sessions, got %d", metrics.ActiveSessions)
	}
}

// TestFileBrowserServiceRaceConditions tests the file browser service for race conditions
func TestFileBrowserServiceRaceConditions(t *testing.T) {
	// Enable race detection: go test -race -run TestFileBrowserServiceRaceConditions

	t.Log("🧪 Testing FileBrowserService for race conditions...")

	eventBus := events.NewEventBus()
	config := services.DefaultFileBrowserConfig()
	config.EnableFileWatching = false // Disable watching to simplify test

	service := services.NewFileBrowserService(eventBus, config)
	defer service.Stop()

	const numGoroutines = 50
	const numOperations = 200

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent directory listings
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// List current directory
				listing := service.ListDirectory()
				if listing == nil {
					t.Errorf("Directory listing should not be nil (goroutine %d, op %d)", goroutineID, j)
				}

				// Get current path
				path := service.GetCurrentPath()
				if path == "" {
					t.Errorf("Current path should not be empty (goroutine %d, op %d)", goroutineID, j)
				}

				// Get workspace root
				root := service.GetWorkspaceRoot()
				if root == "" {
					t.Errorf("Workspace root should not be empty (goroutine %d, op %d)", goroutineID, j)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	t.Log("FileBrowserService race condition test completed successfully")
}

// TestAtomicValueRaceConditions tests atomic value usage patterns
func TestAtomicValueRaceConditions(t *testing.T) {
	// Enable race detection: go test -race -run TestAtomicValueRaceConditions

	t.Log("🧪 Testing atomic value patterns for race conditions...")

	const numGoroutines = 100
	const numOperations = 1000

	var counter atomic.Uint64
	var flag atomic.Bool

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // readers, writers, and flag togglers

	// Writer goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(writerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Atomic increment - should be race-free
				counter.Add(1)

				// Add small delay to increase chance of races
				if j%100 == 0 {
					time.Sleep(time.Nanosecond)
				}
			}
		}(i)
	}

	// Reader goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(readerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Atomic read - should be race-free
				value := counter.Load()

				if value > numGoroutines*numOperations {
					t.Errorf("Counter value %d exceeds maximum possible %d", value, numGoroutines*numOperations)
				}

				// Add small delay
				if j%100 == 0 {
					time.Sleep(time.Nanosecond)
				}
			}
		}(i)
	}

	// Flag toggler goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(togglerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Atomic flag toggle - should be race-free
				current := flag.Load()
				flag.Store(!current)

				// Add small delay
				if j%100 == 0 {
					time.Sleep(time.Nanosecond)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify final state
	finalValue := counter.Load()
	expectedMin := uint64(numGoroutines * numOperations)

	if finalValue < expectedMin {
		t.Errorf("Final counter value %d is less than expected minimum %d", finalValue, expectedMin)
	}

	t.Logf("Atomic operations completed successfully. Final counter: %d", finalValue)
}

// TestChannelTimeoutSafety tests timeout-based channel patterns
func TestChannelTimeoutSafety(t *testing.T) {
	// Enable race detection: go test -race -run TestChannelTimeoutSafety

	t.Log("🧪 Testing channel timeout safety patterns...")

	const numGoroutines = 20
	timeout := 100 * time.Millisecond

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test channels with timeout patterns
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			// Create a channel that won't receive data
			ch := make(chan string)

			for j := 0; j < 10; j++ {
				select {
				case msg := <-ch:
					t.Errorf("Should not receive message on empty channel (goroutine %d, iteration %d): %s", goroutineID, j, msg)
				case <-time.After(timeout):
					// Timeout is expected and prevents deadlock
					// This is the correct pattern for non-blocking channel operations
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	t.Log("Channel timeout safety test completed successfully")
}

// TestDirectPTYPassthroughRaceConditions tests the direct PTY passthrough for race conditions
func TestDirectPTYPassthroughRaceConditions(t *testing.T) {
	// Enable race detection: go test -race -run TestDirectPTYPassthroughRaceConditions

	t.Log("🧪 Testing DirectPTYPassthrough for race conditions...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logStore := logs.NewStore(1000, nil)

	// Create mock PTY session
	mockSession := &aicoder.PTYSession{
		ID:         "test-direct-pty",
		Name:       "Test Direct PTY Session",
		OutputChan: make(chan []byte, 100),
		EventChan:  make(chan aicoder.PTYEvent, 10),
	}

	// Create direct PTY passthrough
	passthrough := NewDirectPTYPassthrough(ctx, mockSession, logStore)

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup

	// Start multiple goroutines that access the passthrough concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					// Test safe session access
					session := passthrough.getSession()
					if session == nil {
						t.Errorf("Session should not be nil (goroutine %d, op %d)", goroutineID, j)
						continue
					}

					// Test metrics access
					metrics := passthrough.GetMetrics()
					// Note: PassthroughMode can be true even when Active is false (not started yet)
					// This is expected behavior, not an invalid state
					_ = metrics // Consume the metrics to ensure no races during access

					// Test mode switching
					passthrough.SetPassthroughMode(j%2 == 0)

					// Small delay to increase chance of races
					time.Sleep(time.Microsecond)
				}
			}
		}(i)
	}

	// Test concurrent data creation
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			testData := []byte(fmt.Sprintf("direct PTY test data from goroutine %d", goroutineID))

			for j := 0; j < numOperations; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					// Test safe command creation
					cmd := CreateDirectPTYCmd(fmt.Sprintf("session-%d", goroutineID), testData, true)
					if cmd == nil {
						t.Errorf("Command should not be nil (goroutine %d, op %d)", goroutineID, j)
						continue
					}

					// Execute command to test message creation
					msg := cmd()
					if msg == nil {
						t.Errorf("Message should not be nil (goroutine %d, op %d)", goroutineID, j)
						continue
					}

					// Verify data integrity
					directMsg, ok := msg.(DirectPTYMessage)
					if !ok {
						t.Errorf("Wrong message type (goroutine %d, op %d)", goroutineID, j)
						continue
					}

					if string(directMsg.Data) != string(testData) {
						t.Errorf("Data corruption detected (goroutine %d, op %d)", goroutineID, j)
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Final verification
	session := passthrough.getSession()
	if session == nil {
		t.Error("Final session access failed")
	}

	metrics := passthrough.GetMetrics()
	t.Logf("Final passthrough metrics: %+v", metrics)

	// Test cleanup
	passthrough.Stop()

	t.Log("Direct PTY passthrough race condition test completed successfully")
}

// BenchmarkConcurrentOperations benchmarks concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("SafeBuffer", func(b *testing.B) {
		buffer := NewSafePTYBuffer(1024)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				msg := &SafePTYMessage{
					SessionID: "benchmark",
					Data:      []byte("benchmark data"),
					Timestamp: time.Now(),
					Type:      SafePTYMessageOutput,
					Sequence:  0,
				}
				buffer.TryWrite(msg)
				buffer.TryRead()
			}
		})
	})

	b.Run("AtomicOperations", func(b *testing.B) {
		var counter atomic.Uint64

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter.Add(1)
				counter.Load()
			}
		})
	})
}