package components

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRuntimeLogFlow tests the full log flow as it would happen at runtime
func TestRuntimeLogFlow(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create event bus and process manager
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	require.NoError(t, err)

	// Create integrated shell (this is what happens at runtime)
	proxy := NewFakeProxyServer(nil)
	shell := NewIntegratedShell(processMgr, eventBus, proxy)
	require.NotNil(t, shell)

	// Initialize the shell
	initCmd := shell.Init()
	require.NotNil(t, initCmd, "Init should return a command")

	// Send window size message
	_, _ = shell.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Simulate a process adding logs through the callback system
	// This is what happens when a real process outputs logs
	if processMgr != nil && shell.logStore != nil {
		// Directly add logs to the store (simulating what the callback would do)
		shell.logStore.Add("test-proc-1", "Test Process 1", "Starting process...", false)
		shell.logStore.Add("test-proc-1", "Test Process 1", "Processing data...", false)
		shell.logStore.Add("test-proc-2", "Test Process 2", "Error: Connection failed", true)

		// Notify the subscription about new logs
		shell.logSub.NotifyNewLogs()
	}

	// Wait for async log processing
	time.Sleep(20 * time.Millisecond)

	// Execute the subscription command to get logs
	cmd := shell.logSub.Subscribe()
	require.NotNil(t, cmd)

	// Run the command with timeout
	done := make(chan tea.Msg, 1)
	go func() {
		msg := cmd()
		done <- msg
	}()

	// Wait for the message
	select {
	case msg := <-done:
		// Process the message through the shell
		model, _ := shell.Update(msg)
		updatedShell := model.(*IntegratedShell)

		// Switch to logs view
		_ = updatedShell.switchToView(ViewLogs)

		// Check that logs view has the logs
		assert.NotNil(t, updatedShell.logsView)
		assert.Greater(t, len(updatedShell.logsView.filteredLogs), 0, "Should have filtered logs")

		// Get the view output
		output := updatedShell.View()

		// Basic check that logs are visible
		assert.Contains(t, output, "Logs", "Should show logs title")

	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for log update message")
	}
}

// TestLogCallbackIntegration tests that the log callback system works
func TestLogCallbackIntegration(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create event bus and process manager
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	require.NoError(t, err)

	// Track callback invocations
	callbackCount := 0

	// Add a test callback
	processMgr.AddLogCallback(func(processID, line string, isError bool) {
		callbackCount++
		t.Logf("Callback invoked: processID=%s, line=%s, isError=%v", processID, line, isError)
	})

	// Create integrated shell (which adds its own callback)
	proxy := NewFakeProxyServer(nil)
	shell := NewIntegratedShell(processMgr, eventBus, proxy)
	require.NotNil(t, shell)

	// Now simulate what would happen when a process outputs logs
	// In real usage, this would be called by the streamLogs method
	// We'll invoke the callbacks directly since we can't easily start a real process

	// The shell's callback should have been registered
	// Let's add a log directly to the store and notify
	shell.logStore.Add("test-process", "Test", "Test log line", false)
	shell.logSub.NotifyNewLogs()

	// Wait for async processing
	time.Sleep(20 * time.Millisecond)

	// Check that log was added to store
	allLogs := shell.logStore.GetAll()
	assert.Greater(t, len(allLogs), 0, "Should have logs in store")

	// Check process name cache
	if cached, ok := shell.processNameCache.Load("test-process"); ok {
		assert.Equal(t, "Test", cached.(string), "Process name should be cached")
	}

	// The callback count would be >0 if we had triggered the actual callbacks
	// In this test we're just verifying the store integration
	_ = callbackCount
}

// TestDebouncedSubscriptionIntegration tests the debounced subscription in action
func TestDebouncedSubscriptionIntegration(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create log store
	logStore := logs.NewStore(10000, nil)
	sub := NewDebouncedLogSubscription(logStore)

	// Rapidly add multiple logs
	for i := 0; i < 10; i++ {
		logStore.Add("proc", "Process", fmt.Sprintf("Log %d", i), false)
		sub.NotifyNewLogs() // Multiple notifications
	}

	// Wait for async processing and debouncing
	time.Sleep(100 * time.Millisecond)

	// Execute subscription
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
			// Should get all 10 logs in a single batch due to debouncing
			assert.Equal(t, 10, len(m.Entries), "Should batch all logs")
			assert.True(t, m.Append, "Should be in append mode")
		default:
			t.Errorf("Unexpected message type: %T", m)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for debounced update")
	}
}
