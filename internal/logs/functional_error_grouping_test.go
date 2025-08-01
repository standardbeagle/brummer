package logs

import (
	"testing"
	"time"
)

// Test helper to create log entries
func createLogEntry(processID, processName, content string, timestamp time.Time, isError bool) LogEntry {
	level := LevelInfo
	if isError {
		level = LevelError
	}

	return LogEntry{
		ID:          processID + "-" + timestamp.Format("20060102150405.000000000"),
		ProcessID:   processID,
		ProcessName: processName,
		Timestamp:   timestamp,
		Content:     content,
		IsError:     isError,
		Level:       level,
	}
}

func TestGroupErrorsByTimeLocality_EmptyInput(t *testing.T) {
	config := DefaultGroupingConfig()
	groups := GroupErrorsByTimeLocality([]LogEntry{}, config)

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestGroupErrorsByTimeLocality_NoErrorEntries(t *testing.T) {
	baseTime := time.Now()
	entries := []LogEntry{
		createLogEntry("process1", "test", "Info message 1", baseTime, false),
		createLogEntry("process1", "test", "Info message 2", baseTime.Add(100*time.Millisecond), false),
	}

	config := DefaultGroupingConfig()
	groups := GroupErrorsByTimeLocality(entries, config)

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups for non-error entries, got %d", len(groups))
	}
}

func TestGroupErrorsByTimeLocality_SingleErrorEntry(t *testing.T) {
	baseTime := time.Now()
	entries := []LogEntry{
		createLogEntry("process1", "test", "Error: Something went wrong", baseTime, true),
	}

	config := DefaultGroupingConfig()
	groups := GroupErrorsByTimeLocality(entries, config)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group for single error, got %d", len(groups))
		return
	}

	group := groups[0]
	if len(group.Entries) != 1 {
		t.Errorf("Expected 1 entry in group, got %d", len(group.Entries))
	}

	if group.ProcessID != "process1" {
		t.Errorf("Expected ProcessID 'process1', got '%s'", group.ProcessID)
	}

	if group.StartTime != baseTime {
		t.Errorf("Expected StartTime %v, got %v", baseTime, group.StartTime)
	}

	if group.EndTime != baseTime {
		t.Errorf("Expected EndTime %v, got %v", baseTime, group.EndTime)
	}
}

func TestGroupErrorsByTimeLocality_ErrorsWithinTimeThreshold(t *testing.T) {
	baseTime := time.Now()
	config := DefaultGroupingConfig()
	config.TimeGapThreshold = 200 * time.Millisecond

	entries := []LogEntry{
		createLogEntry("process1", "test", "Error: First error", baseTime, true),
		createLogEntry("process1", "test", "Error: Second error", baseTime.Add(100*time.Millisecond), true),
		createLogEntry("process1", "test", "Error: Third error", baseTime.Add(150*time.Millisecond), true),
	}

	groups := GroupErrorsByTimeLocality(entries, config)

	if len(groups) != 1 {
		t.Errorf("Expected 1 group for errors within threshold, got %d", len(groups))
		return
	}

	group := groups[0]
	if len(group.Entries) != 3 {
		t.Errorf("Expected 3 entries in group, got %d", len(group.Entries))
	}

	if group.StartTime != baseTime {
		t.Errorf("Expected StartTime %v, got %v", baseTime, group.StartTime)
	}

	expectedEndTime := baseTime.Add(150 * time.Millisecond)
	if group.EndTime != expectedEndTime {
		t.Errorf("Expected EndTime %v, got %v", expectedEndTime, group.EndTime)
	}
}

func TestGroupErrorsByTimeLocality_ErrorsOutsideTimeThreshold(t *testing.T) {
	baseTime := time.Now()
	config := DefaultGroupingConfig()
	config.TimeGapThreshold = 200 * time.Millisecond

	entries := []LogEntry{
		createLogEntry("process1", "test", "Error: First error", baseTime, true),
		createLogEntry("process1", "test", "Error: Second error", baseTime.Add(100*time.Millisecond), true),
		createLogEntry("process1", "test", "Error: Third error", baseTime.Add(500*time.Millisecond), true), // Outside threshold
	}

	groups := GroupErrorsByTimeLocality(entries, config)

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups for errors outside threshold, got %d", len(groups))
		return
	}

	// First group should have 2 entries
	if len(groups[0].Entries) != 2 {
		t.Errorf("Expected 2 entries in first group, got %d", len(groups[0].Entries))
	}

	// Second group should have 1 entry
	if len(groups[1].Entries) != 1 {
		t.Errorf("Expected 1 entry in second group, got %d", len(groups[1].Entries))
	}
}

func TestGroupErrorsByTimeLocality_MultipleProcesses(t *testing.T) {
	baseTime := time.Now()
	config := DefaultGroupingConfig()

	entries := []LogEntry{
		createLogEntry("process1", "test1", "Error: Process 1 error", baseTime, true),
		createLogEntry("process2", "test2", "Error: Process 2 error", baseTime.Add(50*time.Millisecond), true),
		createLogEntry("process1", "test1", "Error: Another process 1 error", baseTime.Add(100*time.Millisecond), true),
	}

	groups := GroupErrorsByTimeLocality(entries, config)

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups for different processes, got %d", len(groups))
		return
	}

	// Groups should be sorted by start time
	// Find which group belongs to which process
	var process1Group, process2Group *ErrorGroup
	for i := range groups {
		if groups[i].ProcessID == "process1" {
			process1Group = &groups[i]
		} else if groups[i].ProcessID == "process2" {
			process2Group = &groups[i]
		}
	}

	if process1Group == nil {
		t.Error("Expected to find process1 group")
		return
	}

	if process2Group == nil {
		t.Error("Expected to find process2 group")
		return
	}

	if len(process1Group.Entries) != 2 {
		t.Errorf("Expected 2 entries in process1 group, got %d", len(process1Group.Entries))
	}

	if len(process2Group.Entries) != 1 {
		t.Errorf("Expected 1 entry in process2 group, got %d", len(process2Group.Entries))
	}
}

func TestGroupErrorsByTimeLocality_MinGroupSize(t *testing.T) {
	baseTime := time.Now()
	config := DefaultGroupingConfig()
	config.MinGroupSize = 2 // Require at least 2 entries

	entries := []LogEntry{
		createLogEntry("process1", "test", "Error: First error", baseTime, true),
		createLogEntry("process1", "test", "Error: Second error", baseTime.Add(100*time.Millisecond), true),
		createLogEntry("process1", "test", "Error: Isolated error", baseTime.Add(1*time.Second), true), // Will be alone
	}

	groups := GroupErrorsByTimeLocality(entries, config)

	// Should only get the first group (2 entries), isolated error should be filtered out
	if len(groups) != 1 {
		t.Errorf("Expected 1 group with MinGroupSize=2, got %d", len(groups))
		return
	}

	if len(groups[0].Entries) != 2 {
		t.Errorf("Expected 2 entries in group, got %d", len(groups[0].Entries))
	}
}

func TestGroupErrorsByTimeLocality_MaxGroupSize(t *testing.T) {
	baseTime := time.Now()
	config := DefaultGroupingConfig()
	config.MaxGroupSize = 2 // Limit to 2 entries per group

	entries := []LogEntry{
		createLogEntry("process1", "test", "Error: First error", baseTime, true),
		createLogEntry("process1", "test", "Error: Second error", baseTime.Add(50*time.Millisecond), true),
		createLogEntry("process1", "test", "Error: Third error", baseTime.Add(100*time.Millisecond), true), // Should start new group
	}

	groups := GroupErrorsByTimeLocality(entries, config)

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups with MaxGroupSize=2, got %d", len(groups))
		return
	}

	if len(groups[0].Entries) != 2 {
		t.Errorf("Expected 2 entries in first group, got %d", len(groups[0].Entries))
	}

	if len(groups[1].Entries) != 1 {
		t.Errorf("Expected 1 entry in second group, got %d", len(groups[1].Entries))
	}
}

func TestGroupErrorsByTimeLocality_MaxGroupDuration(t *testing.T) {
	baseTime := time.Now()
	config := DefaultGroupingConfig()
	config.MaxGroupDuration = 200 * time.Millisecond // Smaller duration to trigger split

	entries := []LogEntry{
		createLogEntry("process1", "test", "Error: First error", baseTime, true),
		createLogEntry("process1", "test", "Error: Second error", baseTime.Add(100*time.Millisecond), true),
		createLogEntry("process1", "test", "Error: Third error", baseTime.Add(150*time.Millisecond), true),
		createLogEntry("process1", "test", "Error: Fourth error", baseTime.Add(250*time.Millisecond), true), // Within time gap but exceeds 200ms duration
	}

	groups := GroupErrorsByTimeLocality(entries, config)

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups with MaxGroupDuration, got %d", len(groups))
		return
	}

	// First group should have 3 entries (up to 150ms, then 250ms would exceed duration)
	if len(groups[0].Entries) != 3 {
		t.Errorf("Expected 3 entries in first group, got %d", len(groups[0].Entries))
	}

	// Second group should have 1 entry
	if len(groups[1].Entries) != 1 {
		t.Errorf("Expected 1 entry in second group, got %d", len(groups[1].Entries))
	}
}

func TestErrorTypeDetection(t *testing.T) {
	tests := []struct {
		content      string
		expectedType string
	}{
		{"TypeError: Cannot read property 'foo' of undefined", "TypeError"},
		{"ReferenceError: myVar is not defined", "ReferenceError"},
		{"SyntaxError: Unexpected token", "SyntaxError"},
		{"Network error: ENOTFOUND", "NetworkError"},
		{"MongoDB connection failed", "MongoError"},
		{"ESLint: Missing semicolon", "LintError"},
		{"Build failed with errors", "CompilationError"},
		{"Unknown error message", "Error"},
	}

	for _, test := range tests {
		result := detectErrorType(test.content)
		if result != test.expectedType {
			t.Errorf("For content '%s', expected type '%s', got '%s'",
				test.content, test.expectedType, result)
		}
	}
}

func TestSeverityDetection(t *testing.T) {
	tests := []struct {
		content          string
		expectedSeverity string
	}{
		{"Critical system failure", "critical"},
		{"Fatal error occurred", "critical"},
		{"Panic: out of memory", "critical"},
		{"Error: something went wrong", "error"},
		{"Build failed", "error"},
		{"Warning: deprecated function", "warning"},
		{"Regular error message", "error"}, // Default
	}

	for _, test := range tests {
		result := determineSeverity(test.content)
		if result != test.expectedSeverity {
			t.Errorf("For content '%s', expected severity '%s', got '%s'",
				test.content, test.expectedSeverity, result)
		}
	}
}

func TestMessageExtraction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[12:34:56] Error: Something went wrong", "Error: Something went wrong"},
		{"(process) TypeError: Cannot read property", "TypeError: Cannot read property"},
		{"Simple error message", "Simple error message"},
		{"", ""},
	}

	for _, test := range tests {
		result := extractMainMessage(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'",
				test.input, test.expected, result)
		}
	}
}

func TestFilterErrorEntries(t *testing.T) {
	baseTime := time.Now()
	entries := []LogEntry{
		createLogEntry("process1", "test", "Info message", baseTime, false),
		createLogEntry("process1", "test", "Error message", baseTime.Add(100*time.Millisecond), true),
		{
			ID:        "test3",
			ProcessID: "process1",
			Timestamp: baseTime.Add(200 * time.Millisecond),
			Content:   "Warning message",
			IsError:   false,
			Level:     LevelWarn, // This should be included as it's >= LevelError
		},
		{
			ID:        "test4",
			ProcessID: "process1",
			Timestamp: baseTime.Add(300 * time.Millisecond),
			Content:   "Error level message",
			IsError:   false,
			Level:     LevelError, // This should be included
		},
	}

	errorEntries := filterErrorEntries(entries)

	// Should have 2 entries: the explicit error and the LevelError entry
	if len(errorEntries) != 2 {
		t.Errorf("Expected 2 error entries, got %d", len(errorEntries))
	}
}

func TestGroupingSortedByTime(t *testing.T) {
	baseTime := time.Now()
	config := DefaultGroupingConfig()
	config.TimeGapThreshold = 200 * time.Millisecond // Smaller gap to create separate groups

	// Create entries in non-chronological order with gaps > 200ms between them
	entries := []LogEntry{
		createLogEntry("process1", "test", "Error: Third", baseTime.Add(600*time.Millisecond), true),
		createLogEntry("process1", "test", "Error: First", baseTime, true),
		createLogEntry("process1", "test", "Error: Second", baseTime.Add(300*time.Millisecond), true),
	}

	groups := GroupErrorsByTimeLocality(entries, config)

	if len(groups) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(groups))
		return
	}

	// Groups should be sorted by start time
	if !groups[0].StartTime.Before(groups[1].StartTime) {
		t.Error("Groups not sorted by start time")
	}

	if !groups[1].StartTime.Before(groups[2].StartTime) {
		t.Error("Groups not sorted by start time")
	}

	// Check content order - groups should be sorted by start time
	if groups[0].Entries[0].Content != "Error: First" {
		t.Errorf("Expected first group to contain 'Error: First', got '%s'", groups[0].Entries[0].Content)
	}

	if groups[1].Entries[0].Content != "Error: Second" {
		t.Errorf("Expected second group to contain 'Error: Second', got '%s'", groups[1].Entries[0].Content)
	}

	if groups[2].Entries[0].Content != "Error: Third" {
		t.Errorf("Expected third group to contain 'Error: Third', got '%s'", groups[2].Entries[0].Content)
	}
}

func TestComplexScenario(t *testing.T) {
	baseTime := time.Now()
	config := DefaultGroupingConfig()
	config.TimeGapThreshold = 200 * time.Millisecond
	config.MinGroupSize = 1
	config.MaxGroupSize = 3

	// Complex scenario: Multiple processes, mixed timing, some grouped, some isolated
	entries := []LogEntry{
		// Process 1: Group of 2
		createLogEntry("process1", "frontend", "TypeError: Cannot read property", baseTime, true),
		createLogEntry("process1", "frontend", "    at Object.render", baseTime.Add(50*time.Millisecond), true),

		// Process 2: Isolated error
		createLogEntry("process2", "backend", "MongoDB connection error", baseTime.Add(100*time.Millisecond), true),

		// Process 1: Another group (after gap)
		createLogEntry("process1", "frontend", "SyntaxError: Unexpected token", baseTime.Add(500*time.Millisecond), true),
		createLogEntry("process1", "frontend", "    at compile", baseTime.Add(550*time.Millisecond), true),
		createLogEntry("process1", "frontend", "    at build", baseTime.Add(600*time.Millisecond), true),

		// Info message (should be filtered out)
		createLogEntry("process1", "frontend", "Info: Build completed", baseTime.Add(650*time.Millisecond), false),

		// Process 1: Isolated error after gap
		createLogEntry("process1", "frontend", "Critical: Memory exhausted", baseTime.Add(1*time.Second), true),
	}

	groups := GroupErrorsByTimeLocality(entries, config)

	// Expected: 4 groups total
	// - Process1: group of 2 (TypeError + stack)
	// - Process2: group of 1 (MongoDB error)
	// - Process1: group of 3 (SyntaxError + 2 stack lines)
	// - Process1: group of 1 (Critical error)
	if len(groups) != 4 {
		t.Errorf("Expected 4 groups in complex scenario, got %d", len(groups))
		return
	}

	// Verify group contents
	expectedSizes := []int{2, 1, 3, 1}
	for i, expectedSize := range expectedSizes {
		if len(groups[i].Entries) != expectedSize {
			t.Errorf("Group %d: expected %d entries, got %d", i, expectedSize, len(groups[i].Entries))
		}
	}

	// Verify error types are detected
	expectedTypes := []string{"TypeError", "MongoError", "SyntaxError", "Error"}
	for i, expectedType := range expectedTypes {
		if groups[i].ErrorType != expectedType {
			t.Errorf("Group %d: expected error type '%s', got '%s'", i, expectedType, groups[i].ErrorType)
		}
	}
}
