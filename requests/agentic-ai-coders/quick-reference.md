# AI Coder PTY Integration - Quick Reference

## Key Components Map

### PTY System (`internal/aicoder/`)
```
pty_session.go      → Core PTY terminal emulation
pty_manager.go      → Multi-session management  
data_injector.go    → Brummer data formatting
brummer_data_provider.go → Interface for data access
```

### TUI Integration Points (`internal/tui/`)
```
model.go            → Main TUI model (needs PTY fields)
ai_coder_pty_view.go → PTY view component (already built)
ai_coder_view.go    → Old view (to be replaced)
```

### Manager Integration
```
manager.go          → AICoderManager with PTY methods
                     - CreateInteractiveCLISession()
                     - CreateTaskCLISession()
                     - GetPTYManager()
```

## Command Flow

### Interactive Mode (SSH-like)
```
/ai claude
  ↓
CreateInteractiveCLISession("claude")
  ↓
PTYSession with NO --print flags
  ↓
Full terminal experience
```

### Task Mode (Structured Output)
```
/ai claude "fix the tests"
  ↓
CreateTaskCLISession("claude", "fix the tests")
  ↓
PTYSession with --print --verbose --output-format stream-json
  ↓
Parsed streaming output
```

## Key Bindings (When Terminal Focused)

### Global Controls
- `F11` - Toggle full-screen mode
- `Ctrl+H` - Show/hide help
- `Ctrl+N` - Next PTY session
- `Ctrl+P` - Previous PTY session
- `Ctrl+D` - Detach from session
- `ESC` - Unfocus terminal

### Data Injection (Contextual)
- `Ctrl+E` - Inject last error
- `Ctrl+L` - Inject recent logs
- `Ctrl+T` - Inject test failure
- `Ctrl+B` - Inject build output
- `Ctrl+U` - Inject detected URLs
- `Ctrl+R` - Inject proxy request

## Integration Checklist

### TUI Model Updates
- [ ] Add `aiCoderPTYView *AICoderPTYView` field
- [ ] Add `ptyManager *aicoder.PTYManager` field  
- [ ] Add `ptyEventSub chan aicoder.PTYEvent` field
- [ ] Initialize in NewModel() with data provider
- [ ] Route View 8 to PTY view

### Event Flow Setup
- [ ] Create `TUIDataProvider` implementation
- [ ] Subscribe to PTY output events
- [ ] Convert PTY events to tea.Msg
- [ ] Handle output streaming
- [ ] Clean up on shutdown

### Command Integration
- [ ] Update `/ai` command handler
- [ ] Parse provider and optional task
- [ ] Create appropriate PTY session
- [ ] Auto-switch to AI Coder tab
- [ ] Attach view to new session

## Common Gotchas

### Thread Safety
```go
// WRONG - Direct access from goroutine
go func() {
    m.logStore.Add(...) // RACE CONDITION
}()

// RIGHT - Use tea.Cmd
return func() tea.Msg {
    return logAddMsg{...}
}
```

### PTY Lifecycle
```go
// Always check session exists
if session != nil && session.IsActive {
    session.WriteInput(data)
}

// Clean up properly
defer session.Close()
```

### Event Subscriptions
```go
// Re-subscribe after each event
case PTYOutputMsg:
    // Process output
    return m, m.subscribeToActivePTY() // Re-subscribe
```

## Debug Commands

### Check PTY Status
```go
// In TUI debug mode
sessions := m.ptyManager.ListSessions()
current := m.aiCoderPTYView.currentSession
```

### Force Cleanup
```go
// Emergency cleanup
if m.ptyManager != nil {
    m.ptyManager.CloseAllSessions()
}
```

## File Structure After Integration

```
internal/
├── aicoder/
│   ├── pty_session.go         # ✓ Complete
│   ├── pty_manager.go         # ✓ Complete
│   ├── data_injector.go       # ✓ Complete
│   └── brummer_data_provider.go # ✓ Complete
└── tui/
    ├── model.go               # ⚠️ Needs PTY integration
    ├── ai_coder_pty_view.go   # ✓ Complete
    ├── brummer_data_provider_impl.go # 🔲 To create
    └── pty_events.go          # 🔲 To create
```

## Next Immediate Steps

1. Create `brummer_data_provider_impl.go` to bridge TUI ↔ PTY
2. Add PTY fields to Model struct
3. Update View() case 8 to use PTY view
4. Test with `/ai claude` command

Remember: The goal is to make AI coders feel like SSH sessions with magical Brummer data injection capabilities!