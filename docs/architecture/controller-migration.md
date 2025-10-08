# Controller to Component Migration Guide

**Feature**: 002-the-controllers-need
**Status**: Phase 3.6 Complete (11/13 controllers migrated, 85%)
**Last Updated**: 2025-10-07

## Overview

This guide documents the systematic migration from a 4-layer controller-based architecture to a clean 2-layer BubbleTea component architecture.

**Before**: Model → Controllers → Components → Services (4 layers)
**After**: Components → Services (2 layers)

## Why Migrate?

### Problems with Controllers

1. **Unnecessary abstraction**: Controllers added indirection without value
2. **Circular dependencies**: Controllers called each other, creating tangled code
3. **Testing complexity**: Had to mock controllers AND services
4. **Code duplication**: Similar patterns repeated in each controller
5. **BubbleTea mismatch**: Controllers fought against BubbleTea's Update/View pattern

### Benefits of Components

1. **Simpler architecture**: 2 layers instead of 4
2. **Direct service calls**: Components call services directly
3. **Better testing**: Test components with mock services
4. **BubbleTea alignment**: Components ARE BubbleTea models
5. **Less code**: ~6,500+ lines of controller code removed

## Migration Process

### Prerequisites

✅ **Required before ANY migration**:
1. Write integration tests for current functionality (Phase 3.2)
2. Verify tests pass with controllers in place
3. Ensure build is clean with zero errors

⚠️ **Critical**: Do NOT skip Phase 3.2 integration tests. TDD principle requires tests before refactoring.

### Phase-by-Phase Approach

Migration follows a systematic 10-phase approach:

#### Phase 3.1: Setup and Documentation (T001-T005)
- ✅ Component interface documentation
- ✅ Migration guide (this document)
- ✅ Test framework setup

#### Phase 3.2: Pre-Migration Tests (T006-T017)
- ⚠️ **CRITICAL**: Must complete before Phase 3.3
- ✅ 5/12 tests complete (T006-T009, T014)
- ⚠️ 7/12 tests missing (T010-T013, T015-T017)

#### Phase 3.3: Foundation Migration (T018-T022)
- ✅ Event controller → model.go
- ✅ Layout controller → layout_manager.go
- **Impact**: Removed 2 controllers, ~400 lines

#### Phase 3.4-3.6: View Migrations (T023-T073)
- ✅ Simple views (URLs, MCP Debug, Command Palette)
- ✅ Core views (Processes, Logs, Errors)
- ✅ Complex views (Web/Proxy, Settings, Script Selector)
- **Impact**: Removed 9 controllers, ~6,100 lines

#### Phase 3.7: AI Coder Migration (T074-T079)
- ⚠️ **BLOCKED**: Missing T011 integration test
- **Status**: NOT STARTED

#### Phase 3.8: Input Controller Migration (T080-T087)
- **Status**: 35% complete (hierarchical_key_handler exists)
- **Remaining**: 467 lines of input logic

#### Phase 3.9: Final Cleanup (T088-T092)
- **Status**: Cannot start until Phases 3.7-3.8 complete

#### Phase 3.10: Polish (T099-T120)
- Performance optimization
- Unit tests
- Documentation updates

## Step-by-Step Migration

### Step 1: Analyze the Controller

Before migrating, understand what the controller does:

```bash
# Find the controller file
ls internal/tui/*_controller.go

# Analyze dependencies
grep "import\|type.*Controller\|func (c \*.*Controller)" internal/tui/process_view_controller.go

# Find all references
grep -r "processViewController" internal/tui/
```

**Document**:
- Public methods and their usage
- Service dependencies
- State management patterns
- Integration points with model.go

### Step 2: Create or Enhance Component

Create the component that will replace the controller:

```go
// Before: internal/tui/process_view_controller.go
type ProcessViewController struct {
    processMgr *process.Manager
    // ... controller state
}

func (c *ProcessViewController) GetProcesses() []*process.Process {
    return c.processMgr.GetAllProcesses()
}

// After: internal/tui/components/processes_view.go
type ProcessesViewComponent struct {
    BaseComponent
    viewport viewport.Model
    processMgr *process.Manager  // Direct service reference
    // ... component state
}

func (c *ProcessesViewComponent) Init() tea.Cmd {
    return nil
}

func (c *ProcessesViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle messages, call services directly
    switch msg := msg.(type) {
    case ProcessStartMsg:
        return c, c.startProcess(msg.Name)
    }
    return c, nil
}

func (c *ProcessesViewComponent) View() string {
    // Render using processes from service
    processes := c.processMgr.GetAllProcesses()
    return c.renderProcessList(processes)
}
```

**Key Changes**:
1. Implement `tea.Model` interface (Init, Update, View)
2. Call services directly (no controller wrapper)
3. Use `tea.Cmd` for async operations
4. Store UI state in component (not service)

### Step 3: Migrate Controller Logic

Move each controller method to the component:

**Method Categories**:

1. **State accessors** → Component fields
   ```go
   // Before: Controller method
   func (c *ProcessViewController) GetSelected() string {
       return c.selected
   }

   // After: Direct field access or getter
   func (c *ProcessesViewComponent) GetSelected() string {
       return c.selectedProcess
   }
   ```

2. **Service calls** → tea.Cmd functions
   ```go
   // Before: Controller delegates to service
   func (c *ProcessViewController) StartProcess(name string) {
       c.processMgr.StartScript(name)
   }

   // After: Component returns tea.Cmd
   func (c *ProcessesViewComponent) startProcess(name string) tea.Cmd {
       return func() tea.Msg {
           err := c.processMgr.StartScript(name)
           if err != nil {
               return ProcessStartErrorMsg{Name: name, Err: err}
           }
           return ProcessStartedMsg{Name: name}
       }
   }
   ```

3. **Rendering logic** → View() method
   ```go
   // Before: Controller has Render() method
   func (c *ProcessViewController) Render() string {
       return c.renderList()
   }

   // After: Component implements View()
   func (c *ProcessesViewComponent) View() string {
       if !c.IsVisible() {
           return ""
       }
       return c.viewport.View()
   }
   ```

### Step 4: Update Service Calls

Change from controller delegation to direct service calls:

```go
// Before: model.go calls controller
func (m *Model) startProcess(name string) {
    m.processViewController.StartProcess(name)
}

// After: component calls service in Update()
func (c *ProcessesViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "enter" && c.canStart() {
            return c, c.startProcess(c.selectedProcess)
        }
    }
    return c, nil
}
```

### Step 5: Handle BubbleTea Messages

Components should handle relevant messages:

```go
func (c *ProcessesViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        // Resize component
        c.SetSize(msg.Width, msg.Height)
        c.viewport.Width = msg.Width
        c.viewport.Height = msg.Height

    case ProcessStartedMsg:
        // Update UI when process starts
        c.processes = c.processMgr.GetAllProcesses()
        c.updateViewport()

    case tea.KeyMsg:
        // Handle keyboard input
        return c.handleKey(msg)
    }

    return c, tea.Batch(cmds...)
}
```

### Step 6: Remove Controller References

Remove the controller from model.go:

```go
// Before: model.go has controller field
type Model struct {
    processViewController *ProcessViewController
    // ...
}

// After: model.go has component field
type Model struct {
    processesView *components.ProcessesViewComponent
    // ...
}
```

Update initialization:

```go
// Before: Create controller
m.processViewController = NewProcessViewController(processMgr)

// After: Create component
m.processesView = components.NewProcessesViewComponent(processMgr, logStore)
```

### Step 7: Delete Controller File

After ALL references are removed:

```bash
# Verify no references remain
grep -r "processViewController" internal/tui/

# If clean, delete controller
rm internal/tui/process_view_controller.go

# Verify build still works
go build ./cmd/brum
```

### Step 8: Run Integration Tests

Verify functionality with tests:

```bash
# Run specific integration test
go test -tags=integration -run TestProcessManagement ./test/integration/

# Run all integration tests
go test -tags=integration ./test/integration/
```

**If tests fail**:
1. Review error messages carefully
2. Check service calls match original behavior
3. Verify message handling in Update()
4. Ensure state updates correctly

### Step 9: Commit Changes

Commit after EACH controller migration:

```bash
git add -A
git commit -m "refactor(tui): migrate process_view_controller to processes_view component

- Removed process_view_controller.go (~200 lines)
- Enhanced processes_view.go component with direct service calls
- Updated model.go to use component instead of controller
- Verified integration tests pass

Part of controller-to-component migration (Phase 3.5, T038-T042)"
```

## Common Patterns

### Pattern 1: List View with Selection

**Before** (Controller):
```go
type ListView Controller struct {
    items []Item
    selected int
}

func (c *ListViewController) SelectNext() {
    c.selected = (c.selected + 1) % len(c.items)
}
```

**After** (Component):
```go
type ListViewComponent struct {
    BaseComponent
    viewport viewport.Model
    items []Item
    selected int
}

func (c *ListViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "down", "j":
            c.selected = (c.selected + 1) % len(c.items)
        case "up", "k":
            if c.selected > 0 {
                c.selected--
            }
        }
    }
    return c, nil
}
```

### Pattern 2: Async Service Call

**Before** (Controller):
```go
func (c *Controller) DoAction() error {
    return c.service.Execute()
}
```

**After** (Component):
```go
func (c *Component) doAction() tea.Cmd {
    return func() tea.Msg {
        err := c.service.Execute()
        if err != nil {
            return ActionErrorMsg{Err: err}
        }
        return ActionSuccessMsg{}
    }
}
```

### Pattern 3: Real-time Updates

**Before** (Controller polling):
```go
func (c *Controller) Update() {
    c.data = c.service.GetData()
}
```

**After** (Component receives messages):
```go
func (c *Component) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case DataUpdatedMsg:
        // Service sends message when data changes
        c.data = msg.Data
        c.updateViewport()
    }
    return c, nil
}
```

### Pattern 4: Error Handling

**Before** (Controller returns error):
```go
func (c *Controller) StartProcess(name string) error {
    return c.processMgr.StartScript(name)
}
```

**After** (Component sends error message):
```go
func (c *Component) startProcess(name string) tea.Cmd {
    return func() tea.Msg {
        err := c.processMgr.StartScript(name)
        if err != nil {
            // Error becomes a message for Update() to handle
            return ProcessStartErrorMsg{
                Name: name,
                Err: err,
            }
        }
        return ProcessStartedMsg{Name: name}
    }
}

func (c *Component) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ProcessStartErrorMsg:
        // Show error in UI
        c.errorMessage = msg.Err.Error()
        return c, nil
    }
    return c, nil
}
```

## Troubleshooting

### Build Errors

**Problem**: `undefined: processViewController`
**Solution**: Update all references in model.go to use component instead

**Problem**: `cannot use component (type *Component) as type tea.Model`
**Solution**: Ensure component implements Init(), Update(), View()

**Problem**: `too many arguments to Update()`
**Solution**: Check Update() signature: `Update(tea.Msg) (tea.Model, tea.Cmd)`

### Runtime Errors

**Problem**: Component not rendering
**Solution**: Check if SetSize() was called and View() returns non-empty string

**Problem**: Key presses not working
**Solution**: Ensure component receives tea.KeyMsg in Update()

**Problem**: Service calls not executing
**Solution**: Verify tea.Cmd is returned from Update() and executed

### Test Failures

**Problem**: Integration test fails after migration
**Solution**: Compare old controller behavior with new component behavior

**Problem**: Race condition detected
**Solution**: Ensure no direct state mutation, always return new state from Update()

## Testing Strategy

### Before Migration

Write integration tests that verify current functionality:

```go
func TestProcessManagement_BeforeMigration(t *testing.T) {
    // Set up with controller
    controller := NewProcessViewController(processMgr)

    // Verify current behavior
    controller.StartProcess("test")
    assert.Equal(t, "running", controller.GetStatus("test"))
}
```

### After Migration

Same test should pass with component:

```go
func TestProcessManagement_AfterMigration(t *testing.T) {
    // Set up with component
    component := NewProcessesViewComponent(processMgr, logStore)

    // Send message to start process
    component.Update(StartProcessMsg{Name: "test"})

    // Verify same behavior
    assert.Equal(t, "running", component.GetProcessStatus("test"))
}
```

### Regression Testing

After each migration:

```bash
# Run ALL integration tests
go test -tags=integration ./test/integration/ -v

# Run with race detection
go test -tags=integration -race ./test/integration/

# Check coverage
go test -tags=integration -cover ./test/integration/
```

## Best Practices

### ✅ DO

- **Write tests first** (Phase 3.2) before migrating
- **Migrate one controller at a time** to minimize risk
- **Verify build after each step** to catch errors early
- **Commit after each controller** to preserve progress
- **Use tea.Cmd for service calls** to keep Update() synchronous
- **Handle all message types** that component needs to respond to
- **Use lipgloss for styling** to keep rendering clean

### ❌ DON'T

- **Don't skip integration tests** - TDD principle is critical
- **Don't migrate multiple controllers at once** - too risky
- **Don't block in Update()** - use tea.Cmd for async work
- **Don't mutate state directly** - always return new state
- **Don't forget SetSize()** - components need to know their dimensions
- **Don't ignore build errors** - fix immediately before proceeding

## Checklist Per Controller

Use this checklist for each controller migration:

- [ ] 1. Analyze controller responsibilities and dependencies
- [ ] 2. Verify integration test exists and passes
- [ ] 3. Create or identify target component
- [ ] 4. Move controller logic to component methods
- [ ] 5. Implement tea.Model interface (Init, Update, View)
- [ ] 6. Convert service calls to tea.Cmd functions
- [ ] 7. Handle relevant BubbleTea messages
- [ ] 8. Update model.go to use component
- [ ] 9. Remove all controller references from model.go
- [ ] 10. Delete controller file
- [ ] 11. Run integration tests
- [ ] 12. Fix any test failures
- [ ] 13. Run manual verification
- [ ] 14. Commit with descriptive message
- [ ] 15. Update tasks.md to mark tasks complete

## Migration Status

### ✅ Completed Controllers (11/13, 85%)

1. event_controller.go → model.go (Foundation)
2. layout_controller.go → layout_manager.go (Foundation)
3. urls_view_controller.go → urls_view.go (Simple Views)
4. mcp_debug_controller.go → mcp_debug_view.go (Simple Views)
5. command_window_controller.go → command_palette.go (Simple Views)
6. process_view_controller.go → processes_view.go (Core Views)
7. logs_view_controller.go → logs_view.go (Core Views)
8. errors_view_controller.go → errors_view.go (Core Views)
9. web_view_controller.go → web_view.go (Complex Views)
10. settings_controller.go → settings_view.go (Complex Views)
11. script_selector_controller.go → script_selector.go (Complex Views)

### ⚠️ Remaining Controllers (2/13, 15%)

1. **ai_coder_controller.go** → aicoders_view.go (Phase 3.7, BLOCKED by missing T011 test)
2. **input_controller.go** → hierarchical_key_handler.go (Phase 3.8, 35% complete)

### 🔒 Controllers to Keep (4 controllers)

These controllers manage stateful UI elements and will be renamed to "managers" in future refactoring:

1. navigation/controller.go - Navigation state management
2. system/controller.go - System message coordination
3. filebrowser/controller.go - File selection dialog state
4. notifications/controller.go - Notification queue management

## Next Steps

To complete the migration:

1. **Write T011** (ai_coder_test.go) integration test
2. **Write T010, T012-T013, T015-T017** remaining integration tests
3. **Complete Phase 3.7** - AI Coder migration with test coverage
4. **Complete Phase 3.8** - Input controller migration
5. **Execute Phase 3.9** - Final cleanup (remove controller fields)
6. **Execute Phase 3.10** - Polish (unit tests, performance, docs)

## Related Documentation

- [Component Interface Documentation](/internal/tui/components/README.md)
- [Implementation Status](/specs/002-the-controllers-need/IMPLEMENTATION_STATUS.md)
- [Architecture Overview](/docs/architecture/overview.md)
- [Concurrency Patterns](/docs/architecture/concurrency-patterns.md)
- [BubbleTea Documentation](https://github.com/charmbracelet/bubbletea)

---

*Last Updated: 2025-10-07*
*Migration Progress: Phase 3.6 Complete (11/13 controllers, 85%)*
