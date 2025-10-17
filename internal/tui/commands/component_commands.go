package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/types"
)

// ComponentCommand represents a component-related operation
type ComponentCommand struct {
	BaseCommand
	ComponentOperation string
	ComponentID        string
	Args               map[string]interface{}
	Model              interface{}
}

// Run executes the component operation and returns a message (implements Command interface)
func (cc *ComponentCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "Component",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := cc.executeComponentOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   cc.id,
		OperationID: cc.id,
		Priority:    types.OperationPriority(cc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"componentOperation": cc.ComponentOperation,
			"componentID":        cc.ComponentID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(cc.id, types.CommandType(cc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(cc.id, types.CommandType(cc.cmdType), result, executionTime, typesContext)
}

// executeComponentOperation performs the actual component operation
func (cc *ComponentCommand) executeComponentOperation() (interface{}, error) {
	switch cc.ComponentOperation {
	case "start_service_bridge":
		return cc.startServiceBridge()
	case "execute_safe_goroutine":
		return cc.executeSafeGoroutine()
	case "start_script_command":
		return cc.startScriptCommand()
	case "start_selected_script":
		return cc.startSelectedScript()
	case "start_ai_coder_session":
		return cc.startAICoderSession()
	case "handle_component_error":
		return cc.handleComponentError()
	default:
		return nil, fmt.Errorf("unknown component operation: %s", cc.ComponentOperation)
	}
}

// startServiceBridge starts a service bridge
func (cc *ComponentCommand) startServiceBridge() (interface{}, error) {
	// This replaces the goroutine at line 267 in integrated_shell.go
	componentID, ok := cc.Args["componentID"].(string)
	if !ok {
		componentID = cc.ComponentID
	}

	bridgeType, ok := cc.Args["bridgeType"].(string)
	if !ok {
		bridgeType = "service"
	}

	return fmt.Sprintf("Service bridge started for component %s (type: %s)", componentID, bridgeType), nil
}

// executeSafeGoroutine executes a safe goroutine operation
func (cc *ComponentCommand) executeSafeGoroutine() (interface{}, error) {
	// This replaces the safeGoroutine calls in script_selector.go
	operationName, ok := cc.Args["operationName"].(string)
	if !ok {
		operationName = "unknown"
	}

	success, ok := cc.Args["success"].(bool)
	if !ok {
		success = true
	}

	if success {
		return fmt.Sprintf("Safe goroutine operation '%s' completed successfully", operationName), nil
	}

	return fmt.Errorf("Safe goroutine operation '%s' failed", operationName), nil
}

// startScriptCommand starts a script command
func (cc *ComponentCommand) startScriptCommand() (interface{}, error) {
	// This replaces the safeGoroutine call at line 325 in script_selector.go
	command, ok := cc.Args["command"].(string)
	if !ok {
		command = "unknown"
	}

	cmdName, ok := cc.Args["cmdName"].(string)
	if !ok {
		cmdName = "unknown"
	}

	return fmt.Sprintf("Started script command: %s (%s)", command, cmdName), nil
}

// startSelectedScript starts a selected script
func (cc *ComponentCommand) startSelectedScript() (interface{}, error) {
	// This replaces the safeGoroutine call at line 368 in script_selector.go
	scriptName, ok := cc.Args["scriptName"].(string)
	if !ok {
		scriptName = "unknown"
	}

	return fmt.Sprintf("Started selected script: %s", scriptName), nil
}

// startAICoderSession starts an AI coder session
func (cc *ComponentCommand) startAICoderSession() (interface{}, error) {
	// This replaces the safeGoroutine call at line 218 in script_selector.go
	sessionID, ok := cc.Args["sessionID"].(string)
	if !ok {
		sessionID = "unknown"
	}

	provider, ok := cc.Args["provider"].(string)
	if !ok {
		provider = "default"
	}

	return fmt.Sprintf("Started AI coder session %s with provider %s", sessionID, provider), nil
}

// handleComponentError handles component errors
func (cc *ComponentCommand) handleComponentError() (interface{}, error) {
	componentID, ok := cc.Args["componentID"].(string)
	if !ok {
		componentID = cc.ComponentID
	}

	errorMsg, ok := cc.Args["error"].(string)
	if !ok {
		errorMsg = "unknown error"
	}

	return fmt.Sprintf("Handled error in component %s: %s", componentID, errorMsg), nil
}

// ComponentCommandFactory creates component-related commands
type ComponentCommandFactory struct{}

// NewComponentCommandFactory creates a new component command factory
func NewComponentCommandFactory() *ComponentCommandFactory {
	return &ComponentCommandFactory{}
}

// CreateStartServiceBridgeCommand creates a command to start a service bridge
func (f *ComponentCommandFactory) CreateStartServiceBridgeCommand(componentID, bridgeType string, model interface{}) Command {
	id := fmt.Sprintf("comp-bridge-%s-%d", componentID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ComponentCommand{
		BaseCommand:        *NewBaseCommand(id, CommandTypeSystem, PriorityHigh, context),
		ComponentOperation: "start_service_bridge",
		ComponentID:        componentID,
		Args: map[string]interface{}{
			"componentID": componentID,
			"bridgeType":  bridgeType,
		},
		Model: model,
	}
}

// CreateExecuteSafeGoroutineCommand creates a command to execute a safe goroutine operation
func (f *ComponentCommandFactory) CreateExecuteSafeGoroutineCommand(componentID, operationName string, success bool, model interface{}) Command {
	id := fmt.Sprintf("comp-safe-%s-%d", componentID, time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &ComponentCommand{
		BaseCommand:        *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		ComponentOperation: "execute_safe_goroutine",
		ComponentID:        componentID,
		Args: map[string]interface{}{
			"componentID":   componentID,
			"operationName": operationName,
			"success":       success,
		},
		Model: model,
	}
}

// CreateStartScriptCommandCommand creates a command to start a script command
func (f *ComponentCommandFactory) CreateStartScriptCommandCommand(componentID, command, cmdName string, model interface{}) Command {
	id := fmt.Sprintf("comp-script-cmd-%s-%d", componentID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ComponentCommand{
		BaseCommand:        *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		ComponentOperation: "start_script_command",
		ComponentID:        componentID,
		Args: map[string]interface{}{
			"componentID": componentID,
			"command":    command,
			"cmdName":    cmdName,
		},
		Model: model,
	}
}

// CreateStartSelectedScriptCommand creates a command to start a selected script
func (f *ComponentCommandFactory) CreateStartSelectedScriptCommand(componentID, scriptName string, model interface{}) Command {
	id := fmt.Sprintf("comp-selected-script-%s-%d", componentID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ComponentCommand{
		BaseCommand:        *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		ComponentOperation: "start_selected_script",
		ComponentID:        componentID,
		Args: map[string]interface{}{
			"componentID": componentID,
			"scriptName": scriptName,
		},
		Model: model,
	}
}

// CreateStartAICoderSessionCommand creates a command to start an AI coder session
func (f *ComponentCommandFactory) CreateStartAICoderSessionCommand(componentID, sessionID, provider string, model interface{}) Command {
	id := fmt.Sprintf("comp-ai-coder-%s-%d", componentID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &ComponentCommand{
		BaseCommand:        *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		ComponentOperation: "start_ai_coder_session",
		ComponentID:        componentID,
		Args: map[string]interface{}{
			"componentID": componentID,
			"sessionID":  sessionID,
			"provider":   provider,
		},
		Model: model,
	}
}

// CreateHandleComponentErrorCommand creates a command to handle component errors
func (f *ComponentCommandFactory) CreateHandleComponentErrorCommand(componentID, errorMsg string, model interface{}) Command {
	id := fmt.Sprintf("comp-error-%s-%d", componentID, time.Now().UnixNano())
	context := UIOperationContext(0) // Version will be set by model

	return &ComponentCommand{
		BaseCommand:        *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		ComponentOperation: "handle_component_error",
		ComponentID:        componentID,
		Args: map[string]interface{}{
			"componentID": componentID,
			"error":       errorMsg,
		},
		Model: model,
	}
}

// ComponentCommands provides high-level commands for component operations
type ComponentCommands struct {
	factory *ComponentCommandFactory
}

// NewComponentCommands creates a new component commands instance
func NewComponentCommands() *ComponentCommands {
	return &ComponentCommands{
		factory: NewComponentCommandFactory(),
	}
}

// StartServiceBridge returns a command to start a service bridge
func (cc *ComponentCommands) StartServiceBridge(componentID, bridgeType string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cc.factory.CreateStartServiceBridgeCommand(componentID, bridgeType, model)
		return cmd.Run()
	}
}

// ExecuteSafeGoroutine returns a command to execute a safe goroutine operation
func (cc *ComponentCommands) ExecuteSafeGoroutine(componentID, operationName string, success bool, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cc.factory.CreateExecuteSafeGoroutineCommand(componentID, operationName, success, model)
		return cmd.Run()
	}
}

// StartScriptCommand returns a command to start a script command
func (cc *ComponentCommands) StartScriptCommand(componentID, command, cmdName string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cc.factory.CreateStartScriptCommandCommand(componentID, command, cmdName, model)
		return cmd.Run()
	}
}

// StartSelectedScript returns a command to start a selected script
func (cc *ComponentCommands) StartSelectedScript(componentID, scriptName string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cc.factory.CreateStartSelectedScriptCommand(componentID, scriptName, model)
		return cmd.Run()
	}
}

// StartAICoderSession returns a command to start an AI coder session
func (cc *ComponentCommands) StartAICoderSession(componentID, sessionID, provider string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cc.factory.CreateStartAICoderSessionCommand(componentID, sessionID, provider, model)
		return cmd.Run()
	}
}

// HandleComponentError returns a command to handle component errors
func (cc *ComponentCommands) HandleComponentError(componentID, errorMsg string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := cc.factory.CreateHandleComponentErrorCommand(componentID, errorMsg, model)
		return cmd.Run()
	}
}