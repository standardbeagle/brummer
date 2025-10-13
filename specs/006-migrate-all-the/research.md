# Research: Controller-to-Component Migration

**Date**: 2025-10-11
**Status**: Complete

## Executive Summary

The controller-to-component migration is a pure architectural refactoring that moves 5 remaining controllers in Brummer from the legacy controller pattern to the established service+component architecture. All functional requirements are well-defined and the existing codebase provides the target patterns to follow. This migration preserves 100% of user-facing functionality while improving code maintainability and consistency.

## Current Architecture Analysis

### Existing Controllers to Migrate

**InputController** (`internal/tui/input_controller.go`)
- **Responsibilities**: Keyboard input routing, global key bindings, view switching logic, hierarchical key handling
- **Dependencies**: Model state, navigation controller, key mappings, process manager
- **Key Methods**: `HandleKeyMsg()`, `handleGlobalKeys()`, `handleQuit()`, `setupHierarchicalKeyHandlers()`
- **State Management**: Last quit press tracking, hierarchical context, debug mode flags

**NavigationController** (`internal/tui/navigation/controller.go`)
- **Responsibilities**: View switching, navigation state management, view order cycling, callback handling
- **Dependencies**: Model state for view-specific operations
- **Key Methods**: `SwitchTo()`, `CycleNext()`, `GetViewForNumber()`, `GetCurrentView()`
- **State Management**: Current view, debug mode, callback functions for view changes

**SystemController** (`internal/tui/system/controller.go`)
- **Responsibilities**: System message display, panel management, unread indicators, message lifecycle
- **Dependencies**: Message formatting, viewport management, navigation integration
- **Key Methods**: `AddMessage()`, `UpdateUnreadIndicator()`, `OverlaySystemPanel()`, `ToggleExpanded()`
- **State Management**: Message buffer, unread indicators, expansion state, viewport dimensions

**NotificationsController** (`internal/tui/notifications/controller.go`)
- **Responsibilities**: Toast notification display, duration management, notification lifecycle
- **Dependencies**: BubbleTea timing system
- **Key Methods**: `Show()`, `ShowWithDuration()`, `HandleMsg()`, `IsActive()`
- **State Management**: Message content, notification timing, active status

**FileBrowserController** (`internal/tui/filebrowser/controller.go`)
- **Responsibilities**: File selection dialogs, directory navigation, file listing, callback handling
- **Dependencies**: OS file system operations, BubbleTea list components
- **Key Methods**: `Show()`, `Hide()`, `HandleInput()`, `LoadFileList()`, `GetSelectedIndex()`
- **State Management**: Showing state, current path, file list, selected index

## Target Architecture Analysis

### Component Framework Patterns (Established)

**Existing Component Structure**: Brummer already uses a mature component framework with:
- Service layer in `internal/tui/services/` for business logic
- Component layer in `internal/tui/components/` for UI implementation
- Message types in `internal/tui/messages/` for service-component communication
- ServiceBridge in `internal/tui/bridge/` for event routing

**Migration Pattern Reference**: The AI Coder migration (Feature 005) provides the exact pattern to follow:
1. Extract business logic into services
2. Create message types for communication
3. Implement components with BubbleTea Model/Update/View pattern
4. Use ServiceBridge to route events
5. Remove controller files and references

### Integration Points

**Model Integration**: Controllers currently integrate with:
- Main TUI model (`internal/tui/model.go`)
- Navigation state management
- Process manager for command execution
- Log store for system messages
- Update channel for message passing

**EventBus Integration**: Controllers need to publish events through EventBus for:
- Navigation changes (view switching)
- System messages (errors, notifications)
- Input handling results
- File selection outcomes

## Research Findings

### Decision 1: Migration Approach

**Decision**: Big bang migration - migrate all controllers simultaneously

**Rationale**:
- All controllers have interdependencies through the model
- Component state synchronization is simpler when done together
- Single release reduces testing complexity and migration overhead
- User experience consistency maintained throughout

**Requirements from Clarifications**:
- "Big bang migration - migrate all controllers simultaneously in one release"
- "No rollback capability - proceed with migration even if issues occur"

**Implementation Strategy**:
1. Create all target components first (parallel development)
2. Update ServiceBridge to include new component event subscriptions
3. Modify model initialization to use components instead of controllers
4. Remove all controller files in single commit
5. Update all references throughout codebase

### Decision 2: Component Design Approach

**Decision**: Follow established AI Coder migration pattern exactly

**Rationale**:
- Proven pattern works with Brummer's architecture
- Existing component framework and tools can be reused
- Message passing patterns already established
- ServiceBridge integration pattern is validated

**Implementation Reference**:
- Service creation: `internal/tui/services/aicoder_service.go` (506 lines)
- Message types: `internal/tui/messages/aicoder_messages.go` (126 lines)
- Component implementation: `internal/tui/components/aicoders_view.go` (enhanced)
- ServiceBridge integration: Event subscriptions and message conversion

### Decision 3: State Management Strategy

**Decision**: Component state managed through BubbleTea Update() cycle only

**Rationale**:
- Constitutional requirement: "TUI components ONLY modify model state within BubbleTea Update() methods"
- Lock-free architecture with atomic operations
- Event-driven design ensures consistency
- Eliminates race conditions in UI state

**State Migration Approach**:
- Controller state becomes component state
- Model updates routed through ServiceBridge events
- Atomic operations prevent corruption
- No direct state mutation outside Update() methods

### Decision 4: Testing Strategy

**Decision**: Comprehensive TDD with TTY allocation

**Rationale**:
- Constitutional requirement: "TDD is MANDATORY"
- TUI functionality requires real terminal testing
- Race condition testing essential for state synchronization
- Migration validation needs thorough testing

**Testing Implementation**:
- Unit tests for component logic
- Integration tests for service-component communication
- TTY tests for UI interaction
- Race condition tests with `make test-race`
- Manual validation for complete user workflows

### Decision 5: Legacy Code Handling

**Decision**: Complete removal of controller files and references

**Rationale**:
- Eliminates architectural inconsistency
- Reduces maintenance burden
- Prevents confusion between old and new patterns
- Forces migration completion

**Cleanup Strategy**:
- Delete all controller files after migration
- Update import statements throughout codebase
- Remove controller interfaces from type definitions
- Update documentation and comments

## Complexity Tracking

| Aspect | Complexity Level | Mitigation Strategy |
|--------|------------------|-------------------|
| State Synchronization | Medium | Atomic operations, event-driven updates |
| Controller Interdependencies | High | Single release migration, ServiceBridge coordination |
| Testing Coverage | Medium | TTY allocation, race condition detection |
| User Experience Risk | Low | 100% functionality preservation requirement |

## Research Validation

### Feasibility Assessment: HIGH
- ✅ All target patterns already proven in codebase
- ✅ Tooling and infrastructure established
- ✅ Team familiarity with component architecture
- ✅ Clear success criteria and acceptance testing

### Risk Assessment: LOW to MEDIUM
- **Medium Risk**: State synchronization during transition (mitigated by atomic operations)
- **Low Risk**: User experience regression (mitigated by functional preservation requirement)
- **Low Risk**: Technical complexity (mitigated by proven patterns)

### Resource Requirements: MEDIUM
- Development time: 2-3 weeks for full migration
- Testing time: 1 week for comprehensive validation
- No additional infrastructure or external dependencies required

## Next Steps

1. **Phase 1**: Design components, contracts, and integration tests
2. **Phase 2**: Generate detailed task breakdown with dependency ordering
3. **Phase 3**: Execute migration following TDD approach
4. **Phase 4**: Validation and cleanup

The research confirms that this migration is technically feasible with low architectural risk, following established patterns in the codebase, and with clear success criteria defined.