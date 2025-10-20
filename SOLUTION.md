# SOLUTION: Fix View() Blocking in ProcessesViewComponent

## Root Cause Found

**Location**: `/home/beagle/work/brummer/internal/tui/components/processes_view.go` line 74-101

**The Problem**: `ViewWithState()` is being called from `View()` and it's **blocking** when rendering the processes view.

From the debug log:
```
VIEW: View() called, width=252, view=processes
VIEW: Checking special views...
VIEW: Calling renderContent()...
[BLOCKS FOREVER - never returns]
```

## Why It Blocks

Looking at `processes_view.go` lines 83-87:
```go
if processesState.IsCacheValid(2*time.Second) {
    processes = processesState.GetProcesses()
} else if processMgr != nil {
    processes = processMgr.GetAllProcesses()  // ← THIS MIGHT BLOCK
}
```

When the cache is invalid, it calls `processMgr.GetAllProcesses()` which:
1. Iterates over a sync.Map
2. Might be blocking if the map is being modified
3. Or might be calling something that blocks

## The Architecture Violation

From your `TUI_DEADLOCK_PREVENTION.md`:
> **NEVER call blocking operations from View()** - causes UI freezes

`ViewWithState()` is violating this by calling `processMgr.GetAllProcesses()` directly from the render path!

## The Fix

**Option 1: Always use cached data in View()**

```go
func (c *ProcessesViewComponent) ViewWithState(masterData *state.MasterData, processMgr *processes.Manager) string {
    if !c.IsVisible() || masterData == nil || masterData.ProcessesState == nil {
        return ""
    }

    processesState := masterData.ProcessesState

    // ALWAYS use cached data in View() - NEVER call processMgr directly
    processes := processesState.GetProcesses()

    if len(processes) == 0 {
        return c.renderEmptyState()
    }

    // Create list items
    items := c.createListItems(processes, processesState)

    // Update list
    c.list.SetItems(items)
    c.list.SetSize(c.width, c.height-2)

    return c.list.View()
}
```

The cache update should happen in `Update()` via `updateProcessList()`, NOT in `View()`.

**Option 2: Remove the conditional cache check**

The problem is the `IsCacheValid()` check. If we always use cached data and update it via `updateProcessList()` in `Update()`, View() becomes pure rendering.

## Why This Works

1. **View() becomes pure** - only renders from state, never blocks
2. **Update() updates state** - `updateProcessList()` is called in TickMsg handler
3. **Follows Redux/Elm** - View is pure function of state
4. **Prevents deadlocks** - No blocking operations in render path

## Testing the Fix

1. Edit `/home/beagle/work/brummer/internal/tui/components/processes_view.go`
2. Remove the cache validity check
3. Always use `processesState.GetProcesses()`
4. Rebuild and test

The TickMsg handler already calls `updateProcessList()` every second (line 1047 in model.go), so the cache will be fresh.

## Why ProgressiveModel Works

ProgressiveModel has a simple View() that doesn't call any component rendering. When we add the real Update() logic to ProgressiveModel, it will likely break too!

The real issue: **Component View() methods are blocking by calling external services**.

Fix all component View() methods to only render from cached state!
