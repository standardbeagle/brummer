# Research: Complete Tab Functionality with Automated Testing

**Date**: 2025-10-19
**Feature**: Complete all 7 TUI tabs with error grouping and automated testing

## Research Areas

### 1. Error Grouping Algorithms

**Decision**: Pattern-based multi-line error grouping with state machine parser

**Rationale**:
- **Stack Traces**: Detect error headers (`Error:`, `Exception:`, etc.) and group all subsequent lines starting with whitespace or stack trace patterns (`at `, `in `, file paths with line numbers)
- **Compilation Errors**: Group TypeScript/JavaScript compilation errors by detecting error patterns (`TS####:`, `SyntaxError:`) and collecting subsequent context lines
- **Test Failures**: Detect test framework patterns (Jest, Vitest, etc.) and group test name + expected/received + stack trace
- **Multi-line Exceptions**: Use indentation and continuation patterns to group exception messages

**Implementation Approach**:
```
1. Stream-based parser that maintains "current error group" state
2. For each log line:
   - If matches error start pattern → begin new group
   - If matches continuation pattern → append to current group
   - If matches end pattern or new error → finalize group
3. Store grouped errors as single entries with preserved line breaks
```

**Alternatives Considered**:
- **Regex-only matching**: Rejected - too brittle for varied error formats
- **AI-based grouping**: Rejected - too slow, adds dependency
- **Fixed line count grouping**: Rejected - misses multi-line errors or groups unrelated lines

### 2. Lock-Free State Updates for Tab Content

**Decision**: Use existing immutable state pattern with atomic pointer swaps

**Rationale**:
- Architecture already proven in spec 009-make-the-application
- Zero mutexes = zero deadlocks
- Atomic.Value for state pointers ensures thread-safe reads
- Copy-on-write pattern for all state mutations

**Pattern**:
```go
// In ErrorsState
func (s *ErrorsState) AddGroupedError(groupedError GroupedError) *ErrorsState {
    newState := s.copy()
    newState.Errors = append(newState.Errors, groupedError)
    newState.version++
    return newState
}

// In Model.Update()
case ErrorGroupedMsg:
    newErrorsState := m.masterData.ErrorsState.AddGroupedError(msg.Error)
    m.masterData.ErrorsState = newErrorsState
    return m, nil
```

**Alternatives Considered**:
- **Mutex-protected state**: Rejected - causes deadlocks (proven in spec 009)
- **Channel-based updates**: Rejected - adds complexity, not needed for single-threaded Update()

### 3. Testing Strategy for TUI Components

**Decision**: Three-tier testing pyramid: Contract Tests → Integration Tests → Unit Tests

**Rationale**:
- **Contract Tests** (60% of tests): Verify state behavior follows Redux/Elm contracts
  - State immutability
  - View() pure functions (no side effects)
  - Update() deterministic returns
  - Service statelessness
- **Integration Tests** (30%): End-to-end keyboard flows with TTY allocation
  - Tab switching preserves state
  - Clear commands work correctly
  - Error grouping in real scenarios
- **Unit Tests** (10%): Pure function testing
  - Error grouping parser logic
  - State mutation correctness
  - View rendering edge cases

**Test Execution**:
```bash
# Contract tests (fast, run always)
go test ./test/tui/contracts/... -v

# Integration tests (slower, require TTY)
BRUMMER_BINARY=./brum go test -tags=integration ./test/tui/integration/... -v

# Race detection (critical for concurrent code)
make test-race
```

**Alternatives Considered**:
- **E2E tests only**: Rejected - too slow, hard to debug failures
- **Unit tests only**: Rejected - miss integration issues
- **Snapshot testing**: Rejected - brittle for TUI output

### 4. Clear Command Implementation

**Decision**: Context-aware 'c' key with confirmation for 'C' (shift+c)

**Rationale**:
- Single key 'c' for quick clearing (common operation)
- Shift+c requires confirmation (safety for accidental clears)
- Context-aware: only active in Errors/Logs/Web tabs
- Implemented as message → Update() → new state pattern

**Implementation**:
```go
// In key_handler.go
case 'c':
    switch currentTab {
    case ErrorsTab:
        return tea.Cmd(func() tea.Msg { return ClearErrorsMsg{} })
    case LogsTab:
        return tea.Cmd(func() tea.Msg { return ClearLogsMsg{} })
    case WebTab:
        return tea.Cmd(func() tea.Msg { return ClearWebMsg{} })
    }
case 'C':  // shift+c
    return tea.Cmd(func() tea.Msg { return ShowClearConfirmationMsg{Tab: currentTab} })
```

**Alternatives Considered**:
- **Slash command only** (`/clear`): Rejected - too slow for common operation
- **Ctrl+k shortcut**: Rejected - conflicts with terminal shortcuts
- **No confirmation**: Rejected - too easy to accidentally lose data

### 5. Web Tab vs Errors Tab Content Separation

**Decision**: Errors = process stderr, Web = proxy HTTP events (already defined in spec)

**Rationale**: See spec.md "Detailed Design: Errors vs Web Tabs" section
- **Errors Tab**: Process output errors, stack traces, compilation errors - unstructured text
- **Web Tab**: HTTP request/response data from proxy - structured HTTP events
- Clear separation prevents confusion and allows specialized filtering

**Implementation Notes**:
- Errors: Parse from process stdout/stderr streams
- Web: Capture from proxy server's HTTP interception layer
- Both support 'c' key clearing but with different state objects

### 6. Test Coverage Tooling

**Decision**: Go's built-in coverage tools with >80% threshold

**Rationale**:
```bash
# Generate coverage report
go test ./internal/tui/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Enforce coverage threshold in CI
go test ./internal/tui/... -cover | grep -E "coverage: [0-9]+\.[0-9]+%" | \
  awk '{if ($2 < 80.0) {print "Coverage below 80%"; exit 1}}'
```

**Coverage Focus**:
- State mutation methods: 100% (critical for correctness)
- View rendering: 90% (visual variations acceptable)
- Services: 100% (business logic critical)
- Handlers: 80% (some edge cases acceptable)

**Alternatives Considered**:
- **External coverage tools**: Rejected - Go's built-in tools sufficient
- **Lower threshold**: Rejected - spec requires >80%
- **100% coverage requirement**: Rejected - diminishing returns on edge cases

## Unknown Resolutions

✅ All technical context items were already known (no NEEDS CLARIFICATION markers)

## Dependencies Analysis

### Existing Dependencies (No Changes Needed)
- **BubbleTea**: v0.25.0 - TUI framework (already integrated)
- **Lipgloss**: v0.9.1 - Terminal styling (already integrated)
- **atomic**: Go stdlib - Lock-free operations (already used)

### New Dependencies Required
- **None** - All functionality can be implemented with existing dependencies

## Architecture Verification

✅ **Redux/Elm Pattern**: Confirmed all tabs follow Model → Update → View unidirectional flow
✅ **Lock-Free State**: Confirmed all state objects are immutable with atomic operations
✅ **Stateless Services**: Confirmed services return tea.Cmd, no internal state
✅ **Test-First**: Confirmed TDD approach with contract tests before implementation
✅ **Performance**: Confirmed <5ms View(), <50ms keyboard response achievable

## Best Practices Applied

1. **Error Grouping**:
   - Use state machine parser (proven pattern for text processing)
   - Preserve original formatting (maintain line breaks, indentation)
   - Fail gracefully (ungroupable errors shown as single lines)

2. **State Management**:
   - Follow existing immutable state pattern from spec 009
   - Use atomic.Value for thread-safe state access
   - Version tracking for state change detection

3. **Testing**:
   - Write contract tests first (TDD)
   - Use table-driven tests for error grouping variations
   - Mock external dependencies (process manager, proxy server)

4. **Performance**:
   - Lazy load tab content (only render visible tab)
   - Cache grouped errors (don't reparse on every render)
   - Use efficient string building (strings.Builder, not concatenation)

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Error grouping misses edge cases | Comprehensive test suite with real-world error examples |
| Performance degradation with many errors | Implement configurable retention limits (e.g., last 1000 errors) |
| Race conditions in concurrent updates | Mandatory `make test-race` before commits |
| Breaking existing tab functionality | Contract tests verify all existing behavior preserved |
| Test flakiness in TTY tests | Use stable test fixtures, avoid timing-dependent tests |

## Next Steps

Phase 1 will generate:
1. **data-model.md**: GroupedError entity, state structures
2. **contracts/**: State behavior contracts, service contracts
3. **quickstart.md**: Manual test scenarios for each tab
4. **Failing tests**: Contract tests that fail until implementation complete

---
*Phase 0 Complete - All research concluded, no blockers*
