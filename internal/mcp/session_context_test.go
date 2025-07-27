package mcp

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSessionContextCreation(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	sm.Start()

	// Create a session
	session := sm.CreateSession("test-session", map[string]string{
		"name":    "test-client",
		"version": "1.0",
	})

	if session.ID != "test-session" {
		t.Errorf("Expected session ID 'test-session', got %s", session.ID)
	}

	if session.ctx == nil {
		t.Error("Session context should not be nil")
	}

	if session.cancel == nil {
		t.Error("Session cancel function should not be nil")
	}

	// Test context is valid
	select {
	case <-session.ctx.Done():
		t.Error("Session context should not be cancelled initially")
	default:
	}
}

func TestSessionContextCancellation(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	sm.Start()

	// Create a session with short timeout for testing
	sm.defaultTimeout = 100 * time.Millisecond
	session := sm.CreateSession("test-session", nil)

	// Wait for context to timeout
	time.Sleep(150 * time.Millisecond)

	// Context should be cancelled
	select {
	case <-session.ctx.Done():
		// Expected
	default:
		t.Error("Session context should be cancelled after timeout")
	}
}

func TestConnectionContextCreation(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	sm.Start()

	session := sm.CreateSession("test-session", nil)

	// Create connection context
	connCtx := session.CreateConnectionContext("instance-123")

	if connCtx.instanceID != "instance-123" {
		t.Errorf("Expected instance ID 'instance-123', got %s", connCtx.instanceID)
	}

	if connCtx.ctx == nil {
		t.Error("Connection context should not be nil")
	}

	if connCtx.cancel == nil {
		t.Error("Connection cancel function should not be nil")
	}

	// Test context is derived from session context
	select {
	case <-connCtx.ctx.Done():
		t.Error("Connection context should not be cancelled initially")
	default:
	}
}

func TestOperationContextCreation(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	sm.Start()

	session := sm.CreateSession("test-session", nil)
	connCtx := session.CreateConnectionContext("instance-123")

	// Create operation context
	opCtx := connCtx.CreateOperationContext("tool_call", 30*time.Second)

	if opCtx.operation != "tool_call" {
		t.Errorf("Expected operation 'tool_call', got %s", opCtx.operation)
	}

	if opCtx.timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", opCtx.timeout)
	}

	if opCtx.ctx == nil {
		t.Error("Operation context should not be nil")
	}

	if opCtx.cancel == nil {
		t.Error("Operation cancel function should not be nil")
	}
}

func TestContextHierarchyCancellation(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	sm.Start()

	// Create session with normal timeout
	session := sm.CreateSession("test-session", nil)
	connCtx := session.CreateConnectionContext("instance-123")
	opCtx := connCtx.CreateOperationContext("tool_call", 5*time.Second)

	// Cancel session context
	session.cancel()

	// Give contexts time to propagate cancellation
	time.Sleep(10 * time.Millisecond)

	// All derived contexts should be cancelled
	select {
	case <-session.ctx.Done():
		// Expected
	default:
		t.Error("Session context should be cancelled")
	}

	select {
	case <-connCtx.ctx.Done():
		// Expected
	default:
		t.Error("Connection context should be cancelled when session is cancelled")
	}

	select {
	case <-opCtx.ctx.Done():
		// Expected
	default:
		t.Error("Operation context should be cancelled when session is cancelled")
	}
}

func TestOperationTimeout(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	sm.Start()

	session := sm.CreateSession("test-session", nil)
	connCtx := session.CreateConnectionContext("instance-123")

	// Create operation with short timeout
	opCtx := connCtx.CreateOperationContext("tool_call", 50*time.Millisecond)

	// Wait for operation to timeout
	time.Sleep(100 * time.Millisecond)

	// Operation context should be cancelled due to timeout
	select {
	case <-opCtx.ctx.Done():
		// Expected
		if opCtx.ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", opCtx.ctx.Err())
		}
	default:
		t.Error("Operation context should be cancelled after timeout")
	}

	// Session and connection contexts should still be active
	select {
	case <-session.ctx.Done():
		t.Error("Session context should not be cancelled")
	default:
	}

	select {
	case <-connCtx.ctx.Done():
		t.Error("Connection context should not be cancelled")
	default:
	}
}

func TestSessionCleanup(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	// Set short cleanup interval for testing
	sm.cleanupInterval = 50 * time.Millisecond
	sm.Start()

	// Create session with short timeout
	sm.defaultTimeout = 100 * time.Millisecond
	sm.CreateSession("test-session", nil)

	// Verify session exists
	_, err := sm.GetSession("test-session")
	if err != nil {
		t.Errorf("Session should exist: %v", err)
	}

	// Wait for session to timeout and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Session should be cleaned up
	_, err = sm.GetSession("test-session")
	if err == nil {
		t.Error("Session should be cleaned up after timeout")
	}
}

func TestConnectionManagerContextIntegration(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Test that session manager is properly initialized
	if cm.sessionManager == nil {
		t.Error("Connection manager should have session manager")
	}

	// Test context-aware connection
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := cm.ConnectWithContext(ctx, "test-instance")
	if err != nil {
		t.Errorf("ConnectWithContext should not return error: %v", err)
	}

	// Test session-based connection
	err = cm.ConnectSessionToInstance("test-session", "test-instance")
	if err != nil {
		t.Errorf("ConnectSessionToInstance should not return error: %v", err)
	}

	// Verify session was created
	session, err := cm.sessionManager.GetSession("test-session")
	if err != nil {
		t.Errorf("Session should be created: %v", err)
		return
	}

	// Verify connection context was created
	_, exists := session.GetConnectionContext("test-instance")
	if !exists {
		t.Error("Connection context should be created for instance")
	}
}

func TestConcurrentContextOperations(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Stop()

	sm.Start()

	session := sm.CreateSession("test-session", nil)

	// Create multiple connection contexts concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			instanceID := fmt.Sprintf("instance-%d", id)
			connCtx := session.CreateConnectionContext(instanceID)

			// Create multiple operation contexts
			for j := 0; j < 5; j++ {
				operation := fmt.Sprintf("op-%d-%d", id, j)
				opCtx := connCtx.CreateOperationContext(operation, 100*time.Millisecond)

				// Verify context is valid
				if opCtx.ctx == nil {
					t.Errorf("Operation context should not be nil")
				}
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all connection contexts were created
	for i := 0; i < 10; i++ {
		instanceID := fmt.Sprintf("instance-%d", i)
		_, exists := session.GetConnectionContext(instanceID)
		if !exists {
			t.Errorf("Connection context for %s should exist", instanceID)
		}
	}
}

func TestContextAwareRetryLogic(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Start connection attempt
	err := cm.ConnectWithContext(ctx, "nonexistent-instance")
	if err != nil {
		t.Errorf("ConnectWithContext should not return error immediately: %v", err)
	}

	// Cancel context after a short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Give some time for the connection attempt to process cancellation
	time.Sleep(50 * time.Millisecond)

	// The connection attempt should have been cancelled and should not retry indefinitely
	// (This is more of an integration test to ensure context cancellation is respected)
}
