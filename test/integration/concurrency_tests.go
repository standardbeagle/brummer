package integration

import (
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/process"
)

// TestConcurrentProcessOperations tests concurrent process management
func TestConcurrentProcessOperations(t *testing.T) {
	SkipIfShort(t, "Integration test for concurrent process operations")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("concurrent_process_starts", func(t *testing.T) {
		// Start multiple processes concurrently
		var wg sync.WaitGroup
		errors := make(chan error, 3)
		processes := make(chan *process.Process, 3)

		scripts := []string{"dev", "test", "build"}

		for _, script := range scripts {
			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				proc, err := ctx.ProcessMgr.StartScript(s)
				if err != nil {
					errors <- err
				} else {
					processes <- proc
				}
			}(script)
		}

		wg.Wait()
		close(errors)
		close(processes)

		// Check for errors
		var errCount int
		for err := range errors {
			t.Logf("Concurrent start error: %v", err)
			errCount++
		}

		// At least some should succeed
		assert.Less(t, errCount, len(scripts), "Not all starts should fail")

		// Stop all processes
		for proc := range processes {
			ctx.ProcessMgr.StopProcess(proc.ID)
		}
	})

	t.Run("concurrent_process_stops", func(t *testing.T) {
		// Start processes
		proc1, _ := ctx.ProcessMgr.StartScript("dev")
		proc2, _ := ctx.ProcessMgr.StartScript("test")
		proc3, _ := ctx.ProcessMgr.StartScript("build")

		time.Sleep(500 * time.Millisecond)

		// Stop concurrently
		var wg sync.WaitGroup
		if proc1 != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.ProcessMgr.StopProcess(proc1.ID)
			}()
		}

		if proc2 != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.ProcessMgr.StopProcess(proc2.ID)
			}()
		}

		if proc3 != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.ProcessMgr.StopProcess(proc3.ID)
			}()
		}

		wg.Wait()
		time.Sleep(500 * time.Millisecond)

		// System should be stable
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent stops")
	})

	t.Run("concurrent_start_and_stop", func(t *testing.T) {
		// Concurrently start and stop different processes
		var wg sync.WaitGroup

		// Start processes
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				proc, err := ctx.ProcessMgr.StartScript("test")
				if err == nil && proc != nil {
					time.Sleep(100 * time.Millisecond)
					ctx.ProcessMgr.StopProcess(proc.ID)
				}
			}()
		}

		wg.Wait()
		time.Sleep(500 * time.Millisecond)

		// System should be stable
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent start/stop")
	})

	t.Run("concurrent_process_queries", func(t *testing.T) {
		// Start a process
		proc, err := ctx.ProcessMgr.StartScript("dev")
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Concurrently query process state
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.ProcessMgr.GetProcess(proc.ID)
				ctx.ProcessMgr.GetAllProcesses()
			}()
		}

		wg.Wait()

		// Should not race or crash
		ctx.ProcessMgr.StopProcess(proc.ID)
	})
}

// TestConcurrentViewOperations tests concurrent view updates
func TestConcurrentViewOperations(t *testing.T) {
	SkipIfShort(t, "Integration test for concurrent view operations")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("concurrent_view_switches", func(t *testing.T) {
		// Concurrently switch views
		var wg sync.WaitGroup

		views := []string{"1", "2", "3", "4", "5"}

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				viewKey := views[idx%len(views)]
				ctx.SendKeyMsg(viewKey)
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		// Should not crash
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent view switches")
	})

	t.Run("concurrent_view_renders", func(t *testing.T) {
		// Concurrently render views
		var wg sync.WaitGroup

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.GetCurrentView()
			}()
		}

		wg.Wait()

		// Should not race
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent renders")
	})

	t.Run("concurrent_resize_operations", func(t *testing.T) {
		// Concurrently resize window
		var wg sync.WaitGroup

		sizes := []struct{ w, h int }{
			{80, 24}, {120, 40}, {100, 30}, {160, 50},
		}

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				size := sizes[idx%len(sizes)]
				ctx.SetWindowSize(size.w, size.h)
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		// Should handle concurrent resizes
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent resizes")
	})
}

// TestConcurrentKeyInput tests concurrent keyboard input
func TestConcurrentKeyInput(t *testing.T) {
	SkipIfShort(t, "Integration test for concurrent key input")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("concurrent_key_presses", func(t *testing.T) {
		// Concurrently send key presses
		var wg sync.WaitGroup

		keys := []string{"1", "2", "3", "down", "up", "enter", "esc"}

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				key := keys[idx%len(keys)]
				ctx.SendKeyMsg(key)
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		// Should handle concurrent input
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent key input")
	})

	t.Run("concurrent_navigation", func(t *testing.T) {
		// Switch to processes view
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)

		// Concurrently navigate
		var wg sync.WaitGroup

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				if idx%2 == 0 {
					ctx.SendKeyMsg("down")
				} else {
					ctx.SendKeyMsg("up")
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond)

		// Should handle concurrent navigation
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent navigation")
	})
}

// TestConcurrentMessageHandling tests concurrent message processing
func TestConcurrentMessageHandling(t *testing.T) {
	SkipIfShort(t, "Integration test for concurrent message handling")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("concurrent_window_size_messages", func(t *testing.T) {
		// Concurrently send window size messages
		var wg sync.WaitGroup

		for i := 0; i < 30; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				msg := tea.WindowSizeMsg{
					Width:  80 + idx,
					Height: 24 + idx,
				}
				ctx.SendMessage(msg)
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		// Should handle concurrent messages
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent window size messages")
	})

	t.Run("concurrent_mixed_messages", func(t *testing.T) {
		// Concurrently send different message types
		var wg sync.WaitGroup

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				if idx%2 == 0 {
					ctx.SendMessage(tea.WindowSizeMsg{Width: 120, Height: 40})
				} else {
					ctx.SendKeyMsg("1")
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		// Should handle mixed concurrent messages
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle mixed concurrent messages")
	})
}

// TestConcurrentLogOperations tests concurrent log access
func TestConcurrentLogOperations(t *testing.T) {
	SkipIfShort(t, "Integration test for concurrent log operations")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Start a process to generate logs
	proc, err := ctx.ProcessMgr.StartScript("dev")
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	t.Run("concurrent_log_reads", func(t *testing.T) {
		// Concurrently read logs
		var wg sync.WaitGroup

		for i := 0; i < 30; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.LogStore.GetAll()
				ctx.LogStore.GetByProcess(proc.ID)
				ctx.LogStore.GetErrors()
			}()
		}

		wg.Wait()

		// Should not race
		logs := ctx.LogStore.GetAll()
		assert.NotNil(t, logs, "Should handle concurrent log reads")
	})

	t.Run("concurrent_view_and_log_access", func(t *testing.T) {
		// Concurrently access logs and render views
		var wg sync.WaitGroup

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				if idx%2 == 0 {
					ctx.LogStore.GetAll()
				} else {
					ctx.GetCurrentView()
				}
			}(i)
		}

		wg.Wait()

		// Should not race
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent view and log access")
	})

	ctx.ProcessMgr.StopProcess(proc.ID)
}

// TestRaceConditionDetection tests potential race conditions
// Note: Run with -race flag to detect actual races
func TestRaceConditionDetection(t *testing.T) {
	SkipIfShort(t, "Integration test for race condition detection")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("stress_test_all_operations", func(t *testing.T) {
		// Stress test with all operations concurrent
		var wg sync.WaitGroup

		// View switches
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 5; j++ {
					ctx.SendKeyMsg("1")
					ctx.SendKeyMsg("2")
					ctx.SendKeyMsg("3")
				}
			}()
		}

		// Process starts/stops
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				proc, err := ctx.ProcessMgr.StartScript("test")
				if err == nil && proc != nil {
					time.Sleep(200 * time.Millisecond)
					ctx.ProcessMgr.StopProcess(proc.ID)
				}
			}()
		}

		// Window resizes
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ctx.SetWindowSize(80+idx*5, 24+idx*2)
			}(i)
		}

		// View renders
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.GetCurrentView()
			}()
		}

		// Log reads
		for i := 0; i < 15; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.LogStore.GetAll()
			}()
		}

		wg.Wait()
		time.Sleep(500 * time.Millisecond)

		// System should still be functional
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle all concurrent operations")

		// Verify basic functionality still works
		proc, err := ctx.ProcessMgr.StartScript("dev")
		assert.NoError(t, err, "Should still start processes after stress test")
		if proc != nil {
			time.Sleep(200 * time.Millisecond)
			ctx.ProcessMgr.StopProcess(proc.ID)
		}
	})
}

// TestConcurrentStateModification tests state modification under concurrency
func TestConcurrentStateModification(t *testing.T) {
	SkipIfShort(t, "Integration test for concurrent state modification")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("concurrent_model_updates", func(t *testing.T) {
		// Concurrently update model state
		var wg sync.WaitGroup

		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				// Mix of different message types that modify state
				if idx%3 == 0 {
					ctx.SendKeyMsg("1")
				} else if idx%3 == 1 {
					ctx.SetWindowSize(120, 40)
				} else {
					ctx.SendKeyMsg("down")
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		// State should be consistent
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should maintain consistent state under concurrent updates")
	})

	t.Run("concurrent_process_state_queries", func(t *testing.T) {
		// Start a process
		proc, err := ctx.ProcessMgr.StartScript("dev")
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Concurrently query and modify process state
		var wg sync.WaitGroup

		// Readers
		for i := 0; i < 30; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx.ProcessMgr.GetProcess(proc.ID)
			}()
		}

		// Writer (stop process)
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond)
			ctx.ProcessMgr.StopProcess(proc.ID)
		}()

		wg.Wait()
		time.Sleep(500 * time.Millisecond)

		// State should be consistent
		finalProc, exists := ctx.ProcessMgr.GetProcess(proc.ID)
		assert.True(t, exists, "Process should still exist")
		assert.NotNil(t, finalProc, "Process should not be nil")
	})
}

// TestDeadlockPrevention tests for potential deadlocks
func TestDeadlockPrevention(t *testing.T) {
	SkipIfShort(t, "Integration test for deadlock prevention")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("no_deadlock_on_concurrent_operations", func(t *testing.T) {
		// Create a timeout to detect deadlocks
		done := make(chan bool, 1)

		go func() {
			var wg sync.WaitGroup

			// Perform various operations that could potentially deadlock
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					proc, _ := ctx.ProcessMgr.StartScript("test")
					if proc != nil {
						ctx.ProcessMgr.StopProcess(proc.ID)
					}
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx.GetCurrentView()
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx.SendKeyMsg("1")
				}()
			}

			wg.Wait()
			done <- true
		}()

		// Wait for completion or timeout
		select {
		case <-done:
			// Success - no deadlock
		case <-time.After(30 * time.Second):
			t.Fatal("Potential deadlock detected - operations did not complete within timeout")
		}
	})
}
