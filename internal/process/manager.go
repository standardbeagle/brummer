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
		process.Status = StatusFailed
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
		process.Status = StatusFailed
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
	m.killProcessesByPort()

	process.Status = StatusStopped
	now := time.Now()
	process.EndTime = &now
	exitCode := -1
	process.ExitCode = &exitCode
	process.mu.Unlock()

	// Give processes a moment to die
	time.Sleep(100 * time.Millisecond)

	// Double-check that the main process is dead
	if mainPID > 0 {
		m.ensureProcessDead(mainPID)
	}

	// Publish stop event
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
		if proc.Status == StatusRunning {
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

	// Also kill any remaining development processes
	m.killProcessesByPort()

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

	// Use ps to find child processes
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(parentPID))
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
		// Also check for common development server patterns
		m.killProcessesByPattern("next dev")
		m.killProcessesByPattern("next-server")
		m.killProcessesByPattern("webpack")
		m.killProcessesByPattern("vite")
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
		// Unix implementation using lsof
		cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
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
				killProcessByPID(pid)
			}
		}
	}
}

// killProcessesByPattern kills processes matching a pattern
func (m *Manager) killProcessesByPattern(pattern string) {
	cmd := exec.Command("pgrep", "-f", pattern)
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
			killProcessByPID(pid)
		}
	}
}

// killWindowsDevProcesses kills common development processes on Windows
func (m *Manager) killWindowsDevProcesses() {
	m.killWindowsDevProcessesImpl()
}
