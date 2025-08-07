package tui

import (
	"fmt"
	"time"
)

// LockFreeConfig contains configuration parameters for the lock-free PTY system
type LockFreeConfig struct {
	// Buffer configuration
	BufferSize    uint64        `toml:"buffer_size"`     // Ring buffer size (power of 2)
	BatchSize     int           `toml:"batch_size"`      // Messages per batch
	PollInterval  time.Duration `toml:"poll_interval"`   // Polling frequency for writer loop
	
	// Performance tuning
	EnableDirectPassthrough bool          `toml:"enable_direct_passthrough"` // Use direct PTY-to-terminal when possible
	VT10XOptimizations     bool          `toml:"vt10x_optimizations"`      // Enable vt10x performance optimizations
	MaxBufferSizeMB        float64       `toml:"max_buffer_size_mb"`       // Maximum buffer memory usage
	
	// Backpressure handling
	DropPolicy          DropPolicy `toml:"drop_policy"`           // How to handle buffer overflow
	BackpressureLogRate int        `toml:"backpressure_log_rate"`  // Log every N dropped messages
	
	// Monitoring
	MetricsEnabled bool `toml:"metrics_enabled"` // Enable performance metrics collection
}

// DropPolicy defines how to handle buffer overflow
type DropPolicy uint8

const (
	DropOldest DropPolicy = iota // Drop oldest messages (default)
	DropNewest                   // Drop newest messages
	DropBatch                    // Drop entire batches
)

// DefaultLockFreeConfig returns the default configuration optimized for typing responsiveness
func DefaultLockFreeConfig() LockFreeConfig {
	return LockFreeConfig{
		BufferSize:              16384,                  // 16K messages
		BatchSize:               64,                     // 64 messages per batch
		PollInterval:            1 * time.Millisecond,   // 1000 Hz polling
		EnableDirectPassthrough: true,                   // Use direct passthrough
		VT10XOptimizations:     true,                   // Enable vt10x optimizations
		MaxBufferSizeMB:        64.0,                   // 64MB max
		DropPolicy:             DropOldest,             // Drop oldest on overflow
		BackpressureLogRate:    100,                    // Log every 100 drops
		MetricsEnabled:         true,                   // Enable metrics by default
	}
}

// HighPerformanceConfig returns configuration optimized for minimal latency
func HighPerformanceConfig() LockFreeConfig {
	return LockFreeConfig{
		BufferSize:              32768,                        // 32K messages
		BatchSize:               128,                          // Larger batches
		PollInterval:            500 * time.Microsecond,       // 2000 Hz polling
		EnableDirectPassthrough: true,                         // Always use direct passthrough
		VT10XOptimizations:     true,                         // All optimizations
		MaxBufferSizeMB:        128.0,                        // More memory
		DropPolicy:             DropNewest,                   // Keep established flow
		BackpressureLogRate:    500,                          // Less logging overhead
		MetricsEnabled:         true,
	}
}

// LowMemoryConfig returns configuration for memory-constrained environments
func LowMemoryConfig() LockFreeConfig {
	return LockFreeConfig{
		BufferSize:              4096,              // 4K messages
		BatchSize:               32,                // Smaller batches
		PollInterval:            2 * time.Millisecond, // 500 Hz polling
		EnableDirectPassthrough: true,              // Still use passthrough
		VT10XOptimizations:     false,             // Disable optimizations
		MaxBufferSizeMB:        16.0,              // Limited memory
		DropPolicy:             DropBatch,         // Drop entire batches
		BackpressureLogRate:    50,                // More frequent logging
		MetricsEnabled:         false,             // Reduce overhead
	}
}

// TypingOptimizedConfig returns configuration optimized for interactive typing (minimal latency)
func TypingOptimizedConfig() LockFreeConfig {
	return LockFreeConfig{
		BufferSize:              2048,                        // 2K messages for ultra-low latency
		BatchSize:               4,                           // Very small batches for immediate responsiveness
		PollInterval:            250 * time.Microsecond,      // 4000 Hz polling for sub-frame latency
		EnableDirectPassthrough: true,                        // Direct passthrough for minimal latency
		VT10XOptimizations:     true,                        // All optimizations for speed
		MaxBufferSizeMB:        8.0,                         // Minimal memory usage
		DropPolicy:             DropNewest,                  // Keep current typing flow
		BackpressureLogRate:    1000,                        // Minimal logging
		MetricsEnabled:         false,                       // Disable metrics for max performance
	}
}

// ValidateConfig ensures configuration is valid and safe
func (c *LockFreeConfig) ValidateConfig() error {
	// Validate buffer size is power of 2
	if c.BufferSize == 0 || (c.BufferSize&(c.BufferSize-1)) != 0 {
		c.BufferSize = nextPowerOfTwo(c.BufferSize)
	}

	// Validate batch size
	if c.BatchSize <= 0 || c.BatchSize > int(c.BufferSize/2) {
		c.BatchSize = int(c.BufferSize / 4) // Default to 1/4 buffer size
	}

	// Validate poll interval
	if c.PollInterval < 100*time.Microsecond {
		c.PollInterval = 100 * time.Microsecond // Floor
	}
	if c.PollInterval > 100*time.Millisecond {
		c.PollInterval = 100 * time.Millisecond // Ceiling
	}

	// Validate memory limits
	if c.MaxBufferSizeMB <= 0 || c.MaxBufferSizeMB > 1024 {
		c.MaxBufferSizeMB = 64.0 // Default to 64MB
	}

	// Validate backpressure log rate
	if c.BackpressureLogRate <= 0 {
		c.BackpressureLogRate = 100
	}

	return nil
}

// GetBufferSizeMB estimates buffer memory usage in MB
func (c *LockFreeConfig) GetBufferSizeMB() float64 {
	// Estimate PTYMessage size:
	// - SessionID: ~32 bytes (typical UUID/timestamp)
	// - Data: ~1KB average (terminal output varies widely)
	// - Timestamp: 24 bytes
	// - Type: 1 byte
	// - Sequence: 8 bytes
	// - Pointer overhead: 8 bytes
	// Total: ~1073 bytes per message
	const avgMessageSize = 1073
	
	totalBytes := float64(c.BufferSize * avgMessageSize)
	return totalBytes / (1024 * 1024) // Convert to MB
}

// GetThroughputEstimate returns estimated throughput in messages/second
func (c *LockFreeConfig) GetThroughputEstimate() float64 {
	// Calculate based on poll interval and batch size
	batchesPerSecond := 1.0 / c.PollInterval.Seconds()
	return batchesPerSecond * float64(c.BatchSize)
}

// String returns a human-readable description of the configuration
func (c *LockFreeConfig) String() string {
	policyName := map[DropPolicy]string{
		DropOldest: "drop-oldest",
		DropNewest: "drop-newest", 
		DropBatch:  "drop-batch",
	}[c.DropPolicy]

	return fmt.Sprintf("LockFreeConfig{buffer=%d, batch=%d, poll=%v, memory=%.1fMB, "+
		"passthrough=%t, vt10x=%t, drop=%s}", 
		c.BufferSize, c.BatchSize, c.PollInterval, c.GetBufferSizeMB(),
		c.EnableDirectPassthrough, c.VT10XOptimizations, policyName)
}