package commands

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestRetryExponentialBackoff tests exponential backoff retry logic
// This test should FAIL initially before retry logic is implemented
func TestRetryExponentialBackoff(t *testing.T) {
	attempts := 0
	expectedAttempts := 3

	cmd := &RetryTestCommand{
		CmdID:        "retry-exponential-test",
		CmdType:      CommandTypeNetwork,
		CmdPriority:  PriorityHigh,
		CmdContext: OperationContext{
			OperationType: "Network",
			Version:       1,
			RetryPolicy: RetryPolicy{
				MaxAttempts:   expectedAttempts,
				InitialDelay:  100 * time.Millisecond,
				MaxDelay:      1000 * time.Millisecond,
				BackoffFactor: 2.0,
			},
		},
		CmdCreatedAt: time.Now(),
		shouldFail:   true,
		failUntil:    expectedAttempts - 1, // Fail first 2 attempts, succeed on 3rd
		attemptCount: &attempts,
	}

	start := time.Now()
	result := cmd.Run()
	duration := time.Since(start)

	// Should have attempted exactly the expected number of times
	if attempts != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attempts)
	}

	// Should have succeeded on final attempt
	if msg, ok := result.(RetryTestResultMsg); !ok {
		t.Fatalf("Expected RetryTestResultMsg, got %T", result)
	} else if msg.Error != nil {
		t.Errorf("Expected success after retries, got error: %v", msg.Error)
	}

	// Should have taken reasonable time with exponential backoff
	// 100ms + 200ms + 400ms = 700ms minimum
	expectedMinDuration := 700 * time.Millisecond
	if duration < expectedMinDuration {
		t.Errorf("Expected duration >= %v, got %v", expectedMinDuration, duration)
	}

	t.Logf("Retry test completed in %v after %d attempts", duration, attempts)
}

// TestRetryMaxAttemptsExceeded tests behavior when max attempts is exceeded
func TestRetryMaxAttemptsExceeded(t *testing.T) {
	attempts := 0
	maxAttempts := 3

	cmd := &RetryTestCommand{
		CmdID:        "retry-max-attempts-test",
		CmdType:      CommandTypeNetwork,
		CmdPriority:  PriorityMedium,
		CmdContext: OperationContext{
			OperationType: "Network",
			Version:       1,
			RetryPolicy: RetryPolicy{
				MaxAttempts:   maxAttempts,
				InitialDelay:  50 * time.Millisecond,
				MaxDelay:      500 * time.Millisecond,
				BackoffFactor: 2.0,
			},
		},
		CmdCreatedAt: time.Now(),
		shouldFail:   true, // Always fail
		failUntil:    999, // Never succeed
		attemptCount: &attempts,
	}

	start := time.Now()
	result := cmd.Run()
	duration := time.Since(start)

	// Should have attempted exactly max attempts
	if attempts != maxAttempts {
		t.Errorf("Expected %d attempts, got %d", maxAttempts, attempts)
	}

	// Should have failed with error
	msg, ok := result.(RetryTestResultMsg)
	if !ok {
		t.Fatalf("Expected RetryTestResultMsg, got %T", result)
	}
	if msg.Error == nil {
		t.Fatal("Expected final error after max attempts, got success")
	}

	// Verify error message indicates max attempts exceeded
	if !contains(msg.Error.Error(), "max attempts") {
		t.Errorf("Expected max attempts error, got: %v", msg.Error)
	}

	t.Logf("Max attempts test completed in %v after %d failed attempts", duration, attempts)
}

// TestRetryWithTransientError tests retry logic for transient errors
func TestRetryWithTransientError(t *testing.T) {
	attempts := 0

	cmd := &RetryTestCommand{
		CmdID:        "retry-transient-test",
		CmdType:      CommandTypeNetwork,
		CmdPriority:  PriorityMedium,
		CmdContext: OperationContext{
			OperationType: "Network",
			Version:       1,
			RetryPolicy: RetryPolicy{
				MaxAttempts:   3,
				InitialDelay:  10 * time.Millisecond,
				MaxDelay:      100 * time.Millisecond,
				BackoffFactor: 2.0,
				RetryableErrors: []error{&TransientError{}, &NetworkTimeoutError{}},
			},
		},
		CmdCreatedAt: time.Now(),
		shouldFail:   true,
		failUntil:    1, // Succeed on 2nd attempt
		errorType:    &TransientError{},
		attemptCount: &attempts,
	}

	result := cmd.Run()

	// Should have attempted 2 times
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	// Should have succeeded
	if msg, ok := result.(RetryTestResultMsg); !ok {
		t.Fatalf("Expected RetryTestResultMsg, got %T", result)
	} else if msg.Error != nil {
		t.Errorf("Expected success after retry, got error: %v", msg.Error)
	}
}

// TestRetryWithNonRetryableError tests that non-retryable errors are not retried
func TestRetryWithNonRetryableError(t *testing.T) {
	attempts := 0

	cmd := &RetryTestCommand{
		CmdID:        "retry-non-retryable-test",
		CmdType:      CommandTypeNetwork,
		CmdPriority:  PriorityMedium,
		CmdContext: OperationContext{
			OperationType: "Network",
			Version:       1,
			RetryPolicy: RetryPolicy{
				MaxAttempts:   3,
				InitialDelay:  10 * time.Millisecond,
				MaxDelay:      100 * time.Millisecond,
				BackoffFactor: 2.0,
				RetryableErrors: []error{&TransientError{}, &NetworkTimeoutError{}},
			},
		},
		CmdCreatedAt: time.Now(),
		shouldFail:   true,
		failUntil:    999, // Never succeed
		errorType:    &NonRetryableError{}, // Non-retryable error
		attemptCount: &attempts,
	}

	result := cmd.Run()

	// Should have attempted only once (no retry for non-retryable errors)
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}

	// Should have failed with original error
	msg, ok := result.(RetryTestResultMsg)
	if !ok {
		t.Fatalf("Expected RetryTestResultMsg, got %T", result)
	}
	if msg.Error == nil {
		t.Fatal("Expected non-retryable error, got success")
	}

	// Verify it's the right error type
	if !IsNonRetryableError(msg.Error) {
		t.Errorf("Expected non-retryable error, got: %T", msg.Error)
	}
}

// TestRetryCancellation tests retry cancellation via context
func TestRetryCancellation(t *testing.T) {
	attempts := 0

	cmd := &RetryTestCommand{
		CmdID:        "retry-cancellation-test",
		CmdType:      CommandTypeSystem,
		CmdPriority:  PriorityLow,
		CmdContext: OperationContext{
			OperationType: "System",
			Version:       1,
			Timeout:       200 * time.Millisecond, // Short timeout
			RetryPolicy: RetryPolicy{
				MaxAttempts:   10, // High, but should be cancelled by timeout
				InitialDelay:  50 * time.Millisecond,
				MaxDelay:      500 * time.Millisecond,
				BackoffFactor: 2.0,
			},
		},
		CmdCreatedAt: time.Now(),
		shouldFail:   true,
		failUntil:    999, // Never succeed
		attemptCount: &attempts,
	}

	start := time.Now()
	result := cmd.Run()
	duration := time.Since(start)

	// Should have been cancelled by timeout
	if duration > 300*time.Millisecond {
		t.Errorf("Expected cancellation around 200ms, took %v", duration)
	}

	// Should have failed with timeout/cancellation error
	if msg, ok := result.(RetryTestResultMsg); !ok {
		t.Fatalf("Expected RetryTestResultMsg, got %T", result)
	} else if msg.Error == nil {
		t.Fatal("Expected timeout error, got success")
	}

	// Should not have attempted all 10 times due to timeout
	if attempts >= 8 {
		t.Errorf("Too many attempts before timeout: %d", attempts)
	}

	t.Logf("Retry cancelled after %v with %d attempts", duration, attempts)
}

// Test types and mock implementations

// RetryTestCommand is a mock command for testing retry logic
type RetryTestCommand struct {
	CmdID        string
	CmdType      CommandType
	CmdPriority  Priority
	CmdContext   OperationContext
	CmdCreatedAt time.Time
	shouldFail   bool
	failUntil    int           // Fail until this attempt (1-based)
	errorType    error         // Type of error to return
	attemptCount *int          // Pointer to counter for tracking attempts
}

// RetryTestResultMsg represents a test result message
type RetryTestResultMsg struct {
	ID          string        `json:"id"`
	Result      interface{}   `json:"result,omitempty"`
	Error       error         `json:"error,omitempty"`
	Attempts    int           `json:"attempts"`
	Duration    time.Duration `json:"duration"`
}

// Implement the Command interface
func (c *RetryTestCommand) Run() tea.Msg {
	start := time.Now()
	*c.attemptCount++
	currentAttempt := *c.attemptCount

	var result interface{}
	var err error

	// Determine if this attempt should succeed
	if c.shouldFail && currentAttempt <= c.failUntil {
		if c.errorType != nil {
			err = c.errorType
		} else {
			err = fmt.Errorf("attempt %d failed", currentAttempt)
		}
	} else {
		result = fmt.Sprintf("success on attempt %d", currentAttempt)
	}

	duration := time.Since(start)

	return RetryTestResultMsg{
		ID:       c.CmdID,
		Result:   result,
		Error:    err,
		Attempts: currentAttempt,
		Duration: duration,
	}
}

func (c *RetryTestCommand) ID() string        { return c.CmdID }
func (c *RetryTestCommand) Type() CommandType { return c.CmdType }
func (c *RetryTestCommand) Priority() Priority { return c.CmdPriority }
func (c *RetryTestCommand) Context() OperationContext { return c.CmdContext }
func (c *RetryTestCommand) CreatedAt() time.Time { return c.CmdCreatedAt }

// Error types for testing
type TransientError struct{}
func (e *TransientError) Error() string { return "transient error" }

type NetworkTimeoutError struct{}
func (e *NetworkTimeoutError) Error() string { return "network timeout" }

type NonRetryableError struct{}
func (e *NonRetryableError) Error() string { return "non-retryable error" }

// Helper functions
func IsNonRetryableError(err error) bool {
	_, ok := err.(*NonRetryableError)
	return ok
}