package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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
	
	currentView  View
	width        int
	height       int
	
	scriptsList     list.Model
	processesList   list.Model
	logsViewport    viewport.Model
	errorsViewport  viewport.Model
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
	
	help         help.Model
	keys         keyMap
	
	updateChan   chan tea.Msg
}

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Back       key.Binding
	Quit       key.Binding
	Tab        key.Binding
	Search     key.Binding
	Filter     key.Binding
	Stop       key.Binding
	Restart    key.Binding
	RestartAll key.Binding
	CopyError  key.Binding
	Priority   key.Binding
	Help       key.Binding
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
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Enter, k.Back, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Tab, k.Search, k.Filter, k.Stop},
		{k.Restart, k.RestartAll, k.CopyError, k.Priority},
		{k.Help, k.Quit},
	}
}

type scriptItem struct {
	name   string
	script string
}

func (i scriptItem) FilterValue() string { return i.name }
func (i scriptItem) Title() string       { return i.name }
func (i scriptItem) Description() string { return i.script }

type processItem struct {
	process *process.Process
}

func (i processItem) FilterValue() string { return i.process.Name }
func (i processItem) Title() string {
	status := string(i.process.Status)
	return fmt.Sprintf("[%s] %s", status, i.process.Name)
}
func (i processItem) Description() string {
	return fmt.Sprintf("PID: %s | Started: %s", i.process.ID, i.process.StartTime.Format("15:04:05"))
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

func NewModel(processMgr *process.Manager, logStore *logs.Store, eventBus *events.EventBus) Model {
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
		currentView:    ViewScripts,
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

	processMgr.RegisterLogCallback(func(processID, line string, isError bool) {
		if proc, exists := processMgr.GetProcess(processID); exists {
			logStore.Add(processID, proc.Name, line, isError)
		}
	})

	// Initialize settings list
	m.updateSettingsList()

	return m
}

func (m Model) Init() tea.Cmd {
	// Set up event subscriptions
	m.eventBus.Subscribe(events.ProcessStarted, func(e events.Event) {
		m.updateChan <- processUpdateMsg{}
	})
	
	m.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		m.updateChan <- processUpdateMsg{}
	})
	
	m.eventBus.Subscribe(events.LogLine, func(e events.Event) {
		m.updateChan <- logUpdateMsg{}
	})

	return tea.Batch(
		textinput.Blink,
		m.waitForUpdates(),
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
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Sequence(
				tea.Printf(renderExitScreen()),
				tea.Quit,
			)

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
			cmds = append(cmds, m.handleRestartAll())

		case key.Matches(msg, m.keys.CopyError):
			cmds = append(cmds, m.handleCopyError())

		case key.Matches(msg, m.keys.Enter):
			cmds = append(cmds, m.handleEnter())
		}

	case processUpdateMsg:
		m.updateProcessList()
		cmds = append(cmds, m.waitForUpdates())

	case logUpdateMsg:
		m.updateLogsView()
		cmds = append(cmds, m.waitForUpdates())
	}

	switch m.currentView {
	case ViewScripts:
		newList, cmd := m.scriptsList.Update(msg)
		m.scriptsList = newList
		cmds = append(cmds, cmd)

	case ViewProcesses:
		newList, cmd := m.processesList.Update(msg)
		m.processesList = newList
		cmds = append(cmds, cmd)

		if msg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, m.keys.Stop) {
				if i, ok := m.processesList.SelectedItem().(processItem); ok {
					m.processMgr.StopProcess(i.process.ID)
				}
			} else if key.Matches(msg, m.keys.Restart) {
				if i, ok := m.processesList.SelectedItem().(processItem); ok {
					cmds = append(cmds, m.handleRestartProcess(i.process))
				}
			}
		}

	case ViewLogs, ViewErrors, ViewURLs:
		newViewport, cmd := m.logsViewport.Update(msg)
		m.logsViewport = newViewport
		cmds = append(cmds, cmd)

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
	
	switch m.currentView {
	case ViewScripts:
		content = m.scriptsList.View()
	case ViewProcesses:
		content = m.processesList.View()
	case ViewLogs:
		content = m.renderLogsView()
	case ViewErrors:
		content = m.renderErrorsView()
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

	helpView := m.help.View(m.keys)
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(),
		content,
		helpView,
	)
}

func (m *Model) updateSizes() {
	headerHeight := 3
	helpHeight := 3
	contentHeight := m.height - headerHeight - helpHeight

	m.scriptsList.SetSize(m.width, contentHeight)
	m.processesList.SetSize(m.width, contentHeight)
	m.settingsList.SetSize(m.width, contentHeight)
	m.logsViewport.Width = m.width
	m.logsViewport.Height = contentHeight
	m.errorsViewport.Width = m.width
	m.errorsViewport.Height = contentHeight
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
	items := make([]list.Item, len(processes))
	for i, p := range processes {
		items[i] = processItem{process: p}
	}
	m.processesList.SetItems(items)
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
		style := m.getLogStyle(log)
		line := fmt.Sprintf("[%s] %s: %s\n", 
			log.Timestamp.Format("15:04:05"),
			log.ProcessName,
			log.Content,
		)
		content.WriteString(style.Render(line))
	}

	m.logsViewport.SetContent(content.String())
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
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Render("🐝 Brummer - Package Script Manager")

	tabs := []string{}
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	for _, v := range []View{ViewScripts, ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewSettings} {
		label := string(v)
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
		"",
		tabBar,
		strings.Repeat("─", m.width),
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
	errors := m.logStore.GetErrors()
	
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent Errors") + "\n\n")
	
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
	
	m.errorsViewport.SetContent(content.String())
	return m.errorsViewport.View()
}

func (m *Model) renderURLsView() string {
	urls := m.logStore.GetURLs()
	
	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Detected URLs") + "\n\n")
	
	if len(urls) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No URLs detected yet"))
	} else {
		urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Underline(true)
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		
		// Show most recent URLs first
		seen := make(map[string]bool)
		for i := len(urls) - 1; i >= 0; i-- {
			url := urls[i]
			// Deduplicate URLs
			if seen[url.URL] {
				continue
			}
			seen[url.URL] = true
			
			content.WriteString(fmt.Sprintf("%s %s\n%s\n",
				timeStyle.Render(url.Timestamp.Format("15:04:05")),
				processStyle.Render(fmt.Sprintf("[%s]", url.ProcessName)),
				urlStyle.Render(url.URL),
			))
			
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

func (m Model) waitForUpdates() tea.Cmd {
	return func() tea.Msg {
		return <-m.updateChan
	}
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
			m.logStore.Add("system", "System", fmt.Sprintf("Restarted process: %s", proc.Name), false)
		}
		return processUpdateMsg{}
	}
}

func (m *Model) handleRestartAll() tea.Cmd {
	return func() tea.Msg {
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
		
		m.logStore.Add("system", "System", fmt.Sprintf("Restarted %d processes", restarted), false)
		return processUpdateMsg{}
	}
}

func (m *Model) handleCopyError() tea.Cmd {
	return func() tea.Msg {
		errors := m.logStore.GetErrors()
		if len(errors) == 0 {
			m.logStore.Add("system", "System", "No recent errors to copy", false)
			return logUpdateMsg{}
		}
		
		// Get the most recent error
		recentError := errors[len(errors)-1]
		errorText := fmt.Sprintf("[%s] %s: %s", 
			recentError.Timestamp.Format("15:04:05"), 
			recentError.ProcessName, 
			recentError.Content)
		
		// Try to copy to system clipboard
		if err := copyToClipboard(errorText); err != nil {
			m.logStore.Add("system", "System", fmt.Sprintf("Failed to copy to clipboard: %v", err), true)
		} else {
			m.logStore.Add("system", "System", "📋 Error copied to clipboard", false)
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