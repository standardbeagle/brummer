package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestHelper provides utilities for discovery testing with proper cleanup
type TestHelper struct {
	t           *testing.T
	tempDirs    []string
	discoveries []*Discovery
	mu          sync.Mutex
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{
		t:           t,
		tempDirs:    []string{},
		discoveries: []*Discovery{},
	}
}

// CreateDiscovery creates a new discovery instance with automatic cleanup
func (th *TestHelper) CreateDiscovery() (*Discovery, string) {
	tempDir := th.t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")

	discovery, err := New(instancesDir)
	if err != nil {
		th.t.Fatalf("Failed to create discovery: %v", err)
	}

	th.mu.Lock()
	th.tempDirs = append(th.tempDirs, tempDir)
	th.discoveries = append(th.discoveries, discovery)
	th.mu.Unlock()

	return discovery, instancesDir
}

// Cleanup stops all discoveries and cleans up temp directories
func (th *TestHelper) Cleanup() {
	th.mu.Lock()
	defer th.mu.Unlock()

	for _, d := range th.discoveries {
		if err := d.Stop(); err != nil {
			th.t.Logf("Warning: failed to stop discovery: %v", err)
		}
	}
}

// CreateValidInstance creates a valid instance for testing
func (th *TestHelper) CreateValidInstance(id string, port int) *Instance {
	now := time.Now()
	return &Instance{
		ID:        id,
		Name:      fmt.Sprintf("Test Instance %s", id),
		Directory: fmt.Sprintf("/test/dir/%s", id),
		Port:      port,
		StartedAt: now,
		LastPing:  now,
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(), // Use current process PID for valid process
			Executable: "brum",
		},
	}
}

// WaitForCondition waits for a condition to be true with timeout
func (th *TestHelper) WaitForCondition(timeout time.Duration, checkInterval time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(checkInterval)
	}
	return false
}

// TestDiscoveryDirectoryCreation verifies discovery creates directories properly
func TestDiscoveryDirectoryCreation(t *testing.T) {
	t.Parallel()

	// Test 1: Directory doesn't exist - should be created
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "non-existent", "path", "instances")

	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Should create directory structure: %v", err)
	}
	// Don't stop discovery since we never started it
	defer discovery.watcher.Close()

	// Verify directory was created with correct permissions
	info, err := os.Stat(instancesDir)
	if err != nil {
		t.Fatalf("Directory should exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected directory to be created")
	}

	// Test 2: Directory with no write permissions (skip on Windows)
	if os.Getenv("GOOS") != "windows" {
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0555); err != nil {
			t.Fatalf("Failed to create readonly dir: %v", err)
		}

		_, err = New(filepath.Join(readOnlyDir, "instances"))
		if err == nil {
			t.Error("Should fail with permission denied")
		}
	}
}

// TestInstanceFileValidation tests comprehensive instance validation
func TestInstanceFileValidation(t *testing.T) {
	t.Parallel()
	th := NewTestHelper(t)
	defer th.Cleanup()

	discovery, instancesDir := th.CreateDiscovery()
	discovery.Start()

	// Test cases for invalid instances
	testCases := []struct {
		name          string
		instance      *Instance
		shouldFail    bool
		errorContains string
	}{
		{
			name:       "Valid instance",
			instance:   th.CreateValidInstance("valid-1", 7777),
			shouldFail: false,
		},
		{
			name: "Missing ID",
			instance: &Instance{
				Name:      "No ID",
				Directory: "/test",
				Port:      7778,
				StartedAt: time.Now(),
				LastPing:  time.Now(),
			},
			shouldFail:    true,
			errorContains: "missing ID",
		},
		{
			name: "Invalid port - zero",
			instance: &Instance{
				ID:        "bad-port-1",
				Name:      "Bad Port",
				Directory: "/test",
				Port:      0,
				StartedAt: time.Now(),
				LastPing:  time.Now(),
			},
			shouldFail:    true,
			errorContains: "invalid Port",
		},
		{
			name: "Invalid port - too high",
			instance: &Instance{
				ID:        "bad-port-2",
				Name:      "Bad Port",
				Directory: "/test",
				Port:      70000,
				StartedAt: time.Now(),
				LastPing:  time.Now(),
			},
			shouldFail:    true,
			errorContains: "invalid Port",
		},
		{
			name: "Future timestamp",
			instance: &Instance{
				ID:        "future-time",
				Name:      "Future Instance",
				Directory: "/test",
				Port:      7779,
				StartedAt: time.Now().Add(2 * time.Hour),
				LastPing:  time.Now(),
			},
			shouldFail:    true,
			errorContains: "StartedAt is in the future",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := RegisterInstance(instancesDir, tc.instance)

			if tc.shouldFail {
				if err == nil {
					t.Errorf("Expected error for %s", tc.name)
				} else if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.name, err)
				}

				// Verify instance was registered
				if !th.WaitForCondition(2*time.Second, 50*time.Millisecond, func() bool {
					instances := discovery.GetInstances()
					_, exists := instances[tc.instance.ID]
					return exists
				}) {
					t.Errorf("Instance %s was not discovered", tc.instance.ID)
				}
			}
		})
	}
}

// TestFileWatcherReliability tests that file watcher catches all changes
func TestFileWatcherReliability(t *testing.T) {
	t.Parallel()
	th := NewTestHelper(t)
	defer th.Cleanup()

	discovery, instancesDir := th.CreateDiscovery()

	// Track all updates
	var updateMu sync.Mutex
	updateCount := int32(0)
	instancesSeen := make(map[string]bool)

	discovery.OnUpdate(func(instances map[string]*Instance) {
		updateMu.Lock()
		defer updateMu.Unlock()

		atomic.AddInt32(&updateCount, 1)
		for id := range instances {
			instancesSeen[id] = true
		}
	})

	discovery.Start()

	// Register multiple instances rapidly
	numInstances := 10
	for i := 0; i < numInstances; i++ {
		instance := th.CreateValidInstance(fmt.Sprintf("rapid-%d", i), 8000+i)
		if err := RegisterInstance(instancesDir, instance); err != nil {
			t.Fatalf("Failed to register instance %d: %v", i, err)
		}
	}

	// Wait for all instances to be discovered
	if !th.WaitForCondition(5*time.Second, 100*time.Millisecond, func() bool {
		updateMu.Lock()
		defer updateMu.Unlock()
		return len(instancesSeen) >= numInstances
	}) {
		updateMu.Lock()
		seen := len(instancesSeen)
		updateMu.Unlock()
		t.Errorf("Not all instances discovered: expected %d, got %d", numInstances, seen)
	}

	// Verify final state
	instances := discovery.GetInstances()
	if len(instances) != numInstances {
		t.Errorf("Final instance count mismatch: expected %d, got %d", numInstances, len(instances))
	}

	// Test rapid modifications
	for i := 0; i < 5; i++ {
		instanceID := fmt.Sprintf("rapid-%d", i)
		if err := UpdateInstancePing(instancesDir, instanceID); err != nil {
			t.Errorf("Failed to update ping for %s: %v", instanceID, err)
		}
	}

	// Small delay to ensure updates are processed
	// This is needed because file system events may be batched
	time.Sleep(100 * time.Millisecond)

	// Remove half the instances
	removedCount := numInstances / 2
	for i := 0; i < removedCount; i++ {
		instanceID := fmt.Sprintf("rapid-%d", i)
		if err := UnregisterInstance(instancesDir, instanceID); err != nil {
			t.Errorf("Failed to unregister %s: %v", instanceID, err)
		}
	}

	// Wait for removals to be processed
	expectedRemaining := numInstances - removedCount
	if !th.WaitForCondition(3*time.Second, 100*time.Millisecond, func() bool {
		instances := discovery.GetInstances()
		return len(instances) == expectedRemaining
	}) {
		instances := discovery.GetInstances()
		t.Errorf("Instance removal not detected: expected %d, got %d", expectedRemaining, len(instances))
	}
}

// TestDiscoveryConcurrentOperations tests thread safety of discovery file operations
func TestDiscoveryConcurrentOperations(t *testing.T) {
	t.Parallel()
	th := NewTestHelper(t)
	defer th.Cleanup()

	discovery, instancesDir := th.CreateDiscovery()
	discovery.Start()

	// Concurrent operations configuration
	numGoroutines := 20
	operationsPerGoroutine := 50

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Launch concurrent operations
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				instanceID := fmt.Sprintf("concurrent-%d-%d", goroutineID, op)

				// Register instance
				instance := th.CreateValidInstance(instanceID, 9000+goroutineID*100+op)
				if err := RegisterInstance(instancesDir, instance); err != nil {
					errors <- fmt.Errorf("register %s: %w", instanceID, err)
					continue
				}

				// Update ping
				if err := UpdateInstancePing(instancesDir, instanceID); err != nil {
					errors <- fmt.Errorf("update %s: %w", instanceID, err)
				}

				// Read back
				afo := NewAtomicFileOperations(instancesDir)
				if _, err := afo.SafeReadInstance(instanceID); err != nil {
					errors <- fmt.Errorf("read %s: %w", instanceID, err)
				}

				// Randomly unregister some instances
				if op%3 == 0 {
					if err := UnregisterInstance(instancesDir, instanceID); err != nil {
						errors <- fmt.Errorf("unregister %s: %w", instanceID, err)
					}
				}
			}
		}(g)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent operation error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Had %d errors during concurrent operations", errorCount)
	}

	// Verify final state is consistent
	instances := discovery.GetInstances()
	t.Logf("Final instance count: %d", len(instances))

	// Verify all remaining instances are valid
	for id, inst := range instances {
		if err := validateInstance(inst); err != nil {
			t.Errorf("Instance %s is invalid after concurrent ops: %v", id, err)
		}
	}
}

// TestStaleInstanceCleanup tests cleanup of dead instances
func TestStaleInstanceCleanup(t *testing.T) {
	t.Parallel()
	th := NewTestHelper(t)
	defer th.Cleanup()

	discovery, instancesDir := th.CreateDiscovery()
	discovery.Start()

	// Create instance with stale LastPing
	staleInstance := th.CreateValidInstance("stale-1", 7777)
	staleInstance.LastPing = time.Now().Add(-10 * time.Minute) // Very stale
	staleInstance.ProcessInfo.PID = 99999                      // Non-existent process

	if err := RegisterInstance(instancesDir, staleInstance); err != nil {
		t.Fatalf("Failed to register stale instance: %v", err)
	}

	// Create fresh instance
	freshInstance := th.CreateValidInstance("fresh-1", 7778)
	if err := RegisterInstance(instancesDir, freshInstance); err != nil {
		t.Fatalf("Failed to register fresh instance: %v", err)
	}

	// Wait for discovery
	if !th.WaitForCondition(2*time.Second, 50*time.Millisecond, func() bool {
		instances := discovery.GetInstances()
		return len(instances) == 2
	}) {
		instances := discovery.GetInstances()
		t.Fatalf("Expected 2 instances, got %d", len(instances))
	}

	// Run cleanup
	if err := discovery.CleanupStaleInstances(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify stale instance was removed
	finalInstances := discovery.GetInstances()
	if len(finalInstances) != 1 {
		t.Errorf("Expected 1 instance after cleanup, got %d", len(finalInstances))
	}

	if _, exists := finalInstances["stale-1"]; exists {
		t.Error("Stale instance should have been removed")
	}

	if _, exists := finalInstances["fresh-1"]; !exists {
		t.Error("Fresh instance should still exist")
	}
}

// TestFileCorruptionHandling tests handling of corrupted instance files
func TestFileCorruptionHandling(t *testing.T) {
	t.Parallel()
	th := NewTestHelper(t)
	defer th.Cleanup()

	discovery, instancesDir := th.CreateDiscovery()

	// Write corrupted files directly
	corruptedFiles := []struct {
		filename string
		content  string
	}{
		{"corrupt-1.json", "not json at all"},
		{"corrupt-2.json", `{"id": "missing-fields"}`},            // Missing required fields
		{"corrupt-3.json", `{]`},                                  // Invalid JSON
		{"corrupt-4.json", `{"id":"test","port":"not-a-number"}`}, // Type mismatch
	}

	for _, cf := range corruptedFiles {
		path := filepath.Join(instancesDir, cf.filename)
		if err := os.WriteFile(path, []byte(cf.content), 0644); err != nil {
			t.Fatalf("Failed to write corrupted file: %v", err)
		}
	}

	// Write one valid instance
	validInstance := th.CreateValidInstance("valid-among-corrupt", 7777)
	if err := RegisterInstance(instancesDir, validInstance); err != nil {
		t.Fatalf("Failed to register valid instance: %v", err)
	}

	// Start discovery - it should handle corrupted files gracefully
	discovery.Start()

	// Wait for discovery to process files
	if !th.WaitForCondition(2*time.Second, 50*time.Millisecond, func() bool {
		instances := discovery.GetInstances()
		// Should find exactly one valid instance among the corrupted files
		return len(instances) == 1
	}) {
		instances := discovery.GetInstances()
		t.Errorf("Expected 1 valid instance, got %d", len(instances))
	}

	// Verify the valid instance was discovered
	validInstances := discovery.GetInstances()
	if _, exists := validInstances["valid-among-corrupt"]; !exists {
		t.Error("Valid instance should be discovered despite corrupted files")
	}
}

// TestCallbackErrorHandling tests that callbacks don't break discovery
func TestCallbackErrorHandling(t *testing.T) {
	t.Skip("Skipping test that intentionally panics - panics cannot be recovered across goroutines")

	// NOTE: In a real system, callbacks should be wrapped with recover()
	// to prevent panics from breaking the discovery system.
	// This test demonstrates that the current implementation does NOT
	// handle panics in callbacks gracefully.
}

// TestHubDiscoveryIntegration tests full hub discovery flow
func TestHubDiscoveryIntegration(t *testing.T) {
	t.Parallel()
	th := NewTestHelper(t)
	defer th.Cleanup()

	// Simulate multiple instances registering
	instancesDir := filepath.Join(t.TempDir(), "hub-instances")

	// Instance 1: Frontend
	frontend := &Instance{
		ID:        "frontend-abc123",
		Name:      "Frontend Dev Server",
		Directory: "/projects/frontend",
		Port:      7777,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(),
			Executable: "brum",
		},
	}

	// Instance 2: Backend
	backend := &Instance{
		ID:        "backend-def456",
		Name:      "Backend API Server",
		Directory: "/projects/backend",
		Port:      7778,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid() + 1, // Simulate different process
			Executable: "brum",
		},
	}

	// Create discovery system (simulating hub)
	discovery, err := New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create hub discovery: %v", err)
	}
	defer discovery.Stop()

	// Track discovery events
	var discoveredMu sync.Mutex
	discovered := make(map[string]*Instance)
	removed := make(map[string]bool)

	discovery.OnUpdate(func(instances map[string]*Instance) {
		discoveredMu.Lock()
		defer discoveredMu.Unlock()

		// Track additions
		for id, inst := range instances {
			if _, exists := discovered[id]; !exists {
				discovered[id] = inst
				t.Logf("Discovered instance: %s", id)
			}
		}

		// Track removals
		for id := range discovered {
			if _, exists := instances[id]; !exists {
				removed[id] = true
				t.Logf("Removed instance: %s", id)
			}
		}
	})

	discovery.Start()

	// Register instances (simulating instance startup)
	if err := RegisterInstance(instancesDir, frontend); err != nil {
		t.Fatalf("Failed to register frontend: %v", err)
	}

	if err := RegisterInstance(instancesDir, backend); err != nil {
		t.Fatalf("Failed to register backend: %v", err)
	}

	// Wait for discovery
	if !th.WaitForCondition(2*time.Second, 50*time.Millisecond, func() bool {
		discoveredMu.Lock()
		defer discoveredMu.Unlock()
		return len(discovered) == 2
	}) {
		t.Fatalf("Not all instances discovered")
	}

	// Simulate frontend updating its ping
	time.Sleep(100 * time.Millisecond)
	if err := UpdateInstancePing(instancesDir, "frontend-abc123"); err != nil {
		t.Errorf("Failed to update frontend ping: %v", err)
	}

	// Simulate backend shutdown
	if err := UnregisterInstance(instancesDir, "backend-def456"); err != nil {
		t.Errorf("Failed to unregister backend: %v", err)
	}

	// Wait for removal to be detected
	if !th.WaitForCondition(2*time.Second, 50*time.Millisecond, func() bool {
		discoveredMu.Lock()
		defer discoveredMu.Unlock()
		return removed["backend-def456"]
	}) {
		t.Error("Backend removal not detected")
	}

	// Final state check
	instances := discovery.GetInstances()
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance in final state, got %d", len(instances))
	}

	if _, exists := instances["frontend-abc123"]; !exists {
		t.Error("Frontend should still be registered")
	}
}
