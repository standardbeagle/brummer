package handlers

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/commands"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	tea "github.com/charmbracelet/bubbletea"
)

// StateHandler manages application state updates in the Bubble Tea Update loop
type StateHandler struct {
	version       uint64
	state         map[string]interface{}
	operations    map[string]*Operation
	mu            sync.RWMutex
	responseTime  atomic.Value // time.Duration
	updateCount   atomic.Int64
	lastUpdate    atomic.Value // time.Time
	collector     *messages.MessageCollector
	maxCacheSize  int
}

// Operation represents an in-flight operation
type Operation struct {
	ID          string                       `json:"id"`
	CommandType commands.CommandType         `json:"commandType"`
	Status      commands.OperationStatus     `json:"status"`
	StartTime   time.Time                    `json:"startTime"`
	EndTime     time.Time                    `json:"endTime"`
	Result      interface{}                  `json:"result,omitempty"`
	Error       error                        `json:"error,omitempty"`
	Context     commands.OperationContext    `json:"context"`
	Retries     int                          `json:"retries"`
	Timeout     time.Duration                `json:"timeout"`
}

// State represents the current application state
type State struct {
	Version      uint64                       `json:"version"`
	Data         map[string]interface{}        `json:"data"`
	UIComponents map[string]*UIComponent       `json:"uiComponents"`
	Operations   map[string]*Operation         `json:"operations"`
	Cache        map[string]*CacheEntry        `json:"cache"`
	LastUpdate   time.Time                    `json:"lastUpdate"`
	ResponseTime time.Duration                `json:"responseTime"`
}

// UIComponent represents the state of a UI component
type UIComponent struct {
	ID         string      `json:"id"`
	Type       string      `json:"type"`
	Active     bool        `json:"active"`
	Loading    bool        `json:"loading"`
	Error      error       `json:"error,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	LastUpdate time.Time   `json:"lastUpdate"`
}

// CacheEntry represents a cached value with TTL
type CacheEntry struct {
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expiresAt"`
	Version   uint64      `json:"version"`
}

// NewStateHandler creates a new state handler
func NewStateHandler(maxCacheSize int) *StateHandler {
	handler := &StateHandler{
		version:      0,
		state:        make(map[string]interface{}),
		operations:   make(map[string]*Operation),
		collector:    messages.NewMessageCollector(1000),
		maxCacheSize: maxCacheSize,
	}
	handler.responseTime.Store(time.Duration(0))
	handler.lastUpdate.Store(time.Time{})
	return handler
}

// Update processes a message and returns the updated model and commands
func (h *StateHandler) Update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		h.updateResponseTime(duration)
		h.updateCount.Add(1)
		h.lastUpdate.Store(time.Now())
	}()

	switch m := msg.(type) {
	case *messages.CommandStartedMessage:
		return h.handleCommandStarted(m, model)
	case *messages.CommandCompletedMessage:
		return h.handleCommandCompleted(m, model)
	case *messages.CommandFailedMessage:
		return h.handleCommandFailed(m, model)
	case *messages.CommandCancelledMessage:
		return h.handleCommandCancelled(m, model)
	case *messages.CommandProgressMessage:
		return h.handleCommandProgress(m, model)
	case *messages.BatchCommandStartedMessage:
		return h.handleBatchCommandStarted(m, model)
	case *messages.BatchCommandCompletedMessage:
		return h.handleBatchCommandCompleted(m, model)
	case *messages.RetryAttemptMessage:
		return h.handleRetryAttempt(m, model)
	case *messages.StateUpdateMessage:
		return h.handleStateUpdate(m, model)
	case *messages.OperationTimeoutMessage:
		return h.handleOperationTimeout(m, model)
	case *messages.ResourceCleanupMessage:
		return h.handleResourceCleanup(m, model)
	default:
		return model, nil
	}
}

// handleCommandStarted processes a command started message
func (h *StateHandler) handleCommandStarted(msg *messages.CommandStartedMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	operation := &Operation{
		ID:          msg.ID,
		CommandType: msg.CommandType,
		Status:      commands.OperationRunning,
		StartTime:   msg.Timestamp,
		Context:     msg.Context,
		Retries:     0,
	}

	h.operations[msg.ID] = operation
	h.incrementVersion()

	// Add to message collector
	h.collector.Add(msg)

	return model, nil
}

// handleCommandCompleted processes a command completed message
func (h *StateHandler) handleCommandCompleted(msg *messages.CommandCompletedMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Update operation
	if op, exists := h.operations[msg.ID]; exists {
		op.Status = commands.OperationCompleted
		op.EndTime = msg.Timestamp
		op.Result = msg.Result
	}

	// Update state based on command type
	h.updateStateForCommand(msg.CommandType, msg.Result, msg.Context.Version)

	// Cache result if applicable
	h.cacheResult(msg.ID, msg.Result, msg.Context.Version)

	h.incrementVersion()

	// Add to message collector
	h.collector.Add(msg)

	// Generate cleanup command if needed
	return model, h.generateCleanupCommand(msg.ID)
}

// handleCommandFailed processes a command failed message
func (h *StateHandler) handleCommandFailed(msg *messages.CommandFailedMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Update operation
	if op, exists := h.operations[msg.ID]; exists {
		op.Status = commands.OperationFailed
		op.EndTime = msg.Timestamp
		op.Error = msg.Error
	}

	// Update error state
	h.updateErrorState(msg.ID, msg.Error, msg.Context.Version)

	h.incrementVersion()

	// Add to message collector
	h.collector.Add(msg)

	// Generate retry command if retryable
	if msg.Retryable {
		return model, h.generateRetryCommand(msg.ID, msg.CommandType, msg.Context)
	}

	return model, h.generateCleanupCommand(msg.ID)
}

// handleCommandCancelled processes a command cancelled message
func (h *StateHandler) handleCommandCancelled(msg *messages.CommandCancelledMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Update operation
	if op, exists := h.operations[msg.ID]; exists {
		op.Status = commands.OperationCancelled
		op.EndTime = msg.Timestamp
	}

	h.incrementVersion()

	// Add to message collector
	h.collector.Add(msg)

	return model, h.generateCleanupCommand(msg.ID)
}

// handleCommandProgress processes a command progress message
func (h *StateHandler) handleCommandProgress(msg *messages.CommandProgressMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Update operation progress
	if op, exists := h.operations[msg.ID]; exists {
		// Progress is tracked at the operation level if needed
		_ = op
	}

	// Add to message collector
	h.collector.Add(msg)

	return model, nil
}

// handleBatchCommandStarted processes a batch command started message
func (h *StateHandler) handleBatchCommandStarted(msg *messages.BatchCommandStartedMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	operation := &Operation{
		ID:          msg.ID,
		CommandType: commands.CommandTypeSystem, // Batch commands are system commands
		Status:      commands.OperationRunning,
		StartTime:   msg.Timestamp,
		Context:     msg.Context,
		Retries:     0,
	}

	h.operations[msg.ID] = operation
	h.incrementVersion()

	// Add to message collector
	h.collector.Add(msg)

	return model, nil
}

// handleBatchCommandCompleted processes a batch command completed message
func (h *StateHandler) handleBatchCommandCompleted(msg *messages.BatchCommandCompletedMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Update operation
	if op, exists := h.operations[msg.ID]; exists {
		op.Status = commands.OperationCompleted
		op.EndTime = msg.Timestamp
		op.Result = msg.Results
	}

	// Process batch results
	for _, result := range msg.Results {
		if result.Error == nil {
			h.cacheResult(result.CommandID, result.Result, msg.Context.Version)
		}
	}

	h.incrementVersion()

	// Add to message collector
	h.collector.Add(msg)

	return model, h.generateCleanupCommand(msg.ID)
}

// handleRetryAttempt processes a retry attempt message
func (h *StateHandler) handleRetryAttempt(msg *messages.RetryAttemptMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Update operation retry count
	if op, exists := h.operations[msg.OriginalCommandID]; exists {
		op.Retries = msg.AttemptNumber
	}

	// Add to message collector
	h.collector.Add(msg)

	return model, nil
}

// handleStateUpdate processes a state update message
func (h *StateHandler) handleStateUpdate(msg *messages.StateUpdateMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Validate version
	if msg.NewVersion <= h.version {
		return model, nil // Ignore stale updates
	}

	// Update version
	h.version = msg.NewVersion

	// Update specific state based on type
	h.updateStateByType(msg.StateType, msg.ChangeType, msg.OldVersion, msg.NewVersion, msg.Description)

	// Add to message collector
	h.collector.Add(msg)

	return model, nil
}

// handleOperationTimeout processes an operation timeout message
func (h *StateHandler) handleOperationTimeout(msg *messages.OperationTimeoutMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Update operation
	if op, exists := h.operations[msg.ID]; exists {
		op.Status = commands.OperationFailed
		op.EndTime = msg.Timestamp
		op.Error = fmt.Errorf("operation timed out after %v", msg.TimeoutDuration)
	}

	h.incrementVersion()

	// Add to message collector
	h.collector.Add(msg)

	return model, h.generateCleanupCommand(msg.ID)
}

// handleResourceCleanup processes a resource cleanup message
func (h *StateHandler) handleResourceCleanup(msg *messages.ResourceCleanupMessage, model tea.Model) (tea.Model, tea.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove operation from tracking
	delete(h.operations, msg.ResourceID)

	// Clean up cache entries if needed
	if msg.ResourceType == "cache" {
		h.cleanupCache(msg.ResourceID)
	}

	// Add to message collector
	h.collector.Add(msg)

	return model, nil
}

// State management helper methods

// incrementVersion atomically increments the state version
func (h *StateHandler) incrementVersion() {
	atomic.AddUint64(&h.version, 1)
}

// updateStateForCommand updates state based on command result
func (h *StateHandler) updateStateForCommand(cmdType commands.CommandType, result interface{}, version uint64) {
	switch cmdType {
	case commands.CommandTypeIO:
		h.state["lastIOOperation"] = result
	case commands.CommandTypeProcess:
		h.state["lastProcessOperation"] = result
	case commands.CommandTypeNetwork:
		h.state["lastNetworkOperation"] = result
	case commands.CommandTypeSystem:
		h.state["lastSystemOperation"] = result
	}
	h.state["version"] = version
}

// updateErrorState updates error state in the application
func (h *StateHandler) updateErrorState(operationID string, err error, version uint64) {
	h.state["lastError"] = map[string]interface{}{
		"operationID": operationID,
		"error":       err.Error(),
		"timestamp":   time.Now(),
		"version":     version,
	}
}

// updateStateByType updates state based on the type of change
func (h *StateHandler) updateStateByType(stateType, changeType string, oldVersion, newVersion uint64, description string) {
	h.state[stateType] = map[string]interface{}{
		"oldVersion":  oldVersion,
		"newVersion":  newVersion,
		"changeType":  changeType,
		"description": description,
		"timestamp":   time.Now(),
	}
}

// cacheResult caches a result with TTL
func (h *StateHandler) cacheResult(key string, result interface{}, version uint64) {
	if len(h.state) >= h.maxCacheSize {
		h.cleanupOldestCacheEntry()
	}

	entry := &CacheEntry{
		Value:     result,
		ExpiresAt: time.Now().Add(5 * time.Minute), // 5 minute TTL
		Version:   version,
	}

	h.state["cache_"+key] = entry
}

// cleanupOldestCacheEntry removes the oldest cache entry
func (h *StateHandler) cleanupOldestCacheEntry() {
	var oldestKey string
	var oldestTime time.Time

	for key, value := range h.state {
		if entry, ok := value.(*CacheEntry); ok {
			if oldestKey == "" || entry.ExpiresAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.ExpiresAt
			}
		}
	}

	if oldestKey != "" {
		delete(h.state, oldestKey)
	}
}

// cleanupCache removes a specific cache entry
func (h *StateHandler) cleanupCache(key string) {
	delete(h.state, "cache_"+key)
}

// updateResponseTime updates the rolling average response time
func (h *StateHandler) updateResponseTime(duration time.Duration) {
	current := h.responseTime.Load().(time.Duration)
	// Simple exponential moving average
	alpha := 0.1
	newAverage := time.Duration(float64(current)*(1-alpha) + float64(duration)*alpha)
	h.responseTime.Store(newAverage)
}

// Command generation helper methods

// generateCleanupCommand creates a command to clean up resources
func (h *StateHandler) generateCleanupCommand(operationID string) tea.Cmd {
	return func() tea.Msg {
		return messages.NewResourceCleanupMessage(
			"cleanup-"+operationID,
			"operation",
			operationID,
			"operation completed",
			commands.SystemOperationContext(h.version),
		)
	}
}

// generateRetryCommand creates a command to retry a failed operation
func (h *StateHandler) generateRetryCommand(operationID string, cmdType commands.CommandType, context commands.OperationContext) tea.Cmd {
	return func() tea.Msg {
		return messages.NewRetryAttemptMessage(
			"retry-"+operationID,
			operationID,
			1, // First retry
			3, // Max retries
			fmt.Errorf("retrying failed operation"),
			time.Second, // Backoff delay
			context,
		)
	}
}

// Public accessor methods

// GetVersion returns the current state version
func (h *StateHandler) GetVersion() uint64 {
	return atomic.LoadUint64(&h.version)
}

// GetState returns a copy of the current state
func (h *StateHandler) GetState() State {
	h.mu.RLock()
	defer h.mu.RUnlock()

	state := State{
		Version:      h.version,
		Data:         make(map[string]interface{}),
		UIComponents: make(map[string]*UIComponent),
		Operations:   make(map[string]*Operation),
		Cache:        make(map[string]*CacheEntry),
		LastUpdate:   h.lastUpdate.Load().(time.Time),
		ResponseTime: h.responseTime.Load().(time.Duration),
	}

	// Copy data
	for k, v := range h.state {
		state.Data[k] = v
	}

	// Copy operations
	for k, v := range h.operations {
		state.Operations[k] = v
	}

	// Extract cache entries
	for k, v := range h.state {
		if entry, ok := v.(*CacheEntry); ok {
			state.Cache[k] = entry
		}
	}

	return state
}

// GetOperation returns an operation by ID
func (h *StateHandler) GetOperation(id string) (*Operation, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	op, exists := h.operations[id]
	if !exists {
		return nil, false
	}

	// Return a copy
	return &Operation{
		ID:          op.ID,
		CommandType: op.CommandType,
		Status:      op.Status,
		StartTime:   op.StartTime,
		EndTime:     op.EndTime,
		Result:      op.Result,
		Error:       op.Error,
		Context:     op.Context,
		Retries:     op.Retries,
		Timeout:     op.Timeout,
	}, true
}

// GetResponseTime returns the current average response time
func (h *StateHandler) GetResponseTime() time.Duration {
	return h.responseTime.Load().(time.Duration)
}

// GetUpdateCount returns the total number of updates processed
func (h *StateHandler) GetUpdateCount() int64 {
	return h.updateCount.Load()
}

// GetMessages returns recent messages from the collector
func (h *StateHandler) GetMessages(limit int) []messages.Message {
	allMessages := h.collector.GetMessages()
	if limit >= len(allMessages) {
		return allMessages
	}
	return allMessages[len(allMessages)-limit:]
}

// Cleanup removes expired cache entries and completed operations
func (h *StateHandler) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	// Clean up expired cache entries
	for key, value := range h.state {
		if entry, ok := value.(*CacheEntry); ok && entry.ExpiresAt.Before(now) {
			delete(h.state, key)
		}
	}

	// Clean up completed operations older than 5 minutes
	cutoff := now.Add(-5 * time.Minute)
	for id, op := range h.operations {
		if (op.Status == commands.OperationCompleted || op.Status == commands.OperationFailed || op.Status == commands.OperationCancelled) &&
			op.EndTime.Before(cutoff) {
			delete(h.operations, id)
		}
	}
}

// GetStats returns handler statistics
func (h *StateHandler) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	activeOps := 0
	completedOps := 0
	failedOps := 0

	for _, op := range h.operations {
		switch op.Status {
		case commands.OperationRunning, commands.OperationPending:
			activeOps++
		case commands.OperationCompleted:
			completedOps++
		case commands.OperationFailed, commands.OperationCancelled:
			failedOps++
		}
	}

	cacheEntries := 0
	for _, value := range h.state {
		if _, ok := value.(*CacheEntry); ok {
			cacheEntries++
		}
	}

	return map[string]interface{}{
		"version":        h.version,
		"responseTime":   h.responseTime.Load().(time.Duration),
		"updateCount":    h.updateCount.Load(),
		"lastUpdate":     h.lastUpdate.Load().(time.Time),
		"activeOps":      activeOps,
		"completedOps":   completedOps,
		"failedOps":      failedOps,
		"totalOps":       len(h.operations),
		"cacheEntries":   cacheEntries,
		"messagesStored": h.collector.Size(),
	}
}