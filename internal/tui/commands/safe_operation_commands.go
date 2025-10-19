package commands

import (
	"fmt"
	"runtime/debug"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// SafeOperationCommand represents a safe operation execution
type SafeOperationCommand struct {
	BaseCommand
	SafeOperation string
	OperationID    string
	Args           map[string]interface{}
	Model          interface{}
}

// Run executes the safe operation and returns a message (implements Command interface)
func (soc *SafeOperationCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "SafeOperation",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := soc.executeSafeOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   soc.id,
		OperationID: soc.id,
		Priority:    types.OperationPriority(soc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"safeOperation": soc.SafeOperation,
			"operationID":    soc.OperationID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(soc.id, types.CommandType(soc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(soc.id, types.CommandType(soc.cmdType), result, executionTime, typesContext)
}

// executeSafeOperation performs the actual safe operation
func (soc *SafeOperationCommand) executeSafeOperation() (interface{}, error) {
	switch soc.SafeOperation {
	case "execute_with_panic_recovery":
		return soc.executeWithPanicRecovery()
	case "execute_no_error_with_panic_recovery":
		return soc.executeNoErrorWithPanicRecovery()
	case "safe_async_operation":
		return soc.safeAsyncOperation()
	case "panic_recovery_test":
		return soc.panicRecoveryTest()
	case "error_handling_test":
		return soc.errorHandlingTest()
	default:
		return nil, fmt.Errorf("unknown safe operation: %s", soc.SafeOperation)
	}
}

// executeWithPanicRecovery executes a function with panic recovery
func (soc *SafeOperationCommand) executeWithPanicRecovery() (interface{}, error) {
	// This replaces SafeGoroutine functionality
	operationName, ok := soc.Args["operationName"].(string)
	if !ok {
		operationName = soc.OperationID
	}

	// Simulate the operation execution
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			// Log the panic recovery
			fmt.Printf("Panic recovered in operation %s: %v\nStack trace:\n%s", operationName, r, string(stack))
		}
	}()

	// Check if this operation should succeed or fail
	shouldFail, ok := soc.Args["shouldFail"].(bool)
	if ok && shouldFail {
		return nil, fmt.Errorf("operation %s failed as expected", operationName)
	}

	return fmt.Sprintf("Operation %s executed successfully with panic recovery", operationName), nil
}

// executeNoErrorWithPanicRecovery executes a function without error return but with panic recovery
func (soc *SafeOperationCommand) executeNoErrorWithPanicRecovery() (interface{}, error) {
	// This replaces SafeGoroutineNoError functionality
	operationName, ok := soc.Args["operationName"].(string)
	if !ok {
		operationName = soc.OperationID
	}

	// Simulate the operation execution
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			// Log the panic recovery
			fmt.Printf("Panic recovered in operation %s: %v\nStack trace:\n%s", operationName, r, string(stack))
		}
	}()

	// Check if this operation should panic
	shouldPanic, ok := soc.Args["shouldPanic"].(bool)
	if ok && shouldPanic {
		panic(fmt.Sprintf("simulated panic in operation %s", operationName))
	}

	return fmt.Sprintf("Operation %s executed successfully with panic recovery (no error return)", operationName), nil
}

// safeAsyncOperation performs a safe async operation
func (soc *SafeOperationCommand) safeAsyncOperation() (interface{}, error) {
	operationName, ok := soc.Args["operationName"].(string)
	if !ok {
		operationName = soc.OperationID
	}

	timeout, ok := soc.Args["timeout"].(time.Duration)
	if !ok {
		timeout = 5 * time.Second
	}

	// Simulate async operation with timeout
	startTime := time.Now()

	// Simulate work
	time.Sleep(min(time.Duration(100*time.Millisecond), timeout))

	elapsed := time.Since(startTime)
	return fmt.Sprintf("Safe async operation %s completed in %v", operationName, elapsed), nil
}

// panicRecoveryTest tests panic recovery functionality
func (soc *SafeOperationCommand) panicRecoveryTest() (interface{}, error) {
	testType, ok := soc.Args["testType"].(string)
	if !ok {
		testType = "recover"
	}

	switch testType {
	case "recover":
		// Test successful panic recovery
		defer func() {
			if r := recover(); r != nil {
				// Successfully recovered
				return
			}
		}()
		panic("test panic for recovery")
	case "no_panic":
		// Test normal operation
		return "No panic occurred - test passed", nil
	default:
		return nil, fmt.Errorf("unknown test type: %s", testType)
	}
}

// errorHandlingTest tests error handling functionality
func (soc *SafeOperationCommand) errorHandlingTest() (interface{}, error) {
	errorType, ok := soc.Args["errorType"].(string)
	if !ok {
		errorType = "none"
	}

	switch errorType {
	case "none":
		return "Error handling test passed - no errors", nil
	case "recoverable":
		return nil, fmt.Errorf("Recoverable error occurred: %s", errorType)
	case "critical":
		return nil, fmt.Errorf("Critical error occurred: %s", errorType)
	default:
		return nil, fmt.Errorf("Unknown error type: %s", errorType)
	}
}

// SafeOperationCommandFactory creates safe operation-related commands
type SafeOperationCommandFactory struct{}

// NewSafeOperationCommandFactory creates a new safe operation command factory
func NewSafeOperationCommandFactory() *SafeOperationCommandFactory {
	return &SafeOperationCommandFactory{}
}

// CreateExecuteWithPanicRecoveryCommand creates a command to execute with panic recovery
func (f *SafeOperationCommandFactory) CreateExecuteWithPanicRecoveryCommand(operationID, operationName string, shouldFail bool, model interface{}) Command {
	id := fmt.Sprintf("safe-panic-recovery-%s-%d", operationID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &SafeOperationCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		SafeOperation: "execute_with_panic_recovery",
		OperationID:   operationID,
		Args: map[string]interface{}{
			"operationName": operationName,
			"shouldFail":    shouldFail,
		},
		Model: model,
	}
}

// CreateExecuteNoErrorWithPanicRecoveryCommand creates a command to execute without error return but with panic recovery
func (f *SafeOperationCommandFactory) CreateExecuteNoErrorWithPanicRecoveryCommand(operationID, operationName string, shouldPanic bool, model interface{}) Command {
	id := fmt.Sprintf("safe-no-error-panic-%s-%d", operationID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &SafeOperationCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		SafeOperation: "execute_no_error_with_panic_recovery",
		OperationID:   operationID,
		Args: map[string]interface{}{
			"operationName": operationName,
			"shouldPanic":   shouldPanic,
		},
		Model: model,
	}
}

// CreateSafeAsyncOperationCommand creates a command for safe async operations
func (f *SafeOperationCommandFactory) CreateSafeAsyncOperationCommand(operationID, operationName string, timeout time.Duration, model interface{}) Command {
	id := fmt.Sprintf("safe-async-%s-%d", operationID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &SafeOperationCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		SafeOperation: "safe_async_operation",
		OperationID:   operationID,
		Args: map[string]interface{}{
			"operationName": operationName,
			"timeout":       timeout,
		},
		Model: model,
	}
}

// CreatePanicRecoveryTestCommand creates a command to test panic recovery
func (f *SafeOperationCommandFactory) CreatePanicRecoveryTestCommand(operationID, testType string, model interface{}) Command {
	id := fmt.Sprintf("safe-panic-test-%s-%d", operationID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &SafeOperationCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		SafeOperation: "panic_recovery_test",
		OperationID:   operationID,
		Args: map[string]interface{}{
			"testType": testType,
		},
		Model: model,
	}
}

// CreateErrorHandlingTestCommand creates a command to test error handling
func (f *SafeOperationCommandFactory) CreateErrorHandlingTestCommand(operationID, errorType string, model interface{}) Command {
	id := fmt.Sprintf("safe-error-test-%s-%d", operationID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &SafeOperationCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		SafeOperation: "error_handling_test",
		OperationID:   operationID,
		Args: map[string]interface{}{
			"errorType": errorType,
		},
		Model: model,
	}
}

// SafeOperationCommands provides high-level commands for safe operations
// NOTE: All operations include panic recovery - this is not optional
type SafeOperationCommands struct {
	factory *SafeOperationCommandFactory
}

// NewSafeOperationCommands creates a new safe operation commands instance
func NewSafeOperationCommands() *SafeOperationCommands {
	return &SafeOperationCommands{
		factory: NewSafeOperationCommandFactory(),
	}
}

// ExecuteOperation returns a command to execute an operation with automatic panic recovery
// The operation function is required and will be wrapped with panic recovery automatically
// All operations are executed safely - panic recovery is always enabled
func (soc *SafeOperationCommands) ExecuteOperation(operationID, operationName string, operation func() error) tea.Cmd {
	if operation == nil {
		panic("ExecuteOperation requires a non-nil operation function")
	}

	return func() tea.Msg {
		// Execute with panic recovery
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				fmt.Printf("Panic recovered in operation %s: %v\nStack trace:\n%s", operationName, r, string(stack))
			}
		}()

		err := operation()
		if err != nil {
			return messages.NewCommandFailedMessage(
				operationID,
				types.CommandTypeSystem,
				err,
				0,
				types.OperationContext{
					RequestID:   operationID,
					OperationID: operationID,
					Priority:    types.PriorityNormal,
					Metadata: map[string]interface{}{
						"operationName": operationName,
					},
				},
				true,
			)
		}

		return messages.NewCommandCompletedMessage(
			operationID,
			types.CommandTypeSystem,
			fmt.Sprintf("Operation %s completed successfully", operationName),
			0,
			types.OperationContext{
				RequestID:   operationID,
				OperationID: operationID,
				Priority:    types.PriorityNormal,
				Metadata: map[string]interface{}{
					"operationName": operationName,
				},
			},
		)
	}
}

// ExecuteAsyncOperation returns a command for async operations with timeout and panic recovery
// The operation function is required and will be wrapped with both timeout and panic recovery
func (soc *SafeOperationCommands) ExecuteAsyncOperation(operationID, operationName string, timeout time.Duration, operation func() error) tea.Cmd {
	if operation == nil {
		panic("ExecuteAsyncOperation requires a non-nil operation function")
	}
	if timeout <= 0 {
		panic("ExecuteAsyncOperation requires a positive timeout duration")
	}

	return func() tea.Msg {
		// Execute with panic recovery and timeout
		done := make(chan error, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					fmt.Printf("Panic recovered in async operation %s: %v\nStack trace:\n%s", operationName, r, string(stack))
					done <- fmt.Errorf("panic recovered: %v", r)
				}
			}()
			done <- operation()
		}()

		select {
		case err := <-done:
			if err != nil {
				return messages.NewCommandFailedMessage(
					operationID,
					types.CommandTypeSystem,
					err,
					0,
					types.OperationContext{
						RequestID:   operationID,
						OperationID: operationID,
						Priority:    types.PriorityNormal,
						Timeout:     timeout,
						Metadata: map[string]interface{}{
							"operationName": operationName,
						},
					},
					true,
				)
			}

			return messages.NewCommandCompletedMessage(
				operationID,
				types.CommandTypeSystem,
				fmt.Sprintf("Async operation %s completed successfully", operationName),
				0,
				types.OperationContext{
					RequestID:   operationID,
					OperationID: operationID,
					Priority:    types.PriorityNormal,
					Timeout:     timeout,
					Metadata: map[string]interface{}{
						"operationName": operationName,
					},
				},
			)
		case <-time.After(timeout):
			return messages.NewCommandFailedMessage(
				operationID,
				types.CommandTypeSystem,
				fmt.Errorf("operation timed out after %v", timeout),
				timeout,
				types.OperationContext{
					RequestID:   operationID,
					OperationID: operationID,
					Priority:    types.PriorityNormal,
					Timeout:     timeout,
					Metadata: map[string]interface{}{
						"operationName": operationName,
					},
				},
				true,
			)
		}
	}
}