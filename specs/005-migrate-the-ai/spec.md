# Feature Specification: Migrate AI Coder to Service and Component Architecture

**Feature Branch**: `005-migrate-the-ai`
**Created**: 2025-10-07
**Status**: Draft
**Input**: User description: "migrate the ai coder code to the new service and component architecture"

## Execution Flow (main)
```
1. Parse user description from Input
   → AI Coder functionality needs architectural migration
2. Extract key concepts from description
   → Actors: Developers using AI coding assistance
   → Actions: Start/stop AI sessions, view PTY output, interact with AI
   → Data: AI session state, PTY buffers, session history
   → Constraints: Must maintain existing functionality during migration
3. For each unclear aspect:
   → [NEEDS CLARIFICATION: Should existing AI coder features remain unchanged or are enhancements expected?]
   → [NEEDS CLARIFICATION: What is the timeline - should this be done in phases?]
4. Fill User Scenarios & Testing section ✓
5. Generate Functional Requirements ✓
6. Identify Key Entities ✓
7. Run Review Checklist
   → Spec has uncertainties (2 clarification markers)
8. Return: SUCCESS (spec ready for planning with clarifications)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers

---

## Clarifications

### Session 2025-10-07
- Q: Should existing AI coder features remain unchanged or are enhancements expected? → A: Migration plus bug fixes - preserve functionality while fixing known issues (cleanup, resource leaks) but no new features
- Q: What is the delivery approach - single-phase or multi-phase? → A: Two-phase delivery to minimize risk - Phase 1: Extract service layer; Phase 2: Migrate component + integration
- Q: What constitutes acceptable performance for PTY handling? → A: Typing responsiveness is the critical performance metric - user input must feel immediate
- Q: What should happen to session data when AI session crashes or PTY fails? → A: Discard session and clean up immediately - user must restart (data loss acceptable for simplicity)
- Q: What is the maximum number of concurrent AI coding sessions? → A: Unlimited - no hard limit, constrained only by system resources

---

## User Scenarios & Testing *(mandatory)*

### Primary User Story
As a developer using Brummer's AI coding assistance, I need the AI coder feature to work reliably and consistently with the rest of the application, so that I can efficiently get AI help for my development tasks without encountering bugs or inconsistencies caused by architectural mismatches.

### Acceptance Scenarios
1. **Given** a developer has Brummer running with an active project, **When** they activate the AI coder view, **Then** they should see all existing AI sessions and be able to interact with them exactly as before the migration
2. **Given** an AI coding session is active, **When** the developer types commands or requests, **Then** the AI should respond through the PTY interface with no loss of functionality or performance degradation
3. **Given** multiple AI sessions exist, **When** the developer switches between sessions, **Then** each session's state (terminal output, cursor position, scroll state) should be preserved correctly
4. **Given** an AI session generates output, **When** the terminal buffer fills up, **Then** output should be displayed without blocking, stuttering, or data loss
5. **Given** the developer closes an AI session, **When** they view the sessions list, **Then** the closed session should be properly cleaned up and resources released

### Edge Cases
- What happens when an AI session crashes or becomes unresponsive? → Session is immediately discarded, resources cleaned up, user notified
- How does the system handle rapid-fire AI session creation and deletion? → Each session lifecycle must complete independently without resource leaks
- What happens to active sessions when the user switches to a different view and back? → Session state preserved (terminal content, cursor, scroll position)
- How are PTY sessions cleaned up when Brummer exits unexpectedly? → All PTY handles and goroutines must be cleaned up on shutdown
- What happens if PTY output arrives faster than the UI can render it? → Lock-free buffer handles backpressure without blocking or data loss

## Requirements *(mandatory)*

### Functional Requirements

**Session Management**
- **FR-001**: System MUST maintain all existing AI coder session management capabilities (create, list, switch, close sessions)
- **FR-002**: System MUST preserve session state (terminal content, cursor position, scroll offset) when switching between sessions
- **FR-003**: System MUST properly clean up session resources (PTY handles, buffers, goroutines) when sessions end
- **FR-004**: System MUST support unlimited concurrent AI coding sessions without interference (constrained only by available system resources)

**PTY Integration**
- **FR-005**: System MUST maintain the existing lock-free PTY architecture for high-performance terminal output
- **FR-006**: System MUST handle PTY output without blocking or dropping data even under high load
- **FR-007**: System MUST support bi-directional PTY communication (user input to AI, AI output to display)
- **FR-008**: System MUST maintain cursor position and terminal state accurately across screen resizes
- **FR-025**: System MUST prioritize typing responsiveness - user keystrokes must appear immediately without perceptible lag (target: <50ms input-to-display latency)

**User Interface**
- **FR-009**: System MUST display AI coding sessions in the dedicated AI coders view
- **FR-010**: System MUST allow users to scroll through terminal history in each session
- **FR-011**: System MUST show visual indicators for active vs inactive sessions
- **FR-012**: System MUST provide keyboard shortcuts for session navigation and control
- **FR-013**: System MUST display session metadata (session ID, creation time, status)

**Architecture Consistency**
- **FR-014**: AI coder functionality MUST follow the same service and component patterns as other Brummer features (processes, logs, errors, URLs)
- **FR-015**: AI coder state MUST be managed through a dedicated service layer separate from UI logic
- **FR-016**: AI coder UI MUST be implemented as a BubbleTea component receiving updates via messages
- **FR-017**: System MUST NOT have direct dependencies between AI coder UI and internal state (use message-passing only)

**Backward Compatibility & Bug Fixes**
- **FR-018**: Migration MUST preserve all existing AI coder functionality without feature regression
- **FR-019**: System MUST maintain existing keyboard shortcuts and user interactions
- **FR-020**: Migration MUST NOT break existing PTY performance characteristics
- **FR-024**: Migration MAY fix known bugs (resource cleanup, memory leaks, goroutine leaks) without adding new features

**Error Handling**
- **FR-021**: System MUST gracefully handle AI session creation failures
- **FR-022**: When PTY communication fails or session crashes, system MUST immediately discard the session and clean up all resources (no recovery attempt, user must restart)
- **FR-023**: System MUST log AI coder errors to the system message panel for user visibility
- **FR-026**: System MUST notify user when session is discarded due to crash or PTY failure

### Scope Boundaries
- **In Scope**: Architectural migration to service/component pattern, bug fixes for resource cleanup and leaks
- **Out of Scope**: New features (session persistence, session sharing, enhanced terminal capabilities)

### Delivery Phases
- **Phase 1**: Extract AI coder service layer (session management, PTY coordination) with isolated testing
- **Phase 2**: Migrate AI coder component (UI, message handling) and complete integration testing

### Scale & Capacity
- **Concurrent Sessions**: No hard limit - system must handle as many sessions as system resources allow
- **Resource Management**: Each session must properly clean up on close/crash to prevent resource exhaustion

### Key Entities *(include if feature involves data)*

- **AI Coding Session**: Represents a single interactive coding session with an AI assistant
  - Attributes: Session ID, creation timestamp, status (active/idle/closed), associated PTY
  - Relationships: Each session has one PTY connection, one terminal buffer, one view state

- **PTY Connection**: Represents the pseudoterminal connection to the AI backend
  - Attributes: PTY file descriptor, terminal dimensions, buffer state
  - Relationships: Belongs to one AI session, produces terminal output events

- **Terminal Buffer**: Contains the terminal output and state for a session
  - Attributes: Screen content, cursor position, scroll offset, terminal dimensions
  - Relationships: Belongs to one AI session, updated by PTY output

- **Session View State**: UI-specific state for displaying a session
  - Attributes: Scroll position, viewport dimensions, focus state, visible help text
  - Relationships: Belongs to one AI session, managed by AI coders component

---

## Review & Acceptance Checklist
*GATE: Automated checks run during main() execution*

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [ ] No [NEEDS CLARIFICATION] markers remain (2 markers present)
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

---

## Execution Status
*Updated by main() during processing*

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked (2 clarifications needed)
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [ ] Review checklist passed (pending clarifications)

---

## Notes

**Current State**: The AI coder functionality exists in `ai_coder_controller.go` and related PTY files. It uses a controller-based architecture that predates the service and component architecture migration completed in Feature 002.

**Migration Context**: This follows the same architectural patterns established in Feature 002 where controllers were migrated to services and components. The AI coder is the last major feature needing this migration.

**Dependencies**:
- Completion of Feature 004 (codebase cleanup) which documented PTY dependencies
- Understanding of Feature 002 patterns (service layer + component separation)
- PTY infrastructure files remain in current location (Feature 003 will extract them later)

**Success Criteria**:
- AI coder works identically to current implementation
- Code follows service and component architecture patterns
- No performance degradation in PTY handling
- All existing tests pass
- Test coverage maintained at ≥87% (or current baseline)
