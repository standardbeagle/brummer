package mcp

import (
	"math"
	"math/rand"
	"time"
)

// ExponentialBackoff implements exponential backoff with jitter for connection retries
type ExponentialBackoff struct {
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
	attemptCount int
}

// NewExponentialBackoff creates a new exponential backoff with sensible defaults
func NewExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}
}

// NextDelay calculates the next delay duration with exponential backoff
func (eb *ExponentialBackoff) NextDelay() time.Duration {
	// Calculate exponential delay
	delay := time.Duration(float64(eb.BaseDelay) * math.Pow(eb.Multiplier, float64(eb.attemptCount)))

	// Cap at maximum delay
	if delay > eb.MaxDelay {
		delay = eb.MaxDelay
	}

	// Add jitter to prevent thundering herd
	if eb.Jitter {
		jitterRange := float64(delay) * 0.1 // ±10% jitter
		jitter := time.Duration((rand.Float64()*2 - 1) * jitterRange)
		delay += jitter
	}

	// Ensure minimum positive delay
	if delay < eb.BaseDelay {
		delay = eb.BaseDelay
	}

	eb.attemptCount++
	return delay
}

// Reset resets the attempt count to start over
func (eb *ExponentialBackoff) Reset() {
	eb.attemptCount = 0
}

// GetAttemptCount returns the current attempt count
func (eb *ExponentialBackoff) GetAttemptCount() int {
	return eb.attemptCount
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for network connections
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	failureCount int
	lastFailTime time.Time
	state        CircuitBreakerState
}

// NewCircuitBreaker creates a new circuit breaker with the specified parameters
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

// DefaultCircuitBreaker creates a circuit breaker with sensible defaults
func DefaultCircuitBreaker() *CircuitBreaker {
	return NewCircuitBreaker(5, 60*time.Second)
}

// Call attempts to execute a function through the circuit breaker
func (cb *CircuitBreaker) Call(fn func() error) error {
	if !cb.AllowRequest() {
		return &CircuitBreakerError{
			State:        cb.state,
			FailureCount: cb.failureCount,
			LastFailTime: cb.lastFailTime,
		}
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// AllowRequest determines if a request should be allowed through
func (cb *CircuitBreaker) AllowRequest() bool {
	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.failureCount = 0
	cb.state = CircuitClosed
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.failureCount++
	cb.lastFailTime = time.Now()

	if cb.failureCount >= cb.maxFailures {
		cb.state = CircuitOpen
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"state":        cb.state.String(),
		"failureCount": cb.failureCount,
		"maxFailures":  cb.maxFailures,
		"lastFailTime": cb.lastFailTime,
		"resetTimeout": cb.resetTimeout,
	}
}

// String returns a string representation of the circuit breaker state
func (state CircuitBreakerState) String() string {
	switch state {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerError is returned when a circuit breaker blocks a request
type CircuitBreakerError struct {
	State        CircuitBreakerState
	FailureCount int
	LastFailTime time.Time
}

// Error implements the error interface
func (e *CircuitBreakerError) Error() string {
	switch e.State {
	case CircuitOpen:
		return "circuit breaker is open - too many failures"
	case CircuitHalfOpen:
		return "circuit breaker is half-open - testing recovery"
	default:
		return "circuit breaker blocked request"
	}
}

// IsCircuitBreakerError checks if an error is a circuit breaker error
func IsCircuitBreakerError(err error) bool {
	_, ok := err.(*CircuitBreakerError)
	return ok
}

// RetryPolicy combines exponential backoff with circuit breaker for robust retry logic
type RetryPolicy struct {
	backoff        *ExponentialBackoff
	circuitBreaker *CircuitBreaker
	maxRetries     int
}

// NewRetryPolicy creates a new retry policy with exponential backoff and circuit breaker
func NewRetryPolicy(maxRetries int) *RetryPolicy {
	return &RetryPolicy{
		backoff:        NewExponentialBackoff(),
		circuitBreaker: DefaultCircuitBreaker(),
		maxRetries:     maxRetries,
	}
}

// ExecuteWithRetry executes a function with retry logic, exponential backoff, and circuit breaker
func (rp *RetryPolicy) ExecuteWithRetry(fn func() error) error {
	for attempt := 0; attempt < rp.maxRetries; attempt++ {
		err := rp.circuitBreaker.Call(fn)

		if err == nil {
			// Success - reset backoff for next time
			rp.backoff.Reset()
			return nil
		}

		// Check if this is a circuit breaker error
		if IsCircuitBreakerError(err) {
			return err // Don't retry if circuit breaker is open
		}

		// If this is the last attempt, return the error
		if attempt == rp.maxRetries-1 {
			return err
		}

		// Wait before retrying
		delay := rp.backoff.NextDelay()
		time.Sleep(delay)
	}

	return nil // Should never reach here
}

// Reset resets the retry policy state
func (rp *RetryPolicy) Reset() {
	rp.backoff.Reset()
	rp.circuitBreaker.RecordSuccess() // Reset circuit breaker
}

// GetStats returns retry policy statistics
func (rp *RetryPolicy) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"maxRetries":     rp.maxRetries,
		"attemptCount":   rp.backoff.GetAttemptCount(),
		"circuitBreaker": rp.circuitBreaker.GetStats(),
	}
}
