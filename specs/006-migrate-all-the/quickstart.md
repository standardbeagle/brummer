# Quick Start Guide: Controller-to-Component Migration

**Feature**: 006-migrate-all-the
**Created**: 2025-10-11
**Purpose**: Validation scenarios and testing steps for migrated component architecture

## Overview

This quick start guide provides step-by-step validation scenarios to verify that the controller-to-component migration has been completed successfully while preserving all existing functionality.

## Pre-Migration Validation

### Step 1: Document Current Controller Behavior
Before migration, capture current behavior for comparison:

```bash
# Build and run current version
make build
./brum

# Test each controller behavior:
# 1. Input handling: Try keyboard shortcuts, number keys, tab navigation
# 2. View switching: Switch between views using number keys (1-5)
# 3. System messages: Look for system panel and messages
# 4. Notifications: Trigger actions that show toast notifications
# 5. File browser: Open any feature that uses file selection dialogs
```

### Step 2: Document Performance Baselines
```bash
# Measure current performance
time ./brum --no-tui  # Startup time
# Test input responsiveness in TUI mode
# Test view switching speed
```

## Post-Migration Validation Scenarios

### Scenario 1: Input Handling Validation

**Objective**: Verify all keyboard input works identically after migration

**Steps**:
1. Start Brummer: `./brum`
2. Test global keyboard shortcuts:
   - Press `q` (should show quit confirmation or quit)
   - Press `?` (should show help overlay)
   - Press `/` (should open command palette)
3. Test number key navigation:
   - Press `1` (should switch to Processes view)
   - Press `2` (should switch to Logs view)
   - Press `3` (should switch to Errors view)
   - Press `4` (should switch to AI Coders view)
   - Press `5` (should switch to Settings view)
4. Test view cycling:
   - Press `Tab` (should cycle to next view)
   - Press `Shift+Tab` (should cycle to previous view)
5. Test arrow keys:
   - Press `Up/Down` (should navigate in lists)
   - Press `Left/Right` (should navigate horizontally where applicable)

**Expected Results**:
- All keyboard shortcuts work exactly as before migration
- No regression in input responsiveness
- Key conflicts resolved properly

**Validation Commands**:
```bash
# Test input performance
# (Manual testing in TUI required)
# Press keys rapidly and verify responsive behavior
```

### Scenario 2: View Navigation Validation

**Objective**: Verify view switching and navigation state management

**Steps**:
1. Start Brummer and observe initial view
2. Switch views using number keys (1-5):
   - Each key should immediately switch to corresponding view
   - View content should load correctly
   - View state should be preserved when switching back
3. Test view cycling:
   - Press `Tab` multiple times to cycle through all views
   - Verify order is: Processes → Logs → Errors → AI Coders → Settings → Processes
4. Test view history:
   - Switch between several views
   - Verify view transitions are smooth
   - Check that view-specific state is maintained

**Expected Results**:
- View switching is immediate (<16ms)
- View order matches predefined sequence
- No visual glitches or artifacts during transitions
- View state preservation works correctly

### Scenario 3: System Messages Validation

**Objective**: Verify system message display and management

**Steps**:
1. Trigger system messages:
   - Start a process that generates logs
   - Perform actions that create system notifications
   - Look for system panel indicators
2. Test system panel:
   - Look for unread message indicators
   - Access system messages (if visible in current UI)
   - Verify message ordering and formatting
3. Test message lifecycle:
   - Generate multiple messages of different levels (info, warning, error)
   - Verify messages appear in correct order
   - Check that message counts update correctly

**Expected Results**:
- System messages appear correctly
- Unread indicators work as before
- Message ordering is maintained
- No message loss or duplication

### Scenario 4: Notification Validation

**Objective**: Verify toast notification behavior

**Steps**:
1. Trigger notifications:
   - Perform actions that show success notifications
   - Trigger warnings or errors that show toast notifications
   - Test multiple notifications appearing simultaneously
2. Test notification lifecycle:
   - Verify notifications appear for correct duration
   - Test notification dismissal
   - Check notification positioning and styling
3. Test notification limits:
   - Generate many notifications rapidly
   - Verify maximum visible notification limit

**Expected Results**:
- Notifications appear with correct timing and positioning
- Auto-dismissal works as expected
- Manual dismissal functions correctly
- Notification overflow is handled gracefully

### Scenario 5: File Browser Validation

**Objective**: Verify file selection dialog functionality

**Steps**:
1. Trigger file browser:
   - Access any feature that opens file selection
   - Test file browser opening and closing
2. Test navigation:
   - Navigate directories using arrow keys
   - Test parent directory navigation
   - Test file selection and confirmation
3. Test file filtering (if applicable):
   - Test file type filters
   - Test search functionality (if present)

**Expected Results**:
- File browser opens and closes smoothly
- Directory navigation works correctly
- File selection returns correct paths
- No file system access errors

## Performance Validation

### Response Time Testing

**Input Response Time**:
```bash
# Manual testing: Press keys rapidly and verify <50ms response
# Target: <50ms for key processing
```

**View Switching Performance**:
```bash
# Manual testing: Switch views rapidly and verify smooth transitions
# Target: <16ms for view rendering (60fps)
```

**Message Processing**:
```bash
# Generate system events and verify timely processing
# Target: <5ms for message processing
```

### Memory Usage Validation

```bash
# Monitor memory usage during extended use
./brum --no-tui &  # Run in background
BRUM_PID=$!

# Monitor memory over time
while true; do
    ps -p $BRUM_PID -o pid,ppid,cmd,%mem,%cpu
    sleep 5
done

# Test for memory leaks by using all features extensively
```

## Integration Testing

### Test Script for Automated Validation

Create a test script to validate core functionality:

```bash
#!/bin/bash
# test-migration.sh

echo "=== Controller-to-Component Migration Validation ==="

# Test 1: Basic startup
echo "1. Testing startup..."
timeout 5s ./brum --no-tui > startup_test.log 2>&1
if [ $? -eq 0 ]; then
    echo "   ✅ Startup successful"
else
    echo "   ❌ Startup failed"
    cat startup_test.log
    exit 1
fi

# Test 2: Service initialization
echo "2. Testing service initialization..."
# This would require integration tests to verify service setup

# Test 3: Event flow
echo "3. Testing event flow..."
# This would require testing EventBus communication

# Test 4: Performance benchmarks
echo "4. Testing performance..."
# This would require performance measurement tools

echo "=== Validation Complete ==="
```

## Regression Testing

### Feature Comparison Matrix

| Feature | Pre-Migration Behavior | Post-Migration Behavior | Status |
|---------|----------------------|------------------------|---------|
| Keyboard shortcuts | Working | ✅ Should work | PASS/FAIL |
| View switching | Working | ✅ Should work | PASS/FAIL |
| System messages | Working | ✅ Should work | PASS/FAIL |
| Notifications | Working | ✅ Should work | PASS/FAIL |
| File browser | Working | ✅ Should work | PASS/FAIL |
| Input responsiveness | <50ms | ✅ Should be <50ms | PASS/FAIL |
| View switching speed | <16ms | ✅ Should be <16ms | PASS/FAIL |

### Acceptance Criteria Validation

**FR-001**: Keyboard shortcuts preserved ✅
- [ ] All global shortcuts work
- [ ] Context-sensitive shortcuts work
- [ ] No shortcut conflicts introduced

**FR-002**: Navigation behavior preserved ✅
- [ ] Number key navigation works
- [ ] View cycling works
- [ ] View order maintained

**FR-003**: System messages preserved ✅
- [ ] Message display works
- [ ] Unread indicators work
- [ ] Message ordering preserved

**FR-004**: Notification behavior preserved ✅
- [ ] Toast notifications appear
- [ ] Duration timing works
- [ ] Positioning correct

**FR-005**: File browser preserved ✅
- [ ] Dialog opens/closes
- [ ] Directory navigation works
- [ ] File selection works

## Troubleshooting Guide

### Common Issues and Solutions

**Issue**: Keyboard input not responding
- **Check**: InputComponent initialization
- **Solution**: Verify event subscriptions in ServiceBridge

**Issue**: View switching not working
- **Check**: NavigationComponent state management
- **Solution**: Verify NavigationService integration

**Issue**: System messages not appearing
- **Check**: SystemComponent message handling
- **Solution**: Verify SystemService event publishing

**Issue**: Notifications not showing
- **Check**: NotificationsComponent lifecycle
- **Solution**: Verify NotificationsService timing logic

**Issue**: File browser not opening
- **Check**: FileBrowserComponent dialog management
- **Solution**: Verify FileBrowserService path handling

### Debug Commands

```bash
# Enable debug logging
export RUM_LOG_LEVEL=debug
./brum

# Check service status
./brum --settings | grep -A 10 "services"

# Test individual components
# (Would require component-specific test commands)
```

## Success Criteria

### Migration Success Indicators

✅ **Functional Parity**: All existing functionality works identically
✅ **Performance Targets**: Input <50ms, rendering <16ms
✅ **No Regressions**: No feature loss or degradation
✅ **Clean Architecture**: All controller references removed
✅ **Test Coverage**: All services and components tested

### Validation Completion Checklist

- [ ] All keyboard shortcuts tested and working
- [ ] View navigation tested and working
- [ ] System messages tested and working
- [ ] Notifications tested and working
- [ ] File browser tested and working
- [ ] Performance targets met
- [ ] No memory leaks detected
- [ ] Integration tests passing
- [ ] Manual testing complete
- [ ] User acceptance validated

## Next Steps

After successful validation:

1. **Documentation Update**: Update user documentation to reflect new architecture
2. **Performance Monitoring**: Set up ongoing performance monitoring
3. **Cleanup**: Remove any remaining controller references
4. **Testing**: Add comprehensive test coverage for new components
5. **Deployment**: Deploy migrated version to production

This quick start guide ensures that the controller-to-component migration maintains full functional parity while improving code architecture and maintainability.