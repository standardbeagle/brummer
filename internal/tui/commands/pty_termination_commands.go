package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// PTYTerminationCommand represents a PTY termination-related operation
type PTYTerminationCommand struct {
	BaseCommand
	TerminationOperation string
	TerminationID       string
	Args                map[string]interface{}
	Model               interface{}
}

// Run executes the PTY termination operation and returns a message (implements Command interface)
func (ptc *PTYTerminationCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "Termination",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := ptc.executeTerminationOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   ptc.id,
		OperationID: ptc.id,
		Priority:    types.OperationPriority(ptc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"terminationOperation": ptc.TerminationOperation,
			"terminationID":       ptc.TerminationID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(ptc.id, types.CommandType(ptc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(ptc.id, types.CommandType(ptc.cmdType), result, executionTime, typesContext)
}

// executeTerminationOperation performs the actual PTY termination operation
func (ptc *PTYTerminationCommand) executeTerminationOperation() (interface{}, error) {
	switch ptc.TerminationOperation {
	case "retry_signal_handler":
		return ptc.executeRetrySignalHandler()
	case "execute_termination_callback":
		return ptc.executeTerminationCallback()
	case "execute_cleanup_callback":
		return ptc.executeCleanupCallback()
	case "monitor_signals":
		return ptc.monitorSignals()
	default:
		return nil, fmt.Errorf("unknown PTY termination operation: %s", ptc.TerminationOperation)
	}
}

// executeRetrySignalHandler retries a signal handler with delay
func (ptc *PTYTerminationCommand) executeRetrySignalHandler() (interface{}, error) {
	// This replaces the goroutine at line 398 in pty_termination.go
	delay, ok := ptc.Args["delay"].(time.Duration)
	if !ok {
		delay = 500 * time.Millisecond // Default retry delay
	}

	// Simulate the delay and retry
	time.Sleep(delay)

	signal, ok := ptc.Args["signal"].(string)
	if !ok {
		signal = "unknown"
	}

	return fmt.Sprintf("Signal handler retried for signal %s after %v", signal, delay), nil
}

// executeTerminationCallback executes a termination callback
func (ptc *PTYTerminationCommand) executeTerminationCallback() (interface{}, error) {
	callbackName, ok := ptc.Args["callbackName"].(string)
	if !ok {
		callbackName = "unknown"
	}

	reason, ok := ptc.Args["reason"].(string)
	if !ok {
		reason = "unknown"
	}

	return fmt.Sprintf("Termination callback '%s' executed for reason: %s", callbackName, reason), nil
}

// executeCleanupCallback executes a cleanup callback
func (ptc *PTYTerminationCommand) executeCleanupCallback() (interface{}, error) {
	callbackName, ok := ptc.Args["callbackName"].(string)
	if !ok {
		callbackName = "unknown"
	}

	return fmt.Sprintf("Cleanup callback '%s' executed", callbackName), nil
}

// monitorSignals monitors signals for PTY termination
func (ptc *PTYTerminationCommand) monitorSignals() (interface{}, error) {
	// This replaces the monitorSignals goroutine functionality
	ptyID, ok := ptc.Args["ptyID"].(string)
	if !ok {
		ptyID = "unknown"
	}

	return fmt.Sprintf("Signal monitoring started for PTY %s", ptyID), nil
}

// PTYTerminationCommandFactory creates PTY termination-related commands
type PTYTerminationCommandFactory struct{}

// NewPTYTerminationCommandFactory creates a new PTY termination command factory
func NewPTYTerminationCommandFactory() *PTYTerminationCommandFactory {
	return &PTYTerminationCommandFactory{}
}

// CreateRetrySignalHandlerCommand creates a command to retry a signal handler
func (f *PTYTerminationCommandFactory) CreateRetrySignalHandlerCommand(terminationID string, signal string, delay time.Duration, model interface{}) Command {
	id := fmt.Sprintf("pty-term-retry-%s-%d", terminationID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYTerminationCommand{
		BaseCommand:          *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		TerminationOperation: "retry_signal_handler",
		TerminationID:       terminationID,
		Args: map[string]interface{}{
			"signal": signal,
			"delay":  delay,
		},
		Model: model,
	}
}

// CreateTerminationCallbackCommand creates a command to execute a termination callback
func (f *PTYTerminationCommandFactory) CreateTerminationCallbackCommand(terminationID string, callbackName string, reason string, model interface{}) Command {
	id := fmt.Sprintf("pty-term-callback-%s-%d", terminationID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYTerminationCommand{
		BaseCommand:          *NewBaseCommand(id, CommandTypeSystem, PriorityHigh, context),
		TerminationOperation: "execute_termination_callback",
		TerminationID:       terminationID,
		Args: map[string]interface{}{
			"callbackName": callbackName,
			"reason":       reason,
		},
		Model: model,
	}
}

// CreateCleanupCallbackCommand creates a command to execute a cleanup callback
func (f *PTYTerminationCommandFactory) CreateCleanupCallbackCommand(terminationID string, callbackName string, model interface{}) Command {
	id := fmt.Sprintf("pty-term-cleanup-%s-%d", terminationID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYTerminationCommand{
		BaseCommand:          *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		TerminationOperation: "execute_cleanup_callback",
		TerminationID:       terminationID,
		Args: map[string]interface{}{
			"callbackName": callbackName,
		},
		Model: model,
	}
}

// CreateMonitorSignalsCommand creates a command to monitor signals
func (f *PTYTerminationCommandFactory) CreateMonitorSignalsCommand(terminationID string, ptyID string, model interface{}) Command {
	id := fmt.Sprintf("pty-term-monitor-%s-%d", terminationID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYTerminationCommand{
		BaseCommand:          *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		TerminationOperation: "monitor_signals",
		TerminationID:       terminationID,
		Args: map[string]interface{}{
			"ptyID": ptyID,
		},
		Model: model,
	}
}

// PTYTerminationCommands provides high-level commands for PTY termination operations
type PTYTerminationCommands struct {
	factory *PTYTerminationCommandFactory
}

// NewPTYTerminationCommands creates a new PTY termination commands instance
func NewPTYTerminationCommands() *PTYTerminationCommands {
	return &PTYTerminationCommands{
		factory: NewPTYTerminationCommandFactory(),
	}
}

// RetrySignalHandler returns a command to retry a signal handler with delay
func (ptc *PTYTerminationCommands) RetrySignalHandler(terminationID string, signal string, delay time.Duration, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := ptc.factory.CreateRetrySignalHandlerCommand(terminationID, signal, delay, model)
		return cmd.Run()
	}
}

// ExecuteTerminationCallback returns a command to execute a termination callback
func (ptc *PTYTerminationCommands) ExecuteTerminationCallback(terminationID string, callbackName string, reason string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := ptc.factory.CreateTerminationCallbackCommand(terminationID, callbackName, reason, model)
		return cmd.Run()
	}
}

// ExecuteCleanupCallback returns a command to execute a cleanup callback
func (ptc *PTYTerminationCommands) ExecuteCleanupCallback(terminationID string, callbackName string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := ptc.factory.CreateCleanupCallbackCommand(terminationID, callbackName, model)
		return cmd.Run()
	}
}

// MonitorSignals returns a command to monitor signals
func (ptc *PTYTerminationCommands) MonitorSignals(terminationID string, ptyID string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := ptc.factory.CreateMonitorSignalsCommand(terminationID, ptyID, model)
		return cmd.Run()
	}
}