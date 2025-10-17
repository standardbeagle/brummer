package handlers

import (
	"testing"

	"github.com/standardbeagle/brummer/internal/tui/commands"
	"github.com/standardbeagle/brummer/internal/tui/state"
	tea "github.com/charmbracelet/bubbletea"
)

// TestCommandHandlerCreation tests creating a new command handler
func TestCommandHandlerCreation(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	if handler == nil {
		t.Fatal("Expected non-nil command handler")
	}

	if handler.model != model {
		t.Error("Expected handler to use provided model")
	}
}

// TestHandleIOCommand tests handling IO commands
func TestHandleIOCommand(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create an IO command
	cmd := commands.NewIOCommand("test-io", commands.IOReadOperation, "/test/path", nil)
	if cmd == nil {
		t.Fatal("Expected non-nil IO command")
	}

	// Execute the command
	result := cmd.Run()
	if result == nil {
		t.Error("Expected non-nil result from command execution")
	}

	// Handle the result through the handler
	msg := handler.Handle(result)
	if msg == nil {
		t.Error("Expected non-nil message from handler")
	}
}

// TestHandleProcessCommand tests handling process commands
func TestHandleProcessCommand(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create a process command
	cmd := commands.NewProcessCommand("test-process", commands.ProcessStartOperation, nil, []string{"echo", "test"})
	if cmd == nil {
		t.Fatal("Expected non-nil process command")
	}

	// Execute the command
	result := cmd.Run()
	if result == nil {
		t.Error("Expected non-nil result from command execution")
	}

	// Handle the result through the handler
	msg := handler.Handle(result)
	if msg == nil {
		t.Error("Expected non-nil message from handler")
	}
}

// TestHandleNetworkCommand tests handling network commands
func TestHandleNetworkCommand(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create a network command
	cmd := commands.NewNetworkCommand("test-network", commands.NetworkGetOperation, "http://example.com", nil, nil)
	if cmd == nil {
		t.Fatal("Expected non-nil network command")
	}

	// Execute the command
	result := cmd.Run()
	if result == nil {
		t.Error("Expected non-nil result from command execution")
	}

	// Handle the result through the handler
	msg := handler.Handle(result)
	if msg == nil {
		t.Error("Expected non-nil message from handler")
	}
}

// TestHandleSystemCommand tests handling system commands
func TestHandleSystemCommand(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create a system command
	cmd := commands.NewSystemCommand("test-system", commands.SystemInfoOperation, nil)
	if cmd == nil {
		t.Fatal("Expected non-nil system command")
	}

	// Execute the command
	result := cmd.Run()
	if result == nil {
		t.Error("Expected non-nil result from command execution")
	}

	// Handle the result through the handler
	msg := handler.Handle(result)
	if msg == nil {
		t.Error("Expected non-nil message from handler")
	}
}

// TestCommandTimeout tests command timeout handling
func TestCommandTimeout(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create a command with very short timeout
	context := commands.SystemOperationContext(0)
	context.Timeout = 1 * time.Millisecond

	cmd := commands.NewIOCommand("test-timeout", commands.IOReadOperation, "/nonexistent/path", context)
	if cmd == nil {
		t.Fatal("Expected non-nil IO command")
	}

	// Execute the command (should timeout quickly)
	result := cmd.Run()
	if result == nil {
		t.Error("Expected non-nil result from command execution")
	}

	// Handle the result through the handler
	msg := handler.Handle(result)
	if msg == nil {
		t.Error("Expected non-nil message from handler")
	}
}

// TestCommandCancellation tests command cancellation
func TestCommandCancellation(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create a cancellable command
	cmd := commands.NewIOCommand("test-cancel", commands.IOReadOperation, "/test/path", nil)
	if cmd == nil {
		t.Fatal("Expected non-nil IO command")
	}

	// Cancel the command before execution
	cancelled := handler.CancelCommand(cmd.ID())
	if !cancelled {
		t.Error("Expected command to be cancelled")
	}

	// Execute the command (should handle cancellation)
	result := cmd.Run()
	if result == nil {
		t.Error("Expected non-nil result from command execution")
	}
}

// TestBatchCommandHandling tests handling batch commands
func TestBatchCommandHandling(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create multiple commands
	cmds := []commands.Command{
		commands.NewIOCommand("batch-1", commands.IOReadOperation, "/test1", nil),
		commands.NewIOCommand("batch-2", commands.IOReadOperation, "/test2", nil),
		commands.NewIOCommand("batch-3", commands.IOReadOperation, "/test3", nil),
	}

	// Create batch command
	batchCmd := commands.NewBatchCommand("test-batch", cmds, commands.BatchSequential, nil)
	if batchCmd == nil {
		t.Fatal("Expected non-nil batch command")
	}

	// Execute the batch command
	result := batchCmd.Run()
	if result == nil {
		t.Error("Expected non-nil result from batch command execution")
	}

	// Handle the result through the handler
	msg := handler.Handle(result)
	if msg == nil {
		t.Error("Expected non-nil message from handler")
	}
}

// TestCommandRetry tests command retry logic
func TestCommandRetry(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create a command that will fail
	cmd := commands.NewIOCommand("test-retry", commands.IOReadOperation, "/nonexistent/path", nil)
	if cmd == nil {
		t.Fatal("Expected non-nil IO command")
	}

	// Configure retry
	retryCmd := commands.NewRetryCommand(cmd, 3, 100*time.Millisecond)
	if retryCmd == nil {
		t.Fatal("Expected non-nil retry command")
	}

	// Execute the retry command
	result := retryCmd.Run()
	if result == nil {
		t.Error("Expected non-nil result from retry command execution")
	}

	// Handle the result through the handler
	msg := handler.Handle(result)
	if msg == nil {
		t.Error("Expected non-nil message from handler")
	}
}

// TestCommandPrioritization tests command priority handling
func TestCommandPrioritization(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create commands with different priorities
	lowCmd := commands.NewIOCommand("low-priority", commands.IOReadOperation, "/test1", nil)
	lowCmd.SetPriority(commands.PriorityLow)

	highCmd := commands.NewIOCommand("high-priority", commands.IOReadOperation, "/test2", nil)
	highCmd.SetPriority(commands.PriorityHigh)

	// Queue commands
	handler.QueueCommand(lowCmd)
	handler.QueueCommand(highCmd)

	// Get next command (should be high priority)
	nextCmd := handler.GetNextCommand()
	if nextCmd == nil {
		t.Error("Expected non-nil next command")
	}

	if nextCmd.ID() != "high-priority" {
		t.Errorf("Expected high priority command, got %s", nextCmd.ID())
	}
}

// TestCommandQueue tests command queue management
func TestCommandQueue(t *testing.T) {
	model := state.NewApplicationState()
	handler := NewCommandHandler(model)

	// Create test commands
	cmds := []commands.Command{
		commands.NewIOCommand("queue-1", commands.IOReadOperation, "/test1", nil),
		commands.NewIOCommand("queue-2", commands.IOReadOperation, "/test2", nil),
		commands.NewIOCommand("queue-3", commands.IOReadOperation, "/test3", nil),
	}

	// Queue all commands
	for _, cmd := range cmds {
		handler.QueueCommand(cmd)
	}

	// Check queue size
	if handler.GetQueueSize() != 3 {
		t.Errorf("Expected queue size 3, got %d", handler.GetQueueSize())
	}

	// Process commands
	for i := 0; i < 3; i++ {
		cmd := handler.GetNextCommand()
		if cmd == nil {
			t.Error("Expected non-nil command from queue")
		}

		// Execute command
		result := cmd.Run()
		handler.Handle(result)
	}

	// Queue should be empty
	if handler.GetQueueSize() != 0 {
		t.Errorf("Expected empty queue, got size %d", handler.GetQueueSize())
	}
}