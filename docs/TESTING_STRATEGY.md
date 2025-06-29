# Brummer Testing Strategy

This document outlines the comprehensive testing strategy for the Brummer project, focusing on maintainable, reliable tests that match the current codebase APIs.

## Overview

The testing strategy ensures code quality through:
- **Unit tests**: Testing individual components in isolation
- **Integration tests**: Testing component interactions
- **End-to-end tests**: Testing complete workflows
- **Concurrent safety tests**: Ensuring thread-safe operations

## Test Organization

### Core Package Tests

#### 1. Process Management (`internal/process/`)
**File**: `manager_test.go`

**Key Test Areas**:
- Manager initialization with/without package.json
- Process lifecycle (start, stop, status monitoring)
- Script execution from package.json
- Log callback integration
- Package manager detection
- Concurrent process operations
- Cleanup and resource management

**Important APIs Tested**:
```go
func NewManager(workDir string, eventBus *events.EventBus, hasPackageJSON bool) (*Manager, error)
func (m *Manager) StartScript(scriptName string) (*Process, error)
func (m *Manager) StartCommand(name string, command string, args []string) (*Process, error)
func (m *Manager) GetAllProcesses() []*Process
func (m *Manager) StopProcess(processID string) error
func (m *Manager) Cleanup() error
```

**Process Status Constants**:
- `StatusPending`, `StatusRunning`, `StatusStopped`, `StatusFailed`, `StatusSuccess`

#### 2. Event System (`pkg/events/`)
**File**: `events_test.go`

**Key Test Areas**:
- EventBus creation and subscription
- Multiple subscribers per event type
- Cross-event-type isolation
- Automatic event metadata (ID, timestamp)
- Concurrent publishing and subscription
- Event type constants validation

**Important APIs Tested**:
```go
func NewEventBus() *EventBus
func (eb *EventBus) Subscribe(eventType EventType, handler Handler)
func (eb *EventBus) Publish(event Event)
```

**Event Types Tested**:
- `ProcessStarted`, `ProcessExited`, `LogLine`, `ErrorDetected`
- `BuildEvent`, `TestFailed`, `TestPassed`
- `MCPActivity`, `MCPConnected`, `MCPDisconnected`

#### 3. Proxy Server (`internal/proxy/`)
**File**: `server_test.go`

**Key Test Areas**:
- Server creation with different modes
- Lifecycle management (start/stop)
- Full proxy mode (traditional HTTP proxy)
- Reverse proxy mode (URL registration)
- URL mapping management
- Mode switching between full/reverse
- Request capture functionality
- Telemetry integration
- Concurrent operations
- Port conflict handling

**Important APIs Tested**:
```go
func NewServer(port int, eventBus *events.EventBus) *Server
func NewServerWithMode(port int, mode ProxyMode, eventBus *events.EventBus) *Server
func (s *Server) Start() error
func (s *Server) RegisterURL(urlStr, processName string) string
func (s *Server) GetURLMappings() []URLMapping
func (s *Server) SwitchMode(newMode ProxyMode) error
```

**Proxy Modes**:
- `ProxyModeFull`: Traditional HTTP proxy
- `ProxyModeReverse`: Reverse proxy with URL registration

#### 4. Configuration (`internal/config/`)
**File**: Existing tests are working ✅

**Coverage**: Configuration loading, hierarchical overrides, defaults

#### 5. Discovery System (`internal/discovery/`)
**File**: Existing tests are working ✅

**Coverage**: Instance discovery, file operations, ping updates

#### 6. Logs Package (`internal/logs/`)
**File**: Fixed concurrency tests ✅

**Coverage**: Log storage, error parsing, filtering, concurrent operations

### User Interface Tests

#### 7. TUI (`internal/tui/`)
**File**: `model_test.go`

**Key Test Areas**:
- View constants validation
- Model creation with different configurations
- Component integration testing
- UI logic validation (filtering, formatting, commands)
- Process status display formatting
- URL validation and display
- Configuration display logic
- Color theme validation

**Focus**: Testing UI logic and data formatting rather than interactive behavior

### Integration Tests

#### 8. Hub Integration (`test/`)
**File**: `hub_integration_test.go` (Fixed)

**Key Test Areas**:
- Full hub workflow (discovery → connection → tool proxying)
- Multiple instance management
- Instance failure and recovery scenarios
- Tool proxy integration
- Health monitoring integration

**Test Coverage**:
- Mock MCP server creation
- Instance registration and discovery
- Tool call forwarding
- Session management
- Health check validation

## Test Patterns and Best Practices

### 1. API Alignment
- All tests match current codebase APIs exactly
- No assumptions about non-existent methods or constants
- Direct field access for structs without getter methods

### 2. Concurrency Testing
- Use `sync.WaitGroup` for coordinating goroutines
- `atomic` package for thread-safe counters
- Proper cleanup with `defer` statements
- Race condition detection

### 3. Resource Management
- Always call `defer store.Close()` for log stores
- Clean up processes with `defer mgr.Cleanup()`
- Use `t.TempDir()` for temporary directories
- Avoid double-cleanup calls (fixed concurrency issue)

### 4. Event Testing
- Use channels and timeouts for async event verification
- Test both successful and failure scenarios
- Verify event metadata (timestamps, IDs)

### 5. Error Handling
- Test both success and failure paths
- Validate error messages and types
- Ensure proper cleanup on errors

## Test Execution

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/process
go test ./pkg/events
go test ./internal/proxy
go test ./internal/tui

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...
```

### Continuous Integration
- All tests must pass before merging
- Race detection enabled in CI
- Coverage reporting configured
- Integration tests run in isolation

## Known Issues and Limitations

### Fixed Issues ✅
1. **Process Manager**: Updated to use correct API (`GetAllProcesses` vs `GetProcesses`, direct status access)
2. **Events System**: Replaced non-existent `Unsubscribe` references with working subscription model
3. **Proxy Server**: Fixed import issues and API mismatches
4. **Logs Package**: Removed duplicate `Close()` calls causing channel close panics
5. **Hub Integration**: Fixed argument handling in mock server

### Current Status
- **Working Tests**: ~35+ tests across core packages
- **Test Coverage**: Focus on critical paths and concurrent operations
- **API Compatibility**: All tests match current codebase exactly

## Future Improvements

1. **Performance Tests**: Add benchmarks for high-throughput scenarios
2. **Memory Leak Tests**: Goroutine and memory leak detection
3. **Integration Tests**: More complex multi-component scenarios
4. **Property-Based Tests**: Using tools like `gopter` for edge case discovery

## Test Maintenance

### Adding New Tests
1. Follow existing patterns in the package
2. Use `testify/assert` and `testify/require` for assertions
3. Include both positive and negative test cases
4. Add concurrent safety tests for shared resources

### Updating Tests
1. Keep tests in sync with API changes
2. Update test data when business logic changes
3. Maintain backward compatibility where possible
4. Document breaking changes

### Review Checklist
- [ ] Tests cover main functionality
- [ ] Error cases are tested
- [ ] Concurrent access is tested where applicable
- [ ] Resources are properly cleaned up
- [ ] Test names are descriptive
- [ ] Assertions are clear and specific

This testing strategy ensures reliable, maintainable tests that accurately reflect the current codebase and provide confidence in the system's behavior.