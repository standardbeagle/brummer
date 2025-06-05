package tui

import (
	"fmt"
	"strings"

	"github.com/beagle/beagle-run/internal/logs"
	"github.com/beagle/beagle-run/internal/parser"
	"github.com/beagle/beagle-run/internal/process"
	"github.com/beagle/beagle-run/pkg/events"
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
	
	help         help.Model
	keys         keyMap
	
	updateChan   chan tea.Msg
}

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Tab      key.Binding
	Search   key.Binding
	Filter   key.Binding
	Stop     key.Binding
	Priority key.Binding
	Help     key.Binding
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
		{k.Priority, k.Help, k.Quit},
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
			return m, tea.Quit

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

		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Stop) {
			if i, ok := m.processesList.SelectedItem().(processItem); ok {
				m.processMgr.StopProcess(i.process.ID)
			}
		}

	case ViewLogs, ViewErrors, ViewURLs:
		newViewport, cmd := m.logsViewport.Update(msg)
		m.logsViewport = newViewport
		cmds = append(cmds, cmd)

	case ViewSettings:
		newList, cmd := m.settingsList.Update(msg)
		m.settingsList = newList
		cmds = append(cmds, cmd)

		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Enter) {
			if i, ok := m.settingsList.SelectedItem().(packageManagerItem); ok {
				if err := m.processMgr.SetUserPackageManager(i.manager.Manager); err != nil {
					// Log error but don't crash
					m.logStore.Add("system", "System", fmt.Sprintf("Error saving preference: %v", err), true)
				}
				m.updateSettingsList()
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
		content = m.settingsList.View()
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
		Foreground(lipgloss.Color("86")).
		Render("🚀 Beagle Run")

	tabs := []string{}
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	for _, v := range []View{ViewScripts, ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewSettings} {
		label := string(v)
		if v == m.currentView {
			tabs = append(tabs, activeStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveStyle.Render(label))
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
	
	items := make([]list.Item, 0, len(installedMgrs))
	
	for _, mgr := range installedMgrs {
		item := packageManagerItem{
			manager:  mgr,
			current:  mgr.Manager == currentMgr,
			fromJSON: false, // TODO: Add method to check if from package.json
		}
		items = append(items, item)
	}
	
	m.settingsList.SetItems(items)
}