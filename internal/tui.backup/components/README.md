# TUI Components Architecture

**Last Updated**: 2025-10-07
**Migration Status**: Phase 3.6 Complete (11/13 controllers migrated)

## Overview

This directory contains BubbleTea components that implement the TUI (Terminal User Interface) for Brummer. Components follow a clean 2-layer architecture where components call services directly, eliminating the need for controller middleware.

**Architecture**: Components → Services (no controllers between)

## Component Interface

All TUI components implement the standard BubbleTea `tea.Model` interface:

```go
type Component interface {
    Init() tea.Cmd              // Initialize component, return initial command
    Update(tea.Msg) (tea.Model, tea.Cmd)  // Handle messages, return updated model and command
    View() string               // Render component to string
}
```

Additional component methods for integration:

```go
// Size management
SetSize(width, height int)    // Update component dimensions
GetSize() (width, height int) // Get current dimensions

// Visibility control
IsVisible() bool              // Check if component should render
SetVisible(visible bool)      // Show/hide component

// Focus management (for input components)
Focus() tea.Cmd               // Give keyboard focus
Blur() tea.Cmd                // Remove keyboard focus
```

## Component Types

### View Components

Display-only components that show data from services:

- **`processes_view.go`**: Process list with status, selection, control
- **`logs_view.go`**: Log display with filtering, auto-scroll, search
- **`errors_view.go`**: Error list with dismissal, details, clearing
- **`urls_view.go`**: Detected URL list with copying, opening
- **`web_view.go`**: Proxy request viewer with filtering, details
- **`settings_view.go`**: Settings editor with save/cancel
- **`mcp_debug_view.go`**: MCP connection status and diagnostics
- **`aicoders_view.go`**: AI coder session list and status

**Pattern**:
```go
type ProcessesViewComponent struct {
    BaseComponent                // Embedded base for common functionality
    viewport viewport.Model      // Bubble Tea viewport for scrolling
    processMgr *process.Manager  // Direct service reference
    // ... component-specific state
}
```

### Interactive Components

Components that accept keyboard input and modify state:

- **`command_palette.go`**: Command input with autocomplete
- **`script_selector.go`**: Script selection with search, launch
- **`hierarchical_key_handler.go`**: Priority-based key routing

**Pattern**:
```go
type CommandPaletteComponent struct {
    BaseComponent
    input textinput.Model        // Bubble Tea text input
    autocomplete *Autocomplete   // Custom autocomplete logic
    processMgr *process.Manager  // Service for command execution
}
```

### Layout Components

Components that compose other components:

- **`integrated_shell.go`**: Top-level shell routing messages to view components
- **`layout_manager.go`**: Pure functions for calculating layouts

**Pattern**:
```go
type IntegratedShell struct {
    // View components
    logsView Component
    processesView Component
    // ... other views

    currentView string  // Active view name
}
```

## Service Integration

Components call services directly without controller intermediaries:

```go
// GOOD: Direct service call from component
func (c *ProcessesViewComponent) handleStart(processName string) tea.Cmd {
    return func() tea.Msg {
        err := c.processMgr.StartScript(processName)
        if err != nil {
            return ProcessStartErrorMsg{Name: processName, Err: err}
        }
        return ProcessStartedMsg{Name: processName}
    }
}

// BAD: Controller wrapper (removed in migration)
func (c *ProcessViewController) StartProcess(name string) {
    c.processManager.StartScript(name) // Extra layer of indirection
}
```

### Available Services

Components can access these services:

- **`process.Manager`**: Process lifecycle, script execution, status
- **`logs.Store`**: Log storage, retrieval, filtering
- **`proxy.Server`**: HTTP request capture, viewing
- **`config.Config`**: Application configuration
- **`events.EventBus`**: Cross-component event coordination
- **`aicoder.AICoderManager`**: AI coding session management
- **`aicoder.PTYManager`**: Terminal emulation

## Message Patterns

### Service → Component Messages

Services send messages to components via BubbleTea's message system:

```go
// Process state changes
type ProcessStartedMsg struct {
    ProcessID string
    Name string
}

type ProcessStoppedMsg struct {
    ProcessID string
    ExitCode int
}

// Log updates
type LogAddedMsg struct {
    ProcessID string
    Message string
    IsError bool
}

// URL detection
type URLDetectedMsg struct {
    ProcessID string
    URL string
}
```

### Component → Service Calls

Components return `tea.Cmd` functions that call services:

```go
func (c *Component) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "enter" {
            // Return command that calls service
            return c, c.startProcess()
        }
    }
    return c, nil
}

func (c *Component) startProcess() tea.Cmd {
    return func() tea.Msg {
        // Call service directly
        err := c.processMgr.StartScript(c.selectedScript)
        if err != nil {
            return ErrorMsg{Err: err}
        }
        return SuccessMsg{}
    }
}
```

## Component Lifecycle

### Initialization

```go
// 1. Create component with service dependencies
component := NewProcessesViewComponent(processMgr, logStore)

// 2. Set initial size
component.SetSize(80, 24)

// 3. Call Init() to get initial command
cmd := component.Init()

// 4. Send initial command to BubbleTea
program.Send(cmd)
```

### Message Handling

```go
// BubbleTea calls Update() for each message
newModel, cmd := component.Update(msg)

// Update returns:
// - Updated model (component state)
// - Command to execute (or nil)

// Common patterns:
switch msg := msg.(type) {
case tea.WindowSizeMsg:
    // Resize component
    component.SetSize(msg.Width, msg.Height)
    return component, nil

case tea.KeyMsg:
    // Handle keyboard input
    return component.handleKey(msg)

case ServiceMsg:
    // Handle service events
    return component.handleServiceEvent(msg)
}
```

### Rendering

```go
// BubbleTea calls View() to render
output := component.View()

// View() should:
// 1. Check if visible
// 2. Build layout with lipgloss
// 3. Return string for terminal

func (c *Component) View() string {
    if !c.IsVisible() {
        return ""
    }

    // Use lipgloss for styling
    return lipgloss.NewStyle().
        Width(c.width).
        Height(c.height).
        Render(c.buildContent())
}
```

## Key Handling Architecture

### Hierarchical Key Handler

Components use priority-based key handling:

```go
type KeyHandler interface {
    CanHandle(msg tea.KeyMsg) bool
    Handle(msg tea.KeyMsg) (tea.Model, tea.Cmd)
    Priority() KeyHandlerPriority
    Name() string
}

// Priority levels (highest to lowest):
const (
    PriorityHighest  // System keys (quit, force quit)
    PriorityOverlay  // Modal overlays
    PriorityFocused  // Currently focused component
    PriorityView     // Current view handlers
    PriorityGlobal   // Global shortcuts
    PriorityLowest   // Catch-all
)
```

### Example Key Handler

```go
type ProcessViewKeyHandler struct {
    component *ProcessesViewComponent
}

func (h *ProcessViewKeyHandler) CanHandle(msg tea.KeyMsg) bool {
    // Check if this handler should process the key
    return msg.String() == "enter" || msg.String() == "s"
}

func (h *ProcessViewKeyHandler) Handle(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "enter":
        return h.component.startSelected()
    case "s":
        return h.component.stopSelected()
    }
    return h.component, nil
}

func (h *ProcessViewKeyHandler) Priority() KeyHandlerPriority {
    return PriorityView
}

func (h *ProcessViewKeyHandler) Name() string {
    return "ProcessViewKeys"
}
```

## Testing Components

### Unit Testing

Test components in isolation:

```go
func TestProcessesView_SelectProcess(t *testing.T) {
    // Create mock service
    mockMgr := &MockProcessManager{
        processes: []*process.Process{
            {ID: "test-1", Name: "test"},
        },
    }

    // Create component
    component := NewProcessesViewComponent(mockMgr, nil)

    // Send key message
    newModel, _ := component.Update(tea.KeyMsg{
        Type: tea.KeyDown,
    })

    // Verify state change
    c := newModel.(*ProcessesViewComponent)
    assert.Equal(t, 0, c.selectedIndex)
}
```

### Integration Testing

Test component with real services:

```go
func TestProcessesView_Integration(t *testing.T) {
    // Create real service
    processMgr := process.NewManager()
    logStore := logs.NewStore()

    // Create component
    component := NewProcessesViewComponent(processMgr, logStore)

    // Send messages
    component.Update(ProcessStartedMsg{ProcessID: "test-1"})

    // Verify service calls
    processes := processMgr.GetAllProcesses()
    assert.Len(t, processes, 1)
}
```

## Best Practices

### ✅ DO

- **Call services directly** from components
- **Use tea.Cmd** for async operations
- **Return immutable state** from Update()
- **Use lipgloss** for styling and layout
- **Implement BaseComponent** for common functionality
- **Handle window resize** with tea.WindowSizeMsg
- **Use viewport** for scrolling content
- **Test components** with mock services

### ❌ DON'T

- **Don't create controllers** between components and services
- **Don't mutate state directly** (return new state)
- **Don't block in Update()** (use tea.Cmd for long operations)
- **Don't store UI state in services** (keep in components)
- **Don't skip error handling** from service calls
- **Don't hardcode dimensions** (use SetSize)

## Migration from Controllers

If migrating from controller-based code:

### Before (Controller Pattern)
```go
// Controller middleware
type ProcessViewController struct {
    manager *process.Manager
}

func (c *ProcessViewController) StartProcess(name string) {
    c.manager.StartScript(name)
}

// Component delegates to controller
func (c *ProcessView) handleStart() {
    c.controller.StartProcess(c.selected)
}
```

### After (Component Pattern)
```go
// Component calls service directly
type ProcessesViewComponent struct {
    manager *process.Manager
}

func (c *ProcessesViewComponent) handleStart() tea.Cmd {
    return func() tea.Msg {
        err := c.manager.StartScript(c.selected)
        if err != nil {
            return ErrorMsg{Err: err}
        }
        return SuccessMsg{}
    }
}
```

**Benefits**:
- Fewer layers (2 instead of 4)
- Clearer data flow
- Easier testing
- Better BubbleTea integration

## Component Checklist

When creating a new component:

- [ ] Implement tea.Model interface (Init, Update, View)
- [ ] Embed BaseComponent for common functionality
- [ ] Add service dependencies in constructor
- [ ] Implement SetSize for responsive layout
- [ ] Handle tea.WindowSizeMsg
- [ ] Use lipgloss for styling
- [ ] Return tea.Cmd for async operations
- [ ] Implement KeyHandler if accepting input
- [ ] Write unit tests with mocks
- [ ] Write integration tests with real services
- [ ] Document public methods
- [ ] Add to integrated_shell.go routing

## Further Reading

- [BubbleTea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Brummer Architecture Overview](/docs/architecture/overview.md)
- [Controller Migration Guide](/docs/architecture/controller-migration.md)
- [Concurrency Patterns](/docs/architecture/concurrency-patterns.md)

---

*This documentation reflects the completed Phase 3.6 migration (11/13 controllers migrated)*
