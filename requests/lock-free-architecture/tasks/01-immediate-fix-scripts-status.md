# Task: Immediate Fix - scripts_status Lockup
**Generated from Master Planning**: January 31, 2025
**Context Package**: `/requests/lock-free-architecture/context/`
**Next Phase**: ProcessSnapshot pattern implementation

## Task Sizing Assessment
**File Count**: 4 files - Within target
**Estimated Time**: 30 minutes - Target: 15-30min
**Token Estimate**: ~50k tokens - Target: <150k
**Complexity Level**: 2 - Moderate (multiple patterns, some integration)
**Parallelization Benefit**: LOW - Sequential fixes required
**Atomicity Assessment**: ✅ ATOMIC - Cannot be meaningfully split further
**Boundary Analysis**: ✅ CLEAR - MODIFY/REVIEW/IGNORE zones defined

## Persona Assignment
**Persona**: Software Engineer
**Expertise Required**: Go concurrency, race condition detection
**Worktree**: Main branch (immediate fix)

## Context Summary
**Risk Level**: HIGH - Currently causing production lockups
**Integration Points**: MCP tools, TUI components
**Architecture Pattern**: Thread-safe getter methods
**Similar Reference**: Process struct already has GetStatus(), GetStartTime() methods

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /internal/mcp/tools.go                # Remaining race conditions in streaming handlers
  - /internal/tui/model.go                # Direct field access violations
  - /internal/tui/brummer_data_provider_impl.go  # Direct field access
  - /internal/tui/command_autocomplete.go  # Direct field access

direct_dependencies:
  - /internal/process/manager.go          # Process struct with thread-safe getters
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /internal/tui/script_selector.go      # May have similar issues
  - /internal/tui/restart_integration_test.go  # Test assertions
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/logs/*                      # Log store refactoring comes later
  - /internal/proxy/*                     # Proxy refactoring comes later
  - /internal/aicoder/*                   # AI coder unrelated
  - /external/*                           # External libraries
```

## Task Requirements
**Objective**: Fix all remaining race conditions causing scripts_status lockup
**Success Criteria**:
- [x] All direct field access replaced with thread-safe getters
- [ ] `go test -race` passes for affected packages
- [ ] scripts_status MCP tool no longer locks up
- [ ] Integration test added for concurrent scripts_status calls

**Validation Commands**:
```bash
# Race condition testing
go test -race ./internal/mcp/
go test -race ./internal/tui/

# Integration test
go test -v ./internal/mcp/ -run TestScriptsStatusConcurrent
```

## Risk Mitigation
**High-Risk Mitigations**:
- Missing conversions could leave race conditions - Use grep to find all instances
- TUI may have complex update patterns - Test thoroughly with concurrent operations

## Execution Notes
- Start by fixing remaining MCP tool issues
- Then fix all TUI direct field access
- Add integration test to verify no lockups
- Run race detector frequently during development