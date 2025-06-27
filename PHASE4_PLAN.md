# Phase 4: Tool Proxying Implementation Plan

## Overview
Phase 4 implements tool proxying, allowing the MCP hub to dynamically expose tools from connected instances and forward tool calls to the appropriate instance.

## Current State
- ✅ Hub mode with stdio transport
- ✅ Connection manager with session mapping
- ✅ HubClient with all MCP methods implemented
- ✅ Instance discovery and connection monitoring
- ✅ Basic hub tools: `instances/list`, `instances/connect`

## Implementation Steps

### 1. Add Dynamic Tool Registration to Hub
**File**: `internal/mcp/streamable_server.go`
- Add methods to dynamically register/unregister tools at runtime
- Modify tool storage to support dynamic updates
- Ensure thread-safe tool registration

### 2. Implement Tool Proxy Handler
**File**: `internal/mcp/tool_proxy.go` (new)
- Create a generic proxy handler that:
  - Parses tool name to extract instance ID (format: `{instanceId}/{toolName}`)
  - Gets the connection from ConnectionManager
  - Forwards the tool call via HubClient
  - Handles streaming responses if applicable
  - Returns errors gracefully

### 3. Enhance Connection Process
**File**: `internal/mcp/connection_manager.go`
- When a session connects to an instance:
  - Call `ListTools()` on the HubClient
  - Register each tool with the hub using instance-prefixed names
  - Store tool metadata for later cleanup
- When a session disconnects:
  - Unregister all tools from that instance
  - Clean up stored metadata

### 4. Update Hub Tools
**File**: `cmd/brum/main.go` (hub tools section)
- Modify `instances/connect` to trigger tool registration
- Add optional `instances/disconnect` tool for clean disconnection
- Ensure proper error handling and status reporting

### 5. Implement Resource and Prompt Proxying
**Files**: `internal/mcp/resource_proxy.go`, `internal/mcp/prompt_proxy.go` (new)
- Similar to tool proxying but for resources and prompts
- Handle resource subscriptions for real-time updates
- Proxy prompt variables correctly

### 6. Add Session Context
**File**: `internal/mcp/session_context.go` (new)
- Track which tools belong to which session/instance
- Manage lifecycle of proxied capabilities
- Handle cleanup on disconnection

## Testing Strategy

### Unit Tests
1. Test dynamic tool registration/unregistration
2. Test tool name parsing and routing
3. Test proxy handler with mock clients
4. Test session lifecycle management

### Integration Tests
1. Full flow: instance startup → discovery → connection → tool listing → tool execution
2. Multiple instances with overlapping tool names
3. Disconnection and cleanup scenarios
4. Error cases (instance down, network issues)

## API Changes

### Tool Naming Convention
- Hub tools: `instances/*`, `hub/*`
- Proxied tools: `{instanceId}/{originalToolName}`
- Example: `myapp-8080/scripts/run`

### New Internal Methods
```go
// StreamableServer additions
func (s *StreamableServer) RegisterTool(tool Tool) error
func (s *StreamableServer) UnregisterTool(name string) error
func (s *StreamableServer) RegisterToolsFromInstance(instanceID string, tools []Tool) error
func (s *StreamableServer) UnregisterToolsFromInstance(instanceID string) error

// ConnectionManager additions  
func (cm *ConnectionManager) GetToolsForInstance(instanceID string) ([]Tool, error)
func (cm *ConnectionManager) RegisterInstanceTools(sessionID, instanceID string) error
func (cm *ConnectionManager) UnregisterInstanceTools(sessionID string) error
```

## Error Handling

1. **Instance Not Found**: Return clear error when tool references non-existent instance
2. **Connection Lost**: Handle gracefully when instance disconnects during tool execution
3. **Tool Not Found**: Distinguish between hub tools and instance tools in error messages
4. **Timeout**: Implement reasonable timeouts for tool proxying

## Performance Considerations

1. Cache tool listings to avoid repeated calls
2. Use goroutines for parallel tool registration
3. Implement connection pooling in HubClient
4. Add metrics for monitoring proxy performance

## Future Enhancements (Post-Phase 4)

1. Tool filtering/permissions
2. Load balancing across multiple instances
3. Tool aggregation (call same tool on multiple instances)
4. Advanced routing strategies