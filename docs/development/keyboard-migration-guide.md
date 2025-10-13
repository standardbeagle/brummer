# Keyboard Handling Migration Guide

## Overview

The keyboard handling system has been completely simplified from a complex multi-layer architecture to a single, priority-based KeyHandlerRegistry. This guide explains how to migrate from the old system to the new one.

## What Changed

### Before: Complex Multi-Layer System
```
KeyEvent → Model.Update → MessageRouter → ViewSpecificHandler → InputComponent → InputService → Handler
```

**Problems:**
- Commands (tea.Cmd) were being discarded at multiple points
- Triple processing of keyboard events in different layers
- Difficult to debug which handler processed a key
- Context awareness was scattered across multiple components
- Registration logic was complex and error-prone

### After: Simple Registry Pattern
```
KeyEvent → Model.Update → KeyHandlerRegistry → Handler
```

**Benefits:**
- Single point of entry for all keyboard events
- Clear priority-based routing (lower number = higher priority)
- Commands are properly returned through the chain
- Built-in logging shows which handler processes each key
- Rich context awareness through KeyContext struct
- Easy to add, modify, or remove handlers

## Migration Steps

### 1. Stop Using InputComponent for Keyboard Registration

**Old Way (DEPRECATED):**
```go
// In model initialization
inputComponent := components.NewInputComponent(inputService, navService)
inputComponent.RegisterGlobalHandler(handleGlobalKeys)
inputComponent.registerContextHandler("processes", handleProcessKeys)
```

**New Way:**
```go
// In model initialization
m.keyRegistry = NewKeyHandlerRegistry()
m.keyRegistry.Register(NewGlobalNavigationHandler(navService))
m.keyRegistry.Register(NewProcessViewKeyHandler())
```

### 2. Update Model.Update() Method

**Old Way (REMOVE):**
```go
case tea.KeyMsg:
    // Route through message router
    return m.messageRouter.Route(msg, m)
```

**New Way:**
```go
case tea.KeyMsg:
    ctx := m.buildKeyContext()
    if m.keyRegistry != nil {
        return m, m.keyRegistry.HandleKey(msg, ctx, m)
    }
    return m, nil
```

### 3. Create New Key Handlers

**Template for a New Handler:**
```go
type MyCustomHandler struct {
    // your fields
}

func NewMyCustomHandler() KeyHandler {
    return &MyCustomHandler{}
}

func (h *MyCustomHandler) Name() string {
    return "MyCustom"
}

func (h *MyCustomHandler) Priority() int {
    return 250 // Between view-specific (200) and overlays (300)
}

func (h *MyCustomHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
    // Return true if this handler should process this key
    return ctx.View == ViewProcesses && key.String() == "r"
}

func (h *MyCustomHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
    // Process the key and return a command (or nil)
    return func() tea.Msg {
        return RestartProcessMsg{}
    }
}
```

### 4. Register Your Handler

```go
// In model initialization
m.keyRegistry.Register(NewMyCustomHandler())
```

## Context Awareness

The new system provides rich context through the KeyContext struct:

```go
type KeyContext struct {
    View            ViewType    // Current view (ViewProcesses, ViewAICoders, etc.)
    Context         string      // Additional context ("modal", "input", "pty")
    ActiveComponent interface{} // The active component (for state checks)
    CursorPosition  int        // Cursor position in input
    InputContent    string     // Current input content
}
```

This allows sophisticated context-aware behaviors like the slash command:

```go
func (h *SlashCommandHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
    if key.Runes[0] != '/' {
        return false
    }

    // In AI Coder PTY mode at start of line → Open command palette
    // In AI Coder PTY mode mid-line → Pass to terminal
    if ctx.View == ViewAICoders && ctx.Context == "pty" {
        return ctx.CursorPosition == 0
    }

    // In other contexts → Always open command palette
    return true
}
```

## Priority Guidelines

Use these standard priorities for consistency:

| Priority Range | Purpose | Examples |
|---------------|---------|----------|
| 0-99 | Emergency/System | Force quit, panic recovery |
| 100-199 | Global Navigation | Number keys, Tab, view switching |
| 200-299 | View-Specific Actions | Enter in process view, arrows in logs |
| 300-399 | Overlays/Modals | Command palette, dialogs |
| 400-499 | Context Menus | Right-click menus, dropdowns |
| 500-599 | Input Fields | Text input, form fields |
| 1000+ | Fallback/Default | Quit handler, help |

## Debugging

### Enable Key Logging
The registry automatically logs which handler processes each key:
```
[keyboard] GlobalNavigation: Handling key: Tab
[keyboard] ProcessViewHandler: Handling key: Enter
```

### List Registered Handlers
```go
// In your debug code
for _, handler := range m.keyRegistry.handlers {
    fmt.Printf("%d: %s\n", handler.Priority(), handler.Name())
}
```

## Deprecated APIs

The following APIs are deprecated and will be removed in a future version:

### InputService Methods (DEPRECATED)
- `SubscribeToGlobalKeys()` - Use KeyHandlerRegistry instead
- `SubscribeWithContextKeys()` - Use KeyHandlerRegistry instead
- `RouteKeyMessage()` - Still used internally but should not be called directly

### InputComponent Methods (DEPRECATED)
- `RegisterGlobalHandler()` - Register handlers with KeyHandlerRegistry
- `registerContextHandler()` - Register handlers with KeyHandlerRegistry

### ViewSpecificHandler (DEPRECATED)
- Entire handler is deprecated - keyboard handling moved to KeyHandlerRegistry

## Complete Example: Process View Handler

```go
// process_view_handler.go
package tui

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
)

type ProcessViewKeyHandler struct {
    processMgr *ProcessManager
}

func NewProcessViewKeyHandler(processMgr *ProcessManager) KeyHandler {
    return &ProcessViewKeyHandler{processMgr: processMgr}
}

func (h *ProcessViewKeyHandler) Name() string {
    return "ProcessView"
}

func (h *ProcessViewKeyHandler) Priority() int {
    return 200 // View-specific priority
}

func (h *ProcessViewKeyHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
    // Only handle keys when in processes view
    if ctx.View != ViewProcesses {
        return false
    }

    // Handle specific keys for this view
    switch key.String() {
    case "s", "r", "enter", "up", "down", "j", "k":
        return true
    }
    return false
}

func (h *ProcessViewKeyHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
    switch key.String() {
    case "s": // Stop process
        return h.stopSelectedProcess(model)
    case "r": // Restart process
        return h.restartSelectedProcess(model)
    case "enter": // View process details
        return h.viewProcessDetails(model)
    case "up", "k": // Navigate up
        model.processesView.NavigateUp()
        return nil
    case "down", "j": // Navigate down
        model.processesView.NavigateDown()
        return nil
    }
    return nil
}

func (h *ProcessViewKeyHandler) stopSelectedProcess(model *Model) tea.Cmd {
    if process := model.getSelectedProcess(); process != nil {
        return func() tea.Msg {
            if err := h.processMgr.StopProcess(process.ID); err != nil {
                return ErrorMsg{Error: err}
            }
            return ProcessStoppedMsg{ProcessID: process.ID}
        }
    }
    return nil
}

// ... other helper methods
```

## Testing the New System

```go
func TestKeyHandler(t *testing.T) {
    // Create test model
    model := &Model{
        keyRegistry: NewKeyHandlerRegistry(),
    }

    // Register test handler
    handled := false
    testHandler := &testKeyHandler{
        handleFunc: func(key tea.KeyMsg, m *Model) tea.Cmd {
            handled = true
            return nil
        },
    }
    model.keyRegistry.Register(testHandler)

    // Test key handling
    ctx := KeyContext{View: ViewProcesses}
    cmd := model.keyRegistry.HandleKey(
        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}},
        ctx,
        model,
    )

    assert.True(t, handled)
}
```

## How It Works: Handler-Based Registration

Unlike traditional key mapping systems, we don't register specific keys. Instead:

1. **Register handlers** that implement the `KeyHandler` interface
2. **Handlers decide** what keys they want to handle via `CanHandle()`
3. **Priority determines order** - first handler to claim a key wins
4. **No conflicts possible** - handlers are checked in deterministic order

```go
// Traditional approach (NOT how we do it):
keymap["ctrl+c"] = quitFunction     // ❌ Static key mapping
keymap["tab"] = switchViewFunction  // ❌ Can conflict

// Our approach:
registry.Register(quitHandler)      // ✅ Handler decides what it handles
registry.Register(navigationHandler) // ✅ Can coexist peacefully
```

This means:
- Same key can behave differently in different contexts
- Handlers can have complex logic for when they handle keys
- No registration-time conflicts
- Dynamic behavior based on application state

## Benefits Summary

1. **Simpler**: One registry, one routing path, no command loss
2. **Predictable**: Priority determines order, no surprises
3. **Debuggable**: Clear logging shows handler processing
4. **Extensible**: Just implement interface and register
5. **Testable**: Each handler can be unit tested independently
6. **Maintainable**: All key handling logic in one place per handler
7. **Context-Aware**: Rich context for sophisticated behaviors
8. **Conflict-Free**: Handler-based registration prevents conflicts

## Questions?

See the [key-handling.md](../architecture/key-handling.md) documentation for complete architectural details.