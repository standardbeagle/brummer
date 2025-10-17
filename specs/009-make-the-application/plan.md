# Implementation Plan: Bubble Tea MVU Pattern Compatibility

**Branch**: `009-make-the-application` | **Date**: 2025-01-14 | **Spec**: [/specs/009-make-the-application/spec.md](/specs/009-make-the-application/spec.md)
**Input**: Feature specification from `/specs/009-make-the-application/spec.md`

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
Transform the Brummer TUI from its current mixed synchronous/asynchronous pattern to a pure Bubble Tea MVU architecture where all I/O operations go through commands, eliminating direct goroutine usage and blocking operations in the Update() method to achieve <50ms UI response times.

## Technical Context
**Language/Version**: Go 1.21+
**Primary Dependencies**: github.com/charmbracelet/bubbletea v0.25.0, github.com/charmbracelet/bubbles
**Storage**: N/A (UI layer only)
**Testing**: Go standard testing, integration tests with TTY allocation
**Target Platform**: Cross-platform terminal (Linux, macOS, Windows)
**Project Type**: Single - Terminal UI application
**Performance Goals**: <50ms UI response time, smooth 60fps rendering
**Constraints**: No goroutines in Bubble Tea programs, all I/O through commands
**Scale/Scope**: ~100 TUI components, ~20 concurrent operations typical

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Required Gates (based on Brummer Constitution v1.0.0):**
- [x] **Lock-Free Architecture**: No mutex-based synchronization unless absolutely justified
  - Current implementation uses atomic operations and sync.Map
- [x] **Event-Driven Design**: All component communication through EventBus
  - EventBus already in use for TUI components
- [x] **Test-First Development**: TDD approach with failing tests written first
  - Will create failing tests for command pattern before implementation
- [x] **Process Management**: Complete lifecycle tracking and cleanup
  - Not directly affected by this refactoring
- [x] **MCP Protocol Compliance**: JSON-RPC 2.0 and official MCP spec adherence
  - MCP layer remains unchanged
- [x] **Performance Standards**: <5ms search, <500ms API response, <100MB memory
  - Target <50ms UI response aligns with constitution
- [x] **Concurrency Safety**: Race condition testing required for concurrent code
  - Bubble Tea's command tumbler handles concurrency safely

## Project Structure

### Documentation (this feature)
```
specs/009-make-the-application/
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
│   ├── model.go              # Main TUI model (needs refactoring)
│   ├── commands/             # Command implementations (new)
│   │   ├── io.go            # I/O commands
│   │   ├── process.go       # Process management commands
│   │   ├── network.go       # Network request commands
│   │   └── system.go        # System operation commands
│   ├── messages/            # Message types (existing)
│   ├── components/          # View components (needs updating)
│   └── services/            # Service layer (needs command pattern)
└── tests/
    └── tui/
        ├── command_test.go   # Command pattern tests
        └── response_test.go  # Response time tests
```

**Structure Decision**: Single project structure as this is a terminal UI application. The refactoring focuses on the internal/tui directory where the Bubble Tea implementation resides.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - Research Bubble Tea command patterns and best practices
   - Investigate command tumbler behavior and ordering
   - Study existing Bubble Tea applications for MVU patterns
   - Analyze current goroutine usage in TUI components

2. **Generate and dispatch research agents**:
   ```
   Task: "Research Bubble Tea command patterns for I/O operations"
   Task: "Find best practices for handling non-deterministic command results"
   Task: "Investigate strategies for context-aware operation ordering"
   Task: "Study migration patterns from goroutines to commands"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: Command pattern implementation approach
   - Rationale: Ensures non-blocking UI and proper MVU pattern
   - Alternatives considered: Direct state updates, channel-based communication

**Output**: research.md with all technical decisions resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Command: Base interface for all commands
   - Message: Event types for command results
   - State: Application state structure
   - Context: Operation context for ordering

2. **Generate API contracts** from functional requirements:
   - Command execution contract
   - Message handling contract
   - State update contract
   - Error handling contract
   - Output to `/contracts/`

3. **Generate contract tests** from contracts:
   - Response time tests (<50ms requirement)
   - Command ordering tests
   - Error retry tests
   - State consistency tests

4. **Extract test scenarios** from user stories:
   - Navigation during I/O operations
   - Rapid input handling
   - Operation cancellation
   - Error recovery with retry

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh claude`
   - Add Bubble Tea command pattern notes
   - Update recent changes section

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, CLAUDE.md updates

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate refactoring tasks for each component with goroutines
- Create command implementation tasks for each I/O type
- Add message handler tasks for command results
- Generate performance validation tasks

**Ordering Strategy**:
- TDD order: Tests before implementation
- Bottom-up: Commands before handlers before UI updates
- Mark [P] for parallel execution where possible

**Estimated Output**: 30-35 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)
**Phase 4**: Implementation (execute tasks.md following constitutional principles)
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*No violations - architecture aligns with constitutional principles*

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [x] Phase 3: Tasks generated (/tasks command)
- [x] Phase 3.1-3.5: Core implementation complete (T001-T050)
- [x] Phase 3.6: Model refactoring partially complete (T051-T053)
- [ ] Phase 3.6-3.8: Component updates remaining (T054-T080)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented (none required)

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*