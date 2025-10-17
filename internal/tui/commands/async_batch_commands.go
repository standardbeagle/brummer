package commands

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/types"
)

// AsyncBatchCommand represents a batch command that executes asynchronously using Bubble Tea commands
type AsyncBatchCommand struct {
	BaseCommand
	commands       []Command
	executionMode  BatchExecutionMode
	results        []BatchCommandResult
	mu             sync.RWMutex
	startTime      time.Time
	timeout        time.Duration
	failFast       bool
	parallelLimit  int
	// Channels for async coordination
	resultChan     chan BatchCommandResult
	doneChan       chan struct{}
	Model          interface{}
}

// NewAsyncBatchCommand creates a new async batch command
func NewAsyncBatchCommand(id string, commands []Command, mode BatchExecutionMode, context OperationContext, model interface{}) *AsyncBatchCommand {
	return &AsyncBatchCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		commands:       commands,
		executionMode:  mode,
		results:        make([]BatchCommandResult, 0, len(commands)),
		timeout:        context.Timeout,
		failFast:       false,
		parallelLimit:  10,
		resultChan:     make(chan BatchCommandResult, len(commands)),
		doneChan:       make(chan struct{}),
		Model:          model,
	}
}

// Run executes the async batch command and returns a message (implements Command interface)
func (abc *AsyncBatchCommand) Run() tea.Msg {
	abc.startTime = time.Now()

	// Start async execution based on mode
	switch abc.executionMode {
	case BatchSequential:
		return abc.runSequentialAsync()
	case BatchParallel:
		return abc.runParallelAsync()
	case BatchParallelLimited:
		return abc.runParallelLimitedAsync()
	default:
		return abc.buildErrorResult(fmt.Errorf("unknown async batch execution mode: %d", abc.executionMode))
	}
}

// runSequentialAsync executes commands sequentially and returns a command for next step
func (abc *AsyncBatchCommand) runSequentialAsync() tea.Msg {
	if len(abc.commands) == 0 {
		return abc.buildFinalResult()
	}

	// Execute first command and return command for remaining
	result := abc.executeCommand(abc.commands[0])
	abc.addResult(result)

	// Check if we should stop due to failFast
	if abc.failFast && result.Error != nil {
		return abc.buildFinalResult()
	}

	// Return command to process remaining commands
	return abc.createSequentialCommand(1)
}

// runParallelAsync executes all commands in parallel using Bubble Tea commands
func (abc *AsyncBatchCommand) runParallelAsync() tea.Msg {
	// Start command that will coordinate parallel execution
	return abc.createParallelCommand()
}

// runParallelLimitedAsync executes commands with parallel limit using Bubble Tea commands
func (abc *AsyncBatchCommand) runParallelLimitedAsync() tea.Msg {
	// Start command that will coordinate limited parallel execution
	return abc.createLimitedParallelCommand()
}

// createSequentialCommand creates a command for processing remaining sequential commands
func (abc *AsyncBatchCommand) createSequentialCommand(index int) tea.Msg {
	if index >= len(abc.commands) {
		return abc.buildFinalResult()
	}

	return func() tea.Msg {
		result := abc.executeCommand(abc.commands[index])
		abc.addResult(result)

		// Check if we should stop due to failFast
		if abc.failFast && result.Error != nil {
			return abc.buildFinalResult()
		}

		// Continue with next command
		return abc.createSequentialCommand(index + 1)
	}
}

// createParallelCommand creates a command that coordinates parallel execution
func (abc *AsyncBatchCommand) createParallelCommand() tea.Msg {
	remainingCommands := len(abc.commands)
	completedCount := 0

	return func() tea.Msg {
		// Execute commands sequentially using Bubble Tea command pattern
		return abc.createSequentialParallelExecution(0, remainingCommands, completedCount)
	}
}

// createLimitedParallelCommand creates a command that coordinates limited parallel execution
func (abc *AsyncBatchCommand) createLimitedParallelCommand() tea.Msg {
	remainingCommands := len(abc.commands)
	completedCount := 0

	return func() tea.Msg {
		// Execute commands sequentially using Bubble Tea command pattern with limit
		return abc.createSequentialLimitedExecution(0, remainingCommands, completedCount, abc.parallelLimit)
	}
}

// createSequentialParallelExecution executes commands sequentially to simulate parallel execution
func (abc *AsyncBatchCommand) createSequentialParallelExecution(index int, totalCommands int, completedCount int) tea.Msg {
	if index >= totalCommands {
		return abc.buildFinalResult()
	}

	// Execute current command
	result := abc.executeCommand(abc.commands[index])
	abc.addResult(result)
	completedCount++

	// Check failFast condition
	if abc.failFast && result.Error != nil {
		return abc.buildFinalResult()
	}

	// Continue with next command
	return abc.createSequentialParallelExecution(index+1, totalCommands, completedCount)
}

// createSequentialLimitedExecution executes commands sequentially with limit tracking
func (abc *AsyncBatchCommand) createSequentialLimitedExecution(index int, totalCommands int, completedCount int, limit int) tea.Msg {
	if index >= totalCommands {
		return abc.buildFinalResult()
	}

	// Execute current command
	result := abc.executeCommand(abc.commands[index])
	abc.addResult(result)
	completedCount++

	// Check failFast condition
	if abc.failFast && result.Error != nil {
		return abc.buildFinalResult()
	}

	// Continue with next command
	return abc.createSequentialLimitedExecution(index+1, totalCommands, completedCount, limit)
}

// createWaitForParallelCompletion creates a command that waits for parallel execution to complete
func (abc *AsyncBatchCommand) createWaitForParallelCompletion() tea.Msg {
	return func() tea.Msg {
		select {
		case <-abc.doneChan:
			return abc.buildFinalResult()
		case <-time.After(abc.timeout):
			return abc.buildErrorResult(fmt.Errorf("async batch operation timed out after %v", abc.timeout))
		}
	}
}

// executeCommand executes a single command and tracks timing
func (abc *AsyncBatchCommand) executeCommand(command Command) BatchCommandResult {
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
func (abc *AsyncBatchCommand) addResult(result BatchCommandResult) {
	abc.mu.Lock()
	defer abc.mu.Unlock()
	abc.results = append(abc.results, result)
}

// buildFinalResult builds the final batch result message
func (abc *AsyncBatchCommand) buildFinalResult() tea.Msg {
	abc.mu.RLock()
	defer abc.mu.RUnlock()

	duration := time.Since(abc.startTime)
	successCount := 0
	failureCount := 0

	for _, result := range abc.results {
		if result.Error != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	var batchError error
	if failureCount > 0 && abc.failFast {
		batchError = fmt.Errorf("async batch operation failed fast after %d failures", failureCount)
	} else if failureCount > 0 && successCount == 0 {
		batchError = fmt.Errorf("all %d async batch commands failed", failureCount)
	}

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   abc.id,
		OperationID: abc.id,
		Priority:    types.OperationPriority(abc.priority),
		Timeout:     abc.timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"executionMode": abc.executionMode,
			"totalCommands": len(abc.commands),
			"failFast":      abc.failFast,
		},
	}

	batchResult := AsyncBatchResultMsg{
		ID:            abc.ID(),
		Type:          abc.Type(),
		Results:       abc.results,
		Error:         batchError,
		Context:       typesContext,
		Duration:      duration,
		ExecutionMode: abc.executionMode,
		TotalCommands: len(abc.commands),
		SuccessCount:  successCount,
		FailureCount:  failureCount,
	}

	return messages.NewCommandCompletedMessage(abc.id, types.CommandType(abc.cmdType), batchResult, duration, typesContext)
}

// buildErrorResult builds an error result message
func (abc *AsyncBatchCommand) buildErrorResult(err error) tea.Msg {
	duration := time.Since(abc.startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   abc.id,
		OperationID: abc.id,
		Priority:    types.OperationPriority(abc.priority),
		Timeout:     abc.timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"executionMode": abc.executionMode,
			"totalCommands": len(abc.commands),
			"error":         err.Error(),
		},
	}

	return messages.NewCommandFailedMessage(abc.id, types.CommandType(abc.cmdType), err, duration, typesContext, true)
}

// WithFailFast sets the fail-fast behavior
func (abc *AsyncBatchCommand) WithFailFast(failFast bool) *AsyncBatchCommand {
	abc.failFast = failFast
	return abc
}

// WithTimeout sets a timeout for the async batch operation
func (abc *AsyncBatchCommand) WithTimeout(timeout time.Duration) *AsyncBatchCommand {
	abc.timeout = timeout
	return abc
}

// WithParallelLimit sets the maximum number of parallel commands
func (abc *AsyncBatchCommand) WithParallelLimit(limit int) *AsyncBatchCommand {
	abc.parallelLimit = limit
	return abc
}

// AsyncBatchResultMsg represents the result of an async batch operation
type AsyncBatchResultMsg struct {
	ID            string                `json:"id"`
	Type          CommandType          `json:"type"`
	Results       []BatchCommandResult `json:"results,omitempty"`
	Error         error                `json:"error,omitempty"`
	Context       types.OperationContext `json:"context"`
	Duration      time.Duration        `json:"duration"`
	ExecutionMode BatchExecutionMode   `json:"executionMode"`
	TotalCommands int                  `json:"totalCommands"`
	SuccessCount  int                  `json:"successCount"`
	FailureCount  int                  `json:"failureCount"`
}

// AsyncBatchCommandFactory creates async batch-related commands
type AsyncBatchCommandFactory struct{}

// NewAsyncBatchCommandFactory creates a new async batch command factory
func NewAsyncBatchCommandFactory() *AsyncBatchCommandFactory {
	return &AsyncBatchCommandFactory{}
}

// CreateAsyncBatchCommand creates an async batch command
func (f *AsyncBatchCommandFactory) CreateAsyncBatchCommand(id string, commands []Command, mode BatchExecutionMode, model interface{}) Command {
	context := SystemOperationContext(0) // Version will be set by model
	return NewAsyncBatchCommand(id, commands, mode, context, model)
}

// CreateAsyncSequentialBatch creates an async sequential batch command
func (f *AsyncBatchCommandFactory) CreateAsyncSequentialBatch(id string, commands []Command, model interface{}) Command {
	return f.CreateAsyncBatchCommand(id, commands, BatchSequential, model)
}

// CreateAsyncParallelBatch creates an async parallel batch command
func (f *AsyncBatchCommandFactory) CreateAsyncParallelBatch(id string, commands []Command, model interface{}) Command {
	return f.CreateAsyncBatchCommand(id, commands, BatchParallel, model)
}

// CreateAsyncLimitedParallelBatch creates an async limited parallel batch command
func (f *AsyncBatchCommandFactory) CreateAsyncLimitedParallelBatch(id string, commands []Command, limit int, model interface{}) Command {
	abc := NewAsyncBatchCommand(id, commands, BatchParallelLimited, SystemOperationContext(0), model)
	return abc.WithParallelLimit(limit)
}

// AsyncBatchCommands provides high-level commands for async batch operations
type AsyncBatchCommands struct {
	factory *AsyncBatchCommandFactory
}

// NewAsyncBatchCommands creates a new async batch commands instance
func NewAsyncBatchCommands() *AsyncBatchCommands {
	return &AsyncBatchCommands{
		factory: NewAsyncBatchCommandFactory(),
	}
}

// ExecuteAsyncBatch returns a command to execute an async batch operation
func (abc *AsyncBatchCommands) ExecuteAsyncBatch(id string, commands []Command, mode BatchExecutionMode, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := abc.factory.CreateAsyncBatchCommand(id, commands, mode, model)
		return cmd.Run()
	}
}

// ExecuteAsyncSequentialBatch returns a command to execute an async sequential batch
func (abc *AsyncBatchCommands) ExecuteAsyncSequentialBatch(id string, commands []Command, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := abc.factory.CreateAsyncSequentialBatch(id, commands, model)
		return cmd.Run()
	}
}

// ExecuteAsyncParallelBatch returns a command to execute an async parallel batch
func (abc *AsyncBatchCommands) ExecuteAsyncParallelBatch(id string, commands []Command, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := abc.factory.CreateAsyncParallelBatch(id, commands, model)
		return cmd.Run()
	}
}

// ExecuteAsyncLimitedParallelBatch returns a command to execute an async limited parallel batch
func (abc *AsyncBatchCommands) ExecuteAsyncLimitedParallelBatch(id string, commands []Command, limit int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := abc.factory.CreateAsyncLimitedParallelBatch(id, commands, limit, model)
		return cmd.Run()
	}
}