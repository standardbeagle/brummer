package tui

import (
	"fmt"
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/navigation"
)

// ModelComponentAdapter adapts the existing Model to work with the component system
// This provides a gradual migration path
type ModelComponentAdapter struct {
	// Existing model for data and business logic
	model *Model
	
	// Component system
	rootComponent    components.Component
	viewSwitcher    *components.ViewSwitcher
	notificationComp *components.NotificationComponent
	processView     *components.ProcessViewComponent
	logsView        *components.LogsViewComponent
	errorsView      *components.ErrorsViewComponent
	urlsView        *components.URLsViewComponent
	scriptSelector  *components.ScriptSelectorComponent
	settingsView    *components.SettingsViewComponent
	commandInput    *components.CommandInputComponent
	webView         *components.WebViewComponent
	aiCodersView    *components.AICodersViewComponent
	
	// Mode flag for gradual migration
	useComponents bool
}

// NewModelComponentAdapter creates an adapter that can switch between old and new UI
func NewModelComponentAdapter(model *Model, useComponents bool) *ModelComponentAdapter {
	adapter := &ModelComponentAdapter{
		model:         model,
		useComponents: useComponents,
	}
	
	if useComponents {
		// Initialize component system
		adapter.initializeComponents()
	}
	
	return adapter
}

// initializeComponents sets up the component system
func (a *ModelComponentAdapter) initializeComponents() {
	// Create notification component
	if a.model.eventBus != nil {
		a.notificationComp = components.NewNotificationComponent(a.model.eventBus)
	} else {
		a.notificationComp = components.NewNotificationComponent(nil)
	}
	
	// Create view components with nil-safe checks
	if a.model.processMgr != nil {
		a.processView = components.NewProcessViewComponent(a.model.processMgr)
	} else {
		// Create empty process view
		a.processView = components.NewProcessViewComponent(nil)
	}
	
	if a.model.logStore != nil {
		a.logsView = components.NewLogsViewComponent(a.model.logStore)
	} else {
		// Create empty logs view
		a.logsView = components.NewLogsViewComponent(nil)
	}
	
	a.errorsView = components.NewErrorsViewComponent()
	
	// Create URLs view
	a.urlsView = components.NewURLsViewComponent()
	
	// Create script selector
	// Note: Package info would need to be obtained from the script selector controller
	// For now, create without package info as it can be loaded separately
	if a.model.processMgr != nil {
		a.scriptSelector = components.NewScriptSelectorComponent(a.model.processMgr, nil)
	} else {
		a.scriptSelector = components.NewScriptSelectorComponent(nil, nil)
	}
	
	// Create settings view
	a.settingsView = components.NewSettingsViewComponent()
	
	// Create command input
	a.commandInput = components.NewCommandInputComponent()
	
	// Create web view
	a.webView = components.NewWebViewComponent()
	
	// Create AI coders view
	a.aiCodersView = components.NewAICodersViewComponent()
	
	// Create view switcher
	a.viewSwitcher = components.NewViewSwitcher()
	
	// Add views to switcher
	if a.processView != nil {
		a.viewSwitcher.AddView("processes", a.processView)
	}
	if a.logsView != nil {
		a.viewSwitcher.AddView("logs", a.logsView)
	}
	if a.errorsView != nil {
		a.viewSwitcher.AddView("errors", a.errorsView)
	}
	if a.urlsView != nil {
		a.viewSwitcher.AddView("urls", a.urlsView)
	}
	if a.scriptSelector != nil {
		a.viewSwitcher.AddView("scripts", a.scriptSelector)
	}
	if a.settingsView != nil {
		a.viewSwitcher.AddView("settings", a.settingsView)
	}
	if a.webView != nil {
		a.viewSwitcher.AddView("web", a.webView)
	}
	if a.aiCodersView != nil {
		a.viewSwitcher.AddView("aicoders", a.aiCodersView)
	}
	
	// Set initial view
	a.viewSwitcher.SwitchTo("processes")
	
	// Create root layout
	mainLayout := components.NewBoxLayout(components.Vertical)
	
	// Set initial size to ensure components render
	initialWidth := 80
	initialHeight := 24
	if a.model.width > 0 {
		initialWidth = a.model.width
	}
	if a.model.height > 0 {
		initialHeight = a.model.height
	}
	mainLayout.SetSize(initialWidth, initialHeight)
	
	// Add header
	header := a.createHeader()
	mainLayout.AddChild(header, 0.1)
	
	// Add view switcher as main content
	mainLayout.AddChild(a.viewSwitcher, 0.8)
	a.viewSwitcher.SetSize(initialWidth, int(float64(initialHeight)*0.8))
	
	// Add footer
	footer := a.createFooter()
	mainLayout.AddChild(footer, 0.1)
	
	// Create overlay stack for notifications
	overlayStack := components.NewStackLayout()
	overlayStack.SetSize(initialWidth, initialHeight)
	overlayStack.AddLayer(mainLayout)
	overlayStack.AddLayer(a.notificationComp)
	
	a.rootComponent = overlayStack
}

// createHeader creates a simple header component
func (a *ModelComponentAdapter) createHeader() components.Component {
	return &simpleHeaderComponent{
		BaseComponent: components.BaseComponent{},
		adapter:       a,
	}
}

// createFooter creates a simple footer component
func (a *ModelComponentAdapter) createFooter() components.Component {
	return &simpleFooterComponent{
		BaseComponent: components.BaseComponent{},
		adapter:       a,
	}
}

// Init initializes both systems
func (a *ModelComponentAdapter) Init() tea.Cmd {
	if a.useComponents {
		var cmds []tea.Cmd
		
		// Initialize existing model for data
		modelCmd := a.model.Init()
		cmds = append(cmds, modelCmd)
		
		// Send initial window size to components
		cmds = append(cmds, func() tea.Msg {
			// Use model's dimensions if available
			if a.model.width > 0 && a.model.height > 0 {
				return tea.WindowSizeMsg{
					Width:  a.model.width,
					Height: a.model.height,
				}
			}
			// Default size
			return tea.WindowSizeMsg{
				Width:  80,
				Height: 24,
			}
		})
		
		return tea.Batch(cmds...)
	}
	// Use existing model only
	return a.model.Init()
}

// Update routes updates based on mode
func (a *ModelComponentAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always update the model for data changes
	newModel, modelCmd := a.model.Update(msg)
	a.model = newModel.(*Model)
	
	if a.useComponents {
		var cmds []tea.Cmd
		cmds = append(cmds, modelCmd)
		
		// Update root component first to handle sizing
		if a.rootComponent != nil {
			newRoot, cmd := a.rootComponent.Update(msg)
			a.rootComponent = newRoot
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		
		// Handle view switching based on model's current view
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			// Ensure components get window size
			if a.rootComponent != nil {
				if sizable, ok := a.rootComponent.(components.Sizable); ok {
					sizable.SetSize(msg.Width, msg.Height)
				}
			}
		case tea.KeyMsg:
			// Map number keys to views like the original model
			switch msg.String() {
			case "1":
				a.viewSwitcher.SwitchTo("processes")
			case "2":
				a.viewSwitcher.SwitchTo("logs")
			case "3":
				a.viewSwitcher.SwitchTo("errors")
			case "4":
				a.viewSwitcher.SwitchTo("urls")
			case "5":
				a.viewSwitcher.SwitchTo("web")
			case "6":
				a.viewSwitcher.SwitchTo("aicoders")
			case "7":
				a.viewSwitcher.SwitchTo("settings")
			case "0":
				// Script selector
				if a.scriptSelector != nil {
					a.viewSwitcher.SwitchTo("scripts")
				}
			}
		
		// Note: navigation.ChangeViewMsg handling can be added when navigation package exports it
		}
		
		// Bridge data changes to components
		cmds = append(cmds, a.bridgeDataChanges())
		
		return a, tea.Batch(cmds...)
	}
	
	// Use existing model UI
	return newModel, modelCmd
}

// View renders based on mode
func (a *ModelComponentAdapter) View() string {
	if a.useComponents {
		// Use component system for rendering
		if a.rootComponent != nil {
			return a.rootComponent.View()
		}
		return "Component system initializing..."
	}
	// Use existing model rendering
	return a.model.View()
}

// mapViewToComponentName maps navigation views to component names
func (a *ModelComponentAdapter) mapViewToComponentName(view navigation.View) string {
	switch view {
	case navigation.ViewProcesses:
		return "processes"
	case navigation.ViewLogs:
		return "logs"
	case navigation.ViewErrors:
		return "errors"
	case navigation.ViewURLs:
		return "urls"
	case navigation.ViewWeb:
		return "web"
	case navigation.ViewAICoders:
		return "aicoders"
	case navigation.ViewSettings:
		return "settings"
	default:
		return "processes"
	}
}

// bridgeDataChanges checks for data changes and notifies components
func (a *ModelComponentAdapter) bridgeDataChanges() tea.Cmd {
	var cmds []tea.Cmd
	
	// Send process updates
	if a.model.processMgr != nil && a.processView != nil {
		processes := a.model.processMgr.GetAllProcesses()
		cmds = append(cmds, func() tea.Msg {
			return components.ProcessListUpdateMsg{
				Processes: processes,
			}
		})
	}
	
	// Send log updates
	if a.model.logStore != nil && a.logsView != nil {
		logEntries := a.model.logStore.GetAll()
		// Only send recent logs to avoid overwhelming
		if len(logEntries) > 100 {
			logEntries = logEntries[len(logEntries)-100:]
		}
		// Convert to pointer slice
		logPtrs := make([]*logs.LogEntry, len(logEntries))
		for i := range logEntries {
			logPtrs[i] = &logEntries[i]
		}
		cmds = append(cmds, func() tea.Msg {
			return components.LogsUpdateMsg{
				Entries: logPtrs,
				Append:  true,
			}
		})
	}
	
	// Send error updates
	if a.model.logStore != nil && a.errorsView != nil {
		errorContexts := a.model.logStore.GetErrorContexts()
		errors := make([]components.ErrorInfo, len(errorContexts))
		for i, ec := range errorContexts {
			errors[i] = components.ErrorInfo{
				ID:          fmt.Sprintf("%s-%d", ec.ProcessName, i),
				ProcessID:   ec.ProcessName,
				ProcessName: ec.ProcessName,
				Message:     ec.Message,
				Context:     ec.Context,
				Timestamp:   ec.Timestamp,
				Type:        ec.Type,
				Count:       1,
			}
		}
		cmds = append(cmds, func() tea.Msg {
			return components.ErrorsUpdateMsg{
				Errors: errors,
			}
		})
	}
	
	// Send URL updates
	if a.model.logStore != nil && a.urlsView != nil {
		urlEntries := a.model.logStore.GetURLs()
		urls := make([]components.URLInfo, len(urlEntries))
		for i, u := range urlEntries {
			urlType := components.URLTypeHTTP
			if strings.HasPrefix(u.URL, "https://") {
				urlType = components.URLTypeHTTPS
			} else if strings.HasPrefix(u.URL, "ws://") || strings.HasPrefix(u.URL, "wss://") {
				urlType = components.URLTypeWebSocket
			}
			
			urls[i] = components.URLInfo{
				URL:       u.URL,
				ProcessID: u.ProcessID,
				Type:      urlType,
				FirstSeen: u.Timestamp,  // Use Timestamp as FirstSeen
				LastSeen:  u.Timestamp,  // Use Timestamp as LastSeen
				Count:     1,
				Active:    true,
			}
		}
		cmds = append(cmds, func() tea.Msg {
			return components.URLsUpdateMsg{
				URLs: urls,
			}
		})
	}
	
	return tea.Batch(cmds...)
}

// Helper methods to extract data from model
func (a *ModelComponentAdapter) getProcessData() []interface{} {
	if a.model.processMgr == nil {
		return nil
	}
	
	processes := a.model.processMgr.GetAllProcesses()
	result := make([]interface{}, len(processes))
	for i, p := range processes {
		result[i] = p
	}
	return result
}

func (a *ModelComponentAdapter) getLogData() []string {
	if a.model.logStore == nil {
		return nil
	}
	
	// Get all logs from log store
	logs := a.model.logStore.GetAll()
	// Get last 100 logs
	if len(logs) > 100 {
		logs = logs[len(logs)-100:]
	}
	result := make([]string, len(logs))
	for i, log := range logs {
		result[i] = log.Content
	}
	return result
}

func (a *ModelComponentAdapter) getErrorData() []interface{} {
	if a.model.logStore == nil {
		return nil
	}
	
	// Get error contexts from log store
	errorContexts := a.model.logStore.GetErrorContexts()
	result := make([]interface{}, len(errorContexts))
	for i, err := range errorContexts {
		result[i] = map[string]interface{}{
			"message": err.Message,
			"process": err.ProcessName,
			"type":    err.Type,
		}
	}
	return result
}

// modelDataProvider implements the DataProvider interface for the existing Model
type modelDataProvider struct {
	model *Model
}

// GetProcesses returns process information
func (p *modelDataProvider) GetProcesses() []interface{} {
	if p.model.processMgr == nil {
		return nil
	}
	
	processes := p.model.processMgr.GetAllProcesses()
	result := make([]interface{}, len(processes))
	for i, proc := range processes {
		result[i] = proc
	}
	return result
}

// StartProcess starts a new process
func (p *modelDataProvider) StartProcess(name string) error {
	if p.model.processMgr == nil {
		return fmt.Errorf("process manager not initialized")
	}
	
	// Start a script by name
	_, err := p.model.processMgr.StartScript(name)
	return err
}

// StopProcess stops a running process
func (p *modelDataProvider) StopProcess(id string) error {
	if p.model.processMgr == nil {
		return fmt.Errorf("process manager not initialized")
	}
	
	return p.model.processMgr.StopProcess(id)
}

// GetLogs returns logs for a process
func (p *modelDataProvider) GetLogs(processID string) []string {
	if p.model.logStore == nil {
		return nil
	}
	
	logs := p.model.logStore.GetByProcess(processID)
	// Get last 100 logs for process
	if len(logs) > 100 {
		logs = logs[len(logs)-100:]
	}
	result := make([]string, len(logs))
	for i, log := range logs {
		result[i] = log.Content
	}
	return result
}

// GetFilteredLogs returns filtered logs
func (p *modelDataProvider) GetFilteredLogs(filter string) []string {
	if p.model.logStore == nil {
		return nil
	}
	
	logEntries := p.model.logStore.GetAll()
	// Get last 1000 logs
	if len(logEntries) > 1000 {
		logEntries = logEntries[len(logEntries)-1000:]
	}
	allLogs := make([]string, len(logEntries))
	for i, log := range logEntries {
		allLogs[i] = log.Content
	}
	if filter == "" {
		return allLogs
	}
	
	// Simple filter implementation
	var filtered []string
	filterLower := strings.ToLower(filter)
	for _, log := range allLogs {
		if strings.Contains(strings.ToLower(log), filterLower) {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// GetErrors returns error information
func (p *modelDataProvider) GetErrors() []interface{} {
	if p.model.logStore == nil {
		return nil
	}
	
	errorContexts := p.model.logStore.GetErrorContexts()
	result := make([]interface{}, len(errorContexts))
	for i, err := range errorContexts {
		result[i] = map[string]interface{}{
			"message": err.Message,
			"process": err.ProcessName,
			"type":    err.Type,
		}
	}
	return result
}

// ClearErrors clears all errors
func (p *modelDataProvider) ClearErrors() error {
	if p.model.logStore == nil {
		return fmt.Errorf("log store not initialized")
	}
	
	// Clear errors in log store
	p.model.logStore.ClearErrors()
	return nil
}

// GetURLs returns discovered URLs
func (p *modelDataProvider) GetURLs() []string {
	if p.model.logStore == nil {
		return nil
	}
	
	urls := p.model.logStore.GetURLs()
	result := make([]string, len(urls))
	for i, u := range urls {
		result[i] = u.URL
	}
	return result
}

// GetSettings returns current settings
func (p *modelDataProvider) GetSettings() map[string]interface{} {
	settings := make(map[string]interface{})
	
	// Get process manager settings if available
	if p.model.processMgr != nil {
		// Add process manager related settings
		settings["processCount"] = len(p.model.processMgr.GetAllProcesses())
	}
	
	return settings
}

// UpdateSetting updates a setting value
func (p *modelDataProvider) UpdateSetting(key string, value interface{}) error {
	// Currently no settings can be updated directly
	// This would need to be extended based on actual settings
	return fmt.Errorf("setting updates not implemented: %s", key)
}

// Migration utilities

// EnableComponentMode switches to component-based UI
func (a *ModelComponentAdapter) EnableComponentMode() {
	if !a.useComponents {
		a.useComponents = true
		a.initializeComponents()
	}
}

// DisableComponentMode switches back to legacy UI
func (a *ModelComponentAdapter) DisableComponentMode() {
	a.useComponents = false
}

// IsComponentMode returns whether component mode is active
func (a *ModelComponentAdapter) IsComponentMode() bool {
	return a.useComponents
}

// MigrateView migrates a specific view to use components
func (a *ModelComponentAdapter) MigrateView(view navigation.View) error {
	if !a.useComponents {
		return fmt.Errorf("component mode not enabled")
	}
	
	// Views are already migrated when component mode is enabled
	// This method is kept for API compatibility
	
	return nil
}

// Gradual migration example
func ExampleGradualMigration() {
	// Step 1: Create existing model
	_ = &Model{
		// ... existing model setup
	}
	
	// Step 2: Wrap with adapter (start in legacy mode)
	// adapter := NewModelComponentAdapter(model, false)
	
	// Step 3: Run the app (uses legacy UI)
	// program := tea.NewProgram(adapter)
	
	// Step 4: Enable component mode when ready
	// adapter.EnableComponentMode()
	
	// Step 5: Migrate views one by one
	// adapter.MigrateView(navigation.ViewProcesses)
	// adapter.MigrateView(navigation.ViewLogs)
	// etc...
}

// Feature flag for gradual rollout
func CreateModelWithFeatureFlag(useNewUI bool) tea.Model {
	// Create base model
	model := &Model{
		// ... initialize model
	}
	
	// Wrap with adapter based on feature flag
	return NewModelComponentAdapter(model, useNewUI)
}

// simpleHeaderComponent provides a basic header
type simpleHeaderComponent struct {
	components.BaseComponent
	adapter *ModelComponentAdapter
}

func (h *simpleHeaderComponent) Update(msg tea.Msg) (components.Component, tea.Cmd) {
	return h, nil
}

func (h *simpleHeaderComponent) View() string {
	title := "🎯 Brummer Process Manager"
	if h.adapter.viewSwitcher != nil {
		current := h.adapter.viewSwitcher.CurrentView()
		title += fmt.Sprintf(" - %s", strings.Title(current))
	}
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render(title)
}

// simpleFooterComponent provides a basic footer
type simpleFooterComponent struct {
	components.BaseComponent
	adapter *ModelComponentAdapter
}

func (f *simpleFooterComponent) Update(msg tea.Msg) (components.Component, tea.Cmd) {
	return f, nil
}

func (f *simpleFooterComponent) View() string {
	help := []string{
		"1-7: views",
		"q: quit",
		"?: help",
	}
	
	if f.adapter.viewSwitcher != nil {
		current := f.adapter.viewSwitcher.CurrentView()
		switch current {
		case "processes":
			help = append(help, "enter: start/stop", "r: restart")
		case "logs":
			help = append(help, "/: filter", "f: follow", "c: clear")
		case "errors":
			help = append(help, "c: clear", "enter: details")
		}
	}
	
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(strings.Join(help, " | "))
}

// Helper types that existing Model might need

// ProcessManager interface (if not already defined)
type ProcessManager interface {
	GetProcesses() []*process.Process
	StartProcess(name string, env []string) error
	StopProcess(id string) error
}

// LogManager interface
type LogManager interface {
	GetRecentLogs(limit int) []string
	GetLogsForProcess(processID string) []string
}

// ErrorManager interface
type ErrorManager interface {
	GetErrors() []ErrorInfo
	ClearErrors()
}

// URLManager interface
type URLManager interface {
	GetURLs() []URLInfo
}

// ErrorInfo represents error information
type ErrorInfo struct {
	ProcessID string
	Message   string
	Timestamp string
}

// URLInfo represents URL information
type URLInfo struct {
	URL       string
	ProcessID string
}