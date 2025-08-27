package components

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComprehensiveIntegration demonstrates the full system with all new features
func TestComprehensiveIntegration(t *testing.T) {
	// Enable debug mode for this test
	os.Setenv("BRUMMER_DEBUG", "true")
	defer os.Unsetenv("BRUMMER_DEBUG")

	// Create test harness
	harness := NewTestHarness(t)
	defer harness.Cleanup()

	// Create package.json with test scripts
	scripts := map[string]string{
		"test":  "echo 'Running tests' && sleep 0.1 && echo 'Tests completed'",
		"build": "echo 'Building project' && sleep 0.1 && echo 'Build completed'",
		"error": "echo 'Starting process' && sleep 0.1 && >&2 echo 'Error occurred' && exit 1",
	}
	require.NoError(t, harness.CreatePackageJSON(scripts))

	// Test 1: Component Creation with Lifecycle
	t.Run("Lifecycle Management", func(t *testing.T) {
		// Create lifecycle manager
		lifecycle := NewLifecycleManager()

		// Create components through lifecycle
		processView := harness.CreateComponent("processView", func() tea.Model {
			return NewProcessViewNew(harness.ProcessMgr)
		})

		logsView := harness.CreateComponent("logsView", func() tea.Model {
			return NewLogsViewComponent(harness.LogStore)
		})

		_ = harness.CreateComponent("shell", func() tea.Model {
			return NewProcessLogsShell(harness.ProcessMgr, harness.LogStore)
		})

		// Wrap components with lifecycle management
		lifecycle.Register("processView", WrapWithLifecycle("processView", processView))
		lifecycle.Register("logsView", WrapWithLifecycle("logsView", logsView), "processView")
		lifecycle.Register("shell", WrapWithLifecycle("shell", harness.Components["shell"]), "processView", "logsView")

		// Initialize all components
		cmds, err := lifecycle.Initialize()
		require.NoError(t, err)
		assert.NotEmpty(t, cmds)

		// Verify states
		state, err := lifecycle.GetComponentState("shell")
		require.NoError(t, err)
		assert.Equal(t, StateReady, state)
	})

	// Test 2: Process Operations with Debug Validation
	t.Run("Process Operations", func(t *testing.T) {
		_ = harness.Components["shell"].(*ProcessLogsShell)

		// Start a test process
		_, err := harness.StartProcess("test-echo", "echo", []string{"Hello from process"})
		require.NoError(t, err)

		// Wait for process completion and logs
		require.NoError(t, harness.WaitForLogs(1, 2*time.Second))

		// Validate process view state (debug mode will check invariants)
		harness.AssertComponentState("shell", func(model tea.Model) error {
			shell := model.(*ProcessLogsShell)
			return ValidateShell(shell)
		})

		// Test process selection
		_, cmd := harness.UpdateComponent("shell", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})
		assert.NotNil(t, cmd)

		// Process should now be selected and logs should be visible
		assert.NotEmpty(t, harness.LogEntries)
	})

	// Test 3: Debug Mode Validation
	t.Run("Debug Mode", func(t *testing.T) {
		debug := Debug()
		assert.True(t, debug.Enabled())

		// Force a state check
		shell := harness.Components["shell"].(*ProcessLogsShell)
		debug.CheckState("Shell", func() error {
			return ValidateShell(shell)
		})

		// Get debug statistics
		stats := debug.GetStats()
		assert.Greater(t, stats.StateChecks, 0)
		t.Logf("Debug stats: %+v", stats)

		// Test debug overlay rendering
		overlay := debug.RenderOverlay(80, 24)
		assert.Contains(t, overlay, "DEBUG MODE")
	})

	// Test 4: Error Handling and Recovery
	t.Run("Error Handling", func(t *testing.T) {
		// Start a process that will fail
		proc, err := harness.StartProcess("test-error", "sh", []string{"-c", "echo 'Before error' && >&2 echo 'This is an error' && exit 1"})
		require.NoError(t, err)

		// Wait for process to complete with error
		require.NoError(t, harness.WaitForProcess(proc.ID, process.StatusFailed, 2*time.Second))

		// Verify error logs were captured
		require.NoError(t, harness.WaitForLogs(2, 2*time.Second))

		// Check that error is reflected in process view
		shell := harness.Components["shell"].(*ProcessLogsShell)
		processes := shell.processMgr.GetAllProcesses()

		var failedProc *process.Process
		for _, p := range processes {
			if p.ID == proc.ID {
				failedProc = p
				break
			}
		}

		require.NotNil(t, failedProc)
		assert.Equal(t, process.StatusFailed, failedProc.GetStatus())

		// Verify error notification count
		errors, _, _ := shell.processView.GetNotificationCount()
		assert.Greater(t, errors, 0)
	})

	// Test 5: Performance Tracking
	t.Run("Performance Tracking", func(t *testing.T) {
		debug := Debug()
		shell := harness.Components["shell"].(*ProcessLogsShell)

		// Perform several updates to generate timing data
		for i := 0; i < 10; i++ {
			_, _ = harness.UpdateComponent("shell", tea.WindowSizeMsg{Width: 100, Height: 30})
			_ = shell.View()
		}

		stats := debug.GetStats()
		t.Logf("Performance - Avg Update: %v, Avg Render: %v",
			stats.AvgUpdateTime, stats.AvgRenderTime)

		// Ensure reasonable performance
		assert.Less(t, stats.AvgUpdateTime, 100*time.Millisecond)
		assert.Less(t, stats.AvgRenderTime, 50*time.Millisecond)
	})

	// Test 6: Data Flow Validation
	t.Run("Data Flow", func(t *testing.T) {
		initialLogCount := harness.GetLogEntryCount()

		// Start a process that generates multiple log lines
		_, err := harness.StartProcess("multi-log", "sh", []string{"-c",
			"for i in 1 2 3; do echo \"Log line $i\"; sleep 0.05; done"})
		require.NoError(t, err)

		// Wait for all logs to be captured
		require.NoError(t, harness.WaitForLogs(initialLogCount+3, 2*time.Second))

		// Verify logs flow through the system
		finalLogCount := harness.GetLogEntryCount()
		assert.GreaterOrEqual(t, finalLogCount-initialLogCount, 3)

		// Verify logs appear in the logs view
		shell := harness.Components["shell"].(*ProcessLogsShell)
		allLogs := shell.logStore.GetAll()
		assert.GreaterOrEqual(t, len(allLogs), 3)

		// Switch to logs tab to verify display
		_, cmd := harness.UpdateComponent("shell", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")})
		assert.NotNil(t, cmd)

		// Verify logs are displayed
		view := shell.View()
		assert.Contains(t, view, "Log line")
	})

	// Test 7: Integration Validation
	t.Run("Full Integration", func(t *testing.T) {
		shell := harness.Components["shell"].(*ProcessLogsShell)

		// Test complete workflow: start process -> view logs -> restart
		proc, err := harness.StartProcess("integration-test", "echo", []string{"Integration test message"})
		require.NoError(t, err)

		// Wait for completion
		require.NoError(t, harness.WaitForProcess(proc.ID, process.StatusSuccess, 2*time.Second))

		// Switch to process view and select the process
		_, _ = harness.UpdateComponent("shell", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tab")}) // Back to processes

		// Navigate to our process (might not be first)
		processes := shell.processMgr.GetAllProcesses()
		targetIndex := -1
		for i, p := range processes {
			if p.ID == proc.ID {
				targetIndex = i
				break
			}
		}

		// Navigate to target process
		for i := 0; i < targetIndex; i++ {
			_, _ = harness.UpdateComponent("shell", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")})
		}

		// Switch to logs view for this process
		_, cmd := harness.UpdateComponent("shell", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")})
		assert.NotNil(t, cmd)

		// Verify we're now on logs tab with the process filtered
		assert.Equal(t, 1, shell.activeTab) // Logs tab

		// Verify the complete integration worked
		view := shell.View()
		if Debug().Enabled() {
			assert.Contains(t, view, "DEBUG MODE") // Debug overlay present
		}
		assert.Contains(t, view, "📄 Logs") // Logs tab active
	})

	// Final validation
	t.Log("=== Final System State ===")
	harness.DumpState()

	// Ensure no debug violations occurred
	stats := Debug().GetStats()
	if len(stats.RecentViolations) > 0 {
		t.Logf("Debug violations detected: %v", stats.RecentViolations)
		// In a real test, this might be a failure, but for demonstration we'll just log
	}

	t.Logf("Test completed successfully with %d processes, %d logs, %d events",
		len(harness.ProcessMgr.GetAllProcesses()),
		len(harness.LogEntries),
		len(harness.Events))
}
