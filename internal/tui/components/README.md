# TUI Component Interface Documentation

## Overview

This document defines the standard interface and patterns for all TUI view components in Brummer. Components follow the BubbleTea component pattern and call services directly without intermediate controllers.

## Architecture Principles

### Component → Service Pattern
Components interact **directly** with services (process.Manager, logs.Store, proxy.Server, etc.) without controller intermediaries:

```
User Input → BubbleTea Component → Service Layer → Storage/State
         ↖                       ↗
          Event Bus (for async updates)
```

### Component Responsibilities
- **View State Management**: Own presentation state (selected item, scroll position, filter text)
- **Service Integration**: Call services directly for data and actions
- **Event Handling**: Subscribe to relevant EventBus events and convert to tea.Msg
- **Key Handling**: Implement KeyHandler interface for hierarchical input routing
- **Rendering**: Implement View() to render current state using lipgloss

### What Components Should NOT Do
- ❌ Manage service lifecycle (services are singletons managed by main model)
- ❌ Directly modify other components' state
- ❌ Make assumptions about other views or global state
- ❌ Block on I/O operations (use async tea.Cmd pattern)

## Standard Component Interface

All view components must implement the BubbleTea Model interface:

```go
type ViewComponent interface {
    // BubbleTea Model interface
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string

    // KeyHandler interface for hierarchical input
    HandleKey(msg tea.KeyMsg, model tea.Model) (tea.Model, tea.Cmd)
    Priority() HandlerPriority
}
```

### Init() - Initialization
Returns initial commands to execute (subscriptions, initial data fetch, etc.)

```go
func (p *ProcessesViewComponent) Init() tea.Cmd {
    // Return batch of initialization commands
    return tea.Batch(
        p.subscribeToEvents(),
        p.fetchInitialData(),
    )
}
```

### Update() - Message Handling
Handles all messages (keyboard, window size, custom events) and updates component state.

**Critical Rule**: This is the ONLY place component state should be modified.

```go
func (p *ProcessesViewComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Delegate to HandleKey for input routing
        return p.HandleKey(msg, p)

    case tea.WindowSizeMsg:
        // Update dimensions
        p.width = msg.Width
        p.height = msg.Height
        p.list.SetSize(p.width, p.height-2)

    case ProcessStartedMsg:
        // Handle custom event message
        p.updateProcessStatus(msg.ID, StatusRunning)
        return p, p.showNotification("Process started")

    case ProcessErrorMsg:
        // Handle error from async operation
        return p, p.showError(msg.Error)
    }

    // Delegate to sub-components
    var cmd tea.Cmd
    p.list, cmd = p.list.Update(msg)
    return p, cmd
}
```

### View() - Rendering
Renders component as styled string. Should be **pure function** of component state.

```go
func (p *ProcessesViewComponent) View() string {
    if p.width == 0 {
        return "Loading..."
    }

    return lipgloss.JoinVertical(
        lipgloss.Left,
        p.renderHeader(),
        p.list.View(),
        p.renderFooter(),
    )
}
```

## Service Interaction Patterns

### Synchronous Service Calls
Use for immediate read operations during Update():

```go
// Get current data from service
processes := p.processMgr.GetAllProcesses()
p.updateList(processes)
```

### Asynchronous Service Calls
Use tea.Cmd for operations with side effects:

```go
func (p *ProcessesViewComponent) startProcess(id string) tea.Cmd {
    return func() tea.Msg {
        err := p.processMgr.StartProcess(id)
        if err != nil {
            return ProcessErrorMsg{ID: id, Error: err}
        }
        return ProcessStartedMsg{ID: id}
    }
}
```

### EventBus Integration
Services emit events, components convert to tea.Msg:

```go
// Service emits event
eventBus.Emit("process.started", ProcessEvent{ID: "dev"})

// Component subscribes in Init()
func (p *ProcessesViewComponent) subscribeToEvents() tea.Cmd {
    return func() tea.Msg {
        p.eventBus.Subscribe("process.*", func(event Event) {
            // Convert to BubbleTea message
            p.updateChan <- ProcessEventMsg{Event: event}
        })
        return nil
    }
}
```

## Key Handling Pattern

Components implement KeyHandler interface for hierarchical input routing:

```go
func (p *ProcessesViewComponent) HandleKey(msg tea.KeyMsg, model tea.Model) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "enter":
        // Handle Enter - start/stop process
        return p, p.toggleProcess()

    case "r":
        // Handle R - restart process
        return p, p.restartProcess()

    case "up", "down":
        // Delegate to list component
        newList, cmd := p.list.Update(msg)
        p.list = newList
        return p, cmd

    default:
        // Not handled - allow propagation to global handlers
        return model, nil
    }
}

func (p *ProcessesViewComponent) Priority() HandlerPriority {
    return PriorityViewSpecific  // 25 - normal view priority
}
```

### Key Handler Priority Levels
- **100 (Focus)**: Terminal emulator, text input (captures all keys)
- **50 (Overlay)**: Command palette, modal dialogs
- **25 (ViewSpecific)**: Current view component
- **10 (Global)**: Navigation shortcuts (q, 1-9, Tab, /)

**Important**: Return `(model, nil)` if key not handled to allow propagation.

## Component State Structure

Components should maintain minimal, view-specific state:

```go
type ProcessesViewComponent struct {
    // Service dependencies (injected at creation)
    processMgr *process.Manager
    eventBus   *events.EventBus

    // BubbleTea sub-components
    list bubbles.List

    // View state
    selectedID  string
    showStopped bool
    sortBy      SortOrder

    // Presentation
    width  int
    height int

    // Event handling
    updateChan chan tea.Msg
}
```

## Error Handling

Components handle errors from services and display to user:

```go
func (p *ProcessesViewComponent) handleEnter() tea.Cmd {
    return func() tea.Msg {
        err := p.processMgr.StartProcess(p.selectedID)
        if err != nil {
            return ProcessErrorMsg{
                ProcessID: p.selectedID,
                Error:     err,
            }
        }
        return ProcessStartedMsg{ProcessID: p.selectedID}
    }
}

// In Update()
case ProcessErrorMsg:
    return p, ShowNotificationCmd(
        "Error",
        fmt.Sprintf("Failed to start %s: %v", msg.ProcessID, msg.Error),
    )
```

## Testing Components

Components should be testable in isolation with mock services:

```go
func TestProcessesView(t *testing.T) {
    // Create mock service
    mockMgr := &MockProcessManager{
        Processes: []Process{
            {ID: "dev", Status: StatusRunning},
            {ID: "test", Status: StatusStopped},
        },
    }

    // Create component with mock
    component := NewProcessesViewComponent(mockMgr)

    // Test initialization
    cmd := component.Init()
    assert.NotNil(t, cmd)

    // Test key handling
    model, cmd := component.HandleKey(
        tea.KeyMsg{Type: tea.KeyEnter},
        component,
    )
    assert.NotNil(t, cmd)

    // Test rendering
    view := component.View()
    assert.Contains(t, view, "dev")
    assert.Contains(t, view, "Running")
}
```

## Component Lifecycle

1. **Creation**: Component created with service dependencies injected
2. **Initialization**: Init() returns commands for subscriptions/data fetch
3. **Message Loop**: Update() handles all messages, modifies state, returns commands
4. **Rendering**: View() called by BubbleTea to render current state
5. **Cleanup**: Component removed when view changes (automatic cleanup via BubbleTea)

## Best Practices

### Do ✅
- Keep state minimal and component-specific
- Inject service dependencies at creation
- Use BubbleTea sub-components (list, viewport, textarea, etc.)
- Handle all messages gracefully (unknown types OK to ignore)
- Return nil for commands when no side effect needed
- Use lipgloss for consistent styling
- Make View() a pure function of state
- Test with mock services

### Don't ❌
- Don't call services from View() (use Update() instead)
- Don't modify state outside Update() method
- Don't block on I/O operations
- Don't make assumptions about other components
- Don't manage service lifecycle
- Don't use global state
- Don't ignore errors from services
- Don't create goroutines without proper cleanup

## Migration from Controllers

When migrating from controller-based code:

1. **Move State**: Controller view state → Component struct fields
2. **Direct Service Calls**: Remove controller wrapper, call service directly
3. **Event Handling**: EventBus subscription → tea.Msg conversion in Update()
4. **Key Handling**: Controller key routing → Component HandleKey() method
5. **Cleanup**: Remove controller file and all references from model.go

Example migration:
```go
// Before (controller pattern)
type ProcessViewController struct {
    state      ProcessViewState
    processMgr *process.Manager
}

func (c *ProcessViewController) HandleInput(key string) {
    // Controller owns state and wraps service calls
    c.state.selectedID = key
    c.processMgr.StartProcess(key)
}

// After (component pattern)
type ProcessesViewComponent struct {
    processMgr *process.Manager
    selectedID string  // State moved to component
}

func (p *ProcessesViewComponent) HandleKey(msg tea.KeyMsg, model tea.Model) (tea.Model, tea.Cmd) {
    // Component calls service directly
    p.selectedID = msg.String()
    return p, func() tea.Msg {
        err := p.processMgr.StartProcess(p.selectedID)
        // ... handle result
    }
}
```

## See Also

- [Service Interaction Contract](../../specs/002-the-controllers-need/contracts/service-interaction.yaml)
- [Key Handler Contract](../../specs/002-the-controllers-need/contracts/key-handler.yaml)
- [Component Interface Contract](../../specs/002-the-controllers-need/contracts/component-interface.yaml)
- [Architecture Overview](../../docs/architecture/overview.md)
- [Concurrency Patterns](../../docs/architecture/concurrency-patterns.md)
