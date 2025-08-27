package components

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/process"
)

// ProcessViewComponent adapts the process view controller to the component system
type ProcessViewComponent struct {
	BaseComponent

	// No controller delegation needed - we implement directly

	// Component-specific state
	processes []*process.Process
	selected  int
	list      list.Model
	width     int
	height    int

	// Process manager reference
	processMgr *process.Manager

	// Styles
	titleStyle    lipgloss.Style
	itemStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

// NewProcessViewComponent creates a new process view component
func NewProcessViewComponent(processMgr *process.Manager) *ProcessViewComponent {
	p := &ProcessViewComponent{
		processMgr: processMgr,
		processes:  []*process.Process{},
		selected:   0,
		width:      80,
		height:     24,
	}

	// Initialize styles
	p.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	p.itemStyle = lipgloss.NewStyle().
		PaddingLeft(2)

	p.selectedStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	// Initialize list
	p.initializeList()

	// Set visible by default
	p.visible = true

	return p
}

// initializeList sets up the bubbles list component
func (p *ProcessViewComponent) initializeList() {
	// Convert processes to list items
	items := p.processesToListItems()

	// Create list
	p.list = list.New(items, processItemDelegate{}, p.width, p.height-4)
	p.list.Title = "Processes"
	p.list.SetShowStatusBar(true)
	p.list.SetFilteringEnabled(true)
	p.list.SetShowHelp(true)

	// Customize key bindings
	p.list.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	)
	p.list.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	)
}

// Update handles messages
func (p *ProcessViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle component-specific messages
	switch msg := msg.(type) {
	case ProcessListUpdateMsg:
		// Update process list
		p.processes = msg.Processes
		p.updateList()
		return p, nil

	case ProcessUpdateMsg:
		// Handle process update
		p.handleProcessUpdate(msg)
		p.updateList()
		return p, nil

	case ProcessSelectMsg:
		// Handle process selection
		for i, proc := range p.processes {
			if proc.ID == msg.ProcessID {
				p.selected = i
				p.list.Select(i)
				break
			}
		}
		return p, nil

	case ProcessStartMsg:
		// Start a process
		if p.processMgr != nil {
			_, err := p.processMgr.StartScript(msg.ScriptName)
			if err != nil {
				return p, SendSystemNotification(NotificationError,
					"Failed to start process", err.Error())
			}
			return p, SendSystemNotification(NotificationSuccess,
				"Process started", msg.ScriptName)
		}

	case ProcessStopMsg:
		// Stop a process
		if p.processMgr != nil {
			err := p.processMgr.StopProcess(msg.ProcessID)
			if err != nil {
				return p, SendSystemNotification(NotificationError,
					"Failed to stop process", err.Error())
			}
			return p, SendSystemNotification(NotificationSuccess,
				"Process stopped", msg.ProcessID)
		}

	case ProcessRestartMsg:
		// Restart a process
		if p.processMgr != nil {
			// First stop
			_ = p.processMgr.StopProcess(msg.ProcessID)
			// Find and restart
			for _, proc := range p.processes {
				if proc.ID == msg.ProcessID {
					_, err := p.processMgr.StartScript(proc.Name)
					if err != nil {
						return p, SendSystemNotification(NotificationError,
							"Failed to restart process", err.Error())
					}
					return p, SendSystemNotification(NotificationSuccess,
						"Process restarted", proc.Name)
				}
			}
		}

	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.list.SetSize(p.width, p.height-4)

	case tea.KeyMsg:
		// Handle key presses
		switch msg.String() {
		case "enter", " ":
			// Toggle process (start/stop)
			if item := p.list.SelectedItem(); item != nil {
				if pi, ok := item.(processItem); ok {
					if pi.process.Status == process.StatusRunning {
						return p, SendProcessStop(pi.process.ID, false)
					} else {
						return p, SendProcessStart(pi.process.Name)
					}
				}
			}

		case "s":
			// Stop process
			if item := p.list.SelectedItem(); item != nil {
				if pi, ok := item.(processItem); ok {
					return p, SendProcessStop(pi.process.ID, false)
				}
			}

		case "r":
			// Restart process
			if item := p.list.SelectedItem(); item != nil {
				if pi, ok := item.(processItem); ok {
					return p, SendProcessRestart(pi.process.ID)
				}
			}

		case "S":
			// Stop all processes
			var stopCmds []tea.Cmd
			for _, proc := range p.processes {
				if proc.Status == process.StatusRunning {
					stopCmds = append(stopCmds, SendProcessStop(proc.ID, false))
				}
			}
			return p, tea.Batch(stopCmds...)

		case "R":
			// Restart all processes
			var restartCmds []tea.Cmd
			for _, proc := range p.processes {
				restartCmds = append(restartCmds, SendProcessRestart(proc.ID))
			}
			return p, tea.Batch(restartCmds...)

		case "l":
			// View logs for selected process
			if item := p.list.SelectedItem(); item != nil {
				if pi, ok := item.(processItem); ok {
					// Switch to logs view with filter
					cmds = append(cmds, SendViewChange("logs", pi.process.ID))
					cmds = append(cmds, SendLogsFilter("", pi.process.ID, ""))
				}
			}
		}
	}

	// Update the list component
	if p.visible {
		newList, cmd := p.list.Update(msg)
		p.list = newList
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return p, tea.Batch(cmds...)
}

// View renders the component
func (p *ProcessViewComponent) View() string {
	if !p.visible {
		return ""
	}

	// Use the list component for rendering
	return p.list.View()
}

// Focus sets focus to this component
func (p *ProcessViewComponent) Focus() tea.Cmd {
	p.focused = true
	return nil
}

// Blur removes focus from this component
func (p *ProcessViewComponent) Blur() tea.Cmd {
	p.focused = false
	return nil
}

// handleProcessUpdate updates the process list based on update type
func (p *ProcessViewComponent) handleProcessUpdate(msg ProcessUpdateMsg) {
	switch msg.UpdateType {
	case ProcessStarted:
		// Add or update process
		found := false
		for i, proc := range p.processes {
			if proc.ID == msg.Process.ID {
				p.processes[i] = msg.Process
				found = true
				break
			}
		}
		if !found {
			p.processes = append(p.processes, msg.Process)
		}

	case ProcessStopped:
		// Update process state
		for i, proc := range p.processes {
			if proc.ID == msg.Process.ID {
				p.processes[i] = msg.Process
				break
			}
		}

	case ProcessStateChange:
		// Update process state
		for i, proc := range p.processes {
			if proc.ID == msg.Process.ID {
				p.processes[i] = msg.Process
				break
			}
		}
	}
}

// updateList updates the list items
func (p *ProcessViewComponent) updateList() {
	items := p.processesToListItems()
	p.list.SetItems(items)
}

// processesToListItems converts processes to list items
func (p *ProcessViewComponent) processesToListItems() []list.Item {
	items := make([]list.Item, len(p.processes))
	for i, proc := range p.processes {
		items[i] = processItem{
			process: proc,
			index:   i,
		}
	}
	return items
}

// processItem implements list.Item
type processItem struct {
	process *process.Process
	index   int
}

func (i processItem) Title() string {
	status := "●"
	switch i.process.Status {
	case process.StatusRunning:
		status = "🟢"
	case process.StatusStopped:
		status = "⭕"
	case process.StatusFailed:
		status = "🔴"
	case process.StatusSuccess:
		status = "✅"
	}
	return fmt.Sprintf("%s %s", status, i.process.Name)
}

func (i processItem) Description() string {
	var parts []string

	// Add PID if running (would need to get from Cmd.Process)
	if i.process.Cmd != nil && i.process.Cmd.Process != nil {
		parts = append(parts, fmt.Sprintf("PID: %d", i.process.Cmd.Process.Pid))
	}

	// Add status
	parts = append(parts, string(i.process.Status))

	// Add exit code if stopped
	if i.process.Status == process.StatusStopped && i.process.ExitCode != nil && *i.process.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("Exit: %d", *i.process.ExitCode))
	}

	return strings.Join(parts, " | ")
}

func (i processItem) FilterValue() string {
	return i.process.Name
}

// processItemDelegate implements list.ItemDelegate
type processItemDelegate struct{}

func (d processItemDelegate) Height() int { return 2 }

func (d processItemDelegate) Spacing() int { return 1 }

func (d processItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d processItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(processItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s\n  %s", i.Title(), i.Description())

	// Create styles inline since we don't have access to component's styles
	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("170")).
		Bold(true)

	var fn func(...string) string
	if index == m.Index() {
		fn = func(strs ...string) string {
			str := strings.Join(strs, "")
			return selectedStyle.Render("> " + str)
		}
	} else {
		fn = itemStyle.Render
	}

	fmt.Fprintf(w, "%s", fn(str))
}

// ProcessRestartMsg helper
func SendProcessRestart(processID string) tea.Cmd {
	return func() tea.Msg {
		return ProcessRestartMsg{
			ProcessID: processID,
		}
	}
}
