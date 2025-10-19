# Tasks: Complete Tab Functionality with Automated Testing

**Feature Branch**: `010-get-all-the`
**Created**: 2025-10-19
**Status**: Ready for execution

## Overview

This task list implements complete TUI tab functionality with error grouping intelligence and comprehensive automated testing following TDD principles. All 45 tasks are organized into 6 phases (A-F) with clear dependencies and parallel execution opportunities.

**Strategy**: Test-First Development (TDD) - Write failing tests, then implement to make tests pass

## Task Categories

- **[P]** = Can be executed in parallel (different files, no dependencies)
- **[S]** = Must be executed sequentially (same file or has dependencies)
- **[TEST]** = Write failing tests first (TDD)
- **[IMPL]** = Implementation to make tests pass

---

## Phase A: Test Infrastructure Setup

### T001: [P] Create test directory structure
**File**: `test/tui/`
**Action**: Create directory structure for contract, integration, and unit tests
```bash
mkdir -p test/tui/contracts
mkdir -p test/tui/integration
mkdir -p test/tui/unit
mkdir -p test/tui/fixtures
```
**Dependencies**: None
**Verification**: Directories exist

### T002: [P] Create test fixtures for error grouping
**File**: `test/tui/fixtures/errors.go`
**Action**: Create test data with various error types:
- Stack traces (JavaScript, TypeScript, Go)
- Compilation errors (TypeScript, Babel)
- Test failures (Jest, Vitest)
- Generic multi-line errors

**Expected**: Fixture functions that return test error strings and expected GroupedError outputs
**Dependencies**: None

### T003: [P] Create test fixtures for state objects
**File**: `test/tui/fixtures/states.go`
**Action**: Create helper functions to generate test state objects:
- `CreateTestErrorsState(n int) *state.ErrorsState`
- `CreateTestLogsState(n int) *state.LogsState`
- `CreateTestWebState(n int) *state.WebState`
- Each with configurable number of entries

**Dependencies**: None

### T004: [P] Create TTY allocation test helpers
**File**: `test/tui/helpers/tty.go`
**Action**: Create helper functions for TTY allocation in integration tests:
- `AllocateTTY() (*os.File, error)`
- `RunWithTTY(cmd *exec.Cmd) error`
- `CleanupTTY(tty *os.File)`

**Dependencies**: None
**Note**: Critical for TUI integration testing

---

## Phase B: Contract Tests (TDD - Failing Tests First)

### T005: [P] [TEST] Write state contract tests
**File**: `test/tui/contracts/state_contract_test.go`
**Action**: Implement all 7 test cases from `contracts/state_contract.md`:
1. `TestStateImmutability` - Verify original unchanged after mutation
2. `TestNoMutexes` - Use reflection to check for mutex fields
3. `TestVersionIncrement` - Verify version++ on every mutation
4. `TestCopyOnWrite` - Multiple updates create different instances
5. `TestConcurrentReads` - 100 concurrent readers with -race flag
6. `TestStateValidation` - Validate() catches invalid state
7. `TestRetentionLimit` - FIFO discard when limit exceeded

**Test For**: ErrorsState, LogsState, WebState, URLsState
**Expected**: All tests FAIL (no implementation yet)
**Dependencies**: T002, T003

### T006: [P] [TEST] Write view contract tests
**File**: `test/tui/contracts/view_contract_test.go`
**Action**: Implement all 10 test cases from `contracts/view_contract.md`:
1. `TestViewPurity` - Multiple calls produce identical output
2. `TestViewDoesNotModifyState` - State version unchanged
3. `TestViewDoesNotBlock` - Completes within 5ms timeout
4. `TestViewNoIO` - No new file descriptors opened
5. `BenchmarkViewPerformance` - <5ms average rendering
6. `TestViewNilState` - Handles nil gracefully
7. `TestViewEmptyState` - Shows empty message
8. `TestViewLargeState` - Handles 10k entries
9. `TestViewConcurrentRendering` - Safe for concurrent calls
10. `TestViewTerminalSizes` - Handles various terminal sizes

**Test For**: ErrorsView, LogsView, WebView
**Expected**: All tests FAIL (views not updated yet)
**Dependencies**: T003

### T007: [P] [TEST] Write service contract tests
**File**: `test/tui/contracts/service_contract_test.go`
**Action**: Implement all 10 test cases from `contracts/service_contract.md`:
1. `TestServiceStatelessness` - Use reflection to check for state fields
2. `TestServiceReturnsCommands` - Returns tea.Cmd
3. `TestServiceDoesNotModifyState` - State unchanged by service
4. `TestServiceDeterministic` - Same input → same output
5. `TestServiceErrorHandling` - Returns error messages on failure
6. `TestServiceNoGoroutines` - No new goroutines launched
7. `TestServiceConfigurationOnly` - Config unchanged after calls
8. `TestServiceConcurrentCalls` - Safe for 100 concurrent calls
9. `BenchmarkServicePerformance` - <50ms average
10. `TestServiceCommandBatching` - Supports tea.Batch

**Test For**: ErrorGroupingService (to be created)
**Expected**: All tests FAIL (service not implemented yet)
**Dependencies**: T002, T003

### T008: [P] [TEST] Write error grouping unit tests
**File**: `test/tui/unit/error_grouping_test.go`
**Action**: Create table-driven tests for error grouping patterns:
- Stack trace grouping (5+ test cases)
- Compilation error grouping (3+ test cases)
- Test failure grouping (3+ test cases)
- Generic multi-line errors (2+ test cases)

**Test Function**: `TestErrorGrouping(t *testing.T)`
**Expected**: All tests FAIL (grouping service not implemented)
**Dependencies**: T002

---

## Phase C: State Implementation (Make Tests Pass)

### T009: [P] [IMPL] Implement GroupedError entity
**File**: `internal/tui/state/grouped_error.go`
**Action**: Create GroupedError struct with all fields from data-model.md:
- ID, ProcessName, ProcessPID, FirstLine, FullContent, ErrorType
- Timestamp, Source, LineNumber, Version
- Add validation method

**Expected**: Contract tests for immutability pass
**Dependencies**: T005 (tests written)

### T010: [P] [IMPL] Implement ErrorType enum
**File**: `internal/tui/state/error_type.go`
**Action**: Create ErrorType enum with values:
- StackTrace, CompilationError, TestFailure, Generic
- Add String() method

**Dependencies**: T009

### T011: [IMPL] Enhance ErrorsState with grouping support
**File**: `internal/tui/state/errors_state.go`
**Action**: Update ErrorsState to support grouped errors:
- Change `Errors` field to `GroupedErrors []GroupedError`
- Add `ParserState ErrorParserState` field
- Add `MaxRetention int` field
- Implement all immutable update methods:
  - `AddGroupedError(error GroupedError) *ErrorsState`
  - `ClearErrors() *ErrorsState`
  - `SetFilterProcess(process string) *ErrorsState`
- Implement `Validate()` method
- Implement retention limit enforcement

**Expected**: State contract tests for ErrorsState pass
**Dependencies**: T005, T009
**Note**: Not parallel - modifies existing file

### T012: [P] [IMPL] Implement ErrorParserState
**File**: `internal/tui/state/error_parser_state.go`
**Action**: Create ErrorParserState struct with fields:
- CurrentGroup []string
- InErrorGroup bool
- LastErrorType ErrorType
- LinesSinceError int

**Dependencies**: T010

### T013: [P] [IMPL] Implement WebState for proxy events
**File**: `internal/tui/state/web_state.go`
**Action**: Create NEW WebState struct with all fields from data-model.md:
- Events []ProxyEvent
- EndpointFilter, StatusCodeFilter, MethodFilter
- SelectedIndex, ShowDetails, MaxRetention
- LastUpdated, Version
- Implement all immutable update methods
- Implement `Validate()` method

**Expected**: State contract tests for WebState pass
**Dependencies**: T005

### T014: [P] [IMPL] Implement ProxyEvent entity
**File**: `internal/tui/state/proxy_event.go`
**Action**: Create ProxyEvent struct with all fields from data-model.md:
- ID, Timestamp, Method, URL, StatusCode
- RequestHeaders, ResponseHeaders
- RequestBody, ResponseBody, Duration, Size

**Dependencies**: None

### T015: [IMPL] Update LogsState with clear support
**File**: `internal/tui/state/logs_state.go`
**Action**: Add `ClearLogs() *LogsState` immutable method
- Preserve filters and settings
- Reset entries to empty slice
- Increment version

**Expected**: State contract tests still pass
**Dependencies**: T005
**Note**: Not parallel - modifies existing file

### T016: [IMPL] Update URLsState validation
**File**: `internal/tui/state/urls_state.go`
**Action**: Add `Validate()` method to existing URLsState
- Check SelectedIndex bounds
- Verify DeduplicationMap consistency

**Dependencies**: T005
**Note**: Not parallel - modifies existing file

### T017: [IMPL] Update MasterData with new states
**File**: `internal/tui/state/master_data.go`
**Action**: Add WebState field to MasterData:
```go
WebState *WebState
```
- Initialize in `NewMasterData()` constructor

**Dependencies**: T013
**Note**: Not parallel - modifies existing file

---

## Phase D: Service Implementation (Make Tests Pass)

### T018: [P] [IMPL] Implement ErrorGroupingService
**File**: `internal/tui/services/error_grouping_service.go`
**Action**: Create NEW stateless service for error grouping:
- `ParseAndGroupError(line string, parserState *ErrorParserState) tea.Cmd`
- Pattern matching for error types (stack traces, compilation, tests)
- State machine logic for multi-line grouping
- Return `ErrorGroupedMsg` when group complete
- Configuration for patterns (no mutable state)

**Expected**: Service contract tests pass, error grouping unit tests pass
**Dependencies**: T007, T008, T012

### T019: [P] [IMPL] Implement clear command services
**File**: `internal/tui/services/clear_service.go`
**Action**: Create NEW stateless service for clear commands:
- `ClearErrors() tea.Cmd` → returns `ClearErrorsMsg`
- `ClearLogs() tea.Cmd` → returns `ClearLogsMsg`
- `ClearWeb() tea.Cmd` → returns `ClearWebMsg`
- All return commands, no direct state modification

**Expected**: Service contract tests pass
**Dependencies**: T007

---

## Phase E: View Implementation (Make Tests Pass)

### T020: [IMPL] Update ErrorsView for grouped errors
**File**: `internal/tui/components/errors_view.go`
**Action**: Update `ViewWithState()` to display grouped errors:
- Iterate through `ErrorsState.GroupedErrors`
- Display each error as single entry with full context
- Show error type indicator (stack/compilation/test)
- Highlight selected error
- Show "No errors" when empty
- Maintain <5ms rendering performance

**Expected**: View contract tests for ErrorsView pass
**Dependencies**: T006, T011
**Note**: Not parallel - modifies existing file

### T021: [IMPL] Update WebView for proxy events
**File**: `internal/tui/components/web_view.go`
**Action**: Update `ViewWithState()` to display proxy events:
- Iterate through `WebState.Events`
- Display method, URL, status code, timing
- Show request/response details when selected
- Apply filters (endpoint, status, method)
- Show "No events" when empty

**Expected**: View contract tests for WebView pass
**Dependencies**: T006, T013
**Note**: Not parallel - modifies existing file

### T022: [IMPL] Update LogsView for clear indication
**File**: `internal/tui/components/logs_view.go`
**Action**: Ensure `ViewWithState()` handles cleared state:
- Show "Logs cleared" message briefly when cleared
- Resume normal rendering after clear

**Expected**: View contract tests for LogsView pass
**Dependencies**: T006
**Note**: Not parallel - modifies existing file

---

## Phase F: Message and Handler Integration

### T023: [IMPL] Add new message types
**File**: `internal/tui/messages.go`
**Action**: Add new message types:
```go
type ErrorGroupedMsg struct {
    Error GroupedError
}

type ClearErrorsMsg struct{}
type ClearLogsMsg struct{}
type ClearWebMsg struct{}

type ShowClearConfirmationMsg struct {
    Tab string  // "errors", "logs", "web"
}
```

**Dependencies**: T009, T014
**Note**: Not parallel - modifies existing file

### T024: [IMPL] Update Model.Update() for new messages
**File**: `internal/tui/model.go`
**Action**: Add message handlers in `Update()` function:
- `ErrorGroupedMsg`: Add to ErrorsState
- `ClearErrorsMsg`: Call ErrorsState.ClearErrors()
- `ClearLogsMsg`: Call LogsState.ClearLogs()
- `ClearWebMsg`: Call WebState.ClearWeb()
- `ShowClearConfirmationMsg`: Show confirmation dialog

**Dependencies**: T023, T011, T013, T015
**Note**: Not parallel - modifies existing file

### T025: [IMPL] Add 'c' key handler for clear commands
**File**: `internal/tui/handlers/key_handler.go`
**Action**: Update key handler to support 'c' key:
- Check current tab (Errors/Logs/Web)
- Return appropriate Clear message based on tab
- Only active in Errors/Logs/Web tabs

**Dependencies**: T023
**Note**: Not parallel - modifies existing file

### T026: [IMPL] Add 'C' key handler for confirmation
**File**: `internal/tui/handlers/key_handler.go`
**Action**: Update key handler to support 'C' (shift+c):
- Return `ShowClearConfirmationMsg` with current tab
- Prompt user to confirm (y/n)
- Execute clear on 'y', cancel on 'n' or Esc

**Dependencies**: T023, T025
**Note**: Not parallel - modifies existing file

### T027: [IMPL] Wire error grouping service to Update()
**File**: `internal/tui/model.go`
**Action**: Integrate ErrorGroupingService:
- On log line received with error pattern
- Call service.ParseAndGroupError()
- Handle returned ErrorGroupedMsg in Update()

**Dependencies**: T018, T024
**Note**: Not parallel - modifies existing file

---

## Phase G: Integration Tests (End-to-End)

### T028: [P] [TEST] Tab navigation integration test
**File**: `test/tui/integration/tab_navigation_test.go`
**Action**: Create integration test for tab switching:
- Start TUI with TTY allocation
- Simulate number key presses (1-7)
- Verify tab switches occur
- Verify state preservation across switches
- Test Tab/Shift+Tab cycling

**Test Scenario**: Quickstart Scenario 4
**Dependencies**: T004
**Build Tags**: `// +build integration`

### T029: [P] [TEST] Error grouping integration test
**File**: `test/tui/integration/error_grouping_test.go`
**Action**: Create end-to-end test for error grouping:
- Start process that generates errors
- Capture errors in Errors tab
- Verify stack traces grouped correctly
- Verify compilation errors grouped
- Verify test failures grouped

**Test Scenario**: Quickstart Scenario 1, 7
**Dependencies**: T004, T018
**Build Tags**: `// +build integration`

### T030: [P] [TEST] Clear commands integration test
**File**: `test/tui/integration/clear_commands_test.go`
**Action**: Create integration test for clear functionality:
- Generate errors, logs, web events
- Test 'c' key clears content immediately
- Test 'C' key shows confirmation prompt
- Test 'y' confirms, 'n' cancels

**Test Scenario**: Quickstart Scenario 2, 3
**Dependencies**: T004, T025, T026
**Build Tags**: `// +build integration`

### T031: [P] [TEST] State preservation integration test
**File**: `test/tui/integration/state_preservation_test.go`
**Action**: Create integration test for state persistence:
- Set scroll position in Logs tab
- Switch to different tab
- Switch back to Logs tab
- Verify scroll position preserved
- Test for all tabs

**Test Scenario**: Quickstart Scenario 5
**Dependencies**: T004
**Build Tags**: `// +build integration`

### T032: [P] [TEST] Real-time updates integration test
**File**: `test/tui/integration/realtime_updates_test.go`
**Action**: Create integration test for real-time display:
- Start long-running process
- Verify logs stream in real-time
- Trigger error, verify appears in Errors tab
- Verify no UI freezes or lag

**Test Scenario**: Quickstart Scenario 6
**Dependencies**: T004
**Build Tags**: `// +build integration`

### T033: [P] [TEST] Performance integration test
**File**: `test/tui/integration/performance_test.go`
**Action**: Create integration test for performance:
- Generate 1000+ log entries
- Generate 100+ errors
- Verify tab switching <50ms
- Verify smooth scrolling
- Verify memory <100MB

**Test Scenario**: Quickstart Scenario 8
**Dependencies**: T004
**Build Tags**: `// +build integration`

---

## Phase H: Validation and Polish

### T034: [S] Run all contract tests
**Command**: `go test ./test/tui/contracts/... -v`
**Action**: Execute all contract tests and verify they pass
**Expected**: 100% pass rate
**Dependencies**: T005, T006, T007, all IMPL tasks

### T035: [S] Run all integration tests
**Command**: `BRUMMER_BINARY=./brum go test -tags=integration ./test/tui/integration/... -v`
**Action**: Execute all integration tests with TTY allocation
**Expected**: 100% pass rate
**Dependencies**: T028, T029, T030, T031, T032, T033

### T036: [S] Run race condition tests
**Command**: `make test-race`
**Action**: Run tests with Go race detector
**Expected**: Zero race conditions detected
**Dependencies**: T034, T035
**Note**: Critical - MUST pass before merge

### T037: [S] Generate coverage report
**Command**:
```bash
go test ./internal/tui/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```
**Action**: Generate and review coverage report
**Expected**: >80% coverage for all tab-related code
**Dependencies**: T034, T035

### T038: [S] Run performance benchmarks
**Command**: `go test -bench=. ./test/tui/... -benchmem`
**Action**: Execute performance benchmarks
**Expected**:
- View rendering: <5ms
- Service operations: <50ms
- Memory: <100MB for typical usage

**Dependencies**: T006, T007

### T039: [S] Execute manual quickstart tests
**File**: `specs/010-get-all-the/quickstart.md`
**Action**: Manually execute all 10 test scenarios
**Expected**: All pass criteria met
**Dependencies**: T034, T035

### T040: [S] Verify error grouping edge cases
**Action**: Test error grouping with:
- Very long stack traces (100+ lines)
- Nested errors
- Special characters and unicode
- ANSI color codes in errors

**Expected**: All handled gracefully
**Dependencies**: T029

### T041: [S] Verify terminal resize handling
**Action**: Test all tabs with terminal resize:
- Resize during active tab viewing
- Resize during tab switching
- Very small terminals (40x10)
- Very large terminals (200x60)

**Expected**: Content reflows correctly, no crashes
**Dependencies**: T035

### T042: [S] Run stress test for race conditions
**Action**: Rapidly switch tabs and clear content:
- Mash number keys for 30 seconds
- Mash 'c' key while content generates
- Monitor for race warnings

**Expected**: No races, no crashes, all tabs functional
**Dependencies**: T036

### T043: [S] Update documentation
**Files**:
- `docs/architecture/tui-architecture.md`
- `docs/features/error-grouping.md` (NEW)
- `docs/features/clear-commands.md` (NEW)

**Action**: Document new features:
- Error grouping algorithms
- Clear command usage
- Web tab vs Errors tab distinction

**Dependencies**: All previous tasks

### T044: [S] Update CLAUDE.md status
**File**: `CLAUDE.md`
**Action**: Update "Current Development Status" section:
- Mark feature 010 complete
- Document error grouping feature
- Update architecture notes

**Dependencies**: T043

### T045: [S] Final verification checklist
**Action**: Verify all completion criteria:
- [ ] All 7 tabs fully functional
- [ ] Error grouping working (stack traces, compilation, tests)
- [ ] Clear commands working ('c' and 'C')
- [ ] >80% test coverage achieved
- [ ] Zero race conditions
- [ ] <5ms View() rendering
- [ ] <50ms keyboard response
- [ ] All contract tests passing
- [ ] All integration tests passing
- [ ] Manual quickstart tests passed

**Dependencies**: All previous tasks

---

## Parallel Execution Examples

### Example 1: Phase A (Test Infrastructure) - All Parallel
```bash
# Can run all 4 tasks simultaneously
Task agent1: "Execute T001 - Create test directory structure"
Task agent2: "Execute T002 - Create test fixtures for error grouping"
Task agent3: "Execute T003 - Create test fixtures for state objects"
Task agent4: "Execute T004 - Create TTY allocation test helpers"
```

### Example 2: Phase B (Contract Tests) - All Parallel
```bash
# All contract tests can be written simultaneously (different files)
Task agent1: "Execute T005 - Write state contract tests"
Task agent2: "Execute T006 - Write view contract tests"
Task agent3: "Execute T007 - Write service contract tests"
Task agent4: "Execute T008 - Write error grouping unit tests"
```

### Example 3: Phase C (State Implementation) - Mostly Parallel
```bash
# Parallel tasks (independent files)
Task agent1: "Execute T009 - Implement GroupedError entity"
Task agent2: "Execute T010 - Implement ErrorType enum"
Task agent3: "Execute T012 - Implement ErrorParserState"
Task agent4: "Execute T013 - Implement WebState"
Task agent5: "Execute T014 - Implement ProxyEvent entity"

# Sequential after parallel (modify shared files)
Then: T011, T015, T016, T017
```

### Example 4: Phase G (Integration Tests) - All Parallel
```bash
# All integration tests are independent files
Task agent1: "Execute T028 - Tab navigation integration test"
Task agent2: "Execute T029 - Error grouping integration test"
Task agent3: "Execute T030 - Clear commands integration test"
Task agent4: "Execute T031 - State preservation integration test"
Task agent5: "Execute T032 - Real-time updates integration test"
Task agent6: "Execute T033 - Performance integration test"
```

---

## Success Metrics

- **Coverage**: >80% code coverage for all tab-related code (T037)
- **Performance**: <5ms View() rendering, <50ms keyboard response (T038)
- **Reliability**: Zero race conditions detected (T036)
- **Architecture**: 100% compliance with lock-free immutable state pattern (T005)
- **Testing**: All contract tests, integration tests, and manual tests pass (T034, T035, T039)
- **Functionality**: All 7 tabs functional with error grouping and clear commands (T045)

---

## Dependencies Graph

```
Phase A (T001-T004) - All parallel
    ↓
Phase B (T005-T008) - All parallel (depends on Phase A)
    ↓
Phase C (T009-T017) - Mixed (some parallel, some sequential)
    ↓
Phase D (T018-T019) - All parallel (depends on Phase C)
    ↓
Phase E (T020-T022) - Sequential (modifies existing files)
    ↓
Phase F (T023-T027) - Sequential (modifies existing files)
    ↓
Phase G (T028-T033) - All parallel (integration tests)
    ↓
Phase H (T034-T045) - Sequential (validation pipeline)
```

---

*Total Tasks: 45*
*Estimated Duration: 3-4 sprints*
*Parallel Execution Opportunities: 25 tasks*
*Implementation Status: Ready for execution*
