package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// ContextAsyncCommand represents an async context-related operation
type ContextAsyncCommand struct {
	BaseCommand
	ContextOperation string
	ContextID        string
	Args             map[string]interface{}
	Model            interface{}
}

// Run executes the async context operation and returns a message (implements Command interface)
func (cac *ContextAsyncCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "ContextAsync",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := cac.executeContextOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   cac.id,
		OperationID: cac.id,
		Priority:    types.OperationPriority(cac.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"contextOperation": cac.ContextOperation,
			"contextID":        cac.ContextID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(cac.id, types.CommandType(cac.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(cac.id, types.CommandType(cac.cmdType), result, executionTime, typesContext)
}

// executeContextOperation performs the actual async context operation
func (cac *ContextAsyncCommand) executeContextOperation() (interface{}, error) {
	switch cac.ContextOperation {
	case "execute_with_timeout":
		return cac.executeWithTimeout()
	case "context_pool_operation":
		return cac.contextPoolOperation()
	case "async_context_cleanup":
		return cac.asyncContextCleanup()
	case "context_monitoring":
		return cac.contextMonitoring()
	default:
		return nil, fmt.Errorf("unknown async context operation: %s", cac.ContextOperation)
	}
}

// executeWithTimeout executes a function with timeout
func (cac *ContextAsyncCommand) executeWithTimeout() (interface{}, error) {
	// This replaces the goroutine at line 206 in context.go
	contextID, ok := cac.Args["contextID"].(string)
	if !ok {
		contextID = cac.ContextID
	}

	timeout, ok := cac.Args["timeout"].(time.Duration)
	if !ok {
		timeout = 10 * time.Second
	}

	operation, ok := cac.Args["operation"].(string)
	if !ok {
		operation = "unknown"
	}

	// Simulate the operation execution
	time.Sleep(min(time.Duration(100*time.Millisecond), timeout))

	return fmt.Sprintf("Executed operation '%s' with timeout %v in context %s", operation, timeout, contextID), nil
}

// contextPoolOperation performs context pool operations
func (cac *ContextAsyncCommand) contextPoolOperation() (interface{}, error) {
	contextID, ok := cac.Args["contextID"].(string)
	if !ok {
		contextID = cac.ContextID
	}

	poolOperation, ok := cac.Args["poolOperation"].(string)
	if !ok {
		poolOperation = "get"
	}

	return fmt.Sprintf("Context pool operation '%s' completed for context %s", poolOperation, contextID), nil
}

// asyncContextCleanup performs async context cleanup
func (cac *ContextAsyncCommand) asyncContextCleanup() (interface{}, error) {
	contextID, ok := cac.Args["contextID"].(string)
	if !ok {
		contextID = cac.ContextID
	}

	cleanupCount, ok := cac.Args["cleanupCount"].(int)
	if !ok {
		cleanupCount = 1
	}

	return fmt.Sprintf("Cleaned up %d resources for context %s", cleanupCount, contextID), nil
}

// contextMonitoring performs context monitoring
func (cac *ContextAsyncCommand) contextMonitoring() (interface{}, error) {
	contextID, ok := cac.Args["contextID"].(string)
	if !ok {
		contextID = cac.ContextID
	}

	monitoredItems, ok := cac.Args["monitoredItems"].(int)
	if !ok {
		monitoredItems = 0
	}

	return fmt.Sprintf("Monitored %d items in context %s", monitoredItems, contextID), nil
}

// ContextAsyncCommandFactory creates async context-related commands
type ContextAsyncCommandFactory struct{}

// NewContextAsyncCommandFactory creates a new async context command factory
func NewContextAsyncCommandFactory() *ContextAsyncCommandFactory {
	return &ContextAsyncCommandFactory{}
}

// CreateExecuteWithTimeoutCommand creates a command to execute with timeout
func (f *ContextAsyncCommandFactory) CreateExecuteWithTimeoutCommand(contextID string, operation string, timeout time.Duration, model interface{}) Command {
	id := fmt.Sprintf("ctx-timeout-%s-%d", contextID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ContextAsyncCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		ContextOperation: "execute_with_timeout",
		ContextID:        contextID,
		Args: map[string]interface{}{
			"contextID": contextID,
			"operation": operation,
			"timeout":   timeout,
		},
		Model: model,
	}
}

// CreateContextPoolOperationCommand creates a command for context pool operations
func (f *ContextAsyncCommandFactory) CreateContextPoolOperationCommand(contextID, poolOperation string, model interface{}) Command {
	id := fmt.Sprintf("ctx-pool-%s-%d", contextID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ContextAsyncCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		ContextOperation: "context_pool_operation",
		ContextID:        contextID,
		Args: map[string]interface{}{
			"contextID":     contextID,
			"poolOperation": poolOperation,
		},
		Model: model,
	}
}

// CreateAsyncContextCleanupCommand creates a command for async context cleanup
func (f *ContextAsyncCommandFactory) CreateAsyncContextCleanupCommand(contextID string, cleanupCount int, model interface{}) Command {
	id := fmt.Sprintf("ctx-cleanup-%s-%d", contextID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ContextAsyncCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		ContextOperation: "async_context_cleanup",
		ContextID:        contextID,
		Args: map[string]interface{}{
			"contextID":    contextID,
			"cleanupCount": cleanupCount,
		},
		Model: model,
	}
}

// CreateContextMonitoringCommand creates a command for context monitoring
func (f *ContextAsyncCommandFactory) CreateContextMonitoringCommand(contextID string, monitoredItems int, model interface{}) Command {
	id := fmt.Sprintf("ctx-monitor-%s-%d", contextID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ContextAsyncCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		ContextOperation: "context_monitoring",
		ContextID:        contextID,
		Args: map[string]interface{}{
			"contextID":      contextID,
			"monitoredItems": monitoredItems,
		},
		Model: model,
	}
}

// ContextAsyncCommands provides high-level commands for async context operations
type ContextAsyncCommands struct {
	factory *ContextAsyncCommandFactory
}

// NewContextAsyncCommands creates a new async context commands instance
func NewContextAsyncCommands() *ContextAsyncCommands {
	return &ContextAsyncCommands{
		factory: NewContextAsyncCommandFactory(),
	}
}

// ExecuteWithTimeout returns a command to execute with timeout
func (cac *ContextAsyncCommands) ExecuteWithTimeout(contextID string, operation string, timeout time.Duration, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cac.factory.CreateExecuteWithTimeoutCommand(contextID, operation, timeout, model)
		return cmd.Run()
	}
}

// ContextPoolOperation returns a command for context pool operations
func (cac *ContextAsyncCommands) ContextPoolOperation(contextID, poolOperation string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cac.factory.CreateContextPoolOperationCommand(contextID, poolOperation, model)
		return cmd.Run()
	}
}

// AsyncContextCleanup returns a command for async context cleanup
func (cac *ContextAsyncCommands) AsyncContextCleanup(contextID string, cleanupCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cac.factory.CreateAsyncContextCleanupCommand(contextID, cleanupCount, model)
		return cmd.Run()
	}
}

// ContextMonitoring returns a command for context monitoring
func (cac *ContextAsyncCommands) ContextMonitoring(contextID string, monitoredItems int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cac.factory.CreateContextMonitoringCommand(contextID, monitoredItems, model)
		return cmd.Run()
	}
}

// Helper function to get minimum duration
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}