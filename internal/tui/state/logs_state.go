package state

import (
	"regexp"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
)

// LogsState manages all logs-related UI state
// This is a sub-model that encapsulates logs display state according to Elm/Redux patterns
//
// IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
// NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
type LogsState struct {
	// Filter settings
	ShowPattern      string
	HidePattern      string
	ShowHighPriority bool
	SelectedProcess  string

	// Display settings
	AutoScroll bool
	AtBottom   bool

	// Search state
	SearchQuery   string
	SearchResults []logs.LogEntry

	// UI state
	Loading   bool
	LastError string

	// Cache for formatted logs
	CachedLogs     []logs.CollapsedLogEntry
	CacheTimestamp time.Time
	CacheProcess   string

	// Viewport state (for persistence)
	ScrollOffset int

	// Version for state tracking
	version uint64

	// Clear tracking for Phase E enhancement
	ClearedAt       time.Time
	ShowClearedMsg  bool
}

// NewLogsState creates a new logs sub-model with initial state
func NewLogsState() *LogsState {
	return &LogsState{
		ShowPattern:      "",
		HidePattern:      "",
		ShowHighPriority: false,
		SelectedProcess:  "",
		AutoScroll:       true,
		AtBottom:         true,
		SearchQuery:      "",
		SearchResults:    []logs.LogEntry{},
		Loading:          false,
		LastError:        "",
		CachedLogs:       []logs.CollapsedLogEntry{},
		CacheTimestamp:   time.Time{},
		CacheProcess:     "",
		ScrollOffset:     0,
		version:          0,
	}
}

// SetShowPattern updates the show pattern filter (immutable)
func (s *LogsState) SetShowPattern(pattern string) *LogsState {

	newState := s.copy()
	newState.ShowPattern = pattern
	newState.incrementVersion()
	newState.invalidateCache()
	return newState
}

// SetHidePattern updates the hide pattern filter (immutable)
func (s *LogsState) SetHidePattern(pattern string) *LogsState {

	newState := s.copy()
	newState.HidePattern = pattern
	newState.incrementVersion()
	newState.invalidateCache()
	return newState
}

// SetShowHighPriority updates the high priority filter (immutable)
func (s *LogsState) SetShowHighPriority(show bool) *LogsState {

	newState := s.copy()
	newState.ShowHighPriority = show
	newState.incrementVersion()
	newState.invalidateCache()
	return newState
}

// SetSelectedProcess updates the selected process filter (immutable)
func (s *LogsState) SetSelectedProcess(processID string) *LogsState {

	newState := s.copy()
	newState.SelectedProcess = processID
	newState.incrementVersion()
	newState.invalidateCache()
	return newState
}

// ToggleAutoScroll toggles the auto-scroll setting (immutable)
func (s *LogsState) ToggleAutoScroll() *LogsState {

	newState := s.copy()
	newState.AutoScroll = !newState.AutoScroll
	newState.incrementVersion()
	return newState
}

// EnableAutoScroll enables auto-scroll (immutable)
func (s *LogsState) EnableAutoScroll() *LogsState {

	newState := s.copy()
	newState.AutoScroll = true
	newState.AtBottom = true
	newState.ScrollOffset = 0
	newState.incrementVersion()
	return newState
}

// DisableAutoScroll disables auto-scroll (immutable)
func (s *LogsState) DisableAutoScroll() *LogsState {

	newState := s.copy()
	newState.AutoScroll = false
	newState.incrementVersion()
	return newState
}

// ScrollUp scrolls up (toward older logs) and disables auto-scroll (immutable)
func (s *LogsState) ScrollUp(lines int) *LogsState {

	newState := s.copy()
	// Scrolling up means viewing older content, so we DECREASE the offset
	// (offset 0 = top of logs, higher offset = scrolled down toward newer logs)
	newState.ScrollOffset -= lines
	if newState.ScrollOffset < 0 {
		newState.ScrollOffset = 0
	}
	// Scrolling up disables auto-scroll (user wants to view history)
	newState.AutoScroll = false
	newState.AtBottom = false
	newState.incrementVersion()
	return newState
}

// ScrollDown scrolls down (toward newer logs) (immutable)
func (s *LogsState) ScrollDown(lines int) *LogsState {

	newState := s.copy()
	// Scrolling down means viewing newer content, so we INCREASE the offset
	newState.ScrollOffset += lines
	// Note: We don't set AutoScroll=true here - user must explicitly enable it
	// by pressing Space or scrolling all the way to bottom
	newState.AtBottom = false
	newState.incrementVersion()
	return newState
}

// SetScrollOffset updates the scroll offset (immutable)
func (s *LogsState) SetScrollOffset(offset int) *LogsState {

	newState := s.copy()
	newState.ScrollOffset = offset
	newState.AtBottom = false
	newState.incrementVersion()
	return newState
}

// SetAtBottom marks whether the view is at the bottom (immutable)
func (s *LogsState) SetAtBottom(atBottom bool) *LogsState {

	newState := s.copy()
	newState.AtBottom = atBottom
	if atBottom {
		newState.ScrollOffset = 0
	}
	newState.incrementVersion()
	return newState
}

// SetSearchQuery updates the search query (immutable)
func (s *LogsState) SetSearchQuery(query string) *LogsState {

	newState := s.copy()
	newState.SearchQuery = query
	newState.incrementVersion()
	return newState
}

// SetSearchResults updates the search results (immutable)
func (s *LogsState) SetSearchResults(results []logs.LogEntry) *LogsState {

	newState := s.copy()
	newState.SearchResults = make([]logs.LogEntry, len(results))
	copy(newState.SearchResults, results)
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *LogsState) SetLoading(loading bool) *LogsState {

	newState := s.copy()
	newState.Loading = loading
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *LogsState) SetError(err error) *LogsState {

	newState := s.copy()
	if err != nil {
		newState.LastError = err.Error()
	} else {
		newState.LastError = ""
	}
	newState.incrementVersion()
	return newState
}

// UpdateCachedLogs updates the cached logs (immutable)
func (s *LogsState) UpdateCachedLogs(logEntries []logs.CollapsedLogEntry, processID string) *LogsState {

	newState := s.copy()
	newState.CachedLogs = make([]logs.CollapsedLogEntry, len(logEntries))
	copy(newState.CachedLogs, logEntries)
	newState.CacheTimestamp = time.Now()
	newState.CacheProcess = processID
	newState.incrementVersion()
	return newState
}

// ApplyFilters applies current filters to log entries
func (s *LogsState) ApplyFilters(entries []logs.CollapsedLogEntry) []logs.CollapsedLogEntry {

	if s.ShowPattern == "" && s.HidePattern == "" && !s.ShowHighPriority {
		return entries
	}

	var filtered []logs.CollapsedLogEntry
	var showRe, hideRe *regexp.Regexp
	var err error

	if s.ShowPattern != "" {
		showRe, err = regexp.Compile(s.ShowPattern)
		if err != nil {
			showRe = nil
		}
	}

	if s.HidePattern != "" {
		hideRe, err = regexp.Compile(s.HidePattern)
		if err != nil {
			hideRe = nil
		}
	}

	for _, entry := range entries {
		if s.ShowHighPriority && entry.Priority == 0 {
			continue
		}

		if showRe != nil && !showRe.MatchString(entry.Content) {
			continue
		}

		if hideRe != nil && hideRe.MatchString(entry.Content) {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// IsCacheValid checks if the cached logs are still valid
func (s *LogsState) IsCacheValid(processID string, maxAge time.Duration) bool {

	if s.CacheTimestamp.IsZero() || s.CacheProcess != processID {
		return false
	}

	return time.Since(s.CacheTimestamp) < maxAge
}

// Getters for current state values
func (s *LogsState) GetShowPattern() string {
	return s.ShowPattern
}

func (s *LogsState) GetHidePattern() string {
	return s.HidePattern
}

func (s *LogsState) GetShowHighPriority() bool {
	return s.ShowHighPriority
}

func (s *LogsState) GetSelectedProcess() string {
	return s.SelectedProcess
}

func (s *LogsState) GetAutoScroll() bool {
	return s.AutoScroll
}

func (s *LogsState) GetAtBottom() bool {
	return s.AtBottom
}

func (s *LogsState) GetScrollOffset() int {
	return s.ScrollOffset
}

func (s *LogsState) GetSearchQuery() string {
	return s.SearchQuery
}

func (s *LogsState) GetSearchResults() []logs.LogEntry {

	results := make([]logs.LogEntry, len(s.SearchResults))
	copy(results, s.SearchResults)
	return results
}

func (s *LogsState) GetLoading() bool {
	return s.Loading
}

func (s *LogsState) GetError() string {
	return s.LastError
}

func (s *LogsState) GetCachedLogs() []logs.CollapsedLogEntry {

	logs := make([]logs.CollapsedLogEntry, len(s.CachedLogs))
	copy(logs, s.CachedLogs)
	return logs
}

func (s *LogsState) GetVersion() uint64 {
	return s.version
}

func (s *LogsState) GetClearedAt() time.Time {
	return s.ClearedAt
}

func (s *LogsState) GetShowClearedMsg() bool {
	return s.ShowClearedMsg
}

// HideClearedMsg hides the cleared message (called when new logs arrive)
func (s *LogsState) HideClearedMsg() *LogsState {
	newState := s.copy()
	newState.ShowClearedMsg = false
	newState.incrementVersion()
	return newState
}

// Helper methods
func (s *LogsState) copy() *LogsState {
	newState := *s
	return &newState
}

func (s *LogsState) incrementVersion() {
	s.version++
}

func (s *LogsState) invalidateCache() {
	s.CacheTimestamp = time.Time{}
	s.CacheProcess = ""
}

// ClearLogs clears all cached logs while preserving filters and settings (immutable)
func (s *LogsState) ClearLogs() *LogsState {
	newState := s.copy()

	// Clear the cached logs
	newState.CachedLogs = []logs.CollapsedLogEntry{}
	newState.CacheTimestamp = time.Time{}
	newState.CacheProcess = ""

	// Clear search results
	newState.SearchResults = []logs.LogEntry{}
	newState.SearchQuery = ""

	// Reset scroll position
	newState.ScrollOffset = 0
	newState.AtBottom = true

	// Clear any error state
	newState.LastError = ""

	// Set clear tracking fields for Phase E enhancement
	newState.ClearedAt = time.Now()
	newState.ShowClearedMsg = true

	// Increment version
	newState.incrementVersion()

	return newState
}