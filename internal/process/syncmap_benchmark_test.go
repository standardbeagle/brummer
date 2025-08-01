package process

import (
	"sync"
	"testing"
	"time"
)

// BenchmarkMapMutexRead tests current map+mutex read performance
func BenchmarkMapMutexRead(b *testing.B) {
	// Setup: Manager with processes
	processes := make(map[string]*Process)
	mu := sync.RWMutex{}

	// Add test processes
	for i := 0; i < 100; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		processes[processID] = &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate GetProcess operation
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))

		mu.RLock()
		_, exists := processes[processID]
		mu.RUnlock()
		_ = exists
	}
}

// BenchmarkSyncMapRead tests sync.Map read performance
func BenchmarkSyncMapRead(b *testing.B) {
	// Setup: sync.Map with processes
	var processes sync.Map

	// Add test processes
	for i := 0; i < 100; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		process := &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}
		processes.Store(processID, process)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate GetProcess operation
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))
		_, exists := processes.Load(processID)
		_ = exists
	}
}

// BenchmarkConcurrentMapMutexRead tests concurrent reads with map+mutex
func BenchmarkConcurrentMapMutexRead(b *testing.B) {
	processes := make(map[string]*Process)
	mu := sync.RWMutex{}

	// Add test processes
	for i := 0; i < 100; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		processes[processID] = &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))

			mu.RLock()
			_, exists := processes[processID]
			mu.RUnlock()
			_ = exists
			i++
		}
	})
}

// BenchmarkConcurrentSyncMapRead tests concurrent reads with sync.Map
func BenchmarkConcurrentSyncMapRead(b *testing.B) {
	var processes sync.Map

	// Add test processes
	for i := 0; i < 100; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		process := &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}
		processes.Store(processID, process)
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))
			_, exists := processes.Load(processID)
			_ = exists
			i++
		}
	})
}

// BenchmarkMapMutexWrite tests map+mutex write performance
func BenchmarkMapMutexWrite(b *testing.B) {
	processes := make(map[string]*Process)
	mu := sync.RWMutex{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))
		process := &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}

		mu.Lock()
		processes[processID] = process
		mu.Unlock()
	}
}

// BenchmarkSyncMapWrite tests sync.Map write performance
func BenchmarkSyncMapWrite(b *testing.B) {
	var processes sync.Map

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))
		process := &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}

		processes.Store(processID, process)
	}
}

// BenchmarkMapMutexDelete tests map+mutex delete performance
func BenchmarkMapMutexDelete(b *testing.B) {
	processes := make(map[string]*Process)
	mu := sync.RWMutex{}

	// Pre-populate for deletion tests
	for i := 0; i < b.N; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))
		processes[processID] = &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))

		mu.Lock()
		delete(processes, processID)
		mu.Unlock()
	}
}

// BenchmarkSyncMapDelete tests sync.Map delete performance
func BenchmarkSyncMapDelete(b *testing.B) {
	var processes sync.Map

	// Pre-populate for deletion tests
	for i := 0; i < b.N; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))
		processes.Store(processID, &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))
		processes.Delete(processID)
	}
}

// BenchmarkGetAllProcessesMapMutex tests iterating over all processes with map+mutex
func BenchmarkGetAllProcessesMapMutex(b *testing.B) {
	processes := make(map[string]*Process)
	mu := sync.RWMutex{}

	// Add test processes
	for i := 0; i < 100; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		processes[processID] = &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		result := make([]*Process, 0, len(processes))
		for _, p := range processes {
			result = append(result, p)
		}
		mu.RUnlock()
		_ = result
	}
}

// BenchmarkGetAllProcessesSyncMap tests iterating over all processes with sync.Map
func BenchmarkGetAllProcessesSyncMap(b *testing.B) {
	var processes sync.Map

	// Add test processes
	for i := 0; i < 100; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		process := &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}
		processes.Store(processID, process)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result []*Process
		processes.Range(func(key, value interface{}) bool {
			if process, ok := value.(*Process); ok {
				result = append(result, process)
			}
			return true
		})
		_ = result
	}
}

// BenchmarkMixedWorkloadMapMutex simulates real-world mixed read/write workload with map+mutex
func BenchmarkMixedWorkloadMapMutex(b *testing.B) {
	processes := make(map[string]*Process)
	mu := sync.RWMutex{}

	// Pre-populate with some processes
	for i := 0; i < 50; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		processes[processID] = &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))

			// 80% reads, 20% writes (realistic for process management)
			if i%5 == 0 {
				// Write operation
				process := &Process{
					ID:        processID,
					Name:      "test-process",
					Script:    "test",
					Status:    StatusRunning,
					StartTime: time.Now(),
				}
				mu.Lock()
				processes[processID] = process
				mu.Unlock()
			} else {
				// Read operation
				mu.RLock()
				_, exists := processes[processID]
				mu.RUnlock()
				_ = exists
			}
			i++
		}
	})
}

// BenchmarkMixedWorkloadSyncMap simulates real-world mixed read/write workload with sync.Map
func BenchmarkMixedWorkloadSyncMap(b *testing.B) {
	var processes sync.Map

	// Pre-populate with some processes
	for i := 0; i < 50; i++ {
		processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		process := &Process{
			ID:        processID,
			Name:      "test-process",
			Script:    "test",
			Status:    StatusRunning,
			StartTime: time.Now(),
		}
		processes.Store(processID, process)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			processID := "process-" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10))

			// 80% reads, 20% writes (realistic for process management)
			if i%5 == 0 {
				// Write operation
				process := &Process{
					ID:        processID,
					Name:      "test-process",
					Script:    "test",
					Status:    StatusRunning,
					StartTime: time.Now(),
				}
				processes.Store(processID, process)
			} else {
				// Read operation
				_, exists := processes.Load(processID)
				_ = exists
			}
			i++
		}
	})
}
