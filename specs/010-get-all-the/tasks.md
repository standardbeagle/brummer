# Tasks: Complete Tab Functionality with Automated Testing

**Feature Branch**: `010-get-all-the`
**Created**: 2025-10-19
**Status**: Ready for execution

## Overview

This task list implements complete TUI tab functionality with error grouping intelligence and comprehensive automated testing following TDD principles. All 59 tasks (50 implementation + 9 build checkpoints) are organized into 8 phases (A-H) with clear dependencies and parallel execution opportunities.

**Strategy**: Test-First Development (TDD) - Write failing tests, then implement to make tests pass

**Build-First Approach**: The application MUST build successfully after every task. 9 mandatory build checkpoints ensure continuous integration throughout development.

**Architecture Compliance**: All tasks enforce lock-free immutable state pattern, Redux/Elm architecture, and View() deadlock prevention as documented in CLAUDE.md.

## Task Categories

- **[P]** = Can be executed in parallel (different files, no dependencies)
- **[S]** = Must be executed sequentially (same file or has dependencies)
- **[TEST]** = Write failing tests first (TDD)
- **[IMPL]** = Implementation to make tests pass
- **[BUILD]** = Build verification checkpoint (MANDATORY)

## 🔨 Build Verification Policy

**CRITICAL**: The application MUST build successfully after EVERY task completion.

**After completing ANY task, you MUST:**
1. Run `make build` to verify the application compiles
2. If build fails, fix compilation errors immediately before proceeding
3. Never leave the codebase in a non-building state

**Build Checkpoints** are inserted throughout the task list to ensure continuous integration. These are MANDATORY gates that cannot be skipped.

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

### T004a: [BUILD] Build Verification Checkpoint - Phase A Complete
**Command**: `make build`
**Action**: Verify application builds successfully after Phase A
**Expected**: Clean build with no compilation errors
**If Failed**: Fix all compilation errors before proceeding to Phase B
**Dependencies**: T001, T002, T003, T004
**Note**: MANDATORY - Do not skip this verification

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

### T007b: [P] [TEST] Write Update() separation contract tests
**File**: `test/tui/contracts/update_separation_test.go`
**Action**: Implement tests verifying Update() is the ONLY mutation point (Redux/Elm pattern):
1. `TestViewNeverCallsUpdate` - View() doesn't trigger state mutations
2. `TestAllMutationsInUpdate` - All state changes flow through Update()
3. `TestNoDirectStateModification` - Services don't modify state directly
4. `TestCommandPattern` - All async operations use tea.Cmd pattern (no direct goroutines)
5. `TestViewReadOnly` - View() only reads state, never writes
6. `TestServiceReturnsCmdOnly` - Services return Commands, never mutate Model

**Test For**: ErrorsView, LogsView, WebView, ErrorGroupingService, ClearService
**Expected**: All tests FAIL initially (pattern not enforced yet)
**Dependencies**: T003, T006, T007
**Note**: Critical for Redux/Elm architecture compliance

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

### T008a: [BUILD] Build Verification Checkpoint - Phase B Complete
**Command**: `make build`
**Action**: Verify application builds with all contract tests
**Expected**: Clean build (tests may fail, but code compiles)
**Test Verification**: `go test ./test/tui/contracts/... -v` - should compile but tests FAIL (expected)
**If Failed**: Fix all compilation errors before proceeding to Phase C
**Dependencies**: T005, T006, T007, T007b, T008
**Note**: MANDATORY - Tests should fail (no implementation), but code must compile

---

## Phase C: State Implementation (Make Tests Pass)

### T009: [P] [IMPL] Implement GroupedError entity
**File**: `internal/tui/state/grouped_error.go`
**Action**: Create GroupedError struct with all fields from data-model.md:
- ID, ProcessName, ProcessPID, FirstLine, FullContent, ErrorType
- Timestamp, Source, LineNumber, Version
- **MANDATORY**: Add immutability documentation comment:
  ```go
  // IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
  // NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
  ```
- Add validation method
- Ensure NO mutex fields (verified by TestNoMutexes contract test)

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
- **MANDATORY**: Add immutability documentation comment:
  ```go
  // IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
  // NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
  ```
- Implement all immutable update methods
- Implement `Validate()` method
- Ensure NO mutex fields (verified by TestNoMutexes contract test)

**Expected**: State contract tests for WebState pass
**Dependencies**: T005

### T014: [P] [IMPL] Implement ProxyEvent entity
**File**: `internal/tui/state/proxy_event.go`
**Action**: Create ProxyEvent struct with all fields from data-model.md:
- ID, Timestamp, Method, URL, StatusCode
- RequestHeaders, ResponseHeaders
- RequestBody, ResponseBody, Duration, Size
- **MANDATORY**: Add immutability documentation comment:
  ```go
  // IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
  // NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
  ```
- Ensure NO mutex fields (verified by TestNoMutexes contract test)

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
- **NEW**: Add `Validate()` method to MasterData that calls Validate() on all child states
- **NEW**: Verify no duplicate state storage (single source of truth pattern)
- **NEW**: Add documentation comment:
  ```go
  // MasterData is the single source of truth for ALL TUI state.
  // All state updates flow through Update() which modifies MasterData.
  // NO state should be stored elsewhere (Redux/Elm pattern).
  ```

**Expected**: MasterData serves as single source of truth for all TUI state
**Dependencies**: T013
**Note**: Not parallel - modifies existing file

### T017a: [BUILD] Build Verification Checkpoint - Phase C Complete
**Command**: `make build`
**Action**: Verify application builds with all state implementations
**Expected**: Clean build with no compilation errors
**Test Verification**:
```bash
# Verify state contract tests pass
go test ./test/tui/contracts/... -v -run TestState
# State contract tests should now PASS

# NEW: Verify lock-free compliance (CRITICAL)
go test ./test/tui/contracts/... -v -run TestNoMutexes
# MUST PASS - No mutexes allowed in state objects

# NEW: Verify concurrent safety with race detector
go test ./test/tui/contracts/... -v -run TestConcurrentReads -race
# MUST PASS with -race flag (no race conditions)

# NEW: Verify immutability contract
go test ./test/tui/contracts/... -v -run TestStateImmutability
# MUST PASS - Original state unchanged after mutations

# NEW: Verify Update() separation contract
go test ./test/tui/contracts/... -v -run TestUpdate
# Should PASS for implemented components
```
**If Failed**:
1. Fix compilation errors first
2. If state contract tests fail, fix state implementations
3. **NEW**: If TestNoMutexes fails, remove ALL mutexes from state objects immediately
4. **NEW**: If race detector fails, ensure proper defensive copying in getters
5. **NEW**: If immutability tests fail, verify all update methods return *State
6. Do not proceed until ALL state tests pass
**Dependencies**: T009-T017
**Note**: MANDATORY - State contract tests MUST pass before Phase D. Lock-free compliance is NON-NEGOTIABLE.

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

### T019a: [BUILD] Build Verification Checkpoint - Phase D Complete
**Command**: `make build`
**Action**: Verify application builds with all service implementations
**Expected**: Clean build with no compilation errors
**Test Verification**:
```bash
go test ./test/tui/contracts/... -v -run TestService
# Service contract tests should now PASS
go test ./test/tui/unit/... -v -run TestErrorGrouping
# Error grouping unit tests should now PASS
```
**If Failed**:
1. Fix compilation errors first
2. If service contract tests fail, fix service implementations
3. Do not proceed until service tests pass
**Dependencies**: T018, T019
**Note**: MANDATORY - Service contract tests MUST pass before Phase E

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

### T022a: [BUILD] Build Verification Checkpoint - Phase E Complete
**Command**: `make build`
**Action**: Verify application builds with all view updates
**Expected**: Clean build with no compilation errors
**Test Verification**:
```bash
go test ./test/tui/contracts/... -v -run TestView
# View contract tests should now PASS
```
**If Failed**:
1. Fix compilation errors first
2. If view contract tests fail, fix view implementations
3. Do not proceed until view tests pass
**Dependencies**: T020, T021, T022
**Note**: MANDATORY - View contract tests MUST pass before Phase F

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
**Action**: Update key handler to support 'C' (shift+c) with complete confirmation workflow:
- Return `ShowClearConfirmationMsg` with current tab
- **NEW**: Set confirmation state in Model (e.g., `awaitingClearConfirmation bool`, `confirmationTab string`)
- **NEW**: While confirmation active (in Update() handler):
  - 'y' or 'Y': Execute clear command, return appropriate ClearMsg (ClearErrorsMsg/ClearLogsMsg/ClearWebMsg)
  - 'n' or 'N': Cancel, clear confirmation state, return to normal mode
  - Esc: Cancel, clear confirmation state, return to normal mode
  - Other keys: Ignore (confirmation takes precedence over normal key handling)
- **NEW**: Display confirmation prompt in View() when `awaitingClearConfirmation == true`:
  - Format: "Clear all [errors/logs/web events]? (y/n)" in status bar or modal
  - Highlight the prompt to draw user attention
- **NEW**: Prevent accidental data loss by requiring explicit confirmation

**Expected**: Confirmation workflow prevents accidental clearing of important data
**Dependencies**: T023, T025
**Note**: Not parallel - modifies existing file. Critical UX safety feature.

### T027: [IMPL] Wire error grouping service to Update()
**File**: `internal/tui/model.go`
**Action**: Integrate ErrorGroupingService:
- On log line received with error pattern
- Call service.ParseAndGroupError()
- Handle returned ErrorGroupedMsg in Update()

**Dependencies**: T018, T024
**Note**: Not parallel - modifies existing file

### T027a: [BUILD] Build Verification Checkpoint - Phase F Complete
**Command**: `make build`
**Action**: Verify application builds with complete integration
**Expected**: Clean build with no compilation errors
**Test Verification**:
```bash
# Verify application runs
./brum --help
# Should display help without errors

# Quick smoke test
timeout 5s ./brum --no-tui || true
# Should start without crashes
```
**If Failed**:
1. Fix compilation errors first
2. If runtime errors occur, fix integration issues
3. Test that TUI starts without panicking
**Dependencies**: T023-T027
**Note**: MANDATORY - Application MUST run before integration tests

---

## Phase G: Integration Tests (End-to-End)

### T028: [P] [TEST] Tab navigation integration test
**File**: `test/tui/integration/tab_navigation_test.go`
**Action**: Create integration test for tab switching with deadlock prevention:
- Start TUI with TTY allocation
- Simulate number key presses (1-7)
- **NEW**: Verify View() never blocks during tab switches (wrap View() calls with 10ms timeout)
- **NEW**: Test rapid tab switching (mash number keys 1-7 repeatedly for 5 seconds)
- **NEW**: Monitor for UI freezes during rapid switching (should never freeze)
- **NEW**: Specifically test tabs 3 (Errors) and 4 (URLs) - historical deadlock locations
- Verify tab switches occur
- Verify state preservation across switches
- Test Tab/Shift+Tab cycling
- **NEW**: Verify tick handlers (1 second interval) don't cause freezes in any tab

**Test Scenario**: Quickstart Scenario 4 + Deadlock Prevention (tab 3/4 lockup issue)
**Dependencies**: T004
**Build Tags**: `// +build integration`
**Note**: Critical for preventing the tab 3/4 deadlock issue documented in tui-deadlock-prevention.md

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

### T033a: [BUILD] Build Verification Checkpoint - Phase G Complete
**Command**: `make build`
**Action**: Verify application builds with all integration tests
**Expected**: Clean build with no compilation errors
**Test Verification**:
```bash
# Verify integration tests compile
go test -tags=integration ./test/tui/integration/... -c
# Should compile successfully

# Build should still work
make build
./brum --version
```
**If Failed**:
1. Fix compilation errors in integration tests
2. Ensure build tags are correct
3. Verify all imports are valid
**Dependencies**: T028-T033
**Note**: MANDATORY - Integration tests must compile before validation phase

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

### T038b: [S] Benchmark lock-free state operations
**Command**: `go test -bench=BenchmarkState ./test/tui/contracts/... -benchmem`
**Action**: Validate lock-free architecture performance claims
**Expected Performance Targets**:
```bash
BenchmarkStateRead-8         20000000    50 ns/op     0 B/op   0 allocs/op
BenchmarkStateWrite-8         5000000   300 ns/op   128 B/op   1 allocs/op
```
**Verification Criteria**:
- Read operations: <100ns per operation (no mutex contention)
- Write operations: <1μs per operation (copy-on-write overhead acceptable)
- Memory: Zero allocations for reads, minimal allocations for writes
- Zero mutex operations (verify with pprof if available)
- **Performance Claim**: Should demonstrate 20-30x improvement over mutex-based state management
  - Mutex-based baseline: ~2-3μs per read (with contention)
  - Lock-free target: <100ns per read
  - Improvement ratio: 20-30x

**Action Steps**:
1. Run state benchmarks with `-benchmem` flag
2. Analyze ns/op and allocs/op metrics
3. Compare against baseline (if mutex-based implementation exists for comparison)
4. Verify no blocking/contention in concurrent scenarios
5. Document performance results in benchmark output

**Dependencies**: T034, T038
**Note**: This validates the core architectural claim from CLAUDE.md about lock-free performance

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

**Build Verification**:
- [ ] `make build` completes successfully
- [ ] `./brum --version` displays correct version
- [ ] `./brum --help` shows all commands
- [ ] Application starts without panics

**Functionality Verification**:
- [ ] All 7 tabs fully functional
- [ ] Error grouping working (stack traces, compilation, tests)
- [ ] Clear commands working ('c' and 'C')
- [ ] Clear confirmation workflow working (prevents accidental data loss)
- [ ] Tab navigation responsive (<50ms)

**Test Verification**:
- [ ] All contract tests passing (100%)
- [ ] All integration tests passing (100%)
- [ ] Manual quickstart tests passed (all 10 scenarios)
- [ ] >80% test coverage achieved
- [ ] Zero race conditions detected
- [ ] <5ms View() rendering verified
- [ ] <50ms keyboard response verified

**Architecture Compliance Verification** (NEW):
- [ ] **Lock-Free State**: All state objects have NO mutexes (TestNoMutexes passes)
- [ ] **Immutability**: All state objects have mandatory documentation comments
- [ ] **Redux/Elm Pattern**: Update() is only mutation point (TestUpdate passes)
- [ ] **View() Purity**: No blocking operations in View() (TestViewDoesNotBlock passes)
- [ ] **Service Statelessness**: All services return tea.Cmd (TestServiceStatelessness passes)
- [ ] **Single Source of Truth**: MasterData is only state storage
- [ ] **Performance**: Lock-free operations achieve <100ns reads (T038b benchmarks)
- [ ] **Deadlock Prevention**: Rapid tab switching doesn't freeze (T028 integration test)

**Production Readiness**:
- [ ] Application builds on all platforms (Linux, macOS, Windows)
- [ ] No compilation warnings
- [ ] No linter errors (`make lint`)
- [ ] Documentation updated

**Final Build Verification**:
```bash
# Clean build from scratch
make clean
make build

# Verify binary works
./brum --version
./brum --help
timeout 10s ./brum --no-tui

# Verify cross-platform builds
GOOS=linux GOARCH=amd64 go build -o brum-linux ./cmd/brum
GOOS=darwin GOARCH=amd64 go build -o brum-darwin ./cmd/brum
GOOS=windows GOARCH=amd64 go build -o brum-windows.exe ./cmd/brum

echo "✅ All builds successful - Feature complete!"
```

**Dependencies**: All previous tasks
**Note**: This is the FINAL gate - all checks must pass

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
- **Lock-Free Performance**: <100ns state reads, 20-30x faster than mutex-based (T038b)
- **Reliability**: Zero race conditions detected (T036)
- **Architecture**: 100% compliance with lock-free immutable state pattern (T005, T017a)
- **Redux/Elm Compliance**: Update() is only mutation point, services stateless (T007b)
- **Deadlock Prevention**: View() never blocks, rapid tab switching works (T028)
- **Testing**: All contract tests, integration tests, and manual tests pass (T034, T035, T039)
- **Functionality**: All 7 tabs functional with error grouping and clear commands (T045)
- **Immutability**: All state objects have mandatory documentation comments (T009, T013, T014)
- **Single Source of Truth**: MasterData validated as only state storage (T017)

---

## Dependencies Graph

```
Phase A (T001-T004) - All parallel
    ↓
Phase B (T005-T008) - All parallel (depends on Phase A)
    + T007b (NEW) - Update() separation tests (parallel with others)
    ↓
Phase C (T009-T017) - Mixed (some parallel, some sequential)
    + Enhanced with mandatory lock-free documentation (T009, T013, T014)
    + Enhanced MasterData validation (T017)
    ↓
Phase D (T018-T019) - All parallel (depends on Phase C)
    ↓
Phase E (T020-T022) - Sequential (modifies existing files)
    ↓
Phase F (T023-T027) - Sequential (modifies existing files)
    + Enhanced confirmation workflow (T026)
    ↓
Phase G (T028-T033) - All parallel (integration tests)
    + Enhanced deadlock prevention testing (T028)
    ↓
Phase H (T034-T045) - Sequential (validation pipeline)
    + T038b (NEW) - Lock-free performance benchmarks
    + Enhanced architecture compliance verification (T045)
```

**New Tasks Added**:
- **T007b**: Update() separation contract tests (Phase B)
- **T038b**: Lock-free state performance benchmarks (Phase H)

**Enhanced Tasks**:
- **T009, T013, T014**: Added mandatory immutability documentation
- **T017**: Added MasterData validation and single source of truth verification
- **T017a**: Added lock-free compliance verification with race detector
- **T026**: Added detailed confirmation workflow implementation
- **T028**: Added View() blocking prevention and deadlock testing
- **T045**: Added comprehensive architecture compliance checklist

---

## Build Checkpoint Summary

**9 Mandatory Build Checkpoints** ensure the application builds throughout development:

1. **T004a**: After Phase A (Test Infrastructure) - Verify fixtures compile
2. **T008a**: After Phase B (Contract Tests) - Verify tests compile (may fail, but compile)
3. **T017a**: After Phase C (State Implementation) - Verify states build, state tests pass, lock-free compliance verified
4. **T019a**: After Phase D (Service Implementation) - Verify services build, service tests pass
5. **T022a**: After Phase E (View Implementation) - Verify views build, view tests pass
6. **T027a**: After Phase F (Integration) - Verify application runs without crashes
7. **T033a**: After Phase G (Integration Tests) - Verify integration tests compile
8. **T038b**: After Performance Testing - Verify lock-free performance benchmarks (20-30x improvement)
9. **T045**: Final Build Verification - Cross-platform builds, all tests pass, architecture compliance verified

**Critical Rule**: If any build checkpoint fails, STOP and fix before proceeding.

**Architecture Enforcement**: Build checkpoints verify lock-free state compliance, Redux/Elm pattern adherence, and View() deadlock prevention.

---

*Total Tasks: 59 (50 implementation + 9 build checkpoints)*
*Estimated Duration: 3-4 sprints*
*Parallel Execution Opportunities: 27 tasks*
*Build Checkpoints: 9 mandatory gates*
*Architecture Compliance: Lock-free immutable state, Redux/Elm pattern, View() deadlock prevention*
*Implementation Status: Ready for execution*
