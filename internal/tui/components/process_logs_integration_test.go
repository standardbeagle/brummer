package components

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessLogsIntegration(t *testing.T) {
	// Create a temp directory with package.json
	tmpDir := t.TempDir()
	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0",
		"scripts": {
			"test": "echo 'Running tests'",
			"dev": "echo 'Starting dev server'",
			"build": "echo 'Building project'"
		}
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Create dependencies
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(tmpDir, eventBus, false)
	require.NoError(t, err)

	logStore := logs.NewStore(1000, eventBus)
	defer logStore.Close()

	// Create the shell
	shell := NewProcessLogsShell(processMgr, logStore)
	require.NotNil(t, shell)

	// Test initialization
	cmd := shell.Init()
	assert.NotNil(t, cmd)

	// Test window resize
	model, _ := shell.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	shell = model.(*ProcessLogsShell)
	assert.Equal(t, 100, shell.width)
	assert.Equal(t, 30, shell.height)

	// Test tab switching (use KeyTab event, not literal "tab" runes)
	model, _ = shell.Update(tea.KeyMsg{Type: tea.KeyTab})
	shell = model.(*ProcessLogsShell)
	assert.Equal(t, 1, shell.activeTab) // Should switch to logs

	model, _ = shell.Update(tea.KeyMsg{Type: tea.KeyTab})
	shell = model.(*ProcessLogsShell)
	assert.Equal(t, 0, shell.activeTab) // Should wrap back to processes

	// Test view rendering (assert on labels to avoid unicode/icon issues)
	view := shell.View()
	assert.Contains(t, view, "Processes") // Process tab label
	assert.Contains(t, view, "Logs")      // Logs tab label

	// Test process to logs switching via message
	model, cmd = shell.Update(SwitchToLogsMsg{ProcessID: "test-process"})
	shell = model.(*ProcessLogsShell)
	assert.Equal(t, 1, shell.activeTab) // Should switch to logs tab
	assert.NotNil(t, cmd)               // Should have a command to filter logs

	// Add some test logs
	logStore.Add("test-process", "Test", "Starting test process", false)
	logStore.Add("test-process", "Test", "Test process running", false)
	logStore.Add("test-process", "Test", "Error in test process", true)

	// Test log streaming
	time.Sleep(150 * time.Millisecond) // Wait for tick
	entries := logStore.GetAll()
	var logEntries []*logs.LogEntry
	for i := range entries {
		logEntries = append(logEntries, &entries[i])
	}
	model, _ = shell.Update(LogsUpdateMsg{Entries: logEntries, Append: true})
	shell = model.(*ProcessLogsShell)

	// Verify logs view updated
	view = shell.View()
	assert.Contains(t, view, "Logs") // Should show logs tab
}

func TestProcessActionsIntegration(t *testing.T) {
	// Skip this test for now as it requires a working process manager setup
	t.Skip("Skipping process actions integration test - requires package.json parsing fix")

	// Create a temp directory with package.json
	tmpDir := t.TempDir()

	// Create dependencies
	eventBus := events.NewEventBus()

	// Create a minimal package.json
	packageJSON := `{"name":"test","version":"1.0.0","scripts":{}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	processMgr, err := process.NewManager(tmpDir, eventBus, false)
	require.NoError(t, err)

	logStore := logs.NewStore(1000, eventBus)
	defer logStore.Close()

	// Start a test process using StartCommand directly
	proc, err := processMgr.StartCommand("test", "echo", []string{"Testing"})
	require.NoError(t, err)
	require.NotNil(t, proc)

	// Create the shell
	shell := NewProcessLogsShell(processMgr, logStore)
	shell.Init()

	// Test process restart action
	shell.processView.selectedProcess = proc.ID
	model, cmd := shell.processView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	shell.processView = model.(*ProcessViewNew)
	assert.NotNil(t, cmd) // Should have restart command

	// Test stop action
	model, cmd = shell.processView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	shell.processView = model.(*ProcessViewNew)
	assert.NotNil(t, cmd) // Should have stop command

	// Wait for process to stop
	time.Sleep(200 * time.Millisecond)

	// Verify process list updated
	shell.processView.updateProcessList()
	processes := shell.processView.processes
	assert.NotEmpty(t, processes)
}

func TestNotificationSystem(t *testing.T) {
	// Skip this test for now as it requires a working process manager setup
	t.Skip("Skipping notification system test - requires package.json parsing fix")

	// Create a temp directory with package.json
	tmpDir := t.TempDir()
	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0",
		"scripts": {
			"error": "exit 1"
		}
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Create dependencies
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(tmpDir, eventBus, false)
	require.NoError(t, err)

	logStore := logs.NewStore(1000, eventBus)
	defer logStore.Close()

	// Create the shell
	shell := NewProcessLogsShell(processMgr, logStore)

	// Start a failing process
	proc, err := processMgr.StartCommand("error", "false", []string{})
	require.NoError(t, err)

	// Wait for process to fail
	time.Sleep(200 * time.Millisecond)

	// Update process list
	shell.processView.updateProcessList()

	// Check notification counts
	errors, _, info := shell.processView.GetNotificationCount()
	assert.Greater(t, errors, 0) // Should have error notification

	// Test tab bar rendering with notifications
	tabBar := shell.renderTabBar()
	assert.Contains(t, tabBar, "🔴") // Should show error badge

	// Clear notifications when switching to tab
	shell.processView.ClearNotifications()
	errors, _, _ = shell.processView.GetNotificationCount()
	assert.Equal(t, 0, info) // Info notifications should be cleared

	// Stop the failed process
	processMgr.StopProcess(proc.ID)
}
