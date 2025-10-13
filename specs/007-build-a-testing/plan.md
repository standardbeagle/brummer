
# Implementation Plan: TUI Integration Testing Harness

**Branch**: `007-build-a-testing` | **Date**: 2025-01-12 | **Spec**: /home/beagle/work/brummer/specs/007-build-a-testing/spec.md
**Input**: Feature specification from `/specs/007-build-a-testing/spec.md`

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
The TUI Integration Testing Harness will provide automated testing capabilities for Brummer's terminal interface without requiring TTY allocation. The approach focuses on agent-based local testing with limited bash shell environments, investigating existing solutions like wish.Command while preventing workarounds that compromise production functionality.

## Technical Context
**Language/Version**: [e.g., Python 3.11, Swift 5.9, Rust 1.75 or NEEDS CLARIFICATION]  
**Primary Dependencies**: [e.g., FastAPI, UIKit, LLVM or NEEDS CLARIFICATION]  
**Storage**: [if applicable, e.g., PostgreSQL, CoreData, files or N/A]  
**Testing**: [e.g., pytest, XCTest, cargo test or NEEDS CLARIFICATION]  
**Target Platform**: [e.g., Linux server, iOS 15+, WASM or NEEDS CLARIFICATION]
**Project Type**: [single/web/mobile - determines source structure]  
**Performance Goals**: [domain-specific, e.g., 1000 req/s, 10k lines/sec, 60 fps or NEEDS CLARIFICATION]  
**Constraints**: [domain-specific, e.g., <200ms p95, <100MB memory, offline-capable or NEEDS CLARIFICATION]  
**Scale/Scope**: [domain-specific, e.g., 10k users, 1M LOC, 50 screens or NEEDS CLARIFICATION]

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Required Gates (based on Brummer Constitution v1.0.0):**
- [x] **Lock-Free Architecture**: No mutex-based synchronization unless absolutely justified - Testing harness will use atomic operations
- [x] **Event-Driven Design**: All component communication through EventBus - Test results via events
- [x] **Test-First Development**: TDD approach with failing tests written first - Core requirement for this feature
- [ ] **Process Management**: Complete lifecycle tracking and cleanup - Not applicable to testing harness
- [ ] **MCP Protocol Compliance**: JSON-RPC 2.0 and official MCP spec adherence - Not applicable to testing harness
- [x] **Performance Standards**: <5ms search, <500ms API response, <100MB memory - Correctness prioritized over performance
- [x] **Concurrency Safety**: Race condition testing required for concurrent code - Essential for TUI testing

## Project Structure

### Documentation (this feature)
```
specs/007-build-a-testing/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
test/tui/
├── integration/         # TUI integration tests
│   ├── keyboard_test.go
│   ├── view_switching_test.go
│   └── command_palette_test.go
├── harness/            # Testing harness framework
│   ├── simulator.go    # Keyboard input simulation
│   ├── validator.go    # TUI state validation
│   └── reporter.go     # Test report generation
└── fixtures/           # Test fixtures and mocks
    ├── test_projects/
    └── mock_processes/

test/scripts/
├── run_tui_tests.sh    # Test execution script
├── setup_test_env.sh   # Test environment setup
└── validate_tui.sh     # TUI validation script

# Existing Brummer structure that will be enhanced
internal/tui/
├── components/         # Existing components (enhanced for testing)
├── services/          # Existing services (enhanced for testing)
└── test_helpers.go     # New testing utilities

githooks/
├── pre-commit-tui-tests.sh  # Pre-commit hook for TUI testing
└── validate-docs.sh         # Documentation validation hook
```

**Structure Decision**: Single project structure with dedicated TUI testing package under test/tui/, integrating with existing Brummer architecture while providing comprehensive testing capabilities without compromising production functionality.

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
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Each contract → contract test task [P]
- Each entity → model creation task [P]
- Each user story → integration test task
- Implementation tasks to make tests pass

**Task Categories**:
1. **Foundation Tasks**: Core testing framework components
2. **Simulation Tasks**: Keyboard input simulation implementation
3. **Validation Tasks**: State validation and comparison logic
4. **Reporting Tasks**: Test execution reporting and metrics
5. **Integration Tasks**: Pre-commit hooks and workflow integration
6. **Documentation Tasks**: API documentation and usage guides

**Ordering Strategy**:
- TDD order: Tests before implementation
- Dependency order: Models before services before UI
- Mark [P] for parallel execution (independent files)
- Critical path: Core harness → validation → reporting → integration

**Estimated Output**: 25-30 numbered, ordered tasks in tasks.md

**Key Task Examples**:
- Task 001: Create TestHarness interface and basic structure
- Task 002: Implement InputSimulator for keyboard input generation
- Task 003: Build StateValidator for TUI state comparison
- Task 004: Develop TestReporter for comprehensive reporting
- Task 005: Create contract tests for core interfaces
- Task 006: Implement integration tests for view switching
- Task 007: Add keyboard navigation testing scenarios
- Task 008: Build command palette testing framework
- Task 009: Create pre-commit hook integration
- Task 010: Develop test execution scripts

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
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented

---
*Based on Constitution v1.0.0 - See `/memory/constitution.md`*
