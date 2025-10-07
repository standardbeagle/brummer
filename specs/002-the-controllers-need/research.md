# Technical Research: Controller Migration Strategy

**Feature**: Controller to Service/Component Migration
**Date**: 2025-10-06
**Status**: Complete

## Executive Summary

This document analyzes the 17 controller files to be removed and defines the migration strategy for each. The goal is to move controller logic into BubbleTea components that call services directly, eliminating the 4-layer architecture (Model → Controllers → Components → Services) in favor of a clean 2-layer design (Components → Services).

## Controller Inventory and Analysis

### 1. input_controller.go (473 lines)
**Responsibilities**:
- Routes all keyboard input to appropriate handlers
- Manages global key bindings (quit, navigation, help)
- Delegates to view-specific controllers
- Handles command window and overlay input
- Implements hierarchical key handling system

**Dependencies**:
- All other controllers (circular dependencies)
- BubbleTea key bindings
- Model reference for state access

**Migration Target**: `components/hierarchical_key_handler.go`
**Strategy**:
- Keep hierarchical key handler component (already exists)
- Move global key handling to component methods
- Remove controller delegation (components handle own keys)
- Use BubbleTea Update() for routing instead of controller methods

**State to Migrate**:
- lastQuitPress (double-quit detection)
- quitTimeout (quit confirmation window)
- Key handler context (already in hierarchical handler)

### 2. web_view_controller.go (969 lines - LARGEST)
**Responsibilities**:
- Manages proxy request display and filtering
- Handles request selection and detail view
- Coordinates with proxy service for live updates
- Manages viewport scrolling and pagination

**Dependencies**:
- proxy.Server (service - direct calls OK)
- Layout controller (for sizing)
- Model for view state

**Migration Target**: `components/web_view.go` (already exists)
**Strategy**:
- Component already calls proxy service directly
- Remove controller wrapper methods
- Move viewport management into component state
- Use BubbleTea messages for proxy updates

**State to Migrate**:
- Selected request index
- Filter state (by status, method, etc.)
- Viewport scroll position
- Detail view toggle state

### 3. ai_coder_controller.go (655 lines)
**Responsibilities**:
- Manages AI coder session lifecycle
- Handles PTY terminal emulator integration
- Routes terminal input when focused
- Manages slash command interception
- Coordinates browser debug proxy

**Dependencies**:
- PTY terminal emulator
- AI coder service/session manager
- Browser proxy service
- Model for focus state

**Migration Target**: `components/integrated_shell.go` (already exists)
**Strategy**:
- Component already has PTY integration
- Remove controller state machine
- Move session management to component
- Use BubbleTea messages for PTY events

**State to Migrate**:
- Active session reference
- Terminal focus state
- PTY size/dimensions
- Slash command interception flag

### 4. errors_view_controller.go (441 lines)
**Responsibilities**:
- Displays error list with filtering
- Manages error selection and detail view
- Handles error dismissal and clearing
- Coordinates with error store service

**Dependencies**:
- Error store service
- Model for view state
- Layout controller for sizing

**Migration Target**: `components/errors_view.go` (already exists)
**Strategy**:
- Component already exists with basic rendering
- Add error store service calls
- Implement filtering and selection logic
- Use BubbleTea Update() for interactions

**State to Migrate**:
- Selected error index
- Filter state (by severity, source)
- Show dismissed errors toggle
- Detail view state

### 5. logs_view_controller.go (estimated 400+ lines)
**Responsibilities**:
- Displays process logs with filtering
- Manages log scrolling and follow mode
- Handles priority filtering
- Coordinates with log store service

**Dependencies**:
- Log store service
- Model for selected process
- Layout controller for sizing

**Migration Target**: `components/logs_view.go` (needs creation or enhancement)
**Strategy**:
- Create or enhance logs component
- Add log store service calls
- Implement auto-scroll and follow mode
- Use BubbleTea viewport for scrolling

**State to Migrate**:
- Current process filter
- Priority filter state
- Auto-follow mode toggle
- Viewport scroll position

### 6. process_view_controller.go (estimated 300+ lines)
**Responsibilities**:
- Displays process list
- Handles process selection
- Shows process status and metadata
- Coordinates with process manager service

**Dependencies**:
- process.Manager service
- Model for process state
- Layout controller for sizing

**Migration Target**: `components/processes_view.go` (needs creation)
**Strategy**:
- Create processes view component
- Call process.Manager directly for process list
- Use BubbleTea list component for display
- Handle Enter for navigation to logs

**State to Migrate**:
- Selected process ID
- Sort order (by name, status, etc.)
- Filter state (running/stopped)

### 7. urls_view_controller.go (estimated 250+ lines)
**Responsibilities**:
- Displays detected URLs
- Handles URL selection and copying
- Shows URL metadata (process, status)
- Coordinates with URL discovery service

**Dependencies**:
- process.Manager (for URL data)
- Model for view state

**Migration Target**: `components/urls_view.go` (needs creation)
**Strategy**:
- Create URLs view component
- Get URLs from process manager
- Use BubbleTea list for display
- Implement copy-to-clipboard

**State to Migrate**:
- Selected URL index
- Group by process toggle
- Show inactive URLs toggle

### 8. settings_controller.go (estimated 300+ lines)
**Responsibilities**:
- Displays settings UI
- Handles setting modifications
- Shows file browser for path selection
- Saves configuration changes

**Dependencies**:
- config.Manager service
- filebrowser.Controller (for path selection)
- Model for settings state

**Migration Target**: `components/settings_view.go` (needs creation)
**Strategy**:
- Create settings view component
- Call config service directly
- Keep filebrowser controller (out of scope)
- Use BubbleTea forms for editing

**State to Migrate**:
- Current setting being edited
- Pending changes (before save)
- Show file browser state

### 9. layout_controller.go (estimated 350+ lines)
**Responsibilities**:
- Calculates view dimensions
- Manages split layouts (main + system panel)
- Handles viewport sizing
- Coordinates header/footer rendering

**Dependencies**:
- Model for all view data
- Services for data to display

**Migration Target**: `components/layout_manager.go` (create as component utility)
**Strategy**:
- Create layout manager as pure function utility
- Remove stateful controller
- Components request layout calculations
- Use in model.View() for rendering

**State to Migrate**:
- System panel height/open state
- Last window size (for resize detection)

### 10. event_controller.go (estimated 200+ lines)
**Responsibilities**:
- Subscribes to EventBus events
- Converts events to BubbleTea messages
- Routes events to appropriate handlers
- Manages event listener lifecycle

**Dependencies**:
- EventBus
- Model update channel
- All services that emit events

**Migration Target**: `model.go` (direct EventBus subscription)
**Strategy**:
- Move EventBus subscription to model Init()
- Convert events to tea.Msg in subscriber
- Remove controller indirection
- Keep EventBus pattern (constitutional requirement)

**State to Migrate**:
- Event subscription handles (for cleanup)

### 11. command_window_controller.go (estimated 300+ lines)
**Responsibilities**:
- Manages command palette display
- Handles command input and autocomplete
- Validates commands before execution
- Routes commands to handlers

**Dependencies**:
- Command autocomplete component
- Command registry
- Model for command execution

**Migration Target**: `components/command_palette.go` (needs creation)
**Strategy**:
- Create command palette component
- Keep autocomplete as sub-component
- Call command handlers directly
- Use BubbleTea messages for command results

**State to Migrate**:
- Show/hide state
- Current input value
- Selected suggestion index
- Validation error state

### 12. script_selector_controller.go (458 lines)
**Responsibilities**:
- Displays available scripts from package.json
- Handles script selection
- Launches selected scripts via process manager
- Manages script metadata display

**Dependencies**:
- Script parser/discovery
- process.Manager service
- Model for selection state

**Migration Target**: `components/script_selector.go` (needs creation)
**Strategy**:
- Create script selector component
- Parse package.json directly
- Call process manager to start scripts
- Use BubbleTea list for display

**State to Migrate**:
- Available scripts list
- Selected script index
- Show script details toggle

### 13. mcp_debug_controller.go (496 lines)
**Responsibilities**:
- Displays MCP connection status
- Shows MCP tool list and usage
- Handles MCP tool invocation
- Manages MCP debug console

**Dependencies**:
- MCP server service
- Model for MCP state

**Migration Target**: `components/mcp_debug_view.go` (needs creation)
**Strategy**:
- Create MCP debug view component
- Call MCP server directly for status
- Display tools and connections
- Use BubbleTea viewport for console

**State to Migrate**:
- Selected tool index
- Show connection details toggle
- Console output buffer

## Controllers to Keep (Out of Scope)

### navigation/controller.go
**Rationale**: Core navigation state (current view, view history) is separate from view rendering. This is view-agnostic state management.
**Future**: Rename to `navigation/manager.go` in a future refactoring.

### system/controller.go
**Rationale**: System message queue management (notifications, errors) is a cross-cutting concern, not tied to a single view.
**Future**: Rename to `system/message_manager.go` in a future refactoring.

### filebrowser/controller.go
**Rationale**: File browser dialog is a complex reusable component with its own state machine (directory navigation, selection).
**Future**: Refactor to pure BubbleTea component in a future iteration.

### notifications/controller.go
**Rationale**: Notification queue management is a cross-cutting concern with timing and priority logic.
**Future**: Rename to `notifications/queue_manager.go` in a future refactoring.

## Migration Patterns

### Pattern 1: View State Migration
```go
// Before (controller)
type ProcessViewController struct {
    selectedIndex int
    processes []process.Process
}

// After (component)
type ProcessesViewComponent struct {
    processMgr *process.Manager  // Service reference
    selectedIndex int             // Component state
    list bubbles.List             // BubbleTea sub-component
}
```

### Pattern 2: Service Interaction
```go
// Before (controller method)
func (pvc *ProcessViewController) GetProcesses() []process.Process {
    return pvc.processMgr.GetAllProcesses()
}
// Then model calls: m.processViewController.GetProcesses()

// After (component direct call)
func (pv *ProcessesViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    processes := pv.processMgr.GetAllProcesses()  // Direct call
    // ... update component state
}
```

### Pattern 3: Keyboard Handling
```go
// Before (controller routing)
func (ic *InputController) HandleKeyMsg(msg tea.KeyMsg) {
    if ic.model.currentView() == ViewProcesses {
        ic.model.processViewController.HandleKey(msg)
    }
}

// After (component handling)
func (pv *ProcessesViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            // Handle enter key directly
        }
    }
}
```

### Pattern 4: Event Handling
```go
// Before (event controller)
func (ec *EventController) HandleEvent(event Event) {
    ec.model.processViewController.OnProcessStarted(event)
}

// After (BubbleTea message)
type ProcessStartedMsg struct {
    ProcessID string
}

func (pv *ProcessesViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ProcessStartedMsg:
        // Handle process started event
    }
}
```

## Dependency Order for Migration

**Phase 1: Foundation** (no dependencies)
1. event_controller.go → model.go EventBus integration
2. layout_controller.go → layout_manager.go utility

**Phase 2: Simple Views** (depend on Phase 1)
3. urls_view_controller.go → components/urls_view.go
4. mcp_debug_controller.go → components/mcp_debug_view.go
5. command_window_controller.go → components/command_palette.go

**Phase 3: Core Views** (depend on Phase 2)
6. process_view_controller.go → components/processes_view.go
7. logs_view_controller.go → components/logs_view.go
8. errors_view_controller.go → components/errors_view.go

**Phase 4: Complex Views** (depend on Phase 3)
9. web_view_controller.go → components/web_view.go
10. settings_controller.go → components/settings_view.go
11. script_selector_controller.go → components/script_selector.go

**Phase 5: Advanced Features** (depend on Phase 4)
12. ai_coder_controller.go → components/integrated_shell.go

**Phase 6: Input System** (must be last - depends on all views)
13. input_controller.go → components/hierarchical_key_handler.go

## Testing Strategy

### Pre-Migration Tests (Write Before Refactoring)
1. **Functional Tests**: Verify each view's current behavior
   - Process view: list, select, navigate to logs
   - Logs view: filter, scroll, follow mode
   - Errors view: filter, dismiss, clear
   - etc.

2. **Keyboard Shortcut Tests**: Verify all keys work
   - Global keys (quit, navigation, help)
   - View-specific keys (filter, clear, etc.)
   - Number keys for view switching

3. **Service Interaction Tests**: Verify components can call services
   - Process manager calls
   - Proxy server calls
   - Log store calls
   - etc.

### Post-Migration Tests (Run After Each Controller Removal)
1. **Regression Tests**: Verify no functionality lost
2. **Performance Tests**: Verify <50ms key event latency
3. **Integration Tests**: Verify full user workflows work

## Risk Mitigation

### Risk 1: Breaking Circular Dependencies
**Issue**: Controllers call each other (input → script selector, layout → all views)
**Mitigation**:
- Migrate in dependency order
- Use events for cross-component communication
- Remove controller references incrementally

### Risk 2: State Management Complexity
**Issue**: Some controllers have complex state machines (ai_coder, script_selector)
**Mitigation**:
- Document state transitions before migration
- Write comprehensive state tests
- Preserve state machine logic in component

### Risk 3: Testing Coverage Gaps
**Issue**: Current test coverage is low, hard to verify no regressions
**Mitigation**:
- Write integration tests BEFORE migration
- Test each view manually after migration
- Add automated E2E tests for critical paths

### Risk 4: Keyboard Handling Regression
**Issue**: Input routing is complex with overlays, focus, priority
**Mitigation**:
- Keep hierarchical key handler intact
- Test all keyboard shortcuts after each migration
- Document key handling flow

## Success Criteria

1. **All 12 controllers deleted** (excluding navigation, system, filebrowser, notifications)
2. **All 203 controller references removed** from model.go
3. **Zero functionality regressions** (all tests pass)
4. **Performance maintained** (<50ms key event response)
5. **Code size reduced** (~8,000 lines removed)
6. **Architecture simplified** (2 layers instead of 4)

## Next Steps

1. Create Phase 1 artifacts (data-model.md, contracts/, quickstart.md)
2. Generate task breakdown (tasks.md)
3. Begin implementation with integration tests first (TDD)
