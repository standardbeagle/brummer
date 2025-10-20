package state

// ErrorParserState maintains state for stream-based error grouping parser
//
// IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
// NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
type ErrorParserState struct {
	CurrentGroup   []string  // Lines accumulated for current error group
	InErrorGroup   bool      // Whether currently inside an error group
	LastErrorType  ErrorType // Type of error being parsed
	LinesSinceError int      // Counter for detecting group end
}

// NewErrorParserState creates a new error parser state
func NewErrorParserState() ErrorParserState {
	return ErrorParserState{
		CurrentGroup:   []string{},
		InErrorGroup:   false,
		LastErrorType:  ErrorTypeGeneric,
		LinesSinceError: 0,
	}
}

// Reset resets the parser state to start a new error group
func (eps ErrorParserState) Reset() ErrorParserState {
	return NewErrorParserState()
}

// StartErrorGroup starts a new error group with the given type
func (eps ErrorParserState) StartErrorGroup(line string, errorType ErrorType) ErrorParserState {
	newState := ErrorParserState{
		CurrentGroup:   []string{line},
		InErrorGroup:   true,
		LastErrorType:  errorType,
		LinesSinceError: 0,
	}
	return newState
}

// AddLine adds a line to the current error group
func (eps ErrorParserState) AddLine(line string) ErrorParserState {
	newState := eps.copy()
	newState.CurrentGroup = append(newState.CurrentGroup, line)
	newState.LinesSinceError++
	return newState
}

// IsInErrorGroup returns whether the parser is currently in an error group
func (eps ErrorParserState) IsInErrorGroup() bool {
	return eps.InErrorGroup
}

// GetCurrentGroup returns the current error group lines
func (eps ErrorParserState) GetCurrentGroup() []string {
	group := make([]string, len(eps.CurrentGroup))
	copy(group, eps.CurrentGroup)
	return group
}

// GetErrorType returns the current error type being parsed
func (eps ErrorParserState) GetErrorType() ErrorType {
	return eps.LastErrorType
}

// GetLinesSinceError returns the count of lines since the error started
func (eps ErrorParserState) GetLinesSinceError() int {
	return eps.LinesSinceError
}

// ShouldEndGroup determines if the current error group should end based on heuristics
func (eps ErrorParserState) ShouldEndGroup(nextLine string) bool {
	if !eps.InErrorGroup {
		return false
	}

	// End group if we've seen too many lines without error indicators
	if eps.LinesSinceError > 20 {
		return true
	}

	// End group if next line starts a new error
	if isErrorStart(nextLine) {
		return true
	}

	// End group if next line is empty and we have some content
	if nextLine == "" && len(eps.CurrentGroup) > 0 {
		return true
	}

	// End group if next line looks like normal log output
	if isNormalLogLine(nextLine) {
		return true
	}

	return false
}

// FinalizeGroup finalizes the current error group and returns the lines
func (eps ErrorParserState) FinalizeGroup() ([]string, ErrorParserState) {
	group := eps.GetCurrentGroup()
	newState := eps.Reset()
	return group, newState
}

// Helper methods
func (eps ErrorParserState) copy() ErrorParserState {
	newGroup := make([]string, len(eps.CurrentGroup))
	copy(newGroup, eps.CurrentGroup)

	return ErrorParserState{
		CurrentGroup:   newGroup,
		InErrorGroup:   eps.InErrorGroup,
		LastErrorType:  eps.LastErrorType,
		LinesSinceError: eps.LinesSinceError,
	}
}

// isErrorStart determines if a line looks like the start of an error
func isErrorStart(line string) bool {
	trimmed := line
	if len(trimmed) > 0 && trimmed[0] != ' ' && trimmed[0] != '\t' {
		// Check for common error patterns
		errorPatterns := []string{
			"Error:", "error:", "ERROR:",
			"Exception:", "exception:",
			"FAIL:", "fail:", "Fail:",
			"TypeError:", "ReferenceError:", "SyntaxError:",
			"panic:", "runtime error:",
			"Traceback", "Traceback (most recent call last):",
		}

		for _, pattern := range errorPatterns {
			if contains(trimmed, pattern) {
				return true
			}
		}
	}
	return false
}

// isNormalLogLine determines if a line looks like normal log output
func isNormalLogLine(line string) bool {
	trimmed := line
	if len(trimmed) == 0 {
		return false
	}

	// If it doesn't start with whitespace and doesn't look like an error, it's probably normal log output
	if trimmed[0] != ' ' && trimmed[0] != '\t' && !isErrorStart(trimmed) {
		return true
	}

	// Check for timestamp patterns (common in logs)
	timestampPatterns := []string{
		"[", // [2025-01-01 12:00:00]
		"2025-", // ISO format
		"12:", // Time format
	}

	for _, pattern := range timestampPatterns {
		if contains(trimmed, pattern) && !isErrorStart(trimmed) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}