# Quickstart: Manual Testing Guide

**Purpose**: Manual test scenarios to validate all tab functionality after implementation

## Prerequisites

```bash
# Build the application
make build

# Ensure test project with errors
cd test-fixtures/error-project
npm install
```

## Test Scenario 1: Error Grouping

**Objective**: Verify that multi-line errors are intelligently grouped in the Errors tab

**Steps**:
1. Start Brummer in a project with errors:
   ```bash
   ./brum
   ```

2. Start a process that generates errors:
   ```bash
   # In the Brummer TUI, press '/' to open command palette
   # Type: /run dev
   ```

3. Wait for errors to appear in logs

4. Switch to Errors tab (press '3')

**Expected Results**:
- ✅ Stack traces are grouped as single entries (not scattered across multiple lines)
- ✅ Each error shows complete context (error message + all stack frames)
- ✅ Errors have timestamps and process names
- ✅ Multi-line compilation errors are grouped together
- ✅ Test failures show test name + expected + received + stack trace as one entry

**Validation**:
```
Error entry should look like:
[2025-10-19 14:32:15] dev (PID: 12345)
Error: Cannot find module './config'
    at Function.Module._resolveFilename (node:internal/modules/cjs/loader:1075:15)
    at Function.Module._load (node:internal/modules/cjs/loader:920:27)
    at Module.require (node:internal/modules/cjs/helpers:123:19)
```

## Test Scenario 2: Clear Commands

**Objective**: Verify 'c' key clears content from Errors, Logs, and Web tabs

**Steps**:
1. Generate errors (follow Scenario 1)

2. Switch to Errors tab (press '3')

3. Verify errors are displayed

4. Press 'c' key

**Expected Results**:
- ✅ All errors cleared immediately
- ✅ Display shows "No errors" message
- ✅ Errors do not reappear unless new errors occur

**Steps (Logs tab)**:
5. Switch to Logs tab (press '2')

6. Verify logs are flowing

7. Press 'c' key

**Expected Results**:
- ✅ All logs cleared immediately
- ✅ New logs continue to appear after clearing
- ✅ Scroll position reset to top

**Steps (Web tab)**:
8. Enable proxy (press '/', type `/toggle-proxy`)

9. Generate HTTP traffic (access URLs from URLs tab)

10. Switch to Web tab (press '5')

11. Press 'c' key

**Expected Results**:
- ✅ All proxy events cleared immediately
- ✅ New events captured after clearing

## Test Scenario 3: Clear with Confirmation

**Objective**: Verify 'C' (shift+c) prompts for confirmation before clearing

**Steps**:
1. Generate errors in Errors tab

2. Switch to Errors tab (press '3')

3. Press 'C' (shift+c)

**Expected Results**:
- ✅ Confirmation prompt appears: "Clear all errors? (y/n)"
- ✅ Pressing 'y' clears all errors
- ✅ Pressing 'n' cancels and keeps errors
- ✅ Pressing 'Esc' cancels prompt

**Validation**: Repeat for Logs tab (press '2', then 'C') and Web tab (press '5', then 'C')

## Test Scenario 4: Tab Navigation

**Objective**: Verify keyboard navigation between all tabs works correctly

**Steps**:
1. Start Brummer with running processes

2. Test number key navigation:
   - Press '1' → Processes tab
   - Press '2' → Logs tab
   - Press '3' → Errors tab
   - Press '4' → URLs tab
   - Press '5' → Web tab
   - Press '6' → Proxy tab
   - Press '7' → Settings tab

3. Test Tab key navigation:
   - Press 'Tab' repeatedly → cycles forward through tabs
   - Press 'Shift+Tab' repeatedly → cycles backward through tabs

**Expected Results**:
- ✅ Each number key switches to correct tab instantly (<50ms)
- ✅ Tab/Shift+Tab cycle through tabs in order
- ✅ Tab content is preserved when switching away and back
- ✅ Scroll positions are preserved per tab

## Test Scenario 5: State Preservation

**Objective**: Verify tab state is preserved across tab switches

**Steps**:
1. Go to Logs tab (press '2')

2. Scroll down in logs (use arrow keys or mouse wheel)

3. Apply filter (if implemented) or search (if implemented)

4. Switch to different tab (press '1' for Processes)

5. Switch back to Logs tab (press '2')

**Expected Results**:
- ✅ Scroll position preserved
- ✅ Filter/search query preserved
- ✅ Content not reset

**Validation**: Repeat for all tabs (Errors, Web, Proxy, etc.)

## Test Scenario 6: Real-Time Updates

**Objective**: Verify tab content updates in real-time

**Steps**:
1. Start a long-running process that generates logs

2. Switch to Logs tab (press '2')

3. Observe logs streaming

4. Switch to Errors tab (press '3') - leave it visible

5. Trigger an error in the process (e.g., invalid input)

**Expected Results**:
- ✅ Logs continue to flow in real-time (no lag)
- ✅ New errors appear in Errors tab immediately when they occur
- ✅ No UI freezes or hangs
- ✅ Keyboard remains responsive

## Test Scenario 7: Error Types

**Objective**: Verify different error types are grouped correctly

**Test Cases**:

### Stack Trace
**Input** (from process):
```
Error: Cannot find module './missing'
    at Object.<anonymous> (/app/index.js:5:15)
    at Module._compile (node:internal/modules:123:14)
```

**Expected**: Single grouped error entry

### Compilation Error
**Input**:
```
src/app.ts:15:7 - error TS2304: Cannot find name 'foo'.
15   const foo = bar();
         ~~~
```

**Expected**: Single grouped error entry with file location and code snippet

### Test Failure
**Input**:
```
FAIL src/app.test.ts
  ● App › should render
    expect(received).toBe(expected)
    Expected: "Hello"
    Received: "World"
      at Object.<anonymous> (src/app.test.ts:10:20)
```

**Expected**: Single grouped error entry with test name, expectation, and stack trace

## Test Scenario 8: Performance

**Objective**: Verify performance with high volume data

**Steps**:
1. Generate 1000+ log entries quickly

2. Switch to Logs tab (press '2')

3. Scroll through logs

4. Generate 100+ errors

5. Switch to Errors tab (press '3')

6. Scroll through errors

**Expected Results**:
- ✅ No noticeable lag when switching tabs
- ✅ Smooth scrolling through large datasets
- ✅ Memory usage stays <100MB (check with `top` or `htop`)
- ✅ Keyboard remains responsive (<50ms)

## Test Scenario 9: Race Conditions

**Objective**: Verify no race conditions under stress

**Steps**:
1. Run application with race detector:
   ```bash
   go build -race -o brum-race ./cmd/brum
   ./brum-race
   ```

2. Rapidly switch between tabs (mash number keys)

3. Rapidly clear and regenerate content (mash 'c' key)

4. Check terminal for race condition warnings

**Expected Results**:
- ✅ No race warnings in output
- ✅ No crashes or panics
- ✅ All tabs remain functional

## Test Scenario 10: Edge Cases

**Objective**: Verify edge cases are handled gracefully

### Empty State
**Steps**: View each tab when no data is available

**Expected**: Shows "No {errors/logs/events}" message

### Very Long Error
**Steps**: Generate error with 100+ line stack trace

**Expected**: Error is grouped, scrollable in detail view

### Special Characters
**Steps**: Generate error with unicode, emojis, ANSI codes

**Expected**: Displays correctly or sanitizes safely

### Terminal Resize
**Steps**: Resize terminal while viewing each tab

**Expected**: Content reflows correctly, no crashes

## Pass Criteria

All test scenarios must pass before the feature is considered complete:
- [ ] Test Scenario 1: Error Grouping
- [ ] Test Scenario 2: Clear Commands
- [ ] Test Scenario 3: Clear with Confirmation
- [ ] Test Scenario 4: Tab Navigation
- [ ] Test Scenario 5: State Preservation
- [ ] Test Scenario 6: Real-Time Updates
- [ ] Test Scenario 7: Error Types (all 3 types)
- [ ] Test Scenario 8: Performance
- [ ] Test Scenario 9: Race Conditions
- [ ] Test Scenario 10: Edge Cases (all 4 cases)

## Automated Test Validation

After manual testing, run automated test suite:
```bash
# Contract tests
go test ./test/tui/contracts/... -v

# Integration tests
BRUMMER_BINARY=./brum go test -tags=integration ./test/tui/integration/... -v

# Race detection
make test-race

# Coverage report
go test ./internal/tui/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
# Verify >80% coverage
```

---
*Quickstart Complete - Ready for Implementation*
