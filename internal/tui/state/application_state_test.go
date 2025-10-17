package state

import (
	"testing"
	"time"
)

// TestApplicationStateCreation tests creating a new application state
func TestApplicationStateCreation(t *testing.T) {
	state := NewApplicationState()

	if state == nil {
		t.Fatal("Expected non-nil application state")
	}

	if state.GetVersion() != 0 {
		t.Errorf("Expected initial version 0, got %d", state.GetVersion())
	}
}

// TestApplicationStateUpdate tests updating application state
func TestApplicationStateUpdate(t *testing.T) {
	state := NewApplicationState()

	// Update state with test data
	data := map[string]interface{}{
		"test_key": "test_value",
		"number":   42,
	}

	state.Update(data)

	// Check version was incremented
	if state.GetVersion() != 1 {
		t.Errorf("Expected version 1 after update, got %d", state.GetVersion())
	}

	// Check data was stored
	storedValue := state.Get("test_key")
	if storedValue != "test_value" {
		t.Errorf("Expected 'test_value', got %v", storedValue)
	}

	storedNumber := state.Get("number")
	if storedNumber != 42 {
		t.Errorf("Expected 42, got %v", storedNumber)
	}
}

// TestApplicationStateSnapshot tests creating state snapshots
func TestApplicationStateSnapshot(t *testing.T) {
	state := NewApplicationState()

	// Add some data
	state.Update(map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	})

	// Create snapshot
	snapshot := state.CreateSnapshot()

	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}

	// Check snapshot contains data
	if snapshot.Version != 1 {
		t.Errorf("Expected snapshot version 1, got %d", snapshot.Version)
	}

	if snapshot.Data["key1"] != "value1" {
		t.Errorf("Expected snapshot key1 to be 'value1', got %v", snapshot.Data["key1"])
	}
}

// TestApplicationStateRestore tests restoring state from snapshot
func TestApplicationStateRestore(t *testing.T) {
	state := NewApplicationState()

	// Create snapshot data
	snapshot := &StateSnapshot{
		Version: 5,
		Data: map[string]interface{}{
			"restored_key": "restored_value",
			"count":        99,
		},
		Timestamp: time.Now(),
	}

	// Restore from snapshot
	err := state.RestoreSnapshot(snapshot)

	if err != nil {
		t.Errorf("Expected no error restoring snapshot, got %v", err)
	}

	// Check state was restored
	if state.GetVersion() != 5 {
		t.Errorf("Expected version 5 after restore, got %d", state.GetVersion())
	}

	restoredValue := state.Get("restored_key")
	if restoredValue != "restored_value" {
		t.Errorf("Expected 'restored_value', got %v", restoredValue)
	}
}

// TestApplicationStateReset tests resetting application state
func TestApplicationStateReset(t *testing.T) {
	state := NewApplicationState()

	// Add some data
	state.Update(map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	})

	// Reset state
	state.Reset()

	// Check state was reset
	if state.GetVersion() != 0 {
		t.Errorf("Expected version 0 after reset, got %d", state.GetVersion())
	}

	value := state.Get("key1")
	if value != nil {
		t.Errorf("Expected nil value after reset, got %v", value)
	}
}

// TestApplicationStateConcurrency tests concurrent access to state
func TestApplicationStateConcurrency(t *testing.T) {
	state := NewApplicationState()

	// Start multiple goroutines updating state
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			for j := 0; j < 100; j++ {
				state.Update(map[string]interface{}{
					"key": index*100 + j,
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check final state
	finalVersion := state.GetVersion()
	if finalVersion != 1000 { // 10 goroutines * 100 updates each
		t.Errorf("Expected final version 1000, got %d", finalVersion)
	}
}

// TestApplicationStatePerformance tests state update performance
func TestApplicationStatePerformance(t *testing.T) {
	state := NewApplicationState()

	// Measure time for 1000 updates
	start := time.Now()

	for i := 0; i < 1000; i++ {
		state.Update(map[string]interface{}{
			"iteration": i,
			"data":      "some test data that might be typical",
		})
	}

	duration := time.Since(start)

	// Should complete quickly (adjust threshold as needed)
	if duration > 100*time.Millisecond {
		t.Errorf("State updates took too long: %v", duration)
	}

	t.Logf("1000 state updates took %v", duration)
}

// TestStateSnapshotValidation tests snapshot validation
func TestStateSnapshotValidation(t *testing.T) {
	state := NewApplicationState()

	// Test invalid snapshot
	invalidSnapshot := &StateSnapshot{
		Version: -1,
		Data:    nil,
	}

	err := state.RestoreSnapshot(invalidSnapshot)
	if err == nil {
		t.Error("Expected error for invalid snapshot")
	}

	// Test nil snapshot
	err = state.RestoreSnapshot(nil)
	if err == nil {
		t.Error("Expected error for nil snapshot")
	}
}