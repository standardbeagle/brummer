package performance

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync/atomic"
	"time"
)

// Profiler provides simple integration with Go's built-in profiling
type Profiler struct {
	sessionID   string
	outputDir   string
	collecting  int32 // 0 = not collecting, 1 = collecting
	cpuFile     *os.File
	memFile     *os.File
	startTime   time.Time
	sampleCount int64
}

// ProfilerConfig configures profiler behavior
type ProfilerConfig struct {
	SessionID      string
	OutputDir      string        // Directory for profile files (defaults to .)
	AutoStart      bool          // Start profiling immediately
	SampleInterval time.Duration // How often to sample (for custom sampling)
}

// NewProfiler creates a new profiler instance
func NewProfiler(config ProfilerConfig) *Profiler {
	if config.OutputDir == "" {
		config.OutputDir = "."
	}

	if config.SessionID == "" {
		config.SessionID = fmt.Sprintf("profile-%d", time.Now().Unix())
	}

	profiler := &Profiler{
		sessionID: config.SessionID,
		outputDir: config.OutputDir,
	}

	if config.AutoStart {
		profiler.Start()
	}

	return profiler
}

// Start begins CPU profiling
func (p *Profiler) Start() error {
	if !atomic.CompareAndSwapInt32(&p.collecting, 0, 1) {
		return fmt.Errorf("profiler already running")
	}

	p.startTime = time.Now()
	atomic.StoreInt64(&p.sampleCount, 0)

	// Start CPU profiling
	cpuFile, err := os.Create(fmt.Sprintf("%s/%s_cpu_%s.prof", p.outputDir, p.sessionID, time.Now().Format("20060102-150405")))
	if err != nil {
		atomic.StoreInt32(&p.collecting, 0)
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}
	p.cpuFile = cpuFile

	if err := pprof.StartCPUProfile(p.cpuFile); err != nil {
		cpuFile.Close()
		atomic.StoreInt32(&p.collecting, 0)
		return fmt.Errorf("failed to start CPU profiling: %w", err)
	}

	return nil
}

// Stop ends profiling and saves memory profile
func (p *Profiler) Stop() error {
	if !atomic.CompareAndSwapInt32(&p.collecting, 1, 0) {
		return fmt.Errorf("profiler not running")
	}

	// Stop CPU profiling
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		p.cpuFile.Close()
		p.cpuFile = nil
	}

	// Write memory profile
	memFile, err := os.Create(fmt.Sprintf("%s/%s_mem_%s.prof", p.outputDir, p.sessionID, time.Now().Format("20060102-150405")))
	if err != nil {
		return fmt.Errorf("failed to create memory profile file: %w", err)
	}
	defer memFile.Close()

	runtime.GC() // Get up-to-date statistics
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		return fmt.Errorf("failed to write memory profile: %w", err)
	}
	p.memFile = memFile

	return nil
}

// IsRunning returns true if profiling is active
func (p *Profiler) IsRunning() bool {
	return atomic.LoadInt32(&p.collecting) == 1
}

// SessionID returns the profiler session ID
func (p *Profiler) SessionID() string {
	return p.sessionID
}

// Duration returns how long profiling has been running
func (p *Profiler) Duration() time.Duration {
	if atomic.LoadInt32(&p.collecting) == 1 {
		return time.Since(p.startTime)
	}
	return time.Duration(0)
}

// SampleCount returns the number of samples collected (if using custom sampling)
func (p *Profiler) SampleCount() int64 {
	return atomic.LoadInt64(&p.sampleCount)
}

// RecordSample increments the sample counter (for custom sampling)
func (p *Profiler) RecordSample() {
	if atomic.LoadInt32(&p.collecting) == 1 {
		atomic.AddInt64(&p.sampleCount, 1)
	}
}

// GetStats returns basic profiling statistics
func (p *Profiler) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"sessionID":   p.sessionID,
		"isRunning":   p.IsRunning(),
		"sampleCount": p.SampleCount(),
		"startTime":   p.startTime,
		"duration":    p.Duration(),
	}

	if p.IsRunning() {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		stats["memoryStats"] = map[string]interface{}{
			"alloc":      m.Alloc,
			"totalAlloc": m.TotalAlloc,
			"sys":        m.Sys,
			"numGC":      m.NumGC,
			"goroutines": runtime.NumGoroutine(),
		}
	}

	return stats
}

// CreateBlockProfile creates a goroutine blocking profile
func (p *Profiler) CreateBlockProfile(duration time.Duration) error {
	blockFile, err := os.Create(fmt.Sprintf("%s/%s_block_%s.prof", p.outputDir, p.sessionID, time.Now().Format("20060102-150405")))
	if err != nil {
		return fmt.Errorf("failed to create block profile file: %w", err)
	}
	defer blockFile.Close()

	// Block for the specified duration while collecting profile
	profileDone := make(chan bool)
	go func() {
		pprof.Lookup("block")
		profileDone <- true
	}()

	// Wait for the specified duration
	time.Sleep(duration)
	pprof.Lookup("")

	// Wait for profiling to complete
	select {
	case <-profileDone:
	case <-time.After(5 * time.Second):
		// Timeout
	}

	return blockFile.Close()
}

// CreateTraceProfile creates a trace profile
func (p *Profiler) CreateTraceProfile(duration time.Duration) error {
	traceFile, err := os.Create(fmt.Sprintf("%s/%s_trace_%s.out", p.outputDir, p.sessionID, time.Now().Format("20060102-150405")))
	if err != nil {
		return fmt.Errorf("failed to create trace file: %w", err)
	}
	defer traceFile.Close()

	// Start trace
	if err := trace.Start(traceFile); err != nil {
		return fmt.Errorf("failed to start trace: %w", err)
	}

	// Run for the specified duration
	time.Sleep(duration)

	// Stop trace
	trace.Stop()

	return traceFile.Close()
}

// ProfileFunction profiles a specific function
func (p *Profiler) ProfileFunction(fn func() error) error {
	// Create a CPU profile specifically for this function
	profileSessionID := fmt.Sprintf("%s-func-%d", p.sessionID, time.Now().UnixNano())

	cpuFile, err := os.Create(fmt.Sprintf("%s/%s_%s.prof", p.outputDir, profileSessionID, time.Now().Format("20060102-150405")))
	if err != nil {
		return err
	}
	defer cpuFile.Close()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		return err
	}
	defer pprof.StopCPUProfile()

	// Run the function
	err = fn()

	return err
}

// ProfileWithMemory profiles a function and captures memory usage
func (p *Profiler) ProfileWithMemory(fn func() error) (error, error) {
	var memErr error

	// Create memory profile after function execution
	defer func() {
		memFile, err := os.Create(fmt.Sprintf("%s/%s_mem_%s.prof", p.outputDir, p.sessionID, time.Now().Format("20060102-150405")))
		if err != nil {
			memErr = err
			return
		}
		defer memFile.Close()

		runtime.GC()
		memErr = pprof.WriteHeapProfile(memFile)
	}()

	// Create CPU profile for the function duration
	cpuFile, err := os.Create(fmt.Sprintf("%s/%s_%s.prof", p.outputDir, p.sessionID, time.Now().Format("20060102-150405")))
	if err != nil {
		return err, memErr
	}
	defer cpuFile.Close()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		return err, memErr
	}
	defer pprof.StopCPUProfile()

	// Run the function
	funcErr := fn()

	return funcErr, memErr
}

// PrintProfileSummary prints a summary of the profiling session
func (p *Profiler) PrintProfileSummary() {
	stats := p.GetStats()

	fmt.Printf("=== Profiling Session: %s ===\n", stats["sessionID"])
	fmt.Printf("Duration: %v\n", stats["duration"])
	fmt.Printf("Samples: %d\n", stats["sampleCount"])
	fmt.Printf("Status: %s\n", map[bool]string{true: "Running", false: "Stopped"}[stats["isRunning"].(bool)])

	if memStats, ok := stats["memoryStats"].(map[string]interface{}); ok {
		fmt.Printf("Memory: Alloc=%s, Total=%s, Goroutines=%d\n",
			formatBytes(memStats["alloc"].(uint64)),
			formatBytes(memStats["totalAlloc"].(uint64)),
			memStats["goroutines"].(int))
	}

	fmt.Printf("Profile files saved with session ID: %s\n", p.sessionID)
	fmt.Printf("Analyze with: go tool pprof %s_*.prof\n", p.sessionID)
}

// formatBytes formats bytes into human readable string
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for i := 0; i < 10; i++ {
		if b < uint64(div)*unit {
			break
		}
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Global profiler instance for easy access
var defaultProfiler *Profiler

// StartDefaultProfiler starts the default profiler
func StartDefaultProfiler() error {
	config := ProfilerConfig{
		SessionID: "default",
		OutputDir: ".",
		AutoStart: true,
	}
	defaultProfiler = NewProfiler(config)
	return nil
}

// StopDefaultProfiler stops the default profiler
func StopDefaultProfiler() error {
	if defaultProfiler == nil {
		return fmt.Errorf("no default profiler running")
	}
	return defaultProfiler.Stop()
}

// GetDefaultProfiler returns the default profiler instance
func GetDefaultProfiler() *Profiler {
	return defaultProfiler
}

// ProfileDefaultFunction profiles a function using the default profiler
func ProfileDefaultFunction(fn func() error) error {
	if defaultProfiler == nil {
		if err := StartDefaultProfiler(); err != nil {
			return err
		}
	}
	return defaultProfiler.ProfileFunction(fn)
}

// ProfileDefaultFunctionWithMemory profiles a function with memory using the default profiler
func ProfileDefaultFunctionWithMemory(fn func() error) (error, error) {
	if defaultProfiler == nil {
		if err := StartDefaultProfiler(); err != nil {
			return err, nil
		}
	}
	return defaultProfiler.ProfileWithMemory(fn)
}
