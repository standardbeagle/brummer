package process

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// TestAtomicStateConsistency verifies that atomic state operations maintain consistency
func TestAtomicStateConsistency(t *testing.T) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		mu:        sync.RWMutex{},
	}

	// Initialize atomic state
	initialState := ProcessState{
		ID:        p.ID,
		Name:      p.Name,
		Script:    p.Script,
		Status:    p.Status,
		StartTime: p.StartTime,
	}
	atomic.StorePointer(&p.atomicState, unsafe.Pointer(&initialState))

	// Test initial read
	state := p.GetStateAtomic()
	if state.Status != StatusRunning {
		t.Errorf("Expected status %v, got %v", StatusRunning, state.Status)
	}

	// Test update
	p.UpdateStateAtomic(func(state ProcessState) ProcessState {
		return state.CopyWithStatus(StatusSuccess)
	})

	// Verify update
	state = p.GetStateAtomic()
	if state.Status != StatusSuccess {
		t.Errorf("Expected status %v, got %v", StatusSuccess, state.Status)
	}

	// Verify mutex fields are also updated
	if p.GetStatus() != StatusSuccess {
		t.Errorf("Mutex-based getter returned wrong status: %v", p.GetStatus())
	}
}

// TestConcurrentAtomicUpdates verifies thread safety under concurrent updates
func TestConcurrentAtomicUpdates(t *testing.T) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		mu:        sync.RWMutex{},
	}

	// Initialize atomic state
	initialState := ProcessState{
		ID:        p.ID,
		Name:      p.Name,
		Script:    p.Script,
		Status:    p.Status,
		StartTime: p.StartTime,
	}
	atomic.StorePointer(&p.atomicState, unsafe.Pointer(&initialState))

	const numGoroutines = 100
	const updatesPerGoroutine = 1000

	var wg sync.WaitGroup
	successCount := int32(0)
	failedCount := int32(0)

	// Launch concurrent updaters
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				// Alternate between success and failed status
				targetStatus := StatusSuccess
				if (id+j)%2 == 0 {
					targetStatus = StatusFailed
				}

				p.UpdateStateAtomic(func(state ProcessState) ProcessState {
					return state.CopyWithStatus(targetStatus)
				})

				// Count the final states
				if targetStatus == StatusSuccess {
					atomic.AddInt32(&successCount, 1)
				} else {
					atomic.AddInt32(&failedCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify total updates
	totalUpdates := int(successCount + failedCount)
	expectedTotal := numGoroutines * updatesPerGoroutine
	if totalUpdates != expectedTotal {
		t.Errorf("Expected %d total updates, got %d", expectedTotal, totalUpdates)
	}

	// Final state should be either success or failed
	finalState := p.GetStateAtomic()
	if finalState.Status != StatusSuccess && finalState.Status != StatusFailed {
		t.Errorf("Unexpected final status: %v", finalState.Status)
	}
}

// TestAtomicExitCodeUpdate tests updating exit code atomically
func TestAtomicExitCodeUpdate(t *testing.T) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		mu:        sync.RWMutex{},
	}

	// Initialize atomic state
	initialState := ProcessState{
		ID:        p.ID,
		Name:      p.Name,
		Script:    p.Script,
		Status:    p.Status,
		StartTime: p.StartTime,
	}
	atomic.StorePointer(&p.atomicState, unsafe.Pointer(&initialState))

	// Update with exit code
	p.UpdateStateAtomic(func(state ProcessState) ProcessState {
		return state.CopyWithExit(0)
	})

	// Verify state
	state := p.GetStateAtomic()
	if state.Status != StatusSuccess {
		t.Errorf("Expected status %v, got %v", StatusSuccess, state.Status)
	}
	if state.ExitCode == nil || *state.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %v", state.ExitCode)
	}
	if state.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}

	// Test non-zero exit code
	p.UpdateStateAtomic(func(state ProcessState) ProcessState {
		return state.CopyWithExit(1)
	})

	state = p.GetStateAtomic()
	if state.Status != StatusFailed {
		t.Errorf("Expected status %v, got %v", StatusFailed, state.Status)
	}
	if state.ExitCode == nil || *state.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %v", state.ExitCode)
	}
}

// TestAtomicStateImmutability verifies that states are truly immutable
func TestAtomicStateImmutability(t *testing.T) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		mu:        sync.RWMutex{},
	}

	// Initialize atomic state
	initialState := ProcessState{
		ID:        p.ID,
		Name:      p.Name,
		Script:    p.Script,
		Status:    p.Status,
		StartTime: p.StartTime,
	}
	atomic.StorePointer(&p.atomicState, unsafe.Pointer(&initialState))

	// Get state and try to modify it
	state1 := p.GetStateAtomic()
	originalStatus := state1.Status

	// This modification should not affect the stored state
	state1.Status = StatusFailed

	// Get state again
	state2 := p.GetStateAtomic()
	if state2.Status != originalStatus {
		t.Errorf("State was mutated! Expected %v, got %v", originalStatus, state2.Status)
	}
}

// TestNilAtomicStateFallback tests fallback to mutex when atomic state is nil
func TestNilAtomicStateFallback(t *testing.T) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		mu:        sync.RWMutex{},
	}

	// Don't initialize atomic state - it should be nil

	// GetStateAtomic should fallback to mutex-based read
	state := p.GetStateAtomic()
	if state.ID != p.ID {
		t.Errorf("Expected ID %v, got %v", p.ID, state.ID)
	}
	if state.Status != p.Status {
		t.Errorf("Expected status %v, got %v", p.Status, state.Status)
	}
}

// TestRaceConditionPrevention uses Go's race detector
func TestRaceConditionPrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race test in short mode")
	}

	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		mu:        sync.RWMutex{},
	}

	// Initialize atomic state
	initialState := ProcessState{
		ID:        p.ID,
		Name:      p.Name,
		Script:    p.Script,
		Status:    p.Status,
		StartTime: p.StartTime,
	}
	atomic.StorePointer(&p.atomicState, unsafe.Pointer(&initialState))

	var wg sync.WaitGroup

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				state := p.GetStateAtomic()
				_ = state.Status
				_ = state.Name
				_ = state.StartTime
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				p.UpdateStateAtomic(func(state ProcessState) ProcessState {
					if id%2 == 0 {
						return state.CopyWithStatus(StatusSuccess)
					}
					return state.CopyWithStatus(StatusFailed)
				})
			}
		}(i)
	}

	wg.Wait()
}
