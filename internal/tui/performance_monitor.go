package tui

import (
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceMonitor tracks UI response times and command execution performance
type PerformanceMonitor struct {
	// Metrics
	responseTimeSum   int64  // Sum of response times in nanoseconds
	responseTimeCount int64  // Number of measurements
	maxResponseTime   int64  // Maximum response time observed
	lastResponseTime  int64  // Last response time measured

	// Command metrics
	commandCount      int64  // Total commands executed
	commandErrors     int64  // Commands that failed

	// State for monitoring
	startTime         time.Time // When monitoring started
	lastUpdate        time.Time // Last measurement time

	// Thread-safe access
	mu sync.RWMutex
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}
}

// RecordResponseTime records a UI response time
func (pm *PerformanceMonitor) RecordResponseTime(duration time.Duration) {
	nanos := duration.Nanoseconds()

	atomic.AddInt64(&pm.responseTimeSum, nanos)
	atomic.AddInt64(&pm.responseTimeCount, 1)

	// Update max if this is larger
	for {
		current := atomic.LoadInt64(&pm.maxResponseTime)
		if nanos <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&pm.maxResponseTime, current, nanos) {
			break
		}
	}

	atomic.StoreInt64(&pm.lastResponseTime, nanos)

	pm.mu.Lock()
	pm.lastUpdate = time.Now()
	pm.mu.Unlock()
}

// RecordCommand records a command execution
func (pm *PerformanceMonitor) RecordCommand(success bool) {
	atomic.AddInt64(&pm.commandCount, 1)
	if !success {
		atomic.AddInt64(&pm.commandErrors, 1)
	}
}

// GetAverageResponseTime returns the average response time in milliseconds
func (pm *PerformanceMonitor) GetAverageResponseTime() float64 {
	count := atomic.LoadInt64(&pm.responseTimeCount)
	if count == 0 {
		return 0
	}

	sum := atomic.LoadInt64(&pm.responseTimeSum)
	return float64(sum) / float64(count) / 1e6 // Convert to milliseconds
}

// GetLastResponseTime returns the last response time in milliseconds
func (pm *PerformanceMonitor) GetLastResponseTime() float64 {
	nanos := atomic.LoadInt64(&pm.lastResponseTime)
	return float64(nanos) / 1e6 // Convert to milliseconds
}

// GetMaxResponseTime returns the maximum response time in milliseconds
func (pm *PerformanceMonitor) GetMaxResponseTime() float64 {
	nanos := atomic.LoadInt64(&pm.maxResponseTime)
	return float64(nanos) / 1e6 // Convert to milliseconds
}

// GetCommandMetrics returns command execution metrics
func (pm *PerformanceMonitor) GetCommandMetrics() (total int64, errors int64, errorRate float64) {
	total = atomic.LoadInt64(&pm.commandCount)
	errors = atomic.LoadInt64(&pm.commandErrors)

	if total > 0 {
		errorRate = float64(errors) / float64(total) * 100
	}

	return total, errors, errorRate
}

// IsHealthy checks if performance meets requirements (<50ms response time)
func (pm *PerformanceMonitor) IsHealthy() bool {
	// Check average response time
	avg := pm.GetAverageResponseTime()
	if avg > 50 {
		return false
	}

	// Check last response time
	last := pm.GetLastResponseTime()
	if last > 50 {
		return false
	}

	// Check error rate (<5%)
	_, _, errorRate := pm.GetCommandMetrics()
	if errorRate > 5.0 {
		return false
	}

	return true
}

// Reset resets all metrics
func (pm *PerformanceMonitor) Reset() {
	atomic.StoreInt64(&pm.responseTimeSum, 0)
	atomic.StoreInt64(&pm.responseTimeCount, 0)
	atomic.StoreInt64(&pm.maxResponseTime, 0)
	atomic.StoreInt64(&pm.lastResponseTime, 0)
	atomic.StoreInt64(&pm.commandCount, 0)
	atomic.StoreInt64(&pm.commandErrors, 0)

	pm.mu.Lock()
	pm.startTime = time.Now()
	pm.lastUpdate = time.Now()
	pm.mu.Unlock()
}

// GetUptime returns how long the monitor has been running
func (pm *PerformanceMonitor) GetUptime() time.Duration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return time.Since(pm.startTime)
}

// PerformanceSnapshot represents a snapshot of current performance
type PerformanceSnapshot struct {
	AverageResponseTime float64 `json:"average_response_time_ms"`
	LastResponseTime    float64 `json:"last_response_time_ms"`
	MaxResponseTime     float64 `json:"max_response_time_ms"`
	TotalCommands       int64   `json:"total_commands"`
	CommandErrors       int64   `json:"command_errors"`
	ErrorRate           float64 `json:"error_rate_percent"`
	IsHealthy           bool    `json:"is_healthy"`
	Uptime              string  `json:"uptime"`
	Timestamp           string  `json:"timestamp"`
}

// GetSnapshot returns a snapshot of current performance metrics
func (pm *PerformanceMonitor) GetSnapshot() PerformanceSnapshot {
	totalCommands, commandErrors, errorRate := pm.GetCommandMetrics()

	return PerformanceSnapshot{
		AverageResponseTime: pm.GetAverageResponseTime(),
		LastResponseTime:    pm.GetLastResponseTime(),
		MaxResponseTime:     pm.GetMaxResponseTime(),
		TotalCommands:       totalCommands,
		CommandErrors:       commandErrors,
		ErrorRate:           errorRate,
		IsHealthy:           pm.IsHealthy(),
		Uptime:              pm.GetUptime().String(),
		Timestamp:           time.Now().Format(time.RFC3339),
	}
}