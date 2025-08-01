# Concurrency Patterns & Race Condition Prevention

## Critical Guidelines for Safe Concurrent Programming

### TUI Components (BubbleTea Architecture)
- ✅ **DO**: Use message-passing via tea.Cmd for all state changes
- ✅ **DO**: Handle state modifications only in Update() method  
- ❌ **DON'T**: Modify Model state directly from goroutines
- ❌ **DON'T**: Call methods like `m.logStore.Add()` from tea.Cmd functions

```go
// ✅ CORRECT: Message-based state update
func (m *Model) handleRestartProcess(proc *process.Process) tea.Cmd {
    return func() tea.Msg {
        err := m.processMgr.StopProcess(proc.ID)
        return restartProcessMsg{
            processName: proc.Name,
            message:     fmt.Sprintf("Restart result: %v", err),
            isError:     err != nil,
        }
    }
}

// ❌ INCORRECT: Direct state modification from goroutine
func (m *Model) handleBadRestart(proc *process.Process) tea.Cmd {
    return func() tea.Msg {
        m.logStore.Add("system", "System", "Restarting...", false) // RACE CONDITION
        return processUpdateMsg{}
    }
}
```

### EventBus Usage
- ✅ **DO**: Use worker pools with bounded goroutines (CPU cores × 2.5)
- ✅ **DO**: Implement graceful degradation when pools are full
- ❌ **DON'T**: Create unlimited goroutines for event handling

### Process Manager
- ✅ **DO**: Use RWMutex for concurrent map operations
- ✅ **DO**: Implement consistent lock ordering to prevent deadlocks  
- ❌ **DON'T**: Access shared maps without synchronization

### Testing Requirements
- ✅ **ALWAYS**: Run `go test -race -v ./...` before commits
- ✅ **ALWAYS**: Use `make test-race` for targeted race detection
- ✅ **CI/CD**: Race detection integrated in GitHub Actions pipeline

## Code Review Checklist

Before approving any changes involving concurrency:

### 1. TUI Changes
- [ ] All Model state changes go through Update() method
- [ ] No direct state modification in tea.Cmd goroutines
- [ ] Message types defined for complex state updates

### 2. EventBus Changes
- [ ] Worker pool limits respected (bounded goroutines)
- [ ] Graceful handling when pools are full
- [ ] Proper event handler cleanup

### 3. Shared State Access
- [ ] Mutexes used for concurrent map/slice operations
- [ ] Consistent lock ordering documented
- [ ] Read/write separation with RWMutex where applicable

### 4. Testing Validation
- [ ] `go test -race` passes on modified packages
- [ ] Integration tests include concurrent scenarios
- [ ] Performance impact assessed (< 10% regression)