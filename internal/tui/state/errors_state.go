package state

import (
	"sync"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
)

// ErrorsState manages all errors-related UI state
// This is a sub-model that encapsulates errors display state according to Elm/Redux patterns
type ErrorsState struct {
	// Error data cache
	ErrorContexts    []logs.ErrorContext
	CacheTimestamp   time.Time
	NeedsRefresh     bool

	// Selection state
	SelectedError    *logs.ErrorContext
	SelectedErrorID  string

	// Display settings
	ShowDetailView   bool
	ShowList         bool
	AutoRefresh      bool

	// UI state
	Loading          bool
	LastError        string
	ErrorCount       int

	// Version for atomic updates and state tracking
	version          uint64
	mu               sync.RWMutex
}

// NewErrorsState creates a new errors sub-model with initial state
func NewErrorsState() *ErrorsState {
	return &ErrorsState{
		ErrorContexts:   []logs.ErrorContext{},
		CacheTimestamp:  time.Time{},
		NeedsRefresh:    true,
		SelectedError:   nil,
		SelectedErrorID: "",
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
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.ShowDetailView = show
	newState.incrementVersion()
	return newState
}

// SetShowList updates the list view setting (immutable)
func (s *ErrorsState) SetShowList(show bool) *ErrorsState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.ShowList = show
	newState.incrementVersion()
	return newState
}

// SetAutoRefresh updates the auto-refresh setting (immutable)
func (s *ErrorsState) SetAutoRefresh(refresh bool) *ErrorsState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.AutoRefresh = refresh
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *ErrorsState) SetLoading(loading bool) *ErrorsState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.Loading = loading
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *ErrorsState) SetError(err error) *ErrorsState {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.NeedsRefresh = true
	newState.incrementVersion()
	return newState
}

// ClearErrors clears all errors (immutable)
func (s *ErrorsState) ClearErrors() *ErrorsState {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.CacheTimestamp.IsZero() || s.NeedsRefresh {
		return false
	}

	return time.Since(s.CacheTimestamp) < maxAge
}

// GetErrorByID finds an error by its ID
func (s *ErrorsState) GetErrorByID(id string) *logs.ErrorContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ec := range s.ErrorContexts {
		if ec.ID == id {
			return &ec
		}
	}
	return nil
}

// GetLatestError returns the most recent error
func (s *ErrorsState) GetLatestError() *logs.ErrorContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	contexts := make([]logs.ErrorContext, len(s.ErrorContexts))
	copy(contexts, s.ErrorContexts)
	return contexts
}

func (s *ErrorsState) GetSelectedError() *logs.ErrorContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SelectedError
}

func (s *ErrorsState) GetSelectedErrorID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SelectedErrorID
}

func (s *ErrorsState) GetShowDetailView() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ShowDetailView
}

func (s *ErrorsState) GetShowList() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ShowList
}

func (s *ErrorsState) GetAutoRefresh() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AutoRefresh
}

func (s *ErrorsState) GetLoading() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Loading
}

func (s *ErrorsState) GetError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastError
}

func (s *ErrorsState) GetErrorCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ErrorCount
}

func (s *ErrorsState) GetNeedsRefresh() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.NeedsRefresh
}

func (s *ErrorsState) GetVersion() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
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