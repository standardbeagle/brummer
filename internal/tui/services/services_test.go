package services

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/components"
)

// FakeProcessService for testing
type FakeProcessService struct {
	processes []ProcessInfo
	scripts   map[string]string
	started   []string
	stopped   []string
	restarted []string
}

func (f *FakeProcessService) Start(name string) tea.Cmd {
	f.started = append(f.started, name)
	return func() tea.Msg {
		return components.CommandResultMsg{
			Success: true,
			Message: "Started " + name,
		}
	}
}

func (f *FakeProcessService) Stop(processID string) tea.Cmd {
	f.stopped = append(f.stopped, processID)
	return func() tea.Msg {
		return components.CommandResultMsg{
			Success: true,
			Message: "Stopped " + processID,
		}
	}
}

func (f *FakeProcessService) Restart(processID string) tea.Cmd {
	f.restarted = append(f.restarted, processID)
	return func() tea.Msg {
		return components.CommandResultMsg{
			Success: true,
			Message: "Restarted " + processID,
		}
	}
}

func (f *FakeProcessService) GetStatus() []ProcessInfo {
	return f.processes
}

func (f *FakeProcessService) GetScripts() map[string]string {
	return f.scripts
}

// FakeLogsService for testing
type FakeLogsService struct {
	logs     []LogEntry
	cleared  []string
	searches []string
}

func (f *FakeLogsService) GetLogs(processID string, limit int) []LogEntry {
	if limit > 0 && limit < len(f.logs) {
		return f.logs[:limit]
	}
	return f.logs
}

func (f *FakeLogsService) FilterByLevel(level string) []LogEntry {
	var filtered []LogEntry
	for _, log := range f.logs {
		if log.Level == level {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

func (f *FakeLogsService) Search(query string) []LogEntry {
	f.searches = append(f.searches, query)
	var results []LogEntry
	for _, log := range f.logs {
		if strings.Contains(log.Message, query) {
			results = append(results, log)
		}
	}
	return results
}

func (f *FakeLogsService) Clear(processID string) tea.Cmd {
	f.cleared = append(f.cleared, processID)
	return func() tea.Msg {
		return components.CommandResultMsg{
			Success: true,
			Message: "Cleared logs",
		}
	}
}

func (f *FakeLogsService) Subscribe() tea.Cmd {
	return func() tea.Msg {
		return components.CommandResultMsg{
			Success: true,
			Message: "Subscribed",
		}
	}
}

// FakeProxyService for testing
type FakeProxyService struct {
	enabled  bool
	toggled  int
	setCount int
}

func (f *FakeProxyService) SetEnabled(enabled bool) tea.Cmd {
	f.enabled = enabled
	f.setCount++
	return func() tea.Msg {
		return components.CommandResultMsg{
			Success: true,
			Message: "Proxy set",
		}
	}
}

func (f *FakeProxyService) Toggle() tea.Cmd {
	f.enabled = !f.enabled
	f.toggled++
	return func() tea.Msg {
		return components.CommandResultMsg{
			Success: true,
			Message: "Proxy toggled",
		}
	}
}

func (f *FakeProxyService) GetInfo() ProxyInfo {
	return ProxyInfo{
		Enabled: f.enabled,
		Port:    3000,
	}
}

func (f *FakeProxyService) GetRequests(limit int) []ProxyRequest {
	return []ProxyRequest{}
}

// Tests

func TestServiceCommandExecutor_ProcessCommands(t *testing.T) {
	processService := &FakeProcessService{
		scripts: map[string]string{
			"dev":  "Development server",
			"test": "Test runner",
		},
	}
	
	executor := NewServiceCommandExecutor(processService, nil, nil)
	
	// Test start command
	cmd := executor.Execute("/start dev")
	if cmd == nil {
		t.Error("Expected command for start")
	}
	
	// Execute the command to trigger the service
	msg := cmd()
	if result, ok := msg.(components.CommandResultMsg); ok {
		if !result.Success {
			t.Error("Expected successful start")
		}
	}
	
	if len(processService.started) != 1 || processService.started[0] != "dev" {
		t.Errorf("Expected 'dev' to be started, got %v", processService.started)
	}
	
	// Test stop command
	cmd = executor.Execute("/stop process-1")
	if cmd == nil {
		t.Error("Expected command for stop")
	}
	
	msg = cmd()
	if result, ok := msg.(components.CommandResultMsg); ok {
		if !result.Success {
			t.Error("Expected successful stop")
		}
	}
	
	if len(processService.stopped) != 1 || processService.stopped[0] != "process-1" {
		t.Errorf("Expected 'process-1' to be stopped, got %v", processService.stopped)
	}
}

func TestServiceCommandExecutor_ProxyCommands(t *testing.T) {
	proxyService := &FakeProxyService{}
	executor := NewServiceCommandExecutor(nil, nil, proxyService)
	
	// Test proxy on
	cmd := executor.Execute("/proxy on")
	if cmd == nil {
		t.Error("Expected command for proxy on")
	}
	
	msg := cmd()
	if result, ok := msg.(components.CommandResultMsg); ok {
		if !result.Success {
			t.Error("Expected successful proxy enable")
		}
	}
	
	if !proxyService.enabled {
		t.Error("Expected proxy to be enabled")
	}
	
	// Test proxy toggle
	cmd = executor.Execute("/proxy toggle")
	msg = cmd()
	
	if proxyService.enabled {
		t.Error("Expected proxy to be toggled off")
	}
	
	if proxyService.toggled != 1 {
		t.Errorf("Expected 1 toggle, got %d", proxyService.toggled)
	}
}

func TestServiceCommandExecutor_NavigationCommands(t *testing.T) {
	executor := NewServiceCommandExecutor(nil, nil, nil)
	
	// Test navigation commands
	testCases := []string{"logs", "processes", "errors", "urls", "web", "ai", "settings"}
	
	for _, tc := range testCases {
		cmd := executor.Execute("/" + tc)
		if cmd == nil {
			t.Errorf("Expected command for /%s", tc)
			continue
		}
		
		msg := cmd()
		if navMsg, ok := msg.(components.NavigateToViewMsg); ok {
			if navMsg.View != tc {
				t.Errorf("Expected navigation to %s, got %s", tc, navMsg.View)
			}
		} else {
			t.Errorf("Expected NavigateToViewMsg for /%s", tc)
		}
	}
}

func TestLogsService_Operations(t *testing.T) {
	logs := &FakeLogsService{
		logs: []LogEntry{
			{Level: "info", Message: "Starting server", ProcessID: "p1"},
			{Level: "error", Message: "Connection failed", ProcessID: "p1"},
			{Level: "warn", Message: "Slow response", ProcessID: "p2"},
		},
	}
	
	// Test GetLogs with limit
	results := logs.GetLogs("", 2)
	if len(results) != 2 {
		t.Errorf("Expected 2 logs with limit, got %d", len(results))
	}
	
	// Test FilterByLevel
	errors := logs.FilterByLevel("error")
	if len(errors) != 1 {
		t.Errorf("Expected 1 error log, got %d", len(errors))
	}
	
	// Test Search
	searchResults := logs.Search("server")
	if len(searchResults) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(searchResults))
	}
	
	if len(logs.searches) != 1 || logs.searches[0] != "server" {
		t.Errorf("Expected search for 'server', got %v", logs.searches)
	}
	
	// Test Clear
	cmd := logs.Clear("p1")
	msg := cmd()
	
	if result, ok := msg.(components.CommandResultMsg); ok {
		if !result.Success {
			t.Error("Expected successful clear")
		}
	}
	
	if len(logs.cleared) != 1 || logs.cleared[0] != "p1" {
		t.Errorf("Expected 'p1' to be cleared, got %v", logs.cleared)
	}
}

func TestProcessService_GetStatus(t *testing.T) {
	service := &FakeProcessService{
		processes: []ProcessInfo{
			{ID: "p1", Name: "dev", Status: "running"},
			{ID: "p2", Name: "test", Status: "stopped"},
		},
		scripts: map[string]string{
			"dev":  "Development server",
			"test": "Test runner",
		},
	}
	
	// Test GetStatus
	status := service.GetStatus()
	if len(status) != 2 {
		t.Errorf("Expected 2 processes, got %d", len(status))
	}
	
	// Test GetScripts
	scripts := service.GetScripts()
	if len(scripts) != 2 {
		t.Errorf("Expected 2 scripts, got %d", len(scripts))
	}
	
	if scripts["dev"] != "Development server" {
		t.Errorf("Expected 'Development server' for dev script, got %s", scripts["dev"])
	}
}