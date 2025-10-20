package state

import (
	"time"
)

// GroupedError represents a multi-line error entry that has been intelligently grouped from log streams
//
// IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
// NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
type GroupedError struct {
	ID           string    `json:"id"`            // Unique identifier for the error (timestamp + hash)
	ProcessName  string    `json:"process_name"`  // Name of the process that emitted the error
	ProcessPID   int       `json:"process_pid"`   // Process ID of the source process
	FirstLine    string    `json:"first_line"`    // The initial error line (e.g., "Error: Cannot find module")
	FullContent  []string  `json:"full_content"`  // All lines belonging to this error group
	ErrorType    ErrorType `json:"error_type"`    // Classification of error (StackTrace, CompilationError, TestFailure, Generic)
	Timestamp    time.Time `json:"timestamp"`     // When the first line of the error was captured
	Source       string    `json:"source"`        // File path if extracted from error content
	LineNumber   int       `json:"line_number"`   // Line number if extracted from error content
	Version      uint64    `json:"version"`       // State version for tracking changes
}

// ErrorType represents different types of errors that can be grouped
type ErrorType int

const (
	ErrorTypeGeneric ErrorType = iota
	ErrorTypeStackTrace
	ErrorTypeCompilationError
	ErrorTypeTestFailure
)

// String returns the string representation of ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeStackTrace:
		return "StackTrace"
	case ErrorTypeCompilationError:
		return "CompilationError"
	case ErrorTypeTestFailure:
		return "TestFailure"
	case ErrorTypeGeneric:
		return "Generic"
	default:
		return "Unknown"
	}
}

// Validate checks if the GroupedError has valid data
func (ge *GroupedError) Validate() []string {
	var issues []string

	if ge.ID == "" {
		issues = append(issues, "ID cannot be empty")
	}

	if ge.ProcessName == "" {
		issues = append(issues, "ProcessName cannot be empty")
	}

	if len(ge.FullContent) == 0 {
		issues = append(issues, "FullContent must have at least one line")
	}

	if ge.FirstLine != ge.FullContent[0] {
		issues = append(issues, "FirstLine must match the first element of FullContent")
	}

	if ge.Timestamp.IsZero() {
		issues = append(issues, "Timestamp cannot be zero time")
	}

	return issues
}

// Copy creates a deep copy of the GroupedError
func (ge *GroupedError) Copy() *GroupedError {
	if ge == nil {
		return nil
	}

	newContent := make([]string, len(ge.FullContent))
	copy(newContent, ge.FullContent)

	return &GroupedError{
		ID:          ge.ID,
		ProcessName: ge.ProcessName,
		ProcessPID:  ge.ProcessPID,
		FirstLine:   ge.FirstLine,
		FullContent: newContent,
		ErrorType:   ge.ErrorType,
		Timestamp:   ge.Timestamp,
		Source:      ge.Source,
		LineNumber:  ge.LineNumber,
		Version:     ge.Version,
	}
}

// WithVersion returns a new GroupedError with incremented version
func (ge *GroupedError) WithVersion() *GroupedError {
	if ge == nil {
		return nil
	}

	newError := ge.Copy()
	newError.Version++
	return newError
}

// Equals checks if two GroupedErrors are equal (excluding version)
func (ge *GroupedError) Equals(other *GroupedError) bool {
	if ge == nil && other == nil {
		return true
	}
	if ge == nil || other == nil {
		return false
	}

	if ge.ID != other.ID ||
		ge.ProcessName != other.ProcessName ||
		ge.ProcessPID != other.ProcessPID ||
		ge.FirstLine != other.FirstLine ||
		ge.ErrorType != other.ErrorType ||
		!ge.Timestamp.Equal(other.Timestamp) ||
		ge.Source != other.Source ||
		ge.LineNumber != other.LineNumber {
		return false
	}

	if len(ge.FullContent) != len(other.FullContent) {
		return false
	}

	for i := range ge.FullContent {
		if ge.FullContent[i] != other.FullContent[i] {
			return false
		}
	}

	return true
}