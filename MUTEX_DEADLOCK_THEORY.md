# Mutex Deadlock Theory

## The Problem

ViewWithState() blocks after "setSize done, calling viewwithstate" message.

## Theory

ProcessesState uses `sync.RWMutex` for thread-safety:
- `GetProcesses()` needs `RLock` (read lock)
- `UpdateProcesses()` needs `Lock` (write lock)

### Deadlock Scenario

1. **Init()** triggers tickCmd()
2. **TickMsg** handler calls `refreshProcessesCmd()`
3. **refreshProcessesCmd()** spawns goroutine that:
   - Calls `processMgr.GetAllProcesses()`
   - Returns `ProcessesRefreshedMsg`
4. **ProcessesRefreshedMsg** handler calls:
   ```go
   m.masterData.ProcessesState = m.masterData.ProcessesState.UpdateProcesses(msg.Processes)
   ```
   This needs `Lock()` on ProcessesState

5. Meanwhile, **View()** is called (rendering happens continuously)
6. **ViewWithState()** calls:
   ```go
   processes := processesState.GetProcesses()
   ```
   This needs `RLock()` on ProcessesState

### The Deadlock

If UpdateProcesses() grabs the Lock() **while** GetProcesses() is waiting for RLock(), we deadlock!

Actually wait, that's not right. RWMutex allows multiple readers OR one writer, but:
- Multiple `RLock()` can coexist
- `Lock()` waits for all `RLock()` to finish
- New `RLock()` waits if `Lock()` is waiting

## The Real Issue

Looking at `UpdateProcesses()`:

```go
func (s *ProcessesState) UpdateProcesses(processList []*processes.Process) *ProcessesState {
    s.mu.Lock()          // ← Locks the ORIGINAL object
    defer s.mu.Unlock()

    newState := s.copy()  // ← Makes shallow copy
    newState.Processes = make([]*processes.Process, len(processList))
    copy(newState.Processes, processList)
    newState.CacheTimestamp = time.Now()
    newState.NeedsRefresh = false
    newState.incrementVersion()
    return newState       // ← Returns NEW object
}
```

Then in the message handler:
```go
m.masterData.ProcessesState = m.masterData.ProcessesState.UpdateProcesses(msg.Processes)
```

This:
1. Locks the OLD ProcessesState
2. Creates NEW ProcessesState
3. Assigns NEW ProcessesState to m.masterData.ProcessesState
4. Unlocks the OLD ProcessesState (which is now orphaned!)

Meanwhile View() might be:
1. Reading `m.masterData.ProcessesState` (gets reference to OLD object)
2. Calling `GetProcesses()` on OLD object
3. Waiting for `RLock()` on OLD object
4. OLD object's mutex is locked by `UpdateProcesses()`!

## The Fix

ProcessesState should be **immutable** and use **atomic.Value** instead of mutexes!

```go
type ProcessesState struct {
    // No mutex!
    SelectedProcess string
    Processes       []*processes.Process
    // ... rest of fields
}

// In MasterData:
type MasterData struct {
    processesState atomic.Value // stores *ProcessesState
}

// Update without locking:
func (m *MasterData) UpdateProcesses(processList []*processes.Process) {
    old := m.processesState.Load().(*ProcessesState)
    newState := &ProcessesState{
        // Copy all fields from old
        SelectedProcess: old.SelectedProcess,
        Processes: processList,
        // ... etc
    }
    m.processesState.Store(newState)
}

// Read without locking:
func (m *MasterData) GetProcessesState() *ProcessesState {
    return m.processesState.Load().(*ProcessesState)
}
```

This is lock-free and follows the atomic/immutable pattern preferred in CLAUDE.md!

## Alternative Quick Fix

Make UpdateProcesses() NOT lock the mutex, since it returns a new object anyway:

```go
func (s *ProcessesState) UpdateProcesses(processList []*processes.Process) *ProcessesState {
    // No lock needed - we're creating a new object!
    newState := &ProcessesState{
        SelectedProcess: s.SelectedProcess,
        Processes: make([]*processes.Process, len(processList)),
        CacheTimestamp: time.Now(),
        NeedsRefresh: false,
        GroupByStatus: s.GroupByStatus,
        SortByUptime: s.SortByUptime,
        ShowStopped: s.ShowStopped,
        version: s.version + 1,
    }
    copy(newState.Processes, processList)
    return newState
}
```

Then GetProcesses() also doesn't need to lock:

```go
func (s *ProcessesState) GetProcesses() []*processes.Process {
    // No lock needed - we're just reading a slice reference
    // The slice itself is immutable (never modified in place)
    return s.Processes
}
```

This makes ProcessesState effectively immutable!
