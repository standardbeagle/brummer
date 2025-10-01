# Component System Message Flow Architecture

## Executive Summary

The current component system has a **fundamental architecture gap**: there's no unified event-driven message flow from core services to UI components. Instead, it uses a mix of callbacks, polling, and direct coupling that leads to:

1. **Lost Updates**: AI PTY logs don't appear because nothing notifies the log view
2. **Polling Fallbacks**: Need 1-second polling to catch missed events
3. **Tight Coupling**: Components directly access shared stores instead of receiving messages
4. **Race Conditions**: Multiple systems trying to read/write shared state
5. **Inconsistent Patterns**: Some updates use EventBus, some use callbacks, some use polling

## Current State Analysis

### Core Services (Data Sources)

1. **Process Manager** (`internal/process/manager.go`)
   - **Data**: Process lifecycle, stdout/stderr, exit codes
   - **Current Mechanism**: Callbacks (`AddLogCallback`) + EventBus
   - **Problem**: Callbacks are for logs only, UI must poll for process state changes

2. **Log Store** (`internal/logs/store.go`)
   - **Data**: Aggregated logs from all sources
   - **Current Mechanism**: Passive store, NO notifications
   - **Problem**: Components must poll or be explicitly notified

3. **Proxy Server** (`internal/proxy/`)
   - **Data**: URL mappings, HTTP requests, browser sessions
   - **Current Mechanism**: Direct method calls (GetURLMappings)
   - **Problem**: UI must poll for new URLs

4. **AI Coder Manager** (`internal/aicoder/`)
   - **Data**: PTY sessions, AI provider status, output streams
   - **Current Mechanism**: Component directly manages PTY
   - **Problem**: No integration with unified event system

5. **EventBus** (`pkg/events/events.go`)
   - **Data**: Generic events (LogLine, ProcessStateChange, etc.)
   - **Current Mechanism**: Pub/sub with bounded worker pool
   - **Problem**: Not integrated with BubbleTea message system

### Current UI Layers

#### Old Monolithic Model (`internal/tui/model.go`)
- **Pattern**: Direct access to all services
- **Update Flow**: Periodic polling (100ms) + callbacks
- **State**: Single giant struct with all UI state
- **Works**: Yes, but not maintainable/testable

#### New Component System (`internal/tui/components/`)
- **Pattern**: Component-based with message passing
- **Update Flow**: **INCONSISTENT** - mix of messages, polling, direct access
- **State**: Distributed across components
- **Works**: Partially - missing fundamental message routing

### Message Flow Gaps

#### Gap 1: No Service-to-UI Bridge
```
Process Manager → ❌ → IntegratedShell → ❌ → ProcessViewComponent
                  (callback only for logs)
```

**What's Missing**: A unified adapter that converts service events to BubbleTea messages.

#### Gap 2: No Centralized Event Router
```
EventBus.Publish(event) → ❌ → BubbleTea Message Loop
                           (different systems, no bridge)
```

**What's Missing**: A bridge that subscribes to EventBus and converts events to tea.Msg.

#### Gap 3: Store Updates Don't Notify
```
logStore.Add("ai-pty", "message") → ❌ → LogsViewComponent
                                      (no notification mechanism)
```

**What's Missing**: Observer pattern or event emission on store mutations.

#### Gap 4: Component-to-Component Communication
```
AIPTYComponent (adds log) → ❌ → LogsViewComponent (displays logs)
                              (no message routing)
```

**What's Missing**: Parent shell must route messages between components.

## Proposed Architecture

### Design Principles

1. **Single Source of Truth**: Core services own data, UI components are views
2. **Event-Driven**: All state changes emit events
3. **Message Passing**: BubbleTea messages are the ONLY way to update UI
4. **Loose Coupling**: Components don't know about each other or services
5. **Testability**: Each layer can be tested independently

### Architecture Layers

```
┌─────────────────────────────────────────────────────────────────┐
│                        BubbleTea Program                         │
│                     (Message Loop Orchestrator)                  │
└────────────────────────────────┬────────────────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   IntegratedShell       │
                    │  (Message Router)       │
                    │                         │
                    │  Routes messages to:    │
                    │  - Active component     │
                    │  - Specific components  │
                    │  - All components       │
                    └────┬────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ProcessView   │ │LogsView      │ │AIPTYComponent│
│              │ │              │ │              │
│Receives:     │ │Receives:     │ │Receives:     │
│- ProcessMsg  │ │- LogsMsg     │ │- AIPTYMsg    │
│- StateChange │ │- NewLogs     │ │- OutputMsg   │
└──────────────┘ └──────────────┘ └──────────────┘
        ▲                ▲                ▲
        │                │                │
        └────────────────┴────────────────┘
                         │
                ┌────────▼────────┐
                │  Service Bridge │
                │  (Adapter Layer)│
                │                 │
                │  Converts:      │
                │  Events → Msgs  │
                │  Callbacks →Msg │
                └────┬────────────┘
                     │
        ┌────────────┼────────────────┐
        │            │                │
        ▼            ▼                ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ProcessManager│ │LogStore      │ │ProxyServer   │
│              │ │              │ │              │
│Emits:        │ │Emits:        │ │Emits:        │
│- Started     │ │- LogAdded    │ │- URLDetected │
│- Stopped     │ │- LogsCleared │ │- RequestMade │
│- Output      │ │              │ │              │
└──────────────┘ └──────────────┘ └──────────────┘
```

### Core Components

#### 1. Service Bridge (NEW)
**Location**: `internal/tui/bridge/service_bridge.go`

**Purpose**: Converts service events to BubbleTea messages

```go
type ServiceBridge struct {
    processMgr  *process.Manager
    logStore    *logs.Store
    proxyServer *proxy.Server
    eventBus    *events.EventBus
    program     *tea.Program
}

// Subscribe to all service events and convert to messages
func (b *ServiceBridge) Start() {
    // Process Manager: convert callbacks to messages
    b.processMgr.AddLogCallback(func(processID, line string, isError bool) {
        b.program.Send(LogLineMsg{ProcessID: processID, Line: line, IsError: isError})
    })

    // EventBus: subscribe to process state changes
    b.eventBus.Subscribe(events.ProcessStateChange, func(evt events.Event) {
        b.program.Send(ProcessStateChangedMsg{
            ProcessID: evt.ProcessID,
            OldState:  evt.Data["old_state"].(process.ProcessStatus),
            NewState:  evt.Data["new_state"].(process.ProcessStatus),
        })
    })

    // Log Store: observe additions (NEEDS TO BE ADDED)
    b.logStore.OnAdd(func(entry *logs.LogEntry) {
        b.program.Send(LogAddedMsg{Entry: entry})
    })
}
```

#### 2. Unified Message Types (NEW)
**Location**: `internal/tui/messages/service_messages.go`

```go
// Process lifecycle messages
type ProcessStartedMsg struct {
    ProcessID string
    Name      string
    StartTime time.Time
}

type ProcessStoppedMsg struct {
    ProcessID string
    ExitCode  int
    EndTime   time.Time
}

type ProcessStateChangedMsg struct {
    ProcessID string
    OldState  process.ProcessStatus
    NewState  process.ProcessStatus
}

// Log messages
type LogLineMsg struct {
    ProcessID string
    Line      string
    IsError   bool
}

type LogAddedMsg struct {
    Entry *logs.LogEntry
}

type LogsClearedMsg struct{}

// URL/Proxy messages
type URLDetectedMsg struct {
    URL         string
    ProcessID   string
    ProxyPort   int
    ProxyURL    string
}

type HTTPRequestMsg struct {
    Method    string
    URL       string
    Status    int
    ProcessID string
}

// AI Coder messages
type AIPTYOutputMsg struct {
    SessionID string
    Data      []byte
}

type AIPTYEventMsg struct {
    Type    string
    Session interface{}
}
```

#### 3. Enhanced Log Store (MODIFY EXISTING)
**Location**: `internal/logs/store.go`

Add observer pattern:

```go
type LogObserver func(entry *LogEntry)

type Store struct {
    // ... existing fields ...
    observers []LogObserver
    observerMu sync.RWMutex
}

func (s *Store) OnAdd(observer LogObserver) {
    s.observerMu.Lock()
    defer s.observerMu.Unlock()
    s.observers = append(s.observers, observer)
}

func (s *Store) Add(processID, processName, content string, isError bool) *LogEntry {
    // ... existing add logic ...

    // Notify observers
    s.observerMu.RLock()
    observers := s.observers
    s.observerMu.RUnlock()

    for _, obs := range observers {
        obs(entry)
    }

    return entry
}
```

#### 4. IntegratedShell Message Router (ENHANCE EXISTING)
**Location**: `internal/tui/components/integrated_shell.go`

```go
func (s *IntegratedShell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    // Service messages - route to appropriate components
    case LogAddedMsg:
        if s.logsView != nil {
            _, cmd := s.logsView.Update(msg)
            cmds = append(cmds, cmd)
        }

    case ProcessStateChangedMsg:
        if s.processView != nil {
            _, cmd := s.processView.Update(msg)
            cmds = append(cmds, cmd)
        }

    case URLDetectedMsg:
        if s.urlsView != nil {
            _, cmd := s.urlsView.Update(msg)
            cmds = append(cmds, cmd)
        }

    // Component messages - route to specific components
    case AIPTYOutputMsg:
        if s.aiView != nil {
            _, cmd := s.aiView.Update(msg)
            cmds = append(cmds, cmd)
        }

    // ... rest of handlers ...
    }

    return s, tea.Batch(cmds...)
}
```

## Migration Plan

### Phase 1: Foundation (Week 1)
1. Create `ServiceBridge` adapter
2. Add observer pattern to `LogStore`
3. Define unified message types in `internal/tui/messages/`
4. Write tests for message conversion

### Phase 2: Process Integration (Week 1)
1. Route process state changes through ServiceBridge
2. Update ProcessViewComponent to receive messages instead of polling
3. Remove polling from process view
4. Test process lifecycle: start → logs → stop

### Phase 3: Log Integration (Week 2)
1. Route all log additions through ServiceBridge
2. Update LogsViewComponent to receive LogAddedMsg
3. Remove debounced subscription polling fallback
4. Test: process logs, AI PTY logs, manual logs

### Phase 4: URL/Proxy Integration (Week 2)
1. Add URL detection events to proxy server
2. Route URL detections through ServiceBridge
3. Update URLsViewComponent to receive URLDetectedMsg
4. Remove direct GetURLMappings calls

### Phase 5: AI Coder Integration (Week 3)
1. Route PTY output through ServiceBridge
2. Ensure AIPTYComponent receives messages properly
3. Test terminal output display
4. Remove any remaining direct PTY access

### Phase 6: Cleanup (Week 3)
1. Remove old callback-based integrations
2. Remove polling fallbacks
3. Remove DebouncedLogSubscription (no longer needed)
4. Update documentation

## Benefits

### Performance
- **No polling**: Event-driven is more efficient than periodic polling
- **Lock-free**: Message passing avoids lock contention
- **Batching**: ServiceBridge can batch rapid updates

### Reliability
- **No missed updates**: Every state change generates a message
- **No race conditions**: Single-threaded BubbleTea model
- **Testable**: Each layer can be tested in isolation

### Maintainability
- **Clear boundaries**: Services don't know about UI
- **Loose coupling**: Components receive messages, don't query services
- **Consistent patterns**: All updates flow the same way

### Developer Experience
- **Predictable**: Same pattern for all component updates
- **Debuggable**: Can log all messages in ServiceBridge
- **Extensible**: Easy to add new components or services

## Testing Strategy

### Unit Tests
- **ServiceBridge**: Mock services, verify correct messages emitted
- **LogStore observers**: Verify observer notifications
- **Components**: Mock messages, verify state updates

### Integration Tests
- **End-to-end**: Start process → verify ProcessView updates
- **Cross-component**: Add log → verify LogsView updates
- **AI PTY**: Start terminal → verify output appears

### Performance Tests
- **Message latency**: Measure time from event to UI update
- **Throughput**: Handle 1000s of log lines without lag
- **Memory**: Verify no memory leaks in ServiceBridge

## Current Issues Resolved

| Issue | Current State | After Migration |
|-------|--------------|----------------|
| AI PTY logs don't appear | No notification mechanism | LogAddedMsg routes to LogsView |
| Need 1-second polling fallback | Store doesn't emit events | Observer pattern notifies immediately |
| Process state polling | UI polls Manager | ProcessStateChangedMsg pushed to UI |
| URL detection polling | UI polls GetURLMappings | URLDetectedMsg pushed to UI |
| Tight coupling | Components access services directly | ServiceBridge decouples |
| Race conditions | Multiple systems access shared state | Single-threaded message loop |
| Testing difficulty | Can't test components without real services | Mock messages for unit tests |

## Conclusion

The current component system lacks a fundamental architectural layer: **the Service-to-UI bridge**. Without this, we're forced to use polling, callbacks, and direct coupling, which leads to bugs like missing logs and inconsistent updates.

The proposed architecture adds a clean adapter layer that converts service events into BubbleTea messages, enabling true event-driven UI updates with no polling, no race conditions, and clear separation of concerns.

This is not a rewrite - it's adding the missing architectural layer that should have been designed from the start.