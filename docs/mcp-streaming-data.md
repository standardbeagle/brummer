# MCP Streaming Data Documentation

This document describes what data is streamed through Brummer's MCP Server-Sent Events (SSE) connections.

## Overview

Brummer's MCP server streams real-time data to connected clients using SSE. There are two types of streaming:
1. **Event Notifications** - Automatic broadcasts when events occur
2. **Resource Updates** - Updates for subscribed resources

## 1. Event Notifications (Broadcast to All Clients)

These are automatically sent to all connected SSE clients when events occur:

### Process Events
- **`notifications/process/started`** - When a process starts
  ```json
  {
    "processId": "dev-1234567890",
    "processName": "dev",
    "command": "npm run dev",
    "pid": 12345,
    "startTime": "2024-01-20T10:30:00Z"
  }
  ```

- **`notifications/process/exited`** - When a process exits
  ```json
  {
    "processId": "dev-1234567890",
    "processName": "dev",
    "exitCode": 0,
    "signal": null,
    "duration": 45000
  }
  ```

### Log Events
- **`notifications/logs/new`** - New log line from any process
  ```json
  {
    "id": "log-1234567890",
    "processId": "dev-1234567890",
    "processName": "dev",
    "content": "[vite] ready in 523ms",
    "timestamp": "2024-01-20T10:30:01Z",
    "isError": false,
    "tags": [],
    "priority": 0
  }
  ```

### Error Events
- **`notifications/error/detected`** - When an error is detected in logs
  ```json
  {
    "processId": "dev-1234567890",
    "error": {
      "message": "TypeError: Cannot read property 'foo' of undefined",
      "file": "src/App.tsx",
      "line": 42,
      "column": 15,
      "stack": "..."
    }
  }
  ```

### REPL Events
- **REPL Response** (internal) - Responses from browser JavaScript execution

## 2. Resource Updates (Subscription-Based)

Clients must subscribe to these resources to receive updates:

### Log Resources
- **`logs://recent`** - Recent log entries (last 100)
  - Updated when: New logs are added
  - Contains: Array of recent log entries

- **`logs://errors`** - Error logs only (last 50)
  - Updated when: New error logs are detected
  - Contains: Array of error log entries

### Process Resources
- **`processes://active`** - Currently running processes
  - Updated when: Process starts or exits
  - Contains: Array of active process information

### Telemetry Resources (Browser Extension)
- **`telemetry://sessions`** - Active browser telemetry sessions
- **`telemetry://errors`** - JavaScript errors from browser
- **`telemetry://console-errors`** - Console error outputs

### Proxy Resources
- **`proxy://requests`** - Recent HTTP requests (last 100)
- **`proxy://mappings`** - Active reverse proxy URL mappings

### Script Resources
- **`scripts://available`** - Available npm/yarn/pnpm scripts

## 3. System Events

### Heartbeat
- **`ping`** event - Sent every 30 seconds to keep connection alive
  ```json
  {
    "timestamp": "2024-01-20T10:30:00Z"
  }
  ```

### Session Info
- Initial SSE comments when connection established:
  ```
  : MCP Streamable HTTP Transport
  : Session-Id: abc123-def456-ghi789
  ```

## 4. Streaming Tool Responses

Some MCP tools support streaming responses:

### `logs/stream`
- Streams real-time logs as they occur
- Can be filtered by process, level, or pattern

### `scripts/run`
- Streams command output in real-time as script executes

## Usage Example

```javascript
// Connect to SSE endpoint
const eventSource = new EventSource('http://localhost:7777/mcp', {
  headers: {
    'Accept': 'text/event-stream',
    'Mcp-Session-Id': 'my-session-123'
  }
});

// Listen for all messages
eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch(data.method) {
    case 'notifications/logs/new':
      console.log('New log:', data.params);
      break;
    case 'notifications/process/started':
      console.log('Process started:', data.params);
      break;
    case 'notifications/resources/updated':
      console.log('Resource updated:', data.params.uri);
      break;
  }
};

// Subscribe to resources via POST
fetch('http://localhost:7777/mcp', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Mcp-Session-Id': 'my-session-123'
  },
  body: JSON.stringify({
    jsonrpc: '2.0',
    id: 1,
    method: 'resources/subscribe',
    params: { uri: 'logs://recent' }
  })
});
```

## Data Flow

1. **Brummer Process** → Generates logs/events
2. **Event Bus** → Distributes events internally
3. **MCP Server** → Converts to JSON-RPC notifications
4. **SSE Connection** → Streams to connected clients
5. **Client** → Receives and processes real-time updates

This streaming architecture allows clients to receive real-time updates about:
- Process lifecycle (start/stop)
- Log output from all processes
- Error detection and context
- HTTP proxy traffic
- Resource changes

The combination of broadcast events and subscription-based updates provides flexibility for different use cases while maintaining efficient network usage.