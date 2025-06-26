# Implementation Step 1: Stdio Hub Foundation

## Overview

This step implements the core MCP hub server that runs exclusively over stdio transport. The hub will start instantly, respond to MCP protocol messages, and provide the foundation for instance discovery and management.

## Goals

1. Implement `--mcp` flag to run hub mode
2. Create stdio-only MCP server using mcp-go library
3. Implement basic hub tools (instances/list as placeholder)
4. Ensure instant startup and MCP ping compliance
5. Clean separation from instance mode

## Technical Design

### Command Line Changes

```go
// cmd/brum/main.go
var (
    mcpMode bool  // New flag for hub mode
)

func init() {
    rootCmd.Flags().BoolVar(&mcpMode, "mcp", false, 
        "Run as MCP hub for instance discovery (stdio only)")
}
```

### Hub Mode Detection

```go
// In runApp function
if mcpMode {
    // Hub mode - stdio only for MCP clients
    runMCPHub()
    return
}

if noTUI {
    // Instance mode - HTTP server for single project
    runHeadlessInstance()
    return
}
```

### Hub Server Structure

```go
// internal/mcp/hub_server.go
package mcp

import (
    "context"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/transport/stdio"
)

type HubServer struct {
    connMgr *ConnectionManager
    // Will add instance discovery in step 2
}

func NewHubServer() *HubServer {
    return &HubServer{
        connMgr: NewConnectionManager(),
    }
}

// Implement mcp.Server interface
func (h *HubServer) Initialize(ctx context.Context) (*mcp.InitializeResult, error) {
    return &mcp.InitializeResult{
        ProtocolVersion: "1.0",
        ServerInfo: &mcp.ServerInfo{
            Name:    "brummer-hub",
            Version: Version,
        },
        Capabilities: &mcp.ServerCapabilities{
            Tools: &mcp.ToolsServerCapabilities{},
        },
    }, nil
}

func (h *HubServer) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
    return &mcp.ListToolsResult{
        Tools: []mcp.Tool{
            {
                Name:        "instances/list",
                Description: "List all running brummer instances",
                InputSchema: mcp.ToolInputSchema{
                    Type:       "object",
                    Properties: map[string]interface{}{},
                },
            },
            {
                Name:        "instances/connect",
                Description: "Connect to a specific brummer instance",
                InputSchema: mcp.ToolInputSchema{
                    Type:       "object",
                    Properties: map[string]interface{}{
                        "instance_id": map[string]interface{}{
                            "type":        "string",
                            "description": "The ID of the instance to connect to",
                        },
                    },
                    Required: []string{"instance_id"},
                },
            },
        },
    }, nil
}

func (h *HubServer) CallTool(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    switch request.Name {
    case "instances/list":
        // Placeholder - will implement in step 2
        return &mcp.CallToolResult{
            Content: []mcp.Content{{
                Type: "text",
                Text: "[]", // Empty list for now
            }},
        }, nil
        
    case "instances/connect":
        // Placeholder - will implement in step 4
        return &mcp.CallToolResult{
            Content: []mcp.Content{{
                Type: "text",  
                Text: "Not implemented yet",
            }},
        }, nil
        
    default:
        return nil, fmt.Errorf("unknown tool: %s", request.Name)
    }
}
```

### Main Hub Runner

```go
// cmd/brum/main.go
func runMCPHub() {
    // Create hub server
    hubServer := mcp.NewHubServer()
    
    // CRITICAL: Use stdio transport only
    transport := stdio.NewTransport()
    
    // Create MCP server
    server := mcp.NewServer(hubServer)
    
    // Start server
    if err := server.Serve(transport); err != nil {
        // Log to stderr to avoid corrupting stdio protocol
        fmt.Fprintf(os.Stderr, "Hub server error: %v\n", err)
        os.Exit(1)
    }
}
```

## Implementation Steps

### 1. Add Hub Mode Flag

```diff
// cmd/brum/main.go
var (
    // ... existing vars ...
+   mcpMode bool  // Run as MCP hub for instance discovery
)

func init() {
    // ... existing flags ...
+   rootCmd.Flags().BoolVar(&mcpMode, "mcp", false, 
+       "Run as MCP hub for instance discovery (stdio only)")
}
```

### 2. Update Help Text

```diff
Long: `...existing help text...

+Hub Mode (for MCP clients):
+  brum --mcp                    # Run as hub for discovering instances
+                                # Uses stdio transport for MCP clients
+                                # Allows single connection to control multiple projects
`
```

### 3. Implement Hub Mode Detection

```diff
func runApp(cmd *cobra.Command, args []string) {
    if showVersion {
        fmt.Printf("brummer version %s\n", Version)
        return
    }
    
+   // Hub mode - stdio MCP server for instance discovery
+   if mcpMode {
+       runMCPHub()
+       return
+   }
    
    // ... rest of existing runApp ...
}
```

### 4. Create Hub Server File

Create `internal/mcp/hub_server.go` with the implementation above.

### 5. Create Stub Connection Manager

```go
// internal/mcp/connection_manager_stub.go
// Temporary stub - will be replaced in step 3
package mcp

type ConnectionManager struct{}

func NewConnectionManager() *ConnectionManager {
    return &ConnectionManager{}
}
```

### 6. Update MCP Imports

```diff
// go.mod
require (
    github.com/mark3labs/mcp-go v0.1.0
    // ... other deps ...
)
```

## Testing Plan

### 1. Manual Testing

```bash
# Test hub startup
brum --mcp

# In another terminal, test with MCP inspector
npm install -g @modelcontextprotocol/inspector
mcp-inspector stdio -- brum --mcp

# Verify:
# - Server info shows "brummer-hub"
# - Tools list shows instances/list and instances/connect
# - Calling instances/list returns empty array
# - No errors on startup
```

### 2. Unit Tests

```go
// internal/mcp/hub_server_test.go
func TestHubServerInitialize(t *testing.T) {
    server := NewHubServer()
    result, err := server.Initialize(context.Background())
    
    assert.NoError(t, err)
    assert.Equal(t, "brummer-hub", result.ServerInfo.Name)
    assert.Equal(t, "1.0", result.ProtocolVersion)
}

func TestHubServerListTools(t *testing.T) {
    server := NewHubServer()
    result, err := server.ListTools(context.Background())
    
    assert.NoError(t, err)
    assert.Len(t, result.Tools, 2)
    assert.Equal(t, "instances/list", result.Tools[0].Name)
    assert.Equal(t, "instances/connect", result.Tools[1].Name)
}
```

### 3. Integration Test Script

```bash
#!/bin/bash
# test_hub_startup.sh

# Start hub in background
brum --mcp &
HUB_PID=$!

# Give it time to start
sleep 0.1

# Send initialize request
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1.0"}}' | \
    nc -U /proc/$HUB_PID/fd/0

# Check if response contains brummer-hub
# ... parse and verify response ...

# Cleanup
kill $HUB_PID
```

## Success Criteria

1. ✅ `brum --mcp` starts without errors
2. ✅ Responds to MCP initialize request
3. ✅ Lists hub-specific tools
4. ✅ Returns empty instance list (placeholder)
5. ✅ Uses stdio transport exclusively
6. ✅ Starts in < 100ms
7. ✅ No output to stdout except MCP protocol
8. ✅ Errors go to stderr only

## Common Issues & Solutions

### Issue 1: Output Corruption
**Problem**: Debug output corrupts stdio protocol  
**Solution**: All logging must go to stderr or be disabled

### Issue 2: Slow Startup
**Problem**: Hub takes too long to start  
**Solution**: Defer any heavy initialization, start minimal server first

### Issue 3: Transport Confusion
**Problem**: Accidentally using HTTP transport  
**Solution**: Explicitly check mcpMode flag, never create HTTP transport

## Next Steps

After this foundation is working:
1. Step 2: Add instance discovery file watching
2. Step 3: Implement connection manager
3. Step 4: Add tool proxying to instances

## Code Checklist

- [ ] Add `--mcp` flag to command line
- [ ] Update help text with hub mode info
- [ ] Create `runMCPHub()` function
- [ ] Create `hub_server.go` with MCP implementation
- [ ] Ensure stdio-only transport
- [ ] Add unit tests
- [ ] Test with MCP inspector
- [ ] Verify < 100ms startup time
- [ ] Update documentation