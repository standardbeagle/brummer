# Stateless Service Architecture Design

## Overview
This document outlines the architecture for completely eliminating goroutines from the TUI stack and making all services stateless with the core Model as the single source of truth.

## Current Goroutine Analysis

### Files with Goroutines (Production Code)
1. **PTY Integration Layer**:
   - `internal/tui/pty_termination.go` - Process termination monitoring
   - `internal/tui/pty_integration.go` - PTY process management
   - `internal/tui/pty_main_wrapper.go` - Main PTY wrapper logic

2. **Service Layer**:
   - `internal/tui/services/input_service.go` - Input handling service
   - `internal/tui/services/pty_input_adapter.go` - PTY input adaptation
   - `internal/tui/services/pty_navigation_adapter.go` - Navigation handling
   - `internal/tui/services/aicoder_service.go` - AI coder functionality
   - `internal/tui/services/notifications_service.go` - Notification management

3. **Component Layer**:
   - `internal/tui/components/integrated_shell.go` - Shell integration
   - `internal/tui/components/script_selector.go` - Script selection UI

4. **Command Infrastructure**:
   - `internal/tui/commands/context.go` - Operation context handling
   - `internal/tui/commands/goroutine_helper.go` - Goroutine utilities (TO BE REMOVED)
   - `internal/tui/commands/batch.go` - Batch command execution

5. **State Management**:
   - `internal/tui/state/cache.go` - Cache management
   - `internal/tui/handlers/progress_handler.go` - Progress tracking

6. **Utilities**:
   - `internal/tui/goroutine_helper.go` - General goroutine utilities (TO BE REMOVED)

## Stateless Service Architecture

### Core Principles

1. **Model as Single Source of Truth**
   - All application state lives in the central Model struct
   - Services never store state internally
   - All data reads come from the Model
   - All state changes go through Bubble Tea Update()

2. **Pure Function Services**
   - Services contain only pure functions
   - No internal state or caching in services
   - Functions take parameters and return results
   - No side effects outside of returned values

3. **Command Pattern for Async Operations**
   - All async operations return `tea.Cmd`
   - No direct goroutine creation
   - Bubble Tea manages all concurrency
   - Results flow back through messages

4. **Event-Driven Communication**
   - Components communicate through immutable messages
   - No direct service-to-service calls
   - All communication mediated by the Model

### Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    Bubble Tea Runtime                        │
│                 (Only goroutines allowed)                    │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                     Model (Master Data)                      │
│  - Single source of truth                                   │
│  - All application state                                     │
│  - Immutable update patterns                                │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                Command Layer (tea.Cmd)                       │
│  - Async operations                                          │
│  - No direct goroutines                                     │
│  - Framework-managed execution                              │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│              Stateless Services (Pure Functions)             │
│  - No internal state                                        │
│  - Pure functions only                                      │
│  - Take parameters, return results                          │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                Components (View Layer)                       │
│  - Read-only access to Model                                │
│  - Generate commands for user actions                       │
│  - No direct state mutations                                │
└─────────────────────────────────────────────────────────────┘
```

### Migration Strategy

#### Phase 1: Master Data Object Design
1. **Enhanced Model State** (`internal/tui/state/master_data.go`)
   - Centralize all state management
   - Add atomic operations for thread safety
   - Implement version tracking for changes

2. **Service Interfaces** (`internal/tui/services/interfaces.go`)
   - Define stateless service interfaces
   - Pure function signatures only
   - No internal state methods

#### Phase 2: PTY Integration Refactoring
1. **Async PTY Operations**
   - Replace goroutines with tea.Cmd
   - Process management through commands
   - PTY events as messages to Model

2. **Process Lifecycle Management**
   - Start/stop operations as commands
   - Status monitoring through messages
   - Cleanup operations in Update()

#### Phase 3: Service Layer Refactoring
1. **Input Services**
   - Keyboard input as command stream
   - PTY input adaptation through messages
   - Navigation as stateless functions

2. **AI Coder Service**
   - Session management in Model
   - AI interactions as commands
   - Response handling through messages

3. **Notification Service**
   - Notification queue in Model
   - Display logic in components
   - Lifecycle management through commands

#### Phase 4: Component Refactoring
1. **Shell Integration**
   - Command execution through Model
   - Output capture as messages
   - UI updates from Model state

2. **Script Selection**
   - Script list in Model
   - Selection state in Model
   - Actions as commands

#### Phase 5: Command Infrastructure
1. **Remove Goroutine Helpers**
   - Delete `goroutine_helper.go` files
   - Replace with command patterns
   - Update all callers

2. **Batch Operations**
   - Sequential execution through commands
   - Parallel execution via tea.Sequence
   - Error handling through messages

3. **Cache Management**
   - Cache state in Model
   - TTL management in Update()
   - Async refresh through commands

### Implementation Patterns

#### 1. Service Function Pattern
```go
// Before (stateful service with goroutine)
type InputService struct {
    inputChan chan KeyEvent
    buffer    []rune
}

func (s *InputService) StartReading() {
    go func() {
        for event := range s.inputChan {
            s.buffer = append(s.buffer, event.Rune)
        }
    }()
}

// After (stateless service)
type InputService interface{}

func (s InputService) ProcessKeyEvent(model *Model, event KeyEvent) InputResult {
    // Pure function - no internal state
    return InputResult{
        NewBuffer: append(model.InputBuffer, event.Rune),
        Command:   s.createUpdateCommand(event),
    }
}
```

#### 2. Command Pattern for Async Operations
```go
// Before (direct goroutine)
func StartProcessInBackground(cmd string) {
    go func() {
        process := exec.Command(cmd)
        process.Run()
        // Handle result...
    }()
}

// After (command pattern)
func StartProcessCommand(model *Model, cmd string) tea.Cmd {
    return func() tea.Msg {
        // Async execution in Bubble Tea context
        return ProcessStartedMsg{Command: cmd}
    }
}
```

#### 3. State Management in Model
```go
// Master data structure
type Model struct {
    // ... existing fields ...

    // Centralized PTY state
    ActivePTYs map[string]*PTYState
    PTYEvents  []PTYEvent

    // Centralized service state
    InputBuffer     []rune
    Notifications   []Notification
    AISessions      map[string]*AISession

    // Version tracking
    Version uint64
}

// Atomic update pattern
func (m *Model) UpdatePTYState(id string, state *PTYState) {
    atomic.AddUint64(&m.Version, 1)
    if m.ActivePTYs == nil {
        m.ActivePTYs = make(map[string]*PTYState)
    }
    m.ActivePTYs[id] = state
}
```

### Validation Criteria

1. **Goroutine Elimination**
   - `grep -r "go func" internal/tui/ --include="*.go" | grep -v test` returns empty
   - Only Bubble Tea goroutines remain

2. **Stateless Services**
   - All services have no exported fields
   - All service methods are pure functions
   - No internal state in service structs

3. **Single Source of Truth**
   - All state reads come from Model
   - All state changes go through Update()
   - No distributed state across services

4. **Performance Requirements**
   - UI response time <50ms
   - Command overhead <1ms
   - No race conditions

### Benefits

1. **Predictable Concurrency**
   - Only Bubble Tea manages goroutines
   - No race conditions from custom goroutines
   - Easier debugging and testing

2. **Clean Architecture**
   - Clear separation of concerns
   - Pure functions are easy to test
   - Immutable state prevents bugs

3. **Better Performance**
   - Lock-free atomic operations
   - No mutex contention
   - Efficient memory usage

4. **Maintainability**
   - Single source of truth
   - No hidden state in services
   - Clear data flow patterns

## Next Steps

1. Implement master data object design
2. Create stateless service interfaces
3. Migrate PTY integration to command pattern
4. Refactor service layer to pure functions
5. Update components to use Model state
6. Remove all goroutine helpers
7. Validate complete goroutine elimination

This architecture ensures a truly stateless, lock-free TUI that relies entirely on Bubble Tea's proven concurrency model.