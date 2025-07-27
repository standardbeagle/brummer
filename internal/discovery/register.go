package discovery

import (
	"os"
	"path/filepath"
)

// RegisterInstance registers a brummer instance atomically.
// The instance data is validated before writing.
func RegisterInstance(instancesDir string, instance *Instance) error {
	afo := NewAtomicFileOperations(instancesDir)
	return afo.SafeRegisterInstance(instance)
}

// UnregisterInstance removes an instance registration
func UnregisterInstance(instancesDir string, instanceID string) error {
	afo := NewAtomicFileOperations(instancesDir)
	return afo.SafeUnregisterInstance(instanceID)
}

// UpdateInstancePing updates the last ping time for an instance
func UpdateInstancePing(instancesDir string, instanceID string) error {
	afo := NewAtomicFileOperations(instancesDir)
	return afo.SafeUpdateInstancePing(instanceID)
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
