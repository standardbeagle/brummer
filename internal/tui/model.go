package tui

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/parser"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/commands"
	"github.com/standardbeagle/brummer/internal/tui/filebrowser"
	"github.com/standardbeagle/brummer/internal/tui/navigation"
	"github.com/standardbeagle/brummer/internal/tui/notifications"
	"github.com/standardbeagle/brummer/internal/tui/system"
	"github.com/standardbeagle/brummer/pkg/events"
)

type View = navigation.View

// Re-export navigation constants for convenience
const (
	ViewScripts        = navigation.ViewScripts
	ViewProcesses      = navigation.ViewProcesses
	ViewLogs           = navigation.ViewLogs
	ViewErrors         = navigation.ViewErrors
	ViewURLs           = navigation.ViewURLs
	ViewWeb            = navigation.ViewWeb
	ViewSettings       = navigation.ViewSettings
	ViewMCPConnections = navigation.ViewMCPConnections
	ViewSearch         = navigation.ViewSearch
	ViewFilters        = navigation.ViewFilters
	ViewScriptSelector = navigation.ViewScriptSelector
	ViewAICoders       = navigation.ViewAICoders
)

// ViewConfig holds configuration for each view
type ViewConfig struct {
	Title       string
	Description string
	KeyBinding  string
	Icon        string
}

// viewConfigs defines the configuration for each view
var viewConfigs = map[View]ViewConfig{
	ViewProcesses: {
		Title:       "Processes",
		Description: "Process management",
		KeyBinding:  "1",
		Icon:        "🏃",
	},
	ViewLogs: {
		Title:       "Logs",
		Description: "Process logs",
		KeyBinding:  "2",
		Icon:        "📄",
	},
	ViewErrors: {
		Title:       "Errors",
		Description: "Error tracking",
		KeyBinding:  "3",
		Icon:        "❌",
	},
	ViewURLs: {
		Title:       "URLs",
		Description: "Detected URLs",
		KeyBinding:  "4",
		Icon:        "🔗",
	},
	ViewWeb: {
		Title:       "Web",
		Description: "Proxy requests",
		KeyBinding:  "5",
		Icon:        "🌐",
	},
	ViewAICoders: {
		Title:       "AI Coders",
		Description: "Manage and monitor agentic AI coding assistants",
		KeyBinding:  "6",
		Icon:        "🤖",
	},
	ViewSettings: {
		Title:       "Settings",
		Description: "Configuration",
		KeyBinding:  "7",
		Icon:        "⚙️",
	},
	ViewMCPConnections: {
		Title:       "MCP",
		Description: "MCP Connections",
		KeyBinding:  "8",
		Icon:        "🔌",
	},
}

// MCPServerInterface defines the methods needed by the TUI
type MCPServerInterface interface {
	IsRunning() bool
	GetPort() int
}

type Model struct {
	processMgr  *process.Manager
	logStore    *logs.Store
	eventBus    *events.EventBus
	mcpServer   MCPServerInterface
	mcpPort     int
	proxyServer *proxy.Server
	debugMode   bool

	// Navigation controller
	navController *navigation.Controller
	width         int
	height        int

	// Controllers for different views
	processViewController   *ProcessViewController   // New controller for processes view
	logsViewController      *LogsViewController      // New controller for logs view
	errorsViewController    *ErrorsViewController    // New controller for errors view
	urlsViewController      *URLsViewController      // New controller for URLs view
	webViewController       *WebViewController       // New controller for web view
	commandWindowController *CommandWindowController // New controller for command windows
	settingsController      *SettingsController      // New controller for settings view
	layoutController        *LayoutController        // New controller for layout rendering
	inputController         *InputController         // New controller for input handling
	eventController         *EventController         // New controller for event handling

	// System message controller
	systemController      *system.Controller
	systemPanelRenderer   *system.PanelRenderer
	systemOverlayRenderer *system.OverlayRenderer

	selectedProcess string // Shared state - used by multiple views

	// File browser controller
	fileBrowserController *filebrowser.Controller

	// Notifications controller
	notificationsController *notifications.Controller

	// Script selector controller
	scriptSelectorController *ScriptSelectorController

	// Layout components
	tabsComponent *TabsComponent
	contentLayout *ContentLayout
	headerHeight  int // Calculated height of the header
	footerHeight  int // Calculated height of the footer

	keys keyMap

	updateChan chan tea.Msg

	// MCP debug controller
	mcpDebugController *MCPDebugController // New controller for MCP debug view

	// AI Coder controller
	aiCoderController *AICoderController // New controller for AI Coder functionality

	// Message routing
	messageRouter *MessageRouter // Routes messages to appropriate handlers
}

type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	Back          key.Binding
	Quit          key.Binding
	Tab           key.Binding
	Command       key.Binding
	Filter        key.Binding
	Stop          key.Binding
	Restart       key.Binding
	RestartAll    key.Binding
	CopyError     key.Binding
	Priority      key.Binding
	ClearLogs     key.Binding
	ClearErrors   key.Binding
	ClearScreen   key.Binding
	Help          key.Binding
	RunDialog     key.Binding
	AutoScroll    key.Binding
	ToggleError   key.Binding
	ClearMessages key.Binding
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
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch view"),
	),
	Command: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "command palette"),
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
	ClearScreen: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear screen"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	RunDialog: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new process"),
	),
	AutoScroll: key.NewBinding(
		key.WithKeys("end"),
		key.WithHelp("end", "auto-scroll"),
	),
	ToggleError: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "toggle system messages"),
	),
	ClearMessages: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "clear system messages"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Enter, k.Back, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Tab, k.Command, k.Filter, k.Priority},
		{k.Stop, k.Restart, k.RestartAll, k.CopyError},
		{k.ClearLogs, k.ClearErrors, k.ToggleError, k.ClearMessages, k.Help, k.Quit},
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

type processItem struct {
	process    *process.Process
	isHeader   bool
	headerText string
}

func (i processItem) FilterValue() string {
	if i.isHeader || i.process == nil {
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

	// Safety check for nil process
	if i.process == nil {
		return ""
	}

	// Use ProcessState for atomic access to status and name
	state := i.process.GetStateAtomic()
	status := string(state.Status)
	var statusEmoji string
	switch state.Status {
	case process.StatusRunning:
		statusEmoji = "🟢"
	case process.StatusStopped:
		statusEmoji = "✓" // Thin checkmark for gracefully stopped
	case process.StatusFailed:
		statusEmoji = "❌"
	case process.StatusSuccess:
		statusEmoji = "✓" // Thin checkmark for success
	default:
		statusEmoji = "⏸️"
	}
	title := fmt.Sprintf("%s [%s] %s", statusEmoji, status, state.Name)

	return title
}
func (i processItem) Description() string {
	if i.isHeader {
		// Return separator line for headers using lipgloss
		separatorStyle := lipgloss.NewStyle().
			Width(40).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderBottom(false).
			BorderLeft(false).
			BorderRight(false).
			BorderForeground(lipgloss.Color("240"))
		return separatorStyle.Render("")
	}

	// Safety check for nil process
	if i.process == nil {
		return ""
	}

	var parts []string

	// Use ProcessState for atomic access to multiple fields
	state := i.process.GetStateAtomic()

	// Add PID and start time
	parts = append(parts, fmt.Sprintf("PID: %s", state.ID))
	parts = append(parts, fmt.Sprintf("Started: %s", state.StartTime.Format("15:04:05")))

	// Add exit code for failed/stopped processes
	if state.IsFinished() && state.ExitCode != nil {
		parts = append(parts, fmt.Sprintf("Exit: %d", *state.ExitCode))
	}

	// Add runtime if process has ended
	if state.EndTime != nil {
		runtime := state.EndTime.Sub(state.StartTime)
		parts = append(parts, fmt.Sprintf("Runtime: %s", runtime.Round(time.Millisecond).String()))
	}

	// Add actions
	var actions string
	if state.IsRunning() {
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
func (i settingsSectionItem) Title() string {
	// Add visual styling for section headers
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("238")).
		Padding(0, 1).
		Width(60)
	return style.Render(i.title)
}
func (i settingsSectionItem) Description() string { return "" }
func (i settingsSectionItem) isSettingsItem()     {}

type mcpFileBrowserItem struct{}

func (i mcpFileBrowserItem) FilterValue() string { return "custom file" }
func (i mcpFileBrowserItem) Title() string       { return "Browse for Custom Config..." }
func (i mcpFileBrowserItem) Description() string {
	return "Browse for a JSON config file to add Brummer"
}
func (i mcpFileBrowserItem) isSettingsItem() {}

type proxyInfoItem struct {
	pacURL string
	mode   proxy.ProxyMode
	port   int
}

func (i proxyInfoItem) FilterValue() string { return "proxy" }
func (i proxyInfoItem) Title() string {
	modeStr := "Full Proxy"
	if i.mode == proxy.ProxyModeReverse {
		modeStr = "Reverse Proxy"
	}
	return fmt.Sprintf("🌐 %s (Port %d)", modeStr, i.port)
}
func (i proxyInfoItem) Description() string {
	return fmt.Sprintf("📄 PAC URL: %s • Press Enter to copy", i.pacURL)
}
func (i proxyInfoItem) isSettingsItem() {}

type mcpServerInfoItem struct {
	port   int
	status string
}

func (i mcpServerInfoItem) FilterValue() string { return "mcp server" }
func (i mcpServerInfoItem) Title() string {
	return fmt.Sprintf("🔗 MCP Server (Port %d)", i.port)
}
func (i mcpServerInfoItem) Description() string {
	return fmt.Sprintf("Model Context Protocol server • %s • Multiple tools via single endpoint", i.status)
}
func (i mcpServerInfoItem) isSettingsItem() {}

type infoDisplayItem struct {
	title       string
	description string
	value       string
	copyable    bool
}

func (i infoDisplayItem) FilterValue() string { return i.title }
func (i infoDisplayItem) Title() string       { return i.title }
func (i infoDisplayItem) Description() string {
	if i.copyable {
		return i.description + " (Press Enter to copy)"
	}
	return i.description
}
func (i infoDisplayItem) isSettingsItem() {}

func NewModel(processMgr *process.Manager, logStore *logs.Store, eventBus *events.EventBus, mcpServer MCPServerInterface, proxyServer *proxy.Server, mcpPort int, cfg *config.Config) *Model {
	return NewModelWithView(processMgr, logStore, eventBus, mcpServer, proxyServer, mcpPort, ViewProcesses, false, cfg)
}

func NewModelWithView(processMgr *process.Manager, logStore *logs.Store, eventBus *events.EventBus, mcpServer MCPServerInterface, proxyServer *proxy.Server, mcpPort int, initialView View, debugMode bool, cfg *config.Config) *Model {
	scripts := processMgr.GetScripts()

	// processesList initialization moved to ProcessViewController

	// Create settings list with package managers
	// settingsList initialization moved to SettingsController

	m := Model{
		processMgr:  processMgr,
		logStore:    logStore,
		eventBus:    eventBus,
		mcpServer:   mcpServer,
		mcpPort:     mcpPort,
		proxyServer: proxyServer,
		debugMode:   debugMode,
		// processesList initialization moved to ProcessViewController
		processViewController: NewProcessViewController(processMgr),
		// settingsList initialization moved to SettingsController
		logsViewController:      NewLogsViewController(logStore),
		errorsViewController:    NewErrorsViewController(logStore),
		urlsViewController:      NewURLsViewController(logStore, mcpServer),
		webViewController:       NewWebViewController(proxyServer),
		commandWindowController: NewCommandWindowController(processMgr),
		keys:                    keys,
		updateChan:              make(chan tea.Msg, UpdateChannelBufferSize),
	}

	// Note: Log callback is registered in main.go to avoid duplication

	// INITIALIZATION ORDER IS CRITICAL - DO NOT CHANGE WITHOUT CAREFUL REVIEW
	// The following controllers have dependencies and must be initialized in this specific order:

	// 1. Initialize file browser controller first
	//    - Required by: SettingsController (for file selection dialogs)
	//    - Dependencies: None
	m.fileBrowserController = filebrowser.NewController()

	// 2. Initialize settings controller second
	//    - Required by: UpdateSettingsList() call below
	//    - Dependencies: fileBrowserController (passed as parameter)
	workingDir, _ := os.Getwd()
	m.settingsController = NewSettingsController(cfg, mcpServer, processMgr, workingDir, m.fileBrowserController, logStore, proxyServer)
	// settingsController now owns its own settingsList

	// 3. Initialize settings list
	//    - Required by: UI display
	//    - Dependencies: settingsController must be initialized
	m.settingsController.UpdateSettingsList()

	// 4. Initialize process list with current processes
	//    - Required by: Process view display
	//    - Dependencies: processViewController already initialized
	m.updateProcessList()

	// Error list and detail view now managed by ErrorsViewController

	// 5. Initialize navigation controller
	//    - Required by: View switching, unread indicators
	//    - Dependencies: None initially, but callbacks reference systemController (created next)
	m.navController = navigation.NewController(initialView, debugMode)

	// Note: These callbacks reference systemController which doesn't exist yet!
	// This works because the callbacks are stored but not executed until after
	// systemController is created in step 6
	m.navController.SetOnViewChange(func(from, to navigation.View) {
		// Clear unread indicators when switching views
		m.systemController.ClearUnreadIndicator(to)
	})

	// Set up additional navigation callbacks
	m.navController.SetOnClearUnreadIndicator(func(view View) {
		m.systemController.ClearUnreadIndicator(view)
	})
	m.navController.SetOnUpdateLogsView(func() {
		m.updateLogsView()
	})
	m.navController.SetOnUpdateMCPConnections(func() {
		m.mcpDebugController.UpdateConnectionsList()
	})

	// 6. Initialize system message controller
	//    - Required by: Navigation callbacks (step 5), system messages
	//    - Dependencies: None
	m.systemController = system.NewController(SystemPanelMaxMessages)
	m.systemPanelRenderer = system.NewPanelRenderer(m.systemController)
	m.systemOverlayRenderer = system.NewOverlayRenderer(m.systemController)

	// 7. Initialize layout controller
	//    - Required by: View rendering
	//    - Dependencies: All data sources (processMgr, logStore, mcpServer, proxyServer)
	version := "1.0.0" // TODO: Get from build info
	m.layoutController = NewLayoutController(processMgr, logStore, mcpServer, proxyServer, version, workingDir)

	// 8. Initialize input controller
	//    - Required by: Keyboard handling
	//    - Dependencies: Full model reference (uses &m)
	m.inputController = NewInputController(&m, keys, viewConfigs, debugMode)

	// 9. Initialize event controller
	//    - Required by: Event bus integration
	//    - Dependencies: Full model reference, eventBus, updateChan
	m.eventController = NewEventController(&m, eventBus, m.updateChan, debugMode)

	// 10. Initialize notifications controller
	//     - Required by: User notifications
	//     - Dependencies: None
	m.notificationsController = notifications.NewController()

	// Unread indicators now managed by system controller

	// 11. Initialize MCP debug controller
	//     - Required by: MCP debugging features
	//     - Dependencies: debugMode flag
	m.mcpDebugController = NewMCPDebugController(debugMode)

	// Check for monorepo on startup - this will be handled by command window controller when needed

	// Script selector is now handled by ScriptSelectorController

	// 12. Initialize MCP connections list on first view if in debug mode
	//     - Required by: Debug view
	//     - Dependencies: mcpDebugController must be initialized
	if debugMode && initialView == ViewMCPConnections {
		m.mcpDebugController.UpdateConnectionsList()
	}

	// 13. Initialize AI Coder controller
	//     - Required by: AI coder features
	//     - Dependencies: cfg, eventBus, logStore, updateChan
	m.aiCoderController = NewAICoderController(cfg, eventBus, logStore, m.updateChan)

	// 14. Initialize script selector controller with navigation adapter
	//     - Required by: Script selection UI
	//     - Dependencies: navController (for navigation adapter)
	navAdapter := NewNavigationAdapter(m.navController)
	m.scriptSelectorController = NewScriptSelectorController(scripts, processMgr, logStore, m.updateChan, navAdapter)

	// 15. Initialize tabs component
	//     - Required by: Header rendering
	//     - Dependencies: processMgr, notificationsController, systemController
	m.tabsComponent = NewTabsComponent(processMgr, m.notificationsController, m.systemController, debugMode)

	// 16. Initialize content layout
	//     - Required by: Content area rendering
	//     - Dependencies: None
	m.contentLayout = NewContentLayout()

	// 17. Initialize message router
	//     - Required by: Message routing and handling
	//     - Dependencies: None
	m.messageRouter = NewMessageRouter()

	// 18. Set model reference for data provider and debug forwarder
	//     - Required by: AI coder data access
	//     - Dependencies: aiCoderController must be initialized, full model must be constructed
	m.aiCoderController.SetModelReference(&m)

	// 19. Set up event subscriptions immediately in constructor (not in Init)
	//     - Required by: Event handling
	//     - Dependencies: eventController must be initialized
	//     - Critical: This ensures subscriptions are active before MCP server starts
	m.setupEventSubscriptions()

	return &m
}

// setupEventSubscriptions sets up all event bus subscriptions
func (m *Model) setupEventSubscriptions() {
	// Delegate to EventController
	m.eventController.SetupEventSubscriptions()
}

func (m *Model) Init() tea.Cmd {
	// Send startup message via EventController
	m.eventController.SendStartupMessage()

	cmds := []tea.Cmd{
		textinput.Blink,
		m.waitForUpdates(),
		m.tickCmd(),
	}

	// PTY events are now handled by AICoderController

	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle notification messages first
	if cmd := m.notificationsController.HandleMsg(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Route message through Message Router for all handling
	if m.messageRouter != nil {
		newModel, cmd := m.messageRouter.Route(msg, m)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		m = newModel.(*Model)
	}

	// Handle only the most essential messages that can't be delegated
	switch msg := msg.(type) {
	case restartProcessMsg:
		if msg.clearLogs {
			m.logStore.ClearLogs()
			m.logStore.ClearErrors()
		}
		m.logStore.Add("system", "System", msg.message, msg.isError)
		m.updateProcessList()
		m.updateLogsView()
		cmds = append(cmds, m.waitForUpdates())

	case restartAllMsg:
		if msg.clearLogs {
			m.logStore.ClearLogs()
			m.logStore.ClearErrors()
		}
		m.logStore.Add("system", "System", msg.message, msg.isError)
		m.updateProcessList()
		m.updateLogsView()
		cmds = append(cmds, m.waitForUpdates())

	case errorUpdateMsg:
		m.updateErrorsList()
		cmds = append(cmds, m.waitForUpdates())

	case tickMsg:
		// Continue ticking for periodic updates
		cmds = append(cmds, m.tickCmd())

		// Update logs view if we're currently viewing logs to ensure real-time updates
		if m.currentView() == ViewLogs || m.currentView() == ViewURLs {
			m.updateLogsView()
		}

	case mcpActivityMsg:
		m.mcpDebugController.HandleActivity(msg)
		if m.currentView() == ViewMCPConnections && m.mcpDebugController.GetSelectedClient() != "" {
			m.mcpDebugController.UpdateActivityView()
		}
		cmds = append(cmds, m.waitForUpdates())

	case switchToAICodersMsg:
		// Switch to AI Coders view
		m.switchToView(ViewAICoders)
		cmds = append(cmds, m.waitForUpdates())

	case PTYOutputMsg:
		// Handle PTY output - update AI Coder view if it's active
		if m.currentView() == ViewAICoders && m.aiCoderController != nil {
			// The AI Coder controller will handle the PTY output internally
			if cmd := m.aiCoderController.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		// Always trigger a view update when PTY output arrives
		cmds = append(cmds, m.waitForUpdates())

	case PTYEventMsg:
		// Handle PTY events - update AI Coder view if it's active
		if m.currentView() == ViewAICoders && m.aiCoderController != nil {
			// The AI Coder controller will handle the PTY event internally
			if cmd := m.aiCoderController.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		// Always trigger a view update when PTY events arrive
		cmds = append(cmds, m.waitForUpdates())

	default:
		// Handle generic update messages (including nil messages from script selector)
		if msg == nil {
			// This is a generic update request - refresh views
			m.updateProcessList()
			m.updateLogsView()
			cmds = append(cmds, m.waitForUpdates())
		}
	}

	return m, tea.Batch(cmds...)
}
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Special views that take over the entire screen
	if m.currentView() == ViewScriptSelector && m.scriptSelectorController != nil {
		// Ensure the script selector is shown when this view is active
		if !m.scriptSelectorController.IsVisible() {
			m.scriptSelectorController.Show(false)
		}
		return m.scriptSelectorController.View()
	}
	if m.commandWindowController.IsShowingCommandWindow() {
		return m.commandWindowController.RenderCommandWindow()
	}

	// AI Coder PTY view in full screen mode should render raw
	if m.currentView() == ViewAICoders && m.aiCoderController != nil && m.aiCoderController.IsFullScreen() {
		// In full screen mode, return raw PTY output without any BubbleTea styling
		return m.aiCoderController.GetRawOutput()
	}
	content := m.renderContent()

	// Render main content with consistent layout
	return m.renderLayout(content)
}

// renderLayout provides consistent layout for all views
func (m *Model) renderLayout(content string) string {
	// Update layout controller state
	m.layoutController.UpdateSize(m.width, m.height)
	m.layoutController.SetCurrentView(string(m.currentView()))
	m.layoutController.SetSystemPanelOpen(m.systemController.IsExpanded())
	m.layoutController.SetSelectedProcess(m.selectedProcess)

	// Update tabs component
	if m.tabsComponent != nil {
		m.tabsComponent.SetActiveView(m.currentView())
		m.tabsComponent.SetWidth(m.width)
	}

	// Get header and footer heights from layout controller
	m.headerHeight = m.layoutController.GetHeaderHeight()
	m.footerHeight = m.layoutController.GetFooterHeight()

	// Update content layout dimensions
	if m.contentLayout != nil {
		m.contentLayout.UpdateDimensions(m.width, m.height, m.headerHeight, m.footerHeight)
	}

	// If system panel is expanded, show it instead of main content
	if m.systemController.IsExpanded() && m.systemController.HasMessages() {
		// Full screen mode - system panel takes most of the space
		contentHeight := m.height - m.headerHeight - m.footerHeight
		m.systemController.UpdateSize(m.width, contentHeight, 0, 0)
		content = m.systemPanelRenderer.RenderPanel()
		if m.contentLayout != nil {
			content = m.contentLayout.RenderContent(content)
		}
	} else if m.contentLayout != nil {
		// Use content layout for proper rendering
		content = m.contentLayout.RenderContent(content)
	}

	// Build the complete view
	var parts []string

	// Use tabs component for header
	if m.tabsComponent != nil {
		parts = append(parts, m.tabsComponent.Render())
		m.headerHeight = m.tabsComponent.GetHeight()
	} else {
		// Fallback to old header rendering
		parts = append(parts, m.renderHeader())
	}
	parts = append(parts, content)
	// Add footer via layout controller
	if m.layoutController != nil {
		parts = append(parts, m.layoutController.RenderFooter())
	}

	mainLayout := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// If we have system messages in non-expanded mode, overlay them
	if m.systemController.HasMessages() && !m.systemController.IsExpanded() {
		return m.overlaySystemPanel(mainLayout)
	}

	return mainLayout
}

// renderContent renders the main content area based on current view
func (m *Model) renderContent() string {
	if m.commandWindowController.IsShowingRunDialog() {
		return m.commandWindowController.RenderRunDialog()
	}

	if m.commandWindowController.IsShowingCustomCommand() {
		return m.commandWindowController.RenderCustomCommandDialog()
	}

	// Calculate content height once for all views
	contentHeight := m.calculateContentHeight()

	switch m.currentView() {
	case ViewProcesses:
		// Update controller dimensions and render
		m.processViewController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.processViewController.Render()
	case ViewLogs:
		// Update controller dimensions and sync state
		m.logsViewController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		m.logsViewController.SetSelectedProcess(m.selectedProcess)
		return m.logsViewController.Render()
	case ViewErrors:
		// Update controller dimensions and render
		m.errorsViewController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		// Use content layout for split view if available and screen is wide enough
		if m.contentLayout != nil && m.width >= 100 {
			// Calculate split sizes
			leftWidth := int(float64(m.width) * DefaultSplitRatio)
			rightWidth := m.width - leftWidth
			contentHeight := m.height - m.headerHeight - m.footerHeight

			// Update view sizes for split layout
			m.errorsViewController.GetErrorsList().SetSize(leftWidth-4, contentHeight-2)
			m.errorsViewController.GetErrorDetailView().Width = rightWidth - 4
			m.errorsViewController.GetErrorDetailView().Height = contentHeight - 2

			// Update the views
			m.errorsViewController.UpdateErrorsList()
			m.errorsViewController.UpdateErrorDetailView()

			leftContent := m.errorsViewController.GetErrorsList().View()
			rightContent := m.errorsViewController.GetErrorDetailView().View()

			// Use content layout for split view
			return m.contentLayout.RenderSplitView(leftContent, rightContent, DefaultSplitRatio)
		}
		// Fall back to regular rendering for narrow screens
		return m.errorsViewController.Render()
	case ViewURLs:
		// Update controller dimensions and render
		m.urlsViewController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.urlsViewController.Render()
	case ViewWeb:
		// Update controller dimensions and render
		m.webViewController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.webViewController.Render()
	case ViewSettings:
		if m.fileBrowserController.IsShowing() {
			return m.fileBrowserController.Render(m.width, m.height)
		}
		// Update controller dimensions and render
		m.settingsController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.settingsController.Render()
	case ViewMCPConnections:
		if m.debugMode {
			return m.mcpDebugController.Render(m.width, m.height, m.headerHeight, m.footerHeight)
		}
		// Fallback to settings if not in debug mode
		m.settingsController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.settingsController.Render()
	case ViewAICoders:
		if m.aiCoderController != nil {
			// Update controller dimensions before rendering
			m.aiCoderController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
			return m.aiCoderController.Render()
		}
		return "AI Coder feature is not initialized"
	case ViewFilters:
		return m.renderFiltersView()
	default:
		return "Unknown view"
	}
}

// getViewStatus returns status information for the current view
func (m *Model) getViewStatus() string {
	switch m.currentView() {
	case ViewProcesses:
		processes := m.processMgr.GetAllProcesses()
		running := 0
		for _, p := range processes {
			if p.GetStatus() == process.StatusRunning {
				running++
			}
		}
		return fmt.Sprintf("%d processes, %d running", len(processes), running)

	case ViewLogs:
		if m.selectedProcess != "" {
			return fmt.Sprintf("Process: %s", m.selectedProcess)
		}
		return "All processes"

	case ViewErrors:
		errors := m.logStore.GetErrors()
		errorCount := len(errors)
		return fmt.Sprintf("%d errors", errorCount)

	case ViewURLs:
		urls := m.logStore.GetURLs()
		return fmt.Sprintf("%d URLs detected", len(urls))

	case ViewWeb:
		requests := m.proxyServer.GetRequests()
		return fmt.Sprintf("%d requests", len(requests))

	case ViewAICoders:
		running, total := m.aiCoderController.GetStatusInfo()
		return fmt.Sprintf("%d AI coders, %d running", total, running)

	default:
		return ""
	}
}

func (m *Model) updateSizes() {
	// Update layout controller first
	if m.layoutController != nil {
		m.layoutController.UpdateSizes(m.width, m.height)
		m.headerHeight = m.layoutController.GetHeaderHeight()
		m.footerHeight = m.layoutController.GetFooterHeight()
	}

	// Use calculated header and footer heights for consistency with other views
	contentHeight := m.height - m.headerHeight - m.footerHeight

	// processesList sizing now handled by ProcessViewController

	// Calculate content height once for all controllers
	calculatedContentHeight := m.calculateContentHeight()

	// Update process controller if initialized
	if m.processViewController != nil {
		m.processViewController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, calculatedContentHeight)
	}

	// settingsList sizing now handled by SettingsController
	m.commandWindowController.GetCommandsList().SetSize(m.width, contentHeight)
	m.errorsViewController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, calculatedContentHeight)
	m.logsViewController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, calculatedContentHeight)

	// Update URLs controller if initialized
	if m.urlsViewController != nil {
		m.urlsViewController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, calculatedContentHeight)
	}
	// Web view sizing handled by WebViewController during rendering

	// Update AI Coder controller size if initialized
	if m.aiCoderController != nil {
		m.aiCoderController.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, calculatedContentHeight)
	}

	// Update command window controller size
	m.commandWindowController.UpdateSize(m.width, m.height)
}

func (m *Model) cycleView() {
	// Delegate to navigation controller with view-specific setup
	m.navController.CycleView()
}

func (m *Model) cyclePrevView() {
	// Delegate to navigation controller with view-specific setup
	m.navController.CyclePreviousView()
}

// switchToView changes the current view and performs any necessary setup
func (m *Model) switchToView(view View) {
	// Delegate to navigation controller with view-specific setup
	m.navController.SwitchToView(view)
}

func (m *Model) updateProcessList() {
	// Delegate to controller
	if m.processViewController != nil {
		m.processViewController.UpdateProcessList()
		// Sync the controller's list back to the model's list
		// processesList synchronization no longer needed - managed by ProcessViewController
	}
}

func (m *Model) updateLogsView() {
	// Update the logs view controller with current state
	m.logsViewController.SetSelectedProcess(m.selectedProcess)
	// Show/hide patterns are now managed by LogsViewController

	// Delegate to logs view controller
	m.logsViewController.UpdateLogsView()
}

func (m *Model) renderHeader() string {
	// Get process count information
	processes := m.processMgr.GetAllProcesses()
	runningCount := 0
	for _, proc := range processes {
		if proc.GetStatus() == process.StatusRunning {
			runningCount++
		}
	}

	// Build title with process info
	baseTitle := "🐝 Brummer - Development Buddy"
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
	if m.notificationsController.IsActive() {
		notificationStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
		notification = " " + notificationStyle.Render(m.notificationsController.GetMessage())
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Render(baseTitle + processInfo + browserIcon + notification)

	tabs := []string{}
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Use ordered list of views
	orderedViews := []View{ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewWeb, ViewAICoders, ViewSettings}
	if m.debugMode {
		orderedViews = append(orderedViews, ViewMCPConnections)
	}
	for i, viewType := range orderedViews {
		if cfg, ok := viewConfigs[viewType]; ok {
			// Build the base label with icon and space before number
			label := fmt.Sprintf("%s.%s", cfg.KeyBinding, cfg.Title)
			if cfg.Icon != "" {
				label = cfg.Icon + " " + label
			}

			// Get unread indicator for this view
			var indicatorIcon string
			indicators := m.systemController.GetUnreadIndicators()
			if indicator, exists := indicators[viewType]; exists && indicator.Count > 0 {
				indicatorIcon = indicator.Icon
			} else {
				indicatorIcon = "" // No space when no indicator
			}

			// Format the tab
			var tab string
			if viewType == m.currentView() {
				// Active tab: ▶icon1.Titleindicator
				tab = activeStyle.Render("▶" + label + indicatorIcon)
			} else {
				// Inactive tab:  icon1.Titleindicator
				tab = inactiveStyle.Render(" " + label + indicatorIcon)
			}

			// Add separator except for the last tab
			if i < len(orderedViews)-1 {
				tab += separatorStyle.Render(" | ")
			}

			tabs = append(tabs, tab)
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Left, tabs...)

	// Use Lipgloss border instead of manual line
	headerStyle := lipgloss.NewStyle().
		Width(m.width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(lipgloss.Color("240"))

	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		tabBar,
	)

	header := headerStyle.Render(headerContent)

	// Store header height for layout calculations
	m.headerHeight = strings.Count(header, "\n") + 1

	return header
}

// View-specific render methods have been moved to their respective controllers

func (m *Model) renderFiltersView() string {
	filters := m.logStore.GetFilters()
	if len(filters) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No filters configured. Use / commands: /show <pattern> or /hide <pattern> to filter logs")
	}

	var content strings.Builder
	for _, f := range filters {
		content.WriteString(fmt.Sprintf("• %s: %s (Priority +%d)\n", f.Name, f.Pattern, f.PriorityBoost))
	}

	return content.String()
}

// renderURLsViewSimple has been moved to URLsViewController

// renderURLsList has been moved to URLsViewController

type logUpdateMsg struct{}
type tickMsg struct{}
type switchToAICodersMsg struct{}

func (m *Model) waitForUpdates() tea.Cmd {
	// Delegate to EventController
	return m.eventController.WaitForUpdates()
}

func (m *Model) tickCmd() tea.Cmd {
	// Delegate to EventController
	return m.eventController.TickCmd()
}

func (m *Model) handleRestartProcess(proc *process.Process) tea.Cmd {
	// Delegate to ProcessViewController
	return m.processViewController.HandleRestartProcess(proc)
}

func (m *Model) handleRestartAll() tea.Cmd {
	// Delegate to ProcessViewController
	return m.processViewController.HandleRestartAll()
}

func (m *Model) handleCopyError() tea.Cmd {
	// Delegate to ErrorsViewController
	return m.errorsViewController.HandleCopyError()
}

func (m *Model) handleClearLogs() {
	m.logStore.ClearLogs()
	m.logStore.Add("system", "System", "📝 Logs cleared", false)
	m.updateLogsView()
}

func (m *Model) handleClearErrors() {
	// Delegate to ErrorsViewController
	m.errorsViewController.HandleClearErrors()
	m.updateLogsView()
}

func (m *Model) handleClearScreen() {
	switch m.currentView() {
	case ViewLogs:
		m.logStore.ClearLogs()
		m.logStore.Add("system", "System", "📝 Logs cleared", false)
		m.updateLogsView()
	case ViewErrors:
		m.logStore.ClearErrors()
		m.logStore.Add("system", "System", "🗑️ Error history cleared", false)
		m.updateLogsView()
	case ViewWeb:
		if m.proxyServer != nil {
			m.proxyServer.ClearRequests()
			m.logStore.Add("system", "System", "🌐 Web requests cleared", false)
			m.updateLogsView()
		}
	}
}

func (m *Model) showRunDialog() {
	// Get detected commands
	detectedCommands := m.processMgr.GetDetectedCommands()

	// Get monorepo info
	monorepoInfo, _ := m.processMgr.GetMonorepoInfo()

	m.commandWindowController.ShowRunDialog(detectedCommands, monorepoInfo)
}

func (m *Model) handleRunCommand() tea.Cmd {
	if !m.commandWindowController.IsShowingRunDialog() {
		return nil
	}

	selectedCommand := m.commandWindowController.GetSelectedCommand()
	if selectedCommand == nil {
		return nil
	}

	m.commandWindowController.HideRunDialog()

	// Check if this is a "custom command" selection or a regular command
	commandsList := m.commandWindowController.GetCommandsList()
	selected := commandsList.SelectedItem()

	if _, ok := selected.(runCustomItem); ok {
		// Show custom command input dialog
		m.commandWindowController.ShowCustomCommandDialog()
		return nil
	}

	// Handle regular command execution
	errorHandler := NewStandardErrorHandler(m.logStore, m.updateChan)
	SafeGoroutine(
		fmt.Sprintf("start command '%s'", selectedCommand.Name),
		func() error {
			_, err := m.processMgr.StartCommand(selectedCommand.Name, selectedCommand.Command, selectedCommand.Args)
			return err
		},
		func(err error) {
			ctx := ProcessStartContext(selectedCommand.Name, "Command Window", m.logStore, m.updateChan)
			errorHandler.HandleError(err, ctx)
		},
	)
	m.navController.SwitchTo(ViewProcesses)
	m.updateProcessList()
	return m.waitForUpdates()
}

func (m *Model) updateErrorsList() {
	// Delegate to ErrorsViewController and handle unread indicators
	countChange := m.errorsViewController.UpdateErrorsList()

	// Update unread indicator if we have new errors
	if countChange > 0 && m.currentView() != ViewErrors {
		// Update system controller with current view and delegate
		m.systemController.SetCurrentView(m.currentView())
		m.systemController.UpdateUnreadIndicator(ViewErrors, "error", countChange)
	}
}

func (m *Model) updateErrorDetailView() {
	// Delegate to ErrorsViewController
	m.errorsViewController.UpdateErrorDetailView()
}

// calculateContentHeight returns the available content height with consistent calculation
func (m *Model) calculateContentHeight() int {
	// Subtract 1 to account for proper spacing between header and content
	contentHeight := m.height - m.headerHeight - m.footerHeight - 1
	if contentHeight < 1 {
		contentHeight = 1
	}
	return contentHeight
}

// renderErrorsViewSplit moved to ErrorsViewController

func (m *Model) handleSlashCommand(input string) {
	// Create the context for the slash command handler
	currentView := string(m.currentView())
	ctx := &commands.SlashCommandContext{
		ProcessManager: m.processMgr,
		LogStore:       m.logStore,
		UpdateChan:     m.updateChan,

		// Current state pointers - get from LogsViewController
		ShowPattern:   m.logsViewController.GetShowPatternPtr(),
		HidePattern:   m.logsViewController.GetHidePatternPtr(),
		SearchResults: m.logsViewController.GetSearchResultsPtr(),
		CurrentView:   &currentView,

		// Callbacks
		UpdateLogsView: func() { m.updateLogsView() },
		ClearLogs:      func(target string) { m.handleClearCommand(target) },
		SetProxyURL:    func(urlStr string) { m.handleProxyCommand(urlStr) },
		ToggleProxy:    func() { m.toggleProxyMode() },
		StartAICoder:   func(providerName string) { m.handleAICommand(providerName) },
		ShowTerminal:   func() { m.showTerminal() },
	}

	// Delegate to the functional handler
	commands.HandleSlashCommand(ctx, input)
}

// Helper methods for slash command callbacks
func (m *Model) handleClearCommand(target string) {
	switch target {
	case "all":
		m.logStore.ClearLogs()
		m.errorsViewController.HandleClearErrors()
		if m.proxyServer != nil {
			m.proxyServer.ClearRequests()
		}
		m.logStore.Add("system", "System", "🧹 Cleared all logs, errors, and web requests", false)
	case "logs":
		m.logStore.ClearLogs()
		m.logStore.Add("system", "System", "🧹 Cleared all logs", false)
	case "errors":
		m.errorsViewController.HandleClearErrors()
	case "web":
		if m.proxyServer != nil {
			m.proxyServer.ClearRequests()
			m.logStore.Add("system", "System", "🧹 Cleared all web requests", false)
		} else {
			m.logStore.Add("system", "System", "⚠️ Web proxy is not enabled", true)
		}
	default:
		// Check if it's a script name
		scripts := m.processMgr.GetScripts()
		if _, ok := scripts[target]; ok {
			// Clear logs for a specific process
			m.logStore.ClearLogsForProcess(target)
			m.logStore.Add("system", "System", fmt.Sprintf("🧹 Cleared logs for process: %s", target), false)
		} else {
			m.logStore.Add("system", "System", fmt.Sprintf("⚠️ Unknown clear target: %s", target), true)
			m.logStore.Add("system", "System", "Usage: /clear [all|logs|errors|web|<script-name>]", false)
		}
	}
	m.updateLogsView()
}

func (m *Model) handleProxyCommand(urlStr string) {
	if m.proxyServer == nil {
		m.logStore.Add("system", "System", "Error: Proxy server is not enabled", true)
		return
	}

	// Validate URL
	_, err := url.Parse(urlStr)
	if err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Invalid URL: %v", err), true)
		return
	}

	// In reverse proxy mode, URLs are automatically detected and mapped
	m.logStore.Add("system", "System", fmt.Sprintf("🔗 URLs starting with %s will be proxied when detected", urlStr), false)
	m.navController.SwitchTo(ViewWeb)
}

func (m *Model) toggleProxyMode() {
	m.handleToggleProxy()
}

func (m *Model) handleAICommand(providerName string) {
	if m.aiCoderController == nil {
		m.logStore.Add("system", "System", "Error: AI Coder feature is not initialized", true)
		return
	}
	m.aiCoderController.HandleAICommand(providerName)
}

func (m *Model) showTerminal() {
	// Terminal view not implemented yet
	m.logStore.Add("system", "System", "Terminal view is not yet implemented", true)
}

func (m *Model) handleToggleProxy() {
	if m.proxyServer == nil {
		m.logStore.Add("system", "System", "Error: Proxy server is not enabled", true)
		return
	}

	currentMode := m.proxyServer.GetMode()
	var newMode proxy.ProxyMode
	var modeDesc string

	if currentMode == proxy.ProxyModeReverse {
		newMode = proxy.ProxyModeFull
		modeDesc = "full proxy mode"
	} else {
		newMode = proxy.ProxyModeReverse
		modeDesc = "reverse proxy mode"
	}

	if err := m.proxyServer.SwitchMode(newMode); err != nil {
		msg := fmt.Sprintf("Error switching proxy mode: %v", err)
		m.logStore.Add("system", "System", msg, true)
	} else {
		msg := fmt.Sprintf("🔄 Switched to %s", modeDesc)
		m.logStore.Add("system", "System", msg, false)

		// Switch to URLs view to show the change
		m.navController.SwitchTo(ViewURLs)
	}
}

func (m *Model) showCommandWindow() {
	scripts := m.processMgr.GetScripts()

	// Get available AI providers from AI coder controller (with nil check)
	var aiProviders []string
	if m.aiCoderController != nil {
		aiProviders = m.aiCoderController.GetProviders()
	}

	m.commandWindowController.ShowCommandWindow(scripts, aiProviders)
}

// System message methods have been moved to systemController

// NOTE: HTTP errors from proxied requests are tracked in the Web tab,
// not in the system message panel. The system panel is only for internal
// Brummer messages (process control errors, settings errors, etc.)

// NOTE: JavaScript errors from telemetry are tracked in the Errors tab,
// not in the system message panel. The system panel is only for internal
// Brummer messages.

// Unread indicator methods moved to systemController

// overlaySystemPanel overlays the system panel on top of the main content
func (m *Model) overlaySystemPanel(mainContent string) string {
	// Use the new overlay renderer if available
	if m.systemOverlayRenderer != nil {
		return m.systemOverlayRenderer.RenderOverlay(mainContent, m.width, m.height)
	}
	// Fallback to old implementation
	return m.systemController.OverlaySystemPanel(mainContent)
}

// renderSettings moved to direct controller usage

// getCLICommandFromConfig retrieves CLI command and args from configuration
func (m *Model) getCLICommandFromConfig(configKey string, task string) (string, []string, error) {
	// For now, use the auto-detected CLI tool mappings
	// This is a simplified implementation that would be enhanced with actual config lookup

	cliMappings := map[string]struct {
		command  string
		baseArgs []string
		taskFlag string
	}{
		"claude": {
			command:  "claude",
			baseArgs: []string{},
			taskFlag: "", // Claude CLI accepts task as direct input
		},
		"sonnet": {
			command:  "claude",
			baseArgs: []string{"--model", "sonnet"},
			taskFlag: "",
		},
		"opus": {
			command:  "claude",
			baseArgs: []string{"--model", "opus"},
			taskFlag: "",
		},
		"aider": {
			command:  "aider",
			baseArgs: []string{"--yes"},
			taskFlag: "--message",
		},
		"opencode": {
			command:  "opencode",
			baseArgs: []string{"run"},
			taskFlag: "--prompt",
		},
		"gemini": {
			command:  "gemini",
			baseArgs: []string{},
			taskFlag: "--prompt",
		},
	}

	mapping, exists := cliMappings[configKey]
	if !exists {
		return "", nil, fmt.Errorf("CLI configuration not found for '%s'", configKey)
	}

	args := make([]string, len(mapping.baseArgs))
	copy(args, mapping.baseArgs)

	// If task is provided, add it based on the tool's requirements
	if task != "" {
		if mapping.taskFlag != "" {
			// Tools that use flags for tasks (aider, opencode, gemini)
			args = append(args, mapping.taskFlag, task)
		} else {
			// Tools that accept task as direct input (claude)
			// For non-interactive mode, use structured streaming output
			if configKey == "claude" || configKey == "sonnet" || configKey == "opus" {
				args = append(args, "--print", "--verbose", "--output-format", "stream-json")
			}
			args = append(args, task)
		}
	}

	return mapping.command, args, nil
}

// File browser list is now managed entirely by filebrowserController

// currentView returns the current view from the navigation controller
func (m *Model) currentView() View {
	return m.navController.CurrentView()
}

// GetAICoderManager returns the AI coder manager instance
func (m *Model) GetAICoderManager() *aicoder.AICoderManager {
	return m.aiCoderController.GetAICoderManager()
}
