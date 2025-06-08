---
sidebar_position: 1
---

# Architecture

Understanding Brummer's architecture helps contributors and advanced users extend and customize the tool.

## Overview

Brummer follows a modular architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────┐
│                    User Interface                        │
│                  (TUI - Bubble Tea)                      │
├─────────────────────────────────────────────────────────┤
│                    Core Engine                           │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │   Process    │  │     Log      │  │      MCP      │  │
│  │  Management  │  │   Storage    │  │    Server     │  │
│  └─────────────┘  └──────────────┘  └───────────────┘  │
├─────────────────────────────────────────────────────────┤
│                  Package Managers                        │
│     npm      │     yarn     │     pnpm    │    bun     │
└─────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Terminal User Interface (TUI)

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), a Go framework for terminal UIs.

**Key files**:
- `internal/tui/model.go` - Main TUI model and update logic
- `internal/tui/views/` - Individual view components

**Features**:
- Responsive layout system
- Keyboard navigation
- Real-time updates
- Theme support

### 2. Process Management

Handles lifecycle of npm/yarn/pnpm/bun scripts.

**Key files**:
- `internal/process/manager.go` - Process lifecycle management
- `internal/process/executor.go` - Command execution

**Responsibilities**:
- Process spawning with proper environment
- Signal handling (SIGTERM, SIGINT)
- Resource monitoring
- State tracking

**Process States**:
```go
type ProcessState int

const (
    ProcessPending ProcessState = iota
    ProcessRunning
    ProcessStopped
    ProcessFailed
    ProcessSuccess
)
```

### 3. Log Management

Efficient log storage and retrieval system.

**Key files**:
- `internal/logs/store.go` - Log storage implementation
- `internal/logs/detector.go` - Pattern detection

**Features**:
- Circular buffer for memory efficiency
- Pattern matching for errors/warnings
- URL detection
- ANSI color preservation
- Search indexing

**Log Entry Structure**:
```go
type LogEntry struct {
    ID        string
    ProcessID string
    Timestamp time.Time
    Level     LogLevel
    Content   string
    Metadata  map[string]interface{}
}
```

### 4. MCP Server

Model Context Protocol server for external integrations.

**Key files**:
- `internal/mcp/server.go` - MCP server implementation
- `internal/mcp/handlers.go` - Request handlers

**Protocol Implementation**:
- JSON-RPC 2.0 over stdio
- Event streaming
- Resource management

### 5. Package Manager Abstraction

Unified interface for different package managers.

**Key files**:
- `internal/parser/package_json.go` - Package.json parsing
- `internal/parser/detector.go` - Package manager detection

**Detection Priority**:
1. Lock files (yarn.lock, pnpm-lock.yaml, bun.lockb)
2. Explicit configuration
3. Available binaries
4. Default to npm

## Data Flow

### 1. Script Execution Flow

```
User Input → TUI → Process Manager → Package Manager → Child Process
                                                            ↓
                                                         Log Store
                                                            ↓
                                                      Pattern Detector
                                                            ↓
                                                        TUI Update
```

### 2. MCP Communication Flow

```
External Client → JSON-RPC Request → MCP Server
                                         ↓
                                    Request Handler
                                         ↓
                                    Core Components
                                         ↓
                                   JSON-RPC Response
```

## Key Design Patterns

### 1. Event-Driven Architecture

All components communicate through events:

```go
type Event interface {
    Type() EventType
    Timestamp() time.Time
}

type EventBus interface {
    Subscribe(EventType, EventHandler)
    Publish(Event)
}
```

### 2. Command Pattern

User actions are encapsulated as commands:

```go
type Command interface {
    Execute() error
    Undo() error
}
```

### 3. Observer Pattern

TUI components observe model changes:

```go
type Observable interface {
    Attach(Observer)
    Detach(Observer)
    Notify()
}
```

## Memory Management

### Log Rotation

Logs are stored in a circular buffer with configurable size:

```go
type CircularBuffer struct {
    data     []LogEntry
    capacity int
    head     int
    tail     int
    mu       sync.RWMutex
}
```

### Process Cleanup

Automatic cleanup of terminated processes:
- Signal handlers for graceful shutdown
- Orphan process detection
- Resource leak prevention

## Concurrency Model

### Goroutine Architecture

```
Main Goroutine (TUI)
    ├── Process Monitor Goroutines (1 per process)
    ├── Log Reader Goroutines (1 per process)
    ├── MCP Server Goroutine
    ├── Event Bus Goroutine
    └── File Watcher Goroutine
```

### Synchronization

- **Channels** for communication
- **Mutexes** for shared state
- **Context** for cancellation

## Configuration

### Configuration Layers

1. **Default Configuration** - Built-in defaults
2. **System Configuration** - `/etc/brummer/config.yaml`
3. **User Configuration** - `~/.config/brummer/config.yaml`
4. **Project Configuration** - `.brummer.yaml`
5. **Environment Variables** - `BRUMMER_*`
6. **Command Line Flags** - Highest priority

### Configuration Schema

```yaml
# .brummer.yaml
version: 1
ui:
  theme: dark
  refresh_rate: 100ms
  max_log_lines: 10000
process:
  default_shell: /bin/bash
  env_inherit: true
  signal_timeout: 10s
mcp:
  enabled: true
  port: 3280
  auth: false
filters:
  error_patterns:
    - "ERROR"
    - "FAIL"
    - "Error:"
  ignore_patterns:
    - "node_modules"
```

## Extension Points

### 1. Custom Log Detectors

```go
type LogDetector interface {
    Detect(string) []Detection
    Priority() int
}
```

### 2. Process Hooks

```go
type ProcessHook interface {
    PreStart(Process) error
    PostStart(Process) error
    PreStop(Process) error
    PostStop(Process) error
}
```

### 3. UI Components

```go
type View interface {
    Init() tea.Cmd
    Update(tea.Msg) (View, tea.Cmd)
    View() string
}
```

## Performance Considerations

### 1. Log Processing

- **Lazy evaluation** for pattern matching
- **Batch updates** to reduce UI refreshes
- **Index-based search** for large logs

### 2. Process Monitoring

- **Polling intervals** based on process activity
- **Resource caching** to reduce system calls
- **Differential updates** for efficiency

### 3. UI Rendering

- **Virtual scrolling** for large lists
- **Debounced updates** for rapid changes
- **Minimal redraws** using dirty flags

## Security Considerations

### 1. Process Isolation

- Processes run with user privileges
- Environment variable sanitization
- Command injection prevention

### 2. MCP Security

- Optional authentication
- Request validation
- Rate limiting

### 3. File Access

- Restricted to project directory
- Symlink resolution
- Path traversal prevention

## Testing Architecture

### 1. Unit Tests

```go
// process_test.go
func TestProcessLifecycle(t *testing.T) {
    // Test process start, stop, restart
}
```

### 2. Integration Tests

```go
// integration_test.go
func TestFullWorkflow(t *testing.T) {
    // Test complete user workflow
}
```

### 3. Mocking

```go
type MockProcessManager struct {
    mock.Mock
}
```

## Future Architecture Considerations

### Planned Enhancements

1. **Plugin System** - Dynamic loading of extensions
2. **Clustering** - Multi-machine process management
3. **Web UI** - Browser-based alternative to TUI
4. **Metrics Collection** - Prometheus integration
5. **Distributed Tracing** - OpenTelemetry support