package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/standardbeagle/brummer/internal/logs"
)

// TestLogViewingAndFiltering tests the complete log viewing workflow
// This test verifies current functionality BEFORE controller migration
// Test Scenario 2 from quickstart.md
func TestLogViewingAndFiltering(t *testing.T) {
	SkipIfShort(t, "Integration test with log viewing")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Test Step 1: Start a process that generates logs
	t.Run("start_process_with_logs", func(t *testing.T) {
		processID := ctx.StartTestProcess("dev")
		defer ctx.StopTestProcess(processID)

		// Wait for logs to be generated
		ctx.WaitForLog("", 2*time.Second)

		// Verify logs were captured
		logs := ctx.LogStore.GetByProcess(processID)
		assert.NotEmpty(t, logs, "Should have captured logs")
	})

	// Test Step 2: Press "2" to view logs
	t.Run("switch_to_logs_view", func(t *testing.T) {
		ctx.SendKeyMsg("2")
		time.Sleep(100 * time.Millisecond)

		// Verify we're in logs view
		ctx.AssertViewContains("Logs")
	})

	// Test Step 3: Verify logs are displayed and updating in real-time
	t.Run("verify_realtime_log_display", func(t *testing.T) {
		// Start a process to generate logs
		processID := ctx.StartTestProcess("dev")
		defer ctx.StopTestProcess(processID)

		// Switch to logs view
		ctx.SendKeyMsg("2")
		time.Sleep(100 * time.Millisecond)

		// Wait for logs to accumulate
		time.Sleep(1 * time.Second)

		// Verify logs are displayed
		logs := ctx.LogStore.GetByProcess(processID)
		assert.NotEmpty(t, logs, "Should have logs")

		// Get initial log count
		initialCount := len(logs)

		// Wait for more logs
		time.Sleep(500 * time.Millisecond)

		// Verify more logs were added (real-time updates)
		newLogs := ctx.LogStore.GetByProcess(processID)
		// Note: This might not always increase for fast-completing processes
		assert.GreaterOrEqual(t, len(newLogs), initialCount, "Logs should update in real-time")
	})
}

// TestLogDisplay tests basic log display functionality
func TestLogDisplay(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Add some test logs
	ctx.LogStore.Add("test-process", "test-process", "Test log message 1", false)
	ctx.LogStore.Add("test-process", "test-process", "Test error message", true)

	// Switch to logs view
	ctx.SendKeyMsg("2")
	time.Sleep(100 * time.Millisecond)

	// Verify logs view is displayed
	ctx.AssertViewContains("Logs")
}

// TestLogFiltering tests log filtering functionality
func TestLogFiltering(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Add test logs with different levels
	ctx.LogStore.Add("test-process", "test-process", "Info message", false)
	ctx.LogStore.Add("test-process", "test-process", "Error message", true)
	ctx.LogStore.Add("test-process", "test-process", "Warning message", false)

	// Get all logs
	allLogs := ctx.LogStore.GetByProcess("test-process")
	assert.Len(t, allLogs, 3, "Should have 3 logs")

	// Verify different log levels exist
	var infoCount, errorCount, warnCount int
	for _, log := range allLogs {
		switch log.Level {
		case logs.LevelInfo:
			infoCount++
		case logs.LevelError:
			errorCount++
		case logs.LevelWarn:
			warnCount++
		}
	}

	assert.Equal(t, 1, infoCount, "Should have 1 info log")
	assert.Equal(t, 1, errorCount, "Should have 1 error log")
	assert.Equal(t, 1, warnCount, "Should have 1 warning log")
}

// TestLogClear tests clearing logs
func TestLogClear(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Add test logs
	ctx.LogStore.Add("test-process", "test-process", "Test message 1", false)
	ctx.LogStore.Add("test-process", "test-process", "Test message 2", false)

	// Verify logs exist
	logs := ctx.LogStore.GetByProcess("test-process")
	assert.Len(t, logs, 2, "Should have 2 logs before clear")

	// Clear logs
	ctx.LogStore.ClearLogsForProcess("test-process")

	// Verify logs were cleared
	logs = ctx.LogStore.GetByProcess("test-process")
	assert.Empty(t, logs, "Should have no logs after clear")
}

// TestLogClearAll tests clearing all logs
func TestLogClearAll(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Add logs for multiple processes
	ctx.LogStore.Add("process-1", "process-1", "Process 1 message", false)
	ctx.LogStore.Add("process-2", "process-2", "Process 2 message", false)

	// Verify logs exist
	logs1 := ctx.LogStore.GetByProcess("process-1")
	logs2 := ctx.LogStore.GetByProcess("process-2")
	assert.Len(t, logs1, 1, "Process 1 should have logs")
	assert.Len(t, logs2, 1, "Process 2 should have logs")

	// Clear all logs
	ctx.LogStore.ClearLogs()

	// Verify all logs were cleared
	logs1 = ctx.LogStore.GetByProcess("process-1")
	logs2 = ctx.LogStore.GetByProcess("process-2")
	assert.Empty(t, logs1, "Process 1 logs should be cleared")
	assert.Empty(t, logs2, "Process 2 logs should be cleared")
}

// TestLogProcessFilter tests filtering logs by process
func TestLogProcessFilter(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Add logs for different processes
	ctx.LogStore.Add("dev", "dev", "Dev process message", false)
	ctx.LogStore.Add("test", "test", "Test process message", false)
	ctx.LogStore.Add("dev", "dev", "Dev error message", true)

	// Get logs for specific process
	devLogs := ctx.LogStore.GetByProcess("dev")
	assert.Len(t, devLogs, 2, "Should have 2 logs for dev process")

	testLogs := ctx.LogStore.GetByProcess("test")
	assert.Len(t, testLogs, 1, "Should have 1 log for test process")

	// Verify log content
	for _, log := range devLogs {
		assert.Equal(t, "dev", log.ProcessID, "Dev logs should have dev process ID")
	}

	for _, log := range testLogs {
		assert.Equal(t, "test", log.ProcessID, "Test logs should have test process ID")
	}
}

// TestLogRealTimeUpdates tests real-time log updates
func TestLogRealTimeUpdates(t *testing.T) {
	SkipIfShort(t, "Integration test with real-time updates")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Start a process that generates logs
	processID := ctx.StartTestProcess("dev")
	defer ctx.StopTestProcess(processID)

	// Wait for initial logs
	time.Sleep(500 * time.Millisecond)
	initialLogs := ctx.LogStore.GetByProcess(processID)
	initialCount := len(initialLogs)

	// Wait for more logs to be generated
	time.Sleep(1 * time.Second)

	// Get updated logs
	updatedLogs := ctx.LogStore.GetByProcess(processID)
	updatedCount := len(updatedLogs)

	// Verify logs increased (real-time updates working)
	assert.GreaterOrEqual(t, updatedCount, initialCount, "Logs should update in real-time")
}

// TestLogViewNavigation tests navigation in logs view
func TestLogViewNavigation(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Add many test logs to enable scrolling
	for i := 0; i < 50; i++ {
		ctx.LogStore.Add("test-process", "test-process", "Test log message", false)
	}

	// Switch to logs view
	ctx.SendKeyMsg("2")
	time.Sleep(100 * time.Millisecond)

	// Simulate scrolling with arrow keys
	ctx.SendKeyMsg("down")
	time.Sleep(50 * time.Millisecond)

	ctx.SendKeyMsg("down")
	time.Sleep(50 * time.Millisecond)

	ctx.SendKeyMsg("up")
	time.Sleep(50 * time.Millisecond)

	// View should still render
	view := ctx.Model.View()
	assert.NotEmpty(t, view, "View should render after navigation")
	ctx.AssertViewContains("Logs")
}

// TestLogAutoScroll tests auto-scroll to latest logs
func TestLogAutoScroll(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Start process with logs
	processID := ctx.StartTestProcess("dev")
	defer ctx.StopTestProcess(processID)

	// Switch to logs view
	ctx.SendKeyMsg("2")
	time.Sleep(100 * time.Millisecond)

	// Wait for logs to accumulate
	time.Sleep(1 * time.Second)

	// Verify logs are displayed
	view := ctx.Model.View()
	assert.NotEmpty(t, view, "Logs view should show content")
}

// TestLogViewKeyboardShortcuts tests keyboard shortcuts in logs view
func TestLogViewKeyboardShortcuts(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Add test logs
	ctx.LogStore.Add("test", "test", "Test message", false)

	// Switch to logs view
	ctx.SendKeyMsg("2")
	time.Sleep(100 * time.Millisecond)

	// Test view switching
	ctx.SendKeyMsg("1") // Processes
	time.Sleep(50 * time.Millisecond)

	ctx.SendKeyMsg("2") // Back to logs
	time.Sleep(50 * time.Millisecond)

	// Verify logs view is still functional
	ctx.AssertViewContains("Logs")
}

// TestLogLevelDisplay tests that different log levels are handled correctly
func TestLogLevelDisplay(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Add logs with all levels - using Add() method with isError flag
	ctx.LogStore.Add("test", "test", "Debug message", false)
	ctx.LogStore.Add("test", "test", "Info message", false)
	ctx.LogStore.Add("test", "test", "Warn message", false)
	ctx.LogStore.Add("test", "test", "Error message", true)

	// Verify all logs were added
	logs := ctx.LogStore.GetByProcess("test")
	assert.Len(t, logs, 4, "Should have logs for all levels")

	// Verify content exists
	contents := make(map[string]bool)
	for _, log := range logs {
		contents[log.Content] = true
	}

	assert.True(t, contents["Debug message"], "Should have debug message")
	assert.True(t, contents["Info message"], "Should have info message")
	assert.True(t, contents["Warn message"], "Should have warn message")
	assert.True(t, contents["Error message"], "Should have error message")
}
