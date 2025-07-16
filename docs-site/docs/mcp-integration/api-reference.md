---
sidebar_position: 2
---

# MCP Tools Reference

Complete reference for all Brummer Model Context Protocol (MCP) tools, resources, and prompts.

## Overview

Brummer provides a comprehensive MCP server implementing the official JSON-RPC 2.0 protocol with streaming support. The server exposes multiple tools for script management, log monitoring, proxy analysis, browser automation, and more.

### Connection Details

- **Protocol**: JSON-RPC 2.0 with Server-Sent Events
- **Primary Endpoint**: `http://localhost:7777/mcp`
- **Default Port**: 7777 (configurable with `-p` flag)
- **Server Name**: `brummer-mcp`
- **Protocol Version**: `2024-11-05`

## Available Tools

### Script Management

#### `scripts_list`

List all available npm/yarn/pnpm/bun scripts from package.json.

**Parameters**: None

**Response**:
```json
{
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "test": "jest",
    "lint": "eslint ."
  }
}
```

**Example**:
```javascript
const result = await mcp.call('scripts_list');
console.log(result.scripts);
```

---

#### `scripts_run`

Execute a package.json script with real-time output streaming.

**Parameters**:
- `name` (string, required): Script name to execute

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "description": "The name of the script to run"
    }
  },
  "required": ["name"]
}
```

**Response** (Non-streaming):
```json
{
  "processId": "dev-1704804000",
  "name": "dev",
  "script": "next dev",
  "status": "running"
}
```

**Streaming Response**:
```json
{"type": "started", "processId": "dev-1704804000", "name": "dev", "script": "next dev"}
{"type": "log", "line": "ready - started server on 0.0.0.0:3000"}
{"type": "log", "line": "Local: http://localhost:3000"}
{"processId": "dev-1704804000", "status": "running", "exitCode": null}
```

**Streaming**: ✅ Yes

---

#### `scripts_stop`

Stop a running script process.

**Parameters**:
- `processId` (string, required): Process ID to stop

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "processId": {
      "type": "string",
      "description": "The process ID to stop"
    }
  },
  "required": ["processId"]
}
```

**Response**:
```json
{
  "success": true,
  "processId": "dev-1704804000"
}
```

---

#### `scripts_status`

Check the status of running scripts.

**Parameters**:
- `name` (string, optional): Specific script name to check

**Response** (Single process):
```json
{
  "processId": "dev-1704804000",
  "name": "dev",
  "status": "running",
  "startTime": "2024-01-09T10:00:00Z",
  "uptime": "1h23m45s"
}
```

**Response** (All processes):
```json
[
  {
    "processId": "dev-1704804000",
    "name": "dev",
    "status": "running",
    "startTime": "2024-01-09T10:00:00Z",
    "uptime": "1h23m45s"
  }
]
```

### Log Management

#### `logs_stream`

Stream real-time logs from running processes with filtering support.

**Parameters**:
- `processId` (string, optional): Filter by process ID
- `level` (string, optional): Log level filter ("all", "error", "warn", "info")
- `follow` (boolean, optional, default: true): Stream new logs
- `limit` (integer, optional, default: 100): Historical log count

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "processId": {"type": "string"},
    "level": {"type": "string", "enum": ["all", "error", "warn", "info"]},
    "follow": {"type": "boolean", "default": true},
    "limit": {"type": "integer", "default": 100}
  }
}
```

**Log Entry Schema**:
```json
{
  "id": "log_12345",
  "timestamp": "2024-01-09T10:00:00Z",
  "processId": "dev-1704804000",
  "processName": "dev",
  "content": "Server started on port 3000",
  "level": "info",
  "isError": false,
  "tags": ["server", "startup"],
  "priority": 1
}
```

**Streaming Response**:
```json
{"type": "log", "data": {...log_entry}}
{"type": "log", "data": {...log_entry}}
{"count": 150, "timedOut": false}
```

**Streaming**: ✅ Yes

---

#### `logs_search`

Search through historical logs using text or regex patterns.

**Parameters**:
- `query` (string, required): Search query
- `regex` (boolean, optional, default: false): Use regex matching
- `level` (string, optional): Filter by log level
- `processId` (string, optional): Filter by process
- `since` (string, optional): ISO 8601 timestamp
- `limit` (integer, optional, default: 100): Max results

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "query": {"type": "string"},
    "regex": {"type": "boolean", "default": false},
    "level": {"type": "string", "enum": ["all", "error", "warn", "info"]},
    "processId": {"type": "string"},
    "since": {"type": "string", "format": "date-time"},
    "limit": {"type": "integer", "default": 100}
  },
  "required": ["query"]
}
```

**Response**: Array of log entries matching the search criteria.

**Example**:
```javascript
const results = await mcp.call('logs_search', {
  query: "error",
  level: "error",
  limit: 50
});
```

### Proxy & Telemetry

#### `proxy_requests`

Get HTTP requests captured by the proxy server.

**Parameters**:
- `processName` (string, optional): Filter by process
- `status` (string, optional): Filter by status ("all", "success", "error")
- `limit` (integer, optional, default: 100): Max requests

**Request Object Schema**:
```json
{
  "URL": "http://localhost:3000/api/users",
  "Method": "GET",
  "StatusCode": 200,
  "Duration": "123ms",
  "Timestamp": "2024-01-09T10:00:00Z",
  "ProcessName": "dev",
  "Headers": {
    "Content-Type": "application/json"
  },
  "Body": "{\"users\": []}"
}
```

---

#### `telemetry_sessions`

Get browser telemetry sessions with performance metrics.

**Parameters**:
- `processName` (string, optional): Filter by process
- `sessionId` (string, optional): Get specific session
- `limit` (integer, optional, default: 10): Max sessions

**Session Schema**:
```json
{
  "sessionId": "session_abc123",
  "url": "http://localhost:3000",
  "startTime": "2024-01-09T10:00:00Z",
  "duration": "5m30s",
  "pageViews": 3,
  "errors": 1,
  "warnings": 2,
  "performance": {
    "loadTime": "1.2s",
    "memoryUsage": "45MB",
    "fps": 58
  }
}
```

---

#### `telemetry_events`

Stream real-time telemetry events from the browser.

**Parameters**:
- `sessionId` (string, optional): Filter by session
- `eventType` (string, optional): Event type filter ("all", "error", "console", "performance", "interaction")
- `follow` (boolean, optional, default: true): Stream new events
- `limit` (integer, optional, default: 50): Historical events

**Telemetry Event Schema**:
```json
{
  "sessionId": "session_abc123",
  "timestamp": "2024-01-09T10:00:00Z",
  "type": "javascript_error",
  "data": {
    "message": "TypeError: Cannot read property 'name' of undefined",
    "stack": "...",
    "filename": "app.js",
    "line": 42
  },
  "url": "http://localhost:3000/dashboard"
}
```

**Streaming**: ✅ Yes

### Browser Automation

#### `browser_open`

Open a URL in the default browser with automatic proxy configuration.

**Parameters**:
- `url` (string, required): URL to open
- `processName` (string, optional): Associate with process

**Response**:
```json
{
  "originalUrl": "http://localhost:3000",
  "proxyUrl": "http://localhost:20888",
  "opened": true
}
```

**Cross-platform support**: Windows, Mac, Linux, WSL2

---

#### `browser_refresh`

Send refresh command to connected browser tabs.

**Parameters**:
- `sessionId` (string, optional): Specific session to refresh

**Response**:
```json
{
  "sent": true
}
```

---

#### `browser_navigate`

Navigate browser tabs to a different URL.

**Parameters**:
- `url` (string, required): URL or path to navigate to
- `sessionId` (string, optional): Specific session

**Response**:
```json
{
  "sent": true,
  "url": "/new-page"
}
```

---

#### `browser_screenshot`

Capture screenshots of browser tabs for debugging, documentation, and visual testing.

**Parameters**:
- `sessionId` (string, optional): Session to capture from (defaults to most recent)
- `format` (string, optional): Image format ("png", "jpeg", "webp", default: "png")
- `quality` (integer, optional): Quality for jpeg/webp formats (0-100, default: 90)
- `fullPage` (boolean, optional): Capture full page instead of viewport (default: false)
- `selector` (string, optional): CSS selector to capture specific element

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "sessionId": {
      "type": "string",
      "description": "Session to capture screenshot from (defaults to most recent)"
    },
    "format": {
      "type": "string",
      "enum": ["png", "jpeg", "webp"],
      "default": "png",
      "description": "Image format for the screenshot"
    },
    "quality": {
      "type": "integer",
      "minimum": 0,
      "maximum": 100,
      "default": 90,
      "description": "Quality for jpeg/webp formats (0-100)"
    },
    "fullPage": {
      "type": "boolean",
      "default": false,
      "description": "Capture full page instead of just viewport"
    },
    "selector": {
      "type": "string",
      "description": "CSS selector to capture specific element"
    }
  }
}
```

**Response**:
```json
{
  "success": true,
  "screenshot": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
  "format": "png",
  "dimensions": {
    "width": 1920,
    "height": 1080
  }
}
```

**Features**:
- Multiple image formats (PNG, JPEG, WebP)
- Full-page capture including scrolled content
- Element-specific capture with CSS selectors
- High-DPI display support with proper scaling
- Cross-platform compatibility

**Example**:
```javascript
// Standard viewport screenshot
await mcp.call('browser_screenshot', {
  format: "png"
});

// Full page capture for documentation
await mcp.call('browser_screenshot', {
  format: "png",
  fullPage: true
});

// Capture specific component
await mcp.call('browser_screenshot', {
  selector: "#main-content",
  format: "png"
});

// High-quality JPEG for photo content
await mcp.call('browser_screenshot', {
  format: "jpeg",
  quality: 95
});
```

### JavaScript REPL

#### `repl_execute`

Execute JavaScript code in the browser context with async/await support.

**Parameters**:
- `code` (string, required): JavaScript code to execute
- `sessionId` (string, optional): Target session (defaults to most recent)

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "code": {"type": "string"},
    "sessionId": {"type": "string"}
  },
  "required": ["code"]
}
```

**Success Response**:
```json
{
  "result": "42",
  "type": "number",
  "error": null
}
```

**Error Response**:
```json
{
  "error": "ReferenceError: undefinedVariable is not defined",
  "stack": "ReferenceError: undefinedVariable is not defined\n    at <anonymous>:1:1"
}
```

**Features**:
- Async/await support
- Multi-line code execution
- Variable inspection
- DOM manipulation
- Function calls

**Example**:
```javascript
await mcp.call('repl_execute', {
  code: `
    const response = await fetch('/api/users');
    const users = await response.json();
    console.log('User count:', users.length);
    return users.length;
  `
});
```

## Resources

MCP resources provide read-only access to structured data:

### Available Resources

| Resource | Description |
|----------|-------------|
| `logs://recent` | Recent log entries from all processes |
| `logs://errors` | Recent error log entries only |
| `telemetry://sessions` | Active browser telemetry sessions |
| `telemetry://errors` | JavaScript errors from browser sessions |
| `proxy://requests` | Recent HTTP requests captured by proxy |
| `proxy://mappings` | Active reverse proxy URL mappings |
| `processes://active` | Currently running processes |
| `scripts://available` | Scripts defined in package.json |

### Resource Capabilities

- **Subscribe**: ✅ Yes (real-time updates)
- **ListChanged**: ✅ Yes (notifications when resource list changes)

## Prompts

Pre-configured prompt templates for debugging scenarios:

### `debug_error`

Analyze error logs and suggest fixes.

**Arguments**:
- `error_message` (required): The error message to debug
- `context` (optional): Additional context about when the error occurred

### `performance_analysis`

Analyze telemetry data for performance issues.

**Arguments**:
- `session_id` (optional): Telemetry session ID to analyze
- `metric_type` (optional): Specific metric to focus on

### `api_troubleshooting`

Examine proxy requests to debug API issues.

**Arguments**:
- `endpoint` (optional): API endpoint pattern to analyze
- `status_code` (optional): Filter by HTTP status code

### `script_configuration`

Help configure npm scripts for common tasks.

**Arguments**:
- `task_type` (required): Type of task (dev, build, test, lint, etc.)
- `framework` (optional): Framework being used

## Streaming Protocol

### Streaming Tools

The following tools support real-time streaming:

- ✅ `scripts_run` - Real-time script execution logs
- ✅ `logs_stream` - Live log monitoring
- ✅ `telemetry_events` - Real-time browser events

### Protocol Details

- **Transport**: Server-Sent Events (SSE) over HTTP
- **Format**: JSON-RPC 2.0 notifications
- **Heartbeat**: Every 30 seconds
- **Timeout**: 5 minutes for streaming operations
- **Endpoint**: `GET /mcp` with appropriate parameters

### Event Types

- `message` - Standard JSON-RPC message
- `ping` - Heartbeat/keepalive 
- `done` - Stream completion notification

## Error Handling

Standard JSON-RPC 2.0 error format:

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": "Missing required parameter: name"
  },
  "id": 1
}
```

### Error Codes

| Code | Meaning |
|------|---------|
| -32700 | Parse error |
| -32600 | Invalid Request |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |

## Connection Examples

### Direct HTTP

```javascript
// JSON-RPC 2.0 request
const response = await fetch('http://localhost:7777/mcp', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    jsonrpc: "2.0",
    id: 1,
    method: "scripts_list"
  })
});

const result = await response.json();
```

### Streaming Connection

```javascript
// Server-Sent Events for streaming
const eventSource = new EventSource('http://localhost:7777/mcp');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};
```

### MCP Client Configuration

For standard MCP clients (Claude Desktop, VSCode, etc.):

```json
{
  "servers": {
    "brummer": {
      "command": "brum",
      "args": ["--no-tui", "--port", "7777"]
    }
  }
}
```

## Hub Mode Tools

Hub mode enables coordination of multiple Brummer instances through a central hub. These tools are only available when running in hub mode (`brum --mcp`).

### Instance Management

#### `instances_list`

List all discovered Brummer instances with their connection states and details.

**Parameters**: None

**Response**:
```json
[
  {
    "id": "frontend-abc123",
    "name": "frontend",
    "directory": "/home/user/projects/frontend",
    "state": "active",
    "port": 7777,
    "pid": 12345,
    "startupTime": "2024-01-09T10:00:00Z",
    "timeInState": "5m30s",
    "retryCount": 0,
    "stateStats": {
      "discovered": 1,
      "connecting": 1,
      "active": 1
    }
  }
]
```

**Connection States**:
- `discovered`: Instance file found, not yet connected
- `connecting`: Attempting initial connection
- `active`: Connected and ready for hub tools
- `retrying`: Connection lost, attempting reconnection
- `dead`: Maximum retries exceeded

---

#### `instances_connect`

Establish session routing to a specific instance for subsequent hub tool calls.

**Parameters**:
- `instance_id` (string, required): Instance ID from `instances_list`

**Response**:
```json
{
  "success": true,
  "instanceId": "frontend-abc123",
  "previousConnection": null
}
```

**Session Routing**: After connecting, all `hub_*` tools automatically route to the connected instance without needing to specify `instance_id`.

---

#### `instances_disconnect`

Clear the current session routing context.

**Parameters**: None

**Response**:
```json
{
  "success": true,
  "previousConnection": "frontend-abc123"
}
```

### Hub Proxy Tools

All single-instance tools are available with `hub_` prefix for multi-instance coordination:

#### `hub_scripts_list`

List scripts from a specific instance. Same as `scripts_list` but requires `instance_id`.

**Parameters**:
- `instance_id` (string, required): Target instance ID

---

#### `hub_scripts_run`

Run a script on a specific instance. Same as `scripts_run` but requires `instance_id`.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `name` (string, required): Script name to execute

---

#### `hub_scripts_stop`

Stop a process on a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `processId` (string, required): Process ID to stop

---

#### `hub_scripts_status`

Check script status on a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `name` (string, optional): Specific script name

---

#### `hub_logs_stream`

Stream logs from a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `processId` (string, optional): Filter by process
- `level` (string, optional): Log level filter
- `follow` (boolean, optional): Stream real-time

---

#### `hub_logs_search`

Search logs from a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `query` (string, required): Search query
- `regex` (boolean, optional): Use regex matching
- `level` (string, optional): Filter by log level

---

#### `hub_proxy_requests`

Get proxy requests from a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `processName` (string, optional): Filter by process
- `status` (string, optional): Filter by status

---

#### `hub_telemetry_sessions`

Get telemetry sessions from a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `active` (boolean, optional): Only active sessions
- `limit` (integer, optional): Maximum sessions

---

#### `hub_telemetry_events`

Stream telemetry events from a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `sessionId` (string, optional): Filter by session
- `eventType` (string, optional): Event type filter
- `follow` (boolean, optional): Stream real-time

---

#### `hub_browser_open`

Open URL in browser via a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `url` (string, required): URL to open
- `processName` (string, optional): Associate with process

---

#### `hub_browser_refresh`

Refresh browser tabs connected to a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `sessionId` (string, optional): Specific session

---

#### `hub_browser_navigate`

Navigate browser tabs via a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `url` (string, required): URL to navigate to
- `sessionId` (string, optional): Specific session

---

#### `hub_browser_screenshot`

Capture screenshot via a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `sessionId` (string, optional): Specific session
- `format` (string, optional): Image format
- `fullPage` (boolean, optional): Full page capture
- `selector` (string, optional): Element selector

---

#### `hub_repl_execute`

Execute JavaScript in browser context via a specific instance.

**Parameters**:
- `instance_id` (string, required): Target instance ID
- `code` (string, required): JavaScript code
- `sessionId` (string, optional): Target session

### Hub Workflow Examples

**Multi-Instance Setup**:
```javascript
// 1. List available instances
const instances = await mcp.call('instances_list');

// 2. Connect to frontend instance
await mcp.call('instances_connect', {
  instance_id: 'frontend-abc123'
});

// 3. Now hub tools route automatically
const scripts = await mcp.call('hub_scripts_list');
await mcp.call('hub_scripts_run', { name: 'dev' });

// 4. Switch to backend instance
await mcp.call('instances_connect', {
  instance_id: 'backend-def456'
});

const logs = await mcp.call('hub_logs_stream', { follow: true });
```

**Direct Instance Targeting**:
```javascript
// Target specific instances without session routing
await mcp.call('hub_scripts_run', {
  instance_id: 'frontend-abc123',
  name: 'dev'
});

await mcp.call('hub_scripts_run', {
  instance_id: 'backend-def456', 
  name: 'start'
});
```

## Special Features

- **Cross-Platform**: Full support for Windows, Mac, Linux, and WSL2
- **Process Management**: Automatic process tracking and cleanup
- **Proxy Integration**: HTTP request interception and analysis
- **Real-Time Monitoring**: Live logs, telemetry, and events
- **Browser Automation**: Remote control of browser tabs
- **Hub Mode**: Multi-instance coordination and management
- **Session Routing**: Simplified hub tool usage with automatic routing
- **Security**: Token-based authentication for client connections
- **Memory Management**: Configurable storage limits and automatic cleanup