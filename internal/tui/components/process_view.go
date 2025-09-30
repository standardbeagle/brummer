package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// ProcessViewComponent displays running processes in a table format.
// It receives ProcessStartedMsg and ProcessStoppedMsg messages to update the display.
//
// Design principles:
// - No direct access to Manager (read-only initial load is acceptable)
// - All updates via ProcessStartedMsg/ProcessStoppedMsg messages
// - Table view with columns: Name, Status, Runtime, PID
// - Supports selection and scrolling
type ProcessViewComponent struct {
	BaseComponent

	viewport  viewport.Model
	processes map[string]*ProcessInfo // processID -> ProcessInfo

	// Read-only reference for initial load
	processMgr *process.Manager

	// Styles
	headerStyle  lipgloss.Style
	runningStyle lipgloss.Style
	stoppedStyle lipgloss.Style
	failedStyle  lipgloss.Style
	normalStyle  lipgloss.Style
}

// ProcessInfo holds display information for a process
type ProcessInfo struct {
	ID        string
	Name      string
	Status    process.ProcessStatus
	StartTime time.Time
	EndTime   *time.Time
	ExitCode  *int
}

// NewProcessViewComponent creates a process view with initial processes from manager
func NewProcessViewComponent(processMgr *process.Manager) *ProcessViewComponent {
	c := &ProcessViewComponent{
		BaseComponent: NewBaseComponent(),
		processMgr:    processMgr,
		viewport:      viewport.New(80, 24),
		processes:     make(map[string]*ProcessInfo),
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")), // Bright Blue
		runningStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")),            // Bright Green
		stoppedStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("8")),             // Gray
		failedStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")),             // Bright Red
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),            // White
	}

	// Load existing processes (one-time read)
	if processMgr != nil {
		allProcesses := processMgr.GetAllProcesses()
		for _, proc := range allProcesses {
			snapshot := proc.GetSnapshot()
			c.processes[snapshot.ID] = &ProcessInfo{
				ID:        snapshot.ID,
				Name:      snapshot.Name,
				Status:    snapshot.Status,
				StartTime: snapshot.StartTime,
				EndTime:   snapshot.EndTime,
				ExitCode:  snapshot.ExitCode,
			}
		}
		c.updateViewport()
	}

	return c
}

// Init initializes the component
func (c *ProcessViewComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *ProcessViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.ProcessStartedMsg:
		// NEW PROCESS! Event-driven update
		c.processes[msg.ProcessID] = &ProcessInfo{
			ID:        msg.ProcessID,
			Name:      msg.Name,
			Status:    process.StatusRunning,
			StartTime: msg.StartTime,
			EndTime:   nil,
			ExitCode:  nil,
		}
		c.updateViewport()
		return c, nil

	case messages.ProcessStoppedMsg:
		// PROCESS STOPPED! Update existing process
		if info, exists := c.processes[msg.ProcessID]; exists {
			info.Status = process.StatusStopped
			info.EndTime = &msg.EndTime
			info.ExitCode = &msg.ExitCode
			c.updateViewport()
		}
		return c, nil

	case messages.ProcessStateChangedMsg:
		// PROCESS STATE CHANGED! Update status
		if info, exists := c.processes[msg.ProcessID]; exists {
			info.Status = msg.NewState
			c.updateViewport()
		}
		return c, nil

	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		c.viewport.Width = msg.Width
		c.viewport.Height = msg.Height
		c.updateViewport()
		return c, nil

	case tea.KeyMsg:
		// Forward keyboard input to viewport for scrolling
		c.viewport, cmd = c.viewport.Update(msg)
		return c, cmd
	}

	return c, nil
}

// View renders the process table
func (c *ProcessViewComponent) View() string {
	if !c.IsVisible() {
		return ""
	}

	return c.viewport.View()
}

// SetSize updates component dimensions
func (c *ProcessViewComponent) SetSize(width, height int) {
	c.BaseComponent.SetSize(width, height)
	c.viewport.Width = width
	c.viewport.Height = height
	c.updateViewport()
}

// updateViewport updates the viewport content with formatted process table
func (c *ProcessViewComponent) updateViewport() {
	var lines []string

	// Header
	header := c.headerStyle.Render(fmt.Sprintf("%-20s %-12s %-15s %s",
		"PROCESS", "STATUS", "RUNTIME", "EXIT CODE"))
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", c.viewport.Width))

	// Process rows
	if len(c.processes) == 0 {
		lines = append(lines, c.normalStyle.Render("No processes running"))
	} else {
		for _, info := range c.processes {
			lines = append(lines, c.formatProcessRow(info))
		}
	}

	content := strings.Join(lines, "\n")
	c.viewport.SetContent(content)
}

// formatProcessRow formats a single process row
func (c *ProcessViewComponent) formatProcessRow(info *ProcessInfo) string {
	// Truncate long process names
	name := info.Name
	if len(name) > 20 {
		name = name[:17] + "..."
	}

	// Format status with color
	status := string(info.Status)
	var styledStatus string
	switch info.Status {
	case process.StatusRunning:
		styledStatus = c.runningStyle.Render(fmt.Sprintf("%-12s", status))
	case process.StatusStopped:
		styledStatus = c.stoppedStyle.Render(fmt.Sprintf("%-12s", status))
	case process.StatusFailed:
		styledStatus = c.failedStyle.Render(fmt.Sprintf("%-12s", status))
	default:
		styledStatus = c.normalStyle.Render(fmt.Sprintf("%-12s", status))
	}

	// Calculate runtime
	var runtime string
	if info.EndTime != nil {
		duration := info.EndTime.Sub(info.StartTime)
		runtime = formatDuration(duration)
	} else {
		duration := time.Since(info.StartTime)
		runtime = formatDuration(duration)
	}

	// Format exit code
	exitCode := "-"
	if info.ExitCode != nil {
		exitCode = fmt.Sprintf("%d", *info.ExitCode)
		if *info.ExitCode != 0 {
			exitCode = c.failedStyle.Render(exitCode)
		}
	}

	return fmt.Sprintf("%-20s %s %-15s %s",
		name,
		styledStatus,
		runtime,
		exitCode,
	)
}

// formatDuration formats a duration in human-readable form
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}