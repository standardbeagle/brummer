# Implementation Step 5: Health Monitoring

## Overview

This step implements health monitoring using the MCP ping/pong protocol. The hub sends periodic pings to connected instances and tracks their responsiveness, marking unresponsive instances for retry or removal.

## Goals

1. Implement MCP ping/pong protocol
2. Send pings every 5 seconds to active connections
3. Mark instances as retrying after 3 missed pings (20 seconds)
4. Retry connections with exponential backoff
5. Clean up dead connections after max retries

## Technical Design

### Health Check Flow

```
Hub                     Instance
 │                         │
 ├──ping (every 5s)──────>│
 │<──────pong─────────────┤
 │                         │
 ├──ping──────────────────>│
 │     (no response)       │
 │     (wait 5s)           │
 ├──ping──────────────────>│
 │     (no response)       │
 │     (wait 5s)           │
 ├──ping──────────────────>│
 │     (no response)       │
 │                         │
 └─> Mark as RETRYING      │
     Start reconnection    │
```

### Timing Configuration

```go
const (
    // Ping interval - how often to send pings
    PingInterval = 5 * time.Second
    
    // Ping timeout - how long to wait for pong
    PingTimeout = 2 * time.Second
    
    // Max missed pings before marking unhealthy
    MaxMissedPings = 3
    
    // Retry backoff intervals
    RetryBackoff = []time.Duration{
        200 * time.Millisecond,
        400 * time.Millisecond,
        800 * time.Millisecond,
    }
)
```

## Implementation

### 1. Health Monitor Component

```go
// internal/mcp/health_monitor.go
package mcp

import (
    "context"
    "log"
    "sync"
    "time"
)

type HealthMonitor struct {
    connMgr *ConnectionManager
    
    // Ping tracking
    pingTrackers map[string]*PingTracker // instanceID -> tracker
    mu           sync.RWMutex
    
    // Control
    stopCh chan struct{}
    wg     sync.WaitGroup
}

type PingTracker struct {
    instanceID   string
    lastPing     time.Time
    lastPong     time.Time
    missedPings  int
    pingInFlight bool
}

func NewHealthMonitor(connMgr *ConnectionManager) *HealthMonitor {
    return &HealthMonitor{
        connMgr:      connMgr,
        pingTrackers: make(map[string]*PingTracker),
        stopCh:       make(chan struct{}),
    }
}

func (hm *HealthMonitor) Start() {
    hm.wg.Add(1)
    go hm.monitorLoop()
}

func (hm *HealthMonitor) Stop() {
    close(hm.stopCh)
    hm.wg.Wait()
}

func (hm *HealthMonitor) monitorLoop() {
    defer hm.wg.Done()
    
    ticker := time.NewTicker(PingInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            hm.sendPings()
            
        case <-hm.stopCh:
            return
        }
    }
}

func (hm *HealthMonitor) sendPings() {
    // Get active connections
    instances := hm.connMgr.ListInstances()
    
    for _, info := range instances {
        if info.State != StateActive {
            continue
        }
        
        // Get or create tracker
        hm.mu.Lock()
        tracker, exists := hm.pingTrackers[info.InstanceID]
        if !exists {
            tracker = &PingTracker{
                instanceID: info.InstanceID,
                lastPong:   time.Now(), // Assume healthy initially
            }
            hm.pingTrackers[info.InstanceID] = tracker
        }
        hm.mu.Unlock()
        
        // Check if previous ping is still in flight
        if tracker.pingInFlight {
            tracker.missedPings++
            log.Printf("Instance %s: ping timeout (missed: %d)", info.InstanceID, tracker.missedPings)
            
            if tracker.missedPings >= MaxMissedPings {
                log.Printf("Instance %s: marking as unhealthy after %d missed pings", 
                    info.InstanceID, tracker.missedPings)
                hm.connMgr.UpdateState(info.InstanceID, StateRetrying)
                tracker.missedPings = 0
                tracker.pingInFlight = false
            }
            continue
        }
        
        // Send ping asynchronously
        hm.wg.Add(1)
        go hm.sendPing(info, tracker)
    }
    
    // Clean up trackers for dead instances
    hm.cleanupTrackers(instances)
}

func (hm *HealthMonitor) sendPing(info *ConnectionInfo, tracker *PingTracker) {
    defer hm.wg.Done()
    
    // Get client for this session
    // Note: We need to find a session for this instance
    client := hm.getClientForInstance(info.InstanceID)
    if client == nil {
        return
    }
    
    // Mark ping in flight
    hm.mu.Lock()
    tracker.pingInFlight = true
    tracker.lastPing = time.Now()
    hm.mu.Unlock()
    
    // Send ping with timeout
    ctx, cancel := context.WithTimeout(context.Background(), PingTimeout)
    defer cancel()
    
    err := client.Ping(ctx)
    
    hm.mu.Lock()
    tracker.pingInFlight = false
    
    if err == nil {
        // Successful pong
        tracker.lastPong = time.Now()
        tracker.missedPings = 0
        
        // Update activity in connection manager
        hm.connMgr.UpdateActivity(info.InstanceID)
    } else {
        // Failed ping
        tracker.missedPings++
        log.Printf("Instance %s: ping failed: %v (missed: %d)", 
            info.InstanceID, err, tracker.missedPings)
    }
    hm.mu.Unlock()
}

func (hm *HealthMonitor) getClientForInstance(instanceID string) *HubClient {
    // Find any session connected to this instance
    instances := hm.connMgr.ListInstances()
    for _, info := range instances {
        if info.InstanceID == instanceID && info.Client != nil {
            return info.Client
        }
    }
    return nil
}

func (hm *HealthMonitor) cleanupTrackers(activeInstances []*ConnectionInfo) {
    hm.mu.Lock()
    defer hm.mu.Unlock()
    
    // Build set of active instance IDs
    activeIDs := make(map[string]bool)
    for _, info := range activeInstances {
        activeIDs[info.InstanceID] = true
    }
    
    // Remove trackers for non-existent instances
    for id := range hm.pingTrackers {
        if !activeIDs[id] {
            delete(hm.pingTrackers, id)
        }
    }
}

// GetHealthStatus returns current health information
func (hm *HealthMonitor) GetHealthStatus(instanceID string) (lastPong time.Time, missedPings int, ok bool) {
    hm.mu.RLock()
    defer hm.mu.RUnlock()
    
    tracker, exists := hm.pingTrackers[instanceID]
    if !exists {
        return time.Time{}, 0, false
    }
    
    return tracker.lastPong, tracker.missedPings, true
}
```

### 2. Update Connection Manager

```go
// internal/mcp/connection_manager.go

// Add health monitor to ConnectionManager
type ConnectionManager struct {
    // ... existing fields ...
    
    healthMon *HealthMonitor
}

func NewConnectionManager() *ConnectionManager {
    cm := &ConnectionManager{
        // ... existing initialization ...
    }
    
    // Create health monitor
    cm.healthMon = NewHealthMonitor(cm)
    
    go cm.run()
    go cm.retryLoop() // New retry loop
    
    // Start health monitoring
    cm.healthMon.Start()
    
    return cm
}

// Add retry loop for disconnected instances
func (cm *ConnectionManager) retryLoop() {
    // Stagger retries to avoid thundering herd
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    retrySchedule := make(map[string]time.Time) // instanceID -> next retry time
    
    for {
        select {
        case <-ticker.C:
            now := time.Now()
            instances := cm.ListInstances()
            
            for _, info := range instances {
                if info.State != StateRetrying {
                    continue
                }
                
                // Check if it's time to retry
                nextRetry, exists := retrySchedule[info.InstanceID]
                if exists && now.Before(nextRetry) {
                    continue
                }
                
                // Calculate next retry time
                backoffIndex := info.RetryCount
                if backoffIndex >= len(RetryBackoff) {
                    // Max retries reached
                    cm.updateState(info.InstanceID, StateDead)
                    delete(retrySchedule, info.InstanceID)
                    continue
                }
                
                // Schedule retry
                retrySchedule[info.InstanceID] = now.Add(RetryBackoff[backoffIndex])
                
                // Attempt reconnection
                go cm.attemptReconnection(info.InstanceID)
            }
            
        case <-cm.stopCh:
            return
        }
    }
}

func (cm *ConnectionManager) attemptReconnection(instanceID string) {
    log.Printf("Attempting to reconnect to instance %s", instanceID)
    
    // Get instance info
    instances := cm.ListInstances()
    var info *ConnectionInfo
    for _, i := range instances {
        if i.InstanceID == instanceID {
            info = i
            break
        }
    }
    
    if info == nil || info.State != StateRetrying {
        return
    }
    
    // Increment retry count
    cm.incrementRetryCount(instanceID)
    
    // Try to connect
    client, err := NewHubClient(info.Port)
    if err != nil {
        log.Printf("Failed to create client for reconnection: %v", err)
        return
    }
    
    // Test connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := client.Initialize(ctx); err != nil {
        log.Printf("Reconnection failed for %s: %v", instanceID, err)
        return
    }
    
    // Success! Update state
    cm.setClient(instanceID, client)
    cm.updateState(instanceID, StateActive)
    
    // Reset retry count
    if i, exists := cm.connections[instanceID]; exists {
        i.RetryCount = 0
    }
    
    log.Printf("Successfully reconnected to instance %s", instanceID)
}

// Update Stop to clean up health monitor
func (cm *ConnectionManager) Stop() {
    cm.healthMon.Stop()
    close(cm.stopCh)
    <-cm.doneCh
}
```

### 3. MCP Ping Implementation

```go
// internal/mcp/ping.go
package mcp

import (
    "context"
    "encoding/json"
    "time"
)

// Ensure hub client supports ping
func (c *HubClient) Ping(ctx context.Context) error {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(),
        "method":  "ping",
    }
    
    result, err := c.sendRequest(ctx, request)
    if err != nil {
        return err
    }
    
    // Verify we got a pong response
    var response string
    if err := json.Unmarshal(result, &response); err != nil {
        return err
    }
    
    if response != "pong" {
        return fmt.Errorf("unexpected ping response: %s", response)
    }
    
    return nil
}

// Instance server must handle ping
func (s *Server) handlePing(ctx context.Context) (interface{}, error) {
    return "pong", nil
}
```

### 4. Health Status Tool

```go
// Add to hub tools
{
    Name:        "instances/health",
    Description: "Get health status of all instances",
    InputSchema: mcp.ToolInputSchema{
        Type:       "object",
        Properties: map[string]interface{}{},
    },
}

// Handler
func (h *HubServer) handleInstancesHealth(ctx context.Context) (*mcp.CallToolResult, error) {
    instances := h.connMgr.ListInstances()
    
    var output []map[string]interface{}
    for _, info := range instances {
        lastPong, missedPings, ok := h.healthMon.GetHealthStatus(info.InstanceID)
        
        healthInfo := map[string]interface{}{
            "id":           info.InstanceID,
            "name":         info.Name,
            "state":        info.State.String(),
            "last_activity": info.LastActivity,
        }
        
        if ok {
            healthInfo["last_pong"] = lastPong
            healthInfo["missed_pings"] = missedPings
            healthInfo["healthy"] = missedPings < MaxMissedPings
        }
        
        output = append(output, healthInfo)
    }
    
    data, _ := json.MarshalIndent(output, "", "  ")
    return &mcp.CallToolResult{
        Content: []mcp.Content{{
            Type: "text",
            Text: string(data),
        }},
    }, nil
}
```

## Testing Plan

### 1. Unit Tests

```go
// internal/mcp/health_monitor_test.go
func TestHealthMonitor(t *testing.T) {
    // Create mock connection manager
    connMgr := NewMockConnectionManager()
    monitor := NewHealthMonitor(connMgr)
    
    // Add test instance
    instance := &ConnectionInfo{
        InstanceID: "test-123",
        State:      StateActive,
    }
    connMgr.AddInstance(instance)
    
    // Start monitoring
    monitor.Start()
    defer monitor.Stop()
    
    // Wait for first ping
    time.Sleep(6 * time.Second)
    
    // Check health status
    lastPong, missedPings, ok := monitor.GetHealthStatus("test-123")
    assert.True(t, ok)
    assert.Equal(t, 0, missedPings)
    assert.WithinDuration(t, time.Now(), lastPong, 10*time.Second)
}

func TestMissedPings(t *testing.T) {
    // Test that 3 missed pings triggers state change
    // Mock client that doesn't respond to pings
    // Verify state changes to retrying
}

func TestReconnection(t *testing.T) {
    // Test reconnection with exponential backoff
    // Verify retry intervals are correct
    // Test max retries leads to dead state
}
```

### 2. Integration Test

```bash
#!/bin/bash
# test_health_monitoring.sh

# Start instance
cd test-project
brum --no-tui &
INSTANCE_PID=$!
sleep 2

# Start hub with debug logging
BRUMMER_MCP_DEBUG=true brum --mcp &
HUB_PID=$!
sleep 2

# Monitor health for 30 seconds
for i in {1..6}; do
    echo "Health check $i:"
    echo '{"jsonrpc":"2.0","id":'$i',"method":"tools/call","params":{"name":"instances/health"}}' | \
        nc -q 1 localhost 7777 | jq '.result.content[0].text' | jq '.'
    sleep 5
done

# Kill instance to test failure detection
kill $INSTANCE_PID

# Check health after instance death
sleep 25
echo "Health after instance death:"
echo '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"instances/health"}}' | \
    nc -q 1 localhost 7777 | jq '.result.content[0].text' | jq '.'

# Cleanup
kill $HUB_PID
```

### 3. Stress Test

```go
func TestHealthMonitorStress(t *testing.T) {
    // Add 100 instances
    // Verify all get pinged within reasonable time
    // Kill random instances and verify detection
    // Check no goroutine leaks
}
```

## Success Criteria

1. ✅ Ping sent every 5 seconds to active instances
2. ✅ Pong response updates last activity
3. ✅ 3 missed pings triggers state change
4. ✅ Exponential backoff for reconnection
5. ✅ Dead instances cleaned up properly
6. ✅ Health status available via tool
7. ✅ No goroutine leaks
8. ✅ Graceful shutdown

## Edge Cases

### 1. Slow Pong Response
- Use 2-second timeout
- Don't wait for previous ping

### 2. Instance Restarts
- Quick reconnection on same port
- Reset health tracking

### 3. Network Hiccup
- Don't immediately mark as dead
- Allow for transient failures

### 4. Many Instances
- Stagger pings to avoid load spikes
- Use goroutine pool if needed

### 5. Ping During Shutdown
- Cancel all in-flight pings
- Clean shutdown

## Performance Considerations

1. **Goroutine Management**
   - Pool goroutines for pings
   - Limit concurrent pings

2. **Memory Usage**
   - Clean up old trackers
   - Bounded data structures

3. **CPU Usage**
   - Efficient timer usage
   - Avoid busy loops

## Next Steps

1. Step 6: End-to-end testing and verification

## Code Checklist

- [ ] Implement health monitor component
- [ ] Add ping/pong to hub client
- [ ] Update connection manager with retry logic
- [ ] Add health status tool
- [ ] Create comprehensive tests
- [ ] Test failure detection timing
- [ ] Verify reconnection works
- [ ] Check for goroutine leaks