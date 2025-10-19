package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// ProcessesViewComponent is a stateless BubbleTea component for displaying processes
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type ProcessesViewComponent struct {
	BaseComponent

	// UI configuration only (no state)
	width  int
	height int

	// List model for display (minimal, stateless)
	list   list.Model

	// Styles
	headerStyle      lipgloss.Style
	runningStyle     lipgloss.Style
	stoppedStyle     lipgloss.Style
	emptyStyle       lipgloss.Style
	titleStyle       lipgloss.Style
	selectedStyle    lipgloss.Style
	descriptionStyle lipgloss.Style
}

// NewProcessesViewComponent creates a new stateless processes view component
func NewProcessesViewComponent() *ProcessesViewComponent {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("12"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("12"))

	processList := list.New([]list.Item{}, delegate, 0, 0)
	processList.Title = "Running Processes"
	processList.SetShowStatusBar(false)
	processList.SetShowHelp(false)
	processList.SetFilteringEnabled(false)
	processList.SetShowPagination(false)

	return &ProcessesViewComponent{
		BaseComponent: NewBaseComponent(),
		width:         80,
		height:        24,
		list:          processList,

		// Initialize styles
		headerStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244")),
		runningStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		stoppedStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		emptyStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Align(lipgloss.Center, lipgloss.Center),
		titleStyle:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		selectedStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		descriptionStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	}
}

// Init initializes the component
func (c *ProcessesViewComponent) Init() tea.Cmd {
	return nil
}

// ViewWithState renders the component using external state (stateless/Elm-compliant)
func (c *ProcessesViewComponent) ViewWithState(masterData *state.MasterData, processMgr *processes.Manager) string {
	if !c.IsVisible() || masterData == nil || masterData.ProcessesState == nil {
		return ""
	}

	processesState := masterData.ProcessesState

	// Get processes from cache or process manager
	var processes []*processes.Process
	if processesState.IsCacheValid(2*time.Second) {
		processes = processesState.GetProcesses()
	} else if processMgr != nil {
		processes = processMgr.GetAllProcesses()
	}

	if len(processes) == 0 {
		return c.renderEmptyState()
	}

	// Create list items
	items := c.createListItems(processes, processesState)

	// Update list
	c.list.SetItems(items)
	c.list.SetSize(c.width, c.height-2) // Reserve space for title

	return c.list.View()
}

// renderEmptyState renders the empty state when no processes are running
func (c *ProcessesViewComponent) renderEmptyState() string {
	content := []string{
		"No processes running.",
		"",
		"Use / for commands:",
		"/run <script> to start scripts",
		"/restart all",
		"/stop <process>",
	}

	emptyText := strings.Join(content, "\n")
	return c.emptyStyle.Width(c.width).Height(c.height).Render(emptyText)
}

// createListItems creates list items from processes
func (c *ProcessesViewComponent) createListItems(processList []*processes.Process, processesState *state.ProcessesState) []list.Item {
	var items []list.Item

	// Group processes by status
	var running, stopped []*processes.Process
	for _, proc := range processList {
		if proc.GetStatus() == processes.StatusRunning {
			running = append(running, proc)
		} else {
			stopped = append(stopped, proc)
		}
	}

	// Sort if needed
	if processesState.GetSortByUptime() {
		sort.Slice(running, func(i, j int) bool {
			return running[i].StartTime.After(running[j].StartTime)
		})
		sort.Slice(stopped, func(i, j int) bool {
			return stopped[i].StartTime.After(stopped[j].StartTime)
		})
	}

	// Add running processes
	if len(running) > 0 {
		if processesState.GetGroupByStatus() {
			items = append(items, ProcessItemHeader{
				HeaderText: "Running Processes",
				IsHeader:   true,
			})
		}

		for _, proc := range running {
			items = append(items, ProcessItemStateless{
				Process:     proc,
				IsSelected:  proc.ID == processesState.GetSelectedProcess(),
				ShowUptime:  true,
				ShowPID:     true,
			})
		}
	}

	// Add stopped processes
	if len(stopped) > 0 && processesState.GetShowStopped() {
		if processesState.GetGroupByStatus() && len(running) > 0 {
			items = append(items, ProcessItemHeader{
				HeaderText: "",
				IsHeader:   true,
			})
		}

		if processesState.GetGroupByStatus() {
			items = append(items, ProcessItemHeader{
				HeaderText: "Stopped Processes",
				IsHeader:   true,
			})
		}

		for _, proc := range stopped {
			items = append(items, ProcessItemStateless{
				Process:     proc,
				IsSelected:  proc.ID == processesState.GetSelectedProcess(),
				ShowUptime:  true,
				ShowPID:     false,
			})
		}
	}

	return items
}

// SetSize updates component dimensions
func (c *ProcessesViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.width = width
	c.height = height
	c.list.SetSize(width, height-2)
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *ProcessesViewComponent) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.width = width
	c.height = height
	c.list.SetSize(width, contentHeight)
}

// GetProcessByIndex returns process at given index (for selection handling)
func (c *ProcessesViewComponent) GetProcessByIndex(index int, processes []*processes.Process) *processes.Process {
	if index < 0 || index >= len(processes) {
		return nil
	}
	return processes[index]
}

// FormatProcessDescription formats a process description
func (c *ProcessesViewComponent) FormatProcessDescription(proc *processes.Process) string {
	uptime := time.Since(proc.StartTime).Round(time.Second)
	statusStr := string(proc.GetStatus())

	if proc.Cmd != nil && proc.Cmd.Process != nil {
		pid := proc.Cmd.Process.Pid
		return fmt.Sprintf("PID: %d | Uptime: %s | Status: %s", pid, uptime, statusStr)
	}
	return fmt.Sprintf("Uptime: %s | Status: %s", uptime, statusStr)
}

// FormatProcessTitle formats a process title
func (c *ProcessesViewComponent) FormatProcessTitle(proc *processes.Process) string {
	status := "🟢"
	if proc.GetStatus() != processes.StatusRunning {
		status = "🔴"
	}

	return fmt.Sprintf("%s %s", status, proc.Name)
}

// GetProcessStats returns process statistics
func (c *ProcessesViewComponent) GetProcessStats(processList []*processes.Process) (total, running, stopped int) {
	total = len(processList)
	for _, proc := range processList {
		if proc.GetStatus() == processes.StatusRunning {
			running++
		} else {
			stopped++
		}
	}
	return
}

// ProcessItemStateless represents a process in the list (stateless version)
type ProcessItemStateless struct {
	Process     *processes.Process
	IsSelected  bool
	ShowUptime  bool
	ShowPID     bool
}

func (i ProcessItemStateless) Title() string {
	if i.Process == nil {
		return ""
	}
	return fmt.Sprintf("%s %s", i.getStatusIcon(), i.Process.Name)
}

func (i ProcessItemStateless) Description() string {
	if i.Process == nil {
		return ""
	}

	var parts []string

	if i.ShowPID && i.Process.Cmd != nil && i.Process.Cmd.Process != nil {
		parts = append(parts, fmt.Sprintf("PID: %d", i.Process.Cmd.Process.Pid))
	}

	if i.ShowUptime {
		uptime := time.Since(i.Process.StartTime).Round(time.Second)
		parts = append(parts, fmt.Sprintf("Uptime: %s", uptime))
	}

	parts = append(parts, fmt.Sprintf("Status: %s", string(i.Process.GetStatus())))

	return strings.Join(parts, " | ")
}

func (i ProcessItemStateless) FilterValue() string {
	if i.Process == nil {
		return ""
	}
	return i.Process.Name
}

func (i ProcessItemStateless) getStatusIcon() string {
	if i.Process.GetStatus() == processes.StatusRunning {
		return "🟢"
	}
	return "🔴"
}

// ProcessItemHeader represents a header in the process list
type ProcessItemHeader struct {
	HeaderText string
	IsHeader   bool
}

func (i ProcessItemHeader) Title() string {
	return i.HeaderText
}

func (i ProcessItemHeader) Description() string {
	return ""
}

func (i ProcessItemHeader) FilterValue() string {
	return ""
}