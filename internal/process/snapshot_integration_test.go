package process

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessSnapshotAtomicConsistency tests that ProcessSnapshot provides atomic access
func TestProcessSnapshotAtomicConsistency(b *testing.T) {
	proc := &Process{
		ID:        "atomic-test",
		Name:      "atomic",
		Script:    "echo test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		EndTime:   nil,
		ExitCode:  nil,
	}

	const numReaders = 100
	const numIterations = 1000
	var inconsistencyCount int32

	var wg sync.WaitGroup
	wg.Add(numReaders + 1) // +1 for writer

	// Start a writer that modifies process state
	go func() {
		defer wg.Done()
		for i := 0; i < numIterations; i++ {
			proc.mu.Lock()
			// Simulate state transition
			switch proc.Status {
			case StatusRunning:
				proc.Status = StatusStopped
				now := time.Now()
				proc.EndTime = &now
				code := 0
				proc.ExitCode = &code
			case StatusStopped:
				proc.Status = StatusFailed
				code := 1
				proc.ExitCode = &code
			case StatusFailed:
				proc.Status = StatusRunning
				proc.EndTime = nil
				proc.ExitCode = nil
			}
			proc.mu.Unlock()
			time.Sleep(time.Microsecond)
		}
	}()

	// Start readers that check consistency
	for i := 0; i < numReaders; i++ {
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < numIterations/10; j++ {
				snapshot := proc.GetSnapshot()

				// Check consistency rules
				if snapshot.Status == StatusRunning {
					// Running processes should not have EndTime or ExitCode
					if snapshot.EndTime != nil || snapshot.ExitCode != nil {
						inconsistencyCount++
						b.Errorf("Reader %d: Running process has EndTime=%v or ExitCode=%v",
							readerID, snapshot.EndTime, snapshot.ExitCode)
					}
				} else if snapshot.Status == StatusStopped || snapshot.Status == StatusFailed {
					// Finished processes should have EndTime and ExitCode
					if snapshot.EndTime == nil {
						inconsistencyCount++
						b.Errorf("Reader %d: Finished process (%s) missing EndTime",
							readerID, snapshot.Status)
					}
					if snapshot.ExitCode == nil {
						inconsistencyCount++
						b.Errorf("Reader %d: Finished process (%s) missing ExitCode",
							readerID, snapshot.Status)
					}
				}

				// Verify snapshot convenience methods work correctly
				isRunning := snapshot.IsRunning()
				isFinished := snapshot.IsFinished()

				if isRunning && isFinished {
					inconsistencyCount++
					b.Errorf("Reader %d: Process cannot be both running and finished", readerID)
				}

				if !isRunning && !isFinished && snapshot.Status != StatusPending {
					inconsistencyCount++
					b.Errorf("Reader %d: Process must be either running, finished, or pending", readerID)
				}
			}
		}(i)
	}

	wg.Wait()

	if inconsistencyCount > 0 {
		b.Errorf("Found %d consistency violations", inconsistencyCount)
	}
}

// TestProcessSnapshotVsIndividualGetters compares consistency between approaches
func TestProcessSnapshotVsIndividualGetters(t *testing.T) {
	proc := &Process{
		ID:        "compare-test",
		Name:      "compare",
		Script:    "echo test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		EndTime:   nil,
		ExitCode:  nil,
	}

	const numTests = 1000
	var snapshotInconsistencies int
	var getterInconsistencies int

	for i := 0; i < numTests; i++ {
		// Test ProcessSnapshot approach
		go func() {
			proc.mu.Lock()
			// Simulate state transition
			if proc.Status == StatusRunning {
				proc.Status = StatusStopped
				now := time.Now()
				proc.EndTime = &now
				code := 0
				proc.ExitCode = &code
			}
			proc.mu.Unlock()
		}()

		// Immediately read with ProcessSnapshot (atomic)
		snapshot := proc.GetSnapshot()
		if snapshot.Status == StatusRunning && (snapshot.EndTime != nil || snapshot.ExitCode != nil) {
			snapshotInconsistencies++
		}

		// Reset for next test
		proc.mu.Lock()
		proc.Status = StatusRunning
		proc.EndTime = nil
		proc.ExitCode = nil
		proc.mu.Unlock()

		// Test individual getters approach (non-atomic)
		go func() {
			proc.mu.Lock()
			// Simulate state transition
			if proc.Status == StatusRunning {
				proc.Status = StatusStopped
				now := time.Now()
				proc.EndTime = &now
				code := 0
				proc.ExitCode = &code
			}
			proc.mu.Unlock()
		}()

		// Immediately read with individual getters (potential race)
		status := proc.GetStatus()
		endTime := proc.GetEndTime()
		exitCode := proc.GetExitCode()

		if status == StatusRunning && (endTime != nil || exitCode != nil) {
			getterInconsistencies++
		}

		// Reset for next test
		proc.mu.Lock()
		proc.Status = StatusRunning
		proc.EndTime = nil
		proc.ExitCode = nil
		proc.mu.Unlock()

		time.Sleep(time.Microsecond) // Small delay to allow races
	}

	t.Logf("Snapshot inconsistencies: %d/%d", snapshotInconsistencies, numTests)
	t.Logf("Individual getter inconsistencies: %d/%d", getterInconsistencies, numTests)

	// ProcessSnapshot should have significantly fewer inconsistencies
	assert.LessOrEqual(t, snapshotInconsistencies, getterInconsistencies,
		"ProcessSnapshot should have fewer or equal inconsistencies compared to individual getters")
}

// TestProcessSnapshotMethods tests ProcessSnapshot convenience methods
func TestProcessSnapshotMethods(t *testing.T) {
	testCases := []struct {
		name           string
		status         ProcessStatus
		endTime        *time.Time
		exitCode       *int
		expectRunning  bool
		expectFinished bool
	}{
		{
			name:           "Running process",
			status:         StatusRunning,
			endTime:        nil,
			exitCode:       nil,
			expectRunning:  true,
			expectFinished: false,
		},
		{
			name:           "Pending process",
			status:         StatusPending,
			endTime:        nil,
			exitCode:       nil,
			expectRunning:  false,
			expectFinished: false,
		},
		{
			name:           "Stopped process",
			status:         StatusStopped,
			endTime:        timePtr(time.Now().Add(-2 * time.Minute)), // Ended 2 minutes ago
			exitCode:       intPtr(0),
			expectRunning:  false,
			expectFinished: true,
		},
		{
			name:           "Failed process",
			status:         StatusFailed,
			endTime:        timePtr(time.Now().Add(-1 * time.Minute)), // Ended 1 minute ago
			exitCode:       intPtr(1),
			expectRunning:  false,
			expectFinished: true,
		},
		{
			name:           "Success process",
			status:         StatusSuccess,
			endTime:        timePtr(time.Now()), // Just ended
			exitCode:       intPtr(0),
			expectRunning:  false,
			expectFinished: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			baseTime := time.Now().Add(-5 * time.Minute) // Start 5 minutes ago
			snapshot := ProcessSnapshot{
				ID:        "test",
				Name:      "test",
				Script:    "echo test",
				Status:    tc.status,
				StartTime: baseTime,
				EndTime:   tc.endTime,
				ExitCode:  tc.exitCode,
			}

			assert.Equal(t, tc.expectRunning, snapshot.IsRunning(), "IsRunning() result")
			assert.Equal(t, tc.expectFinished, snapshot.IsFinished(), "IsFinished() result")

			// Test Duration method
			duration := snapshot.Duration()
			assert.NotZero(t, duration, "Duration should not be zero")

			if tc.endTime != nil && !tc.endTime.IsZero() {
				expectedDuration := tc.endTime.Sub(snapshot.StartTime)
				assert.Equal(t, expectedDuration, duration, "Duration should match EndTime - StartTime")
			} else {
				// For running/pending processes, duration should be positive (time since start)
				assert.Positive(t, duration, "Duration should be positive for running processes")
			}
		})
	}
}

// TestProcessSnapshotStringRepresentation tests String() method
func TestProcessSnapshotStringRepresentation(t *testing.T) {
	snapshot := ProcessSnapshot{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "echo test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		EndTime:   nil,
		ExitCode:  nil,
	}

	str := snapshot.String()
	assert.Contains(t, str, "test-123", "String should contain process ID")
	assert.Contains(t, str, "test-process", "String should contain process name")
	assert.Contains(t, str, "running", "String should contain status")
}

// TestProcessSnapshotConcurrentAccess tests concurrent access to ProcessSnapshot
func TestProcessSnapshotConcurrentAccess(t *testing.T) {
	proc := &Process{
		ID:        "concurrent-snapshot-test",
		Name:      "concurrent-snapshot",
		Script:    "echo test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		EndTime:   nil,
		ExitCode:  nil,
	}

	const numGoroutines = 100
	const numIterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// All goroutines read concurrently using ProcessSnapshot
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				snapshot := proc.GetSnapshot()

				// Verify snapshot data is consistent
				require.NotEmpty(t, snapshot.ID, "Snapshot ID should not be empty")
				require.NotEmpty(t, snapshot.Name, "Snapshot Name should not be empty")
				require.NotZero(t, snapshot.StartTime, "Snapshot StartTime should not be zero")

				// Use snapshot methods
				_ = snapshot.IsRunning()
				_ = snapshot.IsFinished()
				_ = snapshot.Duration()
				_ = snapshot.String()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("Test timed out - possible deadlock in ProcessSnapshot")
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
