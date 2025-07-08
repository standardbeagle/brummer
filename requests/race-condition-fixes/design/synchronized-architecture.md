# Synchronized Architecture Design for Brummer

## Executive Summary

This document defines the comprehensive synchronized architecture for Brummer that eliminates race conditions while maintaining high performance and backward compatibility. The design implements a layered approach with specific synchronization patterns for each component, prioritizing critical fixes while enabling incremental rollout.

## Architecture Overview

### Current State Analysis

Based on code analysis, the following race conditions have been identified:

1. **EventBus**: Unlimited goroutine spawning in `Publish()` method
2. **TUI Model**: Value receivers causing concurrent access issues
3. **Process Manager**: Mixed RWMutex usage and concurrent map access
4. **Log Store**: Hybrid async/sync pattern with timeout fallbacks
5. **Proxy Server**: Multiple mutexes and concurrent request handling
6. **MCP Connection Manager**: Already well-designed with channels

### Design Principles

1. **Lock Hierarchy**: Clear ordering to prevent deadlocks
2. **Bounded Resources**: All goroutines and channels have limits
3. **Channel-First Design**: Prefer channels over mutexes where appropriate
4. **Copy-on-Write**: Use atomic.Value for read-heavy data structures
5. **Progressive Enhancement**: Enable incremental rollout with fallbacks

## Core Synchronization Patterns

### 1. EventBus Worker Pool Architecture

**Current Issue**: Unlimited goroutine spawning
```go
// Current problematic pattern
for _, handler := range handlers {
    go handler(event) // Creates unlimited goroutines
}
```

**New Architecture**: Bounded worker pool with priority queues
```go
type EventBus struct {
    handlers   atomic.Value // map[EventType][]Handler (copy-on-write)
    updateMu   sync.Mutex   // Only for handler registration
    
    // Worker pool
    workers        int
    eventQueues    [3]chan eventJob // Priority levels: High, Medium, Low
    workerPool     *sync.WaitGroup
    stopCh         chan struct{}
    
    // Configuration
    config         EventBusConfig
    metrics        *EventBusMetrics
}

type eventJob struct {
    Event     events.Event
    Handlers  []Handler
    Priority  int
    Submitted time.Time
}

type EventBusConfig struct {
    Workers           int           // Default: CPU cores * 2.5
    QueueSizes        [3]int        // [High, Medium, Low] queue sizes
    BackpressureMode  BackpressurePolicy
    MonitoringEnabled bool
}
```

**Worker Pool Implementation**:
- **Worker Count**: `max(4, min(runtime.NumCPU() * 2.5, 32))`
- **Queue Strategy**: Priority-based with separate channels
- **Backpressure**: Configurable drop policies (oldest, newest, low-priority)
- **Error Isolation**: Handler panics don't crash workers

### 2. TUI Model Synchronization Pattern

**Current Issue**: Value receivers with concurrent access
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) // Value receiver
```

**New Architecture**: Pointer receivers with RWMutex protection
```go
type Model struct {
    mu sync.RWMutex // Protects all mutable fields
    
    // UI State (protected by mu)
    currentView    View
    processes      []*process.Process
    logs           []logs.LogEntry
    selectedIndex  int
    
    // Immutable or atomic fields
    eventBus       *events.EventBus  // Immutable after init
    processManager *process.Manager  // Immutable after init
    
    // UI Components (have internal synchronization)
    viewport       viewport.Model
    list           list.Model
    textInput      textinput.Model
}

// All methods converted to pointer receivers
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Safe concurrent access to all fields
    return m.updateInternal(msg)
}

func (m *Model) View() string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    // Safe read access to all fields
    return m.renderView()
}
```

**Key Changes**:
- All 60+ methods converted to pointer receivers
- Single RWMutex protects all mutable state
- Read-heavy operations use RLock
- Update operations use Lock
- Atomic operations for simple counters

### 3. Process Manager Thread-Safe Architecture

**Current Issue**: Mixed mutex patterns and concurrent map access
```go
type Manager struct {
    processes map[string]*Process
    mu        sync.RWMutex
    // Inconsistent locking patterns
}
```

**New Architecture**: Consistent synchronization with atomic status
```go
type Manager struct {
    // Process registry (protected by processMu)
    processes   map[string]*Process
    processMu   sync.RWMutex
    
    // Callbacks (protected by callbackMu)
    logCallbacks []LogCallback
    callbackMu   sync.RWMutex
    
    // Configuration (immutable after init)
    packageJSON    *parser.PackageJSON
    workDir        string
    eventBus       *events.EventBus
    
    // Cleanup coordination
    shutdownOnce   sync.Once
    shutdownCh     chan struct{}
}

type Process struct {
    // Immutable fields
    ID        string
    Name      string
    Script    string
    StartTime time.Time
    
    // Atomic fields (lock-free access)
    status    int32  // ProcessStatus as int32
    exitCode  int32  // Exit code
    endTime   int64  // Unix timestamp
    
    // Protected fields (use processMu from Manager)
    Cmd       *exec.Cmd
    cancel    context.CancelFunc
}
```

**Synchronization Strategy**:
- **Process Map**: RWMutex for process registry operations
- **Process Status**: Atomic operations for status, exit code, end time
- **Callbacks**: Separate RWMutex for callback management
- **Lock Hierarchy**: Never hold multiple locks simultaneously

### 4. Log Store Single-Path Architecture

**Current Issue**: Hybrid async/sync pattern with timeout fallbacks
```go
// Problematic timeout-based fallback
select {
case s.addChan <- req:
    // Wait for result with timeout
case <-time.After(100 * time.Millisecond):
    return s.addSync(...) // Race condition potential
}
```

**New Architecture**: Single channel-based path with bounded buffers
```go
type Store struct {
    // Single-path async architecture
    addCh       chan addRequest
    searchCh    chan searchRequest
    getCh       chan getRequest
    
    // Worker state (owned by single goroutine)
    entries       []LogEntry
    byProcess     map[string][]int
    errors        []LogEntry
    urls          []URLEntry
    
    // Configuration
    config        LogStoreConfig
    
    // Control
    stopCh        chan struct{}
    workerWG      sync.WaitGroup
}

type LogStoreConfig struct {
    MaxEntries      int           // Default: 10000
    BatchSize       int           // Default: 100
    BufferSize      int           // Default: 1000
    WorkerTimeout   time.Duration // Default: 50ms
    EnableBatching  bool          // Default: true
}

type addRequest struct {
    processID   string
    processName string
    content     string
    isError     bool
    responseCh  chan *LogEntry
}
```

**Key Features**:
- **Single Worker**: All operations serialized through one goroutine
- **Bounded Buffers**: Request channels have size limits
- **Batch Processing**: Multiple entries processed together
- **No Timeouts**: Eliminates sync fallback path entirely

### 5. Proxy Server Unified Mutex Architecture

**Current Issue**: Multiple mutexes with potential deadlock
```go
type Server struct {
    mu        sync.RWMutex  // For requests
    wsMutex   sync.RWMutex  // For WebSocket clients
    // Different locks for different data
}
```

**New Architecture**: Single mutex with clear sections
```go
type Server struct {
    // Single mutex for all shared state
    mu sync.RWMutex
    
    // State protected by mu
    requests     []Request
    urlMap       map[string]string
    urlMappings  map[string]*URLMapping
    wsClients    map[*websocket.Conn]bool
    
    // Atomic fields (lock-free)
    totalRequests    int64
    successRequests  int64
    failedRequests   int64
    bytesTransferred int64
    
    // Immutable after startup
    port         int
    mode         ProxyMode
    eventBus     *events.EventBus
    telemetry    *TelemetryStore
}
```

**Atomic Metrics Pattern**:
```go
func (s *Server) recordRequest(req Request) {
    // Atomic counters (lock-free)
    atomic.AddInt64(&s.totalRequests, 1)
    if req.IsError {
        atomic.AddInt64(&s.failedRequests, 1)
    } else {
        atomic.AddInt64(&s.successRequests, 1)
    }
    atomic.AddInt64(&s.bytesTransferred, req.Size)
    
    // Structured data (requires lock)
    s.mu.Lock()
    s.requests = append(s.requests, req)
    if len(s.requests) > 1000 {
        s.requests = s.requests[1:]
    }
    s.mu.Unlock()
}
```

## Resource Management Architecture

### 1. Goroutine Pool Management

**Global Goroutine Budget**:
```go
type ResourceManager struct {
    maxGoroutines int32  // CPU cores * 8
    activeGoRoutines int32
    
    // Per-component allocations
    eventBusWorkers   int
    processWorkers    int
    logWorkers        int
    proxyWorkers      int
}

func (rm *ResourceManager) AllocateGoroutine(component string) bool {
    current := atomic.LoadInt32(&rm.activeGoRoutines)
    if current >= rm.maxGoroutines {
        return false
    }
    return atomic.CompareAndSwapInt32(&rm.activeGoRoutines, current, current+1)
}
```

### 2. Memory Management

**Bounded Collections**:
```go
type BoundedConfig struct {
    MaxLogEntries     int   // 10,000
    MaxProcessHistory int   // 1,000
    MaxRequestHistory int   // 1,000
    MaxEventQueue     int   // 5,000
    MaxErrorEntries   int   // 500
}
```

**Memory Monitoring**:
```go
type MemoryMonitor struct {
    thresholds struct {
        warning  uint64  // 80% of available
        critical uint64  // 95% of available
    }
    
    checkInterval time.Duration
    stopCh        chan struct{}
}
```

### 3. Channel Management

**Channel Configuration**:
```go
type ChannelConfig struct {
    EventBusQueues [3]int  // [100, 500, 1000] High/Med/Low priority
    LogStore       int     // 1000
    ProcessMgr     int     // 100
    ProxyRequests  int     // 500
}
```

## Configuration Schema Integration

### Concurrency Configuration

```go
type ConcurrencyConfig struct {
    // EventBus configuration
    EventBus struct {
        Workers           int    `toml:"worker_pool_size"`
        QueueSizes        [3]int `toml:"queue_sizes"`
        BackpressureMode  string `toml:"backpressure_strategy"`
        MonitoringEnabled bool   `toml:"enable_monitoring"`
    } `toml:"eventbus"`
    
    // Process Manager configuration
    ProcessManager struct {
        MaxConcurrentStarts int    `toml:"max_concurrent_starts"`
        StatusReadStrategy  string `toml:"status_read_strategy"`
        WorkerTimeout       string `toml:"worker_timeout"`
    } `toml:"process_manager"`
    
    // Log Store configuration
    LogStore struct {
        BufferSize   int    `toml:"buffer_size"`
        BatchSize    int    `toml:"batch_size"`
        WorkerCount  int    `toml:"worker_count"`
        TimeoutMs    int    `toml:"timeout_ms"`
    } `toml:"log_store"`
    
    // Proxy Server configuration
    ProxyServer struct {
        RequestBufferSize   int `toml:"request_buffer_size"`
        ConnectionTimeoutMs int `toml:"connection_timeout_ms"`
        MaxWSClients        int `toml:"max_ws_clients"`
    } `toml:"proxy_server"`
    
    // MCP Manager configuration (already good)
    MCPManager struct {
        HealthCheckIntervalMs int `toml:"health_check_interval_ms"`
        SessionTimeoutMs      int `toml:"session_timeout_ms"`
        MaxRetryAttempts      int `toml:"max_retry_attempts"`
    } `toml:"mcp_manager"`
    
    // Global resource limits
    Resources struct {
        MaxGoroutines      int `toml:"max_goroutines"`
        MemoryLimitMB      int `toml:"memory_limit_mb"`
        MonitoringInterval int `toml:"monitoring_interval_ms"`
    } `toml:"resources"`
}
```

### Default Configuration

```go
func DefaultConcurrencyConfig() ConcurrencyConfig {
    cpuCores := runtime.NumCPU()
    
    return ConcurrencyConfig{
        EventBus: struct {
            Workers           int    `toml:"worker_pool_size"`
            QueueSizes        [3]int `toml:"queue_sizes"`
            BackpressureMode  string `toml:"backpressure_strategy"`
            MonitoringEnabled bool   `toml:"enable_monitoring"`
        }{
            Workers:           max(4, min(cpuCores*2.5, 32)),
            QueueSizes:        [3]int{100, 500, 1000},
            BackpressureMode:  "drop_oldest",
            MonitoringEnabled: true,
        },
        ProcessManager: struct {
            MaxConcurrentStarts int    `toml:"max_concurrent_starts"`
            StatusReadStrategy  string `toml:"status_read_strategy"`
            WorkerTimeout       string `toml:"worker_timeout"`
        }{
            MaxConcurrentStarts: 10,
            StatusReadStrategy:  "atomic",
            WorkerTimeout:       "5s",
        },
        LogStore: struct {
            BufferSize   int `toml:"buffer_size"`
            BatchSize    int `toml:"batch_size"`
            WorkerCount  int `toml:"worker_count"`
            TimeoutMs    int `toml:"timeout_ms"`
        }{
            BufferSize:  10000,
            BatchSize:   100,
            WorkerCount: 1,
            TimeoutMs:   50,
        },
        ProxyServer: struct {
            RequestBufferSize   int `toml:"request_buffer_size"`
            ConnectionTimeoutMs int `toml:"connection_timeout_ms"`
            MaxWSClients        int `toml:"max_ws_clients"`
        }{
            RequestBufferSize:   1000,
            ConnectionTimeoutMs: 5000,
            MaxWSClients:        50,
        },
        MCPManager: struct {
            HealthCheckIntervalMs int `toml:"health_check_interval_ms"`
            SessionTimeoutMs      int `toml:"session_timeout_ms"`
            MaxRetryAttempts      int `toml:"max_retry_attempts"`
        }{
            HealthCheckIntervalMs: 30000,
            SessionTimeoutMs:      300000,
            MaxRetryAttempts:      3,
        },
        Resources: struct {
            MaxGoroutines      int `toml:"max_goroutines"`
            MemoryLimitMB      int `toml:"memory_limit_mb"`
            MonitoringInterval int `toml:"monitoring_interval_ms"`
        }{
            MaxGoroutines:      cpuCores * 8,
            MemoryLimitMB:      512,
            MonitoringInterval: 10000,
        },
    }
}
```

## Error Handling and Recovery

### 1. Panic Recovery

```go
type PanicRecovery struct {
    component string
    eventBus  *events.EventBus
}

func (pr *PanicRecovery) WrapHandler(handler func()) func() {
    return func() {
        defer func() {
            if r := recover(); r != nil {
                pr.eventBus.Publish(events.Event{
                    Type: events.ErrorDetected,
                    Data: map[string]interface{}{
                        "component": pr.component,
                        "panic":     r,
                        "stack":     debug.Stack(),
                    },
                })
            }
        }()
        handler()
    }
}
```

### 2. Graceful Degradation

```go
type GracefulDegradation struct {
    components map[string]ComponentHealth
    mu         sync.RWMutex
}

type ComponentHealth struct {
    Status         HealthStatus
    LastError      error
    ErrorCount     int
    LastHealthTime time.Time
}

type HealthStatus int
const (
    Healthy HealthStatus = iota
    Degraded
    Failing
    Failed
)
```

## Performance Monitoring

### 1. Metrics Collection

```go
type SynchronizationMetrics struct {
    // Lock contention metrics
    LockContentionCount atomic.Int64
    LockWaitTime        atomic.Int64
    
    // Channel metrics
    ChannelBlockedSends atomic.Int64
    ChannelBufferUsage  map[string]atomic.Int64
    
    // Goroutine metrics
    ActiveGoroutines    atomic.Int64
    GoroutineLeaks     atomic.Int64
    
    // Performance metrics
    EventProcessingTime atomic.Int64
    LogProcessingRate   atomic.Int64
    ProxyThroughput     atomic.Int64
}
```

### 2. Performance Thresholds

```go
type PerformanceThresholds struct {
    MaxEventProcessingTime time.Duration // 100ms
    MaxLockWaitTime        time.Duration // 10ms
    MaxChannelBlockTime    time.Duration // 50ms
    MaxGoroutineCount      int           // CPU cores * 8
    MaxMemoryUsageMB       int           // 512MB
}
```

## Implementation Strategy

### Phase 1: Critical Components (Week 1)
1. **EventBus Worker Pool**: Replace unlimited goroutines
2. **TUI Model Pointers**: Convert to pointer receivers
3. **Process Manager Status**: Implement atomic status operations

### Phase 2: Data Consistency (Week 2)
1. **Log Store Refactor**: Single-channel architecture
2. **Proxy Server Cleanup**: Unified mutex pattern
3. **Configuration Integration**: Concurrency settings

### Phase 3: Performance & Monitoring (Week 3)
1. **Resource Management**: Goroutine and memory limits
2. **Metrics Collection**: Performance monitoring
3. **Graceful Degradation**: Error recovery patterns

### Rollback Strategy

Each component maintains backward compatibility:
```go
type ComponentConfig struct {
    EnableNewSync bool `toml:"enable_new_sync"`
    FallbackMode  bool `toml:"fallback_mode"`
}
```

## Testing Strategy

### 1. Race Detection
```bash
# Continuous race testing
go test -race -timeout 30s ./internal/...
go test -race -run TestConcurrency ./...
```

### 2. Load Testing
```go
func TestConcurrentLoad(t *testing.T) {
    const (
        goroutines = 100
        operations = 1000
        duration   = 30 * time.Second
    )
    
    // Test each component under load
    // Verify no deadlocks or race conditions
    // Check performance stays within thresholds
}
```

### 3. Stress Testing
```go
func TestResourceExhaustion(t *testing.T) {
    // Test behavior when limits are reached
    // Verify graceful degradation
    // Check recovery after resource release
}
```

## Compatibility Guarantees

### API Compatibility
- All public interfaces maintain existing signatures
- New configuration options are optional with sensible defaults
- Existing client code works without modification

### Performance Compatibility
- No more than 10% performance degradation on existing workloads
- Better performance under high concurrency
- Reduced memory usage through bounded collections

### Behavioral Compatibility
- Event ordering preserved where semantically required
- Error handling maintains existing behavior
- Log output format unchanged

## Conclusion

This synchronized architecture eliminates race conditions through:

1. **Bounded Resources**: No unlimited goroutines or unbounded queues
2. **Clear Synchronization**: Well-defined lock hierarchies and ownership
3. **Lock-Free Optimizations**: Atomic operations where appropriate
4. **Progressive Rollout**: Safe incremental implementation
5. **Comprehensive Monitoring**: Performance and health tracking

The design prioritizes the critical race conditions (EventBus, TUI Model) while providing a coherent overall architecture that can be implemented incrementally with full rollback capability.