# Goroutine Migration Inventory
# Generated: 2025-01-14
# Purpose: Track all goroutine usage for migration to Bubble Tea commands

## Critical Goroutine Usage (Production Code - Must Migrate)

### Components
1. **internal/tui/components/processes_view.go:195**
   - Purpose: Background process monitoring
   - Migration: Convert to ProcessCommand with tea.Cmd

2. **internal/tui/components/logs_view.go**
   - Purpose: Real-time log streaming
   - Migration: Convert to IOCommand for log fetching

3. **internal/tui/components/urls_view.go**
   - Purpose: Background URL discovery and fetching
   - Migration: Convert to NetworkCommand for URL operations

4. **internal/tui/components/errors_view.go**
   - Purpose: Error aggregation and monitoring
   - Migration: Convert to SystemCommand for error collection

5. **internal/tui/components/mcp_view.go** (if exists)
   - Purpose: MCP client network operations
   - Migration: Convert to NetworkCommand for MCP calls

6. **internal/tui/components/settings_view.go**
   - Purpose: Settings I/O operations
   - Migration: Convert to IOCommand for settings persistence

### Services
7. **internal/tui/services/process_service.go** (if exists)
   - Purpose: Process lifecycle management
   - Migration: Command pattern for process operations

8. **internal/tui/services/log_service.go** (if exists)
   - Purpose: Log streaming and filtering
   - Migration: IOCommand for log operations

9. **internal/tui/services/mcp_service.go** (if exists)
   - Purpose: MCP protocol handling
   - Migration: NetworkCommand for MCP communication

### System Services
10. **internal/tui/services/notifications_service.go:97, 325**
    - Purpose: Background notification cleanup
    - Migration: SystemCommand with tea.Tick

11. **internal/tui/services/input_service.go:409, 425**
    - Purpose: Background input processing
    - Migration: IOCommand for input handling

12. **internal/tui/services/pty_navigation_adapter.go:213, 218, 246**
    - Purpose: PTY navigation and monitoring
    - Migration: SystemCommand for PTY operations

### PTY/Terminal Operations
13. **internal/tui/pty_integration.go:281, 303, 380, 549, 560**
    - Purpose: PTY event handling and monitoring
    - Migration: SystemCommand for PTY events

14. **internal/tui/pty_main_wrapper.go:438, 574**
    - Purpose: PTY wrapper operations
    - Migration: SystemCommand for PTY lifecycle

15. **internal/tui/lockfree_pty_manager.go:118, 237**
    - Purpose: PTY session monitoring
    - Migration: SystemCommand for session management

## Helper Files (To Be Removed)
- **internal/tui/goroutine_helper.go:10, 29**
- **internal/tui/commands/goroutine_helper.go:10, 29**
- **internal/tui/direct_pty_passthrough.go:71**
- **internal/tui/pty_termination.go:223, 398**
- **internal/tui/services/pty_input_adapter.go:152, 157, 183, 389**
- **internal/tui/services/aicoder_service.go:183, 500**
- **internal/tui/services/file_browser_service.go:355**

## Test Files (Can Keep for Testing)
- All *_test.go files with goroutines
- Test-specific goroutines are acceptable

## Migration Priority
1. **P0**: UI Components (1-6) - Direct impact on user experience
2. **P1**: Services (7-12) - Backend operations
3. **P2**: PTY Operations (13-15) - Terminal management
4. **P3**: Helper Files - Remove after migration complete

## Migration Strategy
1. Create command equivalents for each goroutine operation
2. Replace goroutine launches with tea.Cmd returns
3. Update message handlers for command results
4. Remove helper files once all migrations complete
5. Validate no goroutines remain in production TUI code

## Validation Commands
```bash
# Check for remaining goroutines in production code
grep -r "go func" internal/tui/ --include="*.go" | grep -v test | grep -v disabled

# Verify command pattern usage
grep -r "tea.Cmd" internal/tui/ --include="*.go" | wc -l
```