package testutil

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// WaitForCondition waits for a condition to be true, checking every 10ms
// Returns true if condition met, false if timeout
func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return true
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return false
			}
		}
	}
}

// WaitForValue waits for a function to return the expected value
func WaitForValue[T comparable](t *testing.T, timeout time.Duration, getter func() T, expected T) bool {
	t.Helper()
	return WaitForCondition(t, timeout, func() bool {
		return getter() == expected
	})
}

// EventuallyTrue asserts that a condition becomes true within timeout
func EventuallyTrue(t *testing.T, timeout time.Duration, condition func() bool, msg string) {
	t.Helper()
	if !WaitForCondition(t, timeout, condition) {
		t.Errorf("Condition not met within %v: %s", timeout, msg)
	}
}

// RequireEventually asserts that a condition becomes true within timeout and fails the test if not
func RequireEventually(t *testing.T, timeout time.Duration, condition func() bool, msg string) {
	t.Helper()
	require.True(t, WaitForCondition(t, timeout, condition), "Condition not met within %v: %s", timeout, msg)
}

// WaitForState waits for a getter to return a specific state value
func WaitForState[T comparable](t *testing.T, timeout time.Duration, getter func() T, expected T) {
	t.Helper()
	RequireEventually(t, timeout, func() bool {
		return getter() == expected
	}, fmt.Sprintf("Expected state %v", expected))
}

// WaitForCount waits for a count function to return the expected number
func WaitForCount(t *testing.T, timeout time.Duration, counter func() int, expected int) {
	t.Helper()
	RequireEventually(t, timeout, func() bool {
		return counter() == expected
	}, fmt.Sprintf("Expected count %d", expected))
}

// WaitForNoError waits for a function to return no error
func WaitForNoError(t *testing.T, timeout time.Duration, operation func() error) {
	t.Helper()
	RequireEventually(t, timeout, func() bool {
		return operation() == nil
	}, "Expected no error")
}

// WaitGroup with timeout support
type TimedWaitGroup struct {
	wg sync.WaitGroup
}

// Add wraps sync.WaitGroup.Add
func (twg *TimedWaitGroup) Add(delta int) {
	twg.wg.Add(delta)
}

// Done wraps sync.WaitGroup.Done
func (twg *TimedWaitGroup) Done() {
	twg.wg.Done()
}

// WaitWithTimeout waits for the WaitGroup with a timeout
func (twg *TimedWaitGroup) WaitWithTimeout(timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		twg.wg.Wait()
	}()

	select {
	case <-c:
		return true
	case <-time.After(timeout):
		return false
	}
}

// RequireWaitWithTimeout requires the WaitGroup to complete within timeout
func (twg *TimedWaitGroup) RequireWaitWithTimeout(t *testing.T, timeout time.Duration, msg string) {
	t.Helper()
	require.True(t, twg.WaitWithTimeout(timeout), "WaitGroup did not complete within %v: %s", timeout, msg)
}

// ContextWithDeadline creates a context with deadline for testing
func ContextWithDeadline(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
