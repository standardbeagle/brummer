package components

import (
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

func TestLogsViewComponent_NewComponent(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	// Add some initial logs
	logStore.Add("proc1", "Process1", "Initial log 1", false)
	logStore.Add("proc2", "Process2", "Initial log 2", true)

	// Wait for async add to complete
	time.Sleep(10 * time.Millisecond)

	// Create component (should load existing logs)
	component := NewLogsViewComponent(logStore)

	// Verify initial logs were loaded
	if len(component.logs) != 2 {
		t.Errorf("Expected 2 initial logs, got %d", len(component.logs))
	}

	view := component.View()
	if !strings.Contains(view, "Initial log 1") {
		t.Error("View should contain 'Initial log 1'")
	}
	if !strings.Contains(view, "Initial log 2") {
		t.Error("View should contain 'Initial log 2'")
	}
}

func TestLogsViewComponent_LogAddedMsg(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	component := NewLogsViewComponent(logStore)

	// Add a log via message (event-driven update)
	entry := &logs.LogEntry{
		ID:          "test-1",
		ProcessID:   "test-proc",
		ProcessName: "TestProc",
		Timestamp:   time.Now(),
		Content:     "New log via message",
		IsError:     false,
	}

	msg := messages.LogAddedMsg{Entry: entry}
	component.Update(msg)

	// Verify log was added
	if len(component.logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(component.logs))
	}

	view := component.View()
	if !strings.Contains(view, "New log via message") {
		t.Error("View should contain 'New log via message'")
	}
}

func TestLogsViewComponent_MultipleMessages(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	component := NewLogsViewComponent(logStore)

	// Add multiple logs via messages
	for i := 1; i <= 5; i++ {
		entry := &logs.LogEntry{
			ID:          "test-" + string(rune(i)),
			ProcessID:   "test-proc",
			ProcessName: "Test",
			Timestamp:   time.Now(),
			Content:     "Log " + string(rune('0'+i)),
			IsError:     false,
		}

		component.Update(messages.LogAddedMsg{Entry: entry})
	}

	// Verify all logs were added
	if len(component.logs) != 5 {
		t.Errorf("Expected 5 logs, got %d", len(component.logs))
	}
}

func TestLogsViewComponent_ErrorFormatting(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	component := NewLogsViewComponent(logStore)

	// Add error log
	errorEntry := &logs.LogEntry{
		ID:          "error-1",
		ProcessID:   "test",
		ProcessName: "Test",
		Timestamp:   time.Now(),
		Content:     "Error message",
		IsError:     true,
	}

	component.Update(messages.LogAddedMsg{Entry: errorEntry})

	// Verify log was formatted with timestamp and content
	formattedLog := component.logs[0]
	if !strings.Contains(formattedLog, "Error message") {
		t.Error("Log should contain 'Error message'")
	}

	// Verify timestamp format is present
	if !strings.Contains(formattedLog, "[") || !strings.Contains(formattedLog, "]") {
		t.Error("Log should have timestamp in brackets")
	}

	// Verify process name is included
	if !strings.Contains(formattedLog, "Test") {
		t.Error("Log should contain process name")
	}
}

func TestLogsViewComponent_SetSize(t *testing.T) {
	component := NewLogsViewComponent(nil)

	// Set size
	component.SetSize(100, 50)

	if component.Width() != 100 {
		t.Errorf("Expected width 100, got %d", component.Width())
	}
	if component.Height() != 50 {
		t.Errorf("Expected height 50, got %d", component.Height())
	}
	if component.viewport.Width != 100 {
		t.Errorf("Expected viewport width 100, got %d", component.viewport.Width)
	}
	if component.viewport.Height != 50 {
		t.Errorf("Expected viewport height 50, got %d", component.viewport.Height)
	}
}

func TestLogsViewComponent_FocusBlur(t *testing.T) {
	component := NewLogsViewComponent(nil)

	// Test focus
	component.Focus()
	if !component.IsFocused() {
		t.Error("Component should be focused")
	}

	// Test blur
	component.Blur()
	if component.IsFocused() {
		t.Error("Component should not be focused")
	}
}

func TestLogsViewComponent_Visibility(t *testing.T) {
	component := NewLogsViewComponent(nil)

	// Default is visible
	if !component.IsVisible() {
		t.Error("Component should be visible by default")
	}

	// Test hiding
	component.SetVisible(false)
	if component.IsVisible() {
		t.Error("Component should be hidden")
	}

	// View should be empty when hidden
	view := component.View()
	if view != "" {
		t.Error("Hidden component should return empty view")
	}
}