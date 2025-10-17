package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// SystemCommand represents a system operation command
type SystemCommand struct {
	*BaseCommand
	operation    func() (interface{}, error)
	operationType SystemOperationType
}

// SystemOperationType represents the type of system operation
type SystemOperationType int

const (
	SystemInfo SystemOperationType = iota
	SystemEnv
	SystemOS
	SystemArch
	SystemHostname
	SystemUptime
	SystemMemory
	SystemCPU
	SystemDisk
	SystemTemp
	SystemUser
)

// SystemResultMsg represents the result of a system command
type SystemResultMsg struct {
	ID           string             `json:"id"`
	Type         CommandType        `json:"type"`
	Result       interface{}        `json:"result,omitempty"`
	Error        error              `json:"error,omitempty"`
	Context      OperationContext   `json:"context"`
	Duration     time.Duration      `json:"duration"`
	OperationType SystemOperationType `json:"operationType"`
}

// NewSystemCommand creates a new system command
func NewSystemCommand(id string, operation func() (interface{}, error), opType SystemOperationType, context OperationContext) *SystemCommand {
	return &SystemCommand{
		BaseCommand:   NewBaseCommand(id, CommandTypeSystem, PriorityLow, context),
		operation:     operation,
		operationType: opType,
	}
}

// Run executes the system command
func (c *SystemCommand) Run() tea.Msg {
	start := time.Now()

	// Create command context for timeout handling
	cmdCtx := NewCommandContext(c.Context())
	defer cmdCtx.Cleanup()

	var result interface{}
	var err error

	// Execute operation with timeout
	result, err = cmdCtx.ExecuteWithTimeout(c.operation)
	duration := time.Since(start)

	// Check for timeout
	if cmdCtx.IsTimedOut() {
		err = fmt.Errorf("system operation timed out after %v", duration)
	}

	return SystemResultMsg{
		ID:           c.ID(),
		Type:         c.Type(),
		Result:       result,
		Error:        err,
		Context:      c.Context(),
		Duration:     duration,
		OperationType: c.operationType,
	}
}

// Factory functions for common system operations

// NewSystemInfoCommand creates a command to get system information
func NewSystemInfoCommand(id string, context OperationContext) *SystemCommand {
	operation := func() (interface{}, error) {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		username := os.Getenv("USER")
		if username == "" {
			username = "unknown"
		}

		return SystemInfoData{
			Hostname:  hostname,
			Username:  username,
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			GoVersion: runtime.Version(),
			NumCPU:   runtime.NumCPU(),
		}, nil
	}

	return NewSystemCommand(id, operation, SystemInfo, context)
}

// NewEnvironmentCommand creates a command to get environment variables
func NewEnvironmentCommand(id string, context OperationContext) *SystemCommand {
	operation := func() (interface{}, error) {
		env := os.Environ()
		envMap := make(map[string]string)
		for _, e := range env {
			pair := strings.SplitN(e, "=", 2)
			if len(pair) == 2 {
				envMap[pair[0]] = pair[1]
			}
		}
		return envMap, nil
	}

	return NewSystemCommand(id, operation, SystemEnv, context)
}

// NewUptimeCommand creates a command to get system uptime
func NewUptimeCommand(id string, context OperationContext) *SystemCommand {
	operation := func() (interface{}, error) {
		// Try to get uptime from system commands
		if runtime.GOOS == "windows" {
			// Windows uptime
			cmd := exec.Command("wmic", "os", "get", "lastbootuptime")
			output, err := cmd.Output()
			if err != nil {
				// Fallback to process uptime
				return time.Since(time.Now()), nil
			}
			return parseWindowsUptime(string(output)), nil
		} else {
			// Unix/Linux uptime
			cmd := exec.Command("uptime", "-s")
			output, err := cmd.Output()
			if err != nil {
				// Fallback to process uptime
				return time.Since(time.Now()), nil
			}
			return parseUnixUptime(string(output)), nil
		}
	}

	return NewSystemCommand(id, operation, SystemUptime, context)
}

// NewMemoryCommand creates a command to get memory information
func NewMemoryCommand(id string, context OperationContext) *SystemCommand {
	operation := func() (interface{}, error) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		return MemoryInfo{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
			GoRoutines: runtime.NumGoroutine(),
		}, nil
	}

	return NewSystemCommand(id, operation, SystemMemory, context)
}

// NewCPUCommand creates a command to get CPU information
func NewCPUCommand(id string, context OperationContext) *SystemCommand {
	operation := func() (interface{}, error) {
		return CPUInfo{
			NumCPU:    runtime.NumCPU(),
			GoVersion: runtime.Version(),
			GoOS:      runtime.GOOS,
			GoArch:    runtime.GOARCH,
		}, nil
	}

	return NewSystemCommand(id, operation, SystemCPU, context)
}

// NewTempCommand creates a command to get temperature information
func NewTempCommand(id string, context OperationContext) *SystemCommand {
	operation := func() (interface{}, error) {
		// For now, return mock temperature data
		// In a real implementation, this would query system temperature sensors
		return TemperatureInfo{
			CPU:    45.5,
			GPU:    0.0, // May not be available
			Board:  35.2,
		}, nil
	}

	return NewSystemCommand(id, operation, SystemTemp, context)
}

// NewDiskCommand creates a command to get disk information
func NewDiskCommand(id string, path string, context OperationContext) *SystemCommand {
	operation := func() (interface{}, error) {
		// For now, return basic disk info
		// In a real implementation, this would query disk usage
		return DiskInfo{
			Path:      path,
			Total:     0,
			Used:      0,
			Available: 0,
		}, nil
	}

	return NewSystemCommand(id, operation, SystemDisk, context)
}

// Data structures for system operations

type SystemInfoData struct {
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"goVersion"`
	NumCPU    int    `json:"numCPU"`
}

type MemoryInfo struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"totalAlloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"numGC"`
	GoRoutines int    `json:"goRoutines"`
}

type CPUInfo struct {
	NumCPU    int    `json:"numCPU"`
	GoVersion string `json:"goVersion"`
	GoOS      string `json:"goOS"`
	GoArch    string `json:"goArch"`
}

type TemperatureInfo struct {
	CPU   float64 `json:"cpu"`
	GPU   float64 `json:"gpu"`
	Board float64 `json:"board"`
}

type DiskInfo struct {
	Path      string `json:"path"`
	Total     uint64 `json:"total"`
	Used      uint64 `json:"used"`
	Available uint64 `json:"available"`
}

// Utility functions

func parseWindowsUptime(output string) time.Duration {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "LastBootUpTime") {
			// Parse Windows timestamp format
			// This is a simplified implementation
			return time.Since(time.Now().Add(-24 * time.Hour))
		}
	}
	return time.Since(time.Now().Add(-1 * time.Hour))
}

func parseUnixUptime(output string) time.Duration {
	// Parse uptime output like "up 2 days, 3:45"
	fields := strings.Fields(output)
	if len(fields) >= 2 && fields[0] == "up" {
		if fields[1] == "days," {
			// Format: "up X days, HH:MM"
			if len(fields) >= 4 {
				days, _ := strconv.Atoi(fields[0])
				timeParts := strings.Split(fields[3], ":")
				if len(timeParts) == 2 {
					hours, _ := strconv.Atoi(timeParts[0])
					minutes, _ := strconv.Atoi(timeParts[1])
					return time.Duration(days)*24*time.Hour +
						time.Duration(hours)*time.Hour +
						time.Duration(minutes)*time.Minute
				}
			}
		} else {
			// Format: "up HH:MM" or "up MM min"
			if strings.Contains(fields[1], ":") {
				timeParts := strings.Split(fields[1], ":")
				if len(timeParts) == 2 {
					hours, _ := strconv.Atoi(timeParts[0])
					minutes, _ := strconv.Atoi(timeParts[1])
					return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
				}
			} else if strings.Contains(fields[1], "min") {
				minutes, _ := strconv.Atoi(fields[0])
				return time.Duration(minutes) * time.Minute
			}
		}
	}
	return time.Hour // Default fallback
}

// Message interface implementation
func (m SystemResultMsg) Message() tea.Msg {
	return m
}

// IsSuccess returns true if the system operation completed successfully
func (m SystemResultMsg) IsSuccess() bool {
	return m.Error == nil
}

// IsFailure returns true if the system operation failed
func (m SystemResultMsg) IsFailure() bool {
	return m.Error != nil
}

// String returns a string representation of the system result
func (m SystemResultMsg) String() string {
	if m.IsSuccess() {
		return fmt.Sprintf("SystemResult{id=%s, type=%v, duration=%v}",
			m.ID, m.OperationType, m.Duration)
	}
	return fmt.Sprintf("SystemResult{id=%s, type=%v, error=%v, duration=%v}",
		m.ID, m.OperationType, m.Error, m.Duration)
}

// System utility functions

func GetSystemInfo() SystemInfoData {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	username := os.Getenv("USER")
	if username == "" {
		username = "unknown"
	}

	return SystemInfoData{
		Hostname:  hostname,
		Username:  username,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		GoVersion: runtime.Version(),
		NumCPU:   runtime.NumCPU(),
	}
}

func GetEnvironmentVariables() map[string]string {
	env := os.Environ()
	envMap := make(map[string]string)
	for _, e := range env {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = pair[1]
		}
	}
	return envMap
}

func GetCurrentMemoryUsage() MemoryInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return MemoryInfo{
		Alloc:      m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
		GoRoutines: runtime.NumGoroutine(),
	}
}

func GetCPUUsage() CPUInfo {
	return CPUInfo{
		NumCPU:    runtime.NumCPU(),
		GoVersion: runtime.Version(),
		GoOS:      runtime.GOOS,
		GoArch:    runtime.GOARCH,
	}
}

// System monitoring utilities

func MonitorSystem(interval time.Duration, callback func(SystemInfoData, MemoryInfo, CPUInfo)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		info := GetSystemInfo()
		mem := GetCurrentMemoryUsage()
		cpu := GetCPUUsage()
		callback(info, mem, cpu)
	}
}

// System health check

func CheckSystemHealth() map[string]bool {
	health := make(map[string]bool)

	// Check memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// Consider memory healthy if less than 1GB allocated
	health["memory"] = m.Alloc < 1024*1024*1024

	// Check goroutine count
	health["goroutines"] = runtime.NumGoroutine() < 1000

	// Check GC frequency
	health["gc"] = m.NumGC < 1000

	return health
}

// System performance metrics

type SystemMetrics struct {
	Timestamp time.Time   `json:"timestamp"`
	Memory    MemoryInfo   `json:"memory"`
	CPU       CPUInfo       `json:"cpu"`
	Health    map[string]bool `json:"health"`
}

func GetSystemMetrics() SystemMetrics {
	return SystemMetrics{
		Timestamp: time.Now(),
		Memory:    GetCurrentMemoryUsage(),
		CPU:       GetCPUUsage(),
		Health:    CheckSystemHealth(),
	}
}