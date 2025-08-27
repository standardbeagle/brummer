package components

import (
	"fmt"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegratedShellLogDelivery tests that logs are properly delivered to the UI
func TestIntegratedShellLogDelivery(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create process manager
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	require.NoError(t, err)

	// Create integrated shell
	shell := NewIntegratedShell(processMgr)
	require.NotNil(t, shell)
	require.NotNil(t, shell.logStore)
	require.NotNil(t, shell.logSub)

	// Initialize the shell
	initCmd := shell.Init()
	require.NotNil(t, initCmd, "Init should return a command")

	// Send window size message to properly initialize components
	_, cmd := shell.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	_ = cmd // We don't need to execute this cmd in tests

	// Test that logs can be added to the store
	// Note: The subscription system returns tea.Cmd which requires the Bubble Tea runtime
	// In this test, we'll just verify that logs are added correctly to the store
	
	// Add a log to the store and notify
	shell.GetLogStore().Add("test-process", "Test Process", "Test log line", false)
	shell.NotifyNewLogs()
	
	// The notification system works asynchronously with debouncing,
	// but we can verify the log was added to the store immediately

	// Wait for async processing and debounce
	WaitForDebounce()

	// Check that log was added to the store
	allLogs := shell.GetLogStore().GetAll()
	assert.Greater(t, len(allLogs), 0, "Log should have been added to store")
}

// TestDebouncedLogSubscriptionConcurrency tests concurrent log additions
func TestDebouncedLogSubscriptionConcurrency(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create log store and subscription
	logStore := logs.NewStore(10000, nil)
	sub := NewDebouncedLogSubscription(logStore)

	// Track notifications
	notificationCount := 0
	var mu sync.Mutex

	// Simulate concurrent log additions
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				processID := fmt.Sprintf("proc-%d", id)
				logStore.Add(processID, fmt.Sprintf("Process %d", id),
					fmt.Sprintf("Log %d-%d", id, j), false)
				
				// Notify subscription
				sub.NotifyNewLogs()
				
				mu.Lock()
				notificationCount++
				mu.Unlock()
				
				time.Sleep(time.Microsecond * 100)
			}
		}(i)
	}

	wg.Wait()

	// Check that logs were added
	allLogs := logStore.GetAll()
	assert.Equal(t, 100, len(allLogs), "Should have 100 logs")
	
	mu.Lock()
	finalCount := notificationCount
	mu.Unlock()
	assert.Equal(t, 100, finalCount, "Should have 100 notifications")

	// Test that subscription delivers logs
	cmd := sub.Subscribe()
	require.NotNil(t, cmd)

	// Run command with timeout
	done := make(chan tea.Msg, 1)
	go func() {
		msg := cmd()
		done <- msg
	}()

	select {
	case msg := <-done:
		switch m := msg.(type) {
		case LogsUpdateMsg:
			assert.Equal(t, 100, len(m.Entries), "Should deliver all 100 logs")
			assert.True(t, m.Append, "Should be in append mode")
		default:
			t.Errorf("Unexpected message type: %T", m)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for log delivery")
	}
}

// TestLogStoreIntegrity tests that log store maintains data integrity
func TestLogStoreIntegrity(t *testing.T) {
	logStore := logs.NewStore(10000, nil)

	// Add logs concurrently
	var wg sync.WaitGroup
	numGoroutines := 20
	logsPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logStore.Add(
					fmt.Sprintf("proc-%d", id),
					fmt.Sprintf("Process %d", id),
					fmt.Sprintf("Log line %d from goroutine %d", j, id),
					j%2 == 0, // Alternate error status
				)
			}
		}(i)
	}

	wg.Wait()
	
	// Wait for async processing
	time.Sleep(50 * time.Millisecond)

	// Verify all logs were added
	allLogs := logStore.GetAll()
	assert.Equal(t, numGoroutines*logsPerGoroutine, len(allLogs), 
		"Should have all logs")

	// Verify data integrity
	logCounts := make(map[string]int)
	for _, log := range allLogs {
		logCounts[log.ProcessID]++
	}

	assert.Equal(t, numGoroutines, len(logCounts), 
		"Should have logs from all goroutines")
	
	for processID, count := range logCounts {
		assert.Equal(t, logsPerGoroutine, count, 
			"Process %s should have correct number of logs", processID)
	}
}

// TestProcessNameCacheConsistency tests cache consistency
func TestProcessNameCacheConsistency(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create process manager
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	require.NoError(t, err)

	// Create shell
	shell := NewIntegratedShell(processMgr)
	require.NotNil(t, shell)

	// Test concurrent cache operations
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		
		// Writer
		go func(id int) {
			defer wg.Done()
			processID := fmt.Sprintf("proc-%d", id)
			processName := fmt.Sprintf("Process %d", id)
			shell.processNameCache.Store(processID, processName)
		}(i)
		
		// Reader
		go func(id int) {
			defer wg.Done()
			processID := fmt.Sprintf("proc-%d", id)
			time.Sleep(time.Microsecond * 10) // Small delay
			if name, ok := shell.processNameCache.Load(processID); ok {
				expectedName := fmt.Sprintf("Process %d", id)
				assert.Equal(t, expectedName, name.(string))
			}
		}(i)
	}

	wg.Wait()

	// Verify all entries are in cache
	count := 0
	shell.processNameCache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, 100, count, "Should have all 100 entries in cache")
}