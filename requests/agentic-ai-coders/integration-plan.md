# AI Coder PTY Integration Plan

## Overview
This plan outlines the integration of the PTY (pseudo-terminal) system into Brummer's main TUI, transforming the AI Coder tab from a simple process view to a full-featured tmux-style terminal interface.

## Current State Analysis

### What's Already Built
1. **PTY Foundation** (`internal/aicoder/`)
   - `pty_session.go` - Core PTY session with terminal emulation
   - `pty_manager.go` - Multi-session management
   - `data_injector.go` - Contextual data injection system
   - `brummer_data_provider.go` - Integration with Brummer's data

2. **TUI Components** (`internal/tui/`)
   - `ai_coder_pty_view.go` - Standalone PTY view component
   - `ai_coder_view.go` - Current AI coder tab (needs integration)
   - `model.go` - Main TUI model with tab switching

3. **Manager Integration**
   - `AICoderManager` has PTY support methods
   - CLI command configuration for various tools
   - Interactive vs non-interactive mode handling

### Integration Gaps
1. AI Coder tab (View 8) still shows simple process list
2. PTY view not connected to main TUI navigation
3. `/ai` command doesn't create PTY sessions
4. No automatic PTY output streaming to TUI
5. Missing session persistence across tab switches

## Integration Architecture

### Phase 1: Connect PTY View to AI Coder Tab
**Goal**: Replace the current AI coder view with the PTY-enabled view

1. **Update `model.go`**
   - Add `aiCoderPTYView *AICoderPTYView` field
   - Initialize PTY view with BrummerDataProvider
   - Route View 8 rendering to PTY view

2. **Create BrummerDataProvider Implementation**
   - Bridge between TUI Model and PTY data requirements
   - Access to logStore, errorDetector, proxy data
   - Real-time data injection capabilities

3. **Update Tab Switching Logic**
   - Preserve PTY session state when switching tabs
   - Handle terminal focus state transitions
   - Maintain full-screen mode across tab switches

### Phase 2: Integrate /ai Command with PTY
**Goal**: Make `/ai` command create interactive PTY sessions

1. **Update Command Handler**
   - Modify `handleAICommand` in `model.go`
   - Create PTY session instead of regular process
   - Auto-attach to new session in AI Coder tab

2. **Session Creation Flow**
   ```
   /ai claude → CreateInteractiveCLISession("claude")
   /ai claude "fix tests" → CreateTaskCLISession("claude", "fix tests")
   ```

3. **Automatic Tab Switching**
   - Switch to AI Coder tab when creating session
   - Focus terminal for immediate interaction
   - Show session info in status bar

### Phase 3: Event Integration
**Goal**: Connect PTY events to TUI update cycle

1. **PTY Event Routing**
   - Create `PTYEventListener` in TUI model
   - Convert PTY events to tea.Msg types
   - Handle output, close, resize events

2. **Automatic Event Forwarding**
   - Monitor Brummer events (errors, test failures)
   - Forward to active PTY session in debug mode
   - Visual indicators for auto-injected data

3. **Session Lifecycle Management**
   - Handle session creation/deletion
   - Update session list in real-time
   - Clean up resources on session close

### Phase 4: Enhanced Features
**Goal**: Polish the integration for natural workflow

1. **Session Persistence**
   - Maintain sessions across Brummer restarts
   - Save/restore terminal buffer state
   - Reconnect to running CLI processes

2. **Multi-Session Workflow**
   - Quick session switching (Ctrl+1-9)
   - Session overview panel
   - Visual session activity indicators

3. **Smart Defaults**
   - Auto-create session on first `/ai` use
   - Remember last used AI provider
   - Context-aware initial prompts

## Implementation Tasks

### Task 1: Create TUI Data Provider
**File**: `internal/tui/brummer_data_provider_impl.go`
- Implement `BrummerDataProvider` interface
- Access TUI model's data stores
- Thread-safe data access methods

### Task 2: Update Model Structure
**File**: `internal/tui/model.go`
- Add PTY-related fields
- Initialize PTY manager with data provider
- Update view initialization

### Task 3: Route AI Coder View
**File**: `internal/tui/model.go` (View() method)
- Replace case 8 rendering
- Pass through window size messages
- Handle PTY-specific commands

### Task 4: Create PTY Event Bridge
**File**: `internal/tui/pty_events.go`
- PTY event to tea.Msg conversion
- Event subscription management
- Async event handling

### Task 5: Update AI Command Handler
**File**: `internal/tui/model.go` (handleAICommand)
- Parse provider and task from command
- Create appropriate PTY session
- Auto-switch to AI Coder tab

### Task 6: Add Output Streaming
**File**: `internal/tui/model.go` (Init/Update)
- Subscribe to PTY output events
- Convert to tea.Cmd for updates
- Handle backpressure

### Task 7: Implement Session Management
**File**: `internal/tui/ai_session_manager.go`
- Session state persistence
- Graceful reconnection logic
- Resource cleanup

### Task 8: Add Visual Polish
**Files**: Various TUI components
- Session activity indicators
- Status bar integration
- Help text updates

## Testing Strategy

### Unit Tests
1. Data provider implementation
2. Event conversion logic
3. Session management operations

### Integration Tests
1. `/ai` command → PTY session creation
2. Tab switching with active sessions
3. Data injection key bindings
4. Full-screen mode transitions

### Manual Testing Scenarios
1. Create multiple AI sessions
2. Switch between sessions rapidly
3. Inject data during active conversation
4. Test terminal resize handling
5. Verify cleanup on exit

## Migration Path

### Step 1: Parallel Implementation
- Keep existing AI coder view functional
- Build PTY integration alongside
- Feature flag for PTY mode

### Step 2: Gradual Rollout
- Enable PTY for new sessions only
- Migrate existing sessions on demand
- Collect feedback and iterate

### Step 3: Full Cutover
- Remove old AI coder view code
- Clean up legacy process handling
- Update documentation

## Success Criteria

1. **Seamless Integration**
   - `/ai` command "just works"
   - Natural tab navigation
   - Intuitive key bindings

2. **Performance**
   - Smooth terminal rendering
   - No lag in typing
   - Efficient event handling

3. **Reliability**
   - Sessions persist correctly
   - Clean shutdown/cleanup
   - No resource leaks

4. **User Experience**
   - Clear visual feedback
   - Helpful error messages
   - Discoverable features

## Next Steps

1. Review and approve this plan
2. Create detailed task breakdown
3. Begin implementation with Task 1
4. Iterate based on testing feedback

This integration will transform Brummer's AI Coder feature from a simple process viewer to a powerful, tmux-style development environment where AI assistants feel like true pair programming partners.