package services

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/process"
)

// ProcessServiceBridge implements ProcessService using process.Manager
type ProcessServiceBridge struct {
	manager *process.Manager
}

// NewProcessServiceBridge creates a bridge to the existing process manager
func NewProcessServiceBridge(manager *process.Manager) *ProcessServiceBridge {
	return &ProcessServiceBridge{
		manager: manager,
	}
}

// Start a script or command
func (s *ProcessServiceBridge) Start(name string) tea.Cmd {
	return func() tea.Msg {
		if s.manager == nil {
			return CommandResultMsg{
				Success: false,
				Message: "Process manager not available",
			}
		}
		
		// Check if it's a script
		scripts := s.manager.GetScripts()
		if _, isScript := scripts[name]; isScript {
			_, err := s.manager.StartScript(name)
			if err != nil {
				return CommandResultMsg{
					Success: false,
					Message: fmt.Sprintf("Failed to start %s: %v", name, err),
				}
			}
			return CommandResultMsg{
				Success: true,
				Message: fmt.Sprintf("Started %s", name),
			}
		}
		
		// Try as a command
		_, err := s.manager.StartCommand(name, name, []string{})
		if err != nil {
			return CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("Failed to start %s: %v", name, err),
			}
		}
		return CommandResultMsg{
			Success: true,
			Message: fmt.Sprintf("Started %s", name),
		}
	}
}

// Stop a running process
func (s *ProcessServiceBridge) Stop(processID string) tea.Cmd {
	return func() tea.Msg {
		if s.manager == nil {
			return CommandResultMsg{
				Success: false,
				Message: "Process manager not available",
			}
		}
		
		err := s.manager.StopProcess(processID)
		if err != nil {
			return CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("Failed to stop %s: %v", processID, err),
			}
		}
		
		return CommandResultMsg{
			Success: true,
			Message: fmt.Sprintf("Stopped %s", processID),
		}
	}
}

// Restart a process
func (s *ProcessServiceBridge) Restart(processID string) tea.Cmd {
	return func() tea.Msg {
		if s.manager == nil {
			return CommandResultMsg{
				Success: false,
				Message: "Process manager not available",
			}
		}
		
		// Get process info before stopping
		proc, exists := s.manager.GetProcess(processID)
		if !exists {
			return CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("Process %s not found", processID),
			}
		}
		
		// Stop the process
		err := s.manager.StopProcess(processID)
		if err != nil {
			return CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("Failed to stop %s: %v", processID, err),
			}
		}
		
		// Wait a moment for cleanup
		time.Sleep(100 * time.Millisecond)
		
		// Restart with same name
		_, err = s.manager.StartScript(proc.Name)
		if err != nil {
			// Try as command if script fails  
			_, err = s.manager.StartCommand(proc.Name, proc.Name, []string{})
			if err != nil {
				return CommandResultMsg{
					Success: false,
					Message: fmt.Sprintf("Failed to restart %s: %v", proc.Name, err),
				}
			}
		}
		
		return CommandResultMsg{
			Success: true,
			Message: fmt.Sprintf("Restarted %s", proc.Name),
		}
	}
}

// GetStatus returns status of all processes
func (s *ProcessServiceBridge) GetStatus() []ProcessInfo {
	if s.manager == nil {
		return []ProcessInfo{}
	}
	
	processes := s.manager.GetAllProcesses()
	result := make([]ProcessInfo, 0, len(processes))
	
	for _, p := range processes {
		// Get PID if process has a Cmd
		pid := 0
		if p.Cmd != nil && p.Cmd.Process != nil {
			pid = p.Cmd.Process.Pid
		}
		
		result = append(result, ProcessInfo{
			ID:        p.ID,
			Name:      p.Name,
			Status:    string(p.Status),
			StartTime: p.StartTime,
			Command:   p.Script,
			PID:       pid,
		})
	}
	
	return result
}

// GetScripts returns available scripts
func (s *ProcessServiceBridge) GetScripts() map[string]string {
	if s.manager == nil {
		return map[string]string{}
	}
	return s.manager.GetScripts()
}