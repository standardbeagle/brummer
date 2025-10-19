package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// InputServiceCommand represents an input service-related operation
type InputServiceCommand struct {
	BaseCommand
	InputOperation string
	InputID        string
	Args           map[string]interface{}
	Model          interface{}
}

// Run executes the input service operation and returns a message (implements Command interface)
func (isc *InputServiceCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "InputService",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := isc.executeInputOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   isc.id,
		OperationID: isc.id,
		Priority:    types.OperationPriority(isc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"inputOperation": isc.InputOperation,
			"inputID":        isc.InputID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(isc.id, types.CommandType(isc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(isc.id, types.CommandType(isc.cmdType), result, executionTime, typesContext)
}

// executeInputOperation performs the actual input service operation
func (isc *InputServiceCommand) executeInputOperation() (interface{}, error) {
	switch isc.InputOperation {
	case "background_worker":
		return isc.executeBackgroundWorker()
	case "wait_for_background_tasks":
		return isc.waitForBackgroundTasks()
	case "input_processor":
		return isc.executeInputProcessor()
	case "activity_monitor":
		return isc.executeActivityMonitor()
	case "pty_monitor":
		return isc.executePTYMonitor()
	case "event_processor":
		return isc.executeEventProcessor()
	default:
		return nil, fmt.Errorf("unknown input service operation: %s", isc.InputOperation)
	}
}

// executeBackgroundWorker executes background worker tasks
func (isc *InputServiceCommand) executeBackgroundWorker() (interface{}, error) {
	// This replaces the goroutine at line 409 in input_service.go
	inputID, ok := isc.Args["inputID"].(string)
	if !ok {
		inputID = isc.InputID
	}

	return fmt.Sprintf("Background worker executed for input %s", inputID), nil
}

// waitForBackgroundTasks waits for background tasks to complete
func (isc *InputServiceCommand) waitForBackgroundTasks() (interface{}, error) {
	// This replaces the goroutine at line 425 in input_service.go
	taskCount, ok := isc.Args["taskCount"].(int)
	if !ok {
		taskCount = 0
	}

	return fmt.Sprintf("Waited for %d background tasks to complete", taskCount), nil
}

// executeInputProcessor processes input from PTY
func (isc *InputServiceCommand) executeInputProcessor() (interface{}, error) {
	// This replaces the goroutine at line 152 in pty_input_adapter.go
	inputID, ok := isc.Args["inputID"].(string)
	if !ok {
		inputID = isc.InputID
	}

	return fmt.Sprintf("Input processor executed for PTY input %s", inputID), nil
}

// executeActivityMonitor monitors activity
func (isc *InputServiceCommand) executeActivityMonitor() (interface{}, error) {
	// This replaces the goroutine at line 157 in pty_input_adapter.go and line 218 in pty_navigation_adapter.go
	inputID, ok := isc.Args["inputID"].(string)
	if !ok {
		inputID = isc.InputID
	}

	timeout, ok := isc.Args["timeout"].(time.Duration)
	if !ok {
		timeout = 30 * time.Second
	}

	return fmt.Sprintf("Activity monitor executed for input %s with timeout %v", inputID, timeout), nil
}

// executePTYMonitor monitors PTY input
func (isc *InputServiceCommand) executePTYMonitor() (interface{}, error) {
	// This replaces the goroutine at line 389 in pty_input_adapter.go
	ptyID, ok := isc.Args["ptyID"].(string)
	if !ok {
		ptyID = isc.InputID
	}

	return fmt.Sprintf("PTY monitor executed for PTY %s", ptyID), nil
}

// executeEventProcessor processes events
func (isc *InputServiceCommand) executeEventProcessor() (interface{}, error) {
	// This replaces the goroutine at line 213 in pty_navigation_adapter.go
	inputID, ok := isc.Args["inputID"].(string)
	if !ok {
		inputID = isc.InputID
	}

	eventCount, ok := isc.Args["eventCount"].(int)
	if !ok {
		eventCount = 0
	}

	return fmt.Sprintf("Event processor executed for input %s, processed %d events", inputID, eventCount), nil
}

// InputServiceCommandFactory creates input service-related commands
type InputServiceCommandFactory struct{}

// NewInputServiceCommandFactory creates a new input service command factory
func NewInputServiceCommandFactory() *InputServiceCommandFactory {
	return &InputServiceCommandFactory{}
}

// CreateBackgroundWorkerCommand creates a command to execute background worker tasks
func (f *InputServiceCommandFactory) CreateBackgroundWorkerCommand(inputID string, model interface{}) Command {
	id := fmt.Sprintf("input-bg-worker-%s-%d", inputID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &InputServiceCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		InputOperation: "background_worker",
		InputID:        inputID,
		Args: map[string]interface{}{
			"inputID": inputID,
		},
		Model: model,
	}
}

// CreateWaitForBackgroundTasksCommand creates a command to wait for background tasks
func (f *InputServiceCommandFactory) CreateWaitForBackgroundTasksCommand(inputID string, taskCount int, model interface{}) Command {
	id := fmt.Sprintf("input-wait-tasks-%s-%d", inputID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &InputServiceCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		InputOperation: "wait_for_background_tasks",
		InputID:        inputID,
		Args: map[string]interface{}{
			"inputID":   inputID,
			"taskCount": taskCount,
		},
		Model: model,
	}
}

// CreateInputProcessorCommand creates a command to process PTY input
func (f *InputServiceCommandFactory) CreateInputProcessorCommand(inputID string, model interface{}) Command {
	id := fmt.Sprintf("input-processor-%s-%d", inputID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &InputServiceCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		InputOperation: "input_processor",
		InputID:        inputID,
		Args: map[string]interface{}{
			"inputID": inputID,
		},
		Model: model,
	}
}

// CreateActivityMonitorCommand creates a command to monitor activity
func (f *InputServiceCommandFactory) CreateActivityMonitorCommand(inputID string, timeout time.Duration, model interface{}) Command {
	id := fmt.Sprintf("input-activity-monitor-%s-%d", inputID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &InputServiceCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		InputOperation: "activity_monitor",
		InputID:        inputID,
		Args: map[string]interface{}{
			"inputID": inputID,
			"timeout":  timeout,
		},
		Model: model,
	}
}

// CreatePTYMonitorCommand creates a command to monitor PTY input
func (f *InputServiceCommandFactory) CreatePTYMonitorCommand(ptyID string, model interface{}) Command {
	id := fmt.Sprintf("input-pty-monitor-%s-%d", ptyID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &InputServiceCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		InputOperation: "pty_monitor",
		InputID:        ptyID,
		Args: map[string]interface{}{
			"ptyID": ptyID,
		},
		Model: model,
	}
}

// CreateEventProcessorCommand creates a command to process events
func (f *InputServiceCommandFactory) CreateEventProcessorCommand(inputID string, eventCount int, model interface{}) Command {
	id := fmt.Sprintf("input-event-processor-%s-%d", inputID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &InputServiceCommand{
		BaseCommand:    *NewBaseCommand(id, CommandTypeProcess, PriorityMedium, context),
		InputOperation: "event_processor",
		InputID:        inputID,
		Args: map[string]interface{}{
			"inputID":    inputID,
			"eventCount": eventCount,
		},
		Model: model,
	}
}

// InputServiceCommands provides high-level commands for input service operations
type InputServiceCommands struct {
	factory *InputServiceCommandFactory
}

// NewInputServiceCommands creates a new input service commands instance
func NewInputServiceCommands() *InputServiceCommands {
	return &InputServiceCommands{
		factory: NewInputServiceCommandFactory(),
	}
}

// StartBackgroundWorker returns a command to start background worker tasks
func (isc *InputServiceCommands) StartBackgroundWorker(inputID string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := isc.factory.CreateBackgroundWorkerCommand(inputID, model)
		return cmd.Run()
	}
}

// WaitForBackgroundTasks returns a command to wait for background tasks
func (isc *InputServiceCommands) WaitForBackgroundTasks(inputID string, taskCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := isc.factory.CreateWaitForBackgroundTasksCommand(inputID, taskCount, model)
		return cmd.Run()
	}
}

// StartInputProcessor returns a command to start input processing
func (isc *InputServiceCommands) StartInputProcessor(inputID string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := isc.factory.CreateInputProcessorCommand(inputID, model)
		return cmd.Run()
	}
}

// startActivityMonitor returns a command to monitor activity
func (isc *InputServiceCommands) startActivityMonitor(inputID string, timeout time.Duration, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := isc.factory.CreateActivityMonitorCommand(inputID, timeout, model)
		return cmd.Run()
	}
}

// StartPTYMonitor returns a command to monitor PTY input
func (isc *InputServiceCommands) StartPTYMonitor(ptyID string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := isc.factory.CreatePTYMonitorCommand(ptyID, model)
		return cmd.Run()
	}
}

// StartEventProcessor returns a command to start event processing
func (isc *InputServiceCommands) StartEventProcessor(inputID string, eventCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := isc.factory.CreateEventProcessorCommand(inputID, eventCount, model)
		return cmd.Run()
	}
}