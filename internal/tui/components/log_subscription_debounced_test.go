package components

import (
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/stretchr/testify/assert"
)

func TestDebouncedLogSubscription(t *testing.T) {
	// Create a log store
	store := logs.NewStore(1000, nil)

	// Create debounced subscription
	sub := NewDebouncedLogSubscription(store)
	defer sub.Stop()

	// Add some logs rapidly
	for i := 0; i < 10; i++ {
		store.Add("test-proc", "TestApp", "Log line", false)
		sub.NotifyNewLogs() // Trigger notification
	}

	// Start subscription
	cmd := sub.Subscribe()

	// Wait for debounce to complete
	time.Sleep(100 * time.Millisecond)

	// Execute the command
	msg := cmd()

	// Should get a LogsUpdateMsg with all 10 logs
	updateMsg, ok := msg.(LogsUpdateMsg)
	assert.True(t, ok, "Expected LogsUpdateMsg")
	assert.Equal(t, 10, len(updateMsg.Entries), "Should batch all 10 logs")
	assert.True(t, updateMsg.Append, "Should be append mode")
}

func TestDebouncedSubscriptionNoPolling(t *testing.T) {
	// Create a log store
	store := logs.NewStore(1000, nil)

	// Create debounced subscription
	sub := NewDebouncedLogSubscription(store)
	defer sub.Stop()

	// Start subscription without any logs
	cmd := sub.Subscribe()

	// Should block waiting for notification
	done := make(chan bool)
	go func() {
		cmd()
		done <- true
	}()

	// Wait a bit to ensure it's not polling
	select {
	case <-done:
		t.Fatal("Should not return without notification")
	case <-time.After(200 * time.Millisecond):
		// Good, it's waiting
	}

	// Now trigger an update
	store.Add("test-proc", "TestApp", "New log", false)
	sub.NotifyNewLogs()

	// Should now complete
	select {
	case <-done:
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Should have returned after notification")
	}
}
