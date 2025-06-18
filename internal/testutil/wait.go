package testutil

import (
	"testing"
	"time"
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
