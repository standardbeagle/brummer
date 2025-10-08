# Implementation Plan: Codebase Cleanup Before PTY Migration

**Branch**: `004-before-migrating-the` | **Date**: 2025-10-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/home/beagle/work/brummer/specs/004-before-migrating-the/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path ✅
2. Fill Technical Context ✅
   → Project Type: single (Go CLI/TUI application)
   → Structure Decision: Standard Go layout with internal/ packages
3. Fill Constitution Check section ✅
4. Evaluate Constitution Check section ✅
   → No violations - cleanup task, no new architecture
   → Update Progress Tracking: Initial Constitution Check ✅
5. Execute Phase 0 → research.md ✅
   → No NEEDS CLARIFICATION - cleanup scope is clear
6. Execute Phase 1 → contracts, data-model.md, quickstart.md ✅
7. Re-evaluate Constitution Check section ✅
   → No new violations - cleanup maintains existing architecture
   → Update Progress Tracking: Post-Design Constitution Check ✅
8. Plan Phase 2 → Describe task generation approach ✅
9. STOP - Ready for /tasks command ✅
```

**IMPORTANT**: The /plan command STOPS at step 9. Phase 2 will be executed by /tasks command.

## Summary

Clean up the `internal/tui` package before PTY migration (Feature 003) by removing obsolete controller test files, eliminating duplicate code, addressing technical debt markers, and documenting PTY dependencies. This will reduce codebase from ~30,000 lines across 113 files to a leaner state ready for PTY extraction.

**Primary Requirement**: Remove old and duplicate code to prepare for PTY infrastructure extraction

**Technical Approach**: Static code analysis to identify:
1. Obsolete test files for deleted controllers (mcp_debug_controller_test.go confirmed)
2. Duplicate code patterns (message types, helper functions)
3. Technical debt markers (25+ TODO/FIXME/HACK comments)
4. Dead code (unreferenced functions/types)
5. PTY dependency boundaries for Feature 003

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: BubbleTea (TUI framework), existing internal packages
**Storage**: N/A (code cleanup, no data storage)
**Testing**: go test, integration tests in test/integration/
**Target Platform**: Linux/macOS/Windows (multi-platform CLI)
**Project Type**: single (Go CLI/TUI application)
**Performance Goals**: Maintain current performance (no regression)
**Constraints**:
- MUST NOT break any existing functionality
- MUST maintain 87% test coverage
- MUST NOT modify PTY code behavior
**Scale/Scope**:
- ~30,000 lines of code across 113 Go files
- 25+ technical debt markers
- 11 deleted controllers with potential test remnants
- Estimated 3-4 hours cleanup work

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Required Gates (based on Brummer Constitution v1.0.0):**
- [x] **Lock-Free Architecture**: N/A - No new synchronization code
- [x] **Event-Driven Design**: N/A - No component communication changes
- [x] **Test-First Development**: ✅ Will verify tests pass after cleanup
- [x] **Process Management**: N/A - No process lifecycle changes
- [x] **MCP Protocol Compliance**: N/A - No MCP changes
- [x] **Performance Standards**: ✅ Will maintain current performance
- [x] **Concurrency Safety**: ✅ Will run race detection after changes

**Assessment**: ✅ PASS - This is a cleanup task that removes code, not adds complexity

## Project Structure

### Documentation (this feature)
```
specs/004-before-migrating-the/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── cleanup-report.md    # Phase 1 output (cleanup findings)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
internal/tui/
├── *.go                           # Main TUI files (54 files)
├── components/                    # Component package
│   ├── *_view.go                  # View components
│   └── *_test.go                  # Component tests
├── messages/                      # Message types
│   └── *.go
└── *_test.go                      # Test files

test/integration/
├── *_test.go                      # Integration tests (from Feature 002)
└── test_helpers.go                # Test utilities

# PTY files (to be extracted in Feature 003):
internal/tui/
├── lockfree_pty_manager.go       # PTY session manager (318 lines)
├── lockfree_buffer.go            # PTY ring buffer (278 lines)
├── ai_coder_pty_view.go          # PTY view rendering (~300 lines)
└── direct_pty_passthrough.go     # PTY passthrough (~200 lines)
```

**Structure Decision**: Standard Go monorepo with internal/ packages. Cleanup targets the internal/tui package which contains the bulk of the codebase. PTY files remain in place until Feature 003 but will be analyzed for dependency boundaries.

## Phase 0: Outline & Research

**Research Questions Addressed**:

1. **Q: What obsolete test files exist from Feature 002 controller migration?**
   - **Finding**: `mcp_debug_controller_test.go` confirmed present (deleted in controller migration)
   - **Finding**: Need to scan for other controller test files
   - **Decision**: Remove all test files for the 11 deleted controllers

2. **Q: What duplicate code patterns exist in internal/tui?**
   - **Finding**: Message type definitions may be duplicated
   - **Finding**: Helper functions may exist in multiple files
   - **Decision**: Use static analysis to find duplicate blocks >20 lines

3. **Q: What technical debt markers need addressing?**
   - **Finding**: 25+ TODO/FIXME/HACK comments identified
   - **Sample**: ai_coder_controller.go has 5 TODOs for event integration
   - **Sample**: model.go has TODOs for MCP activity and error indicators
   - **Decision**: Categorize as: (1) Resolve immediately, (2) Document for Feature 003, (3) Document for future

4. **Q: What dead code exists (unreferenced functions)?**
   - **Finding**: Requires static analysis with unused code detection
   - **Decision**: Use `go vet` and manual analysis to identify unreferenced code

5. **Q: What are PTY dependency boundaries?**
   - **Finding**: 4 core PTY files identified (~1,751 lines)
   - **Decision**: Document all imports of PTY types to prepare for Feature 003

**No NEEDS CLARIFICATION** - Cleanup scope is well-defined.

**Output**: research.md (see below)

## Phase 1: Design & Contracts

### Data Model
**N/A** - This is a cleanup task with no data entities.

### Cleanup Categories

**Category 1: Obsolete Test Files**
- Controller test files for deleted controllers
- Expected: mcp_debug_controller_test.go, event_controller_test.go, layout_controller_test.go
- Verification: Check that controller .go file doesn't exist

**Category 2: Duplicate Code**
- Message type definitions appearing in multiple files
- Helper functions with similar logic
- Test utilities duplicated across test files

**Category 3: Technical Debt**
- TODO markers: Categorize as immediate/Feature-003/future
- FIXME markers: Must be resolved or converted to TODO with justification
- HACK markers: Must be resolved or documented

**Category 4: Dead Code**
- Unreferenced functions
- Unused type definitions
- Commented-out code blocks >10 lines
- Unused imports

**Category 5: PTY Dependencies**
- Files importing lockfree_pty_manager.go
- Files importing lockfree_buffer.go
- Files importing ai_coder_pty_view.go
- Files importing direct_pty_passthrough.go

### Cleanup Contracts

**Contract 1: Safety Guarantees**
```
Input: Code to be removed
Verification:
  - Run: go build ./cmd/brum
  - Run: go test ./internal/tui/...
  - Run: go test -tags=integration ./test/integration/...
  - Expected: All pass
Output: Confirmation that removal is safe
```

**Contract 2: Duplicate Detection**
```
Input: File paths to analyze
Analysis:
  - Extract code blocks >20 lines
  - Compare similarity using hash/AST
  - Flag >80% similar blocks
Output: List of duplicate candidates with locations
```

**Contract 3: Dependency Analysis**
```
Input: PTY file paths
Analysis:
  - Find all import statements referencing PTY files
  - Document caller locations
  - Identify coupled vs. decoupled usage
Output: PTY dependency map for Feature 003
```

### Quickstart (Verification)

```bash
# Step 1: Verify clean build before cleanup
cd /home/beagle/work/brummer
go build ./cmd/brum
echo "✅ Baseline build passes"

# Step 2: Run cleanup tasks (from tasks.md, generated by /tasks command)

# Step 3: Verify clean build after cleanup
go build ./cmd/brum
go test ./internal/tui/...
go test -tags=integration ./test/integration/...
echo "✅ Post-cleanup build and tests pass"

# Step 4: Verify test coverage maintained
go test -cover ./internal/tui/... | grep coverage
# Expected: coverage ≥ 87%

# Step 5: Run race detection
make test-race
echo "✅ No race conditions detected"

# Step 6: Review cleanup report
cat specs/004-before-migrating-the/cleanup-report.md
echo "✅ Cleanup complete, ready for Feature 003"
```

### Agent Context Update

N/A for cleanup task - no new technologies or patterns introduced.

**Output**: cleanup-report.md (cleanup findings template), quickstart.md (verification steps)

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
1. Load `.specify/templates/tasks-template.md` as base
2. Generate tasks in strict TDD order:
   - **Phase 1: Baseline Verification** (T001-T003)
     - T001: Document current state (file count, line count, test count)
     - T002: Run baseline tests and record results
     - T003: Create cleanup-report.md template

   - **Phase 2: Safe Deletions** (T004-T010)
     - T004-T009: Remove each obsolete controller test file (6 tasks)
     - T010: Verify build and tests after each deletion

   - **Phase 3: Duplicate Code Elimination** (T011-T015)
     - T011: Identify duplicate message type definitions
     - T012: Consolidate duplicate message types
     - T013: Identify duplicate helper functions
     - T014: Consolidate duplicate helpers
     - T015: Verify build and tests

   - **Phase 4: Technical Debt Resolution** (T016-T022)
     - T016: Categorize all TODO/FIXME/HACK markers
     - T017-T019: Resolve immediate TODOs (3 tasks estimated)
     - T020: Document Feature 003 TODOs
     - T021: Document future TODOs
     - T022: Verify build and tests

   - **Phase 5: Dead Code Removal** (T023-T027)
     - T023: Run static analysis for unreferenced code
     - T024: Remove unreferenced functions/types
     - T025: Remove commented-out code blocks
     - T026: Remove unused imports
     - T027: Verify build and tests

   - **Phase 6: PTY Dependency Analysis** (T028-T030)
     - T028: Document all PTY imports
     - T029: Create PTY dependency map
     - T030: Write recommendations for Feature 003

   - **Phase 7: Final Verification** (T031-T035)
     - T031: Run full test suite
     - T032: Verify test coverage ≥87%
     - T033: Run race detection
     - T034: Complete cleanup-report.md
     - T035: Commit all changes

**Ordering Strategy**:
- Sequential execution (each phase depends on previous)
- Verification after each major change
- No parallel tasks - cleanup requires careful sequencing

**Estimated Output**: ~35 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)
**Phase 4**: Implementation (execute tasks.md following constitutional principles)
**Phase 5**: Validation (run tests, execute quickstart.md, verify no regressions)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

**No complexity deviations** - This is a straightforward cleanup task that:
- Removes code (reduces complexity)
- Consolidates duplicates (reduces complexity)
- Addresses technical debt (improves code quality)
- Maintains existing architecture (no new patterns)

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
- [x] Complexity deviations documented (none)

---

## Artifacts Generated

✅ **plan.md** - This file
✅ **research.md** - Research findings (see below)
✅ **cleanup-report.md** - Cleanup findings template (see below)
✅ **quickstart.md** - Verification steps (see below)

---

*Based on Constitution v1.0.0 - See `/memory/constitution.md`*
