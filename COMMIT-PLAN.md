# Atomic Commit Plan for Component System Changes

## Overview
66 modified files + 26 new files = 92 total changes
These need to be organized into logical, atomic commits that each serve a single purpose.

## Commit Sequence

### 1. Documentation Update - Current State Clarification
**Purpose**: Update CLAUDE.md to accurately reflect system state
**Files**:
- `CLAUDE.md` - Update current state section

**Commit Message**: 
```
docs: clarify component system is not enabled by default

- Update CLAUDE.md to reflect actual system state
- Component system requires BRUMMER_USE_COMPONENTS=true
- Monolithic TUI is still the default
```

### 2. Component Test Infrastructure
**Purpose**: Add comprehensive test harness for component system
**New Files**:
- `internal/tui/components/test_harness.go`
- `internal/tui/components/test_constants.go`
- `internal/tui/components/comprehensive_test.go`
- `internal/tui/components/simple_stress_test.go`
- `internal/tui/components/stress_test.go`

**Commit Message**:
```
test(components): add comprehensive test infrastructure

- Add test harness for component testing
- Add stress tests for race condition detection
- Add test constants for consistent testing
- Validates component message handling
```

### 3. Component Log System Foundation
**Purpose**: Add log subscription and debugging infrastructure
**New Files**:
- `internal/tui/components/log_subscription.go`
- `internal/tui/components/debug.go`
- `internal/tui/components/lifecycle.go`

**Modified Files**:
- `internal/tui/components/component.go`
- `internal/tui/components/messages.go`
- `internal/tui/components/custom_messages.go`

**Commit Message**:
```
feat(components): add log subscription system

- Implement async log subscription with debouncing
- Add debug utilities for component development
- Add lifecycle management for proper cleanup
- Enable push-based log updates via messages
```

### 4. Fix Initial Log Loading Issue
**Purpose**: Fix components not showing existing logs
**Modified Files**:
- `internal/tui/components/integrated_shell.go`
- `internal/tui/components/logs_view_component.go` (if exists)

**New Test Files**:
- `internal/tui/components/log_flow_test.go`
- `internal/tui/components/log_integration_test.go`
- `internal/tui/components/logs_debug_test.go`
- `internal/tui/components/logs_runtime_test.go`
- `internal/tui/components/process_logs_integration.go`

**Commit Message**:
```
fix(components): load existing logs on component initialization

- Components now load existing logs in Init()
- Fix "No logs available" issue when logs exist
- Add comprehensive tests for log display
- Ensure push model works with existing data
```

### 5. Component View Adapters
**Purpose**: Bridge between old controllers and new components
**Modified Files**:
- `internal/tui/components/view_adapter.go`
- `internal/tui/components/logs_view_adapter.go`
- `internal/tui/components/process_view_adapter.go`
- `internal/tui/components/errors_view_adapter.go`
- `internal/tui/components/urls_view_adapter.go`
- `internal/tui/components/web_view_adapter.go`
- `internal/tui/components/settings_view_adapter.go`
- `internal/tui/components/ai_coders_view_adapter.go`
- `internal/tui/components/command_input_adapter.go`
- `internal/tui/components/script_selector_adapter.go`

**Commit Message**:
```
refactor(components): update view adapters for migration

- Update all view adapters to support component system
- Maintain backward compatibility with controllers
- Enable gradual migration path
```

### 6. Component Layout System
**Purpose**: Flexible layout management for components
**Modified Files**:
- `internal/tui/components/layout_component.go`
- `internal/tui/components/box_layout.go`
- `internal/tui/components/stack_layout.go`
- `internal/tui/components/border_window.go`
- `internal/tui/components/window.go`
- `internal/tui/components/width_helpers.go`

**Commit Message**:
```
feat(components): enhance layout system

- Add flexible box and stack layouts
- Fix border sizing issues with BorderWindow
- Add width calculation helpers
- Support responsive component layouts
```

### 7. Shell Components Enhancement
**Purpose**: Improve shell components with proper event handling
**Modified Files**:
- `internal/tui/components/minimal_shell.go`
- `internal/tui/components/minimal_shell_v2.go`
- `internal/tui/components/enhanced_shell.go`
- `internal/tui/components/safe_shell.go`

**Commit Message**:
```
feat(components): enhance shell components

- Add tab navigation support
- Improve notification handling
- Add safe shell variant for testing
- Support multiple shell configurations
```

### 8. Process View Component Updates
**Purpose**: Update process view for component system
**Modified Files**:
- `internal/tui/components/process_view_new.go`
- `internal/tui/components/bubbles_integration.go`

**Commit Message**:
```
feat(components): update process view component

- Ensure initial process list loads on Init()
- Add proper Bubble Tea integration
- Support process selection and updates
```

### 9. Component System Integration
**Purpose**: Integrate components with main model
**Modified Files**:
- `internal/tui/model_with_components.go`
- `internal/tui/model_component_adapter.go`
- `internal/tui/model_refactored.go`
- `internal/tui/components/model_integration.go`

**Commit Message**:
```
feat(tui): integrate component system with main model

- Add model wrapper for component support
- Enable BRUMMER_USE_COMPONENTS flag
- Maintain backward compatibility
```

### 10. Component Examples and UI Builder
**Purpose**: Add examples and builder patterns
**Modified Files**:
- `internal/tui/components/example_app.go`
- `internal/tui/components/migration_example.go`
- `internal/tui/components/notification_example.go`
- `internal/tui/components/ui_builder.go`

**Commit Message**:
```
docs(components): add examples and UI builder

- Add example applications for reference
- Add migration examples for developers
- Add UI builder for rapid prototyping
```

### 11. Notification and System Components
**Purpose**: Add notification and system panel components
**Modified Files**:
- `internal/tui/components/notification_component.go`
- `internal/tui/components/system_panel.go`
- `internal/tui/components/command_palette.go`
- `internal/tui/components/view_switcher.go`
- `internal/tui/components/header.go`
- `internal/tui/components/focus_chain.go`

**Commit Message**:
```
feat(components): add notification and system components

- Add unified notification component
- Add system panel for status display
- Add command palette for slash commands
- Add view switcher and header components
```

### 12. AI Coder Integration Updates
**Purpose**: Update AI coder for component system
**Modified Files**:
- `internal/tui/ai_coder_controller.go`
- `internal/tui/ai_coder_pty_view.go`
- `internal/tui/ai_coder_pty_view_test.go`
- `internal/aicoder/pty_session.go`
- `internal/aicoder/pty_performance_test.go`
- `internal/aicoder/terminal_unix.go`
- `internal/aicoder/terminal_windows.go`

**Commit Message**:
```
feat(aicoder): update AI coder for component system

- Update PTY session handling
- Improve terminal compatibility
- Add performance optimizations
- Support component-based rendering
```

### 13. Lock-Free Architecture Updates
**Purpose**: Optimize with lock-free patterns
**Modified Files**:
- `internal/tui/lockfree_buffer.go`
- `internal/tui/lockfree_config.go`
- `internal/tui/lockfree_dashboard.go`
- `internal/tui/lockfree_pty_manager.go`
- `internal/tui/lockfree_test.go`
- `internal/tui/direct_pty_passthrough.go`

**Commit Message**:
```
perf(tui): implement lock-free architecture

- Use atomic operations for shared state
- Reduce mutex contention
- Improve concurrent performance
- Add lock-free buffer management
```

### 14. Controller Updates
**Purpose**: Update controllers for component compatibility
**Modified Files**:
- `internal/tui/logs_view_controller.go`
- `internal/tui/process_view_controller.go`
- `internal/tui/errors_view_controller.go`
- `internal/tui/urls_view_controller.go`
- `internal/tui/web_view_controller.go`
- `internal/tui/settings_controller.go`

**Commit Message**:
```
refactor(controllers): update for component compatibility

- Maintain controller interface for gradual migration
- Add hooks for component integration
- Support both old and new architectures
```

### 15. Core TUI Model Updates
**Purpose**: Update core model for dual-mode operation
**Modified Files**:
- `internal/tui/model.go`
- `internal/tui/constants.go`

**Commit Message**:
```
feat(tui): support dual-mode operation

- Support both monolithic and component modes
- Check BRUMMER_USE_COMPONENTS environment variable
- Maintain full backward compatibility
```

### 16. Supporting System Updates
**Purpose**: Update supporting systems
**Modified Files**:
- `internal/notifications/publisher.go`
- `internal/mcp/tools.go`
- `internal/tui/commands/slash_handlers.go`

**Commit Message**:
```
feat: update supporting systems for components

- Update notification publisher for components
- Update MCP tools for new architecture
- Update slash handlers for component routing
```

### 17. Test Application Updates
**Purpose**: Update test applications
**Modified Files**:
- `cmd/test-integrated/main.go`
- `cmd/test-render/main.go`

**Commit Message**:
```
test: update test applications for components

- Add component system testing
- Update integrated test harness
- Add rendering tests for components
```

### 18. Component System Documentation
**Purpose**: Add comprehensive documentation
**New Files**:
- `COMPONENT_MIGRATION_PLAN.md`
- `docs/COMPONENT-SYSTEM-RESEARCH.md`
- `docs/COMPONENT-SYSTEM-SETUP.md`
- `docs/component-contracts.md`
- `docs/component-log-fix.md`
- `docs/architecture/data-flow.md`

**Commit Message**:
```
docs: add component system documentation

- Add migration plan and timeline
- Add component research and design docs
- Add setup and contract documentation
- Document log fix and data flow
```

### 19. Test Files for Verification
**Purpose**: Add verification tests
**New Files**:
- `test_component_output.go`
- `test_fix_verification.go`
- `test_subscription.go`
- `test_subscription_proper.go`
- `package.json` (if needed for testing)

**Commit Message**:
```
test: add component verification tests

- Add tests to verify log fix
- Add subscription system tests
- Verify component output handling
```

### 20. Build Test Updates
**Purpose**: Update build and benchmark tests
**Modified Files**:
- `internal/tui/components/build_test.go`
- `internal/tui/components/benchmark_simple_test.go`

**Commit Message**:
```
test(components): update build and benchmark tests

- Update build verification tests
- Add performance benchmarks
- Ensure components build correctly
```

## Execution Commands

```bash
# Start with a clean working directory
git status

# Commit 1: Documentation update
git add CLAUDE.md
git commit -m "docs: clarify component system is not enabled by default"

# Commit 2: Test infrastructure
git add internal/tui/components/test_harness.go \
        internal/tui/components/test_constants.go \
        internal/tui/components/comprehensive_test.go \
        internal/tui/components/simple_stress_test.go \
        internal/tui/components/stress_test.go
git commit -m "test(components): add comprehensive test infrastructure"

# Continue with remaining commits...
# Each following the pattern above
```

## Verification After Commits

```bash
# Verify no changes remain
git status

# Run tests to ensure everything works
make test-race
go test -v ./internal/tui/components/...

# Test component mode
BRUMMER_USE_COMPONENTS=true ./brum
```

## Notes

- Each commit should be atomic and focused on a single purpose
- Run tests after each commit to ensure nothing breaks
- The order matters - infrastructure before features
- Documentation commits can be reordered if needed
- Consider squashing related commits if they're too granular