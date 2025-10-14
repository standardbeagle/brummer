package components

import (
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

func TestProcessViewComponent_NewComponent(t *testing.T) {
	// Create component with nil manager (no initial processes)
	component := NewProcessViewComponent(nil)
	component.SetSize(80, 24) // Initialize viewport size

	if len(component.processes) != 0 {
		t.Errorf("Expected 0 initial processes, got %d", len(component.processes))
	}

	view := component.View()
	if !strings.Contains(view, "No processes running") {
		t.Error("View should show 'No processes running' when empty")
	}
}

func TestProcessViewComponent_ProcessStartedMsg(t *testing.T) {
	component := NewProcessViewComponent(nil)
	component.SetSize(80, 24)

	// Add a process via message
	startTime := time.Now()
	msg := messages.ProcessStartedMsg{
		ProcessID: "dev-123",
		Name:      "dev",
		Script:    "npm run dev",
		StartTime: startTime,
	}

	component.Update(msg)

	// Verify process was added
	if len(component.processes) != 1 {
		t.Fatalf("Expected 1 process, got %d", len(component.processes))
	}

	info, exists := component.processes["dev-123"]
	if !exists {
		t.Fatal("Process 'dev-123' should exist")
	}

	if info.Name != "dev" {
		t.Errorf("Expected name 'dev', got '%s'", info.Name)
	}

	if info.Status != process.StatusRunning {
		t.Errorf("Expected status Running, got %v", info.Status)
	}

	// Verify view contains process
	view := component.View()
	if !strings.Contains(view, "dev") {
		t.Error("View should contain process name 'dev'")
	}
}

func TestProcessViewComponent_ProcessStoppedMsg(t *testing.T) {
	component := NewProcessViewComponent(nil)
	component.SetSize(80, 24)

	// Start a process
	startTime := time.Now()
	startMsg := messages.ProcessStartedMsg{
		ProcessID: "dev-123",
		Name:      "dev",
		Script:    "npm run dev",
		StartTime: startTime,
	}
	component.Update(startMsg)

	// Stop the process
	endTime := time.Now().Add(5 * time.Second)
	exitCode := 0
	stopMsg := messages.ProcessStoppedMsg{
		ProcessID: "dev-123",
		Name:      "dev",
		ExitCode:  exitCode,
		EndTime:   endTime,
	}
	component.Update(stopMsg)

	// Verify process was updated
	info, exists := component.processes["dev-123"]
	if !exists {
		t.Fatal("Process 'dev-123' should still exist after stopping")
	}

	if info.Status != process.StatusStopped {
		t.Errorf("Expected status Stopped, got %v", info.Status)
	}

	if info.ExitCode == nil {
		t.Fatal("ExitCode should be set")
	}

	if *info.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", *info.ExitCode)
	}

	if info.EndTime == nil {
		t.Fatal("EndTime should be set")
	}
}

func TestProcessViewComponent_ProcessStateChangedMsg(t *testing.T) {
	component := NewProcessViewComponent(nil)
	component.SetSize(80, 24)

	// Start a process
	startMsg := messages.ProcessStartedMsg{
		ProcessID: "dev-123",
		Name:      "dev",
		Script:    "npm run dev",
		StartTime: time.Now(),
	}
	component.Update(startMsg)

	// Change state to failed
	stateMsg := messages.ProcessStateChangedMsg{
		ProcessID: "dev-123",
		OldState:  process.StatusRunning,
		NewState:  process.StatusFailed,
	}
	component.Update(stateMsg)

	// Verify status was updated
	info, exists := component.processes["dev-123"]
	if !exists {
		t.Fatal("Process 'dev-123' should exist")
	}

	if info.Status != process.StatusFailed {
		t.Errorf("Expected status Failed, got %v", info.Status)
	}
}

func TestProcessViewComponent_MultipleProcesses(t *testing.T) {
	component := NewProcessViewComponent(nil)
	component.SetSize(80, 24)

	// Add multiple processes
	processes := []string{"dev", "test", "build"}
	for i, name := range processes {
		msg := messages.ProcessStartedMsg{
			ProcessID: name + "-" + string(rune('1'+i)),
			Name:      name,
			Script:    "npm run " + name,
			StartTime: time.Now(),
		}
		component.Update(msg)
	}

	// Verify all processes were added
	if len(component.processes) != 3 {
		t.Errorf("Expected 3 processes, got %d", len(component.processes))
	}

	// Verify view contains all process names
	view := component.View()
	for _, name := range processes {
		if !strings.Contains(view, name) {
			t.Errorf("View should contain process name '%s'", name)
		}
	}
}

func TestProcessViewComponent_RuntimeDisplay(t *testing.T) {
	component := NewProcessViewComponent(nil)
	component.SetSize(80, 24)

	// Add a process that started 65 seconds ago
	startTime := time.Now().Add(-65 * time.Second)
	msg := messages.ProcessStartedMsg{
		ProcessID: "dev-123",
		Name:      "dev",
		Script:    "npm run dev",
		StartTime: startTime,
	}
	component.Update(msg)

	// Verify view contains runtime information
	view := component.View()

	// Should show "1m" format (at least 1 minute runtime)
	if !strings.Contains(view, "1m") && !strings.Contains(view, "65s") {
		t.Error("View should contain runtime information")
		t.Logf("View content: %s", view)
	}
}

func TestProcessViewComponent_SetSize(t *testing.T) {
	component := NewProcessViewComponent(nil)

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

func TestProcessViewComponent_Visibility(t *testing.T) {
	component := NewProcessViewComponent(nil)

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

func TestProcessViewComponent_ExitCodeFormatting(t *testing.T) {
	component := NewProcessViewComponent(nil)
	component.SetSize(80, 24)

	// Start and stop a process with non-zero exit code
	startMsg := messages.ProcessStartedMsg{
		ProcessID: "test-123",
		Name:      "test",
		Script:    "npm test",
		StartTime: time.Now(),
	}
	component.Update(startMsg)

	exitCode := 1
	stopMsg := messages.ProcessStoppedMsg{
		ProcessID: "test-123",
		Name:      "test",
		ExitCode:  exitCode,
		EndTime:   time.Now(),
	}
	component.Update(stopMsg)

	// Verify exit code is displayed
	view := component.View()
	if !strings.Contains(view, "1") {
		t.Error("View should contain exit code '1'")
	}
}
