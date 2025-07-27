// Package discovery provides file-based instance discovery for brummer.
// It watches a directory for JSON files containing instance metadata and
// maintains an in-memory registry of running instances.
//
// The discovery system is thread-safe and supports concurrent access.
// Callbacks are notified of changes but must not call back into the
// discovery system to avoid deadlocks.
package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	// DefaultFileMode is the default permission for instance files (owner read/write only)
	DefaultFileMode = 0600
	// DefaultDirMode is the default permission for the instances directory
	DefaultDirMode = 0700
	// StaleInstanceTimeout is the duration after which an instance is considered stale
	StaleInstanceTimeout = 5 * time.Minute
)

// Instance represents a running brummer instance.
// All fields are required unless otherwise noted.
type Instance struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Directory   string    `json:"directory"`
	Port        int       `json:"port"`
	StartedAt   time.Time `json:"started_at"`
	LastPing    time.Time `json:"last_ping"`
	ProcessInfo struct {
		PID        int    `json:"pid"`
		Executable string `json:"executable"`
	} `json:"process_info"`
}

// Discovery manages instance discovery via file watching.
// It is thread-safe and can be safely accessed concurrently.
type Discovery struct {
	mu              sync.RWMutex
	instances       map[string]*Instance
	instancesDir    string
	watcher         *fsnotify.Watcher
	updateCallbacks []func(instances map[string]*Instance)
	stopCh          chan struct{}
	stoppedCh       chan struct{}
	atomicOps       *AtomicFileOperations
}

// New creates a new instance discovery system.
// The instancesDir will be created if it doesn't exist.
func New(instancesDir string) (*Discovery, error) {
	// Create instances directory if it doesn't exist
	if err := os.MkdirAll(instancesDir, DefaultDirMode); err != nil {
		return nil, fmt.Errorf("failed to create instances directory: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Ensure watcher is cleaned up on any error
	var watcherClosed bool
	defer func() {
		if !watcherClosed && err != nil {
			watcher.Close()
		}
	}()

	d := &Discovery{
		instances:    make(map[string]*Instance),
		instancesDir: instancesDir,
		watcher:      watcher,
		stopCh:       make(chan struct{}),
		stoppedCh:    make(chan struct{}),
		atomicOps:    NewAtomicFileOperations(instancesDir),
	}

	// Add the instances directory to the watcher
	if err = watcher.Add(instancesDir); err != nil {
		return nil, fmt.Errorf("failed to watch instances directory: %w", err)
	}

	// Initial scan of existing files
	if err = d.scanDirectory(); err != nil {
		return nil, fmt.Errorf("initial directory scan failed: %w", err)
	}

	watcherClosed = true // Prevent deferred cleanup
	return d, nil
}

// Start begins watching for instance changes.
// This method returns immediately and watches in the background.
func (d *Discovery) Start() {
	go d.watch()
}

// Stop stops the discovery system and waits for the watch goroutine to exit.
func (d *Discovery) Stop() error {
	// Signal stop
	close(d.stopCh)

	// Close watcher to trigger event loop exit
	err := d.watcher.Close()

	// Wait for watch goroutine to complete
	<-d.stoppedCh

	return err
}

// GetInstances returns a copy of the current instances
func (d *Discovery) GetInstances() map[string]*Instance {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	result := make(map[string]*Instance)
	for k, v := range d.instances {
		result[k] = v
	}
	return result
}

// OnUpdate registers a callback for instance updates
func (d *Discovery) OnUpdate(callback func(instances map[string]*Instance)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.updateCallbacks = append(d.updateCallbacks, callback)
}

// scanDirectory scans the instances directory for existing instance files
func (d *Discovery) scanDirectory() error {
	// Use atomic operations to safely list all instances
	instances, err := d.atomicOps.SafeListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Update in-memory cache with discovered instances
	d.mu.Lock()
	for id, instance := range instances {
		d.instances[id] = instance
	}
	d.mu.Unlock()

	return nil
}

// watch monitors the instances directory for changes
func (d *Discovery) watch() {
	defer close(d.stoppedCh)

	for {
		select {
		case <-d.stopCh:
			return

		case event, ok := <-d.watcher.Events:
			if !ok {
				return
			}

			if !isInstanceFile(filepath.Base(event.Name)) {
				continue
			}

			switch {
			case event.Has(fsnotify.Write) || event.Has(fsnotify.Create):
				if err := d.loadInstance(event.Name); err != nil {
					// TODO: Add proper logging
					fmt.Fprintf(os.Stderr, "Failed to load instance %s: %v\n", event.Name, err)
				}
			case event.Has(fsnotify.Remove):
				d.removeInstance(event.Name)
			}

		case err, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			// TODO: Add proper logging
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

// loadInstance loads an instance from a file
func (d *Discovery) loadInstance(path string) error {
	// Extract instance ID from the filename
	filename := filepath.Base(path)
	instanceID := extractInstanceID(filename)
	if instanceID == "" {
		return fmt.Errorf("invalid instance filename: %s", filename)
	}

	// Use atomic operations to safely read the instance
	instance, err := d.atomicOps.SafeReadInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to read instance %s: %w", instanceID, err)
	}

	// Get callbacks while holding lock to ensure consistency
	d.mu.Lock()
	d.instances[instance.ID] = instance
	// Create a copy of callbacks to avoid holding lock during callback execution
	callbacks := make([]func(map[string]*Instance), len(d.updateCallbacks))
	copy(callbacks, d.updateCallbacks)
	// Get instances snapshot while still holding the lock
	instancesCopy := d.getInstancesLocked()
	d.mu.Unlock()

	// Notify callbacks without holding lock
	for _, callback := range callbacks {
		callback(instancesCopy)
	}

	return nil
}

// validateInstance validates that an instance has all required fields
func validateInstance(inst *Instance) error {
	if inst.ID == "" {
		return fmt.Errorf("missing ID")
	}
	if inst.Name == "" {
		return fmt.Errorf("missing Name")
	}
	if inst.Directory == "" {
		return fmt.Errorf("missing Directory")
	}
	if inst.Port <= 0 || inst.Port > 65535 {
		return fmt.Errorf("invalid Port: %d", inst.Port)
	}
	if inst.StartedAt.IsZero() {
		return fmt.Errorf("missing StartedAt")
	}
	if inst.LastPing.IsZero() {
		return fmt.Errorf("missing LastPing")
	}
	// Check if timestamps are reasonable (not in the future)
	now := time.Now()
	if inst.StartedAt.After(now.Add(time.Minute)) {
		return fmt.Errorf("StartedAt is in the future")
	}
	if inst.LastPing.After(now.Add(time.Minute)) {
		return fmt.Errorf("LastPing is in the future")
	}
	return nil
}

// getInstancesLocked returns a copy of instances while holding the read lock
// Caller must hold at least a read lock
func (d *Discovery) getInstancesLocked() map[string]*Instance {
	result := make(map[string]*Instance, len(d.instances))
	for k, v := range d.instances {
		// Create a copy of the instance to prevent external modification
		instCopy := *v
		result[k] = &instCopy
	}
	return result
}

// removeInstance removes an instance when its file is deleted
func (d *Discovery) removeInstance(path string) {
	// Extract instance ID from filename
	filename := filepath.Base(path)
	instanceID := extractInstanceID(filename)
	if instanceID == "" {
		return
	}

	d.mu.Lock()
	// Check if instance exists
	if _, exists := d.instances[instanceID]; !exists {
		d.mu.Unlock()
		return
	}

	delete(d.instances, instanceID)
	// Create a copy of callbacks to avoid holding lock during callback execution
	callbacks := make([]func(map[string]*Instance), len(d.updateCallbacks))
	copy(callbacks, d.updateCallbacks)
	// Get instances snapshot while still holding the lock
	instancesCopy := d.getInstancesLocked()
	d.mu.Unlock()

	// Notify callbacks without holding lock
	for _, callback := range callbacks {
		callback(instancesCopy)
	}
}

// isInstanceFile checks if a filename is an instance file
func isInstanceFile(name string) bool {
	return filepath.Ext(name) == ".json" && len(name) > 5
}

// extractInstanceID extracts the instance ID from a filename
func extractInstanceID(filename string) string {
	if !isInstanceFile(filename) {
		return ""
	}
	return filename[:len(filename)-5] // Remove .json extension
}

// CleanupStaleInstances removes instances that haven't been updated recently.
// This should be called periodically to clean up after crashed instances.
func (d *Discovery) CleanupStaleInstances() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	staleIDs := []string{}

	// Find stale instances
	for id, inst := range d.instances {
		isStale := now.Sub(inst.LastPing) > StaleInstanceTimeout

		// Also check if the process is actually running
		if isStale || !d.isProcessRunning(inst.ProcessInfo.PID) {
			staleIDs = append(staleIDs, id)
		}
	}

	// Remove stale instances using atomic operations
	for _, id := range staleIDs {
		// Remove from memory
		delete(d.instances, id)

		// Remove file atomically
		if err := d.atomicOps.SafeUnregisterInstance(id); err != nil {
			// Log error but continue cleanup
			fmt.Fprintf(os.Stderr, "Failed to remove stale instance file %s: %v\n", id, err)
		}
	}

	// Notify callbacks if any instances were removed
	if len(staleIDs) > 0 {
		callbacks := make([]func(map[string]*Instance), len(d.updateCallbacks))
		copy(callbacks, d.updateCallbacks)
		instancesCopy := d.getInstancesLocked()

		// Release lock before callbacks
		d.mu.Unlock()
		for _, callback := range callbacks {
			callback(instancesCopy)
		}
		d.mu.Lock() // Re-acquire for defer
	}

	return nil
}

// isProcessRunning checks if a process with the given PID is currently running
func (d *Discovery) isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Use Linux /proc filesystem approach which is more reliable
	procPath := fmt.Sprintf("/proc/%d", pid)
	if _, err := os.Stat(procPath); os.IsNotExist(err) {
		return false
	}

	// Check if the process is actually running by reading its status
	statusPath := filepath.Join(procPath, "stat")
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		return false
	}

	// If /proc/PID/stat exists, the process is running
	return true
}
