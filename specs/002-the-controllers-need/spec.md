# Feature Specification: Controller to Service/Component Migration

**Feature Branch**: `002-the-controllers-need`
**Created**: 2025-10-06
**Status**: Draft
**Input**: User description: "The controllers need to be fully migrated to the service and component model"

## Execution Flow (main)
```
1. Parse user description from Input
   → Controllers currently exist but should be removed
2. Extract key concepts from description
   → Actors: Developers maintaining TUI code
   → Actions: Migrate controller logic to services and components
   → Data: TUI state, user interactions, service integrations
   → Constraints: Must preserve existing functionality, use BubbleTea patterns
3. For each unclear aspect:
   → [RESOLVED] Architecture pattern clear from codebase analysis
4. Fill User Scenarios & Testing section
   → User flow: Component needs data → calls service → updates via BubbleTea
5. Generate Functional Requirements
   → Each requirement is testable via build and integration tests
6. Identify Key Entities
   → Controllers (to remove), Services (to use), Components (BubbleTea)
7. Run Review Checklist
   → No implementation ambiguities - pattern is established
8. Return: SUCCESS (spec ready for planning)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT the architecture should achieve and WHY
- ❌ Avoid HOW to refactor each controller (no specific code changes)
- 👥 Written for developers understanding the architecture debt

---

## User Scenarios & Testing

### Primary User Story
As a developer maintaining Brummer's TUI, I need the codebase to follow a consistent service-and-component architecture so that I can understand, modify, and test the code without navigating through multiple layers of controller indirection.

### Acceptance Scenarios
1. **Given** a TUI component needs to execute a script, **When** the component is triggered by user input, **Then** it should call the process service directly without going through a controller intermediary
2. **Given** a developer wants to add a new TUI view, **When** they create a new component, **Then** they should only need to implement BubbleTea methods and call existing services
3. **Given** the TUI receives a keyboard event, **When** a component handles the key, **Then** it should process the input directly or delegate to a child component using BubbleTea patterns
4. **Given** a service emits an event, **When** a component needs to react, **Then** it should receive the event through BubbleTea messages, not controller method calls

### Edge Cases
- What happens when a controller has complex state management logic that spans multiple services?
- How does the system handle keyboard input routing that currently goes through InputController?
- What happens to view-specific logic currently in controllers like LogsViewController or ErrorsViewController?

## Requirements

### Functional Requirements
- **FR-001**: TUI components MUST interact with services (process, proxy, logs, etc.) directly without controller intermediaries
- **FR-002**: TUI MUST handle all keyboard input using BubbleTea's Update() pattern and component-level key handlers
- **FR-003**: TUI components MUST manage their own view state using BubbleTea model patterns
- **FR-004**: TUI MUST route service events to components through BubbleTea's message system
- **FR-005**: System MUST preserve all existing TUI functionality after controller removal
- **FR-006**: Components MUST be self-contained and testable in isolation from the main TUI model
- **FR-007**: System MUST eliminate circular dependencies between TUI components and controllers
- **FR-008**: TUI initialization MUST not instantiate any controller objects
- **FR-009**: All controller files MUST be deleted from internal/tui directory tree
- **FR-010**: Model.go MUST not contain any controller field declarations or method calls

### Key Entities

- **Controllers (to remove)**: Currently 17 files handling input routing, view management, and service coordination. These represent an incomplete refactoring attempt that created architectural debt.

- **Services (to use)**: Core business logic in internal/process, internal/proxy, internal/logs, etc. These provide the actual functionality and should be called directly by components.

- **Components (BubbleTea pattern)**: Self-contained UI elements implementing Init(), Update(), and View() methods. These should manage their own state and interact with services directly.

- **Messages (BubbleTea events)**: tea.Msg types that flow through the Update() cycle. Service events should be wrapped as messages for component consumption.

---

## Review & Acceptance Checklist

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on architecture value and maintainability
- [x] Written for developers maintaining the codebase
- [x] All mandatory sections completed

### Requirement Completeness
- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable (build succeeds, tests pass, no controller files exist)
- [x] Success criteria are measurable (count of controller files = 0, grep for controller references = 0)
- [x] Scope is clearly bounded (TUI layer only, services unchanged)
- [x] Dependencies identified (requires understanding of existing controllers' responsibilities)

---

## Execution Status

- [x] User description parsed
- [x] Key concepts extracted (controllers, services, components, BubbleTea)
- [x] Ambiguities marked (none - architecture pattern is clear)
- [x] User scenarios defined (developer experience and component patterns)
- [x] Requirements generated (10 functional requirements)
- [x] Entities identified (controllers to remove, services to use, components to enhance)
- [x] Review checklist passed

---

## Context from Codebase Analysis

### Current State
- **17 controller files** still exist in internal/tui
- **Model.go has 16 controller fields** with 203 references
- **Controllers delegate to each other** creating circular dependencies
- **Components exist** (37 files) but operate in parallel with controllers
- **32,489 lines of TUI code** across 119 files indicates excessive complexity

### Controllers to Remove
1. input_controller.go (473 lines) - Keyboard input routing
2. web_view_controller.go (969 lines) - Web view management
3. ai_coder_controller.go (655 lines) - AI coder integration
4. errors_view_controller.go (441 lines) - Error display
5. logs_view_controller.go (large) - Log view management
6. process_view_controller.go - Process list management
7. urls_view_controller.go - URL display
8. settings_controller.go - Settings UI
9. layout_controller.go - Layout rendering
10. event_controller.go - Event handling
11. command_window_controller.go - Command palette
12. script_selector_controller.go (458 lines) - Script selection
13. mcp_debug_controller.go (496 lines) - MCP debugging
14. navigation/controller.go - View navigation
15. filebrowser/controller.go - File browser
16. system/controller.go - System operations
17. notifications/controller.go - Notification management

### Architecture Debt
The controllers represent a **failed refactoring attempt** where someone tried to separate concerns but ended up creating:
- Multiple layers of indirection (Model → Controllers → Components → Services)
- Circular dependencies (controllers calling each other)
- Duplicate state management (controllers and components both tracking state)
- Difficult testing (need to mock controllers to test components)

The goal is to complete the refactoring to a clean two-layer architecture:
- **Components** (BubbleTea) → **Services** (business logic)
