package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/parser"
	"github.com/standardbeagle/brummer/pkg/events"
)

type Process struct {
	ID        string
	Name      string
	Script    string
	Cmd       *exec.Cmd
	Status    ProcessStatus
	StartTime time.Time
	EndTime   *time.Time
	ExitCode  *int
	cancel    context.CancelFunc
	mu        sync.RWMutex

	// Atomic state for lock-free reads (30-300x faster than mutex)
	atomicState unsafe.Pointer // *ProcessState
}

// Thread-safe getters for Process fields
func (p *Process) GetStatus() ProcessStatus {
	// Fast path: try atomic first
	if statePtr := (*ProcessState)(atomic.LoadPointer(&p.atomicState)); statePtr != nil {
		return statePtr.Status
	}
	// Fallback: mutex path
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Status
}

func (p *Process) GetStartTime() time.Time {
	// Fast path: try atomic first
	if statePtr := (*ProcessState)(atomic.LoadPointer(&p.atomicState)); statePtr != nil {
		return statePtr.StartTime
	}
	// Fallback: mutex path
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.StartTime
}

func (p *Process) GetEndTime() *time.Time {
	// Fast path: try atomic first
	if statePtr := (*ProcessState)(atomic.LoadPointer(&p.atomicState)); statePtr != nil {
		return statePtr.EndTime
	}
	// Fallback: mutex path
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.EndTime
}

func (p *Process) GetExitCode() *int {
	// Fast path: try atomic first
	if statePtr := (*ProcessState)(atomic.LoadPointer(&p.atomicState)); statePtr != nil {
		return statePtr.ExitCode
	}
	// Fallback: mutex path
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ExitCode
}

// GetSnapshot returns an atomic snapshot of all Process fields
// This is more efficient than multiple individual getter calls when you need multiple fields
func (p *Process) GetSnapshot() ProcessSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return ProcessSnapshot{
		ID:        p.ID,
		Name:      p.Name,
		Script:    p.Script,
		Status:    p.Status,
		StartTime: p.StartTime,
		EndTime:   p.EndTime,
		ExitCode:  p.ExitCode,
	}
}

// Thread-safe setters for Process fields
func (p *Process) SetStatus(status ProcessStatus) {
	// Use atomic update for consistency
	p.UpdateStateAtomic(func(state ProcessState) ProcessState {
		return state.CopyWithStatus(status)
	})
}

// GetStateAtomic returns the current process state atomically
// This is the PRIMARY method for lock-free state access (30-300x faster than mutex)
func (p *Process) GetStateAtomic() ProcessState {
	statePtr := (*ProcessState)(atomic.LoadPointer(&p.atomicState))
	if statePtr == nil {
		// Fallback: build from mutex-protected fields
		p.mu.RLock()
		defer p.mu.RUnlock()
		return ProcessState{
			ID:        p.ID,
			Name:      p.Name,
			Script:    p.Script,
			Status:    p.Status,
			StartTime: p.StartTime,
			EndTime:   p.EndTime,
			ExitCode:  p.ExitCode,
		}
	}
	return *statePtr
}

// UpdateStateAtomic performs atomic state update using CAS
func (p *Process) UpdateStateAtomic(updater func(ProcessState) ProcessState) {
	for {
		currentPtr := (*ProcessState)(atomic.LoadPointer(&p.atomicState))
		var current ProcessState
		if currentPtr == nil {
			// Initialize from mutex fields
			p.mu.RLock()
			current = ProcessState{
				ID:        p.ID,
				Name:      p.Name,
				Script:    p.Script,
				Status:    p.Status,
				StartTime: p.StartTime,
				EndTime:   p.EndTime,
				ExitCode:  p.ExitCode,
			}
			p.mu.RUnlock()
		} else {
			current = *currentPtr
		}

		newState := updater(current)
		newStatePtr := &newState

		// Try to swap the pointer atomically
		if atomic.CompareAndSwapPointer(
			&p.atomicState,
			unsafe.Pointer(currentPtr),
			unsafe.Pointer(newStatePtr),
		) {
			// Also update mutex-protected fields for compatibility
			p.updateMutexFields(newState)
			break
		}
		// If CAS failed, another update happened - retry
	}
}

// Helper to keep mutex fields in sync
func (p *Process) updateMutexFields(state ProcessState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Status = state.Status
	p.EndTime = state.EndTime
	p.ExitCode = state.ExitCode
}

type ProcessStatus string

const (
	StatusPending ProcessStatus = "pending"
	StatusRunning ProcessStatus = "running"
	StatusStopped ProcessStatus = "stopped"
	StatusFailed  ProcessStatus = "failed"
	StatusSuccess ProcessStatus = "success"
)

// ProcessSnapshot provides atomic access to multiple Process fields
// This reduces lock contention by capturing all frequently-accessed fields in a single operation
type ProcessSnapshot struct {
	ID        string
	Name      string
	Script    string
	Status    ProcessStatus
	StartTime time.Time
	EndTime   *time.Time
	ExitCode  *int
}

// String implements fmt.Stringer for ProcessSnapshot
func (ps ProcessSnapshot) String() string {
	return fmt.Sprintf("Process{ID: %s, Name: %s, Status: %s}", ps.ID, ps.Name, ps.Status)
}

// IsRunning returns true if the process is currently running
func (ps ProcessSnapshot) IsRunning() bool {
	return ps.Status == StatusRunning
}

// IsFinished returns true if the process has completed (success, failed, or stopped)
func (ps ProcessSnapshot) IsFinished() bool {
	return ps.Status == StatusSuccess || ps.Status == StatusFailed || ps.Status == StatusStopped
}

// Duration returns how long the process has been running or ran for
func (ps ProcessSnapshot) Duration() time.Duration {
	if ps.EndTime != nil {
		return ps.EndTime.Sub(ps.StartTime)
	}
	return time.Since(ps.StartTime)
}

type Manager struct {
	processes      sync.Map // map[string]*Process - now lock-free for concurrent access
	packageJSON    *parser.PackageJSON
	packageMgr     parser.PackageManager
	userPackageMgr *parser.PackageManager
	workDir        string
	eventBus       *events.EventBus
	logCallbacks   []LogCallback
	installedMgrs  []parser.InstalledPackageManager
	mu             sync.RWMutex // Still needed for logCallbacks and other fields

	// AI Coder integration
	aiCoderMgr         *aicoder.AICoderManager
	aiCoderIntegration *AICoderIntegration
}

type LogCallback func(processID string, line string, isError bool)

func NewManager(workDir string, eventBus *events.EventBus, hasPackageJSON bool) (*Manager, error) {
	var pkgJSON *parser.PackageJSON
	var err error

	if hasPackageJSON {
		pkgJSON, err = parser.ParsePackageJSON(workDir + "/package.json")
		if err != nil {
			return nil, err
		}
	} else {
		// Create empty package.json structure for fallback mode
		pkgJSON = &parser.PackageJSON{
			Scripts: make(map[string]string),
		}
	}

	// Load config
	cfg, _ := config.Load()

	// Detect installed package managers
	installedMgrs := parser.DetectInstalledPackageManagers()

	m := &Manager{
		// processes is sync.Map - zero value is ready to use
		packageJSON:    pkgJSON,
		workDir:        workDir,
		eventBus:       eventBus,
		installedMgrs:  installedMgrs,
		userPackageMgr: cfg.PreferredPackageManager,
	}

	// Set initial package manager based on detection
	m.updatePackageManager()

	return m, nil
}

func (m *Manager) GetScripts() map[string]string {
	return m.packageJSON.Scripts
}

// GetDetectedCommands returns all detected executable commands
func (m *Manager) GetDetectedCommands() []parser.ExecutableCommand {
	return parser.DetectProjectCommands(m.workDir)
}

// GetMonorepoInfo returns monorepo information if detected
func (m *Manager) GetMonorepoInfo() (*parser.MonorepoInfo, error) {
	return parser.DetectMonorepo(m.workDir)
}

func (m *Manager) StartScript(scriptName string) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	script, exists := m.packageJSON.Scripts[scriptName]
	if !exists {
		return nil, fmt.Errorf("script '%s' not found in package.json", scriptName)
	}

	processID := fmt.Sprintf("%s-%d", scriptName, time.Now().Unix())

	ctx, cancel := context.WithCancel(context.Background())
	cmdArgs := m.packageMgr.RunScriptCommand(scriptName)
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = m.workDir

	// Set up environment
	cmd.Env = os.Environ()
	// Force color output for common tools
	cmd.Env = append(cmd.Env, "FORCE_COLOR=1")
	cmd.Env = append(cmd.Env, "COLORTERM=truecolor")
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

	// Set process group for easier cleanup (platform-specific)
	setupProcessGroup(cmd)

	process := &Process{
		ID:        processID,
		Name:      scriptName,
		Script:    script,
		Cmd:       cmd,
		Status:    StatusPending,
		StartTime: time.Now(),
		cancel:    cancel,
	}

	m.processes.Store(processID, process)

	if err := m.runProcess(process); err != nil {
		process.SetStatus(StatusFailed)
		return nil, err
	}

	return process, nil
}

// StartCommand starts a custom command (not from package.json)
func (m *Manager) StartCommand(name string, command string, args []string) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	processID := fmt.Sprintf("%s-%d", name, time.Now().Unix())

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = m.workDir

	// Set up environment
	cmd.Env = os.Environ()
	// Force color output for common tools
	cmd.Env = append(cmd.Env, "FORCE_COLOR=1")
	cmd.Env = append(cmd.Env, "COLORTERM=truecolor")
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

	// Set process group for easier cleanup (platform-specific)
	setupProcessGroup(cmd)

	process := &Process{
		ID:        processID,
		Name:      name,
		Script:    fmt.Sprintf("%s %s", command, strings.Join(args, " ")),
		Cmd:       cmd,
		Status:    StatusPending,
		StartTime: time.Now(),
		cancel:    cancel,
	}

	m.processes.Store(processID, process)

	if err := m.runProcess(process); err != nil {
		process.SetStatus(StatusFailed)
		return nil, err
	}

	return process, nil
}

func (m *Manager) runProcess(p *Process) error {
	stdout, err := p.Cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := p.Cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := p.Cmd.Start(); err != nil {
		// Add more context to the error
		return fmt.Errorf("failed to start command %v: %w", p.Cmd.Args, err)
	}

	p.mu.Lock()
	p.Status = StatusRunning
	p.mu.Unlock()

	m.eventBus.Publish(events.Event{
		Type:      events.ProcessStarted,
		ProcessID: p.ID,
		Data: map[string]interface{}{
			"name":   p.Name,
			"script": p.Script,
			"cmd":    p.Cmd.Args,
		},
	})

	go m.streamLogs(p.ID, stdout, false)
	go m.streamLogs(p.ID, stderr, true)

	go func() {
		err := p.Cmd.Wait()

		// Ensure clean log separation when process exits
		// This adds a newline to ensure the next process starts on a new line
		m.mu.RLock()
		callbacks := m.logCallbacks
		m.mu.RUnlock()

		for _, cb := range callbacks {
			cb(p.ID, "", false) // Empty line to ensure separation
		}

		p.mu.Lock()
		now := time.Now()
		p.EndTime = &now

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				code := exitErr.ExitCode()
				p.ExitCode = &code
				p.Status = StatusFailed
			} else {
				p.Status = StatusFailed
			}
		} else {
			code := 0
			p.ExitCode = &code
			p.Status = StatusSuccess
		}
		p.mu.Unlock()

		// Add a failure summary log for failed processes
		if p.Status == StatusFailed {
			m.mu.RLock()
			callbacks := m.logCallbacks
			m.mu.RUnlock()

			var exitCodeStr string
			if p.ExitCode != nil {
				exitCodeStr = fmt.Sprintf(" (exit code: %d)", *p.ExitCode)
			}

			failureMsg := fmt.Sprintf("❌ Process '%s' failed%s", p.Name, exitCodeStr)
			for _, cb := range callbacks {
				cb(p.ID, failureMsg, true)
			}
		}

		m.eventBus.Publish(events.Event{
			Type:      events.ProcessExited,
			ProcessID: p.ID,
			Data: map[string]interface{}{
				"name":     p.Name,
				"status":   p.Status,
				"exitCode": p.ExitCode,
			},
		})
	}()

	return nil
}

func (m *Manager) streamLogs(processID string, reader io.Reader, isError bool) {
	// Use a buffered reader to handle partial lines
	bufReader := bufio.NewReader(reader)

	for {
		line, err := bufReader.ReadString('\n')
		if err != nil {
			// If we have partial data when EOF is reached, still process it
			if err == io.EOF && len(line) > 0 {
				// Remove any trailing newline if present
				line = strings.TrimSuffix(line, "\n")

				m.mu.RLock()
				callbacks := m.logCallbacks
				m.mu.RUnlock()

				for _, cb := range callbacks {
					cb(processID, line, isError)
				}

				m.eventBus.Publish(events.Event{
					Type:      events.LogLine,
					ProcessID: processID,
					Data: map[string]interface{}{
						"line":    line,
						"isError": isError,
					},
				})
			}
			break
		}

		// Remove the newline character
		line = strings.TrimSuffix(line, "\n")

		m.mu.RLock()
		callbacks := m.logCallbacks
		m.mu.RUnlock()

		for _, cb := range callbacks {
			cb(processID, line, isError)
		}

		m.eventBus.Publish(events.Event{
			Type:      events.LogLine,
			ProcessID: processID,
			Data: map[string]interface{}{
				"line":    line,
				"isError": isError,
			},
		})
	}
}

func (m *Manager) StopProcess(processID string) error {
	// Use sync.Map for lock-free process lookup
	value, exists := m.processes.Load(processID)
	if !exists {
		return fmt.Errorf("process %s not found", processID)
	}

	process, ok := value.(*Process)
	if !ok {
		return fmt.Errorf("invalid process type for %s", processID)
	}

	process.mu.Lock()
	if process.Status != StatusRunning {
		process.mu.Unlock()
		return fmt.Errorf("process %s is not running", processID)
	}

	// Get the PID before we start killing
	var mainPID int
	if process.Cmd != nil && process.Cmd.Process != nil {
		mainPID = process.Cmd.Process.Pid
	}

	// First try graceful shutdown
	if process.cancel != nil {
		process.cancel()
	}

	// Kill the process tree aggressively
	if process.Cmd != nil && process.Cmd.Process != nil {
		m.killProcessTree(process.Cmd.Process.Pid)
	}

	// Also kill any processes that might be using development ports
	// Do this asynchronously but with immediate action for dev processes
	if process.Name == "dev" || process.Name == "start" ||
		strings.Contains(process.Script, "npm") ||
		strings.Contains(process.Script, "pnpm") ||
		strings.Contains(process.Script, "yarn") {
		// Don't wait - kill immediately in background
		go m.KillProcessesByPort()
	}

	process.Status = StatusStopped
	now := time.Now()
	process.EndTime = &now
	exitCode := -1
	process.ExitCode = &exitCode
	process.mu.Unlock()

	// Asynchronously verify process termination
	go func() {
		// Small delay to allow processes to terminate gracefully
		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Double-check that the main process is dead
			if mainPID > 0 {
				m.ensureProcessDead(mainPID)
			}

			// For package manager processes, be extra thorough
			if process.Name == "dev" || process.Name == "start" ||
				strings.Contains(process.Script, "npm") ||
				strings.Contains(process.Script, "pnpm") ||
				strings.Contains(process.Script, "yarn") {
				// Give package managers extra time then do additional cleanup
				time.Sleep(200 * time.Millisecond)
				m.KillProcessesByPort()
			}
		}
	}()

	// Publish stop event immediately
	m.eventBus.Publish(events.Event{
		Type:      events.ProcessExited,
		ProcessID: processID,
		Data: map[string]interface{}{
			"exitCode": exitCode,
			"forced":   true,
		},
	})

	return nil
}

// StopProcessAndWait stops a process and waits for it to terminate completely
func (m *Manager) StopProcessAndWait(processID string, timeout time.Duration) error {
	// First stop the process normally
	if err := m.StopProcess(processID); err != nil {
		return err
	}

	// Get the process to check its PID
	value, exists := m.processes.Load(processID)
	if !exists {
		return nil // Process already cleaned up
	}

	process, ok := value.(*Process)
	if !ok {
		return nil // Invalid process type
	}

	// Get the PID to monitor
	var mainPID int
	process.mu.RLock()
	if process.Cmd != nil && process.Cmd.Process != nil {
		mainPID = process.Cmd.Process.Pid
	}
	process.mu.RUnlock()

	if mainPID <= 0 {
		return nil // No PID to monitor
	}

	// Wait for the process to actually terminate
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Timeout reached - force one more cleanup and return
			m.ensureProcessDead(mainPID)
			return fmt.Errorf("timeout waiting for process %s (PID %d) to terminate", processID, mainPID)
		case <-ticker.C:
			// Check if process still exists
			if proc, err := os.FindProcess(mainPID); err == nil {
				// Send signal 0 to check if process exists
				if err := proc.Signal(syscall.Signal(0)); err != nil {
					// Process is dead
					return nil
				}
			} else {
				// Process is dead
				return nil
			}
		}
	}
}

func (m *Manager) GetProcess(processID string) (*Process, bool) {
	// Use sync.Map for lock-free process lookup
	value, exists := m.processes.Load(processID)
	if !exists {
		return nil, false
	}
	process, ok := value.(*Process)
	if !ok {
		return nil, false
	}
	return process, true
}

func (m *Manager) GetAllProcesses() []*Process {
	// Use sync.Map.Range for lock-free iteration
	var processes []*Process
	m.processes.Range(func(key, value interface{}) bool {
		if process, ok := value.(*Process); ok {
			processes = append(processes, process)
		}
		return true // continue iteration
	})
	return processes
}

// CleanupFinishedProcesses removes terminated processes from the map
// This prevents accumulation of stopped/failed processes
func (m *Manager) CleanupFinishedProcesses() {
	// Use sync.Map.Range for lock-free iteration and deletion
	m.processes.Range(func(key, value interface{}) bool {
		if process, ok := value.(*Process); ok {
			// Use atomic state access for consistent read
			state := process.GetStateAtomic()

			// Remove processes that have finished (failed, stopped, or succeeded)
			if state.Status == StatusFailed || state.Status == StatusStopped || state.Status == StatusSuccess {
				m.processes.Delete(key)
			}
		}
		return true // continue iteration
	})
}

func (m *Manager) RegisterLogCallback(cb LogCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCallbacks = append(m.logCallbacks, cb)
}

func (m *Manager) GetInstalledPackageManagers() []parser.InstalledPackageManager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.installedMgrs
}

func (m *Manager) GetCurrentPackageManager() parser.PackageManager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.packageMgr
}

// IsPackageManagerFromJSON checks if the given package manager was specified in package.json
func (m *Manager) IsPackageManagerFromJSON(pm parser.PackageManager) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.packageJSON == nil {
		return false
	}

	// Check packageManager field
	if m.packageJSON.PackageManager != "" {
		parts := strings.Split(m.packageJSON.PackageManager, "@")
		if len(parts) > 0 && strings.EqualFold(parts[0], string(pm)) {
			return true
		}
	}

	// Check engines field
	if m.packageJSON.Engines != nil {
		_, hasEngine := m.packageJSON.Engines[string(pm)]
		return hasEngine
	}

	return false
}

func (m *Manager) SetUserPackageManager(pm parser.PackageManager) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userPackageMgr = &pm
	m.updatePackageManager()

	// Save to config
	cfg := &config.Config{
		PreferredPackageManager: &pm,
	}
	return cfg.Save()
}

func (m *Manager) updatePackageManager() {
	m.packageMgr = parser.GetPreferredPackageManager(m.packageJSON, m.workDir, m.userPackageMgr)
}

// AddLogCallback adds a callback function to be called when log lines are received
func (m *Manager) AddLogCallback(cb LogCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logCallbacks = append(m.logCallbacks, cb)
}

// StopAllProcesses stops all running processes
func (m *Manager) StopAllProcesses() error {
	var processIDs []string

	// Use sync.Map.Range for lock-free iteration
	m.processes.Range(func(key, value interface{}) bool {
		if id, ok := key.(string); ok {
			if proc, ok := value.(*Process); ok && proc.GetStatus() == StatusRunning {
				processIDs = append(processIDs, id)
			}
		}
		return true // continue iteration
	})

	var lastError error
	for _, id := range processIDs {
		if err := m.StopProcess(id); err != nil {
			lastError = err
		}
	}

	return lastError
}

// Cleanup stops all processes and cleans up resources
func (m *Manager) Cleanup() error {
	err := m.StopAllProcesses()

	// Kill any remaining development processes with minimal blocking
	done := make(chan bool, 1)
	go func() {
		m.KillProcessesByPort()
		done <- true
	}()

	// Wait for cleanup but don't block forever
	select {
	case <-done:
		// Cleanup completed
	case <-time.After(1 * time.Second):
		// Timeout - continue shutdown
	}

	return err
}

// killProcessTree kills a process and all its children
func (m *Manager) killProcessTree(pid int) {
	killProcessByPID(pid)

	// Also try to find and kill child processes on Unix
	if runtime.GOOS != "windows" {
		m.killChildProcesses(pid)
	}
}

// killChildProcesses finds and kills child processes
func (m *Manager) killChildProcesses(parentPID int) {
	if runtime.GOOS == "windows" {
		return // Skip for Windows
	}

	// Use ps to find child processes with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pgrep", "-P", strconv.Itoa(parentPID))
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.TrimSpace(string(output))
	if lines == "" {
		return
	}

	for _, line := range strings.Split(lines, "\n") {
		if childPID, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			// Recursively kill children
			m.killChildProcesses(childPID)
			// Then kill this child
			killProcessByPID(childPID)
		}
	}
}

// ensureProcessDead makes sure a process is really dead
func (m *Manager) ensureProcessDead(pid int) {
	ensureProcessDead(pid)
}

// KillProcessesByPort kills processes using development ports
func (m *Manager) KillProcessesByPort() {
	// Find processes using development ports (3000-3009)
	for port := 3000; port <= 3009; port++ {
		m.killProcessUsingPort(port)
	}

	if runtime.GOOS == "windows" {
		// On Windows, kill common development processes by name
		m.killWindowsDevProcesses()
	} else {
		// Also check for common development server patterns and package managers
		// Be more specific to avoid killing system processes
		m.killProcessesByPattern("pnpm run dev")
		m.killProcessesByPattern("npm run dev")
		m.killProcessesByPattern("yarn run dev")
		m.killProcessesByPattern("pnpm run start")
		m.killProcessesByPattern("npm run start")
		m.killProcessesByPattern("yarn run start")
		m.killProcessesByPattern("next dev")
		m.killProcessesByPattern("next-server")
		m.killProcessesByPattern("webpack-dev-server")
		m.killProcessesByPattern("vite")
		m.killProcessesByPattern("dev-server")

		// Kill any remaining node processes that are part of dev servers
		// But be careful not to kill system node processes
		m.killNodeDevProcesses()
	}
}

// killProcessUsingPort finds and kills the process using a specific port
func (m *Manager) killProcessUsingPort(port int) {
	if runtime.GOOS == "windows" {
		// Windows implementation
		if pid, err := findProcessUsingPort(port); err == nil && pid > 0 {
			killProcessByPID(pid)
		}
	} else {
		// Unix implementation using lsof with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "lsof", "-ti", fmt.Sprintf(":%d", port))
		output, err := cmd.Output()
		if err != nil {
			return
		}

		lines := strings.TrimSpace(string(output))
		if lines == "" {
			return
		}

		for _, line := range strings.Split(lines, "\n") {
			if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
				// Use the more aggressive killProcessTree for port-based killing
				m.killProcessTree(pid)
			}
		}
	}
}

// killProcessesByPattern kills processes matching a pattern
func (m *Manager) killProcessesByPattern(pattern string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pgrep", "-f", pattern)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.TrimSpace(string(output))
	if lines == "" {
		return
	}

	for _, line := range strings.Split(lines, "\n") {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			// Use the more aggressive killProcessTree for pattern-based killing
			m.killProcessTree(pid)
		}
	}
}

// killNodeDevProcesses kills node processes that are likely development servers
func (m *Manager) killNodeDevProcesses() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get all node processes with their command lines
	cmd := exec.CommandContext(ctx, "ps", "-eo", "pid,cmd")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(output), "\n")
	var devNodePIDs []int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split PID and command
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pid, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		cmdLine := strings.Join(parts[1:], " ")

		// Check if this looks like a development server node process
		if strings.Contains(cmdLine, "node") && (strings.Contains(cmdLine, "next/dist/bin/next") ||
			strings.Contains(cmdLine, "webpack") ||
			strings.Contains(cmdLine, "dev-server") ||
			strings.Contains(cmdLine, "turbopack") ||
			(strings.Contains(cmdLine, "pnpm") && strings.Contains(cmdLine, "dev")) ||
			(strings.Contains(cmdLine, "npm") && strings.Contains(cmdLine, "dev")) ||
			(strings.Contains(cmdLine, "yarn") && strings.Contains(cmdLine, "dev"))) {

			// Exclude system processes and VSCode processes
			if !strings.Contains(cmdLine, "vscode-server") &&
				!strings.Contains(cmdLine, "mcp-inspector") &&
				!strings.Contains(cmdLine, "/usr/lib/") &&
				!strings.Contains(cmdLine, "/opt/") {
				devNodePIDs = append(devNodePIDs, pid)
			}
		}
	}

	// Kill the identified development server processes
	for _, pid := range devNodePIDs {
		m.killProcessTree(pid)
	}
}

// killWindowsDevProcesses kills common development processes on Windows
func (m *Manager) killWindowsDevProcesses() {
	m.killWindowsDevProcessesImpl()
}

// AI Coder Integration Methods

// SetAICoderManager initializes AI coder integration
func (m *Manager) SetAICoderManager(mgr *aicoder.AICoderManager) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.aiCoderMgr = mgr

	// Initialize AI coder integration if not already done
	if m.aiCoderIntegration == nil {
		m.aiCoderIntegration = NewAICoderIntegration(m, m.eventBus)
	}

	// Initialize the integration with the AI coder manager
	if err := m.aiCoderIntegration.Initialize(mgr); err != nil {
		// Log error but don't fail - integration should be optional
		if m.eventBus != nil {
			m.eventBus.Publish(events.Event{
				Type:      "process.integration.error",
				ProcessID: "ai-coder-integration",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"error": fmt.Sprintf("Failed to initialize AI coder integration: %v", err),
				},
			})
		}
	}

	// Start monitoring AI coder processes
	go m.monitorAICoders()
}

// monitorAICoders monitors AI coder processes and syncs with process manager
func (m *Manager) monitorAICoders() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		aiCoderMgr := m.aiCoderMgr
		m.mu.RUnlock()

		if aiCoderMgr == nil {
			continue
		}

		coders := aiCoderMgr.ListCoders()
		m.syncAICoderProcesses(coders)
	}
}

// syncAICoderProcesses synchronizes AI coder processes with process manager
func (m *Manager) syncAICoderProcesses(coders []*aicoder.AICoderProcess) {
	// Create or update process entries for AI coders
	for _, coder := range coders {
		processID := fmt.Sprintf("ai-coder-%s", coder.ID)

		if _, exists := m.processes.Load(processID); exists {
			// Update existing process - for AI coder processes, we need to handle updates differently
			// Since we can't easily cast from Process to AICoderProcess, we'll recreate it
			m.processes.Delete(processID)
		}

		// Create new AI coder process entry
		aiCoderProcess := NewAICoderProcess(coder)
		m.processes.Store(processID, aiCoderProcess.Process)

		// Emit process started event
		if m.eventBus != nil {
			m.eventBus.Publish(events.Event{
				Type:      events.ProcessStarted,
				ProcessID: processID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"name":      fmt.Sprintf("AI Coder: %s", coder.Name),
					"type":      "ai-coder",
					"status":    string(coder.Status),
					"provider":  coder.Provider,
					"workspace": coder.WorkspaceDir,
					"task":      coder.Task,
					"progress":  coder.Progress,
				},
			})
		}
	}

	// Remove processes for deleted AI coders
	m.cleanupStaleAICoderProcesses(coders)
}

// cleanupStaleAICoderProcesses removes AI coder processes that no longer exist
func (m *Manager) cleanupStaleAICoderProcesses(currentCoders []*aicoder.AICoderProcess) {
	// Create map of current coder IDs for quick lookup
	currentCoderIDs := make(map[string]bool)
	for _, coder := range currentCoders {
		currentCoderIDs[coder.ID] = true
	}

	// Find and remove stale AI coder processes
	m.processes.Range(func(key, value interface{}) bool {
		processID, ok := key.(string)
		if !ok {
			return true // continue iteration
		}

		process, ok := value.(*Process)
		if !ok {
			return true // continue iteration
		}

		if strings.HasPrefix(processID, "ai-coder-") {
			coderID := strings.TrimPrefix(processID, "ai-coder-")
			if !currentCoderIDs[coderID] {
				// This AI coder no longer exists, remove it
				m.processes.Delete(processID)

				// Emit process exited event
				if m.eventBus != nil {
					m.eventBus.Publish(events.Event{
						Type:      events.ProcessExited,
						ProcessID: processID,
						Timestamp: time.Now(),
						Data: map[string]interface{}{
							"name":      process.Name,
							"type":      "ai-coder",
							"exit_code": 0, // AI coders don't have exit codes
							"reason":    "deleted",
						},
					})
				}
			}
		}
		return true // continue iteration
	})
}

// GetAICoderProcesses returns all AI coder processes
func (m *Manager) GetAICoderProcesses() []*AICoderProcess {
	var aiCoderProcesses []*AICoderProcess

	// Use sync.Map.Range for lock-free iteration
	m.processes.Range(func(key, value interface{}) bool {
		if processID, ok := key.(string); ok {
			if strings.HasPrefix(processID, "ai-coder-") {
				// Create AI coder process wrapper
				coderID := strings.TrimPrefix(processID, "ai-coder-")
				if m.aiCoderMgr != nil {
					if coder, exists := m.aiCoderMgr.GetCoder(coderID); exists {
						aiCoderProcess := NewAICoderProcess(coder)
						aiCoderProcesses = append(aiCoderProcesses, aiCoderProcess)
					}
				}
			}
		}
		return true // continue iteration
	})

	return aiCoderProcesses
}

// IsAICoderProcess checks if a process ID belongs to an AI coder
func (m *Manager) IsAICoderProcess(processID string) bool {
	return strings.HasPrefix(processID, "ai-coder-")
}

// GetAICoderIntegration returns the AI coder integration instance
func (m *Manager) GetAICoderIntegration() *AICoderIntegration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.aiCoderIntegration
}
