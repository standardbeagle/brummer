package prototype

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// ProcessState holds immutable process state
type ProcessState struct {
	ID        string
	Status    string
	StartTime time.Time
	EndTime   *time.Time
	ExitCode  *int
}

// AtomicProcess uses atomic pointer swapping for lock-free updates
type AtomicProcess struct {
	state unsafe.Pointer // *ProcessState
}

func NewAtomicProcess(id string) *AtomicProcess {
	initial := &ProcessState{
		ID:        id,
		Status:    "pending",
		StartTime: time.Now(),
	}
	return &AtomicProcess{
		state: unsafe.Pointer(initial),
	}
}

func (p *AtomicProcess) GetState() *ProcessState {
	return (*ProcessState)(atomic.LoadPointer(&p.state))
}

func (p *AtomicProcess) SetStatus(status string) {
	for {
		current := p.GetState()
		newState := &ProcessState{
			ID:        current.ID,
			Status:    status,
			StartTime: current.StartTime,
			EndTime:   current.EndTime,
			ExitCode:  current.ExitCode,
		}

		// Add end time if transitioning to stopped/failed
		if (status == "stopped" || status == "failed") && current.EndTime == nil {
			now := time.Now()
			newState.EndTime = &now
		}

		// Compare-and-swap
		if atomic.CompareAndSwapPointer(&p.state, unsafe.Pointer(current), unsafe.Pointer(newState)) {
			break
		}
		// Retry if state changed during update
	}
}

// SyncMapManager uses sync.Map for lock-free process registry
type SyncMapManager struct {
	processes sync.Map // string -> *AtomicProcess
}

func NewSyncMapManager() *SyncMapManager {
	return &SyncMapManager{}
}

func (m *SyncMapManager) AddProcess(id string) {
	proc := NewAtomicProcess(id)
	m.processes.Store(id, proc)
}

func (m *SyncMapManager) GetProcess(id string) *AtomicProcess {
	if val, ok := m.processes.Load(id); ok {
		return val.(*AtomicProcess)
	}
	return nil
}

func (m *SyncMapManager) GetAllProcesses() []*AtomicProcess {
	var procs []*AtomicProcess
	m.processes.Range(func(key, value interface{}) bool {
		procs = append(procs, value.(*AtomicProcess))
		return true
	})
	return procs
}

// Benchmarks for atomic operations
func BenchmarkAtomicOperations(b *testing.B) {
	b.Run("Atomic-SingleReader", func(b *testing.B) {
		proc := NewAtomicProcess("test")
		proc.SetStatus("running")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			state := proc.GetState()
			_ = state.Status
		}
	})

	b.Run("Atomic-ConcurrentReaders-10", func(b *testing.B) {
		proc := NewAtomicProcess("test")
		proc.SetStatus("running")

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				state := proc.GetState()
				_ = state.Status
			}
		})
	})

	b.Run("Atomic-WriterReader-Contention", func(b *testing.B) {
		proc := NewAtomicProcess("test")
		proc.SetStatus("running")

		done := make(chan struct{})
		var writeCount int64

		// Start a writer
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					proc.SetStatus("running")
					atomic.AddInt64(&writeCount, 1)
					time.Sleep(time.Microsecond)
				}
			}
		}()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			state := proc.GetState()
			_ = state.Status
		}

		b.StopTimer()
		close(done)
		b.Logf("Write operations: %d", atomic.LoadInt64(&writeCount))
	})

	// Compare with mutex baseline
	b.Run("Mutex-Baseline-SingleReader", func(b *testing.B) {
		proc := &MutexProcess{
			id:        "test",
			status:    "running",
			startTime: time.Now(),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = proc.GetStatus()
		}
	})
}

// Benchmark sync.Map for process registry
func BenchmarkSyncMapRegistry(b *testing.B) {
	b.Run("SyncMap-Lookup", func(b *testing.B) {
		mgr := NewSyncMapManager()
		// Add 100 processes
		for i := 0; i < 100; i++ {
			mgr.AddProcess(string(rune('a' + i)))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			proc := mgr.GetProcess("a")
			if proc != nil {
				_ = proc.GetState().Status
			}
		}
	})

	b.Run("SyncMap-Range", func(b *testing.B) {
		mgr := NewSyncMapManager()
		// Add 10 processes
		for i := 0; i < 10; i++ {
			mgr.AddProcess(string(rune('a' + i)))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			procs := mgr.GetAllProcesses()
			for _, proc := range procs {
				_ = proc.GetState().Status
			}
		}
	})

	// Compare with regular map + mutex
	b.Run("MutexMap-Lookup", func(b *testing.B) {
		processes := make(map[string]*MutexProcess)
		var mu sync.RWMutex

		// Add 100 processes
		for i := 0; i < 100; i++ {
			processes[string(rune('a'+i))] = &MutexProcess{
				id:        string(rune('a' + i)),
				status:    "running",
				startTime: time.Now(),
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mu.RLock()
			proc := processes["a"]
			mu.RUnlock()
			if proc != nil {
				_ = proc.GetStatus()
			}
		}
	})
}

// Memory allocation benchmarks
func BenchmarkAtomicMemoryAllocation(b *testing.B) {
	b.Run("Atomic-MemoryPerRead", func(b *testing.B) {
		proc := NewAtomicProcess("test")
		proc.SetStatus("running")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			state := proc.GetState()
			_ = state.Status
		}
	})

	b.Run("Atomic-MemoryPerWrite", func(b *testing.B) {
		proc := NewAtomicProcess("test")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			proc.SetStatus("running")
		}
	})
}

// Test atomic consistency under high contention
func TestAtomicConsistency(t *testing.T) {
	proc := NewAtomicProcess("test")

	const numWriters = 10
	const numReaders = 50
	const numOps = 1000

	var wg sync.WaitGroup
	wg.Add(numWriters + numReaders)

	// Writers constantly update status
	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				if j%2 == 0 {
					proc.SetStatus("running")
				} else {
					proc.SetStatus("stopped")
				}
			}
		}(i)
	}

	// Readers check consistency
	inconsistencies := int64(0)
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				state := proc.GetState()
				// Check consistency: stopped status should have EndTime
				if state.Status == "stopped" && state.EndTime == nil {
					atomic.AddInt64(&inconsistencies, 1)
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if inconsistencies > 0 {
			t.Errorf("Found %d inconsistencies in atomic operations", inconsistencies)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Test timeout - possible issue with atomic operations")
	}
}
