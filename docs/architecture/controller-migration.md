# Controller to Service/Component Migration Guide

## Overview

This guide documents the systematic migration from the controller-based architecture to the direct Component → Service pattern. The migration removes ~8,000 lines of controller code and eliminates architectural debt from a previous incomplete refactoring.

## Architecture Evolution

### Before: Controller-Based (4 Layers)
```
User Input
    ↓
Model (routes to controllers)
    ↓
Controllers (coordinate & wrap services)
    ↓
Components (passive view logic)
    ↓
Services (process.Manager, logs.Store, etc.)
```

**Problems:**
- Tight coupling between Model and Controllers (203 references)
- Circular dependencies between controllers
- Duplicate state management (controllers + components)
- Difficult to test (requires mocking controllers AND services)
- Performance overhead from extra coordination layer

### After: Component-Based (2 Layers)
```
User Input
    ↓
Components (active BubbleTea models)
    ↓
Services (process.Manager, logs.Store, etc.)
    ↓
EventBus (async updates back to components)
```

**Benefits:**
- Direct service integration (no wrapper layer)
- Components own view-specific state
- Testable with mock services only
- Standard BubbleTea patterns throughout
- Reduced complexity (~8,000 fewer lines)

## Migration Phases

### Phase 1: Foundation (Event & Layout)
**Controllers to Remove:**
- `event_controller.go` - EventBus bridge
- `layout_controller.go` - Layout calculations

**Migration:**
- Move EventBus subscription to `model.go` Init()
- Convert layout logic to pure function utility
- No other controllers depend on these

**Verification:**
- EventBus messages still reach components
- Layout calculations produce identical results
- No performance regression

### Phase 2: Simple Views
**Controllers to Remove:**
- `urls_view_controller.go` - URL display
- `mcp_debug_controller.go` - MCP debugging view
- `command_window_controller.go` - Command palette

**Migration:**
- Create/enhance components: `urls_view.go`, `mcp_debug_view.go`, `command_palette.go`
- Move state to component structs
- Implement direct service calls
- Add KeyHandler implementation

**Verification:**
- URL discovery still works
- MCP debug view displays status
- Command palette opens on `/`

### Phase 3: Core Views
**Controllers to Remove:**
- `process_view_controller.go` - Process list/management
- `logs_view_controller.go` - Log viewing/filtering
- `errors_view_controller.go` - Error display

**Migration:**
- Create/enhance components for each view
- Move process manager, log store, error store integration
- Implement view-specific key handling
- Add real-time update via EventBus

**Verification:**
- Process start/stop works
- Logs update in real-time
- Error dismissal works
- All keyboard shortcuts functional

### Phase 4: Complex Views
**Controllers to Remove:**
- `web_view_controller.go` - Proxy request viewing
- `settings_controller.go` - Settings management
- `script_selector_controller.go` - Script selection

**Migration:**
- Enhance existing `web_view.go` component
- Create `settings_view.go` and `script_selector.go`
- Implement proxy server, config service integration
- Add filtering, editing, and save logic

**Verification:**
- Proxy requests display correctly
- Settings edits persist
- Script selection launches processes

### Phase 5: Advanced Features
**Controllers to Remove:**
- `ai_coder_controller.go` - AI coder session management

**Migration:**
- Enhance existing `integrated_shell.go` component
- Move PTY terminal integration
- Implement slash command interception
- Handle session lifecycle

**Verification:**
- Terminal emulator works
- Slash commands route correctly
- Session creation/deletion works

### Phase 6: Input System (MUST BE LAST)
**Controllers to Remove:**
- `input_controller.go` - Global keyboard routing

**Migration:**
- Enhance `hierarchical_key_handler.go`
- Move global key handling logic
- Update all components to implement KeyHandler
- Migrate quit confirmation logic

**Verification:**
- All keyboard shortcuts work
- Key priority system functions
- Quit confirmation works

### Phase 7: Model Cleanup
**Actions:**
- Remove all controller field declarations from `model.go`
- Remove all controller initialization code
- Update Update() to route messages to components directly
- Update View() to call component View() methods directly

**Verification:**
- Zero references to deleted controllers
- All views render correctly
- All functionality intact

## Migration Checklist (Per Controller)

For each controller removal:

1. ✅ **Create or enhance target component**
   - Component struct with service dependencies
   - BubbleTea Init(), Update(), View() methods
   - KeyHandler implementation if needed

2. ✅ **Migrate controller logic to component**
   - Move view state to component struct
   - Convert controller methods to component methods
   - Remove controller wrapper around service calls

3. ✅ **Update service calls to be direct**
   - Replace `controller.DoAction()` with `service.DoAction()`
   - Add async tea.Cmd for operations with side effects
   - Handle errors and convert to tea.Msg

4. ✅ **Implement BubbleTea Update() and View() methods**
   - Handle all relevant tea.Msg types
   - Delegate to sub-components (list, viewport, etc.)
   - Pure View() function of component state

5. ✅ **Add component to model (if new)**
   - Add field to Model struct
   - Initialize in NewModel()
   - Wire up in model Update() and View()

6. ✅ **Remove controller file**
   - Delete `internal/tui/*_controller.go`
   - Verify no compilation errors

7. ✅ **Remove all controller references from model.go**
   - Remove field declaration
   - Remove initialization code
   - Remove method calls
   - Verify with grep: `grep -n "ControllerName" internal/tui/model.go`

8. ✅ **Run corresponding integration test**
   - Verify functionality unchanged
   - Check for regressions
   - Measure performance

9. ✅ **Run manual verification**
   - Test keyboard shortcuts
   - Verify real-time updates
   - Check edge cases

10. ✅ **Commit with clear message**
    - Format: `refactor(tui): migrate <controller> to component pattern`
    - Include verification test results
    - Note any behavior changes

## Testing Strategy

### Pre-Migration (TDD Requirement)
**CRITICAL**: Write integration tests BEFORE removing any controllers

```bash
# Create integration test that verifies current behavior
go test -v ./test/integration/ -run TestProcessManagement

# Test MUST PASS with controller present
# This establishes baseline functionality
```

Integration tests (T006-T017):
- Process management workflow
- Log viewing and filtering
- Error display and management
- URL discovery and display
- Proxy request viewing
- AI coder session lifecycle
- Command palette functionality
- Settings management
- Keyboard navigation
- Script selector functionality

### During Migration
After each controller removal:

```bash
# Run corresponding integration test
go test -v ./test/integration/ -run TestProcessManagement

# Run full test suite
make test

# Run with race detection
make test-race

# Manual verification
./brum  # Test all functionality
```

### Post-Migration
Final verification:

```bash
# Full integration test suite
make test-integration

# Performance tests
make test-performance

# Race condition tests
make test-race

# Manual testing checklist
- [ ] All keyboard shortcuts work
- [ ] All views display correctly
- [ ] Real-time updates work
- [ ] No error messages in logs
- [ ] No performance regressions
```

## Common Migration Patterns

### Pattern 1: Simple State Migration
```go
// Before (controller owns state)
type URLsViewController struct {
    state URLsViewState
    mgr   *process.Manager
}

type URLsViewState struct {
    urls       []URL
    selectedID string
}

// After (component owns state)
type URLsViewComponent struct {
    mgr        *process.Manager  // Service dependency
    list       bubbles.List      // BubbleTea component
    selectedID string             // View state
    width      int
    height     int
}
```

### Pattern 2: Service Call Migration
```go
// Before (controller wraps service)
func (c *ProcessViewController) StartProcess(id string) error {
    c.state.Loading = true
    err := c.processMgr.StartProcess(id)
    c.state.Loading = false
    return err
}

// After (component calls service directly)
func (p *ProcessesViewComponent) startProcess(id string) tea.Cmd {
    return func() tea.Msg {
        err := p.processMgr.StartProcess(id)
        if err != nil {
            return ProcessErrorMsg{ID: id, Error: err}
        }
        return ProcessStartedMsg{ID: id}
    }
}
```

### Pattern 3: Key Handling Migration
```go
// Before (controller handles keys)
func (c *ProcessViewController) HandleKey(key string) tea.Cmd {
    switch key {
    case "enter":
        return c.startProcess(c.state.SelectedID)
    }
    return nil
}

// After (component implements KeyHandler)
func (p *ProcessesViewComponent) HandleKey(msg tea.KeyMsg, model tea.Model) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "enter":
        return p, p.startProcess(p.selectedID)
    default:
        return model, nil  // Not handled - allow propagation
    }
}

func (p *ProcessesViewComponent) Priority() HandlerPriority {
    return PriorityViewSpecific  // 25
}
```

### Pattern 4: EventBus Integration
```go
// Before (controller subscribes and updates state)
func (c *ProcessViewController) Init() {
    c.eventBus.Subscribe("process.*", func(event Event) {
        c.state.Processes = c.processMgr.GetAllProcesses()
        c.Refresh()
    })
}

// After (component converts events to tea.Msg)
func (p *ProcessesViewComponent) Init() tea.Cmd {
    return func() tea.Msg {
        p.eventBus.Subscribe("process.*", func(event Event) {
            p.updateChan <- ProcessEventMsg{Event: event}
        })
        return nil
    }
}

func (p *ProcessesViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ProcessEventMsg:
        // Update state in response to event
        p.refreshProcessList()
        return p, nil
    }
    return p, nil
}
```

## Troubleshooting

### Issue: Compilation errors after controller removal
**Solution:**
- Use grep to find all references: `grep -r "ControllerName" internal/tui/`
- Remove field declarations, initialization, method calls
- Update imports if controller was exported

### Issue: Tests fail after migration
**Solution:**
- Compare behavior before/after migration
- Check service calls are identical
- Verify EventBus subscriptions still work
- Run with `-v` flag to see detailed output

### Issue: Key handling broken
**Solution:**
- Verify component implements KeyHandler interface
- Check Priority() returns correct level
- Ensure HandleKey() returns `(model, nil)` for unhandled keys
- Test hierarchical routing with debug logging

### Issue: Real-time updates stop working
**Solution:**
- Verify EventBus subscriptions in Init()
- Check event-to-message conversion in Update()
- Ensure updateChan is not blocked
- Test event emission from services

### Issue: Performance regression
**Solution:**
- Profile with `go test -cpuprofile=cpu.prof`
- Check for accidental O(n²) operations
- Verify no goroutine leaks
- Run performance baseline tests (T016-T017)

## Success Criteria

Migration is complete when:

- ✅ All 13 controller files deleted
- ✅ Zero references to deleted controllers in model.go (verified with grep)
- ✅ All integration tests pass (T006-T017)
- ✅ All unit tests pass (T102-T111)
- ✅ Performance tests pass (T016-T017, T099-T101)
- ✅ Race condition tests pass (`make test-race`)
- ✅ Manual testing passes all scenarios
- ✅ Code coverage >80%
- ✅ Documentation updated
- ✅ Code review approved

## Timeline

- **Phase 1 (Foundation)**: 2-3 days
- **Phase 2 (Simple Views)**: 3-4 days
- **Phase 3 (Core Views)**: 5-7 days
- **Phase 4 (Complex Views)**: 5-7 days
- **Phase 5 (Advanced)**: 3-4 days
- **Phase 6 (Input System)**: 2-3 days
- **Phase 7 (Cleanup)**: 2-3 days

**Total Estimated Time**: 3-4 weeks for systematic migration

## References

- [Component Interface Documentation](../../internal/tui/components/README.md)
- [Service Interaction Contract](../../specs/002-the-controllers-need/contracts/service-interaction.yaml)
- [Key Handler Contract](../../specs/002-the-controllers-need/contracts/key-handler.yaml)
- [Architecture Overview](./overview.md)
- [Testing Strategy](../../specs/002-the-controllers-need/quickstart.md)
