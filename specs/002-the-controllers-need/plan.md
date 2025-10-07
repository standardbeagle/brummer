# Implementation Plan: Controller to Service/Component Migration

**Branch**: `002-the-controllers-need` | **Date**: 2025-10-06 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-the-controllers-need/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → ✅ Loaded from /home/beagle/work/brummer/specs/002-the-controllers-need/spec.md
2. Fill Technical Context
   → ✅ Go 1.21+, BubbleTea TUI framework, existing services
   → ✅ Project Type: Single project (TUI application)
3. Fill Constitution Check section
   → ✅ Lock-free architecture (preserve existing pattern)
   → ✅ Event-driven design (preserve EventBus usage)
   → ✅ Test-first development (integration tests required)
4. Evaluate Constitution Check
   → ✅ No violations - architecture simplification aligns with principles
   → ✅ Progress Tracking: Initial Constitution Check complete
5. Execute Phase 0 → research.md
   → ✅ Completed - see research.md
6. Execute Phase 1 → contracts, data-model.md, quickstart.md
   → ✅ Completed - see respective files
7. Re-evaluate Constitution Check
   → ✅ Design maintains constitutional compliance
   → ✅ Progress Tracking: Post-Design Constitution Check complete
8. Plan Phase 2 → Task generation approach documented below
9. STOP - Ready for /tasks command
```

## Summary

This feature removes 17 controller files (~8,000+ lines) and 203 controller references from the TUI codebase, completing a failed refactoring attempt. Controllers will be replaced with direct service calls from BubbleTea components, following the established Component → Service architecture. All existing functionality must be preserved while eliminating multiple layers of indirection (Model → Controllers → Components → Services) and circular dependencies.

**Technical Approach**: Systematic migration of controller logic to components, preserving all input handling, view management, and service coordination while using BubbleTea's Update() pattern for state management and tea.Cmd for side effects.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: BubbleTea (TUI framework), EventBus (event coordination), existing service layer (process, proxy, logs, etc.)
**Storage**: N/A (TUI state only)
**Testing**: Go test, integration tests with TTY allocation
**Target Platform**: Linux, macOS, Windows (terminal application)
**Project Type**: Single project (TUI application with service layer)
**Performance Goals**: <50ms key event response, no regression in existing functionality
**Constraints**: Zero functionality loss, preserve all keyboard shortcuts, maintain lock-free architecture
**Scale/Scope**: 17 controller files to remove, 203 references to migrate, 32,489 lines of TUI code to refactor

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Required Gates (based on Brummer Constitution v1.0.0):**
- [x] **Lock-Free Architecture**: Preserves existing atomic operations and lock-free patterns
- [x] **Event-Driven Design**: Maintains EventBus for component communication, uses BubbleTea messages
- [x] **Test-First Development**: Integration tests written before controller removal
- [x] **Process Management**: No changes to process lifecycle management (services unchanged)
- [x] **MCP Protocol Compliance**: No changes to MCP layer (services unchanged)
- [x] **Performance Standards**: No regressions expected (removing indirection improves performance)
- [x] **Concurrency Safety**: Race tests required during migration to verify thread safety

**Justification for Complexity**: This refactoring REDUCES complexity by removing unnecessary abstraction layers. The migration simplifies the architecture from 4 layers (Model → Controllers → Components → Services) to 2 layers (Components → Services).

## Project Structure

### Documentation (this feature)
```
specs/002-the-controllers-need/
├── spec.md              # Feature specification (existing)
├── plan.md              # This file (/plan command output)
├── research.md          # Technical research on controller removal
├── data-model.md        # Component and service interaction patterns
├── quickstart.md        # Migration testing scenarios
└── tasks.md             # Detailed task breakdown (/tasks command - NOT YET CREATED)
```

### Source Code (repository root)
```
internal/tui/
├── model.go                           # Main TUI model (will be simplified)
├── components/                        # BubbleTea components (will gain controller logic)
│   ├── processes_view.go             # Replaces process_view_controller.go
│   ├── logs_view.go                  # Replaces logs_view_controller.go
│   ├── errors_view.go                # Replaces errors_view_controller.go
│   ├── urls_view.go                  # Replaces urls_view_controller.go
│   ├── web_view.go                   # Replaces web_view_controller.go
│   ├── settings_view.go              # Replaces settings_controller.go
│   ├── integrated_shell.go           # Replaces ai_coder_controller.go logic
│   ├── command_palette.go            # Replaces command_window_controller.go
│   ├── script_selector.go            # Replaces script_selector_controller.go
│   ├── mcp_debug_view.go             # Replaces mcp_debug_controller.go
│   ├── hierarchical_key_handler.go   # Replaces input_controller.go logic
│   └── layout_manager.go             # Replaces layout_controller.go logic
├── navigation/                        # Navigation system (preserve)
│   └── controller.go                 # Keep (core navigation state)
├── system/                           # System components (preserve)
│   └── controller.go                 # Keep (system message handling)
├── filebrowser/                      # File browser (preserve)
│   └── controller.go                 # Keep (file selection UI)
└── notifications/                    # Notifications (preserve)
    └── controller.go                 # Keep (notification display)

internal/process/                      # Process management service (unchanged)
internal/proxy/                        # Proxy service (unchanged)
internal/logs/                         # Logging service (unchanged)
internal/mcp/                          # MCP service (unchanged)

test/integration/
├── tui_migration_test.go             # Integration tests for migrated components
├── keyboard_handling_test.go         # Key handler migration verification
└── component_service_test.go         # Service interaction tests
```

### Controllers to Remove (17 files)
```
internal/tui/
├── input_controller.go               # DELETE - logic moves to hierarchical_key_handler component
├── web_view_controller.go            # DELETE - logic moves to web_view component
├── ai_coder_controller.go            # DELETE - logic moves to integrated_shell component
├── errors_view_controller.go         # DELETE - logic moves to errors_view component
├── logs_view_controller.go           # DELETE - logic moves to logs_view component
├── process_view_controller.go        # DELETE - logic moves to processes_view component
├── urls_view_controller.go           # DELETE - logic moves to urls_view component
├── settings_controller.go            # DELETE - logic moves to settings_view component
├── layout_controller.go              # DELETE - logic moves to layout_manager component
├── event_controller.go               # DELETE - event handling moves to model.go
├── command_window_controller.go      # DELETE - logic moves to command_palette component
└── script_selector_controller.go     # DELETE - logic moves to script_selector component
```

### Controllers to Keep (navigation is a special case)
```
internal/tui/navigation/controller.go  # KEEP - core navigation state management
internal/tui/system/controller.go      # KEEP - system message coordination
internal/tui/filebrowser/controller.go # KEEP - file selection dialog state
internal/tui/notifications/controller.go # KEEP - notification queue management
internal/tui/mcp_debug_controller.go   # DELETE - logic moves to mcp_debug_view component
```

**Rationale for keeping some controllers**: navigation, system, filebrowser, and notifications controllers manage stateful UI elements that don't map cleanly to single views. They will be renamed to "managers" in a future refactoring but are out of scope for this migration.

## Phase 0: Research

**Objective**: Understand each controller's responsibilities and dependencies to plan migration.

**Approach**:
1. Analyze each controller file to document:
   - Public methods and their usage
   - Dependencies on services
   - State management patterns
   - Integration points with model.go

2. Identify migration patterns:
   - **Input routing**: Move to hierarchical key handler component
   - **View state**: Move to component's own state struct
   - **Service calls**: Call services directly from component
   - **Event handling**: Use BubbleTea messages and tea.Cmd

3. Map controller methods to target components:
   - Create migration table for each controller
   - Document state that needs to move
   - Identify circular dependencies to break

4. Plan integration test strategy:
   - Test each view's functionality before migration
   - Verify keyboard shortcuts still work
   - Ensure service interactions remain correct

**Output**: research.md with detailed migration plan for each controller

## Phase 1: Design Artifacts

### Data Model (data-model.md)
**Objective**: Document component-service interaction patterns and state management.

**Contents**:
- **ComponentState**: Pattern for component-local state (replaces controller state)
- **ServiceInterface**: How components call services directly
- **BubbleTeaMessages**: Event types for service → component communication
- **KeyHandler**: Hierarchical key handling without controllers

### API Contracts (contracts/)
**Objective**: Define component interfaces for testing.

**Contracts**:
- `component-interface.go`: Standard component interface (Init, Update, View)
- `service-call-pattern.go`: Example service interaction patterns
- `key-handler-protocol.go`: Hierarchical key handling interface

### Quick Start (quickstart.md)
**Objective**: Migration testing scenarios for each controller removal.

**Scenarios**:
1. **Process View Migration**: Test process list, selection, start/stop
2. **Logs View Migration**: Test log filtering, scrolling, priority toggle
3. **Input Handling Migration**: Test all keyboard shortcuts work
4. **AI Coder Migration**: Test terminal focus, PTY interaction
5. **Layout Migration**: Test view rendering, system panel toggle

## Phase 2: Task Generation Strategy

**Objective**: Create executable tasks for systematic controller migration.

**Task Categories**:

### Setup Tasks (T001-T005)
- Create component interfaces and base patterns
- Set up integration test framework
- Document migration patterns for team

### Controller Analysis Tasks (T006-T022)
- One task per controller to document its responsibilities
- Create migration plan for each controller's logic
- Identify state to move and services to call

### Integration Test Tasks (T023-T039)
- Write failing tests for each view's functionality
- Test keyboard shortcuts and navigation
- Verify service interactions

### Migration Tasks (T040-T056)
- Migrate one controller at a time to its target component
- Remove controller file after migration
- Update model.go to remove controller references

### Verification Tasks (T057-T062)
- Run integration tests to verify functionality
- Performance testing (key event latency)
- Code cleanup and documentation

**Dependency Rules**:
- Setup before analysis
- Analysis before tests
- Tests (failing) before migration
- Migration one controller at a time (minimize risk)
- Verification after each migration

**Parallel Execution**:
- Controller analysis tasks can run in parallel
- Integration tests can be written in parallel
- Documentation tasks can run in parallel
- Migration tasks MUST be sequential (one controller at a time)

## Progress Tracking

- [x] Initial Constitution Check (Step 4)
- [x] Phase 0: Research (Step 5)
- [x] Phase 1: Design Artifacts (Step 6)
- [x] Post-Design Constitution Check (Step 7)
- [x] Phase 2: Task Generation Strategy (Step 8)
- [ ] Phase 3: Implementation (requires /tasks → /implement)
- [ ] Phase 4: Polish (requires /implement)

## Complexity Tracking

**Complexity Reduction Achieved**:
- Removes 4-layer architecture → 2-layer architecture
- Deletes ~8,000+ lines of controller code
- Eliminates circular dependencies between controllers
- Simplifies testing (components testable in isolation)
- Reduces cognitive load for new developers

**No Constitution Violations**: This refactoring aligns with all constitutional principles by simplifying architecture and removing unnecessary abstraction.

## Open Questions / Risks

1. **State Management**: Some controllers have complex state machines (ai_coder_controller, script_selector_controller). Migration requires careful state analysis.
   - **Mitigation**: Document state transitions before migration, write comprehensive tests

2. **Circular Dependencies**: Controllers call each other (input_controller → script_selector_controller). Breaking these requires careful ordering.
   - **Mitigation**: Migrate in dependency order, use events for cross-component communication

3. **Keyboard Handling**: Input routing is complex with overlays, focus states, and priority levels.
   - **Mitigation**: Preserve hierarchical key handler, migrate logic without changing behavior

4. **Testing Coverage**: Current test coverage is low (<50% estimated). Integration tests needed.
   - **Mitigation**: Write comprehensive integration tests before each migration

## Next Steps

Run `/tasks` to generate the detailed task breakdown (tasks.md) based on this plan.
