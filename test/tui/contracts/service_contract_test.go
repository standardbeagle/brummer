package contracts

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Extend the mock types from update_contract_test.go
type LogEntry struct {
	Text      string
	Timestamp time.Time
	ProcessID string
	Stream    string
}

// Error message type for testing
type ErrorMsg struct {
	Error   error
	Context string
}

// Mock ProcessService for testing the contract
type MockProcessService struct {
	workDir string
	env     []string
}

// Mock implementations that will be replaced by actual ones
func (s *MockProcessService) Start(name, cmd string) tea.Cmd {
	return func() tea.Msg {
		// Stub implementation
		return ProcessStartedMsg{
			Process: &Process{
				ID:   "mock-id",
				Name: name,
				Cmd:  cmd,
			},
		}
	}
}

func (s *MockProcessService) Stop(id string) tea.Cmd {
	return func() tea.Msg {
		return ProcessStoppedMsg{
			ProcessID: id,
			ExitCode:  0,
		}
	}
}

func (s *MockProcessService) Restart(id string) tea.Cmd {
	return func() tea.Msg {
		// For restart, we'd typically stop then start
		return ProcessStartedMsg{
			Process: &Process{
				ID:   id,
				Name: "restarted",
				Cmd:  "echo restarted",
			},
		}
	}
}

func (s *MockProcessService) GetOutput(id string) tea.Cmd {
	return func() tea.Msg {
		return ProcessOutputMsg{
			ProcessID: id,
			Line:      "mock output",
			Stream:    "stdout",
		}
	}
}

// Mock LogService for testing the contract
type MockLogService struct {
	logDir string
}

func (s *MockLogService) Load(processID string) tea.Cmd {
	return func() tea.Msg {
		return struct {
			Logs      []LogEntry
			ProcessID string
		}{
			Logs: []LogEntry{
				{Text: "mock log line", ProcessID: processID},
			},
			ProcessID: processID,
		}
	}
}

func (s *MockLogService) Save(logs []LogEntry, path string) tea.Cmd {
	return func() tea.Msg {
		return struct {
			Success bool
			Path    string
			Count   int
		}{
			Success: true,
			Path:    path,
			Count:   len(logs),
		}
	}
}

func (s *MockLogService) Stream(processID string) tea.Cmd {
	return func() tea.Msg {
		return ProcessOutputMsg{
			ProcessID: processID,
			Line:      "streaming log line",
			Stream:    "stdout",
		}
	}
}

func (s *MockLogService) Filter(logs []LogEntry, query string) []LogEntry {
	var filtered []LogEntry
	for _, log := range logs {
		if contains(log.Text, query) {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// Helper function for string containment
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}

// Test ProcessService contract with mock
func TestProcessServiceContractWithMock(t *testing.T) {
	service := &MockProcessService{
		workDir: "/tmp",
		env:     []string{"PATH=/usr/bin"},
	}

	TestProcessServiceContract(t, service)
	TestServiceStatelessness(t, service)
}

// Test LogService contract with mock
func TestLogServiceContractWithMock(t *testing.T) {
	service := &MockLogService{
		logDir: "/tmp/logs",
	}

	TestLogServiceContract(t, service)
	TestServiceStatelessness(t, service)
}

// Test command immutability principle
func TestCommandImmutabilityWithMock(t *testing.T) {
	service := &MockProcessService{}

	// Commands should not modify external state
	cmd := service.Start("test", "echo test")

	// Execute command multiple times - should produce consistent results
	msg1 := cmd()
	msg2 := cmd()

	// Both should produce the same type of message
	if (msg1 == nil) != (msg2 == nil) {
		t.Error("Command execution should be consistent")
	}

	// Verify message types are expected
	switch msg1.(type) {
	case ProcessStartedMsg, ErrorMsg:
		// Expected types
	default:
		t.Errorf("Unexpected message type: %T", msg1)
	}

	TestCommandImmutability(t)
}

// Integration test when actual implementation is ready
func TestServiceContractImplementation(t *testing.T) {
	// This test will be implemented when we have the actual services
	// For now, it serves as a placeholder showing the intended usage
	t.Skip("Implementation not ready - will be added in Phase 5")

	// When implementation is ready:
	// processService := NewProcessService(workDir, env)
	// logService := NewLogService(logDir)
	//
	// TestProcessServiceContract(t, processService)
	// TestLogServiceContract(t, logService)
	// TestServiceStatelessness(t, processService)
	// TestServiceStatelessness(t, logService)
}

// Test service statelessness at compile time
func TestServiceStatelessnessCompileTime(t *testing.T) {
	// This test documents the statelessness requirements
	// Services should only contain configuration and dependencies

	// ✅ Good: Only configuration
	type GoodService struct {
		workDir  string
		env      []string
		timeout  time.Duration
	}

	// ✅ Good: Configuration + dependencies
	type GoodServiceWithDeps struct {
		workDir string
		logger  interface{} // Logger interface
		config  interface{} // Config interface
	}

	// ❌ Bad: Contains mutable state
	type BadService struct {
		workDir   string
		processes map[string]*Process // ❌ Mutable state
		counter   int                  // ❌ Mutable counter
		cache     []string             // ❌ Mutable cache
	}

	// The test passes by documenting these principles
	// Actual enforcement happens through code review and architecture

	t.Log("Services must be stateless - only configuration and dependencies allowed")
	_ = GoodService{}
	_ = GoodServiceWithDeps{}
	_ = BadService{} // This would be caught in code review
}