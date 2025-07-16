# Todo: Network & File Communication Robustness Improvements

**Generated from**: Full Planning on 2025-07-12
**Analysis Date**: 2025-07-12
**Risk Level**: HIGH | **Project Phase**: Production
**Estimated Effort**: 24-32 hours | **Files**: ~12 files
**Feature Flag Required**: YES (replacing critical communication infrastructure)

## Context & Background

**Request**: Improve robustness of file/network communication system for hours-long connections with network interruption handling
**Business Impact**: Critical for production stability - connection failures block all development workflows
**Technical Debt**: Current system vulnerable to network interruptions, file races, and lacks proper long-running connection support

### Codebase Context

**Current File-Based Discovery Issues**:
- ❌ Race conditions in `internal/discovery/instance.go:201-233` during concurrent registration
- ❌ TOCTOU vulnerabilities in file operations without proper locking
- ❌ Stale file cleanup only uses timestamps, doesn't verify process existence
- ❌ Hardcoded timeouts don't adapt to network conditions

**Current Network Connection Issues**:
- ❌ HTTP client timeout (30s) unsuitable for long operations in `internal/mcp/hub_client.go:26-29`
- ❌ Connection attempts lack context cancellation in `internal/mcp/connection_manager.go:367-417`
- ❌ No distinction between temporary vs permanent network failures
- ❌ Missing exponential backoff and circuit breaker patterns

**Missing Long-Running Connection Support**:
- ❌ No sleep/wake cycle detection and recovery
- ❌ No network interface change handling
- ❌ Context lifecycle doesn't support hours-long sessions
- ❌ Health monitoring oscillates during rapid network interruptions

### Architecture Integration

**Current Communication Flow**:
```
Hub Process → File Discovery → HTTP Connections → Instance Process Managers
     ↓              ↓                ↓                    ↓
Race Conditions  Stale Files   Connection Churn    Process Communication
```

**Target Robust Flow**:
```
Hub Process → Locked Discovery → Persistent Connections → Resilient Process Managers
     ↓              ↓                    ↓                        ↓
File Locking   Process Validation   Connection Pooling      Reliable Communication
```

## Implementation Plan

### Phase 1: File System Robustness (Risk: MEDIUM)
**Files**: `internal/discovery/instance.go`, `internal/discovery/register.go`, `internal/discovery/polling_watcher.go`
**Objective**: Eliminate race conditions and improve file-based discovery reliability
**Validation**: Concurrent instance registration works without conflicts

- [ ] **Task 1.1**: Implement atomic file operations with proper locking
  - **Risk**: MEDIUM - File locking can cause deadlocks if not implemented carefully
  - **Files**: `internal/discovery/register.go:55-139`, `internal/discovery/instance.go:201-306`
  - **Action**: Add file locking using `github.com/gofrs/flock` for atomic read-modify-write operations
  - **Success Criteria**:
    - [ ] Concurrent instance registration doesn't corrupt files
    - [ ] File operations are atomic with proper cleanup on interruption
    - [ ] Command to verify: `go test -race ./internal/discovery/... -count=100`
  - **Implementation**:
    ```go
    import "github.com/gofrs/flock"
    
    func (d *Discovery) safeUpdateInstance(instance *Instance) error {
        lockFile := filepath.Join(d.instancesDir, ".lock")
        fileLock := flock.New(lockFile)
        
        if err := fileLock.Lock(); err != nil {
            return fmt.Errorf("failed to acquire lock: %w", err)
        }
        defer fileLock.Unlock()
        
        return d.atomicWriteInstance(instance)
    }
    ```

- [ ] **Task 1.2**: Add process verification to stale instance detection
  - **Risk**: LOW - Process verification is straightforward
  - **Files**: `internal/discovery/instance.go:321-366`
  - **Action**: Verify process existence before marking instances as stale
  - **Success Criteria**:
    - [ ] Stale detection accurately identifies dead processes
    - [ ] False positives for running processes eliminated
    - [ ] Command to verify: Integration test with process killing
  - **Implementation**:
    ```go
    func (d *Discovery) isProcessRunning(pid int) bool {
        if pid <= 0 {
            return false
        }
        
        process, err := os.FindProcess(pid)
        if err != nil {
            return false
        }
        
        // Send signal 0 to check if process exists (Unix/Windows compatible)
        return process.Signal(syscall.Signal(0)) == nil
    }
    ```

- [ ] **Task 1.3**: Improve file watcher reliability and error recovery
  - **Risk**: MEDIUM - File system events can be lost during system suspension
  - **Files**: `internal/discovery/polling_watcher.go:205-275`
  - **Action**: Add fallback polling and handle file system event losses
  - **Success Criteria**:
    - [ ] File watcher recovers from system sleep/wake cycles
    - [ ] Polling fallback ensures no instances are missed
    - [ ] Large files (>1MB) are handled correctly
  - **Implementation**:
    ```go
    func (pw *PollingWatcher) startHybridWatcher() {
        // Primary: fsnotify watcher
        go pw.watchFileEvents()
        
        // Fallback: periodic polling every 30 seconds
        ticker := time.NewTicker(30 * time.Second)
        go func() {
            for range ticker.C {
                pw.fullDirectoryScan()
            }
        }()
    }
    ```

### Phase 2: Network Connection Persistence (Risk: HIGH)
**Files**: `internal/mcp/hub_client.go`, `internal/mcp/connection_manager.go`
**Objective**: Implement persistent connections with proper lifecycle management
**Validation**: Connections survive network interruptions and support hours-long sessions

- [ ] **Task 2.1**: Replace HTTP client with persistent connection management
  - **Risk**: HIGH - Fundamental change to connection architecture
  - **Files**: `internal/mcp/hub_client.go:14-199`
  - **Action**: Implement persistent HTTP/1.1 connections with keep-alive and multiplexing
  - **Success Criteria**:
    - [ ] Single connection per instance maintained for hours
    - [ ] Request multiplexing works correctly with concurrent operations
    - [ ] Connection reuse eliminates TCP handshake overhead
  - **Rollback**: Feature flag to switch back to current HTTP client
  - **Implementation**:
    ```go
    type PersistentHubClient struct {
        baseURL     string
        transport   *http.Transport
        client      *http.Client
        connMu      sync.Mutex
        established bool
    }
    
    func NewPersistentHubClient(port int) (*PersistentHubClient, error) {
        transport := &http.Transport{
            MaxIdleConns:        1,
            MaxIdleConnsPerHost: 1,
            IdleConnTimeout:     24 * time.Hour, // Long-lived connections
            KeepAlive:          30 * time.Second,
            DisableKeepAlives:  false,
            
            // Handle network interruptions
            DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                d := &net.Dialer{
                    Timeout:   10 * time.Second,
                    KeepAlive: 30 * time.Second,
                }
                return d.DialContext(ctx, network, addr)
            },
        }
        
        return &PersistentHubClient{
            baseURL: fmt.Sprintf("http://localhost:%d/mcp", port),
            transport: transport,
            client: &http.Client{
                Transport: transport,
                Timeout:   0, // No global timeout for persistent connections
            },
        }, nil
    }
    ```

- [ ] **Task 2.2**: Add exponential backoff and circuit breaker patterns
  - **Risk**: MEDIUM - Retry logic complexity
  - **Files**: `internal/mcp/connection_manager.go:367-417`, `internal/mcp/health_monitor.go:158-233`
  - **Action**: Implement smart retry with exponential backoff and circuit breaker
  - **Success Criteria**:
    - [ ] Failed connections retry with increasing delays
    - [ ] Circuit breaker prevents cascading failures
    - [ ] Recovery happens automatically when instances become healthy
  - **Implementation**:
    ```go
    type ExponentialBackoff struct {
        BaseDelay    time.Duration
        MaxDelay     time.Duration
        Multiplier   float64
        Jitter       bool
        attemptCount int
    }
    
    func (eb *ExponentialBackoff) NextDelay() time.Duration {
        delay := time.Duration(float64(eb.BaseDelay) * math.Pow(eb.Multiplier, float64(eb.attemptCount)))
        if delay > eb.MaxDelay {
            delay = eb.MaxDelay
        }
        
        if eb.Jitter {
            jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
            delay += jitter
        }
        
        eb.attemptCount++
        return delay
    }
    ```

- [ ] **Task 2.3**: Implement context-aware connection lifecycle
  - **Risk**: HIGH - Context propagation affects all operations
  - **Files**: `internal/mcp/connection_manager.go:398-405`, `internal/mcp/session_context.go:292-313`
  - **Action**: Add proper context handling for long-running operations with cancellation
  - **Success Criteria**:
    - [ ] All operations respect context cancellation
    - [ ] Long-running operations can be properly cancelled
    - [ ] Context timeouts are appropriate for operation types
  - **Implementation**:
    ```go
    func (cm *ConnectionManager) attemptConnectionWithContext(ctx context.Context, instanceID string) {
        // Create cancellable context for this connection attempt
        connCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
        
        // Use context throughout the connection process
        client, err := NewPersistentHubClientWithContext(connCtx, info.Port)
        if err != nil {
            select {
            case <-ctx.Done():
                return // Parent context cancelled
            default:
                cm.scheduleRetry(instanceID, err)
            }
            return
        }
        
        // Test connection with context
        if err := client.Initialize(connCtx); err != nil {
            cm.handleConnectionError(instanceID, err)
            return
        }
        
        cm.setActiveConnection(instanceID, client)
    }
    ```

### Phase 3: Network Interruption Handling (Risk: HIGH)
**Files**: `internal/mcp/network_monitor.go` (new), `internal/mcp/connection_manager.go`
**Objective**: Detect and handle network interruptions including sleep/wake cycles
**Validation**: Connections recover automatically after system sleep or network changes

- [ ] **Task 3.1**: Implement network state monitoring
  - **Risk**: HIGH - Platform-specific network monitoring
  - **Files**: `internal/mcp/network_monitor.go` (new), `internal/mcp/connection_manager.go:420-465`
  - **Action**: Add network connectivity monitoring with sleep/wake detection
  - **Success Criteria**:
    - [ ] System sleep/wake cycles are detected
    - [ ] Network interface changes trigger reconnection
    - [ ] Connection health reflects actual network state
  - **Implementation**:
    ```go
    package mcp
    
    import (
        "context"
        "net"
        "time"
        "github.com/pkg/errors"
    )
    
    type NetworkMonitor struct {
        connectivity chan bool
        interfaces   map[string]net.Interface
        sleepWake    chan SleepWakeEvent
    }
    
    type SleepWakeEvent struct {
        Type      string // "sleep" or "wake"
        Timestamp time.Time
    }
    
    func (nm *NetworkMonitor) Start(ctx context.Context) {
        go nm.monitorConnectivity(ctx)
        go nm.monitorInterfaces(ctx)
        go nm.monitorSleepWake(ctx) // Platform-specific implementation
    }
    
    func (nm *NetworkMonitor) monitorConnectivity(ctx context.Context) {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                // Test connectivity to known endpoint
                connected := nm.testConnectivity()
                select {
                case nm.connectivity <- connected:
                default: // Non-blocking send
                }
            }
        }
    }
    ```

- [ ] **Task 3.2**: Add connection recovery after network interruptions
  - **Risk**: MEDIUM - Recovery logic complexity
  - **Files**: `internal/mcp/connection_manager.go:420-465`, `internal/mcp/health_monitor.go:64-120`
  - **Action**: Implement automatic reconnection after detecting network recovery
  - **Success Criteria**:
    - [ ] Connections automatically recover after network interruptions
    - [ ] Recovery happens without user intervention
    - [ ] Connection state accurately reflects network status
  - **Implementation**:
    ```go
    func (cm *ConnectionManager) handleNetworkEvents() {
        for {
            select {
            case event := <-cm.networkMonitor.SleepWakeEvents():
                switch event.Type {
                case "wake":
                    // Aggressive reconnection after wake
                    cm.reconnectAllInstances(5 * time.Second)
                case "sleep":
                    // Mark all connections as potentially stale
                    cm.markConnectionsSuspect()
                }
                
            case connectivity := <-cm.networkMonitor.ConnectivityEvents():
                if connectivity {
                    // Network back online - test all connections
                    cm.validateAllConnections()
                } else {
                    // Network offline - pause health checks
                    cm.pauseHealthChecks()
                }
            }
        }
    }
    ```

### Phase 4: Enhanced Error Handling and Recovery (Risk: MEDIUM)
**Files**: `internal/mcp/errors.go` (new), `internal/mcp/hub_client.go`, `internal/mcp/health_monitor.go`
**Objective**: Implement comprehensive error classification and recovery strategies
**Validation**: Different error types are handled appropriately with proper recovery

- [ ] **Task 4.1**: Create structured error types for network operations
  - **Risk**: LOW - Error type definition is straightforward
  - **Files**: `internal/mcp/errors.go` (new), `internal/mcp/hub_client.go:148-193`
  - **Action**: Define error types to distinguish temporary vs permanent failures
  - **Success Criteria**:
    - [ ] Network errors are properly classified
    - [ ] Retry decisions based on error type
    - [ ] Error context preserved for debugging
  - **Implementation**:
    ```go
    package mcp
    
    import "time"
    
    type NetworkError struct {
        Type        ErrorType
        Underlying  error
        Temporary   bool
        RetryAfter  time.Duration
        Context     map[string]interface{}
    }
    
    type ErrorType int
    
    const (
        ErrorTypeConnRefused ErrorType = iota
        ErrorTypeTimeout
        ErrorTypeDNS
        ErrorTypeNetworkUnreachable
        ErrorTypeConnReset
        ErrorTypeProcessNotFound
    )
    
    func (ne *NetworkError) Error() string {
        return fmt.Sprintf("network error [%s]: %v", ne.Type, ne.Underlying)
    }
    
    func (ne *NetworkError) IsTemporary() bool {
        return ne.Temporary
    }
    
    func ClassifyNetworkError(err error) *NetworkError {
        // Classify standard network errors
        if netErr, ok := err.(net.Error); ok {
            if netErr.Timeout() {
                return &NetworkError{
                    Type:       ErrorTypeTimeout,
                    Underlying: err,
                    Temporary:  true,
                    RetryAfter: 5 * time.Second,
                }
            }
        }
        
        // Handle specific error patterns
        errStr := err.Error()
        switch {
        case strings.Contains(errStr, "connection refused"):
            return &NetworkError{
                Type:       ErrorTypeConnRefused,
                Underlying: err,
                Temporary:  true,
                RetryAfter: 10 * time.Second,
            }
        case strings.Contains(errStr, "connection reset"):
            return &NetworkError{
                Type:       ErrorTypeConnReset,
                Underlying: err,
                Temporary:  true,
                RetryAfter: 2 * time.Second,
            }
        default:
            return &NetworkError{
                Type:       ErrorTypeNetworkUnreachable,
                Underlying: err,
                Temporary:  false,
            }
        }
    }
    ```

- [ ] **Task 4.2**: Implement adaptive health monitoring
  - **Risk**: MEDIUM - Health check algorithm complexity
  - **Files**: `internal/mcp/health_monitor.go:158-233`
  - **Action**: Add adaptive health checks that adjust to network conditions
  - **Success Criteria**:
    - [ ] Health check frequency adapts to network conditions
    - [ ] False positives reduced during network instability
    - [ ] Health status accurately reflects instance availability
  - **Implementation**:
    ```go
    type AdaptiveHealthMonitor struct {
        baseInterval    time.Duration
        maxInterval     time.Duration
        currentInterval time.Duration
        consecutiveFails int
        networkQuality  float64 // 0.0 = poor, 1.0 = excellent
    }
    
    func (ahm *AdaptiveHealthMonitor) adjustInterval() {
        if ahm.consecutiveFails == 0 {
            // Gradually return to base interval when healthy
            ahm.currentInterval = time.Duration(float64(ahm.currentInterval) * 0.9)
            if ahm.currentInterval < ahm.baseInterval {
                ahm.currentInterval = ahm.baseInterval
            }
        } else {
            // Exponentially back off on failures
            multiplier := math.Pow(2, float64(ahm.consecutiveFails))
            ahm.currentInterval = time.Duration(float64(ahm.baseInterval) * multiplier)
            if ahm.currentInterval > ahm.maxInterval {
                ahm.currentInterval = ahm.maxInterval
            }
        }
        
        // Adjust for network quality
        ahm.currentInterval = time.Duration(float64(ahm.currentInterval) / ahm.networkQuality)
    }
    ```

### Phase 5: Resource Management and Monitoring (Risk: LOW)
**Files**: `internal/mcp/resource_manager.go` (new), `internal/mcp/metrics.go` (new)
**Objective**: Add resource limits and comprehensive monitoring
**Validation**: Resource usage is bounded and monitoring provides actionable insights

- [ ] **Task 5.1**: Implement connection resource management
  - **Risk**: LOW - Resource limiting is well-understood
  - **Files**: `internal/mcp/resource_manager.go` (new), `internal/mcp/connection_manager.go`
  - **Action**: Add connection limits and resource tracking
  - **Success Criteria**:
    - [ ] Connection count is bounded per instance
    - [ ] Memory usage is monitored and limited
    - [ ] Resource exhaustion is handled gracefully
  - **Implementation**:
    ```go
    type ResourceManager struct {
        maxConnectionsPerInstance int
        maxTotalConnections      int
        connectionSemaphore      chan struct{}
        instanceConnections      map[string]int
        mu                       sync.RWMutex
    }
    
    func (rm *ResourceManager) AcquireConnection(instanceID string) error {
        rm.mu.Lock()
        defer rm.mu.Unlock()
        
        if rm.instanceConnections[instanceID] >= rm.maxConnectionsPerInstance {
            return fmt.Errorf("too many connections to instance %s", instanceID)
        }
        
        select {
        case rm.connectionSemaphore <- struct{}{}:
            rm.instanceConnections[instanceID]++
            return nil
        default:
            return fmt.Errorf("global connection limit exceeded")
        }
    }
    ```

- [ ] **Task 5.2**: Add comprehensive monitoring and metrics
  - **Risk**: LOW - Metrics collection is non-critical
  - **Files**: `internal/mcp/metrics.go` (new)
  - **Action**: Add metrics for connection health, error rates, and performance
  - **Success Criteria**:
    - [ ] Key metrics are collected and exposed
    - [ ] Metrics help with debugging connection issues
    - [ ] Performance degradation is visible in metrics

## Gotchas & Considerations

**File System Race Conditions**:
- Windows file locking behavior differs from Unix systems
- NFS/network filesystems may have different consistency guarantees
- File watchers can miss events during high filesystem activity

**Network Edge Cases**:
- Docker networking can cause localhost connectivity issues
- VPN connections may change routing and break connections
- Mobile/laptop sleep states can suspend network interfaces

**Long-Running Connection Issues**:
- HTTP/1.1 keep-alive timeouts vary by platform and configuration
- Load balancers may terminate idle connections
- Firewall state tables may expire long-idle connections

**Cross-Platform Compatibility**:
- Process signaling differs between Windows and Unix
- Network interface enumeration is platform-specific
- Sleep/wake detection requires different APIs per platform

## Definition of Done

- [ ] All file operations are atomic with proper locking
- [ ] Network connections survive system sleep/wake cycles
- [ ] Connections persist for hours without unnecessary reconnection
- [ ] Network interruptions trigger automatic recovery
- [ ] Error types are properly classified and handled
- [ ] Resource usage is bounded and monitored
- [ ] Tests pass: `make test && make integration-test`
- [ ] Manual testing with network disconnection scenarios
- [ ] System sleep/wake testing on target platforms
- [ ] Performance benchmarking shows improved stability
- [ ] Production deployment successful with monitoring

## Validation Commands

```bash
# Test file discovery race conditions
go test -race ./internal/discovery/... -count=50

# Test network connection robustness
go test ./internal/mcp/... -tags=integration

# Manual network interruption test
sudo iptables -A OUTPUT -p tcp --dport 7777-7999 -j DROP
sleep 30
sudo iptables -D OUTPUT -p tcp --dport 7777-7999 -j DROP

# System sleep simulation (where supported)
systemctl suspend  # Test reconnection after wake

# Resource limit testing
BRUMMER_MAX_CONNECTIONS=5 brum --test-resource-limits
```

## Execution Notes

- **Start with**: Task 1.1 (File locking) - Foundation for all file operations
- **Validation**: Run integration tests after each phase
- **Commit pattern**: `network: [action taken]` or `discovery: [action taken]`
- **Feature Flag**: `BRUMMER_USE_ROBUST_NETWORKING=true` for gradual rollout
- **Monitoring**: Track connection success rates and recovery times during rollout

## Risk Communication

⚠️ **HIGH RISK ITEMS REQUIRING APPROVAL**:
- Network connection architecture changes (affects all hub-instance communication)
- Context lifecycle modifications (impacts all operations)
- File locking introduction (potential for deadlocks)

✅ **MITIGATION**: 
- Feature flags allow instant rollback to current implementation
- Comprehensive testing plan includes failure scenarios
- Gradual rollout with monitoring ensures early issue detection

The robustness improvements are essential for production stability but require careful implementation to avoid introducing new failure modes.