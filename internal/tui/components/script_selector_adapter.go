package components

import (
	"fmt"
	"io"
	"sort"
	"strings"
	
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/parser"
	"github.com/standardbeagle/brummer/internal/process"
)

// ScriptSelectorComponent provides script selection and management
type ScriptSelectorComponent struct {
	BaseComponent
	
	// Component state
	scripts        []ScriptInfo
	list           list.Model
	filterInput    textinput.Model
	showModal      bool
	selectedScript *ScriptInfo
	
	// Display options
	showRunning   bool
	showCommand   bool
	groupByStatus bool
	
	// Process manager reference
	processMgr    *process.Manager
	packageInfo   *parser.PackageJSON
	
	// Styles
	titleStyle    lipgloss.Style
	runningStyle  lipgloss.Style
	stoppedStyle  lipgloss.Style
	commandStyle  lipgloss.Style
	helpStyle     lipgloss.Style
	modalStyle    lipgloss.Style
}

// NewScriptSelectorComponent creates a new script selector component
func NewScriptSelectorComponent(processMgr *process.Manager, packageInfo *parser.PackageJSON) *ScriptSelectorComponent {
	s := &ScriptSelectorComponent{
		scripts:       []ScriptInfo{},
		processMgr:    processMgr,
		packageInfo:   packageInfo,
		showRunning:   true,
		showCommand:   false,
		groupByStatus: false,
	}
	
	// Initialize styles
	s.initStyles()
	
	// Initialize components
	s.initializeList()
	s.initializeFilterInput()
	
	// Load scripts
	s.loadScripts()
	
	// Set visible by default
	s.visible = true
	
	return s
}

// initStyles initializes the component styles
func (s *ScriptSelectorComponent) initStyles() {
	s.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
	
	s.runningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)
	
	s.stoppedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	s.commandStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Italic(true)
	
	s.helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	s.modalStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(60)
}

// initializeList sets up the bubbles list component
func (s *ScriptSelectorComponent) initializeList() {
	items := s.scriptsToListItems()
	
	s.list = list.New(items, scriptItemDelegate{}, 80, 20)
	s.list.Title = "Scripts"
	s.list.SetShowStatusBar(true)
	s.list.SetFilteringEnabled(false) // We use custom filter
	s.list.SetShowHelp(true)
	
	// Customize key bindings
	s.list.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	)
	s.list.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	)
}

// initializeFilterInput sets up the filter input
func (s *ScriptSelectorComponent) initializeFilterInput() {
	s.filterInput = textinput.New()
	s.filterInput.Placeholder = "Type to filter scripts..."
	s.filterInput.CharLimit = 50
	s.filterInput.Width = 40
}

// loadScripts loads scripts from package.json
func (s *ScriptSelectorComponent) loadScripts() {
	s.scripts = []ScriptInfo{}
	
	if s.packageInfo == nil || s.packageInfo.Scripts == nil {
		return
	}
	
	// Get running processes
	runningProcesses := make(map[string]bool)
	if s.processMgr != nil {
		for _, p := range s.processMgr.GetAllProcesses() {
			if p.Status == process.StatusRunning {
				runningProcesses[p.Name] = true
			}
		}
	}
	
	// Convert package.json scripts to ScriptInfo
	for name, command := range s.packageInfo.Scripts {
		script := ScriptInfo{
			Name:        name,
			Command:     command,
			Description: s.getScriptDescription(name, command),
			Running:     runningProcesses[name],
		}
		
		// Parse environment variables from command
		script.Env = s.parseEnvFromCommand(command)
		
		s.scripts = append(s.scripts, script)
	}
	
	// Sort scripts by name
	sort.Slice(s.scripts, func(i, j int) bool {
		// Running scripts first
		if s.scripts[i].Running != s.scripts[j].Running {
			return s.scripts[i].Running
		}
		return s.scripts[i].Name < s.scripts[j].Name
	})
}

// getScriptDescription generates a description for a script
func (s *ScriptSelectorComponent) getScriptDescription(name, command string) string {
	// Common script patterns
	descriptions := map[string]string{
		"dev":      "Start development server",
		"start":    "Start production server",
		"build":    "Build for production",
		"test":     "Run tests",
		"lint":     "Run linter",
		"format":   "Format code",
		"clean":    "Clean build artifacts",
		"install":  "Install dependencies",
		"deploy":   "Deploy to production",
		"watch":    "Watch for changes",
	}
	
	if desc, ok := descriptions[name]; ok {
		return desc
	}
	
	// Try to infer from command
	if strings.Contains(command, "webpack") {
		return "Webpack bundler"
	}
	if strings.Contains(command, "vite") {
		return "Vite development server"
	}
	if strings.Contains(command, "jest") || strings.Contains(command, "mocha") {
		return "Test runner"
	}
	if strings.Contains(command, "eslint") {
		return "Code linter"
	}
	
	return ""
}

// parseEnvFromCommand extracts environment variables from command
func (s *ScriptSelectorComponent) parseEnvFromCommand(command string) map[string]string {
	env := make(map[string]string)
	
	// Simple parser for ENV=value patterns
	parts := strings.Fields(command)
	for _, part := range parts {
		if strings.Contains(part, "=") && !strings.HasPrefix(part, "-") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 && strings.ToUpper(kv[0]) == kv[0] {
				env[kv[0]] = kv[1]
			}
		}
	}
	
	return env
}

// Update handles messages
func (s *ScriptSelectorComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle component-specific messages
	switch msg := msg.(type) {
	case ScriptListMsg:
		// Update script list
		s.scripts = msg.Scripts
		s.updateList()
		return s, nil
		
	case ScriptSelectMsg:
		// Handle script selection and action
		for _, script := range s.scripts {
			if script.Name == msg.Script.Name {
				switch msg.Action {
				case ScriptActionRun:
					return s, s.runScript(script)
				case ScriptActionStop:
					return s, s.stopScript(script)
				case ScriptActionRestart:
					return s, s.restartScript(script)
				case ScriptActionView:
					s.showModal = true
					return s, nil
				case ScriptActionEdit:
					// Would open editor
					return s, SendSystemNotification(NotificationInfo,
						"Edit Script", "Editor integration not implemented")
				}
			}
		}
		return s, nil
		
	case ProcessListUpdateMsg:
		// Update running status
		s.loadScripts()
		s.updateList()
		return s, nil
		
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.list.SetSize(msg.Width, msg.Height-4)
		
	case tea.KeyMsg:
		// Handle modal first
		if s.showModal {
			switch msg.String() {
			case "esc", "q", "enter":
				s.showModal = false
				return s, nil
			}
			return s, nil
		}
		
		// Handle filter input
		if s.filterInput.Focused() {
			switch msg.String() {
			case "enter":
				// Apply filter
				s.applyFilter()
				s.filterInput.Blur()
				return s, nil
				
			case "esc":
				// Cancel filter
				s.filterInput.SetValue("")
				s.applyFilter()
				s.filterInput.Blur()
				return s, nil
			}
			
			// Update filter input
			var cmd tea.Cmd
			s.filterInput, cmd = s.filterInput.Update(msg)
			// Live filter
			s.applyFilter()
			return s, cmd
		}
		
		// Handle list navigation
		switch msg.String() {
		case "enter", " ":
			// Run/stop selected script
			if item := s.list.SelectedItem(); item != nil {
				if si, ok := item.(scriptItem); ok {
					if si.script.Running {
						return s, s.stopScript(si.script)
					} else {
						return s, s.runScript(si.script)
					}
				}
			}
			
		case "r":
			// Run selected script
			if item := s.list.SelectedItem(); item != nil {
				if si, ok := item.(scriptItem); ok {
					return s, s.runScript(si.script)
				}
			}
			
		case "s":
			// Stop selected script
			if item := s.list.SelectedItem(); item != nil {
				if si, ok := item.(scriptItem); ok {
					return s, s.stopScript(si.script)
				}
			}
			
		case "R":
			// Restart selected script
			if item := s.list.SelectedItem(); item != nil {
				if si, ok := item.(scriptItem); ok {
					return s, s.restartScript(si.script)
				}
			}
			
		case "v":
			// View script details
			if item := s.list.SelectedItem(); item != nil {
				if si, ok := item.(scriptItem); ok {
					s.selectedScript = &si.script
					s.showModal = true
				}
			}
			
		case "/":
			// Focus filter input
			s.filterInput.Focus()
			return s, textinput.Blink
			
		case "c":
			// Toggle command display
			s.showCommand = !s.showCommand
			s.updateList()
			return s, nil
			
		case "g":
			// Toggle grouping by status
			s.groupByStatus = !s.groupByStatus
			s.updateList()
			return s, nil
			
		case "a":
			// Run all scripts
			var runCmds []tea.Cmd
			for _, script := range s.scripts {
				if !script.Running {
					runCmds = append(runCmds, s.runScript(script))
				}
			}
			return s, tea.Batch(runCmds...)
			
		case "S":
			// Stop all scripts
			var stopCmds []tea.Cmd
			for _, script := range s.scripts {
				if script.Running {
					stopCmds = append(stopCmds, s.stopScript(script))
				}
			}
			return s, tea.Batch(stopCmds...)
		}
	}
	
	// Update the list component
	if s.visible && !s.showModal && !s.filterInput.Focused() {
		newList, cmd := s.list.Update(msg)
		s.list = newList
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	return s, tea.Batch(cmds...)
}

// View renders the component
func (s *ScriptSelectorComponent) View() string {
	if !s.visible {
		return ""
	}
	
	// Show modal if active
	if s.showModal && s.selectedScript != nil {
		return s.renderModal()
	}
	
	var b strings.Builder
	
	// Title
	title := fmt.Sprintf("📜 Scripts (%d)", len(s.scripts))
	running := 0
	for _, script := range s.scripts {
		if script.Running {
			running++
		}
	}
	if running > 0 {
		title += fmt.Sprintf(" - %d running", running)
	}
	b.WriteString(s.titleStyle.Render(title) + "\n")
	
	// Filter input or help
	if s.filterInput.Focused() {
		b.WriteString("Filter: " + s.filterInput.View() + "\n")
	} else {
		help := []string{
			"enter: run/stop",
			"r: run",
			"s: stop",
			"R: restart",
			"v: view",
			"/: filter",
			"c: commands",
		}
		b.WriteString(s.helpStyle.Render(strings.Join(help, " | ")) + "\n")
	}
	
	// List
	b.WriteString(s.list.View())
	
	return b.String()
}

// renderModal renders the script details modal
func (s *ScriptSelectorComponent) renderModal() string {
	if s.selectedScript == nil {
		return ""
	}
	
	var content strings.Builder
	
	// Title
	status := "⭕ Stopped"
	if s.selectedScript.Running {
		status = "🟢 Running"
	}
	content.WriteString(fmt.Sprintf("Script: %s %s\n\n", s.selectedScript.Name, status))
	
	// Command
	content.WriteString("Command:\n")
	content.WriteString(s.commandStyle.Render(s.selectedScript.Command) + "\n\n")
	
	// Description
	if s.selectedScript.Description != "" {
		content.WriteString("Description:\n")
		content.WriteString(s.selectedScript.Description + "\n\n")
	}
	
	// Environment variables
	if len(s.selectedScript.Env) > 0 {
		content.WriteString("Environment:\n")
		for k, v := range s.selectedScript.Env {
			content.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
		content.WriteString("\n")
	}
	
	// Help
	content.WriteString(s.helpStyle.Render("Press ESC to close"))
	
	return s.modalStyle.Render(content.String())
}

// Helper methods for script management

func (s *ScriptSelectorComponent) runScript(script ScriptInfo) tea.Cmd {
	if s.processMgr == nil || script.Running {
		return nil
	}
	
	return SendProcessStart(script.Name)
}

func (s *ScriptSelectorComponent) stopScript(script ScriptInfo) tea.Cmd {
	if s.processMgr == nil || !script.Running {
		return nil
	}
	
	// Find process ID
	for _, p := range s.processMgr.GetAllProcesses() {
		if p.Name == script.Name && p.Status == process.StatusRunning {
			return SendProcessStop(p.ID, false)
		}
	}
	return nil
}

func (s *ScriptSelectorComponent) restartScript(script ScriptInfo) tea.Cmd {
	// Stop then start
	var cmds []tea.Cmd
	if script.Running {
		cmds = append(cmds, s.stopScript(script))
	}
	cmds = append(cmds, s.runScript(script))
	return tea.Batch(cmds...)
}

func (s *ScriptSelectorComponent) applyFilter() {
	filter := strings.ToLower(s.filterInput.Value())
	if filter == "" {
		s.updateList()
		return
	}
	
	// Filter scripts
	var filtered []list.Item
	for _, script := range s.scripts {
		nameLower := strings.ToLower(script.Name)
		commandLower := strings.ToLower(script.Command)
		if strings.Contains(nameLower, filter) || strings.Contains(commandLower, filter) {
			filtered = append(filtered, scriptItem{
				script:      script,
				showCommand: s.showCommand,
			})
		}
	}
	
	s.list.SetItems(filtered)
}

func (s *ScriptSelectorComponent) updateList() {
	items := s.scriptsToListItems()
	s.list.SetItems(items)
}

func (s *ScriptSelectorComponent) scriptsToListItems() []list.Item {
	if s.groupByStatus {
		var items []list.Item
		
		// Running scripts
		var running []ScriptInfo
		var stopped []ScriptInfo
		
		for _, script := range s.scripts {
			if script.Running {
				running = append(running, script)
			} else {
				stopped = append(stopped, script)
			}
		}
		
		// Add running group
		if len(running) > 0 {
			items = append(items, scriptGroupItem{
				title: "Running",
				count: len(running),
			})
			for _, script := range running {
				items = append(items, scriptItem{
					script:      script,
					showCommand: s.showCommand,
					indented:    true,
				})
			}
		}
		
		// Add stopped group
		if len(stopped) > 0 {
			items = append(items, scriptGroupItem{
				title: "Available",
				count: len(stopped),
			})
			for _, script := range stopped {
				items = append(items, scriptItem{
					script:      script,
					showCommand: s.showCommand,
					indented:    true,
				})
			}
		}
		
		return items
	}
	
	// Ungrouped
	items := make([]list.Item, len(s.scripts))
	for i, script := range s.scripts {
		items[i] = scriptItem{
			script:      script,
			showCommand: s.showCommand,
			indented:    false,
		}
	}
	return items
}


// Focus sets focus to this component
func (s *ScriptSelectorComponent) Focus() tea.Cmd {
	s.focused = true
	return nil
}

// Blur removes focus from this component
func (s *ScriptSelectorComponent) Blur() tea.Cmd {
	s.focused = false
	s.filterInput.Blur()
	return nil
}

// List item implementations

type scriptItem struct {
	script      ScriptInfo
	showCommand bool
	indented    bool
}

func (i scriptItem) Title() string {
	prefix := ""
	if i.indented {
		prefix = "  "
	}
	
	status := "⭕"
	if i.script.Running {
		status = "🟢"
	}
	
	return fmt.Sprintf("%s%s %s", prefix, status, i.script.Name)
}

func (i scriptItem) Description() string {
	desc := ""
	if i.indented {
		desc = "  "
	}
	
	if i.showCommand {
		cmd := i.script.Command
		if len(cmd) > 60 {
			cmd = cmd[:57] + "..."
		}
		desc += cmd
	} else if i.script.Description != "" {
		desc += i.script.Description
	}
	
	return desc
}

func (i scriptItem) FilterValue() string {
	return i.script.Name + " " + i.script.Command
}

type scriptGroupItem struct {
	title string
	count int
}

func (i scriptGroupItem) Title() string {
	return fmt.Sprintf("📦 %s (%d)", i.title, i.count)
}

func (i scriptGroupItem) Description() string {
	return ""
}

func (i scriptGroupItem) FilterValue() string {
	return i.title
}

// scriptItemDelegate implements list.ItemDelegate
type scriptItemDelegate struct{}

func (d scriptItemDelegate) Height() int { return 2 }

func (d scriptItemDelegate) Spacing() int { return 1 }

func (d scriptItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d scriptItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// Create styles inline
	itemStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("170")).
		Bold(true)
	groupStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
	
	var str string
	
	switch item := listItem.(type) {
	case scriptItem:
		str = fmt.Sprintf("%s\n  %s", item.Title(), item.Description())
	case scriptGroupItem:
		str = groupStyle.Render(item.Title())
	default:
		return
	}
	
	var fn func(...string) string
	if index == m.Index() {
		fn = func(strs ...string) string {
			s := strings.Join(strs, "")
			return selectedStyle.Render("> " + s)
		}
	} else {
		fn = itemStyle.Render
	}
	
	fmt.Fprintf(w, "%s", fn(str))
}