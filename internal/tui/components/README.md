# Brummer Component System

This directory contains the new component-based UI architecture for Brummer TUI.

## Overview

The component system provides a modern, modular alternative to the monolithic Model approach, using the Bubbles component library for improved UI handling.

## Components

### View Components
- `process_view_adapter.go` - Process management UI
- `logs_view_adapter.go` - Log viewing with filtering
- `errors_view_adapter.go` - Error tracking and display
- `urls_view_adapter.go` - URL detection and management
- `web_view_adapter.go` - Web/proxy inspector
- `ai_coders_view_adapter.go` - AI coder sessions
- `settings_view_adapter.go` - Settings management
- `script_selector_adapter.go` - Script selection

### UI Components
- `command_input_adapter.go` - Command palette
- `notification_component.go` - System notifications
- `layout_component.go` - Layout management

### Core Infrastructure
- `component.go` - Component interfaces
- `base.go` - Base component implementation
- `messages.go` - Core message types
- `custom_messages.go` - Component-specific messages
- `view_switcher.go` - View navigation

## Usage

### Enable Component System

```bash
# Method 1: Environment variable
export BRUMMER_USE_COMPONENTS=true
./brum

# Method 2: Helper script
./run-with-components.sh
```

### Creating a Component

```go
type MyComponent struct {
    BaseComponent
    // Component state
}

func (c *MyComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
    // Handle messages
    return c, nil
}

func (c *MyComponent) View() string {
    // Render UI
    return "Component UI"
}
```

### Message Communication

```go
// Define custom message
type MyCustomMsg struct {
    Data string
}

// Send message
return c, func() tea.Msg {
    return MyCustomMsg{Data: "test"}
}

// Handle message in another component
case MyCustomMsg:
    // Process message
```

## Architecture

### Component Hierarchy
```
LayoutComponent
├── ViewSwitcher
│   ├── ProcessView
│   ├── LogsView
│   ├── ErrorsView
│   └── ...
├── NotificationComponent
└── CommandInputComponent
```

### Message Flow
1. User input → BubbleTea
2. BubbleTea → Component.Update()
3. Component generates custom messages
4. Messages routed to target components
5. Components update state
6. Component.View() renders UI

## Testing

```bash
# Run component tests
go test ./internal/tui/components

# Build with components
go build ./internal/tui/components

# Run example app
go run ./internal/tui/components/example_app.go
```

## Migration from Model

The `ModelComponentAdapter` provides a bridge between the old Model and new component system:

1. Components read data from existing Model
2. Feature flag controls which UI is used
3. Gradual migration possible
4. Both UIs maintained during transition

## Benefits

- **Modularity**: Self-contained components
- **Testability**: Components tested in isolation
- **Maintainability**: Clear separation of concerns
- **Modern UI**: Bubbles components for better UX
- **Type Safety**: Typed message system

## Status

✅ **Feature Complete** - All major components implemented
✅ **Tested** - Compiles and passes basic tests
✅ **Documented** - Comprehensive documentation
🚧 **Beta** - Ready for testing and feedback

## Next Steps

1. Integration testing with real usage
2. Performance optimization
3. Theme system implementation
4. Additional component variations
5. Full migration from Model