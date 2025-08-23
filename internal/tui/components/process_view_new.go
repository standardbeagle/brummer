// @lci:labels[component,process-view,ui,tui]
// @lci:category[ui-component]
// @lci:deps[process:manager:read,bubbletea:model:write]
// @lci:tags[component=process-view,pattern=component]
package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
)

// ProcessViewNew is a component for displaying and managing processes with proper width handling
type ProcessViewNew struct {
	width  int
	height int
	
	// Process management
	processMgr      *process.Manager
	processes       []*process.Process
	selectedIndex   int
	selectedProcess string
	
	// UI state
	focused         bool
	showEmptyState  bool
	lastUpdate      time.Time
	
	// Notification tracking
	errorCount      int
	warningCount    int
	newProcessCount int
	
	// Styles
	titleStyle       lipgloss.Style
	headerStyle      lipgloss.Style
	itemStyle        lipgloss.Style
	selectedStyle    lipgloss.Style
	runningStyle     lipgloss.Style
	stoppedStyle     lipgloss.Style
	errorStyle       lipgloss.Style
	emptyStateStyle  lipgloss.Style
	separatorStyle   lipgloss.Style
}

// processDisplayItem represents a single item in the process list
type processDisplayItem struct {
	process    *process.Process
	isHeader   bool
	headerText string
	isEmpty    bool // for separator rows
}

// NewProcessViewNew creates a new process view component
func NewProcessViewNew(processMgr *process.Manager) *ProcessViewNew {
	// ALWAYS initialize colors first
	MustInitColors()
	
	c := &ProcessViewNew{
		width:      80,
		height:     24,
		processMgr: processMgr,
		processes:  []*process.Process{},
	}
	
	c.initializeStyles()
	c.updateProcessList()
	
	return c
}

// initializeStyles sets up all the component styles
func (c *ProcessViewNew) initializeStyles() {
	c.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
	
	c.headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)
	
	c.itemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	c.selectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("236"))
	
	c.runningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))  // Green
	
	c.stoppedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")) // Gray
	
	c.errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))  // Red
	
	c.emptyStateStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Align(lipgloss.Center, lipgloss.Center)
	
	c.separatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
}

// Init implements Component interface
func (c *ProcessViewNew) Init() tea.Cmd {
	return c.tickCmd()
}

// Update implements tea.Model interface
func (c *ProcessViewNew) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		return c, nil
		
	case tea.KeyMsg:
		if c.focused {
			return c.handleKeyPress(msg)
		}
		
	case tickMsg:
		c.updateProcessList()
		return c, c.tickCmd()
		
	case processUpdateMsg:
		c.updateProcessList()
		return c, nil
	}
	
	return c, nil
}

// handleKeyPress handles keyboard input when focused
func (c *ProcessViewNew) handleKeyPress(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	processes := c.getSelectableProcesses()
	
	switch key.String() {
	case "up", "k":
		if c.selectedIndex > 0 {
			c.selectedIndex--
			if c.selectedIndex < len(processes) {
				c.selectedProcess = processes[c.selectedIndex].ID
			}
		}
		
	case "down", "j":
		if c.selectedIndex < len(processes)-1 {
			c.selectedIndex++
			if c.selectedIndex < len(processes) {
				c.selectedProcess = processes[c.selectedIndex].ID
			}
		}
		
	case "r":
		// Restart selected process
		if c.selectedProcess != "" {
			for _, p := range processes {
				if p.ID == c.selectedProcess {
					return c, c.restartProcess(p)
				}
			}
		}
		
	case "s":
		// Stop selected process
		if c.selectedProcess != "" {
			for _, p := range processes {
				if p.ID == c.selectedProcess {
					return c, c.stopProcess(p)
				}
			}
		}
		
	case "R":
		// Restart all processes
		return c, c.restartAllProcesses()
		
	case "enter":
		// Could switch to logs view for selected process
		// This would be handled by parent shell
		return c, nil
	}
	
	return c, nil
}

// View implements Component interface
func (c *ProcessViewNew) View() string {
	if c.showEmptyState || len(c.processes) == 0 {
		return c.renderEmptyState()
	}
	
	return c.renderProcessList()
}

// renderEmptyState shows when no processes are running
func (c *ProcessViewNew) renderEmptyState() string {
	emptyText := `No processes running.

Use / for commands:
  /run <script>    Start a script
  /restart all     Restart all
  /stop <process>  Stop a process

Available scripts:
  Use /list to see package.json scripts`
	
	// Use safe width calculation for centering
	availableWidth := CalculateAvailableWidth(c.width, false)
	availableHeight := c.height
	
	return c.emptyStateStyle.
		Width(availableWidth).
		Height(availableHeight).
		Render(emptyText)
}

// renderProcessList renders the list of processes
func (c *ProcessViewNew) renderProcessList() string {
	var lines []string
	
	// Group processes
	running, stopped := c.groupProcesses()
	
	// Render running processes
	if len(running) > 0 {
		lines = append(lines, c.headerStyle.Render("🏃 Running Processes"))
		for _, proc := range running {
			selected := proc.ID == c.selectedProcess
			line := c.renderProcessItem(proc, selected)
			lines = append(lines, line)
		}
	}
	
	// Add separator if both sections exist
	if len(running) > 0 && len(stopped) > 0 {
		lines = append(lines, "")
	}
	
	// Render stopped processes
	if len(stopped) > 0 {
		lines = append(lines, c.headerStyle.Render("⏹️  Stopped Processes"))
		for _, proc := range stopped {
			selected := proc.ID == c.selectedProcess
			line := c.renderProcessItem(proc, selected)
			lines = append(lines, line)
		}
	}
	
	// Add stats footer
	stats := c.renderStats()
	if stats != "" {
		lines = append(lines, "", c.separatorStyle.Render(strings.Repeat("─", 40)), stats)
	}
	
	return strings.Join(lines, "\n")
}

// renderProcessItem renders a single process item
func (c *ProcessViewNew) renderProcessItem(proc *process.Process, selected bool) string {
	// Build status indicator with emoji
	statusEmoji := c.getStatusEmoji(proc)
	
	// Calculate widths properly
	name := proc.Name
	maxNameWidth := 20
	if SafeStringWidth(name) > maxNameWidth {
		name = SafeTruncateWithEllipsis(name, maxNameWidth)
	}
	
	// Format uptime
	uptime := c.formatUptime(proc)
	
	// Build the line with proper spacing
	// Format: "  🟢 name                 3m 42s  PID: 12345"
	indicator := "  "
	if selected {
		indicator = "▶ "
	}
	
	// Use safe padding to align columns
	nameField := SafePadRight(name, maxNameWidth)
	uptimeField := SafePadLeft(uptime, 10)
	pidField := fmt.Sprintf("PID: %s", proc.ID)
	
	// Combine with proper spacing
	line := fmt.Sprintf("%s%s %s %s  %s", 
		indicator, 
		statusEmoji,
		nameField,
		uptimeField,
		pidField)
	
	// Apply style based on selection and status
	if selected {
		return c.selectedStyle.Render(line)
	}
	
	switch proc.GetStatus() {
	case process.StatusRunning:
		return c.runningStyle.Render(line)
	case process.StatusFailed:
		return c.errorStyle.Render(line)
	default:
		return c.stoppedStyle.Render(line)
	}
}

// getStatusEmoji returns the appropriate emoji for process status
func (c *ProcessViewNew) getStatusEmoji(proc *process.Process) string {
	switch proc.GetStatus() {
	case process.StatusRunning:
		return "🟢" // Green for healthy
	case process.StatusFailed:
		return "🔴" // Red for failed
	case process.StatusStopped:
		return "⚫" // Black for stopped
	case process.StatusSuccess:
		return "✅" // Check for success
	case process.StatusPending:
		return "🔵" // Blue for pending
	default:
		return "⚪" // White for unknown
	}
}

// formatUptime formats the process uptime
func (c *ProcessViewNew) formatUptime(proc *process.Process) string {
	if proc.GetStatus() != process.StatusRunning {
		return "-"
	}
	
	uptime := time.Since(proc.StartTime)
	
	if uptime < time.Minute {
		return fmt.Sprintf("%ds", int(uptime.Seconds()))
	} else if uptime < time.Hour {
		mins := int(uptime.Minutes())
		secs := int(uptime.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	} else {
		hours := int(uptime.Hours())
		mins := int(uptime.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
}

// renderStats renders process statistics
func (c *ProcessViewNew) renderStats() string {
	running := 0
	stopped := 0
	failed := 0
	
	for _, proc := range c.processes {
		switch proc.GetStatus() {
		case process.StatusRunning:
			running++
		case process.StatusFailed:
			failed++
		case process.StatusStopped, process.StatusSuccess:
			stopped++
		}
	}
	
	stats := fmt.Sprintf("Total: %d | Running: %d | Stopped: %d",
		len(c.processes), running, stopped)
		
	if failed > 0 {
		stats += fmt.Sprintf(" | ⚠️ Failed: %d", failed)
	}
	
	return stats
}

// groupProcesses groups processes by status
func (c *ProcessViewNew) groupProcesses() ([]*process.Process, []*process.Process) {
	var running, stopped []*process.Process
	
	for _, proc := range c.processes {
		if proc.GetStatus() == process.StatusRunning {
			running = append(running, proc)
		} else {
			stopped = append(stopped, proc)
		}
	}
	
	return running, stopped
}

// getSelectableProcesses returns only processes that can be selected (not headers)
func (c *ProcessViewNew) getSelectableProcesses() []*process.Process {
	return c.processes
}

// updateProcessList refreshes the process list from the manager
func (c *ProcessViewNew) updateProcessList() {
	if c.processMgr != nil {
		c.processes = c.processMgr.GetAllProcesses()
		c.showEmptyState = len(c.processes) == 0
		c.lastUpdate = time.Now()
		
		// Update notification counts
		c.errorCount = 0
		c.warningCount = 0
		for _, proc := range c.processes {
			if proc.GetStatus() == process.StatusFailed {
				c.errorCount++
			}
		}
	}
}

// Focus sets the component focus state
func (c *ProcessViewNew) Focus() tea.Cmd {
	c.focused = true
	return nil
}

// Blur removes component focus
func (c *ProcessViewNew) Blur() tea.Cmd {
	c.focused = false
	return nil
}

// Focused returns the focus state (implements Component interface)
func (c *ProcessViewNew) Focused() bool {
	return c.focused
}

// GetNotificationCount returns notification counts for this view
func (c *ProcessViewNew) GetNotificationCount() (errors, warnings, info int) {
	return c.errorCount, c.warningCount, c.newProcessCount
}

// ClearNotifications clears the notification counts
func (c *ProcessViewNew) ClearNotifications() {
	c.newProcessCount = 0
}

// Process management commands

func (c *ProcessViewNew) restartProcess(proc *process.Process) tea.Cmd {
	return func() tea.Msg {
		// Stop if running
		if proc.GetStatus() == process.StatusRunning {
			if err := c.processMgr.StopProcess(proc.ID); err != nil {
				return errorMsg{err: err}
			}
			// Wait a bit for clean shutdown
			time.Sleep(500 * time.Millisecond)
		}
		
		// Start again
		_, err := c.processMgr.StartScript(proc.Name)
		if err != nil {
			return errorMsg{err: err}
		}
		
		return processUpdateMsg{}
	}
}

func (c *ProcessViewNew) stopProcess(proc *process.Process) tea.Cmd {
	return func() tea.Msg {
		if err := c.processMgr.StopProcess(proc.ID); err != nil {
			return errorMsg{err: err}
		}
		return processUpdateMsg{}
	}
}

func (c *ProcessViewNew) restartAllProcesses() tea.Cmd {
	return func() tea.Msg {
		processes := c.processMgr.GetAllProcesses()
		
		for _, proc := range processes {
			if proc.GetStatus() == process.StatusRunning {
				// Stop process
				if err := c.processMgr.StopProcess(proc.ID); err != nil {
					continue
				}
				
				// Restart it
				time.Sleep(200 * time.Millisecond)
				c.processMgr.StartScript(proc.Name)
			}
		}
		
		return processUpdateMsg{}
	}
}

// tickCmd creates a command for periodic updates
func (c *ProcessViewNew) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// SetSize sets the component size
func (c *ProcessViewNew) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Message types
type tickMsg time.Time
type processUpdateMsg struct{}
type errorMsg struct {
	err error
}