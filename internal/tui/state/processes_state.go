package state

import (
	"sync"
	"time"

	"github.com/standardbeagle/brummer/internal/domain/processes"
)

// ProcessesState manages all processes-related UI state
// This is a sub-model that encapsulates processes display state according to Elm/Redux patterns
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

	// Version for atomic updates and state tracking
	version         uint64
	mu              sync.RWMutex
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
func (s *ProcessesState) SetSelectedProcess(processID string) *ProcessesState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.SelectedProcess = processID
	newState.incrementVersion()
	return newState
}

// UpdateProcesses updates the cached process list (immutable)
func (s *ProcessesState) UpdateProcesses(processList []*processes.Process) *ProcessesState {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.GroupByStatus = group
	newState.incrementVersion()
	return newState
}

// SetSortByUptime updates the sorting setting (immutable)
func (s *ProcessesState) SetSortByUptime(sort bool) *ProcessesState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.SortByUptime = sort
	newState.incrementVersion()
	return newState
}

// SetShowStopped updates the show stopped setting (immutable)
func (s *ProcessesState) SetShowStopped(show bool) *ProcessesState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.ShowStopped = show
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *ProcessesState) SetLoading(loading bool) *ProcessesState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.Loading = loading
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *ProcessesState) SetError(err error) *ProcessesState {
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

// MarkForRefresh marks that the process list needs refreshing (immutable)
func (s *ProcessesState) MarkForRefresh() *ProcessesState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.NeedsRefresh = true
	newState.incrementVersion()
	return newState
}

// IsCacheValid checks if the cached processes are still valid
func (s *ProcessesState) IsCacheValid(maxAge time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.CacheTimestamp.IsZero() || s.NeedsRefresh {
		return false
	}

	return time.Since(s.CacheTimestamp) < maxAge
}

// GetRunningProcesses returns only running processes
func (s *ProcessesState) GetRunningProcesses() []*processes.Process {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, proc := range s.Processes {
		if proc.ID == id {
			return proc
		}
	}
	return nil
}

// FindProcessByName finds a process by its name
func (s *ProcessesState) FindProcessByName(name string) *processes.Process {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, proc := range s.Processes {
		if proc.Name == name {
			return proc
		}
	}
	return nil
}

// GetProcessCount returns the total number of processes
func (s *ProcessesState) GetProcessCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Processes)
}

// GetRunningCount returns the number of running processes
func (s *ProcessesState) GetRunningCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SelectedProcess
}

func (s *ProcessesState) GetProcesses() []*processes.Process {
	s.mu.RLock()
	defer s.mu.RUnlock()

	processes := make([]*processes.Process, len(s.Processes))
	copy(processes, s.Processes)
	return processes
}

func (s *ProcessesState) GetGroupByStatus() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GroupByStatus
}

func (s *ProcessesState) GetSortByUptime() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SortByUptime
}

func (s *ProcessesState) GetShowStopped() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ShowStopped
}

func (s *ProcessesState) GetLoading() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Loading
}

func (s *ProcessesState) GetError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastError
}

func (s *ProcessesState) GetNeedsRefresh() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.NeedsRefresh
}

func (s *ProcessesState) GetVersion() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// Helper methods
func (s *ProcessesState) copy() *ProcessesState {
	newState := *s
	return &newState
}

func (s *ProcessesState) incrementVersion() {
	s.version++
}