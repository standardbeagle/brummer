package components

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/stretchr/testify/assert"
)

// TestSimpleLogStoreConcurrency tests basic log store concurrent operations
func TestSimpleLogStoreConcurrency(t *testing.T) {
	store := logs.NewStore(MaxLogEntries, nil)

	var wg sync.WaitGroup
	numWriters := DefaultConcurrentWriters * 2 // More writers for simple test
	logsPerWriter := DefaultLogsPerWriter * 2  // More logs for simple test

	// Concurrent writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerWriter; j++ {
				store.Add(
					fmt.Sprintf("proc-%d", id),
					fmt.Sprintf("Process %d", id),
					fmt.Sprintf("Log %d from writer %d", j, id),
					j%5 == 0, // Some errors
				)
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				all := store.GetAll()
				_ = len(all)
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Wait for async processing
	WaitForDebounce()

	// Verify results
	allLogs := store.GetAll()
	t.Logf("Total logs: %d (expected: %d)", len(allLogs), numWriters*logsPerWriter)
	assert.Equal(t, numWriters*logsPerWriter, len(allLogs), "Should have all logs")
}

// TestProcessNameCacheConcurrency tests concurrent cache operations
func TestProcessNameCacheConcurrency(t *testing.T) {
	cache := &sync.Map{}

	var wg sync.WaitGroup
	numWorkers := 50
	opsPerWorker := 100

	// Writers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				key := fmt.Sprintf("proc-%d-%d", id, j%10)
				value := fmt.Sprintf("Process %d", id)
				cache.Store(key, value)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				key := fmt.Sprintf("proc-%d-%d", id, j%10)
				if val, ok := cache.Load(key); ok {
					_ = val.(string)
				}
			}
		}(i)
	}

	// Iterators
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				count := 0
				cache.Range(func(key, value interface{}) bool {
					count++
					return count < 100 // Limit iteration
				})
				time.Sleep(time.Millisecond * 5)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache integrity
	finalCount := 0
	cache.Range(func(key, value interface{}) bool {
		finalCount++
		return true
	})
	t.Logf("Final cache entries: %d", finalCount)
	assert.Greater(t, finalCount, 0, "Should have cache entries")
}
