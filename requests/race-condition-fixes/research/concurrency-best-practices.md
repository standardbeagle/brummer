# Go Concurrency Best Practices for Brummer

## Executive Summary

Based on analysis of the Brummer codebase, this document provides comprehensive concurrency best practices specifically tailored to the identified race conditions and synchronization patterns. The recommendations focus on practical solutions that align with Go's memory model and the project's architecture.

## Current Concurrency Patterns Analysis

### 1. EventBus Pattern (pkg/events/events.go)
**Current Implementation:**
```go
func (eb *EventBus) Publish(event Event) {
    eb.mu.RLock()
    handlers := eb.handlers[event.Type]
    eb.mu.RUnlock()

    for _, handler := range handlers {
        go handler(event) // ISSUE: Unlimited goroutine spawning
    }
}
```

**Issues Identified:**
- Unlimited goroutine creation per event
- No backpressure mechanism
- Potential goroutine leak
- No error handling for handler panics

### 2. Process Manager Pattern (internal/process/manager.go)
**Current Implementation:**
```go
type Manager struct {
    processes      map[string]*Process
    mu             sync.RWMutex
    logCallbacks   []LogCallback
    // ...
}
```

**Issues Identified:**
- Mixed RWMutex usage patterns
- Multiple read locks without proper coordination
- Callback slice modification without synchronization

### 3. Log Store Pattern (internal/logs/store.go)
**Current Implementation:**
```go
type Store struct {
    entries    []LogEntry
    byProcess  map[string][]int
    mu         sync.RWMutex
    addChan    chan *addLogRequest
    // Mixed async/sync operations
}
```

**Issues Identified:**
- Hybrid async/sync pattern creates complexity
- Timeout-based fallback to sync operations
- Channel buffer size may cause blocking

## Best Practice Recommendations

### 1. Mutex vs RWMutex Selection Guidelines

#### Use RWMutex When:
- Read operations are **>3x** more frequent than writes
- Critical sections contain **>50 lines** of read-heavy code
- Data structure is relatively stable (infrequent updates)

#### Use Mutex When:
- Write frequency is **>25%** of total operations
- Critical sections are **<20 lines**
- Simple coordination needed

#### For Brummer Components:
```go
// EventBus: Use RWMutex (read-heavy for handler lookup)
type EventBus struct {
    handlers map[EventType][]Handler
    mu       sync.RWMutex
}

// Process Manager: Use RWMutex (frequent status reads)
type Manager struct {
    processes map[string]*Process
    mu        sync.RWMutex
}

// URL mappings: Use Mutex (frequent updates)
type Server struct {
    urlMappings map[string]*URLMapping
    mu          sync.Mutex
}
```

### 2. Channel vs Mutex Decision Matrix

| Scenario | Recommendation | Rationale |
|----------|----------------|-----------|
| EventBus Publishing | Channel-based worker pool | Backpressure, rate limiting |
| Log Storage | Hybrid with bounded channels | High throughput with fallback |
| Process Status Updates | Mutex-protected maps | Low latency, direct access |
| URL Registration | Mutex-protected maps | Immediate consistency needed |
| Telemetry Collection | Channels with buffering | Natural producer-consumer |

### 3. Worker Pool Implementation Patterns

#### EventBus Worker Pool (Recommended)
```go
type EventBus struct {
    handlers   map[EventType][]Handler
    mu         sync.RWMutex
    
    // Worker pool
    eventChan  chan eventJob
    workers    int
    workerWG   sync.WaitGroup
    stopCh     chan struct{}
}

type eventJob struct {
    event    Event
    handlers []Handler
}

func NewEventBus(workers int) *EventBus {
    if workers <= 0 {
        workers = runtime.NumCPU() * 2 // Default: 2x CPU cores
    }
    
    eb := &EventBus{
        handlers:  make(map[EventType][]Handler),
        eventChan: make(chan eventJob, workers*10), // Buffer: 10x workers
        workers:   workers,
        stopCh:    make(chan struct{}),
    }
    
    // Start workers
    for i := 0; i < workers; i++ {
        eb.workerWG.Add(1)
        go eb.worker(i)
    }
    
    return eb
}

func (eb *EventBus) worker(id int) {
    defer eb.workerWG.Done()
    
    for {
        select {
        case job := <-eb.eventChan:
            eb.processEvent(job)
        case <-eb.stopCh:
            return
        }
    }
}

func (eb *EventBus) processEvent(job eventJob) {
    defer func() {
        if r := recover(); r != nil {
            // Log panic but don't crash worker
            log.Printf("Event handler panicked: %v", r)
        }
    }()
    
    for _, handler := range job.handlers {
        handler(job.event)
    }
}

func (eb *EventBus) Publish(event Event) error {
    event.Timestamp = time.Now()
    event.ID = generateEventID()

    eb.mu.RLock()
    handlers := make([]Handler, len(eb.handlers[event.Type]))
    copy(handlers, eb.handlers[event.Type])
    eb.mu.RUnlock()

    if len(handlers) == 0 {
        return nil
    }

    job := eventJob{
        event:    event,
        handlers: handlers,
    }

    select {
    case eb.eventChan <- job:
        return nil
    case <-time.After(100 * time.Millisecond):
        return fmt.Errorf("event queue full, dropping event")
    }
}
```

### 4. Context-Based Cancellation Patterns

#### Process Management with Context
```go
func (m *Manager) StartScript(ctx context.Context, scriptName string) (*Process, error) {
    // Create derived context for this process
    processCtx, cancel := context.WithCancel(ctx)
    
    process := &Process{
        ID:     generateID(),
        Name:   scriptName,
        ctx:    processCtx,
        cancel: cancel,
    }
    
    // Start with context monitoring
    go m.runProcessWithContext(process)
    
    return process, nil
}

func (m *Manager) runProcessWithContext(p *Process) {
    defer p.cancel()
    
    // Start the actual process
    cmd := exec.CommandContext(p.ctx, args...)
    
    // Monitor context cancellation
    go func() {
        <-p.ctx.Done()
        if cmd.Process != nil {
            cmd.Process.Kill()
        }
    }()
    
    err := cmd.Run()
    // Handle completion
}
```

### 5. Error Handling in Concurrent Operations

#### Structured Error Collection
```go
type ErrorCollector struct {
    errors []error
    mu     sync.Mutex
}

func (ec *ErrorCollector) Add(err error) {
    if err == nil {
        return
    }
    
    ec.mu.Lock()
    ec.errors = append(ec.errors, err)
    ec.mu.Unlock()
}

func (ec *ErrorCollector) HasErrors() bool {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    return len(ec.errors) > 0
}

func (ec *ErrorCollector) Errors() []error {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    
    result := make([]error, len(ec.errors))
    copy(result, ec.errors)
    return result
}

// Usage in concurrent operations
func (m *Manager) StopAllProcesses() error {
    collector := &ErrorCollector{}
    var wg sync.WaitGroup
    
    processes := m.GetAllProcesses()
    for _, p := range processes {
        wg.Add(1)
        go func(proc *Process) {
            defer wg.Done()
            if err := m.StopProcess(proc.ID); err != nil {
                collector.Add(fmt.Errorf("failed to stop %s: %w", proc.ID, err))
            }
        }(p)
    }
    
    wg.Wait()
    
    if collector.HasErrors() {
        return fmt.Errorf("multiple errors: %v", collector.Errors())
    }
    return nil
}
```

### 6. Resource Cleanup Strategies

#### Graceful Shutdown Pattern
```go
type Component interface {
    Start(ctx context.Context) error
    Stop() error
    Name() string
}

type Application struct {
    components []Component
    mu         sync.RWMutex
    running    bool
}

func (app *Application) Start(ctx context.Context) error {
    app.mu.Lock()
    defer app.mu.Unlock()
    
    if app.running {
        return fmt.Errorf("already running")
    }
    
    // Start components in order
    for i, comp := range app.components {
        if err := comp.Start(ctx); err != nil {
            // Cleanup already started components
            for j := i - 1; j >= 0; j-- {
                app.components[j].Stop()
            }
            return fmt.Errorf("failed to start %s: %w", comp.Name(), err)
        }
    }
    
    app.running = true
    return nil
}

func (app *Application) Stop() error {
    app.mu.Lock()
    defer app.mu.Unlock()
    
    if !app.running {
        return nil
    }
    
    var lastError error
    
    // Stop components in reverse order
    for i := len(app.components) - 1; i >= 0; i-- {
        if err := app.components[i].Stop(); err != nil {
            lastError = err
            log.Printf("Error stopping %s: %v", app.components[i].Name(), err)
        }
    }
    
    app.running = false
    return lastError
}
```

### 7. Memory Management Patterns

#### Bounded Collections
```go
type BoundedSlice struct {
    items    []interface{}
    maxSize  int
    mu       sync.RWMutex
}

func NewBoundedSlice(maxSize int) *BoundedSlice {
    return &BoundedSlice{
        items:   make([]interface{}, 0, maxSize),
        maxSize: maxSize,
    }
}

func (bs *BoundedSlice) Add(item interface{}) {
    bs.mu.Lock()
    defer bs.mu.Unlock()
    
    bs.items = append(bs.items, item)
    
    // Trim to max size
    if len(bs.items) > bs.maxSize {
        // Remove oldest items
        removeCount := len(bs.items) - bs.maxSize
        copy(bs.items, bs.items[removeCount:])
        bs.items = bs.items[:bs.maxSize]
    }
}

func (bs *BoundedSlice) GetAll() []interface{} {
    bs.mu.RLock()
    defer bs.mu.RUnlock()
    
    result := make([]interface{}, len(bs.items))
    copy(result, bs.items)
    return result
}
```

## Implementation Priorities for Brummer

### High Priority (Critical Race Conditions)
1. **EventBus Worker Pool**: Implement bounded worker pool to prevent goroutine leaks
2. **Process Manager Synchronization**: Fix concurrent map access in process management
3. **TUI Model Pointer Receivers**: Convert value receivers to pointer receivers

### Medium Priority (Performance Improvements)
1. **Log Store Optimization**: Streamline async/sync operations
2. **Proxy Server Cleanup**: Consolidate multiple mutexes
3. **Connection Manager Channels**: Implement proper channel-based state management

### Low Priority (Defensive Programming)
1. **Error Handling**: Add structured error collection
2. **Resource Cleanup**: Implement graceful shutdown patterns
3. **Memory Management**: Add bounded collections where appropriate

## Testing Strategies

### Race Detection Integration
```bash
# In Makefile, add race detection to critical tests
test-race-critical:
	@go test -race -timeout 30s \
		./pkg/events \
		./internal/process \
		./internal/logs \
		./internal/proxy
```

### Stress Testing
```go
func TestEventBusStress(t *testing.T) {
    eb := NewEventBus(4) // 4 workers
    defer eb.Stop()
    
    const numEvents = 10000
    const numGoroutines = 100
    
    var wg sync.WaitGroup
    
    // Subscribe handlers
    for i := 0; i < 10; i++ {
        eb.Subscribe(ProcessStarted, func(e Event) {
            time.Sleep(time.Microsecond) // Simulate work
        })
    }
    
    // Publish events concurrently
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < numEvents/numGoroutines; j++ {
                eb.Publish(Event{Type: ProcessStarted})
            }
        }()
    }
    
    wg.Wait()
    
    // Verify no goroutine leaks
    time.Sleep(100 * time.Millisecond)
    // Check runtime.NumGoroutine() hasn't exploded
}
```

## Performance Considerations

### Lock Contention Reduction
- Use read-heavy optimizations (RWMutex)
- Implement lock-free operations where possible
- Consider atomic operations for counters

### Memory Allocation
- Pre-allocate slices with known capacity
- Reuse objects through sync.Pool
- Implement bounded collections

### Goroutine Management
- Use worker pools instead of unlimited goroutines
- Implement proper backpressure
- Monitor goroutine counts in production

## Conclusion

These patterns provide a solid foundation for eliminating race conditions while maintaining high performance. The key is to be consistent in application and to always consider the specific access patterns of each data structure when choosing synchronization primitives.

The next step is to implement these patterns systematically, starting with the highest-priority issues identified in the static analysis phase.