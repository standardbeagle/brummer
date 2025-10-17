package commands

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestCommandResponseTime tests that commands execute within <50ms
// This test should FAIL initially before performance optimizations are implemented
func TestCommandResponseTime(t *testing.T) {
	// Create a simple command that should be fast
	cmd := &TestCommand{
		CmdID:        "response-time-test",
		CmdType:      CommandTypeIO,
		CmdPriority:  PriorityHigh,
		CmdContext:   DefaultOperationContext(1),
		CmdCreatedAt: time.Now(),
		Operation: func() (interface{}, error) {
			// Simulate a very fast operation
			return "fast result", nil
		},
		shouldSleep: false,
	}

	// Measure execution time
	start := time.Now()
	result := cmd.Run()
	duration := time.Since(start)

	// Verify the result
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Check response time requirement: <50ms
	if duration > 50*time.Millisecond {
		t.Errorf("Command execution took %v, expected <50ms", duration)
	}

	t.Logf("Command execution time: %v", duration)
}

// TestCommandResponseTimeWithLoad tests response time under load
func TestCommandResponseTimeWithLoad(t *testing.T) {
	const numCommands = 10
	const maxResponseTime = 50 * time.Millisecond

	commands := make([]*TestCommand, numCommands)
	for i := 0; i < numCommands; i++ {
		commands[i] = &TestCommand{
			CmdID:        fmt.Sprintf("load-test-%d", i),
			CmdType:      CommandTypeIO,
			CmdPriority:  PriorityMedium,
			CmdContext:   DefaultOperationContext(1),
			CmdCreatedAt: time.Now(),
			Operation: func() (interface{}, error) {
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
				return "load result", nil
			},
			shouldSleep: false,
		}
	}

	// Execute all commands and measure total time
	start := time.Now()
	results := make([]tea.Msg, numCommands)
	for i, cmd := range commands {
		results[i] = cmd.Run()
	}
	totalDuration := time.Since(start)

	// Average response time should still be <50ms
	avgDuration := totalDuration / time.Duration(numCommands)
	if avgDuration > maxResponseTime {
		t.Errorf("Average command execution took %v, expected <%v", avgDuration, maxResponseTime)
	}

	// Verify all commands completed successfully
	for i, result := range results {
		if result == nil {
			t.Errorf("Command %d returned nil result", i)
		}
	}

	t.Logf("Average execution time for %d commands: %v", numCommands, avgDuration)
}

// TestSlowCommandResponseTime tests that slow commands are properly handled
func TestSlowCommandResponseTime(t *testing.T) {
	// Create a command that takes longer than acceptable
	cmd := &TestCommand{
		CmdID:        "slow-command-test",
		CmdType:      CommandTypeSystem,
		CmdPriority:  PriorityLow,
		CmdContext:   SystemOperationContext(1),
		CmdCreatedAt: time.Now(),
		Operation: func() (interface{}, error) {
			// Simulate a slow operation (100ms)
			time.Sleep(100 * time.Millisecond)
			return "slow result", nil
		},
		shouldSleep: false,
	}

	// This command should either:
	// 1. Complete quickly via async handling, or
	// 2. Return an error for taking too long
	start := time.Now()
	result := cmd.Run()
	duration := time.Since(start)

	// If the command completes, it should be under 50ms
	// If it takes longer, it should indicate this via error or special result
	if duration > 50*time.Millisecond {
		// Check if the result indicates timeout or async handling
		if msg, ok := result.(TestResultMsg); ok {
			if msg.Status != "timeout" && msg.Status != "async" {
				t.Errorf("Slow command took %v but didn't indicate timeout or async handling, status: %s", duration, msg.Status)
			}
		} else {
			t.Errorf("Slow command took %v without proper timeout/async handling", duration)
		}
	}

	t.Logf("Slow command execution time: %v", duration)
}

// TestConcurrentCommandResponseTime tests response time with concurrent execution
func TestConcurrentCommandResponseTime(t *testing.T) {
	const numCommands = 5
	results := make(chan tea.Msg, numCommands)
	start := time.Now()

	// Start multiple commands concurrently
	for i := 0; i < numCommands; i++ {
		go func(id int) {
			cmd := &TestCommand{
				CmdID:        fmt.Sprintf("concurrent-test-%d", id),
				CmdType:      CommandTypeIO,
				CmdPriority:  PriorityMedium,
				CmdContext:   DefaultOperationContext(1),
				CmdCreatedAt: time.Now(),
				Operation: func() (interface{}, error) {
					time.Sleep(10 * time.Millisecond) // Simulate work
					return fmt.Sprintf("concurrent result %d", id), nil
				},
				shouldSleep: false,
			}
			results <- cmd.Run()
		}(i)
	}

	// Collect results
	completedCommands := 0
	for completedCommands < numCommands {
		select {
		case result := <-results:
			if result == nil {
				t.Errorf("Received nil result from concurrent command")
			}
			completedCommands++
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Timeout waiting for concurrent commands to complete")
		}
	}

	totalDuration := time.Since(start)
	avgDuration := totalDuration / time.Duration(numCommands)

	// Concurrent execution should be faster than sequential
	if avgDuration > 25*time.Millisecond {
		t.Errorf("Concurrent commands took avg %v, expected faster execution", avgDuration)
	}

	t.Logf("Concurrent execution avg time for %d commands: %v", numCommands, avgDuration)
}

// TestCommand is a mock command implementation for testing
type TestCommand struct {
	CmdID        string
	CmdType      CommandType
	CmdPriority  Priority
	CmdContext   OperationContext
	CmdCreatedAt time.Time
	Operation    func() (interface{}, error)
	shouldSleep  bool
}

// TestResultMsg represents a test result message
type TestResultMsg struct {
	ID       string      `json:"id"`
	Result   interface{} `json:"result,omitempty"`
	Error    error       `json:"error,omitempty"`
	Status   string      `json:"status"` // "success", "timeout", "async"
	Duration time.Duration `json:"duration"`
}

// Implement the Command interface
func (c *TestCommand) Run() tea.Msg {
	start := time.Now()

	// Simulate timeout behavior for slow operations
	if c.CmdContext.Timeout > 0 && c.shouldSleep {
		// In real implementation, this would be handled by context with timeout
		time.Sleep(c.CmdContext.Timeout)
	}

	result, err := c.Operation()
	duration := time.Since(start)

	// Determine status based on execution time
	status := "success"
	if duration > 50*time.Millisecond {
		status = "slow"
	}
	if duration > c.CmdContext.Timeout {
		status = "timeout"
	}

	return TestResultMsg{
		ID:       c.CmdID,
		Result:   result,
		Error:    err,
		Status:   status,
		Duration: duration,
	}
}

func (c *TestCommand) ID() string { return c.CmdID }
func (c *TestCommand) Type() CommandType { return c.CmdType }
func (c *TestCommand) Priority() Priority { return c.CmdPriority }
func (c *TestCommand) Context() OperationContext { return c.CmdContext }
func (c *TestCommand) CreatedAt() time.Time { return c.CmdCreatedAt }