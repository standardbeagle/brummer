package state

import (
	"sync"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/types"
)

// OperationsTracker tracks in-flight operations and their status
type OperationsTracker struct {
	operations map[string]*Operation
	mu         sync.RWMutex
	maxSize    int
	stats      *OperationStats
}

// Operation represents an in-flight operation
type Operation struct {
	ID          string                  `json:"id"`
	CommandType types.CommandType       `json:"commandType"`
	Status      OperationStatus         `json:"status"`
	StartTime   time.Time               `json:"startTime"`
	EndTime     time.Time               `json:"endTime,omitempty"`
	Duration    time.Duration           `json:"duration,omitempty"`
	Result      interface{}              `json:"result,omitempty"`
	Error       error                   `json:"error,omitempty"`
	Context     types.OperationContext   `json:"context"`
	Retries     int                     `json:"retries"`
	Timeout     time.Duration           `json:"timeout"`
	Progress    *Progress               `json:"progress,omitempty"`
	Metadata    map[string]interface{}  `json:"metadata,omitempty"`

	// Internal tracking
	createdAt   time.Time `json:"createdAt"`
	updatedAt   time.Time `json:"updatedAt"`
}

// Progress represents operation progress
type Progress struct {
	Current     int     `json:"current"`
	Total       int     `json:"total"`
	Percentage  float64 `json:"percentage"`
	Message     string  `json:"message"`
	EstimatedEnd time.Time `json:"estimatedEnd,omitempty"`
}

// OperationStatus represents the status of an operation
type OperationStatus int

const (
	OperationPending OperationStatus = iota
	OperationRunning
	OperationCompleted
	OperationFailed
	OperationCancelled
	OperationTimedOut
	OperationPaused
)

// OperationStats provides statistics about operations
type OperationStats struct {
	TotalOperations     int                        `json:"totalOperations"`
	ActiveOperations    int                        `json:"activeOperations"`
	CompletedOperations  int                        `json:"completedOperations"`
	FailedOperations     int                        `json:"failedOperations"`
	AverageDuration      time.Duration              `json:"averageDuration"`
	FastestOperation     time.Duration              `json:"fastestOperation"`
	SlowestOperation     time.Duration              `json:"slowestOperation"`
	OperationsByType    map[types.CommandType]int `json:"operationsByType"`
	SuccessRate          float64                    `json:"successRate"`
	RetryRate            float64                    `json:"retryRate"`
	TimeoutRate          float64                    `json:"timeoutRate"`
}

// OperationFilter provides filtering capabilities for operations
type OperationFilter struct {
	Statuses     []OperationStatus      `json:"statuses"`
	CommandTypes []types.CommandType    `json:"commandTypes"`
	StartTime   *time.Time             `json:"startTime,omitempty"`
	EndTime     *time.Time             `json:"endTime,omitempty"`
	MinDuration *time.Duration         `json:"minDuration,omitempty"`
	MaxDuration *time.Duration         `json:"maxDuration,omitempty"`
	HasError    *bool                  `json:"hasError,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
}

// NewOperationsTracker creates a new operations tracker
func NewOperationsTracker() *OperationsTracker {
	return &OperationsTracker{
		operations: make(map[string]*Operation),
		maxSize:    1000,
		stats:      NewOperationStats(),
	}
}

// NewOperationStats creates new operation statistics
func NewOperationStats() *OperationStats {
	return &OperationStats{
		OperationsByType: make(map[types.CommandType]int),
	}
}

// AddOperation adds a new operation to track
func (t *OperationsTracker) AddOperation(id string, commandType types.CommandType, context types.OperationContext, timeout time.Duration) *Operation {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if we need to clean up old operations
	if len(t.operations) >= t.maxSize {
		t.cleanupOldOperations()
	}

	now := time.Now()
	operation := &Operation{
		ID:          id,
		CommandType: commandType,
		Status:      OperationPending,
		StartTime:   now,
		Context:     context,
		Timeout:     timeout,
		Retries:     0,
		Progress:    nil,
		Metadata:    make(map[string]interface{}),
		createdAt:   now,
		updatedAt:   now,
	}

	t.operations[id] = operation
	t.stats.TotalOperations++
	t.stats.ActiveOperations++
	t.stats.OperationsByType[commandType]++

	return operation
}

// UpdateOperationStatus updates the status of an operation
func (t *OperationsTracker) UpdateOperationStatus(id string, status OperationStatus) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	operation, exists := t.operations[id]
	if !exists {
		return false
	}

	oldStatus := operation.Status
	operation.Status = status
	operation.updatedAt = time.Now()

	// Update stats based on status change
	switch status {
	case OperationRunning:
		if oldStatus == OperationPending {
			operation.StartTime = time.Now()
		}
	case OperationCompleted, OperationFailed, OperationCancelled, OperationTimedOut:
		if oldStatus == OperationRunning || oldStatus == OperationPending {
			operation.EndTime = time.Now()
			operation.Duration = operation.EndTime.Sub(operation.StartTime)
			t.stats.ActiveOperations--

			// Update completion stats
			switch status {
			case OperationCompleted:
				t.stats.CompletedOperations++
			case OperationFailed:
				t.stats.FailedOperations++
			case OperationTimedOut:
				t.stats.FailedOperations++
			}

			// Update duration stats
			t.updateDurationStats(operation.Duration)

			// Update rates
			t.updateRates()
		}
	case OperationPaused:
		// No special handling needed
	}

	return true
}

// SetOperationResult sets the result of a completed operation
func (t *OperationsTracker) SetOperationResult(id string, result interface{}) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	operation, exists := t.operations[id]
	if !exists {
		return false
	}

	operation.Result = result
	operation.updatedAt = time.Now()

	return true
}

// SetOperationError sets the error of a failed operation
func (t *OperationsTracker) SetOperationError(id string, err error) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	operation, exists := t.operations[id]
	if !exists {
		return false
	}

	operation.Error = err
	operation.updatedAt = time.Now()

	return true
}

// UpdateOperationProgress updates the progress of an operation
func (t *OperationsTracker) UpdateOperationProgress(id string, current, total int, message string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	operation, exists := t.operations[id]
	if !exists {
		return false
	}

	progress := &Progress{
		Current:    current,
		Total:      total,
		Percentage: float64(current) / float64(total) * 100,
		Message:    message,
	}

	if total > 0 && current > 0 {
			elapsed := time.Since(operation.StartTime)
			estimatedTotal := time.Duration(float64(elapsed) * float64(total) / float64(current))
			progress.EstimatedEnd = operation.StartTime.Add(estimatedTotal)
		}

	operation.Progress = progress
	operation.updatedAt = time.Now()

	return true
}

// IncrementOperationRetries increments the retry count for an operation
func (t *OperationsTracker) IncrementOperationRetries(id string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	operation, exists := t.operations[id]
	if !exists {
		return 0
	}

	operation.Retries++
	operation.updatedAt = time.Now()

	return operation.Retries
}

// SetOperationMetadata sets metadata for an operation
func (t *OperationsTracker) SetOperationMetadata(id string, key string, value interface{}) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	operation, exists := t.operations[id]
	if !exists {
		return false
	}

	if operation.Metadata == nil {
		operation.Metadata = make(map[string]interface{})
	}

	operation.Metadata[key] = value
	operation.updatedAt = time.Now()

	return true
}

// GetOperation returns an operation by ID
func (t *OperationsTracker) GetOperation(id string) (*Operation, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	operation, exists := t.operations[id]
	if !exists {
		return nil, false
	}

	// Return a copy
	return t.copyOperation(operation), true
}

// GetAllOperations returns all operations
func (t *OperationsTracker) GetAllOperations() map[string]*Operation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*Operation)
	for id, operation := range t.operations {
		result[id] = t.copyOperation(operation)
	}

	return result
}

// GetActiveOperations returns all currently active operations
func (t *OperationsTracker) GetActiveOperations() map[string]*Operation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*Operation)
	for id, operation := range t.operations {
		if operation.Status == OperationRunning || operation.Status == OperationPending {
			result[id] = t.copyOperation(operation)
		}
	}

	return result
}

// GetOperationsByStatus returns operations filtered by status
func (t *OperationsTracker) GetOperationsByStatus(statuses ...OperationStatus) map[string]*Operation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*Operation)
	for id, operation := range t.operations {
		for _, status := range statuses {
			if operation.Status == status {
				result[id] = t.copyOperation(operation)
				break
			}
		}
	}

	return result
}

// GetOperationsByType returns operations filtered by command type
func (t *OperationsTracker) GetOperationsByType(commandTypes ...types.CommandType) map[string]*Operation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*Operation)
	for id, operation := range t.operations {
		for _, cmdType := range commandTypes {
			if operation.CommandType == cmdType {
				result[id] = t.copyOperation(operation)
				break
			}
		}
	}

	return result
}

// GetFilteredOperations returns operations filtered by the provided filter
func (t *OperationsTracker) GetFilteredOperations(filter *OperationFilter) map[string]*Operation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*Operation)
	count := 0

	for id, operation := range t.operations {
		if t.matchesFilter(operation, filter) {
			result[id] = t.copyOperation(operation)
			count++

			// Check limit
			if filter.Limit > 0 && count >= filter.Limit {
				break
			}
		}
	}

	return result
}

// matchesFilter checks if an operation matches the provided filter
func (t *OperationsTracker) matchesFilter(operation *Operation, filter *OperationFilter) bool {
	// Check status filter
	if len(filter.Statuses) > 0 {
		found := false
		for _, status := range filter.Statuses {
			if operation.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check command type filter
	if len(filter.CommandTypes) > 0 {
		found := false
		for _, cmdType := range filter.CommandTypes {
			if operation.CommandType == cmdType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check start time filter
	if filter.StartTime != nil && operation.StartTime.Before(*filter.StartTime) {
		return false
	}

	// Check end time filter
	if filter.EndTime != nil {
		if operation.EndTime.IsZero() {
			return false
		}
		if operation.EndTime.After(*filter.EndTime) {
			return false
		}
	}

	// Check duration filters
	if !operation.EndTime.IsZero() {
		duration := operation.EndTime.Sub(operation.StartTime)
		if filter.MinDuration != nil && duration < *filter.MinDuration {
			return false
		}
		if filter.MaxDuration != nil && duration > *filter.MaxDuration {
			return false
		}
	}

	// Check error filter
	if filter.HasError != nil {
		hasError := operation.Error != nil
		if hasError != *filter.HasError {
			return false
		}
	}

	return true
}

// GetOperationsWithTimeout returns operations that have timed out
func (t *OperationsTracker) GetOperationsWithTimeout() map[string]*Operation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*Operation)
	now := time.Now()

	for id, operation := range t.operations {
		if operation.Status == OperationRunning && operation.Timeout > 0 {
			if now.Sub(operation.StartTime) > operation.Timeout {
				result[id] = t.copyOperation(operation)
			}
		}
	}

	return result
}

// GetLongRunningOperations returns operations that have been running longer than the specified duration
func (t *OperationsTracker) GetLongRunningOperations(duration time.Duration) map[string]*Operation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*Operation)
	now := time.Now()

	for id, operation := range t.operations {
		if operation.Status == OperationRunning || operation.Status == OperationPending {
			if now.Sub(operation.StartTime) > duration {
				result[id] = t.copyOperation(operation)
			}
		}
	}

	return result
}

// RemoveOperation removes an operation from tracking
func (t *OperationsTracker) RemoveOperation(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	operation, exists := t.operations[id]
	if !exists {
		return false
	}

	// Update stats if operation was active
	if operation.Status == OperationRunning || operation.Status == OperationPending {
		t.stats.ActiveOperations--
	}

	delete(t.operations, id)
	return true
}

// Cleanup removes old completed operations
func (t *OperationsTracker) Cleanup(maxAge time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, operation := range t.operations {
		if !operation.EndTime.IsZero() && operation.EndTime.Before(cutoff) {
			delete(t.operations, id)
		}
	}
}

// cleanupOldOperations removes the oldest operations when at capacity
func (t *OperationsTracker) cleanupOldOperations() {
	if len(t.operations) <= t.maxSize {
		return
	}

	// Find oldest completed operations
	var oldestID string
	var oldestTime time.Time

	for id, operation := range t.operations {
		if !operation.EndTime.IsZero() && (oldestTime.IsZero() || operation.EndTime.Before(oldestTime)) {
			oldestID = id
			oldestTime = operation.EndTime
		}
	}

	if oldestID != "" {
		delete(t.operations, oldestID)
	}
}

// GetStats returns operation statistics
func (t *OperationsTracker) GetStats() *OperationStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Update counts
	t.stats.TotalOperations = len(t.operations)
	t.stats.ActiveOperations = 0
	t.stats.CompletedOperations = 0
	t.stats.FailedOperations = 0

	totalDuration := time.Duration(0)
	durationCount := 0
	var minDuration time.Duration = 0
	var maxDuration time.Duration = 0

	for _, operation := range t.operations {
		switch operation.Status {
		case OperationRunning, OperationPending:
			t.stats.ActiveOperations++
		case OperationCompleted:
			t.stats.CompletedOperations++
		case OperationFailed, OperationTimedOut:
			t.stats.FailedOperations++
		}

		// Calculate duration stats
		if !operation.EndTime.IsZero() {
			duration := operation.EndTime.Sub(operation.StartTime)
			totalDuration += duration
			durationCount++

			if minDuration == 0 || duration < minDuration {
				minDuration = duration
			}
			if maxDuration == 0 || duration > maxDuration {
				maxDuration = duration
			}
		}
	}

	// Calculate average duration
	if durationCount > 0 {
		t.stats.AverageDuration = totalDuration / time.Duration(durationCount)
		t.stats.FastestOperation = minDuration
		t.stats.SlowestOperation = maxDuration
	}

	// Calculate rates
	t.updateRates()

	return t.stats
}

// updateDurationStats updates duration statistics
func (t *OperationsTracker) updateDurationStats(duration time.Duration) {
	if t.stats.AverageDuration == 0 {
		t.stats.AverageDuration = duration
		t.stats.FastestOperation = duration
		t.stats.SlowestOperation = duration
		return
	}

	// Update average
	t.stats.AverageDuration = time.Duration(
		(float64(t.stats.AverageDuration) + float64(duration)) / 2,
	)

	// Update min/max
	if duration < t.stats.FastestOperation {
		t.stats.FastestOperation = duration
	}
	if duration > t.stats.SlowestOperation {
		t.stats.SlowestOperation = duration
	}
}

// updateRates calculates success, retry, and timeout rates
func (t *OperationsTracker) updateRates() {
	total := t.stats.CompletedOperations + t.stats.FailedOperations
	if total == 0 {
		t.stats.SuccessRate = 0
		t.stats.TimeoutRate = 0
		return
	}

	t.stats.SuccessRate = float64(t.stats.CompletedOperations) / float64(total)

	// Count timeouts and retries
	timeoutCount := 0
	retryCount := 0
	totalRetries := 0

	for _, operation := range t.operations {
		if operation.Status == OperationTimedOut {
			timeoutCount++
		}
		if operation.Retries > 0 {
			retryCount++
			totalRetries += operation.Retries
		}
	}

	if total > 0 {
		t.stats.TimeoutRate = float64(timeoutCount) / float64(total)
	}

	if retryCount > 0 {
		t.stats.RetryRate = float64(totalRetries) / float64(retryCount)
	}
}

// copyOperation creates a deep copy of an operation
func (t *OperationsTracker) copyOperation(original *Operation) *Operation {
	copy := &Operation{
		ID:          original.ID,
		CommandType: original.CommandType,
		Status:      original.Status,
		StartTime:   original.StartTime,
		EndTime:     original.EndTime,
		Duration:    original.Duration,
		Result:      original.Result,
		Error:       original.Error,
		Context:     original.Context,
		Retries:     original.Retries,
		Timeout:     original.Timeout,
		Metadata:    make(map[string]interface{}),
		createdAt:   original.createdAt,
		updatedAt:   original.updatedAt,
	}

	if original.Progress != nil {
		copy.Progress = &Progress{
			Current:      original.Progress.Current,
			Total:        original.Progress.Total,
			Percentage:   original.Progress.Percentage,
			Message:      original.Progress.Message,
			EstimatedEnd: original.Progress.EstimatedEnd,
		}
	}

	for k, v := range original.Metadata {
		copy.Metadata[k] = v
	}

	return copy
}

// Reset clears all operations and resets statistics
func (t *OperationsTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.operations = make(map[string]*Operation)
	t.stats = NewOperationStats()
}

// ExportOperations exports operations in a format suitable for persistence
func (t *OperationsTracker) ExportOperations() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]interface{})
	result["operations"] = t.operations
	result["stats"] = t.stats
	result["exportedAt"] = time.Now()

	return result
}

// ImportOperations imports operations from exported data
func (t *OperationsTracker) ImportOperations(data map[string]interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear current state
	t.Reset()

	// Import operations
	if operationsData, ok := data["operations"].(map[string]*Operation); ok {
		for id, operation := range operationsData {
			t.operations[id] = operation
		}
	}

	// Import stats
	if statsData, ok := data["stats"].(*OperationStats); ok {
		t.stats = statsData
	}

	return nil
}