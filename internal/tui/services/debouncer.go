package services

import (
	"sync"
	"time"
)

// PTYDebouncer batches PTY output updates to prevent flooding the UI.
// It coalesces rapid updates into batches at a configurable interval.
// Default is 16ms for 60 FPS rendering.
type PTYDebouncer struct {
	interval  time.Duration                            // Debounce interval (16ms for 60 FPS)
	callback  func(sessionID string, data []byte)     // Callback for batched data
	sessions  sync.Map                                 // Session-specific buffers
	smartMode bool                                     // Enable smart mode (adaptive rates)
}

// sessionDebouncer tracks pending data for a specific session
type sessionDebouncer struct {
	sessionID string
	pending   []byte      // Accumulated data waiting to be sent
	mu        sync.Mutex  // Protects pending buffer
	timer     *time.Timer // Debounce timer
	isActive  bool        // Whether this session is currently active/focused
	lastSend  time.Time   // Last time we sent data (for rate limiting)
}

// NewPTYDebouncer creates a new debouncer with the specified interval.
// For 60 FPS, use 16 * time.Millisecond.
func NewPTYDebouncer(interval time.Duration, callback func(string, []byte)) *PTYDebouncer {
	if interval <= 0 {
		interval = 16 * time.Millisecond // Default to 60 FPS
	}
	return &PTYDebouncer{
		interval:  interval,
		callback:  callback,
		smartMode: true, // Enable smart mode by default
	}
}

// Add adds data to be debounced for a specific session.
// The data will be batched and sent after the debounce interval.
func (d *PTYDebouncer) Add(sessionID string, data []byte) {
	if len(data) == 0 {
		return
	}

	// Get or create session debouncer
	value, _ := d.sessions.LoadOrStore(sessionID, &sessionDebouncer{
		sessionID: sessionID,
		pending:   make([]byte, 0, 4096), // Start with 4KB buffer
		isActive:  true,                  // Assume active by default
		lastSend:  time.Now(),
	})

	sd := value.(*sessionDebouncer)

	sd.mu.Lock()
	defer sd.mu.Unlock()

	// Append data to pending buffer
	sd.pending = append(sd.pending, data...)

	// Determine debounce interval based on smart mode
	interval := d.getInterval(sd)

	// Cancel existing timer if present
	if sd.timer != nil {
		sd.timer.Stop()
	}

	// Start new timer with appropriate interval
	sd.timer = time.AfterFunc(interval, func() {
		d.flush(sessionID)
	})
}

// SetSessionActive marks a session as active or inactive for smart mode.
// Active sessions get 16ms updates (60 FPS), background get 100ms (10 FPS).
func (d *PTYDebouncer) SetSessionActive(sessionID string, active bool) {
	value, exists := d.sessions.Load(sessionID)
	if !exists {
		return
	}

	sd := value.(*sessionDebouncer)
	sd.mu.Lock()
	sd.isActive = active
	sd.mu.Unlock()

	// If becoming active and has pending data, flush immediately
	if active && len(sd.pending) > 0 {
		d.flush(sessionID)
	}
}

// flush sends all pending data for a session
func (d *PTYDebouncer) flush(sessionID string) {
	value, exists := d.sessions.Load(sessionID)
	if !exists {
		return
	}

	sd := value.(*sessionDebouncer)

	sd.mu.Lock()
	defer sd.mu.Unlock()

	// Nothing to flush
	if len(sd.pending) == 0 {
		return
	}

	// Make a copy of the data to send
	data := make([]byte, len(sd.pending))
	copy(data, sd.pending)

	// Clear pending buffer
	sd.pending = sd.pending[:0]

	// Clear timer
	if sd.timer != nil {
		sd.timer.Stop()
		sd.timer = nil
	}

	// Update last send time
	sd.lastSend = time.Now()

	// Send data via callback (outside of lock)
	go d.callback(sessionID, data)
}

// getInterval returns the appropriate debounce interval for a session.
// Implements smart mode: active = 16ms, background = 100ms, suspended = 1000ms.
func (d *PTYDebouncer) getInterval(sd *sessionDebouncer) time.Duration {
	if !d.smartMode {
		return d.interval
	}

	// Active session: 60 FPS (16ms)
	if sd.isActive {
		return 16 * time.Millisecond
	}

	// Background session: check how long since last update
	timeSinceLastSend := time.Since(sd.lastSend)

	// If recently active (< 1 second), use moderate rate
	if timeSinceLastSend < 1*time.Second {
		return 100 * time.Millisecond // 10 FPS for background
	}

	// Suspended/idle session: very slow rate
	return 1000 * time.Millisecond // 1 FPS for idle
}

// FlushAll immediately flushes all pending data for all sessions.
// Useful for cleanup or immediate updates.
func (d *PTYDebouncer) FlushAll() {
	d.sessions.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		d.flush(sessionID)
		return true
	})
}

// RemoveSession cleans up resources for a session.
func (d *PTYDebouncer) RemoveSession(sessionID string) {
	value, exists := d.sessions.LoadAndDelete(sessionID)
	if !exists {
		return
	}

	sd := value.(*sessionDebouncer)
	sd.mu.Lock()
	defer sd.mu.Unlock()

	// Stop timer if running
	if sd.timer != nil {
		sd.timer.Stop()
		sd.timer = nil
	}

	// Clear pending data
	sd.pending = nil
}

// SetSmartMode enables or disables smart mode.
// When enabled, inactive sessions get slower update rates.
func (d *PTYDebouncer) SetSmartMode(enabled bool) {
	d.smartMode = enabled
}

// GetStats returns statistics about the debouncer.
func (d *PTYDebouncer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"interval":   d.interval.String(),
		"smart_mode": d.smartMode,
		"sessions":   0,
		"pending":    0,
	}

	sessionCount := 0
	totalPending := 0

	d.sessions.Range(func(key, value interface{}) bool {
		sessionCount++
		sd := value.(*sessionDebouncer)
		sd.mu.Lock()
		totalPending += len(sd.pending)
		sd.mu.Unlock()
		return true
	})

	stats["sessions"] = sessionCount
	stats["pending"] = totalPending

	return stats
}

// === ADAPTIVE DEBOUNCER ===
// This variant automatically adjusts the debounce interval based on output rate.

// AdaptivePTYDebouncer extends PTYDebouncer with dynamic interval adjustment.
type AdaptivePTYDebouncer struct {
	*PTYDebouncer
	minInterval time.Duration // Minimum interval (highest rate)
	maxInterval time.Duration // Maximum interval (lowest rate)
}

// NewAdaptivePTYDebouncer creates a debouncer that adapts to output rate.
// It speeds up for high-throughput output and slows down when idle.
func NewAdaptivePTYDebouncer(callback func(string, []byte)) *AdaptivePTYDebouncer {
	base := NewPTYDebouncer(16*time.Millisecond, callback)
	return &AdaptivePTYDebouncer{
		PTYDebouncer: base,
		minInterval:  8 * time.Millisecond,  // 125 FPS max
		maxInterval:  100 * time.Millisecond, // 10 FPS min
	}
}

// getAdaptiveInterval calculates interval based on data rate.
func (a *AdaptivePTYDebouncer) getAdaptiveInterval(sd *sessionDebouncer) time.Duration {
	sd.mu.Lock()
	pendingSize := len(sd.pending)
	timeSinceLastSend := time.Since(sd.lastSend)
	sd.mu.Unlock()

	// High data rate: use minimum interval
	if pendingSize > 10000 { // > 10KB pending
		return a.minInterval
	}

	// Medium data rate: scale linearly
	if pendingSize > 1000 { // 1-10KB pending
		ratio := float64(pendingSize-1000) / 9000.0
		intervalRange := float64(a.maxInterval - a.minInterval)
		scaledInterval := a.maxInterval - time.Duration(ratio*intervalRange)
		return scaledInterval
	}

	// Low data rate or idle: use maximum interval
	if timeSinceLastSend > 100*time.Millisecond {
		return a.maxInterval
	}

	// Default to base interval
	return a.PTYDebouncer.interval
}