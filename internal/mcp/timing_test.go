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

	// Get initial state
	connections := connMgr.ListInstances()
	require.Len(t, connections, 1)

	conn := connections[0]
	assert.Equal(t, StateDiscovered, conn.State)
	assert.NotZero(t, conn.DiscoveredAt)
	assert.Equal(t, conn.DiscoveredAt, conn.StateChangedAt)
	assert.Empty(t, conn.StateHistory)

	// Wait a bit then change state
	time.Sleep(100 * time.Millisecond)

	// Update to connecting
	err = connMgr.updateStateWithReason(instance.ID, StateConnecting, "Test transition")
	require.NoError(t, err)

	// Check timing was updated
	connections = connMgr.ListInstances()
	conn = connections[0]
	assert.Equal(t, StateConnecting, conn.State)
	assert.True(t, conn.StateChangedAt.After(conn.DiscoveredAt))
	assert.Len(t, conn.StateHistory, 1)

	// Check transition history
	trans := conn.StateHistory[0]
	assert.Equal(t, StateDiscovered, trans.From)
	assert.Equal(t, StateConnecting, trans.To)
	assert.Equal(t, "Test transition", trans.Reason)
	assert.NotZero(t, trans.Timestamp)

	// Update to active
	time.Sleep(50 * time.Millisecond)
	err = connMgr.updateStateWithReason(instance.ID, StateActive, "Connection established")
	require.NoError(t, err)

	connections = connMgr.ListInstances()
	conn = connections[0]
	assert.Equal(t, StateActive, conn.State)
	assert.Len(t, conn.StateHistory, 2)

	// Calculate time spent in each state
	timeInDiscovered := conn.StateHistory[0].Timestamp.Sub(conn.DiscoveredAt)
	timeInConnecting := conn.StateHistory[1].Timestamp.Sub(conn.StateHistory[0].Timestamp)

	// Should have spent at least 100ms in discovered, 50ms in connecting
	assert.Greater(t, timeInDiscovered.Milliseconds(), int64(90))
	assert.Greater(t, timeInConnecting.Milliseconds(), int64(40))
}
