package process

import (
	"fmt"
	"time"
)

// ProcessState represents an immutable snapshot of process state.
// This struct is designed for atomic pointer swapping - no mutex needed.
// All fields should be treated as read-only after creation.
type ProcessState struct {
	// Identity fields (never change after creation)
	ID     string
	Name   string
	Script string

	// State fields (change atomically together)
	Status    ProcessStatus
	StartTime time.Time
	EndTime   *time.Time
	ExitCode  *int

	// Process execution details
	Command string
	Args    []string
	Env     []string
	Dir     string
}

// IsRunning returns true if the process is currently running
func (ps ProcessState) IsRunning() bool {
	return ps.Status == StatusRunning
}

// IsFinished returns true if the process has finished (stopped, failed, or success)
func (ps ProcessState) IsFinished() bool {
	return ps.Status == StatusStopped ||
		ps.Status == StatusFailed ||
		ps.Status == StatusSuccess
}

// Duration returns how long the process has been running or ran for
func (ps ProcessState) Duration() time.Duration {
	if ps.EndTime != nil {
		return ps.EndTime.Sub(ps.StartTime)
	}
	return time.Since(ps.StartTime)
}

// String returns a string representation of the process state
func (ps ProcessState) String() string {
	return fmt.Sprintf("Process[%s-%s: %s]", ps.ID, ps.Name, ps.Status)
}

// CopyWithStatus creates a new ProcessState with updated status.
// This is the core pattern for atomic updates.
func (ps ProcessState) CopyWithStatus(status ProcessStatus) ProcessState {
	newState := ps // Struct copy
	newState.Status = status

	// Handle status-specific field updates
	if status.IsFinished() && ps.EndTime == nil {
		now := time.Now()
		newState.EndTime = &now
	}

	return newState
}

// CopyWithExit creates a new ProcessState with exit information
func (ps ProcessState) CopyWithExit(exitCode int) ProcessState {
	newState := ps
	newState.ExitCode = &exitCode

	// Set status based on exit code
	if exitCode == 0 {
		newState.Status = StatusSuccess
	} else {
		newState.Status = StatusFailed
	}

	// Ensure we have an end time
	if ps.EndTime == nil {
		now := time.Now()
		newState.EndTime = &now
	}

	return newState
}

// CopyWithEndTime creates a new ProcessState with updated end time
func (ps ProcessState) CopyWithEndTime(endTime time.Time) ProcessState {
	newState := ps
	newState.EndTime = &endTime
	return newState
}

// IsFinished returns true if the status represents a finished state
func (s ProcessStatus) IsFinished() bool {
	return s == StatusStopped || s == StatusFailed || s == StatusSuccess
}
