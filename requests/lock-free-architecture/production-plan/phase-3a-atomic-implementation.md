# Phase 3A: Production Implementation - Atomic ProcessState

**Created**: January 31, 2025  
**Status**: Ready for Implementation  
**Validated Approach**: Atomic pointer swapping (30-300x faster than mutex)  
**Timeline**: Week 1 (5 days)

## Implementation Overview

Based on our assumption testing that proved atomic operations are 30-300x faster than mutexes (and channels were 15-67x slower), we're implementing the validated atomic pointer swapping pattern.

## Day 1-2: Create Immutable ProcessState Struct

### Task 1: Define ProcessState Structure
```go
// File: internal/process/state.go
package process

import (
    "time"
    "fmt"
)

// ProcessState represents an immutable snapshot of process state
// This struct is designed for atomic pointer swapping - no mutex needed
type ProcessState struct {
    // Identity fields (never change)
    ID     string
    Name   string
    Script string
    
    // State fields (change atomically together)
    Status    ProcessStatus
    StartTime time.Time
    EndTime   *time.Time
    ExitCode  *int
    
    // Additional fields for completeness
    Command   string
    Args      []string
    Env       []string
    Dir       string
}

// Convenience methods for state inspection
func (ps ProcessState) IsRunning() bool {
    return ps.Status == StatusRunning
}

func (ps ProcessState) IsFinished() bool {
    return ps.Status == StatusStopped || 
           ps.Status == StatusFailed || 
           ps.Status == StatusSuccess
}

func (ps ProcessState) Duration() time.Duration {
    if ps.EndTime != nil {
        return ps.EndTime.Sub(ps.StartTime)
    }
    return time.Since(ps.StartTime)
}

func (ps ProcessState) String() string {
    return fmt.Sprintf("Process[%s-%s: %s]", ps.ID, ps.Name, ps.Status)
}
```

### Task 2: Copy Constructor Pattern
```go
// CopyWithStatus creates a new ProcessState with updated status
// This is the core pattern for atomic updates
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
    
    if exitCode == 0 {
        newState.Status = StatusSuccess
    } else {
        newState.Status = StatusFailed
    }
    
    if ps.EndTime == nil {
        now := time.Now()
        newState.EndTime = &now
    }
    
    return newState
}
```

## Day 3-4: Implement Atomic Operations in Process

### Task 3: Add Atomic State to Process Struct
```go
// File: internal/process/manager.go
// Add to existing Process struct

import (
    "sync/atomic"
    "unsafe"
)

type Process struct {
    // Keep existing fields for backward compatibility
    mu        sync.RWMutex
    ID        string
    Name      string
    Script    string
    Status    ProcessStatus
    StartTime time.Time
    EndTime   *time.Time
    ExitCode  *int
    
    // NEW: Atomic state pointer for lock-free reads
    atomicState unsafe.Pointer // *ProcessState
    
    // Keep other existing fields...
    Cmd         *exec.Cmd
    cancelFunc  context.CancelFunc
    eventBus    *events.EventBus
    // ...
}
```

### Task 4: Implement Atomic Getters
```go
// GetStateAtomic returns the current process state atomically
// This is the PRIMARY method for lock-free state access
func (p *Process) GetStateAtomic() ProcessState {
    statePtr := (*ProcessState)(atomic.LoadPointer(&p.atomicState))
    if statePtr == nil {
        // Fallback: build from mutex-protected fields
        p.mu.RLock()
        defer p.mu.RUnlock()
        return ProcessState{
            ID:        p.ID,
            Name:      p.Name,
            Script:    p.Script,
            Status:    p.Status,
            StartTime: p.StartTime,
            EndTime:   p.EndTime,
            ExitCode:  p.ExitCode,
        }
    }
    return *statePtr
}

// UpdateStateAtomic performs atomic state update using CAS
func (p *Process) UpdateStateAtomic(updater func(ProcessState) ProcessState) {
    for {
        current := p.GetStateAtomic()
        newState := updater(current)
        newStatePtr := &newState
        
        // Try to swap the pointer atomically
        if atomic.CompareAndSwapPointer(
            &p.atomicState,
            unsafe.Pointer(&current),
            unsafe.Pointer(newStatePtr),
        ) {
            // Also update mutex-protected fields for compatibility
            p.updateMutexFields(newState)
            break
        }
        // If CAS failed, another update happened - retry
    }
}

// Helper to keep mutex fields in sync
func (p *Process) updateMutexFields(state ProcessState) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.Status = state.Status
    p.EndTime = state.EndTime
    p.ExitCode = state.ExitCode
}
```

### Task 5: Migrate Existing Methods to Use Atomics
```go
// Update existing getter methods to use atomic state
func (p *Process) GetStatus() ProcessStatus {
    // Fast path: try atomic first
    if state := p.GetStateAtomic(); state.ID != "" {
        return state.Status
    }
    // Fallback: mutex path
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.Status
}

func (p *Process) GetStartTime() time.Time {
    if state := p.GetStateAtomic(); state.ID != "" {
        return state.StartTime
    }
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.StartTime
}

// Similar updates for GetEndTime, GetExitCode, etc.
```

## Day 5: Update Hot-Path Code

### Task 6: Update MCP Tools
```go
// File: internal/mcp/tools.go
// Update scripts_status handler to use atomic state

func (s *MCPServer) handleScriptsStatus(params map[string]interface{}) (interface{}, error) {
    processes := s.processManager.GetAllProcesses()
    
    statuses := make([]map[string]interface{}, 0, len(processes))
    for _, proc := range processes {
        // Use atomic state for consistent multi-field access
        state := proc.GetStateAtomic()
        
        status := map[string]interface{}{
            "id":        state.ID,
            "name":      state.Name,
            "script":    state.Script,
            "status":    string(state.Status),
            "startTime": state.StartTime.Format(time.RFC3339),
        }
        
        if state.EndTime != nil {
            status["endTime"] = state.EndTime.Format(time.RFC3339)
            status["duration"] = state.Duration().Seconds()
        }
        
        if state.ExitCode != nil {
            status["exitCode"] = *state.ExitCode
        }
        
        statuses = append(statuses, status)
    }
    
    return map[string]interface{}{
        "processes": statuses,
    }, nil
}
```

### Task 7: Update TUI Components
```go
// File: internal/tui/model.go
// Update processItem methods to use atomic state

func (pi processItem) Title() string {
    // Get atomic snapshot for consistent view
    state := pi.process.GetStateAtomic()
    
    icon := getProcessIcon(state.Status)
    return fmt.Sprintf("%s %s", icon, state.Name)
}

func (pi processItem) Description() string {
    state := pi.process.GetStateAtomic()
    
    desc := fmt.Sprintf("Status: %s | Started: %s",
        state.Status,
        state.StartTime.Format("15:04:05"))
    
    if state.IsFinished() && state.EndTime != nil {
        desc += fmt.Sprintf(" | Duration: %s", state.Duration())
    }
    
    return desc
}
```

## Implementation Checklist

### Core Implementation
- [ ] Create ProcessState struct with all fields
- [ ] Implement convenience methods (IsRunning, Duration, etc.)
- [ ] Add copy constructors for state transitions
- [ ] Add atomicState pointer to Process struct
- [ ] Implement GetStateAtomic() method
- [ ] Implement UpdateStateAtomic() with CAS loop
- [ ] Update existing getters to use atomic path

### Hot-Path Updates
- [ ] Update MCP scripts_status handler
- [ ] Update MCP scripts_run duplicate detection
- [ ] Update TUI processItem methods
- [ ] Update data provider GetProcessInfo
- [ ] Update process list sorting logic

### Testing
- [ ] Create atomic operation benchmarks
- [ ] Add race condition tests
- [ ] Verify backward compatibility
- [ ] Test CAS retry behavior
- [ ] Benchmark improvements

## Success Metrics

### Performance Targets (from prototype)
- Single read: <1ns (prototype: 0.54ns)
- Concurrent reads: <1ns (prototype: 0.24ns)
- Zero allocations for reads
- 30x improvement over mutex baseline

### Compatibility Requirements
- All existing tests must pass
- No breaking API changes
- Gradual migration path
- Mutex fallback for safety

## Risk Mitigation

### ABA Problem
- Use pointer comparison in CAS
- Each state update creates new object
- No pointer reuse

### Memory Ordering
- Use atomic package correctly
- Document memory barriers
- Test on multiple architectures

### Backward Compatibility
- Keep mutex-based fields
- Dual-write for transitions
- Atomic as optimization only

## Next Phase Preview

After Phase 3A is complete and validated:
- **Phase 3B**: Migrate process registry to sync.Map
- **Phase 3C**: End-to-end optimization and rollout