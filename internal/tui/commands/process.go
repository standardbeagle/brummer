package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ProcessCommand represents a process management command
type ProcessCommand struct {
	*BaseCommand
	command     []string
	workingDir  string
	env         []string
	operationType ProcessOperationType
	startTime   time.Time
	pid         int
}

// ProcessOperationType represents the type of process operation
type ProcessOperationType int

const (
	ProcessStart ProcessOperationType = iota
	ProcessStop
	ProcessRestart
	ProcessStatus
	ProcessList
	ProcessKill
)

// ProcessResultMsg represents the result of a process command
type ProcessResultMsg struct {
	ID           string             `json:"id"`
	Type         CommandType        `json:"type"`
	Result       interface{}        `json:"result,omitempty"`
	Error        error              `json:"error,omitempty"`
	Context      OperationContext   `json:"context"`
	Duration     time.Duration      `json:"duration"`
	OperationType ProcessOperationType `json:"operationType"`
	PID          int                `json:"pid,omitempty"`
	Command      []string           `json:"command,omitempty"`
}

// NewProcessCommand creates a new process command
func NewProcessCommand(id string, command []string, opType ProcessOperationType, workingDir string, context OperationContext) *ProcessCommand {
	return &ProcessCommand{
		BaseCommand:   NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		command:       command,
		workingDir:    workingDir,
		env:           os.Environ(),
		operationType: opType,
	}
}

// WithEnv sets environment variables for the process
func (c *ProcessCommand) WithEnv(env []string) *ProcessCommand {
	c.env = append(c.env, env...)
	return c
}

// WithWorkingDir sets the working directory for the process
func (c *ProcessCommand) WithWorkingDir(dir string) *ProcessCommand {
	c.workingDir = dir
	return c
}

// Run executes the process command
func (c *ProcessCommand) Run() tea.Msg {
	start := time.Now()

	// Create command context for timeout handling
	cmdCtx := NewCommandContext(c.Context())
	defer cmdCtx.Cleanup()

	var result interface{}
	var err error

	switch c.operationType {
	case ProcessStart:
		result, err = c.startProcess(cmdCtx)
	case ProcessStop:
		result, err = c.stopProcess(cmdCtx)
	case ProcessRestart:
		result, err = c.restartProcess(cmdCtx)
	case ProcessStatus:
		result, err = c.getProcessStatus(cmdCtx)
	case ProcessList:
		result, err = c.listProcesses(cmdCtx)
	case ProcessKill:
		result, err = c.killProcess(cmdCtx)
	default:
		err = fmt.Errorf("unknown process operation type: %d", c.operationType)
	}

	duration := time.Since(start)

	// Check for timeout
	if cmdCtx.IsTimedOut() {
		err = fmt.Errorf("process operation timed out after %v", duration)
	}

	return ProcessResultMsg{
		ID:           c.ID(),
		Type:         c.Type(),
		Result:       result,
		Error:        err,
		Context:      c.Context(),
		Duration:     duration,
		OperationType: c.operationType,
		PID:          c.pid,
		Command:      c.command,
	}
}

// startProcess starts a new process
func (c *ProcessCommand) startProcess(cmdCtx *CommandContext) (interface{}, error) {
	if len(c.command) == 0 {
		return nil, fmt.Errorf("no command specified")
	}

	// Create the command
	cmd := exec.Command(c.command[0], c.command[1:]...)

	// Set working directory
	if c.workingDir != "" {
		cmd.Dir = c.workingDir
	}

	// Set environment
	cmd.Env = c.env

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	c.pid = cmd.Process.Pid
	c.startTime = time.Now()

	return ProcessInfo{
		PID:       c.pid,
		Command:   strings.Join(c.command, " "),
		WorkingDir: c.workingDir,
		StartTime: c.startTime,
		Status:    "running",
	}, nil
}

// stopProcess stops a running process
func (c *ProcessCommand) stopProcess(cmdCtx *CommandContext) (interface{}, error) {
	if c.pid <= 0 {
		return nil, fmt.Errorf("no process ID available")
	}

	// Try to find the process
	process, err := os.FindProcess(c.pid)
	if err != nil {
		return nil, fmt.Errorf("failed to find process %d: %w", c.pid, err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return nil, fmt.Errorf("failed to send SIGTERM to process %d: %w", c.pid, err)
	}

	// Wait a bit for graceful shutdown
	time.Sleep(100 * time.Millisecond)

	// Check if process is still running
	if isProcessRunning(c.pid) {
		// Force kill with SIGKILL
		if err := process.Kill(); err != nil {
			return nil, fmt.Errorf("failed to kill process %d: %w", c.pid, err)
		}
	}

	return ProcessActionResult{
		PID:    c.pid,
		Action: "stopped",
		Status: "success",
	}, nil
}

// restartProcess restarts a process
func (c *ProcessCommand) restartProcess(cmdCtx *CommandContext) (interface{}, error) {
	// Stop the process first
	if c.pid > 0 {
		if _, err := c.stopProcess(cmdCtx); err != nil {
			return nil, fmt.Errorf("failed to stop process before restart: %w", err)
		}
	}

	// Wait a moment for cleanup
	time.Sleep(100 * time.Millisecond)

	// Start the process again
	return c.startProcess(cmdCtx)
}

// getProcessStatus gets the status of a process
func (c *ProcessCommand) getProcessStatus(cmdCtx *CommandContext) (interface{}, error) {
	if c.pid <= 0 {
		return nil, fmt.Errorf("no process ID available")
	}

	status := "unknown"
	if isProcessRunning(c.pid) {
		status = "running"
	} else {
		status = "stopped"
	}

	return ProcessStatusInfo{
		PID:     c.pid,
		Command: strings.Join(c.command, " "),
		Status:  status,
		Uptime:  time.Since(c.startTime),
	}, nil
}

// listProcesses lists all processes managed by the application
func (c *ProcessCommand) listProcesses(cmdCtx *CommandContext) (interface{}, error) {
	// For now, return basic process info
	// In a real implementation, this would query a process registry
	return ProcessListInfo{
		Processes: []ProcessInfo{},
		Count:     0,
	}, nil
}

// killProcess forcefully kills a process
func (c *ProcessCommand) killProcess(cmdCtx *CommandContext) (interface{}, error) {
	if c.pid <= 0 {
		return nil, fmt.Errorf("no process ID available")
	}

	process, err := os.FindProcess(c.pid)
	if err != nil {
		return nil, fmt.Errorf("failed to find process %d: %w", c.pid, err)
	}

	if err := process.Kill(); err != nil {
		return nil, fmt.Errorf("failed to kill process %d: %w", c.pid, err)
	}

	return ProcessActionResult{
		PID:    c.pid,
		Action: "killed",
		Status: "success",
	}, nil
}

// isProcessRunning checks if a process is still running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Data structures for process operations

type ProcessInfo struct {
	PID        int       `json:"pid"`
	Command    string    `json:"command"`
	WorkingDir string    `json:"workingDir"`
	StartTime  time.Time `json:"startTime"`
	Status     string    `json:"status"`
}

type ProcessActionResult struct {
	PID    int    `json:"pid"`
	Action string `json:"action"`
	Status string `json:"status"`
}

type ProcessStatusInfo struct {
	PID     int           `json:"pid"`
	Command string        `json:"command"`
	Status  string        `json:"status"`
	Uptime  time.Duration `json:"uptime"`
}

type ProcessListInfo struct {
	Processes []ProcessInfo `json:"processes"`
	Count     int           `json:"count"`
}

// Factory functions for common process operations

// NewStartScriptCommand creates a command to start a script
func NewStartScriptCommand(id string, scriptName string, args []string, workingDir string, context OperationContext) *ProcessCommand {
	// Try to find the script in common locations
	scriptPath := findScriptPath(scriptName, workingDir)

	command := append([]string{scriptPath}, args...)
	return NewProcessCommand(id, command, ProcessStart, workingDir, context)
}

// NewStartNPMCommand creates a command to start an npm script
func NewStartNPMCommand(id string, scriptName string, workingDir string, context OperationContext) *ProcessCommand {
	command := []string{"npm", "run", scriptName}
	return NewProcessCommand(id, command, ProcessStart, workingDir, context)
}

// NewStartNodeCommand creates a command to start a Node.js process
func NewStartNodeCommand(id string, scriptFile string, args []string, workingDir string, context OperationContext) *ProcessCommand {
	command := append([]string{"node", scriptFile}, args...)
	return NewProcessCommand(id, command, ProcessStart, workingDir, context)
}

// NewStopProcessCommand creates a command to stop a process
func NewStopProcessCommand(id string, pid int, context OperationContext) *ProcessCommand {
	cmd := &ProcessCommand{
		BaseCommand:   NewBaseCommand(id, CommandTypeProcess, PriorityHigh, context),
		command:       []string{},
		operationType: ProcessStop,
		pid:          pid,
	}
	return cmd
}

// Helper functions

func findScriptPath(scriptName string, workingDir string) string {
	// Check if it's already a path
	if filepath.IsAbs(scriptName) {
		return scriptName
	}

	// Check working directory
	if workingDir != "" {
		path := filepath.Join(workingDir, scriptName)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Check current directory
	if path, err := filepath.Abs(scriptName); err == nil {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Return the original name if not found
	return scriptName
}

// Message interface implementation
func (m ProcessResultMsg) Message() tea.Msg {
	return m
}

// IsSuccess returns true if the process operation completed successfully
func (m ProcessResultMsg) IsSuccess() bool {
	return m.Error == nil
}

// IsFailure returns true if the process operation failed
func (m ProcessResultMsg) IsFailure() bool {
	return m.Error != nil
}

// String returns a string representation of the process result
func (m ProcessResultMsg) String() string {
	if m.IsSuccess() {
		return fmt.Sprintf("ProcessResult{id=%s, type=%v, pid=%d, duration=%v}",
			m.ID, m.OperationType, m.PID, m.Duration)
	}
	return fmt.Sprintf("ProcessResult{id=%s, type=%v, error=%v, duration=%v}",
		m.ID, m.OperationType, m.Error, m.Duration)
}

// Process management utilities

func GetProcessStatus(pid int) string {
	if isProcessRunning(pid) {
		return "running"
	}
	return "stopped"
}

func GetProcessUptime(pid int, startTime time.Time) time.Duration {
	if !isProcessRunning(pid) {
		return 0
	}
	return time.Since(startTime)
}

// Process monitoring functions

func MonitorProcess(pid int, timeout time.Duration) bool {
	start := time.Now()
	for time.Since(start) < timeout {
		if !isProcessRunning(pid) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
	return isProcessRunning(pid)
}

func WaitForProcess(pid int, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		if !isProcessRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("process %d did not finish within %v", pid, timeout)
}

// Process registry for tracking managed processes

type ProcessRegistry struct {
	processes map[int]*ProcessInfo
}

func NewProcessRegistry() *ProcessRegistry {
	return &ProcessRegistry{
		processes: make(map[int]*ProcessInfo),
	}
}

func (r *ProcessRegistry) AddProcess(info *ProcessInfo) {
	r.processes[info.PID] = info
}

func (r *ProcessRegistry) RemoveProcess(pid int) {
	delete(r.processes, pid)
}

func (r *ProcessRegistry) GetProcess(pid int) (*ProcessInfo, bool) {
	info, exists := r.processes[pid]
	return info, exists
}

func (r *ProcessRegistry) ListProcesses() []ProcessInfo {
	var processes []ProcessInfo
	for _, info := range r.processes {
		processes = append(processes, *info)
	}
	return processes
}

func (r *ProcessRegistry) Cleanup() {
	// Remove dead processes
	for pid, info := range r.processes {
		if !isProcessRunning(pid) {
			delete(r.processes, pid)
			info.Status = "stopped"
		}
	}
}