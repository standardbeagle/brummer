package handlers

import (
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/commands"
	"github.com/standardbeagle/brummer/internal/tui/types"
	tea "github.com/charmbracelet/bubbletea"
)

// TestProgressHandlerCreation tests creating a new progress handler
func TestProgressHandlerCreation(t *testing.T) {
	handler := NewProgressHandler(10, time.Second)

	if handler == nil {
		t.Fatal("Expected non-nil progress handler")
	}

	if handler.maxHistory != 10 {
		t.Errorf("Expected max history 10, got %d", handler.maxHistory)
	}

	if handler.updateInterval != time.Second {
		t.Errorf("Expected update interval 1s, got %v", handler.updateInterval)
	}
}

// TestProgressStart tests starting a new progress operation
func TestProgressStart(t *testing.T) {
	handler := NewProgressHandler(10, time.Second)

	progress := handler.StartProgress("test-cmd", commands.CommandTypeIO, 100, "Test operation")

	if progress == nil {
		t.Fatal("Expected non-nil progress")
	}

	if progress.CommandID != "test-cmd" {
		t.Errorf("Expected command ID 'test-cmd', got '%s'", progress.CommandID)
	}

	if progress.Total != 100 {
		t.Errorf("Expected total 100, got %d", progress.Total)
	}

	if progress.Status != ProgressRunning {
		t.Errorf("Expected status %v, got %v", ProgressRunning, progress.Status)
	}
}

// TestProgressUpdate tests updating a progress operation
func TestProgressUpdate(t *testing.T) {
	handler := NewProgressHandler(10, time.Second)
	progress := handler.StartProgress("test-cmd", commands.CommandTypeIO, 100, "Test operation")

	cmd := handler.UpdateProgress(progress.ID, 50, 100, "Half way")

	if cmd == nil {
		t.Error("Expected non-nil command from progress update")
	}

	// Check progress was updated
	updatedProgress, exists := handler.GetProgress(progress.ID)
	if !exists {
		t.Fatal("Expected progress to exist")
	}

	if updatedProgress.Current != 50 {
		t.Errorf("Expected current 50, got %d", updatedProgress.Current)
	}

	if updatedProgress.Percentage != 50.0 {
		t.Errorf("Expected percentage 50.0, got %f", updatedProgress.Percentage)
	}
}

// TestProgressCompletion tests completing a progress operation
func TestProgressCompletion(t *testing.T) {
	handler := NewProgressHandler(10, time.Second)
	progress := handler.StartProgress("test-cmd", commands.CommandTypeIO, 100, "Test operation")

	// Complete the progress
	cmd := handler.UpdateProgress(progress.ID, 100, 100, "Complete")

	if cmd == nil {
		t.Error("Expected non-nil command from progress completion")
	}

	// Progress should be moved to completed history
	_, exists := handler.GetProgress(progress.ID)
	if exists {
		t.Error("Expected progress to be removed from active progress")
	}

	completed := handler.GetCompletedProgress(1)
	if len(completed) != 1 {
		t.Errorf("Expected 1 completed progress, got %d", len(completed))
	}
}

// TestProgressCancel tests cancelling a progress operation
func TestProgressCancel(t *testing.T) {
	handler := NewProgressHandler(10, time.Second)
	progress := handler.StartProgress("test-cmd", commands.CommandTypeIO, 100, "Test operation")

	cancelled := handler.CancelProgress(progress.ID)

	if !cancelled {
		t.Error("Expected progress to be cancelled")
	}

	// Check progress status
	updatedProgress, exists := handler.GetProgress(progress.ID)
	if !exists {
		t.Fatal("Expected progress to still exist")
	}

	if updatedProgress.Status != ProgressCancelled {
		t.Errorf("Expected status %v, got %v", ProgressCancelled, updatedProgress.Status)
	}
}

// TestProgressStats tests getting progress statistics
func TestProgressStats(t *testing.T) {
	handler := NewProgressHandler(10, time.Second)

	// Start some progress operations
	handler.StartProgress("test-1", commands.CommandTypeIO, 100, "Test 1")
	handler.StartProgress("test-2", commands.CommandTypeProcess, 200, "Test 2")

	stats := handler.GetStats()

	if stats.ActiveOperations != 2 {
		t.Errorf("Expected 2 active operations, got %d", stats.ActiveOperations)
	}

	if stats.TotalOperations != 2 {
		t.Errorf("Expected 2 total operations, got %d", stats.TotalOperations)
	}

	// Check operation counts by type
	if stats.OperationsByType[commands.CommandTypeIO] != 1 {
		t.Errorf("Expected 1 IO operation, got %d", stats.OperationsByType[commands.CommandTypeIO])
	}

	if stats.OperationsByType[commands.CommandTypeProcess] != 1 {
		t.Errorf("Expected 1 Process operation, got %d", stats.OperationsByType[commands.CommandTypeProcess])
	}
}

// TestSubProgress tests adding and updating sub-progress
func TestSubProgress(t *testing.T) {
	handler := NewProgressHandler(10, time.Second)
	progress := handler.StartProgress("test-cmd", commands.CommandTypeIO, 100, "Test operation")

	// Add sub-progress
	handler.AddSubProgress(progress.ID, "sub-1", "Sub task 1", 50)

	// Update sub-progress
	cmd := handler.UpdateSubProgress(progress.ID, "sub-1", 25, 50)

	if cmd == nil {
		t.Error("Expected non-nil command from sub-progress update")
	}

	// Check sub-progress was updated
	updatedProgress, exists := handler.GetProgress(progress.ID)
	if !exists {
		t.Fatal("Expected progress to exist")
	}

	subProgress, exists := updatedProgress.SubProgress["sub-1"]
	if !exists {
		t.Fatal("Expected sub-progress to exist")
	}

	if subProgress.Current != 25 {
		t.Errorf("Expected sub-progress current 25, got %d", subProgress.Current)
	}

	if subProgress.Percentage != 50.0 {
		t.Errorf("Expected sub-progress percentage 50.0, got %f", subProgress.Percentage)
	}
}

// TestProgressCleanup tests cleaning up old completed progress
func TestProgressCleanup(t *testing.T) {
	handler := NewProgressHandler(10, time.Second)
	progress := handler.StartProgress("test-cmd", commands.CommandTypeIO, 100, "Test operation")

	// Complete the progress
	handler.UpdateProgress(progress.ID, 100, 100, "Complete")

	// Verify we have completed progress
	completed := handler.GetCompletedProgress(1)
	if len(completed) != 1 {
		t.Errorf("Expected 1 completed progress, got %d", len(completed))
	}

	// Cleanup with very short age (should remove nothing)
	handler.Cleanup(time.Nanosecond)

	completed = handler.GetCompletedProgress(1)
	if len(completed) != 1 {
		t.Error("Expected completed progress to remain after cleanup with very short age")
	}

	// Cleanup with future time (should remove everything)
	handler.Cleanup(time.Hour)

	completed = handler.GetCompletedProgress(1)
	if len(completed) != 0 {
		t.Error("Expected no completed progress after cleanup with future time")
	}
}

// TestFormatProgress tests the FormatProgress utility function
func TestFormatProgress(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		expected string
	}{
		{"Zero values", 0, 0, "0/?"},
		{"Partial progress", 25, 100, "25/100 (25.0%)"},
		{"Complete", 100, 100, "100/100 (100.0%)"},
		{"Over 100%", 150, 100, "150/100 (150.0%)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatProgress(tt.current, tt.total)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCreateProgressBar tests the CreateProgressBar utility function
func TestCreateProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		width    int
		expected string
	}{
		{"Zero progress", 0, 100, 10, "[░░░░░░░░░░] 0.0%"},
		{"Half progress", 50, 100, 10, "[█████░░░░░] 50.0%"},
		{"Full progress", 100, 100, 10, "[██████████] 100.0%"},
		{"Unknown total", 25, 0, 10, "[??????????]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateProgressBar(tt.current, tt.total, tt.width)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}