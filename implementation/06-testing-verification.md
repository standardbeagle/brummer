# Implementation Step 6: Testing and Verification

## Overview

This final step provides comprehensive testing strategies and verification procedures to ensure the MCP hub architecture works correctly end-to-end. It includes unit tests, integration tests, stress tests, and real-world usage scenarios.

## Goals

1. Verify all components work together correctly
2. Test error handling and edge cases
3. Ensure performance meets requirements
4. Validate with real MCP clients
5. Create automated test suite

## Test Categories

### 1. Unit Tests
- Individual component testing
- Mock dependencies
- Fast, isolated tests

### 2. Integration Tests
- Component interaction testing
- Real network connections
- File system operations

### 3. End-to-End Tests
- Complete workflows
- Real MCP clients
- Multi-instance scenarios

### 4. Stress Tests
- Performance under load
- Resource leak detection
- Concurrent operations

## Test Implementation

### 1. Test Infrastructure

```go
// test/helpers/test_helpers.go
package helpers

import (
    "context"
    "fmt"
    "io/ioutil"
    "net"
    "os"
    "path/filepath"
    "testing"
    "time"
    
    "github.com/standardbeagle/brummer/internal/discovery"
    "github.com/standardbeagle/brummer/internal/mcp"
)

// TestInstance represents a test brummer instance
type TestInstance struct {
    ID       string
    Port     int
    Path     string
    Server   *mcp.Server
    Registry *discovery.Registry
    Cancel   context.CancelFunc
}

// StartTestInstance starts a test instance on a random port
func StartTestInstance(t *testing.T, name string) *TestInstance {
    // Get free port
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatal(err)
    }
    port := listener.Addr().(*net.TCPAddr).Port
    listener.Close()
    
    // Create temp directory
    tempDir := t.TempDir()
    
    // Create test instance
    instance := &TestInstance{
        ID:   fmt.Sprintf("test-%s-%d", name, time.Now().Unix()),
        Port: port,
        Path: tempDir,
    }
    
    // Start MCP server
    ctx, cancel := context.WithCancel(context.Background())
    instance.Cancel = cancel
    
    go func() {
        server := mcp.NewServer()
        instance.Server = server
        
        // Start listening
        listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
        if err != nil {
            t.Error(err)
            return
        }
        
        // Register with discovery
        registry := discovery.NewRegistry()
        instance.Registry = registry
        
        if err := registry.Register(tempDir, port, name, true); err != nil {
            t.Error(err)
        }
        
        // Serve
        server.Serve(listener)
    }()
    
    // Wait for server to start
    WaitForPort(t, port, 5*time.Second)
    
    return instance
}

// StopTestInstance stops a test instance
func (ti *TestInstance) Stop() {
    ti.Cancel()
    if ti.Registry != nil {
        ti.Registry.Unregister()
    }
}

// WaitForPort waits for a port to be listening
func WaitForPort(t *testing.T, port int, timeout time.Duration) {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
        if err == nil {
            conn.Close()
            return
        }
        time.Sleep(100 * time.Millisecond)
    }
    t.Fatalf("Port %d not listening after %s", port, timeout)
}

// TestHub represents a test hub
type TestHub struct {
    Server *mcp.HubServer
    Port   int
    Cancel context.CancelFunc
}

// StartTestHub starts a test hub
func StartTestHub(t *testing.T) *TestHub {
    hub, err := mcp.NewHubServer()
    if err != nil {
        t.Fatal(err)
    }
    
    // Start hub
    ctx, cancel := context.WithCancel(context.Background())
    
    go func() {
        if err := hub.Start(ctx); err != nil {
            t.Error(err)
        }
    }()
    
    return &TestHub{
        Server: hub,
        Cancel: cancel,
    }
}

// MCPClient is a test MCP client
type MCPClient struct {
    t    *testing.T
    conn net.Conn
}

// NewMCPClient creates a test MCP client
func NewMCPClient(t *testing.T) *MCPClient {
    // For stdio, we'd use exec.Command
    // For testing, we'll use direct connection
    return &MCPClient{t: t}
}

// Initialize sends initialize request
func (c *MCPClient) Initialize() error {
    // Send initialize request
    // Parse response
    return nil
}

// CallTool calls an MCP tool
func (c *MCPClient) CallTool(name string, args map[string]interface{}) (interface{}, error) {
    // Send tool call
    // Parse response
    return nil, nil
}
```

### 2. Component Tests

```go
// internal/mcp/hub_server_test.go
package mcp

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestHubServerLifecycle(t *testing.T) {
    // Create hub
    hub, err := NewHubServer()
    require.NoError(t, err)
    
    // Start hub
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    err = hub.Start(ctx)
    require.NoError(t, err)
    
    // Initialize
    result, err := hub.Initialize(ctx)
    require.NoError(t, err)
    assert.Equal(t, "brummer-hub", result.ServerInfo.Name)
    
    // List tools - should have hub tools
    tools, err := hub.ListTools(ctx)
    require.NoError(t, err)
    assert.True(t, len(tools.Tools) >= 3) // At least 3 hub tools
    
    // Shutdown
    cancel()
    // Verify clean shutdown
}

func TestInstanceDiscovery(t *testing.T) {
    // Start hub
    hub := helpers.StartTestHub(t)
    defer hub.Cancel()
    
    // Start instance
    instance := helpers.StartTestInstance(t, "test-project")
    defer instance.Stop()
    
    // Wait for discovery
    time.Sleep(500 * time.Millisecond)
    
    // List instances
    ctx := context.Background()
    result, err := hub.Server.CallTool(ctx, &mcp.CallToolRequest{
        Name: "instances/list",
    })
    require.NoError(t, err)
    
    // Verify instance appears
    // Parse result and check
}

func TestConnectionManagement(t *testing.T) {
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
    
    // Test state transitions
    instances := cm.ListInstances()
    assert.Len(t, instances, 1)
    assert.Equal(t, StateDiscovered, instances[0].State)
    
    // Test session mapping
    err = cm.ConnectSession("session-1", "test-123")
    assert.Error(t, err) // Should fail - not active
    
    // Update to active
    err = cm.UpdateState("test-123", StateActive)
    assert.NoError(t, err)
    
    // Now connect should work
    err = cm.ConnectSession("session-1", "test-123")
    assert.NoError(t, err)
}
```

### 3. Integration Tests

```go
// test/integration/hub_integration_test.go
package integration

import (
    "context"
    "encoding/json"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCompleteWorkflow(t *testing.T) {
    // Start hub
    hub := helpers.StartTestHub(t)
    defer hub.Cancel()
    
    // Start two instances
    instance1 := helpers.StartTestInstance(t, "project1")
    defer instance1.Stop()
    
    instance2 := helpers.StartTestInstance(t, "project2")
    defer instance2.Stop()
    
    // Create MCP client
    client := helpers.NewMCPClient(t)
    
    // Initialize connection
    err := client.Initialize()
    require.NoError(t, err)
    
    // List instances
    result, err := client.CallTool("instances/list", nil)
    require.NoError(t, err)
    
    instances := parseInstances(result)
    assert.Len(t, instances, 2)
    
    // Connect to first instance
    err = client.CallTool("instances/connect", map[string]interface{}{
        "instance_id": instance1.ID,
    })
    require.NoError(t, err)
    
    // Call instance tool
    scripts, err := client.CallTool("scripts/list", nil)
    require.NoError(t, err)
    assert.NotNil(t, scripts)
    
    // Disconnect
    err = client.CallTool("instances/disconnect", nil)
    require.NoError(t, err)
    
    // Connect to second instance
    err = client.CallTool("instances/connect", map[string]interface{}{
        "instance_id": instance2.ID,
    })
    require.NoError(t, err)
}

func TestHealthMonitoring(t *testing.T) {
    hub := helpers.StartTestHub(t)
    defer hub.Cancel()
    
    instance := helpers.StartTestInstance(t, "health-test")
    defer instance.Stop()
    
    // Wait for connection
    time.Sleep(2 * time.Second)
    
    // Check health
    health, err := hub.Server.CallTool(context.Background(), &mcp.CallToolRequest{
        Name: "instances/health",
    })
    require.NoError(t, err)
    
    // Verify healthy
    // Parse and check missed_pings = 0
    
    // Stop instance
    instance.Stop()
    
    // Wait for detection (20+ seconds)
    time.Sleep(25 * time.Second)
    
    // Check health again
    health, err = hub.Server.CallTool(context.Background(), &mcp.CallToolRequest{
        Name: "instances/health",
    })
    require.NoError(t, err)
    
    // Should show as retrying or dead
}
```

### 4. End-to-End Test Script

```bash
#!/bin/bash
# test/e2e/test_complete_flow.sh

set -e

echo "=== End-to-End MCP Hub Test ==="

# Clean up any existing instances
rm -rf ~/.local/share/brummer/instances/*

# Start test projects
echo "Starting test instances..."
cd test/fixtures/project1
brum --no-tui &
INSTANCE1_PID=$!

cd ../project2
brum --no-tui &
INSTANCE2_PID=$!

sleep 2

# Start hub
echo "Starting hub..."
cd ../../..
brum --mcp > hub.log 2>&1 &
HUB_PID=$!

sleep 1

# Test with MCP inspector
echo "Testing with MCP inspector..."
npm install -g @modelcontextprotocol/inspector

# Create test script for inspector
cat > test_inspector.js << 'EOF'
const { spawn } = require('child_process');

async function runTest() {
    // Connect to hub
    const hub = spawn('brum', ['--mcp']);
    
    // Send initialize
    hub.stdin.write(JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: { protocolVersion: "1.0" }
    }) + '\n');
    
    // Wait for response
    // ... parse and verify ...
    
    // List instances
    hub.stdin.write(JSON.stringify({
        jsonrpc: "2.0",
        id: 2,
        method: "tools/call",
        params: {
            name: "instances/list"
        }
    }) + '\n');
    
    // ... continue test ...
}

runTest().catch(console.error);
EOF

node test_inspector.js

# Test with Claude Desktop config
echo "Testing Claude Desktop configuration..."
cat > claude_desktop_test.json << EOF
{
  "mcpServers": {
    "brummer-hub": {
      "command": "brum",
      "args": ["--mcp"]
    }
  }
}
EOF

# Verify configuration works
# This would require actual Claude Desktop

# Cleanup
echo "Cleaning up..."
kill $INSTANCE1_PID $INSTANCE2_PID $HUB_PID

echo "=== Test Complete ==="
```

### 5. Stress Tests

```go
// test/stress/stress_test.go
package stress

import (
    "context"
    "sync"
    "testing"
    "time"
)

func TestManyInstances(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping stress test")
    }
    
    hub := helpers.StartTestHub(t)
    defer hub.Cancel()
    
    // Start 50 instances
    var instances []*helpers.TestInstance
    for i := 0; i < 50; i++ {
        instance := helpers.StartTestInstance(t, fmt.Sprintf("stress-%d", i))
        instances = append(instances, instance)
        defer instance.Stop()
    }
    
    // Wait for discovery
    time.Sleep(5 * time.Second)
    
    // List all instances
    result, err := hub.Server.CallTool(context.Background(), &mcp.CallToolRequest{
        Name: "instances/list",
    })
    require.NoError(t, err)
    
    // Should see all 50
    
    // Connect to random instances concurrently
    var wg sync.WaitGroup
    for i := 0; i < 20; i++ {
        wg.Add(1)
        go func(sessionID int) {
            defer wg.Done()
            
            // Connect to random instance
            instanceID := instances[sessionID%len(instances)].ID
            
            ctx := WithSessionID(context.Background(), fmt.Sprintf("session-%d", sessionID))
            err := hub.Server.CallTool(ctx, &mcp.CallToolRequest{
                Name: "instances/connect",
                Arguments: map[string]interface{}{
                    "instance_id": instanceID,
                },
            })
            assert.NoError(t, err)
            
            // Call some tools
            for j := 0; j < 10; j++ {
                _, err = hub.Server.CallTool(ctx, &mcp.CallToolRequest{
                    Name: "scripts/list",
                })
                assert.NoError(t, err)
            }
        }(i)
    }
    
    wg.Wait()
    
    // Check for goroutine leaks
    initialGoroutines := runtime.NumGoroutine()
    time.Sleep(1 * time.Second)
    finalGoroutines := runtime.NumGoroutine()
    
    assert.InDelta(t, initialGoroutines, finalGoroutines, 10)
}

func TestRapidConnectionChurn(t *testing.T) {
    // Test instances appearing and disappearing rapidly
    // Verify hub handles it gracefully
}
```

### 6. Performance Benchmarks

```go
// test/bench/benchmark_test.go
package bench

import (
    "context"
    "testing"
)

func BenchmarkInstanceList(b *testing.B) {
    hub := helpers.StartTestHub(b)
    defer hub.Cancel()
    
    // Add 10 instances
    for i := 0; i < 10; i++ {
        instance := helpers.StartTestInstance(b, fmt.Sprintf("bench-%d", i))
        defer instance.Stop()
    }
    
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := hub.Server.CallTool(ctx, &mcp.CallToolRequest{
            Name: "instances/list",
        })
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkToolProxy(b *testing.B) {
    // Benchmark tool call throughput
}
```

## Test Scenarios

### 1. Basic Functionality
- [ ] Hub starts with no instances
- [ ] Instance discovery works
- [ ] Connection establishment
- [ ] Tool proxying
- [ ] Clean disconnect

### 2. Error Handling
- [ ] Instance dies during operation
- [ ] Network failures
- [ ] Invalid tool calls
- [ ] Malformed requests
- [ ] Resource exhaustion

### 3. Edge Cases
- [ ] Instance restart on same port
- [ ] Port conflicts
- [ ] File permission issues
- [ ] Disk full scenarios
- [ ] Clock skew

### 4. Performance
- [ ] Startup time < 100ms
- [ ] Discovery time < 50ms
- [ ] Tool response < 200ms overhead
- [ ] Handle 100+ instances
- [ ] No memory leaks

### 5. Compatibility
- [ ] Works with Claude Desktop
- [ ] Works with VSCode MCP
- [ ] Works with MCP inspector
- [ ] Cross-platform (Windows, Mac, Linux)

## CI/CD Integration

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run unit tests
        run: |
          go test -v -race ./internal/...
          go test -v -race ./pkg/...
  
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      
      - name: Build
        run: make build
      
      - name: Run integration tests
        run: |
          go test -v ./test/integration/...
  
  e2e-tests:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      
      - name: Run E2E tests
        run: |
          make test-e2e
```

## Monitoring and Debugging

### 1. Debug Logging

```go
// Enable debug logging
os.Setenv("BRUMMER_MCP_DEBUG", "true")

// Add structured logging
log.WithFields(log.Fields{
    "instance_id": instanceID,
    "state":       state,
    "action":      "state_change",
}).Debug("Instance state changed")
```

### 2. Metrics Collection

```go
// Add metrics for monitoring
var (
    instancesTotal = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "brummer_hub_instances_total",
        Help: "Total number of discovered instances",
    })
    
    connectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "brummer_hub_connections_active",
        Help: "Number of active connections",
    })
    
    toolCallsTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "brummer_hub_tool_calls_total",
        Help: "Total number of tool calls",
    })
)
```

### 3. Debug Tools

```bash
#!/bin/bash
# debug/debug_hub.sh

# Start hub with verbose logging
BRUMMER_MCP_DEBUG=true brum --mcp 2>&1 | tee hub_debug.log &

# Monitor instance files
watch -n 1 'ls -la ~/.local/share/brummer/instances/'

# Test with netcat
echo '{"jsonrpc":"2.0","id":1,"method":"instances/list"}' | nc -q 1 localhost 7777

# Check goroutines
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Success Criteria

1. ✅ All unit tests pass
2. ✅ Integration tests complete successfully
3. ✅ E2E tests work with real MCP clients
4. ✅ Stress tests show no leaks
5. ✅ Performance benchmarks meet targets
6. ✅ Works on all platforms
7. ✅ Documentation complete
8. ✅ CI/CD pipeline green

## Next Steps

After verification:
1. Deploy to beta users
2. Gather feedback
3. Fix any issues found
4. Create release notes
5. Tag stable release

## Troubleshooting Guide

### Common Issues

1. **Hub doesn't see instances**
   - Check file permissions in ~/.local/share/brummer
   - Verify instances are writing files
   - Check debug logs for errors

2. **Connection failures**
   - Verify firewall allows localhost connections
   - Check instance MCP server is running
   - Look for port conflicts

3. **Tool calls fail**
   - Ensure session is connected to instance
   - Check instance has the tool
   - Verify request format

4. **Performance issues**
   - Check number of instances
   - Monitor goroutine count
   - Look for blocking operations

## Summary

This comprehensive testing plan ensures the MCP hub architecture is:
- Functionally correct
- Performant and scalable
- Reliable and robust
- Compatible with MCP ecosystem
- Ready for production use