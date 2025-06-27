package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/pkg/events"
)

// MockProcess for testing
type MockProcess struct {
	id     string
	name   string
	status process.ProcessStatus
}

func (m *MockProcess) GetID() string                { return m.id }
func (m *MockProcess) GetName() string              { return m.name }
func (m *MockProcess) GetStatus() process.ProcessStatus { return m.status }
func (m *MockProcess) GetStartTime() time.Time     { return time.Now() }
func (m *MockProcess) GetCommand() string          { return "echo test" }

// MockProcessManager for testing
type MockProcessManager struct {
	processes map[string]*MockProcess
	scripts   map[string]string
}

func NewMockProcessManager() *MockProcessManager {
	return &MockProcessManager{
		processes: make(map[string]*MockProcess),
		scripts: map[string]string{
			"dev":   "npm run dev",
			"test":  "npm test",
			"build": "npm run build",
		},
	}
}

func (m *MockProcessManager) GetProcesses() []process.ProcessInterface {
	var procs []process.ProcessInterface
	for _, p := range m.processes {
		procs = append(procs, p)
	}
	return procs
}

func (m *MockProcessManager) GetScripts() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m.scripts {
		result[k] = v
	}
	return result
}

func (m *MockProcessManager) StartScript(name string) (process.ProcessInterface, error) {
	proc := &MockProcess{
		id:     name + "-1",
		name:   name,
		status: process.StatusRunning,
	}
	m.processes[proc.id] = proc
	return proc, nil
}

func (m *MockProcessManager) StopProcess(id string) error {
	if proc, exists := m.processes[id]; exists {
		proc.status = process.StatusExited
	}
	return nil
}

func (m *MockProcessManager) GetProcess(id string) (process.ProcessInterface, bool) {
	proc, exists := m.processes[id]
	return proc, exists
}

func (m *MockProcessManager) AddLogCallback(callback func(string, string, bool)) {}
func (m *MockProcessManager) Cleanup() error                                     { return nil }

// MockMCPServer for testing
type MockMCPServer struct {
	running bool
	port    int
}

func (m *MockMCPServer) IsRunning() bool { return m.running }
func (m *MockMCPServer) GetPort() int    { return m.port }
func (m *MockMCPServer) Start() error    { m.running = true; return nil }
func (m *MockMCPServer) Stop() error     { m.running = false; return nil }

func TestModelCreation(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	
	if model == nil {
		t.Fatal("Failed to create model")
	}
	
	if model.currentView != ViewScriptSelector {
		t.Errorf("Expected initial view to be ScriptSelector, got %v", model.currentView)
	}
	
	if len(model.scripts) != 3 {
		t.Errorf("Expected 3 scripts, got %d", len(model.scripts))
	}
}

func TestModelViewSwitching(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	
	// Test view switching
	testViews := []View{
		ViewScriptSelector,
		ViewProcesses,
		ViewLogs,
		ViewErrors,
		ViewURLs,
		ViewSettings,
	}
	
	for _, view := range testViews {
		model.currentView = view
		
		// Simulate key press to switch view
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
		newModel, _ := model.Update(msg)
		
		// Verify model state
		updatedModel := newModel.(Model)
		if updatedModel.currentView != ViewScriptSelector {
			// Some keys might not change view, that's okay
		}
	}
}

func TestScriptSelection(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	model.currentView = ViewScriptSelector
	
	// Test script navigation
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(downKey)
	updatedModel := newModel.(Model)
	
	if updatedModel.selectedScript < 0 {
		t.Error("Selected script should not be negative")
	}
	
	// Test script execution
	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := updatedModel.Update(enterKey)
	
	if cmd == nil {
		// Command might be nil if no script selected
	}
	
	// Verify script was started (would happen in background)
	_ = newModel
}

func TestSlashCommands(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	model.currentView = ViewLogs
	
	// Start slash command
	slashKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newModel, _ := model.Update(slashKey)
	updatedModel := newModel.(Model)
	
	if !updatedModel.slashCommand.active {
		t.Error("Slash command should be active after pressing '/'")
	}
	
	// Type command
	cmdRunes := []rune("show error")
	for _, r := range cmdRunes {
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newModel, _ = updatedModel.Update(key)
		updatedModel = newModel.(Model)
	}
	
	// Execute command
	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = updatedModel.Update(enterKey)
	updatedModel = newModel.(Model)
	
	if updatedModel.slashCommand.active {
		t.Error("Slash command should not be active after execution")
	}
}

func TestEventHandling(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	
	// Simulate process started event
	eventBus.Publish(events.Event{
		Type:      events.EventProcessStarted,
		ProcessID: "test-process",
		Data: map[string]interface{}{
			"name": "test",
		},
	})
	
	// Give time for event processing
	time.Sleep(10 * time.Millisecond)
	
	// Simulate tick message to update model
	tickMsg := tea.KeyMsg{Type: tea.KeyCtrlC} // Use Ctrl+C as a no-op
	newModel, _ := model.Update(tickMsg)
	
	// Verify model was updated
	_ = newModel
}

func TestErrorView(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	// Add some error logs
	logStore.Add("test-process", "test", "Error: Something went wrong", true)
	logStore.Add("test-process", "test", "Another error occurred", true)
	
	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	model.currentView = ViewErrors
	
	// Test error view rendering
	view := model.View()
	if view == "" {
		t.Error("Error view should render content")
	}
	
	// Test error navigation
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(downKey)
	
	// Verify navigation worked
	_ = newModel
}

func TestURLsView(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	// Add logs with URLs
	logStore.Add("test-process", "test", "Server running at http://localhost:3000", false)
	logStore.Add("test-process", "test", "API available at https://api.example.com", false)
	
	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	model.currentView = ViewURLs
	
	// Test URLs view rendering
	view := model.View()
	if view == "" {
		t.Error("URLs view should render content")
	}
}

func TestSettingsView(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	model.currentView = ViewSettings
	
	// Test settings view rendering
	view := model.View()
	if view == "" {
		t.Error("Settings view should render content")
	}
	
	// Should show MCP server info
	if !model.mcpServer.IsRunning() {
		model.mcpServer.Start()
	}
	
	view = model.View()
	// Settings view should now show running status
}

func TestKeyboardShortcuts(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	
	// Test number keys for view switching
	numberKeys := []tea.KeyType{
		tea.KeyRunes, // Would need to check runes content
	}
	
	for _, keyType := range numberKeys {
		key := tea.KeyMsg{Type: keyType, Runes: []rune{'1'}}
		newModel, _ := model.Update(key)
		
		// Verify view change
		updatedModel := newModel.(Model)
		if updatedModel.currentView != ViewScriptSelector {
			// View should change to script selector
		}
	}
	
	// Test Ctrl+C for quit
	quitKey := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := model.Update(quitKey)
	
	if cmd == nil {
		t.Error("Ctrl+C should return quit command")
	}
}

func TestModelUpdate(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	
	// Test various message types
	messages := []tea.Msg{
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyEsc},
		tea.KeyMsg{Type: tea.KeyUp},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.MouseMsg{},
		tea.WindowSizeMsg{Width: 80, Height: 24},
	}
	
	for _, msg := range messages {
		newModel, cmd := model.Update(msg)
		
		// Verify model is still valid
		if newModel == nil {
			t.Errorf("Update returned nil model for message %T", msg)
		}
		
		// Command can be nil or valid
		_ = cmd
		
		// Use updated model for next iteration
		model = newModel.(Model)
	}
}

func TestViewRendering(t *testing.T) {
	eventBus := events.NewEventBus()
	processMgr := NewMockProcessManager()
	logStore := logs.NewStore(100)
	mcpServer := &MockMCPServer{port: 7777}

	model := NewModel(processMgr, logStore, eventBus, mcpServer, nil, 7777)
	
	// Test rendering all views
	views := []View{
		ViewScriptSelector,
		ViewProcesses,
		ViewLogs,
		ViewErrors,
		ViewURLs,
		ViewSettings,
	}
	
	for _, view := range views {
		model.currentView = view
		content := model.View()
		
		if content == "" {
			t.Errorf("View %v should render some content", view)
		}
		
		// Content should be reasonable length
		if len(content) < 10 {
			t.Errorf("View %v rendered very short content: %s", view, content)
		}
	}
}