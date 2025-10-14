package tui

import (
	"os"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestartProcessIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a package.json with a test script that fails on restart
	packageJSON := `{
		"name": "test-app",
		"scripts": {
			"server": "echo 'Server starting' && sleep 1 && echo 'Server running' && sleep 10"
		}
	}`

	err := writeFile(tempDir+"/package.json", packageJSON)
	require.NoError(t, err)

	// Create process manager
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(tempDir, eventBus, true) // true = has package.json
	require.NoError(t, err)

	// Start the initial process
	proc, err := processMgr.StartScript("server")
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Wait for process to be running
	time.Sleep(100 * time.Millisecond)

	// Verify initial process is running
	allProcs := processMgr.GetAllProcesses()
	require.Len(t, allProcs, 1)
	assert.Equal(t, process.StatusRunning, allProcs[0].Status)
	initialPID := allProcs[0].ID

	t.Logf("Initial process: %s (PID: %s, Status: %s)", allProcs[0].Name, allProcs[0].ID, allProcs[0].Status)

	// Now simulate the restart operation that happens when 'r' is pressed
	// This mimics handleRestartProcess function
	timeout := 5 * time.Second

	// Step 1: Stop the process and wait
	err = processMgr.StopProcessAndWait(proc.ID, timeout)
	require.NoError(t, err, "StopProcessAndWait should succeed")

	// Step 2: Start the script again
	newProc, err := processMgr.StartScript("server")
	if err != nil {
		t.Logf("StartScript failed: %v", err)
	}

	// Wait a moment for processes to settle
	time.Sleep(200 * time.Millisecond)

	// Check the final state
	finalProcs := processMgr.GetAllProcesses()

	t.Logf("Final process count: %d", len(finalProcs))
	for i, p := range finalProcs {
		t.Logf("Process %d: %s (PID: %s, Status: %s)", i+1, p.Name, p.ID, p.Status)
	}

	// The bug would manifest as:
	// 1. Original process showing as failed
	// 2. New process also showing as failed
	// 3. Total of 2 processes instead of 1

	// What we expect:
	// - Either 1 successful process (old one cleaned up)
	// - Or 1 failed + 1 successful (if cleanup doesn't happen)
	//
	// What the bug shows:
	// - 2 failed processes

	// Count processes by status
	var runningCount, failedCount, stoppedCount, successCount int
	for _, p := range finalProcs {
		switch p.Status {
		case process.StatusRunning:
			runningCount++
		case process.StatusFailed:
			failedCount++
		case process.StatusStopped:
			stoppedCount++
		case process.StatusSuccess:
			successCount++
		}
	}

	t.Logf("Status counts - Running: %d, Failed: %d, Stopped: %d, Success: %d",
		runningCount, failedCount, stoppedCount, successCount)

	// The restart should result in exactly one process being in a good state
	// Either the new process should be running, or if there was an error,
	// we should have a clear reason why
	if err != nil {
		// If StartScript failed, we should have 1 stopped/failed process (the original)
		assert.LessOrEqual(t, len(finalProcs), 2, "Should not have more than 2 processes after failed restart")
		// The original process should not be running
		for _, p := range finalProcs {
			if p.ID == initialPID {
				assert.NotEqual(t, process.StatusRunning, p.Status, "Original process should not be running after restart")
			}
		}
	} else {
		// If StartScript succeeded, we should have a running process
		require.NotNil(t, newProc, "New process should be created")
		assert.Equal(t, process.StatusRunning, newProc.Status, "New process should be running")

		// The key assertion: we should not have 2 failed processes
		assert.Less(t, failedCount, 2, "Should not have 2 failed processes after restart")

		// Ideally, we should have exactly 1 running process
		assert.Equal(t, 1, runningCount, "Should have exactly 1 running process after successful restart")
	}
}

func writeFile(path, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

func TestRestartProcessWithPortConflict(t *testing.T) {
	// This test simulates a scenario where restart might fail due to port conflicts
	// which could be similar to what happens with the python server
	tempDir := t.TempDir()

	// Create a package.json with a script that uses a specific port
	packageJSON := `{
		"name": "test-app",  
		"scripts": {
			"server": "python3 -m http.server 3001"
		}
	}`

	err := writeFile(tempDir+"/package.json", packageJSON)
	require.NoError(t, err)

	// Create process manager
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(tempDir, eventBus, true)
	require.NoError(t, err)

	// Start the initial process
	proc, err := processMgr.StartScript("server")
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Wait for process to be running
	time.Sleep(200 * time.Millisecond)

	// Verify initial process is running
	allProcs := processMgr.GetAllProcesses()
	require.Len(t, allProcs, 1)

	t.Logf("Initial process: %s (PID: %s, Status: %s)", allProcs[0].Name, allProcs[0].ID, allProcs[0].Status)

	// Simulate restart with short timeout (might cause issues)
	timeout := 1 * time.Second // Shorter timeout

	// Step 1: Stop the process and wait
	err = processMgr.StopProcessAndWait(proc.ID, timeout)
	if err != nil {
		t.Logf("StopProcessAndWait failed: %v", err)
	}

	// Step 2: Immediately try to start the script again (potential conflict)
	_, err = processMgr.StartScript("server")
	if err != nil {
		t.Logf("StartScript failed: %v", err)
	}

	// Wait a moment for processes to settle
	time.Sleep(500 * time.Millisecond)

	// Check the final state
	finalProcs := processMgr.GetAllProcesses()

	t.Logf("Final process count: %d", len(finalProcs))
	for i, p := range finalProcs {
		t.Logf("Process %d: %s (PID: %s, Status: %s)", i+1, p.Name, p.ID, p.Status)
	}

	// Count failed processes - this is where the bug manifests
	var failedCount int
	for _, p := range finalProcs {
		if p.GetStatus() == process.StatusFailed {
			failedCount++
		}
	}

	// The bug: restart creates 2 failed processes instead of 1 working process
	if failedCount >= 2 {
		t.Errorf("BUG REPRODUCED: Found %d failed processes after restart (should be 0 or 1)", failedCount)

		// Additional debugging
		for _, p := range finalProcs {
			if p.GetStatus() == process.StatusFailed {
				t.Logf("Failed process: %s (PID: %s)", p.Name, p.ID)
			}
		}
	}
}
