# Resource Limits and Monitoring Specifications

## Overview

This document defines comprehensive resource management, monitoring, and alerting specifications for the synchronized Brummer architecture. It ensures bounded resource usage, prevents resource exhaustion, and provides observability into system behavior.

## Resource Management Architecture

### 1. Resource Categories and Limits

#### Computational Resources

```go
type ComputationalLimits struct {
    // Goroutine management
    MaxGoroutines      int `json:"max_goroutines"`      // Default: CPU cores * 8
    MaxWorkerPools     int `json:"max_worker_pools"`    // Default: 10
    MaxChannelBuffers  int `json:"max_channel_buffers"` // Default: 100,000 total
    
    // CPU utilization
    CPUThresholdWarn   float64 `json:"cpu_threshold_warn"`   // Default: 80%
    CPUThresholdCrit   float64 `json:"cpu_threshold_crit"`   // Default: 95%
    CPUMonitorInterval time.Duration `json:"cpu_monitor_interval"` // Default: 5s
    
    // Context management
    MaxConcurrentOps   int `json:"max_concurrent_ops"`   // Default: 1000
    OperationTimeout   time.Duration `json:"operation_timeout"` // Default: 30s
    
    // Scheduler configuration
    GoroutineLeakThreshold int `json:"goroutine_leak_threshold"` // Default: 50
    SchedulerPreemption    bool `json:"scheduler_preemption"`    // Default: true
}

// Default computational limits based on system capabilities
func DefaultComputationalLimits() ComputationalLimits {
    cpuCores := runtime.NumCPU()
    
    return ComputationalLimits{
        MaxGoroutines:          max(32, min(cpuCores*8, 512)),
        MaxWorkerPools:         10,
        MaxChannelBuffers:      100000,
        CPUThresholdWarn:       0.80,
        CPUThresholdCrit:       0.95,
        CPUMonitorInterval:     5 * time.Second,
        MaxConcurrentOps:       1000,
        OperationTimeout:       30 * time.Second,
        GoroutineLeakThreshold: 50,
        SchedulerPreemption:    true,
    }
}
```

#### Memory Resources

```go
type MemoryLimits struct {
    // Heap management
    MaxHeapSize         uint64 `json:"max_heap_size_mb"`         // Default: 512MB
    HeapGrowthThreshold uint64 `json:"heap_growth_threshold_mb"` // Default: 100MB
    GCTargetPercent     int    `json:"gc_target_percent"`        // Default: 100
    
    // Memory pools
    MaxLogEntries       int `json:"max_log_entries"`       // Default: 10,000
    MaxRequestHistory   int `json:"max_request_history"`   // Default: 1,000
    MaxProcessHistory   int `json:"max_process_history"`   // Default: 1,000
    MaxErrorEntries     int `json:"max_error_entries"`     // Default: 500
    MaxURLEntries       int `json:"max_url_entries"`       // Default: 100
    
    // Buffer management
    MaxStringBufferSize int `json:"max_string_buffer_size"` // Default: 1MB
    MaxJSONBufferSize   int `json:"max_json_buffer_size"`   // Default: 10MB
    
    // Memory monitoring
    MemoryCheckInterval time.Duration `json:"memory_check_interval"` // Default: 10s
    MemoryWarnThreshold float64       `json:"memory_warn_threshold"` // Default: 80%
    MemoryCritThreshold float64       `json:"memory_crit_threshold"` // Default: 95%
    
    // Leak detection
    EnableLeakDetection bool          `json:"enable_leak_detection"` // Default: true
    LeakCheckInterval   time.Duration `json:"leak_check_interval"`   // Default: 60s
}

func DefaultMemoryLimits() MemoryLimits {
    return MemoryLimits{
        MaxHeapSize:         512 * 1024 * 1024, // 512MB
        HeapGrowthThreshold: 100 * 1024 * 1024, // 100MB
        GCTargetPercent:     100,
        MaxLogEntries:       10000,
        MaxRequestHistory:   1000,
        MaxProcessHistory:   1000,
        MaxErrorEntries:     500,
        MaxURLEntries:       100,
        MaxStringBufferSize: 1024 * 1024,  // 1MB
        MaxJSONBufferSize:   10 * 1024 * 1024, // 10MB
        MemoryCheckInterval: 10 * time.Second,
        MemoryWarnThreshold: 0.80,
        MemoryCritThreshold: 0.95,
        EnableLeakDetection: true,
        LeakCheckInterval:   60 * time.Second,
    }
}
```

#### Network Resources

```go
type NetworkLimits struct {
    // Connection management
    MaxTotalConnections    int `json:"max_total_connections"`    // Default: 100
    MaxProxyConnections    int `json:"max_proxy_connections"`    // Default: 50
    MaxMCPConnections      int `json:"max_mcp_connections"`      // Default: 10
    MaxWebSocketClients    int `json:"max_websocket_clients"`    // Default: 25
    
    // Bandwidth management
    MaxBandwidthMbps       float64 `json:"max_bandwidth_mbps"`       // Default: 100
    MaxRequestsPerSecond   int     `json:"max_requests_per_second"`   // Default: 1000
    MaxResponseSizeMB      int     `json:"max_response_size_mb"`      // Default: 100
    
    // Timeout configuration
    ConnectionTimeout      time.Duration `json:"connection_timeout"`      // Default: 10s
    ReadTimeout           time.Duration `json:"read_timeout"`            // Default: 30s
    WriteTimeout          time.Duration `json:"write_timeout"`           // Default: 30s
    IdleTimeout           time.Duration `json:"idle_timeout"`            // Default: 120s
    
    // Rate limiting
    EnableRateLimit       bool    `json:"enable_rate_limit"`       // Default: true
    RateLimitWindow       time.Duration `json:"rate_limit_window"` // Default: 1m
    RateLimitMaxRequests  int     `json:"rate_limit_max_requests"` // Default: 6000
    
    // Circuit breaker
    CircuitBreakerEnabled bool    `json:"circuit_breaker_enabled"` // Default: true
    CircuitBreakerThreshold int   `json:"circuit_breaker_threshold"` // Default: 5
    CircuitBreakerTimeout time.Duration `json:"circuit_breaker_timeout"` // Default: 30s
}

func DefaultNetworkLimits() NetworkLimits {
    return NetworkLimits{
        MaxTotalConnections:     100,
        MaxProxyConnections:     50,
        MaxMCPConnections:       10,
        MaxWebSocketClients:     25,
        MaxBandwidthMbps:        100.0,
        MaxRequestsPerSecond:    1000,
        MaxResponseSizeMB:       100,
        ConnectionTimeout:       10 * time.Second,
        ReadTimeout:            30 * time.Second,
        WriteTimeout:           30 * time.Second,
        IdleTimeout:            120 * time.Second,
        EnableRateLimit:        true,
        RateLimitWindow:        time.Minute,
        RateLimitMaxRequests:   6000,
        CircuitBreakerEnabled:  true,
        CircuitBreakerThreshold: 5,
        CircuitBreakerTimeout:  30 * time.Second,
    }
}
```

#### File System Resources

```go
type FileSystemLimits struct {
    // File descriptor management
    MaxFileDescriptors     int `json:"max_file_descriptors"`     // Default: 1000
    MaxOpenFiles          int `json:"max_open_files"`           // Default: 100
    MaxLogFiles           int `json:"max_log_files"`            // Default: 5
    
    // Storage limits
    MaxLogFileSizeMB      int `json:"max_log_file_size_mb"`     // Default: 10MB
    MaxTotalLogSizeMB     int `json:"max_total_log_size_mb"`    // Default: 100MB
    MaxTempFileSizeMB     int `json:"max_temp_file_size_mb"`    // Default: 50MB
    
    // I/O performance
    MaxConcurrentReads    int `json:"max_concurrent_reads"`     // Default: 10
    MaxConcurrentWrites   int `json:"max_concurrent_writes"`    // Default: 5
    IOTimeout            time.Duration `json:"io_timeout"`     // Default: 10s
    
    // Disk space monitoring
    MinFreeDiskSpaceMB    int `json:"min_free_disk_space_mb"`   // Default: 1000MB
    DiskSpaceCheckInterval time.Duration `json:"disk_space_check_interval"` // Default: 60s
    
    // File watching
    MaxWatchedFiles       int `json:"max_watched_files"`        // Default: 100
    FileWatchTimeout      time.Duration `json:"file_watch_timeout"` // Default: 5s
}

func DefaultFileSystemLimits() FileSystemLimits {
    return FileSystemLimits{
        MaxFileDescriptors:     1000,
        MaxOpenFiles:          100,
        MaxLogFiles:           5,
        MaxLogFileSizeMB:      10,
        MaxTotalLogSizeMB:     100,
        MaxTempFileSizeMB:     50,
        MaxConcurrentReads:    10,
        MaxConcurrentWrites:   5,
        IOTimeout:            10 * time.Second,
        MinFreeDiskSpaceMB:    1000,
        DiskSpaceCheckInterval: 60 * time.Second,
        MaxWatchedFiles:       100,
        FileWatchTimeout:      5 * time.Second,
    }
}
```

### 2. Component-Specific Resource Allocations

#### EventBus Resource Budget

```go
type EventBusResources struct {
    // Worker pool allocation
    WorkerGoroutines     int `json:"worker_goroutines"`     // Default: CPU cores * 2.5
    MaxQueuedEvents      int `json:"max_queued_events"`     // Default: 10,000
    MaxHandlersPerType   int `json:"max_handlers_per_type"` // Default: 100
    
    // Memory allocation
    EventBufferSizeMB    int `json:"event_buffer_size_mb"`  // Default: 50MB
    HandlerMemoryMB      int `json:"handler_memory_mb"`     // Default: 100MB
    
    // Performance limits
    MaxEventsPerSecond   int           `json:"max_events_per_second"`   // Default: 10,000
    HandlerTimeout       time.Duration `json:"handler_timeout"`         // Default: 5s
    QueueFullBackoff     time.Duration `json:"queue_full_backoff"`      // Default: 100ms
    
    // Error handling
    MaxHandlerFailures   int `json:"max_handler_failures"`   // Default: 10 per minute
    PanicRecoveryEnabled bool `json:"panic_recovery_enabled"` // Default: true
}

func (eb *EventBus) GetResourceUsage() EventBusResourceUsage {
    return EventBusResourceUsage{
        ActiveWorkers:     eb.getActiveWorkerCount(),
        QueuedEvents:      eb.getQueuedEventCount(),
        RegisteredHandlers: eb.getHandlerCount(),
        MemoryUsageMB:     eb.getMemoryUsage() / 1024 / 1024,
        EventsPerSecond:   eb.getEventsPerSecond(),
        FailedHandlers:    eb.getFailedHandlerCount(),
    }
}
```

#### Process Manager Resource Budget

```go
type ProcessManagerResources struct {
    // Process limits
    MaxConcurrentProcesses int `json:"max_concurrent_processes"` // Default: 50
    MaxProcessHistory     int `json:"max_process_history"`      // Default: 1000
    MaxLogCallbacks       int `json:"max_log_callbacks"`        // Default: 100
    
    // Memory allocation
    ProcessBufferMB       int `json:"process_buffer_mb"`        // Default: 100MB
    LogBufferMB          int `json:"log_buffer_mb"`            // Default: 50MB
    
    // Performance limits
    MaxProcessStartsPerMin int           `json:"max_process_starts_per_min"` // Default: 60
    ProcessStartTimeout    time.Duration `json:"process_start_timeout"`      // Default: 30s
    ProcessStopTimeout     time.Duration `json:"process_stop_timeout"`       // Default: 10s
    
    // Resource monitoring
    ProcessCPUThreshold    float64 `json:"process_cpu_threshold"`    // Default: 80%
    ProcessMemoryThresholdMB int   `json:"process_memory_threshold_mb"` // Default: 100MB
}

func (pm *ProcessManager) GetResourceUsage() ProcessManagerResourceUsage {
    return ProcessManagerResourceUsage{
        ActiveProcesses:   len(pm.GetAllProcesses()),
        MemoryUsageMB:    pm.getMemoryUsage() / 1024 / 1024,
        CPUUsagePercent:  pm.getCPUUsage(),
        FileDescriptors:  pm.getFileDescriptorCount(),
        LogCallbackCount: len(pm.logCallbacks),
    }
}
```

### 3. Resource Allocation Strategies

#### Dynamic Resource Allocation

```go
type ResourceAllocator struct {
    totalLimits     ResourceLimits
    componentBudgets map[ComponentType]ResourceBudget
    currentUsage    ResourceUsage
    mu              sync.RWMutex
}

type ResourceBudget struct {
    GoroutineQuota    int     `json:"goroutine_quota"`
    MemoryQuotaMB     int     `json:"memory_quota_mb"`
    ConnectionQuota   int     `json:"connection_quota"`
    Priority          int     `json:"priority"`      // 1=highest, 10=lowest
    Elasticity        float64 `json:"elasticity"`    // 0.0-1.0, how much can expand
}

func (ra *ResourceAllocator) AllocateResource(
    component ComponentType, 
    resourceType ResourceType, 
    amount int,
) (*ResourceHandle, error) {
    ra.mu.Lock()
    defer ra.mu.Unlock()
    
    budget := ra.componentBudgets[component]
    usage := ra.currentUsage.GetComponentUsage(component)
    
    // Check if allocation is within budget
    if !ra.canAllocate(budget, usage, resourceType, amount) {
        // Try elastic expansion
        if budget.Elasticity > 0 {
            if ra.tryElasticExpansion(component, resourceType, amount) {
                return ra.doAllocate(component, resourceType, amount)
            }
        }
        return nil, ErrResourceQuotaExceeded
    }
    
    return ra.doAllocate(component, resourceType, amount)
}

func (ra *ResourceAllocator) canAllocate(
    budget ResourceBudget,
    usage ComponentResourceUsage,
    resourceType ResourceType,
    amount int,
) bool {
    switch resourceType {
    case ResourceTypeGoroutines:
        return usage.Goroutines + amount <= budget.GoroutineQuota
    case ResourceTypeMemory:
        return usage.MemoryMB + amount <= budget.MemoryQuotaMB
    case ResourceTypeConnections:
        return usage.Connections + amount <= budget.ConnectionQuota
    default:
        return false
    }
}

func (ra *ResourceAllocator) tryElasticExpansion(
    component ComponentType,
    resourceType ResourceType,
    amount int,
) bool {
    // Check if system has available resources
    available := ra.getAvailableResources(resourceType)
    if available < amount {
        return false
    }
    
    // Check if other components can spare resources
    return ra.borrowFromLowerPriority(component, resourceType, amount)
}
```

#### Resource Borrowing and Load Balancing

```go
type ResourceRebalancer struct {
    allocator *ResourceAllocator
    metrics   *ResourceMetrics
    policies  []RebalancingPolicy
}

type RebalancingPolicy interface {
    ShouldRebalance(usage ResourceUsage) bool
    CalculateNewAllocations(usage ResourceUsage) map[ComponentType]ResourceBudget
    Priority() int
}

// Load-based rebalancing policy
type LoadBasedPolicy struct {
    thresholds map[ResourceType]float64
}

func (lbp *LoadBasedPolicy) ShouldRebalance(usage ResourceUsage) bool {
    for resourceType, threshold := range lbp.thresholds {
        utilization := usage.GetUtilization(resourceType)
        if utilization > threshold {
            return true
        }
    }
    return false
}

func (lbp *LoadBasedPolicy) CalculateNewAllocations(
    usage ResourceUsage,
) map[ComponentType]ResourceBudget {
    allocations := make(map[ComponentType]ResourceBudget)
    
    // Sort components by current utilization
    components := usage.GetComponentsSortedByUtilization()
    
    for _, component := range components {
        budget := lbp.calculateOptimalBudget(component, usage)
        allocations[component.Type] = budget
    }
    
    return allocations
}

// Predictive rebalancing based on historical patterns
type PredictivePolicy struct {
    history     *ResourceUsageHistory
    predictor   *UsagePredictor
    lookahead   time.Duration
}

func (pp *PredictivePolicy) ShouldRebalance(usage ResourceUsage) bool {
    prediction := pp.predictor.PredictUsage(pp.lookahead)
    
    // Rebalance if predicted usage will exceed thresholds
    for resourceType, threshold := range pp.getThresholds() {
        if prediction.GetUtilization(resourceType) > threshold {
            return true
        }
    }
    return false
}
```

## Monitoring and Observability

### 1. Metrics Collection Architecture

#### Core Metrics Framework

```go
type MetricsCollector struct {
    registry    *MetricsRegistry
    collectors  map[ComponentType]ComponentCollector
    exporters   []MetricsExporter
    aggregator  *MetricsAggregator
    retention   time.Duration
}

type MetricsRegistry struct {
    counters    map[string]*Counter
    gauges      map[string]*Gauge
    histograms  map[string]*Histogram
    timers      map[string]*Timer
    mu          sync.RWMutex
}

// Resource utilization metrics
type ResourceMetrics struct {
    // Goroutine metrics
    GoroutineCount        *Gauge     `json:"goroutine_count"`
    GoroutineCreationRate *Counter   `json:"goroutine_creation_rate"`
    GoroutineLeaks        *Counter   `json:"goroutine_leaks"`
    
    // Memory metrics
    HeapSize              *Gauge     `json:"heap_size_bytes"`
    HeapObjects           *Gauge     `json:"heap_objects"`
    GCPauses              *Histogram `json:"gc_pauses_ms"`
    AllocRate             *Gauge     `json:"alloc_rate_bytes_per_sec"`
    
    // CPU metrics
    CPUUsage              *Gauge     `json:"cpu_usage_percent"`
    ThreadCount           *Gauge     `json:"thread_count"`
    ContextSwitches       *Counter   `json:"context_switches"`
    
    // Network metrics
    ActiveConnections     *Gauge     `json:"active_connections"`
    BytesSent             *Counter   `json:"bytes_sent"`
    BytesReceived         *Counter   `json:"bytes_received"`
    ConnectionErrors      *Counter   `json:"connection_errors"`
    
    // File system metrics
    OpenFileDescriptors   *Gauge     `json:"open_file_descriptors"`
    DiskUsageBytes        *Gauge     `json:"disk_usage_bytes"`
    IOOperations          *Counter   `json:"io_operations"`
    IOErrors              *Counter   `json:"io_errors"`
}

func (mc *MetricsCollector) CollectResourceMetrics() {
    // Goroutine metrics
    mc.recordGoroutineMetrics()
    
    // Memory metrics
    mc.recordMemoryMetrics()
    
    // CPU metrics
    mc.recordCPUMetrics()
    
    // Network metrics
    mc.recordNetworkMetrics()
    
    // File system metrics
    mc.recordFileSystemMetrics()
}

func (mc *MetricsCollector) recordGoroutineMetrics() {
    count := runtime.NumGoroutine()
    mc.registry.GetGauge("goroutine_count").Set(float64(count))
    
    // Detect goroutine leaks
    if count > mc.config.GoroutineLeakThreshold {
        mc.registry.GetCounter("goroutine_leaks").Inc()
        mc.alertManager.Trigger(AlertGoroutineLeak, map[string]interface{}{
            "current_count": count,
            "threshold":     mc.config.GoroutineLeakThreshold,
        })
    }
}

func (mc *MetricsCollector) recordMemoryMetrics() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    mc.registry.GetGauge("heap_size_bytes").Set(float64(m.HeapInuse))
    mc.registry.GetGauge("heap_objects").Set(float64(m.HeapObjects))
    mc.registry.GetGauge("alloc_rate_bytes_per_sec").Set(float64(m.TotalAlloc))
    
    // Record GC pause times
    gcPause := float64(m.PauseNs[(m.NumGC+255)%256]) / 1e6 // Convert to ms
    mc.registry.GetHistogram("gc_pauses_ms").Observe(gcPause)
    
    // Check memory thresholds
    heapPercent := float64(m.HeapInuse) / float64(mc.config.MaxHeapSize) * 100
    if heapPercent > mc.config.MemoryWarnThreshold {
        mc.alertManager.Trigger(AlertHighMemoryUsage, map[string]interface{}{
            "usage_percent": heapPercent,
            "heap_size":     m.HeapInuse,
            "threshold":     mc.config.MemoryWarnThreshold,
        })
    }
}
```

#### Component-Specific Metrics

```go
// EventBus metrics
type EventBusMetrics struct {
    EventsPublished       *Counter   `json:"events_published"`
    EventsDropped         *Counter   `json:"events_dropped"`
    HandlerExecutions     *Counter   `json:"handler_executions"`
    HandlerFailures       *Counter   `json:"handler_failures"`
    HandlerPanics         *Counter   `json:"handler_panics"`
    
    QueueDepth            *Gauge     `json:"queue_depth"`
    WorkerUtilization     *Gauge     `json:"worker_utilization"`
    EventProcessingTime   *Histogram `json:"event_processing_time_ms"`
    
    BackpressureEvents    *Counter   `json:"backpressure_events"`
    CircuitBreakerTrips   *Counter   `json:"circuit_breaker_trips"`
}

// Process Manager metrics
type ProcessManagerMetrics struct {
    ProcessesStarted      *Counter   `json:"processes_started"`
    ProcessesStopped      *Counter   `json:"processes_stopped"`
    ProcessesFailed       *Counter   `json:"processes_failed"`
    ProcessStartTime      *Histogram `json:"process_start_time_ms"`
    ProcessStopTime       *Histogram `json:"process_stop_time_ms"`
    
    ActiveProcesses       *Gauge     `json:"active_processes"`
    LogCallbacks          *Gauge     `json:"log_callbacks"`
    ProcessMemoryUsage    *Gauge     `json:"process_memory_usage_mb"`
    ProcessCPUUsage       *Gauge     `json:"process_cpu_usage_percent"`
}

// Log Store metrics
type LogStoreMetrics struct {
    LogEntriesAdded       *Counter   `json:"log_entries_added"`
    LogEntriesDropped     *Counter   `json:"log_entries_dropped"`
    LogSearchQueries      *Counter   `json:"log_search_queries"`
    LogSearchTime         *Histogram `json:"log_search_time_ms"`
    
    LogStoreSize          *Gauge     `json:"log_store_size_entries"`
    LogStoreMemoryMB      *Gauge     `json:"log_store_memory_mb"`
    URLEntriesTracked     *Gauge     `json:"url_entries_tracked"`
    ErrorEntriesTracked   *Gauge     `json:"error_entries_tracked"`
}

// Proxy Server metrics
type ProxyServerMetrics struct {
    RequestsHandled       *Counter   `json:"requests_handled"`
    RequestsFailed        *Counter   `json:"requests_failed"`
    BytesTransferred      *Counter   `json:"bytes_transferred"`
    ResponseTime          *Histogram `json:"response_time_ms"`
    
    ActiveConnections     *Gauge     `json:"active_connections"`
    WebSocketClients      *Gauge     `json:"websocket_clients"`
    URLMappings           *Gauge     `json:"url_mappings"`
    TelemetrySessions     *Gauge     `json:"telemetry_sessions"`
}
```

### 2. Alerting and Notification System

#### Alert Definitions

```go
type AlertDefinition struct {
    Name        string             `json:"name"`
    Description string             `json:"description"`
    Condition   AlertCondition     `json:"condition"`
    Severity    AlertSeverity      `json:"severity"`
    Cooldown    time.Duration      `json:"cooldown"`
    Actions     []AlertAction      `json:"actions"`
    Runbook     string             `json:"runbook"`
}

type AlertSeverity int
const (
    SeverityInfo AlertSeverity = iota
    SeverityWarning
    SeverityError
    SeverityCritical
)

type AlertCondition interface {
    Evaluate(metrics ResourceMetrics) bool
    Description() string
}

// Threshold-based alert condition
type ThresholdCondition struct {
    MetricName string  `json:"metric_name"`
    Operator   string  `json:"operator"` // >, <, >=, <=, ==, !=
    Threshold  float64 `json:"threshold"`
    Duration   time.Duration `json:"duration"` // Must be true for this long
}

func (tc *ThresholdCondition) Evaluate(metrics ResourceMetrics) bool {
    value := metrics.GetMetricValue(tc.MetricName)
    
    switch tc.Operator {
    case ">":
        return value > tc.Threshold
    case "<":
        return value < tc.Threshold
    case ">=":
        return value >= tc.Threshold
    case "<=":
        return value <= tc.Threshold
    case "==":
        return value == tc.Threshold
    case "!=":
        return value != tc.Threshold
    default:
        return false
    }
}

// Rate-based alert condition
type RateCondition struct {
    MetricName string        `json:"metric_name"`
    Window     time.Duration `json:"window"`
    Threshold  float64       `json:"threshold"` // Events per window
}

func (rc *RateCondition) Evaluate(metrics ResourceMetrics) bool {
    rate := metrics.GetRate(rc.MetricName, rc.Window)
    return rate > rc.Threshold
}

// Anomaly detection condition
type AnomalyCondition struct {
    MetricName      string  `json:"metric_name"`
    Sensitivity     float64 `json:"sensitivity"`     // Standard deviations
    MinSamples      int     `json:"min_samples"`     // Minimum samples for baseline
    LearningPeriod  time.Duration `json:"learning_period"`
}

func (ac *AnomalyCondition) Evaluate(metrics ResourceMetrics) bool {
    baseline := metrics.GetBaseline(ac.MetricName, ac.LearningPeriod)
    current := metrics.GetMetricValue(ac.MetricName)
    
    if baseline.SampleCount < ac.MinSamples {
        return false // Not enough data
    }
    
    deviation := math.Abs(current - baseline.Mean) / baseline.StdDev
    return deviation > ac.Sensitivity
}
```

#### Built-in Alert Definitions

```go
var DefaultAlerts = []AlertDefinition{
    // Resource exhaustion alerts
    {
        Name:        "HighGoroutineCount",
        Description: "Too many goroutines active",
        Condition: &ThresholdCondition{
            MetricName: "goroutine_count",
            Operator:   ">",
            Threshold:  float64(runtime.NumCPU() * 8),
            Duration:   30 * time.Second,
        },
        Severity: SeverityWarning,
        Cooldown: 5 * time.Minute,
        Actions: []AlertAction{
            &LogAction{Level: "warn"},
            &MetricAction{Name: "goroutine_leak_detected"},
        },
    },
    
    {
        Name:        "HighMemoryUsage",
        Description: "Memory usage exceeds threshold",
        Condition: &ThresholdCondition{
            MetricName: "heap_size_bytes",
            Operator:   ">",
            Threshold:  512 * 1024 * 1024 * 0.8, // 80% of 512MB
            Duration:   60 * time.Second,
        },
        Severity: SeverityError,
        Cooldown: 2 * time.Minute,
        Actions: []AlertAction{
            &LogAction{Level: "error"},
            &GCAction{},
            &DegradationAction{Level: ReducedService},
        },
    },
    
    {
        Name:        "EventQueueBackpressure",
        Description: "EventBus experiencing backpressure",
        Condition: &RateCondition{
            MetricName: "backpressure_events",
            Window:     time.Minute,
            Threshold:  10, // More than 10 backpressure events per minute
        },
        Severity: SeverityWarning,
        Cooldown: 1 * time.Minute,
        Actions: []AlertAction{
            &LogAction{Level: "warn"},
            &EventBusScaleAction{Direction: "up"},
        },
    },
    
    {
        Name:        "HighErrorRate",
        Description: "Error rate exceeds acceptable threshold",
        Condition: &ThresholdCondition{
            MetricName: "error_rate_per_minute",
            Operator:   ">",
            Threshold:  50, // More than 50 errors per minute
            Duration:   2 * time.Minute,
        },
        Severity: SeverityCritical,
        Cooldown: 5 * time.Minute,
        Actions: []AlertAction{
            &LogAction{Level: "error"},
            &CircuitBreakerAction{Component: "all"},
            &NotificationAction{Channel: "emergency"},
        },
    },
    
    // Performance degradation alerts
    {
        Name:        "SlowEventProcessing",
        Description: "Event processing time increasing",
        Condition: &AnomalyCondition{
            MetricName:     "event_processing_time_ms",
            Sensitivity:    2.0, // 2 standard deviations
            MinSamples:     100,
            LearningPeriod: 10 * time.Minute,
        },
        Severity: SeverityWarning,
        Cooldown: 3 * time.Minute,
        Actions: []AlertAction{
            &LogAction{Level: "warn"},
            &PerformanceAnalysisAction{},
        },
    },
    
    // Resource leak alerts
    {
        Name:        "ConnectionLeak",
        Description: "Network connections not being properly closed",
        Condition: &ThresholdCondition{
            MetricName: "active_connections",
            Operator:   ">",
            Threshold:  100,
            Duration:   5 * time.Minute,
        },
        Severity: SeverityError,
        Cooldown: 5 * time.Minute,
        Actions: []AlertAction{
            &LogAction{Level: "error"},
            &ConnectionCleanupAction{},
        },
    },
    
    // File system alerts
    {
        Name:        "LowDiskSpace",
        Description: "Available disk space running low",
        Condition: &ThresholdCondition{
            MetricName: "available_disk_space_mb",
            Operator:   "<",
            Threshold:  1000, // Less than 1GB available
            Duration:   30 * time.Second,
        },
        Severity: SeverityError,
        Cooldown: 10 * time.Minute,
        Actions: []AlertAction{
            &LogAction{Level: "error"},
            &LogRotationAction{},
            &TempFileCleanupAction{},
        },
    },
}
```

#### Alert Actions

```go
type AlertAction interface {
    Execute(alert Alert, context AlertContext) error
    Description() string
}

// Logging action
type LogAction struct {
    Level   string `json:"level"`
    Message string `json:"message"`
}

func (la *LogAction) Execute(alert Alert, context AlertContext) error {
    message := la.Message
    if message == "" {
        message = fmt.Sprintf("Alert triggered: %s - %s", alert.Name, alert.Description)
    }
    
    switch la.Level {
    case "debug":
        log.Debug(message)
    case "info":
        log.Info(message)
    case "warn":
        log.Warn(message)
    case "error":
        log.Error(message)
    default:
        log.Info(message)
    }
    
    return nil
}

// Metric increment action
type MetricAction struct {
    Name  string  `json:"name"`
    Value float64 `json:"value"`
}

func (ma *MetricAction) Execute(alert Alert, context AlertContext) error {
    if ma.Value == 0 {
        ma.Value = 1
    }
    
    context.MetricsCollector.GetCounter(ma.Name).Add(ma.Value)
    return nil
}

// Garbage collection action
type GCAction struct {
    Force bool `json:"force"`
}

func (ga *GCAction) Execute(alert Alert, context AlertContext) error {
    if ga.Force {
        runtime.GC()
        runtime.GC() // Force complete collection
    } else {
        go runtime.GC() // Async GC
    }
    return nil
}

// Service degradation action
type DegradationAction struct {
    Level DegradationLevel `json:"level"`
}

func (da *DegradationAction) Execute(alert Alert, context AlertContext) error {
    return context.Application.SetDegradationLevel(da.Level)
}

// Circuit breaker action
type CircuitBreakerAction struct {
    Component string        `json:"component"`
    Duration  time.Duration `json:"duration"`
}

func (cba *CircuitBreakerAction) Execute(alert Alert, context AlertContext) error {
    if cba.Duration == 0 {
        cba.Duration = 30 * time.Second
    }
    
    if cba.Component == "all" {
        return context.Application.EnableCircuitBreaker(cba.Duration)
    }
    
    return context.Application.EnableComponentCircuitBreaker(cba.Component, cba.Duration)
}
```

### 3. Health Check System

#### Component Health Monitoring

```go
type HealthChecker struct {
    components map[ComponentType]HealthCheck
    checks     []SystemHealthCheck
    interval   time.Duration
    timeout    time.Duration
    history    *HealthHistory
}

type HealthCheck interface {
    CheckHealth(ctx context.Context) HealthStatus
    GetHealthMetrics() HealthMetrics
    GetLastError() error
}

type HealthStatus int
const (
    HealthUnknown HealthStatus = iota
    HealthHealthy
    HealthDegraded
    HealthUnhealthy
    HealthFailed
)

type HealthMetrics struct {
    Status            HealthStatus  `json:"status"`
    LastCheck         time.Time     `json:"last_check"`
    ResponseTime      time.Duration `json:"response_time"`
    ErrorCount        int           `json:"error_count"`
    ConsecutiveErrors int           `json:"consecutive_errors"`
    Uptime            time.Duration `json:"uptime"`
    Details           map[string]interface{} `json:"details"`
}

// EventBus health check
func (eb *EventBus) CheckHealth(ctx context.Context) HealthStatus {
    startTime := time.Now()
    
    // Check worker pool health
    if eb.getActiveWorkerCount() == 0 {
        return HealthFailed
    }
    
    // Check queue depths
    queueDepths := eb.getQueueDepths()
    for _, depth := range queueDepths {
        if depth > eb.config.QueueSizes[0]*0.9 { // 90% of capacity
            return HealthDegraded
        }
    }
    
    // Test event processing
    testEvent := events.Event{
        Type: events.EventType("health.check"),
        Data: map[string]interface{}{
            "timestamp": startTime,
            "id":        "health-check",
        },
    }
    
    done := make(chan bool, 1)
    eb.Subscribe(testEvent.Type, func(e events.Event) {
        done <- true
    })
    
    if err := eb.Publish(testEvent); err != nil {
        return HealthUnhealthy
    }
    
    select {
    case <-done:
        responseTime := time.Since(startTime)
        if responseTime > 100*time.Millisecond {
            return HealthDegraded
        }
        return HealthHealthy
    case <-time.After(1 * time.Second):
        return HealthUnhealthy
    case <-ctx.Done():
        return HealthUnknown
    }
}

// System-wide health checks
type SystemHealthCheck interface {
    Name() string
    Check(ctx context.Context) (bool, error)
    Critical() bool
}

// Goroutine leak check
type GoroutineLeakCheck struct {
    baselineCount int
    threshold     int
}

func (glc *GoroutineLeakCheck) Check(ctx context.Context) (bool, error) {
    currentCount := runtime.NumGoroutine()
    
    if glc.baselineCount == 0 {
        glc.baselineCount = currentCount
        return true, nil
    }
    
    growth := currentCount - glc.baselineCount
    if growth > glc.threshold {
        return false, fmt.Errorf(
            "goroutine leak detected: %d current, %d baseline, %d growth (threshold: %d)",
            currentCount, glc.baselineCount, growth, glc.threshold,
        )
    }
    
    return true, nil
}

// Memory health check
type MemoryHealthCheck struct {
    maxHeapSize   uint64
    warnThreshold float64
}

func (mhc *MemoryHealthCheck) Check(ctx context.Context) (bool, error) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    usage := float64(m.HeapInuse) / float64(mhc.maxHeapSize)
    if usage > mhc.warnThreshold {
        return false, fmt.Errorf(
            "high memory usage: %.2f%% (threshold: %.2f%%)",
            usage*100, mhc.warnThreshold*100,
        )
    }
    
    return true, nil
}

// Deadlock detection check
type DeadlockDetectionCheck struct {
    timeout time.Duration
}

func (ddc *DeadlockDetectionCheck) Check(ctx context.Context) (bool, error) {
    // Create test goroutines that acquire locks in different orders
    // If they don't complete within timeout, potential deadlock
    
    mutex1 := &sync.Mutex{}
    mutex2 := &sync.Mutex{}
    
    done := make(chan bool, 2)
    
    // Goroutine 1: lock mutex1 then mutex2
    go func() {
        mutex1.Lock()
        time.Sleep(10 * time.Millisecond) // Small delay
        mutex2.Lock()
        mutex2.Unlock()
        mutex1.Unlock()
        done <- true
    }()
    
    // Goroutine 2: lock mutex2 then mutex1
    go func() {
        mutex2.Lock()
        time.Sleep(10 * time.Millisecond) // Small delay
        mutex1.Lock()
        mutex1.Unlock()
        mutex2.Unlock()
        done <- true
    }()
    
    // Wait for both to complete or timeout
    completed := 0
    timeout := time.After(ddc.timeout)
    
    for completed < 2 {
        select {
        case <-done:
            completed++
        case <-timeout:
            return false, fmt.Errorf("potential deadlock detected")
        case <-ctx.Done():
            return false, ctx.Err()
        }
    }
    
    return true, nil
}
```

### 4. Performance Benchmarking

#### Continuous Performance Monitoring

```go
type PerformanceBenchmark struct {
    name        string
    baseline    BenchmarkResult
    current     BenchmarkResult
    threshold   float64 // Maximum acceptable degradation (e.g., 0.1 for 10%)
    history     []BenchmarkResult
    enabled     bool
}

type BenchmarkResult struct {
    Timestamp       time.Time     `json:"timestamp"`
    Duration        time.Duration `json:"duration"`
    OperationsPerSec float64      `json:"operations_per_sec"`
    MemoryAllocated uint64        `json:"memory_allocated"`
    GCPauses        time.Duration `json:"gc_pauses"`
    CPUUsage        float64       `json:"cpu_usage"`
    Success         bool          `json:"success"`
    Error           string        `json:"error,omitempty"`
}

func (pb *PerformanceBenchmark) Run() BenchmarkResult {
    startTime := time.Now()
    var memBefore runtime.MemStats
    runtime.ReadMemStats(&memBefore)
    
    // Run the benchmark
    operations, err := pb.execute()
    
    duration := time.Since(startTime)
    var memAfter runtime.MemStats
    runtime.ReadMemStats(&memAfter)
    
    result := BenchmarkResult{
        Timestamp:       startTime,
        Duration:        duration,
        OperationsPerSec: float64(operations) / duration.Seconds(),
        MemoryAllocated: memAfter.TotalAlloc - memBefore.TotalAlloc,
        GCPauses:        time.Duration(memAfter.PauseTotalNs - memBefore.PauseTotalNs),
        Success:         err == nil,
    }
    
    if err != nil {
        result.Error = err.Error()
    }
    
    pb.current = result
    pb.history = append(pb.history, result)
    
    // Keep only recent history
    if len(pb.history) > 100 {
        pb.history = pb.history[1:]
    }
    
    return result
}

func (pb *PerformanceBenchmark) CheckRegression() (bool, float64) {
    if !pb.current.Success || !pb.baseline.Success {
        return false, 0
    }
    
    currentOps := pb.current.OperationsPerSec
    baselineOps := pb.baseline.OperationsPerSec
    
    if baselineOps == 0 {
        return false, 0
    }
    
    degradation := (baselineOps - currentOps) / baselineOps
    
    return degradation > pb.threshold, degradation
}

// Built-in performance benchmarks
var DefaultBenchmarks = []*PerformanceBenchmark{
    {
        name:      "EventBusPublish",
        threshold: 0.1, // 10% degradation threshold
        execute: func() (int, error) {
            eventBus := setupTestEventBus()
            defer eventBus.Stop()
            
            events := generateTestEvents(1000)
            operations := 0
            
            for _, event := range events {
                if err := eventBus.Publish(event); err != nil {
                    return operations, err
                }
                operations++
            }
            
            return operations, nil
        },
    },
    
    {
        name:      "LogStoreAdd",
        threshold: 0.15, // 15% degradation threshold
        execute: func() (int, error) {
            logStore := setupTestLogStore()
            defer logStore.Close()
            
            operations := 0
            for i := 0; i < 1000; i++ {
                entry, err := logStore.Add(
                    fmt.Sprintf("process-%d", i%10),
                    "test-process",
                    fmt.Sprintf("Test log line %d", i),
                    false,
                )
                if err != nil {
                    return operations, err
                }
                if entry != nil {
                    operations++
                }
            }
            
            return operations, nil
        },
    },
    
    {
        name:      "ProcessManagerStartStop",
        threshold: 0.2, // 20% degradation threshold (process ops are more variable)
        execute: func() (int, error) {
            manager := setupTestProcessManager()
            defer manager.Cleanup()
            
            operations := 0
            for i := 0; i < 10; i++ {
                process, err := manager.StartCommand(
                    fmt.Sprintf("test-%d", i),
                    "echo",
                    []string{"hello"},
                )
                if err != nil {
                    return operations, err
                }
                operations++
                
                if err := manager.StopProcess(process.ID); err != nil {
                    return operations, err
                }
                operations++
            }
            
            return operations, nil
        },
    },
}
```

## Conclusion

This resource management and monitoring specification provides:

1. **Comprehensive Resource Limits**: Detailed limits for all resource types
2. **Dynamic Allocation**: Intelligent resource allocation and rebalancing
3. **Advanced Monitoring**: Real-time metrics collection and analysis
4. **Proactive Alerting**: Early warning system for resource issues
5. **Health Monitoring**: Continuous component health assessment
6. **Performance Benchmarking**: Regression detection and baseline tracking
7. **Graceful Degradation**: Automatic service level adjustment under pressure
8. **Resource Recovery**: Automatic cleanup and optimization actions

The system ensures that Brummer operates within safe resource bounds while maintaining high performance and providing excellent observability into its operation.