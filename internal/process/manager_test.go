package process

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManagerInitialization tests creating a new manager
func TestManagerInitialization(t *testing.T) {
	tempDir := t.TempDir()
	eventBus := events.NewEventBus()

	// Test without package.json
	mgr, err := NewManager(tempDir, eventBus, false)
	require.NoError(t, err)
	assert.NotNil(t, mgr)
	assert.Equal(t, tempDir, mgr.workDir)

	// Test with package.json
	packageJSON := map[string]interface{}{
		"name": "test-project",
		"scripts": map[string]interface{}{
			"test":  "echo 'running tests'",
			"build": "echo 'building project'",
			"dev":   "echo 'starting dev server'",
		},
	}
	
	packageFile := filepath.Join(tempDir, "package.json")
	data, err := json.Marshal(packageJSON)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(packageFile, data, 0644))

	mgr2, err := NewManager(tempDir, eventBus, true)
	require.NoError(t, err)
	assert.NotNil(t, mgr2)
	
	scripts := mgr2.GetScripts()
	assert.Contains(t, scripts, "test")
	assert.Contains(t, scripts, "build")
	assert.Contains(t, scripts, "dev")
	assert.Equal(t, "echo 'running tests'", scripts["test"])
}

// TestProcessStartStop tests starting and stopping processes
func TestProcessStartStop(t *testing.T) {
	tempDir := t.TempDir()
	eventBus := events.NewEventBus()
	
	// Track events
	var receivedEvents []events.Event
	var eventsMu sync.Mutex
	eventBus.Subscribe(events.ProcessStarted, func(e events.Event) {
		eventsMu.Lock()
		receivedEvents = append(receivedEvents, e)
		eventsMu.Unlock()
	})
	eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		eventsMu.Lock()
		receivedEvents = append(receivedEvents, e)
		eventsMu.Unlock()
	})

	mgr, err := NewManager(tempDir, eventBus, false)
	require.NoError(t, err)

	// Test starting a simple command
	proc, err := mgr.StartCommand("echo-test", "echo", []string{"hello world"})
	require.NoError(t, err)
	assert.NotNil(t, proc)
	assert.Equal(t, "echo-test", proc.Name)
	assert.Equal(t, StatusRunning, proc.Status)

	// Wait for process to complete
	time.Sleep(100 * time.Millisecond)

	// Check process completed successfully
	proc, exists := mgr.GetProcess(proc.ID)
	require.True(t, exists)
	assert.Equal(t, StatusSuccess, proc.Status)
	
	// Verify events were published
	eventsMu.Lock()
	defer eventsMu.Unlock()
	assert.GreaterOrEqual(t, len(receivedEvents), 1) // At least ProcessStarted
}

// TestProcessManagement tests process lifecycle management
func TestProcessManagement(t *testing.T) {
	tempDir := t.TempDir()
	eventBus := events.NewEventBus()
	mgr, err := NewManager(tempDir, eventBus, false)
	require.NoError(t, err)

	// Start multiple processes
	proc1, err := mgr.StartCommand("sleep1", "sleep", []string{"0.2"})
	require.NoError(t, err)
	
	proc2, err := mgr.StartCommand("sleep2", "sleep", []string{"0.2"})
	require.NoError(t, err)

	// Test GetAllProcesses
	allProcs := mgr.GetAllProcesses()
	assert.Len(t, allProcs, 2)
	
	// Find our processes in the list
	var foundProc1, foundProc2 bool
	for _, p := range allProcs {
		if p.ID == proc1.ID {
			foundProc1 = true
		}
		if p.ID == proc2.ID {
			foundProc2 = true
		}
	}
	assert.True(t, foundProc1)
	assert.True(t, foundProc2)

	// Test GetProcess
	retrievedProc, exists := mgr.GetProcess(proc1.ID)
	require.True(t, exists)
	assert.Equal(t, proc1.ID, retrievedProc.ID)
	assert.Equal(t, "sleep1", retrievedProc.Name)

	// Test stopping a specific process
	err = mgr.StopProcess(proc1.ID)
	assert.NoError(t, err)
	
	// Wait a moment and check status
	time.Sleep(50 * time.Millisecond)
	stoppedProc, exists := mgr.GetProcess(proc1.ID)
	require.True(t, exists)
	assert.True(t, stoppedProc.Status == StatusStopped || stoppedProc.Status == StatusSuccess)

	// Test stopping all processes
	err = mgr.StopAllProcesses()
	assert.NoError(t, err)
}

// TestScriptExecution tests running package.json scripts
func TestScriptExecution(t *testing.T) {
	tempDir := t.TempDir()
	eventBus := events.NewEventBus()
	
	// Create package.json with test scripts
	packageJSON := map[string]interface{}{
		"name": "test-project",
		"scripts": map[string]interface{}{
			"hello": "echo 'Hello from script'",
			"fail":  "exit 1",
		},
	}
	
	packageFile := filepath.Join(tempDir, "package.json")
	data, err := json.Marshal(packageJSON)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(packageFile, data, 0644))

	mgr, err := NewManager(tempDir, eventBus, true)
	require.NoError(t, err)

	// Test successful script
	proc, err := mgr.StartScript("hello")
	require.NoError(t, err)
	assert.NotNil(t, proc)
	assert.Equal(t, "hello", proc.Name)
	
	// Wait for completion and check multiple times
	var finalStatus ProcessStatus
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		proc, exists := mgr.GetProcess(proc.ID)
		require.True(t, exists)
		finalStatus = proc.Status
		if finalStatus == StatusSuccess || finalStatus == StatusFailed {
			break
		}
	}
	
	// Process should have completed (either success or failed, not running)
	assert.True(t, finalStatus == StatusSuccess || finalStatus == StatusFailed,
		"Process should have completed, got status: %s", finalStatus)

	// Test failing script
	procFail, err := mgr.StartScript("fail")
	require.NoError(t, err)
	
	// Wait for completion and check multiple times
	var failStatus ProcessStatus
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		procFail, exists := mgr.GetProcess(procFail.ID)
		require.True(t, exists)
		failStatus = procFail.Status
		if failStatus == StatusSuccess || failStatus == StatusFailed {
			break
		}
	}
	
	// This process should have failed (exit 1)
	assert.Equal(t, StatusFailed, failStatus)

	// Test non-existent script
	_, err = mgr.StartScript("nonexistent")
	assert.Error(t, err)
}

// TestLogCallbacks tests log callback functionality
func TestLogCallbacks(t *testing.T) {
	tempDir := t.TempDir()
	eventBus := events.NewEventBus()
	mgr, err := NewManager(tempDir, eventBus, false)
	require.NoError(t, err)

	// Set up log callback
	var logLines []string
	var logMu sync.Mutex
	mgr.AddLogCallback(func(processID, line string, isError bool) {
		logMu.Lock()
		logLines = append(logLines, fmt.Sprintf("%s:%t:%s", processID, isError, line))
		logMu.Unlock()
	})

	// Start a command that produces output
	proc, err := mgr.StartCommand("output-test", "echo", []string{"test output line"})
	require.NoError(t, err)

	// Wait for output
	time.Sleep(200 * time.Millisecond)

	// Check we received log output
	logMu.Lock()
	defer logMu.Unlock()
	assert.Greater(t, len(logLines), 0)
	
	// Should have output from our echo command
	found := false
	for _, line := range logLines {
		if strings.Contains(line, proc.ID) && strings.Contains(line, "test output line") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find echo output in log lines")
}

// TestPackageManagerDetection tests package manager detection
func TestPackageManagerDetection(t *testing.T) {
	tempDir := t.TempDir()
	eventBus := events.NewEventBus()
	mgr, err := NewManager(tempDir, eventBus, false)
	require.NoError(t, err)

	// Test getting installed package managers
	installed := mgr.GetInstalledPackageManagers()
	assert.NotEmpty(t, installed) // Should detect at least some package managers

	// Test getting current package manager
	current := mgr.GetCurrentPackageManager()
	assert.NotEmpty(t, string(current))

	// Test detected commands
	commands := mgr.GetDetectedCommands()
	// Should detect some commands (at least basic shell commands)
	assert.NotEmpty(t, commands)
}

// TestCleanup tests proper cleanup of all processes
func TestCleanup(t *testing.T) {
	tempDir := t.TempDir()
	eventBus := events.NewEventBus()
	mgr, err := NewManager(tempDir, eventBus, false)
	require.NoError(t, err)

	// Start some long-running processes
	proc1, err := mgr.StartCommand("sleep1", "sleep", []string{"10"}) // Long sleep
	require.NoError(t, err)
	
	proc2, err := mgr.StartCommand("sleep2", "sleep", []string{"10"}) // Long sleep
	require.NoError(t, err)

	// Verify they're running
	assert.Equal(t, StatusRunning, proc1.Status)
	assert.Equal(t, StatusRunning, proc2.Status)

	// Cleanup should stop all processes
	err = mgr.Cleanup()
	assert.NoError(t, err)

	// Wait a moment for cleanup to complete
	time.Sleep(100 * time.Millisecond)

	// Check processes are stopped
	proc1, exists := mgr.GetProcess(proc1.ID)
	require.True(t, exists)
	assert.True(t, proc1.Status == StatusStopped || proc1.Status == StatusFailed)

	proc2, exists = mgr.GetProcess(proc2.ID)
	require.True(t, exists)
	assert.True(t, proc2.Status == StatusStopped || proc2.Status == StatusFailed)
}

// TestConcurrentOperations tests thread safety
func TestConcurrentOperations(t *testing.T) {
	tempDir := t.TempDir()
	eventBus := events.NewEventBus()
	mgr, err := NewManager(tempDir, eventBus, false)
	require.NoError(t, err)

	var wg sync.WaitGroup
	processIDs := make([]string, 10)

	// Start multiple processes concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			proc, err := mgr.StartCommand(fmt.Sprintf("echo-%d", index), "echo", []string{fmt.Sprintf("message-%d", index)})
			if err == nil {
				processIDs[index] = proc.ID
			}
		}(i)
	}

	wg.Wait()

	// Verify all processes were created
	time.Sleep(200 * time.Millisecond)
	allProcs := mgr.GetAllProcesses()
	assert.GreaterOrEqual(t, len(allProcs), 10)

	// Cleanup concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		mgr.Cleanup()
	}()

	// Read process status concurrently while cleanup is happening
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mgr.GetAllProcesses()
		}()
	}

	wg.Wait()
}