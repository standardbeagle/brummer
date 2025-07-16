package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestInstanceDiscovery(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")

	// Create discovery system
	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer discovery.Stop()

	// Start discovery
	discovery.Start()

	// Test 1: Initial state should be empty
	instances := discovery.GetInstances()
	if len(instances) != 0 {
		t.Errorf("Expected 0 instances, got %d", len(instances))
	}

	// Test 2: Add an instance
	instance1 := &Instance{
		ID:        "test-instance-1",
		Name:      "Test Instance 1",
		Directory: "/test/dir1",
		Port:      7777,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        12345,
			Executable: "brum",
		},
	}

	if err := RegisterInstance(instancesDir, instance1); err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	// Wait for file watcher to pick up the change
	time.Sleep(100 * time.Millisecond)

	instances = discovery.GetInstances()
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(instances))
	}

	if inst, ok := instances["test-instance-1"]; !ok {
		t.Error("Instance test-instance-1 not found")
	} else {
		if inst.Name != "Test Instance 1" {
			t.Errorf("Expected name 'Test Instance 1', got '%s'", inst.Name)
		}
		if inst.Port != 7777 {
			t.Errorf("Expected port 7777, got %d", inst.Port)
		}
	}

	// Test 3: Add another instance
	instance2 := &Instance{
		ID:        "test-instance-2",
		Name:      "Test Instance 2",
		Directory: "/test/dir2",
		Port:      7778,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        12346,
			Executable: "brum",
		},
	}

	if err := RegisterInstance(instancesDir, instance2); err != nil {
		t.Fatalf("Failed to register instance2: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	instances = discovery.GetInstances()
	if len(instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instances))
	}

	// Test 4: Remove an instance
	if err := UnregisterInstance(instancesDir, "test-instance-1"); err != nil {
		t.Fatalf("Failed to unregister instance: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	instances = discovery.GetInstances()
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance after removal, got %d", len(instances))
	}

	if _, ok := instances["test-instance-1"]; ok {
		t.Error("Instance test-instance-1 should have been removed")
	}

	if _, ok := instances["test-instance-2"]; !ok {
		t.Error("Instance test-instance-2 should still exist")
	}
}

func TestInstanceUpdateCallback(t *testing.T) {
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")

	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer discovery.Stop()

	discovery.Start()

	// Set up callback to track updates
	var mu sync.Mutex
	var updateCount int
	var lastInstances map[string]*Instance

	discovery.OnUpdate(func(instances map[string]*Instance) {
		mu.Lock()
		defer mu.Unlock()
		updateCount++
		lastInstances = instances
	})

	// Register an instance
	instance := &Instance{
		ID:        "callback-test",
		Name:      "Callback Test",
		Directory: "/test",
		Port:      9999,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        12347,
			Executable: "brum",
		},
	}

	if err := RegisterInstance(instancesDir, instance); err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	// Wait for callback
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	if updateCount != 1 {
		t.Errorf("Expected 1 update callback, got %d", updateCount)
	}
	if len(lastInstances) != 1 {
		t.Errorf("Expected 1 instance in callback, got %d", len(lastInstances))
	}
	mu.Unlock()
}

func TestAtomicFileOperations(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Test basic atomic file operations through AtomicFileOperations
	testData := []byte("Hello, World!")
	testFile := filepath.Join(tempDir, "test.txt")

	if err := afo.atomicWriteFile(testFile, testData, 0644); err != nil {
		t.Fatalf("atomicWriteFile failed: %v", err)
	}

	// Verify file contents
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected '%s', got '%s'", testData, data)
	}
}

func TestInstancePingUpdate(t *testing.T) {
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")

	// Register an instance
	instance := &Instance{
		ID:        "ping-test",
		Name:      "Ping Test",
		Directory: "/test",
		Port:      8888,
		StartedAt: time.Now(),
		LastPing:  time.Now().Add(-1 * time.Hour), // Old ping time
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        12348,
			Executable: "brum",
		},
	}

	if err := RegisterInstance(instancesDir, instance); err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	// Update ping
	beforeUpdate := time.Now()
	if err := UpdateInstancePing(instancesDir, "ping-test"); err != nil {
		t.Fatalf("Failed to update ping: %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(filepath.Join(instancesDir, "ping-test.json"))
	if err != nil {
		t.Fatalf("Failed to read instance file: %v", err)
	}

	var updated Instance
	if err := json.Unmarshal(data, &updated); err != nil {
		t.Fatalf("Failed to unmarshal instance: %v", err)
	}

	if updated.LastPing.Before(beforeUpdate) {
		t.Error("LastPing was not updated")
	}
}

func TestGetDefaultInstancesDir(t *testing.T) {
	// Test with XDG_RUNTIME_DIR set
	oldXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", oldXDG)

	os.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	dir := GetDefaultInstancesDir()
	expected := "/run/user/1000/brummer/instances"
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}

	// Test without XDG_RUNTIME_DIR
	os.Unsetenv("XDG_RUNTIME_DIR")
	dir = GetDefaultInstancesDir()
	if !filepath.IsAbs(dir) {
		t.Error("Expected absolute path")
	}
	if !filepath.HasPrefix(dir, os.TempDir()) {
		t.Errorf("Expected path under temp dir, got %s", dir)
	}
}
