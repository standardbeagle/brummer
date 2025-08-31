package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDirectLogFlow tests the direct flow of logs through the component system
func TestDirectLogFlow(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create a simple log store
	logStore := logs.NewStore(10000, nil)

	// Add some test logs directly
	logStore.Add("proc1", "Process 1", "First log line", false)
	logStore.Add("proc1", "Process 1", "Second log line", false)
	logStore.Add("proc2", "Process 2", "Third log line", true) // error log

	// Wait for async processing
	time.Sleep(10 * time.Millisecond)

	// Create logs view component
	logsView := NewLogsViewComponent(logStore)
	require.NotNil(t, logsView)

	// Initialize the component
	logsView.Init()

	// Set size
	logsView.SetSize(120, 30)

	// Get the view output
	output := logsView.View()

	// Check that logs are displayed
	assert.Contains(t, output, "First log line", "Should display first log")
	assert.Contains(t, output, "Second log line", "Should display second log")
	assert.Contains(t, output, "Third log line", "Should display third log")
}

// TestLogSubscriptionFlow tests the subscription-based log flow
func TestLogSubscriptionFlow(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create log store
	logStore := logs.NewStore(10000, nil)

	// Create debounced subscription
	sub := NewDebouncedLogSubscription(logStore)
	require.NotNil(t, sub)

	// Add initial logs
	logStore.Add("proc1", "Process 1", "Initial log", false)

	// Notify subscription
	sub.NotifyNewLogs()

	// Get subscription command
	cmd := sub.Subscribe()
	require.NotNil(t, cmd)

	// Execute command with timeout
	done := make(chan tea.Msg, 1)
	go func() {
		msg := cmd()
		done <- msg
	}()

	select {
	case msg := <-done:
		switch m := msg.(type) {
		case LogsUpdateMsg:
			assert.Equal(t, 1, len(m.Entries), "Should have 1 log entry")
			assert.True(t, m.Append, "Should be in append mode")
			assert.Equal(t, "Initial log", m.Entries[0].Content)
		default:
			t.Errorf("Unexpected message type: %T", m)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for log update")
	}
}

// TestIntegratedShellInitialization tests proper initialization of the integrated shell
func TestIntegratedShellInitialization(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create process manager
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	require.NoError(t, err)

	// Create integrated shell
	proxy := NewFakeProxyServer(nil)
	shell := NewIntegratedShell(processMgr, eventBus, proxy)
	require.NotNil(t, shell)

	// Verify all components are initialized
	assert.NotNil(t, shell.processView, "Process view should be initialized")
	assert.NotNil(t, shell.logsView, "Logs view should be initialized")
	assert.NotNil(t, shell.logStore, "Log store should be initialized")
	assert.NotNil(t, shell.logSub, "Log subscription should be initialized")
	assert.NotNil(t, shell.processNameCache, "Process name cache should be initialized")

	// Initialize the shell
	initCmd := shell.Init()
	assert.NotNil(t, initCmd, "Init should return a command")

	// Send window size to initialize properly
	model, cmd := shell.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	assert.NotNil(t, model)
	_ = cmd

	// Verify dimensions were updated
	assert.Equal(t, 120, shell.width)
	assert.Equal(t, 30, shell.height)
}

// TestLogDeliveryThroughShell tests that logs added to the store are delivered through the shell
func TestLogDeliveryThroughShell(t *testing.T) {
	// Initialize colors
	MustInitColors()

	// Create process manager
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(".", eventBus, false)
	require.NoError(t, err)

	// Create integrated shell
	proxy := NewFakeProxyServer(nil)
	shell := NewIntegratedShell(processMgr, eventBus, proxy)
	require.NotNil(t, shell)

	// Initialize shell
	shell.Init()
	shell.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Add logs directly to the store
	shell.logStore.Add("test-proc", "Test Process", "Test log line 1", false)
	shell.logStore.Add("test-proc", "Test Process", "Test log line 2", false)

	// Notify the subscription about new logs
	shell.logSub.NotifyNewLogs()

	// Create and execute the subscription command
	cmd := shell.logSub.Subscribe()
	require.NotNil(t, cmd)

	// Execute command with timeout
	done := make(chan tea.Msg, 1)
	go func() {
		msg := cmd()
		done <- msg
	}()

	select {
	case msg := <-done:
		// Process the message through the shell
		model, _ := shell.Update(msg)
		updatedShell := model.(*IntegratedShell)

		// Switch to logs view
		updatedShell.switchToView(ViewLogs)

		// Get the view output
		output := updatedShell.View()

		// Verify logs are visible
		assert.Contains(t, output, "Test log line 1", "First log should be visible")
		assert.Contains(t, output, "Test log line 2", "Second log should be visible")

	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for log update")
	}
}
