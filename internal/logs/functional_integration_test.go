package logs

import (
	"testing"
	"time"
)

// TestFunctionalGroupingIntegration tests that the functional error grouping
// works correctly when integrated with the Store
func TestFunctionalGroupingIntegration(t *testing.T) {
	store := NewStore(100, nil)
	defer store.Close()

	// Add some error logs that should be grouped
	store.Add("process1", "frontend", "TypeError: Cannot read property 'foo'", true)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	store.Add("process1", "frontend", "    at Object.render (/app/src/component.js:45:12)", true)
	time.Sleep(1 * time.Millisecond)
	store.Add("process1", "frontend", "    at Component.componentDidMount (/app/src/component.js:20:8)", true)

	// Add some logs from different process (should be separate group)
	time.Sleep(1 * time.Millisecond)
	store.Add("process2", "backend", "MongoDB connection failed", true)

	// Add another error group from process1 after a gap
	time.Sleep(300 * time.Millisecond) // Larger gap to create separate group
	store.Add("process1", "frontend", "SyntaxError: Unexpected token", true)

	// Wait for async processing to complete
	time.Sleep(10 * time.Millisecond)

	// Get error contexts using functional grouping
	contexts := store.GetErrorContexts()

	// Should have 3 groups:
	// 1. TypeError + stack traces from process1
	// 2. MongoDB error from process2
	// 3. SyntaxError from process1 (after gap)
	if len(contexts) != 3 {
		t.Errorf("Expected 3 error groups, got %d", len(contexts))
		for i, ctx := range contexts {
			t.Logf("Group %d: %s - %s (%d entries)", i, ctx.ProcessID, ctx.Type, len(ctx.Raw))
		}
		return
	}

	// Verify first group (TypeError with stack)
	group1 := contexts[0]
	if group1.Type != "TypeError" {
		t.Errorf("Expected first group to be TypeError, got %s", group1.Type)
	}
	if group1.ProcessID != "process1" {
		t.Errorf("Expected first group from process1, got %s", group1.ProcessID)
	}
	if len(group1.Raw) != 3 {
		t.Errorf("Expected first group to have 3 entries, got %d", len(group1.Raw))
	}

	// Verify second group (MongoDB error)
	group2 := contexts[1]
	if group2.Type != "MongoError" {
		t.Errorf("Expected second group to be MongoError, got %s", group2.Type)
	}
	if group2.ProcessID != "process2" {
		t.Errorf("Expected second group from process2, got %s", group2.ProcessID)
	}
	if len(group2.Raw) != 1 {
		t.Errorf("Expected second group to have 1 entry, got %d", len(group2.Raw))
	}

	// Verify third group (SyntaxError)
	group3 := contexts[2]
	if group3.Type != "SyntaxError" {
		t.Errorf("Expected third group to be SyntaxError, got %s", group3.Type)
	}
	if group3.ProcessID != "process1" {
		t.Errorf("Expected third group from process1, got %s", group3.ProcessID)
	}
	if len(group3.Raw) != 1 {
		t.Errorf("Expected third group to have 1 entry, got %d", len(group3.Raw))
	}

	// Verify groups are sorted by time
	if !group1.Timestamp.Before(group2.Timestamp) {
		t.Error("Groups should be sorted by timestamp")
	}
	if !group2.Timestamp.Before(group3.Timestamp) {
		t.Error("Groups should be sorted by timestamp")
	}
}

// TestFunctionalGroupingOnDemandGeneration tests that error contexts are generated
// on-demand and reflect the current state of log entries
func TestFunctionalGroupingOnDemandGeneration(t *testing.T) {
	store := NewStore(100, nil)
	defer store.Close()

	// Initially no errors
	contexts := store.GetErrorContexts()
	if len(contexts) != 0 {
		t.Errorf("Expected 0 error contexts initially, got %d", len(contexts))
	}

	// Add an error
	store.Add("process1", "test", "Error: Something went wrong", true)
	time.Sleep(5 * time.Millisecond)

	// Should now have 1 error context
	contexts = store.GetErrorContexts()
	if len(contexts) != 1 {
		t.Errorf("Expected 1 error context after adding error, got %d", len(contexts))
	}

	// Clear errors
	store.ClearErrors()

	// Should still show the error context since it's generated from entries, not the errors slice
	contexts = store.GetErrorContexts()
	if len(contexts) != 1 {
		t.Errorf("Expected error context to persist after ClearErrors (generated from entries), got %d", len(contexts))
	}

	// Clear all logs
	store.ClearLogs()

	// Now should have no error contexts
	contexts = store.GetErrorContexts()
	if len(contexts) != 0 {
		t.Errorf("Expected 0 error contexts after clearing logs, got %d", len(contexts))
	}
}

// TestFunctionalGroupingNonErrorFiltering tests that non-error entries are filtered out
func TestFunctionalGroupingNonErrorFiltering(t *testing.T) {
	store := NewStore(100, nil)
	defer store.Close()

	// Add mix of error and non-error entries
	store.Add("process1", "test", "Info: Starting application", false)
	store.Add("process1", "test", "Error: Configuration missing", true)
	store.Add("process1", "test", "Debug: Loading modules", false)
	store.Add("process1", "test", "Warning: Deprecated API used", false)

	time.Sleep(5 * time.Millisecond)

	// Should only group the error entry
	contexts := store.GetErrorContexts()
	if len(contexts) != 1 {
		t.Errorf("Expected 1 error context (only the error entry), got %d", len(contexts))
	}

	if contexts[0].Type != "Error" {
		t.Errorf("Expected error type 'Error', got %s", contexts[0].Type)
	}

	if len(contexts[0].Raw) != 1 {
		t.Errorf("Expected 1 raw entry in error context, got %d", len(contexts[0].Raw))
	}
}
