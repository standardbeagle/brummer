package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
)

// Message types for process operations
type processUpdateMsg struct{}

type restartProcessMsg struct {
	processName string
	message     string
	isError     bool
	clearLogs   bool
}

type restartAllMsg struct {
	message   string
	isError   bool
	clearLogs bool
	restarted int
}

// ProcessViewController manages the processes view state and rendering
type ProcessViewController struct {
	processesList   list.Model
	selectedProcess string

	// Dependencies injected from parent Model
	processMgr   *process.Manager
	width        int
	height       int
	headerHeight int
	footerHeight int
}

// NewProcessViewController creates a new process view controller
func NewProcessViewController(processMgr *process.Manager) *ProcessViewController {
	processesList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	processesList.Title = "Running Processes"
	processesList.SetShowStatusBar(false)

	return &ProcessViewController{
		processesList: processesList,
		processMgr:    processMgr,
	}
}

// UpdateSize updates the list dimensions
func (v *ProcessViewController) UpdateSize(width, height, headerHeight, footerHeight int) {
	v.width = width
	v.height = height
	v.headerHeight = headerHeight
	v.footerHeight = footerHeight

	contentHeight := height - headerHeight - footerHeight
	v.processesList.SetSize(width, contentHeight)
}

// SetSelectedProcess sets the currently selected process
func (v *ProcessViewController) SetSelectedProcess(processID string) {
	v.selectedProcess = processID
}

// GetSelectedProcess returns the currently selected process
func (v *ProcessViewController) GetSelectedProcess() string {
	return v.selectedProcess
}

// GetProcessesList returns the processes list for direct manipulation
func (v *ProcessViewController) GetProcessesList() *list.Model {
	return &v.processesList
}

// UpdateProcessList refreshes the process list with current data
func (v *ProcessViewController) UpdateProcessList() {
	processes := v.processMgr.GetAllProcesses()

	// Convert processes to list items
	var items []list.Item

	if len(processes) == 0 {
		// Add a placeholder item for empty state
		items = append(items, processItem{
			process:  nil,
			isHeader: false,
		})
	} else {
		// Group processes by status
		var running, stopped []*process.Process
		for _, proc := range processes {
			if proc.GetStatus() == process.StatusRunning {
				running = append(running, proc)
			} else {
				stopped = append(stopped, proc)
			}
		}

		// Add running processes first
		if len(running) > 0 {
			items = append(items, processItem{
				headerText: "Running Processes",
				isHeader:   true,
			})
			for _, proc := range running {
				items = append(items, processItem{
					process: proc,
				})
			}
		}

		// Add stopped processes
		if len(stopped) > 0 {
			if len(running) > 0 {
				// Add separator - empty item
				items = append(items, processItem{
					headerText: "",
					isHeader:   false,
				})
			}
			items = append(items, processItem{
				headerText: "Stopped Processes",
				isHeader:   true,
			})
			for _, proc := range stopped {
				items = append(items, processItem{
					process: proc,
				})
			}
		}
	}

	v.processesList.SetItems(items)
}

// Render renders the processes view
func (v *ProcessViewController) Render() string {
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("Select process: ↑/↓ | Stop: s | Restart: r | Restart All: Ctrl+R | View Logs: Enter")

	processes := v.processMgr.GetAllProcesses()
	if len(processes) == 0 {
		emptyState := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("No processes running. Use / for commands: /run <script> to start scripts, /restart all, /stop <process>")

		return lipgloss.JoinVertical(lipgloss.Left,
			instructions,
			"",
			emptyState,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		instructions,
		"",
		v.processesList.View(),
	)
}

// processItem uses the existing type defined in model.go

// HandleRestartProcess creates a command to restart a specific process
func (v *ProcessViewController) HandleRestartProcess(proc *process.Process) tea.Cmd {
	return func() tea.Msg {
		// Check if process is still running before trying to stop it
		if proc.GetStatus() == process.StatusRunning {
			// Stop the process and wait for it to terminate completely
			timeout := 5 * time.Second
			if err := v.processMgr.StopProcessAndWait(proc.ID, timeout); err != nil {
				return restartProcessMsg{
					processName: proc.Name,
					message:     fmt.Sprintf("Error stopping process %s: %v", proc.Name, err),
					isError:     true,
					clearLogs:   false,
				}
			}
		}

		// Clean up any finished processes before starting new one
		v.processMgr.CleanupFinishedProcesses()

		// Also clean up any processes that might be using development ports
		if proc.Name == "server" || proc.Name == "dev" || proc.Name == "start" {
			v.processMgr.KillProcessesByPort()
			// Give a moment for ports to be freed
			time.Sleep(500 * time.Millisecond)
		}

		// Now start it again
		_, err := v.processMgr.StartScript(proc.Name)
		if err != nil {
			return restartProcessMsg{
				processName: proc.Name,
				message:     fmt.Sprintf("Error restarting script %s: %v", proc.Name, err),
				isError:     true,
				clearLogs:   true,
			}
		}

		return restartProcessMsg{
			processName: proc.Name,
			message:     fmt.Sprintf("🔄 Restarted process: %s (logs cleared)", proc.Name),
			isError:     false,
			clearLogs:   true,
		}
	}
}

// HandleRestartAll creates a command to restart all processes
func (v *ProcessViewController) HandleRestartAll() tea.Cmd {
	return func() tea.Msg {
		processes := v.processMgr.GetAllProcesses()
		restarted := 0
		var errors []string

		for _, proc := range processes {
			if proc.GetStatus() == process.StatusRunning {
				// Stop the process
				if err := v.processMgr.StopProcess(proc.ID); err != nil {
					errors = append(errors, fmt.Sprintf("Error stopping process %s: %v", proc.Name, err))
					continue
				}

				// Start it again
				_, err := v.processMgr.StartScript(proc.Name)
				if err != nil {
					errors = append(errors, fmt.Sprintf("Error restarting script %s: %v", proc.Name, err))
				} else {
					restarted++
				}
			}
		}

		var message string
		var isError bool
		if len(errors) > 0 {
			message = fmt.Sprintf("🔄 Restarted %d processes with %d errors (logs cleared): %s", restarted, len(errors), strings.Join(errors, "; "))
			isError = true
		} else {
			message = fmt.Sprintf("🔄 Restarted %d processes (logs cleared)", restarted)
			isError = false
		}

		return restartAllMsg{
			message:   message,
			isError:   isError,
			clearLogs: true,
			restarted: restarted,
		}
	}
}
