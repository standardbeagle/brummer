# Todo: Consolidated Task Plan - Critical Issues & Unfinished Work

**Generated from**: Full Planning on 2025-07-15  
**Analysis Date**: 2025-07-15  
**Risk Level**: CRITICAL | **Project Phase**: Production Stability  
**Estimated Effort**: 16-24 hours total | **Files**: 15+ files affected
**Feature Flag Required**: No (internal fixes and infrastructure improvements)

## Context & Background

**Request**: Complete all outstanding critical tasks, fix TUI race conditions introduced in recent changes, finish race condition elimination work, and establish proper CI/CD integration.

**Business Impact**: CRITICAL - Current TUI changes introduce new race conditions that compromise system stability while previous race condition work is 90% complete but not finished.

**Technical Debt**: Multiple critical issues requiring immediate attention:
1. **CRITICAL**: New TUI race conditions from hard-coded sleep patterns
2. **HIGH**: Incomplete race condition elimination (tasks 19-20)  
3. **MEDIUM**: Unarchived work creating repository clutter
4. **LOW**: Missing CI/CD infrastructure for race detection

## Codebase Context

### Current Critical Issues in TUI (From Code Review)
**Newly Introduced Race Conditions**:
- ❌ **CRITICAL**: Concurrent TUI state modification in `internal/tui/model.go:3513-3516, 4143-4146`
  - `m.selectedProcess = newProc.ID` in goroutines without synchronization
  - `m.currentView = ViewLogs` concurrent modifications
  - `m.updateLogsView()` called from multiple goroutines

- ❌ **CRITICAL**: Hard-coded sleep patterns in production code `internal/tui/model.go`
  - Lines 3496: `time.Sleep(50 * time.Millisecond)` polling pattern
  - Lines 3503: `time.Sleep(200 * time.Millisecond)` arbitrary delay
  - Lines 3545: `time.Sleep(500 * time.Millisecond)` batch processing delay
  - Lines 4174, 4219: `time.Sleep(300 * time.Millisecond)` restart delays

- ❌ **HIGH**: Blocking operations in UI thread context
  - Up to 2.2 seconds of sleep operations blocking user interface
  - Non-deterministic behavior under load or slow systems

### Outstanding Race Condition Tasks (90% Complete)
**Existing Functionality**:
- ✅ **Tasks 1-18 COMPLETED**: Core race condition elimination done - Files: All major components
- ❌ **Task 19**: CI/CD Race Detection Integration - Location: `.github/workflows/build.yml`
- ❌ **Task 20**: Documentation and Prevention Guidelines - Location: `docs/`, `CLAUDE.md`

### Archival Requirements
**Completed Work Needing Archive**:
- ⚠️ **Race condition work**: `todo.md` 90% complete, needs proper closure - Files: Multiple todo files
- ⚠️ **Network improvements**: `todo-network-robustness-improvements.md` active status unknown
- ⚠️ **MCP migration**: `todo-mcp-go-sdk-migration.md` status unknown
- ❌ **72 uncommitted files**: Major changes not properly committed and organized

## External Context Sources

### Primary Documentation
- [Go Race Detector](https://go.dev/doc/articles/race_detector.html) - CI integration: `go test -race`, 5-10x memory overhead, requires cgo
- [Go Memory Model](https://go.dev/ref/mem) - Channel-based coordination vs mutex patterns, happens-before relationships
- [BubbleTea Architecture](https://github.com/charmbracelet/bubbletea) - Elm Architecture, unidirectional data flow, single-threaded update semantics

### Synchronization Patterns Research
**Event-Driven Coordination** (Recommended for TUI):
```go
// Channel-based process event coordination
type ProcessEvent struct {
    Type      string
    ProcessID string
    NewState  ProcessStatus
}

// Replace sleep polling with event subscription
func (m *Model) waitForProcessCleanup(processID string) <-chan ProcessEvent {
    eventChan := make(chan ProcessEvent, 1)
    m.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
        if e.ProcessID == processID {
            eventChan <- ProcessEvent{
                Type:      "ProcessExited",
                ProcessID: processID,
                NewState:  StatusStopped,
            }
        }
    })
    return eventChan
}
```

**TUI State Synchronization** (Thread-Safe Updates):
```go
// Channel-based TUI state updates
type TUIStateUpdate struct {
    SelectedProcess string
    CurrentView     View
    UpdateLogs      bool
}

func (m *Model) updateTUIState(update TUIStateUpdate) tea.Cmd {
    return func() tea.Msg {
        return tuiStateUpdateMsg{update}
    }
}
```

### CI/CD Integration Patterns
**GitHub Actions Race Detection**:
```yaml
- name: Run race detector tests
  run: go test -race -timeout=10m ./...
  env:
    CGO_ENABLED: 1  # Required for race detector
```

## Implementation Plan

### Phase 1: Critical TUI Race Condition Fixes (Risk: CRITICAL)
**Files**: `internal/tui/model.go`  
**Objective**: Eliminate newly introduced race conditions and blocking operations  
**Validation**: `go test -race -v ./internal/tui/...` passes without race warnings

- [ ] **Task 1**: Replace Sleep-Based Polling with Event-Driven Synchronization
  - **Risk**: CRITICAL - System stability, non-deterministic behavior
  - **Files**: `internal/tui/model.go:3493-3503, 3545, 4174, 4219`
  - **Research**: Use existing EventBus process events instead of sleep polling
  - **Pattern**: Subscribe to `events.ProcessExited` and use channels for coordination
  - **Success Criteria**:
    - [ ] Remove all `time.Sleep` calls from restart functions
    - [ ] Implement event-based process cleanup detection
    - [ ] Test with `go test -race -v ./internal/tui/...` - no race warnings
    - [ ] Functional test: restart operations complete reliably
  - **Implementation**:
    ```go
    // Replace sleep polling with event subscription
    func (m *Model) waitForProcessExit(processID string) <-chan struct{} {
        done := make(chan struct{}, 1)
        m.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
            if e.ProcessID == processID {
                select {
                case done <- struct{}{}:
                default:
                }
            }
        })
        return done
    }
    ```

- [ ] **Task 2**: Fix Concurrent TUI State Modifications
  - **Risk**: CRITICAL - Race conditions, data corruption, UI inconsistency
  - **Files**: `internal/tui/model.go:3513-3516, 4143-4146, 4228-4229`
  - **Research**: Use channel-based state updates following BubbleTea patterns
  - **Pattern**: State changes through tea.Cmd messages, not direct goroutine modification
  - **Success Criteria**:
    - [ ] Remove direct TUI state modifications from goroutines
    - [ ] Implement message-based state updates
    - [ ] Test with `go test -race -v ./internal/tui/...` - no race warnings
    - [ ] UI behavior remains consistent and responsive
  - **Implementation**:
    ```go
    type tuiStateUpdateMsg struct {
        selectedProcess string
        currentView     View
        updateLogs      bool
    }
    
    func (m *Model) updateTUIState(selectedProcess string, view View) tea.Cmd {
        return func() tea.Msg {
            return tuiStateUpdateMsg{
                selectedProcess: selectedProcess,
                currentView:     view, 
                updateLogs:      true,
            }
        }
    }
    ```

- [ ] **Task 3**: Implement Non-Blocking Restart Operations
  - **Risk**: HIGH - UI responsiveness, user experience
  - **Files**: `internal/tui/model.go` restart handler functions
  - **Research**: Async operations with progress feedback, timeout handling
  - **Pattern**: Use context.WithTimeout for bounded waiting
  - **Success Criteria**:
    - [ ] Restart operations don't block UI thread
    - [ ] Progress feedback for long operations
    - [ ] Timeout handling with user notification
    - [ ] UI remains responsive during restart operations
  - **Implementation**:
    ```go
    func (m *Model) handleRestartProcessAsync(proc *process.Process) tea.Cmd {
        return func() tea.Msg {
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()
            
            // Async restart with timeout
            done := m.waitForProcessExit(proc.ID)
            select {
            case <-done:
                // Process cleaned up, safe to restart
            case <-ctx.Done():
                return restartTimeoutMsg{proc.Name}
            }
            
            newProc, err := m.processMgr.StartScript(proc.Name)
            return restartCompleteMsg{proc: newProc, err: err}
        }
    }
    ```

### Phase 2: Complete Outstanding Race Condition Work (Risk: LOW)
**Files**: `.github/workflows/build.yml`, `docs/`, `CLAUDE.md`  
**Objective**: Finish tasks 19-20 from race condition elimination project  
**Validation**: CI pipeline includes race detection, documentation complete

- [ ] **Task 4**: CI/CD Race Detection Integration (Task 19)
  - **Risk**: LOW - Process improvement, no functional risk
  - **Files**: `.github/workflows/build.yml`, `Makefile`
  - **Research**: GitHub Actions with CGO_ENABLED=1, race detector requirements
  - **Success Criteria**:
    - [ ] Race detector integrated into CI: `go test -race` in pipeline
    - [ ] CGO enabled for race detection support
    - [ ] Build matrix includes race testing on Linux/macOS
    - [ ] Pipeline fails on race condition detection
  - **Implementation**:
    ```yaml
    - name: Test with race detector
      run: |
        export CGO_ENABLED=1
        go test -race -timeout=10m ./...
    ```

- [ ] **Task 5**: Documentation and Prevention Guidelines (Task 20)  
  - **Risk**: LOW - Documentation only
  - **Files**: `CLAUDE.md`, `docs/concurrency-patterns.md`, `docs/code-review-checklist.md`
  - **Research**: Go concurrency best practices, race condition prevention
  - **Success Criteria**:
    - [ ] Concurrency patterns documented with examples
    - [ ] Code review checklist includes race condition checks
    - [ ] Updated architecture documentation
    - [ ] Developer guidelines for safe concurrent programming
  - **Implementation**: Document the event-driven patterns, mutex hierarchies, and testing requirements

### Phase 3: Repository Organization and Archival (Risk: LOW)
**Files**: Archive directories, todo file organization  
**Objective**: Clean up completed work and organize active tasks  
**Validation**: Repository in clean state with proper archival structure

- [ ] **Task 6**: Archive Completed Race Condition Work
  - **Risk**: LOW - Organization only
  - **Files**: `todo.md`, `requests/race-condition-fixes/`, archive structure
  - **Success Criteria**:
    - [ ] Create `archive/2025-07-15-race-condition-fixes/` directory
    - [ ] Move completed todo.md to archive with completion report
    - [ ] Document completion status and final results
    - [ ] Update project status documentation

- [ ] **Task 7**: Consolidate and Organize Active Tasks
  - **Risk**: LOW - Organization only  
  - **Files**: Multiple todo files, task organization
  - **Success Criteria**:
    - [ ] Single active todo file for current work
    - [ ] Clear status on network improvements and MCP migration
    - [ ] Proper prioritization of remaining tasks
    - [ ] Clean git status with committed changes

## Gotchas & Considerations

### Technical Challenges
- **BubbleTea Constraints**: Model must follow single-threaded update patterns
- **Event Subscription**: Avoid memory leaks with proper unsubscribe patterns
- **Performance Impact**: Race detector has 5-10x memory overhead in CI
- **CGO Dependency**: Race detector requires C compiler in CI environment

### Edge Cases to Test
- **Rapid restart operations**: Multiple restart commands in quick succession
- **Process exit timing**: Process exits during restart waiting period
- **UI responsiveness**: Heavy process operations don't block interface
- **Error propagation**: Network issues or process failures handled gracefully

### Backwards Compatibility
- **TUI behavior**: Restart functionality maintains same user experience
- **API compatibility**: No breaking changes to public interfaces
- **Configuration**: Existing settings and preferences preserved

## Validation Strategy

### Testing Commands
```bash
# Critical race condition testing
go test -race -v ./internal/tui/... 

# Full system race detection
go test -race -timeout=10m ./...

# Performance regression testing  
go test -bench=. ./internal/tui/...

# Integration testing
make test

# Build verification
make build
```

### Success Metrics
- **Zero race conditions**: `go test -race` clean across all components
- **UI responsiveness**: No blocking operations > 100ms
- **Functional correctness**: All restart operations work reliably
- **CI integration**: Automated race detection prevents regression
- **Documentation completeness**: Code review checklist prevents future issues

## Definition of Done

- [ ] **TUI race conditions eliminated**: No concurrent state modifications or sleep polling
- [ ] **Tests pass with race detection**: `go test -race -v ./...` clean
- [ ] **CI/CD integration complete**: Race detection in build pipeline
- [ ] **Documentation updated**: Concurrency patterns and guidelines documented  
- [ ] **Repository organized**: Completed work archived, active tasks consolidated
- [ ] **Performance maintained**: No regression in restart functionality
- [ ] **Code review passed**: All changes follow established patterns

## Priority and Dependencies

**Critical Path** (Must complete in order):
1. Task 1-3: TUI race condition fixes (blocks all other work)
2. Task 4-5: Complete race condition project (finishes 90% complete work)
3. Task 6-7: Archive and organize (cleanup for future work)

**Estimated Timeline**:
- Phase 1: 8-12 hours (TUI fixes)
- Phase 2: 4-6 hours (CI/docs)
- Phase 3: 2-4 hours (archival)
- **Total**: 14-22 hours over 2-3 days

**Next Phase**: [tasks-execute.md](tasks-execute.md) - Begin with Task 1 (highest risk)