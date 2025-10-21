# Footgun Analysis - Places Developers Might Violate Lock-Free Pattern

## Overview

This document identifies locations where developers might be tempted to add mutexes, blocking operations, or other patterns that violate the lock-free immutable architecture.

## Critical Footgun Locations

### 1. State Struct Definitions (HIGH RISK)

**Location**: `internal/tui/state/*_state.go`
**Temptation**: "I need thread-safety, let me add a mutex!"

**Wrong Pattern** (What developers might do):
```go
type ProcessesState struct {
    Processes []*Process
    mu        sync.RWMutex  // ❌ DON'T ADD THIS!
}
```

**Why Tempting**:
- Coming from traditional Go patterns
- Looks like "proper" thread-safety
- Works in other contexts

**Why Deadly**:
- Creates immutable-update-with-mutable-lock deadlock
- Update() locks OLD object, View() waits forever
- Breaks keyboard navigation

**Prevention**: Add warning comment to struct definition

---

### 2. ViewWithState() Methods (HIGH RISK)

**Location**: `internal/tui/components/*_view.go`
**Temptation**: "Let me fetch fresh data directly from the manager!"

**Wrong Pattern**:
```go
func (c *Component) ViewWithState(masterData *state.MasterData, processMgr *processes.Manager) string {
    // ❌ DON'T DO THIS - blocks View()!
    processes := processMgr.GetAllProcesses()
    return c.render(processes)
}
```

**Why Tempting**:
- Seems more "correct" to get latest data
- Cache might be stale
- Direct access looks simpler

**Why Deadly**:
- GetAllProcesses() can block (sync.Map iteration, locks, etc.)
- Blocks BubbleTea event loop
- Freezes keyboard input

**Prevention**: Add warning comment at method entry

---

### 3. State Update Methods (MEDIUM RISK)

**Location**: `internal/tui/state/*_state.go` update methods
**Temptation**: "Let me just modify this field in place!"

**Wrong Pattern**:
```go
func (s *ProcessesState) UpdateProcesses(processes []*Process) *ProcessesState {
    // ❌ DON'T DO THIS - mutates state!
    s.Processes = processes
    s.version++
    return s
}
```

**Why Tempting**:
- Looks more efficient
- Avoids creating new object
- Common pattern in other contexts

**Why Wrong**:
- Breaks immutability guarantee
- Old references now see new data (race condition)
- View() might read mid-modification

**Prevention**: Add comment explaining immutability requirement

---

### 4. Model.Update() Direct State Modification (MEDIUM RISK)

**Location**: `internal/tui/model.go` Update() method
**Temptation**: "Let me just update this field directly!"

**Wrong Pattern**:
```go
case ProcessStartedMsg:
    // ❌ DON'T DO THIS - violates Redux/Elm
    m.processMgr.AddProcess(msg.Process)
    m.refreshView()
```

**Why Tempting**:
- Direct modification seems simpler
- Fewer lines of code
- Immediate effect

**Why Wrong**:
- Violates Redux/Elm pattern (only Update() should modify state)
- Makes state changes unpredictable
- Can't track state history
- Breaks time-travel debugging

**Prevention**: Add comment explaining proper message/command pattern

---

### 5. Service Methods Storing State (HIGH RISK)

**Location**: `internal/tui/services/*.go`
**Temptation**: "Let me cache this in the service!"

**Wrong Pattern**:
```go
type ProcessService struct {
    cachedProcesses []*Process  // ❌ DON'T CACHE IN SERVICES!
    mu              sync.Mutex
}

func (s *ProcessService) GetProcesses() []*Process {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.cachedProcesses
}
```

**Why Tempting**:
- Seems like good performance optimization
- Common pattern in service-oriented architectures
- Looks like "proper" caching

**Why Wrong**:
- Services should be stateless (Redux action creators)
- Creates multiple sources of truth
- Breaks Redux/Elm pattern
- Introduces mutexes unnecessarily

**Prevention**: Add comment to service struct definition

---

### 6. Getter Methods with Locks (MEDIUM RISK)

**Location**: `internal/tui/state/*_state.go` getter methods
**Temptation**: "Let me add a lock for thread-safety!"

**Wrong Pattern**:
```go
func (s *ProcessesState) GetProcesses() []*Process {
    s.mu.RLock()  // ❌ DON'T ADD LOCKS!
    defer s.mu.RUnlock()
    return s.Processes
}
```

**Why Tempting**:
- Looks like "proper" concurrent programming
- Standard pattern in mutex-based code
- Feels safer

**Why Wrong**:
- Immutable state doesn't need locks
- Creates deadlock opportunity
- Adds unnecessary overhead
- Violates lock-free principle

**Prevention**: Add comment explaining immutability provides safety

---

### 7. Update() with Blocking Operations (MEDIUM RISK)

**Location**: `internal/tui/model.go` Update() method
**Temptation**: "Let me do the work right here in Update()!"

**Wrong Pattern**:
```go
case FetchDataMsg:
    // ❌ DON'T BLOCK IN UPDATE!
    data := m.api.FetchData()  // Blocks!
    m.data = data
    return m, nil
```

**Why Tempting**:
- Looks simpler/more direct
- Fewer abstractions
- Immediate execution

**Why Wrong**:
- Blocks BubbleTea event loop
- Freezes UI during network/disk I/O
- Prevents other messages from processing

**Prevention**: Add comment explaining command pattern

---

### 8. Init() with Long-Running Operations (LOW RISK)

**Location**: Component Init() methods
**Temptation**: "Let me initialize everything here!"

**Wrong Pattern**:
```go
func (c *Component) Init() tea.Cmd {
    // ❌ DON'T BLOCK IN INIT!
    c.loadConfiguration()  // Reads from disk
    c.connectToDatabase()  // Network call
    return nil
}
```

**Why Tempting**:
- Init() feels like right place for initialization
- Want everything ready before first render
- Common pattern in other frameworks

**Why Wrong**:
- Blocks application startup
- Init() should return quickly
- Long operations should be commands

**Prevention**: Add comment to template Init() methods

---

### 9. Creating Goroutines in Update() (HIGH RISK)

**Location**: `internal/tui/model.go` Update() method
**Temptation**: "Let me spawn a goroutine for async work!"

**Wrong Pattern**:
```go
case ProcessDataMsg:
    // ❌ DON'T SPAWN GOROUTINES IN UPDATE!
    go func() {
        result := processData(msg.Data)
        // How do I send this back???
    }()
    return m, nil
```

**Why Tempting**:
- Common Go pattern for async work
- Seems like right way to do background work
- Prevents blocking

**Why Wrong**:
- Can't communicate back to Update() safely
- Violates BubbleTea/Elm architecture
- Creates race conditions
- Proper way: return tea.Cmd

**Prevention**: Add comment explaining command pattern

---

### 10. Storing Mutable References in State (MEDIUM RISK)

**Location**: State struct fields
**Temptation**: "Let me store this channel/map/pointer!"

**Wrong Pattern**:
```go
type ProcessesState struct {
    Processes   []*Process
    UpdateChan  chan Update     // ❌ MUTABLE!
    ProcessMap  map[string]*Process  // ❌ MUTABLE!
}
```

**Why Tempting**:
- Channels seem like good way to communicate
- Maps are efficient
- Pointers avoid copying

**Why Wrong**:
- Channels/maps are mutable
- Breaks immutability guarantee
- Can be modified from multiple goroutines
- Creates race conditions

**Prevention**: Add comment explaining immutable-only fields

---

## Summary of Footguns

| Location | Risk Level | Main Temptation | Why Deadly |
|----------|------------|-----------------|------------|
| State struct fields | HIGH | Add mutex for thread-safety | Deadlocks View() |
| ViewWithState() | HIGH | Fetch fresh data directly | Blocks event loop |
| Service state | HIGH | Cache data in service | Multiple sources of truth |
| Update goroutines | HIGH | Spawn goroutine for async | Can't communicate back |
| State update methods | MEDIUM | Modify in place | Breaks immutability |
| Model.Update() | MEDIUM | Direct state modification | Violates Redux/Elm |
| Getter methods | MEDIUM | Add locks | Unnecessary overhead |
| Update() blocking | MEDIUM | Do work directly | Freezes UI |
| Init() blocking | LOW | Load everything upfront | Slow startup |
| Mutable fields | MEDIUM | Store channels/maps | Race conditions |

## Prevention Strategy

For each location, add comments that:
1. **Explain what NOT to do** (with ❌ marker)
2. **Explain WHY it's wrong** (consequences)
3. **Show the CORRECT pattern** (with ✅ marker)
4. **Reference documentation** (for more details)

Next: Add these preventive comments to the codebase.
