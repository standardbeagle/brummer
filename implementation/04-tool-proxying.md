# Implementation Step 4: Tool Proxying

## Overview

This step implements tool proxying, allowing the hub to forward MCP tool calls to connected instances. Once a session connects to an instance, all that instance's tools become available through the hub.

## Goals

1. Proxy tool calls from hub to connected instances
2. Dynamically expose instance tools after connection
3. Handle streaming responses properly
4. Provide clear error messages for unconnected sessions
5. Support both synchronous and streaming tools

## Technical Design

### Tool Discovery Flow

```
1. Client calls instances/connect
   └─> Hub maps session to instance
   
2. Client calls tools/list
   └─> Hub checks session mapping
   └─> If connected: fetch tools from instance
   └─> Merge with hub tools
   
3. Client calls tool
   └─> Hub checks if hub tool or instance tool
   └─> Route appropriately
```

### Proxying Architecture

```
MCP Client          Hub                Instance
    │                │                    │
    ├──tools/list───>│                    │
    │                ├──tools/list──────->│
    │                │<──tool list────────┤
    │<──merged list──┤                    │
    │                │                    │
    ├──tools/call───>│                    │
    │                ├──tools/call──────->│
    │                │<──result───────────┤
    │<──result───────┤                    │
```

## Implementation

### 1. Enhanced Hub Server

```go
// internal/mcp/hub_server.go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    
    "github.com/mark3labs/mcp-go/mcp"
)

type HubServer struct {
    connMgr       *ConnectionManager
    watcher       *InstanceWatcher
    instancesChan chan *discovery.Instance
    errorsChan    chan error
    
    // Cache of instance tools (updated periodically)
    toolsCache    map[string][]mcp.Tool // instanceID -> tools
    resourceCache map[string][]mcp.Resource
}

// Hub-specific tools that are always available
var hubTools = []mcp.Tool{
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
    {
        Name:        "instances/disconnect",
        Description: "Disconnect from the current instance",
        InputSchema: mcp.ToolInputSchema{
            Type:       "object",
            Properties: map[string]interface{}{},
        },
    },
}

func (h *HubServer) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
    // Start with hub tools
    tools := make([]mcp.Tool, len(hubTools))
    copy(tools, hubTools)
    
    // Get session ID from context
    sessionID := GetSessionID(ctx)
    if sessionID == "" {
        // No session, return hub tools only
        return &mcp.ListToolsResult{Tools: tools}, nil
    }
    
    // Get connected instance client
    client := h.connMgr.GetClient(sessionID)
    if client == nil {
        // Not connected to instance, return hub tools only
        return &mcp.ListToolsResult{Tools: tools}, nil
    }
    
    // Fetch tools from instance
    instanceTools, err := client.ListTools(ctx)
    if err != nil {
        // Log error but return hub tools
        log.Printf("Failed to list instance tools: %v", err)
        return &mcp.ListToolsResult{Tools: tools}, nil
    }
    
    // Merge instance tools
    tools = append(tools, instanceTools...)
    
    return &mcp.ListToolsResult{Tools: tools}, nil
}

func (h *HubServer) ListResources(ctx context.Context) (*mcp.ListResourcesResult, error) {
    // Similar pattern to ListTools
    resources := []mcp.Resource{}
    
    sessionID := GetSessionID(ctx)
    if sessionID == "" {
        return &mcp.ListResourcesResult{Resources: resources}, nil
    }
    
    client := h.connMgr.GetClient(sessionID)
    if client == nil {
        return &mcp.ListResourcesResult{Resources: resources}, nil
    }
    
    instanceResources, err := client.ListResources(ctx)
    if err != nil {
        log.Printf("Failed to list instance resources: %v", err)
        return &mcp.ListResourcesResult{Resources: resources}, nil
    }
    
    return &mcp.ListResourcesResult{Resources: instanceResources}, nil
}

func (h *HubServer) CallTool(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Check if it's a hub tool
    if h.isHubTool(request.Name) {
        return h.callHubTool(ctx, request)
    }
    
    // Must be an instance tool
    sessionID := GetSessionID(ctx)
    if sessionID == "" {
        return nil, fmt.Errorf("not connected to any instance")
    }
    
    client := h.connMgr.GetClient(sessionID)
    if client == nil {
        return nil, fmt.Errorf("session not connected to an instance")
    }
    
    // Proxy to instance
    return client.CallTool(ctx, request.Name, request.Arguments)
}

func (h *HubServer) isHubTool(name string) bool {
    for _, tool := range hubTools {
        if tool.Name == name {
            return true
        }
    }
    return false
}

func (h *HubServer) callHubTool(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    switch request.Name {
    case "instances/list":
        return h.handleInstancesList(ctx)
        
    case "instances/connect":
        return h.handleInstancesConnect(ctx, request.Arguments)
        
    case "instances/disconnect":
        return h.handleInstancesDisconnect(ctx)
        
    default:
        return nil, fmt.Errorf("unknown hub tool: %s", request.Name)
    }
}

func (h *HubServer) handleInstancesList(ctx context.Context) (*mcp.CallToolResult, error) {
    instances := h.connMgr.ListInstances()
    
    var output []map[string]interface{}
    for _, info := range instances {
        output = append(output, map[string]interface{}{
            "id":               info.InstanceID,
            "name":             info.Name,
            "path":             info.Path,
            "port":             info.Port,
            "pid":              info.PID,
            "state":            info.State.String(),
            "connected_at":     info.ConnectedAt,
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
}

func (h *HubServer) handleInstancesConnect(ctx context.Context, args json.RawMessage) (*mcp.CallToolResult, error) {
    var params struct {
        InstanceID string `json:"instance_id"`
    }
    if err := json.Unmarshal(args, &params); err != nil {
        return nil, fmt.Errorf("invalid arguments: %w", err)
    }
    
    sessionID := GetSessionID(ctx)
    if sessionID == "" {
        return nil, fmt.Errorf("no session ID in context")
    }
    
    // Connect session to instance
    if err := h.connMgr.ConnectSession(sessionID, params.InstanceID); err != nil {
        return nil, err
    }
    
    // Get instance info
    instances := h.connMgr.ListInstances()
    var instanceName string
    for _, info := range instances {
        if info.InstanceID == params.InstanceID {
            instanceName = info.Name
            break
        }
    }
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{{
            Type: "text",
            Text: fmt.Sprintf("Connected to instance '%s' (%s)", instanceName, params.InstanceID),
        }},
    }, nil
}

func (h *HubServer) handleInstancesDisconnect(ctx context.Context) (*mcp.CallToolResult, error) {
    sessionID := GetSessionID(ctx)
    if sessionID == "" {
        return nil, fmt.Errorf("no session ID in context")
    }
    
    if err := h.connMgr.DisconnectSession(sessionID); err != nil {
        return nil, err
    }
    
    return &mcp.CallToolResult{
        Content: []mcp.Content{{
            Type: "text",
            Text: "Disconnected from instance",
        }},
    }, nil
}

// Resource proxying
func (h *HubServer) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
    sessionID := GetSessionID(ctx)
    if sessionID == "" {
        return nil, fmt.Errorf("not connected to any instance")
    }
    
    client := h.connMgr.GetClient(sessionID)
    if client == nil {
        return nil, fmt.Errorf("session not connected to an instance")
    }
    
    return client.ReadResource(ctx, uri)
}

// Prompt proxying
func (h *HubServer) ListPrompts(ctx context.Context) (*mcp.ListPromptsResult, error) {
    sessionID := GetSessionID(ctx)
    if sessionID == "" {
        return &mcp.ListPromptsResult{Prompts: []mcp.Prompt{}}, nil
    }
    
    client := h.connMgr.GetClient(sessionID)
    if client == nil {
        return &mcp.ListPromptsResult{Prompts: []mcp.Prompt{}}, nil
    }
    
    return client.ListPrompts(ctx)
}

func (h *HubServer) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
    sessionID := GetSessionID(ctx)
    if sessionID == "" {
        return nil, fmt.Errorf("not connected to any instance")
    }
    
    client := h.connMgr.GetClient(sessionID)
    if client == nil {
        return nil, fmt.Errorf("session not connected to an instance")
    }
    
    return client.GetPrompt(ctx, name, args)
}
```

### 2. Enhanced Hub Client

```go
// internal/mcp/hub_client.go

// Add these methods to HubClient

func (c *HubClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(),
        "method":  "tools/list",
    }
    
    result, err := c.sendRequest(ctx, request)
    if err != nil {
        return nil, err
    }
    
    var response struct {
        Tools []mcp.Tool `json:"tools"`
    }
    if err := json.Unmarshal(result, &response); err != nil {
        return nil, err
    }
    
    return response.Tools, nil
}

func (c *HubClient) CallTool(ctx context.Context, name string, args json.RawMessage) (*mcp.CallToolResult, error) {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(),
        "method":  "tools/call",
        "params": map[string]interface{}{
            "name":      name,
            "arguments": args,
        },
    }
    
    result, err := c.sendRequest(ctx, request)
    if err != nil {
        return nil, err
    }
    
    var response mcp.CallToolResult
    if err := json.Unmarshal(result, &response); err != nil {
        return nil, err
    }
    
    return &response, nil
}

func (c *HubClient) ListResources(ctx context.Context) ([]mcp.Resource, error) {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(),
        "method":  "resources/list",
    }
    
    result, err := c.sendRequest(ctx, request)
    if err != nil {
        return nil, err
    }
    
    var response struct {
        Resources []mcp.Resource `json:"resources"`
    }
    if err := json.Unmarshal(result, &response); err != nil {
        return nil, err
    }
    
    return response.Resources, nil
}

func (c *HubClient) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(),
        "method":  "resources/read",
        "params": map[string]interface{}{
            "uri": uri,
        },
    }
    
    result, err := c.sendRequest(ctx, request)
    if err != nil {
        return nil, err
    }
    
    var response mcp.ReadResourceResult
    if err := json.Unmarshal(result, &response); err != nil {
        return nil, err
    }
    
    return &response, nil
}

func (c *HubClient) ListPrompts(ctx context.Context) (*mcp.ListPromptsResult, error) {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(),
        "method":  "prompts/list",
    }
    
    result, err := c.sendRequest(ctx, request)
    if err != nil {
        return nil, err
    }
    
    var response mcp.ListPromptsResult
    if err := json.Unmarshal(result, &response); err != nil {
        return nil, err
    }
    
    return &response, nil
}

func (c *HubClient) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(),
        "method":  "prompts/get",
        "params": map[string]interface{}{
            "name":      name,
            "arguments": args,
        },
    }
    
    result, err := c.sendRequest(ctx, request)
    if err != nil {
        return nil, err
    }
    
    var response mcp.GetPromptResult
    if err := json.Unmarshal(result, &response); err != nil {
        return nil, err
    }
    
    return &response, nil
}
```

### 3. Session Context Management

```go
// internal/mcp/session.go
package mcp

import (
    "context"
)

type contextKey string

const sessionIDKey contextKey = "mcp-session-id"

// GetSessionID extracts session ID from context
func GetSessionID(ctx context.Context) string {
    if id, ok := ctx.Value(sessionIDKey).(string); ok {
        return id
    }
    return ""
}

// WithSessionID adds session ID to context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
    return context.WithValue(ctx, sessionIDKey, sessionID)
}
```

### 4. Streaming Support

```go
// internal/mcp/streaming.go
package mcp

import (
    "bufio"
    "context"
    "encoding/json"
    "io"
    "net/http"
)

// StreamingResult handles SSE responses
type StreamingResult struct {
    Events <-chan StreamEvent
    Errors <-chan error
    Cancel func()
}

type StreamEvent struct {
    Type string
    Data json.RawMessage
}

func (c *HubClient) CallToolStreaming(ctx context.Context, name string, args json.RawMessage) (*StreamingResult, error) {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      time.Now().UnixNano(),
        "method":  "tools/call",
        "params": map[string]interface{}{
            "name":      name,
            "arguments": args,
        },
    }
    
    body, _ := json.Marshal(request)
    req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "text/event-stream")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    
    if resp.StatusCode != http.StatusOK {
        resp.Body.Close()
        return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
    }
    
    events := make(chan StreamEvent, 100)
    errors := make(chan error, 1)
    
    ctx, cancel := context.WithCancel(ctx)
    
    go func() {
        defer close(events)
        defer close(errors)
        defer resp.Body.Close()
        
        scanner := bufio.NewScanner(resp.Body)
        var eventType string
        var eventData []byte
        
        for scanner.Scan() {
            line := scanner.Text()
            
            if strings.HasPrefix(line, "event: ") {
                eventType = strings.TrimPrefix(line, "event: ")
            } else if strings.HasPrefix(line, "data: ") {
                eventData = []byte(strings.TrimPrefix(line, "data: "))
            } else if line == "" && eventType != "" {
                // End of event
                select {
                case events <- StreamEvent{Type: eventType, Data: eventData}:
                case <-ctx.Done():
                    return
                }
                eventType = ""
                eventData = nil
            }
        }
        
        if err := scanner.Err(); err != nil {
            select {
            case errors <- err:
            case <-ctx.Done():
            }
        }
    }()
    
    return &StreamingResult{
        Events: events,
        Errors: errors,
        Cancel: cancel,
    }, nil
}
```

## Testing Plan

### 1. Unit Tests

```go
// internal/mcp/hub_server_test.go
func TestToolProxying(t *testing.T) {
    // Setup
    hub := createTestHub()
    instance := createTestInstance()
    
    // Connect session to instance
    sessionID := "test-session"
    err := hub.connMgr.ConnectSession(sessionID, instance.ID)
    require.NoError(t, err)
    
    // List tools should include instance tools
    ctx := WithSessionID(context.Background(), sessionID)
    result, err := hub.ListTools(ctx)
    require.NoError(t, err)
    
    // Should have hub tools + instance tools
    assert.True(t, len(result.Tools) > len(hubTools))
    
    // Should be able to call instance tool
    toolResult, err := hub.CallTool(ctx, &mcp.CallToolRequest{
        Name: "scripts/list",
    })
    require.NoError(t, err)
    assert.NotNil(t, toolResult)
}

func TestDisconnectedSession(t *testing.T) {
    hub := createTestHub()
    
    // Without connection, only hub tools available
    ctx := context.Background()
    result, err := hub.ListTools(ctx)
    require.NoError(t, err)
    assert.Len(t, result.Tools, len(hubTools))
    
    // Calling instance tool should fail
    _, err = hub.CallTool(ctx, &mcp.CallToolRequest{
        Name: "scripts/list",
    })
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not connected")
}
```

### 2. Integration Test

```bash
#!/bin/bash
# test_tool_proxying.sh

# Start instance
cd test-project
brum --no-tui &
INSTANCE_PID=$!
sleep 2

# Start hub
brum --mcp > hub.log 2>&1 &
HUB_PID=$!
sleep 1

# Connect to instance
INSTANCE_ID=$(echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"instances/list"}}' | \
    nc -q 1 localhost 7777 | jq -r '.result.content[0].text' | jq -r '.[0].id')

echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"instances/connect","arguments":{"instance_id":"'$INSTANCE_ID'"}}}' | \
    nc -q 1 localhost 7777

# Now list tools - should include instance tools
TOOLS=$(echo '{"jsonrpc":"2.0","id":3,"method":"tools/list"}' | nc -q 1 localhost 7777)
echo "$TOOLS" | grep -q "scripts/list"

# Call instance tool through hub
echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"scripts/list"}}' | \
    nc -q 1 localhost 7777

# Cleanup
kill $INSTANCE_PID $HUB_PID
```

### 3. Streaming Test

```go
func TestStreamingProxy(t *testing.T) {
    // Test that streaming responses are properly proxied
    // Verify events flow through correctly
    // Check cancellation works
}
```

## Success Criteria

1. ✅ Hub tools always available
2. ✅ Instance tools available after connection
3. ✅ Tool calls properly proxied
4. ✅ Error messages for unconnected sessions
5. ✅ Resource and prompt proxying works
6. ✅ Streaming responses handled correctly
7. ✅ Session context properly managed
8. ✅ Clean disconnect removes instance tools

## Edge Cases

### 1. Instance Goes Down
- Tool calls should fail gracefully
- tools/list should update on next call

### 2. Multiple Sessions
- Each session can connect to different instance
- Tools isolated per session

### 3. Tool Name Conflicts
- Hub tools take precedence
- Document naming conventions

### 4. Large Responses
- Stream if possible
- Set reasonable size limits

### 5. Slow Instance
- Timeout on tool calls
- Return error to client

## Security Considerations

1. **Input Validation**
   - Validate all tool arguments
   - Prevent injection attacks

2. **Session Isolation**
   - Sessions can't access other sessions' instances
   - Clear session data on disconnect

3. **Resource Limits**
   - Limit concurrent connections
   - Timeout long-running operations

## Next Steps

1. Step 5: Add health monitoring with MCP ping
2. Step 6: End-to-end testing

## Code Checklist

- [ ] Update hub server with proxying logic
- [ ] Add all MCP methods to hub client
- [ ] Implement session context management
- [ ] Add streaming support
- [ ] Create comprehensive tests
- [ ] Handle all error cases
- [ ] Document tool naming conventions
- [ ] Test with real MCP client