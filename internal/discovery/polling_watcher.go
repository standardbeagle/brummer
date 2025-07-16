package discovery

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PollingWatcher provides file watching through periodic polling
// This is more reliable than fsnotify for network filesystems
type PollingWatcher struct {
	mu         sync.RWMutex
	interval   time.Duration
	paths      map[string]bool
	fileStates map[string]fileState
	events     chan FileEvent
	errors     chan error
	stop       chan struct{}
	wg         sync.WaitGroup
}

type fileState struct {
	modTime time.Time
	size    int64
	hash    string
}

type FileEvent struct {
	Path string
	Op   FileOp
}

type FileOp int

const (
	Create FileOp = iota
	Write
	Remove
)

func (op FileOp) String() string {
	switch op {
	case Create:
		return "CREATE"
	case Write:
		return "WRITE"
	case Remove:
		return "REMOVE"
	default:
		return "UNKNOWN"
	}
}

// NewPollingWatcher creates a new polling-based file watcher
func NewPollingWatcher(interval time.Duration) *PollingWatcher {
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond // Minimum interval
	}

	return &PollingWatcher{
		interval:   interval,
		paths:      make(map[string]bool),
		fileStates: make(map[string]fileState),
		events:     make(chan FileEvent, 100),
		errors:     make(chan error, 10),
		stop:       make(chan struct{}),
	}
}

// Add adds a path to watch (can be file or directory)
func (pw *PollingWatcher) Add(path string) error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Verify path exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path: %w", err)
	}

	pw.paths[path] = info.IsDir()

	// If it's a directory, scan existing files
	if info.IsDir() {
		if err := pw.scanDirectoryLocked(path); err != nil {
			return fmt.Errorf("initial scan: %w", err)
		}
	} else {
		// It's a file, record its state
		state, err := pw.getFileState(path)
		if err != nil {
			return fmt.Errorf("get file state: %w", err)
		}
		pw.fileStates[path] = state
	}

	return nil
}

// Remove removes a path from watching
func (pw *PollingWatcher) Remove(path string) error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	delete(pw.paths, path)

	// Remove file states for this path
	if pw.paths[path] { // It was a directory
		prefix := path + string(filepath.Separator)
		for filePath := range pw.fileStates {
			if filepath.HasPrefix(filePath, prefix) || filePath == path {
				delete(pw.fileStates, filePath)
			}
		}
	} else {
		delete(pw.fileStates, path)
	}

	return nil
}

// Start begins watching
func (pw *PollingWatcher) Start() {
	pw.wg.Add(1)
	go pw.pollLoop()
}

// Stop stops the watcher and closes channels
func (pw *PollingWatcher) Stop() error {
	close(pw.stop)
	pw.wg.Wait()
	close(pw.events)
	close(pw.errors)
	return nil
}

// Events returns the events channel
func (pw *PollingWatcher) Events() <-chan FileEvent {
	return pw.events
}

// Errors returns the errors channel
func (pw *PollingWatcher) Errors() <-chan error {
	return pw.errors
}

func (pw *PollingWatcher) pollLoop() {
	defer pw.wg.Done()

	ticker := time.NewTicker(pw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-pw.stop:
			return
		case <-ticker.C:
			pw.poll()
		}
	}
}

func (pw *PollingWatcher) poll() {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Track seen files in this poll
	seenFiles := make(map[string]bool)

	// Check each watched path
	for path, isDir := range pw.paths {
		if isDir {
			// Scan directory
			entries, err := os.ReadDir(path)
			if err != nil {
				select {
				case pw.errors <- fmt.Errorf("read dir %s: %w", path, err):
				default:
				}
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue // Skip subdirectories
				}

				filePath := filepath.Join(path, entry.Name())
				seenFiles[filePath] = true

				pw.checkFile(filePath)
			}
		} else {
			// Check single file
			seenFiles[path] = true
			pw.checkFile(path)
		}
	}

	// Check for removed files
	for filePath := range pw.fileStates {
		if !seenFiles[filePath] {
			// File was removed
			delete(pw.fileStates, filePath)
			select {
			case pw.events <- FileEvent{Path: filePath, Op: Remove}:
			default:
			}
		}
	}
}

func (pw *PollingWatcher) checkFile(path string) {
	newState, err := pw.getFileState(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File was removed
			if _, exists := pw.fileStates[path]; exists {
				delete(pw.fileStates, path)
				select {
				case pw.events <- FileEvent{Path: path, Op: Remove}:
				default:
				}
			}
		}
		return
	}

	oldState, exists := pw.fileStates[path]
	if !exists {
		// New file
		pw.fileStates[path] = newState
		select {
		case pw.events <- FileEvent{Path: path, Op: Create}:
		default:
		}
		return
	}

	// Check for modifications
	if oldState.modTime != newState.modTime ||
		oldState.size != newState.size ||
		oldState.hash != newState.hash {
		pw.fileStates[path] = newState
		select {
		case pw.events <- FileEvent{Path: path, Op: Write}:
		default:
		}
	}
}

func (pw *PollingWatcher) getFileState(path string) (fileState, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileState{}, err
	}

	state := fileState{
		modTime: info.ModTime(),
		size:    info.Size(),
	}

	// For small files, compute hash for better change detection
	if info.Size() < 1024*1024 { // 1MB
		hash, err := pw.hashFile(path)
		if err == nil {
			state.hash = hash
		}
	}

	return state, nil
}

func (pw *PollingWatcher) hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (pw *PollingWatcher) scanDirectoryLocked(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		state, err := pw.getFileState(filePath)
		if err != nil {
			continue
		}
		pw.fileStates[filePath] = state
	}

	return nil
}
