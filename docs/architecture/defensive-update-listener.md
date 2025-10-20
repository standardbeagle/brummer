# Defensive Update Listener Pattern

## Problem: Fragile Manual Update Scheduling

### The Old (Fragile) Pattern

Previously, every message handler that modified state needed to remember to call:

```go
case messages.ProxyRequestMsg:
    // Update state
    m.masterData.WebViewState = m.masterData.WebViewState.AddRequest(requestInfo)

    // ❌ EASY TO FORGET! If missing, UI doesn't update until next keypress
    cmds = append(cmds, m.waitForUpdates())
```

**Problems with this approach:**
1. **Error-prone**: Easy to forget in new handlers
2. **Hard to debug**: Events arrive but UI doesn't update, appears broken
3. **Requires keypress to "pump"**: Events queue up, only show one per keypress
4. **Not defensive**: Relies on developer memory, not system design

### Example Bug

When the `ProxyRequestMsg` handler was first implemented without `waitForUpdates()`:

```go
case messages.ProxyRequestMsg:
    m.masterData.WebViewState = m.masterData.WebViewState.AddRequest(requestInfo)
    // Missing: cmds = append(cmds, m.waitForUpdates())
```

**Result**: Proxy events were received and queued in `updateChan`, but the UI only showed one request each time the user pressed a key (like 'q'). The events were there, just not being processed!

## Solution: Persistent Auto-Scheduling Listener

### The New (Defensive) Pattern

**Start once in Init(), forget everywhere else:**

```go
func (m *Model) Init() tea.Cmd {
    cmds := []tea.Cmd{
        textinput.Blink,
        m.tickCmd(),
        m.startUpdateListener(), // ✅ Start persistent listener
    }
    return tea.Batch(cmds...)
}
```

**Listener automatically re-schedules itself:**

```go
func (m *Model) startUpdateListener() tea.Cmd {
    return func() tea.Msg {
        select {
        case msg := <-m.updateChan:
            // Wrap message for automatic re-scheduling
            return UpdateChanMsg{Inner: msg}
        case <-time.After(100 * time.Millisecond):
            // Timeout - keep listening
            return UpdateChanMsg{Inner: nil}
        }
    }
}
```

**Update() handles wrapped messages and re-schedules:**

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle updateChan messages automatically
    if updateMsg, ok := msg.(UpdateChanMsg); ok {
        // ✅ Re-schedule listener FIRST - always listening!
        cmds = append(cmds, m.startUpdateListener())

        if updateMsg.Inner != nil {
            // Process the unwrapped message
            newModel, cmd := m.Update(updateMsg.Inner)
            // ... return updated model
        }
        return m, tea.Batch(cmds...)
    }

    // ... rest of Update() logic
}
```

**Message handlers now simplified:**

```go
case messages.ProxyRequestMsg:
    // Update state
    m.masterData.WebViewState = m.masterData.WebViewState.AddRequest(requestInfo)
    // ✅ NO NEED for waitForUpdates() - automatic!
```

## Benefits

### 1. **Impossible to Forget**
- Start listener once in `Init()`, it runs forever
- New message handlers don't need to know about update scheduling
- Can't accidentally break UI updates

### 2. **Real-Time Updates**
- Events processed immediately as they arrive
- No "one event per keypress" behavior
- Smooth, responsive UI

### 3. **Easier to Maintain**
- Less code in message handlers (no `waitForUpdates()` calls)
- Clear separation: listener handles scheduling, handlers handle logic
- New developers can't introduce update bugs

### 4. **Self-Healing**
- Timeout ensures listener restarts even if no messages
- No chance of listener stopping unexpectedly
- Graceful handling of empty updateChan

## Architecture Flow

```
┌─────────────────────────────────────────────────────────┐
│                    Init() - Startup                      │
│  ┌────────────────────────────────────────────────────┐ │
│  │ 1. Start persistent listener once                  │ │
│  │    m.startUpdateListener()                         │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│           startUpdateListener() - Forever Loop          │
│  ┌────────────────────────────────────────────────────┐ │
│  │ 2. Wait for updateChan with timeout                │ │
│  │    select {                                         │ │
│  │      case msg := <-m.updateChan:                   │ │
│  │        return UpdateChanMsg{Inner: msg}            │ │
│  │      case <-time.After(100ms):                     │ │
│  │        return UpdateChanMsg{Inner: nil}            │ │
│  │    }                                                │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│              Update() - Message Handler                 │
│  ┌────────────────────────────────────────────────────┐ │
│  │ 3. Receive UpdateChanMsg                           │ │
│  │    if updateMsg, ok := msg.(UpdateChanMsg)         │ │
│  │                                                     │ │
│  │ 4. Re-schedule listener FIRST                      │ │
│  │    cmds = append(cmds, m.startUpdateListener())    │ │
│  │                                                     │ │
│  │ 5. Process wrapped message (if not nil)            │ │
│  │    m.Update(updateMsg.Inner)                       │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                         │
                         │ (listener re-scheduled)
                         ▼
              ┌──────────────────────┐
              │  Back to step 2!     │
              │  Loop continues...   │
              └──────────────────────┘
```

## Event Sources

The defensive listener handles ALL events sent to `updateChan` from:

1. **EventBus subscriptions** (ProxyRequest, ProcessStarted, etc.)
2. **AI Coder sessions** (PTY output, state changes)
3. **Background operations** (async tasks, timers)
4. **Service callbacks** (process monitoring, log capture)

All of these now automatically trigger UI updates without manual intervention!

## Implementation Checklist

When implementing this pattern in other TUI applications:

- [ ] Define `UpdateChanMsg` wrapper type
- [ ] Implement `startUpdateListener()` with timeout
- [ ] Add listener to `Init()` commands
- [ ] Add `UpdateChanMsg` handler early in `Update()`
- [ ] Handler re-schedules listener FIRST, then processes message
- [ ] Remove all manual `waitForUpdates()` calls from handlers
- [ ] Test real-time event flow (should update immediately)

## Migration Notes

### Removing Legacy waitForUpdates() Calls

After implementing the defensive listener, all `waitForUpdates()` calls become obsolete:

```bash
# Find all calls (should only be in comments/docs after migration)
grep -n "waitForUpdates()" internal/tui/model.go

# Remove all append calls (automated cleanup)
sed -i '/^\t*cmds = append(cmds, m\.waitForUpdates())$/d' internal/tui/model.go
```

### Keeping Legacy Functions (For Now)

The old `waitForUpdates()`, `waitForUpdatesSafe()`, etc. functions are kept for:
1. **Documentation**: Show what the pattern replaced
2. **Gradual migration**: Allow incremental refactoring if needed
3. **Testing**: Can still use for specific test scenarios

But they're **no longer called** from the main Update() loop.

## Testing

To verify the defensive listener works:

1. **Start application**: Listener should start automatically
2. **Trigger updateChan events**: (proxy requests, process events, etc.)
3. **Observe UI**: Should update in real-time without keypresses
4. **Check logs**: No "UI frozen" or "events queued" issues

### Example Test

```bash
# Start server that generates traffic
./brum server

# Switch to web tab (tab 5)
# Should see requests appearing in real-time as they happen
# NO need to press keys to "pump" the display
```

## Key Takeaways

1. **Start once, run forever**: Listener initialized in Init(), never stopped
2. **Automatic re-scheduling**: Each message automatically schedules next listen
3. **Timeout protection**: 100ms timeout prevents blocking, keeps loop alive
4. **Zero manual calls**: Message handlers don't need to know about listener
5. **Defensive by design**: Can't forget, can't break, self-healing

This pattern makes UI updates **automatic, reliable, and impossible to break by accident**.
