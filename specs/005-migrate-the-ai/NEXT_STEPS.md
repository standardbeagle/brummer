# Next Steps: AI Coder Migration Implementation Guide

**Feature**: 005-migrate-the-ai
**Current Status**: Phase 1 Foundation Complete (30%)
**Last Updated**: 2025-10-08
**For**: Next developer continuing this work

## Quick Start for Continuation

### Prerequisites Check
```bash
# Verify you're on the correct branch
git checkout 005-migrate-the-ai
git pull origin 005-migrate-the-ai

# Verify foundation is in place
ls -la internal/tui/services/aicoder_service.go
ls -la internal/tui/messages/aicoder_messages.go

# Run existing tests
go test -v ./internal/tui/services
# Should show: 3 tests passing (TestNewAICoderService, TestServiceEventBusReference, TestServiceShutdownChannel)
```

### Read These First
1. **IMPLEMENTATION_STATUS.md** - Current state and what's complete
2. **tasks.md** - Full task breakdown with acceptance criteria
3. **contracts/aicoder_service.go** - Service API contract
4. **data-model.md** - Entity definitions

## Critical Path: Complete Phase 1 (8-10 hours)

### Step 1: PTY Integration (T009-T010) - 4-5 hours

**Objective**: Wire the service to the existing PTY infrastructure

**Current State**:
- Service has TODO markers for PTY integration
- PTY infrastructure exists: `lockfree_pty_manager.go`, `direct_pty_passthrough.go`
- `aicoder` package has `PTYSession` type

**Implementation Plan**:

#### 1.1 Understand Existing PTY Infrastructure (30 min)
```bash
# Review existing PTY code
cat internal/tui/lockfree_pty_manager.go | head -200
cat internal/aicoder/pty_session.go  # If exists

# Check how controller currently uses PTY
grep -A 20 "LockFreePTYManager" internal/tui/ai_coder_controller.go
```

**Key Questions to Answer**:
- How is `LockFreePTYManager` initialized?
- What's the relationship between `aicoder.PTYSession` and the manager?
- How does PTY output flow to the UI currently?

#### 1.2 Add PTY Manager to Service (1 hour)
```go
// In aicoder_service.go

import (
    "github.com/standardbeagle/brummer/internal/aicoder"
)

type AICoderSession struct {
    // ... existing fields ...

    // Add PTY integration
    PTYManager *LockFreePTYManager  // If global manager
    PTYSession *aicoder.PTYSession  // If per-session
}

// In NewAICoderService - may need to accept PTY dependencies
func NewAICoderService(ctx context.Context, ptyManager *LockFreePTYManager) *AICoderService {
    // Initialize with PTY capabilities
}
```

**Alternative**: If PTY should be per-session, create it in `StartSession()`.

#### 1.3 Implement StartSession with Real PTY (1.5 hours)
```go
func (s *AICoderService) StartSession(ctx context.Context, sessionID string, width, height int) error {
    // ... validation ...

    // Create PTY session
    ptySession, err := aicoder.NewPTYSession(session.Command, session.WorkingDir, width, height)
    if err != nil {
        return fmt.Errorf("failed to create PTY: %w", err)
    }

    session.PTYSession = ptySession

    // Start monitoring goroutine
    s.wg.Add(1)
    go s.monitorPTYOutput(sessionID, ptySession)

    // Update status
    session.Status = messages.SessionActive
    // ... publish event ...

    return nil
}

// monitorPTYOutput reads from PTY and publishes output events
func (s *AICoderService) monitorPTYOutput(sessionID string, ptySession *aicoder.PTYSession) {
    defer s.wg.Done()

    buffer := make([]byte, 8192)
    sequence := atomic.Uint64{}

    for {
        select {
        case <-s.shutdownCh:
            return
        default:
            n, err := ptySession.Read(buffer)
            if err != nil {
                // Handle PTY closure or error
                s.handlePTYError(sessionID, err)
                return
            }

            if n > 0 {
                // Publish output event
                s.eventBus.Publish(events.Event{
                    Type:      "aicoder.pty.output",
                    ProcessID: sessionID,
                    Timestamp: time.Now(),
                    Data: map[string]interface{}{
                        "sessionID": sessionID,
                        "data":      buffer[:n],
                        "sequence":  sequence.Add(1),
                    },
                })
            }
        }
    }
}
```

**Test This Step**:
```bash
go test -v ./internal/tui/services -run TestStartSession
# Should create real PTY and start monitoring
```

#### 1.4 Implement GetTerminalState with Real PTY (1 hour)
```go
func (s *AICoderService) GetTerminalState(sessionID string) (*messages.TerminalState, error) {
    session, err := s.GetSession(sessionID)
    if err != nil {
        return nil, err
    }

    if session.Status != messages.SessionActive && session.Status != messages.SessionIdle {
        return nil, ErrSessionNotStarted
    }

    // Get terminal state from PTY session
    // This depends on PTY infrastructure API
    lines, cursor, width, height := session.PTYSession.GetTerminalSnapshot()

    return &messages.TerminalState{
        Lines:       lines,
        Cursor:      messages.CursorPos{Row: cursor.Row, Col: cursor.Col, Visible: true},
        Width:       width,
        Height:      height,
        TotalLines:  len(lines),
        ViewportTop: 0,
        UpdatedAt:   time.Now(),
    }, nil
}
```

#### 1.5 Implement SendInput and ResizePTY (1 hour)
```go
func (s *AICoderService) SendInput(sessionID string, input []byte) error {
    // ... validation ...

    // Forward to PTY
    n, err := session.PTYSession.Write(input)
    if err != nil {
        return fmt.Errorf("failed to write to PTY: %w", err)
    }

    session.BytesWritten.Add(uint64(n))
    session.LastActivity = time.Now()

    return nil
}

func (s *AICoderService) ResizePTY(sessionID string, width, height int) error {
    // ... validation ...

    // Resize PTY and send SIGWINCH
    if err := session.PTYSession.Resize(width, height); err != nil {
        return fmt.Errorf("failed to resize PTY: %w", err)
    }

    return nil
}
```

#### 1.6 Implement StopSession Cleanup (30 min)
```go
func (s *AICoderService) StopSession(ctx context.Context, sessionID string) error {
    // ... get session ...

    // Close PTY session
    if session.PTYSession != nil {
        if err := session.PTYSession.Close(); err != nil {
            s.eventBus.Publish(events.Event{
                Type:      "aicoder.error",
                ProcessID: sessionID,
                Data:      map[string]interface{}{"error": err.Error()},
            })
        }
        // Wait for monitoring goroutine to exit (already handled via shutdownCh)
    }

    // ... rest of cleanup ...
}
```

**Acceptance Criteria**:
- [ ] Real PTY sessions created on StartSession
- [ ] Monitoring goroutine publishes output events
- [ ] GetTerminalState returns actual terminal content
- [ ] SendInput forwards to PTY
- [ ] ResizePTY sends SIGWINCH
- [ ] StopSession closes PTY and cleans up goroutines

---

### Step 2: Comprehensive Tests (T004-T007) - 2-3 hours

Once PTY integration is complete, add comprehensive tests.

#### 2.1 Session Lifecycle Tests (T004) - 45 min
```go
// In aicoder_service_test.go

func TestCreateSession_Valid(t *testing.T) {
    ctx := context.Background()
    service := NewAICoderService(ctx)

    sessionID, err := service.CreateSession(ctx, "Test Session", "echo 'hello'", "/tmp")
    if err != nil {
        t.Fatalf("CreateSession failed: %v", err)
    }

    session, err := service.GetSession(sessionID)
    if err != nil {
        t.Fatalf("GetSession failed: %v", err)
    }

    if session.Name != "Test Session" {
        t.Errorf("Expected name 'Test Session', got '%s'", session.Name)
    }
    if session.Status != messages.SessionPending {
        t.Errorf("Expected status Pending, got %s", session.Status)
    }
}

func TestCreateSession_InvalidName(t *testing.T) {
    ctx := context.Background()
    service := NewAICoderService(ctx)

    tests := []struct {
        name    string
        wantErr error
    }{
        {"", ErrInvalidName},
        {strings.Repeat("a", 101), ErrInvalidName},
    }

    for _, tt := range tests {
        _, err := service.CreateSession(ctx, tt.name, "echo test", "/tmp")
        if !errors.Is(err, tt.wantErr) {
            t.Errorf("Expected error %v, got %v", tt.wantErr, err)
        }
    }
}

// Add tests for:
// - TestStartSession_Valid
// - TestStartSession_InvalidDimensions
// - TestStopSession_Valid
// - TestStopSession_Idempotent
// - TestSessionStatusTransitions
```

#### 2.2 PTY Integration Tests (T005) - 1 hour
```go
func TestPTYStartup(t *testing.T) {
    ctx := context.Background()
    service := NewAICoderService(ctx)

    sessionID, _ := service.CreateSession(ctx, "PTY Test", "cat", "/tmp")

    err := service.StartSession(ctx, sessionID, 80, 24)
    if err != nil {
        t.Fatalf("StartSession failed: %v", err)
    }

    session, _ := service.GetSession(sessionID)
    if session.Status != messages.SessionActive {
        t.Errorf("Expected Active status, got %s", session.Status)
    }

    // Verify PTY is running
    if session.PTYSession == nil {
        t.Error("PTY session not initialized")
    }
}

func TestGetTerminalState(t *testing.T) {
    // ... create and start session ...

    state, err := service.GetTerminalState(sessionID)
    if err != nil {
        t.Fatalf("GetTerminalState failed: %v", err)
    }

    if state.Width != 80 || state.Height != 24 {
        t.Errorf("Expected 80x24, got %dx%d", state.Width, state.Height)
    }
}

func TestSendInput(t *testing.T) {
    // ... create and start session with 'cat' ...

    input := []byte("hello\n")
    err := service.SendInput(sessionID, input)
    if err != nil {
        t.Fatalf("SendInput failed: %v", err)
    }

    // Wait for echo and verify terminal state contains "hello"
    time.Sleep(100 * time.Millisecond)
    state, _ := service.GetTerminalState(sessionID)

    found := false
    for _, line := range state.Lines {
        if strings.Contains(line, "hello") {
            found = true
            break
        }
    }
    if !found {
        t.Error("Input not echoed to terminal")
    }
}

func TestPTYCleanup(t *testing.T) {
    // ... create and start session ...

    start := time.Now()
    err := service.StopSession(ctx, sessionID)
    elapsed := time.Since(start)

    if err != nil {
        t.Fatalf("StopSession failed: %v", err)
    }

    if elapsed > 100*time.Millisecond {
        t.Errorf("Cleanup took %v, expected <100ms", elapsed)
    }

    // Verify PTY is closed
    _, err = service.GetSession(sessionID)
    if !errors.Is(err, ErrSessionNotFound) {
        t.Error("Session still exists after stop")
    }
}
```

#### 2.3 Concurrent Sessions Tests (T006) - 30 min
```go
func TestConcurrentSessionCreation(t *testing.T) {
    ctx := context.Background()
    service := NewAICoderService(ctx)

    var wg sync.WaitGroup
    sessionIDs := make([]string, 5)

    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            sessionID, err := service.CreateSession(ctx, fmt.Sprintf("Session %d", idx), "cat", "/tmp")
            if err != nil {
                t.Errorf("Failed to create session %d: %v", idx, err)
                return
            }
            sessionIDs[idx] = sessionID
        }(i)
    }

    wg.Wait()

    sessions := service.ListSessions()
    if len(sessions) != 5 {
        t.Errorf("Expected 5 sessions, got %d", len(sessions))
    }
}

func TestNoSessionIDCollisions(t *testing.T) {
    ctx := context.Background()
    service := NewAICoderService(ctx)

    ids := make(map[string]bool)
    for i := 0; i < 100; i++ {
        sessionID, _ := service.CreateSession(ctx, fmt.Sprintf("Session %d", i), "cat", "/tmp")
        if ids[sessionID] {
            t.Fatalf("Duplicate session ID: %s", sessionID)
        }
        ids[sessionID] = true
    }
}
```

#### 2.4 Error Handling Tests (T007) - 30 min
```go
func TestSessionNotFound(t *testing.T) {
    ctx := context.Background()
    service := NewAICoderService(ctx)

    _, err := service.GetSession("nonexistent")
    if !errors.Is(err, ErrSessionNotFound) {
        t.Errorf("Expected ErrSessionNotFound, got %v", err)
    }
}

func TestSessionAlreadyStarted(t *testing.T) {
    // ... create and start session ...

    err := service.StartSession(ctx, sessionID, 80, 24)
    if !errors.Is(err, ErrSessionAlreadyStarted) {
        t.Errorf("Expected ErrSessionAlreadyStarted, got %v", err)
    }
}
```

**Run Tests**:
```bash
go test -v ./internal/tui/services
# Should show: 15+ tests passing
```

---

### Step 3: Resource Management Tests (T013-T016) - 2-3 hours

#### 3.1 Goroutine Leak Detection (T013) - 1 hour
```go
func TestNoGoroutineLeaks_SessionLifecycle(t *testing.T) {
    before := runtime.NumGoroutine()

    ctx := context.Background()
    service := NewAICoderService(ctx)

    // Create and start 10 sessions
    for i := 0; i < 10; i++ {
        sessionID, _ := service.CreateSession(ctx, fmt.Sprintf("Leak Test %d", i), "cat", "/tmp")
        service.StartSession(ctx, sessionID, 80, 24)
    }

    // Stop all sessions
    sessions := service.ListSessions()
    for _, s := range sessions {
        service.StopSession(ctx, s.ID)
    }

    // Shutdown service
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    service.Shutdown(shutdownCtx)

    // Wait for goroutines to exit
    time.Sleep(100 * time.Millisecond)

    after := runtime.NumGoroutine()
    delta := after - before

    // Allow some tolerance for test framework goroutines
    if delta > 2 {
        t.Errorf("Goroutine leak detected: before=%d, after=%d, delta=%d", before, after, delta)
    }
}
```

#### 3.2 Resource Cleanup Tests (T014) - 1 hour
```go
func TestPTYFileDescriptorCleanup(t *testing.T) {
    // This test is platform-specific
    // On Unix, check /proc/self/fd count
}

func TestSyncMapCleanup(t *testing.T) {
    ctx := context.Background()
    service := NewAICoderService(ctx)

    // Create sessions
    for i := 0; i < 10; i++ {
        sessionID, _ := service.CreateSession(ctx, fmt.Sprintf("Session %d", i), "cat", "/tmp")
        service.StartSession(ctx, sessionID, 80, 24)
    }

    // Shutdown
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    service.Shutdown(shutdownCtx)

    // Verify all sessions removed
    sessions := service.ListSessions()
    if len(sessions) != 0 {
        t.Errorf("Expected 0 sessions after shutdown, got %d", len(sessions))
    }
}
```

#### 3.3 Shutdown Tests (T015-T016) - 1 hour
```go
func TestShutdown_AllSessionsStopped(t *testing.T) {
    // ... create multiple sessions ...

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := service.Shutdown(shutdownCtx)
    if err != nil {
        t.Fatalf("Shutdown failed: %v", err)
    }

    // Verify all sessions stopped
    sessions := service.ListSessions()
    if len(sessions) != 0 {
        t.Errorf("Sessions still active after shutdown: %d", len(sessions))
    }
}

func TestShutdown_Idempotent(t *testing.T) {
    // ... shutdown once ...

    // Shutdown again
    err := service.Shutdown(ctx)
    if err != nil {
        t.Error("Second shutdown should not error")
    }
}

func TestShutdown_GracefulTimeout(t *testing.T) {
    // Create service with fast timeout
    // ... add sessions ...

    // Shutdown with very short timeout
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
    defer cancel()

    err := service.Shutdown(ctx)
    if !errors.Is(err, ErrShutdownTimeout) {
        t.Errorf("Expected ErrShutdownTimeout, got %v", err)
    }
}
```

**Run Race Detector**:
```bash
go test -race -v ./internal/tui/services
# Should pass with no race conditions
```

---

### Step 4: Phase 1 Validation - 30 min

```bash
# Run all service tests
go test -v ./internal/tui/services

# Run with race detector
make test-race
# Or: go test -race ./internal/tui/services

# Check coverage
go test -coverprofile=coverage.out ./internal/tui/services
go tool cover -func=coverage.out | grep total
# Target: ≥87% coverage

# Verify build
go build ./internal/tui/services
```

**Mark Phase 1 Complete in tasks.md**:
```bash
# Update acceptance criteria for T004-T016 to [x]
```

---

## Phase 2: Component Migration (12-14 hours)

### Overview
Once Phase 1 is complete and validated, proceed to Phase 2. This involves migrating the component layer and removing the old controller.

### Step 5: Component Tests (T017-T021) - 3-4 hours

Create integration tests in `test/integration/aicoder_component_test.go`. These tests require TTY allocation.

**Key Points**:
- Use `-tags=integration` for these tests
- Tests need real BubbleTea environment
- Reference existing component tests in `test/integration/`

### Step 6: Component Implementation (T022-T027) - 5-6 hours

Enhance `internal/tui/components/aicoders_view.go`:
1. Add service reference
2. Implement message handlers (SessionCreated, Updated, Ended, PTYOutput)
3. Implement keyboard routing (list navigation, PTY input forwarding)
4. Implement terminal rendering
5. Implement view mode switching (list ↔ PTY)
6. Implement command constructors

### Step 7: Model Integration (T028) - 1-2 hours

Wire the component to the service in `internal/tui/model.go`:
1. Create AICoderService during model initialization
2. Pass service to component constructor
3. Subscribe to service events
4. Forward events as BubbleTea messages
5. Add service.Shutdown() to model cleanup

### Step 8: Controller Removal (T029-T032) - 30 min

Delete the old controller files:
```bash
rm internal/tui/ai_coder_controller.go
rm internal/tui/ai_coder_controller_test.go
rm internal/tui/ai_coder_pty_view.go
rm internal/tui/ai_coder_pty_view_test.go
rm internal/tui/ai_coder_debug_forwarder.go
rm internal/tui/ai_coder_message_handler.go

git rm <files>

# Verify no references remain
grep -r "AICoderController" internal/tui/
# Should return nothing
```

### Step 9: Full Validation (T033-T038) - 2-3 hours

1. Run full test suite
2. Execute quickstart Phase 1-3 validations
3. Manual TUI testing
4. Verify controller removal
5. Performance validation
6. Race condition detection

---

## Common Pitfalls & Solutions

### PTY Integration Issues

**Problem**: PTY read/write errors
**Solution**: Check file descriptors are valid, handle EAGAIN errors

**Problem**: Terminal state is stale
**Solution**: Ensure lock-free buffer is being read correctly

**Problem**: Goroutine leaks
**Solution**: Always close shutdown channel, use WaitGroup properly

### Testing Issues

**Problem**: Tests fail without TTY
**Solution**: Use `-tags=integration` and proper test framework

**Problem**: Race detector failures
**Solution**: Review sync.Map usage, ensure no shared mutable state

### Component Issues

**Problem**: Messages not received
**Solution**: Verify EventBus subscription topics match published events

**Problem**: Keyboard input not forwarded
**Solution**: Check IsFocused() state and message routing

---

## Verification Checklist

Before marking work complete:

### Phase 1
- [ ] All T001-T016 acceptance criteria marked [x] in tasks.md
- [ ] All service tests passing (15+ tests)
- [ ] Race detector passes (`make test-race`)
- [ ] Test coverage ≥87%
- [ ] PTY sessions create successfully
- [ ] Terminal state reflects actual PTY output
- [ ] No goroutine leaks detected
- [ ] Shutdown completes in <5s

### Phase 2
- [ ] All T017-T038 acceptance criteria marked [x] in tasks.md
- [ ] Component tests passing (integration tests)
- [ ] Controller files deleted
- [ ] No controller references in codebase
- [ ] Manual TUI testing successful
- [ ] Quickstart validations complete
- [ ] Performance targets met (<50ms typing latency)

---

## Resources

**Documentation**:
- `IMPLEMENTATION_STATUS.md` - Current state
- `tasks.md` - Full task breakdown
- `contracts/aicoder_service.go` - Service API
- `contracts/aicoder_component.go` - Component API
- `quickstart.md` - Validation scenarios
- `data-model.md` - Entity definitions
- `research.md` - Technical decisions

**Existing Code**:
- `internal/tui/lockfree_pty_manager.go` - PTY infrastructure
- `internal/aicoder/` - AI coder package
- `internal/tui/components/` - Component patterns
- `test/integration/` - Integration test patterns

**Test Commands**:
```bash
# Service tests
go test -v ./internal/tui/services

# With race detector
go test -race -v ./internal/tui/services

# Integration tests (Phase 2)
go test -tags=integration -v ./test/integration -run TestAICoder

# Coverage
go test -coverprofile=coverage.out ./internal/tui/services
go tool cover -html=coverage.out

# Full build
make build

# Full test suite
make test-race
```

---

## Getting Help

If stuck:
1. Review the contract files - they define the expected API
2. Check existing component implementations for patterns
3. Look at how the current controller handles PTY (for migration reference)
4. Review the constitution for architectural requirements

**Key Architecture Principles**:
- Lock-free: Use sync.Map and atomic operations
- Event-driven: All communication via EventBus or BubbleTea messages
- Test-first: Write tests before implementation
- No blocking operations: Especially in PTY path

---

## Success Criteria

**Phase 1 Complete When**:
- Service layer fully functional with real PTY integration
- All tests passing including PTY, concurrent, cleanup
- Race detector passes
- Coverage ≥87%

**Phase 2 Complete When**:
- Component functional in TUI
- Controller completely removed
- All quickstart validations passing
- Manual testing successful
- Performance targets met

**Migration Complete When**:
- AI coder works identically to before migration
- Architecture follows service/component pattern
- Test coverage maintained
- No performance regression
- All 38 tasks marked complete in tasks.md

---

## Estimated Timeline

**Experienced Developer**:
- Phase 1 completion: 8-10 hours
- Phase 2 completion: 12-14 hours
- **Total**: 20-24 hours

**Learning the Codebase**:
- Add 30-50% for initial orientation
- **Total**: 26-36 hours

Good luck! The foundation is solid - you're building on well-architected base.
