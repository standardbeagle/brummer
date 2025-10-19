package state

import (
	"sync"
	"time"
)

// TestTab represents the active tab in the test view
type TestTab int

const (
	TestTabSummary TestTab = iota
	TestTabResults
	TestTabCoverage
	TestTabPerformance
)

// TestRunInfo holds display information for a test run
type TestRunInfo struct {
	ID            string
	Name          string
	Status        string // "running", "completed", "failed", "cancelled"
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	TotalTests    int
	PassedTests   int
	FailedTests   int
	SkippedTests  int
	Coverage      float64
	ErrorMessage  string
	LastUpdated   time.Time
}

// TestResultInfo holds display information for a test case result
type TestResultInfo struct {
	RunID       string
	Name        string
	Status      string // "passed", "failed", "skipped", "running"
	Duration    time.Duration
	ErrorMessage string
	Output      string
	Coverage    float64
}

// TestViewState manages all test view UI state
// This is a sub-model that encapsulates test view display according to Elm/Redux patterns
type TestViewState struct {
	// Test runs data
	ActiveRuns      []TestRunInfo
	SelectedRunID   string
	Results         map[string][]TestResultInfo // runID -> test results

	// UI state
	SelectedTab     TestTab
	ShowDetails     bool
	FilterStatus    string // "all", "passed", "failed", "skipped", "running"
	AutoRefresh     bool

	// Viewport state
	ViewportOffset  int
	ViewportHeight  int

	// Layout
	Width           int
	Height          int

	// Current execution state
	Running         bool
	CurrentRun      *TestRunInfo

	// UI state
	Loading         bool
	LastError       string
	LastUpdated     time.Time
	MaxRuns         int // Maximum runs to keep in memory

	// Statistics
	TotalRuns       int
	RunsByStatus    map[string]int

	// Version for atomic updates and state tracking
	version         uint64
	mu              sync.RWMutex
}

// NewTestViewState creates a new test view sub-model with initial state
func NewTestViewState() *TestViewState {
	return &TestViewState{
		ActiveRuns:      []TestRunInfo{},
		SelectedRunID:   "",
		Results:         make(map[string][]TestResultInfo),
		SelectedTab:     TestTabSummary,
		ShowDetails:     false,
		FilterStatus:    "all",
		AutoRefresh:     true,
		ViewportOffset:  0,
		ViewportHeight:  24,
		Width:           80,
		Height:          24,
		Running:         false,
		CurrentRun:      nil,
		Loading:         false,
		LastError:       "",
		LastUpdated:     time.Time{},
		MaxRuns:         50, // Keep last 50 test runs
		TotalRuns:       0,
		RunsByStatus:    make(map[string]int),
		version:         0,
	}
}

// StartTestRun starts a new test run (immutable)
func (s *TestViewState) StartTestRun(run TestRunInfo) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()

	// Add run to the beginning (newest first)
	newState.ActiveRuns = append([]TestRunInfo{run}, newState.ActiveRuns...)

	// Limit the number of runs kept in memory
	if len(newState.ActiveRuns) > newState.MaxRuns {
		newState.ActiveRuns = newState.ActiveRuns[:newState.MaxRuns]
	}

	// Set as current run if it's running
	if run.Status == "running" {
		newState.Running = true
		newState.CurrentRun = &newState.ActiveRuns[0]
	}

	// Update statistics
	newState.TotalRuns++
	newState.RunsByStatus[run.Status]++

	// Select this run if no run is selected
	if newState.SelectedRunID == "" {
		newState.SelectedRunID = run.ID
	}

	newState.LastUpdated = time.Now()
	newState.incrementVersion()
	return newState
}

// UpdateTestRun updates an existing test run (immutable)
func (s *TestViewState) UpdateTestRun(runID string, updates TestRunInfo) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()

	// Find and update the run
	for i, run := range newState.ActiveRuns {
		if run.ID == runID {
			// Update statistics
			oldStatus := run.Status
			if oldStatus != updates.Status {
				newState.RunsByStatus[oldStatus]--
				newState.RunsByStatus[updates.Status]++
			}

			// Update the run
			newState.ActiveRuns[i] = updates

			// Update current run if this is the running one
			if newState.CurrentRun != nil && newState.CurrentRun.ID == runID {
				newState.CurrentRun = &newState.ActiveRuns[i]

				// Update running state
				if updates.Status == "completed" || updates.Status == "failed" || updates.Status == "cancelled" {
					newState.Running = false
					newState.CurrentRun = nil
				}
			}

			break
		}
	}

	newState.LastUpdated = time.Now()
	newState.incrementVersion()
	return newState
}

// AddTestResult adds a test result to a run (immutable)
func (s *TestViewState) AddTestResult(runID string, result TestResultInfo) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()

	// Initialize results for this run if needed
	if newState.Results[runID] == nil {
		newState.Results[runID] = []TestResultInfo{}
	}

	// Add result
	newState.Results[runID] = append(newState.Results[runID], result)

	newState.LastUpdated = time.Now()
	newState.incrementVersion()
	return newState
}

// SetSelectedRun updates the selected run (immutable)
func (s *TestViewState) SetSelectedRun(runID string) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.SelectedRunID = runID
	newState.incrementVersion()
	return newState
}

// SetSelectedTab updates the selected tab (immutable)
func (s *TestViewState) SetSelectedTab(tab TestTab) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.SelectedTab = tab
	newState.incrementVersion()
	return newState
}

// ToggleDetails toggles the detail view (immutable)
func (s *TestViewState) ToggleDetails() *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.ShowDetails = !newState.ShowDetails
	newState.incrementVersion()
	return newState
}

// SetFilter updates the filter setting (immutable)
func (s *TestViewState) SetFilter(filter string) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.FilterStatus = filter
	newState.incrementVersion()
	return newState
}

// ToggleAutoRefresh toggles auto-refresh setting (immutable)
func (s *TestViewState) ToggleAutoRefresh() *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.AutoRefresh = !newState.AutoRefresh
	newState.incrementVersion()
	return newState
}

// SetViewportOffset updates the viewport offset (immutable)
func (s *TestViewState) SetViewportOffset(offset int) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.ViewportOffset = offset
	newState.incrementVersion()
	return newState
}

// SetSize updates the component dimensions (immutable)
func (s *TestViewState) SetSize(width, height int) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.Width = width
	newState.Height = height
	newState.ViewportHeight = height
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *TestViewState) SetLoading(loading bool) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.Loading = loading
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *TestViewState) SetError(err error) *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	if err != nil {
		newState.LastError = err.Error()
	} else {
		newState.LastError = ""
	}
	newState.Loading = false
	newState.incrementVersion()
	return newState
}

// ClearTestRuns clears all test runs (immutable)
func (s *TestViewState) ClearTestRuns() *TestViewState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.ActiveRuns = []TestRunInfo{}
	newState.SelectedRunID = ""
	newState.Results = make(map[string][]TestResultInfo)
	newState.TotalRuns = 0
	newState.RunsByStatus = make(map[string]int)
	newState.Running = false
	newState.CurrentRun = nil
	newState.incrementVersion()
	return newState
}

// GetSelectedRun returns the currently selected run
func (s *TestViewState) GetSelectedRun() *TestRunInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, run := range s.ActiveRuns {
		if run.ID == s.SelectedRunID {
			return &run
		}
	}
	return nil
}

// GetFilteredResults returns test results filtered by current filter
func (s *TestViewState) GetFilteredResults(runID string) []TestResultInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results, exists := s.Results[runID]
	if !exists {
		return []TestResultInfo{}
	}

	if s.FilterStatus == "all" {
		return results
	}

	var filtered []TestResultInfo
	for _, result := range results {
		if result.Status == s.FilterStatus {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// GetStateSummary returns a summary of current state
func (s *TestViewState) GetStateSummary() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	selectedRun := s.GetSelectedRun()
	var selectedRunName string
	if selectedRun != nil {
		selectedRunName = selectedRun.Name
	}

	return map[string]interface{}{
		"active_runs":      len(s.ActiveRuns),
		"selected_run_id":  s.SelectedRunID,
		"selected_run_name": selectedRunName,
		"selected_tab":     int(s.SelectedTab),
		"show_details":     s.ShowDetails,
		"filter_status":    s.FilterStatus,
		"auto_refresh":     s.AutoRefresh,
		"viewport_offset":  s.ViewportOffset,
		"viewport_height":  s.ViewportHeight,
		"width":            s.Width,
		"height":           s.Height,
		"is_running":       s.Running,
		"loading":          s.Loading,
		"last_error":       s.LastError,
		"last_updated":     s.LastUpdated,
		"max_runs":         s.MaxRuns,
		"total_runs":       s.TotalRuns,
		"runs_by_status":   s.RunsByStatus,
		"version":          s.version,
	}
}

// Validate checks for state consistency
func (s *TestViewState) Validate() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var issues []string

	// Basic validation
	if s.Width <= 0 || s.Height <= 0 {
		issues = append(issues, "invalid dimensions")
	}

	if s.MaxRuns <= 0 {
		issues = append(issues, "invalid max runs setting")
	}

	// Validate selected run exists
	if s.SelectedRunID != "" {
		found := false
		for _, run := range s.ActiveRuns {
			if run.ID == s.SelectedRunID {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, "selected run does not exist")
		}
	}

	// Validate selected tab
	if s.SelectedTab < TestTabSummary || s.SelectedTab > TestTabPerformance {
		issues = append(issues, "invalid selected tab")
	}

	// Validate filter
	validFilters := []string{"all", "passed", "failed", "skipped", "running"}
	filterValid := false
	for _, valid := range validFilters {
		if s.FilterStatus == valid {
			filterValid = true
			break
		}
	}
	if !filterValid {
		issues = append(issues, "invalid filter setting")
	}

	return issues
}

// Getters for current state values
func (s *TestViewState) GetActiveRuns() []TestRunInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runs := make([]TestRunInfo, len(s.ActiveRuns))
	copy(runs, s.ActiveRuns)
	return runs
}

func (s *TestViewState) GetSelectedRunID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SelectedRunID
}

func (s *TestViewState) GetSelectedTab() TestTab {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SelectedTab
}

func (s *TestViewState) IsShowingDetails() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ShowDetails
}

func (s *TestViewState) GetFilterStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.FilterStatus
}

func (s *TestViewState) IsAutoRefresh() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AutoRefresh
}

func (s *TestViewState) GetViewportOffset() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ViewportOffset
}

func (s *TestViewState) GetSize() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Width, s.Height
}

func (s *TestViewState) GetLoading() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Loading
}

func (s *TestViewState) GetError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastError
}

func (s *TestViewState) GetLastUpdated() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastUpdated
}

func (s *TestViewState) GetVersion() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

func (s *TestViewState) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Running
}

func (s *TestViewState) GetCurrentRun() *TestRunInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentRun
}

// Helper methods
func (s *TestViewState) copy() *TestViewState {
	newState := *s

	// Deep copy slices
	newState.ActiveRuns = make([]TestRunInfo, len(s.ActiveRuns))
	copy(newState.ActiveRuns, s.ActiveRuns)

	// Deep copy maps
	newState.Results = make(map[string][]TestResultInfo)
	for k, v := range s.Results {
		newState.Results[k] = make([]TestResultInfo, len(v))
		copy(newState.Results[k], v)
	}

	newState.RunsByStatus = make(map[string]int)
	for k, v := range s.RunsByStatus {
		newState.RunsByStatus[k] = v
	}

	return &newState
}

func (s *TestViewState) incrementVersion() {
	s.version++
}