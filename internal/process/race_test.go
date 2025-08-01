package process

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessGettersRaceCondition tests that the getter methods are thread-safe
func TestProcessGettersRaceCondition(t *testing.T) {
	// Create a process with properly initialized atomic state
	proc := &Process{
		ID:        "test-123",
		Name:      "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		EndTime:   nil,
		ExitCode:  nil,
		mu:        sync.RWMutex{},
	}

	// Initialize atomic state
	initialState := ProcessState{
		ID:        proc.ID,
		Name:      proc.Name,
		Script:    "",
		Status:    proc.Status,
		StartTime: proc.StartTime,
		EndTime:   proc.EndTime,
		ExitCode:  proc.ExitCode,
	}
	atomic.StorePointer(&proc.atomicState, unsafe.Pointer(&initialState))

	// Number of concurrent readers
	numReaders := 100
	numIterations := 1000

	var wg sync.WaitGroup
	wg.Add(numReaders + 1) // +1 for writer

	// Error channel
	errors := make(chan error, numReaders*numIterations)

	// Start concurrent readers
	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				// Use atomic state reader for consistent multi-field access
				state := proc.GetStateAtomic()

				// Validate individual fields
				if state.Status == "" {
					errors <- assert.AnError
				}
				if state.StartTime.IsZero() {
					errors <- assert.AnError
				}

				// Validate consistency within atomic state
				if state.Status == StatusStopped && state.EndTime == nil {
					errors <- assert.AnError
				}
				if state.Status == StatusFailed && state.ExitCode == nil {
					errors <- assert.AnError
				}
			}
		}(i)
	}

	// Start a writer that modifies the process state using atomic operations
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			// Simulate state transitions using atomic operations
			proc.UpdateStateAtomic(func(state ProcessState) ProcessState {
				switch state.Status {
				case StatusRunning:
					now := time.Now()
					return state.CopyWithStatus(StatusStopped).CopyWithEndTime(now)
				case StatusStopped:
					return state.CopyWithExit(1) // This sets StatusFailed
				case StatusFailed:
					newState := state
					newState.Status = StatusRunning
					newState.EndTime = nil
					newState.ExitCode = nil
					return newState
				default:
					return state
				}
			})
			time.Sleep(time.Microsecond * 100)
		}
	}()

	// Wait for completion
	done := make(chan bool)
	go func() {
		wg.Wait()
		close(errors)
		done <- true
	}()

	// Timeout after 5 seconds
	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}

	// Check for errors
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
		}
	}

	assert.Equal(t, 0, errorCount, "Race condition detected")
}

// TestManagerConcurrentScriptsStatus tests the scripts_status pattern for race conditions
func TestManagerConcurrentScriptsStatus(t *testing.T) {
	// Create dependencies
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	// Create manager
	mgr, err := NewManager(t.TempDir(), eventBus, false)
	require.NoError(t, err)

	// Add some mock processes
	for i := 0; i < 5; i++ {
		proc := &Process{
			ID:        fmt.Sprintf("test-%d", i),
			Name:      fmt.Sprintf("script-%d", i),
			Status:    StatusRunning,
			StartTime: time.Now(),
			Cmd:       nil, // Mock process
		}
		mgr.processes.Store(proc.ID, proc)
	}

	// Number of concurrent readers
	numReaders := 50
	numIterations := 100

	var wg sync.WaitGroup
	wg.Add(numReaders)

	// Start concurrent readers (simulating scripts_status calls)
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				// Get all processes
				processes := mgr.GetAllProcesses()

				// Access fields through getters (simulating what scripts_status does)
				for _, proc := range processes {
					_ = proc.ID
					_ = proc.Name
					_ = proc.GetStatus()
					_ = proc.GetStartTime()
					_ = proc.GetEndTime()
					_ = proc.GetExitCode()
				}
			}
		}()
	}

	// Also have a writer that modifies process states
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			processes := mgr.GetAllProcesses()
			if len(processes) > 0 {
				// Pick a process to modify
				proc := processes[i%len(processes)]

				// Lock and modify
				proc.mu.Lock()
				if proc.Status == StatusRunning {
					proc.Status = StatusStopped
					now := time.Now()
					proc.EndTime = &now
					code := 0
					proc.ExitCode = &code
				} else {
					proc.Status = StatusRunning
					proc.EndTime = nil
					proc.ExitCode = nil
				}
				proc.mu.Unlock()
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// Wait for completion
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	// Timeout after 10 seconds
	select {
	case <-done:
		t.Log("Concurrent access test completed successfully")
	case <-time.After(10 * time.Second):
		t.Fatal("Test timed out - possible deadlock")
	}
}
