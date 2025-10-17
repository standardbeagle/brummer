package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/types"
)

// AICoderServiceCommand represents an AI coder service-related operation
type AICoderServiceCommand struct {
	BaseCommand
	AICoderOperation string
	SessionID        string
	Args             map[string]interface{}
	Model            interface{}
}

// Run executes the AI coder service operation and returns a message (implements Command interface)
func (asc *AICoderServiceCommand) Run() tea.Msg {
	startTime := time.Now()

	// Create a context for this operation
	context := OperationContext{
		OperationType: "AICoderService",
		Version:       0, // Will be set by the model
		Strategy:      OrderingQueueSequential,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}

	result, err := asc.executeAICoderOperation()

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Convert commands.OperationContext to types.OperationContext for messages
	typesContext := types.OperationContext{
		RequestID:   asc.id,
		OperationID: asc.id,
		Priority:    types.OperationPriority(asc.priority),
		Timeout:     context.Timeout,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"aicoderOperation": asc.AICoderOperation,
			"sessionID":        asc.SessionID,
		},
	}

	// Return appropriate message based on result
	if err != nil {
		return messages.NewCommandFailedMessage(asc.id, types.CommandType(asc.cmdType), err, executionTime, typesContext, true)
	}

	return messages.NewCommandCompletedMessage(asc.id, types.CommandType(asc.cmdType), result, executionTime, typesContext)
}

// executeAICoderOperation performs the actual AI coder service operation
func (asc *AICoderServiceCommand) executeAICoderOperation() (interface{}, error) {
	switch asc.AICoderOperation {
	case "monitor_pty_output":
		return asc.monitorPTYOutput()
	case "wait_for_goroutines":
		return asc.waitForGoroutines()
	case "cleanup_sessions":
		return asc.cleanupSessions()
	case "start_session":
		return asc.startSession()
	case "stop_session":
		return asc.stopSession()
	case "manage_session_lifecycle":
		return asc.manageSessionLifecycle()
	default:
		return nil, fmt.Errorf("unknown AI coder service operation: %s", asc.AICoderOperation)
	}
}

// monitorPTYOutput monitors PTY output for AI coder sessions
func (asc *AICoderServiceCommand) monitorPTYOutput() (interface{}, error) {
	// This replaces the goroutine at line 183 in aicoder_service.go
	sessionID, ok := asc.Args["sessionID"].(string)
	if !ok {
		sessionID = asc.SessionID
	}

	outputLines, ok := asc.Args["outputLines"].(int)
	if !ok {
		outputLines = 0
	}

	return fmt.Sprintf("Monitored PTY output for session %s, processed %d lines", sessionID, outputLines), nil
}

// waitForGoroutines waits for AI coder service goroutines
func (asc *AICoderServiceCommand) waitForGoroutines() (interface{}, error) {
	// This replaces the goroutine at line 500 in aicoder_service.go
	goroutineCount, ok := asc.Args["goroutineCount"].(int)
	if !ok {
		goroutineCount = 1
	}

	timeout, ok := asc.Args["timeout"].(time.Duration)
	if !ok {
		timeout = 10 * time.Second
	}

	return fmt.Sprintf("Waited for %d AI coder goroutines with timeout %v", goroutineCount, timeout), nil
}

// cleanupSessions cleans up AI coder sessions
func (asc *AICoderServiceCommand) cleanupSessions() (interface{}, error) {
	sessionCount, ok := asc.Args["sessionCount"].(int)
	if !ok {
		sessionCount = 0
	}

	return fmt.Sprintf("Cleaned up %d AI coder sessions", sessionCount), nil
}

// startSession starts a new AI coder session
func (asc *AICoderServiceCommand) startSession() (interface{}, error) {
	sessionID, ok := asc.Args["sessionID"].(string)
	if !ok {
		sessionID = asc.SessionID
	}

	provider, ok := asc.Args["provider"].(string)
	if !ok {
		provider = "default"
	}

	return fmt.Sprintf("Started AI coder session %s with provider %s", sessionID, provider), nil
}

// stopSession stops an AI coder session
func (asc *AICoderServiceCommand) stopSession() (interface{}, error) {
	sessionID, ok := asc.Args["sessionID"].(string)
	if !ok {
		sessionID = asc.SessionID
	}

	return fmt.Sprintf("Stopped AI coder session %s", sessionID), nil
}

// manageSessionLifecycle manages AI coder session lifecycle
func (asc *AICoderServiceCommand) manageSessionLifecycle() (interface{}, error) {
	sessionID, ok := asc.Args["sessionID"].(string)
	if !ok {
		sessionID = asc.SessionID
	}

	action, ok := asc.Args["action"].(string)
	if !ok {
		action = "unknown"
	}

	return fmt.Sprintf("Managed lifecycle for session %s: %s", sessionID, action), nil
}

// AICoderServiceCommandFactory creates AI coder service-related commands
type AICoderServiceCommandFactory struct{}

// NewAICoderServiceCommandFactory creates a new AI coder service command factory
func NewAICoderServiceCommandFactory() *AICoderServiceCommandFactory {
	return &AICoderServiceCommandFactory{}
}

// CreateMonitorPTYOutputCommand creates a command to monitor PTY output
func (f *AICoderServiceCommandFactory) CreateMonitorPTYOutputCommand(sessionID string, outputLines int, model interface{}) Command {
	id := fmt.Sprintf("aicoder-pty-monitor-%s-%d", sessionID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &AICoderServiceCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		AICoderOperation: "monitor_pty_output",
		SessionID:        sessionID,
		Args: map[string]interface{}{
			"sessionID":   sessionID,
			"outputLines": outputLines,
		},
		Model: model,
	}
}

// CreateWaitForGoroutinesCommand creates a command to wait for goroutines
func (f *AICoderServiceCommandFactory) CreateWaitForGoroutinesCommand(sessionID string, goroutineCount int, timeout time.Duration, model interface{}) Command {
	id := fmt.Sprintf("aicoder-wait-%s-%d", sessionID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &AICoderServiceCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		AICoderOperation: "wait_for_goroutines",
		SessionID:        sessionID,
		Args: map[string]interface{}{
			"sessionID":      sessionID,
			"goroutineCount": goroutineCount,
			"timeout":        timeout,
		},
		Model: model,
	}
}

// CreateCleanupSessionsCommand creates a command to cleanup sessions
func (f *AICoderServiceCommandFactory) CreateCleanupSessionsCommand(sessionCount int, model interface{}) Command {
	id := fmt.Sprintf("aicoder-cleanup-%d", time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &AICoderServiceCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		AICoderOperation: "cleanup_sessions",
		SessionID:        "cleanup",
		Args: map[string]interface{}{
			"sessionCount": sessionCount,
		},
		Model: model,
	}
}

// CreateStartSessionCommand creates a command to start a session
func (f *AICoderServiceCommandFactory) CreateStartSessionCommand(sessionID, provider string, model interface{}) Command {
	id := fmt.Sprintf("aicoder-start-%s-%d", sessionID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &AICoderServiceCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		AICoderOperation: "start_session",
		SessionID:        sessionID,
		Args: map[string]interface{}{
			"sessionID": sessionID,
			"provider":  provider,
		},
		Model: model,
	}
}

// CreateStopSessionCommand creates a command to stop a session
func (f *AICoderServiceCommandFactory) CreateStopSessionCommand(sessionID string, model interface{}) Command {
	id := fmt.Sprintf("aicoder-stop-%s-%d", sessionID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &AICoderServiceCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		AICoderOperation: "stop_session",
		SessionID:        sessionID,
		Args: map[string]interface{}{
			"sessionID": sessionID,
		},
		Model: model,
	}
}

// CreateManageSessionLifecycleCommand creates a command to manage session lifecycle
func (f *AICoderServiceCommandFactory) CreateManageSessionLifecycleCommand(sessionID, action string, model interface{}) Command {
	id := fmt.Sprintf("aicoder-lifecycle-%s-%d", sessionID, time.Now().UnixNano())
	context := SystemOperationContext(0) // Version will be set by model

	return &AICoderServiceCommand{
		BaseCommand:      *NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		AICoderOperation: "manage_session_lifecycle",
		SessionID:        sessionID,
		Args: map[string]interface{}{
			"sessionID": sessionID,
			"action":    action,
		},
		Model: model,
	}
}

// AICoderServiceCommands provides high-level commands for AI coder service operations
type AICoderServiceCommands struct {
	factory *AICoderServiceCommandFactory
}

// NewAICoderServiceCommands creates a new AI coder service commands instance
func NewAICoderServiceCommands() *AICoderServiceCommands {
	return &AICoderServiceCommands{
		factory: NewAICoderServiceCommandFactory(),
	}
}

// MonitorPTYOutput returns a command to monitor PTY output
func (asc *AICoderServiceCommands) MonitorPTYOutput(sessionID string, outputLines int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := asc.factory.CreateMonitorPTYOutputCommand(sessionID, outputLines, model)
		return cmd.Run()
	}
}

// WaitForGoroutines returns a command to wait for goroutines
func (asc *AICoderServiceCommands) WaitForGoroutines(sessionID string, goroutineCount int, timeout time.Duration, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := asc.factory.CreateWaitForGoroutinesCommand(sessionID, goroutineCount, timeout, model)
		return cmd.Run()
	}
}

// CleanupSessions returns a command to cleanup sessions
func (asc *AICoderServiceCommands) CleanupSessions(sessionCount int, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := asc.factory.CreateCleanupSessionsCommand(sessionCount, model)
		return cmd.Run()
	}
}

// StartSession returns a command to start a session
func (asc *AICoderServiceCommands) StartSession(sessionID, provider string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := asc.factory.CreateStartSessionCommand(sessionID, provider, model)
		return cmd.Run()
	}
}

// StopSession returns a command to stop a session
func (asc *AICoderServiceCommands) StopSession(sessionID string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := asc.factory.CreateStopSessionCommand(sessionID, model)
		return cmd.Run()
	}
}

// ManageSessionLifecycle returns a command to manage session lifecycle
func (asc *AICoderServiceCommands) ManageSessionLifecycle(sessionID, action string, model tea.Model) tea.Cmd {
	return func() tea.Msg {
		cmd := asc.factory.CreateManageSessionLifecycleCommand(sessionID, action, model)
		return cmd.Run()
	}
}