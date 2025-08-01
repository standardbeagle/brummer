package logs

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ErrorGroup represents a group of related error log entries
type ErrorGroup struct {
	ID          string
	ProcessID   string
	ProcessName string
	StartTime   time.Time
	EndTime     time.Time
	Entries     []LogEntry
	ErrorType   string
	Message     string
	Severity    string
}

// GroupingConfig contains parameters for error grouping
type GroupingConfig struct {
	TimeGapThreshold time.Duration // Max time gap between entries in same group
	MinGroupSize     int           // Minimum entries to form a group
	MaxGroupSize     int           // Maximum entries per group
	MaxGroupDuration time.Duration // Maximum time span for a single group
}

// DefaultGroupingConfig returns sensible defaults for error grouping
func DefaultGroupingConfig() GroupingConfig {
	return GroupingConfig{
		TimeGapThreshold: 200 * time.Millisecond,
		MinGroupSize:     1,
		MaxGroupSize:     50,
		MaxGroupDuration: 5 * time.Second,
	}
}

// GroupErrorsByTimeLocality groups error log entries based on temporal proximity
// This is a pure functional approach that takes entries and returns grouped errors
func GroupErrorsByTimeLocality(entries []LogEntry, config GroupingConfig) []ErrorGroup {
	// Filter and sort error entries
	errorEntries := filterErrorEntries(entries)
	sortEntriesByTime(errorEntries)

	// Group entries by process first
	processGroups := groupEntriesByProcess(errorEntries)

	// Apply time-based grouping within each process
	var allGroups []ErrorGroup
	for _, processEntries := range processGroups {
		timeGroups := groupEntriesByTimeGaps(processEntries, config)
		allGroups = append(allGroups, timeGroups...)
	}

	// Sort final groups by start time
	sortGroupsByTime(allGroups)

	return allGroups
}

// filterErrorEntries returns only entries that should be considered for error grouping
func filterErrorEntries(entries []LogEntry) []LogEntry {
	var errorEntries []LogEntry
	for _, entry := range entries {
		if isErrorEntry(entry) {
			errorEntries = append(errorEntries, entry)
		}
	}
	return errorEntries
}

// isErrorEntry determines if a log entry should be considered an error
func isErrorEntry(entry LogEntry) bool {
	return entry.IsError || entry.Level >= LevelError
}

// sortEntriesByTime sorts entries by timestamp (oldest first)
func sortEntriesByTime(entries []LogEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
}

// groupEntriesByProcess groups entries by their ProcessID
func groupEntriesByProcess(entries []LogEntry) map[string][]LogEntry {
	processGroups := make(map[string][]LogEntry)

	for _, entry := range entries {
		processGroups[entry.ProcessID] = append(processGroups[entry.ProcessID], entry)
	}

	return processGroups
}

// groupEntriesByTimeGaps groups entries from the same process based on time gaps
func groupEntriesByTimeGaps(entries []LogEntry, config GroupingConfig) []ErrorGroup {
	if len(entries) == 0 {
		return nil
	}

	var groups []ErrorGroup
	currentGroup := []LogEntry{entries[0]}

	for i := 1; i < len(entries); i++ {
		entry := entries[i]
		lastEntry := currentGroup[len(currentGroup)-1]

		timeSinceLastEntry := entry.Timestamp.Sub(lastEntry.Timestamp)
		groupDuration := entry.Timestamp.Sub(currentGroup[0].Timestamp)

		shouldStartNewGroup := timeSinceLastEntry > config.TimeGapThreshold ||
			len(currentGroup) >= config.MaxGroupSize ||
			groupDuration > config.MaxGroupDuration

		if shouldStartNewGroup {
			// Finalize current group if it meets minimum size
			if len(currentGroup) >= config.MinGroupSize {
				group := createErrorGroup(currentGroup)
				groups = append(groups, group)
			}

			// Start new group
			currentGroup = []LogEntry{entry}
		} else {
			// Add to current group
			currentGroup = append(currentGroup, entry)
		}
	}

	// Don't forget the last group
	if len(currentGroup) >= config.MinGroupSize {
		group := createErrorGroup(currentGroup)
		groups = append(groups, group)
	}

	return groups
}

// createErrorGroup creates an ErrorGroup from a slice of log entries
func createErrorGroup(entries []LogEntry) ErrorGroup {
	if len(entries) == 0 {
		return ErrorGroup{}
	}

	firstEntry := entries[0]
	lastEntry := entries[len(entries)-1]

	group := ErrorGroup{
		ID:          generateGroupID(firstEntry),
		ProcessID:   firstEntry.ProcessID,
		ProcessName: firstEntry.ProcessName,
		StartTime:   firstEntry.Timestamp,
		EndTime:     lastEntry.Timestamp,
		Entries:     entries,
	}

	// Analyze the group content
	analyzeErrorGroup(&group)

	return group
}

// generateGroupID creates a unique ID for an error group
func generateGroupID(firstEntry LogEntry) string {
	return fmt.Sprintf("%s-group-%d", firstEntry.ProcessID, firstEntry.Timestamp.UnixNano())
}

// analyzeErrorGroup analyzes the content of an error group to extract metadata
func analyzeErrorGroup(group *ErrorGroup) {
	if len(group.Entries) == 0 {
		return
	}

	// Combine all content for analysis
	var allContent []string
	for _, entry := range group.Entries {
		allContent = append(allContent, entry.Content)
	}

	combinedContent := strings.Join(allContent, "\n")

	group.ErrorType = detectErrorType(combinedContent)
	group.Message = extractMainMessage(group.Entries[0].Content)
	group.Severity = determineSeverity(combinedContent)
}

// detectErrorType identifies the type of error from content
func detectErrorType(content string) string {
	content = strings.ToLower(content)

	// Check error types in order of specificity (most specific first)
	// This ensures that specific errors aren't misclassified by generic keywords

	// Check for very specific error types first
	if strings.Contains(content, "mongoerror") || strings.Contains(content, "mongodb") {
		return "MongoError"
	}

	if strings.Contains(content, "typeerror") || strings.Contains(content, "cannot read property") || strings.Contains(content, "is not a function") {
		return "TypeError"
	}

	if strings.Contains(content, "referenceerror") || strings.Contains(content, "is not defined") {
		return "ReferenceError"
	}

	if strings.Contains(content, "syntaxerror") || strings.Contains(content, "unexpected token") || strings.Contains(content, "unexpected end") {
		return "SyntaxError"
	}

	if strings.Contains(content, "compilation failed") || strings.Contains(content, "build failed") || strings.Contains(content, "compile error") {
		return "CompilationError"
	}

	if strings.Contains(content, "eslint") || strings.Contains(content, "lint error") || strings.Contains(content, "tslint") {
		return "LintError"
	}

	if strings.Contains(content, "runtime error") || strings.Contains(content, "panic") || strings.Contains(content, "exception") {
		return "RuntimeError"
	}

	// Check for network errors last since they have generic keywords that might match other error types
	if strings.Contains(content, "fetcherror") || strings.Contains(content, "enotfound") || strings.Contains(content, "network") {
		return "NetworkError"
	}

	// Special case for MongoDB with "connection" - check if mongo is also mentioned
	if strings.Contains(content, "connection") && strings.Contains(content, "mongo") {
		return "MongoError"
	}

	// Generic connection errors
	if strings.Contains(content, "connection") {
		return "NetworkError"
	}

	return "Error"
}

// extractMainMessage extracts a clean error message from the first line
func extractMainMessage(firstLine string) string {
	cleaned := stripLogPrefixes(firstLine)

	// Limit message length for display
	if len(cleaned) > 200 {
		return cleaned[:197] + "..."
	}

	return cleaned
}

// stripLogPrefixes removes common log prefixes to extract the core message
func stripLogPrefixes(content string) string {
	// This is a simplified version - could be more sophisticated
	content = strings.TrimSpace(content)

	// Remove common timestamp patterns
	patterns := []string{
		`[`,
		`(`,
	}

	for _, pattern := range patterns {
		if strings.HasPrefix(content, pattern) {
			// Find the closing bracket/paren and remove everything up to it
			var closing string
			switch pattern {
			case `[`:
				closing = `]`
			case `(`:
				closing = `)`
			}

			if idx := strings.Index(content, closing); idx != -1 {
				content = strings.TrimSpace(content[idx+1:])
				if strings.HasPrefix(content, ":") {
					content = strings.TrimSpace(content[1:])
				}
			}
		}
	}

	return content
}

// determineSeverity determines the severity level from content
func determineSeverity(content string) string {
	content = strings.ToLower(content)

	if strings.Contains(content, "critical") ||
		strings.Contains(content, "fatal") ||
		strings.Contains(content, "panic") {
		return "critical"
	}

	if strings.Contains(content, "error") ||
		strings.Contains(content, "fail") {
		return "error"
	}

	if strings.Contains(content, "warn") {
		return "warning"
	}

	return "error" // Default
}

// sortGroupsByTime sorts error groups by their start time
func sortGroupsByTime(groups []ErrorGroup) {
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].StartTime.Before(groups[j].StartTime)
	})
}
