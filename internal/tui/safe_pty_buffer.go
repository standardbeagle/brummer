package tui

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/smallnest/ringbuffer"
)

// SafePTYBuffer implements a high-performance, deadlock-safe ring buffer for PTY messages
// Uses the production-ready smallnest/ringbuffer library (which is already thread-safe)
//
// THREAD SAFETY: The underlying ringbuffer.RingBuffer is thread-safe for Read/Write operations.
// We only use atomic.Pointer to protect against reading/writing after Close().
type SafePTYBuffer struct {
	// Core ring buffer implementation (already thread-safe)
	ringBuffer atomic.Pointer[ringbuffer.RingBuffer]

	// Performance metrics (atomic operations)
	messagesWritten atomic.Uint64
	messagesDropped atomic.Uint64
	bufferFullCount atomic.Uint64
	maxUsedSlots    atomic.Uint64
}

// SafePTYMessage represents a single PTY message with metadata
type SafePTYMessage struct {
	SessionID string
	Data      []byte
	Timestamp time.Time
	Type      SafePTYMessageType
	Sequence  uint64 // For ordering verification
}

// SafePTYMessageType categorizes different types of PTY messages
type SafePTYMessageType uint8

const (
	SafePTYMessageOutput SafePTYMessageType = iota
	SafePTYMessageEvent
	SafePTYMessageClose
	SafePTYMessageResize
)

// NewSafePTYBuffer creates a new safe ring buffer
// Uses production-ready library with proper error handling
func NewSafePTYBuffer(bufferSize uint64) *SafePTYBuffer {
	// Ensure buffer size is reasonable
	if bufferSize < 64 {
		bufferSize = 64
	}
	if bufferSize > 1000000 {
		bufferSize = 1000000 // Limit to 1M messages
	}

	buf := &SafePTYBuffer{}
	buf.ringBuffer.Store(ringbuffer.New(int(bufferSize)))
	return buf
}

// TryWrite attempts to write a message to the buffer
// Returns true if successful, false if buffer is full (backpressure)
// Thread-safe: ringbuffer.Write() is already thread-safe
func (b *SafePTYBuffer) TryWrite(msg *SafePTYMessage) bool {
	// Get current ring buffer (atomic read)
	rb := b.ringBuffer.Load()
	if rb == nil {
		return false // Buffer has been closed
	}

	// Convert PTYMessage to bytes for storage
	data, err := b.serializeMessage(msg)
	if err != nil {
		b.messagesDropped.Add(1)
		return false
	}

	// Write to ring buffer (already thread-safe)
	n, err := rb.Write(data)
	if err != nil {
		// Buffer is full or other error
		b.messagesDropped.Add(1)
		if n == 0 {
			b.bufferFullCount.Add(1)
		}
		return false
	}

	// Update metrics (atomic operations)
	b.messagesWritten.Add(1)
	b.updateMaxUsedSlots(rb)

	return true
}

// TryRead attempts to read a message from the buffer
// Returns message and true if successful, nil and false if buffer is empty
// Thread-safe: ringbuffer.Read() is already thread-safe
func (b *SafePTYBuffer) TryRead() (*SafePTYMessage, bool) {
	// Get current ring buffer (atomic read)
	rb := b.ringBuffer.Load()
	if rb == nil {
		return nil, false // Buffer has been closed
	}

	// Create buffer for reading
	data := make([]byte, 4096) // Max message size
	n, err := rb.Read(data)
	if err != nil {
		// Buffer is empty or other error
		return nil, false
	}

	// Deserialize message
	msg, err := b.deserializeMessage(data[:n])
	if err != nil {
		// Corrupted data, skip
		return nil, false
	}

	return msg, true
}

// TryReadBatch attempts to read up to maxMessages from the buffer
// Returns slice of messages and actual count read
// Thread-safe: Uses TryRead() which is already thread-safe
func (b *SafePTYBuffer) TryReadBatch(maxMessages int) ([]*SafePTYMessage, int) {
	if maxMessages <= 0 {
		return nil, 0
	}

	batch := make([]*SafePTYMessage, 0, maxMessages)

	// Read messages one at a time (TryRead is already thread-safe)
	for len(batch) < maxMessages {
		msg, ok := b.TryRead()
		if !ok {
			break // Buffer empty or closed
		}
		batch = append(batch, msg)
	}

	return batch, len(batch)
}

// GetMetrics returns current buffer performance metrics
// Thread-safe: Uses atomic operations for all reads
func (b *SafePTYBuffer) GetMetrics() SafeBufferMetrics {
	var bufferSize, currentUsed uint64

	// Get current ring buffer (atomic read)
	rb := b.ringBuffer.Load()
	if rb != nil {
		bufferSize = uint64(rb.Capacity())
		currentUsed = uint64(rb.Length())
	}

	return SafeBufferMetrics{
		BufferSize:      bufferSize,
		CurrentUsed:     currentUsed,
		MaxUsedSlots:    b.maxUsedSlots.Load(),
		MessagesWritten: b.messagesWritten.Load(),
		MessagesDropped: b.messagesDropped.Load(),
		BufferFullCount: b.bufferFullCount.Load(),
		SaturationRatio: b.calculateSaturationRatio(),
	}
}

// SafeBufferMetrics contains performance statistics for the buffer
type SafeBufferMetrics struct {
	BufferSize      uint64  // Total buffer capacity
	CurrentUsed     uint64  // Currently used slots
	MaxUsedSlots    uint64  // Peak usage
	MessagesWritten uint64  // Total successful writes
	MessagesDropped uint64  // Total dropped messages (backpressure)
	BufferFullCount uint64  // Number of times buffer was full
	SaturationRatio float64 // Percentage of messages dropped
}

// calculateSaturationRatio computes the ratio of dropped messages
func (b *SafePTYBuffer) calculateSaturationRatio() float64 {
	written := b.messagesWritten.Load()
	dropped := b.messagesDropped.Load()
	total := written + dropped

	if total == 0 {
		return 0.0
	}

	return float64(dropped) / float64(total)
}

// IsEmpty returns true if the buffer is empty
// Thread-safe: ringbuffer.Length() is thread-safe
func (b *SafePTYBuffer) IsEmpty() bool {
	rb := b.ringBuffer.Load()
	if rb == nil {
		return true
	}

	return rb.Length() == 0
}

// IsFull returns true if the buffer is full
// Thread-safe: ringbuffer.Length() and Capacity() are thread-safe
func (b *SafePTYBuffer) IsFull() bool {
	rb := b.ringBuffer.Load()
	if rb == nil {
		return false
	}

	return rb.Length() >= rb.Capacity()
}

// Close safely closes the buffer and releases resources
// Thread-safe: atomic.Pointer.Store is atomic
func (b *SafePTYBuffer) Close() error {
	// Set ring buffer to nil (atomic operation)
	b.ringBuffer.Store(nil)
	return nil
}

// Reset clears the buffer and resets all metrics
// WARNING: This will drop any pending messages
// Thread-safe: Uses atomic operations
func (b *SafePTYBuffer) Reset() {
	// Get current buffer to preserve capacity
	rb := b.ringBuffer.Load()
	if rb != nil {
		// Create new ring buffer with same capacity (atomic swap)
		b.ringBuffer.Store(ringbuffer.New(rb.Capacity()))
	}

	// Reset metrics (atomic operations)
	b.messagesWritten.Store(0)
	b.messagesDropped.Store(0)
	b.bufferFullCount.Store(0)
	b.maxUsedSlots.Store(0)
}

// updateMaxUsedSlots updates the maximum used slots metric
// Thread-safe: Uses atomic compare-and-swap
func (b *SafePTYBuffer) updateMaxUsedSlots(rb *ringbuffer.RingBuffer) {
	if rb == nil {
		return
	}

	current := uint64(rb.Length())
	// Atomic CAS loop to update max value
	for {
		max := b.maxUsedSlots.Load()
		if current <= max || b.maxUsedSlots.CompareAndSwap(max, current) {
			break
		}
	}
}

// serializeMessage converts SafePTYMessage to bytes for storage
func (b *SafePTYBuffer) serializeMessage(msg *SafePTYMessage) ([]byte, error) {
	// Simple serialization - in production, use a more efficient format
	var data []byte
	data = append(data, []byte(msg.SessionID)...)
	data = append(data, 0) // Null terminator
	data = append(data, msg.Data...)
	data = append(data, 0) // Null terminator
	data = append(data, byte(msg.Type))
	data = append(data, byte(msg.Sequence))
	data = append(data, byte(msg.Sequence>>8))
	data = append(data, byte(msg.Sequence>>16))
	data = append(data, byte(msg.Sequence>>24))
	data = append(data, byte(msg.Timestamp.UnixNano()))
	data = append(data, byte(msg.Timestamp.UnixNano()>>8))
	data = append(data, byte(msg.Timestamp.UnixNano()>>16))
	data = append(data, byte(msg.Timestamp.UnixNano()>>24))
	data = append(data, byte(msg.Timestamp.UnixNano()>>32))
	data = append(data, byte(msg.Timestamp.UnixNano()>>40))
	data = append(data, byte(msg.Timestamp.UnixNano()>>48))
	data = append(data, byte(msg.Timestamp.UnixNano()>>56))

	return data, nil
}

// deserializeMessage converts bytes back to SafePTYMessage
func (b *SafePTYBuffer) deserializeMessage(data []byte) (*SafePTYMessage, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("invalid message data")
	}

	// Find first null terminator for session ID
	sessionEnd := 0
	for i, b := range data {
		if b == 0 {
			sessionEnd = i
			break
		}
	}

	if sessionEnd == 0 || sessionEnd >= len(data) {
		return nil, fmt.Errorf("invalid session ID")
	}

	sessionID := string(data[:sessionEnd])

	// Find second null terminator for message data
	dataStart := sessionEnd + 1
	dataEnd := dataStart
	for i := dataStart; i < len(data); i++ {
		if data[i] == 0 {
			dataEnd = i
			break
		}
	}

	if dataEnd >= len(data) {
		return nil, fmt.Errorf("invalid message data")
	}

	msgData := data[dataStart:dataEnd]

	// Extract remaining fields
	if len(data) < dataEnd+12 {
		return nil, fmt.Errorf("invalid message format")
	}

	typeVal := SafePTYMessageType(data[dataEnd])

	sequence := uint64(data[dataEnd+1]) |
		uint64(data[dataEnd+2])<<8 |
		uint64(data[dataEnd+3])<<16 |
		uint64(data[dataEnd+4])<<24

	timestampNanos := int64(data[dataEnd+5]) |
		int64(data[dataEnd+6])<<8 |
		int64(data[dataEnd+7])<<16 |
		int64(data[dataEnd+8])<<24 |
		int64(data[dataEnd+9])<<32 |
		int64(data[dataEnd+10])<<40 |
		int64(data[dataEnd+11])<<48 |
		int64(data[dataEnd+12])<<56

	return &SafePTYMessage{
		SessionID: sessionID,
		Data:      msgData,
		Timestamp: time.Unix(0, timestampNanos),
		Type:      typeVal,
		Sequence:  sequence,
	}, nil
}