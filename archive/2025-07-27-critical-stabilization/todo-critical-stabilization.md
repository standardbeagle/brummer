# Todo: Critical Brummer Stabilization - Test Fixes & Race Condition Resolution

**Generated from**: Full Planning on July 27, 2025  
**Next Phase**: [tasks-execute.md](tasks-execute.md)

## Context Summary
- **Risk Level**: CRITICAL | **Project Phase**: Production Stability  
- **Estimated Effort**: 4-6 hours | **Files**: 8 core files affected
- **Feature Flag Required**: No (internal stability fixes)

## Context & Background
**Request**: Fix 3 failing tests blocking feature completion and resolve remaining TUI race condition  
**Analysis Date**: July 27, 2025  
**Estimated Effort**: 4-6 hours  
**Risk Level**: CRITICAL (Production stability compromised)

### Current Critical Issues Identified

#### **FAILING TESTS** (3 total - blocking merge):
1. **TestErrorParser_JavaScriptRuntimeErrors/Network_Error**: Expected NetworkError but got JavaScriptError
   - Location: `internal/logs/error_parser_test.go:291`
   - Issue: Error type classification logic changed in recent refactor
2. **TestLogCollapsing**: Third entry should have count 2, got 1
   - Location: `internal/logs/store_collapsed_test.go:59,62`
   - Issue: Log collapsing algorithm not working correctly
3. **TestLogCollapsingByProcess**: Index out of range panic
   - Location: `internal/logs/store_collapsed_test.go:86`
   - Issue: Empty slice access in collapsed log retrieval

#### **TUI RACE CONDITION** (1 critical issue):
- **time.Sleep** in production code at `internal/tui/model.go:3719`
- **Impact**: 500ms blocking operation in UI thread

### Codebase Context
**Existing Functionality**: 
- ✅ **Log error parsing works** - Files: `internal/logs/error_parser.go`
- ✅ **TUI system mostly stable** - Files: `internal/tui/model.go`
- ❌ **Test failures blocking merge** - Location: `internal/logs/` package
- ⚠️ **Race condition in TUI** - File: `internal/tui/model.go:3719`

**Similar Implementations**: 
- `internal/logs/configurable_parser_test.go:205` - Working network error test pattern
- `internal/tui/restart_integration_test.go` - Proper async TUI testing patterns
- `internal/discovery/atomic_ops.go` - Proper lock-free patterns for reference

**Dependencies**: 
- Go@1.24+ - Testing framework and race detection
- BubbleTea@0.25+ - TUI framework constraints for message-based updates
- No external dependencies for these fixes

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /internal/logs/error_parser.go           # Fix error type classification
  - /internal/logs/store.go                  # Fix log collapsing logic  
  - /internal/logs/store_collapsed_test.go   # Fix test expectations
  - /internal/logs/error_parser_test.go      # Fix test assertions
  - /internal/tui/model.go                   # Remove time.Sleep race condition

test_files:
  - /internal/logs/error_parser_test.go      # Verify error parsing fixes
  - /internal/logs/store_collapsed_test.go   # Verify collapsing fixes
  - /internal/tui/restart_integration_test.go # Verify TUI fixes
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /internal/mcp/tools.go                   # Uses log store, verify no breaking changes
  - /internal/process/manager.go             # Uses error parsing, verify compatibility
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/proxy/*                        # Proxy system not affected
  - /internal/discovery/*                    # Discovery system not affected
  - /cmd/brum/*                             # CLI not affected by these fixes
  - /pkg/events/*                           # Event system not affected
```

**Boundary Analysis Results**:
- **Usage Count**: Limited scope - 5 files in MODIFY zone, 2 files in REVIEW zone
- **Scope Assessment**: LIMITED scope - focused on specific test failures
- **Impact Radius**: Internal log processing only, no external API changes

### External Context Sources
**Primary Documentation**:
- [Go Testing Best Practices](https://golang.org/doc/tutorial/add-a-test) - Test writing patterns
- [BubbleTea Architecture](https://github.com/charmbracelet/bubbletea) - Message-based UI updates

**Standards Applied**:
- Go testing conventions for error condition validation
- BubbleTea message-passing patterns for TUI state updates
- Lock-free programming patterns for concurrent access

## Implementation Plan

### Phase 1: Fix Log Error Classification (Risk: MEDIUM)
**Files**: `internal/logs/error_parser.go`, `internal/logs/error_parser_test.go`
**Objective**: Fix NetworkError vs JavaScriptError classification
**Validation**: `go test -v ./internal/logs -run TestErrorParser_JavaScriptRuntimeErrors`

- [ ] **Task 1**: Fix error type classification in error parser
  - **Risk**: MEDIUM - Core error detection logic
  - **Files**: `internal/logs/error_parser.go`
  - **Success Criteria**: 
    - [ ] NetworkError test case passes with "FetchError" patterns
    - [ ] Test command passes: `go test -v ./internal/logs -run TestErrorParser_JavaScriptRuntimeErrors`
  - **Validation**: Pattern detection for "FetchError", "ENOTFOUND" should classify as NetworkError

### Phase 2: Fix Log Collapsing Logic (Risk: HIGH)
**Files**: `internal/logs/store.go`, `internal/logs/store_collapsed_test.go`
**Objective**: Fix log collapsing count logic and prevent index out of range
**Validation**: `go test -v ./internal/logs -run TestLogCollapsing`

- [ ] **Task 2**: Fix log collapsing count algorithm
  - **Risk**: HIGH - Core log storage functionality
  - **Files**: `internal/logs/store.go`
  - **Success Criteria**:
    - [ ] Collapsed entries show correct count (2 for repeated messages)
    - [ ] No index out of range panics in GetByProcessCollapsed
    - [ ] Test command passes: `go test -v ./internal/logs -run TestLogCollapsing`
  - **Validation**: Repeated log messages properly collapse with accurate counts

### Phase 3: Remove TUI Race Condition (Risk: LOW)
**Files**: `internal/tui/model.go`
**Objective**: Replace time.Sleep with proper message-based async pattern
**Validation**: `go test -race ./internal/tui`

- [ ] **Task 3**: Replace blocking sleep with BubbleTea message pattern
  - **Risk**: LOW - Single line change following established pattern  
  - **Files**: `internal/tui/model.go:3719`
  - **Success Criteria**:
    - [ ] No time.Sleep in production TUI code
    - [ ] Async operation uses tea.Cmd message pattern
    - [ ] Race detection passes: `go test -race ./internal/tui`
  - **Pattern**: Use `tea.Tick()` or message-based delay instead of blocking sleep

### Phase 4: Validation & Integration (Risk: LOW)
**Files**: All modified files
**Objective**: Verify all fixes work together
**Validation**: `make test` and `make test-race`

- [ ] **Task 4**: Run comprehensive test suite
  - **Risk**: LOW - Validation only
  - **Files**: All test files
  - **Success Criteria**:
    - [ ] All tests pass: `make test`
    - [ ] No race conditions: `make test-race` 
    - [ ] Ready for feature completion and merge

## Gotchas & Considerations
- **Error Parser**: Pattern matching order affects detection - ensure NetworkError patterns checked before generic JavaScriptError
- **Log Collapsing**: Thread-safe access required due to concurrent log writing
- **TUI Updates**: BubbleTea requires all state changes via message passing - no direct modifications
- **Test Order**: Some tests may be order-dependent due to shared state

## Definition of Done
- [ ] All 3 failing tests pass consistently
- [ ] No time.Sleep in TUI production code  
- [ ] Race condition testing passes: `go test -race ./...`
- [ ] Full test suite passes: `make test`
- [ ] Feature branch ready for merge to main
- [ ] No regression in existing functionality

## Validation Commands
```bash
# Test specific failing areas
go test -v ./internal/logs -run TestErrorParser_JavaScriptRuntimeErrors
go test -v ./internal/logs -run TestLogCollapsing
go test -race ./internal/tui

# Full validation
make test
make test-race
```

## Ready-to-Execute Tasks

### Task 1: Fix Error Type Classification
- **File**: `internal/logs/error_parser.go`
- **Problem**: "FetchError" with "ENOTFOUND" incorrectly classified as JavaScriptError instead of NetworkError
- **Solution**: Move NetworkError patterns before JavaScriptError in classification logic
- **Test**: `go test -v ./internal/logs -run TestErrorParser_JavaScriptRuntimeErrors/Network_Error`

### Task 2: Fix Log Collapsing Algorithm  
- **File**: `internal/logs/store.go`
- **Problem**: Collapsing count logic incorrect, index out of range in GetByProcessCollapsed
- **Solution**: Fix count increment logic and add bounds checking
- **Test**: `go test -v ./internal/logs -run TestLogCollapsing`

### Task 3: Remove TUI Race Condition
- **File**: `internal/tui/model.go:3719`
- **Problem**: `time.Sleep(500 * time.Millisecond)` blocks UI thread
- **Solution**: Replace with `tea.Tick(500*time.Millisecond, func() tea.Msg { ... })`
- **Test**: `go test -race ./internal/tui`

### Task 4: Full Validation
- **Command**: `make test && make test-race`
- **Objective**: Verify all fixes work together without regressions
- **Success**: Clean test run enables feature branch merge

## Execution Notes
- **Start with**: Task 1 (error classification) - lowest risk, quick validation
- **Validation**: Run individual test after each task before proceeding
- **Commit pattern**: `fix(logs): correct error type classification for network errors`
- **Priority**: Complete all tasks before returning to file output feature work

## Next Phase After Completion
1. **Archive this todo** to `archive/2025-07-27-critical-stabilization/`
2. **Complete file output feature** testing and documentation
3. **Archive remaining todo files** as outlined in consolidated todo
4. **Merge feature branch** to main with clean test suite

---

**CRITICAL**: This stabilization work blocks all other development. Complete these fixes before proceeding with any new features.