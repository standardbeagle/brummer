# Component System Migration Plan

## Current Reality Check (As of Now)

### What Actually Exists:
1. **Traditional Model** (`internal/tui/model.go`) - STILL THE DEFAULT
   - This is what users see when they run `./brum`
   - Monolithic architecture with all views in one place
   - Tightly coupled with direct field access

2. **Component System** (`internal/tui/components/`) - BUILT BUT NOT ACTIVE
   - Only activated with `BRUMMER_USE_COMPONENTS=true ./brum`
   - IntegratedShell, ProcessViewNew, LogsViewComponent exist
   - Tests written and passing (8 fixed tests)
   - NOT connected to real log data flow in production

3. **Bridge Layer** (`internal/tui/model_with_components.go`)
   - Wraps traditional model
   - Conditionally uses components based on environment variable
   - Currently using IntegratedShell when components enabled

## The Problem

The component system was built in parallel but never actually replaced the old system. Users are still seeing the old TUI. The new components don't show logs because they're not properly connected to the real data flow.

## Migration Phases

### Phase 1: Fix Component Data Flow (CRITICAL - Do This First!)
**Goal**: Make components actually work when enabled

1. [ ] Verify IntegratedShell receives real log data from process.Manager
   - Check processMgr.AddLogCallback is properly wired
   - Ensure log store is receiving updates
   - Fix subscription notification system

2. [ ] Test with real processes
   ```bash
   BRUMMER_USE_COMPONENTS=true ./brum
   # Start a script and verify logs appear
   ```

3. [ ] Debug why logs aren't showing:
   - Add debug logging to track log flow
   - Check if callbacks are being triggered
   - Verify subscription debouncing isn't blocking

### Phase 2: Feature Parity Verification
**Goal**: Ensure components have all features of old system

Features to verify:
- [ ] Process list with status indicators
- [ ] Log viewing with filtering
- [ ] Error view with context
- [ ] URL extraction and display
- [ ] Web view (browser integration)
- [ ] AI Coder sessions
- [ ] Settings management
- [ ] Tab navigation (1-7 shortcuts)
- [ ] Search/filter functionality
- [ ] Process control (start/stop/restart)

### Phase 3: Side-by-Side Testing
**Goal**: Run both systems in parallel for comparison

1. [ ] Create test script that exercises all features
2. [ ] Run with traditional model, document behavior
3. [ ] Run with components, document differences
4. [ ] Fix any discrepancies

### Phase 4: Gradual Default Switch
**Goal**: Make components the default gradually

1. [ ] Add config option (not just env var) to enable components
   ```toml
   [ui]
   use_components = true
   ```

2. [ ] Create feature flag with rollback capability
   ```go
   if cfg.UI.UseComponents || os.Getenv("BRUMMER_USE_COMPONENTS") == "true" {
       // Use components
   }
   ```

3. [ ] Beta test with subset of users

### Phase 5: Complete Migration
**Goal**: Remove old system entirely

1. [ ] Make components the only code path
2. [ ] Remove model.go views code
3. [ ] Remove model_with_components.go bridge
4. [ ] Clean up unused code

## Immediate Next Steps (DO THESE NOW)

### Step 1: Test Current Component System
```bash
# Build fresh
make build

# Test with components enabled
BRUMMER_USE_COMPONENTS=true ./brum

# Start a npm script and check if logs appear
# If logs don't appear, THIS IS THE FIRST PROBLEM TO FIX
```

### Step 2: Add Debug Logging
Add temporary debug output to track data flow:

```go
// In integrated_shell.go handleLogStoreUpdate
func (s *IntegratedShell) handleLogStoreUpdate() tea.Cmd {
    fmt.Fprintf(os.Stderr, "DEBUG: handleLogStoreUpdate called\n")
    // ... existing code
}

// In process manager callback
processMgr.AddLogCallback(func(processID, line string, isError bool) {
    fmt.Fprintf(os.Stderr, "DEBUG: Log received: %s: %s\n", processID, line)
    // ... existing code
})
```

### Step 3: Fix Data Connection
The likely issue is that IntegratedShell isn't properly receiving logs from the process manager. Check:

1. Is the callback registered?
2. Is the log store being updated?
3. Is the subscription being notified?
4. Is the debounced update blocking?

## Success Criteria

You'll know the migration is complete when:

1. `./brum` (without any env vars) uses the component system
2. All features from old system work in new system
3. Tests pass for all components
4. Old model code is deleted
5. No references to BRUMMER_USE_COMPONENTS remain

## Current Blockers

1. **Logs not displaying in component mode** - THIS MUST BE FIXED FIRST
2. **66 uncommitted files** - Need to be reviewed and committed properly
3. **No integration tests** comparing old vs new behavior
4. **No user documentation** for component system

## Testing Commands

```bash
# Test old system (current default)
./brum

# Test new component system
BRUMMER_USE_COMPONENTS=true ./brum

# Test with debug output
BRUMMER_USE_COMPONENTS=true BRUMMER_DEBUG=true ./brum

# Test minimal shell (for debugging)
BRUMMER_USE_COMPONENTS=true BRUMMER_MINIMAL_SHELL=true ./brum

# Run component tests
go test -v ./internal/tui/components/
```

## DO NOT:
- Don't assume components are working just because tests pass
- Don't delete old code until new code is proven in production
- Don't make components default until they actually show logs
- Don't trust that the wiring is correct - verify with real processes

## REMEMBER:
The component system currently DOES NOT SHOW LOGS when running real processes. This is the #1 priority to fix before any other migration work.