package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/beagle/brummer/internal/config"
	"github.com/beagle/brummer/internal/parser"
	"github.com/beagle/brummer/pkg/events"
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
	StatusPending  ProcessStatus = "pending"
	StatusRunning  ProcessStatus = "running"
	StatusStopped  ProcessStatus = "stopped"
	StatusFailed   ProcessStatus = "failed"
	StatusSuccess  ProcessStatus = "success"
)

type Manager struct {
	processes       map[string]*Process
	packageJSON     *parser.PackageJSON
	packageMgr      parser.PackageManager
	userPackageMgr  *parser.PackageManager
	workDir         string
	eventBus        *events.EventBus
	logCallbacks    []LogCallback
	installedMgrs   []parser.InstalledPackageManager
	mu              sync.RWMutex
}

type LogCallback func(processID string, line string, isError bool)

func NewManager(workDir string, eventBus *events.EventBus) (*Manager, error) {
	pkgJSON, err := parser.ParsePackageJSON(workDir + "/package.json")
	if err != nil {
		return nil, err
	}

	// Load config
	cfg, _ := config.Load()
	
	// Detect installed package managers
	installedMgrs := parser.DetectInstalledPackageManagers()

	m := &Manager{
		processes:     make(map[string]*Process),
		packageJSON:   pkgJSON,
		workDir:       workDir,
		eventBus:      eventBus,
		installedMgrs: installedMgrs,
		userPackageMgr: cfg.PreferredPackageManager,
	}

	// Set initial package manager based on detection
	m.updatePackageManager()

	return m, nil
}

func (m *Manager) GetScripts() map[string]string {
	return m.packageJSON.Scripts
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
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		
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
	process.mu.Unlock()

	process.cancel()
	
	process.mu.Lock()
	process.Status = StatusStopped
	process.mu.Unlock()

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