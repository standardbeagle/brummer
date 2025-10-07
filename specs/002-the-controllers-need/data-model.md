# Data Model: Component-Service Architecture Patterns

**Feature**: Controller to Service/Component Migration
**Date**: 2025-10-06

## Overview

This document defines the data structures and interaction patterns for the Component → Service architecture, replacing the Controller-based approach.

## Core Patterns

### 1. Component Interface (BubbleTea Standard)

All view components implement the BubbleTea interface:

```go
type ViewComponent interface {
    Init() tea.Cmd
    Update(tea.Msg) (tea.Model, tea.Cmd)
    View() string
}
```

### 2. Component State Structure

Components maintain their own state without controller intermediaries:

```go
type ComponentState struct {
    // Service references (injected at creation)
    serviceDep ServiceInterface

    // Component-local state
    selectedIndex int
    filterState   FilterConfig
    viewport      viewport.Model  // BubbleTea sub-component

    // Presentation state
    width  int
    height int
    focus  bool
}
```

### 3. Service Interface Pattern

Components call services directly through well-defined interfaces:

```go
type ProcessServiceInterface interface {
    GetAllProcesses() []Process
    GetProcess(id string) (*Process, error)
    StartProcess(id string) error
    StopProcess(id string) error
}

// Component uses service
func (c *ComponentState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    processes := c.processMgr.GetAllProcesses()  // Direct call
    // ... handle processes
}
```

### 4. Event Communication (BubbleTea Messages)

Services emit events that become BubbleTea messages:

```go
// Service emits event via EventBus
eventBus.Emit("process.started", ProcessEvent{ID: "dev"})

// Model converts to BubbleTea message
type ProcessStartedMsg struct {
    ProcessID string
    Timestamp time.Time
}

// Component handles in Update()
func (c *ComponentState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ProcessStartedMsg:
        // React to process started
    }
}
```

## Component State Models

### ProcessesViewComponent

**Purpose**: Display process list with selection and metadata

**State**:
```go
type ProcessesViewState struct {
    // Service dependencies
    processMgr *process.Manager

    // Component state
    list           bubbles.List
    selectedID     string
    showStopped    bool
    sortBy         SortOrder

    // Presentation
    width  int
    height int
}

type SortOrder int
const (
    SortByName SortOrder = iota
    SortByStatus
    SortByStartTime
)
```

**Service Interactions**:
- `processMgr.GetAllProcesses()` → get process list
- `processMgr.GetProcess(id)` → get process details
- `processMgr.StartProcess(id)` → start selected process
- `processMgr.StopProcess(id)` → stop selected process

**Events Handled**:
- `ProcessStartedMsg` → refresh list, update status
- `ProcessStoppedMsg` → refresh list, update status
- `ProcessOutputMsg` → update process metadata

### LogsViewComponent

**Purpose**: Display filtered logs with scrolling and follow mode

**State**:
```go
type LogsViewState struct {
    // Service dependencies
    logStore *logs.Store

    // Component state
    processID      string
    viewport       viewport.Model
    filter         LogFilter
    followMode     bool
    highPriority   bool

    // Presentation
    width  int
    height int
}

type LogFilter struct {
    Level    LogLevel
    Search   string
    TimeFrom time.Time
    TimeTo   time.Time
}
```

**Service Interactions**:
- `logStore.GetLogs(processID)` → get all logs for process
- `logStore.GetFilteredLogs(filter)` → get filtered logs
- `logStore.Subscribe(processID)` → subscribe to new logs

**Events Handled**:
- `LogAddedMsg` → append to viewport, auto-scroll if following
- `ProcessStoppedMsg` → disable follow mode

### ErrorsViewComponent

**Purpose**: Display error list with filtering and detail view

**State**:
```go
type ErrorsViewState struct {
    // Service dependencies
    errorStore *errors.Store

    // Component state
    list           bubbles.List
    selectedIndex  int
    filter         ErrorFilter
    showDismissed  bool
    detailView     bool

    // Presentation
    width  int
    height int
}

type ErrorFilter struct {
    Severity []ErrorSeverity
    Source   []string
    TimeFrom time.Time
}
```

**Service Interactions**:
- `errorStore.GetErrors()` → get all errors
- `errorStore.GetFilteredErrors(filter)` → get filtered errors
- `errorStore.DismissError(id)` → dismiss selected error
- `errorStore.ClearAll()` → clear all errors

**Events Handled**:
- `ErrorAddedMsg` → add to list, show notification
- `ErrorDismissedMsg` → remove from list

### WebViewComponent

**Purpose**: Display proxy requests with filtering and detail view

**State**:
```go
type WebViewState struct {
    // Service dependencies
    proxyServer *proxy.Server

    // Component state
    list          bubbles.List
    selectedIndex int
    filter        RequestFilter
    detailView    bool
    viewport      viewport.Model

    // Presentation
    width  int
    height int
}

type RequestFilter struct {
    Status []int  // HTTP status codes
    Method []string
    Path   string  // Regex pattern
}
```

**Service Interactions**:
- `proxyServer.GetRequests()` → get all captured requests
- `proxyServer.GetFilteredRequests(filter)` → get filtered requests
- `proxyServer.GetRequestDetail(id)` → get request/response details

**Events Handled**:
- `ProxyRequestMsg` → add to list, update count
- `ProxyResponseMsg` → update request status

### AICoderComponent (IntegratedShell)

**Purpose**: Manage AI coder sessions with terminal and browser integration

**State**:
```go
type AICoderState struct {
    // Service dependencies
    ptyManager *pty.Manager
    sessionMgr *aicoder.SessionManager

    // Component state
    activeSession *aicoder.Session
    ptyView       *pty.View
    terminalFocus bool
    browserURL    string

    // Presentation
    width  int
    height int
}
```

**Service Interactions**:
- `sessionMgr.CreateSession()` → create new AI coder session
- `sessionMgr.GetSession(id)` → get session details
- `ptyManager.Write(data)` → send input to PTY
- `ptyManager.Resize(cols, rows)` → resize PTY

**Events Handled**:
- `PTYOutputMsg` → update terminal display
- `SessionStartedMsg` → show session info
- `SessionStoppedMsg` → cleanup session

## Message Types

### System Messages
```go
type WindowSizeMsg struct {
    Width  int
    Height int
}

type ViewSwitchedMsg struct {
    PreviousView navigation.View
    CurrentView  navigation.View
}
```

### Process Messages
```go
type ProcessStartedMsg struct {
    ProcessID string
    Timestamp time.Time
}

type ProcessStoppedMsg struct {
    ProcessID string
    ExitCode  int
    Timestamp time.Time
}

type ProcessOutputMsg struct {
    ProcessID string
    Line      string
    Timestamp time.Time
}
```

### Log Messages
```go
type LogAddedMsg struct {
    ProcessID string
    Log       logs.Entry
}

type LogFilterChangedMsg struct {
    Filter LogFilter
}
```

### Error Messages
```go
type ErrorAddedMsg struct {
    Error errors.Entry
}

type ErrorDismissedMsg struct {
    ErrorID string
}
```

### Proxy Messages
```go
type ProxyRequestMsg struct {
    Request proxy.Request
}

type ProxyResponseMsg struct {
    RequestID string
    Response  proxy.Response
}
```

### PTY Messages
```go
type PTYOutputMsg struct {
    SessionID string
    Data      []byte
}

type PTYResizeMsg struct {
    Cols int
    Rows int
}
```

## Service Interfaces

### Process Manager Interface
```go
type ProcessManager interface {
    GetAllProcesses() []Process
    GetProcess(id string) (*Process, error)
    StartProcess(id string) error
    StopProcess(id string) error
    RestartProcess(id string) error
    GetProcessLogs(id string) []logs.Entry
    GetProcessURLs(id string) []URL
}
```

### Log Store Interface
```go
type LogStore interface {
    GetLogs(processID string) []logs.Entry
    GetFilteredLogs(filter LogFilter) []logs.Entry
    Subscribe(processID string, handler func(logs.Entry))
    Unsubscribe(processID string)
    Clear(processID string)
    ClearAll()
}
```

### Proxy Server Interface
```go
type ProxyServer interface {
    GetRequests() []proxy.Request
    GetFilteredRequests(filter RequestFilter) []proxy.Request
    GetRequestDetail(id string) (*proxy.RequestDetail, error)
    GetStats() proxy.Stats
    ClearRequests()
}
```

## Key Handler Patterns

### Hierarchical Key Handling

Components register key handlers at different priority levels:

```go
type KeyHandlerContext struct {
    FocusedComponent KeyHandler  // Highest priority (e.g., PTY terminal)
    OverlayHandlers  []KeyHandler // Medium priority (e.g., dialogs)
    GlobalHandlers   []KeyHandler  // Lowest priority (e.g., quit, navigation)
}

type KeyHandler interface {
    HandleKey(msg tea.KeyMsg, model tea.Model) (tea.Model, tea.Cmd)
    Priority() HandlerPriority
}

const (
    PriorityFocus   HandlerPriority = 100
    PriorityOverlay HandlerPriority = 50
    PriorityGlobal  HandlerPriority = 10
)
```

### Key Handler Delegation

```go
// Component implements KeyHandler
func (c *ProcessesViewComponent) HandleKey(msg tea.KeyMsg, model tea.Model) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "enter":
        // Handle enter - navigate to logs
        return model, NavigateToLogsCmd(c.selectedID)
    case "r":
        // Restart selected process
        return model, c.restartProcess()
    default:
        // Let list handle other keys
        newList, cmd := c.list.Update(msg)
        c.list = newList
        return model, cmd
    }
}
```

## Layout Management

### Layout Calculation Pattern

Layout is calculated as a pure function:

```go
type LayoutDimensions struct {
    HeaderHeight      int
    FooterHeight      int
    MainViewHeight    int
    SystemPanelHeight int
    TotalWidth        int
    TotalHeight       int
}

func CalculateLayout(windowSize WindowSize, systemPanelOpen bool) LayoutDimensions {
    // Pure function - no state
    dims := LayoutDimensions{
        HeaderHeight: 3,
        FooterHeight: 2,
        TotalWidth:   windowSize.Width,
        TotalHeight:  windowSize.Height,
    }

    if systemPanelOpen {
        dims.SystemPanelHeight = 10
    }

    dims.MainViewHeight = dims.TotalHeight - dims.HeaderHeight - dims.FooterHeight - dims.SystemPanelHeight
    return dims
}
```

### Component Sizing

Components receive dimensions and are responsible for their own layout:

```go
func (c *ProcessesViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case WindowSizeMsg:
        c.width = msg.Width
        c.height = msg.Height
        c.list.SetSize(c.width, c.height-2)  // Leave room for borders
    }
}
```

## Event Bus Integration

### EventBus to BubbleTea Bridge

Model subscribes to EventBus and converts events to messages:

```go
func (m *Model) Init() tea.Cmd {
    // Subscribe to EventBus
    m.eventBus.Subscribe("process.*", func(event events.Event) {
        switch event.Type {
        case "process.started":
            m.updateChan <- ProcessStartedMsg{ProcessID: event.Data.(string)}
        case "process.stopped":
            m.updateChan <- ProcessStoppedMsg{ProcessID: event.Data.(string)}
        }
    })

    return tea.Batch(
        listenForEvents(m.updateChan),
        // ... other init commands
    )
}

func listenForEvents(ch chan tea.Msg) tea.Cmd {
    return func() tea.Msg {
        return <-ch
    }
}
```

## Testing Patterns

### Component Testing

Components are testable in isolation:

```go
func TestProcessesViewSelection(t *testing.T) {
    // Create component with mock service
    mockMgr := &MockProcessManager{
        Processes: []Process{
            {ID: "dev", Status: StatusRunning},
            {ID: "test", Status: StatusStopped},
        },
    }

    component := NewProcessesViewComponent(mockMgr)

    // Send key message
    model, cmd := component.Update(tea.KeyMsg{Type: tea.KeyDown})

    // Verify state
    assert.Equal(t, "test", component.selectedID)
}
```

### Integration Testing

Full workflows test component + service interaction:

```go
func TestProcessSelectionToLogs(t *testing.T) {
    // Create real services
    processMgr := process.NewManager()
    logStore := logs.NewStore()

    // Create model with components
    model := NewModel(processMgr, logStore, ...)

    // Simulate user flow
    model = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})  // Switch to processes
    model = model.Update(tea.KeyMsg{Type: tea.KeyDown})  // Select process
    model = model.Update(tea.KeyMsg{Type: tea.KeyEnter})  // Navigate to logs

    // Verify state
    assert.Equal(t, ViewLogs, model.currentView())
    assert.Equal(t, "dev", model.selectedProcess)
}
```

## Migration Checklist

For each controller migration:

- [ ] Create or enhance target component
- [ ] Define component state structure
- [ ] Implement service interface calls
- [ ] Handle BubbleTea messages
- [ ] Implement key handling
- [ ] Write component tests
- [ ] Write integration tests
- [ ] Remove controller file
- [ ] Remove controller references from model.go
- [ ] Verify all tests pass
- [ ] Verify manual testing (keyboard shortcuts, workflows)
