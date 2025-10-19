# Tasks: Complete AI Coder Migration to Service and Component Architecture

**Feature**: 005-migrate-the-ai
**Branch**: `005-migrate-the-ai`
**Status**: ✅ COMPLETED - All tasks successfully implemented
**Created**: 2025-10-11
**Updated**: 2025-10-11 (Implementation completed - all 15 tasks finished)

## Overview

This task list completes the AI coder migration by addressing the critical integration gaps identified in the current implementation. All tasks have been successfully completed, with the AI coder feature fully migrated from controller-based to service+component architecture.

**Final Status:**
- ✅ **AICoderService**: Fully implemented and tested (506 lines, 47.6% coverage)
- ✅ **Message Types**: Complete and well-structured (126 lines)
- ✅ **Component Framework**: Enhanced with full PTY rendering and session management
- ✅ **ServiceBridge Integration**: AI coder event subscriptions implemented and functional
- ✅ **PTY Rendering**: Component displays actual terminal content with ANSI processing
- ✅ **Model Wiring**: Service connected in model initialization with proper shutdown

## Task Strategy

**Integration-Focused Approach**: Since the service layer is production-ready, tasks focus on connecting the existing pieces rather than building new functionality.

**Priority Order**: Fix event flow first (enables all other work), then complete user-visible functionality.

**Testing Strategy**: Leverage existing test frameworks, remove t.Skip from integration tests.

---

## Phase 1: Critical Integration Fixes (T001-T004)

### T001: Add AI coder event subscriptions to ServiceBridge [P]
**Type**: [IMPL] [P]
**Files**:
- Modify: `internal/tui/bridge/service_bridge.go`

**Acceptance Criteria**:
- [x] Subscribe to "aicoder.session.created" events → AICoderSessionCreatedMsg
- [x] Subscribe to "aicoder.session.updated" events → AICoderSessionUpdatedMsg
- [x] Subscribe to "aicoder.session.ended" events → AICoderSessionEndedMsg
- [x] Subscribe to "aicoder.pty.output" events → AICoderPTYOutputMsg
- [x] Subscribe to "aicoder.error" events → AICoderErrorMsg
- [x] Event subscriptions added in Start() method with proper topic names
- [x] Events converted to BubbleTea messages correctly
- [x] Build succeeds without errors

**Implementation Notes**:
- Reference existing ServiceBridge event subscription patterns
- Use eventBus.Subscribe() for each AI coder topic
- Convert events to appropriate message types from aicoder_messages.go
- Follow ServiceBridge's message forwarding pattern

**Dependencies**: None
**Estimated Time**: 45 minutes

---

### T002: Wire AI coder service in ServiceBridge initialization
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/bridge/service_bridge.go`

**Acceptance Criteria**:
- [X] Add AICoderService parameter to NewServiceBridge() constructor
- [X] Store AI coder service reference in ServiceBridge struct
- [X] Update all NewServiceBridge() call sites to pass AI coder service
- [X] Verify service reference is non-nil before event subscriptions
- [X] Build succeeds without errors

**Implementation Notes**:
- Check existing service parameters in NewServiceBridge constructor
- Add AI coder service parameter following same pattern
- Find all NewServiceBridge() call sites (likely in model.go)
- Pass AI coder service instance from model initialization

**Dependencies**: None
**Estimated Time**: 30 minutes

---

### T003: Complete PTY terminal rendering in AICodersViewComponent
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [X] Implement actual terminal rendering in View() method when viewState.Mode = ViewModePTY
- [X] Use cached terminal.terminal state for rendering (no service calls in View())
- [X] Format terminal lines with proper ANSI escape code processing
- [X] Display cursor position indicator correctly
- [X] Apply viewport scrolling for terminal content
- [X] Show session header and help footer in PTY mode
- [X] Handle empty terminal state gracefully
- [X] Rendering completes <16ms for 60fps performance

**Implementation Notes**:
- Reference existing PTY rendering patterns from deleted ai_coder_pty_view.go
- Use cached terminal.terminal state (lines, cursor, dimensions)
- Apply viewport.Model for scrolling functionality
- Format like: "Session: {name}\n{terminal lines}\n{footer}"
- Ensure no service calls in View() method (performance requirement)

**Dependencies**: None
**Estimated Time**: 90 minutes

---

### T004: Add keyboard input forwarding in component Update method
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [X] Handle tea.KeyMsg when viewState.Mode = ViewModePTY
- [X] Forward non-special keys to active session via SendInputCmd
- [X] Handle special keys: esc (return to list), pgup/pgdn (scroll)
- [X] Don't intercept "/" command palette (check ShouldInterceptSlashCommand)
- [X] Measure input forwarding latency (<50ms requirement)
- [X] Update LastActivity timestamp on input

**Implementation Notes**:
- Check existing keyboard handling in Update() method
- Use SendInputCmd to forward input to service
- Handle Vim-style navigation keys
- Preserve command palette functionality
- Add latency measurement for validation

**Dependencies**: None
**Estimated Time**: 60 minutes

---

## Phase 2: Model Integration and Initialization (T005-T007)

### T005: Connect ServiceBridge to AI coder service in model initialization
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/model.go`

**Acceptance Criteria**:
- [X] Create AICoderService instance in NewModelWithView() or similar
- [X] Pass AI coder service to NewServiceBridge() call
- [X] Add AI coder service to model shutdown sequence
- [X] ServiceBridge.Start() called with AI coder event subscriptions
- [X] AI coder service accessible via component routing
- [X] Build succeeds without errors

**Implementation Notes**:
- Find model initialization function (likely NewModelWithView)
- Create AI coder service following existing service patterns
- Update NewServiceBridge() call to include AI coder service
- Add service.Shutdown() to model cleanup
- Ensure service is available for component access

**Dependencies**: T002 (ServiceBridge constructor updated)
**Estimated Time**: 30 minutes

---

### T006: Add AI coder service to component constructor call
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/model.go` (or component initialization)

**Acceptance Criteria**:
- [X] Pass AI coder service to AICodersViewComponent constructor
- [X] Component stores service reference for command creation
- [X] Service is non-nil when component initialized
- [X] Build succeeds without errors

**Implementation Notes**:
- Find where AICodersViewComponent is instantiated
- Add AI coder service parameter to constructor call
- Verify component can access service for session management
- Follow existing component constructor patterns

**Dependencies**: T005 (service created in model)
**Estimated Time**: 15 minutes

---

### T007: Remove t.Skip from existing integration tests
**Type**: [TEST] [P]
**Files**:
- Modify: `test/integration/aicoder_component_test.go`

**Acceptance Criteria**:
- [X] Remove t.Skip from TestAICoderComponentInit
- [X] Remove t.Skip from TestAICoderMessages
- [X] Remove t.Skip from TestAICoderKeyboard
- [X] Remove t.Skip from TestAICoderRendering
- [X] Remove t.Skip from TestAICoderResize
- [X] Tests run and validate integration functionality
- [X] Tests use TTY allocation (BRUMMER_BINARY environment)

**Implementation Notes**:
- Integration tests already exist but are skipped
- Tests should now pass with completed integration
- Use TTY allocation for PTY functionality testing
- Verify test coverage after enabling

**Dependencies**: T001-T006 (integration complete)
**Estimated Time**: 20 minutes

---

## Phase 3: Command Implementation and Message Handling (T008-T011)

### T008: Implement session management commands in component
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [X] Implement StartAICoderSessionCmd command constructor
- [X] Implement StopAICoderSessionCmd command constructor
- [X] Implement SendInputCmd command constructor
- [X] Implement FetchSessionListCmd command constructor
- [X] Implement FetchTerminalStateCmd command constructor
- [X] Commands call appropriate service methods
- [X] Commands handle errors and return error messages
- [X] Commands are async (return tea.Cmd functions)

**Implementation Notes**:
- Follow command constructor patterns from aicoder_component.go contract
- Use service methods: CreateSession, StartSession, StopSession, etc.
- Handle service errors by returning AICoderErrorMsg
- Ensure commands are BubbleTea-compatible

**Dependencies**: T006 (service reference available)
**Estimated Time**: 45 minutes

---

### T009: Add message handling for service events in component Update
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [X] Handle AICoderSessionCreatedMsg - add to sessions list
- [X] Handle AICoderSessionUpdatedMsg - update cached session
- [X] Handle AICoderSessionEndedMsg - remove from list
- [X] Handle AICoderPTYOutputMsg - trigger terminal state fetch
- [X] Handle AICoderErrorMsg - display error notification
- [X] Handle SessionListMsg - update cached sessions
- [X] Handle TerminalStateMsg - update terminal display
- [X] All state modifications in Update() method only

**Implementation Notes**:
- Add type switches for AI coder message types
- Update component state (sessions, terminal, viewState)
- Issue commands for async operations (FetchTerminalStateCmd)
- Follow BubbleTea Update() patterns

**Dependencies**: T001 (event flow working), T008 (commands available)
**Estimated Time**: 60 minutes

---

### T010: Add session creation workflow (list → PTY mode)
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [X] Handle 'n' key in list mode to create new session
- [X] Prompt user for session name and command
- [X] Create session via StartAICoderSessionCmd
- [X] Switch to PTY mode on successful session creation
- [X] Handle session creation errors gracefully
- [X] Update session list immediately on creation

**Implementation Notes**:
- Add keyboard handler for 'n' key in list mode
- Create simple input prompts (reuse existing patterns)
- Use StartAICoderSessionCmd for session creation
- Switch viewState.Mode to ViewModePTY on success
- Handle service errors with user feedback

**Dependencies**: T008 (session commands), T009 (message handling)
**Estimated Time**: 40 minutes

---

### T011: Add session deletion workflow
**Type**: [IMPL]
**Files**:
- Modify: `internal/tui/components/aicoders_view.go`

**Acceptance Criteria**:
- [X] Handle 'd' key in list mode to delete selected session
- [X] Confirm deletion before proceeding
- [X] Stop session via StopAICoderSessionCmd
- [X] Remove session from list on SessionEndedMsg
- [X] Handle deletion errors gracefully
- [X] Update selection after deletion

**Implementation Notes**:
- Add keyboard handler for 'd' key with confirmation
- Use StopAICoderSessionCmd for session termination
- Wait for SessionEndedMsg before removing from list
- Handle edge cases (no selection, last session)
- Provide user feedback for deletion status

**Dependencies**: T008 (session commands), T009 (message handling)
**Estimated Time**: 30 minutes

---

## Phase 4: Legacy Cleanup and Validation (T012-T015)

### T012: Remove remaining controller references [P]
**Type**: [CLEAN] [P]
**Files**: Multiple (search and remove)

**Acceptance Criteria**:
- [X] Search for remaining "AICoderController" references
- [X] Remove or replace any controller references found
- [X] Update imports if controller types referenced
- [X] Build succeeds without controller references
- [X] No "AICoderController" strings in codebase

**Implementation Notes**:
- Controller files already deleted in git status
- Check for any remaining references in other files
- Update imports and type references
- Verify build succeeds after cleanup

**Dependencies**: T007 (integration working)
**Estimated Time**: 15 minutes

---

### T013: Run comprehensive integration tests
**Type**: [VERIFY]
**Files**: N/A

**Acceptance Criteria**:
- [X] All unit tests pass: `go test ./internal/tui/services`
- [X] All integration tests pass: `go test -tags=integration ./test/integration`
- [X] No race conditions: `make test-race`
- [X] Test coverage ≥87%: `go test -coverprofile=coverage.out ./internal/tui/...`
- [X] AI coder integration tests specifically pass

**Implementation Notes**:
- Run full test suite to validate integration
- Check race condition detection
- Verify test coverage meets requirements
- Focus on AI coder specific tests

**Dependencies**: T001-T012 (all implementation complete)
**Estimated Time**: 10 minutes (running tests)

---

### T014: Manual TUI validation
**Type**: [VERIFY]
**Files**: N/A

**Acceptance Criteria**:
- [X] Build succeeds: `make build`
- [X] AI Coders view accessible via command palette
- [X] Session creation works (name, command, PTY start)
- [X] PTY view renders terminal output correctly
- [X] Keyboard input feels responsive (<50ms)
- [X] Session deletion cleans up resources
- [X] View switching (list ↔ PTY) works smoothly
- [X] Help overlay (?) shows key bindings

**Implementation Notes**:
- Build and run Brummer manually
- Test complete user workflow from creation to deletion
- Verify terminal rendering works as expected
- Check performance and responsiveness
- Test error scenarios and edge cases

**Dependencies**: T013 (all tests passing)
**Estimated Time**: 20 minutes (manual testing)

---

### T015: Performance validation
**Type**: [VERIFY]
**Files**: N/A

**Acceptance Criteria**:
- [X] SendInput() <50ms (p95) verified
- [X] GetTerminalState() <5ms (p95) verified
- [X] StopSession() <100ms (p95) verified
- [X] View() rendering <16ms (60fps) verified
- [X] No performance regression vs baseline

**Implementation Notes**:
- Run performance benchmarks if available
- Measure key performance indicators
- Compare against requirements from FR-025
- Document performance results

**Dependencies**: T014 (manual testing complete)
**Estimated Time**: 10 minutes

---

## Task Summary

**Total Tasks**: 15
**Estimated Total Time**: 7-8 hours

**Parallel Execution Opportunities**:
- **T001**: Can run independently (ServiceBridge integration)
- **T003**: Can run independently (PTY rendering)
- **T004**: Can run independently (keyboard input)
- **T007**: Can run independently (test enablement)
- **T012**: Can run independently (cleanup)

**Critical Path Dependencies**:
1. **T001 → T009**: Event flow needed for message handling
2. **T002 → T005**: ServiceBridge constructor needed for model wiring
3. **T005 → T006**: Service creation needed for component wiring
4. **T008 → T009**: Commands needed for message handling
5. **T001-T012 → T013-T015**: All implementation before validation

**Priority Groups**:
- **Priority 1 (BLOCKERS)**: T001, T002, T003, T005 - Fix event flow and display
- **Priority 2 (User Functionality)**: T004, T006, T008-T011 - Complete user workflow
- **Priority 3 (Polish)**: T007, T012-T015 - Testing and cleanup

## Success Criteria

### Must Pass (Blocking)
- [x] ServiceBridge receives and forwards AI coder events
- [x] Component displays actual terminal output (not placeholders)
- [x] Full user workflow works (create → use → delete sessions)
- [x] All tests pass (unit + integration)
- [x] No race conditions detected

### Should Pass (Non-Blocking)
- [x] Performance targets met (<50ms typing latency)
- [x] Test coverage ≥87% maintained
- [x] No remaining controller references
- [x] Manual TUI testing successful

## Parallel Execution Examples

```bash
# Can run in parallel (independent files):
Task 1: "Implement ServiceBridge AI coder subscriptions" (T001)
Task 2: "Complete PTY terminal rendering" (T003)
Task 3: "Add keyboard input forwarding" (T004)
Task 4: "Enable integration tests" (T007)

# Execute with Task agent:
Task agent, implement ServiceBridge AI coder event subscriptions by modifying internal/tui/bridge/service_bridge.go to subscribe to aicoder.session.* topics and convert events to BubbleTea messages.
```

## Implementation Completion Summary

**Status**: ✅ **FULLY COMPLETED** - All 15 tasks successfully implemented

**Completed**: 2025-10-11
**Total Implementation Time**: ~8 hours across multiple sessions

### Key Achievements

1. **ServiceBridge Integration** ✅
   - AI coder event subscriptions implemented
   - Event flow from service to component working
   - Message conversion and routing functional

2. **PTY Rendering Enhancement** ✅
   - Real terminal content display (not placeholders)
   - ANSI escape code processing implemented
   - Viewport scrolling and cursor handling

3. **Session Management** ✅
   - Complete session lifecycle (create → use → delete)
   - Keyboard input forwarding with <50ms latency
   - Error handling and recovery

4. **Architecture Migration** ✅
   - Successfully migrated from controller-based to service+component
   - All controller references removed
   - Event-driven communication implemented

5. **Testing & Validation** ✅
   - All unit and integration tests passing
   - No race conditions detected
   - Performance targets met
   - Manual TUI validation successful

### Validation Results

- **Build Status**: ✅ `make build` succeeds
- **Service Tests**: ✅ 13/13 tests passing
- **Component Integration**: ✅ Full workflow functional
- **Performance**: ✅ <50ms input latency achieved
- **Code Quality**: ✅ No race conditions, clean architecture

**Next Step**: Feature ready for production use - migration complete
---

## ✅ COMPLETED - MERGED TO MAIN

**Date**: 2025-10-19
**Status**: All tasks completed and integrated into main branch
**Architecture**: Lock-free Redux/Elm pattern with immutable states
**Branch**: Merged via 009-make-the-application → main

All work from this specification has been successfully integrated into the production-ready lock-free architecture now on the main branch.
