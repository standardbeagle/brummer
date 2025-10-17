package commands

import (
	"testing"
	"time"
)

// TestIOCommand tests the IO command implementation
// This test should FAIL initially before IOCommand is implemented
func TestIOCommand(t *testing.T) {
	// Create an IO command instance using the factory function
	context := DefaultOperationContext(1)
	cmd := NewIOCommand(
		"test-io-1",
		"test.txt",
		IOReadFile,
		func() (interface{}, error) {
			return "test result", nil
		},
		context,
	)

	// Test basic properties
	if cmd.ID() != "test-io-1" {
		t.Errorf("Expected ID 'test-io-1', got '%s'", cmd.ID())
	}

	if cmd.Type() != CommandTypeIO {
		t.Errorf("Expected type %d, got %d", CommandTypeIO, cmd.Type())
	}

	if cmd.Priority() != PriorityMedium {
		t.Errorf("Expected priority %d, got %d", PriorityMedium, cmd.Priority())
	}

	// Test command execution
	result := cmd.Run()
	msg, ok := result.(IOResultMsg)
	if !ok {
		t.Fatalf("Expected IOResultMsg, got %T", result)
	}

	if msg.ID != "test-io-1" {
		t.Errorf("Expected message ID 'test-io-1', got '%s'", msg.ID)
	}

	if msg.Error != nil {
		t.Errorf("Expected no error, got %v", msg.Error)
	}

	if msg.Result != "test result" {
		t.Errorf("Expected result 'test result', got '%v'", msg.Result)
	}
}

// TestIOCommandError tests error handling in IO commands
func TestIOCommandError(t *testing.T) {
	expectedError := "test error"
	context := DefaultOperationContext(1)
	cmd := NewIOCommand(
		"test-io-error",
		"error.txt",
		IOWriteFile,
		func() (interface{}, error) {
			return nil, &TestError{msg: expectedError}
		},
		context,
	)

	result := cmd.Run()
	msg, ok := result.(IOResultMsg)
	if !ok {
		t.Fatalf("Expected IOResultMsg, got %T", result)
	}

	if msg.Error == nil {
		t.Fatal("Expected error, got nil")
	}

	if msg.Error.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%v'", expectedError, msg.Error)
	}
}

// TestIOCommandTimeout tests timeout behavior
func TestIOCommandTimeout(t *testing.T) {
	context := OperationContext{
		Timeout: 1 * time.Millisecond, // Very short timeout
	}
	cmd := NewIOCommand(
		"test-io-timeout",
		"timeout.txt",
		IOReadFile,
		func() (interface{}, error) {
			time.Sleep(100 * time.Millisecond) // Sleep longer than timeout
			return "should not complete", nil
		},
		context,
	)

	result := cmd.Run()
	msg, ok := result.(IOResultMsg)
	if !ok {
		t.Fatalf("Expected IOResultMsg, got %T", result)
	}

	if msg.Error == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// Check if it's a timeout error (implementation specific)
	if !contains(msg.Error.Error(), "timeout") {
		t.Errorf("Expected timeout error, got '%v'", msg.Error)
	}
}

// TestError is a test error implementation
type TestError struct {
	msg string
}

func (e *TestError) Error() string {
	return e.msg
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 &&
		(len(substr) == 0 || (len(s) > 0 && (s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr || indexOfSubstring(s, substr) >= 0))))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

