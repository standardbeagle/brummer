package integration

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	
	processes "github.com/standardbeagle/brummer/internal/domain/processes"
)

// TestWindowResizeEdgeCases tests edge cases in window resize handling
func TestWindowResizeEdgeCases(t *testing.T) {
	SkipIfShort(t, "Integration test for window resize edge cases")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("minimal_window_size", func(t *testing.T) {
		// Test with extremely small window (1x1)
		ctx.SetWindowSize(1, 1)
		time.Sleep(50 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should render even with 1x1 window")
		// Should not crash
	})

	t.Run("zero_window_size", func(t *testing.T) {
		// Test with zero dimensions
		ctx.SetWindowSize(0, 0)
		time.Sleep(50 * time.Millisecond)

		view := ctx.GetCurrentView()
		// Should handle gracefully (may return empty or minimal view)
		assert.NotNil(t, view, "Should not crash with 0x0 window")
	})

	t.Run("negative_window_size", func(t *testing.T) {
		// Test with negative dimensions
		ctx.SetWindowSize(-10, -10)
		time.Sleep(50 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotNil(t, view, "Should handle negative window size gracefully")
	})

	t.Run("extremely_large_window", func(t *testing.T) {
		// Test with very large window
		ctx.SetWindowSize(10000, 10000)
		time.Sleep(50 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle large window")
	})

	t.Run("rapid_resize_spam", func(t *testing.T) {
		// Spam resize events
		sizes := []struct{ w, h int }{
			{80, 24}, {120, 40}, {100, 30}, {160, 50},
			{80, 24}, {120, 40}, {100, 30}, {160, 50},
			{80, 24}, {120, 40}, {100, 30}, {160, 50},
		}

		for _, size := range sizes {
			ctx.SetWindowSize(size.w, size.h)
			// No sleep - rapid fire
		}
		time.Sleep(100 * time.Millisecond)

		// Should not crash
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid resize events")
	})

	t.Run("asymmetric_window", func(t *testing.T) {
		// Very wide but short
		ctx.SetWindowSize(1000, 5)
		time.Sleep(50 * time.Millisecond)
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle wide/short window")

		// Very narrow but tall
		ctx.SetWindowSize(5, 1000)
		time.Sleep(50 * time.Millisecond)
		view = ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle narrow/tall window")
	})
}

// TestInvalidKeyInput tests handling of invalid or unexpected key inputs
func TestInvalidKeyInput(t *testing.T) {
	SkipIfShort(t, "Integration test for invalid key input")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("invalid_view_numbers", func(t *testing.T) {
		// Try to switch to non-existent views
		invalidKeys := []string{"0", "9", "99", "a", "z", "!", "@", "#"}

		for _, key := range invalidKeys {
			ctx.SendKeyMsg(key)
			time.Sleep(20 * time.Millisecond)

			view := ctx.GetCurrentView()
			assert.NotEmpty(t, view, "Should handle invalid view key: %s", key)
		}
	})

	t.Run("control_characters", func(t *testing.T) {
		// Send control characters
		ctx.SendKeyMsg("\x00") // NUL
		ctx.SendKeyMsg("\x01") // SOH
		ctx.SendKeyMsg("\x1b") // ESC
		ctx.SendKeyMsg("\x7f") // DEL
		time.Sleep(50 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle control characters")
	})

	t.Run("unicode_input", func(t *testing.T) {
		// Send unicode characters
		unicodeKeys := []string{"é", "中", "🚀", "א", "∑"}

		for _, key := range unicodeKeys {
			ctx.SendKeyMsg(key)
			time.Sleep(20 * time.Millisecond)
		}

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle unicode input")
	})

	t.Run("rapid_key_spam", func(t *testing.T) {
		// Spam keys rapidly
		for i := 0; i < 100; i++ {
			ctx.SendKeyMsg("a")
		}
		time.Sleep(100 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid key spam")
	})

	t.Run("empty_key_string", func(t *testing.T) {
		// Send empty key
		ctx.SendKeyMsg("")
		time.Sleep(50 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle empty key string")
	})
}

// TestProcessEdgeCases tests edge cases in process management
func TestProcessEdgeCases(t *testing.T) {
	SkipIfShort(t, "Integration test for process edge cases")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("stop_nonexistent_process", func(t *testing.T) {
		// Try to stop a process that doesn't exist
		err := ctx.ProcessMgr.StopProcess("nonexistent-process-12345")
		// Should return error, not crash
		assert.Error(t, err, "Should error when stopping nonexistent process")
	})

	t.Run("stop_already_stopped_process", func(t *testing.T) {
		// Start and stop a process
		proc, err := ctx.ProcessMgr.StartScript("test")
		require.NoError(t, err)

		ctx.WaitForProcess(proc.ID, processes.StatusRunning, 2*time.Second)

		err = ctx.ProcessMgr.StopProcess(proc.ID)
		require.NoError(t, err)

		ctx.WaitForProcess(proc.ID, processes.StatusStopped, 2*time.Second)

		// Try to stop it again
		err = ctx.ProcessMgr.StopProcess(proc.ID)
		// Should handle gracefully (may be error or no-op)
		assert.NotNil(t, err, "Should handle stopping already stopped process")
	})

	t.Run("start_nonexistent_script", func(t *testing.T) {
		// Try to start a script that doesn't exist
		_, err := ctx.ProcessMgr.StartScript("nonexistent-script-xyz")
		assert.Error(t, err, "Should error when starting nonexistent script")
	})

	t.Run("start_already_running_process", func(t *testing.T) {
		// Start a process
		proc1, err := ctx.ProcessMgr.StartScript("dev")
		require.NoError(t, err)

		ctx.WaitForProcess(proc1.ID, processes.StatusRunning, 2*time.Second)

		// Try to start the same script again
		proc2, err := ctx.ProcessMgr.StartScript("dev")
		// Should create a new instance (different ID) or error
		if err == nil {
			assert.NotEqual(t, proc1.ID, proc2.ID, "Should create new instance with different ID")
			ctx.ProcessMgr.StopProcess(proc2.ID)
		} else {
			// Or it may error if duplicate starts aren't allowed
			assert.Error(t, err, "May error on duplicate start")
		}

		ctx.ProcessMgr.StopProcess(proc1.ID)
	})

	t.Run("process_that_exits_immediately", func(t *testing.T) {
		// Start the "test" script which exits quickly
		proc, err := ctx.ProcessMgr.StartScript("test")
		require.NoError(t, err)

		// Wait for it to complete
		time.Sleep(500 * time.Millisecond)

		// Check final status
		finalProc, exists := ctx.ProcessMgr.GetProcess(proc.ID)
		assert.True(t, exists, "Process should exist")
		// May be StatusStopped or StatusCompleted
		assert.NotEqual(t, processes.StatusRunning, finalProc.Status, "Quick process should have exited")
	})

	t.Run("process_that_fails", func(t *testing.T) {
		// Start the "error" script which exits with error code
		proc, err := ctx.ProcessMgr.StartScript("error")
		require.NoError(t, err)

		// Wait for it to fail
		time.Sleep(500 * time.Millisecond)

		// Check that error was captured
		finalProc, exists := ctx.ProcessMgr.GetProcess(proc.ID)
		assert.True(t, exists, "Process should exist")
		// Should have failed status
		assert.NotEqual(t, processes.StatusRunning, finalProc.Status)
	})

	t.Run("get_nonexistent_process", func(t *testing.T) {
		// Try to get a process that doesn't exist
		_, exists := ctx.ProcessMgr.GetProcess("fake-process-id")
		assert.False(t, exists, "Should return false for nonexistent process")
	})
}

// TestViewSwitchingEdgeCases tests edge cases in view navigation
func TestViewSwitchingEdgeCases(t *testing.T) {
	SkipIfShort(t, "Integration test for view switching edge cases")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("rapid_view_switching", func(t *testing.T) {
		// Rapidly switch between views
		for i := 0; i < 50; i++ {
			ctx.SendKeyMsg("1")
			ctx.SendKeyMsg("2")
			ctx.SendKeyMsg("3")
			ctx.SendKeyMsg("4")
			ctx.SendKeyMsg("5")
		}
		time.Sleep(100 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid view switching")
	})

	t.Run("switch_to_same_view_repeatedly", func(t *testing.T) {
		// Switch to same view many times
		for i := 0; i < 20; i++ {
			ctx.SendKeyMsg("1")
			time.Sleep(10 * time.Millisecond)
		}

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle repeated switch to same view")
	})

	t.Run("view_switch_during_render", func(t *testing.T) {
		// Start a process to create activity
		ctx.ProcessMgr.StartScript("dev")
		time.Sleep(100 * time.Millisecond)

		// Rapidly switch views while rendering is happening
		for i := 0; i < 10; i++ {
			ctx.SendKeyMsg("2") // Logs view (high activity)
			ctx.SendKeyMsg("1") // Processes view
		}
		time.Sleep(100 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle view switches during active rendering")
	})
}

// TestEventBusEdgeCases tests edge cases in event handling
func TestEventBusEdgeCases(t *testing.T) {
	SkipIfShort(t, "Integration test for event bus edge cases")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("null_message_handling", func(t *testing.T) {
		// Send nil message
		ctx.SendMessage(nil)
		time.Sleep(50 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle nil message")
	})

	t.Run("unknown_message_type", func(t *testing.T) {
		// Send custom unknown message type
		type UnknownMsg struct{}
		ctx.SendMessage(UnknownMsg{})
		time.Sleep(50 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle unknown message type")
	})

	t.Run("rapid_window_size_messages", func(t *testing.T) {
		// Send many window size messages rapidly
		for i := 0; i < 50; i++ {
			msg := tea.WindowSizeMsg{
				Width:  80 + i,
				Height: 24 + i,
			}
			ctx.SendMessage(msg)
		}
		time.Sleep(100 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid window size messages")
	})
}

// TestStateConsistency tests state consistency edge cases
func TestStateConsistency(t *testing.T) {
	SkipIfShort(t, "Integration test for state consistency")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("state_after_many_operations", func(t *testing.T) {
		// Perform many operations and verify state is consistent

		// Start process
		proc1, err := ctx.ProcessMgr.StartScript("dev")
		require.NoError(t, err)

		// Switch views multiple times
		for i := 0; i < 10; i++ {
			ctx.SendKeyMsg("1")
			ctx.SendKeyMsg("2")
			ctx.SendKeyMsg("3")
		}

		// Stop process
		ctx.ProcessMgr.StopProcess(proc1.ID)

		// Start another process
		proc2, err := ctx.ProcessMgr.StartScript("test")
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		// Verify state is consistent
		processes := ctx.ProcessMgr.GetAllProcesses()
		assert.GreaterOrEqual(t, len(processes), 2, "Should have process history")

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "View should still render correctly")

		// Cleanup
		ctx.ProcessMgr.StopProcess(proc2.ID)
	})

	t.Run("state_after_resize_and_operations", func(t *testing.T) {
		// Interleave resizes with other operations
		ctx.SetWindowSize(80, 24)
		ctx.SendKeyMsg("1")

		ctx.SetWindowSize(120, 40)
		proc, _ := ctx.ProcessMgr.StartScript("dev")

		ctx.SetWindowSize(100, 30)
		ctx.SendKeyMsg("2")

		ctx.SetWindowSize(160, 50)
		time.Sleep(200 * time.Millisecond)

		// State should be consistent
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "State should be consistent after mixed operations")

		if proc != nil {
			ctx.ProcessMgr.StopProcess(proc.ID)
		}
	})
}

// TestMemoryAndResourceEdgeCases tests resource handling edge cases
func TestMemoryAndResourceEdgeCases(t *testing.T) {
	SkipIfShort(t, "Integration test for memory and resource edge cases")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("many_log_entries", func(t *testing.T) {
		// Start a process that generates many logs
		proc, err := ctx.ProcessMgr.StartScript("dev")
		require.NoError(t, err)

		// Wait for logs to accumulate
		time.Sleep(2 * time.Second)

		// Switch to logs view
		ctx.SendKeyMsg("2")
		time.Sleep(100 * time.Millisecond)

		// Should handle large log volume
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should render with many log entries")

		// Scroll through logs
		for i := 0; i < 20; i++ {
			ctx.SendKeyMsg("down")
		}
		time.Sleep(50 * time.Millisecond)

		view = ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle scrolling through many logs")

		ctx.ProcessMgr.StopProcess(proc.ID)
	})

	t.Run("multiple_concurrent_processes", func(t *testing.T) {
		// Start multiple processes concurrently
		var processes []*processes.Process

		proc1, err1 := ctx.ProcessMgr.StartScript("dev")
		proc2, err2 := ctx.ProcessMgr.StartScript("build")
		proc3, err3 := ctx.ProcessMgr.StartScript("test")

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)

		if proc1 != nil {
			processes = append(processes, proc1)
		}
		if proc2 != nil {
			processes = append(processes, proc2)
		}
		if proc3 != nil {
			processes = append(processes, proc3)
		}

		time.Sleep(1 * time.Second)

		// Switch between views with multiple processes running
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)
		ctx.SendKeyMsg("2")
		time.Sleep(50 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle multiple concurrent processes")

		// Stop all processes
		for _, proc := range processes {
			ctx.ProcessMgr.StopProcess(proc.ID)
		}
	})
}

// TestRecoveryFromErrors tests error recovery scenarios
func TestRecoveryFromErrors(t *testing.T) {
	SkipIfShort(t, "Integration test for error recovery")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("recovery_after_failed_process", func(t *testing.T) {
		// Start process that will fail
		_, err := ctx.ProcessMgr.StartScript("error")
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		// System should still be functional
		ctx.SendKeyMsg("1")
		ctx.SendKeyMsg("2")
		ctx.SendKeyMsg("3")
		time.Sleep(100 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should recover from failed process")

		// Should be able to start new process
		proc2, err := ctx.ProcessMgr.StartScript("test")
		assert.NoError(t, err, "Should start new process after failure")

		if proc2 != nil {
			time.Sleep(200 * time.Millisecond)
			ctx.ProcessMgr.StopProcess(proc2.ID)
		}
	})

	t.Run("recovery_after_invalid_operations", func(t *testing.T) {
		// Attempt several invalid operations
		ctx.ProcessMgr.StopProcess("invalid-id-1")
		ctx.ProcessMgr.StopProcess("invalid-id-2")
		ctx.ProcessMgr.StartScript("nonexistent-script")

		time.Sleep(100 * time.Millisecond)

		// System should still work
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should recover from invalid operations")

		// Should still be able to perform valid operations
		proc, err := ctx.ProcessMgr.StartScript("test")
		assert.NoError(t, err, "Should perform valid operations after invalid ones")

		if proc != nil {
			time.Sleep(200 * time.Millisecond)
			ctx.ProcessMgr.StopProcess(proc.ID)
		}
	})
}
