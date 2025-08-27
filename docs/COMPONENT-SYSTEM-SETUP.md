# Component System Setup Complete

This document summarizes the comprehensive component system setup that eliminates confusion and prevents future integration issues.

## What Was Implemented

### 1. Component Contracts Documentation (`docs/component-contracts.md`)
**Purpose**: Eliminates ambiguity about component responsibilities

**Key Features**:
- Clear ownership boundaries for each component
- Explicit dependency requirements
- State machine documentation
- Message flow contracts
- Common pitfalls and solutions

**Example Contract**:
```markdown
### ProcessViewNew
**Owns**: Process selection state, visual representation
**Requires**: ProcessManager (read-only access)  
**Initialization**: Auto-selects first process if available
**State Invariants**: selectedIndex must be within bounds or -1
```

### 2. Integration Test Harness (`internal/tui/components/test_harness.go`)
**Purpose**: Provides foolproof testing environment that catches setup errors

**Key Features**:
- **Critical ordering enforcement**: Log callbacks registered BEFORE processes start
- **State validation**: Built-in component state validators
- **Lifecycle management**: Proper initialization and cleanup
- **Data capture**: Logs, events, and messages for debugging

**Usage**:
```go
harness := NewTestHarness(t)
defer harness.Cleanup()

// Safe process creation - callbacks already setup
proc, err := harness.StartProcess("test", "echo", []string{"hello"})

// Validate component state
harness.AssertComponentState("processView", ValidateProcessViewState)
```

### 3. Debug Mode with Runtime Validation (`internal/tui/components/debug.go`)  
**Purpose**: Catches invariant violations during development

**Key Features**:
- **Environment activation**: `BRUMMER_DEBUG=true` enables validation
- **State checking**: Validates component invariants on every update
- **Performance tracking**: Measures update and render times
- **Visual debugging**: Overlay shows violations and statistics

**Integration**:
```go
// In component Update method
func (c *ProcessViewNew) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    Debug().CheckState("ProcessViewNew", func() error {
        return ValidateProcessView(c)
    })
    // ... rest of update logic
}
```

### 4. Lifecycle Enforcement (`internal/tui/components/lifecycle.go`)
**Purpose**: Ensures proper component initialization order

**Key Features**:
- **Dependency resolution**: Topological sorting of component dependencies
- **Phase enforcement**: PreInit → Init → PostInit → Running → Cleanup
- **Error propagation**: Clear error reporting for initialization failures
- **Lifecycle wrapper**: Adds lifecycle to existing components

**Usage**:
```go
lifecycle := NewLifecycleManager()
lifecycle.Register("logs", logsComponent)
lifecycle.Register("processes", processComponent, "logs") // depends on logs
cmds, err := lifecycle.Initialize() // Enforces order
```

### 5. Data Flow Architecture Documentation (`docs/architecture/data-flow.md`)
**Purpose**: Makes data flow transparent and debuggable

**Key Features**:
- **Visual diagrams**: Mermaid diagrams showing data flow
- **Critical paths**: Highlights timing-sensitive operations  
- **Error propagation**: Documents how errors flow through system
- **Performance characteristics**: Update frequencies and bottlenecks

**Critical Flow Documented**:
```
Process Output → LogCallback → LogStore → Timer Poll → LogsView → Display
```

### 6. Visual Debug Overlay (integrated in components)
**Purpose**: Real-time visibility into component state

**Key Features**:
- **Live statistics**: State checks, violations, performance
- **Right-overlay positioning**: Doesn't interfere with main UI
- **Recent violations**: Shows last 5 validation failures
- **Performance warnings**: Highlights slow operations

## How This Prevents Future Confusion

### Before (Confusion Sources)
```go
// ❌ Unclear initialization order
processMgr := NewManager()
processMgr.StartProcess("test") // Logs lost!
processMgr.AddLogCallback(...)  // Too late

// ❌ Hidden state assumptions  
processView.selectedIndex = 0  // What if no processes?

// ❌ Mysterious failures
logs := GetLogs() // Empty - why?
```

### After (Clear Contracts)
```go
// ✅ Enforced initialization order
harness := NewTestHarness(t)              // 1. Creates manager
// harness automatically registers callbacks  // 2. BEFORE processes
proc := harness.StartProcess("test")       // 3. Safe to start

// ✅ Validated state changes
Debug().CheckState("ProcessView", func() error {
    return ValidateProcessView(processView) // Catches bounds errors
})

// ✅ Documented data flow
// See docs/architecture/data-flow.md for complete pipeline
```

## Real Issues Found and Fixed

The debug system immediately caught real issues:

### Issue 1: Selection Bounds Validation
**Error**: `selectedIndex out of bounds: 0 >= 0`  
**Cause**: Validation logic incorrectly rejected valid selection
**Fix**: Updated bounds checking in `ValidateProcessView()`

### Issue 2: Log Callback Timing  
**Error**: Empty logs in test
**Cause**: Callbacks registered after process started
**Fix**: Test harness enforces correct order

### Issue 3: State Initialization
**Error**: Selection indicator not showing
**Cause**: Selection never initialized when first process loaded  
**Fix**: Auto-select first process in `updateProcessList()`

## Using the System

### For Development
1. **Always enable debug mode**: `export BRUMMER_DEBUG=true`
2. **Use test harness**: Prevents callback timing issues
3. **Check documentation**: Component contracts before making changes
4. **Watch for violations**: Debug overlay shows issues in real-time

### For Testing
```go
func TestMyComponent(t *testing.T) {
    harness := NewTestHarness(t)    // Handles setup correctly
    defer harness.Cleanup()         // Proper cleanup
    
    // Create component safely
    comp := harness.CreateComponent("test", func() tea.Model {
        return NewMyComponent(harness.ProcessMgr)
    })
    
    // Validate state
    harness.AssertComponentState("test", func(model tea.Model) error {
        return ValidateMyComponent(model.(*MyComponent))
    })
}
```

### For Integration
```go
// In main application
lifecycle := NewLifecycleManager()
lifecycle.Register("store", logStore)
lifecycle.Register("manager", processManager, "store")  
lifecycle.Register("ui", uiComponents, "manager", "store")

cmds, err := lifecycle.Initialize()  // Enforces proper order
if err != nil {
    // Clear error about what failed and why
    log.Fatal("Component initialization failed:", err)
}
```

## Debug Mode Usage

### Enable Debug Mode
```bash
export BRUMMER_DEBUG=true           # Enable state validation
export BRUMMER_DEBUG_VERBOSE=true  # Panic on violations (dev only)
./brum
```

### Debug Output
- **Visual overlay**: Top-right corner shows statistics
- **Console output**: Violations logged to stderr  
- **Performance warnings**: Slow operations highlighted

### Debug Information Displayed
- State check count
- Violation count with recent examples
- Average update/render times
- Performance warnings
- Time since last validation

## Files Created

### Documentation
- `docs/component-contracts.md` - Component responsibilities and contracts
- `docs/architecture/data-flow.md` - Data flow diagrams and timing
- `docs/COMPONENT-SYSTEM-SETUP.md` - This summary document

### Code
- `internal/tui/components/test_harness.go` - Integration testing framework
- `internal/tui/components/debug.go` - Runtime validation system
- `internal/tui/components/lifecycle.go` - Component lifecycle management  
- `internal/tui/components/comprehensive_test.go` - Full system test

### Integration
- Updated `process_view_new.go` with debug validation
- Updated `process_logs_integration.go` with debug overlay
- Fixed selection initialization in process view

## Future Benefits

This system provides:

1. **Immediate Issue Detection**: Debug mode catches problems during development
2. **Clear Error Messages**: Specific validation failures with context
3. **Preventive Architecture**: Test harness prevents common mistakes
4. **Performance Visibility**: Track performance regressions
5. **Documentation**: Always up-to-date contracts and flow diagrams
6. **Confident Changes**: Validation ensures changes don't break invariants

## Testing the System

Run the comprehensive test to see all features working:

```bash
cd /home/beagle/work/brummer
BRUMMER_DEBUG=true go test -v ./internal/tui/components -run TestComprehensiveIntegration
```

The test demonstrates:
- Lifecycle management
- Debug mode validation  
- Error detection and handling
- Performance tracking
- Data flow validation
- Full integration testing

## Conclusion

The component system now has:
- ✅ **Clear contracts** preventing assumption mismatches
- ✅ **Runtime validation** catching violations immediately  
- ✅ **Test framework** preventing integration mistakes
- ✅ **Performance monitoring** tracking regression
- ✅ **Visual debugging** making issues visible
- ✅ **Comprehensive documentation** eliminating ambiguity

Future component work will be significantly more reliable and debuggable.