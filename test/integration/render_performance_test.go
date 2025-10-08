package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRenderPerformance tests view rendering performance
// Performance requirement: <50ms render time per frame
func TestRenderPerformance(t *testing.T) {
	SkipIfShort(t, "Performance test for view rendering")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Set standard window size
	ctx.SetWindowSize(120, 40)

	t.Run("single_view_render_time", func(t *testing.T) {
		// Measure time to render a single view
		ctx.SendKeyMsg("1") // Switch to processes view
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "View should render content")
		assert.Less(t, renderTime, 50*time.Millisecond, "Single view render should be <50ms")
	})

	t.Run("view_switch_render_time", func(t *testing.T) {
		// Measure render time when switching views
		views := []string{"1", "2", "3", "4", "5"}

		for _, viewKey := range views {
			start := time.Now()
			ctx.SendKeyMsg(viewKey)
			time.Sleep(20 * time.Millisecond)
			view := ctx.GetCurrentView()
			renderTime := time.Since(start)

			assert.NotEmpty(t, view, "View %s should render", viewKey)
			assert.Less(t, renderTime, 50*time.Millisecond, "View %s render should be <50ms", viewKey)
		}
	})

	t.Run("rapid_view_switching", func(t *testing.T) {
		// Test render performance during rapid view switching
		const switches = 10
		views := []string{"1", "2", "3", "4", "5"}

		start := time.Now()
		for i := 0; i < switches; i++ {
			viewKey := views[i%len(views)]
			ctx.SendKeyMsg(viewKey)
			time.Sleep(10 * time.Millisecond)
		}
		totalTime := time.Since(start)

		avgTime := totalTime / time.Duration(switches)
		assert.Less(t, avgTime, 50*time.Millisecond, "Average render time during rapid switching should be <50ms")
	})
}

// TestRenderPerformanceByView tests rendering performance for each view
func TestRenderPerformanceByView(t *testing.T) {
	SkipIfShort(t, "Performance test for individual view rendering")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("processes_view_render", func(t *testing.T) {
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Processes view should render")
		assert.Less(t, renderTime, 50*time.Millisecond, "Processes view render should be <50ms")
		t.Logf("Processes view render time: %v", renderTime)
	})

	t.Run("logs_view_render", func(t *testing.T) {
		ctx.SendKeyMsg("2")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Logs view should render")
		assert.Less(t, renderTime, 50*time.Millisecond, "Logs view render should be <50ms")
		t.Logf("Logs view render time: %v", renderTime)
	})

	t.Run("errors_view_render", func(t *testing.T) {
		ctx.SendKeyMsg("3")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Errors view should render")
		assert.Less(t, renderTime, 50*time.Millisecond, "Errors view render should be <50ms")
		t.Logf("Errors view render time: %v", renderTime)
	})

	t.Run("urls_view_render", func(t *testing.T) {
		ctx.SendKeyMsg("4")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "URLs view should render")
		assert.Less(t, renderTime, 50*time.Millisecond, "URLs view render should be <50ms")
		t.Logf("URLs view render time: %v", renderTime)
	})

	t.Run("proxy_view_render", func(t *testing.T) {
		ctx.SendKeyMsg("5")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Proxy view should render")
		assert.Less(t, renderTime, 50*time.Millisecond, "Proxy view render should be <50ms")
		t.Logf("Proxy view render time: %v", renderTime)
	})

	t.Run("settings_view_render", func(t *testing.T) {
		ctx.SendKeyMsg("8")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Settings view should render")
		assert.Less(t, renderTime, 50*time.Millisecond, "Settings view render should be <50ms")
		t.Logf("Settings view render time: %v", renderTime)
	})
}

// TestRenderPerformanceUnderLoad tests rendering with high data volume
func TestRenderPerformanceUnderLoad(t *testing.T) {
	SkipIfShort(t, "Performance test for rendering under load")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("render_with_running_processes", func(t *testing.T) {
		// Start a process
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Measure render time with process running
		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Should render with running process")
		assert.Less(t, renderTime, 50*time.Millisecond, "Render with running process should be <50ms")
	})

	t.Run("render_with_high_log_volume", func(t *testing.T) {
		// Switch to logs view (may have high volume if process is running)
		ctx.SendKeyMsg("2")
		time.Sleep(100 * time.Millisecond)

		// Measure render time with potentially many logs
		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Should render logs view")
		assert.Less(t, renderTime, 50*time.Millisecond, "Logs render with high volume should be <50ms")
	})

	t.Run("render_with_scroll_position", func(t *testing.T) {
		// Switch to logs view
		ctx.SendKeyMsg("2")
		time.Sleep(50 * time.Millisecond)

		// Scroll down multiple times
		for i := 0; i < 10; i++ {
			ctx.SendKeyMsg("down")
			time.Sleep(10 * time.Millisecond)
		}

		// Measure render time at scrolled position
		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Should render at scroll position")
		assert.Less(t, renderTime, 50*time.Millisecond, "Render at scroll position should be <50ms")
	})
}

// TestRenderPerformanceWindowSize tests rendering at different window sizes
func TestRenderPerformanceWindowSize(t *testing.T) {
	SkipIfShort(t, "Performance test for rendering at different window sizes")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	windowSizes := []struct {
		name   string
		width  int
		height int
	}{
		{"minimal", 20, 10},
		{"small", 80, 24},
		{"medium", 120, 40},
		{"large", 200, 60},
		{"wide", 300, 40},
		{"tall", 120, 100},
	}

	for _, size := range windowSizes {
		t.Run(size.name, func(t *testing.T) {
			ctx.SetWindowSize(size.width, size.height)
			time.Sleep(50 * time.Millisecond)

			// Measure render time at this window size
			start := time.Now()
			view := ctx.GetCurrentView()
			renderTime := time.Since(start)

			assert.NotEmpty(t, view, "Should render at %s size", size.name)
			assert.Less(t, renderTime, 50*time.Millisecond, "Render at %s size should be <50ms", size.name)
			t.Logf("%s (%dx%d) render time: %v", size.name, size.width, size.height, renderTime)
		})
	}
}

// TestRenderConsistency tests that render times remain consistent
func TestRenderConsistency(t *testing.T) {
	SkipIfShort(t, "Performance test for render consistency")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("render_time_stability", func(t *testing.T) {
		// Measure render time over multiple iterations
		const iterations = 20
		renderTimes := make([]time.Duration, iterations)

		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			view := ctx.GetCurrentView()
			renderTimes[i] = time.Since(start)
			assert.NotEmpty(t, view, "View should render on iteration %d", i)
			time.Sleep(20 * time.Millisecond)
		}

		// Calculate statistics
		var total time.Duration
		for _, rt := range renderTimes {
			total += rt
		}
		avgRenderTime := total / time.Duration(iterations)

		// All render times should be reasonable
		for i, rt := range renderTimes {
			assert.Less(t, rt, 50*time.Millisecond, "Render time iteration %d should be <50ms", i)
		}

		t.Logf("Average render time over %d iterations: %v", iterations, avgRenderTime)
	})

	t.Run("no_render_degradation", func(t *testing.T) {
		// Test that render time doesn't degrade over time
		const initialSamples = 5
		const finalSamples = 5
		const operationsInBetween = 50

		// Initial render times
		initialTimes := make([]time.Duration, initialSamples)
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)

		for i := 0; i < initialSamples; i++ {
			start := time.Now()
			ctx.GetCurrentView()
			initialTimes[i] = time.Since(start)
			time.Sleep(20 * time.Millisecond)
		}

		// Do many operations
		for i := 0; i < operationsInBetween; i++ {
			ctx.SendKeyMsg("down")
			ctx.GetCurrentView()
			time.Sleep(10 * time.Millisecond)
		}

		// Final render times
		finalTimes := make([]time.Duration, finalSamples)
		for i := 0; i < finalSamples; i++ {
			start := time.Now()
			ctx.GetCurrentView()
			finalTimes[i] = time.Since(start)
			time.Sleep(20 * time.Millisecond)
		}

		// Calculate averages
		var initialTotal, finalTotal time.Duration
		for i := 0; i < 5; i++ {
			initialTotal += initialTimes[i]
			finalTotal += finalTimes[i]
		}
		initialAvg := initialTotal / initialSamples
		finalAvg := finalTotal / finalSamples

		// Final render time should not be significantly worse
		// Allow 20% degradation tolerance
		maxAcceptable := initialAvg + (initialAvg / 5)
		assert.Less(t, finalAvg, maxAcceptable, "Render time should not degrade significantly")

		t.Logf("Initial avg render time: %v, Final avg render time: %v", initialAvg, finalAvg)
	})
}

// TestRenderContentSize tests rendering performance with varying content sizes
func TestRenderContentSize(t *testing.T) {
	SkipIfShort(t, "Performance test for rendering different content sizes")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("empty_view_render", func(t *testing.T) {
		// Render view with no content (e.g., no processes running)
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Empty view should render")
		assert.Less(t, renderTime, 50*time.Millisecond, "Empty view render should be <50ms")
	})

	t.Run("view_with_content", func(t *testing.T) {
		// Start a process to add content
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Render with content
		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "View with content should render")
		assert.Less(t, renderTime, 50*time.Millisecond, "View with content render should be <50ms")
	})
}

// TestRenderBenchmark provides detailed rendering benchmarks
func TestRenderBenchmark(t *testing.T) {
	SkipIfShort(t, "Performance benchmark for view rendering")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("comprehensive_render_report", func(t *testing.T) {
		// Comprehensive render time measurement across all views
		const samplesPerView = 10

		views := map[string]string{
			"processes": "1",
			"logs":      "2",
			"errors":    "3",
			"urls":      "4",
			"proxy":     "5",
			"settings":  "8",
		}

		results := make(map[string]time.Duration)

		for viewName, viewKey := range views {
			var totalRenderTime time.Duration

			ctx.SendKeyMsg(viewKey)
			time.Sleep(50 * time.Millisecond)

			for i := 0; i < samplesPerView; i++ {
				start := time.Now()
				view := ctx.GetCurrentView()
				renderTime := time.Since(start)
				totalRenderTime += renderTime

				assert.NotEmpty(t, view, "%s view should render (sample %d)", viewName, i)
				time.Sleep(20 * time.Millisecond)
			}

			avgRenderTime := totalRenderTime / samplesPerView
			results[viewName] = avgRenderTime

			t.Logf("%s view - Average render time: %v (across %d samples)", viewName, avgRenderTime, samplesPerView)
			assert.Less(t, avgRenderTime, 50*time.Millisecond, "%s view render should be <50ms on average", viewName)
		}

		// Overall statistics
		var overallTotal time.Duration
		var count int
		for _, rt := range results {
			overallTotal += rt
			count++
		}
		overallAvg := overallTotal / time.Duration(count)
		t.Logf("Overall average render time across all views: %v", overallAvg)
	})

	t.Run("render_size_analysis", func(t *testing.T) {
		// Analyze relationship between window size and render time
		sizes := []struct{ w, h int }{
			{80, 24},
			{120, 40},
			{160, 50},
			{200, 60},
		}

		for _, size := range sizes {
			ctx.SetWindowSize(size.w, size.h)
			time.Sleep(50 * time.Millisecond)

			start := time.Now()
			view := ctx.GetCurrentView()
			renderTime := time.Since(start)

			viewSize := len(view)
			t.Logf("Window %dx%d - Render time: %v, Output size: %d chars", size.w, size.h, renderTime, viewSize)
			assert.Less(t, renderTime, 50*time.Millisecond, "Render at %dx%d should be <50ms", size.w, size.h)
		}
	})
}

// TestRenderEdgeCases tests rendering in edge case scenarios
func TestRenderEdgeCases(t *testing.T) {
	SkipIfShort(t, "Performance test for render edge cases")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("render_after_rapid_resize", func(t *testing.T) {
		// Rapidly resize window
		sizes := []struct{ w, h int }{
			{80, 24},
			{120, 40},
			{100, 30},
			{140, 50},
		}

		for _, size := range sizes {
			ctx.SetWindowSize(size.w, size.h)
			time.Sleep(10 * time.Millisecond)
		}

		// Test render immediately after rapid resizing
		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		assert.NotEmpty(t, view, "Should render after rapid resize")
		assert.Less(t, renderTime, 50*time.Millisecond, "Render after rapid resize should be <50ms")
	})

	t.Run("render_with_special_characters", func(t *testing.T) {
		// Views might contain special characters (colors, unicode, etc.)
		ctx.SendKeyMsg("2") // Logs view likely has colored output
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		view := ctx.GetCurrentView()
		renderTime := time.Since(start)

		// Check for ANSI escape sequences or unicode
		hasSpecialChars := strings.Contains(view, "\x1b") || len(view) != len([]rune(view))

		assert.NotEmpty(t, view, "Should render with special characters")
		assert.Less(t, renderTime, 50*time.Millisecond, "Render with special chars should be <50ms")
		if hasSpecialChars {
			t.Logf("View contains special characters (ANSI/Unicode)")
		}
	})
}
