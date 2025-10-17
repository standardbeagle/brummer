package commands

import "time"

// CommandType represents the category of command
type CommandType int

const (
	CommandTypeIO CommandType = iota
	CommandTypeProcess
	CommandTypeNetwork
	CommandTypeSystem
)

// Priority defines command execution priority (0 = highest)
type Priority int

const (
	PriorityCritical Priority = 0
	PriorityHigh     Priority = 1
	PriorityMedium   Priority = 5
	PriorityLow      Priority = 8
	PriorityBackground Priority = 10
)

// OrderingStrategy defines how to handle out-of-order command completion
type OrderingStrategy int

const (
	OrderingLastWriteWins OrderingStrategy = iota
	OrderingVersionTracking
	OrderingQueueSequential
	OrderingParallelMerge
)

// OperationStatus represents the current state of an operation
type OperationStatus int

const (
	OperationPending OperationStatus = iota
	OperationRunning
	OperationCompleted
	OperationFailed
	OperationCancelled
)

// MessageType represents the type of message from command execution
type MessageType int

const (
	MessageTypeResult MessageType = iota
	MessageTypeError
	MessageTypeProgress
	MessageTypeStateChange
)

// RetryPolicy defines how operations should be retried on failure
type RetryPolicy struct {
	MaxAttempts    int           `json:"maxAttempts"`
	InitialDelay   time.Duration `json:"initialDelay"`
	MaxDelay       time.Duration `json:"maxDelay"`
	BackoffFactor  float64       `json:"backoffFactor"`
	RetryableErrors []error       `json:"-"`
}

// DefaultRetryPolicy returns a standard retry policy for most operations
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:   3,
		InitialDelay:  1000 * time.Millisecond,
		MaxDelay:      10000 * time.Millisecond,
		BackoffFactor: 2.0,
	}
}

// NetworkRetryPolicy returns a more aggressive retry policy for network operations
func NetworkRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:   5,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      30000 * time.Millisecond,
		BackoffFactor: 1.5,
	}
}