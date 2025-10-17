package tui

import (
	"context"
	"testing"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
)

// BenchmarkPTYIOOptimization tests PTY I/O performance improvements
func BenchmarkPTYIOOptimization(b *testing.B) {
	// Create test session
	session := &aicoder.PTYSession{
		ID:         "test-pty-session",
		Name:       "Test PTY Session",
		OutputChan: make(chan []byte, 1000),
		EventChan:  make(chan aicoder.PTYEvent, 100),
	}

	logStore := logs.NewStore(1000, nil)

	b.Run("OriginalDirectPTY", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		passthrough := NewDirectPTYPassthrough(ctx, session, logStore)
		passthrough.Start()
		defer passthrough.Stop()

		testData := make([]byte, 128) // Typical PTY output size

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Simulate PTY output
			session.OutputChan <- testData

			// Process output
			cmd := passthrough.ProcessPTYOutput()
			if cmd != nil {
				msg := cmd()
				_ = msg
			}
		}
	})

	b.Run("OptimizedPTYIO", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ptyIO := NewOptimizedPTYIO(ctx, session, logStore)
		ptyIO.Start()
		defer ptyIO.Stop()

		testData := make([]byte, 128) // Typical PTY output size

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Simulate PTY output
			session.OutputChan <- testData

			// Process output with batch optimization
			cmd := ptyIO.ProcessPTYOutputOptimized()
			if cmd != nil {
				msg := cmd()
				_ = msg
			}
		}
	})
}

// BenchmarkPTYBatchProcessing tests batch processing performance
func BenchmarkPTYBatchProcessing(b *testing.B) {
	session := &aicoder.PTYSession{
		ID:         "batch-test-session",
		Name:       "Batch Test Session",
		OutputChan: make(chan []byte, 10000),
		EventChan:  make(chan aicoder.PTYEvent, 100),
	}

	logStore := logs.NewStore(1000, nil)

	b.Run("SingleMessageProcessing", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ptyIO := NewOptimizedPTYIO(ctx, session, logStore)
		ptyIO.batchMode.Store(false) // Disable batch mode
		ptyIO.Start()
		defer ptyIO.Stop()

		testData := make([]byte, 64)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			session.OutputChan <- testData

			cmd := ptyIO.ProcessPTYOutputOptimized()
			if cmd != nil {
				msg := cmd()
				_ = msg
			}
		}
	})

	b.Run("BatchMessageProcessing", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ptyIO := NewOptimizedPTYIO(ctx, session, logStore)
		ptyIO.batchMode.Store(true) // Enable batch mode
		ptyIO.Start()
		defer ptyIO.Stop()

		testData := make([]byte, 64)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Send multiple messages for batch processing
			for j := 0; j < 10; j++ {
				session.OutputChan <- testData
			}

			cmd := ptyIO.ProcessPTYOutputOptimized()
			if cmd != nil {
				msg := cmd()
				_ = msg
			}
		}
	})
}

// BenchmarkPTYMemoryUsage tests memory efficiency across different data sizes
func BenchmarkPTYMemoryUsage(b *testing.B) {
	session := &aicoder.PTYSession{
		ID:         "memory-test-session",
		Name:       "Memory Test Session",
		OutputChan: make(chan []byte, 1000),
		EventChan:  make(chan aicoder.PTYEvent, 100),
	}

	logStore := logs.NewStore(1000, nil)

	dataSizes := []int{32, 64, 128, 256, 512, 1024} // Various PTY output sizes

	for _, size := range dataSizes {
		testData := make([]byte, size)

		b.Run("DirectPTY_"+string(rune('0'+size/100))+"B", func(b *testing.B) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			passthrough := NewDirectPTYPassthrough(ctx, session, logStore)
			passthrough.Start()
			defer passthrough.Stop()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				session.OutputChan <- testData

				cmd := passthrough.ProcessPTYOutput()
				if cmd != nil {
					msg := cmd()
					_ = msg
				}
			}
		})

		b.Run("OptimizedPTY_"+string(rune('0'+size/100))+"B", func(b *testing.B) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ptyIO := NewOptimizedPTYIO(ctx, session, logStore)
			ptyIO.Start()
			defer ptyIO.Stop()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				session.OutputChan <- testData

				cmd := ptyIO.ProcessPTYOutputOptimized()
				if cmd != nil {
					msg := cmd()
					_ = msg
				}
			}
		})
	}
}

// BenchmarkPTYConcurrency tests performance under high concurrency
func BenchmarkPTYConcurrency(b *testing.B) {
	session := &aicoder.PTYSession{
		ID:         "concurrency-test-session",
		Name:       "Concurrency Test Session",
		OutputChan: make(chan []byte, 10000),
		EventChan:  make(chan aicoder.PTYEvent, 100),
	}

	logStore := logs.NewStore(1000, nil)
	testData := make([]byte, 128)

	b.Run("DirectPTYConcurrent", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		passthrough := NewDirectPTYPassthrough(ctx, session, logStore)
		passthrough.Start()
		defer passthrough.Stop()

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				session.OutputChan <- testData

				cmd := passthrough.ProcessPTYOutput()
				if cmd != nil {
					msg := cmd()
					_ = msg
				}
			}
		})
	})

	b.Run("OptimizedPTYConcurrent", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ptyIO := NewOptimizedPTYIO(ctx, session, logStore)
		ptyIO.Start()
		defer ptyIO.Stop()

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				session.OutputChan <- testData

				cmd := ptyIO.ProcessPTYOutputOptimized()
				if cmd != nil {
					msg := cmd()
					_ = msg
				}
			}
		})
	})
}

// BenchmarkPTYStressTest performs high-intensity stress testing
func BenchmarkPTYStressTest(b *testing.B) {
	session := &aicoder.PTYSession{
		ID:         "stress-test-session",
		Name:       "Stress Test Session",
		OutputChan: make(chan []byte, 50000),
		EventChan:  make(chan aicoder.PTYEvent, 100),
	}

	logStore := logs.NewStore(1000, nil)
	testData := make([]byte, 256) // Medium-sized PTY output

	b.Run("DirectPTYStress", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		passthrough := NewDirectPTYPassthrough(ctx, session, logStore)
		passthrough.Start()
		defer passthrough.Stop()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// High-frequency output simulation
			for j := 0; j < 100; j++ {
				session.OutputChan <- testData
			}

			// Process all accumulated messages
			for j := 0; j < 100; j++ {
				cmd := passthrough.ProcessPTYOutput()
				if cmd != nil {
					msg := cmd()
					_ = msg
				}
			}
		}
	})

	b.Run("OptimizedPTYStress", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ptyIO := NewOptimizedPTYIO(ctx, session, logStore)
		ptyIO.Start()
		defer ptyIO.Stop()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// High-frequency output simulation
			for j := 0; j < 100; j++ {
				session.OutputChan <- testData
			}

			// Process all accumulated messages (batch optimized)
			for j := 0; j < 100; j++ {
				cmd := ptyIO.ProcessPTYOutputOptimized()
				if cmd != nil {
					msg := cmd()
					_ = msg
				}
			}
		}
	})
}

// BenchmarkPTYThroughput measures maximum throughput capabilities
func BenchmarkPTYThroughput(b *testing.B) {
	session := &aicoder.PTYSession{
		ID:         "throughput-test-session",
		Name:       "Throughput Test Session",
		OutputChan: make(chan []byte, 100000),
		EventChan:  make(chan aicoder.PTYEvent, 100),
	}

	logStore := logs.NewStore(1000, nil)
	testData := make([]byte, 512) // Larger output for throughput testing

	b.Run("DirectPTYThroughput", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		passthrough := NewDirectPTYPassthrough(ctx, session, logStore)
		passthrough.Start()
		defer passthrough.Stop()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Maximize throughput
			for j := 0; j < 1000; j++ {
				session.OutputChan <- testData
			}

			// Drain the channel
			for len(session.OutputChan) > 0 {
				cmd := passthrough.ProcessPTYOutput()
				if cmd != nil {
					msg := cmd()
					_ = msg
				}
			}
		}
	})

	b.Run("OptimizedPTYThroughput", func(b *testing.B) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ptyIO := NewOptimizedPTYIO(ctx, session, logStore)
		ptyIO.Start()
		defer ptyIO.Stop()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Maximize throughput with batch optimization
			for j := 0; j < 1000; j++ {
				session.OutputChan <- testData
			}

			// Drain the channel (batch optimized)
			for len(session.OutputChan) > 0 {
				cmd := ptyIO.ProcessPTYOutputOptimized()
				if cmd != nil {
					msg := cmd()
					_ = msg
				}
			}
		}
	})
}