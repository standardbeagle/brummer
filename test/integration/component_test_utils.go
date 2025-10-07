package integration

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/mock"

	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
)

// MockProcessManager is a mock implementation of process.Manager for testing
type MockProcessManager struct {
	mock.Mock
	processes map[string]*process.Process
}

func NewMockProcessManager() *MockProcessManager {
	return &MockProcessManager{
		processes: make(map[string]*process.Process),
	}
}

func (m *MockProcessManager) GetAllProcesses() []*process.Process {
	args := m.Called()
	if result := args.Get(0); result != nil {
		return result.([]*process.Process)
	}

	// Return internal processes if not mocked
	processes := make([]*process.Process, 0, len(m.processes))
	for _, proc := range m.processes {
		processes = append(processes, proc)
	}
	return processes
}

func (m *MockProcessManager) GetProcess(id string) *process.Process {
	args := m.Called(id)
	if result := args.Get(0); result != nil {
		return result.(*process.Process)
	}
	return m.processes[id]
}

func (m *MockProcessManager) StartScript(scriptName string) (string, error) {
	args := m.Called(scriptName)
	return args.String(0), args.Error(1)
}

func (m *MockProcessManager) StartCommand(command string) (string, error) {
	args := m.Called(command)
	return args.String(0), args.Error(1)
}

func (m *MockProcessManager) StopProcess(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockProcessManager) RestartProcess(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// Note: GetURLs is actually on logs.Store, not process.Manager
// This method is here for backward compatibility in tests

// AddTestProcess adds a test process to the mock
func (m *MockProcessManager) AddTestProcess(id string, name string, status process.ProcessStatus) {
	m.processes[id] = &process.Process{
		ID:     id,
		Name:   name,
		Script: name,
		Status: status,
	}
}

// MockLogStore is a mock implementation of logs.Store for testing
type MockLogStore struct {
	mock.Mock
	logs []logs.LogEntry
}

func NewMockLogStore() *MockLogStore {
	return &MockLogStore{
		logs: []logs.LogEntry{},
	}
}

func (m *MockLogStore) GetLogs(processID string, limit int) []logs.LogEntry {
	args := m.Called(processID, limit)
	if result := args.Get(0); result != nil {
		return result.([]logs.LogEntry)
	}

	// Filter by processID if specified
	if processID != "" {
		filtered := []logs.LogEntry{}
		for _, log := range m.logs {
			if log.ProcessID == processID {
				filtered = append(filtered, log)
			}
		}
		return filtered
	}

	return m.logs
}

func (m *MockLogStore) Add(processID, processName, content string, isError bool) *logs.LogEntry {
	args := m.Called(processID, processName, content, isError)
	_ = args

	entry := logs.LogEntry{
		ProcessID:   processID,
		ProcessName: processName,
		Content:     content,
		IsError:     isError,
	}
	if isError {
		entry.Level = logs.LevelError
	} else {
		entry.Level = logs.LevelInfo
	}
	m.logs = append(m.logs, entry)
	return &entry
}

func (m *MockLogStore) ClearLogs() {
	args := m.Called()
	_ = args
	m.logs = []logs.LogEntry{}
}

func (m *MockLogStore) ClearLogsForProcess(processName string) {
	args := m.Called(processName)
	_ = args

	filtered := []logs.LogEntry{}
	for _, log := range m.logs {
		if log.ProcessID != processName {
			filtered = append(filtered, log)
		}
	}
	m.logs = filtered
}

func (m *MockLogStore) Subscribe(handler func(logs.LogEntry)) {
	args := m.Called(handler)
	_ = args
}

// AddTestLog adds a test log entry
func (m *MockLogStore) AddTestLog(processID string, level logs.LogLevel, content string) {
	isError := level == logs.LevelError || level == logs.LevelCritical
	entry := logs.LogEntry{
		ProcessID: processID,
		Level:     level,
		Content:   content,
		IsError:   isError,
	}
	m.logs = append(m.logs, entry)
}

// MockProxyServer is a mock implementation of proxy.Server for testing
type MockProxyServer struct {
	mock.Mock
	requests []proxy.Request
}

func NewMockProxyServer() *MockProxyServer {
	return &MockProxyServer{
		requests: []proxy.Request{},
	}
}

func (m *MockProxyServer) GetRequests(processName string, limit int) []proxy.Request {
	args := m.Called(processName, limit)
	if result := args.Get(0); result != nil {
		return result.([]proxy.Request)
	}

	// Filter by processName if specified
	if processName != "" {
		filtered := []proxy.Request{}
		for _, req := range m.requests {
			if req.ProcessName == processName {
				filtered = append(filtered, req)
			}
		}
		return filtered
	}

	return m.requests
}

func (m *MockProxyServer) GetRequestDetail(id string) (*proxy.Request, error) {
	args := m.Called(id)
	if result := args.Get(0); result != nil {
		return result.(*proxy.Request), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProxyServer) ClearRequests(processName string) {
	args := m.Called(processName)
	_ = args

	if processName == "" {
		m.requests = []proxy.Request{}
	} else {
		filtered := []proxy.Request{}
		for _, req := range m.requests {
			if req.ProcessName != processName {
				filtered = append(filtered, req)
			}
		}
		m.requests = filtered
	}
}

// AddTestRequest adds a test HTTP request
func (m *MockProxyServer) AddTestRequest(processName string, method string, url string, statusCode int) {
	m.requests = append(m.requests, proxy.Request{
		ID:          "test-req-id",
		ProcessName: processName,
		Method:      method,
		URL:         url,
		StatusCode:  statusCode,
	})
}

// MockEventBus is a mock implementation of events.EventBus for testing
type MockEventBus struct {
	mock.Mock
	handlers map[string][]func(events.Event)
}

func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		handlers: make(map[string][]func(events.Event)),
	}
}

func (m *MockEventBus) Emit(eventName string, data interface{}) {
	args := m.Called(eventName, data)
	_ = args

	// Call registered handlers
	var eventData map[string]interface{}
	if dataMap, ok := data.(map[string]interface{}); ok {
		eventData = dataMap
	} else {
		eventData = map[string]interface{}{"data": data}
	}

	event := events.Event{
		Type: events.EventType(eventName),
		Data: eventData,
	}

	if handlers, ok := m.handlers[eventName]; ok {
		for _, handler := range handlers {
			handler(event)
		}
	}
}

func (m *MockEventBus) Subscribe(pattern string, handler func(events.Event)) {
	args := m.Called(pattern, handler)
	_ = args

	m.handlers[pattern] = append(m.handlers[pattern], handler)
}

func (m *MockEventBus) Unsubscribe(pattern string) {
	args := m.Called(pattern)
	_ = args

	delete(m.handlers, pattern)
}

// ComponentTestCase defines a reusable test case structure
type ComponentTestCase struct {
	Name        string
	Setup       func(*TestContext)
	Action      func(*TestContext)
	Verify      func(*TestContext)
	Timeout     int // seconds
}

// RunComponentTests runs a series of component test cases
func RunComponentTests(t *testing.T, cases []ComponentTestCase) {
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := NewTestContext(t)
			defer ctx.Cleanup()

			if tc.Setup != nil {
				tc.Setup(ctx)
			}

			if tc.Action != nil {
				tc.Action(ctx)
			}

			if tc.Verify != nil {
				tc.Verify(ctx)
			}
		})
	}
}

// KeyPressSequence simulates a sequence of key presses for testing
type KeyPressSequence struct {
	Keys  []string
	Delay int // milliseconds between keys
}

// Execute performs the key press sequence
func (s *KeyPressSequence) Execute(ctx *TestContext) {
	for _, key := range s.Keys {
		ctx.SendKey(key)
		if s.Delay > 0 {
			// Small delay handled in SendKey
		}
	}
}

// ViewAssertion defines an assertion about the rendered view
type ViewAssertion struct {
	Contains    []string
	NotContains []string
}

// Assert performs the view assertions
func (a *ViewAssertion) Assert(ctx *TestContext) {
	for _, text := range a.Contains {
		ctx.AssertViewContains(text)
	}

	for _, text := range a.NotContains {
		ctx.AssertViewNotContains(text)
	}
}

// CreateTestMessage creates a custom test message
func CreateTestMessage(msgType string, data interface{}) tea.Msg {
	// This is a helper for creating custom messages
	// Specific message types should be defined in their respective packages
	return struct {
		Type string
		Data interface{}
	}{
		Type: msgType,
		Data: data,
	}
}
