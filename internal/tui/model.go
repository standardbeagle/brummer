package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/parser"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
)

type View string

const (
	ViewScripts        View = "scripts"
	ViewProcesses      View = "processes"
	ViewLogs           View = "logs"
	ViewErrors         View = "errors"
	ViewURLs           View = "urls"
	ViewWeb            View = "web"
	ViewSettings       View = "settings"
	ViewMCPConnections View = "mcp-connections"
	ViewSearch         View = "search"
	ViewFilters        View = "filters"
	ViewScriptSelector View = "script-selector"
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
	ViewSettings: {
		Title:       "Settings",
		Description: "Configuration",
		KeyBinding:  "6",
		Icon:        "⚙️",
	},
	ViewMCPConnections: {
		Title:       "MCP",
		Description: "MCP Connections",
		KeyBinding:  "7",
		Icon:        "🔌",
	},
}

// SystemMessage represents an internal Brummer system message for display at the bottom
type SystemMessage struct {
	Timestamp time.Time
	Level     string // "error", "warning", "info", "success"
	Message   string
	Context   string // Where the message originated (e.g., "Process Control", "Settings")
}

// UnreadIndicator tracks unread content in a view
type UnreadIndicator struct {
	Count    int    // Number of unread items
	Severity string // "error", "warning", "info", "success"
	Icon     string // Icon to display (based on severity)
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

	currentView View
	width       int
	height      int

	processesList     list.Model
	logsViewport      viewport.Model
	errorsViewport    viewport.Model
	errorsList        list.Model
	selectedError     *logs.ErrorContext
	errorDetailView   viewport.Model
	urlsViewport      viewport.Model
	webDetailViewport viewport.Model
	settingsList      list.Model
	searchInput       textinput.Model

	selectedProcess  string
	searchResults    []logs.LogEntry
	showHighPriority bool

	// Slash command filters
	showPattern string // Regex pattern for /show command
	hidePattern string // Regex pattern for /hide command

	// Logs view state
	logsAutoScroll bool // Whether to auto-scroll to bottom
	logsAtBottom   bool // Whether viewport is at bottom

	// Command window state
	showingCommandWindow bool
	commandAutocomplete  CommandAutocomplete

	// Web view state
	webFilter       string         // Current filter: "all", "pages", "api", "images", "other"
	webAutoScroll   bool           // Whether to auto-scroll to bottom
	selectedRequest *proxy.Request // Selected request for detail view
	webRequestsList list.Model     // List of proxy requests (replaces webViewport)

	// Script selector state (for initial view)
	scriptSelector CommandAutocomplete

	// File browser state
	showingFileBrowser bool
	currentPath        string
	fileList           []FileItem

	// Run dialog state
	showingRunDialog bool
	commandsList     list.Model
	detectedCommands []parser.ExecutableCommand
	monorepoInfo     *parser.MonorepoInfo

	// Custom command dialog state
	showingCustomCommand bool
	customCommandInput   textinput.Model

	// UI state
	copyNotification string
	notificationTime time.Time

	// System message panel state (for internal Brummer messages)
	systemPanelExpanded bool            // Whether system message panel is expanded to full screen
	systemMessages      []SystemMessage // Recent system messages (errors, warnings, info)
	systemPanelViewport viewport.Model  // Viewport for full-screen system message view

	// Unread content tracking
	unreadIndicators map[View]UnreadIndicator // Track unread content per view
	lastErrorCount   int                      // Track last error count
	lastWebCount     int                      // Track last web request count

	help help.Model
	keys keyMap

	updateChan chan tea.Msg

	// MCP connections view state
	mcpConnectionsList  list.Model                    // List of MCP connections
	mcpActivityViewport viewport.Model                // Activity log for selected connection
	selectedMCPClient   string                        // Selected MCP client ID
	mcpConnections      map[string]*mcpConnectionItem // sessionId -> connection
	mcpActivities       map[string][]MCPActivity      // sessionId -> activities
	mcpActivityMu       sync.RWMutex
}

// handleGlobalKeys handles keys that should work in all views
func (m *Model) handleGlobalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
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
			return *m, tea.Sequence(
				tea.Printf("Stopping %d running processes...\n", runningProcesses),
				func() tea.Msg {
					_ = m.processMgr.Cleanup() // Ignore cleanup errors during shutdown
					return tea.Msg(nil)
				},
				tea.Printf("%s", renderExitScreen()),
				tea.Quit,
			), true
		} else {
			return *m, tea.Sequence(
				tea.Printf("%s", renderExitScreen()),
				tea.Quit,
			), true
		}

	case key.Matches(msg, m.keys.Tab):
		m.cycleView()
		return *m, nil, true

	case msg.String() == "shift+tab":
		m.cyclePrevView()
		return *m, nil, true

	case msg.String() == "left":
		m.cyclePrevView()
		return *m, nil, true

	case msg.String() == "right":
		m.cycleView()
		return *m, nil, true

	case key.Matches(msg, m.keys.ClearScreen):
		m.handleClearScreen()
		return *m, nil, true

	case key.Matches(msg, m.keys.Back):
		if m.currentView == ViewFilters {
			m.currentView = ViewLogs
		} else if m.currentView == ViewLogs || m.currentView == ViewErrors || m.currentView == ViewURLs {
			m.currentView = ViewProcesses
		}
		return *m, nil, true

	case key.Matches(msg, m.keys.Priority):
		if m.currentView == ViewLogs {
			m.showHighPriority = !m.showHighPriority
			m.updateLogsView()
		}
		return *m, nil, true

	case key.Matches(msg, m.keys.RestartAll):
		if m.currentView == ViewProcesses {
			m.logStore.Add("system", "System", "Restarting all running processes...", false)
			return *m, m.handleRestartAll(), true
		}
		return *m, nil, true

	case key.Matches(msg, m.keys.CopyError):
		return *m, m.handleCopyError(), true

	case key.Matches(msg, m.keys.ClearLogs):
		if m.currentView == ViewLogs {
			m.handleClearLogs()
		}
		return *m, nil, true

	case key.Matches(msg, m.keys.ToggleError):
		if len(m.systemMessages) > 0 {
			m.systemPanelExpanded = !m.systemPanelExpanded
			if m.systemPanelExpanded {
				// Update system panel viewport when expanding
				m.updateSystemPanelViewport()
			}
		}
		return *m, nil, true

	case key.Matches(msg, m.keys.ClearMessages):
		// Clear system messages
		if len(m.systemMessages) > 0 {
			m.systemMessages = []SystemMessage{}
			m.systemPanelExpanded = false
			// Clear the viewport content explicitly
			m.systemPanelViewport.SetContent("")
			// Force immediate re-render
			return *m, tea.ClearScreen, true
		}
		return *m, nil, true
	}

	// Handle number keys for view switching
	for viewType, cfg := range viewConfigs {
		if msg.String() == cfg.KeyBinding {
			// Skip MCP connections view if not in debug mode
			if viewType == ViewMCPConnections && !m.debugMode {
				continue
			}
			m.switchToView(viewType)
			return *m, nil, true
		}
	}

	return *m, nil, false // Key not handled
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
	process    *process.Process
	isHeader   bool
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
		statusEmoji = "✓" // Thin checkmark for gracefully stopped
	case process.StatusFailed:
		statusEmoji = "❌"
	case process.StatusSuccess:
		statusEmoji = "✓" // Thin checkmark for success
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

// proxyRequestItem implements list.Item for proxy requests
type proxyRequestItem struct {
	Request proxy.Request
}

func (i proxyRequestItem) FilterValue() string {
	return i.Request.URL + " " + i.Request.Method
}

func (i proxyRequestItem) Title() string {
	// Basic title - actual rendering with truncation is handled in delegate
	return fmt.Sprintf("%s %d %s %s",
		i.Request.StartTime.Format("15:04:05"),
		i.Request.StatusCode,
		i.Request.Method,
		i.Request.URL)
}

func (i proxyRequestItem) Description() string {
	if i.Request.Error != "" {
		return "Error: " + i.Request.Error
	}
	if i.Request.Size > 0 {
		return fmt.Sprintf("Size: %s", formatBytes(i.Request.Size))
	}
	return fmt.Sprintf("Duration: %dms", i.Request.Duration.Milliseconds())
}

// proxyRequestDelegate implements list.ItemDelegate for proxy requests
type proxyRequestDelegate struct{}

func (d proxyRequestDelegate) Height() int                               { return 1 }
func (d proxyRequestDelegate) Spacing() int                              { return 0 }
func (d proxyRequestDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d proxyRequestDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if item, ok := listItem.(proxyRequestItem); ok {
		// Calculate available width for URL based on list width
		listWidth := m.Width()

		// Fixed parts: time(8) + space + status(3) + space + method(7 max) + space + indicators(6 max) + padding(4)
		// Conservative estimate for fixed content
		timeWidth := 8       // "15:04:05"
		statusWidth := 3     // "200"
		methodWidth := 7     // "DELETE" (longest common method)
		indicatorsWidth := 6 // " ❌ 🔐 📊" (worst case)
		spacesWidth := 4     // spaces between elements
		paddingWidth := 4    // general padding/margins

		fixedWidth := timeWidth + statusWidth + methodWidth + indicatorsWidth + spacesWidth + paddingWidth

		// Available width for URL (ensure minimum of 20 chars)
		maxURLLength := listWidth - fixedWidth
		if maxURLLength < 20 {
			maxURLLength = 20
		}

		url := item.Request.URL
		if len(url) > maxURLLength {
			url = url[:maxURLLength-3] + "..."
		}

		// Build the line
		line := fmt.Sprintf("%s %d %s %s",
			item.Request.StartTime.Format("15:04:05"),
			item.Request.StatusCode,
			item.Request.Method,
			url)

		// Add indicators
		if item.Request.Error != "" {
			line += " ❌"
		}
		if item.Request.HasAuth {
			line += " 🔐"
		}
		if item.Request.HasTelemetry {
			line += " 📊"
		}

		var str string
		if index == m.Index() {
			// Selected item - highlighted
			str = lipgloss.NewStyle().Background(lipgloss.Color("240")).Render(line)
		} else {
			// Normal item
			str = line
		}
		fmt.Fprint(w, str)
	}
}

// Helper function for formatting bytes
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

func NewModel(processMgr *process.Manager, logStore *logs.Store, eventBus *events.EventBus, mcpServer MCPServerInterface, proxyServer *proxy.Server, mcpPort int) Model {
	return NewModelWithView(processMgr, logStore, eventBus, mcpServer, proxyServer, mcpPort, ViewProcesses, false)
}

func NewModelWithView(processMgr *process.Manager, logStore *logs.Store, eventBus *events.EventBus, mcpServer MCPServerInterface, proxyServer *proxy.Server, mcpPort int, initialView View, debugMode bool) Model {
	scripts := processMgr.GetScripts()

	processesList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	processesList.Title = "Running Processes"
	processesList.SetShowStatusBar(false)

	searchInput := textinput.New()
	searchInput.Placeholder = "Commands: /show <pattern> | /hide <pattern>"
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
		mcpPort:        mcpPort,
		proxyServer:    proxyServer,
		debugMode:      debugMode,
		currentView:    initialView,
		processesList:  processesList,
		settingsList:   settingsList,
		logsViewport:   viewport.New(0, 0),
		errorsViewport: viewport.New(0, 0),
		urlsViewport:   viewport.New(0, 0),
		webRequestsList: func() list.Model {
			l := list.New([]list.Item{}, proxyRequestDelegate{}, 0, 0)
			l.SetShowTitle(false)
			l.SetShowStatusBar(false)
			l.SetShowHelp(false)
			return l
		}(),
		webDetailViewport: viewport.New(0, 0),
		searchInput:       searchInput,
		webFilter:         "all", // Default to showing all requests
		webAutoScroll:     true,  // Start with auto-scroll enabled
		help:              help.New(),
		keys:              keys,
		updateChan:        make(chan tea.Msg, 100),
		currentPath:       getCurrentDir(),
		logsAutoScroll:    true, // Start with auto-scroll enabled
	}

	// Note: Log callback is registered in main.go to avoid duplication

	// Initialize settings list
	m.updateSettingsList()

	// Initialize process list with current processes
	m.updateProcessList()

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

	// Initialize system message panel
	m.systemPanelViewport = viewport.New(0, 0)
	m.systemMessages = make([]SystemMessage, 0, 100) // Keep up to 100 messages

	// Initialize unread indicators
	m.unreadIndicators = make(map[View]UnreadIndicator)

	// Initialize MCP connections view if in debug mode
	if debugMode {
		mcpConnectionsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
		mcpConnectionsList.Title = "MCP Connections"
		mcpConnectionsList.SetShowStatusBar(false)
		m.mcpConnectionsList = mcpConnectionsList
		m.mcpActivityViewport = viewport.New(0, 0)
		m.mcpConnections = make(map[string]*mcpConnectionItem)
		m.mcpActivities = make(map[string][]MCPActivity)
	}

	// Check for monorepo on startup
	m.monorepoInfo, _ = processMgr.GetMonorepoInfo()

	// Initialize script selector if starting in that view
	if initialView == ViewScriptSelector {
		m.scriptSelector = NewScriptSelectorAutocompleteWithProcessManager(scripts, processMgr)
		m.scriptSelector.SetWidth(60)
		m.scriptSelector.Focus()
	}

	// Initialize MCP connections list on first view if in debug mode
	if debugMode && initialView == ViewMCPConnections {
		m.updateMCPConnectionsList()
	}

	// Set up event subscriptions immediately in constructor (not in Init)
	// This ensures subscriptions are active before MCP server starts
	m.setupEventSubscriptions()

	return m
}

// setupEventSubscriptions sets up all event bus subscriptions
func (m *Model) setupEventSubscriptions() {
	// Set up event subscriptions
	m.eventBus.Subscribe(events.ProcessStarted, func(e events.Event) {
		m.updateChan <- processUpdateMsg{}
	})

	m.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		// Clean up URLs from the exited process
		if e.ProcessID != "" {
			m.logStore.RemoveURLsForProcess(e.ProcessID)
		}

		// Check if process failed and add unread indicator
		if exitCode, ok := e.Data["exitCode"].(int); ok && exitCode != 0 && m.currentView != ViewProcesses {
			m.updateUnreadIndicator(ViewProcesses, "error", 1)
		}

		m.updateChan <- processUpdateMsg{}
	})

	m.eventBus.Subscribe(events.LogLine, func(e events.Event) {
		m.updateChan <- logUpdateMsg{}
	})

	m.eventBus.Subscribe(events.ErrorDetected, func(e events.Event) {
		m.updateChan <- errorUpdateMsg{}
	})

	// Subscribe to proxy events
	m.eventBus.Subscribe(events.EventType("proxy.request"), func(e events.Event) {
		m.updateChan <- webUpdateMsg{}
	})

	// Subscribe to telemetry events
	m.eventBus.Subscribe(events.EventType("telemetry.received"), func(e events.Event) {
		m.updateChan <- webUpdateMsg{} // Update web view when telemetry is received
	})

	// Subscribe to system messages
	m.eventBus.Subscribe(events.EventType("system.message"), func(e events.Event) {
		level, _ := e.Data["level"].(string)
		context, _ := e.Data["context"].(string)
		message, _ := e.Data["message"].(string)
		if message != "" {
			// Send the message data through the update channel
			go func() {
				m.updateChan <- systemMessageMsg{
					level:   level,
					context: context,
					message: message,
				}
			}()
		}
	})

	// Subscribe to MCP events if in debug mode
	if m.debugMode {
		m.eventBus.Subscribe(events.MCPConnected, func(e events.Event) {
			sessionId, _ := e.Data["sessionId"].(string)
			clientInfo, _ := e.Data["clientInfo"].(string)
			connectedAt, _ := e.Data["connectedAt"].(time.Time)
			connectionType, _ := e.Data["connectionType"].(string)
			method, _ := e.Data["method"].(string)

			m.updateChan <- mcpConnectionMsg{
				sessionId:      sessionId,
				clientInfo:     clientInfo,
				connected:      true,
				connectedAt:    connectedAt,
				connectionType: connectionType,
				method:         method,
			}
		})

		m.eventBus.Subscribe(events.MCPDisconnected, func(e events.Event) {
			sessionId, _ := e.Data["sessionId"].(string)

			m.updateChan <- mcpConnectionMsg{
				sessionId: sessionId,
				connected: false,
			}
		})

		m.eventBus.Subscribe(events.MCPActivity, func(e events.Event) {
			sessionId, _ := e.Data["sessionId"].(string)
			method, _ := e.Data["method"].(string)
			params, _ := e.Data["params"].(string)
			response, _ := e.Data["response"].(string)
			errMsg, _ := e.Data["error"].(string)
			duration, _ := e.Data["duration"].(time.Duration)

			activity := MCPActivity{
				Timestamp: time.Now(),
				Method:    method,
				Params:    params,
				Response:  response,
				Error:     errMsg,
				Duration:  duration,
			}

			m.updateChan <- mcpActivityMsg{
				sessionId: sessionId,
				activity:  activity,
			}
		})
	}
}

func (m Model) Init() tea.Cmd {
	// Add startup system message
	go func() {
		m.updateChan <- systemMessageMsg{
			level:   "info",
			context: "System",
			message: "🚀 Brummer started - initializing services...",
		}
	}()

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
		// Handle script selector view
		if m.currentView == ViewScriptSelector {
			return m.handleScriptSelector(msg)
		}

		// Handle command window first
		if m.showingCommandWindow {
			return m.handleCommandWindow(msg)
		}

		// Handle "/" key to open command window
		if msg.String() == "/" && m.width > 0 && m.height > 0 {
			m.showCommandWindow()
			return m, nil
		}

		// Handle global keys first
		if model, cmd, handled := m.handleGlobalKeys(msg); handled {
			return model, cmd
		}

		switch {
		case key.Matches(msg, m.keys.ClearErrors):
			if m.currentView == ViewErrors {
				m.handleClearErrors()
			}

		case key.Matches(msg, m.keys.Enter):
			cmds = append(cmds, m.handleEnter())

		case key.Matches(msg, m.keys.RunDialog):
			if !m.showingRunDialog {
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

	case webUpdateMsg:
		if m.currentView == ViewWeb {
			m.updateWebView()
		}
		cmds = append(cmds, m.waitForUpdates())

	case systemMessageMsg:
		m.addSystemMessage(msg.level, msg.context, msg.message)
		// Debug log to verify system messages are being received
		if strings.Contains(msg.message, "MCP") {
			m.logStore.Add("system-debug", "TUI", fmt.Sprintf("Received MCP system message: %s", msg.message), false)
		}
		cmds = append(cmds, m.waitForUpdates())

	case tickMsg:
		// Continue ticking for periodic updates (e.g., browser status)
		cmds = append(cmds, m.tickCmd())

	case mcpConnectionMsg:
		m.handleMCPConnection(msg)
		if m.currentView == ViewMCPConnections {
			m.updateMCPConnectionsList()
		}
		cmds = append(cmds, m.waitForUpdates())

	case mcpActivityMsg:
		m.handleMCPActivity(msg)
		if m.currentView == ViewMCPConnections && m.selectedMCPClient != "" {
			m.updateMCPActivityView()
		}
		cmds = append(cmds, m.waitForUpdates())
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

	// Handle custom command dialog
	if m.showingCustomCommand {
		// Handle escape key to close dialog
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Back) {
			m.showingCustomCommand = false
			return m, nil
		}

		// Handle enter key to run command
		if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Enter) {
			command := strings.TrimSpace(m.customCommandInput.Value())
			if command != "" {
				m.showingCustomCommand = false
				// Parse the command and arguments
				parts := strings.Fields(command)
				if len(parts) > 0 {
					cmdName := parts[0]
					args := parts[1:]
					go func() {
						_, err := m.processMgr.StartCommand(command, cmdName, args)
						if err != nil {
							m.logStore.Add("system", "System", fmt.Sprintf("Error starting command: %v", err), true)
							m.updateChan <- logUpdateMsg{}
						}
					}()
					m.currentView = ViewProcesses
					m.updateProcessList()
					return m, m.waitForUpdates()
				}
			}
			return m, nil
		}

		// Update the text input
		newInput, cmd := m.customCommandInput.Update(msg)
		m.customCommandInput = newInput
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}

	switch m.currentView {
	case ViewWeb:
		// Handle web view specific keys
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "f":
				// Cycle through filters: all -> pages -> api -> images -> other -> all
				switch m.webFilter {
				case "all":
					m.webFilter = "pages"
				case "pages":
					m.webFilter = "api"
				case "api":
					m.webFilter = "images"
				case "images":
					m.webFilter = "other"
				case "other":
					m.webFilter = "all"
				default:
					m.webFilter = "all"
				}

				// Update the list with new filtered requests and reset selection
				requests := m.getFilteredRequests()
				m.updateWebRequestsList(requests)

				// Reset selection to first item if available
				if len(requests) > 0 {
					m.webRequestsList.Select(0)
					m.updateSelectedRequestFromList()
				}

				return m, nil
			case "up", "k":
				// Navigate up in request list - delegate to list component
				m.webRequestsList, _ = m.webRequestsList.Update(msg)
				m.updateSelectedRequestFromList()
				m.webAutoScroll = false // Disable auto-scroll when manually navigating
				return m, nil
			case "down", "j":
				// Navigate down in request list - delegate to list component
				m.webRequestsList, _ = m.webRequestsList.Update(msg)
				m.updateSelectedRequestFromList()
				m.webAutoScroll = false // Disable auto-scroll when manually navigating
				return m, nil
			case "enter":
				// Select request for detail view
				m.updateSelectedRequestFromList()
				return m, nil
			case "pgup":
				// Page up in web list, disable auto-scroll
				m.webAutoScroll = false
				m.webRequestsList, _ = m.webRequestsList.Update(msg)
				m.updateSelectedRequestFromList()
				return m, nil
			case "pgdown":
				// Page down in web list
				m.webRequestsList, _ = m.webRequestsList.Update(msg)
				m.updateSelectedRequestFromList()
				return m, nil
			case "end":
				// End key re-enables auto-scroll and goes to bottom
				m.webAutoScroll = true
				// Go to last item in list
				if len(m.webRequestsList.Items()) > 0 {
					m.webRequestsList.Select(len(m.webRequestsList.Items()) - 1)
					m.updateSelectedRequestFromList()
				}
				return m, nil
			case "home":
				// Home key goes to top and disables auto-scroll
				m.webAutoScroll = false
				m.webRequestsList.Select(0)
				m.updateSelectedRequestFromList()
				return m, nil
			}
		}

		// Handle mouse wheel for scrolling
		if msg, ok := msg.(tea.MouseMsg); ok {
			switch msg.Type {
			case tea.MouseWheelUp:
				m.webAutoScroll = false
				// Move selection up in list
				if m.webRequestsList.Index() > 0 {
					m.webRequestsList.CursorUp()
					m.updateSelectedRequestFromList()
				}
				return m, nil
			case tea.MouseWheelDown:
				// Move selection down in list
				if m.webRequestsList.Index() < len(m.webRequestsList.Items())-1 {
					m.webRequestsList.CursorDown()
					m.updateSelectedRequestFromList()
				}
				return m, nil
			}
		}

	case ViewProcesses:
		// Handle process-specific key commands BEFORE list update
		// This ensures our keys take precedence over list navigation
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(msg, m.keys.Stop):
				if i, ok := m.processesList.SelectedItem().(processItem); ok && !i.isHeader && i.process != nil {
					if err := m.processMgr.StopProcess(i.process.ID); err != nil {
						msg := fmt.Sprintf("Failed to stop process %s: %v", i.process.Name, err)
						m.logStore.Add("system", "System", msg, true)
						m.addSystemMessage("error", "Process Control", msg)
					} else {
						msg := fmt.Sprintf("Stopping process: %s", i.process.Name)
						m.logStore.Add("system", "System", msg, false)
						m.addSystemMessage("info", "Process Control", msg)
					}
					cmds = append(cmds, m.waitForUpdates())
				} else {
					msg := "No process selected to stop"
					m.logStore.Add("system", "System", msg, true)
					m.addSystemMessage("error", "Process Control", msg)
				}
				// Don't update the list for this key, we handled it
				return m, tea.Batch(cmds...)

			case key.Matches(msg, m.keys.Restart):
				if i, ok := m.processesList.SelectedItem().(processItem); ok && !i.isHeader && i.process != nil {
					cmds = append(cmds, m.handleRestartProcess(i.process))
					msg := fmt.Sprintf("Restarting process: %s", i.process.Name)
					m.logStore.Add("system", "System", msg, false)
					m.addSystemMessage("info", "Process Control", msg)
				} else {
					msg := "No process selected to restart"
					m.logStore.Add("system", "System", msg, true)
					m.addSystemMessage("error", "Process Control", msg)
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
		// Handle manual scrolling
		if m.currentView == ViewLogs {
			if msg, ok := msg.(tea.KeyMsg); ok {
				switch {
				case key.Matches(msg, m.keys.Up):
					// Disable auto-scroll when user scrolls up
					m.logsAutoScroll = false
					m.logsViewport.LineUp(1)
					return m, nil
				case key.Matches(msg, m.keys.Down):
					m.logsViewport.LineDown(1)
					// Check if we're at the bottom
					if m.logsViewport.AtBottom() {
						m.logsAutoScroll = true
					}
					return m, nil
				case msg.String() == "pgup":
					m.logsAutoScroll = false
					m.logsViewport.ViewUp()
					return m, nil
				case msg.String() == "pgdown":
					m.logsViewport.ViewDown()
					if m.logsViewport.AtBottom() {
						m.logsAutoScroll = true
					}
					return m, nil
				case msg.String() == "end":
					// End key re-enables auto-scroll and goes to bottom
					m.logsAutoScroll = true
					m.logsViewport.GotoBottom()
					return m, nil
				case msg.String() == "home":
					// Home key goes to top and disables auto-scroll
					m.logsAutoScroll = false
					m.logsViewport.GotoTop()
					return m, nil
				}
			}

			// Handle mouse wheel
			if msg, ok := msg.(tea.MouseMsg); ok {
				if msg.Type == tea.MouseWheelUp {
					m.logsAutoScroll = false
					m.logsViewport.LineUp(3)
					return m, nil
				} else if msg.Type == tea.MouseWheelDown {
					m.logsViewport.LineDown(3)
					if m.logsViewport.AtBottom() {
						m.logsAutoScroll = true
					}
					return m, nil
				}
			}
		}

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
				} else if i, ok := m.settingsList.SelectedItem().(proxyInfoItem); ok {
					// Copy PAC URL to clipboard
					if err := copyToClipboard(i.pacURL); err != nil {
						m.logStore.Add("system", "System", fmt.Sprintf("Failed to copy PAC URL: %v", err), true)
					} else {
						m.logStore.Add("system", "System", "PAC URL copied to clipboard", false)
					}
				} else if i, ok := m.settingsList.SelectedItem().(mcpServerInfoItem); ok {
					// Copy MCP server URL to clipboard
					url := fmt.Sprintf("http://localhost:%d", i.port)
					if err := copyToClipboard(url); err != nil {
						m.logStore.Add("system", "System", fmt.Sprintf("Failed to copy MCP URL: %v", err), true)
					} else {
						m.logStore.Add("system", "System", "MCP server URL copied to clipboard", false)
					}
				} else if i, ok := m.settingsList.SelectedItem().(infoDisplayItem); ok && i.copyable {
					// Copy value to clipboard
					if err := copyToClipboard(i.value); err != nil {
						m.logStore.Add("system", "System", fmt.Sprintf("Failed to copy %s: %v", i.title, err), true)
					} else {
						m.logStore.Add("system", "System", fmt.Sprintf("%s copied to clipboard", i.title), false)
					}
				}
			}
		}

	case ViewMCPConnections:
		if m.debugMode {
			// Update connections list
			newList, cmd := m.mcpConnectionsList.Update(msg)
			m.mcpConnectionsList = newList
			cmds = append(cmds, cmd)

			// Update activity viewport
			newViewport, cmd := m.mcpActivityViewport.Update(msg)
			m.mcpActivityViewport = newViewport
			cmds = append(cmds, cmd)

			// Handle selection
			if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, m.keys.Enter) {
				if i, ok := m.mcpConnectionsList.SelectedItem().(mcpConnectionItem); ok {
					m.selectedMCPClient = i.clientID
					// Update activity viewport with selected client's activity
					m.updateMCPActivityView()
				}
			}
		}

	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Special views that take over the entire screen
	if m.currentView == ViewScriptSelector {
		return m.renderScriptSelector()
	}
	if m.showingCommandWindow {
		return m.renderCommandWindow()
	}

	// Render main content with consistent layout
	return m.renderLayout(m.renderContent())
}

// renderLayout provides consistent layout for all views
func (m Model) renderLayout(content string) string {
	// Calculate content height
	contentHeight := m.height - 4 // Header (2) + Help (2)

	// Build main layout parts
	parts := []string{m.renderHeader()}

	// If system panel is expanded, show it instead of main content
	if m.systemPanelExpanded && len(m.systemMessages) > 0 {
		// Full screen mode - system panel takes most of the space
		errorPanelHeight := m.height - 8 // Leave room for header and help
		m.systemPanelViewport.Height = errorPanelHeight
		parts = append(parts, m.renderSystemPanel())
	} else {
		// Normal content - always use full height
		contentStyle := lipgloss.NewStyle().Height(contentHeight)
		parts = append(parts, contentStyle.Render(content))
	}

	// Add help at the bottom
	parts = append(parts, m.help.View(m.keys))

	// Join all parts
	mainLayout := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// If we have system messages in non-expanded mode, overlay them
	if len(m.systemMessages) > 0 && !m.systemPanelExpanded {
		return m.overlaySystemPanel(mainLayout)
	}

	return mainLayout
}

// renderContent renders the main content area based on current view
func (m Model) renderContent() string {
	if m.showingRunDialog {
		return m.renderRunDialog()
	}

	if m.showingCustomCommand {
		return m.renderCustomCommandDialog()
	}

	switch m.currentView {
	case ViewProcesses:
		return m.renderProcessesView()
	case ViewLogs:
		return m.renderLogsView()
	case ViewErrors:
		return m.renderErrorsViewSplit()
	case ViewURLs:
		return m.renderURLsView()
	case ViewWeb:
		return m.renderWebView()
	case ViewSettings:
		if m.showingFileBrowser {
			return m.renderFileBrowser()
		}
		return m.renderSettings()
	case ViewMCPConnections:
		if m.debugMode {
			return m.renderMCPConnections()
		}
		return m.renderSettings() // Fallback if not in debug mode
	case ViewFilters:
		return m.renderFiltersView()
	default:
		return "Unknown view"
	}
}

// getViewStatus returns status information for the current view
func (m Model) getViewStatus() string {
	switch m.currentView {
	case ViewProcesses:
		processes := m.processMgr.GetAllProcesses()
		running := 0
		for _, p := range processes {
			if p.Status == process.StatusRunning {
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

	default:
		return ""
	}
}

func (m *Model) updateSizes() {
	headerHeight := 3 // title + tabs + separator
	helpHeight := 3
	contentHeight := m.height - headerHeight - helpHeight

	m.processesList.SetSize(m.width, contentHeight)
	m.settingsList.SetSize(m.width, contentHeight)
	m.commandsList.SetSize(m.width, contentHeight)
	m.errorsList.SetSize(m.width/3, contentHeight) // Split view
	m.logsViewport.Width = m.width
	m.logsViewport.Height = contentHeight
	m.errorsViewport.Width = m.width
	m.errorsViewport.Height = contentHeight
	m.errorDetailView.Width = m.width * 2 / 3
	m.errorDetailView.Height = contentHeight
	m.urlsViewport.Width = m.width
	m.urlsViewport.Height = contentHeight
	// webRequestsList size is set in render functions as needed
}

func (m *Model) cycleView() {
	views := []View{ViewProcesses, ViewURLs, ViewLogs, ViewWeb, ViewErrors, ViewSettings}
	if m.debugMode {
		views = append(views, ViewMCPConnections)
	}
	for i, v := range views {
		if v == m.currentView {
			m.currentView = views[(i+1)%len(views)]
			// Update MCP connections list when switching to that view
			if m.currentView == ViewMCPConnections && m.debugMode {
				m.updateMCPConnectionsList()
			}
			break
		}
	}
}

func (m *Model) cyclePrevView() {
	views := []View{ViewProcesses, ViewURLs, ViewLogs, ViewWeb, ViewErrors, ViewSettings}
	if m.debugMode {
		views = append(views, ViewMCPConnections)
	}
	for i, v := range views {
		if v == m.currentView {
			// Go to previous view (with wrap-around)
			prevIndex := (i - 1 + len(views)) % len(views)
			m.currentView = views[prevIndex]
			// Update MCP connections list when switching to that view
			if m.currentView == ViewMCPConnections && m.debugMode {
				m.updateMCPConnectionsList()
			}
			break
		}
	}
}

// switchToView changes the current view and performs any necessary setup
func (m *Model) switchToView(view View) {
	m.currentView = view

	// Clear unread indicator for this view
	m.clearUnreadIndicator(view)

	// Perform view-specific initialization if needed
	switch view {
	case ViewLogs:
		// Ensure logs are updated
		m.updateLogsView()
	case ViewErrors:
		// Errors view updates automatically via subscription
	case ViewWeb:
		// Web view updates automatically via list component
	case ViewMCPConnections:
		if m.debugMode {
			// Initialize MCP connections list
			m.updateMCPConnectionsList()
		}
	}
}

func (m *Model) handleEnter() tea.Cmd {
	switch m.currentView {
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
	var collapsedEntries []logs.CollapsedLogEntry

	if len(m.searchResults) > 0 {
		// For search results, we still use regular log entries and convert them
		logEntries := m.searchResults
		collapsedEntries = m.convertToCollapsedEntries(logEntries)
	} else if m.showHighPriority {
		// For high priority, we still use regular log entries and convert them
		logEntries := m.logStore.GetHighPriority(30)
		collapsedEntries = m.convertToCollapsedEntries(logEntries)
	} else if m.selectedProcess != "" {
		collapsedEntries = m.logStore.GetByProcessCollapsed(m.selectedProcess)
	} else {
		collapsedEntries = m.logStore.GetAllCollapsed()
	}

	// Apply regex filters if set
	if m.showPattern != "" || m.hidePattern != "" {
		var filtered []logs.CollapsedLogEntry

		// Compile regex patterns
		var showRegex, hideRegex *regexp.Regexp
		var err error

		if m.showPattern != "" {
			showRegex, err = regexp.Compile(m.showPattern)
			if err != nil {
				// Invalid regex, show error in logs
				m.logStore.Add("system", "System", fmt.Sprintf("Invalid /show regex: %v", err), true)
				showRegex = nil
			}
		}

		if m.hidePattern != "" {
			hideRegex, err = regexp.Compile(m.hidePattern)
			if err != nil {
				// Invalid regex, show error in logs
				m.logStore.Add("system", "System", fmt.Sprintf("Invalid /hide regex: %v", err), true)
				hideRegex = nil
			}
		}

		// Apply filters
		for _, log := range collapsedEntries {
			// For /show: only include if pattern matches
			if showRegex != nil {
				if !showRegex.MatchString(log.Content) {
					continue
				}
			}

			// For /hide: exclude if pattern matches
			if hideRegex != nil {
				if hideRegex.MatchString(log.Content) {
					continue
				}
			}

			filtered = append(filtered, log)
		}

		collapsedEntries = filtered
	}

	var content strings.Builder

	// Check if we have any logs to display
	hasVisibleLogs := false
	for _, log := range collapsedEntries {
		if strings.TrimSpace(log.Content) != "" {
			hasVisibleLogs = true
			break
		}
	}

	if !hasVisibleLogs {
		// Show empty state message
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		emptyMessage := "No logs yet. Start processes with /run <script>. Use /show or /hide <pattern> to filter logs."
		content.WriteString(emptyStyle.Render(emptyMessage))
	} else {
		for _, log := range collapsedEntries {
			// Skip empty log entries (used for separation)
			if strings.TrimSpace(log.Content) == "" {
				continue
			}

			style := m.getLogStyle(log.LogEntry)

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
			var prefix string
			if log.IsCollapsed {
				// Show collapsed log with count and time range
				prefix = fmt.Sprintf("[%s-%s] %s: ",
					log.FirstSeen.Format("15:04:05"),
					log.LastSeen.Format("15:04:05"),
					log.ProcessName,
				)
			} else {
				// Show regular log entry
				prefix = fmt.Sprintf("[%s] %s: ",
					log.Timestamp.Format("15:04:05"),
					log.ProcessName,
				)
			}

			// Apply style only to the prefix, not the content
			content.WriteString(style.Render(prefix))
			content.WriteString(cleanContent)

			// If collapsed, add the count information
			if log.IsCollapsed {
				countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Faint(true)
				countText := fmt.Sprintf("  (repeated %d times)\r\n", log.Count)
				content.WriteString(countStyle.Render(countText))
			}
		}
	}

	m.logsViewport.SetContent(content.String())

	// Auto-scroll to bottom if enabled
	if m.logsAutoScroll {
		m.logsViewport.GotoBottom()
	}
}

// convertToCollapsedEntries converts regular log entries to collapsed entries
// This is used for search results and high priority logs that aren't natively collapsed
func (m *Model) convertToCollapsedEntries(logEntries []logs.LogEntry) []logs.CollapsedLogEntry {
	if len(logEntries) == 0 {
		return []logs.CollapsedLogEntry{}
	}

	result := make([]logs.CollapsedLogEntry, 0, len(logEntries))

	// Start with the first entry
	current := logs.CollapsedLogEntry{
		LogEntry:    logEntries[0],
		Count:       1,
		FirstSeen:   logEntries[0].Timestamp,
		LastSeen:    logEntries[0].Timestamp,
		IsCollapsed: false,
	}

	for i := 1; i < len(logEntries); i++ {
		entry := logEntries[i]

		// Check if this entry is identical to the current one (same process and content)
		if m.areLogsIdentical(current.LogEntry, entry) {
			// Increment count and update last seen timestamp
			current.Count++
			current.LastSeen = entry.Timestamp
			current.IsCollapsed = current.Count > 1
		} else {
			// Different log entry, save the current one and start a new one
			result = append(result, current)
			current = logs.CollapsedLogEntry{
				LogEntry:    entry,
				Count:       1,
				FirstSeen:   entry.Timestamp,
				LastSeen:    entry.Timestamp,
				IsCollapsed: false,
			}
		}
	}

	// Add the last entry
	result = append(result, current)

	return result
}

// areLogsIdentical checks if two log entries are identical for collapsing purposes
func (m *Model) areLogsIdentical(a, b logs.LogEntry) bool {
	// Consider logs identical if they have the same process and content
	// We ignore timestamp and ID since those will naturally be different
	return a.ProcessID == b.ProcessID &&
		a.ProcessName == b.ProcessName &&
		a.Content == b.Content &&
		a.Level == b.Level &&
		a.IsError == b.IsError
}

func (m Model) cleanLogContent(content string) string {
	// Keep the original content with ANSI codes
	cleaned := content

	// Handle different line ending styles - ensure proper line endings
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n") // Windows line endings -> Unix
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")   // Lone CR -> newline (for terminal resets)

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
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Use ordered list of views
	orderedViews := []View{ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewWeb, ViewSettings}
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
			if indicator, exists := m.unreadIndicators[viewType]; exists && indicator.Count > 0 {
				indicatorIcon = indicator.Icon
			} else {
				indicatorIcon = "" // No space when no indicator
			}

			// Format the tab
			var tab string
			if viewType == m.currentView {
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
	if m.showPattern != "" {
		title += fmt.Sprintf(" [Show: %s]", m.showPattern)
	}
	if m.hidePattern != "" {
		title += fmt.Sprintf(" [Hide: %s]", m.hidePattern)
	}

	header := lipgloss.NewStyle().Bold(true).Render(title)

	// Add auto-scroll indicator
	var scrollIndicator string
	if !m.logsAutoScroll {
		scrollStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
			Bold(true)
		scrollIndicator = scrollStyle.Render("⏸ PAUSED - Press End to resume auto-scroll")
	}

	// Combine header with scroll indicator
	headerContent := header
	if scrollIndicator != "" {
		headerContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			header,
			"  ",
			scrollIndicator,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerContent, m.logsViewport.View())
}

func (m Model) renderFiltersView() string {
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

func (m *Model) renderErrorsView() string {
	errorContexts := m.logStore.GetErrorContexts()

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent Errors") + "\n\n")

	if len(errorContexts) == 0 {
		// Fall back to simple errors if no contexts
		errors := m.logStore.GetErrors()
		if len(errors) == 0 {
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No errors detected yet. Use /clear errors to clear when errors appear"))
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
					content.WriteString(stackStyle.Render("  "+strings.TrimSpace(stackLine)) + "\n")
				}
			}

			// Additional context if available
			if len(errorCtx.Context) > 0 && len(errorCtx.Context) <= 5 {
				for _, ctxLine := range errorCtx.Context {
					if strings.TrimSpace(ctxLine) != "" {
						content.WriteString(contextStyle.Render("  "+strings.TrimSpace(ctxLine)) + "\n")
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

	// Header with count
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	content.WriteString(headerStyle.Render(fmt.Sprintf("🔗 Detected URLs (%d)", len(urls))) + "\n\n")

	if len(urls) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
		content.WriteString(emptyStyle.Render("No URLs detected yet. Start servers with /run <script>. Use /proxy or /toggle-proxy for URL management."))
	} else {
		urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("120")).Bold(true)
		metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
		originalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

		// Get proxy mappings to show labels
		var proxyMappings map[string]string // URL -> Label
		if m.proxyServer != nil {
			proxyMappings = make(map[string]string)
			for _, mapping := range m.proxyServer.GetURLMappings() {
				if mapping.Label != "" && mapping.Label != mapping.ProcessName {
					proxyMappings[mapping.TargetURL] = mapping.Label
				}
			}
		}

		// URLs are already deduplicated and sorted by the store
		for i, urlEntry := range urls {
			// Use proxy URL if available, otherwise original URL
			displayURL := urlEntry.URL
			isProxied := urlEntry.ProxyURL != ""
			if isProxied {
				displayURL = urlEntry.ProxyURL
			}

			// Get label if available
			var labelText string
			if label, hasLabel := proxyMappings[urlEntry.URL]; hasLabel {
				labelText = fmt.Sprintf(" %s", labelStyle.Render(fmt.Sprintf("(%s)", label)))
			}

			// Clean, single-line format: [process] URL (label) (time)
			content.WriteString(fmt.Sprintf("%s %s%s %s\n",
				processStyle.Render(fmt.Sprintf("[%s]", urlEntry.ProcessName)),
				urlStyle.Render(displayURL),
				labelText,
				timeStyle.Render(fmt.Sprintf("(%s)", urlEntry.Timestamp.Format("15:04:05"))),
			))

			// Show original URL if using proxy (more compact)
			if isProxied {
				content.WriteString(metaStyle.Render(fmt.Sprintf("   ↳ Original: %s", originalStyle.Render(urlEntry.URL))) + "\n")
			}

			// Add spacing between entries, but not after the last one
			if i < len(urls)-1 {
				content.WriteString("\n")
			}
		}
	}

	m.urlsViewport.SetContent(content.String())
	return m.urlsViewport.View()
}

func (m Model) renderWebView() string {
	if m.width < 100 {
		// For narrow screens, use the simple view
		return m.renderWebViewSimple()
	}

	// Split view: requests list on left, detail on right
	listWidth := m.width * 2 / 3
	detailWidth := m.width - listWidth - 3
	// Use standard content height calculation (header + help already handled by renderLayout)
	// Account for compact 2-line header within the list area
	contentHeight := m.height - 8

	// Update list and detail viewport sizes
	m.webRequestsList.SetSize(listWidth, contentHeight)
	m.webDetailViewport.Width = detailWidth
	m.webDetailViewport.Height = contentHeight

	// Get filtered requests and update list
	requests := m.getFilteredRequests()
	m.updateWebRequestsList(requests)

	// Update selected request from list
	m.updateSelectedRequestFromList()

	// Render detail panel
	detailContent := m.renderRequestDetail()
	m.webDetailViewport.SetContent(detailContent)

	// Create bordered views
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	listView := borderStyle.
		Width(listWidth).
		Height(contentHeight).
		Render(m.renderWebRequestsListWithHeader())

	detailView := borderStyle.
		Width(detailWidth).
		Height(contentHeight).
		Render(m.webDetailViewport.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, " ", detailView)
}

func (m Model) renderWebViewSimple() string {
	var content strings.Builder

	// Compact header: combine status + filter on one line, help + indicators on another
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeFilterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	// Line 1: Status + Filter
	var statusAndFilter strings.Builder
	if m.proxyServer != nil && m.proxyServer.IsRunning() {
		modeStr := "Full Proxy"
		if m.proxyServer.GetMode() == proxy.ProxyModeReverse {
			modeStr = "Reverse Proxy"
		}
		statusAndFilter.WriteString(statusStyle.Render(fmt.Sprintf("🟢 %s", modeStr)))
	} else {
		statusAndFilter.WriteString(statusStyle.Render("🔴 Proxy not running"))
		content.WriteString(statusAndFilter.String() + "\n")
		return content.String()
	}

	// Add filter to same line
	filters := []string{"all", "pages", "api", "images", "other"}
	var filterParts []string
	for _, filter := range filters {
		if filter == m.webFilter {
			filterParts = append(filterParts, activeFilterStyle.Render("["+filter+"]"))
		} else {
			filterParts = append(filterParts, filterStyle.Render(filter))
		}
	}
	filterText := " | Filter: " + strings.Join(filterParts, " ") + " (f)"
	if !m.webAutoScroll {
		filterText += " ⏸"
	}
	statusAndFilter.WriteString(filterText)
	content.WriteString(statusAndFilter.String() + "\n")

	// Line 2: Help + Indicators (compact)
	content.WriteString("↑/↓ navigate, Enter select | Indicators: ❌🔐📊\n")

	// Line 3: Separator
	content.WriteString(strings.Repeat("─", m.width) + "\n")

	// Calculate list height correctly
	// renderLayout gives us m.height - 6 for content (header + help already handled)
	// Our filter headers are WITHIN this content area, so subtract them
	totalContentHeight := m.height - 6 // renderLayout standard allocation
	filterHeaderLines := 3             // status+filter + help+indicators + separator (compact)
	listHeight := totalContentHeight - filterHeaderLines

	// Setup list size and update with filtered requests
	m.webRequestsList.SetSize(m.width, listHeight)
	requests := m.getFilteredRequests()
	m.updateWebRequestsList(requests)
	m.updateSelectedRequestFromList()

	// Add the list view
	content.WriteString(m.webRequestsList.View())

	return content.String()
}

func (m Model) renderRequestsList(requests []proxy.Request, width int) string {
	var content strings.Builder

	// Header with filter info and auto-scroll indicator
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	title := "Web Proxy Requests"
	if !m.webAutoScroll {
		scrollStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("235")).
			Padding(0, 1).
			Bold(true)
		scrollIndicator := scrollStyle.Render("⏸ PAUSED")
		title += " " + scrollIndicator
	}
	content.WriteString(headerStyle.Render(title) + "\n")

	// Filter buttons
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeFilterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	filters := []string{"all", "pages", "api", "images", "other"}
	var filterParts []string
	for _, filter := range filters {
		if filter == m.webFilter {
			filterParts = append(filterParts, activeFilterStyle.Render("["+filter+"]"))
		} else {
			filterParts = append(filterParts, filterStyle.Render(filter))
		}
	}
	filterLine := "Filter: " + strings.Join(filterParts, " ") + " (f to cycle)"
	if !m.webAutoScroll {
		filterLine += " ⏸"
	}
	content.WriteString(filterLine + "\n\n")

	// Proxy status
	if m.proxyServer != nil && m.proxyServer.IsRunning() {
		modeStr := "Full Proxy"
		if m.proxyServer.GetMode() == proxy.ProxyModeReverse {
			modeStr = "Reverse Proxy"
		}
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("🟢 "+modeStr) + "\n\n")
	}

	if len(requests) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("No matching requests"))
		return content.String()
	}

	// Requests table header
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	content.WriteString(headerStyle.Render("Time     St Method  URL") + "\n")
	content.WriteString(strings.Repeat("─", width-4) + "\n")

	// Show recent requests (limit for performance)
	startIdx := 0
	if len(requests) > 100 {
		startIdx = len(requests) - 100
	}

	for i := startIdx; i < len(requests); i++ {
		req := requests[i]

		// Highlight selected request
		isSelected := m.selectedRequest != nil && req.ID == m.selectedRequest.ID

		// Color code status
		var statusColor string
		switch {
		case req.StatusCode >= 200 && req.StatusCode < 300:
			statusColor = "82" // Green
		case req.StatusCode >= 300 && req.StatusCode < 400:
			statusColor = "220" // Yellow
		case req.StatusCode >= 400 && req.StatusCode < 500:
			statusColor = "208" // Orange
		case req.StatusCode >= 500:
			statusColor = "196" // Red
		default:
			statusColor = "245" // Gray
		}

		// Color code method
		var methodColor string
		switch req.Method {
		case "GET":
			methodColor = "82"
		case "POST":
			methodColor = "220"
		case "PUT", "PATCH":
			methodColor = "208"
		case "DELETE":
			methodColor = "196"
		default:
			methodColor = "245"
		}

		// Truncate URL for display
		urlStr := req.URL
		maxURLLen := width - 25
		if len(urlStr) > maxURLLen {
			urlStr = urlStr[:maxURLLen-3] + "..."
		}

		// Format line
		line := fmt.Sprintf("%s %s %s %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(req.StartTime.Format("15:04:05")),
			lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Bold(true).Render(fmt.Sprintf("%3d", req.StatusCode)),
			lipgloss.NewStyle().Foreground(lipgloss.Color(methodColor)).Render(fmt.Sprintf("%-6s", req.Method)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(urlStr),
		)

		// Add indicators
		if req.Error != "" {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(" ❌")
		}
		if req.HasAuth {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(" 🔐")
		}
		if req.HasTelemetry {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(" 📊")
		}

		// Highlight if selected
		if isSelected {
			line = lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(line)
		}

		content.WriteString(line + "\n")
	}

	// Navigation help
	content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑/↓ navigate, Enter select, f filter"))
	content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Indicators: ❌ error, 🔐 auth, 📊 telemetry"))

	return content.String()
}

func (m Model) renderRequestDetail() string {
	if m.selectedRequest == nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Select a request to view details")
	}

	req := *m.selectedRequest
	var content strings.Builder

	// Request header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	content.WriteString(headerStyle.Render("Request Details") + "\n\n")

	// Basic info
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	content.WriteString(labelStyle.Render("Method: ") + valueStyle.Render(req.Method) + "\n")
	content.WriteString(labelStyle.Render("URL: ") + valueStyle.Render(req.URL) + "\n")
	content.WriteString(labelStyle.Render("Status: ") + m.formatStatus(req.StatusCode) + "\n")
	content.WriteString(labelStyle.Render("Duration: ") + valueStyle.Render(fmt.Sprintf("%.0fms", req.Duration.Seconds()*1000)) + "\n")
	content.WriteString(labelStyle.Render("Time: ") + valueStyle.Render(req.StartTime.Format("15:04:05")) + "\n")
	content.WriteString(labelStyle.Render("Process: ") + valueStyle.Render(req.ProcessName) + "\n")

	if req.Size > 0 {
		content.WriteString(labelStyle.Render("Size: ") + valueStyle.Render(formatSize(req.Size)) + "\n")
	}

	if req.Error != "" {
		content.WriteString(labelStyle.Render("Error: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(req.Error) + "\n")
	}

	// Authentication section
	if req.HasAuth {
		content.WriteString("\n" + headerStyle.Render("🔐 Authentication") + "\n\n")
		content.WriteString(labelStyle.Render("Type: ") + valueStyle.Render(req.AuthType) + "\n")

		if req.JWTError != "" {
			content.WriteString(labelStyle.Render("JWT Error: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(req.JWTError) + "\n")
		} else if req.JWTClaims != nil && len(req.JWTClaims) > 0 {
			content.WriteString(labelStyle.Render("JWT Claims:") + "\n")

			// Display common JWT claims
			claimOrder := []string{"sub", "iss", "aud", "exp", "iat", "nbf", "jti", "email", "name", "role", "scope"}
			displayedClaims := make(map[string]bool)

			// Display known claims in order
			for _, key := range claimOrder {
				if value, exists := req.JWTClaims[key]; exists {
					formattedValue := fmt.Sprintf("%v", value)

					// Format timestamp claims
					if key == "exp" || key == "iat" || key == "nbf" {
						if numVal, ok := value.(float64); ok {
							t := time.Unix(int64(numVal), 0)
							formattedValue = fmt.Sprintf("%v (%s)", value, t.Format("2006-01-02 15:04:05"))
						}
					}

					content.WriteString(fmt.Sprintf("  %s: %s\n",
						lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(key),
						valueStyle.Render(formattedValue)))
					displayedClaims[key] = true
				}
			}

			// Display any remaining claims
			for key, value := range req.JWTClaims {
				if !displayedClaims[key] {
					content.WriteString(fmt.Sprintf("  %s: %v\n",
						lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(key),
						valueStyle.Render(fmt.Sprintf("%v", value))))
				}
			}
		}
	}

	// Telemetry section
	if req.HasTelemetry && req.Telemetry != nil {
		content.WriteString("\n" + headerStyle.Render("📊 Telemetry Data") + "\n\n")
		content.WriteString(m.renderTelemetryDetails(req.Telemetry))
	} else if m.isPageRequest(req) {
		content.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No telemetry data available\n(Page may not have loaded completely)") + "\n")
	}

	return content.String()
}

func (m Model) formatStatus(status int) string {
	var color string
	switch {
	case status >= 200 && status < 300:
		color = "82" // Green
	case status >= 300 && status < 400:
		color = "220" // Yellow
	case status >= 400 && status < 500:
		color = "208" // Orange
	case status >= 500:
		color = "196" // Red
	default:
		color = "245" // Gray
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(fmt.Sprintf("%d", status))
}

func (m Model) renderTelemetryDetails(session *proxy.PageSession) string {
	var content strings.Builder

	if len(session.Events) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No telemetry events recorded"))
		return content.String()
	}

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Summary stats
	var pageLoadTime, domReadyTime float64
	var jsErrors, consoleLogs, interactions int
	var hasMemoryData bool

	for _, event := range session.Events {
		switch event.Type {
		case proxy.TelemetryPageLoad:
			if timing, ok := event.Data["timing"].(map[string]interface{}); ok {
				if domComplete, ok := timing["domComplete"].(float64); ok {
					domReadyTime = domComplete
				}
				if loadEventEnd, ok := timing["loadEventEnd"].(float64); ok {
					pageLoadTime = loadEventEnd
				}
			}
		case proxy.TelemetryJSError, proxy.TelemetryUnhandledReject:
			jsErrors++
		case proxy.TelemetryConsoleOutput:
			consoleLogs++
		case proxy.TelemetryUserInteraction:
			interactions++
		case proxy.TelemetryMemoryUsage:
			hasMemoryData = true
		}
	}

	// Display summary
	content.WriteString(labelStyle.Render("Events: ") + valueStyle.Render(fmt.Sprintf("%d", len(session.Events))) + "\n")

	if pageLoadTime > 0 {
		content.WriteString(labelStyle.Render("Page Load: ") + valueStyle.Render(fmt.Sprintf("%.0fms", pageLoadTime)) + "\n")
	}
	if domReadyTime > 0 {
		content.WriteString(labelStyle.Render("DOM Ready: ") + valueStyle.Render(fmt.Sprintf("%.0fms", domReadyTime)) + "\n")
	}
	if jsErrors > 0 {
		content.WriteString(labelStyle.Render("JS Errors: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("%d", jsErrors)) + "\n")
	}
	if consoleLogs > 0 {
		content.WriteString(labelStyle.Render("Console Logs: ") + valueStyle.Render(fmt.Sprintf("%d", consoleLogs)) + "\n")
	}
	if interactions > 0 {
		content.WriteString(labelStyle.Render("Interactions: ") + valueStyle.Render(fmt.Sprintf("%d", interactions)) + "\n")
	}
	if hasMemoryData {
		content.WriteString(labelStyle.Render("Memory Data: ") + lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("✓") + "\n")
	}

	// Show recent events
	content.WriteString("\n" + labelStyle.Render("Recent Events:") + "\n")

	// Get last few events
	startIdx := 0
	if len(session.Events) > 10 {
		startIdx = len(session.Events) - 10
	}

	for i := startIdx; i < len(session.Events); i++ {
		event := session.Events[i]
		eventTime := time.Unix(event.Timestamp/1000, (event.Timestamp%1000)*1000000)

		eventStr := fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(eventTime.Format("15:04:05")),
			m.formatTelemetryEvent(event),
		)
		content.WriteString("  " + eventStr + "\n")
	}

	return content.String()
}

func (m Model) formatTelemetryEvent(event proxy.TelemetryEvent) string {
	switch event.Type {
	case proxy.TelemetryPageLoad:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("Page Loaded")
	case proxy.TelemetryDOMState:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("DOM Ready")
	case proxy.TelemetryJSError:
		if msg, ok := event.Data["message"].(string); ok {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("JS Error: " + msg)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("JS Error")
	case proxy.TelemetryConsoleOutput:
		if level, ok := event.Data["level"].(string); ok {
			if msg, ok := event.Data["message"].(string); ok {
				return fmt.Sprintf("Console %s: %s", level, msg)
			}
			return fmt.Sprintf("Console %s", level)
		}
		return "Console Output"
	case proxy.TelemetryUserInteraction:
		if eventType, ok := event.Data["type"].(string); ok {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("User " + eventType)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("User Interaction")
	case proxy.TelemetryMemoryUsage:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("Memory Snapshot")
	default:
		return string(event.Type)
	}
}

// getFilteredRequests returns requests filtered by current filter
func (m Model) getFilteredRequests() []proxy.Request {
	if m.proxyServer == nil {
		return []proxy.Request{}
	}

	allRequests := m.proxyServer.GetRequests()
	if m.webFilter == "all" {
		return allRequests
	}

	var filtered []proxy.Request
	for _, req := range allRequests {
		switch m.webFilter {
		case "pages":
			if m.isPageRequest(req) {
				filtered = append(filtered, req)
			}
		case "api":
			if m.isAPIRequest(req) {
				filtered = append(filtered, req)
			}
		case "images":
			if m.isImageRequest(req) {
				filtered = append(filtered, req)
			}
		case "other":
			if !m.isPageRequest(req) && !m.isAPIRequest(req) && !m.isImageRequest(req) {
				filtered = append(filtered, req)
			}
		}
	}
	return filtered
}

// isPageRequest checks if request is for an HTML page
func (m Model) isPageRequest(req proxy.Request) bool {
	// XHR requests are never pages
	if req.IsXHR {
		return false
	}
	return strings.Contains(req.Path, ".html") || req.Path == "/" || (!strings.Contains(req.Path, ".") && !strings.Contains(req.Path, "/api/"))
}

// isAPIRequest checks if request is an API call
func (m Model) isAPIRequest(req proxy.Request) bool {
	// Check content type for response (if available)
	contentType := ""
	if req.Telemetry != nil && len(req.Telemetry.Events) > 0 {
		// Look for response headers in telemetry
		for _, event := range req.Telemetry.Events {
			if event.Type == "response" {
				if headers, ok := event.Data["headers"].(map[string]interface{}); ok {
					if ct, ok := headers["content-type"].(string); ok {
						contentType = ct
					}
				}
			}
		}
	}

	// Exclude HTML responses from API category
	if strings.Contains(contentType, "text/html") {
		return false
	}

	return strings.Contains(req.Path, "/api/") || strings.Contains(req.Path, "/graphql") ||
		req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" || req.Method == "PATCH"
}

// isImageRequest checks if request is for an image
func (m Model) isImageRequest(req proxy.Request) bool {
	return strings.HasSuffix(req.Path, ".jpg") || strings.HasSuffix(req.Path, ".jpeg") ||
		strings.HasSuffix(req.Path, ".png") || strings.HasSuffix(req.Path, ".gif") ||
		strings.HasSuffix(req.Path, ".webp") || strings.HasSuffix(req.Path, ".svg") ||
		strings.HasSuffix(req.Path, ".ico")
}

// updateSelectedRequest updates the selected request based on current index
func (m *Model) updateSelectedRequest() {
	// Delegate to the new list-based method
	m.updateSelectedRequestFromList()
}

// updateSelectedRequestFromList updates the selected request based on list selection
func (m *Model) updateSelectedRequestFromList() {
	if len(m.webRequestsList.Items()) == 0 {
		m.selectedRequest = nil
		return
	}

	selectedItem := m.webRequestsList.SelectedItem()
	if selectedItem == nil {
		m.selectedRequest = nil
		return
	}

	if proxyItem, ok := selectedItem.(proxyRequestItem); ok {
		m.selectedRequest = &proxyItem.Request
	}
}

// updateWebRequestsList updates the web requests list with filtered requests
func (m *Model) updateWebRequestsList(requests []proxy.Request) {
	// Convert requests to list items
	items := make([]list.Item, len(requests))
	for i, req := range requests {
		items[i] = proxyRequestItem{Request: req}
	}

	// Store current selection index before updating
	currentIndex := m.webRequestsList.Index()

	// Set the items in the list
	m.webRequestsList.SetItems(items)

	// Handle selection after items are updated
	if len(items) == 0 {
		// No items to select
		return
	}

	if m.webAutoScroll {
		// Auto-scroll: select last item
		m.webRequestsList.Select(len(items) - 1)
	} else {
		// Manual mode: try to maintain current selection or clamp to valid range
		if currentIndex >= len(items) {
			// If current index is out of bounds, select last item
			m.webRequestsList.Select(len(items) - 1)
		} else if currentIndex >= 0 {
			// Keep current selection if valid
			m.webRequestsList.Select(currentIndex)
		} else {
			// Default to first item
			m.webRequestsList.Select(0)
		}
	}
}

// renderWebRequestsListWithHeader renders the filter tabs and requests list for split view
func (m Model) renderWebRequestsListWithHeader() string {
	var content strings.Builder

	// Compact header: combine everything on minimal lines
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeFilterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	// Line 1: Filter + Status combined
	filters := []string{"all", "pages", "api", "images", "other"}
	var filterParts []string
	for _, filter := range filters {
		if filter == m.webFilter {
			filterParts = append(filterParts, activeFilterStyle.Render("["+filter+"]"))
		} else {
			filterParts = append(filterParts, filterStyle.Render(filter))
		}
	}

	// Line 1: Filter tabs
	filterLine := "Filter: " + strings.Join(filterParts, " ") + " (f)"
	if !m.webAutoScroll {
		filterLine += " ⏸"
	}
	content.WriteString(filterLine + "\n")

	// Line 2: Proxy status with count
	if m.proxyServer != nil && m.proxyServer.IsRunning() {
		modeStr := "Full Proxy"
		if m.proxyServer.GetMode() == proxy.ProxyModeReverse {
			modeStr = "Reverse Proxy"
		}

		// Get request count for display
		requests := m.getFilteredRequests()
		countStr := fmt.Sprintf("%d", len(requests))

		statusLine := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("🟢 " + modeStr + " - " + countStr)
		content.WriteString(statusLine + "\n")
	}

	// Add the list view
	content.WriteString(m.webRequestsList.View())

	return content.String()
}

// renderTelemetrySummary renders a one-line summary of telemetry data
func (m Model) renderTelemetrySummary(session *proxy.PageSession) string {
	if session == nil || len(session.Events) == 0 {
		return ""
	}

	// Extract key metrics from telemetry
	var loadTime, domReady float64
	var jsErrors, consoleLogs int
	var hasMemoryData, hasInteractions bool

	for _, event := range session.Events {
		switch event.Type {
		case proxy.TelemetryPageLoad:
			if timing, ok := event.Data["timing"].(map[string]interface{}); ok {
				if domComplete, ok := timing["domComplete"].(float64); ok {
					domReady = domComplete
				}
				if loadEventEnd, ok := timing["loadEventEnd"].(float64); ok {
					loadTime = loadEventEnd
				}
			}
		case proxy.TelemetryJSError, proxy.TelemetryUnhandledReject:
			jsErrors++
		case proxy.TelemetryConsoleOutput:
			consoleLogs++
		case proxy.TelemetryMemoryUsage:
			hasMemoryData = true
		case proxy.TelemetryUserInteraction:
			hasInteractions = true
		}
	}

	// Build summary line
	parts := []string{}
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))

	// Add timing info
	if domReady > 0 {
		parts = append(parts, fmt.Sprintf("DOM: %.0fms", domReady))
	}
	if loadTime > 0 {
		parts = append(parts, fmt.Sprintf("Load: %.0fms", loadTime))
	}

	// Add error count
	if jsErrors > 0 {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		parts = append(parts, errorStyle.Render(fmt.Sprintf("%d errors", jsErrors)))
	}

	// Add console log count
	if consoleLogs > 0 {
		parts = append(parts, fmt.Sprintf("%d logs", consoleLogs))
	}

	// Add indicators for other data
	if hasMemoryData {
		parts = append(parts, "mem")
	}
	if hasInteractions {
		parts = append(parts, "interactions")
	}

	if len(parts) > 0 {
		return "         " + detailStyle.Render("→ "+strings.Join(parts, " | "))
	}
	return ""
}

// formatSize formats bytes into human-readable format
func formatSize(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

type processUpdateMsg struct{}
type logUpdateMsg struct{}
type errorUpdateMsg struct{}
type webUpdateMsg struct{}
type systemMessageMsg struct {
	level   string
	context string
	message string
}
type tickMsg struct{}
type mcpActivityMsg struct {
	sessionId string
	activity  MCPActivity
}
type mcpConnectionMsg struct {
	sessionId      string
	clientInfo     string
	connected      bool
	connectedAt    time.Time
	connectionType string
	method         string
}

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

func (m *Model) handleMCPConnection(msg mcpConnectionMsg) {
	m.mcpActivityMu.Lock()
	defer m.mcpActivityMu.Unlock()

	if msg.connected {
		// Determine client name from user agent
		clientName := "Unknown Client"
		if msg.clientInfo != "" {
			// Extract a readable name from the user agent
			if strings.Contains(strings.ToLower(msg.clientInfo), "claude") {
				clientName = "Claude Desktop"
			} else if strings.Contains(strings.ToLower(msg.clientInfo), "vscode") {
				clientName = "VS Code MCP"
			} else if strings.Contains(strings.ToLower(msg.clientInfo), "test") {
				clientName = "Test Client"
			} else {
				clientName = msg.clientInfo
			}
		}

		m.mcpConnections[msg.sessionId] = &mcpConnectionItem{
			clientID:       msg.sessionId,
			clientName:     clientName,
			connectedAt:    msg.connectedAt,
			lastActivity:   msg.connectedAt,
			requestCount:   0,
			isConnected:    true,
			connectionType: msg.connectionType,
			method:         msg.method,
		}
		m.mcpActivities[msg.sessionId] = []MCPActivity{}
	} else {
		// Mark as disconnected
		if conn, exists := m.mcpConnections[msg.sessionId]; exists {
			conn.isConnected = false
		}
	}
}

func (m *Model) handleMCPActivity(msg mcpActivityMsg) {
	m.mcpActivityMu.Lock()
	defer m.mcpActivityMu.Unlock()

	// Update connection's last activity and request count
	if conn, exists := m.mcpConnections[msg.sessionId]; exists {
		conn.lastActivity = time.Now()
		conn.requestCount++
	} else {
		// Create a connection entry for sessions that only have activity (e.g., POST requests)
		// This ensures all sessions are tracked even if they don't establish persistent connections
		m.mcpConnections[msg.sessionId] = &mcpConnectionItem{
			clientID:       msg.sessionId,
			clientName:     "HTTP Client",
			connectedAt:    msg.activity.Timestamp,
			lastActivity:   msg.activity.Timestamp,
			requestCount:   1,
			isConnected:    false,  // Not a persistent connection
			connectionType: "HTTP", // Inferred from activity without connection event
			method:         "POST", // Most likely POST for activity-only sessions
		}
	}

	// Add activity to the session's history
	activities := m.mcpActivities[msg.sessionId]
	activities = append(activities, msg.activity)

	// Keep only last 100 activities per session
	if len(activities) > 100 {
		activities = activities[len(activities)-100:]
	}

	m.mcpActivities[msg.sessionId] = activities
}

func (m *Model) updateSettingsList() {
	installedMgrs := m.processMgr.GetInstalledPackageManagers()
	currentMgr := m.processMgr.GetCurrentPackageManager()

	items := make([]list.Item, 0)

	// Server Information section (prominently displayed at top)
	items = append(items, settingsSectionItem{title: "🔗 Server Information"})

	// MCP Server info
	mcpStatus := "🔴 Not Running"
	if m.mcpServer != nil && m.mcpServer.IsRunning() {
		mcpStatus = "🟢 Running"
	}

	// Get actual port from MCP server if running
	actualPort := m.mcpPort
	if m.mcpServer != nil && m.mcpServer.IsRunning() {
		actualPort = m.mcpServer.GetPort()
	}

	items = append(items, mcpServerInfoItem{
		port:   actualPort,
		status: mcpStatus,
	})

	// Add MCP endpoint information - always show URL for easy access
	mcpURL := fmt.Sprintf("http://localhost:%d/mcp", actualPort)
	items = append(items, infoDisplayItem{
		title:       "🔗 MCP Endpoint",
		description: fmt.Sprintf("%s (JSON-RPC 2.0 - all tools, resources & prompts)", mcpURL),
		value:       mcpURL,
		copyable:    true,
	})

	// Proxy Server info (if running and in full mode for PAC)
	if m.proxyServer != nil && m.proxyServer.IsRunning() && m.proxyServer.GetMode() == proxy.ProxyModeFull {
		items = append(items, proxyInfoItem{
			pacURL: m.proxyServer.GetPACURL(),
			mode:   m.proxyServer.GetMode(),
			port:   m.proxyServer.GetPort(),
		})
	}

	// Package Manager section
	items = append(items, settingsSectionItem{title: "📦 Package Managers"})
	for _, mgr := range installedMgrs {
		item := packageManagerSettingsItem{packageManagerItem{
			manager:  mgr,
			current:  mgr.Manager == currentMgr,
			fromJSON: m.processMgr.IsPackageManagerFromJSON(mgr.Manager),
		}}
		items = append(items, item)
	}

	// MCP Integration section
	items = append(items, settingsSectionItem{title: "🛠 MCP Integration"})
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
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Current Path: "+m.currentPath) + "\n\n")

	if len(m.fileList) == 0 {
		content.WriteString("No files found\n")
	} else {
		for i, item := range m.fileList {
			style := lipgloss.NewStyle()
			if i == 0 { // Simple selection highlight
				style = style.Background(lipgloss.Color("240"))
			}

			if item.IsDir {
				content.WriteString(style.Render("📁 "+item.Name) + "\n")
			} else if strings.HasSuffix(strings.ToLower(item.Name), ".json") {
				content.WriteString(style.Render("📄 "+item.Name+" (JSON)") + "\n")
			} else {
				content.WriteString(style.Foreground(lipgloss.Color("245")).Render("📄 "+item.Name) + "\n")
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

func (m *Model) handleClearScreen() {
	switch m.currentView {
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
		// Show custom command input dialog
		m.showingCustomCommand = true
		m.customCommandInput = textinput.New()
		m.customCommandInput.Placeholder = "Enter command to run (e.g., npm test, python script.py)"
		m.customCommandInput.Focus()
		m.customCommandInput.CharLimit = 200
		m.customCommandInput.Width = 60
		return nil
	}

	return nil
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

func (m Model) renderCustomCommandDialog() string {
	var content strings.Builder

	// Create a dialog box style
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(70)

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226"))
	title := titleStyle.Render("🚀 Run Custom Command")

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("Enter command to run | Run: Enter | Cancel: Esc")

	// Build dialog content
	dialogContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		instructions,
		"",
		m.customCommandInput.View(),
	)

	// Apply dialog styling
	dialog := dialogStyle.Render(dialogContent)

	// Center the dialog
	width, height := m.width, m.height
	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := lipgloss.Height(dialog)

	// Calculate padding for centering
	horizontalPadding := (width - dialogWidth) / 2
	verticalPadding := (height - dialogHeight) / 2

	// Add padding
	if horizontalPadding > 0 {
		dialog = lipgloss.NewStyle().MarginLeft(horizontalPadding).Render(dialog)
	}
	if verticalPadding > 0 {
		dialog = lipgloss.NewStyle().MarginTop(verticalPadding).Render(dialog)
	}

	content.WriteString(dialog)

	return content.String()
}

func (m *Model) updateErrorsList() {
	errorContexts := m.logStore.GetErrorContexts()
	newCount := len(errorContexts)

	items := make([]list.Item, 0, len(errorContexts))
	for i := len(errorContexts) - 1; i >= 0; i-- {
		items = append(items, errorItem{errorCtx: &errorContexts[i]})
	}

	m.errorsList.SetItems(items)

	// Update unread indicator if we have new errors
	if newCount > m.lastErrorCount && m.currentView != ViewErrors {
		m.updateUnreadIndicator(ViewErrors, "error", newCount-m.lastErrorCount)
	}
	m.lastErrorCount = newCount

	// Select first item if we have errors and nothing selected
	if len(items) > 0 && m.selectedError == nil {
		if item, ok := items[0].(errorItem); ok {
			m.selectedError = item.errorCtx
			m.updateErrorDetailView()
		}
	}
}

func (m *Model) updateWebView() {
	if m.proxyServer == nil {
		return
	}

	// Check for new requests
	requests := m.proxyServer.GetRequests()
	newCount := len(requests)

	// Update unread indicator if we have new requests with errors
	if newCount > m.lastWebCount && m.currentView != ViewWeb {
		// Check if any of the new requests are errors
		hasError := false
		for i := m.lastWebCount; i < newCount; i++ {
			if requests[i].IsError {
				hasError = true
				break
			}
		}

		severity := "info"
		if hasError {
			severity = "error"
		}
		m.updateUnreadIndicator(ViewWeb, severity, newCount-m.lastWebCount)
	}
	m.lastWebCount = newCount

	// Update the web requests list with latest proxy requests
	requests = m.getFilteredRequests()
	m.updateWebRequestsList(requests)

	// Auto-scroll to bottom if enabled
	if m.webAutoScroll && len(m.webRequestsList.Items()) > 0 {
		m.webRequestsList.Select(len(m.webRequestsList.Items()) - 1)
		m.updateSelectedRequestFromList()
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
	contentHeight := m.height - 5          // Adjust for header

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

func (m *Model) handleSlashCommand(input string) {
	// Clear previous search results and filters
	m.searchResults = nil
	m.showPattern = ""
	m.hidePattern = ""

	// Parse the command
	input = strings.TrimSpace(input)
	// Parse the command
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]

	switch command {
	case "/show":
		if len(parts) < 2 {
			return
		}
		m.showPattern = strings.Join(parts[1:], " ")
	case "/hide":
		if len(parts) < 2 {
			return
		}
		m.hidePattern = strings.Join(parts[1:], " ")
	case "/run":
		if len(parts) < 2 {
			return
		}
		scriptName := parts[1]
		// Execute the script
		go func() {
			_, err := m.processMgr.StartScript(scriptName)
			if err != nil {
				m.logStore.Add("system", "System", fmt.Sprintf("Error starting script %s: %v", scriptName, err), true)
				m.updateChan <- logUpdateMsg{}
			} else {
				m.updateChan <- processUpdateMsg{}
			}
		}()
		// Switch to logs view immediately
		m.currentView = ViewLogs
	case "/restart":
		processName := ""
		if len(parts) >= 2 {
			processName = parts[1]
		} else {
			processName = "all"
		}

		if processName == "all" {
			// Restart all running processes
			go func() {
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
				m.logStore.Add("system", "System", fmt.Sprintf("🔄 Restarted %d processes", restarted), false)
				m.updateChan <- processUpdateMsg{}
			}()
		} else {
			// Restart specific process
			go func() {
				// Find the process
				var targetProc *process.Process
				for _, proc := range m.processMgr.GetAllProcesses() {
					if proc.Name == processName && proc.Status == process.StatusRunning {
						targetProc = proc
						break
					}
				}

				if targetProc == nil {
					m.logStore.Add("system", "System", fmt.Sprintf("Process '%s' is not running", processName), true)
					m.updateChan <- logUpdateMsg{}
					return
				}

				// Stop and restart the process
				if err := m.processMgr.StopProcess(targetProc.ID); err != nil {
					m.logStore.Add("system", "System", fmt.Sprintf("Error stopping process %s: %v", processName, err), true)
					m.updateChan <- logUpdateMsg{}
					return
				}

				_, err := m.processMgr.StartScript(processName)
				if err != nil {
					m.logStore.Add("system", "System", fmt.Sprintf("Error restarting script %s: %v", processName, err), true)
				} else {
					m.logStore.Add("system", "System", fmt.Sprintf("🔄 Restarted process: %s", processName), false)
				}
				m.updateChan <- processUpdateMsg{}
			}()
		}
		m.currentView = ViewProcesses

	case "/stop":
		processName := ""
		if len(parts) >= 2 {
			processName = parts[1]
		} else {
			processName = "all"
		}

		if processName == "all" {
			// Stop all running processes
			go func() {
				processes := m.processMgr.GetAllProcesses()
				stopped := 0
				for _, proc := range processes {
					if proc.Status == process.StatusRunning {
						if err := m.processMgr.StopProcess(proc.ID); err != nil {
							m.logStore.Add("system", "System", fmt.Sprintf("Error stopping process %s: %v", proc.Name, err), true)
						} else {
							stopped++
						}
					}
				}
				m.logStore.Add("system", "System", fmt.Sprintf("⏹️ Stopped %d processes", stopped), false)
				m.updateChan <- processUpdateMsg{}
			}()
		} else {
			// Stop specific process
			go func() {
				// Find the process
				var targetProc *process.Process
				for _, proc := range m.processMgr.GetAllProcesses() {
					if proc.Name == processName && proc.Status == process.StatusRunning {
						targetProc = proc
						break
					}
				}

				if targetProc == nil {
					m.logStore.Add("system", "System", fmt.Sprintf("Process '%s' is not running", processName), true)
					m.updateChan <- logUpdateMsg{}
					return
				}

				if err := m.processMgr.StopProcess(targetProc.ID); err != nil {
					m.logStore.Add("system", "System", fmt.Sprintf("Error stopping process %s: %v", processName, err), true)
				} else {
					m.logStore.Add("system", "System", fmt.Sprintf("⏹️ Stopped process: %s", processName), false)
				}
				m.updateChan <- processUpdateMsg{}
			}()
		}
		m.currentView = ViewProcesses

	case "/clear":
		target := ""
		if len(parts) >= 2 {
			target = parts[1]
		} else {
			target = "all"
		}

		switch target {
		case "all":
			m.logStore.ClearLogs()
			m.logStore.ClearErrors()
			if m.proxyServer != nil {
				m.proxyServer.ClearRequests()
			}
			m.logStore.Add("system", "System", "🗑️ Cleared all logs, errors, and web requests", false)

		case "logs":
			m.logStore.ClearLogs()
			m.logStore.Add("system", "System", "📝 Cleared all logs", false)

		case "errors":
			m.logStore.ClearErrors()
			m.logStore.Add("system", "System", "🗑️ Cleared all errors", false)

		case "web":
			if m.proxyServer != nil {
				m.proxyServer.ClearRequests()
				m.logStore.Add("system", "System", "🌐 Cleared all web requests", false)
			} else {
				m.logStore.Add("system", "System", "Proxy server is not enabled", true)
			}

		default:
			// Check if it's a script name
			if _, exists := m.processMgr.GetScripts()[target]; exists {
				m.logStore.ClearLogsForProcess(target)
				m.logStore.Add("system", "System", fmt.Sprintf("🗑️ Cleared logs for script: %s", target), false)
			} else {
				m.logStore.Add("system", "System", fmt.Sprintf("Invalid clear target: %s", target), true)
			}
		}

		// Update the logs view to reflect changes
		m.updateLogsView()
		// Also update errors list if errors were cleared
		if target == "all" || target == "errors" {
			m.updateErrorsList()
		}
		if m.currentView == ViewLogs || m.currentView == ViewErrors {
			// Stay in current view to see the clear message
		} else {
			m.currentView = ViewLogs
		}

	case "/proxy":
		if len(parts) < 2 {
			m.logStore.Add("system", "System", "Usage: /proxy <url> - Register an arbitrary URL for reverse proxy", true)
			return
		}

		urlStr := parts[1]
		// Validate URL format
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			m.logStore.Add("system", "System", "Error: URL must start with http:// or https://", true)
			return
		}

		// Check if proxy server is available and in reverse mode
		if m.proxyServer == nil {
			m.logStore.Add("system", "System", "Error: Proxy server is not enabled. Use --proxy-mode=reverse to enable.", true)
			return
		}

		if m.proxyServer.GetMode() != proxy.ProxyModeReverse {
			m.logStore.Add("system", "System", "Error: Proxy URL registration requires reverse proxy mode. Use /toggle-proxy to switch modes.", true)
			return
		}

		// Register the URL
		proxyResult := m.proxyServer.RegisterURL(urlStr, "manual")
		if proxyResult != urlStr {
			msg := fmt.Sprintf("🌐 Registered proxy: %s -> %s", urlStr, proxyResult)
			m.logStore.Add("custom-proxy", "manual", msg, false)
			// Update the proxy URL mapping for the URLs tab
			m.logStore.UpdateProxyURL(urlStr, proxyResult)
		} else {
			msg := fmt.Sprintf("🌐 Proxy ready for: %s", urlStr)
			m.logStore.Add("custom-proxy", "manual", msg, false)
		}

		// Switch to URLs view to show the result
		m.currentView = ViewURLs

	case "/toggle-proxy":
		if m.proxyServer == nil {
			m.logStore.Add("system", "System", "Error: Proxy server is not enabled", true)
			return
		}

		m.handleToggleProxy()

	default:
		// Unknown command, treat as search
		m.searchResults = m.logStore.Search(input)
	}
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
		m.currentView = ViewURLs
	}
}

func (m *Model) showCommandWindow() {
	m.showingCommandWindow = true
	scripts := m.processMgr.GetScripts()
	m.commandAutocomplete = NewCommandAutocompleteWithProcessManager(scripts, m.processMgr)
	m.commandAutocomplete.SetWidth(min(60, m.width-10))
	// Force initial focus
	m.commandAutocomplete.Focus()
}

func (m *Model) handleCommandWindow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showingCommandWindow = false
		return m, nil

	case "backspace":
		if m.commandAutocomplete.Value() == "" {
			m.showingCommandWindow = false
			return m, nil
		}

	case "enter":
		// If there's a selected suggestion, apply it first
		if m.commandAutocomplete.showDropdown && len(m.commandAutocomplete.suggestions) > 0 {
			m.commandAutocomplete.applySelectedSuggestion()
		}

		// Validate the command first
		if valid, errMsg := m.commandAutocomplete.ValidateInput(); !valid {
			// Set error message in the autocomplete component
			m.commandAutocomplete.errorMessage = errMsg
			return m, nil
		}

		// Execute the command
		value := m.commandAutocomplete.Value()
		// Add slash prefix if not present
		if !strings.HasPrefix(value, "/") && value != "" {
			value = "/" + value
		}
		m.handleSlashCommand(value)
		m.showingCommandWindow = false
		m.updateLogsView()
		return m, nil
	}

	// Let the autocomplete component handle the update
	var cmd tea.Cmd
	m.commandAutocomplete, cmd = m.commandAutocomplete.Update(msg)

	return m, cmd
}

func (m Model) renderCommandWindow() string {
	// Safety check for minimum dimensions
	if m.width < 20 || m.height < 10 {
		// Just return empty string if window is too small
		return ""
	}

	// Create the command window
	windowWidth := min(60, m.width-10)
	maxSuggestions := 10

	windowStyle := lipgloss.NewStyle().
		Width(windowWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(1, 2)

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		MarginBottom(1)

	title := titleStyle.Render("Command Palette")

	// Input
	inputStyle := lipgloss.NewStyle().
		Width(windowWidth - 6).
		MarginBottom(1)

	inputView := inputStyle.Render(m.commandAutocomplete.View())

	// Get the dropdown suggestions
	dropdownView := m.commandAutocomplete.RenderDropdown(maxSuggestions)

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginTop(1)

	helpText := helpStyle.Render("↑↓ Navigate • Tab/Enter Select • Esc Cancel")

	// Error message if any
	errorMsg := m.commandAutocomplete.GetErrorMessage()
	if errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			MarginTop(1)
		errorView := errorStyle.Render("⚠ " + errorMsg)
		inputView = lipgloss.JoinVertical(lipgloss.Left, inputView, errorView)
	}

	// Combine all elements
	var contentParts []string
	contentParts = append(contentParts, title)
	contentParts = append(contentParts, inputView)
	if dropdownView != "" && errorMsg == "" {
		contentParts = append(contentParts, dropdownView)
	}
	contentParts = append(contentParts, helpText)

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)
	window := windowStyle.Render(content)

	// Create a full-screen overlay with the centered window
	overlay := lipgloss.Place(
		m.width, m.height-7, // Account for header and help
		lipgloss.Center, lipgloss.Center,
		window,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("236")), // Dim background
	)

	// Return the complete view with header and help
	helpView := m.help.View(m.keys)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(),
		overlay,
		helpView,
	)
}

func (m *Model) handleScriptSelector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Exit the application
		return m, tea.Quit

	case "/":
		// Switch to command window
		m.showCommandWindow()
		return m, nil

	case "ctrl+n":
		// Switch to arbitrary command mode
		m.scriptSelector.input.SetValue("")
		m.scriptSelector.input.Placeholder = "Type any command (e.g., 'ls', 'node server.js')..."
		m.scriptSelector.suggestions = []string{}
		m.scriptSelector.showDropdown = false
		m.scriptSelector.errorMessage = ""
		// Set a flag to indicate we're in arbitrary command mode
		m.scriptSelector.arbitraryMode = true
		return m, nil

	case "ctrl+s":
		// Skip script selection and go directly to logs view
		m.currentView = ViewLogs
		return m, nil

	case "enter":
		if m.scriptSelector.arbitraryMode {
			// In arbitrary mode, run any command
			command := strings.TrimSpace(m.scriptSelector.input.Value())
			if command == "" {
				m.scriptSelector.errorMessage = "Please enter a command to run"
				return m, nil
			}

			// Start the arbitrary command
			go func() {
				_, err := m.processMgr.StartCommand("custom", command, []string{})
				if err != nil {
					m.logStore.Add("system", "System", fmt.Sprintf("Error starting command '%s': %v", command, err), true)
					m.updateChan <- logUpdateMsg{}
				} else {
					m.updateChan <- processUpdateMsg{}
				}
			}()

			// Switch to logs view
			m.currentView = ViewLogs
			return m, nil
		}

		// Regular script mode
		var scriptName string
		if len(m.scriptSelector.suggestions) > 0 && m.scriptSelector.selected < len(m.scriptSelector.suggestions) {
			scriptName = m.scriptSelector.suggestions[m.scriptSelector.selected]
		} else if m.scriptSelector.input.Value() != "" {
			// Check if the typed value is a valid script
			inputValue := m.scriptSelector.input.Value()
			if _, exists := m.scriptSelector.availableScripts[inputValue]; exists {
				scriptName = inputValue
			} else {
				// Set error message
				m.scriptSelector.errorMessage = fmt.Sprintf("Script '%s' not found", inputValue)
				return m, nil
			}
		} else {
			// No selection and no input
			m.scriptSelector.errorMessage = "Please select a script to run"
			return m, nil
		}

		// Check if script is already running
		if m.scriptSelector.processMgr != nil {
			for _, proc := range m.scriptSelector.processMgr.GetAllProcesses() {
				if proc.Name == scriptName && proc.Status == process.StatusRunning {
					m.scriptSelector.errorMessage = fmt.Sprintf("Script '%s' is already running", scriptName)
					return m, nil
				}
			}
		}

		// Start the script
		go func() {
			_, err := m.processMgr.StartScript(scriptName)
			if err != nil {
				m.logStore.Add("system", "System", fmt.Sprintf("Error starting script %s: %v", scriptName, err), true)
				m.updateChan <- logUpdateMsg{}
			} else {
				m.updateChan <- processUpdateMsg{}
			}
		}()

		// Switch to logs view
		m.currentView = ViewLogs
		return m, nil

	case "up":
		if m.scriptSelector.showDropdown && m.scriptSelector.selected > 0 {
			m.scriptSelector.selected--
		}
		return m, nil

	case "down":
		if m.scriptSelector.showDropdown && m.scriptSelector.selected < len(m.scriptSelector.suggestions)-1 {
			m.scriptSelector.selected++
		}
		return m, nil

	default:
		// Update the input
		prevValue := m.scriptSelector.input.Value()
		var cmd tea.Cmd
		m.scriptSelector.input, cmd = m.scriptSelector.input.Update(msg)

		// Update suggestions if value changed
		if m.scriptSelector.input.Value() != prevValue {
			m.scriptSelector.updateScriptSelectorSuggestions()
		}

		return m, cmd
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// addSystemMessage adds a new system message to the bottom panel
func (m *Model) addSystemMessage(level, context, message string) {
	msg := SystemMessage{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Context:   context,
	}

	// Add to the beginning of the list (most recent first)
	m.systemMessages = append([]SystemMessage{msg}, m.systemMessages...)

	// Keep only the last 100 messages
	if len(m.systemMessages) > 100 {
		m.systemMessages = m.systemMessages[:100]
	}

	// Update the system panel viewport
	m.updateSystemPanelViewport()
}

// NOTE: HTTP errors from proxied requests are tracked in the Web tab,
// not in the system message panel. The system panel is only for internal
// Brummer messages (process control errors, settings errors, etc.)

// NOTE: JavaScript errors from telemetry are tracked in the Errors tab,
// not in the system message panel. The system panel is only for internal
// Brummer messages.

// getSystemMessageIcon returns the appropriate icon for a system message level
func (m Model) getSystemMessageIcon(level string) string {
	switch level {
	case "error":
		return "❌"
	case "warning":
		return "⚠️"
	case "success":
		return "✅"
	default:
		return "ℹ️"
	}
}

// updateUnreadIndicator updates the unread indicator for a specific view
func (m *Model) updateUnreadIndicator(view View, severity string, increment int) {
	if view == m.currentView {
		// Don't mark as unread if we're currently viewing this tab
		return
	}

	indicator := m.unreadIndicators[view]
	indicator.Count += increment

	// Update severity and icon based on priority
	if shouldUpdateSeverity(indicator.Severity, severity) {
		indicator.Severity = severity
		indicator.Icon = getIndicatorIcon(severity)
	}

	m.unreadIndicators[view] = indicator
}

// clearUnreadIndicator clears the unread indicator for a specific view
func (m *Model) clearUnreadIndicator(view View) {
	delete(m.unreadIndicators, view)
}

// shouldUpdateSeverity determines if the new severity is higher priority
func shouldUpdateSeverity(current, new string) bool {
	priority := map[string]int{
		"error":   4,
		"warning": 3,
		"success": 2,
		"info":    1,
		"":        0,
	}
	return priority[new] > priority[current]
}

// getIndicatorIcon returns the appropriate icon for a severity level
func getIndicatorIcon(severity string) string {
	switch severity {
	case "error":
		return "🔴"
	case "warning":
		return "🟡"
	case "success":
		return "🟢"
	case "info":
		return "🔵"
	default:
		return "⚪"
	}
}

// updateSystemPanelViewport updates the system panel viewport with formatted messages
func (m *Model) updateSystemPanelViewport() {
	if m.height == 0 {
		return
	}

	// Calculate viewport height
	height := m.height - 4 // Account for header and help
	if !m.systemPanelExpanded {
		height = 5 // Show only 5 lines when not expanded
	}

	m.systemPanelViewport.Width = m.width
	m.systemPanelViewport.Height = height

	// Format messages for display
	content := m.formatSystemMessagesForDisplay()
	m.systemPanelViewport.SetContent(content)
}

// formatSystemMessagesForDisplay formats system messages for the panel
func (m *Model) formatSystemMessagesForDisplay() string {
	if len(m.systemMessages) == 0 {
		return "No system messages"
	}

	var b strings.Builder

	// Determine how many messages to show
	messagesToShow := m.systemMessages
	if !m.systemPanelExpanded && len(m.systemMessages) > 5 {
		messagesToShow = m.systemMessages[:5]
	}

	// Format each message
	for i, msg := range messagesToShow {
		if i > 0 {
			b.WriteString("\n")
		}

		// Format timestamp
		timestamp := msg.Timestamp.Format("15:04:05")

		// Choose icon based on level
		icon := m.getSystemMessageIcon(msg.Level)

		// Build message line
		msgLine := fmt.Sprintf("[%s] %s %s: %s",
			timestamp,
			icon,
			msg.Context,
			msg.Message,
		)

		b.WriteString(msgLine)
	}

	// Add count if not showing all messages
	if !m.systemPanelExpanded && len(m.systemMessages) > 5 {
		b.WriteString(fmt.Sprintf("\n... and %d more messages (press 'e' to expand, 'm' to clear)", len(m.systemMessages)-5))
	} else if len(m.systemMessages) > 0 {
		// Add clear hint when showing all messages
		b.WriteString("\n(Press 'm' to clear messages)")
	}

	return b.String()
}

// overlaySystemPanel overlays the system panel on top of the main content
func (m Model) overlaySystemPanel(mainContent string) string {
	// Split main content into lines
	lines := strings.Split(mainContent, "\n")

	// Calculate panel height (5 messages + title + border = 8 lines)
	panelHeight := 8
	if len(m.systemMessages) < 5 {
		panelHeight = len(m.systemMessages) + 3 // messages + title + border
	}

	// Position panel at bottom, but above help (2 lines)
	startLine := len(lines) - panelHeight - 2
	if startLine < 2 { // Keep below header
		startLine = 2
	}

	// Render the panel
	panel := m.renderSystemPanelOverlay()
	panelLines := strings.Split(panel, "\n")

	// Overlay panel lines onto main content
	for i, panelLine := range panelLines {
		if startLine+i < len(lines)-2 { // Don't overlay help
			lines[startLine+i] = panelLine
		}
	}

	return strings.Join(lines, "\n")
}

// renderSystemPanelOverlay renders the system panel for overlay mode
func (m Model) renderSystemPanelOverlay() string {
	// Create a semi-transparent style with background
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(m.width - 2).
		Background(lipgloss.Color("235")) // Dark background for visibility

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Background(lipgloss.Color("235"))

	// Title
	title := titleStyle.Render("System Messages")

	// Get messages (max 5 for overlay)
	messageCount := len(m.systemMessages)
	start := 0
	if messageCount > 5 {
		start = messageCount - 5
	}

	// Format messages
	var lines []string
	for i := start; i < messageCount; i++ {
		msg := m.systemMessages[i]
		icon := m.getSystemMessageIcon(msg.Level)

		// Create message with background
		msgStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Width(m.width - 6) // Account for border and padding

		line := fmt.Sprintf("[%s] %s %s: %s",
			msg.Timestamp.Format("15:04:05"),
			icon,
			msg.Context,
			msg.Message)

		lines = append(lines, msgStyle.Render(line))
	}

	// Add hint with background
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Background(lipgloss.Color("235")).
		Width(m.width - 6)

	if messageCount > 5 {
		hint := fmt.Sprintf("... and %d more (press 'e' to expand, 'm' to clear)", messageCount-5)
		lines = append(lines, hintStyle.Render(hint))
	} else {
		lines = append(lines, hintStyle.Render("(Press 'm' to clear messages)"))
	}

	// Combine title and content
	content := strings.Join(lines, "\n")
	panel := lipgloss.JoinVertical(lipgloss.Left, title, content)

	return panelStyle.Render(panel)
}

// renderSystemPanel renders the system message panel at the bottom of the screen
func (m Model) renderSystemPanel() string {
	// Don't show if no messages
	if len(m.systemMessages) == 0 {
		return ""
	}

	// Create styles
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")). // Gray border
		Padding(0, 1).
		Width(m.width - 2)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true)

	// Title
	title := "System Messages"
	if m.systemPanelExpanded {
		title = fmt.Sprintf("All System Messages (%d)", len(m.systemMessages))
	}
	title = titleStyle.Render(title)

	// Content
	content := m.systemPanelViewport.View()

	// Combine title and content
	panel := lipgloss.JoinVertical(lipgloss.Left, title, content)

	return panelStyle.Render(panel)
}

// renderSettings provides an enhanced settings view with better UX design
func (m Model) renderSettings() string {
	var content strings.Builder

	// Header with branding and description
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1).
		Width(m.width)

	content.WriteString(headerStyle.Render("⚙️  Brummer Settings & Configuration"))
	content.WriteString("\n")

	// Subtitle with helpful information
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		MarginBottom(1)

	content.WriteString(subtitleStyle.Render("Configure your development environment and server settings • Press Enter to copy URLs"))
	content.WriteString("\n")

	// Calculate available height for the list
	headerHeight := 4                              // header + subtitle + margins
	availableHeight := m.height - 6 - headerHeight // standard layout minus our header

	// Update list size and render
	m.settingsList.SetSize(m.width, availableHeight)
	content.WriteString(m.settingsList.View())

	return content.String()
}
