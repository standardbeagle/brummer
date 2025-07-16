package discovery

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestAtomicFileOperationsCreation(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	if afo == nil {
		t.Fatal("AtomicFileOperations should not be nil")
	}

	if afo.instancesDir != tempDir {
		t.Errorf("Expected instancesDir %s, got %s", tempDir, afo.instancesDir)
	}

	if afo.lockTimeout != 30*time.Second {
		t.Errorf("Expected default lock timeout 30s, got %v", afo.lockTimeout)
	}
}

func TestAtomicFileOperationsLockTimeout(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Test getter
	if afo.GetLockTimeout() != 30*time.Second {
		t.Errorf("Expected lock timeout 30s, got %v", afo.GetLockTimeout())
	}

	// Test setter
	newTimeout := 10 * time.Second
	afo.SetLockTimeout(newTimeout)

	if afo.GetLockTimeout() != newTimeout {
		t.Errorf("Expected lock timeout %v, got %v", newTimeout, afo.GetLockTimeout())
	}
}

func TestSafeRegisterInstance(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	instance := &Instance{
		ID:        "test-instance",
		Name:      "Test Instance",
		Directory: "/test/dir",
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

	err := afo.SafeRegisterInstance(instance)
	if err != nil {
		t.Errorf("SafeRegisterInstance should not return error: %v", err)
	}

	// Verify file was created
	filename := filepath.Join(tempDir, "test-instance.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("Instance file should have been created")
	}

	// Verify file contents
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("Failed to read instance file: %v", err)
	}

	var readInstance Instance
	if err := json.Unmarshal(data, &readInstance); err != nil {
		t.Errorf("Failed to unmarshal instance data: %v", err)
	}

	if readInstance.ID != instance.ID {
		t.Errorf("Expected ID %s, got %s", instance.ID, readInstance.ID)
	}
}

func TestSafeReadInstance(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Register an instance first
	instance := &Instance{
		ID:        "read-test",
		Name:      "Read Test",
		Directory: "/test/dir",
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

	err := afo.SafeRegisterInstance(instance)
	if err != nil {
		t.Errorf("SafeRegisterInstance should not return error: %v", err)
	}

	// Read it back
	readInstance, err := afo.SafeReadInstance("read-test")
	if err != nil {
		t.Errorf("SafeReadInstance should not return error: %v", err)
	}

	if readInstance.ID != instance.ID {
		t.Errorf("Expected ID %s, got %s", instance.ID, readInstance.ID)
	}

	if readInstance.Port != instance.Port {
		t.Errorf("Expected Port %d, got %d", instance.Port, readInstance.Port)
	}

	// Test reading non-existent instance
	_, err = afo.SafeReadInstance("nonexistent")
	if err == nil {
		t.Error("SafeReadInstance should return error for nonexistent instance")
	}
}

func TestSafeUpdateInstancePing(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Register an instance first
	instance := &Instance{
		ID:        "ping-test",
		Name:      "Ping Test",
		Directory: "/test/dir",
		Port:      7779,
		StartedAt: time.Now(),
		LastPing:  time.Now().Add(-5 * time.Minute), // Old ping time
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        12347,
			Executable: "brum",
		},
	}

	err := afo.SafeRegisterInstance(instance)
	if err != nil {
		t.Errorf("SafeRegisterInstance should not return error: %v", err)
	}

	originalPing := instance.LastPing

	// Update ping time
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	err = afo.SafeUpdateInstancePing("ping-test")
	if err != nil {
		t.Errorf("SafeUpdateInstancePing should not return error: %v", err)
	}

	// Read back and verify ping was updated
	readInstance, err := afo.SafeReadInstance("ping-test")
	if err != nil {
		t.Errorf("SafeReadInstance should not return error: %v", err)
	}

	if !readInstance.LastPing.After(originalPing) {
		t.Error("LastPing should have been updated to a later time")
	}
}

func TestSafeUnregisterInstance(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Register an instance first
	instance := &Instance{
		ID:        "unregister-test",
		Name:      "Unregister Test",
		Directory: "/test/dir",
		Port:      7780,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        12348,
			Executable: "brum",
		},
	}

	err := afo.SafeRegisterInstance(instance)
	if err != nil {
		t.Errorf("SafeRegisterInstance should not return error: %v", err)
	}

	// Verify file exists
	filename := filepath.Join(tempDir, "unregister-test.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("Instance file should exist before unregistering")
	}

	// Unregister
	err = afo.SafeUnregisterInstance("unregister-test")
	if err != nil {
		t.Errorf("SafeUnregisterInstance should not return error: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Error("Instance file should have been removed")
	}

	// Test unregistering non-existent instance (should not error)
	err = afo.SafeUnregisterInstance("nonexistent")
	if err != nil {
		t.Errorf("SafeUnregisterInstance should not return error for nonexistent instance: %v", err)
	}
}

func TestSafeListInstances(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Test empty directory
	instances, err := afo.SafeListInstances()
	if err != nil {
		t.Errorf("SafeListInstances should not return error: %v", err)
	}

	if len(instances) != 0 {
		t.Errorf("Expected empty instances map, got %d instances", len(instances))
	}

	// Register multiple instances
	for i := 0; i < 3; i++ {
		instance := &Instance{
			ID:        "list-test-" + string(rune('a'+i)),
			Name:      "List Test " + string(rune('A'+i)),
			Directory: "/test/dir",
			Port:      7781 + i,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
			ProcessInfo: struct {
				PID        int    `json:"pid"`
				Executable string `json:"executable"`
			}{
				PID:        12349 + i,
				Executable: "brum",
			},
		}

		err := afo.SafeRegisterInstance(instance)
		if err != nil {
			t.Errorf("SafeRegisterInstance should not return error: %v", err)
		}
	}

	// List all instances
	instances, err = afo.SafeListInstances()
	if err != nil {
		t.Errorf("SafeListInstances should not return error: %v", err)
	}

	if len(instances) != 3 {
		t.Errorf("Expected 3 instances, got %d", len(instances))
	}

	// Verify instance IDs
	expectedIDs := []string{"list-test-a", "list-test-b", "list-test-c"}
	for _, expectedID := range expectedIDs {
		if _, exists := instances[expectedID]; !exists {
			t.Errorf("Expected instance %s not found", expectedID)
		}
	}
}

func TestConcurrentFileOperations(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Reduce lock timeout for faster testing
	afo.SetLockTimeout(5 * time.Second)

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperationsPerGoroutine := 5

	// Test concurrent registration
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperationsPerGoroutine; j++ {
				instance := &Instance{
					ID:        "concurrent-" + string(rune('a'+goroutineID)) + "-" + string(rune('0'+j)),
					Name:      "Concurrent Test",
					Directory: "/test/dir",
					Port:      8000 + goroutineID*100 + j,
					StartedAt: time.Now(),
					LastPing:  time.Now(),
					ProcessInfo: struct {
						PID        int    `json:"pid"`
						Executable string `json:"executable"`
					}{
						PID:        13000 + goroutineID*100 + j,
						Executable: "brum",
					},
				}

				err := afo.SafeRegisterInstance(instance)
				if err != nil {
					t.Errorf("Goroutine %d: SafeRegisterInstance should not return error: %v", goroutineID, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all instances were registered correctly
	instances, err := afo.SafeListInstances()
	if err != nil {
		t.Errorf("SafeListInstances should not return error: %v", err)
	}

	expectedCount := numGoroutines * numOperationsPerGoroutine
	if len(instances) != expectedCount {
		t.Errorf("Expected %d instances, got %d", expectedCount, len(instances))
	}
}

func TestConcurrentReadWriteOperations(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Register an initial instance
	instance := &Instance{
		ID:        "readwrite-test",
		Name:      "ReadWrite Test",
		Directory: "/test/dir",
		Port:      7785,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        12355,
			Executable: "brum",
		},
	}

	err := afo.SafeRegisterInstance(instance)
	if err != nil {
		t.Errorf("SafeRegisterInstance should not return error: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 5
	operationsPerGoroutine := 10

	// Half the goroutines do ping updates, half do reads
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		if i%2 == 0 {
			// Ping updater goroutines
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					err := afo.SafeUpdateInstancePing("readwrite-test")
					if err != nil {
						t.Errorf("Goroutine %d: SafeUpdateInstancePing should not return error: %v", goroutineID, err)
					}
					time.Sleep(1 * time.Millisecond) // Small delay
				}
			}(i)
		} else {
			// Reader goroutines
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					_, err := afo.SafeReadInstance("readwrite-test")
					if err != nil {
						t.Errorf("Goroutine %d: SafeReadInstance should not return error: %v", goroutineID, err)
					}
					time.Sleep(1 * time.Millisecond) // Small delay
				}
			}(i)
		}
	}

	wg.Wait()

	// Final verification - should be able to read the instance
	finalInstance, err := afo.SafeReadInstance("readwrite-test")
	if err != nil {
		t.Errorf("Final SafeReadInstance should not return error: %v", err)
	}

	if finalInstance.ID != "readwrite-test" {
		t.Errorf("Final instance should have correct ID")
	}
}

func TestIsInstanceFileCorrupted(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Test with valid instance
	instance := &Instance{
		ID:        "corruption-test",
		Name:      "Corruption Test",
		Directory: "/test/dir",
		Port:      7786,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        12356,
			Executable: "brum",
		},
	}

	err := afo.SafeRegisterInstance(instance)
	if err != nil {
		t.Errorf("SafeRegisterInstance should not return error: %v", err)
	}

	// Should not be corrupted
	if afo.IsInstanceFileCorrupted("corruption-test") {
		t.Error("Valid instance should not be considered corrupted")
	}

	// Create a corrupted file
	corruptedPath := filepath.Join(tempDir, "corrupted.json")
	err = os.WriteFile(corruptedPath, []byte("{invalid json"), 0600)
	if err != nil {
		t.Errorf("Failed to create corrupted file: %v", err)
	}

	// Should be corrupted
	if !afo.IsInstanceFileCorrupted("corrupted") {
		t.Error("Corrupted instance should be considered corrupted")
	}

	// Test with non-existent file
	if !afo.IsInstanceFileCorrupted("nonexistent") {
		t.Error("Non-existent instance should be considered corrupted")
	}
}

func TestLockTimeout(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Set very short timeout for testing
	afo.SetLockTimeout(100 * time.Millisecond)

	// Create a context that will acquire the lock indefinitely
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// First goroutine acquires lock and holds it
	lockAcquired := make(chan bool, 1)
	lockReleased := make(chan bool, 1)

	go func() {
		err := afo.withLock(func() error {
			lockAcquired <- true
			<-ctx.Done() // Hold lock until context is cancelled
			return nil
		})
		if err != nil {
			t.Errorf("First lock acquisition should not fail: %v", err)
		}
		lockReleased <- true
	}()

	// Wait for first goroutine to acquire lock
	<-lockAcquired

	// Second operation should timeout
	start := time.Now()
	err := afo.withLock(func() error {
		return nil
	})
	duration := time.Since(start)

	if err == nil {
		t.Error("Second lock operation should timeout")
	}

	// Should timeout in approximately the configured time
	if duration < 90*time.Millisecond || duration > 200*time.Millisecond {
		t.Errorf("Timeout duration should be around 100ms, got %v", duration)
	}

	// Release the first lock
	cancel()
	<-lockReleased
}

func TestAtomicWriteFileRecovery(t *testing.T) {
	tempDir := t.TempDir()
	afo := NewAtomicFileOperations(tempDir)

	// Test that atomic write cleans up properly on interruption
	testData := []byte("test data")
	targetPath := filepath.Join(tempDir, "test.json")

	// This should succeed
	err := afo.atomicWriteFile(targetPath, testData, DefaultFileMode)
	if err != nil {
		t.Errorf("atomicWriteFile should not return error: %v", err)
	}

	// Verify file exists and has correct content
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Errorf("Failed to read written file: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("File content mismatch: expected %s, got %s", testData, data)
	}

	// Verify no temp files left behind
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Errorf("Failed to read temp directory: %v", err)
	}

	for _, entry := range entries {
		if entry.Name() != "test.json" {
			t.Errorf("Unexpected file found: %s", entry.Name())
		}
	}
}