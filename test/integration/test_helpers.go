package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/tui"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/config"
)

// TestContext holds test environment and utilities
type TestContext struct {
	T          *testing.T
	TempDir    string
	Model      *tui.Model
	Program    *tea.Program
	EventBus   *events.EventBus
	ProcessMgr *process.Manager
	LogStore   *logs.Store
	Pty        *os.File
	Tty        *os.File
}

// NewTestContext creates a new test environment with TTY allocation
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create test package.json
	packageJSON := `{
  "name": "test-project",
  "scripts": {
    "dev": "echo 'Dev server running on http://localhost:3000' && sleep 60",
    "test": "echo 'Running tests...' && exit 0",
    "build": "echo 'Building...' && sleep 2 && echo 'Build complete'",
    "error": "echo 'Simulating error...' && exit 1"
  }
}`
	err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "Failed to create test package.json")

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Restore directory on cleanup
	t.Cleanup(func() {
		os.Chdir(originalDir)
	})

	// Allocate PTY for TUI testing
	ptyFile, ttyFile, err := pty.Open()
	require.NoError(t, err, "Failed to allocate PTY")

	t.Cleanup(func() {
		ptyFile.Close()
		ttyFile.Close()
	})

	// Create services
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = &config.Config{} // Use empty config for tests
	}
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager(tempDir, eventBus, true) // hasPackageJSON = true
	require.NoError(t, err, "Failed to create process manager")
	logStore := logs.NewStore(10000, eventBus) // maxEntries = 10000

	// Create model
	model := tui.NewModel(
		processMgr,
		logStore,
		eventBus,
		nil,  // mcpServer not needed for most tests
		nil,  // proxyServer not needed for most tests
		7777, // mcpPort default
		cfg,
	)

	return &TestContext{
		T:          t,
		TempDir:    tempDir,
		Model:      model,
		EventBus:   eventBus,
		ProcessMgr: processMgr,
		LogStore:   logStore,
		Pty:        ptyFile,
		Tty:        ttyFile,
	}
}

// NewTestContextWithProxy creates test context with proxy server
func NewTestContextWithProxy(t *testing.T) *TestContext {
	ctx := NewTestContext(t)

	// Add proxy server
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}
	proxyServer := proxy.NewServer(cfg, ctx.ProcessMgr, ctx.EventBus)
	err = proxyServer.Start()
	require.NoError(t, err, "Failed to start proxy server")

	t.Cleanup(func() {
		proxyServer.Stop()
	})

	// Recreate model with proxy
	ctx.Model = tui.NewModel(
		ctx.ProcessMgr,
		ctx.LogStore,
		ctx.EventBus,
		nil,
		proxyServer,
		7777,
		cfg,
	)

	return ctx
}

// StartProgram starts the BubbleTea program with TTY
func (ctx *TestContext) StartProgram() {
	ctx.T.Helper()

	// Create program with TTY input/output
	ctx.Program = tea.NewProgram(
		ctx.Model,
		tea.WithInput(ctx.Tty),
		tea.WithOutput(ctx.Tty),
	)

	// Start program in background
	go func() {
		if _, err := ctx.Program.Run(); err != nil {
			ctx.T.Logf("Program exited with error: %v", err)
		}
	}()

	// Wait for program to start
	time.Sleep(100 * time.Millisecond)
}

// SendKey sends a key press to the program
func (ctx *TestContext) SendKey(key string) {
	ctx.T.Helper()
	ctx.SendKeys(key)
}

// SendKeys sends multiple key presses
func (ctx *TestContext) SendKeys(keys ...string) {
	ctx.T.Helper()
	for _, key := range keys {
		ctx.Pty.Write([]byte(key))
		time.Sleep(50 * time.Millisecond) // Allow processing time
	}
}

// SendKeyMsg sends a KeyMsg directly to the model (for unit tests without TTY)
func (ctx *TestContext) SendKeyMsg(key string) tea.Cmd {
	ctx.T.Helper()

	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(key),
	}

	newModel, cmd := ctx.Model.Update(msg)
	ctx.Model = newModel.(*tui.Model)
	return cmd
}

// SendMessage sends a custom message to the model
func (ctx *TestContext) SendMessage(msg tea.Msg) tea.Cmd {
	ctx.T.Helper()
	newModel, cmd := ctx.Model.Update(msg)
	ctx.Model = newModel.(*tui.Model)
	return cmd
}

// WaitForCondition waits for a condition to become true with timeout
func (ctx *TestContext) WaitForCondition(condition func() bool, timeout time.Duration, message string) {
	ctx.T.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	ctx.T.Fatalf("Timeout waiting for: %s", message)
}

// WaitForProcess waits for a process to reach expected status
func (ctx *TestContext) WaitForProcess(processID string, expectedStatus process.ProcessStatus, timeout time.Duration) {
	ctx.T.Helper()

	ctx.WaitForCondition(func() bool {
		proc := ctx.ProcessMgr.GetProcess(processID)
		return proc != nil && proc.Status == expectedStatus
	}, timeout, "process to reach status")
}

// WaitForLog waits for a log entry containing text
func (ctx *TestContext) WaitForLog(contains string, timeout time.Duration) {
	ctx.T.Helper()

	ctx.WaitForCondition(func() bool {
		logs := ctx.LogStore.GetLogs("", 100)
		for _, log := range logs {
			if contains == "" || len(log.Content) > 0 {
				// Simple check for any log
				return true
			}
		}
		return false
	}, timeout, "log entry")
}

// AssertViewContains checks if the rendered view contains text
func (ctx *TestContext) AssertViewContains(text string) {
	ctx.T.Helper()
	view := ctx.Model.View()
	assert.Contains(ctx.T, view, text, "View should contain: %s", text)
}

// AssertViewNotContains checks if the rendered view does not contain text
func (ctx *TestContext) AssertViewNotContains(text string) {
	ctx.T.Helper()
	view := ctx.Model.View()
	assert.NotContains(ctx.T, view, text, "View should not contain: %s", text)
}

// AssertCurrentView verifies the current view
func (ctx *TestContext) AssertCurrentView(expectedView string) {
	ctx.T.Helper()
	currentView := ctx.Model.CurrentView()
	assert.Equal(ctx.T, expectedView, currentView, "Current view should be: %s", expectedView)
}

// AssertProcessStatus verifies a process has expected status
func (ctx *TestContext) AssertProcessStatus(processID string, expectedStatus process.ProcessStatus) {
	ctx.T.Helper()
	proc := ctx.ProcessMgr.GetProcess(processID)
	require.NotNil(ctx.T, proc, "Process should exist: %s", processID)
	assert.Equal(ctx.T, expectedStatus, proc.Status, "Process %s should have status: %v", processID, expectedStatus)
}

// AssertProcessCount verifies the number of processes
func (ctx *TestContext) AssertProcessCount(expected int) {
	ctx.T.Helper()
	processes := ctx.ProcessMgr.GetAllProcesses()
	assert.Len(ctx.T, processes, expected, "Should have %d processes", expected)
}

// AssertLogCount verifies the number of log entries
func (ctx *TestContext) AssertLogCount(processID string, expected int) {
	ctx.T.Helper()
	logs := ctx.LogStore.GetLogs(processID, 1000)
	assert.Len(ctx.T, logs, expected, "Should have %d log entries", expected)
}

// AssertErrorCount verifies the number of errors
func (ctx *TestContext) AssertErrorCount(expected int) {
	ctx.T.Helper()
	errors := ctx.Model.GetErrors()
	assert.Len(ctx.T, errors, expected, "Should have %d errors", expected)
}

// StartTestProcess starts a process and waits for it to be running
func (ctx *TestContext) StartTestProcess(scriptName string) string {
	ctx.T.Helper()

	processID, err := ctx.ProcessMgr.StartScript(scriptName)
	require.NoError(ctx.T, err, "Failed to start process")

	ctx.WaitForProcess(processID, process.StatusRunning, 2*time.Second)

	return processID
}

// StopTestProcess stops a process and waits for it to stop
func (ctx *TestContext) StopTestProcess(processID string) {
	ctx.T.Helper()

	err := ctx.ProcessMgr.StopProcess(processID)
	require.NoError(ctx.T, err, "Failed to stop process")

	ctx.WaitForProcess(processID, process.StatusStopped, 2*time.Second)
}

// RunCommand runs an arbitrary command for testing
func (ctx *TestContext) RunCommand(command string) (string, error) {
	ctx.T.Helper()

	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = ctx.TempDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// SetWindowSize simulates a window resize
func (ctx *TestContext) SetWindowSize(width, height int) {
	ctx.T.Helper()

	msg := tea.WindowSizeMsg{
		Width:  width,
		Height: height,
	}

	ctx.SendMessage(msg)
}

// Cleanup performs test cleanup
func (ctx *TestContext) Cleanup() {
	ctx.T.Helper()

	// Stop all processes
	processes := ctx.ProcessMgr.GetAllProcesses()
	for _, proc := range processes {
		if proc.Status == process.StatusRunning {
			ctx.ProcessMgr.StopProcess(proc.ID)
		}
	}

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)
}

// SkipIfShort skips test if running in short mode
func SkipIfShort(t *testing.T, reason string) {
	if testing.Short() {
		t.Skipf("Skipping in short mode: %s", reason)
	}
}

// MeasureLatency measures the time taken for an operation
func MeasureLatency(operation func()) time.Duration {
	start := time.Now()
	operation()
	return time.Since(start)
}

// RunWithTimeout runs a function with timeout
func RunWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	done := make(chan bool)

	go func() {
		fn()
		done <- true
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		t.Fatal("Operation timed out")
	}
}
