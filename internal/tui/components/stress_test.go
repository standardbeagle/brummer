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

// TestConcurrentLogStress tests the system under heavy concurrent load
func TestConcurrentLogStress(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create event bus and process manager
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	require.NoError(t, err)

	// Create integrated shell
	proxy := NewFakeProxyServer(nil)
	shell := NewIntegratedShell(processMgr, eventBus, proxy)
	require.NotNil(t, shell)

	// Initialize the shell
	shell.Init()
	shell.Update(tea.WindowSizeMsg{Width: DefaultTestWidth, Height: DefaultTestHeight})

	// Stress test parameters
	numWriters := DefaultConcurrentWriters // Number of concurrent log writers
	logsPerWriter := DefaultLogsPerWriter  // Logs each writer will generate
	numReaders := DefaultConcurrentReaders // Number of concurrent readers

	var wg sync.WaitGroup

	// Start log writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			processID := fmt.Sprintf("writer-%d", writerID)
			processName := fmt.Sprintf("Writer %d", writerID)

			for j := 0; j < logsPerWriter; j++ {
				// Add log to store
				shell.GetLogStore().Add(processID, processName,
					fmt.Sprintf("Log %d from writer %d", j, writerID),
					j%10 == 0) // Every 10th log is an error

				// Update cache
				shell.GetProcessNameCacheSync().Store(processID, processName)

				// Notify subscription
				shell.NotifyNewLogs()

				// Small random delay to simulate real behavior
				if j%10 == 0 {
					time.Sleep(time.Microsecond * time.Duration(j%100))
				}
			}
		}(i)
	}

	// Start log readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				// Read logs
				allLogs := shell.GetLogStore().GetAll()
				_ = len(allLogs)

				// Read from cache
				shell.GetProcessNameCacheSync().Range(func(key, value interface{}) bool {
					_ = value.(string)
					return j < 50 // Stop early sometimes
				})

				// Note: We don't call applyFilter() here because it's not thread-safe
				// In production, Bubble Tea ensures single-threaded updates

				time.Sleep(time.Millisecond * 5)
			}
		}(i)
	}

	// Note: We don't test subscription consumers here because
	// the Subscribe() method blocks waiting for notifications,
	// which would require complex coordination in a stress test

	// Wait for all goroutines to complete
	wg.Wait()

	// Give async processing time to settle
	WaitForDebounce()

	// Verify data integrity
	allLogs := shell.GetLogStore().GetAll()
	t.Logf("Total logs after stress test: %d", len(allLogs))
	assert.Greater(t, len(allLogs), 0, "Should have logs after stress test")

	// Check cache consistency
	cacheCount := 0
	shell.GetProcessNameCacheSync().Range(func(key, value interface{}) bool {
		cacheCount++
		return true
	})
	t.Logf("Process cache entries: %d", cacheCount)
	assert.Equal(t, numWriters, cacheCount, "Cache should have all writers")
}

// TestComponentSequentialUpdates tests that components handle rapid sequential updates correctly
// Note: Components are NOT thread-safe and should only be used from the main Bubble Tea update loop
func TestComponentSequentialUpdates(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create components
	logStore := logs.NewStore(MaxLogEntries, nil)
	processView := NewProcessViewNew(nil)
	logsView := NewLogsViewComponent(logStore)

	// Test rapid sequential updates (simulating fast user input)
	for i := 0; i < 100; i++ {
		// Various update messages
		switch i % 5 {
		case 0:
			processView.Update(tea.WindowSizeMsg{Width: 100 + i%20, Height: 30})
			logsView.Update(tea.WindowSizeMsg{Width: 100 + i%20, Height: 30})
		case 1:
			processView.Update(tea.KeyMsg{Type: tea.KeyUp})
			logsView.Update(tea.KeyMsg{Type: tea.KeyDown})
		case 2:
			processView.Focus()
			logsView.Blur()
		case 3:
			// View() calls are safe when sequential
			_ = processView.View()
			_ = logsView.View()
		case 4:
			processView.SetSize(DefaultTestWidth, DefaultTestHeight)
			logsView.SetSize(DefaultTestWidth, DefaultTestHeight)
		}
	}

	// Verify components are still functional
	output := processView.View()
	assert.NotEmpty(t, output, "Process view should render after sequential updates")

	output = logsView.View()
	assert.NotEmpty(t, output, "Logs view should render after sequential updates")
}

// TestLogStoreThreadSafety specifically tests log store concurrent operations
func TestLogStoreThreadSafety(t *testing.T) {
	store := logs.NewStore(MaxLogEntries, nil)

	var wg sync.WaitGroup
	numOps := 1000

	// Concurrent adds
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOps; i++ {
			store.Add(fmt.Sprintf("p%d", i%100), fmt.Sprintf("Process %d", i%100),
				fmt.Sprintf("Log %d", i), i%5 == 0)
		}
	}()

	// Concurrent reads
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOps; i++ {
			all := store.GetAll()
			_ = len(all)

			byProc := store.GetByProcess(fmt.Sprintf("p%d", i%100))
			_ = len(byProc)

			collapsed := store.GetAllCollapsed()
			_ = len(collapsed)
		}
	}()

	// Concurrent filters
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOps/10; i++ {
			filtered := store.GetErrors()
			_ = len(filtered)

			// Get by process collapsed
			procCollapsed := store.GetByProcessCollapsed(fmt.Sprintf("p%d", i%100))
			_ = len(procCollapsed)
		}
	}()

	// Concurrent error context operations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOps/100; i++ {
			contexts := store.GetErrorContexts()
			_ = len(contexts)
			time.Sleep(time.Millisecond * 10)
		}
	}()

	wg.Wait()

	// Store should still be functional
	store.Add("final", "Final", "Final log", false)
	WaitForAsync() // Wait for async processing

	final := store.GetAll()
	assert.NotNil(t, final, "Store should still be functional")
}
