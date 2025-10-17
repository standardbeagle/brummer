package handlers

import (
	"fmt"
	"sync"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/commands"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	tea "github.com/charmbracelet/bubbletea"
)

// CommandHandler manages command execution and result processing
type CommandHandler struct {
	runningCommands map[string]*RunningCommand
	completedQueue  []*CompletedCommand
	mu              sync.RWMutex
	factory         *commands.CommandFactory
	maxQueueSize    int
	timeout         time.Duration
}

// RunningCommand tracks a command that is currently executing
type RunningCommand struct {
	Command      commands.Command `json:"command"`
	StartTime    time.Time        `json:"startTime"`
	Timeout      time.Time        `json:"timeout"`
	RetryCount   int              `json:"retryCount"`
	RetryPolicy  commands.RetryPolicy `json:"retryPolicy"`
	Cancelled    bool             `json:"cancelled"`
}

// CompletedCommand represents a completed command with its result
type CompletedCommand struct {
	CommandID   string        `json:"commandId"`
	CommandType commands.CommandType `json:"commandType"`
	Result      tea.Msg       `json:"result"`
	Duration    time.Duration `json:"duration"`
	CompletedAt time.Time     `json:"completedAt"`
	Error       error         `json:"error,omitempty"`
}

// CommandResult represents the result of command execution
type CommandResult struct {
	Success     bool                   `json:"success"`
	CommandID   string                 `json:"commandId"`
	CommandType commands.CommandType   `json:"commandType"`
	Result      interface{}            `json:"result,omitempty"`
	Error       error                  `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Retries     int                    `json:"retries"`
	Context     commands.OperationContext `json:"context"`
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(factory *commands.CommandFactory, maxQueueSize int, defaultTimeout time.Duration) *CommandHandler {
	return &CommandHandler{
		runningCommands: make(map[string]*RunningCommand),
		completedQueue:  make([]*CompletedCommand, 0, maxQueueSize),
		factory:         factory,
		maxQueueSize:    maxQueueSize,
		timeout:         defaultTimeout,
	}
}

// ExecuteCommand executes a command and returns the appropriate message
func (h *CommandHandler) ExecuteCommand(cmd commands.Command) tea.Cmd {
	return func() tea.Msg {
		startTime := time.Now()

		// Track the running command
		h.mu.Lock()
		runningCmd := &RunningCommand{
			Command:     cmd,
			StartTime:   startTime,
			Timeout:     startTime.Add(cmd.Context().Timeout),
			RetryCount:  0,
			RetryPolicy: cmd.Context().RetryPolicy,
			Cancelled:   false,
		}
		h.runningCommands[cmd.ID()] = runningCmd
		h.mu.Unlock()

		// Send started message (TODO: Fix type mismatch in messages package)
		// _ = messages.NewCommandStartedMessage(
		//	cmd.ID(),
		//	cmd.Type(),
		//	cmd.Context(),
		// )

		// Execute the command
		resultMsg := cmd.Run()
		duration := time.Since(startTime)

		// Process the result
		completedMsg := h.processCommandResult(cmd, resultMsg, duration, runningCmd)

		// Clean up tracking
		h.mu.Lock()
		delete(h.runningCommands, cmd.ID())

		// Add to completed queue
		completed := &CompletedCommand{
			CommandID:   cmd.ID(),
			CommandType: cmd.Type(),
			Result:      completedMsg,
			Duration:    duration,
			CompletedAt: time.Now(),
		}

		// Maintain queue size
		if len(h.completedQueue) >= h.maxQueueSize {
			h.completedQueue = h.completedQueue[1:]
		}
		h.completedQueue = append(h.completedQueue, completed)
		h.mu.Unlock()

		// Return the completed message
		return completedMsg
	}
}

// ExecuteBatchCommand executes a batch command
func (h *CommandHandler) ExecuteBatchCommand(batch *commands.BatchCommand) tea.Cmd {
	return func() tea.Msg {
		// Send batch started message (TODO: Fix type mismatch in messages package)
		// _ = messages.NewBatchCommandStartedMessage(
		//	batch.ID(),
		//	len(batch.Commands()),
		//	batch.ExecutionMode(),
		//	batch.Context(),
		// )

		// Execute the batch command
		resultMsg := batch.Run()

		// Convert batch result to message (TODO: Fix type mismatch in messages package)
		if _, ok := resultMsg.(commands.BatchResultMsg); ok {
			// return messages.NewBatchCommandCompletedMessage(
			//	batchResult.ID,
			//	batchResult.Results,
			//	batchResult.Duration,
			//	batchResult.Context,
			// )
			return resultMsg
		}

		return resultMsg
	}
}

// processCommandResult processes the result of a command execution
func (h *CommandHandler) processCommandResult(cmd commands.Command, resultMsg tea.Msg, duration time.Duration, runningCmd *RunningCommand) tea.Msg {
	// Check if the command was cancelled
	if runningCmd.Cancelled {
		// TODO: Fix type mismatch in messages package
		// return messages.NewCommandCancelledMessage(
		//	cmd.ID(),
		//	cmd.Type(),
		//	"command was cancelled",
		//	duration,
		//	cmd.Context(),
		// )
		return tea.Msg(nil)
	}

	// Check for timeout
	if time.Now().After(runningCmd.Timeout) {
		// TODO: Fix type mismatch in messages package
		// return messages.NewOperationTimeoutMessage(
		//	cmd.ID(),
		//	cmd.Type(),
		//	cmd.Context().Timeout,
		//	duration,
		//	cmd.Context(),
		// )
		return tea.Msg(nil)
	}

	// Process the result based on message type
	switch msg := resultMsg.(type) {
	case commands.IOResultMsg:
		if msg.IsSuccess() {
			// TODO: Fix type mismatch in messages package
			// return messages.NewCommandCompletedMessage(
			//	msg.ID,
			//	msg.Type,
			//	msg.Result,
			//	msg.Duration,
			//	msg.Context,
			// )
			return resultMsg
		} else {
			return h.handleCommandFailure(cmd, msg.Error, duration, runningCmd)
		}

	case commands.ProcessResultMsg:
		if msg.IsSuccess() {
			// TODO: Fix type mismatch in messages package
			// return messages.NewCommandCompletedMessage(
			//	msg.ID,
			//	msg.Type,
			//	msg.Result,
			//	msg.Duration,
			//	msg.Context,
			// )
			return resultMsg
		} else {
			return h.handleCommandFailure(cmd, msg.Error, duration, runningCmd)
		}

	case commands.NetworkResultMsg:
		if msg.IsSuccess() {
			// TODO: Fix type mismatch in messages package
			// return messages.NewCommandCompletedMessage(
			//	msg.ID,
			//	msg.Type,
			//	msg.Result,
			//	msg.Duration,
			//	msg.Context,
			// )
			return resultMsg
		} else {
			return h.handleCommandFailure(cmd, msg.Error, duration, runningCmd)
		}

	case commands.SystemResultMsg:
		if msg.IsSuccess() {
			// TODO: Fix type mismatch in messages package
			// return messages.NewCommandCompletedMessage(
			//	msg.ID,
			//	msg.Type,
			//	msg.Result,
			//	msg.Duration,
			//	msg.Context,
			// )
			return resultMsg
		} else {
			return h.handleCommandFailure(cmd, msg.Error, duration, runningCmd)
		}

	case commands.RetryResultMsg:
		if msg.IsSuccess() {
			// TODO: Fix type mismatch in messages package
			// return messages.NewCommandCompletedMessage(
			//	msg.ID,
			//	msg.Type,
			//	msg.Result,
			//	msg.Duration,
			//	msg.Context,
			// )
			return resultMsg
		} else {
			return h.handleCommandFailure(cmd, msg.Error, duration, runningCmd)
		}

	case commands.BatchResultMsg:
		// TODO: Fix type mismatch in messages package
		// return messages.NewBatchCommandCompletedMessage(
		//	msg.ID,
		//	msg.Results,
		//	msg.Duration,
		//	msg.Context,
		// )
		return resultMsg

	default:
		// Unknown message type, treat as success
		// TODO: Fix type mismatch in messages package
		// return messages.NewCommandCompletedMessage(
		//	cmd.ID(),
		//	cmd.Type(),
		//	resultMsg,
		//	duration,
		//	cmd.Context(),
		// )
		return resultMsg
	}
}

// handleCommandFailure handles command failure and determines if retry is needed
func (h *CommandHandler) handleCommandFailure(cmd commands.Command, err error, duration time.Duration, runningCmd *RunningCommand) tea.Msg {
	// Check if we should retry
	if runningCmd.RetryCount < runningCmd.RetryPolicy.MaxAttempts && h.shouldRetry(err, runningCmd.RetryPolicy) {
		// Schedule retry
		runningCmd.RetryCount++
		_ = h.calculateBackoff(runningCmd.RetryCount, runningCmd.RetryPolicy)

		// TODO: Fix type mismatch in messages package
		// return messages.NewRetryAttemptMessage(
		//	fmt.Sprintf("retry-%s-%d", cmd.ID(), runningCmd.RetryCount),
		//	cmd.ID(),
		//	runningCmd.RetryCount,
		//	runningCmd.RetryPolicy.MaxAttempts,
		//	err,
		//	backoffDelay,
		//	cmd.Context(),
		// )
		return tea.Msg(nil)
	}

	// No more retries, send failure message
	_ = runningCmd.RetryCount < runningCmd.RetryPolicy.MaxAttempts
	// TODO: Fix type mismatch in messages package
	// return messages.NewCommandFailedMessage(
	//	cmd.ID(),
	//	cmd.Type(),
	//	err,
	//	duration,
	//	cmd.Context(),
	//	retryable,
	// )
	return tea.Msg(nil)
}

// shouldRetry determines if an error is retryable based on the retry policy
func (h *CommandHandler) shouldRetry(err error, policy commands.RetryPolicy) bool {
	if err == nil {
		return false
	}

	// Check if error is in the list of retryable errors
	for _, retryableErr := range policy.RetryableErrors {
		if err.Error() == retryableErr.Error() {
			return true
		}
	}

	// Default to true for unknown errors (be optimistic)
	return len(policy.RetryableErrors) == 0
}

// calculateBackoff calculates exponential backoff delay
func (h *CommandHandler) calculateBackoff(attempt int, policy commands.RetryPolicy) time.Duration {
	delay := policy.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
			break
		}
	}
	return delay
}

// CancelCommand cancels a running command
func (h *CommandHandler) CancelCommand(commandID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if runningCmd, exists := h.runningCommands[commandID]; exists {
		runningCmd.Cancelled = true
		return true
	}
	return false
}

// GetRunningCommand returns information about a running command
func (h *CommandHandler) GetRunningCommand(commandID string) (*RunningCommand, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cmd, exists := h.runningCommands[commandID]
	if !exists {
		return nil, false
	}

	// Return a copy
	return &RunningCommand{
		Command:     cmd.Command,
		StartTime:   cmd.StartTime,
		Timeout:     cmd.Timeout,
		RetryCount:  cmd.RetryCount,
		RetryPolicy: cmd.RetryPolicy,
		Cancelled:   cmd.Cancelled,
	}, true
}

// GetRunningCommands returns all running commands
func (h *CommandHandler) GetRunningCommands() map[string]*RunningCommand {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*RunningCommand)
	for id, cmd := range h.runningCommands {
		result[id] = &RunningCommand{
			Command:     cmd.Command,
			StartTime:   cmd.StartTime,
			Timeout:     cmd.Timeout,
			RetryCount:  cmd.RetryCount,
			RetryPolicy: cmd.RetryPolicy,
			Cancelled:   cmd.Cancelled,
		}
	}
	return result
}

// GetCompletedCommands returns recent completed commands
func (h *CommandHandler) GetCompletedCommands(limit int) []*CompletedCommand {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit >= len(h.completedQueue) {
		result := make([]*CompletedCommand, len(h.completedQueue))
		copy(result, h.completedQueue)
		return result
	}

	result := make([]*CompletedCommand, limit)
	start := len(h.completedQueue) - limit
	copy(result, h.completedQueue[start:])
	return result
}

// CheckTimeouts checks for commands that have timed out
func (h *CommandHandler) CheckTimeouts() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	var timedOutCommands []string

	for id, cmd := range h.runningCommands {
		if now.After(cmd.Timeout) {
			timedOutCommands = append(timedOutCommands, id)
		}
	}

	return timedOutCommands
}

// Cleanup removes old completed commands
func (h *CommandHandler) Cleanup(maxAge time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	updatedQueue := make([]*CompletedCommand, 0, len(h.completedQueue))

	for _, cmd := range h.completedQueue {
		if cmd.CompletedAt.After(cutoff) {
			updatedQueue = append(updatedQueue, cmd)
		}
	}

	h.completedQueue = updatedQueue
}

// GetStats returns handler statistics
func (h *CommandHandler) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	totalCompleted := len(h.completedQueue)
	runningByType := make(map[commands.CommandType]int)
	completedByType := make(map[commands.CommandType]int)

	for _, cmd := range h.runningCommands {
		runningByType[cmd.Command.Type()]++
	}

	for _, cmd := range h.completedQueue {
		completedByType[cmd.CommandType]++
	}

	return map[string]interface{}{
		"runningCommands":  len(h.runningCommands),
		"completedCommands": totalCompleted,
		"maxQueueSize":     h.maxQueueSize,
		"queueUtilization": float64(totalCompleted) / float64(h.maxQueueSize),
		"runningByType":    runningByType,
		"completedByType":  completedByType,
	}
}

// CreateCommand creates a command using the factory
func (h *CommandHandler) CreateCommand(commandType string, operation string, params map[string]interface{}) (commands.Command, error) {
	switch commandType {
	case "IO":
		return h.createIOCommand(operation, params)
	case "Process":
		return h.createProcessCommand(operation, params)
	case "Network":
		return h.createNetworkCommand(operation, params)
	case "System":
		return h.createSystemCommand(operation, params)
	default:
		return nil, fmt.Errorf("unknown command type: %s", commandType)
	}
}

// createIOCommand creates an I/O command
func (h *CommandHandler) createIOCommand(operation string, params map[string]interface{}) (commands.Command, error) {
	switch operation {
	case "readFile":
		if path, ok := params["path"].(string); ok {
			return h.factory.NewReadFileCommand(path), nil
		}
	case "writeFile":
		if path, ok := params["path"].(string); ok {
			if content, ok := params["content"].([]byte); ok {
				return h.factory.NewWriteFileCommand(path, content), nil
			}
		}
	case "readDir":
		if path, ok := params["path"].(string); ok {
			return h.factory.NewReadDirCommand(path), nil
		}
	case "mkdir":
		if path, ok := params["path"].(string); ok {
			return h.factory.NewMkdirCommand(path), nil
		}
	case "remove":
		if path, ok := params["path"].(string); ok {
			return h.factory.NewRemoveCommand(path), nil
		}
	case "copy":
		if src, ok := params["src"].(string); ok {
			if dst, ok := params["dst"].(string); ok {
				return h.factory.NewCopyCommand(src, dst), nil
			}
		}
	}

	return nil, fmt.Errorf("invalid parameters for IO operation %s", operation)
}

// createProcessCommand creates a process command
func (h *CommandHandler) createProcessCommand(operation string, params map[string]interface{}) (commands.Command, error) {
	switch operation {
	case "startScript":
		if scriptName, ok := params["scriptName"].(string); ok {
			args, _ := params["args"].([]string)
			workingDir, _ := params["workingDir"].(string)
			return h.factory.NewStartScriptCommand(scriptName, args, workingDir), nil
		}
	case "startNPM":
		if scriptName, ok := params["scriptName"].(string); ok {
			workingDir, _ := params["workingDir"].(string)
			return h.factory.NewStartNPMCommand(scriptName, workingDir), nil
		}
	case "startNode":
		if scriptFile, ok := params["scriptFile"].(string); ok {
			args, _ := params["args"].([]string)
			workingDir, _ := params["workingDir"].(string)
			return h.factory.NewStartNodeCommand(scriptFile, args, workingDir), nil
		}
	case "stopProcess":
		if pid, ok := params["pid"].(int); ok {
			return h.factory.NewStopProcessCommand(pid), nil
		}
	}

	return nil, fmt.Errorf("invalid parameters for Process operation %s", operation)
}

// createNetworkCommand creates a network command
func (h *CommandHandler) createNetworkCommand(operation string, params map[string]interface{}) (commands.Command, error) {
	switch operation {
	case "GET":
		if url, ok := params["url"].(string); ok {
			headers, _ := params["headers"].(map[string]string)
			timeout, _ := params["timeout"].(time.Duration)
			if timeout == 0 {
				timeout = 30 * time.Second
			}
			return h.factory.NewGETCommand(url, headers, timeout), nil
		}
	case "POST":
		if url, ok := params["url"].(string); ok {
			body := params["body"]
			headers, _ := params["headers"].(map[string]string)
			timeout, _ := params["timeout"].(time.Duration)
			if timeout == 0 {
				timeout = 30 * time.Second
			}
			return h.factory.NewPOSTCommand(url, body, headers, timeout), nil
		}
	case "healthCheck":
		if url, ok := params["url"].(string); ok {
			timeout, _ := params["timeout"].(time.Duration)
			if timeout == 0 {
				timeout = 10 * time.Second
			}
			return h.factory.NewHealthCheckCommand(url, timeout), nil
		}
	}

	return nil, fmt.Errorf("invalid parameters for Network operation %s", operation)
}

// createSystemCommand creates a system command
func (h *CommandHandler) createSystemCommand(operation string, params map[string]interface{}) (commands.Command, error) {
	switch operation {
	case "systemInfo":
		return h.factory.NewSystemInfoCommand(), nil
	case "environment":
		return h.factory.NewEnvironmentCommand(), nil
	case "uptime":
		return h.factory.NewUptimeCommand(), nil
	case "memory":
		return h.factory.NewMemoryCommand(), nil
	case "cpu":
		return h.factory.NewCPUCommand(), nil
	case "temp":
		return h.factory.NewTempCommand(), nil
	case "disk":
		if path, ok := params["path"].(string); ok {
			return h.factory.NewDiskCommand(path), nil
		}
	}

	return nil, fmt.Errorf("invalid parameters for System operation %s", operation)
}