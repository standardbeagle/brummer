# Feature Specification: PTY Infrastructure Extraction and AI Coder Migration

**Feature ID**: 003-pty-extraction-refactor
**Status**: Not Started
**Dependency**: Blocked feature 002-the-controllers-need (AI Coder migration)
**Priority**: High
**Estimated Effort**: 6-8 hours

## Problem Statement

The AI Coder controller migration (from feature 002) cannot be completed due to a circular dependency issue:

```
components/aicoders_view.go → imports internal/tui (for PTY types)
internal/tui → imports components
= CIRCULAR DEPENDENCY ❌
```

Additionally, the PTY infrastructure needs refactoring and cleanup as it has grown organically within the `internal/tui` package.

## Current State

**PTY Code Location**: `internal/tui/` (mixed with TUI code)

**PTY-Related Files**:
- `lockfree_pty_manager.go` (318 lines) - Lock-free PTY session manager
- `lockfree_buffer.go` (278 lines) - Lock-free ring buffer for PTY messages
- `direct_pty_passthrough.go` (~200 lines) - Direct PTY passthrough
- `ai_coder_pty_view.go` (~300 lines) - PTY view rendering
- `ai_coder_controller.go` (655 lines) - AI coder controller (to be migrated)

**Total PTY Code**: ~1,751 lines scattered across `internal/tui`

## Proposed Solution

### 1. Extract PTY Infrastructure to `internal/pty` Package

**New Package Structure**:
```
internal/pty/
├── buffer.go           # Lock-free ring buffer (from lockfree_buffer.go)
├── manager.go          # PTY session manager (from lockfree_pty_manager.go)
├── passthrough.go      # Direct passthrough (from direct_pty_passthrough.go)
├── view.go             # PTY view rendering (from ai_coder_pty_view.go)
├── types.go            # Common PTY types and interfaces
├── config.go           # PTY configuration
└── README.md           # PTY architecture documentation
```

### 2. Update Import Dependencies

**Files Requiring Import Updates** (~15-20 files):
- `internal/tui/ai_coder_controller.go`
- `internal/tui/model.go`
- `internal/tui/components/aicoders_view.go` (new file, post-migration)
- `test/integration/ai_coder_test.go`
- Any other files referencing PTY types

**Import Change Pattern**:
```go
// Before
import "github.com/standardbeagle/brummer/internal/tui"
tui.LockFreePTYManager

// After
import "github.com/standardbeagle/brummer/internal/pty"
pty.Manager  // Renamed for clarity
```

### 3. Refactor and Clean Up PTY Code

**Cleanup Goals**:
- Remove duplicate code
- Improve error handling
- Add comprehensive documentation
- Simplify complex lock-free logic where possible
- Add unit tests for PTY infrastructure

### 4. Complete AI Coder Migration

Once PTY is extracted, complete the blocked tasks from feature 002:

- T074: Migrate AI coder session management to components
- T075: Migrate PTY terminal integration
- T076: Migrate slash command interception
- T077: Remove ai_coder_controller.go
- T078: Update model.go references
- T079: Verify tests pass

## Success Criteria

✅ **Architecture**:
- PTY code lives in `internal/pty` package
- No circular dependencies
- Clean separation between PTY and TUI concerns

✅ **Functionality**:
- All AI Coder features work correctly
- PTY sessions start/stop as expected
- Terminal input/output works
- Lock-free performance maintained (30-300x improvement)

✅ **Testing**:
- All existing PTY tests pass
- New unit tests for PTY package
- Integration tests for AI Coder pass
- No regressions in AI Coder functionality

✅ **Code Quality**:
- Comprehensive documentation
- Clean API surface
- Improved error handling
- Reduced code duplication

## Benefits

1. **Unblocks AI Coder Migration**: Allows completion of feature 002 to 100%
2. **Better Architecture**: Clean separation of concerns
3. **Reusability**: PTY infrastructure can be used by other features
4. **Maintainability**: Easier to understand and modify PTY code
5. **Testing**: Can unit test PTY code independently

## Risks

**High Risk Areas**:
- Lock-free buffer implementation (complex, performance-critical)
- PTY session lifecycle management
- Race conditions in concurrent PTY access
- Breaking AI Coder functionality during migration

**Mitigation**:
- Move files incrementally, testing after each change
- Comprehensive test coverage before refactoring
- Keep AI Coder functional throughout migration
- Have rollback plan ready

## Dependencies

**Requires**:
- Feature 002 (controller migration) at 85% completion ✅
- All PTY integration tests passing ✅
- Clean build ✅

**Blocks**:
- Feature 002 completion (can't reach 100% without this)
- Any future features requiring PTY infrastructure

## Implementation Plan

### Phase 1: Preparation (1 hour)
1. Create `internal/pty` package structure
2. Write PTY package documentation
3. Define clean API surface
4. Set up unit tests framework

### Phase 2: Extract Core PTY (2-3 hours)
1. Move `lockfree_buffer.go` → `internal/pty/buffer.go`
2. Move `lockfree_pty_manager.go` → `internal/pty/manager.go`
3. Move `direct_pty_passthrough.go` → `internal/pty/passthrough.go`
4. Move `ai_coder_pty_view.go` → `internal/pty/view.go`
5. Update package declarations

### Phase 3: Update Imports (2-3 hours)
1. Find all files importing PTY types
2. Update import statements
3. Fix type references
4. Update tests
5. Verify build after each file

### Phase 4: Complete AI Coder Migration (1-2 hours)
1. Migrate AI coder controller logic to component
2. Update model.go to use component
3. Remove controller file
4. Run integration tests

### Phase 5: Cleanup and Documentation (1 hour)
1. Remove old PTY files from internal/tui
2. Update architecture documentation
3. Add PTY usage examples
4. Final testing and verification

## Acceptance Criteria

- [ ] All PTY code moved to `internal/pty` package
- [ ] Zero circular dependencies
- [ ] All imports updated correctly
- [ ] Build passes with zero errors
- [ ] All tests pass (unit + integration)
- [ ] AI Coder controller migrated to component
- [ ] Performance maintained or improved
- [ ] Documentation complete

## Out of Scope

- Adding new PTY features
- Changing PTY behavior (beyond cleanup)
- Optimizing PTY performance (already optimal)
- Supporting additional terminal types

## Follow-Up Work

After this feature is complete:
- Consider extracting other TUI infrastructure to separate packages
- Add more comprehensive PTY unit tests
- Document PTY architecture patterns
- Create PTY usage guide

---

**Next Steps**: Run `/specify` on this file to generate the full implementation plan.
