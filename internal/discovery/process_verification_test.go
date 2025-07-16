package discovery

import (
	"os"
	"testing"
	"time"
)

func TestIsProcessRunning(t *testing.T) {
	tempDir := t.TempDir()
	discovery := &Discovery{
		instancesDir: tempDir,
		instances:    make(map[string]*Instance),
	}

	// Test with current process (should be running)
	currentPID := os.Getpid()
	if !discovery.isProcessRunning(currentPID) {
		t.Errorf("Current process %d should be running", currentPID)
	}

	// Test with invalid PID (should not be running)
	if discovery.isProcessRunning(-1) {
		t.Error("Invalid PID -1 should not be running")
	}

	if discovery.isProcessRunning(0) {
		t.Error("PID 0 should not be running")
	}

	// Test with very high PID (very unlikely to exist on most systems)
	if discovery.isProcessRunning(999998) {
		t.Log("PID 999998 appears to be running (this is unusual but not necessarily an error)")
	}
}

func TestStaleInstanceCleanupWithProcessVerification(t *testing.T) {
	tempDir := t.TempDir()
	instancesDir := tempDir + "/instances"

	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer discovery.Stop()

	// Register an instance with a non-existent PID
	deadInstance := &Instance{
		ID:        "dead-process",
		Name:      "Dead Process",
		Directory: "/test",
		Port:      7777,
		StartedAt: time.Now(),
		LastPing:  time.Now(), // Recent ping, but process is dead
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        999999, // Very unlikely to exist
			Executable: "brum",
		},
	}

	err = RegisterInstance(instancesDir, deadInstance)
	if err != nil {
		t.Fatalf("Failed to register dead instance: %v", err)
	}

	// Register an instance with the current process PID (alive)
	aliveInstance := &Instance{
		ID:        "alive-process",
		Name:      "Alive Process",
		Directory: "/test",
		Port:      7778,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(), // Current process
			Executable: "brum",
		},
	}

	err = RegisterInstance(instancesDir, aliveInstance)
	if err != nil {
		t.Fatalf("Failed to register alive instance: %v", err)
	}

	// Initial scan to load instances
	discovery.Start()
	time.Sleep(100 * time.Millisecond)

	// Verify both instances are loaded
	instances := discovery.GetInstances()
	if len(instances) != 2 {
		t.Errorf("Expected 2 instances before cleanup, got %d", len(instances))
	}

	// Run cleanup - should remove dead process but keep alive process
	err = discovery.CleanupStaleInstances()
	if err != nil {
		t.Errorf("CleanupStaleInstances should not return error: %v", err)
	}

	// Verify dead instance was removed but alive instance remains
	instances = discovery.GetInstances()
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance after cleanup, got %d", len(instances))
	}

	if _, exists := instances["dead-process"]; exists {
		t.Error("Dead process instance should have been removed")
	}

	if _, exists := instances["alive-process"]; !exists {
		t.Error("Alive process instance should still exist")
	}
}

func TestStaleInstanceCleanupWithTimeBasedStale(t *testing.T) {
	tempDir := t.TempDir()
	instancesDir := tempDir + "/instances"

	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer discovery.Stop()

	// Register an instance with old ping time but valid PID
	staleInstance := &Instance{
		ID:        "stale-time",
		Name:      "Stale Time",
		Directory: "/test",
		Port:      7777,
		StartedAt: time.Now(),
		LastPing:  time.Now().Add(-StaleInstanceTimeout - time.Minute), // Very old
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(), // Current process (alive)
			Executable: "brum",
		},
	}

	err = RegisterInstance(instancesDir, staleInstance)
	if err != nil {
		t.Fatalf("Failed to register stale instance: %v", err)
	}

	// Register an instance with recent ping time and valid PID
	freshInstance := &Instance{
		ID:        "fresh-time",
		Name:      "Fresh Time",
		Directory: "/test",
		Port:      7778,
		StartedAt: time.Now(),
		LastPing:  time.Now(), // Recent ping
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(), // Current process (alive)
			Executable: "brum",
		},
	}

	err = RegisterInstance(instancesDir, freshInstance)
	if err != nil {
		t.Fatalf("Failed to register fresh instance: %v", err)
	}

	// Initial scan to load instances
	discovery.Start()
	time.Sleep(100 * time.Millisecond)

	// Verify both instances are loaded
	instances := discovery.GetInstances()
	if len(instances) != 2 {
		t.Errorf("Expected 2 instances before cleanup, got %d", len(instances))
	}

	// Run cleanup - should remove stale instance but keep fresh instance
	err = discovery.CleanupStaleInstances()
	if err != nil {
		t.Errorf("CleanupStaleInstances should not return error: %v", err)
	}

	// Verify stale instance was removed but fresh instance remains
	instances = discovery.GetInstances()
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance after cleanup, got %d", len(instances))
	}

	if _, exists := instances["stale-time"]; exists {
		t.Error("Stale time instance should have been removed")
	}

	if _, exists := instances["fresh-time"]; !exists {
		t.Error("Fresh time instance should still exist")
	}
}

func TestStaleInstanceCleanupBothConditions(t *testing.T) {
	tempDir := t.TempDir()
	instancesDir := tempDir + "/instances"

	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer discovery.Stop()

	// Register an instance that meets both stale conditions (old ping + dead process)
	doubleStaleInstance := &Instance{
		ID:        "double-stale",
		Name:      "Double Stale",
		Directory: "/test",
		Port:      7777,
		StartedAt: time.Now(),
		LastPing:  time.Now().Add(-StaleInstanceTimeout - time.Minute), // Very old
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        999999, // Very unlikely to exist
			Executable: "brum",
		},
	}

	err = RegisterInstance(instancesDir, doubleStaleInstance)
	if err != nil {
		t.Fatalf("Failed to register double stale instance: %v", err)
	}

	// Register a healthy instance
	healthyInstance := &Instance{
		ID:        "healthy",
		Name:      "Healthy",
		Directory: "/test",
		Port:      7778,
		StartedAt: time.Now(),
		LastPing:  time.Now(), // Recent ping
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(), // Current process (alive)
			Executable: "brum",
		},
	}

	err = RegisterInstance(instancesDir, healthyInstance)
	if err != nil {
		t.Fatalf("Failed to register healthy instance: %v", err)
	}

	// Initial scan to load instances
	discovery.Start()
	time.Sleep(100 * time.Millisecond)

	// Verify both instances are loaded
	instances := discovery.GetInstances()
	if len(instances) != 2 {
		t.Errorf("Expected 2 instances before cleanup, got %d", len(instances))
	}

	// Run cleanup
	err = discovery.CleanupStaleInstances()
	if err != nil {
		t.Errorf("CleanupStaleInstances should not return error: %v", err)
	}

	// Verify only healthy instance remains
	instances = discovery.GetInstances()
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance after cleanup, got %d", len(instances))
	}

	if _, exists := instances["double-stale"]; exists {
		t.Error("Double stale instance should have been removed")
	}

	if _, exists := instances["healthy"]; !exists {
		t.Error("Healthy instance should still exist")
	}
}