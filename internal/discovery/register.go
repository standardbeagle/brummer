package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// RegisterInstance registers a brummer instance atomically
func RegisterInstance(instancesDir string, instance *Instance) error {
	// Ensure instances directory exists
	if err := os.MkdirAll(instancesDir, 0755); err != nil {
		return fmt.Errorf("failed to create instances directory: %w", err)
	}

	// Marshal instance data
	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instance data: %w", err)
	}

	// Write atomically using temp file + rename
	filename := fmt.Sprintf("%s.json", instance.ID)
	finalPath := filepath.Join(instancesDir, filename)
	tempPath := finalPath + ".tmp"

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, finalPath); err != nil {
		os.Remove(tempPath) // Clean up on error
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// UnregisterInstance removes an instance registration
func UnregisterInstance(instancesDir string, instanceID string) error {
	filename := fmt.Sprintf("%s.json", instanceID)
	path := filepath.Join(instancesDir, filename)
	
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove instance file: %w", err)
	}
	
	return nil
}

// UpdateInstancePing updates the last ping time for an instance
func UpdateInstancePing(instancesDir string, instanceID string) error {
	filename := fmt.Sprintf("%s.json", instanceID)
	path := filepath.Join(instancesDir, filename)
	
	// Read existing instance data
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read instance file: %w", err)
	}
	
	var instance Instance
	if err := json.Unmarshal(data, &instance); err != nil {
		return fmt.Errorf("failed to unmarshal instance data: %w", err)
	}
	
	// Update last ping time
	instance.LastPing = time.Now()
	
	// Write back atomically
	return RegisterInstance(instancesDir, &instance)
}

// GetDefaultInstancesDir returns the default instances directory
func GetDefaultInstancesDir() string {
	// Try to use XDG_RUNTIME_DIR first (Linux standard)
	if runtime := os.Getenv("XDG_RUNTIME_DIR"); runtime != "" {
		return filepath.Join(runtime, "brummer", "instances")
	}
	
	// Fall back to temp directory
	return filepath.Join(os.TempDir(), "brummer", "instances")
}

// AtomicWriteFile writes data to a file atomically
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temp file in same directory (for atomic rename)
	tempFile, err := os.CreateTemp(dir, ".tmp-")
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

	// Sync to disk
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

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// AtomicCopyFile copies a file atomically
func AtomicCopyFile(src, dst string, perm os.FileMode) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Read all data
	data, err := io.ReadAll(srcFile)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write atomically
	return AtomicWriteFile(dst, data, perm)
}