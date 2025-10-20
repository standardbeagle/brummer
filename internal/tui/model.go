// @lci:labels[core,tui-model,terminal-ui,state-management,bubbletea]
// @lci:category[model]
// @lci:deps[process:manager:read,logs:store:read,mcp:server:read,proxy:server:read,events:bus:read]
// @lci:tags[component=tui-model,pattern=mvc,framework=bubbletea,interface=terminal]
// @lci:metrics[responsiveness=high,complexity=moderate]
// @lci:propagate[attr=ui-state,dir=bidirectional,decay=0.8]
package tui

import (
	"context"
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
	
	processes "github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/bridge"
	"github.com/standardbeagle/brummer/internal/tui/commands"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/internal/tui/state"
	"github.com/standardbeagle/brummer/pkg/events"
)

// Local View type based on messages.ViewType for compatibility
type View = messages.ViewType

// Re-export view constants for convenience
const (
	ViewScripts        View = "scripts"
	ViewProcesses      View = "processes"
	ViewLogs           View = "logs"
	ViewErrors         View = "errors"
	ViewURLs           View = "urls"
	ViewWeb            View = "web"
	ViewSettings       View = "settings"
	ViewMCPStatus      View = "mcp_status"
	ViewMCPConnections View = "mcp_connections"
	ViewSearch         View = "search"
	ViewFilters        View = "filters"
	ViewScriptSelector View = "script_selector"
	ViewAICoders       View = "aicoders"
)

// View configuration is now handled in model_views.go

// MCPServerInterface defines the methods needed by the TUI
type MCPServerInterface interface {
	IsRunning() bool
	GetPort() int
}

type Model struct {
	processMgr  *processes.Manager
	logStore    *logs.Store
	eventBus    *events.EventBus
	mcpServer   MCPServerInterface
	mcpPort     int
	proxyServer *proxy.Server
	debugMode   bool
	cfg         *config.Config // Configuration for AI coder and other features
	workingDir  string         // Working directory for the project

		width         int
	height        int

	// View components (migrated from controllers)
	processesView     *components.ProcessesViewComponent // Processes view component (stateless)
	logsView          *components.LogsViewComponent     // Logs view component (stateless)
	errorsView        *components.ErrorsViewComponent   // Errors view component (stateless)
	urlsView          *components.URLsViewComponentNew // URLs view component (stateless)
	webView           *components.WebViewComponentNew      // Web view component (stateless) (T056-T061)
	commandWindowView *components.CommandWindowViewComponentNew     // Command window component (stateless)
	commandAutocomplete components.CommandAutocompleteInterface // Command autocomplete instance (persistent)
	settingsView      *components.SettingsViewComponentNew // Settings view component (stateless)
	aicodersView      *components.AICodersViewComponent // AI Coders view component (Feature 005)
	// New component fields (replacing controllers)
	fileBrowserComponent   *components.FileBrowserComponent
	systemComponent         *components.SystemComponent
	notificationsComponent  *components.NotificationsComponent
	inputComponent          *components.InputComponent
	navigationComponent     *components.NavigationComponent

	// SIMPLIFIED VIEW MANAGEMENT - replaces NavigationComponent
	viewManager *SimpleViewManager

	selectedProcess string // Shared state - used by multiple views

	// Script selector component (T068-T073)
	scriptSelector *components.ScriptSelector

	// Layout components
	tabsComponent *TabsComponent
	contentLayout *ContentLayout
	headerHeight  int // Calculated height of the header
	footerHeight  int // Calculated height of the footer

	keys keyMap

	updateChan chan tea.Msg

	// MCP view components
	mcpView      *components.MCPViewComponentNew  // MCP status view component (stateless)
	mcpDebugView *components.MCPDebugViewComponentNew // MCP debug view component (stateless)

	// AI Coder service (Feature 005 - new service-based architecture)
	aiCoderService interface{} // *services.AICoderService (interface{} to avoid import cycles during transition)

	// Message routing
	messageRouter *MessageRouter // Routes messages to appropriate handlers

	// SIMPLIFIED KEYBOARD HANDLING
	keyRegistry *KeyHandlerRegistry // Single registry for all keyboard handlers

	// CLEAR CONFIRMATION STATE (T026)
	awaitingClearConfirmation bool
	confirmationTab           string

	// ERROR GROUPING SERVICE (T027)
	errorGroupingService *services.ErrorGroupingService

	// STATE MANAGEMENT - Master data with sub-models for Elm/Redux compliance
	masterData   *state.MasterData   // Single source of truth with AICoderState sub-model
	stateManager *state.StateManager // Legacy StateManager (to be deprecated)
}

type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	Back          key.Binding
	Quit          key.Binding
	Tab           key.Binding
	Left          key.Binding
	Right         key.Binding
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
	ClearMessages    key.Binding
	ClearConfirmation key.Binding
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
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "previous view"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "next view"),
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
	ClearConfirmation: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "clear with confirmation"),
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
	process    *processes.Process
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
	case processes.StatusRunning:
		statusEmoji = "🟢"
	case processes.StatusStopped:
		statusEmoji = "✓" // Thin checkmark for gracefully stopped
	case processes.StatusFailed:
		statusEmoji = "❌"
	case processes.StatusSuccess:
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

func NewModel(processMgr *processes.Manager, logStore *logs.Store, eventBus *events.EventBus, mcpServer MCPServerInterface, proxyServer *proxy.Server, mcpPort int, cfg *config.Config, workingDir string) *Model {
	return NewModelWithView(processMgr, logStore, eventBus, mcpServer, proxyServer, mcpPort, ViewLogs, false, cfg, workingDir)
}

func NewModelWithView(processMgr *processes.Manager, logStore *logs.Store, eventBus *events.EventBus, mcpServer MCPServerInterface, proxyServer *proxy.Server, mcpPort int, initialView View, debugMode bool, cfg *config.Config, workingDir string) *Model {
	scripts := processMgr.GetScripts()

	// processesList initialization moved to ProcessViewController

	// Create settings list with package managers
	// settingsList initialization moved to SettingsController

	// DIAGNOSTIC: Log model initialization start
	logStore.Add("system", "Model", "🚀 DIAGNOSTIC: Starting Model initialization...", false)

	// Create the simplified view manager FIRST
	viewManager := NewSimpleViewManager(initialView)

	m := Model{
		processMgr:  processMgr,
		logStore:    logStore,
		eventBus:    eventBus,
		mcpServer:   mcpServer,
		mcpPort:     mcpPort,
		proxyServer: proxyServer,
		debugMode:   debugMode,
		cfg:         cfg,
		workingDir:  workingDir,
		// View components initialization
		processesView:     components.NewProcessesViewComponent(),
		logsView:          components.NewLogsViewComponent(),
		errorsView:        components.NewErrorsViewComponent(),
		urlsView:          components.NewURLsView(),
		webView:           components.NewWebViewComponentNew(),
		commandWindowView: components.NewCommandWindowViewNew(func(scripts map[string]string, processMgr interface{}) components.CommandAutocompleteInterface {
		if pm, ok := processMgr.(*processes.Manager); ok {
			return NewCommandAutocompleteFactory()(scripts, pm)
		}
		return nil
	}),
		commandAutocomplete: NewCommandAutocompleteFactory()(processMgr.GetScripts(), processMgr),
		keys:              keys,
		updateChan:        make(chan tea.Msg, UpdateChannelBufferSize),
		viewManager:       viewManager, // Add the simple view manager
	}

// DIAGNOSTIC: Log basic model creation complete
	logStore.Add("system", "Model", "✅ Model basic initialization complete", false)

	// Initialize STATE MANAGEMENT - Master data with sub-models
	m.masterData = state.NewMasterData()

	// Legacy StateManager (to be deprecated)
	m.stateManager = state.NewStateManager()

	// Note: Log callback is registered in main.go to avoid duplication

	// Create the service instances for the new component architecture
	// These services will be passed to the component constructors
	// and used by the ServiceBridge for event routing

	// MINIMAL SERVICES FOR TUI STARTUP DEBUGGING
	// Create only essential services to prevent nil pointer crashes

	// DIAGNOSTIC: Log service creation start
	logStore.Add("system", "Model", "🔧 DIAGNOSTIC: Creating minimal services...", false)

	// 1. Create ONLY InputService (essential for keyboard handling)
	inputService := services.NewInputService()
	logStore.Add("system", "Model", "✅ InputService created", false)

	// T027: Create ErrorGroupingService for error parsing
	m.errorGroupingService = services.NewErrorGroupingService()
	logStore.Add("system", "Model", "✅ ErrorGroupingService created", false)

	// Skip other services that might have blocking goroutines
	// var navigationService *services.NavigationService
	// var systemService *services.SystemService
	// var notificationsService *services.NotificationsService
	// var fileBrowserService *services.FileBrowserService

	// Create essential components
	var fileBrowserComponent *components.FileBrowserComponent
	var systemComponent *components.SystemComponent
	var notificationsComponent *components.NotificationsComponent
	var inputComponent *components.InputComponent
	var navigationComponent *components.NavigationComponent

	// Create input component since we have InputService
	inputComponent = components.NewInputComponent(inputService)

	// Create navigation component for view switching
	navService := services.NewNavigationService(eventBus)
	navigationComponent = components.NewNavigationComponent(navService)

	// 7. Initialize settings view (still using component)
	//    - Required by: Settings view display
	//    - Dependencies: None
	var err error
	workingDir, err = os.Getwd()
	if err != nil {
		workingDir = "." // fallback to current directory
	}
	m.settingsView = components.NewSettingsView()
	// Legacy InitializeWithLoadingState() call removed - new component is stateless

	// 8. Initialize process list with current processes
	//    - Required by: Process view display
	//    - Dependencies: None
	m.updateProcessList()

	// Error list and detail view now managed by ErrorsViewController

	// 9. Set up navigation callbacks using navigation component
	//    - Required by: View switching, unread indicators
	//    - Dependencies: navigationComponent created earlier
	if navigationComponent != nil {
		// Set up view change callback
		navigationComponent.RegisterTransitionCallback(func(fromView, toView messages.ViewType) error {
			// Clear unread indicators when switching views
			// Note: System component doesn't have ClearUnreadIndicator method
			// This functionality is now handled internally by the component
			return nil
		})
	}

	// 10. Layout controller migrated - layout logic now in components

	// 11. Input component replaces input controller
	//    - Required by: Keyboard handling
	//    - Dependencies: InputService
	// Note: The input component will handle keyboard input via BubbleTea Update() method
	// and communicate with InputService for state management

	// 12. Event controller migrated - event handling now in model Update() method

	// 13. Notifications component replaces notifications controller
	//     - Required by: User notifications
	//     - Dependencies: NotificationsService
	// Note: The notifications component will handle notification display via BubbleTea View() method
	// and communicate with NotificationsService for state management

	// Store references to the new components in the model struct fields
	// These will replace the old controller references once the migration is complete
	// For now, we keep both for compatibility during the transition
	m.fileBrowserComponent = fileBrowserComponent
	m.systemComponent = systemComponent
	m.notificationsComponent = notificationsComponent
	m.inputComponent = inputComponent
	m.navigationComponent = navigationComponent

	// DIAGNOSTIC: Final comprehensive status check
	logStore.Add("system", "Model", "📋 DIAGNOSTIC: Final component status check:", false)

	// Check critical components
	if m.inputComponent != nil {
		logStore.Add("system", "Model", "✅ InputComponent: Available", false)
	} else {
		logStore.Add("system", "Model", "❌ InputComponent: MISSING - KEYBOARD WILL NOT WORK", true)
	}

	if m.navigationComponent != nil {
		logStore.Add("system", "Model", "✅ NavigationComponent: Available", false)
	} else {
		logStore.Add("system", "Model", "❌ NavigationComponent: MISSING - VIEW SWITCHING WILL NOT WORK", true)
	}

	// Check services
	if inputService != nil && inputService.IsRunning() {
		logStore.Add("system", "Model", "✅ InputService: Running", false)
	} else {
		logStore.Add("system", "Model", "❌ InputService: NOT RUNNING - KEYBOARD PROCESSING WILL FAIL", true)
	}

	// Navigation service is now enabled
	logStore.Add("system", "Model", "✅ NavigationService: Running", false)

	// Check handler count
	finalHandlerCount := inputService.GetHandlerCount()
	logStore.Add("system", "Model", fmt.Sprintf("📊 Total keyboard handlers registered: %d", finalHandlerCount), false)

	logStore.Add("system", "Model", "🎯 DIAGNOSTIC: Model initialization complete - keyboard handling should be functional", false)

	// 14. Initialize MCP view components
	//     - Required by: MCP status and debugging features
	//     - Dependencies: debugMode flag
	m.mcpView = components.NewMCPViewComponent() // MCP status view
	m.mcpDebugView = components.NewMCPDebugView(debugMode) // MCP debug view

	// Unread indicators now managed by system controller

	// Check for monorepo on startup - this will be handled by command window controller when needed

	// Script selector is now handled by ScriptSelectorController

	// 12. Initialize MCP connections list on first view if in debug mode
	//     - Required by: Debug view (now stateless, no initialization needed)
	//     - Dependencies: Debug mode flag
	// Legacy UpdateConnectionsList() call removed - new component is stateless

	// 13. Initialize AI Coder service (T028 - new service-based architecture, replaces legacy controller)
	//     - Required by: AI coder features (new architecture)
	//     - Dependencies: context (for lifecycle management), logStore (for unified logging)
	ctx := context.Background() // TODO: Use proper context from application lifecycle
	m.aiCoderService = services.NewAICoderService(ctx, m.logStore)

	// 13.6. Subscribe to AI Coder service events (T028 - event routing)
	//     - Required by: Service-to-component communication
	//     - Dependencies: aiCoderService created, updateChan available
	if m.aiCoderService != nil {
		svc := m.aiCoderService.(*services.AICoderService)
		eventBus := svc.EventBus().(*events.EventBus)

		// Subscribe to events and forward to component as proper message types
		eventBus.Subscribe("aicoder.session.created", func(event events.Event) {
			// Extract session info from event
			sessionID := event.ProcessID
			name := ""
			if nameVal, ok := event.Data["name"]; ok {
				name = nameVal.(string)
			}
			// Forward as proper message type
			m.updateChan <- messages.AICoderSessionCreatedMsg{
				SessionID: sessionID,
				Name:      name,
			}
		})

		eventBus.Subscribe("aicoder.session.updated", func(event events.Event) {
			sessionID := event.ProcessID
			lastActivity := time.Now()
			m.updateChan <- messages.AICoderSessionUpdatedMsg{
				SessionID:    sessionID,
				Status:       messages.SessionActive,
				LastActivity: lastActivity,
			}
		})

		eventBus.Subscribe("aicoder.session.ended", func(event events.Event) {
			sessionID := event.ProcessID
			reason := ""
			if reasonVal, ok := event.Data["reason"]; ok {
				reason = reasonVal.(string)
			}
			m.updateChan <- messages.AICoderSessionEndedMsg{
				SessionID: sessionID,
				Reason:    reason,
			}
		})

		eventBus.Subscribe("aicoder.pty.output", func(event events.Event) {
			sessionID := event.ProcessID
			output := ""
			if outputVal, ok := event.Data["output"]; ok {
				output = outputVal.(string)
			}
			m.updateChan <- messages.AICoderPTYOutputMsg{
				SessionID: sessionID,
				Data:      []byte(output),
			}
		})

		eventBus.Subscribe("aicoder.error", func(event events.Event) {
			sessionID := event.ProcessID
			var err error
			if errVal, ok := event.Data["error"]; ok {
				if e, ok := errVal.(error); ok {
					err = e
				}
			}
			m.updateChan <- messages.AICoderErrorMsg{
				SessionID: sessionID,
				Error:     err,
				Severity:  "error",
			}
		})
	}

	// 13.7. Initialize AI Coders view component with service (Feature 005)
	//     - Required by: AI Coders view rendering
	//     - Dependencies: aiCoderService must be created
	m.aicodersView = components.NewAICodersViewComponent(m.aiCoderService)

	// 13.8. Subscribe to proxy events for web view
	//     - Required by: Web view proxy request display
	//     - Dependencies: eventBus available
	if m.eventBus != nil {
		m.eventBus.Subscribe(events.ProxyRequest, func(event events.Event) {
			// Extract request data from event
			method := ""
			url := ""
			statusCode := 0
			duration := time.Duration(0)
			size := int64(0)
			processName := ""

			if val, ok := event.Data["method"]; ok {
				if s, ok := val.(string); ok {
					method = s
				}
			}
			if val, ok := event.Data["url"]; ok {
				if s, ok := val.(string); ok {
					url = s
				}
			}
			if val, ok := event.Data["status"]; ok {
				if i, ok := val.(int); ok {
					statusCode = i
				}
			}
			if val, ok := event.Data["duration"]; ok {
				if ms, ok := val.(int64); ok {
					duration = time.Duration(ms) * time.Millisecond
				}
			}
			if val, ok := event.Data["size"]; ok {
				if s, ok := val.(int64); ok {
					size = s
				}
			}
			if val, ok := event.Data["processName"]; ok {
				if s, ok := val.(string); ok {
					processName = s
				}
			}

			// Generate ID from timestamp and URL
			timestamp := time.Now()
			id := fmt.Sprintf("%d-%s", timestamp.UnixNano(), method)

			// Send as message to update loop
			m.updateChan <- messages.ProxyRequestMsg{
				RequestID:   id,
				Method:      method,
				URL:         url,
				Path:        url, // Will be same for now
				StatusCode:  statusCode,
				Duration:    duration,
				ProcessName: processName,
				Timestamp:   timestamp,
				Size:        size,
			}
		})
	}

	// 13.9. Connect AI coder service to ServiceBridge (Task 3 - ServiceBridge integration)
	//     - Required by: Service event routing to UI components
	//     - Dependencies: aiCoderService created, shell exists with ServiceBridge
	// Note: This happens after shell creation, so we need to get the ServiceBridge from the shell
	// and subscribe to AI coder service events

	// 14. Initialize script selector component (simplified for debugging)
	//     - Required by: Script selection UI
	//     - Dependencies: None (navigation adapter disabled)
	scriptAutocomplete := NewScriptSelectorAutocompleteWithProcessManager(scripts, processMgr)
	scriptAutocompleteAdapter := NewScriptSelectorAutocompleteAdapter(scriptAutocomplete)
	m.scriptSelector = components.NewScriptSelector(scriptAutocompleteAdapter, processMgr, logStore, m.updateChan, nil) // Pass nil for navAdapter

	// 15. Initialize tabs component (simplified for debugging)
	//     - Required by: Header rendering
	//     - Dependencies: processMgr only (others disabled)
	m.tabsComponent = NewTabsComponent(processMgr, nil, nil, debugMode)

	// 16. Initialize content layout
	//     - Required by: Content area rendering
	//     - Dependencies: None
	m.contentLayout = NewContentLayout()

	// 17. Initialize message router
	//     - Required by: Message routing and handling
	//     - Dependencies: None
	m.messageRouter = NewMessageRouter()

	// 18. Initialize SIMPLIFIED keyboard handler registry
	//     - Required by: All keyboard handling
	//     - Dependencies: navigationService, other services
	m.keyRegistry = NewKeyHandlerRegistry()

	// Register example view-specific handlers (adds logs/errors/process/web handlers)
	RegisterViewSpecificHandlers(m.keyRegistry)

	// Register all keyboard handlers in priority order
	// Priority 100: Global navigation (number keys, Tab)
	// SIMPLIFIED: Pass viewManager directly instead of navigationService
	m.keyRegistry.Register(NewGlobalNavigationHandler(m.viewManager, m.debugMode))

	// Priority 250: AI Coder specific keys (when in AI Coder view)
	if m.aicodersView != nil {
		m.keyRegistry.Register(NewAICoderKeyHandler(m.aicodersView))
	}

	// Priority 300: Slash command (context-aware)
	m.keyRegistry.Register(NewSlashCommandHandler(func() tea.Cmd {
		// Open command palette
		m.showCommandWindow()
		return nil
	}))

	// Priority 900: General key handler (catch-all for common keys)
	m.keyRegistry.Register(NewGeneralKeyHandler())

	// Priority 1000: Quit handler (double-press protection)
	m.keyRegistry.Register(NewQuitHandler())

	// Log successful keyboard setup
	m.logStore.Add("system", "KeyRegistry",
		fmt.Sprintf("✅ Simplified keyboard system initialized with %d handlers (including view-specific)",
			len(m.keyRegistry.handlers)), false)

	// Register keyboard handler with message router
	// This integrates keyboard handling into the unified routing system
	m.messageRouter.RegisterKeyboardHandler(m.keyRegistry)
	m.logStore.Add("system", "MessageRouter",
		"✅ Keyboard handler registered with message router", false)

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
	// Event subscriptions now handled inline in Update() method
}

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textinput.Blink,
		m.tickCmd(),
	}

	return tea.Batch(cmds...)
}

// Update is the ONLY place where Model state should be modified (Redux/Elm reducer pattern)
//
// ⚠️  CRITICAL: This is the PURE REDUCER function - follow these rules strictly!
//
// ✅ CORRECT patterns:
// 1. Receive message → Update state → Return new model + command
// 2. Return tea.Cmd for ALL async operations (network, I/O, timers)
// 3. Keep Update() fast (<5ms) - defer heavy work to commands
// 4. State changes ONLY in Update() - nowhere else!
//
// ❌ WRONG patterns that WILL cause bugs:
//
// 1. DON'T block in Update() (freezes entire TUI):
//    ❌ data := http.Get(url)          // Blocks event loop!
//    ❌ time.Sleep(duration)            // Freezes UI!
//    ❌ result := <-channel             // Waits forever!
//    ✅ return m, fetchDataCmd()        // Async via command
//
// 2. DON'T spawn goroutines directly (can't communicate back):
//    ❌ go func() { result := work(); /* how to send back? */ }()
//    ✅ return m, tea.Cmd(func() tea.Msg { return WorkDoneMsg{work()} })
//
// 3. DON'T modify state outside Update():
//    ❌ m.processMgr.AddProcess(proc)   // Direct modification!
//    ❌ m.state.Processes = newList      // Bypasses Update()!
//    ✅ return m, addProcessCmd(proc)    // Goes through Update()
//
// 4. DON'T call services that might block:
//    ❌ procs := m.service.GetProcesses()  // Might block!
//    ✅ Read from m.masterData (cached)    // Always fast
//
// See: /docs/architecture/redux-elm-architecture.md for full explanation
// See: /docs/architecture/tui-deadlock-prevention.md for deadlock patterns
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd


	// CONSOLIDATED MESSAGE ROUTING
	// All messages (including keyboard) are routed through the message router
	// The router handles messages in priority order:
	//   1. NotificationsMessageHandler (observes all messages)
	//   2. DialogMessageHandler (handles modals/dialogs)
	//   3. System/Process/Log/MCP/AI handlers
	//   4. KeyboardMessageHandler (integrated key registry)
	//   5. ViewMessageHandler (window size, mouse, etc.)

	// T026: Handle confirmation keys with highest priority (before MessageRouter)
	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.awaitingClearConfirmation {
		switch keyMsg.String() {
		case "y", "Y":
			// Execute clear command based on confirmation tab
			m.awaitingClearConfirmation = false
			switch m.confirmationTab {
			case "errors":
				cmds = append(cmds, func() tea.Msg { return messages.ClearErrorsMsg{} })
			case "logs":
				cmds = append(cmds, func() tea.Msg { return messages.ClearLogsMsg{} })
			case "web":
				cmds = append(cmds, func() tea.Msg { return messages.ClearWebMsg{} })
			}
			m.confirmationTab = ""
			return m, tea.Batch(cmds...)

		case "n", "N", "esc":
			// Cancel confirmation
			m.awaitingClearConfirmation = false
			m.confirmationTab = ""
			return m, tea.Batch(cmds...)

		default:
			// Ignore other keys while confirmation is active
			return m, nil
		}
	}

	// Route all messages (keyboard and non-keyboard) through MessageRouter
	if m.messageRouter != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if m.logStore != nil {
						m.logStore.Add("error", "Model.Update", fmt.Sprintf("🚨 MessageRouter panic: %v", r), true)
					}
				}
			}()
			newModel, cmd := m.messageRouter.Route(msg, m)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			if newModel != nil {
				m = newModel.(*Model)
			}
		}()
	}

	// Handle application-level messages that don't need routing
	switch msg := msg.(type) {
	case RestartProcessMsg:
		if msg.clearLogs {
			m.logStore.ClearLogs()
			m.logStore.ClearErrors()
		}
		m.logStore.Add("system", "System", msg.message, msg.isError)
		m.updateProcessList()
		m.updateLogsView()
		cmds = append(cmds, m.waitForUpdates())

	case RestartAllMsg:
		if msg.clearLogs {
			m.logStore.ClearLogs()
			m.logStore.ClearErrors()
		}
		m.logStore.Add("system", "System", msg.message, msg.isError)
		m.updateProcessList()
		m.updateLogsView()
		cmds = append(cmds, m.waitForUpdates())

	case ErrorUpdateMsg:
		m.updateErrorsList()
		cmds = append(cmds, m.waitForUpdates())

	case TickMsg:
		// Continue ticking for periodic updates
		cmds = append(cmds, m.tickCmd())

		// Sync URLs from log store to URLsState (non-blocking)
		if m.masterData != nil && m.masterData.URLsState != nil {
			urls := m.logStore.GetURLs()
			urlInfos := make([]state.URLInfo, len(urls))
			for i, url := range urls {
				urlInfos[i] = state.URLInfo{
					URL:         url.URL,
					Label:       url.URL,
					ProcessName: url.ProcessID,
					Timestamp:   url.Timestamp,
					VisitCount:  0,
				}
			}
			m.masterData.URLsState = m.masterData.URLsState.UpdateURLs(urlInfos)
		}

		// Update views based on current view to ensure real-time updates
		// ⚠️ IMPORTANT: Only call non-blocking update methods from tick handler
		switch m.currentView() {
		case ViewProcesses:
			m.updateProcessList() // Safe: updates list model without blocking
		case ViewLogs:
			m.updateLogsView() // Safe: updates viewport without blocking
		// ⚠️ ViewURLs gets its updates through URLsState sync above
		}

	case MCPActivityMsg:
		// TODO: Fix MCPActivityMsg handling - needs proper sessionId and activity data
		// For now, skip handling to allow build to pass
		// Legacy HandleActivity and UpdateConnectionsList calls removed - new component is stateless
		cmds = append(cmds, m.waitForUpdates())

	case ViewSwitchedMsg:
		// View has been switched by SimpleViewManager
		// No additional processing needed - view manager handles the switch

	case SwitchToAICodersMsg:
		// Switch to AI Coders view
		m.switchToView(ViewAICoders)
		cmds = append(cmds, m.waitForUpdates())

	case CommandStartedMsg:
		// Handle command completion - switch to logs view and update
		if msg.success {
			m.logStore.Add("system", "System", fmt.Sprintf("✅ Started command: %s", msg.commandName), false)
		}
		m.switchToView(ViewLogs)
		m.updateProcessList()
		m.updateLogsView()
		cmds = append(cmds, m.waitForUpdates())

	case messages.AICoderPTYOutputMsg:
		// Handle PTY output using sub-model pattern (Elm/Redux compliant)
		if m.masterData != nil && m.masterData.AICoderState != nil {
			// Pure state transformation through sub-model
			newAICoderState := m.masterData.AICoderState.AddPTYOutput(
				msg.SessionID,
				string(msg.Data),
			)
			m.masterData.AICoderState = newAICoderState
		}
		// Always trigger a view update when PTY output arrives
		cmds = append(cmds, m.waitForUpdates())

	case messages.TerminalStateMsg:
		// Handle VT10x terminal state updates using sub-model pattern
		if m.masterData != nil && m.masterData.AICoderState != nil {
			// Pure state transformation through sub-model
			newAICoderState := m.masterData.AICoderState.UpdateTerminalState(
				msg.SessionID,
				msg.State,
			)
			m.masterData.AICoderState = newAICoderState
		}
		cmds = append(cmds, m.waitForUpdates())

	case messages.AICoderSessionCreatedMsg:
		// Handle AI Coder session created events
		// Log session creation
		m.logStore.Add("system", "AI Coder", fmt.Sprintf("🤖 Session created: %s", msg.Name), false)
		cmds = append(cmds, m.waitForUpdates())

	case messages.AICoderSessionUpdatedMsg:
		// Handle AI Coder session updated events
		cmds = append(cmds, m.waitForUpdates())

	case messages.AICoderSessionEndedMsg:
		// Handle AI Coder session ended events
		// Log session end
		m.logStore.Add("system", "AI Coder", fmt.Sprintf("🔚 Session ended: %s (%s)", msg.SessionID, msg.Reason), false)
		cmds = append(cmds, m.waitForUpdates())

	// Note: aiCoderOutputMsg has been replaced with messages.AICoderPTYOutputMsg
	// This case is kept for compatibility with any remaining legacy code

	case messages.AICoderErrorMsg:
		// Handle AI Coder errors
		// Log AI errors
		if msg.Error != nil {
			m.logStore.Add("system", "AI Coder", fmt.Sprintf("❌ AI Error: %v", msg.Error), true)
		}
		cmds = append(cmds, m.waitForUpdates())

	case messages.CommandExecuteMsg:
		// Handle command execution from command window
		// Close command window if open
		if m.masterData.CommandWindowState.IsShowingCommandWindow() {
			m.masterData.CommandWindowState = m.masterData.CommandWindowState.HideCommandWindow()
		}

		// Execute the command using the model's command handling
		commandStr := "/" + msg.CommandName
		if len(msg.Arguments) > 0 {
			commandStr += " " + strings.Join(msg.Arguments, " ")
		}
		cmd := m.handleSlashCommand(commandStr)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case commands.CommandResultMsg:
		// Handle command results from slash commands
		switch msg.Command {
		case "clear":
			m.handleClearCommand(msg.Message)
		case "show":
			if m.masterData != nil && m.masterData.LogsState != nil {
				m.masterData.LogsState = m.masterData.LogsState.SetShowPattern(msg.Message)
			}
		case "hide":
			if m.masterData != nil && m.masterData.LogsState != nil {
				m.masterData.LogsState = m.masterData.LogsState.SetHidePattern(msg.Message)
			}
		case "proxy":
			if msg.Message == "list" {
				m.listProxies()
			} else {
				m.handleProxyCommand(msg.Message)
			}
		case "toggle-proxy":
			m.toggleProxyMode()
		case "ai":
			m.handleAICommand(msg.Message)
		case "term":
			m.showTerminal()
		}

		// Switch view if specified
		if msg.View != "" {
			m.switchToView(messages.ViewType(msg.View))
		}

	// ===== GET ALL THE: NEW MESSAGE HANDLERS =====
	case state.ErrorGroupedMsg:
		// Handle new grouped error ready for display
		if m.masterData != nil && m.masterData.ErrorsState != nil {
			m.masterData.ErrorsState = m.masterData.ErrorsState.AddGroupedError(msg.Error)
			// Update parser state
			m.masterData.ErrorsState = m.masterData.ErrorsState.SetParserState(msg.NewState)
			// Switch to errors tab to show new grouped error
			if m.tabsComponent != nil {
				m.tabsComponent.SetActiveView(ViewErrors)
			}
		}

	case state.ErrorGroupedBatchMsg:
		// Handle batch error grouping (completed old group, started new group)
		if m.masterData != nil && m.masterData.ErrorsState != nil {
			// Add completed error
			m.masterData.ErrorsState = m.masterData.ErrorsState.AddGroupedError(msg.CompletedError)
			// Add new error
			m.masterData.ErrorsState = m.masterData.ErrorsState.AddGroupedError(msg.NewError)
			// Update parser state
			m.masterData.ErrorsState = m.masterData.ErrorsState.SetParserState(msg.NewState)
			// Switch to errors tab to show new grouped errors
			if m.tabsComponent != nil {
				m.tabsComponent.SetActiveView(ViewErrors)
			}
		}

	case messages.ClearErrorsMsg:
		// Handle clear errors command
		if m.masterData != nil && m.masterData.ErrorsState != nil {
			m.masterData.ErrorsState = m.masterData.ErrorsState.ClearErrors()
		}

	case messages.ClearLogsMsg:
		// Handle clear logs command
		if m.masterData != nil && m.masterData.LogsState != nil {
			m.masterData.LogsState = m.masterData.LogsState.ClearLogs()
		}
		// Also clear from log store
		m.logStore.ClearLogs()
		m.logStore.Add("system", "System", "📝 Logs cleared", false)
		m.updateLogsView()

	case messages.ClearWebMsg:
		// Handle clear web cache command
		if m.masterData != nil && m.masterData.WebViewState != nil {
			m.masterData.WebViewState = m.masterData.WebViewState.ClearRequests()
		}

	case messages.ProxyRequestMsg:
		// Handle proxy request - add to web view state
		if m.masterData != nil && m.masterData.WebViewState != nil {
			requestInfo := state.RequestInfo{
				ID:           msg.RequestID,
				Method:       msg.Method,
				URL:          msg.URL,
				Path:         msg.Path,
				StatusCode:   msg.StatusCode,
				Duration:     msg.Duration,
				ProcessName:  msg.ProcessName,
				Timestamp:    msg.Timestamp,
				Size:         msg.Size,
				Error:        msg.Error,
				HasAuth:      msg.HasAuth,
				AuthType:     msg.AuthType,
				JWTError:     msg.JWTError,
				JWTClaims:    msg.JWTClaims,
				HasTelemetry: msg.HasTelemetry,
				SessionID:    msg.SessionID,
			}
			m.masterData.WebViewState = m.masterData.WebViewState.AddRequest(requestInfo)
		}

	case messages.ScrollLogsUpMsg:
		// Handle scroll up - disables auto-scroll
		if m.masterData != nil && m.masterData.LogsState != nil {
			m.masterData.LogsState = m.masterData.LogsState.ScrollUp(1)
			m.updateLogsView()
		}

	case messages.ScrollLogsDownMsg:
		// Handle scroll down
		if m.masterData != nil && m.masterData.LogsState != nil {
			m.masterData.LogsState = m.masterData.LogsState.ScrollDown(1)
			m.updateLogsView()
		}

	case messages.ToggleAutoScrollMsg:
		// Handle toggle auto-scroll (Space key)
		if m.masterData != nil && m.masterData.LogsState != nil {
			if m.masterData.LogsState.GetAutoScroll() {
				m.masterData.LogsState = m.masterData.LogsState.DisableAutoScroll()
			} else {
				m.masterData.LogsState = m.masterData.LogsState.EnableAutoScroll()
			}
			m.updateLogsView()
		}

	case messages.ShowClearConfirmationMsg:
		// Handle show clear confirmation dialog (T026)
		m.awaitingClearConfirmation = true
		m.confirmationTab = msg.Tab

	case messages.LogAddedMsg:
		// T027: Handle new log entries - process for error grouping
		if msg.Entry != nil && m.errorGroupingService != nil {
			// Get current parser state from ErrorsState
			var parserState *state.ErrorParserState
			if m.masterData != nil && m.masterData.ErrorsState != nil {
				// Get state and take address for pointer
				ps := m.masterData.ErrorsState.GetParserState()
				parserState = &ps
			} else {
				// Create new state and take address for pointer
				ps := state.NewErrorParserState()
				parserState = &ps
			}

			// Process the log line for error grouping
			cmd := m.errorGroupingService.ParseAndGroupError(msg.Entry.Content, parserState)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

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
	if m.currentView() == ViewScriptSelector && m.scriptSelector != nil {
		// Ensure the script selector is shown when this view is active
		if !m.scriptSelector.IsVisible() {
			m.scriptSelector.Show(false)
		}
		return m.scriptSelector.View()
	}
	if m.commandWindowView != nil && m.masterData.CommandWindowState.IsShowingCommandWindow() {
		return m.commandWindowView.ViewWithState(m.masterData)
	}

	// TODO(Feature 005): Re-implement full screen mode for AI Coder view if needed
	// Full screen PTY rendering is now handled by the component's Render() method
	content := m.renderContent()

	// Render main content with consistent layout
	return m.renderLayout(content)
}

// renderLayout provides consistent layout for all views
func (m *Model) renderLayout(content string) string {
	// Update tabs component
	if m.tabsComponent != nil {
		m.tabsComponent.SetActiveView(m.currentView())
		m.tabsComponent.SetWidth(m.width)
	}

	// Calculate header and footer heights directly
	m.headerHeight = 3 // Title (1) + tabs (1) + border (1)
	m.footerHeight = 0 // No footer in current design

	// Update content layout dimensions
	if m.contentLayout != nil {
		m.contentLayout.UpdateDimensions(m.width, m.height, m.headerHeight, m.footerHeight)
	}

	// If system panel is expanded, show it instead of main content
	if m.systemComponent != nil && m.systemComponent.GetPanelState() && len(m.systemComponent.GetMessages()) > 0 {
		// Full screen mode - system panel takes most of the space
		// System component will handle its own rendering via BubbleTea
		content = m.systemComponent.View()
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
	// No footer in current design

	mainLayout := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// If we have system messages in non-expanded mode, overlay them
	if m.systemComponent != nil && len(m.systemComponent.GetMessages()) > 0 && !m.systemComponent.GetPanelState() {
		// System component handles overlay rendering via its View method
		systemView := m.systemComponent.View()
		if systemView != "" {
			return systemView
		}
	}

	// T026: Show clear confirmation dialog if active
	if m.awaitingClearConfirmation {
		return m.renderClearConfirmationDialog(mainLayout)
	}

	return mainLayout
}

// renderClearConfirmationDialog renders the confirmation dialog overlay (T026)
func (m *Model) renderClearConfirmationDialog(backgroundContent string) string {
	// Define styles for the dialog
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")). // Red border for warning
		Padding(1, 2).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("255")).
		Width(60).
		Align(lipgloss.Center, lipgloss.Center)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")). // Red for warning
		MarginBottom(1)

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		MarginBottom(1)

	promptStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")). // Yellow for prompt
		MarginTop(1)

	// Build dialog content
	var tabName string
	switch m.confirmationTab {
	case "errors":
		tabName = "Errors"
	case "logs":
		tabName = "Logs"
	case "web":
		tabName = "Web"
	default:
		tabName = "Data"
	}

	title := titleStyle.Render("⚠️  Clear Confirmation")
	text := textStyle.Render(fmt.Sprintf("This will permanently clear all %s.", tabName))
	prompt := promptStyle.Render("Clear all %s? (y/n)", tabName)

	// Combine dialog content
	dialogContent := lipgloss.JoinVertical(lipgloss.Center, title, text, prompt)
	dialog := dialogStyle.Render(dialogContent)

	// Create a semi-transparent overlay
	overlayStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("255"))

	// Calculate overlay size
	overlayHeight := m.height
	overlayWidth := m.width

	// Create overlay with background blurred
	overlay := overlayStyle.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(strings.Repeat(" ", overlayWidth*overlayHeight))

	// Position dialog in center
	dialogX := (m.width - 60) / 2
	dialogY := (m.height - 7) / 2 // Approximate dialog height

	// Build final view with dialog overlay
	var lines []string
	overlayLines := strings.Split(overlay, "\n")
	dialogLines := strings.Split(dialog, "\n")

	for i := 0; i < len(overlayLines); i++ {
		if i >= dialogY && i < dialogY+len(dialogLines) {
			// Replace this line with dialog content
			dialogLine := dialogLines[i-dialogY]
			if i == dialogY {
				// First line - add padding to center dialog
				padding := strings.Repeat(" ", dialogX)
				lines = append(lines, padding+dialogLine)
			} else {
				// Subsequent lines - add padding to align with dialog
				padding := strings.Repeat(" ", dialogX)
				lines = append(lines, padding+dialogLine)
			}
		} else {
			// Keep overlay line
			lines = append(lines, overlayLines[i])
		}
	}

	return strings.Join(lines, "\n")
}

// renderContent renders the main content area based on current view
func (m *Model) renderContent() string {
	if m.commandWindowView != nil && m.masterData.CommandWindowState.IsShowingRunDialog() {
		return m.commandWindowView.ViewWithState(m.masterData)
	}

	if m.commandWindowView != nil && m.masterData.CommandWindowState.IsShowingCustomCommand() {
		return m.commandWindowView.ViewWithState(m.masterData)
	}

	// Calculate content height once for all views
	contentHeight := m.calculateContentHeight()

	switch m.currentView() {
	case ViewProcesses:
		// Update processes state and render statelessly
		m.processesView.SetSize(m.width, contentHeight)
		return m.processesView.ViewWithState(m.masterData, m.processMgr)
	case ViewLogs:
		// Update logs state and render statelessly
		if m.masterData != nil && m.masterData.LogsState != nil {
			// Update selected process if different
			if m.masterData.LogsState.GetSelectedProcess() != m.selectedProcess {
				m.masterData.LogsState = m.masterData.LogsState.SetSelectedProcess(m.selectedProcess)
			}
		}
		m.logsView.SetSize(m.width, contentHeight)
		return m.logsView.ViewWithState(m.masterData, m.logStore)
	case ViewErrors:
		// Render with calculated dimensions using stateless component
		contentHeight := m.height - m.headerHeight - m.footerHeight

		// Use split view for wide screens
		if m.contentLayout != nil && m.width >= 100 {
			// Split view: errors list on left, details on right
			leftWidth := int(float64(m.width) * DefaultSplitRatio)
			rightWidth := m.width - leftWidth

			// Render errors list (left side)
			m.errorsView.UpdateSize(leftWidth-4, m.height, m.headerHeight, m.footerHeight, contentHeight-2)
			leftContent := m.errorsView.ViewWithState(m.masterData, m.logStore)

			// Render error details (right side)
			rightContent := m.renderErrorDetailsView(rightWidth-4, contentHeight-2)

			// Use content layout for split view
			return m.contentLayout.RenderSplitView(leftContent, rightContent, DefaultSplitRatio)
		}

		// Fall back to regular rendering for narrow screens
		m.errorsView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.errorsView.ViewWithState(m.masterData, m.logStore)
	case ViewURLs:
		// Render URLs view using stateless component
		m.urlsView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.urlsView.ViewWithState(m.masterData)
	case ViewWeb:
		// Update component dimensions and render using stateless pattern
		m.webView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.webView.ViewWithState(m.masterData)
	case ViewSettings:
		if m.fileBrowserComponent != nil {
			// Check if file browser dialog should be shown
			// The component will handle its own state and rendering
			if dialogView := m.fileBrowserComponent.View(); dialogView != "" {
				return dialogView
			}
		}
		// Render settings view using stateless component
		m.settingsView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.settingsView.ViewWithState(m.masterData)
	case ViewMCPStatus:
		// Render MCP status using stateless component
		contentHeight := m.height - m.headerHeight - m.footerHeight
		m.mcpView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.mcpView.ViewWithState(m.masterData)
	case ViewMCPConnections:
		if m.debugMode {
			// Render MCP debug view using stateless component
			contentHeight := m.height - m.headerHeight - m.footerHeight
			m.mcpDebugView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
			return m.mcpDebugView.ViewWithState(m.masterData)
		}
		// Fallback to settings if not in debug mode
		m.settingsView.UpdateSize(m.width, m.height, m.headerHeight, m.footerHeight, contentHeight)
		return m.settingsView.ViewWithState(m.masterData)
	case ViewAICoders:
		if m.aicodersView != nil && m.masterData != nil {
			return m.aicodersView.ViewWithState(m.masterData)
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
		allProcesses := m.processMgr.GetAllProcesses()
		running := 0
		for _, p := range allProcesses {
			if p.GetStatus() == processes.StatusRunning {
				running++
			}
		}
		return fmt.Sprintf("%d processes, %d running", len(allProcesses), running)

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
		// TODO(Feature 005): Get session count from service
		if m.aiCoderService != nil {
			svc := m.aiCoderService.(*services.AICoderService)
			sessions := svc.ListSessions()
			return fmt.Sprintf("%d AI coder sessions", len(sessions))
		}
		return "0 AI coder sessions"

	case ViewMCPStatus:
		// Get MCP status from state
		if m.masterData != nil && m.masterData.MCPState != nil {
			mcpState := m.masterData.MCPState
			if mcpState.GetActive() {
				return fmt.Sprintf("MCP Active - %d tools, %d requests", mcpState.GetToolsCount(), mcpState.GetRequestCount())
			} else {
				return "MCP Inactive"
			}
		}
		return "MCP Status Unknown"

	case ViewMCPConnections:
		// Get MCP debug status from state
		if m.masterData != nil && m.masterData.MCPDebugState != nil {
			debugState := m.masterData.MCPDebugState
			totalConns := debugState.GetConnectionsCount()
			activeConns := debugState.GetActiveConnectionsCount()
			return fmt.Sprintf("MCP Debug - %d total, %d active connections", totalConns, activeConns)
		}
		return "MCP Debug Unavailable"

	default:
		return ""
	}
}

func (m *Model) updateSizes() {
	// Update layout controller first
	if m.tabsComponent != nil {
		// Calculate heights directly
		m.headerHeight = 3 // Title (1) + tabs (1) + border (1)
		m.footerHeight = 0 // No footer
	}

	// Use calculated header and footer heights for consistency with other views
	// contentHeight := m.height - m.headerHeight - m.footerHeight

	// processesList sizing now handled by ProcessViewController

	// Calculate content height once for all controllers
	// (Used directly as contentHeight above)

	// Process controller no longer stores dimensions - they are passed during rendering

	// settingsList sizing now handled by SettingsController
	// Command window list sizing handled by component itself
	// Errors controller no longer stores dimensions - passed during rendering
	// Logs controller no longer stores dimensions - passed during rendering

	// URLs controller no longer stores dimensions - passed during rendering
	// Web view sizing handled by WebViewController during rendering

	// AI Coder controller no longer stores dimensions - they are passed during rendering

	// Update command window controller size (handled by component itself)
	// m.commandWindowView.UpdateSize(m.width, m.height)
}

func (m *Model) cycleView() {
	// Use simple view manager to cycle to next view
	if m.viewManager != nil {
		nextView := CycleToNextView(m.viewManager.GetCurrentView(), m.debugMode)
		m.viewManager.SetCurrentView(nextView)
	}
}

func (m *Model) cyclePrevView() {
	// Use simple view manager to cycle to previous view
	if m.viewManager != nil {
		prevView := CycleToPrevView(m.viewManager.GetCurrentView(), m.debugMode)
		m.viewManager.SetCurrentView(prevView)
	}
}

// switchToView changes the current view and performs any necessary setup
func (m *Model) switchToView(view View) {
	// Use simple view manager instead
	if m.viewManager != nil {
		m.viewManager.SetCurrentView(view)
	}
}

func (m *Model) updateProcessList() {
	// Update processes state using Redux/Elm pattern
	if m.masterData != nil && m.masterData.ProcessesState != nil && m.processMgr != nil {
		processes := m.processMgr.GetAllProcesses()
		m.masterData.ProcessesState = m.masterData.ProcessesState.UpdateProcesses(processes)
	}
}

func (m *Model) updateLogsView() {
	// Update logs state using Redux/Elm pattern
	if m.masterData != nil && m.masterData.LogsState != nil {
		// Update selected process if different
		if m.masterData.LogsState.GetSelectedProcess() != m.selectedProcess {
			m.masterData.LogsState = m.masterData.LogsState.SetSelectedProcess(m.selectedProcess)
		}

		// Invalidate cache to force refresh
		// This will trigger a refresh on next render
		if m.logStore != nil {
			var logEntries []logs.CollapsedLogEntry
			if m.selectedProcess != "" {
				logEntries = m.logStore.GetByProcessCollapsed(m.selectedProcess)
			} else {
				logEntries = m.logStore.GetAllCollapsed()
			}
			m.masterData.LogsState = m.masterData.LogsState.UpdateCachedLogs(logEntries, m.selectedProcess)
		}
	}
}

func (m *Model) renderHeader() string {
	// Get process count information
	allProcesses := m.processMgr.GetAllProcesses()
	runningCount := 0
	for _, proc := range allProcesses {
		if proc.GetStatus() == processes.StatusRunning {
			runningCount++
		}
	}

	// Build title with process info
	baseTitle := "🐝 Brummer - Development Buddy"
	var processInfo string
	if len(allProcesses) > 0 {
		if runningCount > 0 {
			processInfo = fmt.Sprintf(" (%d processes, %d running)", len(allProcesses), runningCount)
		} else {
			processInfo = fmt.Sprintf(" (%d processes)", len(allProcesses))
		}
	}

	// Browser connection removed
	browserIcon := ""

	// Add copy notification if recent
	notification := ""
	if m.notificationsComponent != nil {
		activeNotifications := m.notificationsComponent.GetActiveNotifications()
		if len(activeNotifications) > 0 {
			notificationStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Bold(true)
			notification = " " + notificationStyle.Render(activeNotifications[len(activeNotifications)-1].Message)
		}
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
	orderedViews := []View{ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewWeb, ViewAICoders, ViewMCPStatus, ViewSettings}
	if m.debugMode {
		orderedViews = append(orderedViews, ViewMCPConnections)
	}
	for i, viewType := range orderedViews {
		if cfg, ok := GetViewConfigs()[viewType]; ok {
			// Build the base label with icon and space before number
			label := fmt.Sprintf("%s.%s", cfg.KeyBinding, cfg.Title)
			if cfg.Icon != "" {
				label = cfg.Icon + " " + label
			}

			// Get unread indicator for this view
			var indicatorIcon string
			if m.systemComponent != nil {
				unreadCount := m.systemComponent.GetUnreadCount()
				if unreadCount > 0 {
					indicatorIcon = "🔴"
				} else {
					indicatorIcon = "" // No space when no indicator
				}
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

// logUpdateMsg is now centralized in model_messages.go
// tickMsg, switchToAICodersMsg, and commandStartedMsg are now centralized in model_messages.go

// ============================================================================
// UPDATE CHANNEL PATTERNS - CRITICAL FOR PREVENTING DEADLOCKS
// ============================================================================
//
// The updateChan is used for async operations to notify the UI of changes.
// INCORRECT usage causes the tab switching freezes reported in issue #tab-lockup.
//
// GOLDEN RULES:
//   1. NEVER call blocking channel operations from View() or any render method
//   2. NEVER call blocking operations from tick handlers
//   3. ALWAYS use timeout variants for UI operations
//   4. ONLY use blocking variants when you started an async operation
//
// COMMON PITFALLS:
//   ❌ Calling updateLogsView() from tick handler (blocks every second)
//   ❌ Calling UpdateErrorsList() during View() rendering (blocks on render)
//   ❌ Using waitForUpdatesBlocking() after view switches (nothing to wait for)
//   ✅ Using waitForUpdatesSafe() with reasonable timeouts
//   ✅ Calling update methods only from Update() message handlers
//
// ============================================================================

// waitForUpdatesBlocking waits FOREVER for a message on updateChan
// ⚠️ EXTREME DANGER: Will freeze the entire UI if nothing sends a message!
//
// ONLY use when ALL these conditions are met:
//   1. You just started an async operation (goroutine, command, etc.)
//   2. That operation WILL send a message to updateChan
//   3. You need to wait for that specific message
//
// Example: After starting a process that will send processStartedMsg
func (m *Model) waitForUpdatesBlocking() tea.Cmd {
	return func() tea.Msg {
		return <-m.updateChan // BLOCKS FOREVER - UI WILL FREEZE if no message comes!
	}
}

// waitForUpdates is the LEGACY default method - being phased out
// ⚠️ DEPRECATED: Was the source of tab 3 lockup bug!
//
// This method previously called waitForUpdatesBlocking() which caused deadlocks.
// Now safely uses a 50ms timeout to prevent UI freezes during migration.
//
// MIGRATION: Replace all calls with explicit variants:
//   - waitForUpdatesSafe(50*ms) for most UI operations
//   - waitForUpdatesQuick() for view switches
//   - waitForUpdatesImmediate() for optional refreshes
func (m *Model) waitForUpdates() tea.Cmd {
	// SAFETY: 50ms timeout prevents deadlocks but allows async ops to complete
	// This fixed the tab 3 lockup issue where nothing was sending messages
	return m.waitForUpdatesSafe(50 * time.Millisecond)
}

// waitForUpdatesImmediate performs a non-blocking check for updates
// ✅ ALWAYS SAFE: Never blocks, never freezes the UI
//
// Use cases:
//   - After user actions that might have triggered background updates
//   - Periodic refresh checks that shouldn't block
//   - Any time blocking would degrade user experience
//
// Returns: Message if available, nil if channel is empty
func (m *Model) waitForUpdatesImmediate() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-m.updateChan:
			return msg
		default:
			return nil // No message available, continue immediately
		}
	}
}

// waitForUpdatesSafe waits for updates with timeout protection
// ✅ SAFE: Guaranteed to return within timeout, preventing UI freezes
//
// This is the RECOMMENDED method for most use cases:
//   - View switches: use 10-50ms timeout
//   - User interactions: use 10-50ms timeout
//   - Background operations: use 100-500ms timeout
//
// The timeout is NOT an error condition - it's normal and expected.
// Many operations won't produce updates, and that's fine.
func (m *Model) waitForUpdatesSafe(timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-m.updateChan:
			return msg
		case <-time.After(timeout):
			return nil // Timeout is normal - prevents blocking
		}
	}
}

// waitForUpdatesQuick is optimized for responsive UI operations
// ✅ SAFE: 10ms timeout is imperceptible to users but prevents deadlocks
//
// Perfect for:
//   - Tab/view switching
//   - Keyboard shortcuts
//   - Menu selections
//   - Any UI interaction that needs immediate response
//
// The 10ms timeout ensures the UI never freezes while still catching
// any immediate updates from background operations.
func (m *Model) waitForUpdatesQuick() tea.Cmd {
	return m.waitForUpdatesSafe(10 * time.Millisecond)
}

func (m *Model) tickCmd() tea.Cmd {
	// Periodic tick for real-time updates (event_controller logic migrated)
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

func (m *Model) handleRestartProcess(proc *processes.Process) tea.Cmd {
	return func() tea.Msg {
		var message string
		var isError bool

		// Check if process is still running before trying to stop it
		if proc.GetStatus() == processes.StatusRunning {
			// Try to stop process with timeout
			err := m.processMgr.StopProcess(proc.ID)
			if err != nil {
				// Process didn't stop cleanly, try stop with wait timeout
				if waitErr := m.processMgr.StopProcessAndWait(proc.ID, 100*time.Millisecond); waitErr != nil {
					message = fmt.Sprintf("Error stopping process %s: %v (wait error: %v)", proc.Name, err, waitErr)
					isError = true
				}
			}
		}

		// Clean up any finished processes before starting new one
		m.processMgr.CleanupFinishedProcesses()

		// Also clean up any processes that might be using development ports
		if proc.Name == "server" || proc.Name == "dev" || proc.Name == "start" {
			m.processMgr.KillProcessesByPort()
		}

		// Now start it again
		_, err := m.processMgr.StartScript(proc.Name)
		if err != nil {
			message = fmt.Sprintf("Error restarting script %s: %v", proc.Name, err)
			isError = true
		} else {
			message = fmt.Sprintf("🔄 Restarted process: %s (logs cleared)", proc.Name)
			// Clear logs for the process
			m.logStore.ClearLogsForProcess(proc.Name)
		}

		return RestartProcessMsg{
			processID: proc.ID,
			message:   message,
			isError:   isError,
			clearLogs: true,
		}
	}
}

func (m *Model) handleRestartAll() tea.Cmd {
	return func() tea.Msg {
		allProcesses := m.processMgr.GetAllProcesses()
		restarted := 0
		var errors []string

		for _, proc := range allProcesses {
			if proc.GetStatus() == processes.StatusRunning {
				if err := m.processMgr.StopProcess(proc.ID); err != nil {
					errors = append(errors, fmt.Sprintf("Error stopping process %s: %v", proc.Name, err))
					continue
				}

				_, err := m.processMgr.StartScript(proc.Name)
				if err != nil {
					errors = append(errors, fmt.Sprintf("Error restarting script %s: %v", proc.Name, err))
				} else {
					restarted++
					// Clear logs for the process
					m.logStore.ClearLogsForProcess(proc.Name)
				}
			}
		}

		var message string
		var isError bool
		if len(errors) > 0 {
			message = fmt.Sprintf("🔄 Restarted %d processes with %d errors (logs cleared): %s", restarted, len(errors), strings.Join(errors, "; "))
			isError = true
		} else {
			message = fmt.Sprintf("🔄 Restarted %d processes (logs cleared)", restarted)
			isError = false
		}

		return RestartAllMsg{
			message:   message,
			isError:   isError,
			clearLogs: true,
		}
	}
}

func (m *Model) handleCopyError() tea.Cmd {
	// Delegate to ErrorsViewController (now handled via message pattern)
	return nil // Placeholder - clipboard functionality
}

// renderErrorDetailsView renders the error details view for the selected error
func (m *Model) renderErrorDetailsView(width, height int) string {
	if m.masterData == nil || m.masterData.ErrorsState == nil {
		return ""
	}

	selectedError := m.masterData.ErrorsState.GetSelectedError()
	if selectedError == nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Align(lipgloss.Center, lipgloss.Center).
			Render("No error selected")
	}

	// Format the error details
	var content strings.Builder
	errorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	processStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	content.WriteString(errorStyle.Render(fmt.Sprintf("❌ %s", selectedError.Type)) + "\n")
	content.WriteString(fmt.Sprintf("%s %s\n\n",
		timestampStyle.Render(selectedError.Timestamp.Format("2006-01-02 15:04:05")),
		processStyle.Render(fmt.Sprintf("[%s]", selectedError.ProcessName))))

	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Error Message:") + "\n")
	content.WriteString(selectedError.Message + "\n\n")

	if len(selectedError.Stack) > 0 {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Stack Trace:") + "\n")
		for _, line := range selectedError.Stack {
			content.WriteString("  " + line + "\n")
		}
		content.WriteString("\n")
	}

	if len(selectedError.Context) > 0 {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Context:") + "\n")
		for _, line := range selectedError.Context {
			if strings.TrimSpace(line) != "" {
				content.WriteString("  " + line + "\n")
			}
		}
	}

	return content.String()
}

func (m *Model) handleClearLogs() {
	m.logStore.ClearLogs()
	m.logStore.Add("system", "System", "📝 Logs cleared", false)
	m.updateLogsView()
}

func (m *Model) handleClearErrors() {
	// Delegate to ErrorsViewController
	// Clear errors using Redux/Elm pattern
	if m.masterData != nil && m.masterData.ErrorsState != nil {
		m.masterData.ErrorsState = m.masterData.ErrorsState.ClearErrors()
		m.logStore.ClearErrors()
	}
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

	// Update state directly
	m.masterData.CommandWindowState = m.masterData.CommandWindowState.ShowRunDialog(detectedCommands, monorepoInfo)
}

func (m *Model) handleRunCommand() tea.Cmd {
	if !m.masterData.CommandWindowState.IsShowingRunDialog() {
		return nil
	}

	// TODO: Implement run command execution with Redux/Elm pattern
	// This requires significant refactoring of the command selection logic
	return nil
}

// startCommandAsync starts a command using the command pattern instead of goroutines
func (m *Model) startCommandAsync(selectedCommand parser.ExecutableCommand) tea.Cmd {
	return func() tea.Msg {
		// Create error handler for this operation
		errorHandler := NewStandardErrorHandler(m.logStore, m.updateChan)

		// Execute the command synchronously in the command
		_, err := m.processMgr.StartCommand(selectedCommand.Name, selectedCommand.Command, selectedCommand.Args)

		// Handle any errors
		if err != nil {
			ctx := ProcessStartContext(selectedCommand.Name, "Command Window", m.logStore, m.updateChan)
			errorHandler.HandleError(err, ctx)
		}

		// Return a message to trigger UI updates
		return CommandStartedMsg{
			commandName: selectedCommand.Name,
			success:     err == nil,
		}
	}
}

func (m *Model) updateErrorsList() {
	// Update errors state using Redux/Elm pattern
	if m.masterData != nil && m.masterData.ErrorsState != nil && m.logStore != nil {
		errorContexts := m.logStore.GetErrorContexts()
		m.masterData.ErrorsState = m.masterData.ErrorsState.UpdateErrorContexts(errorContexts)
	}
}

func (m *Model) updateErrorDetailView() {
	// Error detail view is now handled by stateless rendering
	// Selected error is tracked in ErrorsState
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

func (m *Model) handleSlashCommand(input string) tea.Cmd {
	// Create simplified context with only essential dependencies
	ctx := &commands.SlashCommandContext{
		ProcessManager: m.processMgr,
		LogStore:       m.logStore,
		Model:          m, // Pass model for callbacks
	}

	// Delegate to the simplified handler - returns tea.Cmd
	return commands.HandleSlashCommand(ctx, input)
}

// Helper methods for slash command callbacks
func (m *Model) handleClearCommand(target string) {
	switch target {
	case "all":
		m.logStore.ClearLogs()
		// Clear errors using new Redux/Elm pattern
		if m.masterData != nil && m.masterData.ErrorsState != nil {
			m.masterData.ErrorsState = m.masterData.ErrorsState.ClearErrors()
		}
		if m.proxyServer != nil {
			m.proxyServer.ClearRequests()
		}
		m.logStore.Add("system", "System", "🧹 Cleared all logs, errors, and web requests", false)
	case "logs":
		m.logStore.ClearLogs()
		m.logStore.Add("system", "System", "🧹 Cleared all logs", false)
	case "errors":
		// Clear errors using new Redux/Elm pattern
		if m.masterData != nil && m.masterData.ErrorsState != nil {
			m.masterData.ErrorsState = m.masterData.ErrorsState.ClearErrors()
		}
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
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Invalid URL: %v", err), true)
		return
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		urlStr = "http://" + urlStr
		parsedURL, err = url.Parse(urlStr)
		if err != nil {
			m.logStore.Add("system", "System", fmt.Sprintf("Invalid URL after adding scheme: %v", err), true)
			return
		}
	}

	// Register the URL with the proxy server
	proxyURL := m.proxyServer.RegisterURL(urlStr, "manual")
	if proxyURL == urlStr {
		// Registration failed (likely port allocation issue)
		m.logStore.Add("system", "System", fmt.Sprintf("⚠️ Failed to register proxy for %s (port allocation failed)", urlStr), true)
		return
	}

	// Log successful registration
	m.logStore.Add("system", "System", fmt.Sprintf("🚀 Proxy registered: %s → %s", urlStr, proxyURL), false)
	m.logStore.Add("system", "System", fmt.Sprintf("🔗 Access your app at: %s", proxyURL), false)

	// Switch to web view to show the new proxy
	m.switchToView(ViewWeb)
}

func (m *Model) listProxies() {
	if m.proxyServer == nil {
		m.logStore.Add("system", "System", "Error: Proxy server is not enabled", true)
		return
	}

	mappings := m.proxyServer.GetURLMappings()
	if len(mappings) == 0 {
		m.logStore.Add("system", "System", "📋 No active proxy mappings", false)
		m.logStore.Add("system", "System", "Use '/proxy <URL>' to register a new proxy", false)
		return
	}

	m.logStore.Add("system", "System", fmt.Sprintf("📋 Active Proxy Mappings (%d):", len(mappings)), false)
	for i, mapping := range mappings {
		label := mapping.Label
		if label == "" || label == mapping.ProcessName {
			label = mapping.ProcessName
		}
		m.logStore.Add("system", "System", fmt.Sprintf("  %d. %s → %s [%s]", i+1, mapping.TargetURL, mapping.ProxyURL, label), false)
	}

	// Switch to web view to show proxy mappings
	m.switchToView(ViewWeb)
}

func (m *Model) toggleProxyMode() {
	m.handleToggleProxy()
}

func (m *Model) handleAICommand(providerName string) {
	// Use the existing PTY infrastructure to launch Claude CLI tool
	// Get CLI command and args from configuration
	command, args, err := m.getCLICommandFromConfig(providerName, "")
	if err != nil {
		m.logStore.Add("system", "System", fmt.Sprintf("Error getting CLI command for %s: %v", providerName, err), true)
		return
	}

	// Build full command string for session creation
	fullCommand := command
	if len(args) > 0 {
		fullCommand = command + " " + strings.Join(args, " ")
	}

	// Create session using existing PTY infrastructure
	if m.aiCoderService != nil {
		// Use the AI coder service to create a PTY session
		svc := m.aiCoderService.(*services.AICoderService)
		sessionID, err := svc.CreateSession(
			context.Background(),
			fmt.Sprintf("AI: %s", providerName),
			fullCommand,
			m.workingDir,
		)
		if err != nil {
			m.logStore.Add("system", "System", fmt.Sprintf("Error creating AI session: %v", err), true)
			return
		}

		// Start the session with default terminal dimensions
		err = svc.StartSession(context.Background(), sessionID, 80, 24)
		if err != nil {
			m.logStore.Add("system", "System", fmt.Sprintf("Error starting AI session: %v", err), true)
			return
		}

		// Switch to AI Coders view to show the session
		m.switchToView(ViewAICoders)
		m.logStore.Add("system", "System", fmt.Sprintf("Started AI coder session: %s", providerName), false)
	} else {
		m.logStore.Add("system", "System", "Error: AI Coder service is not initialized", true)
	}
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
		m.switchToView(ViewURLs)
	}
}

func (m *Model) showCommandWindow() {
	// Update state directly
	m.masterData.CommandWindowState = m.masterData.CommandWindowState.ShowCommandWindow()
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
	// Use the new system component for overlay rendering
	if m.systemComponent != nil {
		return m.systemComponent.View()
	}
	// Fallback to main content if no system component
	return mainContent
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

// currentView returns the current view from the simple view manager
func (m *Model) currentView() View {
	if m.viewManager != nil {
		return m.viewManager.GetCurrentView()
	}
	// Fallback to default view
	return ViewProcesses
}

// GetAICoderManager returns the AI coder manager instance
// TODO(Feature 005): Remove this method or adapt for service-based architecture
func (m *Model) GetAICoderManager() *aicoder.AICoderManager {
	// Legacy method - return nil as manager is now internal to service
	return nil
}

// GetAICoderService returns the AI coder service for external components
// This allows proxy server and other components to log AI activities
func (m *Model) GetAICoderService() interface{} {
	if m.aiCoderService == nil {
		return nil
	}
	return m.aiCoderService
}

// ConnectAICoderServiceToBridge connects the AI coder service to the ServiceBridge
// This method is called after the model is created and the shell has been set up
func (m *Model) ConnectAICoderServiceToBridge(serviceBridge interface{}) {
	if m.aiCoderService != nil && serviceBridge != nil {
		// Use type assertion to access ServiceBridge methods
		if bridge, ok := serviceBridge.(*bridge.ServiceBridge); ok {
			// Subscribe to AI coder service events
			bridge.SubscribeToAICoderService(m.aiCoderService)
		}
	}
}

// buildKeyContext builds a rich context object for keyboard handlers
// This provides all the information handlers need to make smart decisions

// KeyHandlersForDebug exposes handler slice length safely for diagnostics
func (m *Model) KeyHandlersForDebug() []KeyHandler {
	if m.keyRegistry == nil {
		return nil
	}
	return m.keyRegistry.handlers
}

func (m *Model) buildKeyContext() KeyContext {
	ctx := KeyContext{
		View:    m.currentView(),
		Context: "normal", // Default context
	}

	// Check for special contexts
	// Command window takes precedence
	if m.commandWindowView != nil && m.masterData.CommandWindowState.IsShowingCommandWindow() {
		ctx.Context = "command_window"
		return ctx
	}

	// Check for modal dialogs
	if m.commandWindowView != nil && (m.masterData.CommandWindowState.IsShowingRunDialog() || m.masterData.CommandWindowState.IsShowingCustomCommand()) {
		ctx.Context = "modal"
		return ctx
	}

	// Check for script selector
	if m.currentView() == ViewScriptSelector {
		ctx.Context = "script_selector"
		return ctx
	}

	// AI Coder specific context
	if m.currentView() == ViewAICoders && m.aicodersView != nil {
		ctx.ActiveComponent = m.aicodersView

		// Check if in PTY mode using AICoderState sub-model
		if m.masterData != nil && m.masterData.AICoderState != nil {
			focusedSessionID := m.masterData.AICoderState.FocusedSession
			if focusedSessionID != "" && m.masterData.AICoderState.HasActiveSessions() {
				ctx.Context = "pty"
				// TODO: Get cursor position and input content from PTY
				// For now, assume cursor is at position 0 for slash command logic
				ctx.CursorPosition = 0
			ctx.InputContent = ""
			}
		}
	}

	// File browser dialog
	if m.fileBrowserComponent != nil && m.fileBrowserComponent.IsVisible() {
		ctx.Context = "file_browser"
		ctx.ActiveComponent = m.fileBrowserComponent
	}

	// System panel overlay
	if m.systemComponent != nil && m.systemComponent.GetPanelState() {
		ctx.Context = "system_panel"
	}

	return ctx
}

// getKeyContext returns a simple string context for legacy compatibility
// TODO: Remove this once all code uses buildKeyContext
func (m *Model) getKeyContext() string {
	return m.buildKeyContext().Context
}


