package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/beagle/brummer/internal/logs"
	"github.com/beagle/brummer/internal/mcp"
	"github.com/beagle/brummer/internal/parser"
	"github.com/beagle/brummer/internal/process"
	"github.com/beagle/brummer/pkg/events"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type View string

const (
	ViewScripts View = "scripts"
	ViewProcesses View = "processes"
	ViewLogs View = "logs"
	ViewErrors View = "errors"
	ViewURLs View = "urls"
	ViewSettings View = "settings"
	ViewSearch View = "search"
	ViewFilters View = "filters"
)

type Model struct {
	processMgr   *process.Manager
	logStore     *logs.Store
	eventBus     *events.EventBus
	mcpServer    *mcp.Server
	
	currentView  View
	width        int
	height       int
	
	scriptsList     list.Model
	processesList   list.Model
	logsViewport    viewport.Model
	errorsViewport  viewport.Model
	errorsList      list.Model
	selectedError   *logs.ErrorContext
	errorDetailView viewport.Model
	urlsViewport    viewport.Model
	settingsList    list.Model
	searchInput     textinput.Model
	
	selectedProcess string
	searchResults   []logs.LogEntry
	showHighPriority bool
	
	// File browser state
	showingFileBrowser bool
	currentPath        string
	fileList          []FileItem
	
	// Run dialog state
	showingRunDialog bool
	commandsList     list.Model
	detectedCommands []parser.ExecutableCommand
	monorepoInfo     *parser.MonorepoInfo
	
	// UI state
	copyNotification string
	notificationTime time.Time
	
	help         help.Model
	keys         keyMap
	
	updateChan   chan tea.Msg
}

type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Enter       key.Binding
	Back        key.Binding
	Quit        key.Binding
	Tab         key.Binding
	Search      key.Binding
	Filter      key.Binding
	Stop        key.Binding
	Restart     key.Binding
	RestartAll  key.Binding
	CopyError   key.Binding
	Priority    key.Binding
	ClearLogs   key.Binding
	ClearErrors key.Binding
	Help        key.Binding
	RunDialog   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc/q", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch view"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filters"),
	),
	Stop: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stop process"),
	),
	Restart: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart process"),
	),
	RestartAll: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "restart all"),
	),
	CopyError: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy recent error"),
	),
	Priority: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "toggle priority"),
	),
	ClearLogs: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "clear logs"),
	),
	ClearErrors: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "clear errors"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	RunDialog: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new process"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Enter, k.Back, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Tab, k.Search, k.Filter, k.Priority},
		{k.Stop, k.Restart, k.RestartAll, k.CopyError},
		{k.ClearLogs, k.ClearErrors, k.Help, k.Quit},
	}
}

type scriptItem struct {
	name   string
	script string
}

func (i scriptItem) FilterValue() string { return i.name }
func (i scriptItem) Title() string       { return i.name }
func (i scriptItem) Description() string { return i.script }

type commandItem struct {
	command parser.ExecutableCommand
}

func (i commandItem) FilterValue() string { return i.command.Name }
func (i commandItem) Title() string {
	categoryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	return fmt.Sprintf("%s %s", i.command.Name, categoryStyle.Render(fmt.Sprintf("[%s]", i.command.Category)))
}
func (i commandItem) Description() string { return i.command.Description }

type runCustomItem struct{}

func (i runCustomItem) FilterValue() string { return "custom command" }
func (i runCustomItem) Title() string       { return "➕ Run Custom Command..." }
func (i runCustomItem) Description() string { return "Run a custom command not listed above" }

type errorItem struct {
	errorCtx *logs.ErrorContext
}

func (i errorItem) FilterValue() string { return i.errorCtx.Message }
func (i errorItem) Title() string {
	// Show error type with icon
	icon := "❌"
	if i.errorCtx.Severity == "warning" {
		icon = "⚠️"
	} else if i.errorCtx.Severity == "critical" {
		icon = "🔥"
	}
	
	// Truncate message if too long
	msg := i.errorCtx.Message
	if len(msg) > 50 {
		msg = msg[:47] + "..."
	}
	
	return fmt.Sprintf("%s %s: %s", icon, i.errorCtx.Type, msg)
}
func (i errorItem) Description() string {
	// Show process and time
	return fmt.Sprintf("%s | %s", i.errorCtx.ProcessName, i.errorCtx.Timestamp.Format("15:04:05"))
}

type processItem struct {
	process   *process.Process
	isHeader  bool
	headerText string
}

func (i processItem) FilterValue() string { 
	if i.isHeader {
		return ""
	}
	return i.process.Name 
}
func (i processItem) Title() string {
	if i.isHeader {
		// Return section header with styling
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
		if i.headerText == "Closed Processes" {
			headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
		}
		return headerStyle.Render(i.headerText)
	}
	
	status := string(i.process.Status)
	var statusEmoji string
	switch i.process.Status {
	case process.StatusRunning:
		statusEmoji = "🟢"
	case process.StatusStopped:
		statusEmoji = "✓"  // Thin checkmark for gracefully stopped
	case process.StatusFailed:
		statusEmoji = "❌"
	case process.StatusSuccess:
		statusEmoji = "✓"  // Thin checkmark for success
	default:
		statusEmoji = "⏸️"
	}
	title := fmt.Sprintf("%s [%s] %s", statusEmoji, status, i.process.Name)
	
	
	return title
}
func (i processItem) Description() string {
	if i.isHeader {
		// Return separator line for headers
		return strings.Repeat("─", 40)
	}
	
	var parts []string
	
	// Add PID and start time
	parts = append(parts, fmt.Sprintf("PID: %s", i.process.ID))
	parts = append(parts, fmt.Sprintf("Started: %s", i.process.StartTime.Format("15:04:05")))
	
	
	// Add actions
	var actions string
	if i.process.Status == process.StatusRunning {
		actions = "Press 's' to stop, 'r' to restart"
	} else {
		actions = "Press 'Enter' to view logs"
	}
	parts = append(parts, actions)
	
	return strings.Join(parts, " | ")
}

type packageManagerItem struct {
	manager  parser.InstalledPackageManager
	current  bool
	fromJSON bool
}

func (i packageManagerItem) FilterValue() string { return string(i.manager.Manager) }
func (i packageManagerItem) Title() string {
	title := string(i.manager.Manager)
	if i.current {
		title = "▶ " + title
	} else {
		title = "  " + title
	}
	if i.fromJSON {
		title += " (from package.json)"
	}
	return title
}
func (i packageManagerItem) Description() string {
	return fmt.Sprintf("v%s | %s", i.manager.Version, i.manager.Path)
}

type settingsItem interface {
	list.Item
	isSettingsItem()
}

type packageManagerSettingsItem struct {
	packageManagerItem
}

func (i packageManagerSettingsItem) isSettingsItem() {}

type mcpInstallItem struct {
	tool      mcp.Tool
	installed bool
}

func (i mcpInstallItem) FilterValue() string { return i.tool.Name }
func (i mcpInstallItem) Title() string {
	title := i.tool.Name
	if i.installed {
		title = "✓ " + title
	} else {
		title = "  " + title
	}
	if !i.tool.Supported {
		title += " (experimental)"
	}
	return title
}
func (i mcpInstallItem) Description() string {
	if i.installed {
		return "MCP server installed"
	}
	return "Click to install MCP server"
}
func (i mcpInstallItem) isSettingsItem() {}

type settingsSectionItem struct {
	title string
}

func (i settingsSectionItem) FilterValue() string { return "" }
func (i settingsSectionItem) Title() string       { return i.title }
func (i settingsSectionItem) Description() string { return "" }
func (i settingsSectionItem) isSettingsItem()     {}

type mcpFileBrowserItem struct{}

func (i mcpFileBrowserItem) FilterValue() string  { return "custom file" }
func (i mcpFileBrowserItem) Title() string        { return "Browse for Custom Config..." }
func (i mcpFileBrowserItem) Description() string  { return "Browse for a JSON config file to add Brummer" }
func (i mcpFileBrowserItem) isSettingsItem()      {}

type FileItem struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
}

func (i FileItem) FilterValue() string { return i.Name }
func (i FileItem) Title() string {
	if i.IsDir {
		return "📁 " + i.Name
	}
	if strings.HasSuffix(strings.ToLower(i.Name), ".json") {
		return "📄 " + i.Name
	}
	return "📄 " + i.Name
}
func (i FileItem) Description() string {
	if i.IsDir {
		return "Directory"
	}
	return fmt.Sprintf("File (%d bytes)", i.Size)
}

func NewModel(processMgr *process.Manager, logStore *logs.Store, eventBus *events.EventBus, mcpServer *mcp.Server) Model {
	return NewModelWithView(processMgr, logStore, eventBus, mcpServer, ViewScripts)
}

func NewModelWithView(processMgr *process.Manager, logStore *logs.Store, eventBus *events.EventBus, mcpServer *mcp.Server, initialView View) Model {
	scripts := processMgr.GetScripts()
	scriptItems := make([]list.Item, 0, len(scripts))
	for name, script := range scripts {
		scriptItems = append(scriptItems, scriptItem{name: name, script: script})
	}

	scriptsList := list.New(scriptItems, list.NewDefaultDelegate(), 0, 0)
	scriptsList.Title = "Available Scripts"
	scriptsList.SetShowStatusBar(false)
	scriptsList.SetFilteringEnabled(true)

	processesList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	processesList.Title = "Running Processes"
	processesList.SetShowStatusBar(false)

	searchInput := textinput.New()
	searchInput.Placeholder = "Search logs..."
	searchInput.Focus()

	// Create settings list with package managers
	settingsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	settingsList.Title = "Package Manager Settings"
	settingsList.SetShowStatusBar(false)

	m := Model{
		processMgr:     processMgr,
		logStore:       logStore,
		eventBus:       eventBus,
		mcpServer:      mcpServer,
		currentView:    initialView,
		scriptsList:    scriptsList,
		processesList:  processesList,
		settingsList:   settingsList,
		logsViewport:   viewport.New(0, 0),
		errorsViewport: viewport.New(0, 0),
		urlsViewport:   viewport.New(0, 0),
		searchInput:    searchInput,
		help:           help.New(),
		keys:           keys,
		updateChan:     make(chan tea.Msg, 100),
		currentPath:    getCurrentDir(),
	}

	// Note: Log callback is registered in main.go to avoid duplication

	// Initialize settings list
	m.updateSettingsList()
	
	// Initialize commands list for run dialog
	commandsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	commandsList.Title = "Available Commands"
	commandsList.SetShowStatusBar(false)
	commandsList.SetFilteringEnabled(true)
	m.commandsList = commandsList
	
	// Initialize errors list
	errorsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	errorsList.Title = "Errors"
	errorsList.SetShowStatusBar(false)
	errorsList.SetFilteringEnabled(true)
	m.errorsList = errorsList
	
	// Initialize error detail view
	m.errorDetailView = viewport.New(0, 0)
	
	// Check for monorepo on startup
	m.monorepoInfo, _ = processMgr.GetMonorepoInfo()

	return m
}

func (m Model) Init() tea.Cmd {
	// Set up event subscriptions
	m.eventBus.Subscribe(events.ProcessStarted, func(e events.Event) {
		m.updateChan <- processUpdateMsg{}
	})
	
	m.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		// Clean up URLs from the exited process
		if e.ProcessID != "" {
			m.logStore.RemoveURLsForProcess(e.ProcessID)
		}
		m.updateChan <- processUpdateMsg{}
	})
	
	m.eventBus.Subscribe(events.LogLine, func(e events.Event) {
		m.updateChan <- logUpdateMsg{}
	})
	
	
	m.eventBus.Subscribe(events.ErrorDetected, func(e events.Event) {
		m.updateChan <- errorUpdateMsg{}
	})

	return tea.Batch(
		textinput.Blink,
		m.waitForUpdates(),
		m.tickCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()

	case tea.KeyMsg:
		// Handle number keys 1-6 for tab switching
		switch msg.String() {
		case "1":
			m.currentView = ViewScripts
			return m, nil
		case "2":
			m.currentView = ViewProcesses
			return m, nil
		case "3":
			m.currentView = ViewLogs
			return m, nil
		case "4":
			m.currentView = ViewErrors
			return m, nil
		case "5":
			m.currentView = ViewURLs
			return m, nil
		case "6":
			m.currentView = ViewSettings
			return m, nil
		}
		
		switch {
		case key.Matches(msg, m.keys.Quit):
			// Check if there are running processes
			runningProcesses := 0
			for _, proc := range m.processMgr.GetAllProcesses() {
				if proc.Status == process.StatusRunning {
					runningProcesses++
				}
			}
			
			if runningProcesses > 0 {
				// Add a message about stopping processes
				return m, tea.Sequence(
					tea.Printf("Stopping %d running processes...\n", runningProcesses),
					func() tea.Msg {
						m.processMgr.Cleanup()
						return tea.Msg(nil)
					},
					tea.Printf(renderExitScreen()),
					tea.Quit,
				)
			} else {
				return m, tea.Sequence(
					tea.Printf(renderExitScreen()),
					tea.Quit,
				)
			}

		case key.Matches(msg, m.keys.Tab):
			m.cycleView()

		case key.Matches(msg, m.keys.Back):
			if m.currentView == ViewSearch || m.currentView == ViewFilters {
				m.currentView = ViewLogs
			} else if m.currentView == ViewLogs || m.currentView == ViewErrors || m.currentView == ViewURLs {
				m.currentView = ViewProcesses
			}

		case key.Matches(msg, m.keys.Search):
			if m.currentView == ViewLogs {
				m.currentView = ViewSearch
				m.searchInput.Focus()
			}

		case key.Matches(msg, m.keys.Priority):
			if m.currentView == ViewLogs {
				m.showHighPriority = !m.showHighPriority
				m.updateLogsView()
			}

		case key.Matches(msg, m.keys.RestartAll):
			if m.currentView == ViewProcesses {
				m.logStore.Add("system", "System", "Restarting all running processes...", false)
				cmds = append(cmds, m.handleRestartAll())
			}

		case key.Matches(msg, m.keys.CopyError):
			cmds = append(cmds, m.handleCopyError())

		case key.Matches(msg, m.keys.ClearLogs):
			if m.currentView == ViewLogs {
				m.handleClearLogs()
			}

		case key.Matches(msg, m.keys.ClearErrors):
			if m.currentView == ViewErrors {
				m.handleClearErrors()
			}

		case key.Matches(msg, m.keys.Enter):
			cmds = append(cmds, m.handleEnter())
			
		case key.Matches(msg, m.keys.RunDialog):
			if m.currentView == ViewScripts && !m.showingRunDialog {
				m.showRunDialog()
			}
		}

	case processUpdateMsg:
		m.updateProcessList()
		cmds = append(cmds, m.waitForUpdates())

	case logUpdateMsg:
		m.updateLogsView()
		cmds = append(cmds, m.waitForUpdates())
		
	case errorUpdateMsg:
		m.updateErrorsList()
		cmds = append(cmds, m.waitForUpdates())
		
	case tickMsg:
		// Continue ticking for periodic updates (e.g., browser status)
		cmds = append(cmds, m.tickCmd())
	}

	// Handle run dialog updates
	if m.showingRunDialog {
		// Handle escape key to close dialog
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Back) {
			m.showingRunDialog = false
			return m, nil
		}
		
		// Handle enter key to run command
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Enter) {
			cmds = append(cmds, m.handleRunCommand())
			return m, tea.Batch(cmds...)
		}
		
		// Update the commands list
		newList, cmd := m.commandsList.Update(msg)
		m.commandsList = newList
		cmds = append(cmds, cmd)
		
		return m, tea.Batch(cmds...)
	}

	switch m.currentView {
	case ViewScripts:
		newList, cmd := m.scriptsList.Update(msg)
		m.scriptsList = newList
		cmds = append(cmds, cmd)

	case ViewProcesses:
		// Handle process-specific key commands BEFORE list update
		// This ensures our keys take precedence over list navigation
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(msg, m.keys.Stop):
				if i, ok := m.processesList.SelectedItem().(processItem); ok && !i.isHeader && i.process != nil {
					if err := m.processMgr.StopProcess(i.process.ID); err != nil {
						m.logStore.Add("system", "System", fmt.Sprintf("Failed to stop process %s: %v", i.process.Name, err), true)
					} else {
						m.logStore.Add("system", "System", fmt.Sprintf("Stopping process: %s", i.process.Name), false)
					}
					cmds = append(cmds, m.waitForUpdates())
				} else {
					m.logStore.Add("system", "System", "No process selected to stop", true)
				}
				// Don't update the list for this key, we handled it
				return m, tea.Batch(cmds...)
				
			case key.Matches(msg, m.keys.Restart):
				if i, ok := m.processesList.SelectedItem().(processItem); ok && !i.isHeader && i.process != nil {
					cmds = append(cmds, m.handleRestartProcess(i.process))
					m.logStore.Add("system", "System", fmt.Sprintf("Restarting process: %s", i.process.Name), false)
				} else {
					m.logStore.Add("system", "System", "No process selected to restart", true)
				}
				// Don't update the list for this key, we handled it
				return m, tea.Batch(cmds...)
			}
		}
		
		// Update the list only if we didn't handle the key above
		newList, cmd := m.processesList.Update(msg)
		m.processesList = newList
		cmds = append(cmds, cmd)

	case ViewLogs, ViewURLs:
		newViewport, cmd := m.logsViewport.Update(msg)
		m.logsViewport = newViewport
		cmds = append(cmds, cmd)
		
	case ViewErrors:
		// Update errors list
		newList, cmd := m.errorsList.Update(msg)
		m.errorsList = newList
		cmds = append(cmds, cmd)
		
		// Update detail view
		newDetail, cmd := m.errorDetailView.Update(msg)
		m.errorDetailView = newDetail
		cmds = append(cmds, cmd)
		
		// Handle selection change (both Enter and arrow keys)
		if msg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, m.keys.Enter) || key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down) {
				if i, ok := m.errorsList.SelectedItem().(errorItem); ok {
					m.selectedError = i.errorCtx
					m.updateErrorDetailView()
				}
			}
		}

	case ViewSettings:
		if m.showingFileBrowser {
			cmds = append(cmds, m.handleFileBrowser(msg))
		} else {
			newList, cmd := m.settingsList.Update(msg)
			m.settingsList = newList
			cmds = append(cmds, cmd)

			if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Enter) {
				if i, ok := m.settingsList.SelectedItem().(packageManagerSettingsItem); ok {
					if err := m.processMgr.SetUserPackageManager(i.manager.Manager); err != nil {
						// Log error but don't crash
						m.logStore.Add("system", "System", fmt.Sprintf("Error saving preference: %v", err), true)
					}
					m.updateSettingsList()
				} else if i, ok := m.settingsList.SelectedItem().(mcpInstallItem); ok {
					m.installMCPForTool(i.tool)
				} else if _, ok := m.settingsList.SelectedItem().(mcpFileBrowserItem); ok {
					m.showingFileBrowser = true
					m.loadFileList()
				}
			}
		}

	case ViewSearch:
		newInput, cmd := m.searchInput.Update(msg)
		m.searchInput = newInput
		cmds = append(cmds, cmd)

		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Enter) {
			m.searchResults = m.logStore.Search(m.searchInput.Value())
			m.currentView = ViewLogs
			m.updateLogsView()
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string
	
	// Show run dialog if active
	if m.showingRunDialog {
		content = m.renderRunDialog()
	} else {
		switch m.currentView {
		case ViewScripts:
			content = m.renderScriptsView()
		case ViewProcesses:
			content = m.renderProcessesView()
		case ViewLogs:
			content = m.renderLogsView()
		case ViewErrors:
			content = m.renderErrorsViewSplit()
		case ViewURLs:
			content = m.renderURLsView()
		case ViewSettings:
			if m.showingFileBrowser {
				content = m.renderFileBrowser()
			} else {
				content = m.settingsList.View()
			}
		case ViewSearch:
			content = m.renderSearchView()
		case ViewFilters:
			content = m.renderFiltersView()
		}
	}

	helpView := m.help.View(m.keys)
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(),
		content,
		helpView,
	)
}

func (m *Model) updateSizes() {
	headerHeight := 3  // title + tabs + separator
	helpHeight := 3
	contentHeight := m.height - headerHeight - helpHeight

	m.scriptsList.SetSize(m.width, contentHeight)
	m.processesList.SetSize(m.width, contentHeight)
	m.settingsList.SetSize(m.width, contentHeight)
	m.commandsList.SetSize(m.width, contentHeight)
	m.errorsList.SetSize(m.width/3, contentHeight) // Split view
	m.logsViewport.Width = m.width
	m.logsViewport.Height = contentHeight
	m.errorsViewport.Width = m.width
	m.errorsViewport.Height = contentHeight
	m.errorDetailView.Width = m.width*2/3
	m.errorDetailView.Height = contentHeight
	m.urlsViewport.Width = m.width
	m.urlsViewport.Height = contentHeight
}

func (m *Model) cycleView() {
	views := []View{ViewScripts, ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewSettings}
	for i, v := range views {
		if v == m.currentView {
			m.currentView = views[(i+1)%len(views)]
			break
		}
	}
}

func (m *Model) handleEnter() tea.Cmd {
	switch m.currentView {
	case ViewScripts:
		if i, ok := m.scriptsList.SelectedItem().(scriptItem); ok {
			go func() {
				_, err := m.processMgr.StartScript(i.name)
				if err != nil {
					// Send error as a log message
					m.logStore.Add("system", "System", fmt.Sprintf("Error starting script %s: %v", i.name, err), true)
					m.updateChan <- logUpdateMsg{}
				}
			}()
			m.currentView = ViewProcesses
			// Force immediate process list update
			m.updateProcessList()
			return m.waitForUpdates()
		}
		
	case ViewProcesses:
		if i, ok := m.processesList.SelectedItem().(processItem); ok {
			m.selectedProcess = i.process.ID
			m.currentView = ViewLogs
			m.updateLogsView()
		}
	}
	
	return nil
}

func (m *Model) updateProcessList() {
	processes := m.processMgr.GetAllProcesses()
	
	// Separate and sort processes: running first, then closed
	var runningProcesses []*process.Process
	var closedProcesses []*process.Process
	
	for _, p := range processes {
		if p.Status == process.StatusRunning || p.Status == process.StatusPending {
			runningProcesses = append(runningProcesses, p)
		} else {
			closedProcesses = append(closedProcesses, p)
		}
	}
	
	var items []list.Item
	
	// Add running processes with header
	if len(runningProcesses) > 0 {
		items = append(items, processItem{
			isHeader:   true,
			headerText: "Running Processes",
		})
		
		for _, p := range runningProcesses {
			items = append(items, processItem{
				process: p,
			})
		}
	}
	
	// Add closed processes with header
	if len(closedProcesses) > 0 {
		if len(runningProcesses) > 0 {
			// Add a blank separator between sections
			items = append(items, processItem{
				isHeader:   true,
				headerText: "",
			})
		}
		
		items = append(items, processItem{
			isHeader:   true,
			headerText: "Closed Processes",
		})
		
		for _, p := range closedProcesses {
			items = append(items, processItem{
				process: p,
			})
		}
	}
	
	m.processesList.SetItems(items)
	
	// Update title to show counts
	if len(runningProcesses) > 0 && len(closedProcesses) > 0 {
		m.processesList.Title = fmt.Sprintf("Processes (%d running, %d closed)", len(runningProcesses), len(closedProcesses))
	} else if len(runningProcesses) > 0 {
		m.processesList.Title = fmt.Sprintf("Processes (%d running)", len(runningProcesses))
	} else if len(closedProcesses) > 0 {
		m.processesList.Title = fmt.Sprintf("Processes (%d closed)", len(closedProcesses))
	} else {
		m.processesList.Title = "Processes"
	}
}

func (m *Model) updateLogsView() {
	var logs []logs.LogEntry
	
	if len(m.searchResults) > 0 {
		logs = m.searchResults
	} else if m.showHighPriority {
		logs = m.logStore.GetHighPriority(30)
	} else if m.selectedProcess != "" {
		logs = m.logStore.GetByProcess(m.selectedProcess)
	} else {
		logs = m.logStore.GetAll()
	}

	var content strings.Builder
	for _, log := range logs {
		// Skip empty log entries (used for separation)
		if strings.TrimSpace(log.Content) == "" {
			continue
		}
		
		style := m.getLogStyle(log)
		
		// Clean up the log content
		cleanContent := m.cleanLogContent(log.Content)
		
		// Always ensure each log entry ends with proper line termination (CR+LF)
		// This ensures the cursor resets to column 0 for the next line
		if !strings.HasSuffix(cleanContent, "\n") {
			cleanContent += "\r\n"
		} else {
			// Replace existing \n with \r\n to ensure cursor reset
			cleanContent = strings.TrimSuffix(cleanContent, "\n") + "\r\n"
		}
		
		// Format the timestamp and process name with style, but keep the content raw
		// to preserve ANSI codes in the log output
		prefix := fmt.Sprintf("[%s] %s: ", 
			log.Timestamp.Format("15:04:05"),
			log.ProcessName,
		)
		
		// Apply style only to the prefix, not the content
		content.WriteString(style.Render(prefix))
		content.WriteString(cleanContent)
	}

	m.logsViewport.SetContent(content.String())
}

func (m Model) cleanLogContent(content string) string {
	// Keep the original content with ANSI codes
	cleaned := content
	
	// Handle different line ending styles - ensure proper line endings
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")  // Windows line endings -> Unix
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")    // Lone CR -> newline (for terminal resets)
	
	// Don't trim or limit - preserve the original formatting
	// The terminal/viewport will handle wrapping and display
	
	return cleaned
}

func (m Model) getLogStyle(log logs.LogEntry) lipgloss.Style {
	base := lipgloss.NewStyle()
	
	switch log.Level {
	case logs.LevelError, logs.LevelCritical:
		return base.Foreground(lipgloss.Color("196"))
	case logs.LevelWarn:
		return base.Foreground(lipgloss.Color("214"))
	case logs.LevelDebug:
		return base.Foreground(lipgloss.Color("245"))
	default:
		if log.Priority > 50 {
			return base.Bold(true)
		}
		return base
	}
}

func (m Model) renderHeader() string {
	// Get process count information
	processes := m.processMgr.GetAllProcesses()
	runningCount := 0
	for _, proc := range processes {
		if proc.Status == process.StatusRunning {
			runningCount++
		}
	}
	
	// Build title with process info
	baseTitle := "🐝 Brummer - Package Script Manager"
	var processInfo string
	if len(processes) > 0 {
		if runningCount > 0 {
			processInfo = fmt.Sprintf(" (%d processes, %d running)", len(processes), runningCount)
		} else {
			processInfo = fmt.Sprintf(" (%d processes)", len(processes))
		}
	}
	
	// Browser connection removed
	browserIcon := ""
	
	// Add copy notification if recent
	notification := ""
	if m.copyNotification != "" && time.Since(m.notificationTime) < 3*time.Second {
		notificationStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
		notification = " " + notificationStyle.Render(m.copyNotification)
	}
	
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Render(baseTitle + processInfo + browserIcon + notification)

	tabs := []string{}
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	tabViews := []View{ViewScripts, ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewSettings}
	for i, v := range tabViews {
		label := fmt.Sprintf("%d.%s", i+1, string(v))
		if v == m.currentView {
			tabs = append(tabs, activeStyle.Render("▶ " + label))
		} else {
			tabs = append(tabs, inactiveStyle.Render("  " + label))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Left, 
		tabs[0], " | ", tabs[1], " | ", tabs[2], " | ", tabs[3], " | ", tabs[4], " | ", tabs[5])

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		tabBar,
		strings.Repeat("─", m.width),
	)
}

func (m Model) renderProcessesView() string {
	processes := m.processMgr.GetAllProcesses()
	
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("Select process: ↑/↓ | Stop: s | Restart: r | Restart All: Ctrl+R | View Logs: Enter")
	
	if len(processes) == 0 {
		emptyState := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("No processes yet. Go to Scripts tab and press Enter on a script to start it.")
		
		return lipgloss.JoinVertical(lipgloss.Left, 
			instructions,
			"",
			emptyState,
		)
	}
	
	return lipgloss.JoinVertical(lipgloss.Left, 
		instructions,
		"",
		m.processesList.View(),
	)
}

func (m Model) renderLogsView() string {
	title := "Logs"
	if m.selectedProcess != "" {
		if proc, exists := m.processMgr.GetProcess(m.selectedProcess); exists {
			title = fmt.Sprintf("Logs - %s", proc.Name)
		}
	}
	if m.showHighPriority {
		title += " [High Priority]"
	}
	
	header := lipgloss.NewStyle().Bold(true).Render(title)
	return lipgloss.JoinVertical(lipgloss.Left, header, m.logsViewport.View())
}

func (m Model) renderSearchView() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"Search Logs:",
		m.searchInput.View(),
	)
}

func (m Model) renderFiltersView() string {
	filters := m.logStore.GetFilters()
	if len(filters) == 0 {
		return "No filters configured"
	}

	var content strings.Builder
	for _, f := range filters {
		content.WriteString(fmt.Sprintf("• %s: %s (Priority +%d)\n", f.Name, f.Pattern, f.PriorityBoost))
	}
	
	return content.String()
}

func (m *Model) renderErrorsView() string {
	errorContexts := m.logStore.GetErrorContexts()
	
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent Errors") + "\n\n")
	
	if len(errorContexts) == 0 {
		// Fall back to simple errors if no contexts
		errors := m.logStore.GetErrors()
		if len(errors) == 0 {
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No errors detected yet"))
		} else {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
			
			// Show most recent errors first
			for i := len(errors) - 1; i >= 0 && i >= len(errors)-20; i-- {
				err := errors[i]
				content.WriteString(fmt.Sprintf("%s %s\n%s\n\n",
					timeStyle.Render(err.Timestamp.Format("15:04:05")),
					processStyle.Render(fmt.Sprintf("[%s]", err.ProcessName)),
					errorStyle.Render(err.Content),
				))
			}
		}
	} else {
		// Styles for different parts
		errorTypeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		stackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
		
		// Show most recent error contexts first
		shown := 0
		for i := len(errorContexts) - 1; i >= 0 && shown < 10; i-- {
			errorCtx := errorContexts[i]
			
			// Error header
			content.WriteString(fmt.Sprintf("%s %s %s\n",
				timeStyle.Render(errorCtx.Timestamp.Format("15:04:05")),
				processStyle.Render(fmt.Sprintf("[%s]", errorCtx.ProcessName)),
				errorTypeStyle.Render(errorCtx.Type),
			))
			
			// Main error message
			content.WriteString(messageStyle.Render(errorCtx.Message) + "\n")
			
			// Stack trace if available
			if len(errorCtx.Stack) > 0 {
				content.WriteString(stackStyle.Render("Stack Trace:") + "\n")
				for j, stackLine := range errorCtx.Stack {
					if j > 5 { // Limit stack trace lines
						content.WriteString(stackStyle.Render(fmt.Sprintf("  ... and %d more lines", len(errorCtx.Stack)-j)) + "\n")
						break
					}
					content.WriteString(stackStyle.Render("  " + strings.TrimSpace(stackLine)) + "\n")
				}
			}
			
			// Additional context if available
			if len(errorCtx.Context) > 0 && len(errorCtx.Context) <= 5 {
				for _, ctxLine := range errorCtx.Context {
					if strings.TrimSpace(ctxLine) != "" {
						content.WriteString(contextStyle.Render("  " + strings.TrimSpace(ctxLine)) + "\n")
					}
				}
			}
			
			// Separator between errors
			content.WriteString(separatorStyle.Render("─────────────────────────────────────────") + "\n\n")
			shown++
		}
	}
	
	m.errorsViewport.SetContent(content.String())
	return m.errorsViewport.View()
}

func (m *Model) renderURLsView() string {
	urls := m.logStore.GetURLs()
	
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Detected URLs") + "\n")
	
	
	
	if len(urls) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No URLs detected yet"))
	} else {
		urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Underline(true)
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		
		// URLs are already deduplicated and sorted by the store
		for _, url := range urls {
			// No longer adding browser extension tokens
			urlWithToken := url.URL
			
			content.WriteString(fmt.Sprintf("%s %s\n%s\n",
				timeStyle.Render(url.Timestamp.Format("15:04:05")),
				processStyle.Render(fmt.Sprintf("[%s]", url.ProcessName)),
				urlStyle.Render(urlWithToken),
			))
			
			// Add connection info
			connectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
			content.WriteString(connectionStyle.Render(fmt.Sprintf("  🔗 process=%s time=%d\n", 
				url.ProcessName, url.Timestamp.Unix())))
			
			// Show context if it's not too long
			contextLen := len(url.Context)
			if contextLen > 80 {
				context := url.Context[:40] + "..." + url.Context[contextLen-37:]
				content.WriteString(contextStyle.Render("  → " + context) + "\n")
			} else {
				content.WriteString(contextStyle.Render("  → " + url.Context) + "\n")
			}
			content.WriteString("\n")
		}
	}
	
	m.urlsViewport.SetContent(content.String())
	return m.urlsViewport.View()
}

type processUpdateMsg struct{}
type logUpdateMsg struct{}
type errorUpdateMsg struct{}
type tickMsg struct{}

func (m Model) waitForUpdates() tea.Cmd {
	return func() tea.Msg {
		return <-m.updateChan
	}
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *Model) updateSettingsList() {
	installedMgrs := m.processMgr.GetInstalledPackageManagers()
	currentMgr := m.processMgr.GetCurrentPackageManager()
	
	items := make([]list.Item, 0)
	
	// Package Manager section
	items = append(items, settingsSectionItem{title: "Package Managers"})
	for _, mgr := range installedMgrs {
		item := packageManagerSettingsItem{packageManagerItem{
			manager:  mgr,
			current:  mgr.Manager == currentMgr,
			fromJSON: false, // TODO: Add method to check if from package.json
		}}
		items = append(items, item)
	}
	
	// MCP Integration section
	items = append(items, settingsSectionItem{title: "MCP Integration"})
	mcpTools := mcp.GetSupportedTools()
	installedTools := mcp.GetInstalledTools()
	installedSet := make(map[string]bool)
	for _, tool := range installedTools {
		installedSet[tool] = true
	}
	
	for _, tool := range mcpTools {
		if tool.Supported {
			item := mcpInstallItem{
				tool:      tool,
				installed: installedSet[tool.Name],
			}
			items = append(items, item)
		}
	}
	
	// Add custom file browser option
	items = append(items, mcpFileBrowserItem{})
	
	m.settingsList.SetItems(items)
}

func (m *Model) installMCPForTool(tool mcp.Tool) {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Error getting executable path: %v", err), true)
		return
	}
	
	// Generate config
	config := mcp.GenerateBrummerConfig(execPath, 7777)
	
	// Install
	if err := mcp.InstallForTool(tool, config); err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Error installing MCP for %s: %v", tool.Name, err), true)
	} else {
		m.logStore.Add("system", "System", fmt.Sprintf("Successfully configured %s with Brummer!", tool.Name), false)
		m.updateSettingsList()
	}
}

func getCurrentDir() string {
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		return homeDir
	}
	return "/"
}

func (m *Model) loadFileList() {
	entries, err := os.ReadDir(m.currentPath)
	if err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Error reading directory: %v", err), true)
		return
	}

	m.fileList = []FileItem{}
	
	// Add parent directory if not root
	if m.currentPath != "/" && m.currentPath != filepath.Dir(m.currentPath) {
		m.fileList = append(m.fileList, FileItem{
			Name:  "..",
			Path:  filepath.Dir(m.currentPath),
			IsDir: true,
		})
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := FileItem{
			Name:  entry.Name(),
			Path:  filepath.Join(m.currentPath, entry.Name()),
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		}
		m.fileList = append(m.fileList, item)
	}
}

func (m *Model) handleFileBrowser(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keys.Back):
			m.showingFileBrowser = false
			return nil

		case key.Matches(keyMsg, m.keys.Enter):
			if len(m.fileList) > 0 {
				// Simple selection - just use the first item for now
				// In a real implementation, you'd track the selected index
				selectedIndex := 0 // This should be tracked properly
				if selectedIndex < len(m.fileList) {
					item := m.fileList[selectedIndex]
					if item.IsDir {
						m.currentPath = item.Path
						m.loadFileList()
					} else if strings.HasSuffix(strings.ToLower(item.Name), ".json") {
						m.installMCPToFile(item.Path)
						m.showingFileBrowser = false
					}
				}
			}
		}
	}
	return nil
}

func (m *Model) renderFileBrowser() string {
	var content strings.Builder
	
	content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")).Render("Select Config File") + "\n\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Current Path: " + m.currentPath) + "\n\n")
	
	if len(m.fileList) == 0 {
		content.WriteString("No files found\n")
	} else {
		for i, item := range m.fileList {
			style := lipgloss.NewStyle()
			if i == 0 { // Simple selection highlight
				style = style.Background(lipgloss.Color("240"))
			}
			
			if item.IsDir {
				content.WriteString(style.Render("📁 " + item.Name) + "\n")
			} else if strings.HasSuffix(strings.ToLower(item.Name), ".json") {
				content.WriteString(style.Render("📄 " + item.Name + " (JSON)") + "\n")
			} else {
				content.WriteString(style.Foreground(lipgloss.Color("245")).Render("📄 " + item.Name) + "\n")
			}
		}
	}
	
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Enter: Select | Esc: Back"))
	
	return content.String()
}

func (m *Model) installMCPToFile(filePath string) {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Error getting executable path: %v", err), true)
		return
	}
	
	// Generate config
	config := mcp.GenerateBrummerConfig(execPath, 7777)
	
	// Read existing file
	data, err := os.ReadFile(filePath)
	if err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Error reading file %s: %v", filePath, err), true)
		return
	}
	
	var existingData map[string]interface{}
	if err := json.Unmarshal(data, &existingData); err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Error parsing JSON in %s: %v", filePath, err), true)
		return
	}
	
	// Try common MCP config formats
	if existingData["mcpServers"] == nil {
		existingData["mcpServers"] = make(map[string]interface{})
	}
	
	servers := existingData["mcpServers"].(map[string]interface{})
	servers["brummer"] = map[string]interface{}{
		"command": config.Command,
		"args":    config.Args,
		"env":     config.Env,
	}
	
	// Write back
	newData, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Error marshaling JSON: %v", err), true)
		return
	}
	
	if err := os.WriteFile(filePath, newData, 0644); err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Error writing to %s: %v", filePath, err), true)
		return
	}
	
	m.logStore.Add("system", "System", fmt.Sprintf("Successfully configured %s with Brummer!", filePath), false)
}

func (m *Model) handleRestartProcess(proc *process.Process) tea.Cmd {
	return func() tea.Msg {
		// Clear logs and errors before restarting
		m.logStore.ClearLogs()
		m.logStore.ClearErrors()
		
		// Stop the process first
		if err := m.processMgr.StopProcess(proc.ID); err != nil {
			m.logStore.Add("system", "System", fmt.Sprintf("Error stopping process %s: %v", proc.Name, err), true)
			return logUpdateMsg{}
		}
		
		// Start it again
		_, err := m.processMgr.StartScript(proc.Name)
		if err != nil {
			m.logStore.Add("system", "System", fmt.Sprintf("Error restarting script %s: %v", proc.Name, err), true)
		} else {
			m.logStore.Add("system", "System", fmt.Sprintf("🔄 Restarted process: %s (logs cleared)", proc.Name), false)
		}
		return processUpdateMsg{}
	}
}

func (m *Model) handleRestartAll() tea.Cmd {
	return func() tea.Msg {
		// Clear logs and errors before restarting all
		m.logStore.ClearLogs()
		m.logStore.ClearErrors()
		
		processes := m.processMgr.GetAllProcesses()
		restarted := 0
		
		for _, proc := range processes {
			if proc.Status == process.StatusRunning {
				// Stop the process
				if err := m.processMgr.StopProcess(proc.ID); err != nil {
					m.logStore.Add("system", "System", fmt.Sprintf("Error stopping process %s: %v", proc.Name, err), true)
					continue
				}
				
				// Start it again
				_, err := m.processMgr.StartScript(proc.Name)
				if err != nil {
					m.logStore.Add("system", "System", fmt.Sprintf("Error restarting script %s: %v", proc.Name, err), true)
				} else {
					restarted++
				}
			}
		}
		
		m.logStore.Add("system", "System", fmt.Sprintf("🔄 Restarted %d processes (logs cleared)", restarted), false)
		return processUpdateMsg{}
	}
}

func (m *Model) handleCopyError() tea.Cmd {
	return func() tea.Msg {
		// Try to get error contexts first
		errorContexts := m.logStore.GetErrorContexts()
		
		var errorText string
		
		if len(errorContexts) > 0 {
			// Get the most recent error context
			recentError := errorContexts[len(errorContexts)-1]
			
			// Build comprehensive error text
			var errorBuilder strings.Builder
			errorBuilder.WriteString(fmt.Sprintf("Error Type: %s\n", recentError.Type))
			errorBuilder.WriteString(fmt.Sprintf("Time: %s\n", recentError.Timestamp.Format("15:04:05")))
			errorBuilder.WriteString(fmt.Sprintf("Process: %s\n", recentError.ProcessName))
			errorBuilder.WriteString(fmt.Sprintf("Message: %s\n", recentError.Message))
			
			if len(recentError.Stack) > 0 {
				errorBuilder.WriteString("\nStack Trace:\n")
				for _, line := range recentError.Stack {
					errorBuilder.WriteString("  " + line + "\n")
				}
			}
			
			if len(recentError.Context) > 0 {
				errorBuilder.WriteString("\nAdditional Context:\n")
				for _, line := range recentError.Context {
					errorBuilder.WriteString("  " + line + "\n")
				}
			}
			
			errorText = errorBuilder.String()
		} else {
			// Fall back to simple errors
			errors := m.logStore.GetErrors()
			if len(errors) == 0 {
				m.logStore.Add("system", "System", "No recent errors to copy", false)
				return logUpdateMsg{}
			}
			
			// Get the most recent error
			recentError := errors[len(errors)-1]
			errorText = fmt.Sprintf("[%s] %s: %s", 
				recentError.Timestamp.Format("15:04:05"), 
				recentError.ProcessName, 
				recentError.Content)
		}
		
		// Try to copy to system clipboard
		if err := copyToClipboard(errorText); err != nil {
			m.logStore.Add("system", "System", fmt.Sprintf("Failed to copy to clipboard: %v", err), true)
		} else {
			m.copyNotification = "📋 Error copied to clipboard"
			m.notificationTime = time.Now()
		}
		
		return logUpdateMsg{}
	}
}

func renderExitScreen() string {
	bee := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Render(`
    ╭─╮
   ╱   ╲
  ╱ ● ● ╲    🐝 Thanks for using Brummer!
 ╱   ◡   ╲   
╱  ╲   ╱  ╲   Happy scripting! 
╲   ╲ ╱   ╱  
 ╲   ╱   ╱
  ╲ ─── ╱
   ╲___╱

`)
	return bee
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel
		if exec.Command("which", "xclip").Run() == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if exec.Command("which", "xsel").Run() == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (install xclip or xsel)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func (m *Model) handleClearLogs() {
	m.logStore.ClearLogs()
	m.logStore.Add("system", "System", "📝 Logs cleared", false)
	m.updateLogsView()
}

func (m *Model) handleClearErrors() {
	m.logStore.ClearErrors()
	m.logStore.Add("system", "System", "🗑️ Error history cleared", false)
	m.updateLogsView()
}

func (m *Model) showRunDialog() {
	m.showingRunDialog = true
	
	// Get detected commands
	m.detectedCommands = m.processMgr.GetDetectedCommands()
	
	// Get monorepo info
	m.monorepoInfo, _ = m.processMgr.GetMonorepoInfo()
	
	// Build command list items
	items := make([]list.Item, 0)
	
	// Add detected commands sorted by priority
	for _, cmd := range m.detectedCommands {
		items = append(items, commandItem{command: cmd})
	}
	
	// Add monorepo commands if detected
	if m.monorepoInfo != nil {
		for _, pkg := range m.monorepoInfo.Packages {
			for scriptName, script := range pkg.Scripts {
				items = append(items, commandItem{
					command: parser.ExecutableCommand{
						Name:        fmt.Sprintf("%s: %s", pkg.Name, scriptName),
						Command:     "npm",
						Args:        []string{"run", scriptName, "--workspace", pkg.Name},
						Description: script,
						Category:    "Monorepo",
						ProjectType: parser.ProjectTypeMonorepo,
						Priority:    80,
					},
				})
			}
		}
	}
	
	// Add custom command option at the end
	items = append(items, runCustomItem{})
	
	m.commandsList.SetItems(items)
}

func (m *Model) handleRunCommand() tea.Cmd {
	if !m.showingRunDialog {
		return nil
	}
	
	selected := m.commandsList.SelectedItem()
	if selected == nil {
		return nil
	}
	
	m.showingRunDialog = false
	
	switch item := selected.(type) {
	case commandItem:
		go func() {
			_, err := m.processMgr.StartCommand(item.command.Name, item.command.Command, item.command.Args)
			if err != nil {
				m.logStore.Add("system", "System", fmt.Sprintf("Error starting command %s: %v", item.command.Name, err), true)
				m.updateChan <- logUpdateMsg{}
			}
		}()
		m.currentView = ViewProcesses
		m.updateProcessList()
		return m.waitForUpdates()
		
	case runCustomItem:
		// TODO: Show custom command input dialog
		m.logStore.Add("system", "System", "Custom command dialog not yet implemented", true)
		return nil
	}
	
	return nil
}

func (m Model) renderScriptsView() string {
	var content strings.Builder
	
	// Add instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("Select script: ↑/↓ | Run: Enter | Run Other Command: n | Switch View: Tab")
	
	content.WriteString(instructions)
	content.WriteString("\n\n")
	
	// Show monorepo info if detected
	if m.monorepoInfo != nil {
		monoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
		content.WriteString(monoStyle.Render(fmt.Sprintf("📦 %s Monorepo Detected", m.monorepoInfo.Type)))
		content.WriteString("\n")
		
		pkgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		content.WriteString(pkgStyle.Render(fmt.Sprintf("Found %d packages in workspaces", len(m.monorepoInfo.Packages))))
		content.WriteString("\n\n")
	}
	
	content.WriteString(m.scriptsList.View())
	
	return content.String()
}

func (m Model) renderRunDialog() string {
	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	content.WriteString(titleStyle.Render("🚀 Run Command"))
	content.WriteString("\n\n")
	
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("Select command: ↑/↓ | Run: Enter | Cancel: Esc")
	
	content.WriteString(instructions)
	content.WriteString("\n\n")
	
	// Show monorepo info if detected
	if m.monorepoInfo != nil {
		monoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		content.WriteString(monoStyle.Render(fmt.Sprintf("📦 Monorepo: %s with %d packages", m.monorepoInfo.Type, len(m.monorepoInfo.Packages))))
		content.WriteString("\n\n")
	}
	
	content.WriteString(m.commandsList.View())
	
	return content.String()
}

func (m *Model) updateErrorsList() {
	errorContexts := m.logStore.GetErrorContexts()
	
	items := make([]list.Item, 0, len(errorContexts))
	for i := len(errorContexts) - 1; i >= 0; i-- {
		items = append(items, errorItem{errorCtx: &errorContexts[i]})
	}
	
	m.errorsList.SetItems(items)
	
	// Select first item if we have errors and nothing selected
	if len(items) > 0 && m.selectedError == nil {
		if item, ok := items[0].(errorItem); ok {
			m.selectedError = item.errorCtx
			m.updateErrorDetailView()
		}
	}
}

func (m *Model) updateErrorDetailView() {
	if m.selectedError == nil {
		m.errorDetailView.SetContent("Select an error to view details")
		return
	}
	
	var content strings.Builder
	
	// Header with error type and severity
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	content.WriteString(headerStyle.Render(fmt.Sprintf("%s Error", m.selectedError.Type)))
	content.WriteString("\n\n")
	
	// Error info
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	content.WriteString(infoStyle.Render(fmt.Sprintf("Time: %s | Process: %s | Language: %s",
		m.selectedError.Timestamp.Format("15:04:05"),
		m.selectedError.ProcessName,
		m.selectedError.Language)))
	content.WriteString("\n\n")
	
	// Main error message
	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	content.WriteString(messageStyle.Render("Error Message:"))
	content.WriteString("\n")
	content.WriteString(m.selectedError.Message)
	content.WriteString("\n\n")
	
	// Find the lowest level code reference
	if codeRef := m.findLowestCodeReference(m.selectedError); codeRef != "" {
		codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
		content.WriteString(codeStyle.Render("📍 Code Location:"))
		content.WriteString("\n")
		content.WriteString(codeRef)
		content.WriteString("\n\n")
	}
	
	// Stack trace
	if len(m.selectedError.Stack) > 0 {
		stackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Bold(true)
		content.WriteString(stackStyle.Render("Stack Trace:"))
		content.WriteString("\n")
		stackLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		for _, line := range m.selectedError.Stack {
			content.WriteString(stackLineStyle.Render("  " + strings.TrimSpace(line)))
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}
	
	// Additional context
	if len(m.selectedError.Context) > 0 {
		contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
		content.WriteString(contextStyle.Render("Additional Context:"))
		content.WriteString("\n")
		contextLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		for _, line := range m.selectedError.Context {
			if strings.TrimSpace(line) != "" {
				content.WriteString(contextLineStyle.Render("  " + strings.TrimSpace(line)))
				content.WriteString("\n")
			}
		}
		content.WriteString("\n")
	}
	
	// Raw log lines (collapsed by default)
	if len(m.selectedError.Raw) > 0 {
		rawStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Bold(true)
		content.WriteString(rawStyle.Render("Raw Log Output:"))
		content.WriteString("\n")
		rawLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
		for _, line := range m.selectedError.Raw {
			content.WriteString(rawLineStyle.Render(line))
			content.WriteString("\n")
		}
	}
	
	m.errorDetailView.SetContent(content.String())
}

func (m Model) findLowestCodeReference(errorCtx *logs.ErrorContext) string {
	// Look for file paths with line numbers in stack traces
	filePattern := regexp.MustCompile(`([^\s\(\)]+\.(js|ts|jsx|tsx|go|py|java|rs|rb|php)):(\d+)(?::(\d+))?`)
	
	var lowestRef string
	var lowestInProject bool
	
	// Check stack traces first
	for _, line := range errorCtx.Stack {
		if matches := filePattern.FindStringSubmatch(line); matches != nil {
			filePath := matches[1]
			lineNum := matches[3]
			colNum := ""
			if len(matches) > 4 {
				colNum = matches[4]
			}
			
			// Prioritize project files over node_modules
			isProjectFile := !strings.Contains(filePath, "node_modules") && 
				!strings.Contains(filePath, "/usr/") &&
				!strings.Contains(filePath, "\\Windows\\")
			
			if lowestRef == "" || (isProjectFile && !lowestInProject) {
				if colNum != "" {
					lowestRef = fmt.Sprintf("%s:%s:%s", filePath, lineNum, colNum)
				} else {
					lowestRef = fmt.Sprintf("%s:%s", filePath, lineNum)
				}
				lowestInProject = isProjectFile
			}
		}
	}
	
	// If no stack trace refs, check context
	if lowestRef == "" {
		for _, line := range errorCtx.Context {
			if matches := filePattern.FindStringSubmatch(line); matches != nil {
				filePath := matches[1]
				lineNum := matches[3]
				lowestRef = fmt.Sprintf("%s:%s", filePath, lineNum)
				break
			}
		}
	}
	
	return lowestRef
}

func (m Model) renderErrorsViewSplit() string {
	if m.width < 100 {
		// For narrow screens, use the old view
		return m.renderErrorsView()
	}
	
	// Update the errors list
	errorContexts := m.logStore.GetErrorContexts()
	if len(errorContexts) == 0 {
		return m.renderErrorsView()
	}
	
	// Calculate split dimensions
	listWidth := m.width / 3
	detailWidth := m.width - listWidth - 3 // -3 for border and padding
	contentHeight := m.height - 5 // Adjust for header
	
	// Update sizes
	m.errorsList.SetSize(listWidth, contentHeight)
	m.errorDetailView.Width = detailWidth
	m.errorDetailView.Height = contentHeight
	
	// Update list items if needed
	currentItems := m.errorsList.Items()
	if len(currentItems) != len(errorContexts) {
		const_m := &m
		const_m.updateErrorsList()
	}
	
	// Create border styles
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
	
	// Render list and detail side by side
	listView := borderStyle.
		Width(listWidth).
		Height(contentHeight).
		Render(m.errorsList.View())
	
	detailView := borderStyle.
		Width(detailWidth).
		Height(contentHeight).
		Render(m.errorDetailView.View())
	
	return lipgloss.JoinHorizontal(lipgloss.Top, listView, " ", detailView)
}