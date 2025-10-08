# Research: Codebase Cleanup Analysis

**Feature**: 004-before-migrating-the
**Date**: 2025-10-07
**Purpose**: Document findings from codebase analysis to guide cleanup tasks

## Executive Summary

Analysis of the `internal/tui` package (113 files, ~30,000 lines) identified:
- **1 confirmed obsolete test file**: `mcp_debug_controller_test.go` (7,723 bytes)
- **25+ technical debt markers**: TODO/FIXME/HACK comments requiring categorization
- **4 PTY files**: ~1,751 lines requiring dependency analysis for Feature 003
- **Unknown duplicates**: Requires static analysis
- **Unknown dead code**: Requires static analysis

## Research Findings

### 1. Obsolete Test Files from Feature 002

**Controllers Deleted in Feature 002** (commit 9a730ab):
1. event_controller.go → Deleted ✅
2. layout_controller.go → Deleted ✅
3. urls_view_controller.go → Deleted ✅
4. mcp_debug_controller.go → Deleted ✅
5. command_window_controller.go → Deleted ✅
6. process_view_controller.go → Deleted ✅
7. logs_view_controller.go → Deleted ✅
8. errors_view_controller.go → Deleted ✅
9. web_view_controller.go → Deleted ✅
10. settings_controller.go → Deleted ✅
11. script_selector_controller.go → Deleted ✅

**Test Files Still Present**:
- ✅ `mcp_debug_controller_test.go` (7,723 bytes) - **CONFIRMED FOR DELETION**
- ❓ `event_controller_test.go` - Need to verify
- ❓ `layout_controller_test.go` - Need to verify

**Controllers Remaining** (NOT for deletion):
- `ai_coder_controller.go` (20,897 bytes) - Deferred to Feature 003
- `ai_coder_controller_test.go` (7,303 bytes) - Keep
- `input_controller.go` (15,071 bytes) - Deferred to future
- `input_controller_test.go` (7,314 bytes) - Keep

**Decision**: Remove test files for deleted controllers only. Verify which test files exist during task execution.

---

### 2. Technical Debt Markers

**Count**: 25+ TODO/FIXME/HACK markers identified

**Sample Findings**:

**ai_coder_controller.go** (5 TODOs):
```go
Line 118: // TODO: Convert aicoder events to Brummer events when event integration is ready
Line 122: // TODO: Convert aicoder events to Brummer events when event integration is ready
Line 126: // TODO: Convert aicoder events to Brummer events when event integration is ready
Line 236: // TODO: Implement PTY event subscription when available
Line 257: // TODO: Implement SetModel method on AICoderDebugForwarder when needed
```
**Category**: Feature 003 (PTY-related)

**model.go** (2 TODOs):
```go
Line 763: // TODO: Fix mcpActivityMsg handling - needs proper sessionId and activity data
Line 1350: // TODO: Re-implement unread indicators when UpdateErrorsList returns count
```
**Category**: Future work

**input_controller.go** (1 TODO):
```go
Line 227: // TODO: Implement system panel toggle when TabsComponent has the appropriate methods
```
**Category**: Future work

**components/command_executor.go** (7 TODOs):
```go
Line 179: // TODO: Store show pattern in model state for log filtering
Line 191: // TODO: Store hide pattern in model state for log filtering
Line 210: // TODO: Implement per-process clearing using filters
Line 228: // TODO: Get proxy list from proxy server
Line 243: // TODO: Implement proxy mode toggle
Line 326: // TODO: Show integrated terminal
```
**Category**: Future work (component features)

**constants.go** (1 TODO):
```go
Line 30: // TODO: Replace with lock-free ring buffer (see race analysis)
```
**Category**: Performance optimization (immediate?)

**Categorization Plan**:
1. **Immediate** (resolve now): Performance-critical TODOs, simple fixes
2. **Feature 003** (document for PTY migration): PTY-related TODOs in ai_coder_controller.go
3. **Future** (document for later): Component features, UI improvements

---

### 3. Duplicate Code Patterns

**Requires Static Analysis** during task execution.

**Expected Duplicates**:
- Message type definitions across files
- Helper functions (log filtering, string manipulation)
- Test utilities in multiple test files

**Analysis Method**:
1. Extract code blocks >20 lines
2. Compare using AST similarity (not just text)
3. Flag blocks with >80% similarity
4. Manual review to confirm identical behavior

**Alternatives Considered**:
- Text-based duplicate detection (rejected - too many false positives)
- Manual inspection only (rejected - too error-prone)
- **Chosen**: AST-based detection with manual verification

---

### 4. Dead Code Detection

**Requires Static Analysis** during task execution.

**Detection Methods**:
1. `go vet` for unreferenced symbols (basic)
2. `golangci-lint unused` for comprehensive analysis
3. Manual review of suspicious patterns

**Expected Dead Code**:
- Unreferenced helper functions from deleted controllers
- Unused type definitions for deleted features
- Commented-out code blocks (visual inspection required)

**Alternatives Considered**:
- Third-party tools (deadcode, unused) - considered but golangci-lint is already in project
- Manual-only review (rejected - too error-prone)
- **Chosen**: golangci-lint + manual verification

---

### 5. PTY Dependency Analysis

**PTY Files** (to be extracted in Feature 003):
1. `lockfree_pty_manager.go` (318 lines) - PTY session manager
2. `lockfree_buffer.go` (278 lines) - Lock-free ring buffer
3. `ai_coder_pty_view.go` (~300 lines) - PTY view rendering
4. `direct_pty_passthrough.go` (~200 lines) - Direct PTY passthrough

**Total**: ~1,751 lines of PTY code

**Dependency Analysis Needed**:
- Find all files importing PTY types
- Document coupling level (tight vs. loose)
- Identify refactoring needed for Feature 003

**Analysis Method**:
```bash
grep -r "lockfree_pty_manager\|lockfree_buffer\|ai_coder_pty_view\|direct_pty_passthrough" internal/tui --include="*.go" -l
```

**Expected Importers**:
- `ai_coder_controller.go` (tight coupling)
- `model.go` (message handling)
- Test files for AI coder

**Output**: PTY dependency map document for Feature 003 planning

---

## Decisions Summary

| Research Question | Decision | Rationale |
|-------------------|----------|-----------|
| What test files to delete? | Remove tests for 11 deleted controllers | Controllers deleted in Feature 002 |
| How to handle TODOs? | Categorize: immediate/Feature-003/future | Prioritize by impact and relevance |
| How to find duplicates? | AST-based analysis + manual review | Accurate detection of semantic duplicates |
| How to find dead code? | golangci-lint unused + manual review | Comprehensive and reliable |
| PTY dependency analysis? | grep + manual documentation | Simple and effective for migration planning |

---

## Risks & Mitigations

**Risk 1: Removing code that's actually used**
- **Mitigation**: Run full test suite after every deletion
- **Mitigation**: Use static analysis to verify no references

**Risk 2: Breaking test coverage**
- **Mitigation**: Measure coverage before/after cleanup
- **Mitigation**: Ensure coverage stays ≥87%

**Risk 3: Duplicate consolidation changes behavior**
- **Mitigation**: Manual review of each duplicate pair
- **Mitigation**: Run tests after consolidation

**Risk 4: Missing important TODOs**
- **Mitigation**: Document all TODOs before resolving
- **Mitigation**: Create issues for deferred work

---

## Tools & Commands

**Static Analysis**:
```bash
# Find unreferenced code
golangci-lint run --disable-all --enable=unused internal/tui/...

# Find duplicates (manual with dupl tool if available)
dupl -t 20 internal/tui/*.go

# Find TODOs
grep -r "TODO\|FIXME\|HACK" internal/tui --include="*.go" -n
```

**Build Verification**:
```bash
# Build
go build ./cmd/brum

# Unit tests
go test ./internal/tui/...

# Integration tests
go test -tags=integration ./test/integration/...

# Coverage
go test -cover ./internal/tui/...

# Race detection
make test-race
```

---

## Next Steps

1. ✅ Research complete - documented findings above
2. ⏭️ Generate tasks from cleanup categories
3. ⏭️ Execute cleanup tasks sequentially
4. ⏭️ Verify build and tests after each change
5. ⏭️ Complete cleanup-report.md with final metrics

---

**Status**: ✅ Research complete, ready for task generation
