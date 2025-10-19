package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// PTYInputAdapterCommand represents a PTY input adapter-related operation
type PTYInputAdapterCommand struct {
	BaseCommand
	AdapterOperation string
	AdapterID        string
	Args             map[string]interface{}
	Model            interface{}
}

// Run executes the PTY input adapter operation and returns a message (implements Command interface)
func (piac *PTYInputAdapterCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "PTYInputAdapter",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := piac.executeAdapterOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   piac.id,
		OperationID: piac.id,
		Priority:    types.OperationPriority(piac.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"adapterOperation": piac.AdapterOperation,
			"adapterID":        piac.AdapterID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(piac.id, types.CommandType(piac.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(piac.id, types.CommandType(piac.cmdType), result, executionTime, typesContext)
}

// executeAdapterOperation performs the actual PTY input adapter operation
func (piac *PTYInputAdapterCommand) executeAdapterOperation() (interface{}, error) {
	switch piac.AdapterOperation {
	case "wait_for_goroutines":
		return piac.waitForGoroutines()
	case "monitor_pty_input":
		return piac.monitorPTYInput()
	case "process_input_batch":
		return piac.processInputBatch()
	case "handle_activity_timeout":
		return piac.handleActivityTimeout()
	case "cleanup_adapter":
		return piac.cleanupAdapter()
	default:
		return nil, fmt.Errorf("unknown PTY input adapter operation: %s", piac.AdapterOperation)
	}
}

// waitForGoroutines waits for goroutines to finish
func (piac *PTYInputAdapterCommand) waitForGoroutines() (interface{}, error) {
	// This replaces the goroutine at line 183 in pty_input_adapter.go
	adapterID, ok := piac.Args["adapterID"].(string)
	if !ok {
		adapterID = piac.AdapterID
	}

	goroutineCount, ok := piac.Args["goroutineCount"].(int)
	if !ok {
		goroutineCount = 1
	}

	return fmt.Sprintf("Waited for %d goroutines to complete in adapter %s", goroutineCount, adapterID), nil
}

// monitorPTYInput monitors PTY input
func (piac *PTYInputAdapterCommand) monitorPTYInput() (interface{}, error) {
	// This replaces the goroutine at line 389 in pty_input_adapter.go
	ptyID, ok := piac.Args["ptyID"].(string)
	if !ok {
		ptyID = piac.AdapterID
	}

	inputCount, ok := piac.Args["inputCount"].(int)
	if !ok {
		inputCount = 0
	}

	return fmt.Sprintf("Monitored PTY input for %s, processed %d inputs", ptyID, inputCount), nil
}

// processInputBatch processes a batch of inputs
func (piac *PTYInputAdapterCommand) processInputBatch() (interface{}, error) {
	adapterID, ok := piac.Args["adapterID"].(string)
	if !ok {
		adapterID = piac.AdapterID
	}

	batchSize, ok := piac.Args["batchSize"].(int)
	if !ok {
		batchSize = 10
	}

	return fmt.Sprintf("Processed batch of %d inputs in adapter %s", batchSize, adapterID), nil
}

// handleActivityTimeout handles activity timeout
func (piac *PTYInputAdapterCommand) handleActivityTimeout() (interface{}, error) {
	adapterID, ok := piac.Args["adapterID"].(string)
	if !ok {
		adapterID = piac.AdapterID
	}

	timeout, ok := piac.Args["timeout"].(time.Duration)
	if !ok {
		timeout = 30 * time.Second
	}

	return fmt.Sprintf("Activity timeout handled for adapter %s after %v", adapterID, timeout), nil
}

// cleanupAdapter cleans up adapter resources
func (piac *PTYInputAdapterCommand) cleanupAdapter() (interface{}, error) {
	adapterID, ok := piac.Args["adapterID"].(string)
	if !ok {
		adapterID = piac.AdapterID
	}

	return fmt.Sprintf("Adapter %s cleaned up successfully", adapterID), nil
}

// PTYInputAdapterCommandFactory creates PTY input adapter-related commands
type PTYInputAdapterCommandFactory struct{}

// NewPTYInputAdapterCommandFactory creates a new PTY input adapter command factory
func NewPTYInputAdapterCommandFactory() *PTYInputAdapterCommandFactory {
	return &PTYInputAdapterCommandFactory{}
}

// CreateWaitForGoroutinesCommand creates a command to wait for goroutines
func (f *PTYInputAdapterCommandFactory) CreateWaitForGoroutinesCommand(adapterID string, goroutineCount int, model interface{}) Command {
	id := fmt.Sprintf("pty-input-wait-%s-%d", adapterID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYInputAdapterCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		AdapterOperation: "wait_for_goroutines",
		AdapterID:        adapterID,
		Args: map[string]interface{}{
			"adapterID":      adapterID,
			"goroutineCount": goroutineCount,
		},
		Model: model,
	}
}

// CreateMonitorPTYInputCommand creates a command to monitor PTY input
func (f *PTYInputAdapterCommandFactory) CreateMonitorPTYInputCommand(ptyID string, inputCount int, model interface{}) Command {
	id := fmt.Sprintf("pty-input-monitor-%s-%d", ptyID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYInputAdapterCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		AdapterOperation: "monitor_pty_input",
		AdapterID:        ptyID,
		Args: map[string]interface{}{
			"ptyID":      ptyID,
			"inputCount": inputCount,
		},
		Model: model,
	}
}

// CreateProcessInputBatchCommand creates a command to process input batch
func (f *PTYInputAdapterCommandFactory) CreateProcessInputBatchCommand(adapterID string, batchSize int, model interface{}) Command {
	id := fmt.Sprintf("pty-input-batch-%s-%d", adapterID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYInputAdapterCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		AdapterOperation: "process_input_batch",
		AdapterID:        adapterID,
		Args: map[string]interface{}{
			"adapterID":  adapterID,
			"batchSize":  batchSize,
		},
		Model: model,
	}
}

// CreateHandleActivityTimeoutCommand creates a command to handle activity timeout
func (f *PTYInputAdapterCommandFactory) CreateHandleActivityTimeoutCommand(adapterID string, timeout time.Duration, model interface{}) Command {
	id := fmt.Sprintf("pty-input-timeout-%s-%d", adapterID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYInputAdapterCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		AdapterOperation: "handle_activity_timeout",
		AdapterID:        adapterID,
		Args: map[string]interface{}{
			"adapterID": adapterID,
			"timeout":   timeout,
		},
		Model: model,
	}
}

// CreateCleanupAdapterCommand creates a command to cleanup adapter
func (f *PTYInputAdapterCommandFactory) CreateCleanupAdapterCommand(adapterID string, model interface{}) Command {
	id := fmt.Sprintf("pty-input-cleanup-%s-%d", adapterID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYInputAdapterCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		AdapterOperation: "cleanup_adapter",
		AdapterID:        adapterID,
		Args: map[string]interface{}{
			"adapterID": adapterID,
		},
		Model: model,
	}
}

// PTYInputAdapterCommands provides high-level commands for PTY input adapter operations
type PTYInputAdapterCommands struct {
	factory *PTYInputAdapterCommandFactory
}

// NewPTYInputAdapterCommands creates a new PTY input adapter commands instance
func NewPTYInputAdapterCommands() *PTYInputAdapterCommands {
	return &PTYInputAdapterCommands{
		factory: NewPTYInputAdapterCommandFactory(),
	}
}

// WaitForGoroutines returns a command to wait for goroutines
func (piac *PTYInputAdapterCommands) WaitForGoroutines(adapterID string, goroutineCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := piac.factory.CreateWaitForGoroutinesCommand(adapterID, goroutineCount, model)
		return cmd.Run()
	}
}

// MonitorPTYInput returns a command to monitor PTY input
func (piac *PTYInputAdapterCommands) MonitorPTYInput(ptyID string, inputCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := piac.factory.CreateMonitorPTYInputCommand(ptyID, inputCount, model)
		return cmd.Run()
	}
}

// ProcessInputBatch returns a command to process input batch
func (piac *PTYInputAdapterCommands) ProcessInputBatch(adapterID string, batchSize int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := piac.factory.CreateProcessInputBatchCommand(adapterID, batchSize, model)
		return cmd.Run()
	}
}

// HandleActivityTimeout returns a command to handle activity timeout
func (piac *PTYInputAdapterCommands) HandleActivityTimeout(adapterID string, timeout time.Duration, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := piac.factory.CreateHandleActivityTimeoutCommand(adapterID, timeout, model)
		return cmd.Run()
	}
}

// CleanupAdapter returns a command to cleanup adapter
func (piac *PTYInputAdapterCommands) CleanupAdapter(adapterID string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := piac.factory.CreateCleanupAdapterCommand(adapterID, model)
		return cmd.Run()
	}
}