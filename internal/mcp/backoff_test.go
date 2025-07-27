package mcp

import (
	"errors"
	"testing"
	"time"
)

func TestExponentialBackoff(t *testing.T) {
	backoff := NewExponentialBackoff()

	// Test first delay
	delay1 := backoff.NextDelay()
	if delay1 < backoff.BaseDelay {
		t.Errorf("First delay should be at least base delay, got %v", delay1)
	}

	// Test exponential growth
	delay2 := backoff.NextDelay()
	if delay2 <= delay1 {
		t.Errorf("Second delay should be larger than first delay, got %v <= %v", delay2, delay1)
	}

	// Test max delay cap
	for i := 0; i < 10; i++ {
		delay := backoff.NextDelay()
		if delay > backoff.MaxDelay*2 { // Allow some jitter
			t.Errorf("Delay exceeded max delay significantly: %v > %v", delay, backoff.MaxDelay)
		}
	}

	// Test reset
	backoff.Reset()
	delayAfterReset := backoff.NextDelay()
	if delayAfterReset > delay1*2 { // Should be back to early values
		t.Errorf("Delay after reset should be smaller, got %v", delayAfterReset)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	// Test initial state (closed)
	if cb.GetState() != CircuitClosed {
		t.Errorf("Initial state should be closed, got %v", cb.GetState())
	}

	// Test allowing requests when closed
	if !cb.AllowRequest() {
		t.Error("Should allow requests when circuit is closed")
	}

	// Test failure recording
	for i := 0; i < 2; i++ {
		cb.RecordFailure()
		if cb.GetState() != CircuitClosed {
			t.Errorf("Circuit should still be closed after %d failures", i+1)
		}
	}

	// Test circuit opening after max failures
	cb.RecordFailure()
	if cb.GetState() != CircuitOpen {
		t.Error("Circuit should be open after reaching max failures")
	}

	// Test blocking requests when open
	if cb.AllowRequest() {
		t.Error("Should not allow requests when circuit is open")
	}

	// Test half-open after timeout
	time.Sleep(150 * time.Millisecond)
	if !cb.AllowRequest() {
		t.Error("Should allow requests after timeout (half-open state)")
	}

	if cb.GetState() != CircuitHalfOpen {
		t.Errorf("State should be half-open after timeout, got %v", cb.GetState())
	}

	// Test recovery with success
	cb.RecordSuccess()
	if cb.GetState() != CircuitClosed {
		t.Error("Circuit should be closed after successful recovery")
	}
}

func TestCircuitBreakerCall(t *testing.T) {
	cb := DefaultCircuitBreaker()

	// Test successful call
	err := cb.Call(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Successful call should not return error, got %v", err)
	}

	// Test failing calls
	testError := errors.New("test error")
	for i := 0; i < cb.maxFailures; i++ {
		err := cb.Call(func() error {
			return testError
		})
		if err != testError {
			t.Errorf("Should return original error, got %v", err)
		}
	}

	// Test circuit breaker error when open
	err = cb.Call(func() error {
		return nil
	})
	if err == nil {
		t.Error("Should return circuit breaker error when circuit is open")
	}

	if !IsCircuitBreakerError(err) {
		t.Errorf("Should return circuit breaker error, got %T", err)
	}
}

func TestRetryPolicy(t *testing.T) {
	rp := NewRetryPolicy(3)

	// Test successful execution
	attempts := 0
	err := rp.ExecuteWithRetry(func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Should succeed on first attempt, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("Should only make one attempt for success, made %d", attempts)
	}

	// Test retry on failure
	attempts = 0
	testError := errors.New("test error")
	err = rp.ExecuteWithRetry(func() error {
		attempts++
		if attempts < 3 {
			return testError
		}
		return nil
	})

	if err != nil {
		t.Errorf("Should succeed after retries, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("Should make 3 attempts, made %d", attempts)
	}

	// Test exhausting retries
	rp2 := NewRetryPolicy(2)
	attempts = 0
	err = rp2.ExecuteWithRetry(func() error {
		attempts++
		return testError
	})

	if err != testError {
		t.Errorf("Should return last error after exhausting retries, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("Should make exactly 2 attempts, made %d", attempts)
	}
}

func TestRetryPolicyWithCircuitBreaker(t *testing.T) {
	rp := NewRetryPolicy(10)

	// Force circuit breaker to open by causing many failures
	for i := 0; i < 6; i++ {
		rp.circuitBreaker.RecordFailure()
	}

	// Now attempts should be blocked by circuit breaker
	attempts := 0
	err := rp.ExecuteWithRetry(func() error {
		attempts++
		return nil
	})

	if !IsCircuitBreakerError(err) {
		t.Errorf("Should return circuit breaker error, got %T: %v", err, err)
	}

	if attempts != 0 {
		t.Errorf("Should not make any attempts when circuit breaker is open, made %d", attempts)
	}
}

func TestBackoffJitter(t *testing.T) {
	backoff := &ExponentialBackoff{
		BaseDelay:  1 * time.Second,
		MaxDelay:   10 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}

	// Get multiple delays and ensure they vary (due to jitter)
	delays := make([]time.Duration, 5)
	for i := 0; i < 5; i++ {
		backoff.Reset()
		delays[i] = backoff.NextDelay()
	}

	// Check that not all delays are identical (jitter should cause variation)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("Jitter should cause variation in delays")
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	stats := cb.GetStats()

	expectedKeys := []string{"state", "failureCount", "maxFailures", "lastFailTime", "resetTimeout"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Stats should contain key %s", key)
		}
	}

	if stats["state"] != "closed" {
		t.Errorf("Initial state should be 'closed', got %v", stats["state"])
	}

	if stats["failureCount"] != 0 {
		t.Errorf("Initial failure count should be 0, got %v", stats["failureCount"])
	}
}

func TestRetryPolicyStats(t *testing.T) {
	rp := NewRetryPolicy(5)

	stats := rp.GetStats()

	expectedKeys := []string{"maxRetries", "attemptCount", "circuitBreaker"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Stats should contain key %s", key)
		}
	}

	if stats["maxRetries"] != 5 {
		t.Errorf("Max retries should be 5, got %v", stats["maxRetries"])
	}
}
