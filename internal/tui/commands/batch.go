package commands

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// BatchCommand represents a command that executes multiple commands together
type BatchCommand struct {
	*BaseCommand
	commands       []Command
	executionMode  BatchExecutionMode
	results        []BatchCommandResult
	mu             sync.RWMutex
	startTime      time.Time
	timeout        time.Duration
	failFast       bool
	parallelLimit  int
	Model          interface{}
}

// BatchExecutionMode defines how commands in a batch should be executed
type BatchExecutionMode int

const (
	BatchSequential BatchExecutionMode = iota
	BatchParallel
	BatchParallelLimited
)

// BatchResultMsg represents the result of a batch operation
type BatchResultMsg struct {
	ID            string                `json:"id"`
	Type          CommandType          `json:"type"`
	Results       []BatchCommandResult `json:"results,omitempty"`
	Error         error                `json:"error,omitempty"`
	Context       OperationContext     `json:"context"`
	Duration      time.Duration        `json:"duration"`
	ExecutionMode BatchExecutionMode   `json:"executionMode"`
	TotalCommands int                  `json:"totalCommands"`
	SuccessCount  int                  `json:"successCount"`
	FailureCount  int                  `json:"failureCount"`
}

// BatchCommandResult represents the result of a single command in a batch
type BatchCommandResult struct {
	CommandID    string        `json:"commandId"`
	CommandType  CommandType   `json:"commandType"`
	Result       interface{}   `json:"result,omitempty"`
	Error        error         `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
	StartTime    time.Time     `json:"startTime"`
	EndTime      time.Time     `json:"endTime"`
	AttemptCount int           `json:"attemptCount,omitempty"`
}

// NewBatchCommand creates a new batch command
func NewBatchCommand(id string, commands []Command, mode BatchExecutionMode, context OperationContext, model interface{}) *BatchCommand {
	return &BatchCommand{
		BaseCommand:    NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		commands:       commands,
		executionMode:  mode,
		results:        make([]BatchCommandResult, 0, len(commands)),
		timeout:        context.Timeout,
		failFast:       false,
		parallelLimit:  10,
		Model:          model,
	}
}

// WithFailFast sets the fail-fast behavior - if true, stops on first error
func (b *BatchCommand) WithFailFast(failFast bool) *BatchCommand {
	b.failFast = failFast
	return b
}

// WithTimeout sets a timeout for the entire batch operation
func (b *BatchCommand) WithTimeout(timeout time.Duration) *BatchCommand {
	b.timeout = timeout
	return b
}

// WithParallelLimit sets the maximum number of parallel commands
func (b *BatchCommand) WithParallelLimit(limit int) *BatchCommand {
	b.parallelLimit = limit
	return b
}

// Run executes the batch command
func (b *BatchCommand) Run() tea.Msg {
	b.startTime = time.Now()

	switch b.executionMode {
	case BatchSequential:
		return b.runSequential()
	case BatchParallel:
		return b.runParallel()
	case BatchParallelLimited:
		return b.runParallelLimited()
	default:
		return BatchResultMsg{
			ID:    b.ID(),
			Type:  b.Type(),
			Error: fmt.Errorf("unknown batch execution mode: %d", b.executionMode),
			Context: b.Context(),
			Duration: time.Since(b.startTime),
			ExecutionMode: b.executionMode,
			TotalCommands: len(b.commands),
		}
	}
}

// runSequential executes commands one after another
func (b *BatchCommand) runSequential() tea.Msg {
	for i, command := range b.commands {
		if b.failFast && i > 0 {
			b.mu.RLock()
			lastResult := b.results[len(b.results)-1]
			b.mu.RUnlock()
			if lastResult.Error != nil {
				break
			}
		}

		result := b.executeCommand(command)
		b.addResult(result)

		// Check if we should stop due to failFast
		if b.failFast && result.Error != nil {
			break
		}
	}

	return b.buildBatchResult()
}

// runParallel executes all commands in parallel using async commands
func (b *BatchCommand) runParallel() tea.Msg {
	// Use async batch command instead of goroutines
	asyncCommands := NewAsyncBatchCommands()
	asyncBatch := asyncCommands.factory.CreateAsyncParallelBatch(b.ID()+"-async", b.commands, b.Model)

	// Configure async batch with same settings
	if abc, ok := asyncBatch.(*AsyncBatchCommand); ok {
		abc.WithFailFast(b.failFast)
		abc.WithTimeout(b.timeout)
	}

	// Execute async batch and return result
	result := asyncBatch.Run()

	// Extract results from async batch result
	if batchResult, ok := result.(tea.Msg); ok {
		return batchResult
	}

	// Fallback to sequential execution if async fails
	return b.runSequential()
}

// runParallelLimited executes commands with a parallel limit using async commands
func (b *BatchCommand) runParallelLimited() tea.Msg {
	// Use async batch command instead of goroutines
	asyncCommands := NewAsyncBatchCommands()
	asyncBatch := asyncCommands.factory.CreateAsyncLimitedParallelBatch(b.ID()+"-async", b.commands, b.parallelLimit, b.Model)

	// Configure async batch with same settings
	if abc, ok := asyncBatch.(*AsyncBatchCommand); ok {
		abc.WithFailFast(b.failFast)
		abc.WithTimeout(b.timeout)
	}

	// Execute async batch and return result
	result := asyncBatch.Run()

	// Extract results from async batch result
	if batchResult, ok := result.(tea.Msg); ok {
		return batchResult
	}

	// Fallback to sequential execution if async fails
	return b.runSequential()
}

// executeCommand executes a single command and tracks timing
func (b *BatchCommand) executeCommand(command Command) BatchCommandResult {
	startTime := time.Now()

	result := command.Run()
	duration := time.Since(startTime)

	batchResult := BatchCommandResult{
		CommandID:   command.ID(),
		CommandType: command.Type(),
		Duration:    duration,
		StartTime:   startTime,
		EndTime:     startTime.Add(duration),
	}

	// Extract result and error
	if resultMsg, ok := result.(ResultMessage); ok {
		batchResult.Result = resultMsg.GetResult()
		batchResult.Error = resultMsg.GetError()
	} else {
		// For non-standard messages, assume success
		batchResult.Result = result
		batchResult.Error = nil
	}

	return batchResult
}

// addResult safely adds a result to the results slice
func (b *BatchCommand) addResult(result BatchCommandResult) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.results = append(b.results, result)
}

// buildBatchResult builds the final batch result message
func (b *BatchCommand) buildBatchResult() BatchResultMsg {
	b.mu.RLock()
	defer b.mu.RUnlock()

	duration := time.Since(b.startTime)
	successCount := 0
	failureCount := 0

	for _, result := range b.results {
		if result.Error != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	var batchError error
	if failureCount > 0 && b.failFast {
		batchError = fmt.Errorf("batch operation failed fast after %d failures", failureCount)
	} else if failureCount > 0 && successCount == 0 {
		batchError = fmt.Errorf("all %d batch commands failed", failureCount)
	}

	return BatchResultMsg{
		ID:            b.ID(),
		Type:          b.Type(),
		Results:       b.results,
		Error:         batchError,
		Context:       b.Context(),
		Duration:      duration,
		ExecutionMode: b.executionMode,
		TotalCommands: len(b.commands),
		SuccessCount:  successCount,
		FailureCount:  failureCount,
	}
}

// GetResults returns a copy of the current results
func (b *BatchCommand) GetResults() []BatchCommandResult {
	b.mu.RLock()
	defer b.mu.RUnlock()

	results := make([]BatchCommandResult, len(b.results))
	copy(results, b.results)
	return results
}

// Commands returns the list of commands in this batch
func (b *BatchCommand) Commands() []Command {
	return b.commands
}

// ExecutionMode returns the execution mode
func (b *BatchCommand) ExecutionMode() BatchExecutionMode {
	return b.executionMode
}

// Factory functions for common batch patterns

// NewSequentialBatch creates a batch that executes commands sequentially
func NewSequentialBatch(id string, commands []Command, context OperationContext, model interface{}) *BatchCommand {
	return NewBatchCommand(id, commands, BatchSequential, context, model)
}

// NewParallelBatch creates a batch that executes commands in parallel
func NewParallelBatch(id string, commands []Command, context OperationContext, model interface{}) *BatchCommand {
	return NewBatchCommand(id, commands, BatchParallel, context, model)
}

// NewLimitedParallelBatch creates a batch that executes commands with parallel limit
func NewLimitedParallelBatch(id string, commands []Command, limit int, context OperationContext, model interface{}) *BatchCommand {
	batch := NewBatchCommand(id, commands, BatchParallelLimited, context, model)
	return batch.WithParallelLimit(limit)
}

// NewFailFastBatch creates a batch that stops on first error
func NewFailFastBatch(id string, commands []Command, mode BatchExecutionMode, context OperationContext, model interface{}) *BatchCommand {
	batch := NewBatchCommand(id, commands, mode, context, model)
	return batch.WithFailFast(true)
}

// Batch utilities

// BatchStats represents statistics for batch operations
type BatchStats struct {
	TotalCommands   int           `json:"totalCommands"`
	SuccessCount    int           `json:"successCount"`
	FailureCount    int           `json:"failureCount"`
	AverageDuration time.Duration `json:"averageDuration"`
	MinDuration     time.Duration `json:"minDuration"`
	MaxDuration     time.Duration `json:"maxDuration"`
	TotalDuration   time.Duration `json:"totalDuration"`
	SuccessRate     float64       `json:"successRate"`
}

// CalculateBatchStats calculates statistics from batch results
func CalculateBatchStats(results []BatchCommandResult, totalDuration time.Duration) BatchStats {
	if len(results) == 0 {
		return BatchStats{}
	}

	stats := BatchStats{
		TotalCommands: len(results),
		TotalDuration: totalDuration,
		MinDuration:   results[0].Duration,
		MaxDuration:   results[0].Duration,
	}

	var totalCommandDuration time.Duration

	for _, result := range results {
		totalCommandDuration += result.Duration

		if result.Error != nil {
			stats.FailureCount++
		} else {
			stats.SuccessCount++
		}

		if result.Duration < stats.MinDuration {
			stats.MinDuration = result.Duration
		}
		if result.Duration > stats.MaxDuration {
			stats.MaxDuration = result.Duration
		}
	}

	if len(results) > 0 {
		stats.AverageDuration = totalCommandDuration / time.Duration(len(results))
		stats.SuccessRate = float64(stats.SuccessCount) / float64(len(results))
	}

	return stats
}

// BatchCommandBuilder provides a fluent interface for building batch commands
type BatchCommandBuilder struct {
	id             string
	commands       []Command
	mode           BatchExecutionMode
	context        OperationContext
	failFast       bool
	timeout        time.Duration
	parallelLimit  int
	model          interface{}
}

// NewBatchCommandBuilder creates a new batch command builder
func NewBatchCommandBuilder(id string, context OperationContext, model interface{}) *BatchCommandBuilder {
	return &BatchCommandBuilder{
		id:       id,
		commands: make([]Command, 0),
		mode:     BatchSequential,
		context:  context,
		timeout:  30 * time.Second,
		model:    model,
	}
}

// AddCommand adds a command to the batch
func (b *BatchCommandBuilder) AddCommand(command Command) *BatchCommandBuilder {
	b.commands = append(b.commands, command)
	return b
}

// AddCommands adds multiple commands to the batch
func (b *BatchCommandBuilder) AddCommands(commands ...Command) *BatchCommandBuilder {
	b.commands = append(b.commands, commands...)
	return b
}

// Sequential sets execution mode to sequential
func (b *BatchCommandBuilder) Sequential() *BatchCommandBuilder {
	b.mode = BatchSequential
	return b
}

// Parallel sets execution mode to parallel
func (b *BatchCommandBuilder) Parallel() *BatchCommandBuilder {
	b.mode = BatchParallel
	return b
}

// ParallelLimited sets execution mode to parallel with limit
func (b *BatchCommandBuilder) ParallelLimited(limit int) *BatchCommandBuilder {
	b.mode = BatchParallelLimited
	b.parallelLimit = limit
	return b
}

// FailFast enables fail-fast behavior
func (b *BatchCommandBuilder) FailFast() *BatchCommandBuilder {
	b.failFast = true
	return b
}

// Timeout sets the timeout for the batch
func (b *BatchCommandBuilder) Timeout(timeout time.Duration) *BatchCommandBuilder {
	b.timeout = timeout
	return b
}

// Build creates the batch command
func (b *BatchCommandBuilder) Build() *BatchCommand {
	batch := NewBatchCommand(b.id, b.commands, b.mode, b.context, b.model)
	if b.failFast {
		batch.WithFailFast(true)
	}
	if b.timeout > 0 {
		batch.WithTimeout(b.timeout)
	}
	if b.mode == BatchParallelLimited && b.parallelLimit > 0 {
		batch.WithParallelLimit(b.parallelLimit)
	}
	return batch
}

// Message interface implementation

func (m BatchResultMsg) Message() tea.Msg {
	return m
}

// IsSuccess returns true if the batch operation completed successfully
func (m BatchResultMsg) IsSuccess() bool {
	return m.Error == nil
}

// IsFailure returns true if the batch operation failed
func (m BatchResultMsg) IsFailure() bool {
	return m.Error != nil
}

// GetResult returns the results of the batch operation
func (m BatchResultMsg) GetResult() interface{} {
	return m.Results
}

// GetError returns the error of the batch operation
func (m BatchResultMsg) GetError() error {
	return m.Error
}

// String returns a string representation of the batch result
func (m BatchResultMsg) String() string {
	if m.IsSuccess() {
		return fmt.Sprintf("BatchResult{id=%s, mode=%v, commands=%d, success=%d, duration=%v}",
			m.ID, m.ExecutionMode, m.TotalCommands, m.SuccessCount, m.Duration)
	}
	return fmt.Sprintf("BatchResult{id=%s, mode=%v, commands=%d, success=%d, failures=%d, error=%v, duration=%v}",
		m.ID, m.ExecutionMode, m.TotalCommands, m.SuccessCount, m.FailureCount, m.Error, m.Duration)
}

// Advanced batch patterns

// NewConditionalBatch creates a batch where command execution depends on previous results
func NewConditionalBatch(id string, commands []Command, condition func([]BatchCommandResult, int) bool, context OperationContext, model interface{}) *BatchCommand {
	// This is a placeholder for future conditional batch execution
	// In a full implementation, this would allow dynamic command execution based on results
	return NewBatchCommand(id, commands, BatchSequential, context, model)
}

// NewRetryBatch creates a batch that automatically retries failed commands
func NewRetryBatch(id string, commands []Command, maxRetries int, context OperationContext, model interface{}) *BatchCommand {
	// Wrap commands with retry logic
	retryCommands := make([]Command, len(commands))
	for i, cmd := range commands {
		retryCommands[i] = NewRetryableIOCommand(cmd) // Default retry policy
	}

	return NewBatchCommand(id, retryCommands, BatchSequential, context, model).WithFailFast(false)
}

// NewTimedBatch creates a batch with a hard timeout
func NewTimedBatch(id string, commands []Command, timeout time.Duration, mode BatchExecutionMode, context OperationContext, model interface{}) *BatchCommand {
	batch := NewBatchCommand(id, commands, mode, context, model)
	batch.WithTimeout(timeout)
	return batch
}