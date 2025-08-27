package components

import (
	"fmt"
	"testing"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogStoreBasics tests basic log store functionality
func TestLogStoreBasics(t *testing.T) {
	// Create log store
	logStore := logs.NewStore(MaxLogEntries, nil)
	require.NotNil(t, logStore)
	
	// Add some logs
	logStore.Add("proc1", "Process 1", "First log", false)
	logStore.Add("proc1", "Process 1", "Second log", false)
	logStore.Add("proc2", "Process 2", "Third log", true)
	
	// Wait for async processing
	WaitForAsync()
	
	// Get all logs
	allLogs := logStore.GetAll()
	assert.Equal(t, 3, len(allLogs), "Should have 3 logs")
	
	// Get collapsed logs
	collapsed := logStore.GetAllCollapsed()
	t.Logf("Collapsed logs count: %d", len(collapsed))
	for i, entry := range collapsed {
		t.Logf("Collapsed[%d]: ProcessID=%s, Content=%s, Count=%d", 
			i, entry.ProcessID, entry.Content, entry.Count)
	}
	
	// Get by process
	proc1Logs := logStore.GetByProcess("proc1")
	assert.Equal(t, 2, len(proc1Logs), "Process 1 should have 2 logs")
	
	proc2Logs := logStore.GetByProcess("proc2")
	assert.Equal(t, 1, len(proc2Logs), "Process 2 should have 1 log")
}

// TestLogsViewInitialization tests that LogsViewComponent initializes properly
func TestLogsViewInitialization(t *testing.T) {
	// Initialize colors
	MustInitColors()
	
	// Create log store with logs
	logStore := logs.NewStore(MaxLogEntries, nil)
	logStore.Add("proc1", "Process 1", "First log line", false)
	logStore.Add("proc1", "Process 1", "Second log line", false)
	logStore.Add("proc2", "Process 2", "Third log line", true)
	
	// Wait for async processing
	WaitForAsync()
	
	// Create logs view
	logsView := NewLogsViewComponent(logStore)
	require.NotNil(t, logsView)
	
	// Check initial state
	assert.NotNil(t, logsView.logStore, "Log store should be set")
	assert.NotNil(t, logsView.viewport, "Viewport should be initialized")
	assert.True(t, logsView.visible, "Should be visible by default")
	assert.True(t, logsView.following, "Should be following by default")
	
	// Check filtered logs
	t.Logf("Filtered logs count: %d", len(logsView.filteredLogs))
	for i, log := range logsView.filteredLogs {
		t.Logf("FilteredLog[%d]: %s", i, log)
	}
	
	// Set size to ensure proper rendering
	logsView.SetSize(DefaultTestWidth, DefaultTestHeight)
	
	// Get the view
	output := logsView.View()
	t.Logf("View output length: %d", len(output))
	
	// Check what's in the viewport content
	viewportContent := logsView.viewport.View()
	t.Logf("Viewport content: %s", viewportContent)
	
	// If no logs are showing, check why
	if len(logsView.filteredLogs) == 0 {
		t.Log("No filtered logs found!")
		
		// Check if logs are in the store
		allLogs := logStore.GetAll()
		t.Logf("Store has %d logs", len(allLogs))
		
		// Check collapsed logs
		collapsed := logStore.GetAllCollapsed()
		t.Logf("Store has %d collapsed logs", len(collapsed))
	}
}

// TestLogsViewApplyFilter tests the applyFilter method directly
func TestLogsViewApplyFilter(t *testing.T) {
	// Initialize colors
	MustInitColors()
	
	// Create log store
	logStore := logs.NewStore(MaxLogEntries, nil)
	logStore.Add("proc1", "Process 1", "First log line", false)
	logStore.Add("proc1", "Process 1", "Second log line", false)
	logStore.Add("proc2", "Process 2", "Error: something failed", true)
	
	// Wait for async processing
	WaitForAsync()
	
	// Create logs view
	logsView := NewLogsViewComponent(logStore)
	
	// Manually call applyFilter
	logsView.applyFilter()
	
	// Check filtered logs
	assert.Greater(t, len(logsView.filteredLogs), 0, 
		fmt.Sprintf("Should have filtered logs, got %d", len(logsView.filteredLogs)))
	
	// Update viewport
	logsView.updateViewport()
	
	// Check viewport content
	content := logsView.viewport.View()
	t.Logf("Viewport content after update: %s", content)
	
	// The content should not be the "No logs available" message
	assert.NotContains(t, content, "No logs available", 
		"Should not show 'No logs available' when logs exist")
}