# Component System Migration Guide

This guide explains how to migrate the existing Brummer TUI from the monolithic Model structure to the new component-based architecture.

## Overview

The component system provides a gradual migration path from the current 6000+ line Model to a modular, maintainable architecture that leverages Bubbles components.

## Migration Strategy

### Phase 1: Enable Component Mode (Current)

The component system is now integrated and can be enabled via feature flag:

```go
// In main.go or where you create the TUI model
import "github.com/standardbeagle/brummer/internal/tui"

// Create model with component system enabled
model := tui.CreateModelWithComponentFlag(
    processMgr,
    logStore,
    eventBus,
    mcpServer,
    proxyServer,
    mcpPort,
    cfg,
    true, // Enable component mode
)
```

### Phase 2: Migrate Individual Views

Views can be migrated one at a time while maintaining backward compatibility:

#### 2.1 Process View Migration

Replace `ProcessViewController` with Bubbles-based component:

```go
// Before: Custom process view controller
processViewController := NewProcessViewController(processMgr)

// After: Bubbles-based process view
processView := components.NewMigratedProcessView()
```

The new view uses:
- `BubblesTableAdapter` for process list
- `BubblesViewportAdapter` for process output
- Automatic layout management

#### 2.2 Logs View Migration

Replace `LogsViewController` with scrollable Bubbles viewport:

```go
// Before: Custom logs controller with manual scrolling
logsViewController := NewLogsViewController(logStore)

// After: Bubbles viewport with built-in scrolling
logsView := components.NewMigratedLogsView()
```

Benefits:
- Smooth scrolling handled by viewport
- Built-in search with Bubbles text input
- Filter list using Bubbles list component

#### 2.3 Errors View Migration

Replace `ErrorsViewController` with Bubbles list and viewport:

```go
// Before: Custom error list management
errorsViewController := NewErrorsViewController(logStore)

// After: Bubbles list with custom error rendering
errorsView := components.NewMigratedErrorsView()
```

### Phase 3: Replace Layout Management

The existing layout logic can be replaced with component layouts:

```go
// Before: Manual layout calculation in Model
func (m *Model) renderContent() string {
    // Complex manual layout logic
}

// After: Automatic layout with components
layout := components.NewBoxLayout(components.Vertical)
layout.AddChild(header, 0.1)    // 10% height
layout.AddChild(content, 0.8)   // 80% height
layout.AddChild(footer, 0.1)    // 10% height
```

### Phase 4: Implement Overlays

System panels and command palettes become simple overlay components:

```go
// Create overlay stack
stack := components.NewStackLayout()
stack.AddLayer(mainLayout)

// Add system panel overlay
systemPanel := components.NewSystemPanel(renderer)
stack.AddLayer(systemPanel)

// Add command palette overlay
commandPalette := components.NewCommandPalette(handler)
stack.AddLayer(commandPalette)
```

## Data Provider Pattern

The component system uses a `DataProvider` interface to decouple UI from business logic:

```go
type DataProvider interface {
    GetProcesses() []interface{}
    StartProcess(name string) error
    StopProcess(id string) error
    GetLogs(processID string) []string
    GetErrors() []interface{}
    GetURLs() []string
    GetSettings() map[string]interface{}
}
```

The existing Model implements this through `BrummerDataProvider`:

```go
dataProvider := NewBrummerDataProvider(existingModel)
componentBridge := components.NewModelBridge(dataProvider)
```

## Message-Based Communication

Components communicate via messages instead of direct method calls:

```go
// Send data update
components.SendDataUpdate(
    components.DataUpdateProcess,
    processData,
    "source",
)

// Send notification
components.SendNotification(
    components.NotificationSuccess,
    "Process started",
    "npm run dev",
)

// Change view
components.SendViewChange("logs", nil)
```

## Testing During Migration

### 1. Enable/Disable Toggle

Test toggling between old and new UI:

```go
model := NewModelWithComponents(...)
model.EnableComponents()  // Switch to new UI
model.DisableComponents() // Switch back to old UI
```

### 2. Side-by-Side Testing

Run two instances for comparison:
- Instance 1: Legacy UI
- Instance 2: Component UI

### 3. Component Tests

```bash
# Run component tests
go test ./internal/tui/components/...

# Run specific component test
go test ./internal/tui/components -run TestBubblesIntegration
```

## Common Migration Patterns

### Pattern 1: Wrapping Existing Controllers

For controllers that work well, wrap them:

```go
adapter := components.NewViewAdapter("logs", existingController)
viewSwitcher.AddView("logs", adapter)
```

### Pattern 2: Gradual Internal Migration

Replace internals while keeping the interface:

```go
type LogsViewController struct {
    // Old: custom viewport logic
    // viewport *CustomViewport
    
    // New: Bubbles viewport
    viewport *viewport.Model
}
```

### Pattern 3: Complete Replacement

For complex views, complete replacement may be cleaner:

```go
// Remove old controller entirely
// Replace with Bubbles-based component
newView := components.CreateScrollableLog(content)
```

## Benefits After Migration

1. **Reduced Complexity**
   - No manual scroll management
   - No custom text input handling
   - Automatic focus management
   - Built-in keyboard shortcuts

2. **Better Performance**
   - Lock-free design where possible
   - Efficient message batching
   - Minimal allocations

3. **Improved Maintainability**
   - Clear component boundaries
   - Reusable components
   - Standardized patterns
   - Leverages maintained Bubbles components

4. **Enhanced Features**
   - Smooth animations (via Bubbles)
   - Better keyboard navigation
   - Consistent behavior across views
   - Easy to add new views

## Migration Checklist

- [ ] Enable component mode in development
- [ ] Test basic functionality with components
- [ ] Migrate Process view
- [ ] Migrate Logs view
- [ ] Migrate Errors view
- [ ] Migrate URLs view
- [ ] Migrate Settings view
- [ ] Replace layout management
- [ ] Add overlay support
- [ ] Remove old view controllers
- [ ] Update tests
- [ ] Update documentation
- [ ] Enable components in production

## Troubleshooting

### Issue: Components not rendering
- Check that component mode is enabled
- Verify data provider is returning data
- Check component visibility flags

### Issue: Performance degradation
- Profile with `go tool pprof`
- Check for excessive allocations
- Verify message batching is working

### Issue: Focus not working
- Ensure FocusChain is properly configured
- Check that components implement Focus/Blur
- Verify no modal is capturing input

### Issue: Data not updating
- Check data provider implementation
- Verify message routing
- Ensure components handle DataUpdateMsg

## Rollback Plan

If issues arise, the system can be instantly rolled back:

```go
// Disable component mode
model.DisableComponents()

// Or use environment variable
export BRUMMER_USE_LEGACY_UI=true
```

The dual-mode support will be maintained until migration is complete and stable.

## Next Steps

1. Start with enabling component mode in development
2. Migrate simplest view first (URLs or Settings)
3. Gather feedback and iterate
4. Progressively migrate more complex views
5. Remove legacy code once stable

## Support

For questions or issues during migration:
- Check component tests for examples
- Review `components/README.md` for architecture details
- Look at `migration_example.go` for patterns
- File issues with the `component-migration` label