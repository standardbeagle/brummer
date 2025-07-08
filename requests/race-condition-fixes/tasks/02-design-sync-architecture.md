# Task: Design Synchronized Architecture

## Persona: System Architect
Role: Concurrency Architecture Designer
Expertise: Distributed systems, Go concurrency patterns, system design

## Current State Assessment
**Before starting this task:**
```yaml
existing_state:
  - ✅ Research analysis completed with specific recommendations
  - ✅ Worker pool sizing determined (CPU cores * 2.5)
  - ✅ Lock-free alternatives identified for key components
  - ✅ Performance benchmark framework defined
  - ❌ Architecture design for synchronized components not created
  - ❌ Integration patterns between components not defined
  - ❌ Resource limits and configurations not specified
  - ❓ Unknown: Specific interface contracts for thread-safe operations

current_files:
  - requests/race-condition-fixes/research/: Complete research foundation
  - pkg/events/events.go: Current EventBus implementation
  - internal/tui/model.go: Current TUI Model structure
  - internal/process/manager.go: Current Process Manager
  - internal/logs/store.go: Current Log Store
  - internal/proxy/server.go: Current Proxy Server
```

## File Scope Definition
**Explicit file list for this task:**
```yaml
read_files:
  - requests/race-condition-fixes/research/concurrency-best-practices.md
  - requests/race-condition-fixes/research/worker-pool-sizing.md
  - requests/race-condition-fixes/research/lock-free-alternatives.md
  - pkg/events/events.go                    # EventBus current implementation
  - internal/tui/model.go                   # TUI Model current structure
  - internal/process/manager.go             # Process Manager current structure
  - internal/logs/store.go                  # Log Store current structure
  - internal/proxy/server.go                # Proxy Server current structure
  - internal/mcp/connection_manager.go      # MCP Manager current structure
  - internal/config/config.go               # Configuration structure

create_files:
  - requests/race-condition-fixes/design/synchronized-architecture.md
  - requests/race-condition-fixes/design/interface-contracts.md
  - requests/race-condition-fixes/design/configuration-schema.md
  - requests/race-condition-fixes/design/component-interaction-flows.md
  - requests/race-condition-fixes/design/resource-limits.md

# Total: 15 files (within limits)
```

## Task Requirements
- **Objective**: Design comprehensive synchronized architecture for race-condition-free Brummer
- **Risk Level**: MEDIUM (Architecture changes affect all components)
- **Dependencies**: 01-research-race-analysis (completed)
- **Deliverables**: 
  - Complete synchronized architecture design
  - Thread-safe interface contracts
  - Configuration schema for concurrency settings
  - Component interaction flow diagrams
  - Resource limits and monitoring specifications

## Success Criteria Checklist
- [ ] Design EventBus worker pool architecture with configurable sizing
- [ ] Specify TUI Model pointer-receiver patterns and synchronization
- [ ] Define Process Manager thread-safe operations and lock hierarchy
- [ ] Design Log Store consistency patterns without async/sync conflicts
- [ ] Specify Proxy Server single-mutex architecture
- [ ] Create MCP Connection Manager channel-based design
- [ ] Define configuration schema for all concurrency parameters
- [ ] Document component interaction flows and synchronization points
- [ ] Specify resource limits and monitoring requirements

## Risk Mitigation
- **Architecture Validation**: Design must maintain existing API compatibility
- **Performance Requirements**: No more than 10% performance degradation
- **Progressive Implementation**: Design enables incremental rollout
- **Rollback Capability**: Each component can be reverted independently

## Success Validation
```bash
# Verify design documents created
ls -la requests/race-condition-fixes/design/
# Expected: 5 design documents

# Check design completeness
wc -l requests/race-condition-fixes/design/*.md
# Expected: Comprehensive design documentation (>200 lines each)

# Validate against current interfaces
grep -r "type.*interface" --include="*.go" internal/
# Verify interface compatibility maintained
```

## Design Requirements

### 1. EventBus Worker Pool Architecture
- **Pool Sizing**: Configurable with default CPU cores * 2.5
- **Backpressure Handling**: Graceful degradation when pool full
- **Goroutine Management**: Bounded pool with proper cleanup
- **Event Ordering**: Maintain event ordering guarantees where needed
- **Error Handling**: Isolated handler failures don't affect pool

### 2. TUI Model Synchronization Design
- **Pointer Receivers**: All 60+ methods converted to pointer receivers
- **State Protection**: RWMutex for concurrent read/write operations
- **Channel Operations**: Thread-safe channel access patterns
- **Update Coordination**: Atomic updates for UI state changes
- **Performance**: Minimize lock contention during rendering

### 3. Process Manager Thread-Safety
- **Map Operations**: Consistent RWMutex usage for process map
- **Process Lifecycle**: Atomic state transitions
- **Lock Hierarchy**: Clear lock ordering to prevent deadlocks
- **Resource Cleanup**: Proper goroutine cleanup on process termination
- **Status Queries**: Lock-free status reads using atomic operations

### 4. Log Store Consistency Architecture
- **Single Path**: Eliminate async/sync operation conflicts
- **Channel-Based**: All operations through single channel
- **Timeout Handling**: Graceful timeout without sync fallback
- **Batch Processing**: Efficient batching for high-throughput scenarios
- **Memory Management**: Bounded memory usage with rotation

### 5. Proxy Server Synchronization
- **Single Mutex**: Consolidate multiple mutexes into one
- **Lock Hierarchy**: Clear lock ordering for complex operations
- **WebSocket Management**: Thread-safe client connection handling
- **URL Mapping**: Atomic updates for reverse proxy mappings
- **Request Tracking**: Lock-free request counting and metrics

### 6. MCP Connection Manager Design
- **Channel-Based State**: All state changes through channels
- **Session Management**: Thread-safe session routing
- **Health Monitoring**: Non-blocking health check operations
- **Connection Pooling**: Efficient connection reuse patterns
- **Error Recovery**: Graceful recovery from connection failures

## Configuration Architecture

### Concurrency Configuration Schema
```yaml
concurrency:
  eventbus:
    worker_pool_size: auto  # CPU cores * 2.5, or explicit number
    queue_buffer_size: 1000
    backpressure_strategy: "degrade"  # degrade, block, drop
    
  process_manager:
    max_concurrent_starts: 10
    status_read_strategy: "atomic"  # atomic, locked
    
  log_store:
    buffer_size: 10000
    batch_size: 100
    timeout_ms: 50
    
  proxy_server:
    connection_timeout_ms: 5000
    request_buffer_size: 1000
    
  mcp_manager:
    health_check_interval_ms: 30000
    session_timeout_ms: 300000
```

## Component Integration Patterns

### Thread-Safe Interface Contracts
- All public methods must be thread-safe
- Interfaces define concurrency guarantees
- Error handling includes timeout and cancellation
- Resource cleanup is automatic and reliable

### Synchronization Boundaries
- Clear ownership of data structures
- Explicit handoff points between components
- Immutable data sharing where possible
- Copy-on-write for read-heavy scenarios

### Event Flow Architecture
- EventBus as central coordination point
- Components subscribe to relevant events
- Event ordering preserved within topics
- Error isolation prevents cascade failures

## Resource Limits and Monitoring

### Goroutine Management
- Bounded pools for all async operations
- Automatic cleanup on component shutdown
- Monitoring for goroutine leaks
- Alerts for abnormal goroutine counts

### Memory Management
- Bounded buffers and queues
- Automatic garbage collection assistance
- Memory usage monitoring and alerts
- Configurable limits for all components

### Performance Monitoring
- Latency tracking for critical operations
- Throughput monitoring for high-volume operations
- Lock contention metrics
- Resource utilization dashboards

## Implementation Phases

### Phase 1: Foundation (TUI Model + EventBus)
- Most critical components first
- Establishes synchronization patterns
- Enables other components to build on solid foundation

### Phase 2: Services (Process Manager + Log Store)
- Core service reliability
- Data consistency guarantees
- Performance optimization

### Phase 3: Network (Proxy + MCP)
- External interface reliability
- Connection management
- Error recovery patterns

## Context from PRD
This design must enable the implementation plan outlined in the PRD:
- Critical fixes completed within 48 hours
- High priority fixes within 1 week
- Performance requirements met (< 10% degradation)
- Backward compatibility maintained

## Constraints
- **Time**: 4 hours maximum
- **Compatibility**: Must maintain existing API contracts
- **Performance**: Design for < 10% performance impact
- **Complexity**: Balance robustness with maintainability
- **Token Budget**: Focus on architecture, not implementation details

## Execution Checklist
- [ ] Analyze current architecture and identify synchronization points
- [ ] Design EventBus worker pool with configurable parameters
- [ ] Specify TUI Model synchronization patterns
- [ ] Define Process Manager thread-safe operations
- [ ] Design Log Store consistency architecture
- [ ] Specify Proxy Server mutex consolidation
- [ ] Create MCP Manager channel-based design
- [ ] Document configuration schema for all parameters
- [ ] Create component interaction flow diagrams
- [ ] Specify resource limits and monitoring requirements
- [ ] Validate design against performance and compatibility requirements