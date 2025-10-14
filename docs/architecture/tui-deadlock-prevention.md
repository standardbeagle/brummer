# TUI Deadlock Prevention Guide

## Overview
This document describes critical patterns for preventing UI freezes and deadlocks in the Brummer TUI, particularly focusing on the BubbleTea framework's Update/View cycle and channel operations.

## The Tab 3/4 Lockup Issue

### Problem Description
Users reported that switching to tab 3 (Errors) or tab 4 (URLs) would cause the entire UI to freeze:
- Tab 3 would freeze immediately upon switching
- Tab 4 would display initially but freeze after exactly 1 second
- Other tabs (1, 2, 5, 6, 7) worked normally

### Root Cause
The freezes were caused by **blocking channel operations being called during synchronous rendering**:

1. **Tab 3 (Errors)**: The `View()` method was calling `UpdateErrorsList()` and `UpdateErrorDetailView()` which contain blocking channel reads
2. **Tab 4 (URLs)**: The tick handler (fires every second) was calling `updateLogsView()` which blocks on channels

### Key Insight
The issue was NOT in the event loop itself, but in blocking operations being called from places that must never block:
- The `View()` render method (synchronous, must return quickly)
- Tick handlers (periodic updates, must not block the event loop)

## Golden Rules for Deadlock Prevention

### 1. NEVER Block in View()
```go
// ❌ WRONG - Will cause deadlock
func (m *Model) View() string {
    m.updateDataFromChannel() // Blocks waiting for channel
    return m.render()
}

// ✅ CORRECT - View only reads existing state
func (m *Model) View() string {
    return m.render() // Just render current state
}
```

### 2. NEVER Block in Tick Handlers
```go
// ❌ WRONG - Will freeze after tick interval
case tickMsg:
    m.updateLogsView() // Contains blocking operations

// ✅ CORRECT - Only non-blocking operations
case tickMsg:
    if m.dataAvailable() {
        m.refreshDisplay() // Non-blocking refresh
    }
```

### 3. Updates ONLY in Update()
All state updates that might block should happen in the `Update()` method in response to specific messages:

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case dataAvailableMsg:
        // Safe to block here - Update() is designed for this
        m.updateDataFromChannel()
        return m, nil
    }
}
```

## The Update Channel Pattern

### Purpose
The `updateChan` is used for async operations to notify the UI of changes. It must be used correctly to avoid deadlocks.

### Safe Usage Patterns

#### Pattern 1: Non-blocking Check
```go
// Always safe - never blocks
func (m *Model) waitForUpdatesImmediate() tea.Cmd {
    return func() tea.Msg {
        select {
        case msg := <-m.updateChan:
            return msg
        default:
            return nil
        }
    }
}
```

#### Pattern 2: Timeout Protection
```go
// Safe with reasonable timeout (10-50ms for UI ops)
func (m *Model) waitForUpdatesSafe(timeout time.Duration) tea.Cmd {
    return func() tea.Msg {
        select {
        case msg := <-m.updateChan:
            return msg
        case <-time.After(timeout):
            return nil // Timeout is normal, not an error
        }
    }
}
```

#### Pattern 3: Blocking Only When Guaranteed
```go
// ONLY use when you KNOW a message is coming
func (m *Model) startAsyncOperation() tea.Cmd {
    go func() {
        result := doWork()
        m.updateChan <- resultMsg{result}
    }()
    // Safe because we just started operation that WILL send message
    return m.waitForUpdatesBlocking()
}
```

### Common Pitfalls

| Pitfall | Symptom | Solution |
|---------|---------|----------|
| Calling update methods from View() | UI freezes on render | Move updates to Update() message handlers |
| Blocking in tick handlers | Freezes after tick interval | Use non-blocking checks or skip updates |
| waitForUpdatesBlocking() after view switch | Immediate freeze | Use waitForUpdatesQuick() with timeout |
| Update methods in component Init() | Freeze on component creation | Defer updates to first Update() cycle |

## Testing for Deadlocks

### Manual Testing
1. Switch rapidly between all tabs (1-7)
2. Let each view sit for >1 second (tests tick handlers)
3. Trigger all view-specific actions
4. Monitor for UI freezes

### Code Review Checklist
- [ ] No blocking calls in View() methods
- [ ] No blocking calls in tick handlers
- [ ] All waitForUpdates calls use timeout variants
- [ ] Update methods only called from Update() message handlers

### Debug Logging
Add strategic logging to identify blocking operations:
```go
if msg.Type == tea.KeyRunes && msg.Runes[0] == '3' {
    m.logStore.Add("Debug", "Key 3 received in Update", false)
}
// If this log appears but UI freezes, the problem is after this point
```

## Migration Guide

### Step 1: Identify Blocking Operations
Search for methods that:
- Read from channels without `select`
- Call `waitForUpdatesBlocking()`
- Perform synchronous I/O

### Step 2: Move to Update()
Create message types for triggers:
```go
type refreshDataMsg struct{}

// In Update()
case refreshDataMsg:
    m.updateDataSafely() // Can block here
    return m, nil
```

### Step 3: Use Safe Wait Variants
Replace all `waitForUpdates()` calls:
- View switches → `waitForUpdatesQuick()` (10ms)
- User actions → `waitForUpdatesSafe(50ms)`
- Background ops → `waitForUpdatesSafe(100-500ms)`

## Architecture Recommendations

### 1. Separate Data and Display
- **Data Layer**: Can use blocking operations, runs in Update()
- **Display Layer**: Read-only, never blocks, runs in View()

### 2. Message-Driven Updates
All updates flow through messages:
```
User Input → Message → Update() → State Change → View()
```

### 3. Async Operations Pattern
```go
// Start async operation
func (m *Model) startOperation() tea.Cmd {
    return func() tea.Msg {
        result := blockingOperation()
        return operationCompleteMsg{result}
    }
}

// Handle completion
case operationCompleteMsg:
    m.state = msg.result
    return m, nil
```

## Summary

The key to preventing TUI deadlocks is understanding that **View() must never block**. All potentially blocking operations must be:
1. Moved to the Update() method
2. Protected with timeouts
3. Triggered by messages, not direct calls

Following these patterns ensures a responsive UI that never freezes, even under heavy load or with slow operations.