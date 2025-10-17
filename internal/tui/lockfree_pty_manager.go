package tui

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
)

// LockFreePTYManager manages PTY sessions using lock-free data structures
type LockFreePTYManager struct {
	// Safe buffer for PTY messages (production-ready library)
	buffer *SafePTYBuffer

	// Session state management (lock-free)
	sessions sync.Map // sessionID -> *PTYSessionState

	// Single-writer goroutine management
	writerActive   atomic.Bool
	writerStopChan chan struct{}

	// Dependencies
	updateChan chan tea.Msg
	logStore   *logs.Store
	ctx        context.Context
	cancel     context.CancelFunc

	// Performance monitoring
	metrics           *PTYManagerMetrics
	metricsUpdateChan chan struct{}

	// Configuration
	batchSize    int
	pollInterval time.Duration
}

// PTYSessionState tracks the state of a single PTY session
type PTYSessionState struct {
	session    *aicoder.PTYSession
	active     atomic.Bool
	monitoring atomic.Bool
	stopChan   chan struct{}
	sequence   atomic.Uint64 // Message sequence number
}

// PTYManagerMetrics tracks performance statistics
type PTYManagerMetrics struct {
	activeSessions    atomic.Uint64
	totalMessages     atomic.Uint64
	droppedMessages   atomic.Uint64
	batchesProcessed  atomic.Uint64
	averageBatchSize  atomic.Uint64
	lastPollDuration  atomic.Int64  // nanoseconds
	bufferUtilization atomic.Uint64 // percentage * 100
}

// PTYManagerMetricsSnapshot provides a lock-free snapshot of metrics
// This struct contains regular uint64/int64 values that can be safely passed by value
type PTYManagerMetricsSnapshot struct {
	ActiveSessions    uint64
	TotalMessages     uint64
	DroppedMessages   uint64
	BatchesProcessed  uint64
	AverageBatchSize  uint64
	LastPollDuration  int64  // nanoseconds
	BufferUtilization uint64 // percentage * 100
	Timestamp         time.Time
}

// NewLockFreePTYManager creates a new lock-free PTY manager with default config
func NewLockFreePTYManager(ctx context.Context, updateChan chan tea.Msg, logStore *logs.Store) *LockFreePTYManager {
	return NewLockFreePTYManagerWithConfig(ctx, updateChan, logStore, DefaultLockFreeConfig())
}

// NewLockFreePTYManagerWithConfig creates a new lock-free PTY manager with custom config
func NewLockFreePTYManagerWithConfig(ctx context.Context, updateChan chan tea.Msg, logStore *logs.Store, config LockFreeConfig) *LockFreePTYManager {
	managerCtx, cancel := context.WithCancel(ctx)

	// Validate configuration
	if err := config.ValidateConfig(); err != nil {
		logStore.Add("system", "System", fmt.Sprintf("Lock-free config validation failed, using defaults: %v", err), true)
		config = DefaultLockFreeConfig()
	}

	logStore.Add("system", "System", fmt.Sprintf("Initializing lock-free PTY manager: buffer=%d, batch=%d, poll=%v, memory=%.1fMB",
		config.BufferSize, config.BatchSize, config.PollInterval, config.GetBufferSizeMB()), false)

	mgr := &LockFreePTYManager{
		buffer:            NewSafePTYBuffer(config.BufferSize),
		updateChan:        updateChan,
		logStore:          logStore,
		ctx:               managerCtx,
		cancel:            cancel,
		writerStopChan:    make(chan struct{}),
		metrics:           &PTYManagerMetrics{},
		metricsUpdateChan: make(chan struct{}, 1),
		batchSize:         config.BatchSize,
		pollInterval:      config.PollInterval,
	}

	return mgr
}

// StartSession begins monitoring a PTY session with lock-free management
// Note: This replaces goroutine-based monitoring with command-driven processing
func (m *LockFreePTYManager) StartSession(session *aicoder.PTYSession) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	// Create session state
	state := &PTYSessionState{
		session:  session,
		stopChan: make(chan struct{}),
	}

	// Store in sessions map
	if _, loaded := m.sessions.LoadOrStore(session.ID, state); loaded {
		return fmt.Errorf("session %s already being monitored", session.ID)
	}

	// Mark as active
	state.active.Store(true)
	m.metrics.activeSessions.Add(1)

	// NO MORE GOROUTINES - Session monitoring will be command-driven
	// through BubbleTea Update() cycle instead of background goroutine
	m.logStore.Add("system", "System",
		fmt.Sprintf("Started lock-free PTY monitoring for session %s (command-driven)", session.ID), false)

	return nil
}

// StopSession stops monitoring a PTY session
func (m *LockFreePTYManager) StopSession(sessionID string) error {
	stateInterface, exists := m.sessions.Load(sessionID)
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	state, ok := stateInterface.(*PTYSessionState)
	if !ok {
		return fmt.Errorf("invalid session state for %s", sessionID)
	}

	// Signal stop first
	state.active.Store(false)

	// Close stop channel safely (avoid panic if already closed)
	select {
	case <-state.stopChan:
		// Channel already closed
	default:
		close(state.stopChan)
	}

	// Remove from sessions map only after marking inactive
	m.sessions.Delete(sessionID)

	// Decrement active session count if we successfully stopped
	m.metrics.activeSessions.Add(^uint64(0)) // Atomic decrement

	m.logStore.Add("system", "System",
		fmt.Sprintf("Stopped lock-free PTY monitoring for session %s", sessionID), false)

	return nil
}

// monitorSession monitors a single PTY session and writes to the lock-free buffer
func (m *LockFreePTYManager) monitorSession(state *PTYSessionState) {
	defer func() {
		state.monitoring.Store(false)
		// Clean up on exit
		m.sessions.Delete(state.session.ID)
	}()

	state.monitoring.Store(true)
	session := state.session

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-state.stopChan:
			return

		case output, ok := <-session.OutputChan:
			if !ok {
				// Channel closed, session ended
				return
			}

			// Create PTY message with sequence number
			msg := &SafePTYMessage{
				SessionID: session.ID,
				Data:      make([]byte, len(output)),
				Timestamp: time.Now(),
				Type:      SafePTYMessageOutput,
				Sequence:  state.sequence.Add(1),
			}
			copy(msg.Data, output) // Deep copy to prevent data races

			// Try to write to safe buffer
			if !m.buffer.TryWrite(msg) {
				// Buffer full - message dropped for backpressure
				m.metrics.droppedMessages.Add(1)
			} else {
				m.metrics.totalMessages.Add(1)
			}

		case event := <-session.EventChan:
			// Handle PTY events
			var msgType SafePTYMessageType
			switch event.Type {
			case aicoder.PTYEventClose:
				msgType = SafePTYMessageClose
			case aicoder.PTYEventResize:
				msgType = SafePTYMessageResize
			default:
				msgType = SafePTYMessageEvent
			}

			// Serialize event data (simplified for now)
			eventData := fmt.Sprintf("event:%s", event.Type)

			msg := &SafePTYMessage{
				SessionID: session.ID,
				Data:      []byte(eventData),
				Timestamp: time.Now(),
				Type:      msgType,
				Sequence:  state.sequence.Add(1),
			}

			// Try to write to safe buffer
			if !m.buffer.TryWrite(msg) {
				m.metrics.droppedMessages.Add(1)
			} else {
				m.metrics.totalMessages.Add(1)
			}

			// If it's a close event, exit monitoring
			if event.Type == aicoder.PTYEventClose {
				return
			}
		}
	}
}

// ensureWriterActive is deprecated - writer loop is now command-driven
// through BubbleTea Update() cycle instead of background goroutine
func (m *LockFreePTYManager) ensureWriterActive() {
	// NO MORE GOROUTINES - Writer will be command-driven
	// The writerActive flag is now used for state tracking only
	m.writerActive.Store(true)
}

// ProcessSessionMessages creates a command to process PTY messages for all active sessions
// This replaces the background goroutine with command-driven processing
func (m *LockFreePTYManager) ProcessSessionMessages() tea.Cmd {
	return func() tea.Msg {
		messages := m.collectSessionMessages()
		if len(messages) > 0 {
			return BatchPTYMsg{Messages: messages}
		}
		return nil // No messages to process
	}
}

// collectSessionMessages processes PTY output from all active sessions (non-blocking)
// This replaces the goroutine-based monitorSession with command-driven collection
func (m *LockFreePTYManager) collectSessionMessages() []tea.Msg {
	var messages []tea.Msg

	// Process each active session with timeout to prevent blocking
	m.sessions.Range(func(key, value interface{}) bool {
		sessionID, ok := key.(string)
		if !ok {
			return true // Skip invalid session ID
		}

		state, ok := value.(*PTYSessionState)
		if !ok {
			// Remove invalid session state
			m.sessions.Delete(sessionID)
			return true
		}

		if !state.active.Load() {
			return true // Skip inactive sessions
		}

		// Process session output with timeout
		sessionMessages := m.processSessionOutput(state, sessionID)
		messages = append(messages, sessionMessages...)

		return true
	})

	return messages
}

// processSessionOutput processes output from a single session with safety timeout
func (m *LockFreePTYManager) processSessionOutput(state *PTYSessionState, sessionID string) []tea.Msg {
	var messages []tea.Msg
	timeout := time.After(50 * time.Millisecond) // 50ms timeout for responsiveness

	for {
		select {
		case output, ok := <-state.session.OutputChan:
			if !ok {
				// Channel closed, session ended - mark inactive and clean up
				if state.active.CompareAndSwap(true, false) {
					// Only decrement if we successfully marked as inactive
					m.metrics.activeSessions.Add(^uint64(0)) // Atomic decrement
				}
				m.sessions.Delete(sessionID)
				return messages
			}

			// Create PTY message with sequence number
			msg := PTYOutputMsg{
				SessionID: sessionID,
				Data:      make([]byte, len(output)),
			}
			copy(msg.Data, output) // Deep copy to prevent data races

			messages = append(messages, msg)
			m.metrics.totalMessages.Add(1)

		case event := <-state.session.EventChan:
			// Handle PTY events
			var eventMsg tea.Msg
			sessionEnded := false

			switch event.Type {
			case aicoder.PTYEventClose:
				sessionEnded = true
				eventMsg = PTYEventMsg{Event: event}
			case aicoder.PTYEventResize:
				eventMsg = PTYEventMsg{Event: event}
			default:
				eventMsg = PTYEventMsg{Event: event}
			}

			messages = append(messages, eventMsg)

			// If it's a close event, stop processing this session
			if sessionEnded {
				if state.active.CompareAndSwap(true, false) {
					// Only decrement if we successfully marked as inactive
					m.metrics.activeSessions.Add(^uint64(0)) // Atomic decrement
				}
				m.sessions.Delete(sessionID)
				return messages
			}

		case <-timeout:
			// Timeout reached - stop processing to maintain responsiveness
			return messages

		case <-m.ctx.Done():
			// Manager context cancelled
			return messages
		}
	}
}

// writerLoop is the single writer that processes messages from the lock-free buffer
// This is the ONLY goroutine that sends messages to updateChan, eliminating races
func (m *LockFreePTYManager) writerLoop() {
	defer m.writerActive.Store(false)

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	m.logStore.Add("system", "System", "Started lock-free PTY writer loop", false)

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-m.writerStopChan:
			return

		case <-ticker.C:
			m.processBatch()
		}
	}
}

// processBatch reads and processes a batch of messages from the lock-free buffer
func (m *LockFreePTYManager) processBatch() {
	startTime := time.Now()

	// Read batch of messages
	batch, count := m.buffer.TryReadBatch(m.batchSize)
	if count == 0 {
		// No messages to process
		return
	}

	// Convert to BubbleTea messages and send as batch
	teaBatch := make([]tea.Msg, 0, count)
	for _, msg := range batch {
		switch msg.Type {
		case SafePTYMessageOutput:
			teaBatch = append(teaBatch, PTYOutputMsg{
				SessionID: msg.SessionID,
				Data:      msg.Data,
			})
		case SafePTYMessageEvent, SafePTYMessageClose, SafePTYMessageResize:
			// Create appropriate event message
			// For now, we'll use a generic PTY event
			teaBatch = append(teaBatch, PTYEventMsg{
				Event: aicoder.PTYEvent{
					Type: aicoder.PTYEventType(string(msg.Data)),
				},
			})
		}
	}

	// Send batch to BubbleTea (single send operation)
	if len(teaBatch) > 0 {
		select {
		case m.updateChan <- BatchPTYMsg{Messages: teaBatch}:
			// Batch sent successfully
		default:
			// UpdateChan full - this should be rare with the increased buffer
			// but we don't want to block the writer loop
			m.logStore.Add("system", "System",
				fmt.Sprintf("Dropped PTY batch of %d messages - updateChan full", len(teaBatch)), true)
		}
	}

	// Update metrics
	m.metrics.batchesProcessed.Add(1)
	m.metrics.averageBatchSize.Store(uint64(count))
	m.metrics.lastPollDuration.Store(time.Since(startTime).Nanoseconds())

	// Update buffer utilization
	bufferMetrics := m.buffer.GetMetrics()
	utilization := (bufferMetrics.CurrentUsed * 10000) / bufferMetrics.BufferSize // Percentage * 100
	m.metrics.bufferUtilization.Store(utilization)
}

// GetMetrics returns a lock-free snapshot of current performance metrics
func (m *LockFreePTYManager) GetMetrics() PTYManagerMetricsSnapshot {
	return PTYManagerMetricsSnapshot{
		ActiveSessions:    m.metrics.activeSessions.Load(),
		TotalMessages:     m.metrics.totalMessages.Load(),
		DroppedMessages:   m.metrics.droppedMessages.Load(),
		BatchesProcessed:  m.metrics.batchesProcessed.Load(),
		AverageBatchSize:  m.metrics.averageBatchSize.Load(),
		LastPollDuration:  m.metrics.lastPollDuration.Load(),
		BufferUtilization: m.metrics.bufferUtilization.Load(),
		Timestamp:         time.Now(),
	}
}

// GetBufferMetrics returns lock-free buffer performance metrics
func (m *LockFreePTYManager) GetBufferMetrics() SafeBufferMetrics {
	return m.buffer.GetMetrics()
}

// Shutdown gracefully shuts down the PTY manager (command-driven)
func (m *LockFreePTYManager) Shutdown() {
	m.logStore.Add("system", "System", "Shutting down lock-free PTY manager", false)

	// Stop all sessions
	m.sessions.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		m.StopSession(sessionID)
		return true
	})

	// Close the safe buffer
	if m.buffer != nil {
		if err := m.buffer.Close(); err != nil {
			m.logStore.Add("system", "System",
				fmt.Sprintf("Error closing PTY buffer: %v", err), true)
		}
		m.buffer = nil
	}

	// NO MORE GOROUTINES - No need to stop writer loop
	// The writerActive flag is now used for state tracking only
	m.writerActive.Store(false)

	// Cancel context
	m.cancel()

	m.logStore.Add("system", "System", "Lock-free PTY manager shutdown complete (command-driven)", false)
}

// BatchPTYMsg represents a batch of PTY messages for BubbleTea
type BatchPTYMsg struct {
	Messages []tea.Msg
}
