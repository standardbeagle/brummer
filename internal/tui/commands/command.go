package commands

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Command represents an asynchronous operation that can be executed by Bubble Tea
type Command interface {
	// Run executes the command and returns a message with the result
	Run() tea.Msg

	// ID returns the unique identifier for this command
	ID() string

	// Type returns the category of this command
	Type() CommandType

	// Priority returns the execution priority (0 = highest)
	Priority() Priority

	// Context returns the operation context for ordering and tracking
	Context() OperationContext

	// CreatedAt returns when this command was created
	CreatedAt() time.Time
}

// OperationContext provides metadata for command execution and result handling
type OperationContext struct {
	// OperationType categorizes the operation (UI, Data, System)
	OperationType string `json:"operationType"`

	// Version is the state version when operation was initiated
	Version uint64 `json:"version"`

	// Dependencies are command IDs that must complete first
	Dependencies []string `json:"dependencies,omitempty"`

	// Strategy defines how to handle out-of-order completion
	Strategy OrderingStrategy `json:"strategy"`

	// Timeout is the maximum execution time
	Timeout time.Duration `json:"timeout"`

	// RetryPolicy defines how to handle failures
	RetryPolicy RetryPolicy `json:"retryPolicy"`
}

// DefaultOperationContext returns a standard context for most operations
func DefaultOperationContext(version uint64) OperationContext {
	return OperationContext{
		OperationType: "Data",
		Version:       version,
		Strategy:      OrderingLastWriteWins,
		Timeout:       30 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}
}

// UIOperationContext returns a context for UI operations
func UIOperationContext(version uint64) OperationContext {
	return OperationContext{
		OperationType: "UI",
		Version:       version,
		Strategy:      OrderingQueueSequential,
		Timeout:       5 * time.Second,
		RetryPolicy:   RetryPolicy{MaxAttempts: 1}, // UI operations typically not retried
	}
}

// SystemOperationContext returns a context for system operations
func SystemOperationContext(version uint64) OperationContext {
	return OperationContext{
		OperationType: "System",
		Version:       version,
		Strategy:      OrderingVersionTracking,
		Timeout:       60 * time.Second,
		RetryPolicy:   DefaultRetryPolicy(),
	}
}