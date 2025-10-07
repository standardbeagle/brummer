package integration

import (
	"fmt"
	"sort"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// PerformanceMetrics holds performance measurement results
type PerformanceMetrics struct {
	Samples      []time.Duration
	Count        int
	Min          time.Duration
	Max          time.Duration
	Mean         time.Duration
	Median       time.Duration
	P95          time.Duration
	P99          time.Duration
	TotalTime    time.Duration
}

// PerformanceTest defines a performance test configuration
type PerformanceTest struct {
	Name           string
	Setup          func(*TestContext)
	Operation      func(*TestContext) time.Duration
	Iterations     int
	Target         time.Duration  // Target latency (e.g., 50ms)
	FailThreshold  time.Duration  // Fail if P95 exceeds this
}

// MeasureKeyEventLatency measures the latency of key event processing
func MeasureKeyEventLatency(ctx *TestContext, key string) time.Duration {
	start := time.Now()

	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(key),
	}

	// Send key event
	newModel, cmd := ctx.Model.Update(msg)
	ctx.Model = newModel.(*tui.Model)

	// Execute command if returned
	if cmd != nil {
		cmd()
	}

	return time.Since(start)
}

// MeasureViewRenderLatency measures the latency of view rendering
func MeasureViewRenderLatency(ctx *TestContext) time.Duration {
	start := time.Now()
	_ = ctx.Model.View()
	return time.Since(start)
}

// MeasureServiceCallLatency measures the latency of a service call
func MeasureServiceCallLatency(operation func()) time.Duration {
	start := time.Now()
	operation()
	return time.Since(start)
}

// RunPerformanceTest runs a performance test and returns metrics
func RunPerformanceTest(t *testing.T, test PerformanceTest) *PerformanceMetrics {
	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Run setup if provided
	if test.Setup != nil {
		test.Setup(ctx)
	}

	// Collect samples
	samples := make([]time.Duration, test.Iterations)
	totalStart := time.Now()

	for i := 0; i < test.Iterations; i++ {
		samples[i] = test.Operation(ctx)
	}

	totalTime := time.Since(totalStart)

	// Calculate metrics
	metrics := calculateMetrics(samples, totalTime)

	// Log results
	t.Logf("Performance Test: %s", test.Name)
	t.Logf("  Iterations: %d", metrics.Count)
	t.Logf("  Min:        %v", metrics.Min)
	t.Logf("  Mean:       %v", metrics.Mean)
	t.Logf("  Median:     %v", metrics.Median)
	t.Logf("  P95:        %v", metrics.P95)
	t.Logf("  P99:        %v", metrics.P99)
	t.Logf("  Max:        %v", metrics.Max)
	t.Logf("  Total Time: %v", metrics.TotalTime)

	// Check against target
	if test.Target > 0 && metrics.P95 > test.Target {
		t.Logf("  ⚠️  P95 (%v) exceeds target (%v)", metrics.P95, test.Target)
	}

	// Fail if threshold exceeded
	if test.FailThreshold > 0 && metrics.P95 > test.FailThreshold {
		t.Errorf("  ❌ P95 latency (%v) exceeds fail threshold (%v)", metrics.P95, test.FailThreshold)
	}

	return metrics
}

// calculateMetrics computes performance metrics from samples
func calculateMetrics(samples []time.Duration, totalTime time.Duration) *PerformanceMetrics {
	if len(samples) == 0 {
		return &PerformanceMetrics{}
	}

	// Sort samples for percentile calculation
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate sum for mean
	var sum time.Duration
	for _, s := range samples {
		sum += s
	}

	metrics := &PerformanceMetrics{
		Samples:   samples,
		Count:     len(samples),
		Min:       sorted[0],
		Max:       sorted[len(sorted)-1],
		Mean:      sum / time.Duration(len(samples)),
		Median:    sorted[len(sorted)/2],
		P95:       sorted[int(float64(len(sorted))*0.95)],
		P99:       sorted[int(float64(len(sorted))*0.99)],
		TotalTime: totalTime,
	}

	return metrics
}

// KeyEventLatencyBaseline establishes baseline key event latency
func KeyEventLatencyBaseline(t *testing.T, ctx *TestContext, iterations int) *PerformanceMetrics {
	test := PerformanceTest{
		Name:       "Key Event Latency Baseline",
		Iterations: iterations,
		Target:     50 * time.Millisecond,  // <50ms requirement
		FailThreshold: 100 * time.Millisecond,
		Operation: func(ctx *TestContext) time.Duration {
			return MeasureKeyEventLatency(ctx, "j") // Down arrow in vim mode
		},
	}

	return RunPerformanceTest(t, test)
}

// ViewRenderingBaseline establishes baseline view rendering performance
func ViewRenderingBaseline(t *testing.T, ctx *TestContext, iterations int) *PerformanceMetrics {
	test := PerformanceTest{
		Name:       "View Rendering Baseline",
		Iterations: iterations,
		Target:     16 * time.Millisecond,  // <16ms for 60fps
		FailThreshold: 33 * time.Millisecond,  // 30fps minimum
		Operation: func(ctx *TestContext) time.Duration {
			return MeasureViewRenderLatency(ctx)
		},
	}

	return RunPerformanceTest(t, test)
}

// ViewSwitchLatency measures view switching performance
func ViewSwitchLatency(t *testing.T, ctx *TestContext, iterations int) *PerformanceMetrics {
	viewKeys := []string{"1", "2", "3", "4", "5"}
	currentView := 0

	test := PerformanceTest{
		Name:       "View Switch Latency",
		Iterations: iterations,
		Target:     50 * time.Millisecond,
		FailThreshold: 100 * time.Millisecond,
		Operation: func(ctx *TestContext) time.Duration {
			key := viewKeys[currentView%len(viewKeys)]
			currentView++
			return MeasureKeyEventLatency(ctx, key)
		},
	}

	return RunPerformanceTest(t, test)
}

// ProcessStartLatency measures process startup performance
func ProcessStartLatency(t *testing.T, ctx *TestContext, iterations int) *PerformanceMetrics {
	test := PerformanceTest{
		Name:       "Process Start Latency",
		Iterations: iterations,
		Target:     100 * time.Millisecond,  // Process start can be slower
		FailThreshold: 500 * time.Millisecond,
		Setup: func(ctx *TestContext) {
			// Create test scripts
		},
		Operation: func(ctx *TestContext) time.Duration {
			return MeasureServiceCallLatency(func() {
				// Start a quick test process
				ctx.ProcessMgr.StartScript("test")
				// Note: This measures service call latency, not full startup
			})
		},
	}

	return RunPerformanceTest(t, test)
}

// CompareMetrics compares two performance metrics and returns difference
func CompareMetrics(before, after *PerformanceMetrics) string {
	p95Diff := after.P95 - before.P95
	p95Change := (float64(after.P95) / float64(before.P95)) * 100.0

	result := fmt.Sprintf("Performance Comparison:\n")
	result += fmt.Sprintf("  Before P95: %v\n", before.P95)
	result += fmt.Sprintf("  After P95:  %v\n", after.P95)
	result += fmt.Sprintf("  Difference: %v (%.1f%%)\n", p95Diff, p95Change)

	if p95Change > 110.0 {
		result += fmt.Sprintf("  ⚠️  Performance regression detected (>10%% slower)\n")
	} else if p95Change < 90.0 {
		result += fmt.Sprintf("  ✅ Performance improvement (>10%% faster)\n")
	} else {
		result += fmt.Sprintf("  ✅ Performance stable\n")
	}

	return result
}

// MemoryUsageTest measures memory usage during operations
type MemoryUsageTest struct {
	Name      string
	Setup     func(*TestContext)
	Operation func(*TestContext)
	Iterations int
}

// EstimateMemoryUsage provides basic memory usage estimation
// Note: For detailed memory profiling, use pprof
func EstimateMemoryUsage(t *testing.T, test MemoryUsageTest) {
	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	if test.Setup != nil {
		test.Setup(ctx)
	}

	// Run operation multiple times
	for i := 0; i < test.Iterations; i++ {
		test.Operation(ctx)
	}

	// Note: Actual memory profiling requires runtime/pprof
	t.Logf("Memory usage test '%s' completed %d iterations", test.Name, test.Iterations)
	t.Logf("  Use 'go test -memprofile=mem.prof' for detailed memory analysis")
}

// BenchmarkOperation runs a simple benchmark
func BenchmarkOperation(b *testing.B, operation func(*TestContext)) {
	ctx := NewTestContext(&testing.T{})
	defer ctx.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		operation(ctx)
	}
}

// PerformanceReport generates a comprehensive performance report
type PerformanceReport struct {
	TestName         string
	KeyEventLatency  *PerformanceMetrics
	ViewRendering    *PerformanceMetrics
	ViewSwitch       *PerformanceMetrics
	ProcessStart     *PerformanceMetrics
	Timestamp        time.Time
}

// GeneratePerformanceReport creates a full performance report
func GeneratePerformanceReport(t *testing.T, ctx *TestContext) *PerformanceReport {
	t.Log("Generating comprehensive performance report...")

	report := &PerformanceReport{
		TestName:  t.Name(),
		Timestamp: time.Now(),
	}

	// Run all performance tests
	report.KeyEventLatency = KeyEventLatencyBaseline(t, ctx, 100)
	report.ViewRendering = ViewRenderingBaseline(t, ctx, 100)
	report.ViewSwitch = ViewSwitchLatency(t, ctx, 50)
	report.ProcessStart = ProcessStartLatency(t, ctx, 10)

	return report
}

// FormatPerformanceReport formats a report as string
func FormatPerformanceReport(report *PerformanceReport) string {
	result := fmt.Sprintf("\n=== Performance Report: %s ===\n", report.TestName)
	result += fmt.Sprintf("Generated: %s\n\n", report.Timestamp.Format(time.RFC3339))

	if report.KeyEventLatency != nil {
		result += fmt.Sprintf("Key Event Latency:\n")
		result += fmt.Sprintf("  P95: %v (target: <50ms)\n", report.KeyEventLatency.P95)
		result += fmt.Sprintf("  P99: %v\n\n", report.KeyEventLatency.P99)
	}

	if report.ViewRendering != nil {
		result += fmt.Sprintf("View Rendering:\n")
		result += fmt.Sprintf("  P95: %v (target: <16ms for 60fps)\n", report.ViewRendering.P95)
		result += fmt.Sprintf("  P99: %v\n\n", report.ViewRendering.P99)
	}

	if report.ViewSwitch != nil {
		result += fmt.Sprintf("View Switching:\n")
		result += fmt.Sprintf("  P95: %v (target: <50ms)\n", report.ViewSwitch.P95)
		result += fmt.Sprintf("  P99: %v\n\n", report.ViewSwitch.P99)
	}

	if report.ProcessStart != nil {
		result += fmt.Sprintf("Process Start:\n")
		result += fmt.Sprintf("  P95: %v (target: <100ms)\n", report.ProcessStart.P95)
		result += fmt.Sprintf("  P99: %v\n\n", report.ProcessStart.P99)
	}

	result += fmt.Sprintf("=== End Report ===\n")

	return result
}
