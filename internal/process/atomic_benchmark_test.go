package process

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// BenchmarkAtomicStateRead tests the performance of atomic state reads
func BenchmarkAtomicStateRead(b *testing.B) {
	// Create a process with atomic state
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := p.GetStateAtomic()
		_ = state.Status // Prevent compiler optimization
	}
}

// BenchmarkMutexRead tests the performance of mutex-based reads (baseline)
func BenchmarkMutexRead(b *testing.B) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		mu:        sync.RWMutex{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.mu.RLock()
		status := p.Status
		p.mu.RUnlock()
		_ = status // Prevent compiler optimization
	}
}

// BenchmarkConcurrentAtomicReads tests concurrent atomic reads
func BenchmarkConcurrentAtomicReads(b *testing.B) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
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

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			state := p.GetStateAtomic()
			_ = state.Status
		}
	})
}

// BenchmarkConcurrentMutexReads tests concurrent mutex reads (baseline)
func BenchmarkConcurrentMutexReads(b *testing.B) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		mu:        sync.RWMutex{},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p.mu.RLock()
			status := p.Status
			p.mu.RUnlock()
			_ = status
		}
	})
}

// BenchmarkAtomicStateUpdate tests the performance of atomic state updates
func BenchmarkAtomicStateUpdate(b *testing.B) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.UpdateStateAtomic(func(state ProcessState) ProcessState {
			return state.CopyWithStatus(StatusStopped)
		})
	}
}

// BenchmarkMutexUpdate tests the performance of mutex-based updates (baseline)
func BenchmarkMutexUpdate(b *testing.B) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
		mu:        sync.RWMutex{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.mu.Lock()
		p.Status = StatusStopped
		p.mu.Unlock()
	}
}

// BenchmarkFullStateAccess tests reading all fields atomically
func BenchmarkFullStateAccess(b *testing.B) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
	}

	// Initialize atomic state with all fields
	exitCode := 0
	endTime := time.Now()
	initialState := ProcessState{
		ID:        p.ID,
		Name:      p.Name,
		Script:    p.Script,
		Status:    StatusSuccess,
		StartTime: p.StartTime,
		EndTime:   &endTime,
		ExitCode:  &exitCode,
		Command:   "node",
		Args:      []string{"index.js"},
		Env:       []string{"NODE_ENV=production"},
		Dir:       "/app",
	}
	atomic.StorePointer(&p.atomicState, unsafe.Pointer(&initialState))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := p.GetStateAtomic()
		// Access multiple fields to simulate real usage
		_ = state.Status
		_ = state.StartTime
		_ = state.EndTime
		_ = state.ExitCode
		_ = state.Name
	}
}

// BenchmarkAtomicMemoryAllocation tests allocation behavior
func BenchmarkAtomicMemoryAllocation(b *testing.B) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
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

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := p.GetStateAtomic()
		_ = state.Status
	}
}

// BenchmarkCASRetries tests performance under contention
func BenchmarkCASRetries(b *testing.B) {
	p := &Process{
		ID:        "test-123",
		Name:      "test-process",
		Script:    "test",
		Status:    StatusRunning,
		StartTime: time.Now(),
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

	// Simulate high contention with concurrent updates
	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Start background updaters
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					p.UpdateStateAtomic(func(state ProcessState) ProcessState {
						if state.Status == StatusRunning {
							return state.CopyWithStatus(StatusStopped)
						}
						return state.CopyWithStatus(StatusRunning)
					})
				}
			}
		}()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.UpdateStateAtomic(func(state ProcessState) ProcessState {
			return state.CopyWithStatus(StatusSuccess)
		})
	}

	close(stop)
	wg.Wait()
}
