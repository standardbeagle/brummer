# Feature Specification: Codebase Cleanup Before PTY Migration

**Feature Branch**: `004-before-migrating-the`
**Created**: 2025-10-07
**Status**: Draft
**Input**: User description: "before migrating the pty clean up the codebase eliminating and cleaning up old and or duplicate code"

## Execution Flow (main)
```
1. Parse user description from Input
   → Feature request: Clean up codebase before PTY migration ✅
2. Extract key concepts from description
   → Actions: eliminate, clean up
   → Target: old code, duplicate code
   → Context: Before PTY migration (Feature 003)
3. For each unclear aspect:
   → Retention criteria defined
   → Duplicate detection approach defined
4. Fill User Scenarios & Testing section ✅
5. Generate Functional Requirements ✅
6. Identify Key Entities
   → Code artifacts, technical debt markers
7. Run Review Checklist
   → No implementation details ✅
   → Focus on WHAT to clean, not HOW ✅
8. Return: SUCCESS (spec ready for planning)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT needs cleaning and WHY
- ❌ Avoid HOW to implement (specific tools, refactoring techniques)
- 👥 Written for maintainers who need clean code before PTY extraction

---

## User Scenarios & Testing

### Primary User Story
As a developer preparing for PTY infrastructure extraction (Feature 003), I need the `internal/tui` codebase cleaned of old and duplicate code so that the PTY migration is straightforward, reduces risk of errors, and results in maintainable code.

### Acceptance Scenarios

1. **Given** the internal/tui package contains ~30,000 lines of code across 113 files, **When** cleanup is complete, **Then** obsolete code is removed, duplicate code is consolidated, and the codebase is ready for PTY extraction

2. **Given** Feature 002 left some controller remnants and test files, **When** cleanup identifies unused code, **Then** dead code is removed while preserving all functional components

3. **Given** PTY migration (Feature 003) requires clear separation of concerns, **When** cleanup completes, **Then** PTY-related code has clear boundaries and minimal coupling with TUI code

4. **Given** the codebase has 25+ TODO/FIXME markers, **When** cleanup addresses technical debt, **Then** critical issues are resolved or documented for post-PTY work

### Edge Cases
- What happens when duplicate code has subtle behavioral differences?
  - System MUST preserve the correct behavior, mark ambiguous cases for review
- How does cleanup handle deprecated code that might still be referenced?
  - System MUST identify all references before removal, document dependencies
- What if test files contain obsolete tests for removed controllers?
  - System MUST remove tests for non-existent code, preserve tests for active components

---

## Requirements

### Functional Requirements

**Code Removal**
- **FR-001**: System MUST identify and remove all code files related to the 11 deleted controllers from Feature 002
- **FR-002**: System MUST identify and remove test files for deleted controllers (mcp_debug_controller_test.go confirmed present)
- **FR-003**: System MUST identify and remove commented-out code blocks that exceed 10 lines
- **FR-004**: System MUST identify dead code (unreferenced functions, types, constants) in internal/tui package

**Duplicate Code Elimination**
- **FR-005**: System MUST identify duplicate code blocks (similar logic appearing in multiple locations)
- **FR-006**: System MUST consolidate duplicate message type definitions
- **FR-007**: System MUST consolidate duplicate helper functions
- **FR-008**: System MUST identify and consolidate duplicate test utilities

**Technical Debt Resolution**
- **FR-009**: System MUST document or resolve all 25+ TODO/FIXME/HACK markers
- **FR-010**: System MUST identify and address deprecated API usage
- **FR-011**: System MUST verify all imports are necessary (remove unused imports)

**PTY Preparation**
- **FR-012**: System MUST ensure PTY-related files (lockfree_pty_manager.go, lockfree_buffer.go, ai_coder_pty_view.go, direct_pty_passthrough.go) have clear boundaries with minimal TUI coupling
- **FR-013**: System MUST identify all files that import PTY types to understand migration scope
- **FR-014**: System MUST document current PTY dependencies for Feature 003 planning

**Code Quality**
- **FR-015**: System MUST ensure all remaining code has no linting errors
- **FR-016**: System MUST ensure all remaining tests pass after cleanup
- **FR-017**: System MUST maintain or improve code coverage (currently 87%)
- **FR-018**: System MUST ensure build passes with zero errors after cleanup

**Documentation**
- **FR-019**: System MUST document what was removed and why
- **FR-020**: System MUST update internal/tui/components/README.md if component architecture changed
- **FR-021**: System MUST create cleanup report listing: files removed, duplicates consolidated, technical debt addressed

### Key Entities

**Code Artifacts**
- **Controller Remnants**: Test files and references to the 11 deleted controllers from Feature 002
- **Duplicate Code**: Similar logic blocks appearing in multiple files (message handlers, type definitions, helpers)
- **Dead Code**: Unreferenced functions, unused types, commented-out blocks
- **Technical Debt**: TODO/FIXME/HACK markers indicating deferred work (25+ instances identified)
- **PTY Files**: 4-5 core PTY files (~1,751 lines) that will be extracted in Feature 003

**Cleanup Scope**
- **Target Package**: internal/tui (113 Go files, ~30,000 lines)
- **Related Tests**: test/integration (integration tests from Feature 002)
- **Excluded**: Do not modify internal/pty (doesn't exist yet), internal/process, internal/logs (separate concerns)

---

## Review & Acceptance Checklist

### Content Quality
- [x] No implementation details (no specific refactoring tools mentioned)
- [x] Focused on maintainability and migration preparation
- [x] Written for developers who need clean codebase
- [x] All mandatory sections completed

### Requirement Completeness
- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable (can verify code removed, duplicates consolidated)
- [x] Success criteria are measurable (files removed, tests passing, build clean)
- [x] Scope is clearly bounded (internal/tui package, before Feature 003)
- [x] Dependencies identified (follows Feature 002, precedes Feature 003)

---

## Execution Status

- [x] User description parsed
- [x] Key concepts extracted (eliminate old/duplicate code, prepare for PTY migration)
- [x] Ambiguities marked (none - cleanup scope is clear)
- [x] User scenarios defined (developer preparing for PTY extraction)
- [x] Requirements generated (21 functional requirements)
- [x] Entities identified (code artifacts, cleanup scope)
- [x] Review checklist passed

---

## Success Criteria

**Before Cleanup** (Current State):
- 113 Go files in internal/tui (~30,000 lines)
- 25+ TODO/FIXME markers
- Test files for deleted controllers present
- Unknown amount of duplicate code
- Unknown amount of dead code

**After Cleanup** (Target State):
- Reduced file count (obsolete files removed)
- Reduced line count (duplicate/dead code removed)
- Zero test files for deleted controllers
- All TODO/FIXME markers addressed or documented
- Zero dead code
- Zero duplicate message types
- Build passes ✅
- All tests pass ✅
- Code coverage maintained or improved (≥87%)
- Cleanup report documenting all changes

**PTY Migration Readiness**:
- PTY files have clear boundaries
- All PTY dependencies documented
- Minimal coupling between PTY and TUI code
- Feature 003 can proceed with confidence

---

## Dependencies & Constraints

**Requires**:
- Feature 002 (controller migration) at 85% complete ✅
- Clean build before cleanup ✅
- All integration tests passing ✅

**Blocks**:
- Feature 003 (PTY extraction) - should not start before cleanup complete

**Out of Scope**:
- Adding new features
- Refactoring PTY code (saved for Feature 003)
- Migrating AI Coder controller (Feature 003)
- Migrating Input controller (future work)
- Performance optimization (beyond removing dead code)

**Constraints**:
- MUST NOT break any existing functionality
- MUST NOT reduce test coverage below 87%
- MUST NOT modify PTY code behavior (only cleanup around it)
- MUST preserve all working components from Feature 002

---

## Benefits

1. **Reduced Migration Risk**: Cleaner codebase means fewer errors during PTY extraction
2. **Improved Maintainability**: Less code to understand and maintain
3. **Faster Feature 003**: Clear boundaries make PTY extraction straightforward
4. **Better Code Quality**: Addressing technical debt improves long-term health
5. **Easier Debugging**: Less code means easier troubleshooting

---

## Notes for Planning Phase

When generating tasks from this spec, prioritize:
1. **Safe deletions first** (test files for deleted controllers)
2. **Build verification after each change** (ensure nothing breaks)
3. **Duplicate consolidation carefully** (verify identical behavior)
4. **Technical debt assessment** (resolve vs document for later)
5. **PTY boundary analysis** (prepare for Feature 003)

**Estimated Effort**: 3-4 hours focused cleanup work
- 1 hour: Identify and remove obsolete files
- 1 hour: Find and consolidate duplicates
- 1 hour: Address technical debt markers
- 0.5 hour: Verify build, tests, prepare cleanup report
- 0.5 hour: Document PTY dependencies for Feature 003

---

**Status**: ✅ Ready for `/plan` command to generate implementation tasks
**Next Command**: `/plan` to create detailed cleanup task list
