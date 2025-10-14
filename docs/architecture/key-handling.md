# Keyboard Handling Architecture

## Overview

The keyboard handling system in Brummer uses a **priority-based handler registry** that provides clear, predictable key routing with minimal complexity. This is part of the larger **Redux/Elm architecture pattern** where keyboard events become Messages that flow through the Update function.

**Architecture Context**: Keyboard handling follows the Redux/Elm pattern:
- **KeyMsg** → **Update()** → **KeyRegistry** → **Handler** → **Command** → **Message**
- Handlers are stateless and return Commands (never modify Model directly)
- All state changes happen in Update() when handling returned Messages

See [Redux/Elm Architecture Documentation](./redux-elm-architecture.md) for the complete pattern.

## Core Concepts

### 1. Single Point of Entry
All keyboard events flow through one place in `Model.Update()`:
```go
case tea.KeyMsg:
    return m, m.keyRegistry.HandleKey(msg, m.currentView(), m.getContext(), m)
```

### 2. Handler-Based Registration (Not Key-Based)
**IMPORTANT**: We register **handlers**, not specific keys. Each handler decides at runtime which keys it wants to handle based on the current context.

```go
// We DON'T do this:
registry.RegisterKey("ctrl+c", quitHandler)  // ❌ Not how it works

// We DO this:
registry.Register(quitHandler)  // ✅ Handler decides what keys it handles
```

This means:
- No registration conflicts are possible
- Handlers can dynamically decide what to handle based on context
- The same key can be handled differently by different handlers based on view/context
- Priority determines which handler gets first chance at a key

### 3. Handler Interface
Every key handler implements the same simple interface:
```go
type KeyHandler interface {
    CanHandle(key tea.KeyMsg, ctx KeyContext) bool  // Decides if this handler wants this key
    Handle(key tea.KeyMsg, model *Model) tea.Cmd    // Processes the key
    Priority() int                                   // Determines check order
    Name() string                                    // For debugging
}
```

### 4. Priority-Based Routing
When a key is pressed:
1. Registry iterates through handlers in priority order (lowest number first)
2. First handler that returns `true` from `CanHandle()` gets the key
3. That handler's `Handle()` method is called
4. No other handlers are checked

Priority ranges:
- **100**: Global navigation (number keys, Tab)
- **200**: View-specific handlers
- **300**: Command palette triggers
- **1000**: Quit handler (lowest priority)

## Adding a New Key Handler

### Step 1: Create Your Handler
```go
type MyCustomHandler struct {
    // your fields
}

func (h *MyCustomHandler) Name() string {
    return "MyCustom"
}

func (h *MyCustomHandler) Priority() int {
    return 250 // Between view-specific and command palette
}

func (h *MyCustomHandler) CanHandle(key tea.KeyMsg, view ViewType, context string) bool {
    // Return true if you want to handle this key
    return key.Type == tea.KeyCtrlP && view == ViewProcesses
}

func (h *MyCustomHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
    // Do something and return a command (or nil)
    return func() tea.Msg {
        return MyCustomMsg{Action: "something"}
    }
}
```

### Step 2: Register Your Handler
In `Model` initialization:
```go
m.keyRegistry.Register(&MyCustomHandler{})
```

## Common Handler Patterns

### Global Keys (Available Everywhere)
```go
func (h *GlobalHandler) CanHandle(key tea.KeyMsg, view ViewType, context string) bool {
    return key.Type == tea.KeyCtrlS // Handle Ctrl+S everywhere
}
```

### View-Specific Keys
```go
func (h *ProcessViewHandler) CanHandle(key tea.KeyMsg, view ViewType, context string) bool {
    return view == ViewProcesses && key.Type == tea.KeyEnter
}
```

### Context-Aware Keys
```go
func (h *ModalHandler) CanHandle(key tea.KeyMsg, view ViewType, context string) bool {
    return context == "modal" && key.Type == tea.KeyEsc
}
```

### Conditional Keys
```go
func (h *ConditionalHandler) CanHandle(key tea.KeyMsg, view ViewType, context string) bool {
    // Only handle 'r' key when a process is selected
    return view == ViewProcesses &&
           key.Type == tea.KeyRunes &&
           len(key.Runes) == 1 &&
           key.Runes[0] == 'r'
}

func (h *ConditionalHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
    if model.hasSelectedProcess() {
        return restartProcess(model.selectedProcess)
    }
    return nil // No-op if no process selected
}
```

## Handler Priority Guidelines

Use these priority ranges for consistency:

| Priority | Purpose | Examples |
|----------|---------|----------|
| 0-99 | Emergency/System | Force quit, panic recovery |
| 100-199 | Global Navigation | Number keys, Tab, view switching |
| 200-299 | View-Specific Actions | Enter in process view, arrows in logs |
| 300-399 | Overlays/Modals | Command palette, dialogs |
| 400-499 | Context Menus | Right-click menus, dropdowns |
| 500-599 | Input Fields | Text input, form fields |
| 1000+ | Fallback/Default | Quit handler, help |

## Debugging Key Handling

### Enable Key Logging
The registry automatically logs which handler processes each key:
```
[keyboard] GlobalNavigation: Handling key: Tab
[keyboard] ProcessViewHandler: Handling key: Enter
```

### List Registered Handlers
```go
for _, handler := range m.keyRegistry.handlers {
    fmt.Printf("%d: %s\n", handler.Priority(), handler.Name())
}
```

### Trace Key Flow
Add logging to your CanHandle method:
```go
func (h *MyHandler) CanHandle(key tea.KeyMsg, view ViewType, context string) bool {
    result := key.Type == tea.KeyEnter
    log.Printf("MyHandler.CanHandle: key=%v view=%v result=%v",
               key, view, result)
    return result
}
```

## Migration from Old System

### Before (Complex)
```go
// Key goes through multiple layers:
Model.Update() → MessageRouter → ViewMessageHandler →
InputComponent → InputService → Handler

// Multiple places could handle the same key
// Commands could get lost between layers
// Hard to debug which handler processed a key
```

### After (Simple)
```go
// Key goes through one place:
Model.Update() → KeyRegistry → Handler

// Clear precedence based on priority
// Commands always returned properly
// Easy to see which handler processed a key
```

## Benefits of This Approach

1. **Predictable**: Priority determines order, no surprises
2. **Debuggable**: Clear logging shows which handler processes each key
3. **Extensible**: Just implement interface and register
4. **Testable**: Each handler can be unit tested independently
5. **Maintainable**: All key handling logic in one place per handler
6. **Performant**: Simple array iteration, no complex routing

## Common Pitfalls to Avoid

1. **Don't create overlapping handlers** at the same priority level
2. **Don't forget to return nil** from Handle() if you don't want an update
3. **Don't modify model state directly** - return commands instead
4. **Don't handle keys in View()** methods - only in handlers
5. **Don't create circular dependencies** between handlers

## Context-Aware Key Handling

The key handler system supports rich context awareness for sophisticated behaviors. This is crucial for features like the slash command that behaves differently based on the active component state.

### The KeyContext Structure

```go
type KeyContext struct {
    View            ViewType    // Current view (ViewProcesses, ViewAICoders, etc.)
    Context         string      // Additional context ("modal", "input", "pty")
    ActiveComponent interface{} // The active component (for state checks)
    CursorPosition  int        // Cursor position in input
    InputContent    string     // Current input content
}
```

### Example: Context-Aware Slash Command

The slash command (`/`) needs different behavior depending on context:
- In AI Coder PTY mode at start of line → Open command palette
- In AI Coder PTY mode mid-line → Pass to terminal
- In AI Coder list mode → Open command palette
- In other views → Open command palette

```go
type SlashCommandHandler struct {
    openCommandPalette func() tea.Cmd
}

func (h *SlashCommandHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
    // Only handle "/" key
    if key.Type != tea.KeyRunes || len(key.Runes) != 1 || key.Runes[0] != '/' {
        return false
    }

    // Context-aware behavior
    if ctx.View == ViewAICoders {
        if ctx.Context == "pty" {
            // In PTY mode, only handle if at start of line
            return ctx.CursorPosition == 0
        }
        // In list mode, always handle
        return true
    }

    // In all other views, always handle
    return true
}
```

### Using the Context in Model.Update()

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Build context for the key handler
        ctx := KeyContext{
            View:    m.currentView(),
            Context: m.getKeyContext(),
        }

        // Special handling for AI Coder view
        if m.currentView() == ViewAICoders && m.aiCoderComponent != nil {
            aiCoder := m.aiCoderComponent.(*AICodersViewComponent)
            if aiCoder.viewState.Mode == ViewModePTY {
                ctx.Context = "pty"
                ctx.CursorPosition = m.getCursorPosition()
                ctx.InputContent = m.getCurrentInput()
            }
        }

        // Route to handlers with context
        return m, m.keyRegistry.HandleKey(msg, ctx, m)
    }
}
```

### Component Visibility and State Checks

Handlers can check component visibility and state through the context:

```go
func (h *MyHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
    // Check if a modal is open
    if ctx.Context == "modal" {
        return false // Don't handle when modal is active
    }

    // Check component-specific state
    if ctx.View == ViewAICoders && ctx.ActiveComponent != nil {
        aiCoder := ctx.ActiveComponent.(*AICodersViewComponent)
        if !aiCoder.IsVisible() {
            return false
        }
        // Check specific component state
        if aiCoder.viewState.Mode == ViewModePTY {
            // Special PTY mode handling
        }
    }

    return true
}
```

## Example: Complete Handler Implementation

Here's a complete example of adding a search feature triggered by `/`:

```go
// search_handler.go
type SearchHandler struct {
    searchService *SearchService
}

func NewSearchHandler(searchService *SearchService) KeyHandler {
    return &SearchHandler{searchService: searchService}
}

func (h *SearchHandler) Name() string {
    return "Search"
}

func (h *SearchHandler) Priority() int {
    return 300 // Overlay priority
}

func (h *SearchHandler) CanHandle(key tea.KeyMsg, view ViewType, context string) bool {
    // Handle '/' key when not in input field
    return context != "input" &&
           key.Type == tea.KeyRunes &&
           len(key.Runes) == 1 &&
           key.Runes[0] == '/'
}

func (h *SearchHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
    return func() tea.Msg {
        // Open search overlay
        return OpenSearchMsg{}
    }
}

// In model initialization:
m.keyRegistry.Register(NewSearchHandler(searchService))
```

This approach makes it trivial to add, modify, or remove key handlers without affecting the rest of the system.