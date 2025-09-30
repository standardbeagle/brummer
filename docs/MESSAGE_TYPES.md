# Message Types Documentation

This document describes all BubbleTea message types used in the component system for event-driven communication between services and UI components.

## Design Principles

1. **Single Source of Truth**: Services emit events, components receive messages
2. **Immutable Messages**: All message fields are value types or pointers to immutable data
3. **Type Safety**: Each event type has a dedicated struct
4. **No Callbacks**: Components never call back to services, only receive messages
5. **Testability**: Messages can be easily mocked for component testing

## Message Categories

### Process Lifecycle Messages

These messages represent changes in process execution state.

#### `ProcessStartedMsg`
Sent when a process successfully starts.

```go
type ProcessStartedMsg struct {
    ProcessID string        // Unique process identifier
    Name      string        // Human-readable process name (e.g., "dev")
    Script    string        // The command/script being executed
    StartTime time.Time     // When the process started
}
```

**Emitted by**: ProcessManager via ServiceBridge
**Handled by**: ProcessViewComponent

#### `ProcessStoppedMsg`
Sent when a process terminates (success or failure).

```go
type ProcessStoppedMsg struct {
    ProcessID string        // Process that stopped
    Name      string        // Process name
    ExitCode  int          // Process exit code (0 = success)
    EndTime   time.Time    // When the process stopped
}
```

**Emitted by**: ProcessManager via ServiceBridge
**Handled by**: ProcessViewComponent

#### `ProcessOutputMsg`
Sent for each line of stdout/stderr from a process.

```go
type ProcessOutputMsg struct {
    ProcessID string        // Process generating output
    Line      string        // The output line
    IsError   bool         // true if from stderr, false if stdout
    Timestamp time.Time    // When the line was received
}
```

**Emitted by**: ProcessManager log callback via ServiceBridge
**Handled by**: LogsViewComponent (indirectly via LogStore)

**Note**: This message triggers LogStore.Add(), which in turn emits LogAddedMsg.

#### `ProcessStateChangedMsg`
Sent when a process transitions between states.

```go
type ProcessStateChangedMsg struct {
    ProcessID string                  // Process that changed state
    OldState  process.ProcessStatus   // Previous state
    NewState  process.ProcessStatus   // New state
}
```

**Process States**:
- `StatusPending`: Process created but not started
- `StatusRunning`: Process actively executing
- `StatusStopped`: Process terminated normally
- `StatusFailed`: Process terminated with error
- `StatusSuccess`: Process completed successfully

**Emitted by**: EventBus via ServiceBridge
**Handled by**: ProcessViewComponent

---

### Log Messages

These messages represent changes to the centralized log store.

#### `LogAddedMsg`
Sent when a new log entry is added to the log store.

```go
type LogAddedMsg struct {
    Entry *logs.LogEntry   // Pointer to the new log entry
}
```

**LogEntry Structure**:
```go
type LogEntry struct {
    ID          string      // Unique entry ID
    ProcessID   string      // Source process
    ProcessName string      // Human-readable process name
    Timestamp   time.Time   // When the log was created
    Content     string      // The log message
    IsError     bool        // Error vs info log
}
```

**Emitted by**: LogStore observer pattern via ServiceBridge
**Handled by**: LogsViewComponent

**Critical Design**: This message is how ALL logs reach the UI, including:
- Process stdout/stderr
- AI PTY output
- System logs
- MCP server logs

#### `LogsClearedMsg`
Sent when logs are cleared (all or for specific process).

```go
type LogsClearedMsg struct {
    ProcessID string   // Empty string = clear all logs
}
```

**Emitted by**: LogStore via ServiceBridge
**Handled by**: LogsViewComponent

---

### URL Detection Messages

These messages represent URLs discovered in process output.

#### `URLDetectedMsg`
Sent when a URL is detected and optionally proxied.

```go
type URLDetectedMsg struct {
    URL         string       // Original URL (e.g., http://localhost:3000)
    ProcessID   string       // Process that emitted the URL
    Type        string       // "http", "https", "ws", "wss"
    Timestamp   time.Time    // When URL was detected
    HasProxy    bool         // Whether proxy is enabled for this URL
    ProxyPort   int          // Proxy listen port (if HasProxy)
    ProxyURL    string       // Proxy URL (if HasProxy)
    ProxyLabel  string       // Human-readable label
}
```

**Emitted by**: EventBus (URL detection) via ServiceBridge
**Handled by**: URLsViewComponent

**Use Case**: Component displays both original URL and proxy URL for easy access.

---

### Error Messages

These messages represent errors from services that need UI attention.

#### `ErrorOccurredMsg`
Sent when a service encounters an error.

```go
type ErrorOccurredMsg struct {
    ProcessID   string       // Process that errored (or "system")
    ProcessName string       // Human-readable name
    Error       string       // Error message
    Timestamp   time.Time    // When error occurred
}
```

**Emitted by**: Various services via ServiceBridge
**Handled by**: Multiple components (error notifications)

**Examples**:
- Process failed to start
- Proxy server connection error
- MCP client connection lost

---

### Service Health Messages

These messages represent health status of core services.

#### `ServiceHealthMsg`
Sent periodically or on state change for service health monitoring.

```go
type ServiceHealthMsg struct {
    Service string   // "process", "proxy", "mcp"
    Healthy bool     // true = operational, false = degraded/down
    Message string   // Human-readable status message
}
```

**Emitted by**: Health monitors via ServiceBridge
**Handled by**: Status bar, notifications

**Use Case**: Display service status indicators in the UI.

---

## Message Flow Architecture

```
Services (ProcessManager, LogStore, ProxyServer)
    ↓ (callbacks, observers, EventBus events)
ServiceBridge (Adapter Layer)
    ↓ (tea.Msg via program.Send())
IntegratedShell (Message Router)
    ↓ (tea.Msg via component.Update())
Components (ProcessView, LogsView, URLsView)
    ↓ (Update state, re-render)
BubbleTea Render Loop
```

**Key Properties**:
1. Services never know about components
2. Components never call service methods directly
3. All updates flow through messages
4. ServiceBridge is the ONLY place that converts events to messages
5. No polling - purely event-driven

## Testing Patterns

### Unit Testing Components

```go
func TestComponentHandlesMessage(t *testing.T) {
    // Create component
    component := NewLogsViewComponent(logStore)

    // Create mock message
    msg := messages.LogAddedMsg{
        Entry: &logs.LogEntry{
            Content: "Test log",
        },
    }

    // Send message to component
    component.Update(msg)

    // Verify component state
    view := component.View()
    if !strings.Contains(view, "Test log") {
        t.Error("Component did not handle message")
    }
}
```

### Integration Testing

```go
func TestServiceToComponentFlow(t *testing.T) {
    // Setup services
    logStore := logs.NewStore(1000, nil)

    // Create ServiceBridge
    bridge := bridge.NewServiceBridge(nil, logStore, nil, nil)

    // Capture messages
    messages := []tea.Msg{}
    mockProgram := &MockProgram{
        SendFunc: func(msg tea.Msg) {
            messages = append(messages, msg)
        },
    }

    bridge.Start(mockProgram)

    // Trigger service event
    logStore.Add("test", "Test", "Hello", false)

    // Verify message was sent
    if len(messages) != 1 {
        t.Fatalf("Expected 1 message, got %d", len(messages))
    }

    if msg, ok := messages[0].(messages.LogAddedMsg); ok {
        if msg.Entry.Content != "Hello" {
            t.Error("Wrong message content")
        }
    } else {
        t.Error("Wrong message type")
    }
}
```

## Migration Notes

### Adding New Message Types

1. Define struct in `service_events.go`
2. Add tests in `service_events_test.go`
3. Update ServiceBridge to emit the message
4. Update IntegratedShell to route the message
5. Update components to handle the message
6. Document here

### Message Naming Convention

- Past tense for completed actions: `ProcessStartedMsg`, `LogAddedMsg`
- Use `Msg` suffix for all message types
- Group related messages with common prefix: `Process*`, `Log*`, `URL*`

## Future Enhancements

### Potential New Messages

- `ProcessRestartedMsg`: When a process is auto-restarted
- `ProxyRequestMsg`: Individual HTTP request details
- `MCPClientConnectedMsg`: When MCP client connects
- `FileChangedMsg`: For file watching integration
- `TestResultMsg`: For test runner integration

### Message Batching

For high-frequency messages (logs, process output), consider batching:

```go
type BatchLogAddedMsg struct {
    Entries []*logs.LogEntry
}
```

This reduces BubbleTea message queue pressure during high-volume scenarios.