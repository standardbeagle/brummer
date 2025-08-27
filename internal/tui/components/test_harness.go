// Package components provides testing utilities for component integration
package components

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/require"
)

// ComponentTestHarness provides a standard test environment for components
type ComponentTestHarness struct {
	t          *testing.T
	ProcessMgr *process.Manager
	LogStore   *logs.Store
	EventBus   *events.EventBus
	TempDir    string

	// Track initialization state
	initialized      bool
	callbacksSetup   bool
	processesStarted bool

	// Components under test
	Components map[string]tea.Model

	// Captured data with synchronization
	mu         sync.Mutex // Protects LogEntries, Events, Messages
	LogEntries []logs.LogEntry
	Events     []events.Event
	Messages   []tea.Msg
}

// NewTestHarness creates a properly initialized test harness
func NewTestHarness(t *testing.T) *ComponentTestHarness {
	h := &ComponentTestHarness{
		t:          t,
		TempDir:    t.TempDir(),
		EventBus:   events.NewEventBus(),
		Components: make(map[string]tea.Model),
		LogEntries: []logs.LogEntry{},
		Events:     []events.Event{},
		Messages:   []tea.Msg{},
	}

	// Initialize log store
	h.LogStore = logs.NewStore(1000, h.EventBus)

	// Initialize process manager
	var err error
	h.ProcessMgr, err = process.NewManager(h.TempDir, h.EventBus, false)
	require.NoError(t, err, "Failed to create process manager")

	// CRITICAL: Register callbacks BEFORE starting any processes
	h.setupCallbacks()

	// Subscribe to events for testing
	h.subscribeToEvents()

	h.initialized = true
	return h
}

// setupCallbacks registers all necessary callbacks before processes start
func (h *ComponentTestHarness) setupCallbacks() {
	if h.callbacksSetup {
		h.t.Fatal("Callbacks already setup - duplicate initialization")
	}

	// Register log callback to capture process output
	h.ProcessMgr.AddLogCallback(func(processID, line string, isError bool) {
		// Get process name for the log entry
		processName := "Unknown"
		for _, proc := range h.ProcessMgr.GetAllProcesses() {
			if proc.ID == processID {
				processName = proc.Name
				break
			}
		}

		// Add to log store
		entry := h.LogStore.Add(processID, processName, line, isError)
		if entry != nil {
			h.mu.Lock()
			h.LogEntries = append(h.LogEntries, *entry)
			h.mu.Unlock()
		}
	})

	h.callbacksSetup = true
}

// subscribeToEvents sets up event monitoring for tests
func (h *ComponentTestHarness) subscribeToEvents() {
	// Subscribe to all events for debugging
	// Note: EventTypeAll might not exist, so we'll subscribe to common event types
	// h.EventBus.Subscribe(events.EventTypeAll, func(e events.Event) {
	// 	h.Events = append(h.Events, e)
	// })
	// For now, we'll capture events in the callback directly
}

// StartProcess starts a test process with validation
func (h *ComponentTestHarness) StartProcess(name, command string, args []string) (*process.Process, error) {
	if !h.initialized {
		return nil, fmt.Errorf("harness not initialized")
	}
	if !h.callbacksSetup {
		return nil, fmt.Errorf("callbacks not setup - would lose logs")
	}

	h.processesStarted = true
	return h.ProcessMgr.StartCommand(name, command, args)
}

// StartScriptProcess starts a package.json script
func (h *ComponentTestHarness) StartScriptProcess(scriptName string) (*process.Process, error) {
	if !h.initialized {
		return nil, fmt.Errorf("harness not initialized")
	}
	if !h.callbacksSetup {
		return nil, fmt.Errorf("callbacks not setup - would lose logs")
	}

	h.processesStarted = true
	return h.ProcessMgr.StartScript(scriptName)
}

// CreatePackageJSON creates a test package.json file
func (h *ComponentTestHarness) CreatePackageJSON(scripts map[string]string) error {
	packageJSON := `{
  "name": "test-project",
  "version": "1.0.0",
  "scripts": {`

	first := true
	for name, cmd := range scripts {
		if !first {
			packageJSON += ","
		}
		packageJSON += fmt.Sprintf(`
    "%s": "%s"`, name, cmd)
		first = false
	}

	packageJSON += `
  }
}`

	packageFile := filepath.Join(h.TempDir, "package.json")
	return os.WriteFile(packageFile, []byte(packageJSON), 0644)
}

// CreateComponent creates and initializes a component with proper lifecycle
func (h *ComponentTestHarness) CreateComponent(name string, creator func() tea.Model) tea.Model {
	component := creator()

	// Run initialization
	if initer, ok := component.(interface{ Init() tea.Cmd }); ok {
		cmd := initer.Init()
		// Execute the command if needed
		if cmd != nil {
			msg := cmd()
			if msg != nil {
				h.Messages = append(h.Messages, msg)
			}
		}
	}

	h.Components[name] = component
	return component
}

// UpdateComponent sends a message to a component and captures the result
func (h *ComponentTestHarness) UpdateComponent(name string, msg tea.Msg) (tea.Model, tea.Cmd) {
	component, exists := h.Components[name]
	if !exists {
		h.t.Fatalf("Component %s not found", name)
	}

	updater, ok := component.(interface {
		Update(tea.Msg) (tea.Model, tea.Cmd)
	})
	if !ok {
		h.t.Fatalf("Component %s doesn't implement Update", name)
	}

	newModel, cmd := updater.Update(msg)
	h.Components[name] = newModel

	// Capture any resulting message
	if cmd != nil {
		if msg := cmd(); msg != nil {
			h.Messages = append(h.Messages, msg)
		}
	}

	return newModel, cmd
}

// WaitForLogs waits for a specific number of log entries
func (h *ComponentTestHarness) WaitForLogs(count int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		entries := h.LogStore.GetAll()
		if len(entries) >= count {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for %d logs, got %d", count, len(h.LogStore.GetAll()))
}

// WaitForProcess waits for a process to reach a specific status
func (h *ComponentTestHarness) WaitForProcess(processID string, status process.ProcessStatus, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		for _, proc := range h.ProcessMgr.GetAllProcesses() {
			if proc.ID == processID && proc.GetStatus() == status {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for process %s to reach status %v", processID, status)
}

// AssertComponentState validates component state with detailed errors
func (h *ComponentTestHarness) AssertComponentState(name string, validator func(tea.Model) error) {
	component, exists := h.Components[name]
	if !exists {
		h.t.Fatalf("Component %s not found", name)
	}

	if err := validator(component); err != nil {
		h.t.Errorf("Component %s state validation failed: %v", name, err)
		h.DumpState()
	}
}

// DumpState outputs the current harness state for debugging
func (h *ComponentTestHarness) DumpState() {
	h.t.Logf("=== Test Harness State Dump ===")
	h.t.Logf("Initialized: %v", h.initialized)
	h.t.Logf("Callbacks Setup: %v", h.callbacksSetup)
	h.t.Logf("Processes Started: %v", h.processesStarted)

	h.t.Logf("\n--- Processes ---")
	for _, proc := range h.ProcessMgr.GetAllProcesses() {
		h.t.Logf("  %s: %v (started: %v)", proc.ID, proc.GetStatus(), proc.StartTime)
	}

	h.t.Logf("\n--- Log Entries ---")
	entries := h.LogStore.GetAll()
	for i, entry := range entries {
		if i >= 10 {
			h.t.Logf("  ... and %d more", len(entries)-10)
			break
		}
		h.t.Logf("  [%s] %d: %s", entry.ProcessID, int(entry.Level), entry.Content)
	}

	h.t.Logf("\n--- Events ---")
	events := h.GetEvents()
	for i, event := range events {
		if i >= 10 {
			h.t.Logf("  ... and %d more", len(events)-10)
			break
		}
		h.t.Logf("  %s: %v", event.Type, event.Data)
	}

	h.t.Logf("\n--- Components ---")
	for name := range h.Components {
		h.t.Logf("  %s: initialized", name)
	}

	h.t.Logf("================================")
}

// GetLogEntryCount returns the current number of captured log entries (thread-safe)
func (h *ComponentTestHarness) GetLogEntryCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.LogEntries)
}

// GetLogEntries returns a copy of captured log entries (thread-safe)
func (h *ComponentTestHarness) GetLogEntries() []logs.LogEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Return a copy to prevent external mutations
	entries := make([]logs.LogEntry, len(h.LogEntries))
	copy(entries, h.LogEntries)
	return entries
}

// GetEventCount returns the current number of captured events (thread-safe)
func (h *ComponentTestHarness) GetEventCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.Events)
}

// GetEvents returns a copy of captured events (thread-safe)
func (h *ComponentTestHarness) GetEvents() []events.Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Return a copy to prevent external mutations
	evts := make([]events.Event, len(h.Events))
	copy(evts, h.Events)
	return evts
}

// Cleanup properly shuts down the test harness
func (h *ComponentTestHarness) Cleanup() {
	// Stop all processes
	if h.ProcessMgr != nil {
		h.ProcessMgr.Cleanup()
	}

	// Close log store
	if h.LogStore != nil {
		h.LogStore.Close()
	}

	// Clean up temp directory
	if h.TempDir != "" {
		os.RemoveAll(h.TempDir)
	}
}

// ValidateProcessViewState validates ProcessViewNew state
func ValidateProcessViewState(view tea.Model) error {
	pv, ok := view.(*ProcessViewNew)
	if !ok {
		return fmt.Errorf("not a ProcessViewNew")
	}

	// Validate selection invariants
	if pv.selectedIndex >= len(pv.processes) && pv.selectedIndex != -1 {
		return fmt.Errorf("invalid selectedIndex: %d (processes: %d)",
			pv.selectedIndex, len(pv.processes))
	}

	if pv.selectedProcess != "" {
		found := false
		for _, proc := range pv.processes {
			if proc.ID == pv.selectedProcess {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("selectedProcess '%s' not in process list", pv.selectedProcess)
		}
	}

	// Validate scroll offset
	if pv.scrollOffset < 0 {
		return fmt.Errorf("negative scrollOffset: %d", pv.scrollOffset)
	}

	if pv.scrollOffset > len(pv.processes) {
		return fmt.Errorf("scrollOffset %d exceeds process count %d",
			pv.scrollOffset, len(pv.processes))
	}

	return nil
}

// ValidateLogsViewState validates LogsViewComponent state
func ValidateLogsViewState(view tea.Model) error {
	lv, ok := view.(*LogsViewComponent)
	if !ok {
		return fmt.Errorf("not a LogsViewComponent")
	}

	// No longer checking against logEntries since we fetch from store

	// Validate viewport dimensions
	if lv.viewport.Width <= 0 || lv.viewport.Height <= 0 {
		return fmt.Errorf("invalid viewport dimensions: %dx%d",
			lv.viewport.Width, lv.viewport.Height)
	}

	return nil
}
