package mcp

import (
	"fmt"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
)

// TestConnectionManagerChannels tests channel-based operations
func TestConnectionManagerChannels(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Test registration
	instance := &discovery.Instance{
		ID:        "test-123",
		Name:      "test",
		Directory: "/test",
		Port:      7778,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 12345

	err := cm.RegisterInstance(instance)
	if err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	// Give it time to process and attempt connection
	time.Sleep(200 * time.Millisecond)

	// Test listing
	instances := cm.ListInstances()
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(instances))
	}

	// Instance should be in connecting or retrying state (no server running)
	if instances[0].State != StateConnecting && instances[0].State != StateRetrying {
		t.Errorf("Expected state %v or %v, got %v", StateConnecting, StateRetrying, instances[0].State)
	}
}

// TestConnectionStates tests state transitions
func TestConnectionStates(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Register instance
	instance := &discovery.Instance{
		ID:        "state-test",
		Name:      "State Test",
		Directory: "/test",
		Port:      7779,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 12346

	err := cm.RegisterInstance(instance)
	if err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	// Wait for connection attempt
	time.Sleep(200 * time.Millisecond)

	instances := cm.ListInstances()
	if len(instances) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(instances))
	}

	// Instance should be in connecting or retrying state (no server running)
	if instances[0].State != StateConnecting && instances[0].State != StateRetrying {
		t.Errorf("Expected state %v or %v, got %v", StateConnecting, StateRetrying, instances[0].State)
	}
}

// TestSessionMapping tests session connect/disconnect
func TestSessionMapping(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Register instance
	instance := &discovery.Instance{
		ID:        "session-test",
		Name:      "Session Test",
		Directory: "/test",
		Port:      7780,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 12347

	err := cm.RegisterInstance(instance)
	if err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	// Try to connect session (should fail - not active)
	sessionID := "test-session-1"
	err = cm.ConnectSession(sessionID, "session-test")
	if err == nil {
		t.Error("Expected error connecting to non-active instance")
	}

	// Manually set instance to active for testing
	cm.updateState("session-test", StateActive)

	// Create mock client
	client, _ := NewHubClient(7780)
	respChan := make(chan error)
	cm.setClientChan <- setClientRequest{
		instanceID: "session-test",
		client:     client,
		response:   respChan,
	}
	<-respChan

	// Now connect should work
	err = cm.ConnectSession(sessionID, "session-test")
	if err != nil {
		t.Errorf("Failed to connect session: %v", err)
	}

	// Get client should return the client
	gotClient := cm.GetClient(sessionID)
	if gotClient == nil {
		t.Error("Expected to get client, got nil")
	}

	// Disconnect
	err = cm.DisconnectSession(sessionID)
	if err != nil {
		t.Errorf("Failed to disconnect session: %v", err)
	}

	// Get client should return nil
	gotClient = cm.GetClient(sessionID)
	if gotClient != nil {
		t.Error("Expected nil client after disconnect")
	}
}

// TestConcurrentOperations tests concurrent channel operations
func TestConcurrentOperations(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Register multiple instances concurrently
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			instance := &discovery.Instance{
				ID:        fmt.Sprintf("concurrent-%d", id),
				Name:      fmt.Sprintf("Test %d", id),
				Directory: fmt.Sprintf("/test%d", id),
				Port:      8000 + id,
				StartedAt: time.Now(),
				LastPing:  time.Now(),
			}
			instance.ProcessInfo.PID = 20000 + id

			err := cm.RegisterInstance(instance)
			if err != nil {
				t.Errorf("Failed to register instance %d: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 5; i++ {
		<-done
	}

	// Give time to process
	time.Sleep(200 * time.Millisecond)

	// Should have 5 instances
	instances := cm.ListInstances()
	if len(instances) != 5 {
		t.Errorf("Expected 5 instances, got %d", len(instances))
	}
}

// TestCleanShutdown tests that Stop() cleanly shuts down
func TestCleanShutdown(t *testing.T) {
	cm := NewConnectionManager()

	// Register an instance
	instance := &discovery.Instance{
		ID:        "shutdown-test",
		Name:      "Shutdown Test",
		Directory: "/test",
		Port:      7781,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 12348

	err := cm.RegisterInstance(instance)
	if err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	// Stop should complete without hanging
	done := make(chan bool)
	go func() {
		cm.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Good
	case <-time.After(2 * time.Second):
		t.Error("Stop() timed out")
	}
}
