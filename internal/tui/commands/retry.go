package commands

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ResultMessage represents a message that contains result information
type ResultMessage interface {
	IsSuccess() bool
	IsFailure() bool
	GetResult() interface{}
	GetError() error
}

// RetryWrapper wraps a command and adds retry logic with exponential backoff
type RetryWrapper struct {
	*BaseCommand
	wrappedCommand Command
	retryPolicy    *RetryPolicy
	attemptCount   int
	lastError      error
	jitter         bool
}

// RetryResultMsg represents the result of a retry operation
type RetryResultMsg struct {
	ID            string             `json:"id"`
	Type          CommandType        `json:"type"`
	Result        interface{}        `json:"result,omitempty"`
	Error         error              `json:"error,omitempty"`
	Context       OperationContext   `json:"context"`
	Duration      time.Duration      `json:"duration"`
	AttemptCount  int                `json:"attemptCount"`
	TotalDuration time.Duration      `json:"totalDuration"`
	OriginalError error              `json:"originalError,omitempty"`
	RetryAttempts []RetryAttempt     `json:"retryAttempts,omitempty"`
}

// RetryAttempt represents a single retry attempt
type RetryAttempt struct {
	AttemptNumber int           `json:"attemptNumber"`
	Duration      time.Duration `json:"duration"`
	Error         error         `json:"error,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
}

// NewRetryWrapper creates a new retry wrapper for a command
func NewRetryWrapper(wrappedCommand Command, retryPolicy *RetryPolicy, jitter bool) *RetryWrapper {
	context := wrappedCommand.Context()
	if retryPolicy != nil {
		context.RetryPolicy = *retryPolicy
	}

	return &RetryWrapper{
		BaseCommand:    NewBaseCommand(wrappedCommand.ID(), wrappedCommand.Type(), wrappedCommand.Priority(), context),
		wrappedCommand: wrappedCommand,
		retryPolicy:    retryPolicy,
		attemptCount:   0,
		jitter:         jitter,
	}
}

// Run executes the wrapped command with retry logic
func (r *RetryWrapper) Run() tea.Msg {
	start := time.Now()
	var retryAttempts []RetryAttempt

	for r.attemptCount = 0; r.attemptCount < r.retryPolicy.MaxAttempts; r.attemptCount++ {
		attemptStart := time.Now()

		// Execute the wrapped command
		result := r.wrappedCommand.Run()
		attemptDuration := time.Since(attemptStart)

		// Record this attempt
		retryAttempts = append(retryAttempts, RetryAttempt{
			AttemptNumber: r.attemptCount + 1,
			Duration:      attemptDuration,
			Timestamp:     attemptStart,
		})

		// Check if the result is a success
		if msg, ok := result.(ResultMessage); ok {
			if msg.IsSuccess() {
				return RetryResultMsg{
					ID:            r.ID(),
					Type:          r.Type(),
					Result:        msg.GetResult(),
					Error:         nil,
					Context:       r.Context(),
					Duration:      attemptDuration,
					AttemptCount:  r.attemptCount + 1,
					TotalDuration: time.Since(start),
					RetryAttempts: retryAttempts,
				}
			} else {
				r.lastError = msg.GetError()
				retryAttempts[len(retryAttempts)-1].Error = r.lastError
			}
		} else {
			// For non-standard messages, assume success if no error
			if err := extractErrorFromMsg(result); err != nil {
				r.lastError = err
				retryAttempts[len(retryAttempts)-1].Error = err
			} else {
				return RetryResultMsg{
					ID:            r.ID(),
					Type:          r.Type(),
					Result:        result,
					Error:         nil,
					Context:       r.Context(),
					Duration:      attemptDuration,
					AttemptCount:  r.attemptCount + 1,
					TotalDuration: time.Since(start),
					RetryAttempts: retryAttempts,
				}
			}
		}

		// Check if we should retry
		if !r.shouldRetry(r.lastError, r.attemptCount) {
			break
		}

		// Wait before next attempt with exponential backoff
		if r.attemptCount < r.retryPolicy.MaxAttempts-1 {
			waitTime := r.calculateBackoff(r.attemptCount)
			time.Sleep(waitTime)
		}
	}

	// All attempts failed
	return RetryResultMsg{
		ID:            r.ID(),
		Type:          r.Type(),
		Result:        nil,
		Error:         fmt.Errorf("all %d retry attempts failed, last error: %w", r.attemptCount, r.lastError),
		Context:       r.Context(),
		Duration:      0,
		AttemptCount:  r.attemptCount,
		TotalDuration: time.Since(start),
		OriginalError: r.lastError,
		RetryAttempts: retryAttempts,
	}
}

// shouldRetry determines if the command should be retried based on the error and attempt count
func (r *RetryWrapper) shouldRetry(err error, attemptCount int) bool {
	if attemptCount >= r.retryPolicy.MaxAttempts-1 {
		return false
	}

	if err == nil {
		return false
	}

	// Check if error is in the list of non-retryable errors
	for _, retryableError := range r.retryPolicy.RetryableErrors {
		if err.Error() == retryableError.Error() {
			return true
		}
	}

	// Default: retry all errors unless MaxAttempts exceeded
	return true
}

// calculateBackoff calculates the exponential backoff delay
func (r *RetryWrapper) calculateBackoff(attemptCount int) time.Duration {
	baseDelay := float64(r.retryPolicy.InitialDelay)
	maxDelay := float64(r.retryPolicy.MaxDelay)
	backoffFactor := float64(r.retryPolicy.BackoffFactor)

	// Calculate exponential backoff: baseDelay * backoffFactor^attemptCount
	delay := baseDelay * math.Pow(backoffFactor, float64(attemptCount))

	// Apply maximum delay limit
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter if enabled
	if r.jitter {
		jitterRange := delay * 0.1 // 10% jitter
		delay += (rand.Float64()*2 - 1) * jitterRange
	}

	return time.Duration(delay)
}

// WrappedCommand returns the original wrapped command
func (r *RetryWrapper) WrappedCommand() Command {
	return r.wrappedCommand
}

// AttemptCount returns the number of attempts made
func (r *RetryWrapper) AttemptCount() int {
	return r.attemptCount
}

// LastError returns the last error encountered
func (r *RetryWrapper) LastError() error {
	return r.lastError
}

// Factory functions for common retry patterns

// NewRetryableIOCommand creates a retryable I/O command with default policy
func NewRetryableIOCommand(command Command) *RetryWrapper {
	policy := &RetryPolicy{
		MaxAttempts:    3,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       5 * time.Second,
		BackoffFactor:  2.0,
		RetryableErrors: []error{},
	}
	return NewRetryWrapper(command, policy, true)
}

// NewRetryableNetworkCommand creates a retryable network command with default policy
func NewRetryableNetworkCommand(command Command) *RetryWrapper {
	policy := &RetryPolicy{
		MaxAttempts:    5,
		InitialDelay:   200 * time.Millisecond,
		MaxDelay:       30 * time.Second,
		BackoffFactor:  1.5,
		RetryableErrors: []error{},
	}
	return NewRetryWrapper(command, policy, true)
}

// NewRetryableProcessCommand creates a retryable process command with default policy
func NewRetryableProcessCommand(command Command) *RetryWrapper {
	policy := &RetryPolicy{
		MaxAttempts:    2,
		InitialDelay:   500 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		BackoffFactor:  2.0,
		RetryableErrors: []error{},
	}
	return NewRetryWrapper(command, policy, false)
}

// NewRetryableSystemCommand creates a retryable system command with default policy
func NewRetryableSystemCommand(command Command) *RetryWrapper {
	policy := &RetryPolicy{
		MaxAttempts:    3,
		InitialDelay:   50 * time.Millisecond,
		MaxDelay:       1 * time.Second,
		BackoffFactor:  1.5,
		RetryableErrors: []error{}, // Most system operations are retryable
	}
	return NewRetryWrapper(command, policy, true)
}

// Custom retry predicates (for future use when RetryPolicy supports them)

// RetryOnTimeout returns true only for timeout errors
func RetryOnTimeout(err error, attemptCount int) bool {
	if err == nil {
		return false
	}
	return containsIgnoreCase(err.Error(), "timeout") ||
	       containsIgnoreCase(err.Error(), "deadline exceeded")
}

// RetryOnConnectionError returns true only for connection-related errors
func RetryOnConnectionError(err error, attemptCount int) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return containsIgnoreCase(errorStr, "connection refused") ||
	       containsIgnoreCase(errorStr, "connection reset") ||
	       containsIgnoreCase(errorStr, "network unreachable") ||
	       containsIgnoreCase(errorStr, "temporary failure")
}

// RetryOnTemporaryError returns true only for temporary errors
func RetryOnTemporaryError(err error, attemptCount int) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return containsIgnoreCase(errorStr, "temporary") ||
	       containsIgnoreCase(errorStr, "transient") ||
	       containsIgnoreCase(errorStr, "try again")
}

// RetryPredicate defines a function that determines if an operation should be retried
type RetryPredicate func(err error, attemptCount int) bool

// Utility functions

func extractErrorFromMsg(msg tea.Msg) error {
	if resultMsg, ok := msg.(ResultMessage); ok {
		return resultMsg.GetError()
	}
	return nil
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
	       (s == substr ||
	        (len(s) > len(substr) &&
	         (s[:len(substr)] == substr ||
	          s[len(s)-len(substr):] == substr ||
	          containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// Message interface implementation

func (m RetryResultMsg) Message() tea.Msg {
	return m
}

// IsSuccess returns true if the retry operation completed successfully
func (m RetryResultMsg) IsSuccess() bool {
	return m.Error == nil
}

// IsFailure returns true if the retry operation failed
func (m RetryResultMsg) IsFailure() bool {
	return m.Error != nil
}

// GetResult returns the result of the operation
func (m RetryResultMsg) GetResult() interface{} {
	return m.Result
}

// GetError returns the error of the operation
func (m RetryResultMsg) GetError() error {
	return m.Error
}

// String returns a string representation of the retry result
func (m RetryResultMsg) String() string {
	if m.IsSuccess() {
		return fmt.Sprintf("RetryResult{id=%s, attempts=%d, totalDuration=%v}",
			m.ID, m.AttemptCount, m.TotalDuration)
	}
	return fmt.Sprintf("RetryResult{id=%s, attempts=%d, error=%v, totalDuration=%v}",
		m.ID, m.AttemptCount, m.Error, m.TotalDuration)
}

// Retry statistics

type RetryStats struct {
	TotalCommands    int           `json:"totalCommands"`
	RetriedCommands  int           `json:"retriedCommands"`
	TotalAttempts    int           `json:"totalAttempts"`
	AverageAttempts  float64       `json:"averageAttempts"`
	SuccessRate      float64       `json:"successRate"`
	TotalRetryTime   time.Duration `json:"totalRetryTime"`
	AverageRetryTime time.Duration `json:"averageRetryTime"`
}

// CalculateRetryStats calculates retry statistics from a slice of retry results
func CalculateRetryStats(results []RetryResultMsg) RetryStats {
	if len(results) == 0 {
		return RetryStats{}
	}

	stats := RetryStats{
		TotalCommands: len(results),
	}

	var totalAttempts int
	var totalRetryTime time.Duration
	var successCount int

	for _, result := range results {
		totalAttempts += result.AttemptCount
		totalRetryTime += result.TotalDuration

		if result.IsSuccess() {
			successCount++
		}

		if result.AttemptCount > 1 {
			stats.RetriedCommands++
		}
	}

	stats.TotalAttempts = totalAttempts
	stats.AverageAttempts = float64(totalAttempts) / float64(len(results))
	stats.SuccessRate = float64(successCount) / float64(len(results))
	stats.TotalRetryTime = totalRetryTime
	stats.AverageRetryTime = totalRetryTime / time.Duration(len(results))

	return stats
}

// Advanced retry patterns

// NewExponentialBackoffWrapper creates a retry wrapper with exponential backoff
func NewExponentialBackoffWrapper(command Command, maxAttempts int, initialDelay, maxDelay time.Duration) *RetryWrapper {
	policy := &RetryPolicy{
		MaxAttempts:   maxAttempts,
		InitialDelay:  initialDelay,
		MaxDelay:      maxDelay,
		BackoffFactor: 2.0,
	}
	return NewRetryWrapper(command, policy, true)
}

// NewLinearBackoffWrapper creates a retry wrapper with linear backoff
func NewLinearBackoffWrapper(command Command, maxAttempts int, delay time.Duration) *RetryWrapper {
	policy := &RetryPolicy{
		MaxAttempts:   maxAttempts,
		InitialDelay:  delay,
		MaxDelay:      delay * time.Duration(maxAttempts),
		BackoffFactor: 1.0,
	}
	return NewRetryWrapper(command, policy, false)
}

// NewFixedDelayWrapper creates a retry wrapper with fixed delay between attempts
func NewFixedDelayWrapper(command Command, maxAttempts int, delay time.Duration) *RetryWrapper {
	policy := &RetryPolicy{
		MaxAttempts:   maxAttempts,
		InitialDelay:  delay,
		MaxDelay:      delay,
		BackoffFactor: 1.0,
	}
	return NewRetryWrapper(command, policy, false)
}

// NewCustomRetryWrapper creates a retry wrapper with custom retry predicate (for future use)
func NewCustomRetryWrapper(command Command, maxAttempts int, initialDelay time.Duration, retryPredicate RetryPredicate) *RetryWrapper {
	policy := &RetryPolicy{
		MaxAttempts:   maxAttempts,
		InitialDelay:  initialDelay,
		MaxDelay:      initialDelay * time.Duration(maxAttempts),
		BackoffFactor: 1.5,
	}
	return NewRetryWrapper(command, policy, true)
}