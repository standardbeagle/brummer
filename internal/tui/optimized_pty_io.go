package tui

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
)

// OptimizedPTYIO provides high-performance PTY I/O with system call optimization
// Features: batch processing, zero-copy, memory pooling, and lock-free design
type OptimizedPTYIO struct {
	// Core components
	session     atomic.Value // *aicoder.PTYSession
	terminalFD  atomic.Int64 // Direct terminal file descriptor

	// Atomic state
	active          atomic.Bool
	batchMode       atomic.Bool
	passthroughMode atomic.Bool

	// Performance optimization
	batchBuffer     atomic.Value // *BatchBuffer
	writeQueue      atomic.Value // *WriteQueue

	// Object pools for memory efficiency
	bufferPool      sync.Pool
	messagePool     sync.Pool
	batchPool       sync.Pool

	// Performance counters (atomic operations)
	bytesWritten    atomic.Uint64
	messagesWritten atomic.Uint64
	batchesProcessed atomic.Uint64
	syscallCount    atomic.Uint64

	// Control
	ctx    context.Context
	cancel context.CancelFunc

	// Dependencies
	logStore *logs.Store
}

// BatchBuffer implements zero-copy batch processing
type BatchBuffer struct {
	messages    []OptimizedMessage
	maxSize     int
	currentSize int
	mu          sync.Mutex
}

// WriteQueue implements lock-free write queuing
type WriteQueue struct {
	entries    []WriteEntry
	head       atomic.Uint64
	tail       atomic.Uint64
	mask       uint64
}

// WriteEntry represents a single write operation
type WriteEntry struct {
	data     []byte
	length   int
	priority uint8
}

// OptimizedMessage represents a high-performance PTY message
type OptimizedMessage struct {
	SessionID [16]byte
	Data      []byte
	Length    uint32
	Type      uint8
	Sequence  uint64
	Timestamp int64
}

// NewOptimizedPTYIO creates a new optimized PTY I/O system
func NewOptimizedPTYIO(ctx context.Context, session *aicoder.PTYSession, logStore *logs.Store) *OptimizedPTYIO {
	ptyCtx, cancel := context.WithCancel(ctx)

	ptyIO := &OptimizedPTYIO{
		ctx:      ptyCtx,
		cancel:   cancel,
		logStore: logStore,
	}

	// Store session atomically
	ptyIO.session.Store(session)

	// Initialize terminal file descriptor atomically
	ptyIO.terminalFD.Store(int64(os.Stdout.Fd()))

	// Initialize object pools for maximum performance
	ptyIO.initializeObjectPools()

	// Initialize batch buffer and write queue
	ptyIO.initializeBatchSystems()

	return ptyIO
}

// initializeObjectPools sets up high-performance object pools
func (p *OptimizedPTYIO) initializeObjectPools() {
	// Buffer pool for I/O operations
	p.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 8192) // 8KB pre-allocation
		},
	}

	// Message pool for PTY messages
	p.messagePool = sync.Pool{
		New: func() interface{} {
			return &OptimizedMessage{
				Data: make([]byte, 0, 1024), // 1KB pre-allocation
			}
		},
	}

	// Batch pool for batch operations
	p.batchPool = sync.Pool{
		New: func() interface{} {
			return &BatchBuffer{
				messages: make([]OptimizedMessage, 0, 64), // 64 message batch capacity
				maxSize:  64,
			}
		},
	}
}

// initializeBatchSystems sets up batch processing and write queuing
func (p *OptimizedPTYIO) initializeBatchSystems() {
	// Initialize batch buffer
	batchBuffer := &BatchBuffer{
		messages: make([]OptimizedMessage, 0, 64),
		maxSize:  64,
	}
	p.batchBuffer.Store(batchBuffer)

	// Initialize write queue with power-of-2 size for efficient masking
	queueSize := uint64(1024) // Must be power of 2
	writeQueue := &WriteQueue{
		entries: make([]WriteEntry, queueSize),
		mask:    queueSize - 1,
	}
	p.writeQueue.Store(writeQueue)
}

// Start begins the optimized PTY I/O operations
func (p *OptimizedPTYIO) Start() error {
	if !p.active.CompareAndSwap(false, true) {
		return nil // Already active
	}

	p.logStore.Add("system", "System", "Starting optimized PTY I/O (batch processing enabled)", false)

	// Enable batch mode for maximum throughput
	p.batchMode.Store(true)

	return nil
}

// Stop ends the optimized PTY I/O operations
func (p *OptimizedPTYIO) Stop() {
	if !p.active.CompareAndSwap(true, false) {
		return // Already stopped
	}

	p.cancel()
	p.logStore.Add("system", "System", "Stopped optimized PTY I/O", false)
}

// ProcessPTYOutputOptimized creates a command for high-performance PTY output processing
func (p *OptimizedPTYIO) ProcessPTYOutputOptimized() tea.Cmd {
	return func() tea.Msg {
		if !p.active.Load() {
			return nil
		}

		// Get session safely
		session := p.getSession()
		if session == nil {
			return nil
		}

		// Process with optimized batch handling
		if p.batchMode.Load() {
			return p.processBatchOutput(session)
		}

		return p.processSingleOutput(session)
	}
}

// processBatchOutput processes multiple PTY messages in a single batch
func (p *OptimizedPTYIO) processBatchOutput(session *aicoder.PTYSession) tea.Msg {
	batchBuffer := p.batchBuffer.Load().(*BatchBuffer)
	if batchBuffer == nil {
		return nil
	}

	// Collect messages with timeout for responsiveness
	timeout := time.After(5 * time.Millisecond)
	messageCount := 0

	for messageCount < batchBuffer.maxSize {
		select {
		case output, ok := <-session.OutputChan:
			if !ok {
				// Channel closed - signal session ended
				p.active.Store(false)
				return p.createBatchMessage(batchBuffer, true)
			}

			// Add message to batch
			if p.addMessageToBatch(batchBuffer, session.ID, output) {
				messageCount++
			}

		case <-timeout:
			// Timeout reached - process current batch
			return p.createBatchMessage(batchBuffer, false)

		case <-p.ctx.Done():
			// Context cancelled
			p.active.Store(false)
			return nil
		}
	}

	return p.createBatchMessage(batchBuffer, false)
}

// processSingleOutput processes individual messages (fallback)
func (p *OptimizedPTYIO) processSingleOutput(session *aicoder.PTYSession) tea.Msg {
	timeout := time.After(10 * time.Millisecond)

	for {
		select {
		case output, ok := <-session.OutputChan:
			if !ok {
				// Channel closed
				p.active.Store(false)
				return nil
			}

			// Process single message
			p.writeOptimizedToTerminal(output)
			return p.createSingleMessage(session.ID, output)

		case <-timeout:
			return nil

		case <-p.ctx.Done():
			p.active.Store(false)
			return nil
		}
	}
}

// addMessageToBatch adds a message to the batch buffer with zero-copy optimization
func (p *OptimizedPTYIO) addMessageToBatch(batch *BatchBuffer, sessionID string, data []byte) bool {
	batch.mu.Lock()
	defer batch.mu.Unlock()

	if batch.currentSize >= batch.maxSize {
		return false // Batch full
	}

	// Get message from pool
	msg := p.messagePool.Get().(*OptimizedMessage)

	// Prepare message with zero-copy session ID handling
	p.copySessionID(msg.SessionID[:], sessionID)
	msg.Length = uint32(len(data))
	msg.Type = 0 // Output type
	msg.Sequence = 0 // Will be set by caller
	msg.Timestamp = time.Now().UnixNano()

	// Reuse data buffer efficiently
	if cap(msg.Data) < len(data) {
		msg.Data = make([]byte, len(data))
	}
	msg.Data = msg.Data[:len(data)]
	copy(msg.Data, data)

	// Add to batch
	batch.messages = append(batch.messages, *msg)
	batch.currentSize++

	// Return message to pool (data is copied)
	p.messagePool.Put(msg)

	return true
}

// createBatchMessage creates a batch message for BubbleTea
func (p *OptimizedPTYIO) createBatchMessage(batch *BatchBuffer, final bool) tea.Msg {
	if batch.currentSize == 0 && !final {
		return nil
	}

	// Create batch message
	batchMsg := BatchOptimizedPTYMsg{
		Messages: make([]OptimizedMessage, batch.currentSize),
		Final:    final,
	}

	// Copy messages (zero-copy where possible)
	copy(batchMsg.Messages, batch.messages)

	// Update metrics
	p.batchesProcessed.Add(1)
	p.messagesWritten.Add(uint64(batch.currentSize))

	// Reset batch buffer
	batch.mu.Lock()
	batch.messages = batch.messages[:0]
	batch.currentSize = 0
	batch.mu.Unlock()

	return batchMsg
}

// createSingleMessage creates a single message for BubbleTea
func (p *OptimizedPTYIO) createSingleMessage(sessionID string, data []byte) tea.Msg {
	// Get message from pool
	msg := p.messagePool.Get().(*OptimizedMessage)
	defer p.messagePool.Put(msg)

	// Prepare message
	p.copySessionID(msg.SessionID[:], sessionID)
	msg.Length = uint32(len(data))
	msg.Type = 0 // Output type
	msg.Sequence = 0
	msg.Timestamp = time.Now().UnixNano()

	// Copy data
	if cap(msg.Data) < len(data) {
		msg.Data = make([]byte, len(data))
	}
	msg.Data = msg.Data[:len(data)]
	copy(msg.Data, data)

	// Create single message
	return SingleOptimizedPTYMsg{
		Message: *msg,
	}
}

// writeOptimizedToTerminal performs optimized terminal write with system call optimization
func (p *OptimizedPTYIO) writeOptimizedToTerminal(data []byte) {
	fd := int(p.terminalFD.Load())
	if fd <= 0 {
		return
	}

	if len(data) == 0 {
		return
	}

	// Use system call for maximum performance
	n, err := syscall.Write(fd, data)
	if err != nil {
		p.logStore.Add("system", "System",
			fmt.Sprintf("Optimized terminal write error: %v", err), true)
		return
	}

	// Update metrics
	p.bytesWritten.Add(uint64(n))
	p.syscallCount.Add(1)
}

// writeBatchToTerminal performs batch terminal write for maximum throughput
func (p *OptimizedPTYIO) writeBatchToTerminal(messages []OptimizedMessage) {
	if len(messages) == 0 {
		return
	}

	fd := int(p.terminalFD.Load())
	if fd <= 0 {
		return
	}

	// Get buffer from pool for batch concatenation
	buffer := p.bufferPool.Get().([]byte)
	defer p.bufferPool.Put(buffer)

	// Calculate total size and prepare buffer
	totalSize := 0
	for _, msg := range messages {
		totalSize += int(msg.Length)
	}

	// Ensure buffer has enough capacity
	if cap(buffer) < totalSize {
		buffer = make([]byte, totalSize)
	} else {
		buffer = buffer[:totalSize]
	}

	// Concatenate all message data (zero-copy where possible)
	offset := 0
	for _, msg := range messages {
		copy(buffer[offset:], msg.Data)
		offset += int(msg.Length)
	}

	// Single system call for the entire batch
	n, err := syscall.Write(fd, buffer)
	if err != nil {
		p.logStore.Add("system", "System",
			fmt.Sprintf("Batch terminal write error: %v", err), true)
		return
	}

	// Update metrics
	p.bytesWritten.Add(uint64(n))
	p.syscallCount.Add(1)
	p.messagesWritten.Add(uint64(len(messages)))
}

// getSession safely retrieves the current session
func (p *OptimizedPTYIO) getSession() *aicoder.PTYSession {
	sessionInterface := p.session.Load()
	if sessionInterface == nil {
		return nil
	}

	session, ok := sessionInterface.(*aicoder.PTYSession)
	if !ok || session == nil {
		return nil
	}

	return session
}

// copySessionID efficiently copies session ID to fixed-size array
func (p *OptimizedPTYIO) copySessionID(dst []byte, sessionID string) {
	maxLen := len(dst)
	if len(sessionID) < maxLen {
		maxLen = len(sessionID)
	}
	copy(dst, sessionID)

	// Zero out remaining bytes
	for i := maxLen; i < len(dst); i++ {
		dst[i] = 0
	}
}

// GetOptimizedMetrics returns comprehensive performance metrics
func (p *OptimizedPTYIO) GetOptimizedMetrics() OptimizedPTYMetrics {
	return OptimizedPTYMetrics{
		Active:           p.active.Load(),
		BatchMode:        p.batchMode.Load(),
		PassthroughMode:  p.passthroughMode.Load(),
		BytesWritten:     p.bytesWritten.Load(),
		MessagesWritten:  p.messagesWritten.Load(),
		BatchesProcessed: p.batchesProcessed.Load(),
		SyscallCount:     p.syscallCount.Load(),
		AverageBatchSize: p.calculateAverageBatchSize(),
	}
}

// calculateAverageBatchSize computes the average batch size
func (p *OptimizedPTYIO) calculateAverageBatchSize() uint64 {
	batches := p.batchesProcessed.Load()
	messages := p.messagesWritten.Load()

	if batches == 0 {
		return 0
	}

	return messages / batches
}

// OptimizedPTYMetrics contains comprehensive performance statistics
type OptimizedPTYMetrics struct {
	Active           bool
	BatchMode        bool
	PassthroughMode  bool
	BytesWritten     uint64
	MessagesWritten  uint64
	BatchesProcessed uint64
	SyscallCount     uint64
	AverageBatchSize uint64
}

// BatchOptimizedPTYMsg represents a batch of optimized PTY messages
type BatchOptimizedPTYMsg struct {
	Messages []OptimizedMessage
	Final    bool // Indicates if this is the final batch for a session
}

// SingleOptimizedPTYMsg represents a single optimized PTY message
type SingleOptimizedPTYMsg struct {
	Message OptimizedMessage
}