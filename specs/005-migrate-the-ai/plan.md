
# Implementation Plan: Migrate AI Coder to Service and Component Architecture

**Branch**: `005-migrate-the-ai` | **Date**: 2025-10-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/beagle/work/brummer/specs/005-migrate-the-ai/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code, or `AGENTS.md` for all other agents).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary

Migrate AI coder functionality from legacy controller-based architecture to the service and component pattern established in Feature 002. The AI coder currently exists as a controller (`ai_coder_controller.go`) with PTY handling (`ai_coder_pty_view.go`) that manages interactive coding sessions. This migration will:

1. Extract session management and PTY coordination into a dedicated service layer
2. Migrate the UI component to use BubbleTea message-passing patterns
3. Preserve all existing functionality while fixing resource cleanup bugs
4. Follow two-phase delivery to minimize risk

The migration maintains the lock-free PTY architecture for high-performance terminal emulation while eliminating controller delegation in favor of direct service calls from components.

## Technical Context
**Language/Version**: Go 1.24.2
**Primary Dependencies**: BubbleTea (TUI framework), creack/pty (pseudoterminal), EventBus (event-driven messaging)
**Storage**: In-memory session state, no persistence required
**Testing**: go test with TTY allocation for integration tests, race detection required
**Target Platform**: Linux/macOS/Windows terminal environments
**Project Type**: Single project (CLI TUI application)
**Performance Goals**: <50ms input-to-display latency for typing responsiveness, lock-free PTY output handling
**Constraints**: No blocking operations in PTY path, immediate session cleanup on crash, unlimited concurrent sessions (resource-constrained)
**Scale/Scope**: 4 files to migrate (ai_coder_controller.go, ai_coder_pty_view.go, ai_coder_debug_forwarder.go, ai_coder_message_handler.go), ~70KB total code

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Required Gates (based on Brummer Constitution v1.0.0):**
- [x] **Lock-Free Architecture**: PTY output uses existing lock-free ring buffer (no new mutexes)
- [x] **Event-Driven Design**: Component communicates via BubbleTea messages, service events via EventBus
- [x] **Test-First Development**: Integration tests written first for each migration phase
- [x] **Process Management**: AI sessions have complete lifecycle tracking (create → active → cleanup)
- [N/A] **MCP Protocol Compliance**: Not applicable - internal TUI feature, no MCP integration
- [x] **Performance Standards**: <50ms typing latency enforced, session cleanup <100ms
- [x] **Concurrency Safety**: Race detection required for PTY goroutine management

## Project Structure

### Documentation (this feature)
```
specs/005-migrate-the-ai/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output - PTY architecture analysis
├── data-model.md        # Phase 1 output - Session state entities
├── quickstart.md        # Phase 1 output - Migration validation
├── contracts/           # Phase 1 output - Service interfaces
└── tasks.md             # Phase 2 output (/tasks command)
```

### Source Code (repository root)
```
internal/tui/
├── ai_coder_controller.go         # [TO REMOVE] - Controller to migrate
├── ai_coder_pty_view.go           # [TO MIGRATE] - PTY view → service
├── ai_coder_debug_forwarder.go    # [TO MIGRATE] - Debug proxy → service
├── ai_coder_message_handler.go    # [TO MIGRATE] - Messages → component
├── ai_coder_controller_test.go    # [TO MIGRATE] - Tests → service tests
├── ai_coder_pty_view_test.go      # [TO MIGRATE] - Tests → component tests
│
├── services/                       # [NEW DIRECTORY]
│   └── aicoder_service.go         # Session management + PTY coordination
│
├── components/
│   ├── aicoders_view.go           # [EXISTS] - Basic session list view
│   └── aicoder_component.go       # [NEW] - Full AI coder UI with PTY
│
└── messages/
    └── aicoder_messages.go        # [NEW] - AI coder-specific messages

test/integration/
├── aicoder_service_test.go        # [NEW] - Service layer tests
└── aicoder_component_test.go      # [NEW] - Component integration tests
```

**Structure Decision**: Single project (CLI TUI application). The AI coder migration follows Feature 002 patterns:
- Service layer in `internal/tui/services/` for business logic
- Component in `internal/tui/components/` for UI rendering
- Messages in `internal/tui/messages/` for event communication
- Integration tests in `test/integration/` with TTY allocation

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh claude`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:

The /tasks command will generate tasks.md based on the two-phase migration strategy defined in research.md:

**Phase 1 Tasks (Service Layer)**:
1. Create service directory structure (internal/tui/services/)
2. Create message types (internal/tui/messages/aicoder_messages.go)
3. Write service interface tests (TDD - tests first)
4. Implement AICoderService (session management, PTY coordination)
5. Write service integration tests (PTY lifecycle, cleanup)
6. Verify service tests pass independently (no TUI dependencies)

**Phase 2 Tasks (Component Migration)**:
7. Write component message handling tests (TDD - tests first)
8. Enhance AICodersViewComponent (message routing, service calls)
9. Write component integration tests (with TTY allocation)
10. Update model.go to wire component to service
11. Remove ai_coder_controller.go
12. Remove ai_coder_pty_view.go (logic moved to service)
13. Remove ai_coder_debug_forwarder.go (logic moved to service)
14. Remove ai_coder_message_handler.go (replaced by messages/)
15. Verify all tests pass (unit + integration)
16. Verify race detection passes (make test-race)

**Ordering Strategy**:
- **TDD Order**: All tests written before implementation (Constitutional requirement)
- **Dependency Order**: Service before component (two-phase delivery)
- **Risk Minimization**: Phase 1 fully validated before Phase 2 starts
- **Parallelization**: Within phases, independent tasks marked [P]
  - Service tests [P] (different test files)
  - Component tests [P] (independent scenarios)
  - File removals [P] (after migration complete)

**Task Categorization**:
- **[TEST]**: Test creation tasks (TDD - done first)
- **[IMPL]**: Implementation tasks (make tests pass)
- **[VERIFY]**: Validation tasks (quickstart scenarios)
- **[CLEAN]**: Cleanup tasks (remove old code)
- **[P]**: Parallelizable (independent of other tasks)

**Artifact Dependencies**:
- data-model.md → defines entities for service implementation
- contracts/aicoder_service.go → defines service API contract
- contracts/aicoder_component.go → defines component interface
- quickstart.md → defines validation scenarios for testing
- research.md → defines migration decisions and patterns

**Estimated Output**: 35-40 numbered tasks in dependency order
- Phase 1 (Service): ~15 tasks
- Phase 2 (Component): ~20 tasks
- Each task with acceptance criteria from quickstart.md
- Performance requirements from spec (FR-025) enforced

**Success Criteria for /tasks**:
- All tasks have clear acceptance criteria
- Test tasks precede implementation tasks
- Quickstart validation scenarios covered
- Performance requirements testable
- Resource cleanup verifiable

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS (no violations, all gates satisfied)
- [x] Post-Design Constitution Check: PASS (design maintains constitutional compliance)
- [x] All NEEDS CLARIFICATION resolved (all technical context specified)
- [N/A] Complexity deviations documented (no constitutional violations)

**Post-Design Constitution Verification**:
- ✅ Lock-Free Architecture: Service uses sync.Map, no new mutexes introduced
- ✅ Event-Driven Design: Component → Service calls, Service → Component messages
- ✅ Test-First Development: Quickstart defines tests before implementation
- ✅ Process Management: Session lifecycle fully tracked (FR-003)
- ✅ Performance Standards: <50ms latency enforced, <100ms cleanup guaranteed
- ✅ Concurrency Safety: Race tests required, goroutine cleanup verified

---
*Based on Constitution v1.0.0 - See `/memory/constitution.md`*
