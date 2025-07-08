# Concurrency Configuration Schema for Brummer

## Overview

This document defines the comprehensive configuration schema for all concurrency-related settings in Brummer. The configuration enables fine-tuning of synchronization behavior, resource limits, and performance characteristics across all components.

## Configuration Architecture

### 1. Configuration Hierarchy

```
Command Line Args → Environment Variables → Project Config → Parent Configs → Home Config → Defaults
```

### 2. Configuration Sources

- **Command Line**: `--config-option value` or `--config-file path`
- **Environment**: `BRUMMER_COMPONENT_SETTING=value`
- **Project Files**: `.brum.toml` in current and parent directories
- **User Config**: `~/.brum.toml`
- **Built-in Defaults**: Hardcoded sensible defaults

## Complete Configuration Schema

### Main Configuration Structure

```toml
# Brummer Concurrency Configuration
# This file controls all concurrency, synchronization, and resource management settings

[meta]
version = "1.0"
generated_by = "brummer --export-config"
generated_at = "2024-01-15T10:30:00Z"

# Global resource limits that apply across all components
[resources]
# Maximum total goroutines across all components
max_goroutines = 64  # CPU cores * 8, min 32, max 512

# Memory limit in MB for the entire application
memory_limit_mb = 512  # Default: 512MB

# File descriptor limit
max_file_descriptors = 1000  # Default: 1000

# Network connection limit
max_network_connections = 100  # Default: 100

# Resource monitoring interval
monitoring_interval_ms = 10000  # Default: 10 seconds

# Enable resource usage warnings
enable_resource_warnings = true

# Enable resource usage enforcement (vs warnings only)
enforce_resource_limits = true

# Graceful degradation thresholds (percentage of limits)
warning_threshold_percent = 80   # Warn at 80% of limit
critical_threshold_percent = 95  # Critical at 95% of limit

[eventbus]
# Worker pool configuration
worker_pool_size = "auto"  # "auto", or specific number (4-32)

# Queue sizes for different priority levels [high, medium, low]
queue_sizes = [100, 500, 1000]

# Backpressure handling strategy
backpressure_strategy = "drop_oldest"  # "drop_oldest", "drop_newest", "drop_low_priority", "block"

# Maximum time to wait when backpressure_strategy = "block"
block_timeout_ms = 100  # Default: 100ms

# Enable priority-based event processing
enable_priority_queues = true

# Event processing timeout per handler
handler_timeout_ms = 5000  # Default: 5 seconds

# Enable event handler panic recovery
enable_panic_recovery = true

# Enable event bus performance monitoring
enable_monitoring = true

# Worker idle timeout before shutdown during low load
worker_idle_timeout_ms = 30000  # Default: 30 seconds

# Enable event ordering guarantees within event types
maintain_event_ordering = true

# Maximum events in memory before applying backpressure
max_pending_events = 10000

# Enable event persistence during high load
enable_event_persistence = false

# Event persistence directory (if enabled)
persistence_dir = "/tmp/brummer-events"

[process_manager]
# Maximum number of processes that can be started concurrently
max_concurrent_starts = 10

# Strategy for reading process status
status_read_strategy = "atomic"  # "atomic", "locked"

# Timeout for process startup operations
startup_timeout_ms = 30000  # Default: 30 seconds

# Timeout for process shutdown operations
shutdown_timeout_ms = 10000  # Default: 10 seconds

# Enable aggressive process cleanup
enable_aggressive_cleanup = true

# Process cleanup retry attempts
cleanup_retry_attempts = 3

# Delay between cleanup retry attempts
cleanup_retry_delay_ms = 1000  # Default: 1 second

# Enable process status caching
enable_status_caching = true

# Process status cache TTL
status_cache_ttl_ms = 1000  # Default: 1 second

# Maximum number of log callbacks per process
max_callbacks_per_process = 10

# Log callback invocation timeout
callback_timeout_ms = 1000  # Default: 1 second

# Enable callback error isolation
enable_callback_isolation = true

# Process monitoring interval
monitoring_interval_ms = 5000  # Default: 5 seconds

# Enable automatic process restart on failure
enable_auto_restart = false

# Maximum automatic restart attempts
max_restart_attempts = 3

# Delay between restart attempts
restart_delay_ms = 5000  # Default: 5 seconds

[log_store]
# Maximum number of log entries to store in memory
max_entries = 10000

# Batch size for log processing
batch_size = 100

# Buffer size for incoming log requests
buffer_size = 1000

# Number of worker goroutines for log processing
worker_count = 1  # Single worker for ordering guarantees

# Timeout for log operations
operation_timeout_ms = 50  # Default: 50ms

# Enable log batching for performance
enable_batching = true

# Batch flush interval
batch_flush_interval_ms = 100  # Default: 100ms

# Maximum log line length (longer lines truncated)
max_line_length = 2048

# Enable URL detection in log content
enable_url_detection = true

# Maximum number of URLs to track
max_tracked_urls = 100

# Enable log level auto-detection
enable_level_detection = true

# Enable log tagging
enable_tagging = true

# Maximum number of error entries to keep
max_error_entries = 500

# Error context window size (lines before/after error)
error_context_lines = 5

# Enable log compression for storage
enable_compression = false

# Log rotation settings
enable_rotation = true
max_log_files = 5
max_log_file_size_mb = 10

# Enable log persistence to disk
enable_persistence = false
persistence_directory = "/tmp/brummer-logs"

[proxy_server]
# Buffer size for request history
request_buffer_size = 1000

# Connection timeout for upstream servers
connection_timeout_ms = 5000  # Default: 5 seconds

# Read timeout for HTTP operations
read_timeout_ms = 10000  # Default: 10 seconds

# Write timeout for HTTP operations  
write_timeout_ms = 10000  # Default: 10 seconds

# Idle timeout for connections
idle_timeout_ms = 60000  # Default: 60 seconds

# Maximum number of WebSocket clients
max_ws_clients = 50

# WebSocket message buffer size
ws_buffer_size = 256

# WebSocket ping interval
ws_ping_interval_ms = 30000  # Default: 30 seconds

# WebSocket connection timeout
ws_connection_timeout_ms = 10000  # Default: 10 seconds

# Enable request/response compression
enable_compression = true

# Maximum request body size in MB
max_request_body_mb = 10

# Maximum response body size in MB for processing
max_response_body_mb = 10

# Enable telemetry injection
enable_telemetry = true

# Telemetry batch size
telemetry_batch_size = 50

# Telemetry flush interval
telemetry_flush_interval_ms = 1000  # Default: 1 second

# Enable URL rewriting in responses
enable_url_rewriting = true

# Enable proxy request logging
enable_request_logging = true

# Request log level
request_log_level = "info"  # "debug", "info", "warn", "error"

# Enable reverse proxy mode optimizations
enable_reverse_proxy_optimizations = true

# Reverse proxy port allocation range
reverse_proxy_port_start = 20888
reverse_proxy_port_end = 20999

[mcp_manager]
# Health check interval for instance connections
health_check_interval_ms = 30000  # Default: 30 seconds

# Session timeout for idle sessions
session_timeout_ms = 300000  # Default: 5 minutes

# Maximum retry attempts for failed connections
max_retry_attempts = 3

# Base delay for exponential backoff retry
retry_base_delay_ms = 1000  # Default: 1 second

# Maximum delay for exponential backoff retry
retry_max_delay_ms = 30000  # Default: 30 seconds

# Connection establishment timeout
connection_timeout_ms = 10000  # Default: 10 seconds

# MCP tool call timeout
tool_call_timeout_ms = 30000  # Default: 30 seconds

# Enable connection pooling
enable_connection_pooling = true

# Maximum connections per instance
max_connections_per_instance = 5

# Connection pool idle timeout
connection_pool_idle_timeout_ms = 60000  # Default: 60 seconds

# Enable automatic instance discovery
enable_auto_discovery = true

# Instance discovery interval
discovery_interval_ms = 5000  # Default: 5 seconds

# Instance file cleanup interval
cleanup_interval_ms = 60000  # Default: 60 seconds

# Stale instance timeout
stale_instance_timeout_ms = 300000  # Default: 5 minutes

# Enable instance health monitoring
enable_health_monitoring = true

# Instance response timeout for health checks
health_check_timeout_ms = 5000  # Default: 5 seconds

[tui_model]
# Update operation timeout
update_timeout_ms = 100  # Default: 100ms for 60 FPS

# View rendering timeout
render_timeout_ms = 50   # Default: 50ms for smooth UI

# Data refresh interval
refresh_interval_ms = 1000  # Default: 1 second

# Maximum items per view (for performance)
max_items_per_view = 1000

# Enable concurrent view updates
enable_concurrent_updates = true

# View update batch size
update_batch_size = 10

# Enable view caching
enable_view_caching = true

# View cache TTL
view_cache_ttl_ms = 500  # Default: 500ms

# Enable smooth scrolling
enable_smooth_scrolling = true

# Scroll animation duration
scroll_animation_ms = 200  # Default: 200ms

# Enable keyboard input buffering
enable_input_buffering = true

# Input buffer size
input_buffer_size = 100

# Input processing timeout
input_timeout_ms = 50  # Default: 50ms

# Enable view state persistence
enable_state_persistence = false

# State persistence file
state_persistence_file = "~/.brummer-state.json"

[performance]
# Enable performance monitoring across all components
enable_monitoring = true

# Performance metrics collection interval
metrics_interval_ms = 5000  # Default: 5 seconds

# Enable performance alerts
enable_alerts = true

# Performance degradation threshold (percentage)
degradation_threshold_percent = 10  # Alert if >10% slower

# Enable automatic performance tuning
enable_auto_tuning = false

# Performance history size (number of samples)
history_size = 1000

# Enable performance profiling
enable_profiling = false

# Profiling data directory
profiling_dir = "/tmp/brummer-profiles"

# CPU profiling interval
cpu_profile_interval_ms = 30000  # Default: 30 seconds

# Memory profiling interval
memory_profile_interval_ms = 60000  # Default: 60 seconds

# Enable lock contention tracking
enable_lock_tracking = true

# Lock contention warning threshold
lock_contention_threshold_ms = 10  # Warn if lock held >10ms

# Enable goroutine leak detection
enable_goroutine_leak_detection = true

# Goroutine leak threshold
goroutine_leak_threshold = 50  # Alert if >50 unexpected goroutines

[debugging]
# Enable race condition detection
enable_race_detection = false  # Expensive, use only during development

# Enable deadlock detection
enable_deadlock_detection = true

# Deadlock detection timeout
deadlock_timeout_ms = 30000  # Default: 30 seconds

# Enable debug logging for synchronization
enable_sync_debug_logging = false

# Debug log level
debug_log_level = "warn"  # "debug", "info", "warn", "error"

# Enable stack trace collection on errors
enable_stack_traces = true

# Stack trace depth
stack_trace_depth = 20

# Enable memory debugging
enable_memory_debugging = false

# Memory debugging interval
memory_debug_interval_ms = 10000  # Default: 10 seconds

# Enable channel debugging
enable_channel_debugging = false

# Channel operation timeout for debugging
channel_debug_timeout_ms = 1000  # Default: 1 second

# Enable mutex debugging
enable_mutex_debugging = false

# Mutex operation timeout for debugging
mutex_debug_timeout_ms = 100  # Default: 100ms

[fallback]
# Enable fallback mechanisms for compatibility
enable_fallback_mechanisms = true

# Fallback to legacy implementation on errors
enable_legacy_fallback = false

# Maximum fallback attempts
max_fallback_attempts = 3

# Fallback timeout
fallback_timeout_ms = 5000  # Default: 5 seconds

# Enable graceful degradation
enable_graceful_degradation = true

# Degraded mode timeout
degraded_mode_timeout_ms = 60000  # Default: 60 seconds

# Enable circuit breaker pattern
enable_circuit_breaker = true

# Circuit breaker failure threshold
circuit_breaker_failure_threshold = 5

# Circuit breaker recovery timeout
circuit_breaker_recovery_timeout_ms = 30000  # Default: 30 seconds

# Enable bulkhead isolation
enable_bulkhead_isolation = true

# Critical operations priority boost
critical_priority_boost = 10
```

## Environment Variable Mapping

All configuration options can be overridden via environment variables using the pattern:
`BRUMMER_<SECTION>_<SETTING>=value`

### Examples

```bash
# EventBus configuration
export BRUMMER_EVENTBUS_WORKER_POOL_SIZE=8
export BRUMMER_EVENTBUS_BACKPRESSURE_STRATEGY=block

# Process Manager configuration  
export BRUMMER_PROCESS_MANAGER_MAX_CONCURRENT_STARTS=15
export BRUMMER_PROCESS_MANAGER_STARTUP_TIMEOUT_MS=45000

# Global resource limits
export BRUMMER_RESOURCES_MAX_GOROUTINES=128
export BRUMMER_RESOURCES_MEMORY_LIMIT_MB=1024

# Performance monitoring
export BRUMMER_PERFORMANCE_ENABLE_MONITORING=true
export BRUMMER_PERFORMANCE_ENABLE_ALERTS=true

# Debugging (for development)
export BRUMMER_DEBUGGING_ENABLE_RACE_DETECTION=true
export BRUMMER_DEBUGGING_ENABLE_SYNC_DEBUG_LOGGING=true
```

## Configuration Validation

### Validation Rules

```go
type ConfigValidator struct {
    rules []ValidationRule
}

type ValidationRule interface {
    Validate(config ConcurrencyConfig) []ValidationError
    Category() string
    Severity() ErrorSeverity
}

type ValidationError struct {
    Field    string
    Value    interface{}
    Message  string
    Severity ErrorSeverity
}

type ErrorSeverity int
const (
    SeverityInfo ErrorSeverity = iota
    SeverityWarning
    SeverityError
    SeverityCritical
)
```

### Built-in Validation Rules

```go
var DefaultValidationRules = []ValidationRule{
    // Resource limits validation
    &RangeValidationRule{
        Field: "resources.max_goroutines",
        Min:   16,
        Max:   1024,
        Severity: SeverityError,
    },
    
    // EventBus validation
    &EnumValidationRule{
        Field: "eventbus.backpressure_strategy",
        Values: []string{"drop_oldest", "drop_newest", "drop_low_priority", "block"},
        Severity: SeverityError,
    },
    
    // Timeout validation
    &TimeoutValidationRule{
        Field: "process_manager.startup_timeout_ms",
        Min:   time.Second,
        Max:   5 * time.Minute,
        Severity: SeverityWarning,
    },
    
    // Consistency validation
    &ConsistencyValidationRule{
        Description: "EventBus queue sizes should be increasing",
        Validator: func(c ConcurrencyConfig) bool {
            sizes := c.EventBus.QueueSizes
            return sizes[0] <= sizes[1] && sizes[1] <= sizes[2]
        },
        Severity: SeverityWarning,
    },
}
```

## Configuration Templates

### Development Template

```toml
# Development configuration - optimized for debugging and responsiveness
# Use: brummer --config dev

[resources]
max_goroutines = 32
memory_limit_mb = 256
enable_resource_warnings = true
enforce_resource_limits = false

[eventbus]
worker_pool_size = 4
queue_sizes = [50, 100, 200]
enable_monitoring = true
enable_panic_recovery = true

[process_manager]
max_concurrent_starts = 5
enable_status_caching = false
monitoring_interval_ms = 1000

[log_store]
max_entries = 5000
enable_persistence = true
enable_rotation = false

[debugging]
enable_race_detection = true
enable_deadlock_detection = true
enable_sync_debug_logging = true
enable_stack_traces = true

[performance]
enable_monitoring = true
enable_alerts = true
enable_profiling = true
```

### Production Template

```toml
# Production configuration - optimized for performance and stability
# Use: brummer --config production

[resources]
max_goroutines = 128
memory_limit_mb = 1024
enable_resource_warnings = true
enforce_resource_limits = true

[eventbus]
worker_pool_size = "auto"
queue_sizes = [200, 1000, 2000]
backpressure_strategy = "drop_oldest"
enable_monitoring = true

[process_manager]
max_concurrent_starts = 20
enable_aggressive_cleanup = true
enable_auto_restart = true
max_restart_attempts = 3

[log_store]
max_entries = 20000
enable_batching = true
enable_compression = true
enable_persistence = true
enable_rotation = true

[proxy_server]
max_ws_clients = 100
enable_compression = true
enable_reverse_proxy_optimizations = true

[performance]
enable_monitoring = true
enable_alerts = true
enable_auto_tuning = true

[fallback]
enable_graceful_degradation = true
enable_circuit_breaker = true
enable_bulkhead_isolation = true
```

### Testing Template

```toml
# Testing configuration - optimized for test execution and determinism
# Use: brummer --config testing

[resources]
max_goroutines = 16
memory_limit_mb = 128
monitoring_interval_ms = 1000

[eventbus]
worker_pool_size = 2
queue_sizes = [10, 20, 50]
enable_monitoring = false
handler_timeout_ms = 1000

[process_manager]
max_concurrent_starts = 3
startup_timeout_ms = 10000
shutdown_timeout_ms = 5000

[log_store]
max_entries = 1000
worker_count = 1
enable_persistence = false

[tui_model]
enable_concurrent_updates = false
enable_view_caching = false

[debugging]
enable_deadlock_detection = true
enable_goroutine_leak_detection = true

[fallback]
enable_fallback_mechanisms = false
enable_legacy_fallback = false
```

## Configuration Loading Implementation

### Configuration Loader

```go
type ConfigLoader struct {
    sources []ConfigSource
    validator *ConfigValidator
    defaults ConcurrencyConfig
}

type ConfigSource interface {
    Load() (ConcurrencyConfig, error)
    Priority() int
    Name() string
}

// Load configuration with override chain
func (cl *ConfigLoader) Load() (*ConcurrencyConfig, error) {
    // Start with defaults
    config := cl.defaults
    
    // Apply sources in priority order
    sort.Slice(cl.sources, func(i, j int) bool {
        return cl.sources[i].Priority() < cl.sources[j].Priority()
    })
    
    for _, source := range cl.sources {
        sourceConfig, err := source.Load()
        if err == nil {
            config = cl.mergeConfigs(config, sourceConfig)
        }
    }
    
    // Validate final configuration
    if errors := cl.validator.Validate(config); len(errors) > 0 {
        return nil, &ConfigValidationError{Errors: errors}
    }
    
    return &config, nil
}
```

### Environment Variable Source

```go
type EnvironmentSource struct{}

func (es *EnvironmentSource) Load() (ConcurrencyConfig, error) {
    config := ConcurrencyConfig{}
    
    // Map environment variables to config fields
    mapping := map[string]func(string) error{
        "BRUMMER_EVENTBUS_WORKER_POOL_SIZE": func(v string) error {
            if v == "auto" {
                config.EventBus.WorkerPoolSize = "auto"
                return nil
            }
            val, err := strconv.Atoi(v)
            if err != nil {
                return err
            }
            config.EventBus.WorkerPoolSize = val
            return nil
        },
        
        "BRUMMER_RESOURCES_MAX_GOROUTINES": func(v string) error {
            val, err := strconv.Atoi(v)
            if err != nil {
                return err
            }
            config.Resources.MaxGoroutines = val
            return nil
        },
        
        // ... additional mappings
    }
    
    for envVar, setter := range mapping {
        if value := os.Getenv(envVar); value != "" {
            if err := setter(value); err != nil {
                return config, fmt.Errorf("invalid value for %s: %v", envVar, err)
            }
        }
    }
    
    return config, nil
}
```

## Configuration Hot Reload

### Hot Reload Implementation

```go
type ConfigHotReload struct {
    configPath string
    lastMod    time.Time
    loader     *ConfigLoader
    listeners  []ConfigChangeListener
    mu         sync.RWMutex
}

type ConfigChangeListener interface {
    OnConfigChange(oldConfig, newConfig ConcurrencyConfig) error
}

func (chr *ConfigHotReload) StartWatching(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for range ticker.C {
        if chr.configChanged() {
            if err := chr.reloadConfig(); err != nil {
                log.Printf("Config reload failed: %v", err)
            }
        }
    }
}

func (chr *ConfigHotReload) reloadConfig() error {
    newConfig, err := chr.loader.Load()
    if err != nil {
        return err
    }
    
    chr.mu.RLock()
    oldConfig := chr.currentConfig
    chr.mu.RUnlock()
    
    // Notify listeners
    for _, listener := range chr.listeners {
        if err := listener.OnConfigChange(oldConfig, *newConfig); err != nil {
            log.Printf("Config change listener failed: %v", err)
        }
    }
    
    chr.mu.Lock()
    chr.currentConfig = *newConfig
    chr.mu.Unlock()
    
    return nil
}
```

## Configuration Export and Import

### Export Current Configuration

```bash
# Export current effective configuration
brummer --export-config > current-config.toml

# Export configuration template
brummer --export-template production > production-config.toml

# Export with comments and documentation
brummer --export-config --with-docs > documented-config.toml
```

### Import and Validate Configuration

```bash
# Validate configuration file
brummer --validate-config config.toml

# Test configuration without applying
brummer --test-config config.toml

# Apply configuration with dry-run
brummer --config config.toml --dry-run

# Show configuration diff
brummer --config config.toml --show-diff
```

## Conclusion

This configuration schema provides:

1. **Comprehensive Coverage**: All concurrency aspects configurable
2. **Environment Flexibility**: Support for different deployment scenarios
3. **Validation**: Built-in validation prevents invalid configurations
4. **Templates**: Pre-configured templates for common use cases
5. **Hot Reload**: Dynamic configuration updates without restart
6. **Documentation**: Self-documenting configuration with inline help
7. **Debugging**: Extensive debugging and monitoring options
8. **Fallback**: Graceful degradation and circuit breaker patterns

The schema enables fine-tuning of all synchronization and performance characteristics while maintaining safe defaults and preventing invalid configurations.