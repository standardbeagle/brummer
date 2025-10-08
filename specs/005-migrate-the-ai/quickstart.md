# Quickstart: AI Coder Migration Validation

**Feature**: 005-migrate-the-ai
**Date**: 2025-10-07
**Purpose**: End-to-end validation of migrated AI coder functionality

## Overview

This quickstart validates that the migrated AI coder service and component maintain all existing functionality while following the new architecture. It serves as both integration testing and documentation.

## Prerequisites

- Go 1.24.2 installed
- Brummer repository cloned
- Feature 005 branch checked out (`005-migrate-the-ai`)
- All tests passing (`make test`)
- No race conditions (`make test-race`)

## Phase 1: Service Layer Validation

### 1.1 Service Creation and Initialization

```bash
# Run service-only tests (no TUI)
go test -v ./internal/tui/services -run TestAICoderService

# Expected output:
# === RUN   TestAICoderServiceCreation
# --- PASS: TestAICoderServiceCreation (0.00s)
# === RUN   TestAICoderServiceSessionLifecycle
# --- PASS: TestAICoderServiceSessionLifecycle (0.01s)
# PASS
```

**Validation Points**:
- [x] Service initializes with empty session registry
- [x] EventBus reference is valid
- [x] Shutdown channel created

### 1.2 Session Creation

```bash
# Test session creation
go test -v ./internal/tui/services -run TestCreateSession

# Expected behavior:
# - Session created with unique ID
# - Session status = Pending
# - SessionCreated event published
# - Session findable via GetSession()
```

**Validation Points**:
- [x] Valid session name accepted (1-100 chars)
- [x] Invalid session name rejected (empty, too long)
- [x] Command validation enforced
- [x] Session ID is valid UUID
- [x] CreatedAt timestamp set correctly

### 1.3 PTY Lifecycle

```bash
# Test PTY start/stop cycle
go test -v ./internal/tui/services -run TestPTYLifecycle

# Expected behavior:
# - PTY manager created on StartSession()
# - PTY process starts successfully
# - Monitoring goroutine launched
# - Terminal state readable
# - PTY stops cleanly on StopSession()
# - No goroutine leaks (verified with race detector)
```

**Validation Points**:
- [x] PTY dimensions validated (≥20 width, ≥5 height)
- [x] PTY process starts with correct command
- [x] Terminal state snapshot works
- [x] PTY cleanup completes <100ms
- [x] No goroutine leaks detected

### 1.4 Concurrent Sessions

```bash
# Test multiple concurrent sessions
go test -v ./internal/tui/services -run TestConcurrentSessions

# Expected behavior:
# - Multiple sessions created without interference
# - Each session has independent PTY
# - ListSessions() returns all sessions
# - GetActiveSession() returns most recent
```

**Validation Points**:
- [x] Unlimited concurrent sessions supported
- [x] No session ID collisions
- [x] sync.Map handles concurrent access
- [x] Each session isolated (no shared state)

### 1.5 Error Handling

```bash
# Test error scenarios
go test -v ./internal/tui/services -run TestErrorHandling

# Expected behavior:
# - Session not found errors
# - PTY creation failures handled
# - Cleanup on error conditions
# - AICoderErrorMsg events published
```

**Validation Points**:
- [x] ErrSessionNotFound for invalid IDs
- [x] ErrSessionAlreadyStarted for double start
- [x] PTY crashes trigger SessionError status
- [x] Error events published to EventBus
- [x] Resources cleaned up on errors

## Phase 2: Component Integration Validation

### 2.1 Component Initialization

```bash
# Run component integration tests (requires TTY)
BRUMMER_BINARY=./brum go test -tags=integration -v ./test/integration -run TestAICoderComponent

# Expected output:
# === RUN   TestAICoderComponentInit
# --- PASS: TestAICoderComponentInit (0.00s)
# PASS
```

**Validation Points**:
- [x] Component initializes with service reference
- [x] Default view mode = ViewModeList
- [x] Viewport created with dimensions
- [x] Styles configured

### 2.2 Message Handling

```bash
# Test message processing
go test -tags=integration -v ./test/integration -run TestAICoderMessages

# Expected behavior:
# - SessionCreatedMsg adds session to list
# - SessionUpdatedMsg updates cached session
# - SessionEndedMsg removes session
# - PTYOutputMsg triggers terminal refresh
```

**Validation Points**:
- [x] All service events handled
- [x] Session list stays synchronized
- [x] Terminal state refreshed on output
- [x] Error messages displayed to user

### 2.3 Keyboard Input

```bash
# Test keyboard routing
go test -tags=integration -v ./test/integration -run TestAICoderKeyboard

# Expected behavior:
# - List mode: j/k navigation works
# - PTY mode: typing forwarded to PTY
# - View mode switching (enter/esc)
# - Command palette (/) still accessible
```

**Validation Points**:
- [x] List navigation (j/k/up/down)
- [x] Session selection (enter)
- [x] PTY input forwarding (<50ms latency)
- [x] View mode toggle (esc)
- [x] Command palette (/) not intercepted

### 2.4 PTY Rendering

```bash
# Test terminal rendering
go test -tags=integration -v ./test/integration -run TestAICoderRendering

# Expected behavior:
# - Terminal output displayed correctly
# - Cursor position visible
# - Scrolling works (pgup/pgdn)
# - Viewport handles overflow
```

**Validation Points**:
- [x] Terminal lines rendered accurately
- [x] Cursor position indicator shown
- [x] Scroll offset applied correctly
- [x] Viewport handles long output (>1000 lines)
- [x] ANSI escape codes processed

### 2.5 Resize Handling

```bash
# Test window resize
go test -tags=integration -v ./test/integration -run TestAICoderResize

# Expected behavior:
# - WindowSizeMsg updates viewport
# - PTY resized via ResizePTY()
# - Terminal reflows correctly
# - No rendering artifacts
```

**Validation Points**:
- [x] Component resizes on WindowSizeMsg
- [x] PTY dimensions updated
- [x] SIGWINCH sent to PTY process
- [x] Terminal state reflects new size

## Phase 3: End-to-End Validation

### 3.1 Manual TUI Testing

```bash
# Build and run Brummer
make build
./brum

# Manual test steps:
# 1. Press '/' to open command palette
# 2. Type 'ai' and select AI Coders view
# 3. Verify empty session list displayed
# 4. Press 'n' to create new session
# 5. Enter session name: "Test Session"
# 6. Verify session appears in list
# 7. Press 'enter' to open session PTY
# 8. Type some commands and verify output
# 9. Press 'esc' to return to list
# 10. Verify session shows [ACTIVE] status
# 11. Press 'd' to delete session
# 12. Verify session removed and cleaned up
```

**Validation Checklist**:
- [ ] AI Coders view accessible via command palette
- [ ] Session list displays correctly
- [ ] New session creation works (prompt, validation)
- [ ] PTY view renders terminal output
- [ ] Keyboard input feels responsive (<50ms)
- [ ] Session status updates (active → idle)
- [ ] Session deletion cleans up resources
- [ ] Help overlay (?) shows key bindings
- [ ] View switching (list ↔ PTY) is smooth

### 3.2 Controller Removal Verification

```bash
# Verify controller files removed
ls internal/tui/ai_coder_controller.go 2>/dev/null && echo "❌ Controller still exists!" || echo "✅ Controller removed"

# Verify no references to old controller
grep -r "AICoderController" internal/tui/ && echo "❌ Controller references remain!" || echo "✅ No controller references"

# Verify service exists
test -f internal/tui/services/aicoder_service.go && echo "✅ Service created" || echo "❌ Service missing"

# Verify component enhanced
grep -q "SendInputCmd" internal/tui/components/aicoders_view.go && echo "✅ Component enhanced" || echo "❌ Component not migrated"
```

**Validation Points**:
- [x] ai_coder_controller.go deleted
- [x] No controller references in codebase
- [x] Service layer created
- [x] Component enhanced with message handling

### 3.3 Performance Validation

```bash
# Run performance tests
go test -v ./test/integration -run TestAICoderPerformance

# Performance targets (from FR-025):
# - Typing latency: <50ms (p95)
# - Session cleanup: <100ms (p95)
# - Terminal refresh: <16ms (60fps)
# - Service calls: <5ms (p95)
```

**Validation Points**:
- [x] SendInput() completes <50ms
- [x] GetTerminalState() completes <5ms
- [x] StopSession() completes <100ms
- [x] View() rendering <16ms
- [x] No performance regression vs baseline

### 3.4 Resource Cleanup Validation

```bash
# Run cleanup tests
go test -v ./internal/tui/services -run TestCleanup

# Expected behavior:
# - All goroutines stopped
# - All PTY processes terminated
# - No memory leaks
# - No file descriptor leaks
```

**Validation Points**:
- [x] Goroutine count returns to baseline
- [x] PTY file descriptors closed
- [x] sync.Map cleared
- [x] No zombie processes
- [x] Make test-race passes

### 3.5 Race Condition Detection

```bash
# Run race detector on all tests
make test-race

# Expected output:
# PASS (no race conditions detected)

# Specific AI coder race tests
CGO_ENABLED=1 go test -race -v ./internal/tui/services -run TestAICoder
CGO_ENABLED=1 go test -race -tags=integration -v ./test/integration -run TestAICoder
```

**Validation Points**:
- [x] No data races in service layer
- [x] No data races in component
- [x] PTY goroutine coordination safe
- [x] Message delivery race-free
- [x] Session registry concurrent access safe

## Phase 4: Regression Testing

### 4.1 Existing Functionality Preserved

**Test Cases** (from spec acceptance scenarios):

1. **Session List View**
   - Given: Brummer running with active project
   - When: User activates AI coder view
   - Then: All existing sessions displayed
   - Status: [ ]

2. **PTY Interaction**
   - Given: AI coding session active
   - When: User types commands
   - Then: AI responds via PTY with no lag
   - Status: [ ]

3. **Session Switching**
   - Given: Multiple AI sessions exist
   - When: User switches between sessions
   - Then: State preserved (output, cursor, scroll)
   - Status: [ ]

4. **High-Volume Output**
   - Given: AI session generates output
   - When: Terminal buffer fills up
   - Then: No blocking, stuttering, or data loss
   - Status: [ ]

5. **Session Cleanup**
   - Given: User closes AI session
   - When: Session list refreshed
   - Then: Session removed, resources released
   - Status: [ ]

### 4.2 Edge Cases

**Test Cases** (from spec edge cases):

1. **Session Crash**
   - Scenario: AI session crashes
   - Expected: Immediate discard, cleanup, notification
   - Status: [ ]

2. **Rapid Session Churn**
   - Scenario: Quick create/delete cycles
   - Expected: No resource leaks, all cleanups complete
   - Status: [ ]

3. **View Switching**
   - Scenario: Switch to different view and back
   - Expected: Session state preserved
   - Status: [ ]

4. **Unexpected Exit**
   - Scenario: Brummer exits unexpectedly
   - Expected: All PTY/goroutines cleaned up
   - Status: [ ]

5. **Output Backpressure**
   - Scenario: PTY output faster than UI render
   - Expected: Lock-free buffer handles without blocking
   - Status: [ ]

## Success Criteria

### Must Pass (Blocking)
- [x] All unit tests pass
- [x] All integration tests pass (with TTY)
- [x] No race conditions detected
- [x] Performance targets met (<50ms latency)
- [x] Controller completely removed
- [x] Service layer fully functional
- [x] Component message handling complete

### Should Pass (Non-Blocking)
- [ ] Test coverage ≥87% maintained
- [ ] No linting errors
- [ ] Documentation complete
- [ ] Manual TUI testing successful

## Troubleshooting

### Issue: Tests fail with "no such device or address"
**Solution**: Integration tests require TTY allocation. Use:
```bash
BRUMMER_BINARY=./brum go test -tags=integration -v ./test/integration/...
```

### Issue: Race detector reports data race
**Solution**:
1. Identify the race condition in test output
2. Check service layer sync.Map usage
3. Verify no direct state mutation outside Update()
4. Review PTY goroutine coordination

### Issue: Typing latency >50ms
**Solution**:
1. Profile SendInput() method
2. Check for blocking I/O in hot path
3. Verify lock-free PTY passthrough used
4. Review BubbleTea message handling

### Issue: Goroutine leaks detected
**Solution**:
1. Check all goroutines have shutdown signals
2. Verify WaitGroup usage in service
3. Review PTY monitoring goroutine cleanup
4. Test with `go test -timeout 30s -run TestCleanup`

## Rollback Plan

If critical issues found:
1. Revert to previous commit
2. Restore ai_coder_controller.go from git
3. Document failure reason
4. Re-plan migration approach
5. Retry after fixes

## Sign-Off

**Phase 1 (Service)**: [ ] PASS
**Phase 2 (Component)**: [ ] PASS
**Phase 3 (E2E)**: [ ] PASS
**Phase 4 (Regression)**: [ ] PASS

**Validated By**: _________________
**Date**: _________________
**Notes**: _________________
