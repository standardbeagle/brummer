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
	"time"

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
}

// Thread-safe getters for Process fields
func (p *Process) GetStatus() ProcessStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Status
}

func (p *Process) GetStartTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.StartTime
}

func (p *Process) GetEndTime() *time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.EndTime
}

func (p *Process) GetExitCode() *int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ExitCode
}

// Thread-safe setters for Process fields  
func (p *Process) SetStatus(status ProcessStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Status = status
}

type ProcessStatus string

const (
	StatusPending ProcessStatus = "pending"
	StatusRunning ProcessStatus = "running"
	StatusStopped ProcessStatus = "stopped"
	StatusFailed  ProcessStatus = "failed"
	StatusSuccess ProcessStatus = "success"
)

type Manager struct {
	processes      map[string]*Process
	packageJSON    *parser.PackageJSON
	packageMgr     parser.PackageManager
	userPackageMgr *parser.PackageManager
	workDir        string
	eventBus       *events.EventBus
	logCallbacks   []LogCallback
	installedMgrs  []parser.InstalledPackageManager
	mu             sync.RWMutex
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
		processes:      make(map[string]*Process),
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

	m.processes[processID] = process

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

	m.processes[processID] = process

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
	m.mu.RLock()
	process, exists := m.processes[processID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process %s not found", processID)
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
		go m.killProcessesByPort()
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
				m.killProcessesByPort()
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

func (m *Manager) GetProcess(processID string) (*Process, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	process, exists := m.processes[processID]
	return process, exists
}

func (m *Manager) GetAllProcesses() []*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	processes := make([]*Process, 0, len(m.processes))
	for _, p := range m.processes {
		processes = append(processes, p)
	}
	return processes
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
	m.mu.RLock()
	var processIDs []string
	for id, proc := range m.processes {
		if proc.GetStatus() == StatusRunning {
			processIDs = append(processIDs, id)
		}
	}
	m.mu.RUnlock()

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
		m.killProcessesByPort()
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

// killProcessesByPort kills processes using development ports
func (m *Manager) killProcessesByPort() {
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
		if strings.Contains(cmdLine, "node") && (
			strings.Contains(cmdLine, "next/dist/bin/next") ||
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
