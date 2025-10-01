# Component System Migration Master Plan

## Branching Strategy

### Base Commit
**Commit**: `02e0611` - "feat(search): add comprehensive LCI semantic annotations"
**Date**: Wed Aug 13 22:20:25 2025
**Why**: Last stable commit before component system work began

### Branch Creation
```bash
git checkout 02e0611
git checkout -b feature/service-bridge-architecture
```

## Lessons Learned from First Attempt

### Critical Mistakes
1. **No ServiceBridge**: Built components without a service-to-UI adapter layer
2. **Mixed Update Patterns**: Callbacks + polling + EventBus with no unification
3. **No Message Flow Design**: Components accessing services directly
4. **Tight Coupling**: LogStore, ProcessManager, etc. directly accessed by UI
5. **Race Conditions**: Components modifying shared state from goroutines
6. **Lost Updates**: No notification mechanism for log additions, URL detections
7. **Polling Fallbacks**: 1-second polling to catch missed events
8. **Testing Gaps**: Couldn't test components without real services

### What Worked
1. **BubbleTea Model**: Message-based component pattern is correct
2. **Lock-Free Architecture**: Atomic operations for performance
3. **EventBus Design**: Good pub/sub mechanism, just not integrated with UI
4. **Component Interfaces**: Clean separation of concerns

### What We'll Do Differently
1. **ServiceBridge First**: Build adapter layer before any components
2. **Unified Messages**: All service events become tea.Msg
3. **Observer Pattern**: Services emit events, don't rely on polling
4. **Test-Driven**: Mock messages for component testing
5. **Incremental Migration**: Keep old model working while building new system
6. **Clear Architecture**: Document message flow before coding

## Architecture Overview

```
┌────────────────────────────────────────────────────────────────┐
│                     BubbleTea Program                          │
│                  (Message Loop Orchestrator)                   │
└────────────────────────┬───────────────────────────────────────┘
                         │
              ┌──────────▼──────────┐
              │  ServiceBridge      │ ◄─── NEW: This is the key!
              │  (Event Adapter)    │
              │                     │
              │  Converts:          │
              │  • Callbacks → Msg  │
              │  • EventBus → Msg   │
              │  • Store Obs → Msg  │
              └──────────┬──────────┘
                         │
                 ┌───────▼────────┐
                 │ IntegratedShell│
                 │ (Msg Router)   │
                 └───────┬────────┘
                         │
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
  ┌─────────┐      ┌─────────┐     ┌──────────┐
  │Process  │      │Logs     │     │URLs      │
  │Component│      │Component│     │Component │
  └─────────┘      └─────────┘     └──────────┘
```

## Implementation Phases

### Phase 0: Foundation (Week 1, Days 1-2)

#### Goals
- Set up proper branching
- Create message type definitions
- Establish testing infrastructure

#### Tasks

**1. Create Branch and Setup**
```bash
git checkout 02e0611
git checkout -b feature/service-bridge-architecture
git push -u origin feature/service-bridge-architecture
```

**2. Create Message Type Definitions**
File: `internal/tui/messages/service_events.go`

```go
package messages

import (
    "time"
    "github.com/standardbeagle/brummer/internal/process"
    "github.com/standardbeagle/brummer/internal/logs"
)

// Process lifecycle events
type ProcessStartedMsg struct {
    ProcessID string
    Name      string
    Script    string
    StartTime time.Time
}

type ProcessStoppedMsg struct {
    ProcessID string
    Name      string
    ExitCode  int
    EndTime   time.Time
}

type ProcessOutputMsg struct {
    ProcessID string
    Line      string
    IsError   bool
    Timestamp time.Time
}

type ProcessStateChangedMsg struct {
    ProcessID string
    OldState  process.ProcessStatus
    NewState  process.ProcessStatus
}

// Log events
type LogAddedMsg struct {
    Entry *logs.LogEntry
}

type LogsClearedMsg struct {
    ProcessID string // Empty string = all cleared
}

// URL detection events
type URLDetectedMsg struct {
    URL         string
    ProcessID   string
    Type        string // "http", "https", "ws", "wss"
    Timestamp   time.Time
    HasProxy    bool
    ProxyPort   int
    ProxyURL    string
    ProxyLabel  string
}

// Error events
type ErrorOccurredMsg struct {
    ProcessID   string
    ProcessName string
    Error       string
    Timestamp   time.Time
}

// Service health events
type ServiceHealthMsg struct {
    Service string // "process", "proxy", "mcp"
    Healthy bool
    Message string
}
```

**3. Create Testing Infrastructure**
File: `internal/tui/messages/service_events_test.go`

```go
package messages

import (
    "testing"
    "time"
)

func TestProcessStartedMsg(t *testing.T) {
    msg := ProcessStartedMsg{
        ProcessID: "test-123",
        Name:      "dev",
        Script:    "npm run dev",
        StartTime: time.Now(),
    }

    if msg.ProcessID != "test-123" {
        t.Errorf("Expected ProcessID test-123, got %s", msg.ProcessID)
    }
}

// ... tests for all message types
```

**Deliverables**:
- [ ] Branch created from 02e0611
- [ ] `internal/tui/messages/service_events.go` with all message types
- [ ] Comprehensive tests for message types
- [ ] Documentation: MESSAGE_TYPES.md explaining each message

---

### Phase 1: ServiceBridge Core (Week 1, Days 3-5)

#### Goals
- Build ServiceBridge adapter layer
- Integrate with BubbleTea program
- Add observer pattern to LogStore

#### Tasks

**1. Create ServiceBridge**
File: `internal/tui/bridge/service_bridge.go`

```go
package bridge

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/standardbeagle/brummer/internal/process"
    "github.com/standardbeagle/brummer/internal/logs"
    "github.com/standardbeagle/brummer/internal/proxy"
    "github.com/standardbeagle/brummer/pkg/events"
    "github.com/standardbeagle/brummer/internal/tui/messages"
)

type ServiceBridge struct {
    processMgr  *process.Manager
    logStore    *logs.Store
    proxyServer *proxy.Server
    eventBus    *events.EventBus
    program     *tea.Program

    // Cleanup
    subscriptions []func()
}

func NewServiceBridge(
    processMgr *process.Manager,
    logStore *logs.Store,
    proxyServer *proxy.Server,
    eventBus *events.EventBus,
) *ServiceBridge {
    return &ServiceBridge{
        processMgr:    processMgr,
        logStore:      logStore,
        proxyServer:   proxyServer,
        eventBus:      eventBus,
        subscriptions: []func(){},
    }
}

func (b *ServiceBridge) Start(program *tea.Program) {
    b.program = program

    // Subscribe to process manager callbacks
    b.subscribeToProcessManager()

    // Subscribe to log store events
    b.subscribeToLogStore()

    // Subscribe to event bus
    b.subscribeToEventBus()

    // Subscribe to proxy server events
    b.subscribeToProxyServer()
}

func (b *ServiceBridge) subscribeToProcessManager() {
    // Convert process log callbacks to messages
    b.processMgr.AddLogCallback(func(processID, line string, isError bool) {
        if b.program != nil {
            b.program.Send(messages.ProcessOutputMsg{
                ProcessID: processID,
                Line:      line,
                IsError:   isError,
                Timestamp: time.Now(),
            })
        }
    })
}

func (b *ServiceBridge) subscribeToLogStore() {
    // Observer pattern for log additions
    b.logStore.OnAdd(func(entry *logs.LogEntry) {
        if b.program != nil {
            b.program.Send(messages.LogAddedMsg{
                Entry: entry,
            })
        }
    })
}

func (b *ServiceBridge) subscribeToEventBus() {
    // Process state changes
    unsub := b.eventBus.Subscribe(events.ProcessStateChange, func(evt events.Event) {
        if b.program != nil {
            b.program.Send(messages.ProcessStateChangedMsg{
                ProcessID: evt.ProcessID,
                OldState:  evt.Data["old_state"].(process.ProcessStatus),
                NewState:  evt.Data["new_state"].(process.ProcessStatus),
            })
        }
    })
    b.subscriptions = append(b.subscriptions, unsub)

    // URL detections
    unsub = b.eventBus.Subscribe(events.URLDetected, func(evt events.Event) {
        if b.program != nil {
            b.program.Send(messages.URLDetectedMsg{
                URL:       evt.Data["url"].(string),
                ProcessID: evt.ProcessID,
                Type:      evt.Data["type"].(string),
                Timestamp: time.Now(),
            })
        }
    })
    b.subscriptions = append(b.subscriptions, unsub)
}

func (b *ServiceBridge) subscribeToProxyServer() {
    // Proxy server events (if needed)
}

func (b *ServiceBridge) Stop() {
    for _, unsub := range b.subscriptions {
        unsub()
    }
    b.subscriptions = nil
    b.program = nil
}
```

**2. Add Observer Pattern to LogStore**
File: `internal/logs/store.go` (modify existing)

```go
// Add to Store struct
type LogObserver func(entry *LogEntry)

type Store struct {
    // ... existing fields ...

    observers   []LogObserver
    observersMu sync.RWMutex
}

// Add observer registration
func (s *Store) OnAdd(observer LogObserver) func() {
    s.observersMu.Lock()
    defer s.observersMu.Unlock()

    s.observers = append(s.observers, observer)

    // Return unsubscribe function
    return func() {
        s.observersMu.Lock()
        defer s.observersMu.Unlock()
        for i, obs := range s.observers {
            // Compare function pointers
            if &obs == &observer {
                s.observers = append(s.observers[:i], s.observers[i+1:]...)
                break
            }
        }
    }
}

// Modify Add() to notify observers
func (s *Store) Add(processID, processName, content string, isError bool) *LogEntry {
    // ... existing add logic ...

    entry := &LogEntry{
        ID:          fmt.Sprintf("%s-%d", processID, time.Now().UnixNano()),
        ProcessID:   processID,
        ProcessName: processName,
        Timestamp:   time.Now(),
        Content:     content,
        IsError:     isError,
    }

    // ... store the entry ...

    // Notify observers
    s.observersMu.RLock()
    observers := s.observers
    s.observersMu.RUnlock()

    for _, obs := range observers {
        obs(entry)
    }

    return entry
}
```

**3. Testing**
File: `internal/tui/bridge/service_bridge_test.go`

```go
package bridge

import (
    "testing"
    tea "github.com/charmbracelet/bubbletea"
)

func TestServiceBridge_ProcessOutput(t *testing.T) {
    // Create mock services
    processMgr := process.NewManager("/tmp", nil, false)
    logStore := logs.NewStore(1000, nil)

    // Create bridge
    bridge := NewServiceBridge(processMgr, logStore, nil, nil)

    // Capture messages
    received := []tea.Msg{}
    mockProgram := &MockProgram{
        SendFunc: func(msg tea.Msg) {
            received = append(received, msg)
        },
    }

    bridge.Start(mockProgram)

    // Trigger process output
    processMgr.EmitLog("test-proc", "Hello World", false)

    // Verify message was sent
    if len(received) != 1 {
        t.Fatalf("Expected 1 message, got %d", len(received))
    }

    if msg, ok := received[0].(messages.ProcessOutputMsg); ok {
        if msg.Line != "Hello World" {
            t.Errorf("Expected line 'Hello World', got '%s'", msg.Line)
        }
    } else {
        t.Error("Expected ProcessOutputMsg")
    }
}
```

**Deliverables**:
- [ ] ServiceBridge implementation
- [ ] Observer pattern in LogStore
- [ ] Comprehensive ServiceBridge tests
- [ ] Integration test: service event → message → component
- [ ] Documentation: SERVICE_BRIDGE.md

---

### Phase 2: Base Component Infrastructure (Week 2, Days 1-3)

#### Goals
- Create reusable component base
- Build IntegratedShell message router
- Implement first component (Logs)

#### Tasks

**1. Base Component**
File: `internal/tui/components/base.go`

```go
package components

import tea "github.com/charmbracelet/bubbletea"

type Component interface {
    tea.Model
    SetSize(width, height int)
    Focus() tea.Cmd
    Blur() tea.Cmd
}

type BaseComponent struct {
    width   int
    height  int
    focused bool
    visible bool
}

func (b *BaseComponent) SetSize(width, height int) {
    b.width, b.height = width, height
}

func (b *BaseComponent) Focus() tea.Cmd {
    b.focused = true
    return nil
}

func (b *BaseComponent) Blur() tea.Cmd {
    b.focused = false
    return nil
}

func (b *BaseComponent) Width() int  { return b.width }
func (b *BaseComponent) Height() int { return b.height }
func (b *BaseComponent) IsFocused() bool { return b.focused }
func (b *BaseComponent) IsVisible() bool { return b.visible }
```

**2. IntegratedShell Router**
File: `internal/tui/components/integrated_shell.go`

```go
package components

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/standardbeagle/brummer/internal/tui/messages"
)

type IntegratedShell struct {
    width  int
    height int

    // Components
    logsView Component

    // State
    currentView string
}

func NewIntegratedShell() *IntegratedShell {
    return &IntegratedShell{
        currentView: "logs",
    }
}

func (s *IntegratedShell) Init() tea.Cmd {
    return nil
}

func (s *IntegratedShell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        s.width = msg.Width
        s.height = msg.Height
        if s.logsView != nil {
            s.logsView.SetSize(msg.Width, msg.Height)
        }

    case messages.LogAddedMsg:
        // Route to logs component
        if s.logsView != nil {
            _, cmd := s.logsView.Update(msg)
            cmds = append(cmds, cmd)
        }

    case messages.ProcessOutputMsg:
        // Already handled by LogStore observer
        // This is redundant but shows the pattern
    }

    return s, tea.Batch(cmds...)
}

func (s *IntegratedShell) View() string {
    if s.logsView != nil {
        return s.logsView.View()
    }
    return "No view"
}
```

**3. LogsViewComponent**
File: `internal/tui/components/logs_view.go`

```go
package components

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/standardbeagle/brummer/internal/tui/messages"
    "github.com/standardbeagle/brummer/internal/logs"
)

type LogsViewComponent struct {
    BaseComponent

    viewport viewport.Model
    logs     []string
    logStore *logs.Store  // Read-only access for initial load
}

func NewLogsViewComponent(logStore *logs.Store) *LogsViewComponent {
    c := &LogsViewComponent{
        logStore: logStore,
        viewport: viewport.New(80, 24),
    }
    c.visible = true

    // Load existing logs
    entries := logStore.GetAll()
    for _, entry := range entries {
        c.logs = append(c.logs, c.formatEntry(&entry))
    }
    c.updateViewport()

    return c
}

func (c *LogsViewComponent) Init() tea.Cmd {
    return nil
}

func (c *LogsViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case messages.LogAddedMsg:
        // NEW LOG! Add to display
        c.logs = append(c.logs, c.formatEntry(msg.Entry))
        c.updateViewport()
        return c, nil

    case tea.WindowSizeMsg:
        c.SetSize(msg.Width, msg.Height)
        c.viewport.Width = msg.Width
        c.viewport.Height = msg.Height
        return c, nil
    }

    return c, nil
}

func (c *LogsViewComponent) View() string {
    return c.viewport.View()
}

func (c *LogsViewComponent) formatEntry(entry *logs.LogEntry) string {
    return fmt.Sprintf("[%s] %s: %s",
        entry.Timestamp.Format("15:04:05"),
        entry.ProcessName,
        entry.Content)
}

func (c *LogsViewComponent) updateViewport() {
    content := strings.Join(c.logs, "\n")
    c.viewport.SetContent(content)
    c.viewport.GotoBottom()
}
```

**Deliverables**:
- [ ] BaseComponent interface and implementation
- [ ] IntegratedShell message router
- [ ] LogsViewComponent with message handling
- [ ] Tests for all components
- [ ] Integration test: LogStore.Add() → message → LogsView updates
- [ ] Documentation: COMPONENT_ARCHITECTURE.md

---

### Phase 3: Main Integration (Week 2, Days 4-5)

#### Goals
- Wire ServiceBridge into main.go
- Enable dual-mode operation (old model + new components)
- Verify end-to-end message flow

#### Tasks

**1. Update main.go**
File: `cmd/brum/main.go`

```go
// Add feature flag
useComponents := os.Getenv("BRUMMER_USE_COMPONENTS") == "true"

if useComponents {
    // NEW COMPONENT SYSTEM

    // Create log store with observer support
    logStore := logs.NewStore(10000, nil)

    // Create IntegratedShell
    shell := components.NewIntegratedShell()
    logsView := components.NewLogsViewComponent(logStore)
    shell.SetLogsView(logsView)

    // Create ServiceBridge
    bridge := bridge.NewServiceBridge(
        processMgr,
        logStore,
        proxyServer,
        eventBus,
    )

    // Create BubbleTea program
    p := tea.NewProgram(shell, tea.WithAltScreen())

    // Start ServiceBridge with program reference
    bridge.Start(p)

    // Run program
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }

    // Cleanup
    bridge.Stop()

} else {
    // OLD MONOLITHIC MODEL
    model := tui.NewModel(processMgr, eventBus, proxyServer)
    p := tea.NewProgram(model, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

**2. Integration Testing**
File: `test/integration/component_flow_test.go`

```go
func TestEndToEndMessageFlow(t *testing.T) {
    // Setup services
    processMgr := process.NewManager("/tmp", events.NewEventBus(), false)
    logStore := logs.NewStore(1000, nil)

    // Create component
    logsView := components.NewLogsViewComponent(logStore)

    // Create shell
    shell := components.NewIntegratedShell()
    shell.SetLogsView(logsView)

    // Create bridge
    bridge := bridge.NewServiceBridge(processMgr, logStore, nil, nil)

    // Start program (mock)
    mockProgram := &MockProgram{}
    bridge.Start(mockProgram)

    // Add log entry
    logStore.Add("test-proc", "TestProc", "Hello World", false)

    // Verify message was sent
    messages := mockProgram.GetMessages()
    if len(messages) != 1 {
        t.Fatalf("Expected 1 message, got %d", len(messages))
    }

    // Send message to shell
    shell.Update(messages[0])

    // Verify logs view was updated
    view := logsView.View()
    if !strings.Contains(view, "Hello World") {
        t.Error("LogsView did not update with new log")
    }
}
```

**Deliverables**:
- [ ] ServiceBridge integrated into main.go
- [ ] Dual-mode operation (env var toggle)
- [ ] End-to-end integration test
- [ ] Smoke test: Start app, add log, verify it appears
- [ ] Documentation: INTEGRATION_GUIDE.md

---

### Phase 4: Remaining Components (Week 3-4)

#### Goals
- Implement ProcessViewComponent
- Implement URLsViewComponent
- Implement AIPTYComponent (properly this time!)
- Add tab navigation

#### Process Component
File: `internal/tui/components/process_view.go`

```go
type ProcessViewComponent struct {
    BaseComponent

    processes map[string]ProcessInfo
}

func (c *ProcessViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case messages.ProcessStartedMsg:
        c.processes[msg.ProcessID] = ProcessInfo{
            Name:      msg.Name,
            Status:    "running",
            StartTime: msg.StartTime,
        }
        return c, nil

    case messages.ProcessStoppedMsg:
        if info, ok := c.processes[msg.ProcessID]; ok {
            info.Status = "stopped"
            info.ExitCode = msg.ExitCode
            c.processes[msg.ProcessID] = info
        }
        return c, nil
    }

    return c, nil
}
```

#### URLs Component
Similar pattern - receives `URLDetectedMsg`, updates display.

#### AI PTY Component
**KEY LESSON**: All PTY output must flow through ServiceBridge as messages!

```go
// In ServiceBridge
func (b *ServiceBridge) subscribeToAIPTY(session *aicoder.PTYSession) {
    go func() {
        for data := range session.OutputChan {
            if b.program != nil {
                b.program.Send(messages.AIPTYOutputMsg{
                    SessionID: session.ID,
                    Data:      data,
                })
            }
        }
    }()
}
```

**Deliverables**:
- [ ] ProcessViewComponent with message handling
- [ ] URLsViewComponent with message handling
- [ ] AIPTYComponent with proper message flow
- [ ] Tab navigation system
- [ ] Full integration tests
- [ ] Documentation: COMPONENT_CATALOG.md

---

### Phase 5: Testing & Validation (Week 4)

#### Goals
- Comprehensive test coverage
- Performance testing
- Bug fixing
- Documentation

#### Test Categories

**1. Unit Tests**
- Each component in isolation with mock messages
- ServiceBridge message conversion
- Observer pattern correctness

**2. Integration Tests**
- Service → Bridge → Shell → Component flow
- Multiple components receiving same message
- Message ordering and batching

**3. E2E Tests**
- Real process execution → logs appear
- AI terminal → output displays
- URL detection → URLs list updates

**4. Performance Tests**
- 10,000 logs/sec → UI responsive
- 100 concurrent processes → no lag
- Memory usage stable over time

**Deliverables**:
- [ ] 100% test coverage for ServiceBridge
- [ ] 80%+ test coverage for components
- [ ] E2E test suite
- [ ] Performance benchmarks
- [ ] Bug fixes
- [ ] Documentation: TESTING_GUIDE.md

---

### Phase 6: Migration Complete (Week 5)

#### Goals
- Remove old Model completely
- Remove BRUMMER_USE_COMPONENTS flag
- Final cleanup

#### Tasks
1. Delete `internal/tui/model.go` (old monolithic model)
2. Remove dual-mode toggle
3. Update all tests
4. Update documentation
5. Create migration PR

**Deliverables**:
- [ ] Old model removed
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Migration PR created
- [ ] CHANGELOG.md updated

---

## Success Criteria

### Technical
- ✅ Zero polling fallbacks (all event-driven)
- ✅ No race conditions (race detector clean)
- ✅ 100% test coverage for ServiceBridge
- ✅ All components receive updates via messages only
- ✅ No direct service access from components
- ✅ Memory usage stable (<100MB increase)

### Functional
- ✅ Process logs appear immediately (<50ms)
- ✅ AI terminal output displays correctly
- ✅ URL detection works across all sources
- ✅ Tab navigation smooth
- ✅ No lost updates or missed events

### Code Quality
- ✅ Clear separation of concerns
- ✅ Easy to add new components
- ✅ Testable in isolation
- ✅ Well-documented architecture

## Risk Mitigation

### Risk: Breaking existing functionality
**Mitigation**: Dual-mode operation during migration, feature flag

### Risk: Performance regression
**Mitigation**: Benchmark tests, performance monitoring

### Risk: Message flood overwhelming UI
**Mitigation**: Message batching in ServiceBridge, backpressure handling

### Risk: Complex debugging
**Mitigation**: Message logging in ServiceBridge, detailed error messages

## Timeline Summary

| Week | Phase | Deliverables |
|------|-------|--------------|
| 1 | Foundation + ServiceBridge | Messages, Bridge, LogStore observers |
| 2 | Components + Integration | Base components, Shell, Main.go integration |
| 3 | Remaining Components | Process, URLs, AI PTY components |
| 4 | Testing & Validation | Full test suite, bug fixes |
| 5 | Migration Complete | Remove old model, cleanup |

**Total**: 5 weeks to production-ready component system

## Comparison: Old vs New Approach

| Aspect | First Attempt (Failed) | New Approach (Proper) |
|--------|------------------------|----------------------|
| **Architecture** | Components access services directly | ServiceBridge adapter layer |
| **Updates** | Callbacks + polling + EventBus (mixed) | Unified messages via Bridge |
| **Coupling** | Tight (components know about services) | Loose (message-based) |
| **Testing** | Hard (need real services) | Easy (mock messages) |
| **Missed Events** | Yes (logs, URLs) | No (observer pattern) |
| **Race Conditions** | Yes (goroutine state access) | No (message-based) |
| **Polling** | Required (1-second fallback) | None (event-driven) |
| **Debuggability** | Hard (tracing event flow) | Easy (log all messages) |

## Next Steps

1. **Review this plan** with stakeholders
2. **Create branch** from 02e0611
3. **Start Phase 0** (message definitions)
4. **Weekly progress reviews**
5. **Adjust timeline** as needed

## Documentation Links

After implementation, we'll have:
- `SERVICE_BRIDGE.md` - ServiceBridge architecture
- `MESSAGE_TYPES.md` - All message type definitions
- `COMPONENT_ARCHITECTURE.md` - Component design patterns
- `INTEGRATION_GUIDE.md` - How to integrate new services
- `TESTING_GUIDE.md` - How to test components
- `COMPONENT_CATALOG.md` - All available components

---

**This plan incorporates every lesson learned from the first attempt and provides a clear, testable path to a properly architected component system.**