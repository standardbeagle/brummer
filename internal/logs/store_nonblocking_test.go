package logs

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStoreNonBlockingAdd verifies that Add operations don't block
func TestStoreNonBlockingAdd(t *testing.T) {
	store := NewStore(1000, nil)
	defer store.Close()

	// Track goroutine count
	startGoroutines := runtime.NumGoroutine()

	// Measure time for Add operations
	start := time.Now()

	// Add many logs concurrently
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			store.Add(
				fmt.Sprintf("process-%d", id),
				"test-process",
				fmt.Sprintf("Log message %d", id),
				false,
			)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Should complete quickly (non-blocking)
	assert.Less(t, elapsed, 100*time.Millisecond, "Add operations took too long: %v", elapsed)

	// Check goroutine count didn't explode
	endGoroutines := runtime.NumGoroutine()
	assert.LessOrEqual(t, endGoroutines-startGoroutines, 5, "Too many goroutines created")
}

// TestStoreConcurrentAddGet verifies no deadlocks with concurrent operations
func TestStoreConcurrentAddGet(t *testing.T) {
	store := NewStore(100, nil)
	defer store.Close()

	var addCount atomic.Int32
	var getCount atomic.Int32
	done := make(chan struct{})

	// Start producers
	for i := 0; i < 5; i++ {
		go func(id int) {
			for {
				select {
				case <-done:
					return
				default:
					store.Add(
						fmt.Sprintf("process-%d", id),
						"producer",
						fmt.Sprintf("Message from producer %d", id),
						false,
					)
					addCount.Add(1)
					time.Sleep(time.Microsecond)
				}
			}
		}(i)
	}

	// Start consumers
	for i := 0; i < 3; i++ {
		go func(id int) {
			for {
				select {
				case <-done:
					return
				default:
					logs := store.GetAll()
					if len(logs) > 0 {
						getCount.Add(1)
					}
					time.Sleep(time.Microsecond)
				}
			}
		}(i)
	}

	// Let it run for a short time
	time.Sleep(100 * time.Millisecond)
	close(done)

	// Give goroutines time to exit
	time.Sleep(10 * time.Millisecond)

	// Verify both operations succeeded many times
	adds := addCount.Load()
	gets := getCount.Load()

	assert.Greater(t, adds, int32(100), "Not enough add operations: %d", adds)
	assert.Greater(t, gets, int32(50), "Not enough get operations: %d", gets)
}

// TestStoreChannelBackpressure verifies channel doesn't grow unbounded
func TestStoreChannelBackpressure(t *testing.T) {
	store := NewStore(10, nil)
	defer store.Close()

	// Flood with adds
	start := time.Now()
	for i := 0; i < 10000; i++ {
		store.Add("flood-test", "flood", fmt.Sprintf("Message %d", i), false)
	}
	elapsed := time.Since(start)

	// Should complete in reasonable time
	assert.Less(t, elapsed, 2*time.Second, "Flooding took too long: %v", elapsed)

	// Verify we didn't store all messages (backpressure worked)
	logs := store.GetAll()
	assert.LessOrEqual(t, len(logs), 1100, "Too many logs stored, backpressure failed")
}

// TestStoreAsyncProcessing verifies async processing works correctly
func TestStoreAsyncProcessing(t *testing.T) {
	store := NewStore(100, nil)
	defer store.Close()

	// Add a log with URL
	store.Add("url-test", "server", "Server running on http://localhost:3000", false)

	// Give async processing time
	time.Sleep(10 * time.Millisecond)

	// Check URL was detected
	urls := store.GetURLs()
	require.Len(t, urls, 1)
	assert.Equal(t, "http://localhost:3000", urls[0].URL)
}

// BenchmarkStoreAddNonBlocking benchmarks non-blocking add performance
func BenchmarkStoreAddNonBlocking(b *testing.B) {
	store := NewStore(10000, nil)
	defer store.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			store.Add(
				"bench-process",
				"benchmark",
				fmt.Sprintf("Benchmark message %d", i),
				false,
			)
			i++
		}
	})
}

// BenchmarkStoreAddBlocking benchmarks the old blocking implementation
func BenchmarkStoreAddBlocking(b *testing.B) {
	// This would benchmark the old implementation for comparison
	// Skipping actual implementation as we're testing the new one
	b.Skip("Old implementation not available")
}

// TestStoreNoDeadlock verifies no deadlocks occur
func TestStoreNoDeadlock(t *testing.T) {
	store := NewStore(100, nil)
	defer store.Close()

	timeout := time.After(5 * time.Second)
	done := make(chan bool)

	go func() {
		// Perform many operations that could deadlock
		var wg sync.WaitGroup

		// Add logs
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				store.Add("deadlock-test", "test", fmt.Sprintf("Message %d", i), false)
			}
		}()

		// Get logs
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = store.GetAll()
			}
		}()

		// Get errors
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = store.GetErrors()
			}
		}()

		// Get URLs
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = store.GetURLs()
			}
		}()

		// Clear logs
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				store.ClearLogs()
				time.Sleep(10 * time.Millisecond)
			}
		}()

		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-timeout:
		t.Fatal("Test timed out - possible deadlock")
	}
}
