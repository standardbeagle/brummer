package state

import (
	"time"

	"github.com/standardbeagle/brummer/internal/domain/processes"
)

// ProcessesState manages all processes-related UI state
// This is a sub-model that encapsulates processes display state according to Elm/Redux patterns
//
// IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
// NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
//
// ⚠️  CRITICAL: DO NOT add mutexes to this struct!
// ❌ WRONG:  mu sync.RWMutex  // DON'T DO THIS!
// WHY: Adding mutexes creates deadlocks when Update() locks the OLD object while
//      View() tries to read from it. This caused the keyboard navigation freeze bug.
// ✅ CORRECT: Keep this struct mutex-free. Immutability provides thread-safety.
//
// ⚠️  CRITICAL: DO NOT add mutable fields (channels, maps with pointer values, etc.)!
// ❌ WRONG:  UpdateChan chan Update  // DON'T DO THIS!
// WHY: Mutable fields break the immutability guarantee and create race conditions.
// ✅ CORRECT: Only use immutable types (strings, ints, slices of values, time.Time, etc.)
//
// See: /docs/architecture/redux-elm-architecture.md for detailed explanation
type ProcessesState struct {
	// Selection state
	SelectedProcess string

	// Process data cache
	Processes       []*processes.Process
	CacheTimestamp  time.Time
	NeedsRefresh    bool

	// Display settings
	GroupByStatus   bool
	SortByUptime    bool
	ShowStopped     bool

	// UI state
	Loading         bool
	LastError       string

	// Version for state tracking
	version         uint64
}

// NewProcessesState creates a new processes sub-model with initial state
func NewProcessesState() *ProcessesState {
	return &ProcessesState{
		SelectedProcess: "",
		Processes:       []*processes.Process{},
		CacheTimestamp:  time.Time{},
		NeedsRefresh:    true,
		GroupByStatus:   true,
		SortByUptime:    false,
		ShowStopped:     true,
		Loading:         false,
		LastError:       "",
		version:         0,
	}
}

// SetSelectedProcess updates the selected process (immutable)
//
// ⚠️  CRITICAL: DO NOT modify state in place!
// ❌ WRONG:  s.SelectedProcess = processID; s.version++; return s  // DON'T mutate!
// WHY: Breaks immutability guarantee. Old references would see new data (race condition).
//      View() might read mid-modification, causing inconsistent state.
// ✅ CORRECT: Create NEW object via s.copy(), modify copy, return new object.
func (s *ProcessesState) SetSelectedProcess(processID string) *ProcessesState {

	newState := s.copy()
	newState.SelectedProcess = processID
	newState.incrementVersion()
	return newState
}

// UpdateProcesses updates the cached process list (immutable)
//
// ⚠️  CRITICAL: DO NOT modify state in place!
// ❌ WRONG:  s.Processes = processList; s.version++; return s  // DON'T mutate!
// WHY: Multiple goroutines might hold references to old state. Mutating in place
//      creates race conditions where View() reads partially updated data.
// ✅ CORRECT: Always copy the entire state object and return NEW instance.
//
// Note: Deep copy the slice to prevent external modifications from affecting state.
func (s *ProcessesState) UpdateProcesses(processList []*processes.Process) *ProcessesState {

	newState := s.copy()
	newState.Processes = make([]*processes.Process, len(processList))
	copy(newState.Processes, processList)
	newState.CacheTimestamp = time.Now()
	newState.NeedsRefresh = false
	newState.incrementVersion()
	return newState
}

// SetGroupByStatus updates the grouping setting (immutable)
func (s *ProcessesState) SetGroupByStatus(group bool) *ProcessesState {

	newState := s.copy()
	newState.GroupByStatus = group
	newState.incrementVersion()
	return newState
}

// SetSortByUptime updates the sorting setting (immutable)
func (s *ProcessesState) SetSortByUptime(sort bool) *ProcessesState {

	newState := s.copy()
	newState.SortByUptime = sort
	newState.incrementVersion()
	return newState
}

// SetShowStopped updates the show stopped setting (immutable)
func (s *ProcessesState) SetShowStopped(show bool) *ProcessesState {

	newState := s.copy()
	newState.ShowStopped = show
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *ProcessesState) SetLoading(loading bool) *ProcessesState {

	newState := s.copy()
	newState.Loading = loading
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *ProcessesState) SetError(err error) *ProcessesState {

	newState := s.copy()
	if err != nil {
		newState.LastError = err.Error()
	} else {
		newState.LastError = ""
	}
	newState.incrementVersion()
	return newState
}

// MarkForRefresh marks that the process list needs refreshing (immutable)
func (s *ProcessesState) MarkForRefresh() *ProcessesState {

	newState := s.copy()
	newState.NeedsRefresh = true
	newState.incrementVersion()
	return newState
}

// IsCacheValid checks if the cached processes are still valid
func (s *ProcessesState) IsCacheValid(maxAge time.Duration) bool {

	if s.CacheTimestamp.IsZero() || s.NeedsRefresh {
		return false
	}

	return time.Since(s.CacheTimestamp) < maxAge
}

// GetRunningProcesses returns only running processes
func (s *ProcessesState) GetRunningProcesses() []*processes.Process {

	var running []*processes.Process
	for _, proc := range s.Processes {
		if proc.GetStatus() == processes.StatusRunning {
			running = append(running, proc)
		}
	}
	return running
}

// GetStoppedProcesses returns only stopped processes
func (s *ProcessesState) GetStoppedProcesses() []*processes.Process {

	var stopped []*processes.Process
	for _, proc := range s.Processes {
		if proc.GetStatus() != processes.StatusRunning {
			stopped = append(stopped, proc)
		}
	}
	return stopped
}

// FindProcessByID finds a process by its ID
func (s *ProcessesState) FindProcessByID(id string) *processes.Process {

	for _, proc := range s.Processes {
		if proc.ID == id {
			return proc
		}
	}
	return nil
}

// FindProcessByName finds a process by its name
func (s *ProcessesState) FindProcessByName(name string) *processes.Process {

	for _, proc := range s.Processes {
		if proc.Name == name {
			return proc
		}
	}
	return nil
}

// GetProcessCount returns the total number of processes
func (s *ProcessesState) GetProcessCount() int {
	return len(s.Processes)
}

// GetRunningCount returns the number of running processes
func (s *ProcessesState) GetRunningCount() int {

	count := 0
	for _, proc := range s.Processes {
		if proc.GetStatus() == processes.StatusRunning {
			count++
		}
	}
	return count
}

// Getters for current state values
func (s *ProcessesState) GetSelectedProcess() string {
	return s.SelectedProcess
}

func (s *ProcessesState) GetProcesses() []*processes.Process {

	processes := make([]*processes.Process, len(s.Processes))
	copy(processes, s.Processes)
	return processes
}

func (s *ProcessesState) GetGroupByStatus() bool {
	return s.GroupByStatus
}

func (s *ProcessesState) GetSortByUptime() bool {
	return s.SortByUptime
}

func (s *ProcessesState) GetShowStopped() bool {
	return s.ShowStopped
}

func (s *ProcessesState) GetLoading() bool {
	return s.Loading
}

func (s *ProcessesState) GetError() string {
	return s.LastError
}

func (s *ProcessesState) GetNeedsRefresh() bool {
	return s.NeedsRefresh
}

func (s *ProcessesState) GetVersion() uint64 {
	return s.version
}

// Helper methods
//
// ⚠️  CRITICAL: copy() performs shallow copy - sufficient for immutable pattern
// WHY: Shallow copy creates new struct with same field values. Since we NEVER mutate
//      the Processes slice itself (only replace it entirely), shallow copy is safe.
// ✅ CORRECT: newState := *s creates new struct, then update specific fields.
//
// DANGER: If you add mutable fields (channels, maps), you MUST update copy() to
//         deep copy those fields, or better: DON'T add mutable fields at all!
func (s *ProcessesState) copy() *ProcessesState {
	newState := *s
	return &newState
}

func (s *ProcessesState) incrementVersion() {
	s.version++
}