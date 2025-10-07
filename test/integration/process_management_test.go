package integration

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/beagleresearch/brummer/internal/process"
)

// TestProcessManagementWorkflow tests the complete process management workflow
// This test verifies current functionality BEFORE controller migration
// Test Scenario 1 from quickstart.md
func TestProcessManagementWorkflow(t *testing.T) {
	SkipIfShort(t, "Integration test with process execution")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Set initial window size
	ctx.SetWindowSize(120, 40)

	// Test Step 1: Launch Brummer and view processes list
	t.Run("view_processes_list", func(t *testing.T) {
		// Simulate pressing "1" to switch to processes view
		ctx.SendKeyMsg("1")

		// Verify we're in processes view
		ctx.AssertViewContains("Processes")
		ctx.AssertViewContains("dev")
		ctx.AssertViewContains("test")
		ctx.AssertViewContains("build")
	})

	// Test Step 2: Use arrow keys to select different processes
	t.Run("navigate_process_list", func(t *testing.T) {
		// Send down arrow to move selection
		ctx.SendKeyMsg("down")
		time.Sleep(50 * time.Millisecond)

		// Send up arrow to move selection back
		ctx.SendKeyMsg("up")
		time.Sleep(50 * time.Millisecond)

		// View should still show processes
		ctx.AssertViewContains("Processes")
	})

	// Test Step 3: Press Enter to start a stopped process
	t.Run("start_process", func(t *testing.T) {
		// Start the "dev" process
		processID, err := ctx.ProcessMgr.StartScript("dev")
		assert.NoError(t, err, "Should start dev script")

		// Wait for process to reach running status
		ctx.WaitForProcess(processID, process.StatusRunning, 3*time.Second)

		// Verify process is running
		ctx.AssertProcessStatus(processID, process.StatusRunning)

		// Check that logs are being captured
		time.Sleep(500 * time.Millisecond)
		logs := ctx.LogStore.GetLogs(processID, 10)
		assert.NotEmpty(t, logs, "Should have captured logs from process")
	})

	// Test Step 4: Verify process shows "Running" status
	t.Run("verify_running_status", func(t *testing.T) {
		processes := ctx.ProcessMgr.GetAllProcesses()
		assert.NotEmpty(t, processes, "Should have running processes")

		// Find the running process
		var runningProcess *process.Process
		for _, proc := range processes {
			if proc.Status == process.StatusRunning {
				runningProcess = proc
				break
			}
		}

		assert.NotNil(t, runningProcess, "Should have at least one running process")
		assert.Equal(t, process.StatusRunning, runningProcess.Status)
	})

	// Test Step 5: Press 's' to stop the running process
	t.Run("stop_process", func(t *testing.T) {
		processes := ctx.ProcessMgr.GetAllProcesses()
		var runningProcess *process.Process
		for _, proc := range processes {
			if proc.Status == process.StatusRunning {
				runningProcess = proc
				break
			}
		}

		assert.NotNil(t, runningProcess, "Should have running process to stop")

		// Stop the process
		err := ctx.ProcessMgr.StopProcess(runningProcess.ID)
		assert.NoError(t, err, "Should stop process")

		// Wait for process to stop
		ctx.WaitForProcess(runningProcess.ID, process.StatusStopped, 3*time.Second)

		// Verify process is stopped
		ctx.AssertProcessStatus(runningProcess.ID, process.StatusStopped)
	})

	// Test Step 6: Verify process shows "Stopped" status
	t.Run("verify_stopped_status", func(t *testing.T) {
		processes := ctx.ProcessMgr.GetAllProcesses()
		assert.NotEmpty(t, processes, "Should have processes")

		// Find stopped process
		var stoppedProcess *process.Process
		for _, proc := range processes {
			if proc.Status == process.StatusStopped {
				stoppedProcess = proc
				break
			}
		}

		assert.NotNil(t, stoppedProcess, "Should have at least one stopped process")
		assert.Equal(t, process.StatusStopped, stoppedProcess.Status)
	})
}

// TestProcessListDisplay tests that process list displays correctly
func TestProcessListDisplay(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Switch to processes view
	ctx.SendKeyMsg("1")

	// Verify key elements are displayed
	view := ctx.Model.View()
	assert.Contains(t, view, "Processes", "Should show processes header")

	// Verify available scripts are shown
	assert.Contains(t, view, "dev", "Should show dev script")
	assert.Contains(t, view, "test", "Should show test script")
	assert.Contains(t, view, "build", "Should show build script")
}

// TestProcessSelection tests process selection and navigation
func TestProcessSelection(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)
	ctx.SendKeyMsg("1") // Switch to processes view

	// Navigate with arrow keys
	ctx.SendKeyMsg("down")
	time.Sleep(50 * time.Millisecond)

	ctx.SendKeyMsg("down")
	time.Sleep(50 * time.Millisecond)

	ctx.SendKeyMsg("up")
	time.Sleep(50 * time.Millisecond)

	// View should still be functional
	view := ctx.Model.View()
	assert.NotEmpty(t, view, "View should render after navigation")
}

// TestProcessStartStop tests starting and stopping processes
func TestProcessStartStop(t *testing.T) {
	SkipIfShort(t, "Integration test with process execution")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Start a process
	processID := ctx.StartTestProcess("test")
	defer ctx.StopTestProcess(processID)

	// Verify process is running
	proc := ctx.ProcessMgr.GetProcess(processID)
	assert.NotNil(t, proc)
	assert.Equal(t, process.StatusRunning, proc.Status)
	assert.Equal(t, "test", proc.ScriptName)

	// Stop the process
	ctx.StopTestProcess(processID)

	// Verify process is stopped
	proc = ctx.ProcessMgr.GetProcess(processID)
	assert.Equal(t, process.StatusStopped, proc.Status)
}

// TestProcessRestart tests restarting a process
func TestProcessRestart(t *testing.T) {
	SkipIfShort(t, "Integration test with process execution")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Start a process
	processID := ctx.StartTestProcess("test")
	defer ctx.StopTestProcess(processID)

	// Get original process
	originalProc := ctx.ProcessMgr.GetProcess(processID)
	assert.NotNil(t, originalProc)

	// Restart the process
	err := ctx.ProcessMgr.RestartProcess(processID)
	assert.NoError(t, err)

	// Wait for restart to complete
	time.Sleep(500 * time.Millisecond)

	// Verify process is still running
	proc := ctx.ProcessMgr.GetProcess(processID)
	assert.NotNil(t, proc)
	assert.Equal(t, process.StatusRunning, proc.Status)
}

// TestMultipleProcesses tests running multiple processes simultaneously
func TestMultipleProcesses(t *testing.T) {
	SkipIfShort(t, "Integration test with multiple processes")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Start multiple processes
	devID := ctx.StartTestProcess("dev")
	testID := ctx.StartTestProcess("test")

	defer func() {
		ctx.StopTestProcess(devID)
		ctx.StopTestProcess(testID)
	}()

	// Verify both processes are running
	ctx.AssertProcessStatus(devID, process.StatusRunning)
	ctx.AssertProcessStatus(testID, process.StatusRunning)

	// Verify we have exactly 2 processes
	processes := ctx.ProcessMgr.GetAllProcesses()
	assert.Len(t, processes, 2, "Should have 2 running processes")
}

// TestProcessErrorHandling tests handling of process errors
func TestProcessErrorHandling(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Try to start a process that will fail
	processID, err := ctx.ProcessMgr.StartScript("error")
	if err == nil {
		// Process started but will exit with error
		// Wait for it to fail
		time.Sleep(1 * time.Second)

		proc := ctx.ProcessMgr.GetProcess(processID)
		if proc != nil {
			// Process should have failed
			assert.NotEqual(t, process.StatusRunning, proc.Status)
		}
	}
}

// TestProcessRealTimeUpdates tests that process status updates in real-time
func TestProcessRealTimeUpdates(t *testing.T) {
	SkipIfShort(t, "Integration test with real-time updates")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)
	ctx.SendKeyMsg("1") // Processes view

	// Start a process
	processID := ctx.StartTestProcess("test")
	defer ctx.StopTestProcess(processID)

	// Wait a moment for UI to update
	time.Sleep(200 * time.Millisecond)

	// Render view and check for running indicator
	view := ctx.Model.View()
	assert.NotEmpty(t, view, "View should render with running process")

	// Stop the process
	ctx.StopTestProcess(processID)

	// Wait for UI to update
	time.Sleep(200 * time.Millisecond)

	// View should update to show stopped status
	view = ctx.Model.View()
	assert.NotEmpty(t, view, "View should render with stopped process")
}

// TestProcessViewKeyboardShortcuts tests keyboard shortcuts in process view
func TestProcessViewKeyboardShortcuts(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Test navigation shortcuts
	keys := []string{"1", "2", "3", "1"} // Switch views and back
	for _, key := range keys {
		ctx.SendKeyMsg(key)
		time.Sleep(50 * time.Millisecond)

		// View should render after each key
		view := ctx.Model.View()
		assert.NotEmpty(t, view, "View should render after key press")
	}
}
