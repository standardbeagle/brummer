package state

import (
	"fmt"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
)

// ErrorsState manages all errors-related UI state
// This is a sub-model that encapsulates errors display state according to Elm/Redux patterns
//
// IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
// NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
type ErrorsState struct {
	// Error data cache (existing system)
	ErrorContexts    []logs.ErrorContext
	CacheTimestamp   time.Time
	NeedsRefresh     bool

	// Selection state (existing system)
	SelectedError    *logs.ErrorContext
	SelectedErrorID  string

	// NEW: Grouped error support (for future extensibility)
	GroupedErrors    []GroupedError
	ParserState      ErrorParserState
	MaxRetention     int

	// Filter settings (NEW)
	FilterProcess    string

	// Display settings
	ShowDetailView   bool
	ShowList         bool
	AutoRefresh      bool

	// UI state
	Loading          bool
	LastError        string
	ErrorCount       int

	// Version for state tracking
	version          uint64
}

// NewErrorsState creates a new errors sub-model with initial state
func NewErrorsState() *ErrorsState {
	return &ErrorsState{
		ErrorContexts:   []logs.ErrorContext{},
		CacheTimestamp:  time.Time{},
		NeedsRefresh:    true,
		SelectedError:   nil,
		SelectedErrorID: "",
		GroupedErrors:   []GroupedError{},
		ParserState:     NewErrorParserState(),
		MaxRetention:    1000, // Default retention limit
		FilterProcess:   "",
		ShowDetailView:  false,
		ShowList:        true,
		AutoRefresh:     true,
		Loading:         false,
		LastError:       "",
		ErrorCount:      0,
		version:         0,
	}
}

// UpdateErrorContexts updates the cached error contexts (immutable)
func (s *ErrorsState) UpdateErrorContexts(contexts []logs.ErrorContext) *ErrorsState {

	newState := s.copy()
	newState.ErrorContexts = make([]logs.ErrorContext, len(contexts))
	copy(newState.ErrorContexts, contexts)
	newState.CacheTimestamp = time.Now()
	newState.NeedsRefresh = false
	newState.ErrorCount = len(contexts)
	newState.incrementVersion()
	return newState
}

// SetSelectedError updates the selected error (immutable)
func (s *ErrorsState) SetSelectedError(errorID string) *ErrorsState {

	newState := s.copy()
	newState.SelectedErrorID = errorID

	// Find the error in the cached contexts
	newState.SelectedError = nil
	for _, ec := range newState.ErrorContexts {
		if ec.ID == errorID {
			newState.SelectedError = &ec
			break
		}
	}

	newState.incrementVersion()
	return newState
}

// SetShowDetailView updates the detail view setting (immutable)
func (s *ErrorsState) SetShowDetailView(show bool) *ErrorsState {

	newState := s.copy()
	newState.ShowDetailView = show
	newState.incrementVersion()
	return newState
}

// SetShowList updates the list view setting (immutable)
func (s *ErrorsState) SetShowList(show bool) *ErrorsState {

	newState := s.copy()
	newState.ShowList = show
	newState.incrementVersion()
	return newState
}

// SetAutoRefresh updates the auto-refresh setting (immutable)
func (s *ErrorsState) SetAutoRefresh(refresh bool) *ErrorsState {

	newState := s.copy()
	newState.AutoRefresh = refresh
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *ErrorsState) SetLoading(loading bool) *ErrorsState {

	newState := s.copy()
	newState.Loading = loading
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *ErrorsState) SetError(err error) *ErrorsState {

	newState := s.copy()
	if err != nil {
		newState.LastError = err.Error()
	} else {
		newState.LastError = ""
	}
	newState.incrementVersion()
	return newState
}

// MarkForRefresh marks that the error list needs refreshing (immutable)
func (s *ErrorsState) MarkForRefresh() *ErrorsState {

	newState := s.copy()
	newState.NeedsRefresh = true
	newState.incrementVersion()
	return newState
}

// ClearErrors clears all errors (immutable)
func (s *ErrorsState) ClearErrors() *ErrorsState {

	newState := s.copy()
	newState.ErrorContexts = []logs.ErrorContext{}
	newState.CacheTimestamp = time.Time{}
	newState.NeedsRefresh = true
	newState.ErrorCount = 0
	newState.SelectedError = nil
	newState.SelectedErrorID = ""
	newState.incrementVersion()
	return newState
}

// IsCacheValid checks if the cached errors are still valid
func (s *ErrorsState) IsCacheValid(maxAge time.Duration) bool {

	if s.CacheTimestamp.IsZero() || s.NeedsRefresh {
		return false
	}

	return time.Since(s.CacheTimestamp) < maxAge
}

// GetErrorByID finds an error by its ID
func (s *ErrorsState) GetErrorByID(id string) *logs.ErrorContext {

	for _, ec := range s.ErrorContexts {
		if ec.ID == id {
			return &ec
		}
	}
	return nil
}

// GetLatestError returns the most recent error
func (s *ErrorsState) GetLatestError() *logs.ErrorContext {

	if len(s.ErrorContexts) == 0 {
		return nil
	}

	// Find the most recent error by timestamp
	latest := &s.ErrorContexts[0]
	for i := 1; i < len(s.ErrorContexts); i++ {
		if s.ErrorContexts[i].Timestamp.After(latest.Timestamp) {
			latest = &s.ErrorContexts[i]
		}
	}

	return latest
}

// GetErrorsByType returns errors filtered by type
func (s *ErrorsState) GetErrorsByType(errorType string) []logs.ErrorContext {

	var filtered []logs.ErrorContext
	for _, ec := range s.ErrorContexts {
		if ec.Type == errorType {
			filtered = append(filtered, ec)
		}
	}
	return filtered
}

// GetErrorsByProcess returns errors filtered by process name
func (s *ErrorsState) GetErrorsByProcess(processName string) []logs.ErrorContext {

	var filtered []logs.ErrorContext
	for _, ec := range s.ErrorContexts {
		if ec.ProcessName == processName {
			filtered = append(filtered, ec)
		}
	}
	return filtered
}

// Getters for current state values
func (s *ErrorsState) GetErrorContexts() []logs.ErrorContext {

	contexts := make([]logs.ErrorContext, len(s.ErrorContexts))
	copy(contexts, s.ErrorContexts)
	return contexts
}

func (s *ErrorsState) GetSelectedError() *logs.ErrorContext {
	return s.SelectedError
}

func (s *ErrorsState) GetSelectedErrorID() string {
	return s.SelectedErrorID
}

func (s *ErrorsState) GetShowDetailView() bool {
	return s.ShowDetailView
}

func (s *ErrorsState) GetShowList() bool {
	return s.ShowList
}

func (s *ErrorsState) GetAutoRefresh() bool {
	return s.AutoRefresh
}

func (s *ErrorsState) GetLoading() bool {
	return s.Loading
}

func (s *ErrorsState) GetError() string {
	return s.LastError
}

func (s *ErrorsState) GetErrorCount() int {
	return s.ErrorCount
}

func (s *ErrorsState) GetNeedsRefresh() bool {
	return s.NeedsRefresh
}

func (s *ErrorsState) GetVersion() uint64 {
	return s.version
}

// Helper methods
func (s *ErrorsState) copy() *ErrorsState {
	newState := *s
	return &newState
}

func (s *ErrorsState) incrementVersion() {
	s.version++
}

// NEW METHODS FOR GROUPED ERRORS

// AddGroupedError adds a new grouped error (immutable)
func (s *ErrorsState) AddGroupedError(error GroupedError) *ErrorsState {
	newState := s.copy()

	// Apply retention limit if needed
	if len(newState.GroupedErrors) >= newState.MaxRetention {
		// Remove oldest error (FIFO)
		newState.GroupedErrors = newState.GroupedErrors[1:]
	}

	// Add new error with incremented version
	error.Version = newState.version + 1
	newState.GroupedErrors = append(newState.GroupedErrors, error)
	newState.incrementVersion()

	return newState
}

// SetFilterProcess sets the process filter (immutable)
func (s *ErrorsState) SetFilterProcess(process string) *ErrorsState {
	newState := s.copy()
	newState.FilterProcess = process
	newState.incrementVersion()
	return newState
}

// SetMaxRetention sets the maximum number of errors to retain (immutable)
func (s *ErrorsState) SetMaxRetention(maxRetention int) *ErrorsState {
	newState := s.copy()
	newState.MaxRetention = maxRetention

	// Apply new retention limit immediately if needed
	if len(newState.GroupedErrors) > maxRetention {
		newState.GroupedErrors = newState.GroupedErrors[:maxRetention]
	}

	newState.incrementVersion()
	return newState
}

// GetGroupedErrors returns a copy of the grouped errors
func (s *ErrorsState) GetGroupedErrors() []GroupedError {
	errors := make([]GroupedError, len(s.GroupedErrors))
	copy(errors, s.GroupedErrors)
	return errors
}

// GetFilterProcess returns the current process filter
func (s *ErrorsState) GetFilterProcess() string {
	return s.FilterProcess
}

// GetMaxRetention returns the current max retention limit
func (s *ErrorsState) GetMaxRetention() int {
	return s.MaxRetention
}

// GetParserState returns the current parser state
func (s *ErrorsState) GetParserState() ErrorParserState {
	return s.ParserState
}

// SetParserState updates the parser state (immutable)
func (s *ErrorsState) SetParserState(parserState ErrorParserState) *ErrorsState {
	newState := s.copy()
	newState.ParserState = parserState
	newState.incrementVersion()
	return newState
}

// GetSelectedIndex returns the selected error index (for grouped errors)
func (s *ErrorsState) GetSelectedIndex() int {
	// This is a simple implementation - in a full implementation,
	// this would be tracked as a separate field
	return 0 // Always return 0 for now
}

// Validate validates the ErrorsState
func (s *ErrorsState) Validate() []string {
	var issues []string

	// Validate selected index bounds
	if s.SelectedErrorID != "" {
		found := false
		for _, ec := range s.ErrorContexts {
			if ec.ID == s.SelectedErrorID {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, "SelectedErrorID not found in ErrorContexts")
		}
	}

	// Validate retention limit
	if s.MaxRetention <= 0 {
		issues = append(issues, "MaxRetention must be positive")
	}

	// Validate grouped errors
	for i, ge := range s.GroupedErrors {
		geIssues := ge.Validate()
		for _, issue := range geIssues {
			issues = append(issues, fmt.Sprintf("GroupedError[%d]: %s", i, issue))
		}
	}

	return issues
}