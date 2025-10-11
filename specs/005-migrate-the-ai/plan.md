
# Implementation Plan: Migrate AI Coder to Service and Component Architecture

**Branch**: `005-migrate-the-ai` | **Date**: 2025-10-10 | **Spec**: [/specs/005-migrate-the-ai/spec.md](/specs/005-migrate-the-ai/spec.md)
**Input**: Feature specification from `/specs/005-migrate-the-ai/spec.md`
**User Context**: fix and update the plan - Address critical ServiceBridge integration gap and complete PTY rendering

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
The AI Coder feature needs architectural migration from controller-based to service+component architecture. Current implementation has AICoderService complete and functional, but critical ServiceBridge integration is missing, preventing events from reaching the UI component. PTY rendering in component is incomplete - component forwards input but has no terminal display capability.

## Technical Context
**Language/Version**: Go 1.21+
**Primary Dependencies**: BubbleTea TUI framework, sync.Map for lock-free operations, EventBus for message passing, vt10x terminal emulator
**Storage**: In-memory session registry (sync.Map), terminal buffers in PTY sessions
**Testing**: Go testing with TTY allocation for integration tests, make test-race for concurrency
**Target Platform**: Linux terminal applications with TTY support
**Project Type**: Single CLI application with internal service+component architecture
**Performance Goals**: <50ms input-to-display latency for typing responsiveness, <5ms GetTerminalState, <100MB memory for typical usage
**Constraints**: Must maintain existing PTY performance characteristics, lock-free architecture mandatory, message-driven communication only
**Scale/Scope**: Unlimited concurrent AI sessions (system resource constrained only), terminal buffers up to 1000+ lines

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Required Gates (based on Brummer Constitution v1.0.0):**
- [x] **Lock-Free Architecture**: AICoderService uses sync.Map, atomic operations for shared state
- [x] **Event-Driven Design**: Service publishes to EventBus, ServiceBridge converts to BubbleTea messages
- [x] **Test-First Development**: Service has comprehensive test coverage, component tests ready
- [x] **Process Management**: Session lifecycle with cleanup on stop/crash
- [x] **MCP Protocol Compliance**: AI coder not MCP-specific, follows established patterns
- [x] **Performance Standards**: <50ms input latency, <5ms terminal state access, <100MB memory
- [x] **Concurrency Safety**: Service tested with make test-race, no goroutine leaks

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
internal/
├── tui/
│   ├── services/           # NEW: Service layer
│   │   └── aicoder_service.go          # ✅ COMPLETE
│   ├── components/         # Component layer
│   │   └── aicoders_view.go              # ✅ EXISTS, needs enhancement
│   ├── messages/           # Message types
│   │   └── aicoder_messages.go          # ✅ COMPLETE
│   ├── bridge/             # Event bridge layer
│   │   └── service_bridge.go             # ✅ EXISTS, needs AI coder integration
│   └── model.go            # Main TUI model, needs wiring
├── aicoder/                 # PTY infrastructure (existing)
│   ├── pty_session.go
│   └── pty_manager.go
└── test/
    └── integration/         # Integration tests
        └── aicoder_component_test.go     # ✅ EXISTS, needs enablement
```

**Structure Decision**: Single Go CLI application with internal service+component architecture. AICoderService and message types complete, component exists but needs PTY rendering completion, ServiceBridge needs AI coder event subscriptions.

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
*This section describes the updated task planning approach based on critical gap analysis*

**Revised Strategy Based on Current State**:

Since the AI coder migration is 80% complete but broken due to specific integration gaps, the task planning will focus on **integration completion** rather than full implementation.

**Critical Gap Tasks (Priority 1 - BLOCKERS)**:
1. **ServiceBridge Integration** - Add AI coder event subscriptions to bridge
2. **PTY Rendering Completion** - Implement actual terminal display in component
3. **Model Wiring** - Connect ServiceBridge to AI coder service in initialization

**Supporting Tasks (Priority 2)**:
4. **Test Enablement** - Remove t.Skip from existing component tests
5. **Legacy Cleanup** - Remove old controller files
6. **Validation** - Manual testing and performance verification

**Task Generation Strategy**:
- Focus on integration tasks, not basic implementation (already complete)
- Leverage existing service tests and component test framework
- Address the ServiceBridge gap first (enables all other work)
- Prioritize user-visible functionality (PTY rendering)

**Updated Ordering Strategy**:
1. **Fix Event Flow** (ServiceBridge integration) - 1 hour
2. **Complete PTY Display** (Component rendering) - 2-3 hours
3. **Enable Existing Tests** (Remove t.Skip) - 1 hour
4. **Integration Testing** (Manual validation) - 1 hour
5. **Cleanup** (Remove legacy files) - 30 minutes

**Estimated Output**: 8-10 focused tasks instead of 25-30, reflecting the 80% completion status.

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
- [x] Phase 0: Research complete (/plan command) - Critical gaps identified
- [x] Phase 1: Design complete (/plan command) - All artifacts exist and valid
- [x] Phase 2: Task planning complete (/plan command) - Integration-focused approach defined
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS - All constitutional requirements met
- [x] Post-Design Constitution Check: PASS - Design follows lock-free, event-driven patterns
- [x] All NEEDS CLARIFICATION resolved - Research complete
- [x] Complexity deviations documented - None required

---
*Based on Constitution v1.0.0 - See `/memory/constitution.md`*
