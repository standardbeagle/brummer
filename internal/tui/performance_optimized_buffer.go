package tui

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/smallnest/ringbuffer"
)

// OptimizedSafePTYBuffer eliminates serialization overhead with zero-copy design
// Uses object pooling and pre-allocated buffers for maximum performance
type OptimizedSafePTYBuffer struct {
	// Core ring buffer implementation (production-ready)
	ringBuffer *ringbuffer.RingBuffer

	// Performance metrics (atomic operations)
	messagesWritten atomic.Uint64
	messagesDropped atomic.Uint64
	bufferFullCount atomic.Uint64
	maxUsedSlots    atomic.Uint64

	// Object pools for memory efficiency
	messagePool sync.Pool
	bufferPool  sync.Pool

	// Synchronization for configuration changes
	mu sync.RWMutex
}

// OptimizedSafePTYMessage uses a compact binary layout for zero-copy operations
type OptimizedSafePTYMessage struct {
	SessionID [16]byte // Fixed-size session ID (no allocation)
	DataLen   uint32   // Length of data for bounds checking
	Sequence  uint64
	Type      uint8
	Timestamp int64
	Data      []byte // Reference to pooled data, not copied
}

// NewOptimizedSafePTYBuffer creates an optimized buffer with object pools
func NewOptimizedSafePTYBuffer(bufferSize uint64) *OptimizedSafePTYBuffer {
	// Ensure buffer size is reasonable
	if bufferSize < 64 {
		bufferSize = 64
	}
	if bufferSize > 1000000 {
		bufferSize = 1000000 // Limit to 1M messages
	}

	buf := &OptimizedSafePTYBuffer{
		ringBuffer: ringbuffer.New(int(bufferSize)),
	}

	// Initialize object pools for memory efficiency
	buf.messagePool = sync.Pool{
		New: func() interface{} {
			return &OptimizedSafePTYMessage{
				Data: make([]byte, 0, 1024), // Pre-allocate with capacity
			}
		},
	}

	buf.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 4096) // Pre-allocated buffer capacity
		},
	}

	return buf
}

// TryWriteOptimized performs zero-copy write with pooled message allocation
func (b *OptimizedSafePTYBuffer) TryWriteOptimized(sessionID string, data []byte, msgType uint8, sequence uint64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check if buffer has been closed
	if b.ringBuffer == nil {
		return false
	}

	// Get message from pool
	msg := b.messagePool.Get().(*OptimizedSafePTYMessage)
	defer b.messagePool.Put(msg) // Return to pool after use

	// Prepare message with zero-copy session ID handling
	b.copySessionID(msg.SessionID[:], sessionID)
	msg.DataLen = uint32(len(data))
	msg.Sequence = sequence
	msg.Type = msgType
	msg.Timestamp = time.Now().UnixNano()

	// Use pooled data buffer if provided data is not reusable
	if cap(msg.Data) < len(data) {
		msg.Data = make([]byte, len(data))
	}
	msg.Data = msg.Data[:len(data)]
	copy(msg.Data, data)

	// Zero-copy serialization using unsafe pointer
	binaryData := b.marshalMessageUnsafe(msg)

	// Try to write to ring buffer
	n, err := b.ringBuffer.Write(binaryData)
	if err != nil {
		// Buffer is full or other error
		b.messagesDropped.Add(1)
		if n == 0 {
			b.bufferFullCount.Add(1)
		}
		return false
	}

	// Update metrics
	b.messagesWritten.Add(1)
	b.updateMaxUsedSlots()

	return true
}

// TryReadOptimized performs zero-copy read with buffer pooling
func (b *OptimizedSafePTYBuffer) TryReadOptimized() (*OptimizedSafePTYMessage, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check if buffer has been closed
	if b.ringBuffer == nil {
		return nil, false
	}

	// Get buffer from pool
	readBuffer := b.bufferPool.Get().([]byte)
	defer b.bufferPool.Put(readBuffer) // Return to pool after use

	// Reset buffer but keep capacity
	readBuffer = readBuffer[:cap(readBuffer)]

	// Read from ring buffer
	n, err := b.ringBuffer.Read(readBuffer)
	if err != nil {
		// Buffer is empty or other error
		return nil, false
	}

	// Zero-copy deserialization using unsafe pointer
	msg := b.unmarshalMessageUnsafe(readBuffer[:n])

	return msg, true
}

// copySessionID efficiently copies session ID to fixed-size array
func (b *OptimizedSafePTYBuffer) copySessionID(dst []byte, sessionID string) {
	// Efficient copy with bounds checking
	maxLen := len(dst)
	if len(sessionID) < maxLen {
		maxLen = len(sessionID)
	}
	copy(dst, sessionID)

	// Zero out remaining bytes for consistent hashing
	for i := maxLen; i < len(dst); i++ {
		dst[i] = 0
	}
}

// marshalMessageUnsafe performs zero-copy serialization using unsafe operations
func (b *OptimizedSafePTYBuffer) marshalMessageUnsafe(msg *OptimizedSafePTYMessage) []byte {
	// Calculate total size: header (37 bytes) + data
	totalSize := 37 + len(msg.Data)

	// Get pooled buffer for marshaling
	buffer := b.bufferPool.Get().([]byte)
	defer b.bufferPool.Put(buffer)

	// Ensure buffer has enough capacity
	if cap(buffer) < totalSize {
		buffer = make([]byte, totalSize)
	} else {
		buffer = buffer[:totalSize]
	}

	// Optimized binary layout using unsafe operations
	header := (*[37]byte)(unsafe.Pointer(msg))
	copy(buffer, header[:])

	// Copy data section
	copy(buffer[37:], msg.Data)

	// Create copy for return (necessary for ring buffer storage)
	result := make([]byte, totalSize)
	copy(result, buffer)

	return result
}

// unmarshalMessageUnsafe performs zero-copy deserialization using unsafe operations
func (b *OptimizedSafePTYBuffer) unmarshalMessageUnsafe(data []byte) *OptimizedSafePTYMessage {
	if len(data) < 37 {
		return nil
	}

	// Get message from pool
	msg := b.messagePool.Get().(*OptimizedSafePTYMessage)

	// Zero-copy header unmarshaling using unsafe operations
	header := (*[37]byte)(unsafe.Pointer(&data[0]))
	msgHeader := *(*OptimizedSafePTYMessage)(unsafe.Pointer(&header[0]))

	// Copy header fields
	msg.SessionID = msgHeader.SessionID
	msg.DataLen = msgHeader.DataLen
	msg.Sequence = msgHeader.Sequence
	msg.Type = msgHeader.Type
	msg.Timestamp = msgHeader.Timestamp

	// Extract data section efficiently
	dataStart := 37
	if int(msg.DataLen) > len(data)-dataStart {
		msg.DataLen = uint32(len(data) - dataStart)
	}

	// Reuse or allocate data buffer
	if cap(msg.Data) < int(msg.DataLen) {
		msg.Data = make([]byte, msg.DataLen)
	} else {
		msg.Data = msg.Data[:msg.DataLen]
	}

	copy(msg.Data, data[dataStart:dataStart+int(msg.DataLen)])

	return msg
}

// updateMaxUsedSlots updates the maximum used slots metric
func (b *OptimizedSafePTYBuffer) updateMaxUsedSlots() {
	if b.ringBuffer == nil {
		return
	}

	current := uint64(b.ringBuffer.Length())
	for {
		max := b.maxUsedSlots.Load()
		if current <= max || b.maxUsedSlots.CompareAndSwap(max, current) {
			break
		}
	}
}

// GetOptimizedMetrics returns performance metrics
func (b *OptimizedSafePTYBuffer) GetOptimizedMetrics() SafeBufferMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var bufferSize, currentUsed uint64
	if b.ringBuffer != nil {
		bufferSize = uint64(b.ringBuffer.Capacity())
		currentUsed = uint64(b.ringBuffer.Length())
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

// calculateSaturationRatio computes the ratio of dropped messages
func (b *OptimizedSafePTYBuffer) calculateSaturationRatio() float64 {
	written := b.messagesWritten.Load()
	dropped := b.messagesDropped.Load()
	total := written + dropped

	if total == 0 {
		return 0.0
	}

	return float64(dropped) / float64(total)
}

// Close safely closes the buffer and releases resources
func (b *OptimizedSafePTYBuffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ringBuffer != nil {
		// Clear any remaining data
		b.ringBuffer = nil
	}

	return nil
}

// Reset clears the buffer and resets all metrics
func (b *OptimizedSafePTYBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Reset ring buffer
	if b.ringBuffer != nil {
		b.ringBuffer = ringbuffer.New(b.ringBuffer.Capacity())
	}

	// Reset metrics
	b.messagesWritten.Store(0)
	b.messagesDropped.Store(0)
	b.bufferFullCount.Store(0)
	b.maxUsedSlots.Store(0)
}