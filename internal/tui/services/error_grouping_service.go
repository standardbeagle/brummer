package services

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// ErrorGroupingService is a stateless service for error grouping
// It processes log lines and groups multi-line errors intelligently
//
// IMMUTABLE: This service is stateless. It contains only configuration.
// NO MUTEXES: Thread-safety is achieved through statelessness and immutability.
type ErrorGroupingService struct {
	// Configuration only (no mutable state)
	errorPatterns map[state.ErrorType][]string
}

// NewErrorGroupingService creates a new error grouping service
func NewErrorGroupingService() *ErrorGroupingService {
	return &ErrorGroupingService{
		errorPatterns: map[state.ErrorType][]string{
			state.ErrorTypeStackTrace: {
				"Error:", "error:", "TypeError:", "ReferenceError:", "SyntaxError:",
				"RangeError:", "EvalError:", "URIError:", "InternalError:",
				"at ", "    at ", "node:", "npm ERR!",
			},
			state.ErrorTypeCompilationError: {
				"TS", "error TS", "Compilation error", "Build error",
				"Failed to compile", "Cannot find module", "Module not found",
				"Unexpected token", "Unexpected identifier",
			},
			state.ErrorTypeTestFailure: {
				"FAIL", "fail:", "Test suite failed", "Test Suites:", "●",
				"Expected:", "Received:", "AssertionError:", "jest",
				"vitest", "mocha", "jasmine",
			},
		},
	}
}

// ParseAndGroupError processes a log line and returns a command
func (s *ErrorGroupingService) ParseAndGroupError(line string, parserState *state.ErrorParserState) tea.Cmd {
	// Handle empty input
	if strings.TrimSpace(line) == "" {
		return func() tea.Msg {
			return messages.ErrorMsg{
				Error:   fmt.Errorf("empty input line"),
				Context: "ErrorGroupingService.ParseAndGroupError",
			}
		}
	}

	// Check if we should end the current group
	if parserState.IsInErrorGroup() && parserState.ShouldEndGroup(line) {
		group, newState := parserState.FinalizeGroup()

		if len(group) > 0 {
			// Create grouped error from completed group
			groupedError := s.createGroupedError(group)

			return func() tea.Msg {
				return state.ErrorGroupedMsg{
					Error:     groupedError,
					NewState: newState,
				}
			}
		}
	}

	// Check if this line starts a new error group
	errorType := s.detectErrorType(line)
	if errorType != state.ErrorTypeGeneric || s.isErrorStart(line) {
		if parserState.IsInErrorGroup() {
			// Finalize current group first
			oldGroup, _ := parserState.FinalizeGroup()
			if len(oldGroup) > 0 {
				oldError := s.createGroupedError(oldGroup)

				return func() tea.Msg {
					// Start new group after finalizing old one
					newParserState := state.NewErrorParserState().StartErrorGroup(line, errorType)
					newError := s.createGroupedError([]string{line})

					return state.ErrorGroupedBatchMsg{
						CompletedError: oldError,
						NewError:      newError,
						NewState:     newParserState,
					}
				}
			}
		}

		// Start new error group
		return func() tea.Msg {
			newParserState := parserState.StartErrorGroup(line, errorType)
			return state.ErrorParserStateChangedMsg{
				NewState: newParserState,
			}
		}
	}

	// Add line to current group if we're in one
	if parserState.IsInErrorGroup() {
		return func() tea.Msg {
			newParserState := parserState.AddLine(line)
			return state.ErrorParserStateChangedMsg{
				NewState: newParserState,
			}
		}
	}

	// Line doesn't appear to be part of an error
	return nil
}

// ProcessMultipleErrors processes multiple error lines at once
func (s *ErrorGroupingService) ProcessMultipleErrors(lines []string) tea.Cmd {
	if len(lines) == 0 {
		return nil
	}

	// Create a batch command
	commands := make([]tea.Cmd, 0, len(lines))

	for _, line := range lines {
		// For batch processing, we'll create simple error groups
		if s.isErrorStart(line) {
			groupedError := s.createGroupedError([]string{line})

			commands = append(commands, func() tea.Msg {
				return state.ErrorGroupedMsg{
					Error: groupedError,
				}
			})
		}
	}

	if len(commands) == 0 {
		return nil
	}

	// Return batched command
	return tea.Sequence(commands...)
}

// GetConfig returns the service configuration
func (s *ErrorGroupingService) GetConfig() map[state.ErrorType][]string {
	config := make(map[state.ErrorType][]string)
	for k, v := range s.errorPatterns {
		config[k] = make([]string, len(v))
		copy(config[k], v)
	}
	return config
}

// detectErrorType determines the type of error from a line
func (s *ErrorGroupingService) detectErrorType(line string) state.ErrorType {
	lineUpper := strings.ToUpper(line)

	// Check against patterns for each error type
	for errorType, patterns := range s.errorPatterns {
		for _, pattern := range patterns {
			if strings.Contains(lineUpper, strings.ToUpper(pattern)) {
				return errorType
			}
		}
	}

	return state.ErrorTypeGeneric
}

// isErrorStart determines if a line looks like the start of an error
func (s *ErrorGroupingService) isErrorStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) == 0 {
		return false
	}

	// Lines that don't start with whitespace are more likely to be error starts
	if trimmed[0] != ' ' && trimmed[0] != '\t' {
		// Check for common error indicators
		errorIndicators := []string{
			"ERROR:", "Error:", "error:", "ERR:", "err:",
			"EXCEPTION:", "Exception:", "exception:",
			"FAIL:", "fail:", "Fail:",
			"PANIC:", "panic:", "runtime error:",
			"TRACEBACK:", "Traceback", "Traceback (most recent call last):",
			"TYPEERROR:", "REFERENCEERROR:", "SYNTAXERROR:",
			"RANGEERROR:", "EVALERROR:", "URIERROR:",
			"npm ERR!", "Cannot find", "Module not found",
			"Failed to compile", "Build error",
		}

		for _, indicator := range errorIndicators {
			if strings.Contains(trimmed, indicator) {
				return true
			}
		}
	}

	return false
}

// createGroupedError creates a GroupedError from a group of lines
func (s *ErrorGroupingService) createGroupedError(lines []string) state.GroupedError {
	if len(lines) == 0 {
		return state.GroupedError{}
	}

	// Parse lines to extract information
	firstLine := strings.TrimSpace(lines[0])
	errorType := s.detectErrorType(firstLine)

	// Try to extract process name and PID from first line
	processName := "unknown"
	processPID := 0

	// Look for process patterns in the first line
	if strings.Contains(firstLine, "[") && strings.Contains(firstLine, "]") {
		// Pattern: [process-name] message
		parts := strings.SplitN(firstLine, "]", 2)
		if len(parts) >= 1 {
			processPart := strings.Trim(parts[0], "[]")
			processName = strings.TrimSpace(processPart)
		}
	}

	// Create ID from first line and timestamp
	id := fmt.Sprintf("error-%d", time.Now().UnixNano())

	// Extract source and line number if possible
	source := ""
	lineNumber := 0

	for _, line := range lines {
		if strings.Contains(line, ":") {
			// Look for file:line patterns
			if strings.Contains(line, ".js:") || strings.Contains(line, ".ts:") ||
			   strings.Contains(line, ".go:") || strings.Contains(line, ".py:") {
				// Extract file and line number
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					source = parts[0]
					if len(parts) > 1 {
						if _, err := fmt.Sscanf(parts[1], "%d", &lineNumber); err == nil {
							// Successfully parsed line number
						}
					}
				}
				break
			}
		}
	}

	return state.GroupedError{
		ID:          id,
		ProcessName: processName,
		ProcessPID:  processPID,
		FirstLine:   firstLine,
		FullContent: lines,
		ErrorType:   errorType,
		Timestamp:   time.Now(),
		Source:      source,
		LineNumber:  lineNumber,
		Version:     0,
	}
}