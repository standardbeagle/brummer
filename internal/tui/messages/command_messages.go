package messages

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// BaseMessage provides common functionality for all command messages
type BaseMessage struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// Message interface for all command-related messages
type Message interface {
	tea.Msg
	GetID() string
	GetType() string
	GetTimestamp() time.Time
}

// GetID returns the message ID
func (m BaseMessage) GetID() string {
	return m.ID
}

// GetType returns the message type
func (m BaseMessage) GetType() string {
	return m.Type
}

// GetTimestamp returns when the message was created
func (m BaseMessage) GetTimestamp() time.Time {
	return m.Timestamp
}

// CommandStartedMessage is sent when a command begins execution
type CommandStartedMessage struct {
	BaseMessage
	CommandType types.CommandType    `json:"commandType"`
	Context     types.OperationContext   `json:"context"`
}

// NewCommandStartedMessage creates a new command started message
func NewCommandStartedMessage(id string, commandType types.CommandType, context types.OperationContext) *CommandStartedMessage {
	return &CommandStartedMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "CommandStarted",
			Timestamp: time.Now(),
		},
		CommandType: commandType,
		Context:     context,
	}
}

// CommandCompletedMessage is sent when a command completes successfully
type CommandCompletedMessage struct {
	BaseMessage
	CommandType types.CommandType    `json:"commandType"`
	Result      interface{}          `json:"result"`
	Duration    time.Duration        `json:"duration"`
	Context     types.OperationContext   `json:"context"`
}

// NewCommandCompletedMessage creates a new command completed message
func NewCommandCompletedMessage(id string, commandType types.CommandType, result interface{}, duration time.Duration, context types.OperationContext) *CommandCompletedMessage {
	return &CommandCompletedMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "CommandCompleted",
			Timestamp: time.Now(),
		},
		CommandType: commandType,
		Result:      result,
		Duration:    duration,
		Context:     context,
	}
}

// CommandFailedMessage is sent when a command fails
type CommandFailedMessage struct {
	BaseMessage
	CommandType types.CommandType    `json:"commandType"`
	Error       error                `json:"error"`
	Duration    time.Duration        `json:"duration"`
	Context     types.OperationContext   `json:"context"`
	Retryable   bool                 `json:"retryable"`
}

// NewCommandFailedMessage creates a new command failed message
func NewCommandFailedMessage(id string, commandType types.CommandType, err error, duration time.Duration, context types.OperationContext, retryable bool) *CommandFailedMessage {
	return &CommandFailedMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "CommandFailed",
			Timestamp: time.Now(),
		},
		CommandType: commandType,
		Error:       err,
		Duration:    duration,
		Context:     context,
		Retryable:   retryable,
	}
}

// CommandCancelledMessage is sent when a command is cancelled
type CommandCancelledMessage struct {
	BaseMessage
	CommandType types.CommandType    `json:"commandType"`
	Reason      string               `json:"reason"`
	Duration    time.Duration        `json:"duration"`
	Context     types.OperationContext   `json:"context"`
}

// NewCommandCancelledMessage creates a new command cancelled message
func NewCommandCancelledMessage(id string, commandType types.CommandType, reason string, duration time.Duration, context types.OperationContext) *CommandCancelledMessage {
	return &CommandCancelledMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "CommandCancelled",
			Timestamp: time.Now(),
		},
		CommandType: commandType,
		Reason:      reason,
		Duration:    duration,
		Context:     context,
	}
}

// CommandProgressMessage is sent for long-running commands to report progress
type CommandProgressMessage struct {
	BaseMessage
	CommandType types.CommandType    `json:"commandType"`
	Progress    int                  `json:"progress"` // 0-100
	Message     string               `json:"message"`
	Context     types.OperationContext   `json:"context"`
}

// NewCommandProgressMessage creates a new command progress message
func NewCommandProgressMessage(id string, commandType types.CommandType, progress int, message string, context types.OperationContext) *CommandProgressMessage {
	return &CommandProgressMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "CommandProgress",
			Timestamp: time.Now(),
		},
		CommandType: commandType,
		Progress:    progress,
		Message:     message,
		Context:     context,
	}
}

// BatchCommandStartedMessage is sent when a batch command begins execution
type BatchCommandStartedMessage struct {
	BaseMessage
	CommandCount int                      `json:"commandCount"`
	Mode         types.BatchExecutionMode `json:"mode"`
	Context      types.OperationContext  `json:"context"`
}

// NewBatchCommandStartedMessage creates a new batch command started message
func NewBatchCommandStartedMessage(id string, commandCount int, mode types.BatchExecutionMode, context types.OperationContext) *BatchCommandStartedMessage {
	return &BatchCommandStartedMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "BatchCommandStarted",
			Timestamp: time.Now(),
		},
		CommandCount: commandCount,
		Mode:         mode,
		Context:      context,
	}
}

// BatchCommandCompletedMessage is sent when a batch command completes
type BatchCommandCompletedMessage struct {
	BaseMessage
	Results      []types.BatchCommandResult `json:"results"`
	TotalDuration time.Duration                 `json:"totalDuration"`
	SuccessCount int                            `json:"successCount"`
	FailureCount int                            `json:"failureCount"`
	Context      types.OperationContext     `json:"context"`
}

// NewBatchCommandCompletedMessage creates a new batch command completed message
func NewBatchCommandCompletedMessage(id string, results []types.BatchCommandResult, totalDuration time.Duration, context types.OperationContext) *BatchCommandCompletedMessage {
	successCount := 0
	failureCount := 0
	for _, result := range results {
		if result.Error != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	return &BatchCommandCompletedMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "BatchCommandCompleted",
			Timestamp: time.Now(),
		},
		Results:      results,
		TotalDuration: totalDuration,
		SuccessCount: successCount,
		FailureCount: failureCount,
		Context:      context,
	}
}

// RetryAttemptMessage is sent when a command is being retried
type RetryAttemptMessage struct {
	BaseMessage
	OriginalCommandID string               `json:"originalCommandId"`
	AttemptNumber     int                  `json:"attemptNumber"`
	MaxAttempts       int                  `json:"maxAttempts"`
	OriginalError     error                `json:"originalError"`
	BackoffDelay      time.Duration        `json:"backoffDelay"`
	Context           types.OperationContext `json:"context"`
}

// NewRetryAttemptMessage creates a new retry attempt message
func NewRetryAttemptMessage(id string, originalCommandID string, attemptNumber, maxAttempts int, originalError error, backoffDelay time.Duration, context types.OperationContext) *RetryAttemptMessage {
	return &RetryAttemptMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "RetryAttempt",
			Timestamp: time.Now(),
		},
		OriginalCommandID: originalCommandID,
		AttemptNumber:     attemptNumber,
		MaxAttempts:       maxAttempts,
		OriginalError:     originalError,
		BackoffDelay:      backoffDelay,
		Context:           context,
	}
}

// StateUpdateMessage is sent when application state changes
type StateUpdateMessage struct {
	BaseMessage
	StateType   string    `json:"stateType"`
	OldVersion  uint64    `json:"oldVersion"`
	NewVersion  uint64    `json:"newVersion"`
	ChangeType  string    `json:"changeType"`
	Description string    `json:"description"`
	Context     types.OperationContext   `json:"context"`
}

// NewStateUpdateMessage creates a new state update message
func NewStateUpdateMessage(id string, stateType string, oldVersion, newVersion uint64, changeType, description string, context types.OperationContext) *StateUpdateMessage {
	return &StateUpdateMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "StateUpdate",
			Timestamp: time.Now(),
		},
		StateType:   stateType,
		OldVersion:  oldVersion,
		NewVersion:  newVersion,
		ChangeType:  changeType,
		Description: description,
		Context:     context,
	}
}

// OperationTimeoutMessage is sent when a command times out
type OperationTimeoutMessage struct {
	BaseMessage
	CommandType     types.CommandType `json:"commandType"`
	TimeoutDuration time.Duration        `json:"timeoutDuration"`
	ActualDuration  time.Duration        `json:"actualDuration"`
	Context         types.OperationContext `json:"context"`
}

// NewOperationTimeoutMessage creates a new operation timeout message
func NewOperationTimeoutMessage(id string, commandType types.CommandType, timeoutDuration, actualDuration time.Duration, context types.OperationContext) *OperationTimeoutMessage {
	return &OperationTimeoutMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "OperationTimeout",
			Timestamp: time.Now(),
		},
		CommandType:     commandType,
		TimeoutDuration: timeoutDuration,
		ActualDuration:  actualDuration,
		Context:         context,
	}
}

// ResourceCleanupMessage is sent when resources need cleanup
type ResourceCleanupMessage struct {
	BaseMessage
	ResourceType string               `json:"resourceType"`
	ResourceID   string               `json:"resourceId"`
	Reason       string               `json:"reason"`
	Context      types.OperationContext `json:"context"`
}

// NewResourceCleanupMessage creates a new resource cleanup message
func NewResourceCleanupMessage(id string, resourceType, resourceID, reason string, context types.OperationContext) *ResourceCleanupMessage {
	return &ResourceCleanupMessage{
		BaseMessage: BaseMessage{
			ID:        id,
			Type:      "ResourceCleanup",
			Timestamp: time.Now(),
		},
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Reason:       reason,
		Context:      context,
	}
}

// Message utilities

// MessageFilter provides filtering functionality for message streams
type MessageFilter struct {
	allowedTypes []string
	allowedIDs   []string
	startTime    time.Time
	endTime      time.Time
}

// NewMessageFilter creates a new message filter
func NewMessageFilter() *MessageFilter {
	return &MessageFilter{
		allowedTypes: []string{},
		allowedIDs:   []string{},
		startTime:    time.Time{},
		endTime:      time.Time{},
	}
}

// WithTypes adds type filtering to the filter
func (f *MessageFilter) WithTypes(types ...string) *MessageFilter {
	f.allowedTypes = append(f.allowedTypes, types...)
	return f
}

// WithIDs adds ID filtering to the filter
func (f *MessageFilter) WithIDs(ids ...string) *MessageFilter {
	f.allowedIDs = append(f.allowedIDs, ids...)
	return f
}

// WithTimeRange adds time range filtering to the filter
func (f *MessageFilter) WithTimeRange(start, end time.Time) *MessageFilter {
	f.startTime = start
	f.endTime = end
	return f
}

// ShouldInclude checks if a message should be included based on filter criteria
func (f *MessageFilter) ShouldInclude(msg Message) bool {
	// Check type filter
	if len(f.allowedTypes) > 0 {
		found := false
		for _, allowedType := range f.allowedTypes {
			if msg.GetType() == allowedType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check ID filter
	if len(f.allowedIDs) > 0 {
		found := false
		for _, allowedID := range f.allowedIDs {
			if msg.GetID() == allowedID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check time range filter
	if !f.startTime.IsZero() && msg.GetTimestamp().Before(f.startTime) {
		return false
	}
	if !f.endTime.IsZero() && msg.GetTimestamp().After(f.endTime) {
		return false
	}

	return true
}

// MessageStats provides statistics about a collection of messages
type MessageStats struct {
	TotalMessages    int                    `json:"totalMessages"`
	MessagesByType   map[string]int         `json:"messagesByType"`
	MessagesByHour   map[int]int            `json:"messagesByHour"`
	AverageInterval  time.Duration          `json:"averageInterval"`
	FirstMessageTime time.Time             `json:"firstMessageTime"`
	LastMessageTime  time.Time             `json:"lastMessageTime"`
	CommandTypes     map[types.CommandType]int `json:"commandTypes"`
}

// CalculateMessageStats calculates statistics from a slice of messages
func CalculateMessageStats(messages []Message) MessageStats {
	if len(messages) == 0 {
		return MessageStats{
			MessagesByType:  make(map[string]int),
			MessagesByHour:  make(map[int]int),
			CommandTypes:    make(map[types.CommandType]int),
		}
	}

	stats := MessageStats{
		TotalMessages:   len(messages),
		MessagesByType:  make(map[string]int),
		MessagesByHour:  make(map[int]int),
		FirstMessageTime: messages[0].GetTimestamp(),
		LastMessageTime:  messages[len(messages)-1].GetTimestamp(),
		CommandTypes:    make(map[types.CommandType]int),
	}

	var totalInterval time.Duration
	var intervalCount int

	for i, msg := range messages {
		// Count by type
		stats.MessagesByType[msg.GetType()]++

		// Count by hour
		hour := msg.GetTimestamp().Hour()
		stats.MessagesByHour[hour]++

		// Calculate intervals
		if i > 0 {
			interval := msg.GetTimestamp().Sub(messages[i-1].GetTimestamp())
			totalInterval += interval
			intervalCount++
		}

		// Count command types if applicable
		switch m := msg.(type) {
		case *CommandStartedMessage:
			stats.CommandTypes[m.CommandType]++
		case *CommandCompletedMessage:
			stats.CommandTypes[m.CommandType]++
		case *CommandFailedMessage:
			stats.CommandTypes[m.CommandType]++
		case *CommandCancelledMessage:
			stats.CommandTypes[m.CommandType]++
		case *CommandProgressMessage:
			stats.CommandTypes[m.CommandType]++
		}
	}

	if intervalCount > 0 {
		stats.AverageInterval = totalInterval / time.Duration(intervalCount)
	}

	return stats
}

// MessageCollector collects and stores messages for analysis
type MessageCollector struct {
	messages []Message
	maxSize  int
}

// NewMessageCollector creates a new message collector
func NewMessageCollector(maxSize int) *MessageCollector {
	return &MessageCollector{
		messages: make([]Message, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Add adds a message to the collector
func (c *MessageCollector) Add(msg Message) {
	if len(c.messages) >= c.maxSize {
		// Remove oldest message if at capacity
		c.messages = c.messages[1:]
	}
	c.messages = append(c.messages, msg)
}

// GetMessages returns all collected messages
func (c *MessageCollector) GetMessages() []Message {
	result := make([]Message, len(c.messages))
	copy(result, c.messages)
	return result
}

// GetFilteredMessages returns filtered messages
func (c *MessageCollector) GetFilteredMessages(filter *MessageFilter) []Message {
	if filter == nil {
		return c.GetMessages()
	}

	var result []Message
	for _, msg := range c.messages {
		if filter.ShouldInclude(msg) {
			result = append(result, msg)
		}
	}
	return result
}

// Clear removes all messages from the collector
func (c *MessageCollector) Clear() {
	c.messages = c.messages[:0]
}

// Size returns the current number of messages
func (c *MessageCollector) Size() int {
	return len(c.messages)
}

// IsFull returns true if the collector is at capacity
func (c *MessageCollector) IsFull() bool {
	return len(c.messages) >= c.maxSize
}