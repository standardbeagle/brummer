package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
)

// ProcessesView is a BubbleTea component for displaying running processes
// Migrated from process_view_controller.go (T038-T042)
// Architecture: Component → Service (direct process.Manager access)
type ProcessesView struct {
	processesList   list.Model
	selectedProcess string

	// Size tracking
	width         int
	height        int
	contentHeight int

	// Direct service access (no controller indirection)
	processMgr *process.Manager
}

// NewProcessesView creates a new processes view component
func NewProcessesView(processMgr *process.Manager) *ProcessesView {
	processesList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	processesList.Title = "Running Processes"
	processesList.SetShowStatusBar(false)

	return &ProcessesView{
		processesList: processesList,
		processMgr:    processMgr,
	}
}

// Init initializes the component (BubbleTea interface)
func (v *ProcessesView) Init() tea.Cmd {
	return nil
}

// Update handles messages (BubbleTea interface)
func (v *ProcessesView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	v.processesList, cmd = v.processesList.Update(msg)
	return v, cmd
}

// View renders the component (BubbleTea interface)
func (v *ProcessesView) View() string {
	return v.Render()
}

// SetSize updates the component's dimensions (Component interface)
func (v *ProcessesView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.contentHeight = height - 6 // header + footer
	v.processesList.SetSize(width, v.contentHeight)
}

// Focus gives keyboard focus to the component (Component interface)
func (v *ProcessesView) Focus() tea.Cmd {
	return nil
}

// Blur removes keyboard focus from the component (Component interface)
func (v *ProcessesView) Blur() tea.Cmd {
	return nil
}

// GetSize returns current dimensions (Component interface)
func (v *ProcessesView) GetSize() (width, height int) {
	return v.width, v.height
}

// SetSelectedProcess sets the currently selected process
func (v *ProcessesView) SetSelectedProcess(processID string) {
	v.selectedProcess = processID
}

// GetSelectedProcess returns the currently selected process
func (v *ProcessesView) GetSelectedProcess() string {
	return v.selectedProcess
}

// GetProcessesList returns the processes list for direct manipulation
func (v *ProcessesView) GetProcessesList() *list.Model {
	return &v.processesList
}

// UpdateSize updates the dimensions with pre-calculated content height
func (v *ProcessesView) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	v.width = width
	v.height = height
	v.contentHeight = contentHeight
	v.processesList.SetSize(width, contentHeight)
}

// UpdateProcessList refreshes the process list with current data
func (v *ProcessesView) UpdateProcessList() {
	// Direct service call - no controller indirection (T040)
	processes := v.processMgr.GetAllProcesses()

	// Convert processes to list items
	var items []list.Item

	if len(processes) == 0 {
		v.processesList.SetItems([]list.Item{})
		return
	}

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
		items = append(items, ProcessItem{
			HeaderText: "Running Processes",
			IsHeader:   true,
		})
		for _, proc := range running {
			items = append(items, ProcessItem{
				Process: proc,
			})
		}
	}

	// Add stopped processes
	if len(stopped) > 0 {
		if len(running) > 0 {
			// Add separator
			items = append(items, ProcessItem{
				HeaderText: "",
				IsHeader:   false,
			})
		}
		items = append(items, ProcessItem{
			HeaderText: "Stopped Processes",
			IsHeader:   true,
		})
		for _, proc := range stopped {
			items = append(items, ProcessItem{
				Process: proc,
			})
		}
	}

	v.processesList.SetItems(items)
}

// Render renders the processes view
func (v *ProcessesView) Render() string {
	// Direct service call - no controller indirection (T040)
	processes := v.processMgr.GetAllProcesses()
	if len(processes) == 0 {
		emptyState := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Align(lipgloss.Center, lipgloss.Center).
			Width(v.width).
			Height(v.contentHeight).
			Render("No processes running.\n\nUse / for commands:\n/run <script> to start scripts\n/restart all\n/stop <process>")
		return emptyState
	}

	return v.processesList.View()
}

func (v *ProcessesView) HandleRestartProcess(proc *process.Process) tea.Cmd {
	return func() tea.Msg {
		var message string
		var isError bool

		// Check if process is still running before trying to stop it
		if proc.GetStatus() == process.StatusRunning {
			// Try to stop process with timeout using command pattern
			err := v.processMgr.StopProcess(proc.ID)
			if err != nil {
				// Process didn't stop cleanly, try stop with wait timeout
				if waitErr := v.processMgr.StopProcessAndWait(proc.ID, 100*time.Millisecond); waitErr != nil {
					message = fmt.Sprintf("Error stopping process %s: %v (wait error: %v)", proc.Name, err, waitErr)
					isError = true
				}
			}
		}

		// Clean up any finished processes before starting new one
		v.processMgr.CleanupFinishedProcesses()

		// Also clean up any processes that might be using development ports
		if proc.Name == "server" || proc.Name == "dev" || proc.Name == "start" {
			v.processMgr.KillProcessesByPort()
			// No sleep here - continue immediately
		}

		// Now start it again
		_, err := v.processMgr.StartScript(proc.Name)
		if err != nil {
			message = fmt.Sprintf("Error restarting script %s: %v", proc.Name, err)
			isError = true
		} else {
			message = fmt.Sprintf("🔄 Restarted process: %s (logs cleared)", proc.Name)
		}

		return RestartProcessMsg{
			ProcessName: proc.Name,
			Message:     message,
			IsError:     isError,
			ClearLogs:   true,
			TimeoutUsed: false,
		}
	}
}

// HandleRestartAll creates a command to restart all processes
func (v *ProcessesView) HandleRestartAll() tea.Cmd {
	return func() tea.Msg {
		processes := v.processMgr.GetAllProcesses()
		restarted := 0
		var errors []string

		for _, proc := range processes {
			if proc.GetStatus() == process.StatusRunning {
				if err := v.processMgr.StopProcess(proc.ID); err != nil {
					errors = append(errors, fmt.Sprintf("Error stopping process %s: %v", proc.Name, err))
					continue
				}

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

		return RestartAllMsg{
			Message:   message,
			IsError:   isError,
			ClearLogs: true,
			Restarted: restarted,
		}
	}
}

// Message types for process operations (exported from controller)
type RestartProcessMsg struct {
	ProcessName string
	Message     string
	IsError     bool
	ClearLogs   bool
	TimeoutUsed bool
}

type RestartAllMsg struct {
	Message   string
	IsError   bool
	ClearLogs bool
	Restarted int
}

// ProcessItem represents an item in the process list
type ProcessItem struct {
	Process    *process.Process
	HeaderText string
	IsHeader   bool
}

func (i ProcessItem) Title() string {
	if i.IsHeader {
		return i.HeaderText
	}
	if i.Process == nil {
		return ""
	}

	status := "🟢"
	if i.Process.GetStatus() != process.StatusRunning {
		status = "🔴"
	}

	return fmt.Sprintf("%s %s", status, i.Process.Name)
}

func (i ProcessItem) Description() string {
	if i.IsHeader || i.Process == nil {
		return ""
	}

	uptime := time.Since(i.Process.StartTime).Round(time.Second)
	statusStr := string(i.Process.GetStatus())

	if i.Process.Cmd != nil && i.Process.Cmd.Process != nil {
		pid := i.Process.Cmd.Process.Pid
		return fmt.Sprintf("PID: %d | Uptime: %s | Status: %s", pid, uptime, statusStr)
	}
	return fmt.Sprintf("Uptime: %s | Status: %s", uptime, statusStr)
}

func (i ProcessItem) FilterValue() string {
	if i.IsHeader || i.Process == nil {
		return ""
	}
	return i.Process.Name
}
