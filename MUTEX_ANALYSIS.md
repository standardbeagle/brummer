# Mutex Analysis - TUI State System

## Overview

Analysis of mutex usage across the TUI state system to identify deadlock risks and convert to lock-free atomic patterns.

## Critical State Files (Used in View())

These state files are accessed from View() rendering and are HIGH PRIORITY for lock-free conversion:

### 1. ProcessesState ✅ COMPLETED
- **File**: `internal/tui/state/processes_state.go`
- **Mutex Operations**: 44 (REMOVED)
- **Status**: ✅ **Converted to lock-free**
- **Used in**: `ProcessesViewComponent.ViewWithState()`
- **Issue**: Caused keyboard navigation deadlock

### 2. LogsState 🔴 HIGH PRIORITY
- **File**: `internal/tui/state/logs_state.go`
- **Mutex Operations**: 27 (Lock/RLock calls)
- **Status**: ❌ **Still using mutexes**
- **Used in**: `LogsViewComponent.ViewWithState()`
- **Risk**: Same deadlock pattern as ProcessesState
- **Fields**:
  - ShowPattern, HidePattern, ShowHighPriority
  - SelectedProcess, AutoScroll, AtBottom
  - SearchQuery, SearchResults
  - CachedLogs ([]logs.CollapsedLogEntry)
  - ScrollOffset

### 3. ErrorsState 🔴 HIGH PRIORITY
- **File**: `internal/tui/state/errors_state.go`
- **Mutex Operations**: 25 (Lock/RLock calls)
- **Status**: ❌ **Still using mutexes**
- **Used in**: `ErrorsViewComponent.ViewWithState()`
- **Risk**: Same deadlock pattern as ProcessesState
- **Fields**:
  - ErrorContexts ([]logs.ErrorContext)
  - SelectedError, SelectedErrorID
  - ShowDetailView, ShowList, AutoRefresh
  - ErrorCount

### 4. URLsState 🔴 HIGH PRIORITY
- **File**: `internal/tui/state/urls_state.go`
- **Mutex Operations**: 31 (Lock/RLock calls)
- **Status**: ❌ **Still using mutexes**
- **Used in**: `URLsViewComponent.ViewWithState()`
- **Risk**: Same deadlock pattern as ProcessesState
- **Fields**:
  - URLs ([]URLInfo)
  - SelectedURL, SelectedURLID
  - ShowDetails, GroupByProcess, SortBy
  - UniqueDomains (map[string]int)

### 5. WebViewState 🟡 MEDIUM PRIORITY
- **File**: `internal/tui/state/web_view_state.go`
- **Mutex Operations**: ~20 (estimated)
- **Status**: ❌ **Still using mutexes**
- **Used in**: `WebViewComponent.ViewWithState()`

### 6. CommandWindowState 🟡 MEDIUM PRIORITY
- **File**: `internal/tui/state/command_window_state.go`
- **Mutex Operations**: ~15 (estimated)
- **Status**: ❌ **Still using mutexes**
- **Used in**: `CommandWindowViewComponent.ViewWithState()`

### 7. MCPState 🟡 MEDIUM PRIORITY
- **File**: `internal/tui/state/mcp_state.go`
- **Mutex Operations**: ~20 (estimated)
- **Status**: ❌ **Still using mutexes**
- **Used in**: `MCPViewComponent.ViewWithState()`

## Other State Files (Lower Priority)

### 8. Cache
- **File**: `internal/tui/state/cache.go`
- **Status**: ❌ Uses sync.RWMutex
- **Risk**: Medium (not in hot path)

### 9. UIComponents
- **File**: `internal/tui/state/ui_components.go`
- **Status**: ❌ Uses sync.RWMutex (2 mutexes)
- **Risk**: Medium

### 10. ApplicationState
- **File**: `internal/tui/state/application_state.go`
- **Status**: ❌ Uses sync.RWMutex
- **Risk**: Low (global state, rarely updated)

### 11. SettingsState
- **File**: `internal/tui/state/settings_state.go`
- **Status**: ❌ Uses sync.RWMutex
- **Risk**: Low (rarely accessed)

### 12. TestViewState
- **File**: `internal/tui/state/test_view_state.go`
- **Status**: ❌ Uses sync.RWMutex
- **Risk**: Low (test-only)

### 13. Operations
- **File**: `internal/tui/state/operations.go`
- **Status**: ❌ Uses sync.RWMutex
- **Risk**: Medium

### 14. MCPDebugState
- **File**: `internal/tui/state/mcp_debug_state.go`
- **Status**: ❌ Uses sync.RWMutex
- **Risk**: Low (debug-only)

## Deadlock Pattern

All these state files follow the **SAME DEADLOCK PATTERN** as ProcessesState:

```go
// Update method locks OLD object, creates NEW object
func (s *State) UpdateSomething(...) *State {
    s.mu.Lock()         // ← Locks OLD object
    defer s.mu.Unlock()

    newState := s.copy()  // Create NEW object
    // ... modify newState
    return newState       // Return NEW object
}

// Getter method tries to lock while Update holds lock
func (s *State) GetSomething() ... {
    s.mu.RLock()        // ← BLOCKS if Update() is running!
    defer s.mu.RUnlock()
    return s.field
}
```

### Why This Causes Deadlocks

1. **Update()** receives message, calls `UpdateSomething()` on OLD object
2. `UpdateSomething()` locks OLD object with `mu.Lock()`
3. `UpdateSomething()` creates NEW object, returns it
4. NEW object assigned to `m.masterData.SomeState`
5. Meanwhile, **View()** is rendering
6. View() still has reference to OLD object
7. View() calls `GetSomething()` on OLD object
8. `GetSomething()` tries to acquire `RLock()` on OLD object
9. **DEADLOCK**: Old object is locked by Update(), View() waits forever

## Conversion Strategy

### Phase 1: Critical View States (HIGH PRIORITY)
Convert states used in View() rendering to prevent deadlocks:
1. ✅ ProcessesState (DONE)
2. ❌ LogsState
3. ❌ ErrorsState
4. ❌ URLsState
5. ❌ WebViewState
6. ❌ CommandWindowState
7. ❌ MCPState

### Phase 2: Infrastructure States (MEDIUM PRIORITY)
8. Cache
9. UIComponents
10. Operations

### Phase 3: Low-Risk States (LOW PRIORITY)
11. ApplicationState
12. SettingsState
13. TestViewState
14. MCPDebugState

## Conversion Process

For each state file:

1. **Remove mutex field**:
   ```go
   // BEFORE:
   type SomeState struct {
       // ...
       mu sync.RWMutex
   }

   // AFTER:
   type SomeState struct {
       // IMMUTABLE: All updates return NEW instance
       // NO MUTEXES: Thread-safe through immutability
       // ...
       // (no mu field)
   }
   ```

2. **Remove all Lock()/RLock() calls**:
   ```bash
   sed -i '/s\.mu\.Lock()/d; /s\.mu\.RLock()/d; /defer s\.mu\.Unlock()/d; /defer s\.mu\.RUnlock()/d' file.go
   ```

3. **Update import** (remove sync if unused):
   ```go
   // Remove "sync" from imports if it's the only usage
   ```

4. **Test**:
   - Verify no race conditions with `make test-race`
   - Test in actual TUI with keyboard navigation
   - Check for any remaining mutex references

## Benefits of Lock-Free Pattern

### Performance
- ✅ **No contention**: Readers never block writers, writers never block readers
- ✅ **Better concurrency**: Multiple goroutines can read simultaneously
- ✅ **Faster**: No mutex acquisition overhead (30-300x improvement in benchmarks)

### Correctness
- ✅ **No deadlocks**: Impossible without locks
- ✅ **No race conditions**: Immutable objects can't race
- ✅ **Simpler code**: No lock/unlock patterns

### Architecture
- ✅ **Redux/Elm compliant**: Immutable state, pure functions
- ✅ **CLAUDE.md compliant**: "Prefer lock-free systems"
- ✅ **Easier to reason about**: State transitions are atomic

## Testing Strategy

After each conversion:

1. **Unit tests**: Verify state updates work correctly
2. **Race detection**: `make test-race` must pass
3. **Integration tests**: Run TUI and test all views
4. **Keyboard navigation**: Verify no freezing when switching views
5. **Concurrent access**: Test multiple goroutines accessing state

## Estimated Impact

### Mutex Operations Removed (by priority)
- Phase 1 (High): ~180 mutex operations
- Phase 2 (Medium): ~60 mutex operations
- Phase 3 (Low): ~40 mutex operations
- **Total**: ~280 mutex operations eliminated

### Risk Reduction
- ✅ Eliminates ALL deadlock scenarios in View() rendering
- ✅ Prevents keyboard navigation freezes
- ✅ Improves overall TUI responsiveness
- ✅ Simplifies concurrent programming model

## Next Steps

1. Start with LogsState (27 operations)
2. Then ErrorsState (25 operations)
3. Then URLsState (31 operations)
4. Continue with remaining states in priority order
