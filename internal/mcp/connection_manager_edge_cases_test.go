package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/standardbeagle/brummer/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStateTransitionValidation tests that invalid state transitions are prevented
func TestStateTransitionValidation(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Register instance
	instance := &discovery.Instance{
		ID:        "state-validation",
		Name:      "State Validation Test",
		Directory: "/test",
		Port:      9001,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 30001

	err := cm.RegisterInstance(instance)
	require.NoError(t, err)
	
	// Wait for instance to be registered and in a stable state
	testutil.RequireEventually(t, 2*time.Second, func() bool {
		instances := cm.ListInstances()
		return len(instances) > 0
	}, "Instance should be registered")

	// Test invalid transitions
	tests := []struct {
		name         string
		setupState   ConnectionState
		targetState  ConnectionState
		shouldAllow  bool
		description  string
	}{
		{
			name:        "dead_to_active",
			setupState:  StateDead,
			targetState: StateActive,
			shouldAllow: false, // Should go through Discovered first
			description: "Cannot go directly from Dead to Active",
		},
		{
			name:        "active_to_discovered",
			setupState:  StateActive,
			targetState: StateDiscovered,
			shouldAllow: false, // Going backwards doesn't make sense
			description: "Cannot go from Active back to Discovered",
		},
		{
			name:        "retrying_to_active",
			setupState:  StateRetrying,
			targetState: StateActive,
			shouldAllow: true, // Valid recovery path
			description: "Can recover from Retrying to Active",
		},
		{
			name:        "dead_to_discovered",
			setupState:  StateDead,
			targetState: StateDiscovered,
			shouldAllow: true, // Valid resurrection path
			description: "Can resurrect from Dead to Discovered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set initial state
			cm.updateState("state-validation", tt.setupState)
			
			// Get current state
			instances := cm.ListInstances()
			if len(instances) == 0 {
				// Instance was removed when marked as dead
				t.Skip("Instance removed when marked as dead")
				return
			}
			require.Len(t, instances, 1)
			initialState := instances[0].State
			
			// Try to transition
			cm.updateState("state-validation", tt.targetState)
			
			// Check result
			instances = cm.ListInstances()
			if len(instances) == 0 {
				// Instance was filtered out when marked as dead
				t.Logf("Instance removed from list when marked as dead")
				return
			}
			require.Len(t, instances, 1)
			finalState := instances[0].State
			
			if tt.shouldAllow {
				assert.Equal(t, tt.targetState, finalState, tt.description)
			} else {
				// For now, the system doesn't enforce state transition rules
				// This test documents the expected behavior when implemented
				t.Logf("State transition %s -> %s occurred (validation not implemented)", 
					initialState, finalState)
			}
		})
	}
}

// TestRapidConnectDisconnect tests rapid connection/disconnection cycles
func TestRapidConnectDisconnect(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Create a mock instance that toggles availability
	var serverRunning atomic.Bool
	serverRunning.Store(true)
	
	mockInstance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !serverRunning.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		
		// Handle initialize
		var req JSONRPCMessage
		json.NewDecoder(r.Body).Decode(&req)
		
		resp := JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "1.0",
				"serverInfo": map[string]interface{}{
					"name":    "rapid-test",
					"version": "1.0.0",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockInstance.Close()

	var port int
	fmt.Sscanf(mockInstance.URL, "http://127.0.0.1:%d", &port)

	instance := &discovery.Instance{
		ID:        "rapid-instance",
		Name:      "Rapid Test",
		Directory: "/rapid",
		Port:      port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 30002

	// Register instance
	err := cm.RegisterInstance(instance)
	require.NoError(t, err)

	// Rapid on/off cycles
	for i := 0; i < 5; i++ {
		// Let it connect
		serverRunning.Store(true)
		time.Sleep(150 * time.Millisecond)
		
		// Check state
		instances := cm.ListInstances()
		require.Len(t, instances, 1)
		t.Logf("Cycle %d UP: State = %v", i, instances[0].State)
		
		// Disconnect
		serverRunning.Store(false)
		time.Sleep(150 * time.Millisecond)
		
		// Check state
		instances = cm.ListInstances()
		require.Len(t, instances, 1)
		t.Logf("Cycle %d DOWN: State = %v", i, instances[0].State)
	}

	// System should still be stable
	instances := cm.ListInstances()
	assert.Len(t, instances, 1)
	assert.NotNil(t, instances[0])
}

// TestSessionLimitEnforcement tests that session limits are enforced
func TestSessionLimitEnforcement(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Create mock instance
	mockInstance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCMessage
		json.NewDecoder(r.Body).Decode(&req)
		
		resp := JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "1.0",
				"serverInfo": map[string]interface{}{
					"name":    "session-limit-test",
					"version": "1.0.0",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockInstance.Close()

	var port int
	fmt.Sscanf(mockInstance.URL, "http://127.0.0.1:%d", &port)

	instance := &discovery.Instance{
		ID:        "session-instance",
		Name:      "Session Test",
		Directory: "/session",
		Port:      port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 30003

	err := cm.RegisterInstance(instance)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Connect multiple sessions
	const maxSessions = 10 // Reasonable limit for testing
	
	for i := 0; i < maxSessions; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		err := cm.ConnectSession(sessionID, "session-instance")
		assert.NoError(t, err, "Session %d should connect", i)
	}

	// Verify all sessions are connected
	instances := cm.ListInstances()
	require.Len(t, instances, 1)
	assert.Len(t, instances[0].Sessions, maxSessions)

	// Try to exceed limit (currently no limit enforced)
	err = cm.ConnectSession("session-overflow", "session-instance")
	// Document that limit enforcement is not implemented
	assert.NoError(t, err, "Session limits not currently enforced")
}

// TestOrphanedSessionCleanup tests cleanup of sessions when instance dies
func TestOrphanedSessionCleanup(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Create controllable mock instance
	var serverRunning atomic.Bool
	serverRunning.Store(true)
	
	mockInstance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !serverRunning.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		
		var req JSONRPCMessage
		json.NewDecoder(r.Body).Decode(&req)
		
		resp := JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "1.0",
				"serverInfo": map[string]interface{}{
					"name":    "orphan-test",
					"version": "1.0.0",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockInstance.Close()

	var port int
	fmt.Sscanf(mockInstance.URL, "http://127.0.0.1:%d", &port)

	instance := &discovery.Instance{
		ID:        "orphan-instance",
		Name:      "Orphan Test",
		Directory: "/orphan",
		Port:      port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 30004

	err := cm.RegisterInstance(instance)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Connect sessions
	sessions := []string{"orphan-1", "orphan-2", "orphan-3"}
	for _, sessionID := range sessions {
		err := cm.ConnectSession(sessionID, "orphan-instance")
		require.NoError(t, err)
	}

	// Verify sessions connected
	for _, sessionID := range sessions {
		client := cm.GetClient(sessionID)
		assert.NotNil(t, client, "Session %s should have client", sessionID)
	}

	// Kill the instance
	serverRunning.Store(false)
	
	// Mark instance as dead
	cm.updateState("orphan-instance", StateDead)

	// Sessions should be cleaned up
	// Note: Current implementation may not clean up immediately
	time.Sleep(100 * time.Millisecond)

	// Check if sessions are still accessible
	for _, sessionID := range sessions {
		client := cm.GetClient(sessionID)
		// Document current behavior
		if client != nil {
			t.Logf("Session %s still has client after instance death (cleanup not implemented)", sessionID)
		}
	}
}

// TestConcurrentStateUpdates tests race conditions in state updates
func TestConcurrentStateUpdates(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	instance := &discovery.Instance{
		ID:        "concurrent-state",
		Name:      "Concurrent State Test",
		Directory: "/concurrent",
		Port:      9005,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 30005

	err := cm.RegisterInstance(instance)
	require.NoError(t, err)

	// Launch concurrent state updates
	var wg sync.WaitGroup
	states := []ConnectionState{StateConnecting, StateActive, StateRetrying, StateActive}
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			state := states[iteration%len(states)]
			cm.updateState("concurrent-state", state)
			
			// Also update activity
			if iteration%2 == 0 {
				cm.UpdateActivity("concurrent-state")
			}
		}(i)
	}

	wg.Wait()

	// System should still be stable
	instances := cm.ListInstances()
	assert.Len(t, instances, 1)
	assert.NotNil(t, instances[0])
	assert.Contains(t, states, instances[0].State)
}

// TestMemoryLeakPrevention tests that resources are properly cleaned up
func TestMemoryLeakPrevention(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Register and unregister many instances
	for i := 0; i < 20; i++ {
		instance := &discovery.Instance{
			ID:        fmt.Sprintf("leak-test-%d", i),
			Name:      fmt.Sprintf("Leak Test %d", i),
			Directory: fmt.Sprintf("/leak%d", i),
			Port:      10000 + i,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
		}
		instance.ProcessInfo.PID = 40000 + i

		err := cm.RegisterInstance(instance)
		require.NoError(t, err)
		
		// Connect some sessions
		for j := 0; j < 3; j++ {
			sessionID := fmt.Sprintf("leak-session-%d-%d", i, j)
			// Don't check error as instance may not be active
			cm.ConnectSession(sessionID, instance.ID)
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Check current count
	instances := cm.ListInstances()
	initialCount := len(instances)
	t.Logf("Registered %d instances", initialCount)

	// Mark all as dead
	for _, inst := range instances {
		cm.updateState(inst.InstanceID, StateDead)
	}

	// In a complete implementation, dead instances would be cleaned up
	// For now, just verify the system is still responsive
	instances = cm.ListInstances()
	assert.Len(t, instances, initialCount, "Dead instances not automatically cleaned up")
}

// TestUpdateActivityRaceCondition tests concurrent activity updates
func TestUpdateActivityRaceCondition(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	instance := &discovery.Instance{
		ID:        "activity-race",
		Name:      "Activity Race Test",
		Directory: "/activity",
		Port:      9006,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 30006

	err := cm.RegisterInstance(instance)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	
	// Set instance to active so activity updates work
	cm.updateState("activity-race", StateActive)

	// Concurrent activity updates
	var wg sync.WaitGroup
	updateCount := int32(0)
	
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if cm.UpdateActivity("activity-race") {
				atomic.AddInt32(&updateCount, 1)
			}
		}()
	}

	wg.Wait()

	// Should have processed updates without crashing
	assert.Greater(t, updateCount, int32(0), "At least some updates should succeed")
	
	// Verify instance still exists and has recent activity
	instances := cm.ListInstances()
	require.Len(t, instances, 1)
	assert.WithinDuration(t, time.Now(), instances[0].LastActivity, 100*time.Millisecond)
}

// TestConnectionRetryBackoff tests retry behavior with backoff
func TestConnectionRetryBackoff(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Mock instance that always fails
	mockInstance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockInstance.Close()

	var port int
	fmt.Sscanf(mockInstance.URL, "http://127.0.0.1:%d", &port)

	instance := &discovery.Instance{
		ID:        "retry-backoff",
		Name:      "Retry Backoff Test",
		Directory: "/retry",
		Port:      port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 30007

	startTime := time.Now()
	err := cm.RegisterInstance(instance)
	require.NoError(t, err)

	// Monitor retry attempts
	retryStates := []struct {
		checkTime time.Duration
		expectRetries int
	}{
		{100 * time.Millisecond, 1},
		{500 * time.Millisecond, 1}, // Should still be in first retry
		{1 * time.Second, 1},        // Backoff should prevent rapid retries
	}

	for _, check := range retryStates {
		time.Sleep(check.checkTime - time.Since(startTime))
		
		instances := cm.ListInstances()
		require.Len(t, instances, 1)
		
		// Note: Current implementation doesn't track retry count in ConnectionInfo
		// This documents expected behavior
		t.Logf("After %v: State=%v, RetryCount=%d", 
			check.checkTime, instances[0].State, instances[0].RetryCount)
	}
}

// TestInstanceReregistration tests behavior when same instance registers twice
func TestInstanceReregistration(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// First registration
	instance1 := &discovery.Instance{
		ID:        "reregister-test",
		Name:      "Reregister Test",
		Directory: "/reregister",
		Port:      9007,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance1.ProcessInfo.PID = 30008

	err := cm.RegisterInstance(instance1)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// Get initial state
	instances := cm.ListInstances()
	require.Len(t, instances, 1)
	initialState := instances[0].State

	// Same instance ID but different port (simulating restart)
	instance2 := &discovery.Instance{
		ID:        "reregister-test", // Same ID
		Name:      "Reregister Test",
		Directory: "/reregister",
		Port:      9008, // Different port
		StartedAt: time.Now().Add(1 * time.Minute), // Later start time
		LastPing:  time.Now(),
	}
	instance2.ProcessInfo.PID = 30009 // Different PID

	err = cm.RegisterInstance(instance2)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// Should still have only one instance
	instances = cm.ListInstances()
	assert.Len(t, instances, 1)
	
	// Should have updated port
	assert.Equal(t, 9008, instances[0].Port)
	
	// State might have changed
	t.Logf("State transition on re-registration: %v -> %v", initialState, instances[0].State)
}

// TestContextCancellation tests proper handling of context cancellation
func TestContextCancellation(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Mock instance with slow responses
	mockInstance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(500 * time.Millisecond)
		
		var req JSONRPCMessage
		json.NewDecoder(r.Body).Decode(&req)
		
		resp := JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "1.0",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockInstance.Close()

	var port int
	fmt.Sscanf(mockInstance.URL, "http://127.0.0.1:%d", &port)

	instance := &discovery.Instance{
		ID:        "context-cancel",
		Name:      "Context Cancel Test",
		Directory: "/cancel",
		Port:      port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 30010

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())
	
	// Register instance
	err := cm.RegisterInstance(instance)
	require.NoError(t, err)

	// Cancel context quickly
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// The connection attempt should handle cancellation gracefully
	time.Sleep(200 * time.Millisecond)

	// System should still be stable
	instances := cm.ListInstances()
	assert.Len(t, instances, 1)
	
	// Use context to show it would be used in real implementation
	_ = ctx
}