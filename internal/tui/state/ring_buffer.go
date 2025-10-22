package state

import (
	"sync/atomic"
)

// RingBuffer is a lock-free, circular buffer for raw PTY bytes.
// It automatically drops the oldest data when full (as requested).
// This implementation is thread-safe using atomic operations.
type RingBuffer struct {
	data     []byte  // The circular buffer
	capacity int64   // Total capacity in bytes
	size     int64   // Current size of valid data
	head     int64   // Write position (next position to write)
	tail     int64   // Read position (next position to read)
	dropped  int64   // Total bytes dropped due to overflow
}

// NewRingBuffer creates a new ring buffer with the specified capacity.
// Default capacity is 8MB (suitable for ~100K lines of terminal output).
func NewRingBuffer(capacity int64) *RingBuffer {
	if capacity <= 0 {
		capacity = 8 * 1024 * 1024 // 8MB default
	}
	return &RingBuffer{
		data:     make([]byte, capacity),
		capacity: capacity,
		size:     0,
		head:     0,
		tail:     0,
		dropped:  0,
	}
}

// Write appends data to the ring buffer.
// If the buffer is full, it drops the oldest data to make room.
// Returns the number of bytes written.
func (rb *RingBuffer) Write(p []byte) int {
	if len(p) == 0 {
		return 0
	}

	dataLen := int64(len(p))

	// If data is larger than entire buffer, only keep the most recent part
	if dataLen >= rb.capacity {
		offset := dataLen - rb.capacity
		copy(rb.data, p[offset:])
		atomic.StoreInt64(&rb.head, 0)
		atomic.StoreInt64(&rb.tail, 0)
		atomic.StoreInt64(&rb.size, rb.capacity)
		atomic.AddInt64(&rb.dropped, offset)
		return int(rb.capacity)
	}

	head := atomic.LoadInt64(&rb.head)
	size := atomic.LoadInt64(&rb.size)

	// Calculate how much we need to drop
	spaceAvailable := rb.capacity - size
	if dataLen > spaceAvailable {
		// Need to drop oldest data
		bytesToDrop := dataLen - spaceAvailable
		atomic.AddInt64(&rb.tail, bytesToDrop)
		if atomic.LoadInt64(&rb.tail) >= rb.capacity {
			atomic.StoreInt64(&rb.tail, atomic.LoadInt64(&rb.tail)%rb.capacity)
		}
		atomic.AddInt64(&rb.dropped, bytesToDrop)
		atomic.StoreInt64(&rb.size, rb.capacity-dataLen)
	}

	// Write data in up to two chunks (wrapping around if necessary)
	written := int64(0)
	for written < dataLen {
		writePos := (head + written) % rb.capacity
		chunkSize := dataLen - written

		// Don't write past the end of the buffer
		if writePos+chunkSize > rb.capacity {
			chunkSize = rb.capacity - writePos
		}

		copy(rb.data[writePos:writePos+chunkSize], p[written:written+chunkSize])
		written += chunkSize
	}

	// Update head and size
	newHead := (head + dataLen) % rb.capacity
	atomic.StoreInt64(&rb.head, newHead)

	newSize := size + dataLen
	if newSize > rb.capacity {
		newSize = rb.capacity
	}
	atomic.StoreInt64(&rb.size, newSize)

	return len(p)
}

// Read reads up to len(p) bytes from the buffer.
// Returns the number of bytes read.
func (rb *RingBuffer) Read(p []byte) int {
	if len(p) == 0 {
		return 0
	}

	size := atomic.LoadInt64(&rb.size)
	if size == 0 {
		return 0
	}

	tail := atomic.LoadInt64(&rb.tail)
	toRead := int64(len(p))
	if toRead > size {
		toRead = size
	}

	// Read data in up to two chunks (handling wrap-around)
	bytesRead := int64(0)
	for bytesRead < toRead {
		readPos := (tail + bytesRead) % rb.capacity
		chunkSize := toRead - bytesRead

		// Don't read past the end of the buffer
		if readPos+chunkSize > rb.capacity {
			chunkSize = rb.capacity - readPos
		}

		copy(p[bytesRead:bytesRead+chunkSize], rb.data[readPos:readPos+chunkSize])
		bytesRead += chunkSize
	}

	// Update tail and size
	newTail := (tail + bytesRead) % rb.capacity
	atomic.StoreInt64(&rb.tail, newTail)
	atomic.AddInt64(&rb.size, -bytesRead)

	return int(bytesRead)
}

// ReadFrom reads all data starting from the given offset.
// This is used for replaying buffered data when switching sessions.
// Returns a copy of the data (safe to use after subsequent writes).
func (rb *RingBuffer) ReadFrom(offset int64) []byte {
	size := atomic.LoadInt64(&rb.size)
	tail := atomic.LoadInt64(&rb.tail)

	// Calculate actual read position
	if offset < 0 {
		offset = 0
	}
	if offset >= size {
		return []byte{}
	}

	// Calculate how much to read
	toRead := size - offset
	if toRead <= 0 {
		return []byte{}
	}

	// Allocate result buffer
	result := make([]byte, toRead)

	// Calculate starting position
	startPos := (tail + offset) % rb.capacity

	// Read data (handling wrap-around)
	bytesRead := int64(0)
	for bytesRead < toRead {
		readPos := (startPos + bytesRead) % rb.capacity
		chunkSize := toRead - bytesRead

		// Don't read past the end of the buffer
		if readPos+chunkSize > rb.capacity {
			chunkSize = rb.capacity - readPos
		}

		copy(result[bytesRead:bytesRead+chunkSize], rb.data[readPos:readPos+chunkSize])
		bytesRead += chunkSize
	}

	return result
}

// Peek returns up to n bytes from the buffer without consuming them.
// Useful for preview/debugging.
func (rb *RingBuffer) Peek(n int) []byte {
	size := atomic.LoadInt64(&rb.size)
	if size == 0 || n <= 0 {
		return []byte{}
	}

	toRead := int64(n)
	if toRead > size {
		toRead = size
	}

	result := make([]byte, toRead)
	tail := atomic.LoadInt64(&rb.tail)

	// Read without updating tail
	bytesRead := int64(0)
	for bytesRead < toRead {
		readPos := (tail + bytesRead) % rb.capacity
		chunkSize := toRead - bytesRead

		if readPos+chunkSize > rb.capacity {
			chunkSize = rb.capacity - readPos
		}

		copy(result[bytesRead:bytesRead+chunkSize], rb.data[readPos:readPos+chunkSize])
		bytesRead += chunkSize
	}

	return result
}

// Size returns the current amount of data in the buffer.
func (rb *RingBuffer) Size() int64 {
	return atomic.LoadInt64(&rb.size)
}

// Capacity returns the total capacity of the buffer.
func (rb *RingBuffer) Capacity() int64 {
	return rb.capacity
}

// DroppedBytes returns the total number of bytes dropped due to overflow.
func (rb *RingBuffer) DroppedBytes() int64 {
	return atomic.LoadInt64(&rb.dropped)
}

// Clear empties the buffer.
func (rb *RingBuffer) Clear() {
	atomic.StoreInt64(&rb.head, 0)
	atomic.StoreInt64(&rb.tail, 0)
	atomic.StoreInt64(&rb.size, 0)
	// Note: we don't reset dropped counter, as it's cumulative
}

// Stats returns statistics about the buffer.
func (rb *RingBuffer) Stats() map[string]int64 {
	return map[string]int64{
		"capacity": rb.capacity,
		"size":     atomic.LoadInt64(&rb.size),
		"head":     atomic.LoadInt64(&rb.head),
		"tail":     atomic.LoadInt64(&rb.tail),
		"dropped":  atomic.LoadInt64(&rb.dropped),
		"free":     rb.capacity - atomic.LoadInt64(&rb.size),
	}
}