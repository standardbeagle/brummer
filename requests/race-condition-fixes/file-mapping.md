# Codebase File Mapping - Race Condition Fixes

## Change Points Identified

### Primary Files Requiring Modification

#### Critical TUI Model Issues
- `internal/tui/model.go` - **CRITICAL**: 60+ methods need value→pointer receiver conversion
- `internal/tui/mcp_connections.go` - Model value passing issues  
- `internal/tui/script_selector.go` - Model value passing issues
- `internal/tui/model_test.go` - Test fixes for pointer receivers
- `cmd/brum/main.go` - TUI initialization and Model usage

#### EventBus Goroutine Issues  
- `pkg/events/events.go` - **CRITICAL**: Unlimited goroutine spawning
- `pkg/events/events_test.go` - Test coverage for worker pool

#### Process Manager Race Conditions
- `internal/process/manager.go` - **HIGH**: Concurrent map access
- `internal/process/process.go` - Process state synchronization
- `internal/process/manager_test.go` - Test improvements
- `internal/process/process_failure_test.go` - Race condition tests

#### Log Store Async/Sync Issues
- `internal/logs/store.go` - **HIGH**: Mixed async/sync operations
- `internal/logs/store_nonblocking_test.go` - Failed stress tests need fixing
- `internal/logs/error_parser.go` - Thread safety review

#### Proxy Server Multiple Mutex Issues
- `internal/proxy/server.go` - **HIGH**: Multiple mutex anti-pattern
- `internal/proxy/server_test.go` - Concurrency test improvements

#### MCP Connection Manager Issues
- `internal/mcp/connection_manager.go` - **MEDIUM**: Session mapping races
- `internal/mcp/hub_client.go` - Connection state management
- `internal/mcp/health_monitor.go` - State transition races
- `internal/mcp/connection_manager_test.go` - Enhanced concurrency tests

### Secondary Files with Dependencies

#### Type Definitions and Interfaces
- `internal/tui/types.go` - May need updates for pointer receivers
- `internal/process/types.go` - Process status types
- `pkg/events/types.go` - Event handling types

#### Configuration Files
- `internal/config/config.go` - Worker pool configuration
- `internal/config/defaults.go` - Default concurrency limits

#### Test Infrastructure
- `internal/testutil/wait.go` - Test synchronization utilities
- `internal/testutil/error_injection.go` - Race condition testing
- `test/testutil/helpers.go` - Concurrency test helpers

#### Documentation
- `CLAUDE.md` - Update with race condition prevention guidelines
- `README.md` - Update build/test instructions
- `docs/architecture.md` - Document concurrency patterns

## File Dependency Graph

### Critical Path Dependencies
```
TUI Model Fix → All TUI functionality
    ↓
EventBus Fix → Process Manager, Log Store, MCP Components
    ↓  
Process Manager → Log Store → Proxy Server → MCP Manager
    ↓
Integration Tests → Stress Testing → Documentation
```

### Dependency Relationships

#### TUI Model Dependencies
- `internal/tui/model.go` → All TUI component files
- `cmd/brum/main.go` → TUI Model initialization
- All TUI render methods → Model state access

#### EventBus Dependencies  
- `pkg/events/events.go` → Process Manager, Log Store, MCP components
- All event subscribers → EventBus interface
- Process lifecycle → Event publishing

#### Process Manager Dependencies
- `internal/process/manager.go` → Log Store (for process logs)
- Process Manager → EventBus (for process events)
- Process Manager → Config (for package manager detection)

#### Log Store Dependencies
- `internal/logs/store.go` → EventBus (for log events)
- Log Store → Error Parser (for error detection)
- Log Store → URL detection utilities

#### Cross-Component Dependencies
- TUI → Process Manager, Log Store, Proxy Server, MCP
- MCP Connection Manager → Health Monitor → Hub Client
- Proxy Server → Log Store (for URL detection)

## Integration Boundaries

### API Contracts Between Modules

#### TUI ↔ Process Manager
```go
type ProcessManager interface {
    GetAllProcesses() []*Process  // Thread-safe read
    StartCommand(name, id string, args []string) (*Process, error)  // Thread-safe write
    StopProcess(id string) error  // Thread-safe write
}
```

#### EventBus ↔ All Components
```go
type EventBus interface {
    Subscribe(eventType EventType, handler Handler)  // Thread-safe
    Publish(event Event)  // Thread-safe with worker pool
}
```

#### Log Store ↔ Components
```go
type LogStore interface {
    Add(processID, processName, content string, isError bool) *LogEntry  // Thread-safe
    GetAll() []LogEntry  // Thread-safe read
    GetByProcess(processID string) []LogEntry  // Thread-safe read
}
```

### Shared Types and Interfaces

#### Critical Shared State
- `Process` struct - Process status and metadata
- `LogEntry` struct - Log data with timestamps
- `Event` struct - Event data for pub/sub
- `URLMapping` struct - Proxy URL mappings

#### Synchronization Primitives
- `sync.RWMutex` - Primary synchronization mechanism
- `sync.WaitGroup` - Goroutine coordination
- `chan struct{}` - Worker pool semaphores
- `context.Context` - Cancellation and timeouts

### External Service Touchpoints

#### File System Integration
- Instance discovery files - `/tmp/brummer-instances/`
- Configuration files - `.brum.toml`, `package.json`
- Log file outputs - Process stdout/stderr

#### Network Integration  
- MCP JSON-RPC HTTP server - Port 7777
- Proxy server - Dynamic port allocation
- WebSocket connections - Browser automation

#### Process Integration
- Child process spawning - `os/exec` package
- Signal handling - Process termination
- Resource monitoring - CPU/Memory usage

## Race Condition Risk Assessment

### High-Risk Integration Points

1. **TUI Model State Mutation**
   - Risk: Value receivers create copies, state changes lost
   - Impact: Application-wide data corruption
   - Mitigation: Pointer receivers + explicit synchronization

2. **EventBus Goroutine Explosion**
   - Risk: Unlimited concurrent event handlers
   - Impact: Resource exhaustion, system crash
   - Mitigation: Worker pool with semaphore

3. **Process Manager Map Operations**
   - Risk: Concurrent map read/write
   - Impact: Runtime panics, data corruption
   - Mitigation: Consistent RWMutex usage

4. **Log Store Mixed Operations**
   - Risk: Channel vs direct operations race
   - Impact: Lost logs, inconsistent state
   - Mitigation: Eliminate sync fallback paths

### Medium-Risk Integration Points

1. **MCP Session Management**
   - Risk: Session routing corruption
   - Impact: Incorrect tool routing
   - Mitigation: Channel-based serialization

2. **Proxy URL Mapping**
   - Risk: Multiple mutex deadlock
   - Impact: Proxy server freeze
   - Mitigation: Single mutex hierarchy

### Low-Risk Integration Points

1. **Configuration Loading**
   - Risk: Minimal concurrent access
   - Impact: Configuration inconsistency
   - Mitigation: Read-only after initialization

2. **File Discovery**
   - Risk: Filesystem-level synchronization
   - Impact: Delayed instance discovery
   - Mitigation: File locking patterns

## Testing Strategy

### Unit Test Coverage
- Individual component synchronization
- Goroutine leak detection
- Deadlock prevention verification

### Integration Test Coverage  
- Cross-component event flow
- Concurrent operation scenarios
- Resource exhaustion testing

### Stress Test Coverage
- High-frequency event publishing
- Concurrent TUI operations
- Process lifecycle under load

## Performance Considerations

### Synchronization Overhead
- RWMutex vs Mutex selection
- Atomic operations vs locks
- Channel operations vs direct calls

### Memory Management
- Goroutine pool sizing
- Event buffer management
- Log entry rotation

### Scalability Factors
- Maximum concurrent processes
- Event throughput limits
- Connection pool sizing