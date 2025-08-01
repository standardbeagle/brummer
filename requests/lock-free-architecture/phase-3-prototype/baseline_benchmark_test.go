package prototype

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Baseline mutex-based implementation for comparison
type MutexProcess struct {
	mu        sync.RWMutex
	id        string
	status    string
	startTime time.Time
	endTime   *time.Time
}

func (p *MutexProcess) GetStatus() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

func (p *MutexProcess) SetStatus(status string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = status
}

// Channel-based prototype for comparison
type ChannelProcess struct {
	id       string
	commands chan command
	queries  chan query
	done     chan struct{}
}

type command struct {
	op    string
	value interface{}
}

type query struct {
	op   string
	resp chan interface{}
}

func NewChannelProcess(id string) *ChannelProcess {
	p := &ChannelProcess{
		id:       id,
		commands: make(chan command, 100),
		queries:  make(chan query, 100),
		done:     make(chan struct{}),
	}

	// Start the process manager goroutine
	go p.run()

	return p
}

func (p *ChannelProcess) run() {
	status := "pending"
	startTime := time.Now()
	var endTime *time.Time

	for {
		select {
		case cmd := <-p.commands:
			switch cmd.op {
			case "setStatus":
				status = cmd.value.(string)
				if status == "stopped" || status == "failed" {
					now := time.Now()
					endTime = &now
				}
			}

		case qry := <-p.queries:
			switch qry.op {
			case "getStatus":
				qry.resp <- status
			case "getStartTime":
				qry.resp <- startTime
			case "getEndTime":
				qry.resp <- endTime
			}

		case <-p.done:
			return
		}
	}
}

func (p *ChannelProcess) GetStatus() string {
	resp := make(chan interface{}, 1)
	p.queries <- query{op: "getStatus", resp: resp}
	return (<-resp).(string)
}

func (p *ChannelProcess) SetStatus(status string) {
	p.commands <- command{op: "setStatus", value: status}
}

func (p *ChannelProcess) Close() {
	close(p.done)
}

// Benchmarks comparing mutex vs channel approaches
func BenchmarkMutexVsChannel(b *testing.B) {
	b.Run("Mutex-SingleReader", func(b *testing.B) {
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

	b.Run("Channel-SingleReader", func(b *testing.B) {
		proc := NewChannelProcess("test")
		proc.SetStatus("running")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = proc.GetStatus()
		}

		b.StopTimer()
		proc.Close()
	})

	b.Run("Mutex-ConcurrentReaders-10", func(b *testing.B) {
		proc := &MutexProcess{
			id:        "test",
			status:    "running",
			startTime: time.Now(),
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = proc.GetStatus()
			}
		})
	})

	b.Run("Channel-ConcurrentReaders-10", func(b *testing.B) {
		proc := NewChannelProcess("test")
		proc.SetStatus("running")

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = proc.GetStatus()
			}
		})

		b.StopTimer()
		proc.Close()
	})

	b.Run("Mutex-WriterReader-Contention", func(b *testing.B) {
		proc := &MutexProcess{
			id:        "test",
			status:    "running",
			startTime: time.Now(),
		}

		var writeCount int64
		done := make(chan struct{})

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
			_ = proc.GetStatus()
		}

		b.StopTimer()
		close(done)
		b.Logf("Write operations: %d", atomic.LoadInt64(&writeCount))
	})

	b.Run("Channel-WriterReader-Contention", func(b *testing.B) {
		proc := NewChannelProcess("test")
		proc.SetStatus("running")

		var writeCount int64
		done := make(chan struct{})

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
			_ = proc.GetStatus()
		}

		b.StopTimer()
		close(done)
		proc.Close()
		b.Logf("Write operations: %d", atomic.LoadInt64(&writeCount))
	})
}

// Deadlock detection test
func TestDeadlockDetection(t *testing.T) {
	t.Run("Mutex-DeadlockTest", func(t *testing.T) {
		proc := &MutexProcess{
			id:        "test",
			status:    "running",
			startTime: time.Now(),
		}

		const numGoroutines = 100
		const numOperations = 1000

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					if id%2 == 0 {
						_ = proc.GetStatus()
					} else {
						proc.SetStatus(fmt.Sprintf("status-%d", j))
					}
				}
			}(i)
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			t.Logf("Mutex test completed in %v", time.Since(start))
		case <-time.After(30 * time.Second):
			t.Fatal("Mutex test timeout - possible deadlock")
		}
	})

	t.Run("Channel-DeadlockTest", func(t *testing.T) {
		proc := NewChannelProcess("test")
		defer proc.Close()

		const numGoroutines = 100
		const numOperations = 1000

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					if id%2 == 0 {
						_ = proc.GetStatus()
					} else {
						proc.SetStatus(fmt.Sprintf("status-%d", j))
					}
				}
			}(i)
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			t.Logf("Channel test completed in %v", time.Since(start))
		case <-time.After(30 * time.Second):
			t.Fatal("Channel test timeout - possible deadlock")
		}
	})
}

// Memory allocation comparison
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("Mutex-MemoryPerOperation", func(b *testing.B) {
		proc := &MutexProcess{
			id:        "test",
			status:    "running",
			startTime: time.Now(),
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = proc.GetStatus()
		}
	})

	b.Run("Channel-MemoryPerOperation", func(b *testing.B) {
		proc := NewChannelProcess("test")
		proc.SetStatus("running")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = proc.GetStatus()
		}

		b.StopTimer()
		proc.Close()
	})
}
