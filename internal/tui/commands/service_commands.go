package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/types"
)

// ServiceCommandFactory creates commands for existing stateless services
// This bridges the gap between existing services and the command pattern
type ServiceCommandFactory struct{}

// NewServiceCommandFactory creates a new service command factory
func NewServiceCommandFactory() *ServiceCommandFactory {
	return &ServiceCommandFactory{}
}

// ServiceCommand represents a command that executes a service operation
type ServiceCommand struct {
	BaseCommand
	ServiceName string
	Operation   string
	Args        map[string]interface{}
	Model       interface{} // The current model for stateless operation
}

// Run executes the service operation and returns a message (implements Command interface)
func (sc *ServiceCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation using the commands package OperationContext
	context := OperationContext{
		OperationType: "Service",
		Version:       0, // Will be set by the model
		Strategy:      OrderingLastWriteWins,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := sc.executeServiceOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   sc.id,
		OperationID: sc.id,
		Priority:    types.OperationPriority(sc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"serviceName": sc.ServiceName,
			"operation":   sc.Operation,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(sc.id, types.CommandType(sc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(sc.id, types.CommandType(sc.cmdType), result, executionTime, typesContext)
}

// executeServiceOperation performs the actual service operation
func (sc *ServiceCommand) executeServiceOperation() (interface{}, error) {
	switch sc.ServiceName {
	case "input":
		return sc.executeInputOperation()
	case "pty":
		return sc.executePTYOperation()
	case "process":
		return sc.executeProcessOperation()
	case "network":
		return sc.executeNetworkOperation()
	case "notification":
		return sc.executeNotificationOperation()
	case "aicoder":
		return sc.executeAICoderOperation()
	case "cache":
		return sc.executeCacheOperation()
	case "navigation":
		return sc.executeNavigationOperation()
	case "filebrowser":
		return sc.executeFileBrowserOperation()
	case "system":
		return sc.executeSystemOperation()
	default:
		return nil, fmt.Errorf("unknown service: %s", sc.ServiceName)
	}
}

// Service-specific operation executors

func (sc *ServiceCommand) executeInputOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for input service")
	}

	switch operation {
	case "handle_key":
		key, ok := sc.Args["key"].(string)
		if !ok {
			return nil, fmt.Errorf("missing key for handle_key operation")
		}
		// Return a command that will handle the key press
		return fmt.Sprintf("key_handled: %s", key), nil

	case "handle_mouse":
		// Mouse handling would be implemented here
		return "mouse_handled", nil

	default:
		return nil, fmt.Errorf("unknown input operation: %s", operation)
	}
}

func (sc *ServiceCommand) executePTYOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for PTY service")
	}

	switch operation {
	case "start":
		command, ok := sc.Args["command"].(string)
		if !ok {
			return nil, fmt.Errorf("missing command for PTY start operation")
		}
		// Return PTY start result
		return fmt.Sprintf("pty_started: %s", command), nil

	case "send_input":
		ptyID, ok := sc.Args["pty_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing pty_id for send_input operation")
		}
		input, ok := sc.Args["input"].(string)
		if !ok {
			return nil, fmt.Errorf("missing input for send_input operation")
		}
		return fmt.Sprintf("input_sent: %s -> %s", input, ptyID), nil

	case "resize":
		ptyID, ok := sc.Args["pty_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing pty_id for resize operation")
		}
		width, ok := sc.Args["width"].(int)
		if !ok {
			return nil, fmt.Errorf("missing width for resize operation")
		}
		height, ok := sc.Args["height"].(int)
		if !ok {
			return nil, fmt.Errorf("missing height for resize operation")
		}
		return fmt.Sprintf("pty_resized: %s -> %dx%d", ptyID, width, height), nil

	case "terminate":
		ptyID, ok := sc.Args["pty_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing pty_id for terminate operation")
		}
		return fmt.Sprintf("pty_terminated: %s", ptyID), nil

	default:
		return nil, fmt.Errorf("unknown PTY operation: %s", operation)
	}
}

func (sc *ServiceCommand) executeProcessOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for process service")
	}

	switch operation {
	case "start":
		command, ok := sc.Args["command"].(string)
		if !ok {
			return nil, fmt.Errorf("missing command for process start operation")
		}
		return fmt.Sprintf("process_started: %s", command), nil

	case "stop":
		processID, ok := sc.Args["process_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing process_id for stop operation")
		}
		return fmt.Sprintf("process_stopped: %s", processID), nil

	default:
		return nil, fmt.Errorf("unknown process operation: %s", operation)
	}
}

func (sc *ServiceCommand) executeNetworkOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for network service")
	}

	switch operation {
	case "make_request":
		method, ok := sc.Args["method"].(string)
		if !ok {
			return nil, fmt.Errorf("missing method for make_request operation")
		}
		url, ok := sc.Args["url"].(string)
		if !ok {
			return nil, fmt.Errorf("missing url for make_request operation")
		}
		return fmt.Sprintf("request_made: %s %s", method, url), nil

	default:
		return nil, fmt.Errorf("unknown network operation: %s", operation)
	}
}

func (sc *ServiceCommand) executeNotificationOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for notification service")
	}

	switch operation {
	case "show":
		title, ok := sc.Args["title"].(string)
		if !ok {
			return nil, fmt.Errorf("missing title for show notification operation")
		}
		message, ok := sc.Args["message"].(string)
		if !ok {
			return nil, fmt.Errorf("missing message for show notification operation")
		}
		return fmt.Sprintf("notification_shown: %s - %s", title, message), nil

	default:
		return nil, fmt.Errorf("unknown notification operation: %s", operation)
	}
}

func (sc *ServiceCommand) executeAICoderOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for AI coder service")
	}

	switch operation {
	case "start_session":
		modelID, ok := sc.Args["model_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing model_id for start_session operation")
		}
		return fmt.Sprintf("ai_session_started: %s", modelID), nil

	default:
		return nil, fmt.Errorf("unknown AI coder operation: %s", operation)
	}
}

func (sc *ServiceCommand) executeCacheOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for cache service")
	}

	switch operation {
	case "get":
		key, ok := sc.Args["key"].(string)
		if !ok {
			return nil, fmt.Errorf("missing key for cache get operation")
		}
		return fmt.Sprintf("cache_get: %s", key), nil

	case "set":
		key, ok := sc.Args["key"].(string)
		if !ok {
			return nil, fmt.Errorf("missing key for cache set operation")
		}
		return fmt.Sprintf("cache_set: %s", key), nil

	default:
		return nil, fmt.Errorf("unknown cache operation: %s", operation)
	}
}

func (sc *ServiceCommand) executeNavigationOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for navigation service")
	}

	switch operation {
	case "navigate_to":
		target, ok := sc.Args["target"].(string)
		if !ok {
			return nil, fmt.Errorf("missing target for navigate_to operation")
		}
		return fmt.Sprintf("navigated_to: %s", target), nil

	default:
		return nil, fmt.Errorf("unknown navigation operation: %s", operation)
	}
}

func (sc *ServiceCommand) executeFileBrowserOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for file browser service")
	}

	switch operation {
	case "list_directory":
		path, ok := sc.Args["path"].(string)
		if !ok {
			return nil, fmt.Errorf("missing path for list_directory operation")
		}
		return fmt.Sprintf("directory_listed: %s", path), nil

	default:
		return nil, fmt.Errorf("unknown file browser operation: %s", operation)
	}
}

func (sc *ServiceCommand) executeSystemOperation() (interface{}, error) {
	operation, ok := sc.Args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation for system service")
	}

	switch operation {
	case "execute_command":
		command, ok := sc.Args["command"].(string)
		if !ok {
			return nil, fmt.Errorf("missing command for execute_command operation")
		}
		return fmt.Sprintf("command_executed: %s", command), nil

	default:
		return nil, fmt.Errorf("unknown system operation: %s", operation)
	}
}

// Factory methods for creating specific service commands

// CreateInputCommand creates an input service command
func (f *ServiceCommandFactory) CreateInputCommand(operation string, args map[string]interface{}, model interface{}) Command {
	id := fmt.Sprintf("input-%d", time.Now().UnixNano())
	context := DefaultOperationContext(0) // Version will be set by model

	return &ServiceCommand{
		BaseCommand: *NewBaseCommand(id, CommandTypeIO, PriorityMedium, context),
		ServiceName: "input",
		Operation:   operation,
		Args:        args,
		Model:       model,
	}
}

// CreatePTYCommand creates a PTY service command
func (f *ServiceCommandFactory) CreatePTYCommand(operation string, args map[string]interface{}, model interface{}) Command {
	id := fmt.Sprintf("pty-%d", time.Now().UnixNano())
	context := DefaultOperationContext(0) // Version will be set by model

	return &ServiceCommand{
		BaseCommand: *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		ServiceName: "pty",
		Operation:   operation,
		Args:        args,
		Model:       model,
	}
}

// CreateProcessCommand creates a process service command
func (f *ServiceCommandFactory) CreateProcessCommand(operation string, args map[string]interface{}, model interface{}) Command {
	id := fmt.Sprintf("process-%d", time.Now().UnixNano())
	context := DefaultOperationContext(0) // Version will be set by model

	return &ServiceCommand{
		BaseCommand: *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		ServiceName: "process",
		Operation:   operation,
		Args:        args,
		Model:       model,
	}
}

// CreateNetworkCommand creates a network service command
func (f *ServiceCommandFactory) CreateNetworkCommand(operation string, args map[string]interface{}, model interface{}) Command {
	id := fmt.Sprintf("network-%d", time.Now().UnixNano())
	context := DefaultOperationContext(0) // Version will be set by model

	return &ServiceCommand{
		BaseCommand: *NewBaseCommand(id, CommandTypeNetwork, PriorityMedium, context),
		ServiceName: "network",
		Operation:   operation,
		Args:        args,
		Model:       model,
	}
}

// CreateNotificationCommand creates a notification service command
func (f *ServiceCommandFactory) CreateNotificationCommand(operation string, args map[string]interface{}, model interface{}) Command {
	id := fmt.Sprintf("notification-%d", time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &ServiceCommand{
		BaseCommand: *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		ServiceName: "notification",
		Operation:   operation,
		Args:        args,
		Model:       model,
	}
}

// CreateAICoderCommand creates an AI coder service command
func (f *ServiceCommandFactory) CreateAICoderCommand(operation string, args map[string]interface{}, model interface{}) Command {
	id := fmt.Sprintf("aicoder-%d", time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ServiceCommand{
		BaseCommand: *NewBaseCommand(id, CommandTypeSystem, PriorityCritical, context),
		ServiceName: "aicoder",
		Operation:   operation,
		Args:        args,
		Model:       model,
	}
}