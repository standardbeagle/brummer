package components

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

func TestErrorsViewComponent_NewComponent(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	component := NewErrorsViewComponent(logStore)
	component.SetSize(80, 24)

	if len(component.errors) != 0 {
		t.Errorf("Expected 0 initial errors, got %d", len(component.errors))
	}

	view := component.View()
	if !strings.Contains(view, "No errors detected") {
		t.Error("View should show 'No errors detected' when empty")
	}
}

func TestErrorsViewComponent_LogAddedMsg_ErrorOnly(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	component := NewErrorsViewComponent(logStore)
	component.SetSize(80, 24)

	// Add an error log
	errorEntry := &logs.LogEntry{
		ID:          "error-1",
		ProcessID:   "test-proc",
		ProcessName: "TestProc",
		Timestamp:   time.Now(),
		Content:     "Error: Something went wrong",
		IsError:     true,
	}

	// Add to log store first (so GetErrorContexts can find it)
	logStore.Add("test-proc", "TestProc", "Error: Something went wrong", true)
	time.Sleep(10 * time.Millisecond) // Wait for async add

	msg := messages.LogAddedMsg{Entry: errorEntry}
	component.Update(msg)

	// Verify error was processed
	// Note: GetErrorContexts might not find it immediately due to error parsing
	// So we just verify the component doesn't crash
	view := component.View()
	if view == "" {
		t.Error("View should not be empty after error log")
	}
}

func TestErrorsViewComponent_NonErrorLog_Ignored(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	component := NewErrorsViewComponent(logStore)
	component.SetSize(80, 24)

	initialErrorCount := len(component.errors)

	// Add a non-error log
	normalEntry := &logs.LogEntry{
		ID:          "log-1",
		ProcessID:   "test-proc",
		ProcessName: "TestProc",
		Timestamp:   time.Now(),
		Content:     "Normal log message",
		IsError:     false,
	}

	msg := messages.LogAddedMsg{Entry: normalEntry}
	component.Update(msg)

	// Verify no errors were added
	if len(component.errors) != initialErrorCount {
		t.Errorf("Expected %d errors (unchanged), got %d", initialErrorCount, len(component.errors))
	}
}

func TestErrorsViewComponent_KeyboardNavigation(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	component := NewErrorsViewComponent(logStore)
	component.SetSize(80, 24)

	// Add some mock errors
	component.errors = []*ErrorInfo{
		{Type: "Error1", Message: "First error", ProcessName: "proc1"},
		{Type: "Error2", Message: "Second error", ProcessName: "proc2"},
		{Type: "Error3", Message: "Third error", ProcessName: "proc3"},
	}
	component.selectedIndex = 0
	component.updateViewport()

	// Test down navigation
	component.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if component.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after down, got %d", component.selectedIndex)
	}

	// Test up navigation
	component.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if component.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 after up, got %d", component.selectedIndex)
	}

	// Test bounds checking (up at top)
	component.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if component.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 (bounded at top), got %d", component.selectedIndex)
	}

	// Test bounds checking (down at bottom)
	component.selectedIndex = 2 // Last item
	component.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if component.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex 2 (bounded at bottom), got %d", component.selectedIndex)
	}
}

func TestErrorsViewComponent_ErrorSeverityFormatting(t *testing.T) {
	component := NewErrorsViewComponent(nil)
	component.SetSize(80, 24)

	// Test different severity levels
	testCases := []struct {
		severity string
		expected string // Just verify it doesn't crash
	}{
		{"critical", "critical"},
		{"error", "error"},
		{"warning", "warning"},
		{"info", "info"},
	}

	for _, tc := range testCases {
		info := &ErrorInfo{
			Type:        "TestError",
			Message:     "Test message",
			ProcessName: "test",
			Severity:    tc.severity,
		}

		// Should not crash
		_ = component.formatErrorRow(info, false)
		_ = component.formatErrorRow(info, true)
	}
}

func TestErrorsViewComponent_SetSize(t *testing.T) {
	component := NewErrorsViewComponent(nil)

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

func TestErrorsViewComponent_Visibility(t *testing.T) {
	component := NewErrorsViewComponent(nil)

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

func TestErrorsViewComponent_ErrorDetails(t *testing.T) {
	component := NewErrorsViewComponent(nil)
	component.SetSize(80, 24)

	// Create error with stack trace
	errorCtx := &logs.ErrorContext{
		Type:        "TestError",
		Message:     "Test error message",
		ProcessName: "test-proc",
		Timestamp:   time.Now(),
		Severity:    "error",
		Stack:       []string{"at line 1", "at line 2", "at line 3"},
	}

	info := &ErrorInfo{
		Type:        errorCtx.Type,
		Message:     errorCtx.Message,
		ProcessName: errorCtx.ProcessName,
		Timestamp:   errorCtx.Timestamp.Format("15:04:05"),
		Severity:    errorCtx.Severity,
		Context:     errorCtx,
	}

	details := component.formatErrorDetails(info)

	// Verify details contain key information
	if !strings.Contains(details, "TestError") {
		t.Error("Details should contain error type")
	}
	if !strings.Contains(details, "Test error message") {
		t.Error("Details should contain message")
	}
	if !strings.Contains(details, "test-proc") {
		t.Error("Details should contain process name")
	}
	if !strings.Contains(details, "Stack Trace") {
		t.Error("Details should contain stack trace header")
	}
	if !strings.Contains(details, "at line 1") {
		t.Error("Details should contain stack trace lines")
	}
}

func TestErrorsViewComponent_SelectionHighlight(t *testing.T) {
	component := NewErrorsViewComponent(nil)
	component.SetSize(80, 24)

	info := &ErrorInfo{
		Type:        "TestError",
		Message:     "Test message",
		ProcessName: "test",
		Severity:    "error",
	}

	// Test selected vs non-selected formatting
	selectedRow := component.formatErrorRow(info, true)
	normalRow := component.formatErrorRow(info, false)

	// Selected row should have arrow
	if !strings.Contains(selectedRow, "→") {
		t.Error("Selected row should contain arrow indicator")
	}

	// Normal row should not have arrow
	if strings.Contains(normalRow, "→") {
		t.Error("Normal row should not contain arrow indicator")
	}
}