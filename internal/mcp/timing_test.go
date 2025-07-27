package mcp

import (
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionStateTiming(t *testing.T) {
	// Create connection manager
	connMgr := NewConnectionManager()
	defer connMgr.Stop()

	// Create mock instance
	instance := &discovery.Instance{
		ID:        "test-timing",
		Name:      "Test Timing",
		Directory: "/test",
		Port:      12345,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 12345
	instance.ProcessInfo.Executable = "brum"

	// Register instance
	err := connMgr.RegisterInstance(instance)
	require.NoError(t, err)

	// Get initial state - allow for automatic state transitions
	connections := connMgr.ListInstances()
	require.Len(t, connections, 1)

	conn := connections[0]
	// State may have automatically transitioned from discovered to connecting
	initialState := conn.State
	assert.NotZero(t, conn.DiscoveredAt)
	// Allow some tolerance for automatic state changes
	historyLen := len(conn.StateHistory)

	// Wait a bit then change state
	time.Sleep(100 * time.Millisecond)

	// Update to active (or connecting->active if already connecting)
	targetState := StateActive
	if initialState == StateDiscovered {
		targetState = StateConnecting
	}
	err = connMgr.updateStateWithReason(instance.ID, targetState, "Test transition")
	require.NoError(t, err)

	// Check timing was updated
	connections = connMgr.ListInstances()
	conn = connections[0]
	assert.Equal(t, targetState, conn.State)
	assert.True(t, conn.StateChangedAt.After(conn.DiscoveredAt))
	expectedHistoryLen := historyLen + 1
	assert.Len(t, conn.StateHistory, expectedHistoryLen)

	// Check the most recent transition - it should be our test transition
	trans := conn.StateHistory[len(conn.StateHistory)-1]
	assert.Equal(t, initialState, trans.From)
	assert.Equal(t, targetState, trans.To)
	assert.Equal(t, "Test transition", trans.Reason)
	assert.NotZero(t, trans.Timestamp)

	// Update to active (if not already active)
	time.Sleep(50 * time.Millisecond)
	if targetState != StateActive {
		err = connMgr.updateStateWithReason(instance.ID, StateActive, "Connection established")
		require.NoError(t, err)
	}

	connections = connMgr.ListInstances()
	conn = connections[0]
	assert.Equal(t, StateActive, conn.State)

	// Verify we have the expected number of transitions
	finalHistoryLen := len(conn.StateHistory)
	assert.GreaterOrEqual(t, finalHistoryLen, 1, "Should have at least one state transition")

	// Calculate time between the transitions we made
	if finalHistoryLen >= 2 {
		lastTransition := conn.StateHistory[finalHistoryLen-1]
		prevTransition := conn.StateHistory[finalHistoryLen-2]
		timeBetweenTransitions := lastTransition.Timestamp.Sub(prevTransition.Timestamp)
		// Should have spent at least 40ms between our transitions
		assert.Greater(t, timeBetweenTransitions.Milliseconds(), int64(40))
	}
}
