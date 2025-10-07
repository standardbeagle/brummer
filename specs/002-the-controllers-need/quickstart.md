# Quick Start: Controller Migration Testing

**Feature**: Controller to Service/Component Migration
**Purpose**: Define testing scenarios for verifying functionality during migration

## Testing Philosophy

Before removing any controller, we must verify that:
1. **All functionality works** with the controller present
2. **All functionality still works** after migrating to component
3. **No regressions** in performance or user experience

## Pre-Migration Test Scenarios

These tests verify current functionality BEFORE starting migration.

### Scenario 1: Process Management Workflow

**Objective**: Verify process list, selection, and control

**Steps**:
1. Launch Brummer in a project with package.json scripts
2. Press `1` to view processes list
3. Use arrow keys to select different processes
4. Press `Enter` to start a stopped process
5. Verify process starts and shows "Running" status
6. Press `s` to stop the running process
7. Verify process stops and shows "Stopped" status

**Expected Results**:
- Process list displays correctly
- Selection moves with arrow keys
- Enter starts/stops processes
- Status updates in real-time

**Test File**: `test/integration/process_management_test.go`

### Scenario 2: Log Viewing and Filtering

**Objective**: Verify log display, filtering, and navigation

**Steps**:
1. Start a process that generates logs
2. Press `2` to view logs
3. Verify logs are displayed and updating in real-time
4. Press `f` to toggle filter mode
5. Type a search term and press Enter
6. Verify only matching logs are displayed
7. Press `p` to toggle high-priority filter
8. Verify only high-priority logs are shown
9. Press `c` to clear logs
10. Verify logs are cleared

**Expected Results**:
- Logs update in real-time
- Search filter works correctly
- Priority filter works correctly
- Clear logs works

**Test File**: `test/integration/log_viewing_test.go`

### Scenario 3: Error Display and Management

**Objective**: Verify error list, selection, and dismissal

**Steps**:
1. Trigger an error (e.g., start invalid process)
2. Press `3` to view errors
3. Verify error is displayed in list
4. Use arrow keys to select error
5. Press `Enter` to view error details
6. Press `Esc` to return to list
7. Press `d` to dismiss error
8. Verify error is removed from list
9. Press `C` to clear all errors
10. Verify all errors are cleared

**Expected Results**:
- Errors display correctly
- Detail view shows full error
- Dismiss removes error
- Clear all removes all errors

**Test File**: `test/integration/error_management_test.go`

### Scenario 4: URL Discovery and Display

**Objective**: Verify URL detection and display

**Steps**:
1. Start a process that logs a URL (e.g., dev server on localhost:3000)
2. Wait for URL to be detected
3. Press `4` to view URLs
4. Verify URL is displayed with correct process and status
5. Use arrow keys to select URL
6. Press `c` to copy URL to clipboard
7. Verify URL is copied
8. Press `Enter` to open URL in browser (if supported)

**Expected Results**:
- URLs are detected automatically
- URLs display with process info
- Copy to clipboard works
- URL list updates when processes start/stop

**Test File**: `test/integration/url_discovery_test.go`

### Scenario 5: Proxy Request Viewing

**Objective**: Verify proxy request capture and display

**Steps**:
1. Start a process with proxy enabled
2. Make HTTP requests that are captured
3. Press `5` to view web/proxy view
4. Verify requests are displayed
5. Use arrow keys to select a request
6. Press `Enter` to view request details
7. Verify request method, headers, body are shown
8. Press `f` to filter by status code
9. Type `200` and press Enter
10. Verify only 200 responses are shown

**Expected Results**:
- Requests are captured in real-time
- Request list updates as requests come in
- Detail view shows full request/response
- Filters work correctly

**Test File**: `test/integration/proxy_viewing_test.go`

### Scenario 6: AI Coder Session

**Objective**: Verify AI coder terminal and interaction

**Steps**:
1. Press `6` to view AI Coders
2. Press `n` to create new session
3. Verify terminal emulator is displayed
4. Type commands into terminal
5. Verify terminal responds correctly
6. Press `F11` to toggle fullscreen
7. Press `/` at start of line
8. Verify Brummer command palette opens
9. Type `/` mid-line in terminal
10. Verify slash is sent to terminal, not Brummer

**Expected Results**:
- Terminal emulator works correctly
- Input routing handles slash commands properly
- Fullscreen toggle works
- Session lifecycle managed correctly

**Test File**: `test/integration/ai_coder_test.go`

### Scenario 7: Keyboard Navigation

**Objective**: Verify all global keyboard shortcuts

**Steps**:
1. Press `1` to switch to processes view
2. Verify processes view is displayed
3. Press `2` to switch to logs view
4. Verify logs view is displayed
5. Press `Tab` to cycle through views
6. Verify views cycle forward
7. Press `Shift+Tab` to cycle backwards
8. Verify views cycle backward
9. Press `?` to show help
10. Verify help is displayed
11. Press `q` once
12. Verify "Press quit key again" message
13. Press `q` again within 2 seconds
14. Verify Brummer quits gracefully

**Expected Results**:
- Number keys switch views correctly
- Tab navigation works
- Help displays
- Double-quit confirmation works
- Ctrl+C also quits

**Test File**: `test/integration/keyboard_navigation_test.go`

### Scenario 8: Command Palette

**Objective**: Verify command palette functionality

**Steps**:
1. Press `/` to open command palette
2. Verify command input is displayed
3. Type `start`
4. Verify autocomplete suggestions appear
5. Press `Tab` to accept suggestion
6. Verify command is completed
7. Press `Enter` to execute command
8. Verify command executes correctly
9. Press `/` again
10. Press `Esc` to close command palette
11. Verify command palette closes
12. Press `/` and press `Backspace` with empty input
13. Verify command palette closes

**Expected Results**:
- Command palette opens on `/`
- Autocomplete works
- Commands execute
- Close on Esc and empty backspace

**Test File**: `test/integration/command_palette_test.go`

### Scenario 9: Settings Management

**Objective**: Verify settings display and modification

**Steps**:
1. Press `7` to view settings (or appropriate number)
2. Verify settings list is displayed
3. Use arrow keys to select a setting
4. Press `Enter` to edit setting
5. Modify the setting value
6. Press `Enter` to save
7. Verify setting is updated
8. Press `Esc` to cancel without saving
9. Verify setting reverts to original value

**Expected Results**:
- Settings display correctly
- Edit mode works
- Save persists changes
- Cancel reverts changes

**Test File**: `test/integration/settings_test.go`

## Post-Migration Test Scenarios

After migrating each controller, run the corresponding test scenario to verify no regressions.

### Migration Test Matrix

| Controller Migrated | Test Scenarios to Run | Pass Criteria |
|---------------------|----------------------|---------------|
| process_view_controller.go | Scenario 1, 7 | All process management and navigation works |
| logs_view_controller.go | Scenario 2, 7 | All log viewing and navigation works |
| errors_view_controller.go | Scenario 3, 7 | All error management and navigation works |
| urls_view_controller.go | Scenario 4, 7 | All URL display and navigation works |
| web_view_controller.go | Scenario 5, 7 | All proxy viewing and navigation works |
| ai_coder_controller.go | Scenario 6, 7 | All AI coder and navigation works |
| command_window_controller.go | Scenario 8, 7 | All command palette and navigation works |
| settings_controller.go | Scenario 9, 7 | All settings and navigation works |
| input_controller.go | Scenario 7 | All keyboard shortcuts work |
| layout_controller.go | All scenarios | All views render correctly |
| event_controller.go | All scenarios | All real-time updates work |

## Performance Testing

### Key Event Latency Test

**Objective**: Verify key event response time <50ms

**Setup**:
```bash
go test -run TestKeyEventLatency -v ./test/integration/
```

**Test**:
1. Send 100 key events in rapid succession
2. Measure time from key send to UI update
3. Calculate p50, p95, p99 latencies

**Pass Criteria**:
- p95 latency < 50ms
- p99 latency < 100ms
- No key events dropped

**Test File**: `test/integration/key_latency_test.go`

### View Rendering Performance

**Objective**: Verify view rendering time <16ms (60fps)

**Test**:
1. Switch between views 100 times
2. Measure render time for each view
3. Calculate average render time

**Pass Criteria**:
- Average render time < 16ms
- No dropped frames
- Smooth transitions

**Test File**: `test/integration/render_performance_test.go`

## Integration Test Helpers

### Test Utilities

```go
// test/integration/helpers.go

// Create test model with mock services
func NewTestModel(t *testing.T) *tui.Model {
    processMgr := process.NewManager()
    logStore := logs.NewStore()
    // ... other services
    return tui.NewModel(processMgr, logStore, ...)
}

// Send key and wait for update
func SendKey(t *testing.T, model *tui.Model, key string) *tui.Model {
    msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
    newModel, _ := model.Update(msg)
    return newModel.(*tui.Model)
}

// Verify view content contains text
func AssertViewContains(t *testing.T, model *tui.Model, text string) {
    view := model.View()
    assert.Contains(t, view, text, "View should contain: %s", text)
}

// Verify current view
func AssertCurrentView(t *testing.T, model *tui.Model, expectedView navigation.View) {
    assert.Equal(t, expectedView, model.CurrentView(), "Current view should be: %v", expectedView)
}
```

## Automated Test Execution

### Full Test Suite

```bash
# Run all integration tests
make test-integration

# Run specific scenario
go test -run TestProcessManagement -v ./test/integration/

# Run with race detection
make test-race

# Run with coverage
make test-coverage
```

### CI/CD Integration

Tests run automatically on:
- Every commit to feature branch
- Every PR to main
- Before merging controllers

**Failure Policy**: If any test fails, migration is blocked until fixed.

## Manual Testing Checklist

After each controller migration, manually verify:

- [ ] All keyboard shortcuts work
- [ ] All views display correctly
- [ ] Real-time updates work
- [ ] No error messages in logs
- [ ] No performance regressions
- [ ] Memory usage stable
- [ ] CPU usage normal

## Rollback Strategy

If migration causes issues:

1. **Revert commit**: `git revert <commit-hash>`
2. **Run tests**: Verify functionality restored
3. **Analyze failure**: Review logs and test output
4. **Fix issues**: Update migration approach
5. **Retry**: Attempt migration again with fixes

## Success Criteria

Migration is successful when:
- [x] All integration tests pass
- [x] All manual tests pass
- [x] Performance tests pass
- [x] No functionality regressions
- [x] Code review approved
- [x] Documentation updated
