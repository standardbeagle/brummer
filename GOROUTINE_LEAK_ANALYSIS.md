# Goroutine Leak Analysis for StreamableServer

## Summary
Found several potential goroutine leaks in the StreamableServer implementation:

## 1. BroadcastNotification Goroutines (Line 665)
**Location**: `streamable_server.go:665`
```go
for _, session := range sessions {
    go s.sendSSEEvent(session, "message", notification)
}
```
**Issue**: 
- Creates goroutines without tracking or cleanup
- No panic recovery
- No context cancellation
- Could accumulate if sendSSEEvent blocks

**Fix**: Use a worker pool or add timeout/context to sendSSEEvent

## 2. setupEventBroadcasting Goroutine (Line 711)
**Location**: `streamable_server.go:711`
```go
go s.setupEventBroadcasting()
```
**Issue**:
- Started in Start() but never stopped
- Subscribes to events but doesn't unsubscribe
- No cleanup mechanism when server stops

**Fix**: Add proper cleanup in Stop() method

## 3. Process Monitoring Goroutine in tools.go (Line 216)
**Location**: `tools.go:216-224`
```go
go func() {
    for {
        time.Sleep(100 * time.Millisecond)
        if process.Status != "running" {
            close(logChan)
            break
        }
    }
}()
```
**Issue**:
- No panic recovery
- Polling instead of event-driven
- No context cancellation
- Could leak if process status tracking fails

**Fix**: Use context cancellation and panic recovery

## 4. Connection Manager Goroutines
**Location**: `connection_manager.go`

### 4.1 monitorConnections (Line 150)
```go
go cm.monitorConnections()
```
**Issue**:
- Started in run() but cleanup depends on stopCh
- No panic recovery

### 4.2 attemptConnection (Lines 420, 427)
```go
go cm.attemptConnection(info.InstanceID)
```
**Issue**:
- Spawned without tracking
- No timeout context for the entire operation
- Could accumulate if connections hang

## 5. Missing Cleanup in handleStreamingConnection
**Location**: `streamable_server.go:375-507`
**Issues**:
- Event subscriptions don't get unsubscribed on disconnect
- Resource update handler registration but cleanup only in defer
- Channel cleanup could be improved

## 6. Channel Leaks
**Location**: Various
- `eventChan` and `resourceUpdateChan` in handleStreamingConnection
- `logChan` in tools.go streaming handler
- REPL response channels in `replResponseChans` map

**Issue**: Channels might not be properly closed/drained in all error paths

## Recommendations

1. **Add Panic Recovery**: Wrap all goroutines with defer/recover
2. **Use Context**: Pass context.Context to all goroutines for cancellation
3. **Track Goroutines**: Use sync.WaitGroup or similar for tracking
4. **Proper Cleanup**: Ensure Stop() method cleans up all resources
5. **Event Unsubscribe**: Track and unsubscribe all event subscriptions
6. **Timeout Operations**: Add timeouts to blocking operations
7. **Worker Pools**: Use bounded worker pools instead of spawning unlimited goroutines

## Example Fix Pattern
```go
func (s *StreamableServer) safeGo(ctx context.Context, fn func()) {
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        defer func() {
            if r := recover(); r != nil {
                s.logStore.Add("mcp-server", "ERROR", 
                    fmt.Sprintf("Panic in goroutine: %v", r), true)
            }
        }()
        
        select {
        case <-ctx.Done():
            return
        default:
            fn()
        }
    }()
}
```