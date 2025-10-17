package commands

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// BaseCommand provides common functionality for all commands
type BaseCommand struct {
	id        string
	cmdType   CommandType
	priority  Priority
	context   OperationContext
	createdAt time.Time
}

// NewBaseCommand creates a new base command
func NewBaseCommand(id string, cmdType CommandType, priority Priority, context OperationContext) *BaseCommand {
	return &BaseCommand{
		id:        id,
		cmdType:   cmdType,
		priority:  priority,
		context:   context,
		createdAt: time.Now(),
	}
}

// Command interface implementation
func (c *BaseCommand) ID() string                       { return c.id }
func (c *BaseCommand) Type() CommandType               { return c.cmdType }
func (c *BaseCommand) Priority() Priority               { return c.priority }
func (c *BaseCommand) Context() OperationContext        { return c.context }
func (c *BaseCommand) CreatedAt() time.Time             { return c.createdAt }

// Validate checks if the command is properly configured
func (c *BaseCommand) Validate() error {
	if c.id == "" {
		return fmt.Errorf("command ID cannot be empty")
	}
	if c.cmdType < CommandTypeIO || c.cmdType > CommandTypeSystem {
		return fmt.Errorf("invalid command type: %d", c.cmdType)
	}
	if c.priority < PriorityCritical || c.priority > PriorityBackground {
		return fmt.Errorf("invalid priority: %d", c.priority)
	}
	if c.context.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

// String returns a string representation of the command
func (c *BaseCommand) String() string {
	return fmt.Sprintf("Command{id=%s, type=%d, priority=%d}", c.id, c.cmdType, c.priority)
}

// IsExpired checks if the command has expired based on its creation time and timeout
func (c *BaseCommand) IsExpired() bool {
	return time.Since(c.createdAt) > c.context.Timeout
}

// TimeRemaining returns the time remaining before the command expires
func (c *BaseCommand) TimeRemaining() time.Duration {
	elapsed := time.Since(c.createdAt)
	if elapsed >= c.context.Timeout {
		return 0
	}
	return c.context.Timeout - elapsed
}

// ShouldRetry determines if the command should be retried based on the error
func (c *BaseCommand) ShouldRetry(err error, attemptCount int) bool {
	if attemptCount >= c.context.RetryPolicy.MaxAttempts {
		return false
	}

	// Check if error is in the list of retryable errors
	for _, retryableErr := range c.context.RetryPolicy.RetryableErrors {
		if err.Error() == retryableErr.Error() {
			return true
		}
	}

	// If no specific retryable errors are defined, retry all errors
	if len(c.context.RetryPolicy.RetryableErrors) == 0 {
		return true
	}

	return false
}

// CalculateRetryDelay calculates the delay before the next retry attempt
func (c *BaseCommand) CalculateRetryDelay(attemptCount int) time.Duration {
	if attemptCount <= 0 {
		return c.context.RetryPolicy.InitialDelay
	}

	// Exponential backoff: delay = initialDelay * (backoffFactor ^ (attempt - 1))
	delay := c.context.RetryPolicy.InitialDelay
	for i := 1; i < attemptCount; i++ {
		delay = time.Duration(float64(delay) * c.context.RetryPolicy.BackoffFactor)
	}

	// Cap at max delay
	if delay > c.context.RetryPolicy.MaxDelay {
		delay = c.context.RetryPolicy.MaxDelay
	}

	return delay
}

// WithContext returns a copy of the command with a new context
func (c *BaseCommand) WithContext(context OperationContext) *BaseCommand {
	return &BaseCommand{
		id:        c.id,
		cmdType:   c.cmdType,
		priority:  c.priority,
		context:   context,
		createdAt: c.createdAt,
	}
}

// WithPriority returns a copy of the command with a new priority
func (c *BaseCommand) WithPriority(priority Priority) *BaseCommand {
	return &BaseCommand{
		id:        c.id,
		cmdType:   c.cmdType,
		priority:  priority,
		context:   c.context,
		createdAt: c.createdAt,
	}
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	ID       string        `json:"id"`
	Type     CommandType    `json:"type"`
	Result   interface{}   `json:"result,omitempty"`
	Error    error         `json:"error,omitempty"`
	Attempts int           `json:"attempts"`
	Duration time.Duration `json:"duration"`
	Context  OperationContext `json:"context"`
}

// NewCommandResult creates a new command result
func NewCommandResult(cmd Command, result interface{}, err error, attempts int, duration time.Duration) *CommandResult {
	return &CommandResult{
		ID:       cmd.ID(),
		Type:     cmd.Type(),
		Result:   result,
		Error:    err,
		Attempts: attempts,
		Duration: duration,
		Context:  cmd.Context(),
	}
}

// Message implements the tea.Msg interface
func (r *CommandResult) Message() tea.Msg {
	return r
}

// IsSuccess returns true if the command completed successfully
func (r *CommandResult) IsSuccess() bool {
	return r.Error == nil
}

// IsFailure returns true if the command failed
func (r *CommandResult) IsFailure() bool {
	return r.Error != nil
}

// String returns a string representation of the result
func (r *CommandResult) String() string {
	if r.IsSuccess() {
		return fmt.Sprintf("CommandResult{id=%s, success, attempts=%d, duration=%v}", r.ID, r.Attempts, r.Duration)
	}
	return fmt.Sprintf("CommandResult{id=%s, error=%v, attempts=%d, duration=%v}", r.ID, r.Error, r.Attempts, r.Duration)
}