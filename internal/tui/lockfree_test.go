package tui

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
)

// TestLockFreePTYBufferPerformance validates the lock-free buffer performance
func TestLockFreePTYBufferPerformance(t *testing.T) {
	buffer := NewSafePTYBuffer(8192) // 8K buffer

	// Test parameters
	const numMessages = 100000
	const numWriters = 4
	const numReaders = 2

	var wg sync.WaitGroup
	var writeCount atomic.Uint64
	var readCount atomic.Uint64

	startTime := time.Now()

	// Start writer goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			for j := 0; j < numMessages/numWriters; j++ {
				msg := &SafePTYMessage{
					SessionID: "test-session",
					Data:      []byte("test data from writer"),
					Timestamp: time.Now(),
					Type:      SafePTYMessageOutput,
					Sequence:  uint64(j),
				}

				for !buffer.TryWrite(msg) {
					// Spin until write succeeds (simulates backpressure)
					time.Sleep(time.Microsecond)
				}
				writeCount.Add(1)
			}
		}(i)
	}

	// Start reader goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			for readCount.Load() < numMessages {
				batch, count := buffer.TryReadBatch(32)
				if count > 0 {
					readCount.Add(uint64(count))
					// Process batch (simulate work)
					for _, msg := range batch {
						if msg == nil {
							t.Errorf("Reader %d got nil message", readerID)
						}
					}
				} else {
					time.Sleep(time.Microsecond)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Validate results
	finalWritten := writeCount.Load()
	finalRead := readCount.Load()

	if finalWritten != numMessages {
		t.Errorf("Expected %d writes, got %d", numMessages, finalWritten)
	}

	if finalRead < finalWritten {
		t.Errorf("Read fewer messages (%d) than written (%d)", finalRead, finalWritten)
	}

	// Performance metrics
	throughput := float64(numMessages) / duration.Seconds()
	avgLatency := duration / time.Duration(numMessages)

	t.Logf("Lock-free buffer performance:")
	t.Logf("  Messages: %d", numMessages)
	t.Logf("  Writers: %d, Readers: %d", numWriters, numReaders)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.0f messages/sec", throughput)
	t.Logf("  Avg Latency: %v", avgLatency)

	// Validate performance expectations
	if throughput < 100000 { // Should handle at least 100K msgs/sec
		t.Errorf("Performance too low: %.0f messages/sec", throughput)
	}

	// Check buffer metrics
	metrics := buffer.GetMetrics()
	t.Logf("Buffer metrics:")
	t.Logf("  Buffer size: %d", metrics.BufferSize)
	t.Logf("  Messages written: %d", metrics.MessagesWritten)
	t.Logf("  Messages dropped: %d", metrics.MessagesDropped)
	t.Logf("  Max used slots: %d", metrics.MaxUsedSlots)
	t.Logf("  Saturation ratio: %.2f%%", metrics.SaturationRatio*100)
}

// TestLockFreePTYManagerIntegration validates the PTY manager integration
func TestLockFreePTYManagerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create update channel to capture messages
	updateChan := make(chan tea.Msg, 1000)
	logStore := logs.NewStore(1000, nil)

	// Create lock-free PTY manager
	manager := NewLockFreePTYManager(ctx, updateChan, logStore)
	defer manager.Shutdown()

	// Create mock PTY session
	mockSession := &aicoder.PTYSession{
		ID:         "test-session-123",
		Name:       "Test Session",
		OutputChan: make(chan []byte, 100),
		EventChan:  make(chan aicoder.PTYEvent, 10),
	}

	// Start monitoring the session
	err := manager.StartSession(mockSession)
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Send test data through the session
	const numTestMessages = 1000
	go func() {
		for i := 0; i < numTestMessages; i++ {
			testData := []byte("test output data " + string(rune('0'+i%10)))
			select {
			case mockSession.OutputChan <- testData:
			case <-ctx.Done():
				return
			}
		}
		close(mockSession.OutputChan)
	}()

	// Collect batched messages
	var totalMessages int
	var batchCount int

	timeout := time.After(5 * time.Second)
	for totalMessages < numTestMessages {
		select {
		case msg := <-updateChan:
			switch typedMsg := msg.(type) {
			case BatchPTYMsg:
				batchCount++
				totalMessages += len(typedMsg.Messages)
				t.Logf("Received batch %d with %d messages (total: %d)",
					batchCount, len(typedMsg.Messages), totalMessages)
			case PTYOutputMsg:
				// Legacy single message (should not occur with lock-free manager)
				totalMessages++
				t.Logf("Received legacy PTY message")
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for messages. Got %d of %d", totalMessages, numTestMessages)
		case <-ctx.Done():
			t.Fatalf("Context cancelled")
		}
	}

	// Validate batching efficiency
	if batchCount == 0 {
		t.Error("No batched messages received - batching not working")
	}

	avgBatchSize := float64(totalMessages) / float64(batchCount)
	t.Logf("Batching efficiency:")
	t.Logf("  Total messages: %d", totalMessages)
	t.Logf("  Total batches: %d", batchCount)
	t.Logf("  Average batch size: %.1f", avgBatchSize)

	if avgBatchSize < 2 {
		t.Error("Batching ineffective - average batch size too small")
	}

	// Stop the session
	err = manager.StopSession(mockSession.ID)
	if err != nil {
		t.Errorf("Failed to stop session: %v", err)
	}

	// Get final metrics
	managerMetrics := manager.GetMetrics()
	bufferMetrics := manager.GetBufferMetrics()

	t.Logf("Manager metrics:")
	t.Logf("  Active sessions: %d", managerMetrics.ActiveSessions)
	t.Logf("  Total messages: %d", managerMetrics.TotalMessages)
	t.Logf("  Dropped messages: %d", managerMetrics.DroppedMessages)
	t.Logf("  Batches processed: %d", managerMetrics.BatchesProcessed)

	t.Logf("Buffer metrics:")
	t.Logf("  Buffer utilization: %.1f%%", bufferMetrics.SaturationRatio*100)
	t.Logf("  Max used slots: %d", bufferMetrics.MaxUsedSlots)
}

// BenchmarkLockFreeVsChannel compares lock-free vs channel-based approaches
func BenchmarkLockFreeVsChannel(b *testing.B) {
	b.Run("LockFree", func(b *testing.B) {
		buffer := NewSafePTYBuffer(8192)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			msg := &SafePTYMessage{
				SessionID: "bench-session",
				Data:      []byte("benchmark data"),
				Timestamp: time.Now(),
				Type:      SafePTYMessageOutput,
				Sequence:  uint64(i),
			}

			for !buffer.TryWrite(msg) {
				// Busy wait for write slot
			}

			_, ok := buffer.TryRead()
			if !ok {
				b.Error("Failed to read written message")
			}
		}
	})

	b.Run("Channel", func(b *testing.B) {
		ch := make(chan []byte, 10000) // Same buffer size as emergency fix

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := []byte("benchmark data")

			select {
			case ch <- data:
			default:
				// Channel full - drop message (same as lock-free backpressure)
			}

			select {
			case <-ch:
			default:
				// Channel empty
			}
		}
	})
}

// BenchmarkRealisticPTYScenario simulates real AI coder PTY conditions
func BenchmarkRealisticPTYScenario(b *testing.B) {
	b.Run("LockFreeBatched", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		updateChan := make(chan tea.Msg, 1000)
		logStore := logs.NewStore(100, nil)

		manager := NewLockFreePTYManager(ctx, updateChan, logStore)
		defer manager.Shutdown()

		b.ResetTimer()

		// Simulate high-frequency PTY output (like AI coder typing response)
		for i := 0; i < b.N; i++ {
			// Create burst of messages (simulates AI response)
			for j := 0; j < 50; j++ {
				msg := &SafePTYMessage{
					SessionID: "ai-session",
					Data:      []byte("AI response line with typical content length for realistic testing"),
					Timestamp: time.Now(),
					Type:      SafePTYMessageOutput,
					Sequence:  uint64(i*50 + j),
				}
				manager.buffer.TryWrite(msg)
			}

			// Process batch (simulates BubbleTea update cycle)
			batch, count := manager.buffer.TryReadBatch(32)
			if count > 0 {
				// Simulate processing batched messages
				for _, msg := range batch {
					_ = msg.Data // Simulate processing
				}
			}
		}
	})

	b.Run("ChannelSaturated", func(b *testing.B) {
		// Simulate the problematic channel-based approach that caused the AI coder issues
		updateChan := make(chan tea.Msg, 100) // Original small buffer size

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate burst of PTY output messages
			for j := 0; j < 50; j++ {
				msg := PTYOutputMsg{
					SessionID: "ai-session",
					Data:      []byte("AI response line with typical content length for realistic testing"),
				}

				select {
				case updateChan <- msg:
					// Message sent
				default:
					// Channel saturated - this is where the AI coder issues occurred
					b.Logf("Channel saturated at iteration %d", i)
				}
			}

			// Process individual messages (original approach)
			for len(updateChan) > 0 {
				select {
				case msg := <-updateChan:
					_ = msg // Simulate processing
				default:
					break
				}
			}
		}
	})
}

// TestLockFreeMemoryUsage validates memory characteristics
func TestLockFreeMemoryUsage(t *testing.T) {
	var m1, m2 runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create and use lock-free buffer
	buffer := NewSafePTYBuffer(8192)

	// Write and read many messages
	for i := 0; i < 10000; i++ {
		msg := &SafePTYMessage{
			SessionID: "memory-test",
			Data:      make([]byte, 100), // 100 byte payload
			Timestamp: time.Now(),
			Type:      SafePTYMessageOutput,
			Sequence:  uint64(i),
		}

		if buffer.TryWrite(msg) {
			buffer.TryRead()
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocDiff := m2.TotalAlloc - m1.TotalAlloc
	t.Logf("Memory usage for 10K messages:")
	t.Logf("  Total allocated: %d bytes", allocDiff)
	t.Logf("  Per message: %d bytes", allocDiff/10000)

	// Validate reasonable memory usage
	if allocDiff > 50*1024*1024 { // More than 50MB is excessive
		t.Errorf("Memory usage too high: %d bytes", allocDiff)
	}
}
