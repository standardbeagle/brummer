package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
)

// ProcessViewController manages the processes view state and rendering
type ProcessViewController struct {
	processesList   list.Model
	selectedProcess string
	
	// Dependencies injected from parent Model
	processMgr *process.Manager
	width      int
	height     int
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