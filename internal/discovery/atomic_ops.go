package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// AtomicFileOperations provides thread-safe file operations for instance management
type AtomicFileOperations struct {
	instancesDir string
	lockTimeout  time.Duration
}

// NewAtomicFileOperations creates a new atomic file operations manager
func NewAtomicFileOperations(instancesDir string) *AtomicFileOperations {
	return &AtomicFileOperations{
		instancesDir: instancesDir,
		lockTimeout:  30 * time.Second, // Prevent deadlocks
	}
}

// SafeUpdateInstance atomically updates an instance with proper file locking
func (afo *AtomicFileOperations) SafeUpdateInstance(instance *Instance) error {
	return afo.withLock(func() error {
		return afo.atomicWriteInstance(instance)
	})
}

// SafeUpdateInstancePing atomically updates the last ping time for an instance
func (afo *AtomicFileOperations) SafeUpdateInstancePing(instanceID string) error {
	return afo.withLock(func() error {
		// Read existing instance data
		instance, err := afo.readInstanceLocked(instanceID)
		if err != nil {
			return fmt.Errorf("failed to read instance: %w", err)
		}

		// Update last ping time
		instance.LastPing = time.Now()

		// Write back atomically
		return afo.atomicWriteInstance(instance)
	})
}

// SafeRegisterInstance atomically registers a new instance
func (afo *AtomicFileOperations) SafeRegisterInstance(instance *Instance) error {
	return afo.withLock(func() error {
		// Validate instance before writing
		if err := validateInstance(instance); err != nil {
			return fmt.Errorf("invalid instance: %w", err)
		}

		// Ensure instances directory exists with proper permissions
		if err := os.MkdirAll(afo.instancesDir, DefaultDirMode); err != nil {
			return fmt.Errorf("failed to create instances directory: %w", err)
		}

		return afo.atomicWriteInstance(instance)
	})
}

// SafeUnregisterInstance atomically removes an instance registration
func (afo *AtomicFileOperations) SafeUnregisterInstance(instanceID string) error {
	return afo.withLock(func() error {
		filename := fmt.Sprintf("%s.json", instanceID)
		path := filepath.Join(afo.instancesDir, filename)

		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove instance file: %w", err)
		}

		return nil
	})
}

// SafeReadInstance atomically reads an instance with proper locking
func (afo *AtomicFileOperations) SafeReadInstance(instanceID string) (*Instance, error) {
	var instance *Instance
	var err error

	lockErr := afo.withLock(func() error {
		instance, err = afo.readInstanceLocked(instanceID)
		return err
	})

	if lockErr != nil {
		return nil, lockErr
	}

	return instance, nil
}

// SafeListInstances atomically lists all instances with proper locking
func (afo *AtomicFileOperations) SafeListInstances() (map[string]*Instance, error) {
	var instances map[string]*Instance
	var err error

	lockErr := afo.withLock(func() error {
		instances, err = afo.listInstancesLocked()
		return err
	})

	if lockErr != nil {
		return nil, lockErr
	}

	return instances, nil
}

// withLock executes a function while holding an exclusive file lock
func (afo *AtomicFileOperations) withLock(fn func() error) error {
	// Create lock file path
	lockFile := filepath.Join(afo.instancesDir, ".discovery.lock")
	
	// Ensure the lock directory exists
	if err := os.MkdirAll(afo.instancesDir, DefaultDirMode); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Create file lock
	fileLock := flock.New(lockFile)

	// Set timeout to prevent indefinite blocking
	ctx, cancel := context.WithTimeout(context.Background(), afo.lockTimeout)
	defer cancel()

	// Acquire lock with timeout
	locked, err := fileLock.TryLockContext(ctx, 100*time.Millisecond) // Retry every 100ms
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("failed to acquire lock within timeout (%v)", afo.lockTimeout)
	}

	// Ensure lock is released
	defer func() {
		if unlockErr := fileLock.Unlock(); unlockErr != nil {
			// Log error but don't override the main error
			// In production, this should be logged properly
			fmt.Printf("Warning: failed to release lock: %v\n", unlockErr)
		}
	}()

	// Execute the function while holding the lock
	return fn()
}

// atomicWriteInstance writes an instance to disk atomically (must be called within a lock)
func (afo *AtomicFileOperations) atomicWriteInstance(instance *Instance) error {
	// Marshal instance data
	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instance data: %w", err)
	}

	// Write atomically using temp file + rename
	filename := fmt.Sprintf("%s.json", instance.ID)
	finalPath := filepath.Join(afo.instancesDir, filename)

	// Use enhanced atomic write with proper error handling
	if err := afo.atomicWriteFile(finalPath, data, DefaultFileMode); err != nil {
		return fmt.Errorf("failed to write instance file: %w", err)
	}

	return nil
}

// readInstanceLocked reads an instance from disk (must be called within a lock)
func (afo *AtomicFileOperations) readInstanceLocked(instanceID string) (*Instance, error) {
	filename := fmt.Sprintf("%s.json", instanceID)
	path := filepath.Join(afo.instancesDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read instance file: %w", err)
	}

	var instance Instance
	if err := json.Unmarshal(data, &instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance data: %w", err)
	}

	// Validate instance data
	if err := validateInstance(&instance); err != nil {
		return nil, fmt.Errorf("invalid instance data: %w", err)
	}

	return &instance, nil
}

// listInstancesLocked lists all instances from disk (must be called within a lock)
func (afo *AtomicFileOperations) listInstancesLocked() (map[string]*Instance, error) {
	instances := make(map[string]*Instance)

	// Check if directory exists
	if _, err := os.Stat(afo.instancesDir); os.IsNotExist(err) {
		return instances, nil // Return empty map if directory doesn't exist
	}

	// Read directory entries
	entries, err := os.ReadDir(afo.instancesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read instances directory: %w", err)
	}

	// Process each .json file
	for _, entry := range entries {
		if entry.IsDir() || !entry.Type().IsRegular() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".json" || name == ".discovery.lock" {
			continue
		}

		// Extract instance ID from filename
		instanceID := name[:len(name)-5] // Remove .json extension

		// Read instance
		instance, err := afo.readInstanceLocked(instanceID)
		if err != nil {
			// Log error but continue processing other instances
			fmt.Printf("Warning: failed to read instance %s: %v\n", instanceID, err)
			continue
		}

		instances[instanceID] = instance
	}

	return instances, nil
}

// atomicWriteFile writes data to a file atomically using temp file + rename
func (afo *AtomicFileOperations) atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, DefaultDirMode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temp file in same directory (for atomic rename)
	tempFile, err := os.CreateTemp(dir, ".tmp-instance-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file on error
	defer func() {
		if tempFile != nil {
			tempFile.Close()
			os.Remove(tempPath)
		}
	}()

	// Write data
	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Sync to disk to ensure data is written
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Close before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	tempFile = nil // Prevent defer cleanup

	// Set permissions
	if err := os.Chmod(tempPath, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename (this is the atomic operation)
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// GetLockTimeout returns the current lock timeout
func (afo *AtomicFileOperations) GetLockTimeout() time.Duration {
	return afo.lockTimeout
}

// SetLockTimeout sets the lock timeout (useful for testing)
func (afo *AtomicFileOperations) SetLockTimeout(timeout time.Duration) {
	afo.lockTimeout = timeout
}

// IsInstanceFileCorrupted checks if an instance file is corrupted
func (afo *AtomicFileOperations) IsInstanceFileCorrupted(instanceID string) bool {
	_, err := afo.SafeReadInstance(instanceID)
	return err != nil
}