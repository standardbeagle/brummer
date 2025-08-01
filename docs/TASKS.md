# Development Tasks

This document tracks development tasks, improvements, and technical debt for the Brummer project.

## Current Tasks

### High Priority

#### Fix Windows File Locking Detection
**Status**: Needs Implementation  
**Priority**: High  
**Component**: Discovery System  

The current file locking detection in `internal/discovery/diagnostics.go` uses a simplified approach on Windows that always returns `false`. This provides incomplete diagnostic information on Windows systems.

**Current Implementation**:
- Unix systems: Uses `flock()` syscall for accurate lock detection
- Windows: Returns `false` (stub implementation)

**Required Solution**:
- Implement proper Windows file locking detection using Windows APIs
- Consider using `LockFileEx`/`UnlockFileEx` Windows APIs
- Alternative: Use a cross-platform file locking library like `github.com/gofrs/flock`

**Files Affected**:
- `internal/discovery/diagnostics.go` - Main logic
- `internal/discovery/diagnostics_windows.go` - Windows-specific implementation
- `internal/discovery/diagnostics_unix.go` - Unix implementation (working)

**Acceptance Criteria**:
- Windows systems accurately detect when instance lock files are in use
- Diagnostic reports show correct lock status on all platforms
- Cross-platform build continues to work without issues
- No regression in Unix file locking behavior

---

### Medium Priority

*No medium priority tasks currently identified.*

---

### Low Priority

*No low priority tasks currently identified.*

---

## Completed Tasks

### AI Coder Constraint Documentation ✅
**Completed**: January 30, 2025  
**Component**: AI Coder PTY View  

Added comprehensive documentation for layout constraints in the AI coder PTY view system:
- Documented BORDER_AND_PADDING_WIDTH = 4 constraint
- Added ASCII diagrams showing layout structure
- Explained relationship between getTerminalSize() and border rendering
- Fixed column cutoff issue with consistent width calculations

### Cross-Platform Build Fix ✅
**Completed**: January 30, 2025  
**Component**: Discovery System  

Fixed syscall.Flock compilation errors on Windows:
- Split platform-specific code using build constraints
- Created separate Unix and Windows implementations
- Maintained existing Unix functionality while adding Windows compatibility
- Build now works across all target platforms

---

## Task Guidelines

### Adding New Tasks

When adding tasks to this file:

1. **Use clear, descriptive titles**
2. **Include status, priority, and affected component**
3. **Provide context** about the problem or improvement needed
4. **List specific files** that need changes
5. **Define acceptance criteria** for completion
6. **Move completed tasks** to the completed section with completion date

### Priority Levels

- **High**: Blocks releases, affects core functionality, or impacts user experience
- **Medium**: Important improvements that should be addressed soon
- **Low**: Nice-to-have improvements or minor technical debt

### Status Options

- **Needs Implementation**: Not started
- **In Progress**: Currently being worked on
- **Blocked**: Waiting on external dependencies or decisions
- **Ready for Review**: Implementation complete, needs review
- **Completed**: Task finished and merged

---

## Related Documentation

- [Development Roadmap](/docs/ROADMAP.md) - Long-term feature planning
- [Architecture Overview](/docs/architecture/overview.md) - System design
- [Troubleshooting Guide](/docs/troubleshooting.md) - Common issues and solutions