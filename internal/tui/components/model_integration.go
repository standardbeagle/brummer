package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/standardbeagle/brummer/internal/process"
	// "github.com/standardbeagle/brummer/internal/tui/navigation"
)

// ModelBridge provides integration between the existing Model and the component system
type ModelBridge struct {
	// Component tree root
	root Component

	// View management
	viewSwitcher *ViewSwitcher
	overlayStack *StackLayout

	// Message routing
	router *SimpleMessageRouter

	// Component registry
	components map[string]Component

	// Current dimensions
	width  int
	height int

	// Data provider interface (implemented by existing Model)
	dataProvider DataProvider

	// Flag to indicate if using IntegratedShell (to avoid duplicate key handling)
	usingIntegratedShell bool
}

// DataProvider interface that the existing Model implements
type DataProvider interface {
	// Process data
	GetProcesses() []interface{}
	StartProcess(name string) error
	StopProcess(id string) error

	// Log data
	GetLogs(processID string) []string
	GetFilteredLogs(filter string) []string

	// Error data
	GetErrors() []interface{}
	ClearErrors() error

	// URL data
	GetURLs() []string

	// Settings
	GetSettings() map[string]interface{}
	UpdateSetting(key string, value interface{}) error
}

// NewModelBridge creates a bridge between existing Model and component system
func NewModelBridge(dataProvider DataProvider) *ModelBridge {
	// Force color output for terminal UI
	lipgloss.SetColorProfile(termenv.ANSI256)

	bridge := &ModelBridge{
		dataProvider: dataProvider,
		router:       NewMessageRouter(),
		components:   make(map[string]Component),
		width:        80,
		height:       24,
	}

	// Build the component tree
	bridge.buildComponentTree()

	return bridge
}

// buildComponentTree constructs the UI component hierarchy
func (b *ModelBridge) buildComponentTree() {
	// For now, use IntegratedShell directly if we have a process manager
	if provider, ok := b.dataProvider.(interface{ GetProcessManager() *process.Manager }); ok {
		// Use the IntegratedShell with real process manager
		if processMgr := provider.GetProcessManager(); processMgr != nil {
			shell := NewIntegratedShell(processMgr)
			shell.SetSize(b.width, b.height)
			b.root = shell
			b.usingIntegratedShell = true // Flag to skip duplicate key handling
			return
		}
	}

	// Fallback to the old component tree structure
	// Create main layout with initial size
	mainLayout := NewBoxLayout(Vertical)
	mainLayout.SetSize(b.width, b.height)

	// Header (can be a component wrapping existing header)
	header := b.createHeader()
	mainLayout.AddChild(header, 0.1) // 10% height

	// Content area with view switcher
	b.viewSwitcher = NewViewSwitcher()
	b.viewSwitcher.SetSize(b.width, int(float64(b.height)*0.8))

	// Create and register views
	b.createAndRegisterViews()

	mainLayout.AddChild(b.viewSwitcher, 0.8) // 80% height

	// Footer
	footer := b.createFooter()
	mainLayout.AddChild(footer, 0.1) // 10% height

	// Create overlay stack for modals/overlays with initial size
	b.overlayStack = NewStackLayout()
	b.overlayStack.SetSize(b.width, b.height)
	b.overlayStack.AddLayer(mainLayout)

	// Add system panel overlay (initially hidden)
	systemPanel := b.createSystemPanel()
	b.overlayStack.AddLayer(systemPanel)
	b.components["system_panel"] = systemPanel

	// Add command palette overlay (initially hidden)
	commandPalette := b.createCommandPalette()
	b.overlayStack.AddLayer(commandPalette)
	b.components["command_palette"] = commandPalette

	// Set root
	b.root = b.overlayStack
}

// createAndRegisterViews creates all main views
func (b *ModelBridge) createAndRegisterViews() {
	// Process view using Bubbles components
	processView := b.createProcessView()
	b.viewSwitcher.AddView("processes", processView)
	b.components["process_view"] = processView

	// Logs view
	logsView := b.createLogsView()
	b.viewSwitcher.AddView("logs", logsView)
	b.components["logs_view"] = logsView

	// Errors view
	errorsView := b.createErrorsView()
	b.viewSwitcher.AddView("errors", errorsView)
	b.components["errors_view"] = errorsView

	// URLs view
	urlsView := b.createURLsView()
	b.viewSwitcher.AddView("urls", urlsView)
	b.components["urls_view"] = urlsView

	// Settings view
	settingsView := b.createSettingsView()
	b.viewSwitcher.AddView("settings", settingsView)
	b.components["settings_view"] = settingsView

	// Start with processes view
	b.viewSwitcher.SwitchTo("processes")
}

// View component creators

func (b *ModelBridge) createHeader() Component {
	// Could wrap existing header or create new one
	// For now, create a simple header component
	return &HeaderComponent{
		BaseComponent: BaseComponent{visible: true},
		title:         "Brummer Process Manager",
	}
}

func (b *ModelBridge) createFooter() Component {
	// Could wrap existing footer or create new one
	return &FooterComponent{
		BaseComponent: BaseComponent{visible: true},
		bridge:        b,
	}
}

func (b *ModelBridge) createProcessView() Component {
	// Use the migrated process view from our examples
	return NewMigratedProcessView()
}

func (b *ModelBridge) createLogsView() Component {
	// Use the migrated logs view
	return NewMigratedLogsView()
}

func (b *ModelBridge) createErrorsView() Component {
	// Use the migrated errors view
	return NewMigratedErrorsView()
}

func (b *ModelBridge) createURLsView() Component {
	// Create a simple list view for URLs
	urls := b.dataProvider.GetURLs()
	items := make([]string, len(urls))
	copy(items, urls)
	return CreateSearchableList("URLs", items)
}

func (b *ModelBridge) createSettingsView() Component {
	// Create a settings view
	// This could be more sophisticated
	return &SettingsComponent{
		BaseComponent: BaseComponent{visible: true},
		bridge:        b,
	}
}

func (b *ModelBridge) createSystemPanel() Component {
	// Create system panel that can show/hide
	panel := NewSystemPanel(&systemPanelBridge{bridge: b})
	panel.Hide() // Start hidden
	return panel
}

func (b *ModelBridge) createCommandPalette() Component {
	// Create command palette
	palette := NewCommandPalette(&commandHandlerBridge{bridge: b})
	palette.Hide() // Start hidden
	return palette
}

// Update processes all messages through the component tree
func (b *ModelBridge) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle global messages
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
		// Resize will propagate through component tree
		if b.root != nil {
			if sizable, ok := b.root.(Sizable); ok {
				sizable.SetSize(msg.Width, msg.Height)
			}
		}

	case tea.KeyMsg:
		// If using IntegratedShell, let it handle ALL keyboard input
		// to avoid duplicate processing
		if !b.usingIntegratedShell {
			// Only handle truly global keyboard shortcuts here
			// Let IntegratedShell handle view switching and Tab navigation
			switch msg.String() {
			case "ctrl+c":
				// Always quit on Ctrl+C
				return b, tea.Quit

			case "ctrl+p":
				// Show command palette (global action)
				return b, SendShow("command_palette")

			case "ctrl+s":
				// Toggle system panel (global action)
				if panel, ok := b.components["system_panel"].(*SystemPanel); ok {
					panel.Toggle()
				}
				return b, nil
			}
		}
		// Pass through to IntegratedShell or other root component

	case ViewChangeMsg:
		// Handle view changes
		b.viewSwitcher.SwitchTo(msg.ViewID)

	case DataUpdateMsg:
		// Handle data updates from external sources
		cmds = append(cmds, b.handleDataUpdate(msg))

	case CommandMsg:
		// Handle commands
		cmds = append(cmds, b.handleCommand(msg))

		// Handle navigation messages if available
		// case navigation.ChangeViewMsg:
		//	// Bridge old navigation messages
		//	b.viewSwitcher.SwitchTo(string(msg.View))
	}

	// Let component tree handle the message
	newRoot, cmd := b.root.Update(msg)
	if c, ok := newRoot.(Component); ok {
		b.root = c
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return b, tea.Batch(cmds...)
}

// View renders the component tree
func (b *ModelBridge) View() string {
	return b.root.View()
}

// Init initializes the model bridge
func (b *ModelBridge) Init() tea.Cmd {
	// Initialize any components that need it
	var cmds []tea.Cmd

	// Send initial window size to ensure components are properly sized
	cmds = append(cmds, func() tea.Msg {
		return tea.WindowSizeMsg{
			Width:  b.width,
			Height: b.height,
		}
	})

	// Initialize the root component if it has an Init method
	if b.root != nil {
		if initializable, ok := b.root.(interface{ Init() tea.Cmd }); ok {
			if cmd := initializable.Init(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return tea.Batch(cmds...)
}

// Message handlers

func (b *ModelBridge) handleDataUpdate(msg DataUpdateMsg) tea.Cmd {
	// Route data updates to appropriate components
	switch msg.Type {
	case DataUpdateProcess:
		return b.router.Route("process_view", msg)
	case DataUpdateLogs:
		return b.router.Route("logs_view", msg)
	case DataUpdateErrors:
		return b.router.Route("errors_view", msg)
	case DataUpdateURLs:
		return b.router.Route("urls_view", msg)
	case DataUpdateSettings:
		return b.router.Route("settings_view", msg)
	default:
		// Broadcast to all if no specific target
		return b.router.Broadcast(msg)
	}
}

func (b *ModelBridge) handleCommand(msg CommandMsg) tea.Cmd {
	// Execute commands
	switch msg.Command {
	case "start_process":
		if len(msg.Args) > 0 {
			err := b.dataProvider.StartProcess(msg.Args[0])
			if err != nil {
				return SendNotification(NotificationError, "Failed to start process", err.Error())
			}
			return SendNotification(NotificationSuccess, "Process started", msg.Args[0])
		}

	case "stop_process":
		if len(msg.Args) > 0 {
			err := b.dataProvider.StopProcess(msg.Args[0])
			if err != nil {
				return SendNotification(NotificationError, "Failed to stop process", err.Error())
			}
			return SendNotification(NotificationSuccess, "Process stopped", msg.Args[0])
		}

	case "clear_errors":
		err := b.dataProvider.ClearErrors()
		if err != nil {
			return SendNotification(NotificationError, "Failed to clear errors", err.Error())
		}
		return SendNotification(NotificationSuccess, "Errors cleared", "")

	case "switch_view":
		if len(msg.Args) > 0 {
			return SendViewChange(msg.Args[0], nil)
		}
	}

	return nil
}

func (b *ModelBridge) handleTabNavigation(forward bool) tea.Cmd {
	// Implement tab navigation through focusable components
	// This would cycle through components based on tab order
	return nil
}

// Helper components for the bridge

// HeaderComponent provides a simple header
type HeaderComponent struct {
	BaseComponent
	title string
}

func (h *HeaderComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return h, nil
}

func (h *HeaderComponent) View() string {
	// Render header with proper styling
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Width(h.width).
		Align(lipgloss.Center)
	return style.Render(h.title)
}

// FooterComponent provides a footer with keybindings
type FooterComponent struct {
	BaseComponent
	bridge *ModelBridge
}

func (f *FooterComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return f, nil
}

func (f *FooterComponent) View() string {
	// Show current view and keybindings
	currentView := f.bridge.viewSwitcher.CurrentView()
	return "View: " + currentView + " | Tab: Navigate | Ctrl+P: Command | Ctrl+S: System"
}

// SettingsComponent provides settings management
type SettingsComponent struct {
	BaseComponent
	bridge *ModelBridge
}

func (s *SettingsComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle settings updates
	return s, nil
}

func (s *SettingsComponent) View() string {
	// Get settings but don't need to use the variable for simplified view
	_ = s.bridge.dataProvider.GetSettings()
	// Render settings (simplified)
	return "Settings View"
}

// systemPanelBridge implements SystemPanelRenderer
type systemPanelBridge struct {
	bridge *ModelBridge
}

func (s *systemPanelBridge) RenderPanel() string {
	return "System Messages"
}

func (s *systemPanelBridge) IsExpanded() bool {
	return false
}

func (s *systemPanelBridge) HasMessages() bool {
	// Check if there are system messages
	return false
}

func (s *systemPanelBridge) HandleKey(key string) tea.Cmd {
	return nil
}

// commandHandlerBridge implements CommandHandler
type commandHandlerBridge struct {
	bridge *ModelBridge
}

func (c *commandHandlerBridge) ExecuteCommand(command string) tea.Cmd {
	// Parse and execute command
	return SendCommand(command)
}

func (c *commandHandlerBridge) GetSuggestions(prefix string) []string {
	// Return command suggestions
	return []string{
		"start_process",
		"stop_process",
		"clear_errors",
		"switch_view processes",
		"switch_view logs",
		"switch_view errors",
	}
}

// Integration helpers

// WrapExistingModel creates a ModelBridge that wraps an existing Model
func WrapExistingModel(existingModel interface{}) *ModelBridge {
	// This would create adapters for the existing model
	// to implement the DataProvider interface

	// For now, return a basic bridge
	return NewModelBridge(nil)
}

// MigrateModel provides a migration path from existing Model
func MigrateModel(existingModel interface{}) tea.Model {
	// This would:
	// 1. Create data adapters for existing model
	// 2. Wrap existing view controllers as components
	// 3. Set up message routing
	// 4. Return a tea.Model that uses components

	bridge := WrapExistingModel(existingModel)
	return bridge
}
