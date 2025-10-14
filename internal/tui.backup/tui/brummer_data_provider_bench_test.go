package tui

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/pkg/events"
)

// BenchmarkDataProviderAtomic tests the performance of atomic-based data provider
func BenchmarkDataProviderAtomic(b *testing.B) {
	// Setup
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000, eventBus)
	processMgr, _ := process.NewManager(".", eventBus, false)

	model := &Model{
		logStore:   logStore,
		processMgr: processMgr,
	}

	provider := NewTUIDataProvider(model)

	// Add some test data
	for i := 0; i < 100; i++ {
		logStore.Add("test", "TestProcess", "Test log message", false)
	}

	b.ResetTimer()

	// Run parallel reads
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Perform various read operations
			_ = provider.GetRecentLogs(10)
			_ = provider.GetLastError()
			_ = provider.GetProcessInfo()
			_ = provider.GetDetectedURLs()
		}
	})
}

// mutexDataProvider is the old mutex-based implementation for comparison
type mutexDataProvider struct {
	model *Model
	mu    sync.RWMutex
}

func (p *mutexDataProvider) GetRecentLogs(count int) []logs.LogEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.model == nil || p.model.logStore == nil {
		return []logs.LogEntry{}
	}

	allLogs := p.model.logStore.GetAll()
	if len(allLogs) <= count {
		return allLogs
	}

	return allLogs[len(allLogs)-count:]
}

// BenchmarkDataProviderMutex tests the performance of mutex-based data provider
func BenchmarkDataProviderMutex(b *testing.B) {
	// Setup
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000, eventBus)
	processMgr, _ := process.NewManager(".", eventBus, false)

	model := &Model{
		logStore:   logStore,
		processMgr: processMgr,
	}

	provider := &mutexDataProvider{model: model}

	// Add some test data
	for i := 0; i < 100; i++ {
		logStore.Add("test", "TestProcess", "Test log message", false)
	}

	b.ResetTimer()

	// Run parallel reads
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = provider.GetRecentLogs(10)
		}
	})
}

// BenchmarkAtomicVsMutex compares atomic operations vs mutex in isolation
func BenchmarkAtomicVsMutex(b *testing.B) {
	type testData struct {
		value int
	}

	b.Run("Atomic", func(b *testing.B) {
		var atomicVal atomic.Value
		atomicVal.Store(&testData{value: 42})

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				v := atomicVal.Load().(*testData)
				_ = v.value
			}
		})
	})

	b.Run("Mutex", func(b *testing.B) {
		var mu sync.RWMutex
		data := &testData{value: 42}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mu.RLock()
				_ = data.value
				mu.RUnlock()
			}
		})
	})
}

// BenchmarkHighContention tests performance under high contention
func BenchmarkHighContention(b *testing.B) {
	// Test with many goroutines competing for access
	numGoroutines := 100

	b.Run("Atomic", func(b *testing.B) {
		var atomicVal atomic.Value
		atomicVal.Store(&Model{})

		b.SetParallelism(numGoroutines)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = atomicVal.Load().(*Model)
			}
		})
	})

	b.Run("Mutex", func(b *testing.B) {
		var mu sync.RWMutex
		model := &Model{}

		b.SetParallelism(numGoroutines)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mu.RLock()
				_ = model
				mu.RUnlock()
			}
		})
	})
}
