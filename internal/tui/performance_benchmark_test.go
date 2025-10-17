package tui

import (
	"testing"
	"time"
)

// BenchmarkSerializationOptimization tests serialization performance improvements
func BenchmarkSerializationOptimization(b *testing.B) {
	b.Run("OriginalSerialization", func(b *testing.B) {
		buffer := NewSafePTYBuffer(8192)
		msg := &SafePTYMessage{
			SessionID: "test-session-12345",
			Data:      make([]byte, 256),
			Timestamp: time.Now(),
			Type:      SafePTYMessageOutput,
			Sequence:  12345,
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			data, err := buffer.serializeMessage(msg)
			if err != nil {
				b.Fatal(err)
			}
			_ = data
		}
	})

	b.Run("OptimizedZeroCopy", func(b *testing.B) {
		buffer := NewOptimizedSafePTYBuffer(8192)
		data := make([]byte, 256)
		sequence := uint64(12345)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result := buffer.marshalMessageUnsafe(&OptimizedSafePTYMessage{
				Data:      data,
				Sequence:  sequence,
				Type:      uint8(SafePTYMessageOutput),
				Timestamp: time.Now().UnixNano(),
			})
			_ = result
		}
	})
}

// BenchmarkBufferOperations tests overall buffer performance
func BenchmarkBufferOperations(b *testing.B) {
	sessionID := "test-session-12345"
	testData := make([]byte, 128) // Typical PTY line size
	sequence := uint64(0)

	b.Run("OriginalSafeBuffer", func(b *testing.B) {
		buffer := NewSafePTYBuffer(8192)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			msg := &SafePTYMessage{
				SessionID: sessionID,
				Data:      testData,
				Timestamp: time.Now(),
				Type:      SafePTYMessageOutput,
				Sequence:  sequence,
			}

			if buffer.TryWrite(msg) {
				_, ok := buffer.TryRead()
				_ = ok
			}

			sequence++
		}
	})

	b.Run("OptimizedBuffer", func(b *testing.B) {
		buffer := NewOptimizedSafePTYBuffer(8192)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			if buffer.TryWriteOptimized(sessionID, testData, uint8(SafePTYMessageOutput), sequence) {
				_, ok := buffer.TryReadOptimized()
				_ = ok
			}

			sequence++
		}
	})
}

// BenchmarkOptimizedConcurrentOperations tests performance under high concurrency
func BenchmarkOptimizedConcurrentOperations(b *testing.B) {
	sessionID := "concurrent-test-session"
	testData := make([]byte, 64)
	sequence := uint64(0)

	b.Run("OriginalConcurrent", func(b *testing.B) {
		buffer := NewSafePTYBuffer(16384)

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			localSeq := sequence
			for pb.Next() {
				msg := &SafePTYMessage{
					SessionID: sessionID,
					Data:      testData,
					Timestamp: time.Now(),
					Type:      SafePTYMessageOutput,
					Sequence:  localSeq,
				}

				if buffer.TryWrite(msg) {
					_, ok := buffer.TryRead()
					_ = ok
				}

				localSeq++
			}
		})
	})

	b.Run("OptimizedConcurrent", func(b *testing.B) {
		buffer := NewOptimizedSafePTYBuffer(16384)

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			localSeq := sequence
			for pb.Next() {
				if buffer.TryWriteOptimized(sessionID, testData, uint8(SafePTYMessageOutput), localSeq) {
					_, ok := buffer.TryReadOptimized()
					_ = ok
				}

				localSeq++
			}
		})
	})
}

// BenchmarkMemoryUsage tests memory allocation patterns
func BenchmarkMemoryUsage(b *testing.B) {
	sessionID := "memory-test-session"
	varSizes := []int{64, 256, 1024, 4096} // Different PTY output sizes

	for _, size := range varSizes {
		testData := make([]byte, size)
		sizeStr := string(rune('0' + size/1000)) // Simple size naming

		b.Run("Original_"+sizeStr+"B", func(b *testing.B) {
			buffer := NewSafePTYBuffer(8192)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				msg := &SafePTYMessage{
					SessionID: sessionID,
					Data:      testData,
					Timestamp: time.Now(),
					Type:      SafePTYMessageOutput,
					Sequence:  uint64(i),
				}

				if buffer.TryWrite(msg) {
					_, ok := buffer.TryRead()
					_ = ok
				}
			}
		})

		b.Run("Optimized_"+sizeStr+"B", func(b *testing.B) {
			buffer := NewOptimizedSafePTYBuffer(8192)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				if buffer.TryWriteOptimized(sessionID, testData, uint8(SafePTYMessageOutput), uint64(i)) {
					_, ok := buffer.TryReadOptimized()
					_ = ok
				}
			}
		})
	}
}

// BenchmarkBatchProcessing tests batch operation efficiency
func BenchmarkBatchProcessing(b *testing.B) {
	sessionID := "batch-test-session"
	testData := make([]byte, 128)
	sequence := uint64(0)

	b.Run("OriginalBatch", func(b *testing.B) {
		buffer := NewSafePTYBuffer(8192)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Write multiple messages
			for j := 0; j < 10; j++ {
				msg := &SafePTYMessage{
					SessionID: sessionID,
					Data:      testData,
					Timestamp: time.Now(),
					Type:      SafePTYMessageOutput,
					Sequence:  sequence,
				}
				buffer.TryWrite(msg)
				sequence++
			}

			// Read batch
			batch, _ := buffer.TryReadBatch(10)
			_ = batch
		}
	})

	b.Run("OptimizedBatch", func(b *testing.B) {
		buffer := NewOptimizedSafePTYBuffer(8192)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Write multiple messages
			for j := 0; j < 10; j++ {
				buffer.TryWriteOptimized(sessionID, testData, uint8(SafePTYMessageOutput), sequence)
				sequence++
			}

			// Read batch
			for j := 0; j < 10; j++ {
				_, ok := buffer.TryReadOptimized()
				_ = ok
			}
		}
	})
}

// BenchmarkStressTest performs high-intensity stress testing
func BenchmarkStressTest(b *testing.B) {
	sessionID := "stress-test-session"
	testData := make([]byte, 256) // Medium-sized PTY output

	b.Run("OriginalStress", func(b *testing.B) {
		buffer := NewSafePTYBuffer(65536) // Large buffer for stress test

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Rapid fire writes
			for j := 0; j < 50; j++ {
				msg := &SafePTYMessage{
					SessionID: sessionID,
					Data:      testData,
					Timestamp: time.Now(),
					Type:      SafePTYMessageOutput,
					Sequence:  uint64(i*50 + j),
				}
				buffer.TryWrite(msg)
			}

			// Rapid fire reads
			for j := 0; j < 50; j++ {
				_, ok := buffer.TryRead()
				_ = ok
			}
		}
	})

	b.Run("OptimizedStress", func(b *testing.B) {
		buffer := NewOptimizedSafePTYBuffer(65536) // Large buffer for stress test

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Rapid fire writes
			for j := 0; j < 50; j++ {
				buffer.TryWriteOptimized(sessionID, testData, uint8(SafePTYMessageOutput), uint64(i*50+j))
			}

			// Rapid fire reads
			for j := 0; j < 50; j++ {
				_, ok := buffer.TryReadOptimized()
				_ = ok
			}
		}
	})
}