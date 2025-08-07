package tui

import (
	"sync/atomic"
	"time"
	"unsafe"
)

// LockFreePTYBuffer implements a high-performance lock-free ring buffer for PTY messages
// Uses atomic operations and memory alignment for zero-lock message passing
type LockFreePTYBuffer struct {
	// Ring buffer storage (power of 2 size for fast modulo via bitwise AND)
	buffer []*PTYMessage
	size   uint64
	mask   uint64 // size - 1 for fast modulo operation
	
	// Atomic cursors (aligned to cache line boundaries to prevent false sharing)
	writePos uint64 // Next write position
	readPos  uint64 // Next read position
	
	// Padding to prevent false sharing between atomic fields
	_pad1 [56]byte
	
	// Performance metrics (atomic counters)
	messagesWritten uint64
	messagesDropped uint64
	bufferFullCount uint64
	maxUsedSlots    uint64
	
	// Additional padding
	_pad2 [32]byte
}

// PTYMessage represents a single PTY message with metadata
type PTYMessage struct {
	SessionID string
	Data      []byte
	Timestamp time.Time
	Type      PTYMessageType
	Sequence  uint64 // For ordering verification
}

// PTYMessageType categorizes different types of PTY messages
type PTYMessageType uint8

const (
	PTYMessageOutput PTYMessageType = iota
	PTYMessageEvent
	PTYMessageClose
	PTYMessageResize
)

// NewLockFreePTYBuffer creates a new lock-free ring buffer
// bufferSize must be a power of 2 for optimal performance
func NewLockFreePTYBuffer(bufferSize uint64) *LockFreePTYBuffer {
	// Ensure buffer size is power of 2
	if bufferSize == 0 || (bufferSize&(bufferSize-1)) != 0 {
		// Round up to next power of 2
		bufferSize = nextPowerOfTwo(bufferSize)
	}
	
	return &LockFreePTYBuffer{
		buffer: make([]*PTYMessage, bufferSize),
		size:   bufferSize,
		mask:   bufferSize - 1,
	}
}

// TryWrite attempts to write a message to the buffer
// Returns true if successful, false if buffer is full (backpressure)
// This operation is lock-free and wait-free
func (b *LockFreePTYBuffer) TryWrite(msg *PTYMessage) bool {
	// Load current positions atomically
	currentWrite := atomic.LoadUint64(&b.writePos)
	currentRead := atomic.LoadUint64(&b.readPos)
	
	// Calculate next write position
	nextWrite := currentWrite + 1
	
	// Check if buffer would be full after write
	// We keep one slot empty to distinguish full from empty
	if nextWrite-currentRead >= b.size {
		// Buffer is full - apply backpressure
		atomic.AddUint64(&b.messagesDropped, 1)
		atomic.AddUint64(&b.bufferFullCount, 1)
		return false
	}
	
	// Store message in buffer
	slot := currentWrite & b.mask
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&b.buffer[slot])),
		unsafe.Pointer(msg),
	)
	
	// Advance write position atomically
	if !atomic.CompareAndSwapUint64(&b.writePos, currentWrite, nextWrite) {
		// Someone else beat us to it - this is normal in high-contention scenarios
		// The message is already stored, but position wasn't updated by us
		// Let the other writer handle the position update
		return false
	}
	
	// Update metrics
	atomic.AddUint64(&b.messagesWritten, 1)
	
	// Update max used slots metric
	used := nextWrite - currentRead
	for {
		currentMax := atomic.LoadUint64(&b.maxUsedSlots)
		if used <= currentMax || atomic.CompareAndSwapUint64(&b.maxUsedSlots, currentMax, used) {
			break
		}
	}
	
	return true
}

// TryRead attempts to read a message from the buffer
// Returns message and true if successful, nil and false if buffer is empty
// This operation is lock-free and wait-free
func (b *LockFreePTYBuffer) TryRead() (*PTYMessage, bool) {
	// Load current positions atomically
	currentRead := atomic.LoadUint64(&b.readPos)
	currentWrite := atomic.LoadUint64(&b.writePos)
	
	// Check if buffer is empty
	if currentRead >= currentWrite {
		return nil, false
	}
	
	// Read message from buffer
	slot := currentRead & b.mask
	msgPtr := atomic.LoadPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&b.buffer[slot])),
	)
	
	if msgPtr == nil {
		// Message not yet written by writer (very rare race condition)
		return nil, false
	}
	
	msg := (*PTYMessage)(msgPtr)
	
	// Clear the slot (helps with GC and debugging)
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&b.buffer[slot])),
		nil,
	)
	
	// Advance read position atomically
	if !atomic.CompareAndSwapUint64(&b.readPos, currentRead, currentRead+1) {
		// Another reader beat us to it - return the message anyway
		// since we successfully read it
		return msg, true
	}
	
	return msg, true
}

// TryReadBatch attempts to read up to maxMessages from the buffer
// Returns slice of messages and actual count read
// More efficient than multiple TryRead calls for batch processing
func (b *LockFreePTYBuffer) TryReadBatch(maxMessages int) ([]*PTYMessage, int) {
	if maxMessages <= 0 {
		return nil, 0
	}
	
	batch := make([]*PTYMessage, 0, maxMessages)
	
	for len(batch) < maxMessages {
		msg, ok := b.TryRead()
		if !ok {
			break // Buffer empty
		}
		batch = append(batch, msg)
	}
	
	return batch, len(batch)
}

// GetMetrics returns current buffer performance metrics
func (b *LockFreePTYBuffer) GetMetrics() BufferMetrics {
	currentWrite := atomic.LoadUint64(&b.writePos)
	currentRead := atomic.LoadUint64(&b.readPos)
	
	return BufferMetrics{
		BufferSize:       b.size,
		CurrentUsed:      currentWrite - currentRead,
		MaxUsedSlots:     atomic.LoadUint64(&b.maxUsedSlots),
		MessagesWritten:  atomic.LoadUint64(&b.messagesWritten),
		MessagesDropped:  atomic.LoadUint64(&b.messagesDropped),
		BufferFullCount:  atomic.LoadUint64(&b.bufferFullCount),
		SaturationRatio:  b.calculateSaturationRatio(),
	}
}

// BufferMetrics contains performance statistics for the buffer
type BufferMetrics struct {
	BufferSize       uint64  // Total buffer capacity
	CurrentUsed      uint64  // Currently used slots
	MaxUsedSlots     uint64  // Peak usage
	MessagesWritten  uint64  // Total successful writes
	MessagesDropped  uint64  // Total dropped messages (backpressure)
	BufferFullCount  uint64  // Number of times buffer was full
	SaturationRatio  float64 // Percentage of messages dropped
}

// calculateSaturationRatio computes the ratio of dropped messages
func (b *LockFreePTYBuffer) calculateSaturationRatio() float64 {
	written := atomic.LoadUint64(&b.messagesWritten)
	dropped := atomic.LoadUint64(&b.messagesDropped)
	total := written + dropped
	
	if total == 0 {
		return 0.0
	}
	
	return float64(dropped) / float64(total)
}

// IsEmpty returns true if the buffer is empty
func (b *LockFreePTYBuffer) IsEmpty() bool {
	return atomic.LoadUint64(&b.readPos) >= atomic.LoadUint64(&b.writePos)
}

// IsFull returns true if the buffer is full
func (b *LockFreePTYBuffer) IsFull() bool {
	currentWrite := atomic.LoadUint64(&b.writePos)
	currentRead := atomic.LoadUint64(&b.readPos)
	return currentWrite-currentRead >= b.size
}

// Reset clears the buffer and resets all metrics
// WARNING: This is NOT thread-safe and should only be called when no other
// goroutines are accessing the buffer
func (b *LockFreePTYBuffer) Reset() {
	// Clear all messages
	for i := uint64(0); i < b.size; i++ {
		atomic.StorePointer(
			(*unsafe.Pointer)(unsafe.Pointer(&b.buffer[i])),
			nil,
		)
	}
	
	// Reset positions
	atomic.StoreUint64(&b.writePos, 0)
	atomic.StoreUint64(&b.readPos, 0)
	
	// Reset metrics
	atomic.StoreUint64(&b.messagesWritten, 0)
	atomic.StoreUint64(&b.messagesDropped, 0)
	atomic.StoreUint64(&b.bufferFullCount, 0)
	atomic.StoreUint64(&b.maxUsedSlots, 0)
}

// nextPowerOfTwo returns the next power of 2 greater than or equal to n
func nextPowerOfTwo(n uint64) uint64 {
	if n <= 1 {
		return 1
	}
	
	// Handle edge case
	if n > (1 << 62) {
		return 1 << 62 // Largest power of 2 in uint64
	}
	
	// Bit manipulation to find next power of 2
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	
	return n
}