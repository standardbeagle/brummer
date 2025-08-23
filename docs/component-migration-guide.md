# Component System Migration Guide

## Overview

This guide provides a step-by-step approach to migrating from the monolithic `Model` structure to the new component-based architecture in Brummer TUI.

## Architecture Comparison

### Old Architecture (Monolithic Model)
```
Model
├── All state
├── All rendering logic
├── All update logic
└── All navigation
```

### New Architecture (Component-Based)
```
ModelBridge
├── Component Tree (Root)
│   ├── BoxLayout (Vertical)
│   │   ├── Header Component
│   │   ├── ViewSwitcher
│   │   │   ├── ProcessView
│   │   │   ├── LogView
│   │   │   └── ...
│   │   └── Footer Component
│   └── Overlay Stack
└── Message Router
```

## Migration Strategy

### Phase 1: Enable Component System (Complete)
1. Set environment variable: `export BRUMMER_USE_COMPONENTS=true`
2. Components run alongside existing Model
3. No disruption to existing functionality

### Phase 2: Gradual View Migration
Migrate views one at a time:

#### Example: Migrating ProcessView

**Before (in Model):**
```go
func (m Model) renderProcesses() string {
    // Direct rendering logic
    var content strings.Builder
    for _, proc := range m.processes {
        content.WriteString(proc.Name)
    }
    return content.String()
}
```

**After (as Component):**
```go
type ProcessView struct {
    BaseComponent
    dataProvider DataProvider
}

func (p *ProcessView) View() string {
    processes := p.dataProvider.GetProcesses()
    // Use existing rendering logic
    return p.renderProcessList(processes)
}

func (p *ProcessView) Update(msg tea.Msg) (Component, tea.Cmd) {
    switch msg := msg.(type) {
    case ProcessUpdateMsg:
        // Handle process-specific updates
        return p, nil
    }
    return p.BaseComponent.Update(msg)
}
```

### Phase 3: Extract Shared State
Move shared state to DataProvider interface:

```go
type DataProvider interface {
    // Process data
    GetProcesses() []ProcessInfo
    GetProcessByID(id string) *ProcessInfo
    
    // Log data
    GetLogs(processID string) []LogEntry
    GetErrors() []ErrorEntry
    
    // URL data
    GetURLs() []URLInfo
    
    // Settings
    GetSettings() map[string]interface{}
    UpdateSetting(key string, value interface{}) error
}
```

### Phase 4: Implement Message Routing
Replace direct state updates with message passing:

```go
// Old way (direct state mutation)
m.processes = append(m.processes, newProcess)

// New way (message passing)
return c, func() tea.Msg {
    return ProcessAddedMsg{Process: newProcess}
}
```

## Component Creation Checklist

When creating a new component:

- [ ] Extend `BaseComponent` for standard functionality
- [ ] Implement `Component` interface methods
- [ ] Handle window sizing in `Update(tea.WindowSizeMsg)`
- [ ] Use message passing for state changes
- [ ] Register with ModelBridge if needed
- [ ] Add to ViewSwitcher for view components
- [ ] Enable color output: `lipgloss.SetColorProfile(termenv.ANSI256)`

## Common Patterns

### 1. Border-Box Sizing
Lipgloss adds borders outside the specified width. Use `BorderWindow`:

```go
contentArea := NewBorderWindow(content)
contentArea.SetBorderStyle(lipgloss.NormalBorder())
contentArea.SetSize(width, height) // Automatically adjusts for border
```

### 2. Tab Navigation
```go
case tea.KeyMsg:
    switch msg.String() {
    case "tab":
        return c, SwitchToNextViewCmd()
    case "shift+tab":
        return c, SwitchToPrevViewCmd()
    }
```

### 3. Responsive Layout
```go
func (c *MyComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        c.SetSize(msg.Width, msg.Height)
        c.redistributeSpace()
    }
    return c, nil
}
```

## Testing Components

### Unit Testing
```go
func TestComponent(t *testing.T) {
    comp := NewMyComponent()
    comp.SetSize(80, 24)
    
    // Test initial render
    view := comp.View()
    assert.Contains(t, view, "expected content")
    
    // Test update
    updated, cmd := comp.Update(MyMsg{})
    assert.NotNil(t, updated)
    assert.Nil(t, cmd)
}
```

### Integration Testing
```go
func TestComponentIntegration(t *testing.T) {
    shell := NewMinimalShell()
    shell.Init()
    
    // Simulate window resize
    sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
    model, _ := shell.Update(sizeMsg)
    
    // Test view switching
    keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
    model, _ = model.Update(keyMsg)
    
    view := model.View()
    assert.Contains(t, view, "Logs View")
}
```

## Performance Considerations

1. **Lazy Rendering**: Only render visible components
2. **Caching**: Cache expensive renders when data hasn't changed
3. **Batch Updates**: Group related updates into single messages
4. **Efficient Layouts**: Use BoxLayout for automatic space distribution

## Debugging Tips

1. **Enable Debug Mode**: Set `BRUMMER_DEBUG=true` for verbose logging
2. **Component Inspector**: Use MinimalShell for isolated component testing
3. **Message Tracing**: Log all messages in router for flow analysis
4. **Size Debugging**: Add size info to component renders during development

## Migration Timeline

- **Week 1-2**: Enable component system, test compatibility
- **Week 3-4**: Migrate simple views (Help, Settings)
- **Week 5-6**: Migrate complex views (Processes, Logs)
- **Week 7-8**: Extract shared state to DataProvider
- **Week 9-10**: Implement full message routing
- **Week 11-12**: Remove old Model code, optimize

## Rollback Strategy

If issues arise:
1. Unset `BRUMMER_USE_COMPONENTS` to disable
2. Components and Model can coexist during transition
3. Gradual migration allows partial rollback
4. Keep Model code until all components proven stable

## Common Pitfalls

1. **Forgetting Color Profile**: Always set `lipgloss.SetColorProfile(termenv.ANSI256)`
2. **Border Width**: Remember borders add 2 to width, 2 to height
3. **Direct State Mutation**: Use messages, not direct updates
4. **Missing Size Updates**: Handle WindowSizeMsg in all components
5. **Circular Dependencies**: Use interfaces to break cycles

## Support and Resources

- Example implementations in `/internal/tui/components/`
- MinimalShell for testing: `minimal_shell.go`
- BorderWindow for proper sizing: `border_window.go`
- ModelBridge for integration: `model_integration.go`

## Next Steps

1. Start with Phase 1 - enable components
2. Test thoroughly with existing functionality
3. Begin migrating simplest view first
4. Iterate and refine based on experience
5. Document any project-specific patterns discovered