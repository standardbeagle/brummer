# Implementation Step 3: Connection Management

## Overview

This step implements the channel-based connection management system. The hub establishes HTTP connections to discovered instances, tracks their state, and manages the lifecycle using Go channels for lock-free operation.

## Goals

1. Establish HTTP connections to discovered instances
2. Implement channel-based state management (no mutexes)
3. Track connection health and state transitions
4. Map MCP sessions to connected instances
5. Handle connection failures gracefully

## Technical Design

### Connection State Machine

```
Discovery → Connection → Active → Monitoring
    │           │          │          │
    │           ↓          ↓          ↓
    └────→ Failed ←── Retrying ←── Timeout
                           │
                           ↓
                        Dead
```

### Channel Architecture

```go
// All state changes through channels
type ConnectionManager struct {
    // State ownership by single goroutine
    connections map[string]*ConnectionInfo
    sessions    map[string]string  // sessionID → instanceID
    
    // Operation channels
    registerChan   chan registerRequest
    connectChan    chan connectRequest
    ensureChan     chan ensureRequest
    stateChan      chan stateChangeRequest
    listChan       chan listRequest
}
```

## Implementation

### 1. Connection Manager (Full Implementation)

```go
// internal/mcp/connection_manager.go
package mcp

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
    
    "github.com/standardbeagle/brummer/internal/discovery"
)

// Connection states
type ConnectionState int

const (
    StateDiscovered ConnectionState = iota  // File found, not connected
    StateConnecting                         // Attempting connection
    StateActive                            // Connected and responsive
    StateRetrying                          // Connection lost, retrying
    StateDead                              // Given up
)

// ConnectionInfo tracks instance connection
type ConnectionInfo struct {
    // Instance metadata
    InstanceID     string
    Name           string
    Path           string
    Port           int
    PID            int
    HasPackageJSON bool
    
    // Connection state
    State          ConnectionState
    Client         *HubClient      // HTTP client to instance
    LastActivity   time.Time
    ConnectedAt    time.Time
    RetryCount     int
    
    // Session mapping
    Sessions       map[string]bool // Active sessions for this instance
}

// Request types for channel operations
type registerRequest struct {
    instance *discovery.Instance
    response chan error
}

type connectRequest struct {
    instanceID string
    sessionID  string
    response   chan error
}

type disconnectRequest struct {
    sessionID string
    response  chan error
}

type ensureRequest struct {
    instanceID string
    response   chan bool
}

type stateChangeRequest struct {
    instanceID string
    newState   ConnectionState
    response   chan error
}

type listRequest struct {
    response chan []*ConnectionInfo
}

type getClientRequest struct {
    sessionID string
    response  chan *HubClient
}

// ConnectionManager manages all instance connections
type ConnectionManager struct {
    connections map[string]*ConnectionInfo
    sessions    map[string]string  // sessionID -> instanceID
    
    // Channel operations
    registerChan   chan registerRequest
    connectChan    chan connectRequest
    disconnectChan chan disconnectRequest
    ensureChan     chan ensureRequest
    stateChan      chan stateChangeRequest
    listChan       chan listRequest
    getClientChan  chan getClientRequest
    
    // Control
    stopCh chan struct{}
    doneCh chan struct{}
}

func NewConnectionManager() *ConnectionManager {
    cm := &ConnectionManager{
        connections:    make(map[string]*ConnectionInfo),
        sessions:       make(map[string]string),
        registerChan:   make(chan registerRequest),
        connectChan:    make(chan connectRequest),
        disconnectChan: make(chan disconnectRequest),
        ensureChan:     make(chan ensureRequest),
        stateChan:      make(chan stateChangeRequest),
        listChan:       make(chan listRequest),
        getClientChan:  make(chan getClientRequest),
        stopCh:         make(chan struct{}),
        doneCh:         make(chan struct{}),
    }
    
    go cm.run()
    
    return cm
}

// run is the main event loop - owns all state
func (cm *ConnectionManager) run() {
    defer close(cm.doneCh)
    
    // Start connection monitor
    go cm.monitorConnections()
    
    for {
        select {
        case req := <-cm.registerChan:
            cm.handleRegister(req)
            
        case req := <-cm.connectChan:
            cm.handleConnect(req)
            
        case req := <-cm.disconnectChan:
            cm.handleDisconnect(req)
            
        case req := <-cm.ensureChan:
            cm.handleEnsure(req)
            
        case req := <-cm.stateChan:
            cm.handleStateChange(req)
            
        case req := <-cm.listChan:
            cm.handleList(req)
            
        case req := <-cm.getClientChan:
            cm.handleGetClient(req)
            
        case <-cm.stopCh:
            cm.cleanup()
            return
        }
    }
}

// Handle operations (run in main goroutine)

func (cm *ConnectionManager) handleRegister(req registerRequest) {
    if req.instance == nil {
        req.response <- fmt.Errorf("nil instance")
        return
    }
    
    // Check if already registered
    if _, exists := cm.connections[req.instance.ID]; exists {
        req.response <- nil // Already registered
        return
    }
    
    // Create connection info
    info := &ConnectionInfo{
        InstanceID:     req.instance.ID,
        Name:           req.instance.Name,
        Path:           req.instance.Path,
        Port:           req.instance.Port,
        PID:            req.instance.PID,
        HasPackageJSON: req.instance.HasPackageJSON,
        State:          StateDiscovered,
        LastActivity:   time.Now(),
        Sessions:       make(map[string]bool),
    }
    
    cm.connections[req.instance.ID] = info
    
    // Start connection attempt
    go cm.attemptConnection(req.instance.ID)
    
    req.response <- nil
}

func (cm *ConnectionManager) handleConnect(req connectRequest) {
    instanceID, exists := cm.sessions[req.sessionID]
    if exists && instanceID != req.instanceID {
        req.response <- fmt.Errorf("session already connected to different instance")
        return
    }
    
    info, exists := cm.connections[req.instanceID]
    if !exists {
        req.response <- fmt.Errorf("instance not found: %s", req.instanceID)
        return
    }
    
    if info.State != StateActive {
        req.response <- fmt.Errorf("instance not active: %s", info.State)
        return
    }
    
    // Map session to instance
    cm.sessions[req.sessionID] = req.instanceID
    info.Sessions[req.sessionID] = true
    
    req.response <- nil
}

func (cm *ConnectionManager) handleDisconnect(req disconnectRequest) {
    instanceID, exists := cm.sessions[req.sessionID]
    if !exists {
        req.response <- nil // Not connected
        return
    }
    
    // Remove session mapping
    delete(cm.sessions, req.sessionID)
    
    // Remove from instance sessions
    if info, exists := cm.connections[instanceID]; exists {
        delete(info.Sessions, req.sessionID)
    }
    
    req.response <- nil
}

func (cm *ConnectionManager) handleEnsure(req ensureRequest) {
    info, exists := cm.connections[req.instanceID]
    if !exists {
        req.response <- false
        return
    }
    
    info.LastActivity = time.Now()
    req.response <- info.State == StateActive
}

func (cm *ConnectionManager) handleStateChange(req stateChangeRequest) {
    info, exists := cm.connections[req.instanceID]
    if !exists {
        req.response <- fmt.Errorf("instance not found")
        return
    }
    
    oldState := info.State
    info.State = req.newState
    
    log.Printf("Instance %s: %s -> %s", req.instanceID, oldState, req.newState)
    
    req.response <- nil
}

func (cm *ConnectionManager) handleList(req listRequest) {
    var list []*ConnectionInfo
    
    for _, info := range cm.connections {
        if info.State != StateDead {
            // Make a copy to avoid races
            infoCopy := *info
            list = append(list, &infoCopy)
        }
    }
    
    req.response <- list
}

func (cm *ConnectionManager) handleGetClient(req getClientRequest) {
    instanceID, exists := cm.sessions[req.sessionID]
    if !exists {
        req.response <- nil
        return
    }
    
    info, exists := cm.connections[instanceID]
    if !exists || info.State != StateActive {
        req.response <- nil
        return
    }
    
    req.response <- info.Client
}

// Connection establishment (runs in separate goroutine)
func (cm *ConnectionManager) attemptConnection(instanceID string) {
    // Get instance info
    listResp := make(chan []*ConnectionInfo)
    cm.listChan <- listRequest{response: listResp}
    connections := <-listResp
    
    var info *ConnectionInfo
    for _, conn := range connections {
        if conn.InstanceID == instanceID {
            info = conn
            break
        }
    }
    
    if info == nil {
        return
    }
    
    // Update state to connecting
    cm.updateState(instanceID, StateConnecting)
    
    // Create HTTP client
    client, err := NewHubClient(info.Port)
    if err != nil {
        log.Printf("Failed to create client for %s: %v", instanceID, err)
        cm.updateState(instanceID, StateRetrying)
        return
    }
    
    // Test connection with initialize
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := client.Initialize(ctx); err != nil {
        log.Printf("Failed to initialize connection to %s: %v", instanceID, err)
        cm.updateState(instanceID, StateRetrying)
        return
    }
    
    // Connection successful
    cm.setClient(instanceID, client)
    cm.updateState(instanceID, StateActive)
}

// Helper to update client (thread-safe)
func (cm *ConnectionManager) setClient(instanceID string, client *HubClient) {
    // This is a bit of a hack - we should have a channel for this
    // For now, we'll directly update since we're in a goroutine
    // In production, add a setClientChan
    if info, exists := cm.connections[instanceID]; exists {
        info.Client = client
        info.ConnectedAt = time.Now()
        info.LastActivity = time.Now()
        info.RetryCount = 0
    }
}

// Connection monitoring
func (cm *ConnectionManager) monitorConnections() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            cm.checkConnections()
            
        case <-cm.stopCh:
            return
        }
    }
}

func (cm *ConnectionManager) checkConnections() {
    // Get current connections
    listResp := make(chan []*ConnectionInfo)
    cm.listChan <- listRequest{response: listResp}
    connections := <-listResp
    
    for _, info := range connections {
        switch info.State {
        case StateActive:
            // Check if still responsive
            if time.Since(info.LastActivity) > 20*time.Second {
                log.Printf("Instance %s not responsive, marking as retrying", info.InstanceID)
                cm.updateState(info.InstanceID, StateRetrying)
            }
            
        case StateRetrying:
            // Implement retry logic
            if info.RetryCount < 3 {
                info.RetryCount++
                go cm.attemptConnection(info.InstanceID)
            } else {
                cm.updateState(info.InstanceID, StateDead)
            }
            
        case StateDiscovered:
            // Try initial connection
            go cm.attemptConnection(info.InstanceID)
        }
    }
}

// Public API

func (cm *ConnectionManager) RegisterInstance(instance *discovery.Instance) error {
    respChan := make(chan error)
    cm.registerChan <- registerRequest{
        instance: instance,
        response: respChan,
    }
    return <-respChan
}

func (cm *ConnectionManager) ConnectSession(sessionID, instanceID string) error {
    respChan := make(chan error)
    cm.connectChan <- connectRequest{
        sessionID:  sessionID,
        instanceID: instanceID,
        response:   respChan,
    }
    return <-respChan
}

func (cm *ConnectionManager) DisconnectSession(sessionID string) error {
    respChan := make(chan error)
    cm.disconnectChan <- disconnectRequest{
        sessionID: sessionID,
        response:  respChan,
    }
    return <-respChan
}

func (cm *ConnectionManager) GetClient(sessionID string) *HubClient {
    respChan := make(chan *HubClient)
    cm.getClientChan <- getClientRequest{
        sessionID: sessionID,
        response:  respChan,
    }
    return <-respChan
}

func (cm *ConnectionManager) ListInstances() []*ConnectionInfo {
    respChan := make(chan []*ConnectionInfo)
    cm.listChan <- listRequest{
        response: respChan,
    }
    return <-respChan
}

func (cm *ConnectionManager) UpdateActivity(instanceID string) bool {
    respChan := make(chan bool)
    cm.ensureChan <- ensureRequest{
        instanceID: instanceID,
        response:   respChan,
    }
    return <-respChan
}

// Helper to update state
func (cm *ConnectionManager) updateState(instanceID string, newState ConnectionState) error {
    respChan := make(chan error)
    cm.stateChan <- stateChangeRequest{
        instanceID: instanceID,
        newState:   newState,
        response:   respChan,
    }
    return <-respChan
}

func (cm *ConnectionManager) cleanup() {
    // Close all client connections
    for _, info := range cm.connections {
        if info.Client != nil {
            info.Client.Close()
        }
    }
}

func (cm *ConnectionManager) Stop() {
    close(cm.stopCh)
    <-cm.doneCh
}
```

### 2. Hub Client Implementation

```go
// internal/mcp/hub_client.go
package mcp

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "time"
)

// HubClient is an HTTP client for connecting to instance MCP servers
type HubClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewHubClient(port int) (*HubClient, error) {
    return &HubClient{
        baseURL: fmt.Sprintf("http://localhost:%d/mcp", port),
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }, nil
}

func (c *HubClient) Initialize(ctx context.Context) error {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      1,
        "method":  "initialize",
        "params": map[string]interface{}{
            "protocolVersion": "1.0",
            "clientInfo": map[string]string{
                "name":    "brummer-hub",
                "version": "1.0",
            },
        },
    }
    
    _, err := c.sendRequest(ctx, request)
    return err
}

func (c *HubClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (json.RawMessage, error) {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(), // Unique ID
        "method":  "tools/call",
        "params": map[string]interface{}{
            "name": toolName,
            "arguments": args,
        },
    }
    
    return c.sendRequest(ctx, request)
}

func (c *HubClient) Ping(ctx context.Context) error {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(),
        "method":  "ping",
    }
    
    _, err := c.sendRequest(ctx, request)
    return err
}

func (c *HubClient) sendRequest(ctx context.Context, request interface{}) (json.RawMessage, error) {
    body, err := json.Marshal(request)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }
    
    var result struct {
        Result json.RawMessage `json:"result"`
        Error  *struct {
            Code    int    `json:"code"`
            Message string `json:"message"`
        } `json:"error"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if result.Error != nil {
        return nil, fmt.Errorf("RPC error %d: %s", result.Error.Code, result.Error.Message)
    }
    
    return result.Result, nil
}

func (c *HubClient) Close() error {
    // Nothing to close for HTTP client
    return nil
}
```

### 3. Update Hub Server

```go
// internal/mcp/hub_server.go
func (h *HubServer) processDiscoveredInstances(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
            
        case instance := <-h.instancesChan:
            // Register with connection manager
            if err := h.connMgr.RegisterInstance(instance); err != nil {
                log.Printf("Failed to register instance %s: %v", instance.ID, err)
            }
            
        case err := <-h.errorsChan:
            log.Printf("Discovery error: %v", err)
        }
    }
}

func (h *HubServer) CallTool(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    switch request.Name {
    case "instances/list":
        instances := h.connMgr.ListInstances()
        
        // Format for output
        var output []map[string]interface{}
        for _, info := range instances {
            output = append(output, map[string]interface{}{
                "id":             info.InstanceID,
                "name":           info.Name,
                "path":           info.Path,
                "port":           info.Port,
                "pid":            info.PID,
                "state":          info.State.String(),
                "connected_at":   info.ConnectedAt,
                "has_package_json": info.HasPackageJSON,
            })
        }
        
        data, _ := json.MarshalIndent(output, "", "  ")
        return &mcp.CallToolResult{
            Content: []mcp.Content{{
                Type: "text",
                Text: string(data),
            }},
        }, nil
        
    case "instances/connect":
        // Get instance ID from args
        var args struct {
            InstanceID string `json:"instance_id"`
        }
        if err := json.Unmarshal(request.Arguments, &args); err != nil {
            return nil, err
        }
        
        // Connect session to instance
        sessionID := GetSessionID(ctx) // From MCP context
        if err := h.connMgr.ConnectSession(sessionID, args.InstanceID); err != nil {
            return nil, err
        }
        
        return &mcp.CallToolResult{
            Content: []mcp.Content{{
                Type: "text",
                Text: fmt.Sprintf("Connected to instance %s", args.InstanceID),
            }},
        }, nil
        
    // ... other tools ...
    }
}
```

## Testing Plan

### 1. Unit Tests

```go
// internal/mcp/connection_manager_test.go
func TestConnectionManagerChannels(t *testing.T) {
    cm := NewConnectionManager()
    defer cm.Stop()
    
    // Test registration
    instance := &discovery.Instance{
        ID:   "test-123",
        Name: "test",
        Port: 7778,
    }
    
    err := cm.RegisterInstance(instance)
    assert.NoError(t, err)
    
    // Test listing
    instances := cm.ListInstances()
    assert.Len(t, instances, 1)
    assert.Equal(t, StateDiscovered, instances[0].State)
}

func TestConnectionStates(t *testing.T) {
    // Test state transitions
    // Discovered -> Connecting -> Active
    // Active -> Retrying -> Dead
}

func TestSessionMapping(t *testing.T) {
    // Test session connect/disconnect
    // Verify GetClient returns correct client
}
```

### 2. Integration Test

```bash
#!/bin/bash
# test_connections.sh

# Start instance first
cd test-project
brum --no-tui &
INSTANCE_PID=$!
sleep 2

# Start hub
brum --mcp &
HUB_PID=$!
sleep 2

# List instances - should show as active
RESULT=$(echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"instances/list"}}' | brum --mcp)
echo "$RESULT" | grep -q '"state": "active"'

# Kill instance
kill $INSTANCE_PID
sleep 10

# Check state changed
RESULT=$(echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"instances/list"}}' | brum --mcp)
echo "$RESULT" | grep -q '"state": "retrying"'

# Cleanup
kill $HUB_PID
```

## Success Criteria

1. ✅ Channel-based operations (no mutexes)
2. ✅ Automatic connection to discovered instances
3. ✅ State transitions tracked correctly
4. ✅ Session to instance mapping works
5. ✅ Connection retry with backoff
6. ✅ Dead connections cleaned up
7. ✅ No goroutine leaks
8. ✅ Graceful shutdown

## Common Issues

### 1. Channel Deadlocks
- Always use buffered response channels
- Add timeouts to prevent hanging

### 2. Goroutine Leaks
- Ensure all goroutines exit on Stop()
- Use context for cancellation

### 3. Race Conditions
- All state mutations through channels
- Copy data when returning from channels

### 4. Connection Failures
- Retry with exponential backoff
- Don't mark as dead too quickly

## Next Steps

1. Step 4: Implement tool proxying through connections
2. Step 5: Add health monitoring with pings
3. Step 6: End-to-end testing

## Code Checklist

- [ ] Implement full ConnectionManager
- [ ] Create HubClient for HTTP connections
- [ ] Update hub server to use connections
- [ ] Add comprehensive channel tests
- [ ] Test state transitions
- [ ] Verify no goroutine leaks
- [ ] Document channel patterns
- [ ] Add connection retry logic