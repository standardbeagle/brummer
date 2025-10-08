# Tasks: Migrate AI Coder to Service and Component Architecture

**Feature**: 005-migrate-the-ai
**Branch**: `005-migrate-the-ai`
**Status**: Ready for implementation
**Created**: 2025-10-07

## Overview

This document breaks down the AI coder migration into executable tasks following a two-phase delivery strategy:
- **Phase 1**: Extract service layer (T001-T016) - Can be validated independently
- **Phase 2**: Migrate component and remove controller (T017-T038) - Builds on validated service

All tasks follow TDD (Test-Driven Development) - tests are written before implementation.

## Task Execution Strategy

### Parallel Execution

Tasks marked **[P]** can be executed in parallel as they work on independent files:

```bash
# Phase 1 - Parallel test creation (T002-T004)
# These can run simultaneously as they create different test files
Task 1: "Create service creation and initialization tests"
Task 2: "Create session lifecycle tests"
Task 3: "Create PTY integration tests"

# Phase 1 - Parallel cleanup tests (T011-T013)
Task 1: "Create goroutine leak detection tests"
Task 2: "Create resource cleanup verification tests"
Task 3: "Create shutdown timeout tests"

# Phase 2 - Parallel component tests (T018-T021)
Task 1: "Create component initialization tests"
Task 2: "Create message handling tests"
Task 3: "Create keyboard routing tests"
Task 4: "Create PTY rendering tests"
```

### Sequential Execution

Tasks without **[P]** must run sequentially as they modify the same files or depend on previous tasks.

---

## Phase 1: Service Layer Extraction (T001-T016)

### Setup Tasks

#### T001: Create service directory structure
**Type**: [SETUP]
**Files**:
- Create: `internal/tui/services/` (directory)

**Acceptance Criteria**:
- [x] Directory `internal/tui/services/` exists
- [x] Directory is committed to git

**Commands**:
```bash
mkdir -p internal/tui/services
git add internal/tui/services
```

**Estimated Time**: 1 minute

---

#### T002: Create message types file [P]
**Type**: [SETUP]
**Files**:
- Create: `internal/tui/messages/aicoder_messages.go`

**Acceptance Criteria**:
- [x] File defines all message types from contracts/aicoder_component.go
- [x] Session event messages (Created, Updated, Ended)
- [x] PTY event messages (Output, Error)
- [x] Command messages (StartSession, StopSession, SendInput, ResizePTY)
- [x] Internal component messages (SessionList, TerminalState)
- [x] All messages have godoc comments
- [x] File compiles without errors

**Implementation Notes**:
- Reference: `contracts/aicoder_component.go` lines 176-197
- Reference: `data-model.md` "Message Types" section
- Follow existing message patterns in `internal/tui/messages/`

**Commands**:
```bash
# After implementation
go build ./internal/tui/messages
```

**Estimated Time**: 15 minutes

---

### Service Tests (TDD - Tests First)

#### T003: Create service creation and initialization tests [P]
**Type**: [TEST]
**Files**:
- Create: `internal/tui/services/aicoder_service_test.go`

**Acceptance Criteria**:
- [x] Test: `TestNewAICoderService` - service initializes with empty registry
- [x] Test: `TestServiceEventBusReference` - EventBus is valid
- [x] Test: `TestServiceShutdownChannel` - shutdown channel created
- [x] All tests fail initially (no implementation yet) - THEN PASS after implementation
- [x] Tests follow existing patterns in `internal/tui/services/` (if any exist)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 1.1 "Service Creation and Initialization"
- Reference: `contracts/aicoder_service.go` for interface contract
- Use table-driven tests where appropriate

**Estimated Time**: 20 minutes

---

#### T004: Create session lifecycle tests [P]
**Type**: [TEST]
**Files**:
- Modify: `internal/tui/services/aicoder_service_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestCreateSession_Valid` - accepts valid session parameters
- [ ] Test: `TestCreateSession_InvalidName` - rejects empty/long names
- [ ] Test: `TestCreateSession_InvalidCommand` - rejects empty command
- [ ] Test: `TestStartSession_Valid` - starts PTY successfully
- [ ] Test: `TestStartSession_InvalidDimensions` - rejects small dimensions
- [ ] Test: `TestStopSession_Valid` - stops cleanly
- [ ] Test: `TestStopSession_Idempotent` - safe to call twice
- [ ] Test: `TestSessionStatusTransitions` - validates state machine
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 1.2-1.3
- Reference: `data-model.md` "State Transitions" section
- Validate session status transitions: Pending → Active → Stopped

**Estimated Time**: 30 minutes

---

#### T005: Create PTY integration tests [P]
**Type**: [TEST]
**Files**:
- Modify: `internal/tui/services/aicoder_service_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestPTYStartup` - PTY manager created on StartSession
- [ ] Test: `TestPTYProcessExecution` - PTY process starts with correct command
- [ ] Test: `TestGetTerminalState` - terminal state readable
- [ ] Test: `TestSendInput` - keyboard input forwarded to PTY
- [ ] Test: `TestResizePTY` - terminal dimensions update
- [ ] Test: `TestPTYCleanup` - PTY stops cleanly, <100ms
- [ ] All tests fail initially (TDD)
- [ ] Tests use real PTY (not mocked)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 1.3 "PTY Lifecycle"
- PTY tests require real terminal allocation
- Verify cleanup timing with time.Since()

**Estimated Time**: 40 minutes

---

#### T006: Create concurrent sessions tests [P]
**Type**: [TEST]
**Files**:
- Modify: `internal/tui/services/aicoder_service_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestConcurrentSessionCreation` - multiple sessions without interference
- [ ] Test: `TestSessionIsolation` - sessions have independent PTY state
- [ ] Test: `TestListSessions` - returns all sessions
- [ ] Test: `TestGetSession` - retrieves specific session
- [ ] Test: `TestGetActiveSession` - returns most recent
- [ ] Test: `TestNoSessionIDCollisions` - UUIDs are unique
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 1.4 "Concurrent Sessions"
- Test with 3-5 concurrent sessions
- Verify sync.Map handles concurrent access

**Estimated Time**: 25 minutes

---

#### T007: Create error handling tests [P]
**Type**: [TEST]
**Files**:
- Modify: `internal/tui/services/aicoder_service_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestSessionNotFound` - ErrSessionNotFound for invalid IDs
- [ ] Test: `TestSessionAlreadyStarted` - ErrSessionAlreadyStarted on double start
- [ ] Test: `TestPTYCreationFailure` - handles PTY errors gracefully
- [ ] Test: `TestPTYCrashRecovery` - session Error status on PTY crash
- [ ] Test: `TestErrorEventPublication` - AICoderErrorMsg published
- [ ] Test: `TestResourceCleanupOnError` - cleanup on error conditions
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 1.5 "Error Handling"
- Reference: `contracts/aicoder_service.go` error definitions
- Simulate PTY crashes for testing

**Estimated Time**: 30 minutes

---

### Service Implementation

#### T008: Implement AICoderService struct and constructor
**Type**: [IMPL]
**Files**:
- Create: `internal/tui/services/aicoder_service.go`

**Acceptance Criteria**:
- [x] Struct matches design in `data-model.md` "AICoderService" section
- [x] Uses sync.Map for session registry
- [x] References EventBus for event publishing
- [x] Has shutdown channel and WaitGroup
- [x] Constructor `NewAICoderService()` implemented
- [x] Tests T003 now pass (service creation tests)
- [x] No race conditions detected (`go test -race`)

**Implementation Notes**:
- Reference: `contracts/aicoder_service.go` interface
- Reference: `data-model.md` "Service State" section
- Initialize with empty sync.Map, valid EventBus reference

**Commands**:
```bash
go test -v ./internal/tui/services -run TestNewAICoderService
go test -race ./internal/tui/services -run TestNewAICoderService
```

**Estimated Time**: 20 minutes

---

#### T009: Implement session creation and lifecycle methods
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/services/aicoder_service.go`

**Acceptance Criteria**:
- [x] `CreateSession()` implemented with validation
- [~] `StartSession()` creates PTY manager and monitoring goroutine (TODO: needs PTY integration)
- [~] `StopSession()` stops PTY and cleans up (basic implementation, needs PTY cleanup)
- [x] Session status transitions enforced
- [x] EventBus events published (Created, Updated, Ended)
- [ ] Tests T004 now pass (session lifecycle tests) - BLOCKED by PTY integration
- [ ] No race conditions detected

**Implementation Notes**:
- Reference: `contracts/aicoder_service.go` method signatures
- Reference: `data-model.md` "State Transitions"
- Use existing LockFreePTYManager (don't modify PTY infrastructure)
- Monitoring goroutine must respect shutdown channel

**Commands**:
```bash
go test -v ./internal/tui/services -run TestCreateSession
go test -v ./internal/tui/services -run TestStartSession
go test -v ./internal/tui/services -run TestStopSession
go test -race ./internal/tui/services -run TestSession
```

**Estimated Time**: 60 minutes

---

#### T010: Implement PTY interaction methods
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/services/aicoder_service.go`

**Acceptance Criteria**:
- [~] `GetTerminalState()` returns read-only snapshot (placeholder implementation)
- [~] `SendInput()` forwards to PTY with <50ms latency (placeholder implementation)
- [~] `ResizePTY()` updates dimensions and sends SIGWINCH (placeholder implementation)
- [ ] Terminal state derived from PTY lock-free buffer - BLOCKED by PTY integration
- [ ] Tests T005 now pass (PTY integration tests) - BLOCKED by PTY integration
- [ ] Performance requirement met (<50ms SendInput) - BLOCKED by PTY integration
- [ ] No race conditions detected

**Implementation Notes**:
- Reference: `contracts/aicoder_service.go` PTY methods
- Reference: `research.md` Decision 4 "PTY Ownership"
- Read from existing LockFreePTYManager buffer (lock-free)
- Input forwarding must be non-blocking

**Commands**:
```bash
go test -v ./internal/tui/services -run TestPTY
go test -v ./internal/tui/services -run TestSendInput
go test -race ./internal/tui/services
```

**Estimated Time**: 45 minutes

---

#### T011: Implement session query methods
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/services/aicoder_service.go`

**Acceptance Criteria**:
- [x] `ListSessions()` returns sorted session summaries
- [x] `GetSession()` retrieves session by ID
- [x] `GetActiveSession()` returns most recent session ID
- [x] Efficient sync.Map iteration
- [ ] Tests T006 now pass (concurrent sessions tests) - Needs comprehensive tests
- [ ] No race conditions detected - Needs comprehensive tests

**Implementation Notes**:
- Reference: `contracts/aicoder_service.go` query methods
- Sort sessions by CreatedAt (oldest first) or LastActivity (most recent)
- Use sync.Map.Range() for iteration

**Commands**:
```bash
go test -v ./internal/tui/services -run TestListSessions
go test -v ./internal/tui/services -run TestGetSession
go test -race ./internal/tui/services -run TestConcurrent
```

**Estimated Time**: 30 minutes

---

#### T012: Implement error handling and event publishing
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/services/aicoder_service.go`

**Acceptance Criteria**:
- [x] All sentinel errors defined (ErrSessionNotFound, etc.)
- [x] Error validation in all public methods
- [ ] AICoderErrorMsg published on errors - Partially done (needs PTY error handling)
- [ ] PTY crash detection triggers SessionError status - BLOCKED by PTY integration
- [ ] Resource cleanup on error paths - Basic cleanup, needs PTY cleanup
- [ ] Tests T007 now pass (error handling tests) - Needs comprehensive tests
- [ ] No resource leaks on error paths - Needs comprehensive tests

**Implementation Notes**:
- Reference: `contracts/aicoder_service.go` error definitions
- Reference: `quickstart.md` Phase 1.5
- Publish error events to EventBus topic "aicoder.error"

**Commands**:
```bash
go test -v ./internal/tui/services -run TestError
go test -v ./internal/tui/services -run TestPTYCrash
```

**Estimated Time**: 30 minutes

---

### Service Cleanup Tests

#### T013: Create goroutine leak detection tests [P]
**Type**: [TEST]
**Files**:
- Modify: `internal/tui/services/aicoder_service_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestNoGoroutineLeaks_SessionLifecycle` - goroutine count returns to baseline
- [ ] Test: `TestNoGoroutineLeaks_MultipleStarts` - no leaks with rapid create/stop
- [ ] Test: `TestMonitoringGoroutineCleanup` - monitoring goroutines stopped
- [ ] Use `runtime.NumGoroutine()` to verify counts
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 3.4 "Resource Cleanup Validation"
- Capture goroutine count before/after operations
- Allow for test framework goroutines (use delta comparison)

**Estimated Time**: 25 minutes

---

#### T014: Create resource cleanup verification tests [P]
**Type**: [TEST]
**Files**:
- Modify: `internal/tui/services/aicoder_service_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestPTYFileDescriptorCleanup` - PTY fds closed
- [ ] Test: `TestSyncMapCleanup` - session registry cleared
- [ ] Test: `TestNoZombieProcesses` - PTY processes terminated
- [ ] Test: `TestCleanupTiming` - cleanup completes <100ms
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 3.4
- Use `os.File.Fd()` to track file descriptors
- Check for zombie processes with `ps` or process table

**Estimated Time**: 30 minutes

---

#### T015: Create shutdown timeout tests [P]
**Type**: [TEST]
**Files**:
- Modify: `internal/tui/services/aicoder_service_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestShutdown_AllSessionsStopped` - all sessions cleaned up
- [ ] Test: `TestShutdown_GracefulTimeout` - respects context timeout
- [ ] Test: `TestShutdown_Idempotent` - safe to call multiple times
- [ ] Test: `TestShutdown_NoDeadlock` - completes even with active sessions
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `contracts/aicoder_service.go` Shutdown method
- Test with 10s context timeout
- Verify WaitGroup.Wait() behavior

**Estimated Time**: 20 minutes

---

#### T016: Implement Shutdown and cleanup logic
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/services/aicoder_service.go`

**Acceptance Criteria**:
- [ ] `Shutdown()` stops all sessions
- [ ] Close shutdown channel to signal goroutines
- [ ] WaitGroup.Wait() ensures all goroutines stopped
- [ ] Respects context timeout
- [ ] Idempotent (safe to call multiple times)
- [ ] Tests T013-T015 now pass (cleanup tests)
- [ ] No goroutine leaks detected
- [ ] No resource leaks detected
- [ ] `make test-race` passes for service package

**Implementation Notes**:
- Reference: `contracts/aicoder_service.go` Shutdown method
- Reference: `quickstart.md` Phase 3.4
- Iterate sync.Map and stop all sessions
- Use select with context.Done() for timeout

**Commands**:
```bash
go test -v ./internal/tui/services -run TestShutdown
go test -v ./internal/tui/services -run TestNoGoroutineLeaks
make test-race  # Should pass for service package
```

**Estimated Time**: 40 minutes

---

**Phase 1 Checkpoint**: All service tests should pass before proceeding to Phase 2

```bash
# Verify Phase 1 completion
go test -v ./internal/tui/services
go test -race ./internal/tui/services
go test -coverprofile=coverage.out ./internal/tui/services
go tool cover -html=coverage.out  # Should show ≥87% coverage
```

---

## Phase 2: Component Migration and Controller Removal (T017-T038)

### Component Tests (TDD - Tests First)

#### T017: Create component initialization tests [P]
**Type**: [TEST]
**Files**:
- Create: `test/integration/aicoder_component_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestComponentInit` - component initializes with service reference
- [ ] Test: `TestDefaultViewMode` - view mode = ViewModeList
- [ ] Test: `TestViewportCreation` - viewport created with dimensions
- [ ] Test: `TestStylesConfigured` - all styles initialized
- [ ] All tests fail initially (TDD)
- [ ] Tests use integration test tag

**Implementation Notes**:
- Reference: `quickstart.md` Phase 2.1 "Component Initialization"
- Reference: `contracts/aicoder_component.go` interface
- Tests require TTY allocation (integration tag)

**Commands**:
```bash
go test -tags=integration -v ./test/integration -run TestComponentInit
```

**Estimated Time**: 20 minutes

---

#### T018: Create message handling tests [P]
**Type**: [TEST]
**Files**:
- Modify: `test/integration/aicoder_component_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestSessionCreatedMsg` - adds session to cached list
- [ ] Test: `TestSessionUpdatedMsg` - updates cached session
- [ ] Test: `TestSessionEndedMsg` - removes session from list
- [ ] Test: `TestPTYOutputMsg` - triggers terminal refresh
- [ ] Test: `TestErrorMsg` - displays error to user
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 2.2 "Message Handling"
- Reference: `data-model.md` "Message Types"
- Test message flow through BubbleTea Update() method

**Estimated Time**: 30 minutes

---

#### T019: Create keyboard routing tests [P]
**Type**: [TEST]
**Files**:
- Modify: `test/integration/aicoder_component_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestListModeNavigation` - j/k/up/down navigation
- [ ] Test: `TestSessionSelection` - enter opens PTY view
- [ ] Test: `TestPTYInputForwarding` - typing forwarded to service
- [ ] Test: `TestViewModeToggle` - esc returns to list
- [ ] Test: `TestCommandPaletteNotIntercepted` - / still works
- [ ] Test: `TestTypingLatency` - input forwarding <50ms
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 2.3 "Keyboard Input"
- Reference: `contracts/aicoder_component.go` "Key Bindings"
- Measure latency with time.Since()

**Estimated Time**: 35 minutes

---

#### T020: Create PTY rendering tests [P]
**Type**: [TEST]
**Files**:
- Modify: `test/integration/aicoder_component_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestTerminalLinesRendered` - terminal output displayed correctly
- [ ] Test: `TestCursorPositionVisible` - cursor indicator shown
- [ ] Test: `TestScrollOffset` - pgup/pgdn scrolling works
- [ ] Test: `TestViewportOverflow` - handles >1000 lines
- [ ] Test: `TestANSIEscapeCodes` - escape codes processed
- [ ] Test: `TestRenderPerformance` - View() completes <16ms
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 2.4 "PTY Rendering"
- Reference: `contracts/aicoder_component.go` RenderPTYView
- Test with long output (1000+ lines)

**Estimated Time**: 40 minutes

---

#### T021: Create resize handling tests [P]
**Type**: [TEST]
**Files**:
- Modify: `test/integration/aicoder_component_test.go`

**Acceptance Criteria**:
- [ ] Test: `TestWindowSizeMsg` - viewport resizes
- [ ] Test: `TestPTYResize` - ResizePTY called on service
- [ ] Test: `TestTerminalReflow` - terminal reflows correctly
- [ ] Test: `TestNoRenderingArtifacts` - clean display after resize
- [ ] All tests fail initially (TDD)

**Implementation Notes**:
- Reference: `quickstart.md` Phase 2.5 "Resize Handling"
- Simulate WindowSizeMsg in tests
- Verify ResizePTYCmd issued

**Estimated Time**: 25 minutes

---

### Component Implementation

#### T022: Enhance AICodersViewComponent with service integration
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [ ] Add service reference to component struct
- [ ] Update constructor to accept service parameter
- [ ] Add viewState field (SessionViewState)
- [ ] Add cached sessions and terminal state fields
- [ ] Tests T017 now pass (component initialization)
- [ ] File compiles without errors

**Implementation Notes**:
- Reference: `data-model.md` "Component State" section
- Reference: existing component at `internal/tui/components/aicoders_view.go`
- Preserve existing session list display code

**Commands**:
```bash
go test -tags=integration -v ./test/integration -run TestComponentInit
```

**Estimated Time**: 20 minutes

---

#### T023: Implement BubbleTea message handlers
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [ ] Handle SessionCreatedMsg - update session list
- [ ] Handle SessionUpdatedMsg - update cached session
- [ ] Handle SessionEndedMsg - remove from list
- [ ] Handle PTYOutputMsg - issue FetchTerminalStateCmd
- [ ] Handle ErrorMsg - display notification
- [ ] Handle SessionListMsg - cache sessions
- [ ] Handle TerminalStateMsg - cache terminal state
- [ ] Tests T018 now pass (message handling tests)

**Implementation Notes**:
- Reference: `contracts/aicoder_component.go` Update method
- Reference: `data-model.md` "Message Types"
- All state modifications in Update() method only

**Commands**:
```bash
go test -tags=integration -v ./test/integration -run TestSessionCreatedMsg
go test -tags=integration -v ./test/integration -run TestMessage
```

**Estimated Time**: 45 minutes

---

#### T024: Implement keyboard input routing
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [ ] List mode: j/k navigation
- [ ] List mode: enter opens session
- [ ] PTY mode: forward typing to PTY
- [ ] PTY mode: esc returns to list
- [ ] Global: / opens command palette (not intercepted)
- [ ] Typing latency <50ms verified
- [ ] Tests T019 now pass (keyboard routing tests)

**Implementation Notes**:
- Reference: `contracts/aicoder_component.go` "Key Bindings"
- Reference: spec.md FR-025 for latency requirement
- Issue SendInputCmd for PTY input

**Commands**:
```bash
go test -tags=integration -v ./test/integration -run TestKeyboard
go test -tags=integration -v ./test/integration -run TestTypingLatency
```

**Estimated Time**: 40 minutes

---

#### T025: Implement PTY terminal rendering
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [ ] View() renders terminal lines from cached state
- [ ] Cursor position indicator displayed
- [ ] Scroll offset applied (pgup/pgdn)
- [ ] Viewport handles overflow (>1000 lines)
- [ ] ANSI escape codes processed
- [ ] Rendering completes <16ms (60fps)
- [ ] Tests T020 now pass (PTY rendering tests)

**Implementation Notes**:
- Reference: `contracts/aicoder_component.go` RenderPTYView
- Reference: existing PTY view rendering (ai_coder_pty_view.go) for patterns
- Use cached TerminalState (no service calls in View())

**Commands**:
```bash
go test -tags=integration -v ./test/integration -run TestTerminal
go test -tags=integration -v ./test/integration -run TestRenderPerformance
```

**Estimated Time**: 50 minutes

---

#### T026: Implement command constructors
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [ ] FetchSessionListCmd - calls service.ListSessions()
- [ ] FetchTerminalStateCmd - calls service.GetTerminalState()
- [ ] StartSessionCmd - calls service.CreateSession() + StartSession()
- [ ] StopSessionCmd - calls service.StopSession()
- [ ] SendInputCmd - calls service.SendInput()
- [ ] All commands return appropriate messages or errors
- [ ] Commands are async (tea.Cmd functions)

**Implementation Notes**:
- Reference: `contracts/aicoder_component.go` command constructors
- Use context.Background() or component-provided context
- Return error messages on failures

**Commands**:
```bash
go test -tags=integration -v ./test/integration -run TestCommand
```

**Estimated Time**: 30 minutes

---

#### T027: Implement resize handling
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [ ] Handle WindowSizeMsg - resize viewport
- [ ] Issue ResizePTYCmd when in PTY mode
- [ ] Update viewState dimensions
- [ ] Tests T021 now pass (resize handling tests)
- [ ] No rendering artifacts after resize

**Implementation Notes**:
- Reference: `quickstart.md` Phase 2.5
- Call service.ResizePTY() when active session exists
- Update viewport.Width and viewport.Height

**Commands**:
```bash
go test -tags=integration -v ./test/integration -run TestResize
```

**Estimated Time**: 20 minutes

---

### Integration and Wiring

#### T028: Wire component to service in model.go
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/model.go`

**Acceptance Criteria**:
- [ ] Create AICoderService in model initialization
- [ ] Pass service to AICodersViewComponent constructor
- [ ] Subscribe to service EventBus topics
- [ ] Forward service events as BubbleTea messages
- [ ] Update shutdown to call service.Shutdown()
- [ ] Component accessible via integrated_shell routing

**Implementation Notes**:
- Reference: existing service wiring patterns in model.go
- Subscribe to topics: "aicoder.session.*", "aicoder.pty.output", "aicoder.error"
- Add service.Shutdown() to model cleanup

**Dependencies**: Requires T022-T027 (component implementation)

**Commands**:
```bash
go build ./internal/tui
```

**Estimated Time**: 30 minutes

---

### Controller Removal

#### T029: Remove ai_coder_controller.go
**Type**: [CLEAN] [P]
**Files**:
- Delete: `internal/tui/ai_coder_controller.go`
- Delete: `internal/tui/ai_coder_controller_test.go`

**Acceptance Criteria**:
- [ ] Files deleted
- [ ] No references to AICoderController in codebase
- [ ] Build succeeds without controller

**Implementation Notes**:
- Run grep to verify no remaining references:
  ```bash
  grep -r "AICoderController" internal/tui/
  ```
- Should return no results

**Dependencies**: Requires T028 (component wired to service)

**Commands**:
```bash
rm internal/tui/ai_coder_controller.go
rm internal/tui/ai_coder_controller_test.go
git rm internal/tui/ai_coder_controller.go internal/tui/ai_coder_controller_test.go
go build ./internal/tui
grep -r "AICoderController" internal/tui/ || echo "✓ No controller references"
```

**Estimated Time**: 5 minutes

---

#### T030: Remove ai_coder_pty_view.go [P]
**Type**: [CLEAN]
**Files**:
- Delete: `internal/tui/ai_coder_pty_view.go`
- Delete: `internal/tui/ai_coder_pty_view_test.go`

**Acceptance Criteria**:
- [ ] Files deleted
- [ ] PTY view logic migrated to service
- [ ] No references to AICoderPTYView in codebase
- [ ] Build succeeds

**Implementation Notes**:
- PTY rendering logic should be in component (T025)
- PTY manager coordination should be in service (T009-T010)

**Dependencies**: Requires T025 (PTY rendering in component)

**Commands**:
```bash
rm internal/tui/ai_coder_pty_view.go
rm internal/tui/ai_coder_pty_view_test.go
git rm internal/tui/ai_coder_pty_view.go internal/tui/ai_coder_pty_view_test.go
go build ./internal/tui
grep -r "AICoderPTYView" internal/tui/ || echo "✓ No PTY view references"
```

**Estimated Time**: 5 minutes

---

#### T031: Remove ai_coder_debug_forwarder.go [P]
**Type**: [CLEAN]
**Files**:
- Delete: `internal/tui/ai_coder_debug_forwarder.go`

**Acceptance Criteria**:
- [ ] File deleted
- [ ] Debug forwarding logic in service (if needed)
- [ ] No references to AICoderDebugForwarder in codebase
- [ ] Build succeeds

**Implementation Notes**:
- Debug forwarder was mostly TODO markers (per Feature 004 cleanup)
- Functionality can be added to service layer if/when needed

**Dependencies**: Requires T009-T010 (service PTY interaction)

**Commands**:
```bash
rm internal/tui/ai_coder_debug_forwarder.go
git rm internal/tui/ai_coder_debug_forwarder.go
go build ./internal/tui
```

**Estimated Time**: 3 minutes

---

#### T032: Remove ai_coder_message_handler.go [P]
**Type**: [CLEAN]
**Files**:
- Delete: `internal/tui/ai_coder_message_handler.go`

**Acceptance Criteria**:
- [ ] File deleted
- [ ] Message types migrated to internal/tui/messages/aicoder_messages.go
- [ ] No references to old message handler in codebase
- [ ] Build succeeds

**Implementation Notes**:
- Message types should be in T002 (aicoder_messages.go)

**Dependencies**: Requires T002 (message types created)

**Commands**:
```bash
rm internal/tui/ai_coder_message_handler.go
git rm internal/tui/ai_coder_message_handler.go
go build ./internal/tui
```

**Estimated Time**: 3 minutes

---

### Validation and Testing

#### T033: Run full test suite
**Type**: [VERIFY]
**Files**: N/A

**Acceptance Criteria**:
- [ ] All unit tests pass: `go test ./...`
- [ ] All integration tests pass: `go test -tags=integration ./test/integration/...`
- [ ] No race conditions: `make test-race`
- [ ] Test coverage ≥87%: `go test -coverprofile=coverage.out ./internal/tui/...`

**Implementation Notes**:
- Reference: `quickstart.md` all phases
- Fix any failing tests before proceeding

**Commands**:
```bash
go test -v ./internal/tui/services
go test -tags=integration -v ./test/integration
make test-race
go test -coverprofile=coverage.out ./internal/tui/...
go tool cover -func=coverage.out | grep total
```

**Estimated Time**: 10 minutes (running tests)

---

#### T034: Execute quickstart Phase 1 validations
**Type**: [VERIFY]
**Files**: N/A

**Acceptance Criteria**:
- [ ] Phase 1.1: Service creation tests pass
- [ ] Phase 1.2: Session creation tests pass
- [ ] Phase 1.3: PTY lifecycle tests pass
- [ ] Phase 1.4: Concurrent sessions tests pass
- [ ] Phase 1.5: Error handling tests pass

**Implementation Notes**:
- Reference: `quickstart.md` Phase 1 sections
- Each checklist item should be verified

**Commands**:
```bash
# Follow quickstart.md Phase 1 commands exactly
go test -v ./internal/tui/services -run TestAICoderService
```

**Estimated Time**: 15 minutes (validation)

---

#### T035: Execute quickstart Phase 2 validations
**Type**: [VERIFY]
**Files**: N/A

**Acceptance Criteria**:
- [ ] Phase 2.1: Component initialization tests pass
- [ ] Phase 2.2: Message handling tests pass
- [ ] Phase 2.3: Keyboard input tests pass
- [ ] Phase 2.4: PTY rendering tests pass
- [ ] Phase 2.5: Resize handling tests pass

**Implementation Notes**:
- Reference: `quickstart.md` Phase 2 sections
- Requires TTY allocation for integration tests

**Commands**:
```bash
# Follow quickstart.md Phase 2 commands
BRUMMER_BINARY=./brum go test -tags=integration -v ./test/integration -run TestAICoderComponent
```

**Estimated Time**: 15 minutes (validation)

---

#### T036: Execute quickstart Phase 3 manual TUI testing
**Type**: [VERIFY]
**Files**: N/A

**Acceptance Criteria**:
- [ ] Build succeeds: `make build`
- [ ] AI Coders view accessible
- [ ] Session creation works
- [ ] PTY view renders correctly
- [ ] Typing feels responsive (<50ms)
- [ ] Session deletion cleans up
- [ ] All manual test steps pass

**Implementation Notes**:
- Reference: `quickstart.md` Phase 3.1 "Manual TUI Testing"
- Follow all 12 manual test steps
- Check for visual glitches or performance issues

**Commands**:
```bash
make build
./brum
# Follow manual test steps from quickstart.md
```

**Estimated Time**: 20 minutes (manual testing)

---

#### T037: Verify controller removal
**Type**: [VERIFY]
**Files**: N/A

**Acceptance Criteria**:
- [ ] ai_coder_controller.go does not exist
- [ ] No "AICoderController" references in codebase
- [ ] Service layer exists at internal/tui/services/aicoder_service.go
- [ ] Component enhanced in internal/tui/components/aicoders_view.go

**Implementation Notes**:
- Reference: `quickstart.md` Phase 3.2 "Controller Removal Verification"
- Run all verification commands

**Commands**:
```bash
ls internal/tui/ai_coder_controller.go 2>/dev/null && echo "❌ Controller still exists!" || echo "✅ Controller removed"
grep -r "AICoderController" internal/tui/ && echo "❌ Controller references remain!" || echo "✅ No controller references"
test -f internal/tui/services/aicoder_service.go && echo "✅ Service created" || echo "❌ Service missing"
grep -q "SendInputCmd" internal/tui/components/aicoders_view.go && echo "✅ Component enhanced" || echo "❌ Component not migrated"
```

**Estimated Time**: 5 minutes

---

#### T038: Execute performance validation
**Type**: [VERIFY]
**Files**: N/A

**Acceptance Criteria**:
- [ ] SendInput() <50ms (p95)
- [ ] GetTerminalState() <5ms (p95)
- [ ] StopSession() <100ms (p95)
- [ ] View() rendering <16ms (60fps)
- [ ] No performance regression vs baseline

**Implementation Notes**:
- Reference: `quickstart.md` Phase 3.3 "Performance Validation"
- Reference: spec.md FR-025 for requirements
- Run performance benchmarks

**Commands**:
```bash
go test -v ./test/integration -run TestAICoderPerformance
# Or run specific benchmark tests
```

**Estimated Time**: 10 minutes

---

## Summary

**Total Tasks**: 38
**Phase 1 (Service Layer)**: 16 tasks
**Phase 2 (Component Migration)**: 22 tasks

**Estimated Total Time**: ~14-16 hours

**Parallel Execution Opportunities**:
- T002-T007: Test creation (6 tasks in parallel)
- T013-T015: Cleanup test creation (3 tasks in parallel)
- T017-T021: Component test creation (5 tasks in parallel)
- T029-T032: File removal (4 tasks in parallel)

**Critical Path**:
1. Setup (T001-T002) → 16 min
2. Service tests (T003-T007) → 145 min
3. Service implementation (T008-T012) → 225 min
4. Cleanup tests + impl (T013-T016) → 115 min
5. **Phase 1 Checkpoint**
6. Component tests (T017-T021) → 150 min
7. Component implementation (T022-T027) → 205 min
8. Integration (T028) → 30 min
9. Cleanup (T029-T032) → 16 min
10. Validation (T033-T038) → 75 min

**Success Criteria**:
- ✅ All tests pass (unit + integration)
- ✅ No race conditions detected
- ✅ Test coverage ≥87%
- ✅ Performance requirements met (<50ms latency)
- ✅ Controller completely removed
- ✅ Manual TUI testing successful
- ✅ Quickstart validation complete

**Next Step**: Execute T001 (Create service directory structure)
