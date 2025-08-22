package tui

import (
	"fmt"
	"os"
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/config"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/pkg/events"
)

// ComponentSystem is an interface for component-based UI systems
type ComponentSystem interface {
	tea.Model
	View() string
}

// ModelWithComponents wraps the existing Model with the component system
// This provides a gradual migration path while maintaining backward compatibility
type ModelWithComponents struct {
	// Core model for business logic
	baseModel *Model
	
	// Component system for UI (when enabled)
	componentBridge ComponentSystem
	
	// Feature flag for enabling component mode
	useComponents bool
	
	// Data provider adapter
	dataProvider *BrummerDataProvider
}

// NewModelWithComponents creates a model that can use the component system
func NewModelWithComponents(
	processMgr *process.Manager,
	logStore *logs.Store,
	eventBus *events.EventBus,
	mcpServer MCPServerInterface,
	proxyServer *proxy.Server,
	mcpPort int,
	cfg *config.Config,
	useComponents bool,
) *ModelWithComponents {
	// Create base model
	baseModel := NewModel(processMgr, logStore, eventBus, mcpServer, proxyServer, mcpPort, cfg)
	
	mwc := &ModelWithComponents{
		baseModel:     baseModel,
		useComponents: useComponents,
	}
	
	if useComponents {
		// Use minimal shell for testing
		if useMinimalShell := os.Getenv("BRUMMER_MINIMAL_SHELL") == "true"; useMinimalShell {
			// Check which version to use
			if os.Getenv("BRUMMER_SHELL_V2") == "true" {
				// Use improved V2 shell
				mwc.componentBridge = components.NewMinimalShellV2()
			} else {
				// Use original minimal shell
				mwc.componentBridge = components.NewMinimalShell()
			}
		} else {
			// Create data provider that adapts the Model
			mwc.dataProvider = NewBrummerDataProvider(baseModel)
			
			// Create component bridge
			mwc.componentBridge = components.NewModelBridge(mwc.dataProvider)
		}
	}
	
	return mwc
}

// Init initializes the model
func (m *ModelWithComponents) Init() tea.Cmd {
	var cmds []tea.Cmd
	
	// Always initialize base model for data
	if cmd := m.baseModel.Init(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	
	// Initialize component system if enabled
	if m.useComponents && m.componentBridge != nil {
		if cmd := m.componentBridge.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	return tea.Batch(cmds...)
}

// Update handles all messages
func (m *ModelWithComponents) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Always update base model for data changes
	newModel, modelCmd := m.baseModel.Update(msg)
	m.baseModel = newModel.(*Model)
	if modelCmd != nil {
		cmds = append(cmds, modelCmd)
	}
	
	// If using components, let component system handle UI
	if m.useComponents && m.componentBridge != nil {
		_, compCmd := m.componentBridge.Update(msg)
		if compCmd != nil {
			cmds = append(cmds, compCmd)
		}
		
		// Send data updates to components when needed
		if dataCmd := m.syncDataToComponents(); dataCmd != nil {
			cmds = append(cmds, dataCmd)
		}
	}
	
	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m *ModelWithComponents) View() string {
	if m.useComponents && m.componentBridge != nil {
		// Use component system for rendering
		return m.componentBridge.View()
	}
	// Fall back to base model rendering
	return m.baseModel.View()
}

// syncDataToComponents sends data updates from Model to component system
func (m *ModelWithComponents) syncDataToComponents() tea.Cmd {
	if !m.useComponents || m.dataProvider == nil {
		return nil
	}
	
	var cmds []tea.Cmd
	
	// Check for process updates
	if processes := m.dataProvider.GetProcesses(); len(processes) > 0 {
		cmds = append(cmds, components.SendDataUpdate(
			components.DataUpdateProcess,
			processes,
			"model",
		))
	}
	
	// Check for log updates
	if logs := m.dataProvider.GetLogs(""); len(logs) > 0 {
		cmds = append(cmds, components.SendDataUpdate(
			components.DataUpdateLogs,
			logs,
			"model",
		))
	}
	
	// Check for error updates
	if errors := m.dataProvider.GetErrors(); len(errors) > 0 {
		cmds = append(cmds, components.SendDataUpdate(
			components.DataUpdateErrors,
			errors,
			"model",
		))
	}
	
	// Check for URL updates
	if urls := m.dataProvider.GetURLs(); len(urls) > 0 {
		cmds = append(cmds, components.SendDataUpdate(
			components.DataUpdateURLs,
			urls,
			"model",
		))
	}
	
	return tea.Batch(cmds...)
}

// EnableComponents switches to component-based UI
func (m *ModelWithComponents) EnableComponents() {
	if !m.useComponents {
		m.useComponents = true
		m.dataProvider = NewBrummerDataProvider(m.baseModel)
		m.componentBridge = components.NewModelBridge(m.dataProvider)
	}
}

// DisableComponents switches back to legacy UI
func (m *ModelWithComponents) DisableComponents() {
	m.useComponents = false
}

// IsComponentMode returns whether component mode is active
func (m *ModelWithComponents) IsComponentMode() bool {
	return m.useComponents
}

// BrummerDataProvider adapts the Brummer Model to the DataProvider interface
type BrummerDataProvider struct {
	model *Model
}

// NewBrummerDataProvider creates a data provider for the Brummer model
func NewBrummerDataProvider(model *Model) *BrummerDataProvider {
	return &BrummerDataProvider{model: model}
}

// GetProcesses returns process information
func (p *BrummerDataProvider) GetProcesses() []interface{} {
	if p.model.processMgr == nil {
		return nil
	}
	
	processes := p.model.processMgr.GetAllProcesses()
	result := make([]interface{}, len(processes))
	for i, proc := range processes {
		// Convert to a simpler format for components
		result[i] = map[string]interface{}{
			"id":     proc.ID,
			"name":   proc.Name,
			"status": proc.GetStateAtomic().Status,
		}
	}
	return result
}

// StartProcess starts a new process
func (p *BrummerDataProvider) StartProcess(name string) error {
	if p.model.processMgr == nil {
		return fmt.Errorf("process manager not initialized")
	}
	
	// Try to start as a script first
	_, err := p.model.processMgr.StartScript(name)
	if err != nil {
		// If not a script, try as custom command
		_, err = p.model.processMgr.StartCommand(name, name, []string{})
	}
	return err
}

// StopProcess stops a running process
func (p *BrummerDataProvider) StopProcess(id string) error {
	if p.model.processMgr == nil {
		return fmt.Errorf("process manager not initialized")
	}
	
	return p.model.processMgr.StopProcess(id)
}

// GetLogs returns logs for a process
func (p *BrummerDataProvider) GetLogs(processID string) []string {
	if p.model.logStore == nil {
		return nil
	}
	
	// Get logs from the store
	var logs []string
	if processID == "" {
		// Get all recent logs
		entries := p.model.logStore.GetAll()
		// Get last 100 entries
		if len(entries) > 100 {
			entries = entries[len(entries)-100:]
		}
		logs = make([]string, len(entries))
		for i, entry := range entries {
			logs[i] = fmt.Sprintf("[%s] %s: %s", 
				entry.Timestamp.Format("15:04:05"),
				entry.ProcessID,
				entry.Content)
		}
	} else {
		// Get logs for specific process
		entries := p.model.logStore.GetByProcess(processID)
		// Get last 100 entries
		if len(entries) > 100 {
			entries = entries[len(entries)-100:]
		}
		logs = make([]string, len(entries))
		for i, entry := range entries {
			logs[i] = entry.Content
		}
	}
	
	return logs
}

// GetFilteredLogs returns filtered logs
func (p *BrummerDataProvider) GetFilteredLogs(filter string) []string {
	logs := p.GetLogs("")
	if filter == "" {
		return logs
	}
	
	// Simple filter implementation
	var filtered []string
	for _, log := range logs {
		if containsIgnoreCase(log, filter) {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// GetErrors returns error information
func (p *BrummerDataProvider) GetErrors() []interface{} {
	if p.model.logStore == nil {
		return nil
	}
	
	// Get error contexts from log store
	errorContexts := p.model.logStore.GetErrorContexts()
	result := make([]interface{}, len(errorContexts))
	for i, err := range errorContexts {
		result[i] = map[string]interface{}{
			"process": err.ProcessName,
			"message": err.Message,
			"time":    err.Timestamp,
		}
	}
	return result
}

// ClearErrors clears all errors
func (p *BrummerDataProvider) ClearErrors() error {
	if p.model.logStore == nil {
		return fmt.Errorf("log store not initialized")
	}
	
	p.model.logStore.ClearErrors()
	return nil
}

// GetURLs returns discovered URLs
func (p *BrummerDataProvider) GetURLs() []string {
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
func (p *BrummerDataProvider) GetSettings() map[string]interface{} {
	settings := make(map[string]interface{})
	
	// Get process manager settings
	if p.model.processMgr != nil {
		// Add available settings from process manager
		settings["processCount"] = len(p.model.processMgr.GetAllProcesses())
	}
	
	// Get MCP settings
	if p.model.mcpServer != nil {
		settings["mcpEnabled"] = p.model.mcpServer.IsRunning()
		settings["mcpPort"] = p.model.mcpServer.GetPort()
	}
	
	// Get proxy settings
	if p.model.proxyServer != nil {
		settings["proxyEnabled"] = p.model.proxyServer.IsRunning()
	}
	
	return settings
}

// UpdateSetting updates a setting value
func (p *BrummerDataProvider) UpdateSetting(key string, value interface{}) error {
	switch key {
	// Currently no settings can be updated directly
	// Add cases as needed
	default:
		return fmt.Errorf("unknown setting: %s", key)
	}
	
	return fmt.Errorf("invalid value for setting: %s", key)
}

// Helper function for case-insensitive string contains
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// CreateModelWithComponentFlag creates a model with optional component system
// This is the main entry point for using the component system
func CreateModelWithComponentFlag(
	processMgr *process.Manager,
	logStore *logs.Store,
	eventBus *events.EventBus,
	mcpServer MCPServerInterface,
	proxyServer *proxy.Server,
	mcpPort int,
	cfg *config.Config,
	useComponents bool,
) tea.Model {
	return NewModelWithComponents(
		processMgr,
		logStore,
		eventBus,
		mcpServer,
		proxyServer,
		mcpPort,
		cfg,
		useComponents,
	)
}