package services

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestNewAICoderService verifies that the service initializes with empty registry
func TestNewAICoderService(t *testing.T) {
	// This test will fail until we implement NewAICoderService
	ctx := context.Background()
	service := NewAICoderService(ctx)

	if service == nil {
		t.Fatal("NewAICoderService returned nil")
	}

	// Verify empty session registry
	sessions := service.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("Expected empty session list, got %d sessions", len(sessions))
	}
}

// TestServiceEventBusReference verifies EventBus is valid
func TestServiceEventBusReference(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	if service == nil {
		t.Fatal("NewAICoderService returned nil")
	}

	// EventBus should be accessible
	eventBus := service.EventBus()
	if eventBus == nil {
		t.Error("EventBus() returned nil")
	}
}

// TestServiceShutdownChannel verifies shutdown channel created
func TestServiceShutdownChannel(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	if service == nil {
		t.Fatal("NewAICoderService returned nil")
	}

	// Shutdown should complete without hanging
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}

// ============================================================================
// T013: Goroutine Leak Detection Tests
// ============================================================================

// TestNoGoroutineLeaks_SessionLifecycle verifies goroutine count returns to baseline
func TestNoGoroutineLeaks_SessionLifecycle(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	// Record baseline goroutine count (allow for test framework goroutines)
	baseline := countGoroutines()

	// Create and start a session
	sessionID, err := service.CreateSession(ctx, "Test Session", "sleep 1", ".")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = service.StartSession(ctx, sessionID, 80, 24)
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	// Wait for session to be active
	time.Sleep(100 * time.Millisecond)

	// Stop the session
	err = service.StopSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("StopSession failed: %v", err)
	}

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Check goroutine count returned to baseline (allow ±2 for timing variations)
	final := countGoroutines()
	delta := final - baseline
	if delta > 2 {
		t.Errorf("Goroutine leak detected: baseline=%d, final=%d, delta=%d", baseline, final, delta)
	}

	// Cleanup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = service.Shutdown(shutdownCtx)
}

// TestNoGoroutineLeaks_MultipleStarts verifies no leaks with rapid create/stop
func TestNoGoroutineLeaks_MultipleStarts(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	baseline := countGoroutines()

	// Rapid create/stop cycles
	for i := 0; i < 5; i++ {
		sessionID, err := service.CreateSession(ctx, "Rapid Session", "sleep 0.1", ".")
		if err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}

		err = service.StartSession(ctx, sessionID, 80, 24)
		if err != nil {
			t.Fatalf("StartSession %d failed: %v", i, err)
		}

		// Stop immediately
		err = service.StopSession(ctx, sessionID)
		if err != nil {
			t.Fatalf("StopSession %d failed: %v", i, err)
		}
	}

	// Wait for all cleanups (longer for rapid cycles)
	time.Sleep(1 * time.Second)

	// Check goroutine count (allow ±5 for rapid cycle timing variations)
	final := countGoroutines()
	delta := final - baseline
	if delta > 5 {
		t.Errorf("Goroutine leak detected after rapid cycles: baseline=%d, final=%d, delta=%d", baseline, final, delta)
	}

	// Cleanup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = service.Shutdown(shutdownCtx)
}

// TestMonitoringGoroutineCleanup verifies monitoring goroutines stopped
func TestMonitoringGoroutineCleanup(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	// Create multiple sessions
	sessionIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		id, err := service.CreateSession(ctx, "Monitor Test", "sleep 10", ".")
		if err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}
		sessionIDs[i] = id

		err = service.StartSession(ctx, id, 80, 24)
		if err != nil {
			t.Fatalf("StartSession %d failed: %v", i, err)
		}
	}

	// Wait for sessions to start
	time.Sleep(100 * time.Millisecond)
	baseline := countGoroutines()

	// Stop all sessions
	for i, id := range sessionIDs {
		err := service.StopSession(ctx, id)
		if err != nil {
			t.Fatalf("StopSession %d failed: %v", i, err)
		}
	}

	// Wait for cleanup
	time.Sleep(300 * time.Millisecond)

	// Goroutines should decrease (monitoring goroutines stopped)
	final := countGoroutines()
	if final >= baseline {
		t.Errorf("Monitoring goroutines not cleaned up: baseline=%d, final=%d", baseline, final)
	}

	// Cleanup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = service.Shutdown(shutdownCtx)
}

// ============================================================================
// T014: Resource Cleanup Verification Tests
// ============================================================================

// TestPTYFileDescriptorCleanup verifies PTY fds closed
func TestPTYFileDescriptorCleanup(t *testing.T) {
	t.Skip("PTY file descriptor tracking requires OS-specific implementation")
	// Note: Full implementation would use /proc/self/fd on Linux or lsof
}

// TestSyncMapCleanup verifies session registry cleared
func TestSyncMapCleanup(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	// Create multiple sessions
	sessionIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		id, err := service.CreateSession(ctx, "Cleanup Test", "sleep 5", ".")
		if err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}
		sessionIDs[i] = id

		err = service.StartSession(ctx, id, 80, 24)
		if err != nil {
			t.Fatalf("StartSession %d failed: %v", i, err)
		}
	}

	// Verify sessions exist
	sessions := service.ListSessions()
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}

	// Stop all sessions
	for _, id := range sessionIDs {
		err := service.StopSession(ctx, id)
		if err != nil {
			t.Fatalf("StopSession failed: %v", err)
		}
	}

	// Verify sessions removed from registry
	sessions = service.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("Expected empty session list after cleanup, got %d sessions", len(sessions))
	}

	// Cleanup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = service.Shutdown(shutdownCtx)
}

// TestNoZombieProcesses verifies PTY processes terminated
func TestNoZombieProcesses(t *testing.T) {
	t.Skip("Zombie process detection requires OS-specific implementation")
	// Note: Full implementation would check process table for zombie processes
}

// TestCleanupTiming verifies cleanup completes <100ms
func TestCleanupTiming(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	// Create and start a session
	sessionID, err := service.CreateSession(ctx, "Timing Test", "sleep 5", ".")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = service.StartSession(ctx, sessionID, 80, 24)
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	// Wait for session to be active
	time.Sleep(100 * time.Millisecond)

	// Measure cleanup time
	start := time.Now()
	err = service.StopSession(ctx, sessionID)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("StopSession failed: %v", err)
	}

	// Verify cleanup timing <100ms
	if elapsed > 100*time.Millisecond {
		t.Errorf("Cleanup took too long: %v (expected <100ms)", elapsed)
	}

	// Cleanup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = service.Shutdown(shutdownCtx)
}

// ============================================================================
// T015: Shutdown Timeout Tests
// ============================================================================

// TestShutdown_AllSessionsStopped verifies all sessions cleaned up
func TestShutdown_AllSessionsStopped(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		id, err := service.CreateSession(ctx, "Shutdown Test", "sleep 10", ".")
		if err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}

		err = service.StartSession(ctx, id, 80, 24)
		if err != nil {
			t.Fatalf("StartSession %d failed: %v", i, err)
		}
	}

	// Verify sessions exist
	sessions := service.ListSessions()
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions before shutdown, got %d", len(sessions))
	}

	// Shutdown should stop all sessions
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := service.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	// Verify all sessions stopped
	sessions = service.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions after shutdown, got %d", len(sessions))
	}
}

// TestShutdown_GracefulTimeout verifies respects context timeout
func TestShutdown_GracefulTimeout(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	// Create a session
	sessionID, err := service.CreateSession(ctx, "Timeout Test", "sleep 10", ".")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = service.StartSession(ctx, sessionID, 80, 24)
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	// Shutdown with short timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err = service.Shutdown(shutdownCtx)
	elapsed := time.Since(start)

	// Should complete within timeout
	if elapsed > 3*time.Second {
		t.Errorf("Shutdown took too long: %v (expected <3s)", elapsed)
	}

	// Error is acceptable if timeout was reached
	if err != nil {
		t.Logf("Shutdown returned error (acceptable): %v", err)
	}
}

// TestShutdown_Idempotent verifies safe to call multiple times
func TestShutdown_Idempotent(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	// First shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("First shutdown returned error: %v", err)
	}

	// Second shutdown should be safe (no panic, no error)
	err = service.Shutdown(shutdownCtx)
	if err != nil {
		t.Logf("Second shutdown returned error (acceptable): %v", err)
	}

	// Third shutdown for good measure
	err = service.Shutdown(shutdownCtx)
	if err != nil {
		t.Logf("Third shutdown returned error (acceptable): %v", err)
	}
}

// TestShutdown_NoDeadlock verifies completes even with active sessions
func TestShutdown_NoDeadlock(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	// Create multiple long-running sessions
	for i := 0; i < 5; i++ {
		id, err := service.CreateSession(ctx, "Deadlock Test", "sleep 100", ".")
		if err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}

		err = service.StartSession(ctx, id, 80, 24)
		if err != nil {
			t.Fatalf("StartSession %d failed: %v", i, err)
		}
	}

	// Shutdown with timeout should complete without deadlock
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use a channel to detect deadlock
	done := make(chan error, 1)
	go func() {
		done <- service.Shutdown(shutdownCtx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("Shutdown completed with error (acceptable): %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Shutdown deadlocked (did not complete within 15s)")
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// countGoroutines returns the current goroutine count
func countGoroutines() int {
	return runtime.NumGoroutine()
}
