# MCP Activity Tracking Implementation

## Overview
Updated the MCP diagnostics tab in brummer to show actual MCP activity instead of mock data. The implementation tracks all MCP connections and requests in real-time.

## Changes Made

### 1. Event System Updates (`pkg/events/events.go`)
- Added new event types:
  - `MCPActivity`: Tracks MCP request/response activity
  - `MCPConnected`: Tracks new MCP client connections
  - `MCPDisconnected`: Tracks MCP client disconnections

### 2. MCP Server Activity Tracking (`internal/mcp/streamable_server.go`)
- Modified `processMessage()` to track all MCP activity:
  - Records method, params, response, error, and duration
  - Publishes `MCPActivity` event for each request
- Modified `handleStreamingConnection()` to track connections:
  - Publishes `MCPConnected` event when SSE connection established
  - Publishes `MCPDisconnected` event when connection closes
  - Includes session ID and client info (User-Agent)

### 3. TUI Model Updates (`internal/tui/model.go`)
- Added fields to track MCP data:
  - `mcpConnections`: Map of session ID to connection info
  - `mcpActivities`: Map of session ID to activity history
  - `mcpActivityMu`: Mutex for thread-safe access
- Added message types:
  - `mcpActivityMsg`: Carries MCP activity data
  - `mcpConnectionMsg`: Carries connection/disconnection events
- Added event handlers:
  - `handleMCPConnection()`: Updates connection tracking
  - `handleMCPActivity()`: Records activity and updates stats
- Updated view switching to refresh MCP data when entering MCP view

### 4. MCP Connections View (`internal/tui/mcp_connections.go`)
- Updated `updateMCPConnectionsList()` to use real connection data
- Updated `updateMCPActivityView()` to display real activity logs
- Enhanced formatting:
  - Shows timestamp with milliseconds
  - Shows method names and duration
  - Shows params and responses (truncated if too long)
  - Uses arrows (→ ←) for better visual flow
  - Shows error messages in red

## How It Works

1. **Connection Tracking**:
   - When a client connects via SSE, a `MCPConnected` event is published
   - Connection info includes session ID, timestamp, and client User-Agent
   - Client name is extracted from User-Agent (e.g., "Claude Desktop", "VS Code MCP")

2. **Activity Tracking**:
   - Every MCP request/response is tracked via `MCPActivity` events
   - Activity includes method, parameters, response/error, and duration
   - Activities are stored per session (last 100 per session)

3. **TUI Display**:
   - Left panel shows active/inactive connections with stats
   - Right panel shows activity log for selected connection
   - Activity shown in reverse chronological order (newest first)
   - Real-time updates as new activity occurs

## Usage

1. Start brummer in debug mode to enable MCP diagnostics:
   ```bash
   brum --debug
   ```

2. Navigate to the MCP view:
   - Press Tab or arrow keys to cycle through views
   - Or press 7 (if using number keys)

3. Select a connection to view its activity:
   - Use arrow keys to navigate connections
   - Press Enter to select and view activity

## Testing

A test was added to verify activity tracking:
- `internal/mcp/activity_test.go`: Tests that events are properly published

The implementation has been tested with:
- Multiple concurrent MCP requests
- SSE streaming connections
- Various MCP methods (initialize, tools/list, tools/call, etc.)