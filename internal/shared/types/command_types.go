package types

import "time"

// CommandType represents different types of commands that can be executed
type CommandType int

const (
	CommandTypeIO CommandType = iota
	CommandTypeProcess
	CommandTypeNetwork
	CommandTypeSystem
	CommandTypeBatch
	CommandTypeRetry
)

// OperationPriority represents the priority of an operation
type OperationPriority int

const (
	PriorityLow OperationPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// OrderingStrategy defines how operations should be ordered
type OrderingStrategy int

const (
	OrderingFIFO OrderingStrategy = iota
	OrderingLIFO
	OrderingPriority
	OrderingParallel
)

// OperationContext provides context for command execution
type OperationContext struct {
	RequestID     string                 `json:"requestId"`
	UserID        string                 `json:"userId,omitempty"`
	SessionID     string                 `json:"sessionId,omitempty"`
	TraceID       string                 `json:"traceId,omitempty"`
	OperationID   string                 `json:"operationId,omitempty"`
	Priority      OperationPriority      `json:"priority"`
	Timeout       time.Duration          `json:"timeout"`
	RetryCount    int                    `json:"retryCount"`
	MaxRetries    int                    `json:"maxRetries"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	DryRun        bool                   `json:"dryRun"`
	BatchID       string                 `json:"batchId,omitempty"`
	ParentBatchID string                 `json:"parentBatchId,omitempty"`
}

// BatchExecutionMode represents different batch execution modes
type BatchExecutionMode int

const (
	BatchModeSequential BatchExecutionMode = iota
	BatchModeParallel
	BatchModeRace
	BatchModeFirstSuccess
	BatchModeAllSuccess
)

// BatchCommandResult represents the result of a batch command
type BatchCommandResult struct {
	ID        string        `json:"id"`
	Success   bool          `json:"success"`
	Result    interface{}   `json:"result,omitempty"`
	Error     error         `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Command   string        `json:"command"`
}