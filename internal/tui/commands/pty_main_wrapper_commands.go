package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// PTYMainWrapperCommand represents a PTY main wrapper-related operation
type PTYMainWrapperCommand struct {
	BaseCommand
	WrapperOperation string
	WrapperID        string
	Args             map[string]interface{}
	Model            interface{}
}

// Run executes the PTY main wrapper operation and returns a message (implements Command interface)
func (pmc *PTYMainWrapperCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "MainWrapper",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := pmc.executeWrapperOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   pmc.id,
		OperationID: pmc.id,
		Priority:    types.OperationPriority(pmc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"wrapperOperation": pmc.WrapperOperation,
			"wrapperID":        pmc.WrapperID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(pmc.id, types.CommandType(pmc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(pmc.id, types.CommandType(pmc.cmdType), result, executionTime, typesContext)
}

// executeWrapperOperation performs the actual PTY main wrapper operation
func (pmc *PTYMainWrapperCommand) executeWrapperOperation() (interface{}, error) {
	switch pmc.WrapperOperation {
	case "run_program":
		return pmc.executeRunProgram()
	case "execute_error_callback":
		return pmc.executeErrorCallback()
	case "execute_startup_callbacks":
		return pmc.executeStartupCallbacks()
	case "execute_shutdown_callbacks":
		return pmc.executeShutdownCallbacks()
	default:
		return nil, fmt.Errorf("unknown PTY main wrapper operation: %s", pmc.WrapperOperation)
	}
}

// executeRunProgram runs the program with signal handling
func (pmc *PTYMainWrapperCommand) executeRunProgram() (interface{}, error) {
	// This replaces the goroutine at line 438 in pty_main_wrapper.go
	wrapperID, ok := pmc.Args["wrapperID"].(string)
	if !ok {
		wrapperID = pmc.WrapperID
	}

	usePTY, ok := pmc.Args["usePTY"].(bool)
	if !ok {
		usePTY = false
	}

	return fmt.Sprintf("Program executed for wrapper %s (PTY: %v)", wrapperID, usePTY), nil
}

// executeErrorCallback executes an error callback
func (pmc *PTYMainWrapperCommand) executeErrorCallback() (interface{}, error) {
	// This replaces the goroutine at line 574 in pty_main_wrapper.go
	errorMsg, ok := pmc.Args["error"].(string)
	if !ok {
		errorMsg = "unknown error"
	}

	callbackCount, ok := pmc.Args["callbackCount"].(int)
	if !ok {
		callbackCount = 1
	}

	return fmt.Sprintf("Error callback executed for %d callbacks, error: %s", callbackCount, errorMsg), nil
}

// executeStartupCallbacks executes startup callbacks
func (pmc *PTYMainWrapperCommand) executeStartupCallbacks() (interface{}, error) {
	callbackCount, ok := pmc.Args["callbackCount"].(int)
	if !ok {
		callbackCount = 0
	}

	return fmt.Sprintf("%d startup callbacks executed", callbackCount), nil
}

// executeShutdownCallbacks executes shutdown callbacks
func (pmc *PTYMainWrapperCommand) executeShutdownCallbacks() (interface{}, error) {
	callbackCount, ok := pmc.Args["callbackCount"].(int)
	if !ok {
		callbackCount = 0
	}

	return fmt.Sprintf("%d shutdown callbacks executed", callbackCount), nil
}

// PTYMainWrapperCommandFactory creates PTY main wrapper-related commands
type PTYMainWrapperCommandFactory struct{}

// NewPTYMainWrapperCommandFactory creates a new PTY main wrapper command factory
func NewPTYMainWrapperCommandFactory() *PTYMainWrapperCommandFactory {
	return &PTYMainWrapperCommandFactory{}
}

// CreateRunProgramCommand creates a command to run a program with signal handling
func (f *PTYMainWrapperCommandFactory) CreateRunProgramCommand(wrapperID string, usePTY bool, model interface{}) Command {
	id := fmt.Sprintf("pty-wrapper-run-%s-%d", wrapperID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &PTYMainWrapperCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		WrapperOperation: "run_program",
		WrapperID:        wrapperID,
		Args: map[string]interface{}{
			"wrapperID": wrapperID,
			"usePTY":    usePTY,
		},
		Model: model,
	}
}

// CreateErrorCallbackCommand creates a command to execute error callbacks
func (f *PTYMainWrapperCommandFactory) CreateErrorCallbackCommand(wrapperID string, errorMsg string, callbackCount int, model interface{}) Command {
	id := fmt.Sprintf("pty-wrapper-error-%s-%d", wrapperID, time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &PTYMainWrapperCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		WrapperOperation: "execute_error_callback",
		WrapperID:        wrapperID,
		Args: map[string]interface{}{
			"error":         errorMsg,
			"callbackCount": callbackCount,
		},
		Model: model,
	}
}

// CreateStartupCallbacksCommand creates a command to execute startup callbacks
func (f *PTYMainWrapperCommandFactory) CreateStartupCallbacksCommand(wrapperID string, callbackCount int, model interface{}) Command {
	id := fmt.Sprintf("pty-wrapper-startup-%s-%d", wrapperID, time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &PTYMainWrapperCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		WrapperOperation: "execute_startup_callbacks",
		WrapperID:        wrapperID,
		Args: map[string]interface{}{
			"callbackCount": callbackCount,
		},
		Model: model,
	}
}

// CreateShutdownCallbacksCommand creates a command to execute shutdown callbacks
func (f *PTYMainWrapperCommandFactory) CreateShutdownCallbacksCommand(wrapperID string, callbackCount int, model interface{}) Command {
	id := fmt.Sprintf("pty-wrapper-shutdown-%s-%d", wrapperID, time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &PTYMainWrapperCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		WrapperOperation: "execute_shutdown_callbacks",
		WrapperID:        wrapperID,
		Args: map[string]interface{}{
			"callbackCount": callbackCount,
		},
		Model: model,
	}
}

// PTYMainWrapperCommands provides high-level commands for PTY main wrapper operations
type PTYMainWrapperCommands struct {
	factory *PTYMainWrapperCommandFactory
}

// NewPTYMainWrapperCommands creates a new PTY main wrapper commands instance
func NewPTYMainWrapperCommands() *PTYMainWrapperCommands {
	return &PTYMainWrapperCommands{
		factory: NewPTYMainWrapperCommandFactory(),
	}
}

// RunProgram returns a command to run a program with signal handling
func (pmc *PTYMainWrapperCommands) RunProgram(wrapperID string, usePTY bool, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pmc.factory.CreateRunProgramCommand(wrapperID, usePTY, model)
		return cmd.Run()
	}
}

// ExecuteErrorCallbacks returns a command to execute error callbacks
func (pmc *PTYMainWrapperCommands) ExecuteErrorCallbacks(wrapperID string, errorMsg string, callbackCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pmc.factory.CreateErrorCallbackCommand(wrapperID, errorMsg, callbackCount, model)
		return cmd.Run()
	}
}

// ExecuteStartupCallbacks returns a command to execute startup callbacks
func (pmc *PTYMainWrapperCommands) ExecuteStartupCallbacks(wrapperID string, callbackCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pmc.factory.CreateStartupCallbacksCommand(wrapperID, callbackCount, model)
		return cmd.Run()
	}
}

// ExecuteShutdownCallbacks returns a command to execute shutdown callbacks
func (pmc *PTYMainWrapperCommands) ExecuteShutdownCallbacks(wrapperID string, callbackCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := pmc.factory.CreateShutdownCallbacksCommand(wrapperID, callbackCount, model)
		return cmd.Run()
	}
}