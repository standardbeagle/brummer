package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Instance represents a running brummer instance
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

// Discovery manages instance discovery via file watching
type Discovery struct {
	mu              sync.RWMutex
	instances       map[string]*Instance
	instancesDir    string
	watcher         *fsnotify.Watcher
	updateCallbacks []func(instances map[string]*Instance)
}

// New creates a new instance discovery system
func New(instancesDir string) (*Discovery, error) {
	// Create instances directory if it doesn't exist
	if err := os.MkdirAll(instancesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create instances directory: %w", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	d := &Discovery{
		instances:    make(map[string]*Instance),
		instancesDir: instancesDir,
		watcher:      watcher,
	}

	// Add the instances directory to the watcher
	if err := watcher.Add(instancesDir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch instances directory: %w", err)
	}

	// Initial scan of existing files
	if err := d.scanDirectory(); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("initial directory scan failed: %w", err)
	}

	return d, nil
}

// Start begins watching for instance changes
func (d *Discovery) Start() {
	go d.watch()
}

// Stop stops the discovery system
func (d *Discovery) Stop() error {
	return d.watcher.Close()
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
	entries, err := os.ReadDir(d.instancesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !isInstanceFile(entry.Name()) {
			continue
		}

		instancePath := filepath.Join(d.instancesDir, entry.Name())
		if err := d.loadInstance(instancePath); err != nil {
			// Log error but continue scanning
			fmt.Fprintf(os.Stderr, "Failed to load instance %s: %v\n", entry.Name(), err)
		}
	}

	return nil
}

// watch monitors the instances directory for changes
func (d *Discovery) watch() {
	for {
		select {
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
					fmt.Fprintf(os.Stderr, "Failed to load instance %s: %v\n", event.Name, err)
				}
			case event.Has(fsnotify.Remove):
				d.removeInstance(event.Name)
			}

		case err, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

// loadInstance loads an instance from a file
func (d *Discovery) loadInstance(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var instance Instance
	if err := json.Unmarshal(data, &instance); err != nil {
		return err
	}

	// Validate instance data
	if instance.ID == "" {
		return fmt.Errorf("instance missing ID")
	}

	d.mu.Lock()
	d.instances[instance.ID] = &instance
	callbacks := d.updateCallbacks
	d.mu.Unlock()

	// Notify callbacks
	instances := d.GetInstances()
	for _, callback := range callbacks {
		callback(instances)
	}

	return nil
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
	delete(d.instances, instanceID)
	callbacks := d.updateCallbacks
	d.mu.Unlock()

	// Notify callbacks
	instances := d.GetInstances()
	for _, callback := range callbacks {
		callback(instances)
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