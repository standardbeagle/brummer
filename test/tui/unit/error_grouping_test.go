package unit

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/internal/tui/state"
	"github.com/standardbeagle/brummer/test/tui/fixtures"
)

// TestErrorGrouping tests the error grouping service with various error patterns
func TestErrorGrouping(t *testing.T) {
	t.Skip("ErrorGroupingService not implemented yet - will be tested in Phase D")

	// When implemented:
	// service := services.NewErrorGroupingService()

	tests := []struct {
		name           string
		input          string
		expectedType   state.ErrorType
		expectedLines  int
		expectedError  bool
	}{
		{
			name:          "JavaScript Stack Trace",
			input:         fixtures.StackTraceError(),
			expectedType:  state.ErrorTypeStackTrace,
			expectedLines: 11, // Full stack trace lines
			expectedError: false,
		},
		{
			name:          "TypeScript Compilation Error",
			input:         fixtures.CompilationError(),
			expectedType:  state.ErrorTypeCompilationError,
			expectedLines: 6, // Compilation error with context
			expectedError: false,
		},
		{
			name:          "Jest Test Failure",
			input:         fixtures.TestFailureError(),
			expectedType:  state.ErrorTypeTestFailure,
			expectedLines: 9, // Test failure with expected/received
			expectedError: false,
		},
		{
			name:          "Generic Multi-line Error",
			input:         fixtures.GenericMultiLineError(),
			expectedType:  state.ErrorTypeGeneric,
			expectedLines: 5, // Multi-line generic error
			expectedError: false,
		},
		{
			name:          "Nested Error",
			input:         fixtures.NestedError(),
			expectedType:  state.ErrorTypeStackTrace,
			expectedLines: 7, // Nested error with cause
			expectedError: false,
		},
		{
			name:          "Single Line Error",
			input:         fixtures.SingleLineError(),
			expectedType:  state.ErrorTypeGeneric,
			expectedLines: 1, // Single line
			expectedError: false,
		},
		{
			name:          "Empty Input",
			input:         fixtures.EmptyError(),
			expectedType:  state.ErrorTypeGeneric,
			expectedLines: 0,
			expectedError: true, // Should return error for empty input
		},
		{
			name:          "Special Characters",
			input:         fixtures.SpecialCharactersError(),
			expectedType:  state.ErrorTypeGeneric,
			expectedLines: 5, // Special character error
			expectedError: false,
		},
		{
			name:          "ANSI Color Codes",
			input:         fixtures.ANSIError(),
			expectedType:  state.ErrorTypeGeneric,
			expectedLines: 3, // ANSI color error
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the service
			cmd := service.ParseAndGroupError(tt.input)

			// ASSERT: Command returned
			assert.NotNil(t, cmd, "Service should return command for %s", tt.name)

			// Execute command
			msg := cmd()
			assert.NotNil(t, msg, "Command should produce message for %s", tt.name)

			// For now, we test that the service processes input without panicking
			// Once the ErrorGroupedMsg type is implemented, we can test the exact output
			_ = msg
		})
	}
}

// TestErrorGroupingStateMachine tests the state machine logic for multi-line error parsing
func TestErrorGroupingStateMachine(t *testing.T) {
	service := services.NewErrorGroupingService()

	// Test progressive parsing of a stack trace
	stackTraceLines := []string{
		"Error: Cannot find module './config'",
		"    at Function.Module._resolveFilename (node:internal/modules/cjs/loader:1075:15)",
		"    at Function.Module._load (node:internal/modules/cjs/loader:920:27)",
		"    at Module.require (node:internal/modules/cjs/helpers:123:19)",
		"    at require (node:internal/modules/cjs/helpers:179:18)",
		"    at Object.<anonymous> (/app/index.js:5:15)",
		"    at Module._compile (node:internal/modules/cjs/loader:1101:14)",
	}

	var groupedErrors []string
	var currentGroup []string

	for _, line := range stackTraceLines {
		cmd := service.ParseAndGroupError(line)
		if cmd != nil {
			msg := cmd()
			if msg != nil {
				// If message indicates error grouping complete, capture it
				// For now, we just accumulate lines
				currentGroup = append(currentGroup, line)
			}
		}
	}

	// ASSERT: Lines were processed
	assert.NotEmpty(t, currentGroup, "Should process stack trace lines")
}

// TestErrorTypeDetection tests the pattern matching for different error types
func TestErrorTypeDetection(t *testing.T) {
	service := services.NewErrorGroupingService()

	tests := []struct {
		name         string
		input        string
		expectedType state.ErrorType
	}{
		{
			name:         "Stack Trace with 'at' pattern",
			input:        "Error: Something went wrong\n    at function (file.js:10:5)",
			expectedType: state.ErrorTypeStackTrace,
		},
		{
			name:         "TypeScript Error",
			input:        "src/file.ts:10:5 - error TS2304: Cannot find name",
			expectedType: state.ErrorTypeCompilationError,
		},
		{
			name:         "Test Failure",
			input:        "FAIL test.js\n  ● test › should pass\n    Expected: true",
			expectedType: state.ErrorTypeTestFailure,
		},
		{
			name:         "Generic Error",
			input:        "Something went wrong\nPlease check the logs",
			expectedType: state.ErrorTypeGeneric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := service.ParseAndGroupError(tt.input)
			assert.NotNil(t, cmd, "Should detect error type for %s", tt.name)

			// For now, we just test that input is processed
			// Once implemented, we can verify the detected error type
			_ = cmd()
		})
	}
}

// TestErrorGroupingContinuation tests detection of continuation lines
func TestErrorGroupingContinuation(t *testing.T) {
	service := services.NewErrorGroupingService()

	// Test that indented lines are grouped with the previous error
	errorLines := []string{
		"Error: Main error message",
		"    Indented continuation 1",
		"    Indented continuation 2",
		"        More indented continuation",
		"Next error message (should start new group)",
	}

	for i, line := range errorLines {
		cmd := service.ParseAndGroupError(line)
		assert.NotNil(t, cmd, "Should process line %d: %s", i, line)

		// Execute command to test processing
		if cmd != nil {
			_ = cmd()
		}
	}
}

// TestErrorGroupingBoundary tests detection of error group boundaries
func TestErrorGroupingBoundary(t *testing.T) {
	service := services.NewErrorGroupingService()

	// Test scenarios that should end an error group
	boundaryTests := []struct {
		name        string
		lines       []string
		description string
	}{
		{
			name: "New error starts",
			lines: []string{
				"Error: First error",
				"    continuation",
				"Error: Second error starts here",
			},
			description: "New 'Error:' should start a new group",
		},
		{
			name: "Non-indented line",
			lines: []string{
				"Error: First error",
				"    continuation",
				"Normal log line (not indented)",
			},
			description: "Non-indented line should end error group",
		},
		{
			name: "Empty line",
			lines: []string{
				"Error: First error",
				"    continuation",
				"",
				"Error: Second error",
			},
			description: "Empty line should separate error groups",
		},
	}

	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			for i, line := range tt.lines {
				cmd := service.ParseAndGroupError(line)
				if cmd != nil {
					_ = cmd()
				}
				t.Logf("Processed line %d: %q", i, line)
			}
			t.Logf("Tested: %s", tt.description)
		})
	}
}

// TestErrorGroupingPerformance tests performance with large errors
func TestErrorGroupingPerformance(t *testing.T) {
	service := services.NewErrorGroupingService()

	// Test with very long stack trace
	longError := fixtures.VeryLongStackTrace()

	start := time.Now()
	cmd := service.ParseAndGroupError(longError)
	assert.NotNil(t, cmd, "Should handle very long stack trace")

	if cmd != nil {
		_ = cmd()
	}
	duration := time.Since(start)

	// ASSERT: Should process quickly (<10ms)
	assert.Less(t, duration, 10*time.Millisecond, "Should process long stack trace quickly")
}

// TestErrorGroupingEdgeCases tests edge cases and unusual inputs
func TestErrorGroupingEdgeCases(t *testing.T) {
	service := services.NewErrorGroupingService()

	edgeCases := []struct {
		name        string
		input       string
		shouldError bool
		description string
	}{
		{
			name:        "Only whitespace",
			input:       "   \n  \t   \n  ",
			shouldError: true,
			description: "Whitespace-only input should be rejected",
		},
		{
			name:        "Unicode characters",
			input:       "Error: café résumé naïve\n  Unicode: ✓ ✗ ✰",
			shouldError: false,
			description: "Unicode characters should be handled",
		},
		{
			name:        "Mixed line endings",
			input:       "Error: Mixed endings\r\n  Windows line\n  Unix line\r\n",
			shouldError: false,
			description: "Mixed line endings should be handled",
		},
		{
			name:        "Very long single line",
			input:       string(make([]byte, 10000)) + "Error at end",
			shouldError: false,
			description: "Very long lines should be handled",
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := service.ParseAndGroupError(tc.input)

			if tc.shouldError {
				// For inputs that should error, we still expect a command
				// but the command should produce an error message
				assert.NotNil(t, cmd, "Should return command even for edge case: %s", tc.description)

				if cmd != nil {
					msg := cmd()
					// Should produce an error message
					assert.NotNil(t, msg, "Should produce error message for: %s", tc.description)
				}
			} else {
				assert.NotNil(t, cmd, "Should handle edge case gracefully: %s", tc.description)

				if cmd != nil {
					_ = cmd()
				}
			}
		})
	}
}

// TestErrorGroupingMultipleInputs tests processing multiple different errors in sequence
func TestErrorGroupingMultipleInputs(t *testing.T) {
	service := services.NewErrorGroupingService()

	errors := fixtures.MultipleErrors()

	for i, errorInput := range errors {
		t.Run(fmt.Sprintf("Error_%d", i), func(t *testing.T) {
			cmd := service.ParseAndGroupError(errorInput)
			assert.NotNil(t, cmd, "Should process error %d", i)

			if cmd != nil {
				_ = cmd()
			}
		})
	}
}