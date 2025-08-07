package aicoder

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"testing"
	"time"
)

// BenchmarkPTYTypingPerformance benchmarks the performance of typing simulation
func BenchmarkPTYTypingPerformance(b *testing.B) {
	// Create a simple echo session for testing
	session, err := NewPTYSession("test-typing", "Test Typing", "cat", []string{})
	if err != nil {
		b.Fatalf("Failed to create PTY session: %v", err)
	}
	defer session.Close()

	// Wait for session to be ready
	time.Sleep(100 * time.Millisecond)

	// Test typing a realistic amount of text
	testText := "Hello world! This is a typing performance test. Let's see how fast we can process input and output."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, char := range testText {
			err := session.WriteInput([]byte(string(char)))
			if err != nil {
				b.Errorf("Failed to write input: %v", err)
			}
		}
		
		// Add a newline to flush
		session.WriteInput([]byte("\n"))
		
		// Small delay to allow processing
		time.Sleep(1 * time.Millisecond)
	}
}

// BenchmarkPTYLargeOutput benchmarks handling of large output chunks
func BenchmarkPTYLargeOutput(b *testing.B) {
	// Create a session that generates large output
	session, err := NewPTYSession("test-output", "Test Output", "sh", []string{"-c", "yes 'This is a line of output for testing large chunks of data'"})
	if err != nil {
		b.Fatalf("Failed to create PTY session: %v", err)
	}
	defer session.Close()

	// Let it generate some output
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Read from output channel
		select {
		case <-session.OutputChan:
			// Successfully processed output
		case <-time.After(10 * time.Millisecond):
			// Timeout - no output available
		}
	}
}

// TestPTYPerformanceWithProfiler runs a comprehensive performance test with CPU profiling
func TestPTYPerformanceWithProfiler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping profiler test in short mode")
	}

	// Start CPU profiling
	cpuFile, err := os.Create("pty_cpu_profile.prof")
	if err != nil {
		t.Fatalf("Failed to create CPU profile file: %v", err)
	}
	defer cpuFile.Close()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		t.Fatalf("Failed to start CPU profiling: %v", err)
	}
	defer pprof.StopCPUProfile()

	// Run the actual performance test
	runPTYPerformanceTest(t)

	// Write memory profile
	memFile, err := os.Create("pty_mem_profile.prof")
	if err != nil {
		t.Fatalf("Failed to create memory profile file: %v", err)
	}
	defer memFile.Close()

	runtime.GC() // Get up-to-date statistics
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		t.Fatalf("Failed to write memory profile: %v", err)
	}

	t.Logf("Profiling complete. Run these commands to analyze:")
	t.Logf("CPU: go tool pprof pty_cpu_profile.prof")
	t.Logf("Memory: go tool pprof pty_mem_profile.prof")
}

// runPTYPerformanceTest runs the actual performance test
func runPTYPerformanceTest(t *testing.T) {
	// Create multiple sessions to simulate real usage
	sessions := make([]*PTYSession, 3)
	
	for i := 0; i < 3; i++ {
		session, err := NewPTYSession(
			fmt.Sprintf("perf-test-%d", i), 
			fmt.Sprintf("Performance Test %d", i),
			"bash", 
			[]string{"-c", "while true; do echo 'Line output from session " + fmt.Sprintf("%d", i) + "'; sleep 0.1; done"},
		)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		sessions[i] = session
		defer session.Close()
	}

	// Wait for sessions to start
	time.Sleep(200 * time.Millisecond)

	// Simulate typing in multiple sessions
	testInputs := []string{
		"echo 'Testing performance with multiple commands'",
		"ls -la /tmp",
		"ps aux | grep test",
		"find /usr -name '*.so' | head -10",
		"cat /proc/cpuinfo | head -20",
	}

	startTime := time.Now()
	
	// Send inputs to different sessions
	for round := 0; round < 50; round++ {
		for i, session := range sessions {
			input := testInputs[round%len(testInputs)]
			
			// Type character by character to simulate real typing
			for _, char := range input {
				err := session.WriteInput([]byte(string(char)))
				if err != nil {
					t.Errorf("Failed to write to session %d: %v", i, err)
				}
				// Small delay to simulate human typing
				time.Sleep(1 * time.Millisecond)
			}
			
			// Send enter
			session.WriteInput([]byte("\n"))
			
			// Small delay between commands
			time.Sleep(10 * time.Millisecond)
		}
		
		// Process any output
		for _, session := range sessions {
			select {
			case <-session.OutputChan:
				// Output processed
			default:
				// No output available
			}
		}
	}

	duration := time.Since(startTime)
	t.Logf("Performance test completed in %v", duration)
	
	// Test terminal rendering performance
	testTerminalRendering(t, sessions[0])
}

// testTerminalRendering tests the terminal rendering performance
func testTerminalRendering(t *testing.T, session *PTYSession) {
	if session.Terminal == nil {
		t.Log("No terminal available for rendering test")
		return
	}

	renderStart := time.Now()
	
	// Simulate terminal rendering operations
	for i := 0; i < 1000; i++ {
		session.mu.RLock()
		if session.Terminal != nil {
			// Read terminal dimensions
			width, height := session.Terminal.Size()
			
			// Read some cells (simulate rendering)
			if width > 0 && height > 0 {
				maxCells := 100
				if width*height < maxCells {
					maxCells = width * height
				}
				
				for j := 0; j < maxCells; j++ {
					x := j % width
					y := j / width
					if y >= height {
						break
					}
					_ = session.Terminal.Cell(x, y)
				}
			}
		}
		session.mu.RUnlock()
		
		// Small delay to simulate frame rate
		time.Sleep(16 * time.Microsecond) // ~60 FPS
	}
	
	renderDuration := time.Since(renderStart)
	t.Logf("Terminal rendering test (1000 frames) completed in %v", renderDuration)
}

// BenchmarkTerminalWrite benchmarks the new chunked write method
func BenchmarkTerminalWrite(b *testing.B) {
	session, err := NewPTYSession("bench-write", "Benchmark Write", "cat", []string{})
	if err != nil {
		b.Fatalf("Failed to create PTY session: %v", err)
	}
	defer session.Close()

	// Create test data of various sizes
	testData := [][]byte{
		[]byte("Hello"),                                    // Small
		[]byte(strings.Repeat("Medium sized data ", 10)), // Medium
		[]byte(strings.Repeat("Large chunk of data that might cause partial writes ", 50)), // Large
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := testData[i%len(testData)]
		session.mu.Lock()
		if session.Terminal != nil {
			session.writeToTerminalInChunks(data)
		}
		session.mu.Unlock()
	}
}

// TestTypingLatency measures end-to-end typing latency
func TestTypingLatency(t *testing.T) {
	session, err := NewPTYSession("latency-test", "Latency Test", "cat", []string{})
	if err != nil {
		t.Fatalf("Failed to create PTY session: %v", err)
	}
	defer session.Close()

	// Wait for session to be ready
	time.Sleep(100 * time.Millisecond)

	var totalLatency time.Duration
	measurements := 100

	for i := 0; i < measurements; i++ {
		char := fmt.Sprintf("a")
		
		start := time.Now()
		
		// Send input
		err := session.WriteInput([]byte(char))
		if err != nil {
			t.Errorf("Failed to write input: %v", err)
			continue
		}
		
		// Wait for output to appear
		timeout := time.After(100 * time.Millisecond)
		select {
		case <-session.OutputChan:
			latency := time.Since(start)
			totalLatency += latency
			
		case <-timeout:
			t.Errorf("Timeout waiting for output on iteration %d", i)
		}
		
		// Small delay between measurements
		time.Sleep(10 * time.Millisecond)
	}

	avgLatency := totalLatency / time.Duration(measurements)
	t.Logf("Average typing latency: %v", avgLatency)
	
	if avgLatency > 50*time.Millisecond {
		t.Logf("WARNING: High latency detected (%v > 50ms)", avgLatency)
	} else {
		t.Logf("Good latency performance: %v", avgLatency)
	}
}