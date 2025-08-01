package process

import (
	"sync"
	"testing"
	"time"
)

// BenchmarkProcessGettersVsSnapshot compares individual getters vs ProcessSnapshot
func BenchmarkProcessGettersVsSnapshot(b *testing.B) {
	// Create a process for testing
	proc := &Process{
		ID:        "benchmark-test",
		Name:      "benchmark",
		Script:    "echo test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		EndTime:   nil,
		ExitCode:  nil,
	}

	b.Run("IndividualGetters", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate accessing multiple fields individually
			_ = proc.GetStatus()
			_ = proc.GetStartTime()
			_ = proc.GetEndTime()
			_ = proc.GetExitCode()
			_ = proc.ID
			_ = proc.Name
		}
	})

	b.Run("ProcessSnapshot", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Access all fields with single snapshot
			snapshot := proc.GetSnapshot()
			_ = snapshot.Status
			_ = snapshot.StartTime
			_ = snapshot.EndTime
			_ = snapshot.ExitCode
			_ = snapshot.ID
			_ = snapshot.Name
		}
	})
}

// BenchmarkConcurrentAccess benchmarks concurrent access patterns
func BenchmarkConcurrentAccess(b *testing.B) {
	proc := &Process{
		ID:        "concurrent-test",
		Name:      "concurrent",
		Script:    "echo test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		EndTime:   nil,
		ExitCode:  nil,
	}

	numGoroutines := 10

	b.Run("ConcurrentIndividualGetters", func(b *testing.B) {
		b.ResetTimer()
		var wg sync.WaitGroup

		for i := 0; i < b.N; i++ {
			wg.Add(numGoroutines)
			for j := 0; j < numGoroutines; j++ {
				go func() {
					defer wg.Done()
					// Multiple lock acquisitions
					_ = proc.GetStatus()
					_ = proc.GetStartTime()
					_ = proc.GetEndTime()
					_ = proc.GetExitCode()
				}()
			}
			wg.Wait()
		}
	})

	b.Run("ConcurrentProcessSnapshot", func(b *testing.B) {
		b.ResetTimer()
		var wg sync.WaitGroup

		for i := 0; i < b.N; i++ {
			wg.Add(numGoroutines)
			for j := 0; j < numGoroutines; j++ {
				go func() {
					defer wg.Done()
					// Single lock acquisition
					snapshot := proc.GetSnapshot()
					_ = snapshot.Status
					_ = snapshot.StartTime
					_ = snapshot.EndTime
					_ = snapshot.ExitCode
				}()
			}
			wg.Wait()
		}
	})
}

// BenchmarkProcessSnapshotMethods benchmarks ProcessSnapshot convenience methods
func BenchmarkProcessSnapshotMethods(b *testing.B) {
	proc := &Process{
		ID:        "methods-test",
		Name:      "methods",
		Script:    "echo test",
		Status:    StatusRunning,
		StartTime: time.Now().Add(-5 * time.Minute),
		EndTime:   nil,
		ExitCode:  nil,
	}

	b.Run("IsRunning", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			snapshot := proc.GetSnapshot()
			_ = snapshot.IsRunning()
		}
	})

	b.Run("IsFinished", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			snapshot := proc.GetSnapshot()
			_ = snapshot.IsFinished()
		}
	})

	b.Run("Duration", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			snapshot := proc.GetSnapshot()
			_ = snapshot.Duration()
		}
	})
}

// BenchmarkLockContention benchmarks lock contention scenarios
func BenchmarkLockContention(b *testing.B) {
	proc := &Process{
		ID:        "contention-test",
		Name:      "contention",
		Script:    "echo test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		EndTime:   nil,
		ExitCode:  nil,
	}

	// High contention scenario: many readers, few writers
	b.Run("HighContentionIndividualGetters", func(b *testing.B) {
		b.ResetTimer()
		var wg sync.WaitGroup

		// Start a writer goroutine that modifies process state
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N/100; i++ {
				proc.SetStatus(StatusRunning)
				time.Sleep(time.Microsecond)
			}
		}()

		// Many reader goroutines using individual getters
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < b.N/50; j++ {
					_ = proc.GetStatus()
					_ = proc.GetStartTime()
					_ = proc.GetEndTime()
					_ = proc.GetExitCode()
				}
			}()
		}

		wg.Wait()
	})

	b.Run("HighContentionProcessSnapshot", func(b *testing.B) {
		b.ResetTimer()
		var wg sync.WaitGroup

		// Start a writer goroutine that modifies process state
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N/100; i++ {
				proc.SetStatus(StatusRunning)
				time.Sleep(time.Microsecond)
			}
		}()

		// Many reader goroutines using ProcessSnapshot
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < b.N/50; j++ {
					snapshot := proc.GetSnapshot()
					_ = snapshot.Status
					_ = snapshot.StartTime
					_ = snapshot.EndTime
					_ = snapshot.ExitCode
				}
			}()
		}

		wg.Wait()
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	proc := &Process{
		ID:        "memory-test",
		Name:      "memory",
		Script:    "echo test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		EndTime:   nil,
		ExitCode:  nil,
	}

	b.Run("IndividualGettersMemory", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Individual getters return values directly
			status := proc.GetStatus()
			startTime := proc.GetStartTime()
			endTime := proc.GetEndTime()
			exitCode := proc.GetExitCode()

			// Use values to prevent optimization
			_ = status
			_ = startTime
			_ = endTime
			_ = exitCode
		}
	})

	b.Run("ProcessSnapshotMemory", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// ProcessSnapshot creates a struct
			snapshot := proc.GetSnapshot()

			// Use values to prevent optimization
			_ = snapshot.Status
			_ = snapshot.StartTime
			_ = snapshot.EndTime
			_ = snapshot.ExitCode
		}
	})
}
