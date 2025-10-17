package state

import (
	"sync/atomic"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/types"
)

// MasterData represents the single source of truth for all TUI state
// This wraps existing state structures without redefining them
type MasterData struct {
	// Version tracking for atomic updates
	version atomic.Uint64

	// Existing state structures - use composition
	StateManager      *StateManager
	OperationsTracker *OperationsTracker
	UiComponents      *UIComponentsManager
	CacheState        *Cache

	// Last update timestamp for tracking
	LastUpdate atomic.Value // time.Time
}

// NewMasterData creates a new master data object that wraps existing state
func NewMasterData() *MasterData {
	return &MasterData{
		// Initialize version
		version: atomic.Uint64{},

		// Initialize existing state structures
		StateManager:      NewStateManager(),
		OperationsTracker: NewOperationsTracker(),
		UiComponents:      NewUIComponentsManager(1000), // max size
		CacheState:        NewCache(1000, 5*time.Minute), // max size, TTL

		// Initialize last update time
		LastUpdate: atomic.Value{},
	}
}

// UpdateVersion increments the version and updates timestamp
func (md *MasterData) UpdateVersion() uint64 {
	newVersion := md.version.Add(1)
	md.LastUpdate.Store(time.Now())
	return newVersion
}

// GetVersion returns the current version
func (md *MasterData) GetVersion() uint64 {
	return md.version.Load()
}

// GetLastUpdate returns the last update time
func (md *MasterData) GetLastUpdate() time.Time {
	if val := md.LastUpdate.Load(); val != nil {
		return val.(time.Time)
	}
	return time.Time{}
}

// GetStateSummary returns a summary of current state
func (md *MasterData) GetStateSummary() map[string]interface{} {
	return map[string]interface{}{
		"version":     md.GetVersion(),
		"last_update": md.GetLastUpdate(),
		"operations":  md.OperationsTracker.GetStats(),
		// Note: Add other summaries as methods become available
	}
}

// ValidateState checks for state consistency across all components
func (md *MasterData) ValidateState() []string {
	var issues []string

	// Check version consistency
	if md.GetVersion() == 0 {
		issues = append(issues, "version is 0 - state not initialized")
	}

	// Basic validation - check for nil pointers
	if md.StateManager == nil {
		issues = append(issues, "StateManager is nil")
	}
	if md.OperationsTracker == nil {
		issues = append(issues, "OperationsTracker is nil")
	}
	if md.UiComponents == nil {
		issues = append(issues, "UiComponents is nil")
	}
	if md.CacheState == nil {
		issues = append(issues, "CacheState is nil")
	}

	return issues
}

// State update methods for common operations

// UpdateCurrentView updates the current view in the application state
func (md *MasterData) UpdateCurrentView(view messages.ViewType) {
	// TODO: Implement once we understand the StateManager structure
	md.UpdateVersion()
}

// AddOperation adds an operation to the operations tracker
func (md *MasterData) AddOperation(id string, commandType types.CommandType, context types.OperationContext, timeout time.Duration) {
	if md.OperationsTracker != nil {
		md.OperationsTracker.AddOperation(id, commandType, context, timeout)
	}
	md.UpdateVersion()
}

// UpdateCache updates the cache state
func (md *MasterData) UpdateCache(key string, value interface{}) {
	// TODO: Implement once we understand the Cache structure
	md.UpdateVersion()
}

// Cleanup performs cleanup operations across all state components
func (md *MasterData) Cleanup() {
	// TODO: Implement cleanup operations
	md.UpdateVersion()
}