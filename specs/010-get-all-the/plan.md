
# Implementation Plan: Complete Tab Functionality with Automated Testing

**Branch**: `010-get-all-the` | **Date**: 2025-10-19 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/beagle/work/brummer/specs/010-get-all-the/spec.md`

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
Make all 7 TUI tabs (Processes, Errors, Logs, URLs, Web, Proxy, Settings, Command Window) fully functional with comprehensive automated testing. Focus on completing existing tab functionality, adding error grouping intelligence, implementing clear commands for Errors/Logs/Web tabs, and ensuring 100% compliance with Redux/Elm architecture and lock-free patterns. The Errors tab must intelligently group multi-line errors to prevent them from getting lost in log streams. All tabs must have >80% test coverage with automated service tests.

## Technical Context
**Language/Version**: Go 1.21+
**Primary Dependencies**: BubbleTea (TUI framework), Lipgloss (styling), atomic package (lock-free state)
**Storage**: In-memory state with optional file-based persistence (TOML config)
**Testing**: Go test framework with race detector (`make test-race`), integration tests with TTY allocation
**Target Platform**: Linux, macOS, Windows (cross-platform terminal application)
**Project Type**: Single project (TUI application with integrated services)
**Performance Goals**: <5ms View() rendering, <50ms keyboard response times, <5s initial load
**Constraints**: <100MB memory usage, zero race conditions, zero mutexes in state system, no blocking operations in View()
**Scale/Scope**: 7 tabs, 72 functional requirements, >80% test coverage, lock-free architecture

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Required Gates (based on Brummer Constitution v1.0.0):**
- [x] **Lock-Free Architecture**: ✅ All state objects are immutable with atomic operations, zero mutexes
- [x] **Event-Driven Design**: ✅ TUI uses BubbleTea Update() pattern, EventBus for inter-component communication
- [x] **Test-First Development**: ✅ TDD approach with contract tests, integration tests, service tests
- [x] **Process Management**: ✅ Complete lifecycle tracking already implemented (not modified by this feature)
- [x] **MCP Protocol Compliance**: ✅ Not applicable (feature focuses on TUI tabs, not MCP layer)
- [x] **Performance Standards**: ✅ <5ms View() rendering, <50ms keyboard response, <100MB memory
- [x] **Concurrency Safety**: ✅ Race condition testing with `make test-race` required for all changes

**Assessment**: PASS - This feature builds on the existing lock-free Redux/Elm architecture completed in spec 009. All tabs use immutable state, ViewWithState() pattern, and command-based async operations. No architectural deviations required.

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
internal/tui/
├── state/                    # Lock-free immutable state objects
│   ├── master_data.go       # Central state container (single source of truth)
│   ├── processes_state.go   # Processes tab state
│   ├── errors_state.go      # Errors tab state (needs error grouping logic)
│   ├── logs_state.go        # Logs tab state
│   ├── urls_state.go        # URLs tab state
│   ├── web_state.go         # Web tab state (proxy events)
│   ├── proxy_state.go       # Proxy tab state
│   ├── settings_state.go    # Settings tab state
│   └── command_window_state.go  # Command palette state
├── components/              # ViewWithState() pure rendering functions
│   ├── processes_view.go    # Processes tab view
│   ├── errors_view.go       # Errors tab view (needs grouping display)
│   ├── logs_view.go         # Logs tab view
│   ├── urls_view.go         # URLs tab view
│   ├── web_view.go          # Web tab view
│   ├── proxy_view.go        # Proxy tab view
│   ├── settings_view.go     # Settings tab view
│   └── command_window_view.go   # Command palette view
├── services/                # Stateless command creators
│   ├── error_grouping_service.go    # NEW: Error grouping logic
│   ├── log_service.go
│   ├── process_service.go
│   └── proxy_service.go
├── handlers/                # Keyboard event handlers
│   ├── key_handler.go       # Global key routing (needs 'c' key for clear)
│   └── tab_handlers.go      # Tab-specific handlers
├── model.go                 # BubbleTea Model with Update() function
└── messages.go              # Message types for state updates

test/tui/
├── contracts/               # Contract tests for state behavior
│   ├── state_contract_test.go
│   ├── view_contract_test.go
│   └── service_contract_test.go
├── integration/             # Integration tests with TTY allocation
│   ├── tab_navigation_test.go
│   ├── error_grouping_test.go
│   ├── clear_commands_test.go
│   └── keyboard_flow_test.go
└── unit/                    # Unit tests for pure functions
    ├── error_grouping_test.go
    ├── state_mutations_test.go
    └── view_rendering_test.go
```

**Structure Decision**: Single project (TUI application). This feature extends the existing Redux/Elm TUI architecture without adding new top-level components. All work happens within `internal/tui/` and `test/tui/` directories. The main additions are:
1. Error grouping service for intelligent multi-line error parsing
2. Clear command handlers for Errors/Logs/Web tabs
3. Comprehensive test suites for all tab functionality

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
1. **Contract Tests First** (TDD):
   - State contract tests (7 test cases × 8 state objects = ~56 tests)
   - View contract tests (10 test cases × 7 views = ~70 tests)
   - Service contract tests (10 test cases × 2 services = ~20 tests)
   - All tests MUST fail initially (no implementation yet)

2. **State Implementation**:
   - ErrorsState with grouping support [P]
   - WebState for proxy events [P]
   - Enhance existing state objects [P]
   - All state mutations follow immutable pattern

3. **Service Implementation**:
   - ErrorGroupingService (parse and group errors) [P]
   - LogClearService (clear log commands) [P]
   - ErrorClearService (clear error commands) [P]
   - WebClearService (clear web events) [P]

4. **View Implementation**:
   - Update ErrorsView to display grouped errors
   - Update WebView to display proxy events
   - All views use ViewWithState() pattern
   - All views remain pure (no side effects)

5. **Handler Implementation**:
   - Add 'c' key handler for clear commands
   - Add 'C' key handler for confirmation prompts
   - Context-aware routing (only in Errors/Logs/Web tabs)

6. **Integration Tests**:
   - Tab navigation flow tests
   - Error grouping end-to-end tests
   - Clear command workflow tests
   - State preservation tests

**Ordering Strategy**:
1. **Phase A: Test Infrastructure** (Parallel)
   - Contract test files for all components
   - Test fixtures and mock data
   - Helper functions for TTY allocation

2. **Phase B: Core State** (Parallel after Phase A)
   - State object implementations
   - State validation logic
   - Retention limit enforcement

3. **Phase C: Services** (Parallel after Phase B)
   - Error grouping service
   - Clear command services
   - Service configuration

4. **Phase D: Views** (Parallel after Phase C)
   - View component updates
   - Rendering logic
   - Terminal size handling

5. **Phase E: Integration** (Sequential after Phase D)
   - Handler wiring
   - Message routing
   - End-to-end flows

6. **Phase F: Validation** (Sequential after Phase E)
   - Run all tests
   - Coverage verification (>80%)
   - Performance benchmarking
   - Race condition testing

**Estimated Output**: ~45-50 numbered tasks in tasks.md

**Task Categories**:
- [P] = Parallel (independent files, can run concurrently)
- [S] = Sequential (dependencies on previous tasks)
- [TEST] = Test-first task (write failing tests)
- [IMPL] = Implementation task (make tests pass)

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
- [x] Phase 0: Research complete (/plan command) - research.md created
- [x] Phase 1: Design complete (/plan command) - data-model.md, contracts/, quickstart.md, CLAUDE.md updated
- [x] Phase 2: Task planning complete (/plan command - approach described, ready for /tasks)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS (no architectural deviations in Phase 1 design)
- [x] All NEEDS CLARIFICATION resolved (none existed)
- [x] Complexity deviations documented (none required)

**Artifacts Generated**:
- ✅ research.md (Phase 0)
- ✅ data-model.md (Phase 1)
- ✅ contracts/state_contract.md (Phase 1)
- ✅ contracts/view_contract.md (Phase 1)
- ✅ contracts/service_contract.md (Phase 1)
- ✅ quickstart.md (Phase 1)
- ✅ CLAUDE.md updated (Phase 1)
- ✅ plan.md (this file)

---
*Based on Constitution v1.0.0 - See `/memory/constitution.md`*
