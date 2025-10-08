package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestKeyLatency tests keyboard input responsiveness
// Performance requirement: <50ms latency for key events
func TestKeyLatency(t *testing.T) {
	SkipIfShort(t, "Performance test for keyboard latency")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Set standard window size
	ctx.SetWindowSize(120, 40)

	t.Run("single_key_latency", func(t *testing.T) {
		// Measure time from key press to Update() completion
		start := time.Now()
		ctx.SendKeyMsg("1")
		latency := time.Since(start)

		// Key press should be processed quickly
		assert.Less(t, latency, 50*time.Millisecond, "Single key latency should be <50ms")
	})

	t.Run("rapid_key_sequence_latency", func(t *testing.T) {
		// Measure time for rapid key sequence
		keys := []string{"1", "2", "3", "4", "5"}

		start := time.Now()
		for _, key := range keys {
			ctx.SendKeyMsg(key)
		}
		totalLatency := time.Since(start)

		// Average latency per key
		avgLatency := totalLatency / time.Duration(len(keys))
		assert.Less(t, avgLatency, 50*time.Millisecond, "Average key latency should be <50ms")
		assert.Less(t, totalLatency, 250*time.Millisecond, "Rapid sequence should be <250ms total")
	})

	t.Run("navigation_key_latency", func(t *testing.T) {
		// Test arrow key responsiveness
		start := time.Now()
		ctx.SendKeyMsg("down")
		downLatency := time.Since(start)

		start = time.Now()
		ctx.SendKeyMsg("up")
		upLatency := time.Since(start)

		start = time.Now()
		ctx.SendKeyMsg("left")
		leftLatency := time.Since(start)

		start = time.Now()
		ctx.SendKeyMsg("right")
		rightLatency := time.Since(start)

		// All navigation keys should be responsive
		assert.Less(t, downLatency, 50*time.Millisecond, "Down key latency should be <50ms")
		assert.Less(t, upLatency, 50*time.Millisecond, "Up key latency should be <50ms")
		assert.Less(t, leftLatency, 50*time.Millisecond, "Left key latency should be <50ms")
		assert.Less(t, rightLatency, 50*time.Millisecond, "Right key latency should be <50ms")
	})

	t.Run("special_key_latency", func(t *testing.T) {
		// Test special keys (Enter, Esc, etc.)
		start := time.Now()
		ctx.SendKeyMsg("enter")
		enterLatency := time.Since(start)

		start = time.Now()
		ctx.SendKeyMsg("esc")
		escLatency := time.Since(start)

		start = time.Now()
		ctx.SendKeyMsg("tab")
		tabLatency := time.Since(start)

		assert.Less(t, enterLatency, 50*time.Millisecond, "Enter key latency should be <50ms")
		assert.Less(t, escLatency, 50*time.Millisecond, "Esc key latency should be <50ms")
		assert.Less(t, tabLatency, 50*time.Millisecond, "Tab key latency should be <50ms")
	})
}

// TestKeyLatencyUnderLoad tests keyboard responsiveness under system load
func TestKeyLatencyUnderLoad(t *testing.T) {
	SkipIfShort(t, "Performance test for keyboard latency under load")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("latency_with_running_processes", func(t *testing.T) {
		// Start a process to create background activity
		ctx.SendKeyMsg("1") // Switch to processes view
		time.Sleep(50 * time.Millisecond)

		// Start a process (if available)
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Now test key latency with process running
		start := time.Now()
		ctx.SendKeyMsg("2") // Switch to logs view
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Key latency with running process should be <50ms")
	})

	t.Run("latency_with_high_log_volume", func(t *testing.T) {
		// Switch to logs view
		ctx.SendKeyMsg("2")
		time.Sleep(100 * time.Millisecond)

		// Test key responsiveness while logs are streaming
		// (if process is generating output)
		start := time.Now()
		ctx.SendKeyMsg("down")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Key latency with log streaming should be <50ms")
	})

	t.Run("latency_during_view_switch", func(t *testing.T) {
		// Measure latency when switching between views
		views := []string{"1", "2", "3", "4", "5"}

		var totalLatency time.Duration
		for _, view := range views {
			start := time.Now()
			ctx.SendKeyMsg(view)
			time.Sleep(20 * time.Millisecond) // Small delay between switches
			latency := time.Since(start)
			totalLatency += latency
		}

		avgLatency := totalLatency / time.Duration(len(views))
		assert.Less(t, avgLatency, 50*time.Millisecond, "Average view switch latency should be <50ms")
	})
}

// TestKeyLatencyConsistency tests that latency remains consistent
func TestKeyLatencyConsistency(t *testing.T) {
	SkipIfShort(t, "Performance test for keyboard latency consistency")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("latency_stability_over_time", func(t *testing.T) {
		// Measure latency over multiple iterations
		const iterations = 10
		latencies := make([]time.Duration, iterations)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			ctx.SendKeyMsg("1")
			latencies[i] = time.Since(start)
			time.Sleep(50 * time.Millisecond)
		}

		// Calculate average and standard deviation
		var total time.Duration
		for _, lat := range latencies {
			total += lat
		}
		avgLatency := total / time.Duration(iterations)

		// All measurements should be reasonably consistent
		for i, lat := range latencies {
			assert.Less(t, lat, 50*time.Millisecond, "Latency iteration %d should be <50ms", i)
		}

		t.Logf("Average latency over %d iterations: %v", iterations, avgLatency)
	})

	t.Run("no_latency_degradation", func(t *testing.T) {
		// Test that latency doesn't increase over time (no memory leaks, etc.)
		// Measure initial latency
		initialLatencies := make([]time.Duration, 5)
		for i := 0; i < 5; i++ {
			start := time.Now()
			ctx.SendKeyMsg("1")
			initialLatencies[i] = time.Since(start)
			time.Sleep(20 * time.Millisecond)
		}

		// Do some operations
		for i := 0; i < 50; i++ {
			ctx.SendKeyMsg("down")
			time.Sleep(10 * time.Millisecond)
		}

		// Measure latency again
		finalLatencies := make([]time.Duration, 5)
		for i := 0; i < 5; i++ {
			start := time.Now()
			ctx.SendKeyMsg("1")
			finalLatencies[i] = time.Since(start)
			time.Sleep(20 * time.Millisecond)
		}

		// Calculate averages
		var initialTotal, finalTotal time.Duration
		for i := 0; i < 5; i++ {
			initialTotal += initialLatencies[i]
			finalTotal += finalLatencies[i]
		}
		initialAvg := initialTotal / 5
		finalAvg := finalTotal / 5

		// Final latency should not be significantly worse than initial
		// Allow 20% degradation tolerance
		maxAcceptable := initialAvg + (initialAvg / 5)
		assert.Less(t, finalAvg, maxAcceptable, "Latency should not degrade significantly")

		t.Logf("Initial avg latency: %v, Final avg latency: %v", initialAvg, finalAvg)
	})
}

// TestKeyLatencyByView tests keyboard latency in different views
func TestKeyLatencyByView(t *testing.T) {
	SkipIfShort(t, "Performance test for keyboard latency by view")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("processes_view_latency", func(t *testing.T) {
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		ctx.SendKeyMsg("down")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Processes view latency should be <50ms")
	})

	t.Run("logs_view_latency", func(t *testing.T) {
		ctx.SendKeyMsg("2")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		ctx.SendKeyMsg("down")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Logs view latency should be <50ms")
	})

	t.Run("errors_view_latency", func(t *testing.T) {
		ctx.SendKeyMsg("3")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		ctx.SendKeyMsg("down")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Errors view latency should be <50ms")
	})

	t.Run("urls_view_latency", func(t *testing.T) {
		ctx.SendKeyMsg("4")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		ctx.SendKeyMsg("down")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "URLs view latency should be <50ms")
	})

	t.Run("proxy_view_latency", func(t *testing.T) {
		ctx.SendKeyMsg("5")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		ctx.SendKeyMsg("down")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Proxy view latency should be <50ms")
	})
}

// TestKeyLatencyEdgeCases tests keyboard latency in edge cases
func TestKeyLatencyEdgeCases(t *testing.T) {
	SkipIfShort(t, "Performance test for keyboard latency edge cases")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("latency_with_minimal_window", func(t *testing.T) {
		// Test with very small window
		ctx.SetWindowSize(20, 10)
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		ctx.SendKeyMsg("1")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Latency with minimal window should be <50ms")
	})

	t.Run("latency_with_large_window", func(t *testing.T) {
		// Test with very large window
		ctx.SetWindowSize(300, 100)
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		ctx.SendKeyMsg("1")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Latency with large window should be <50ms")
	})

	t.Run("latency_after_window_resize", func(t *testing.T) {
		// Resize window
		ctx.SetWindowSize(80, 30)
		time.Sleep(50 * time.Millisecond)

		// Test latency immediately after resize
		start := time.Now()
		ctx.SendKeyMsg("1")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Latency after resize should be <50ms")
	})

	t.Run("latency_with_no_processes", func(t *testing.T) {
		// Test latency when no processes are running
		// (System is mostly idle)
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		ctx.SendKeyMsg("down")
		latency := time.Since(start)

		assert.Less(t, latency, 50*time.Millisecond, "Latency with no processes should be <50ms")
	})
}

// TestKeyLatencyBenchmark provides detailed latency benchmarks
func TestKeyLatencyBenchmark(t *testing.T) {
	SkipIfShort(t, "Performance benchmark for keyboard latency")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("comprehensive_latency_report", func(t *testing.T) {
		// Comprehensive latency measurement across different key types
		const samples = 20

		// Test different key categories
		keyCategories := map[string][]string{
			"numbers":    {"1", "2", "3", "4", "5"},
			"navigation": {"up", "down", "left", "right"},
			"special":    {"enter", "esc", "tab", "space"},
			"letters":    {"a", "b", "c", "d", "e"},
		}

		results := make(map[string]time.Duration)

		for category, keys := range keyCategories {
			var totalLatency time.Duration
			var count int

			for _, key := range keys {
				for i := 0; i < samples/len(keys); i++ {
					start := time.Now()
					ctx.SendKeyMsg(key)
					latency := time.Since(start)
					totalLatency += latency
					count++
					time.Sleep(10 * time.Millisecond)
				}
			}

			avgLatency := totalLatency / time.Duration(count)
			results[category] = avgLatency

			t.Logf("%s keys - Average latency: %v (across %d samples)", category, avgLatency, count)
			assert.Less(t, avgLatency, 50*time.Millisecond, "%s key latency should be <50ms", category)
		}
	})
}
