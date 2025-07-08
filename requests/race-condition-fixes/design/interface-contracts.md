# Thread-Safe Interface Contracts for Brummer

## Overview

This document defines the thread-safe interface contracts for all Brummer components. These contracts specify concurrency guarantees, error handling behavior, and resource management requirements that enable safe concurrent usage.

## Core Interface Principles

### 1. Thread-Safety Levels

**Level 1: Thread-Safe (TS)** - Safe for concurrent access from multiple goroutines
**Level 2: Read-Safe (RS)** - Safe for concurrent reads, writes must be synchronized externally  
**Level 3: Single-Threaded (ST)** - Not safe for concurrent access, requires external synchronization

### 2. Contract Annotations

All interfaces use standardized annotations:
```go
// ThreadSafety: TS|RS|ST
// LockRequirement: None|Read|Write|Custom
// TimeoutBehavior: Blocking|NonBlocking|Timeout(duration)
// ResourceCleanup: Automatic|Manual|RAII
```

## EventBus Interface Contracts

### Core EventBus Interface

```go
// EventBus provides thread-safe event publishing and subscription
//
// ThreadSafety: TS - All methods safe for concurrent access
// ResourceCleanup: Automatic - Workers cleaned up on Stop()
type EventBus interface {
    // Subscribe registers an event handler for a specific event type
    //
    // ThreadSafety: TS - Safe to call concurrently with Publish/Unsubscribe
    // TimeoutBehavior: NonBlocking - Returns immediately
    // Guarantees: Handler will receive all events published after registration
    Subscribe(eventType EventType, handler Handler) SubscriptionID
    
    // Unsubscribe removes an event handler
    //
    // ThreadSafety: TS - Safe to call concurrently with Publish/Subscribe
    // TimeoutBehavior: NonBlocking - Returns immediately  
    // Guarantees: Handler will not receive events published after unsubscription
    Unsubscribe(id SubscriptionID) error
    
    // Publish sends an event to all registered handlers
    //
    // ThreadSafety: TS - Safe to call concurrently from multiple goroutines
    // TimeoutBehavior: Timeout(100ms) - Returns error if worker pool full
    // Guarantees: Event delivered to handlers registered before Publish call
    // BackpressureHandling: Configurable (drop_oldest, drop_newest, block)
    Publish(event Event) error
    
    // PublishAsync sends an event without blocking
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: NonBlocking - Returns immediately
    // Guarantees: Best-effort delivery, may drop under extreme load
    PublishAsync(event Event)
    
    // Stop gracefully shuts down the event bus
    //
    // ThreadSafety: TS - Safe to call multiple times
    // TimeoutBehavior: Timeout(5s) - Waits for worker pool to drain
    // Guarantees: All pending events processed before shutdown
    Stop() error
    
    // Metrics returns current performance metrics
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: NonBlocking - Returns current snapshot
    Metrics() EventBusMetrics
}

// Handler function contract
//
// ThreadSafety: ST - Implementation must handle concurrency internally
// TimeoutBehavior: Implementation-defined
// ErrorHandling: Panics are recovered and logged by EventBus
type Handler func(event Event)

type EventBusMetrics struct {
    EventsProcessed   int64         // Total events processed
    EventsDropped     int64         // Events dropped due to backpressure
    AvgProcessingTime time.Duration // Average time per event
    WorkerUtilization float64       // Percentage of workers busy
    QueueDepth        [3]int        // Current queue depths [High, Med, Low]
}
```

### Worker Pool Interface

```go
// WorkerPool manages the execution of event handlers
//
// ThreadSafety: TS - All methods safe for concurrent access
// ResourceCleanup: Automatic - Goroutines cleaned up on Stop()
type WorkerPool interface {
    // Submit queues a job for execution
    //
    // ThreadSafety: TS - Safe to call from multiple goroutines
    // TimeoutBehavior: Timeout(configurable) - Returns error if queue full
    // Guarantees: Job will be executed if Submit returns nil error
    Submit(job Job, priority Priority) error
    
    // Stop shuts down the worker pool
    //
    // ThreadSafety: TS - Safe to call multiple times
    // TimeoutBehavior: Timeout(configurable) - Waits for workers to finish
    // Guarantees: All submitted jobs complete before return
    Stop() error
    
    // Metrics returns current pool metrics
    //
    // ThreadSafety: TS - Atomic read of metrics
    // TimeoutBehavior: NonBlocking
    Metrics() PoolMetrics
}

type Priority int
const (
    PriorityHigh Priority = iota
    PriorityMedium
    PriorityLow
)

type Job interface {
    Execute() error
    ID() string
    Timeout() time.Duration
}
```

## Process Manager Interface Contracts

### Core Process Manager Interface

```go
// ProcessManager manages process lifecycle with thread-safe operations
//
// ThreadSafety: TS - All methods safe for concurrent access
// ResourceCleanup: Manual - Call Cleanup() before shutdown
type ProcessManager interface {
    // StartScript starts a script from package.json
    //
    // ThreadSafety: TS - Safe to start multiple scripts concurrently
    // TimeoutBehavior: Timeout(30s) - Process startup timeout
    // Guarantees: Returns unique Process ID if successful
    // ErrorHandling: Returns error for invalid scripts or system failures
    StartScript(scriptName string) (*Process, error)
    
    // StartCommand starts a custom command
    //
    // ThreadSafety: TS - Safe to start multiple commands concurrently
    // TimeoutBehavior: Timeout(30s) - Process startup timeout
    // Guarantees: Returns unique Process ID if successful
    StartCommand(name, command string, args []string) (*Process, error)
    
    // StopProcess terminates a running process
    //
    // ThreadSafety: TS - Safe to stop multiple processes concurrently
    // TimeoutBehavior: Timeout(10s) - Process termination timeout
    // Guarantees: Process terminated or error returned
    // ForceKill: Applied after graceful termination timeout
    StopProcess(processID string) error
    
    // GetProcess retrieves process information
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: NonBlocking - Returns current snapshot
    // Guarantees: Consistent view of process state at call time
    GetProcess(processID string) (*Process, bool)
    
    // GetAllProcesses returns all managed processes
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: NonBlocking - Returns current snapshot
    // Guarantees: Consistent view of all processes at call time
    GetAllProcesses() []*Process
    
    // RegisterLogCallback adds a log event handler
    //
    // ThreadSafety: TS - Safe to register multiple callbacks concurrently
    // TimeoutBehavior: NonBlocking - Registration is immediate
    // Guarantees: Callback receives all logs for new processes
    RegisterLogCallback(callback LogCallback) CallbackID
    
    // UnregisterLogCallback removes a log event handler
    //
    // ThreadSafety: TS - Safe to unregister concurrently
    // TimeoutBehavior: NonBlocking - Unregistration is immediate
    // Guarantees: Callback stops receiving logs after return
    UnregisterLogCallback(id CallbackID) error
    
    // Cleanup stops all processes and releases resources
    //
    // ThreadSafety: TS - Safe to call multiple times
    // TimeoutBehavior: Timeout(30s) - Waits for all processes to stop
    // Guarantees: All processes terminated and resources released
    Cleanup() error
}

// Process represents a managed process with thread-safe access
//
// ThreadSafety: TS - All methods safe for concurrent access
type Process interface {
    // ID returns the unique process identifier
    //
    // ThreadSafety: TS - Immutable after creation
    // TimeoutBehavior: NonBlocking
    ID() string
    
    // Name returns the process name
    //
    // ThreadSafety: TS - Immutable after creation
    // TimeoutBehavior: NonBlocking
    Name() string
    
    // Status returns the current process status
    //
    // ThreadSafety: TS - Atomic read operation
    // TimeoutBehavior: NonBlocking
    // Guarantees: Returns current status at time of call
    Status() ProcessStatus
    
    // ExitCode returns the process exit code if completed
    //
    // ThreadSafety: TS - Atomic read operation
    // TimeoutBehavior: NonBlocking
    // Guarantees: Valid only when Status() returns stopped/failed/success
    ExitCode() (int, bool)
    
    // StartTime returns when the process was started
    //
    // ThreadSafety: TS - Immutable after start
    // TimeoutBehavior: NonBlocking
    StartTime() time.Time
    
    // EndTime returns when the process ended
    //
    // ThreadSafety: TS - Atomic read operation
    // TimeoutBehavior: NonBlocking
    // Guarantees: Valid only when Status() indicates completion
    EndTime() (time.Time, bool)
    
    // IsRunning checks if process is currently running
    //
    // ThreadSafety: TS - Atomic operation
    // TimeoutBehavior: NonBlocking
    // Guarantees: Accurate at time of call
    IsRunning() bool
}

type ProcessStatus int32
const (
    StatusPending ProcessStatus = iota
    StatusRunning
    StatusStopped
    StatusFailed
    StatusSuccess
)

// LogCallback receives process log output
//
// ThreadSafety: ST - Implementation must handle concurrency
// ErrorHandling: Panics are recovered by ProcessManager
type LogCallback func(processID, line string, isError bool)
```

## Log Store Interface Contracts

### Core Log Store Interface

```go
// LogStore provides thread-safe log storage and retrieval
//
// ThreadSafety: TS - All methods safe for concurrent access
// ResourceCleanup: Automatic - Background worker handles cleanup
type LogStore interface {
    // Add appends a log entry to the store
    //
    // ThreadSafety: TS - Safe to add from multiple goroutines
    // TimeoutBehavior: Timeout(100ms) - Returns error if store overloaded
    // Guarantees: Entry stored with monotonic ordering within process
    // BackpressureHandling: Drops oldest entries when full
    Add(processID, processName, content string, isError bool) (*LogEntry, error)
    
    // AddBatch appends multiple log entries atomically
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(500ms) - Returns error if store overloaded
    // Guarantees: All entries added or none (atomic operation)
    AddBatch(entries []LogEntryInput) ([]LogEntry, error)
    
    // GetByProcess retrieves all logs for a specific process
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(1s) - Returns error if taking too long
    // Guarantees: Returns consistent snapshot of process logs
    GetByProcess(processID string) ([]LogEntry, error)
    
    // GetAll retrieves all stored logs
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(2s) - Returns error if taking too long
    // Guarantees: Returns consistent snapshot of all logs
    GetAll() ([]LogEntry, error)
    
    // Search finds logs matching a query
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(5s) - Returns error if search takes too long
    // Guarantees: Returns all matches at time search started
    Search(query SearchQuery) ([]LogEntry, error)
    
    // GetErrors retrieves error logs
    //
    // ThreadSafety: TS - Safe to call concurrently  
    // TimeoutBehavior: Timeout(1s) - Returns error if taking too long
    // Guarantees: Returns all error entries at call time
    GetErrors() ([]LogEntry, error)
    
    // GetURLs retrieves detected URLs
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(1s) - Returns error if taking too long
    // Guarantees: Returns all URL entries at call time
    GetURLs() ([]URLEntry, error)
    
    // Clear removes all logs
    //
    // ThreadSafety: TS - Safe to call concurrently with reads
    // TimeoutBehavior: Timeout(2s) - Returns error if operation hangs
    // Guarantees: All logs removed atomically
    Clear() error
    
    // ClearForProcess removes logs for a specific process
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(1s) - Returns error if operation hangs
    // Guarantees: All process logs removed atomically
    ClearForProcess(processID string) error
    
    // Close shuts down the log store
    //
    // ThreadSafety: TS - Safe to call multiple times
    // TimeoutBehavior: Timeout(10s) - Waits for pending operations
    // Guarantees: All pending writes completed before return
    Close() error
    
    // Metrics returns current store metrics
    //
    // ThreadSafety: TS - Atomic read of metrics
    // TimeoutBehavior: NonBlocking
    Metrics() LogStoreMetrics
}

type LogEntry struct {
    ID          string
    ProcessID   string
    ProcessName string
    Timestamp   time.Time
    Content     string
    Level       LogLevel
    IsError     bool
    Tags        []string
    Priority    int
}

type LogEntryInput struct {
    ProcessID   string
    ProcessName string
    Content     string
    IsError     bool
}

type SearchQuery struct {
    Text        string
    ProcessID   string
    Level       LogLevel
    Since       time.Time
    Until       time.Time
    Limit       int
}

type LogStoreMetrics struct {
    TotalEntries    int64
    EntriesPerSec   float64
    ErrorCount      int64
    QueueDepth      int
    MemoryUsageMB   float64
}
```

## TUI Model Interface Contracts

### Core TUI Model Interface

```go
// TUIModel represents the thread-safe TUI state
//
// ThreadSafety: TS - All methods use pointer receivers with synchronization
// ResourceCleanup: Manual - Call Cleanup() before shutdown
type TUIModel interface {
    // Update processes a Bubble Tea message and returns updated model
    //
    // ThreadSafety: TS - Safe to call from multiple goroutines
    // TimeoutBehavior: Timeout(100ms) - UI updates must be responsive
    // Guarantees: State changes are atomic and consistent
    // MutationPolicy: Only Update() modifies state
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    
    // View renders the current UI state
    //
    // ThreadSafety: TS - Safe to call concurrently with Update()
    // TimeoutBehavior: Timeout(50ms) - Rendering must be fast
    // Guarantees: Returns consistent view of state at render time
    // SideEffects: None - pure function
    View() string
    
    // Init initializes the model
    //
    // ThreadSafety: ST - Must be called before any concurrent access
    // TimeoutBehavior: Timeout(1s) - Initialization timeout
    // Guarantees: Model ready for concurrent use after return
    Init() tea.Cmd
    
    // SetView changes the current view
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: NonBlocking - View change is immediate
    // Guarantees: View change is atomic
    SetView(view View) error
    
    // GetCurrentView returns the active view
    //
    // ThreadSafety: TS - Atomic read operation
    // TimeoutBehavior: NonBlocking
    GetCurrentView() View
    
    // RefreshData updates the model with fresh data
    //
    // ThreadSafety: TS - Safe to call from background goroutines
    // TimeoutBehavior: Timeout(2s) - Data refresh timeout
    // Guarantees: Data update is atomic
    RefreshData() error
    
    // Cleanup releases model resources
    //
    // ThreadSafety: TS - Safe to call multiple times
    // TimeoutBehavior: Timeout(5s) - Cleanup timeout
    // Guarantees: All resources released
    Cleanup() error
}

// ViewRenderer handles view-specific rendering
//
// ThreadSafety: TS - All implementations must be thread-safe
type ViewRenderer interface {
    // Render generates the view content
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(50ms) - Must render quickly
    // SideEffects: None - pure function
    Render(data ViewData) string
    
    // HandleInput processes view-specific input
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: NonBlocking - Input handling is immediate
    HandleInput(key string) (action ViewAction, handled bool)
}

type ViewData interface {
    // Type returns the data type for runtime type checking
    Type() ViewType
    
    // Timestamp returns when data was last updated
    Timestamp() time.Time
}

type ViewAction interface {
    // Execute performs the action
    Execute(model TUIModel) error
    
    // Description returns human-readable action description
    Description() string
}
```

## Proxy Server Interface Contracts

### Core Proxy Server Interface

```go
// ProxyServer manages HTTP proxy functionality with thread-safe operations
//
// ThreadSafety: TS - All methods safe for concurrent access
// ResourceCleanup: Automatic - Servers and connections cleaned up on Stop()
type ProxyServer interface {
    // Start initializes and starts the proxy server
    //
    // ThreadSafety: TS - Safe to call multiple times (idempotent)
    // TimeoutBehavior: Timeout(10s) - Server startup timeout
    // Guarantees: Server ready to accept connections after return
    Start() error
    
    // Stop gracefully shuts down the proxy server
    //
    // ThreadSafety: TS - Safe to call multiple times
    // TimeoutBehavior: Timeout(30s) - Graceful shutdown timeout
    // Guarantees: All connections closed and resources released
    Stop() error
    
    // RegisterURL associates a URL with a process for request tracking
    //
    // ThreadSafety: TS - Safe to register URLs concurrently
    // TimeoutBehavior: NonBlocking - Registration is immediate
    // Guarantees: URL mapping active immediately for new requests
    RegisterURL(url, processName string) string
    
    // GetRequests retrieves captured HTTP requests
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(1s) - Request retrieval timeout
    // Guarantees: Returns consistent snapshot of requests
    GetRequests() ([]HTTPRequest, error)
    
    // GetRequestsForProcess retrieves requests for a specific process
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(1s) - Request retrieval timeout
    // Guarantees: Returns all requests for process at call time
    GetRequestsForProcess(processName string) ([]HTTPRequest, error)
    
    // ClearRequests removes all captured requests
    //
    // ThreadSafety: TS - Safe to call concurrently with captures
    // TimeoutBehavior: Timeout(2s) - Clear operation timeout
    // Guarantees: All requests removed atomically
    ClearRequests() error
    
    // GetURLMappings returns current URL to process mappings
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: NonBlocking - Returns current mappings
    // Guarantees: Consistent view of mappings at call time
    GetURLMappings() []URLMapping
    
    // EnableTelemetry controls telemetry collection
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: NonBlocking - Setting change is immediate
    // Guarantees: Telemetry state change is atomic
    EnableTelemetry(enabled bool)
    
    // GetTelemetryStore returns the telemetry data store
    //
    // ThreadSafety: TS - Store itself is thread-safe
    // TimeoutBehavior: NonBlocking
    GetTelemetryStore() TelemetryStore
    
    // IsRunning checks if proxy server is active
    //
    // ThreadSafety: TS - Atomic read operation
    // TimeoutBehavior: NonBlocking
    IsRunning() bool
    
    // GetPort returns the proxy server port
    //
    // ThreadSafety: TS - Immutable after Start()
    // TimeoutBehavior: NonBlocking
    GetPort() int
    
    // Metrics returns current proxy metrics
    //
    // ThreadSafety: TS - Atomic read of metrics
    // TimeoutBehavior: NonBlocking
    Metrics() ProxyMetrics
}

// TelemetryStore manages browser telemetry data
//
// ThreadSafety: TS - All methods safe for concurrent access
type TelemetryStore interface {
    // AddSession records a new browser session
    //
    // ThreadSafety: TS - Safe to add sessions concurrently
    // TimeoutBehavior: Timeout(100ms) - Session add timeout
    // Guarantees: Session stored atomically
    AddSession(session TelemetrySession) error
    
    // GetSession retrieves a specific session
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(500ms) - Session retrieval timeout
    GetSession(sessionID string) (TelemetrySession, error)
    
    // GetSessionsForProcess retrieves sessions for a process
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(1s) - Multi-session retrieval timeout
    GetSessionsForProcess(processName string) ([]TelemetrySession, error)
    
    // ClearSessionsForProcess removes sessions for a process
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: Timeout(2s) - Clear operation timeout
    // Guarantees: All process sessions removed atomically
    ClearSessionsForProcess(processName string) error
}

type HTTPRequest struct {
    ID          string
    Method      string
    URL         string
    StatusCode  int
    Duration    time.Duration
    Size        int64
    ProcessName string
    Timestamp   time.Time
    IsError     bool
}

type ProxyMetrics struct {
    TotalRequests     int64
    SuccessfulReqs    int64
    FailedRequests    int64
    BytesTransferred  int64
    AvgResponseTime   time.Duration
    ActiveConnections int
}
```

## MCP Connection Manager Interface Contracts

### Core MCP Manager Interface

```go
// MCPConnectionManager manages MCP instance connections
//
// ThreadSafety: TS - All methods use channel-based synchronization
// ResourceCleanup: Automatic - Connections cleaned up on Stop()
type MCPConnectionManager interface {
    // RegisterInstance adds a new instance for management
    //
    // ThreadSafety: TS - Safe to register instances concurrently
    // TimeoutBehavior: Timeout(5s) - Registration timeout
    // Guarantees: Instance added to management or error returned
    RegisterInstance(instance *Instance) error
    
    // ConnectSession maps a session to an instance
    //
    // ThreadSafety: TS - Safe to connect sessions concurrently
    // TimeoutBehavior: Timeout(10s) - Connection establishment timeout
    // Guarantees: Session routed to instance or error returned
    ConnectSession(sessionID, instanceID string) error
    
    // DisconnectSession removes session routing
    //
    // ThreadSafety: TS - Safe to disconnect concurrently
    // TimeoutBehavior: NonBlocking - Disconnection is immediate
    // Guarantees: Session routing removed
    DisconnectSession(sessionID string) error
    
    // GetClient returns HTTP client for a session
    //
    // ThreadSafety: TS - Safe to get clients concurrently
    // TimeoutBehavior: NonBlocking - Returns cached client
    // Guarantees: Returns active client or nil
    GetClient(sessionID string) MCPClient
    
    // ListInstances returns all managed instances
    //
    // ThreadSafety: TS - Safe to call concurrently
    // TimeoutBehavior: NonBlocking - Returns current snapshot
    // Guarantees: Consistent view of instances at call time
    ListInstances() []ConnectionInfo
    
    // UpdateActivity marks an instance as active
    //
    // ThreadSafety: TS - Safe to update activity concurrently
    // TimeoutBehavior: NonBlocking - Activity update is immediate
    // Guarantees: Instance activity timestamp updated
    UpdateActivity(instanceID string) bool
    
    // Stop shuts down the connection manager
    //
    // ThreadSafety: TS - Safe to call multiple times
    // TimeoutBehavior: Timeout(15s) - Shutdown timeout
    // Guarantees: All connections closed and resources released
    Stop() error
}

// MCPClient represents a client connection to an MCP instance
//
// ThreadSafety: TS - All methods safe for concurrent access
type MCPClient interface {
    // Call executes an MCP tool call
    //
    // ThreadSafety: TS - Safe to make calls concurrently
    // TimeoutBehavior: Timeout(30s) - Tool execution timeout
    // Guarantees: Call completed or timeout error returned
    Call(ctx context.Context, tool string, params map[string]interface{}) (interface{}, error)
    
    // Initialize establishes the MCP connection
    //
    // ThreadSafety: TS - Safe to call multiple times (idempotent)
    // TimeoutBehavior: Timeout(10s) - Connection timeout
    // Guarantees: Connection ready or error returned
    Initialize(ctx context.Context) error
    
    // Close terminates the MCP connection
    //
    // ThreadSafety: TS - Safe to call multiple times
    // TimeoutBehavior: Timeout(5s) - Close timeout
    // Guarantees: Connection closed and resources released
    Close() error
    
    // IsConnected checks connection status
    //
    // ThreadSafety: TS - Atomic read operation
    // TimeoutBehavior: NonBlocking
    IsConnected() bool
}

type ConnectionInfo struct {
    InstanceID      string
    Name           string
    State          ConnectionState
    LastActivity   time.Time
    ConnectedAt    time.Time
    RetryCount     int
    Sessions       []string
}

type ConnectionState int
const (
    StateDiscovered ConnectionState = iota
    StateConnecting
    StateActive
    StateRetrying
    StateDead
)
```

## Error Handling Contracts

### Standard Error Interface

```go
// BrummerError provides structured error information
//
// ThreadSafety: TS - Error values are immutable
type BrummerError interface {
    error
    
    // Component returns the component where error occurred
    Component() string
    
    // Code returns a structured error code
    Code() ErrorCode
    
    // Temporary indicates if error might resolve on retry
    Temporary() bool
    
    // Timeout indicates if error was due to timeout
    Timeout() bool
    
    // Context returns additional error context
    Context() map[string]interface{}
    
    // Unwrap returns the underlying error
    Unwrap() error
}

type ErrorCode string
const (
    ErrCodeTimeout        ErrorCode = "TIMEOUT"
    ErrCodeNotFound      ErrorCode = "NOT_FOUND"
    ErrCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"
    ErrCodeInvalidState  ErrorCode = "INVALID_STATE"
    ErrCodeResourceLimit ErrorCode = "RESOURCE_LIMIT"
    ErrCodeConcurrency   ErrorCode = "CONCURRENCY"
    ErrCodeNetwork       ErrorCode = "NETWORK"
    ErrCodePermission    ErrorCode = "PERMISSION"
)
```

### Context and Cancellation

```go
// All long-running operations must accept context for cancellation
//
// Contract: Operations must check ctx.Done() periodically
// Timeout: Operations must respect context deadlines
// Cleanup: Operations must clean up resources on cancellation

type ContextualOperation interface {
    Execute(ctx context.Context) error
    Cancel() error
    Progress() OperationProgress
}

type OperationProgress struct {
    Completed    int64
    Total        int64
    CurrentPhase string
    StartTime    time.Time
}
```

## Resource Management Contracts

### Resource Lifecycle Interface

```go
// ResourceManager ensures proper resource lifecycle
//
// ThreadSafety: TS - All methods safe for concurrent access
type ResourceManager interface {
    // Acquire reserves a resource
    //
    // ThreadSafety: TS - Safe to acquire concurrently
    // TimeoutBehavior: Timeout(configurable) - Acquisition timeout
    // Guarantees: Resource acquired or timeout error
    Acquire(resourceType ResourceType, amount int) (*ResourceHandle, error)
    
    // Release frees a previously acquired resource
    //
    // ThreadSafety: TS - Safe to release concurrently
    // TimeoutBehavior: NonBlocking - Release is immediate
    // Guarantees: Resource available for other users
    Release(handle *ResourceHandle) error
    
    // Metrics returns current resource usage
    //
    // ThreadSafety: TS - Atomic read of metrics
    // TimeoutBehavior: NonBlocking
    Metrics() ResourceMetrics
}

type ResourceType string
const (
    ResourceGoroutines ResourceType = "goroutines"
    ResourceMemory     ResourceType = "memory"
    ResourceChannels   ResourceType = "channels"
    ResourceFiles      ResourceType = "files"
)

type ResourceHandle struct {
    ID       string
    Type     ResourceType
    Amount   int
    Acquired time.Time
}

type ResourceMetrics struct {
    Available map[ResourceType]int64
    InUse     map[ResourceType]int64
    Peak      map[ResourceType]int64
    Waiters   map[ResourceType]int
}
```

## Testing Interface Contracts

### Mock Interface Requirements

```go
// All interfaces must provide mock implementations for testing
//
// ThreadSafety: ST - Mocks are for single-threaded test use
// Determinism: Mocks must produce deterministic results
// Validation: Mocks must validate input parameters

type MockEventBus interface {
    EventBus
    
    // Test support methods
    GetSubscriberCount(eventType EventType) int
    GetPublishedEvents() []Event
    SimulateBackpressure(enabled bool)
    InjectError(errorType ErrorType)
}

type MockProcessManager interface {
    ProcessManager
    
    // Test support methods
    GetProcessCount() int
    SimulateProcessFailure(processID string)
    SetProcessOutput(processID string, output []string)
    GetProcessStartCount() int
}
```

## Performance Contract Requirements

### Response Time Guarantees

All operations must meet these response time requirements:

- **UI Operations**: < 16ms (60 FPS)
- **API Calls**: < 100ms (responsive)
- **Background Operations**: < 1s (progress indication)
- **Cleanup Operations**: < 5s (shutdown tolerance)

### Throughput Requirements

- **Event Processing**: 10,000 events/second minimum
- **Log Processing**: 1,000 log lines/second minimum  
- **HTTP Requests**: 100 requests/second minimum
- **Process Operations**: 10 concurrent processes minimum

### Resource Usage Limits

- **Memory**: < 512MB under normal load
- **Goroutines**: < CPU cores * 8
- **File Descriptors**: < 1000
- **Network Connections**: < 100

## Conclusion

These interface contracts provide:

1. **Clear Expectations**: Well-defined behavior for all components
2. **Thread Safety**: Explicit concurrency guarantees
3. **Resource Management**: Bounded resource usage
4. **Error Handling**: Structured error information
5. **Testing Support**: Mockable interfaces
6. **Performance**: Quantified response time and throughput requirements

All implementations must satisfy these contracts to ensure reliable concurrent operation of the Brummer system.