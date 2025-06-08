---
sidebar_position: 2
---

# MCP API Reference

Complete reference for the Brummer Model Context Protocol (MCP) server API.

## Overview

The Brummer MCP server provides a JSON-RPC 2.0 interface for external tools to interact with Brummer's functionality.

### Connection Details

- **Protocol**: JSON-RPC 2.0
- **Transport**: stdio (standard input/output)
- **Default Port**: 3280

## Core Methods

### brummer.getProcesses

Get list of all processes and their current status.

**Parameters**: None

**Returns**:
```typescript
{
  processes: Array<{
    id: string;
    name: string;
    status: 'running' | 'stopped' | 'failed' | 'pending';
    pid?: number;
    uptime?: number;
    cpu?: number;
    memory?: number;
    restartCount: number;
  }>
}
```

**Example**:
```javascript
const result = await mcp.call('brummer.getProcesses');
console.log(result.processes);
// [
//   {
//     id: "dev-server",
//     name: "dev",
//     status: "running",
//     pid: 12345,
//     uptime: 3600,
//     cpu: 2.5,
//     memory: 156000000
//   }
// ]
```

### brummer.startProcess

Start a specific process by name.

**Parameters**:
```typescript
{
  name: string;  // Process/script name
  env?: Record<string, string>;  // Optional environment variables
  args?: string[];  // Optional additional arguments
}
```

**Returns**:
```typescript
{
  success: boolean;
  processId: string;
  pid?: number;
  error?: string;
}
```

**Example**:
```javascript
const result = await mcp.call('brummer.startProcess', {
  name: 'dev',
  env: {
    NODE_ENV: 'development',
    PORT: '3001'
  }
});
```

### brummer.stopProcess

Stop a running process.

**Parameters**:
```typescript
{
  name: string;  // Process name
  signal?: 'SIGTERM' | 'SIGKILL';  // Optional signal type
  timeout?: number;  // Grace period in milliseconds
}
```

**Returns**:
```typescript
{
  success: boolean;
  error?: string;
}
```

**Example**:
```javascript
await mcp.call('brummer.stopProcess', {
  name: 'dev',
  signal: 'SIGTERM',
  timeout: 5000
});
```

### brummer.restartProcess

Restart a process.

**Parameters**:
```typescript
{
  name: string;
  graceful?: boolean;  // Wait for process to stop before starting
}
```

**Returns**:
```typescript
{
  success: boolean;
  processId: string;
  pid?: number;
  error?: string;
}
```

## Log Methods

### brummer.getLogs

Retrieve logs for processes.

**Parameters**:
```typescript
{
  processName?: string;  // Filter by process
  level?: 'error' | 'warn' | 'info' | 'debug';  // Filter by level
  limit?: number;  // Maximum entries (default: 100)
  offset?: number;  // Skip entries
  since?: string;  // ISO timestamp
  until?: string;  // ISO timestamp
  search?: string;  // Search term
}
```

**Returns**:
```typescript
{
  logs: Array<{
    id: string;
    processName: string;
    timestamp: string;
    level: string;
    message: string;
    metadata?: Record<string, any>;
  }>;
  total: number;
  hasMore: boolean;
}
```

**Example**:
```javascript
const logs = await mcp.call('brummer.getLogs', {
  processName: 'dev',
  level: 'error',
  limit: 50,
  since: '2024-01-15T10:00:00Z'
});
```

### brummer.streamLogs

Subscribe to real-time log updates.

**Parameters**:
```typescript
{
  processName?: string;
  level?: string;
  follow?: boolean;  // Keep connection open
}
```

**Returns**: Stream of log entries

**Example**:
```javascript
const stream = await mcp.call('brummer.streamLogs', {
  processName: 'dev',
  follow: true
});

stream.on('data', (log) => {
  console.log(log);
});
```

### brummer.clearLogs

Clear logs for a process.

**Parameters**:
```typescript
{
  processName?: string;  // Clear specific process or all
  before?: string;  // Clear logs before timestamp
}
```

**Returns**:
```typescript
{
  success: boolean;
  cleared: number;  // Number of entries cleared
}
```

## Error Detection Methods

### brummer.getErrors

Get detected errors.

**Parameters**:
```typescript
{
  processName?: string;
  severity?: 'critical' | 'error' | 'warning';
  limit?: number;
  resolved?: boolean;
}
```

**Returns**:
```typescript
{
  errors: Array<{
    id: string;
    processName: string;
    timestamp: string;
    severity: string;
    message: string;
    stackTrace?: string;
    occurrences: number;
    firstSeen: string;
    lastSeen: string;
    resolved: boolean;
  }>;
  total: number;
}
```

### brummer.resolveError

Mark an error as resolved.

**Parameters**:
```typescript
{
  errorId: string;
  notes?: string;
}
```

**Returns**:
```typescript
{
  success: boolean;
}
```

## URL Methods

### brummer.getUrls

Get all detected URLs.

**Parameters**: None

**Returns**:
```typescript
{
  urls: Array<{
    url: string;
    processName: string;
    status: 'online' | 'offline' | 'unknown';
    lastChecked?: string;
    responseTime?: number;
    headers?: Record<string, string>;
  }>
}
```

### brummer.checkUrl

Check URL availability.

**Parameters**:
```typescript
{
  url: string;
  method?: string;  // HTTP method
  timeout?: number;
}
```

**Returns**:
```typescript
{
  url: string;
  status: number;  // HTTP status code
  responseTime: number;
  headers: Record<string, string>;
  error?: string;
}
```

## Script Methods

### brummer.getScripts

Get available npm/yarn/pnpm scripts.

**Parameters**:
```typescript
{
  includeWorkspaces?: boolean;
  packagePath?: string;
}
```

**Returns**:
```typescript
{
  scripts: Array<{
    name: string;
    command: string;
    package: string;
    path: string;
    isWorkspace: boolean;
  }>;
  packageManager: 'npm' | 'yarn' | 'pnpm' | 'bun';
}
```

### brummer.runScript

Execute a script directly.

**Parameters**:
```typescript
{
  name: string;
  packagePath?: string;
  args?: string[];
  env?: Record<string, string>;
  detached?: boolean;  // Run in background
}
```

**Returns**:
```typescript
{
  success: boolean;
  processId?: string;
  output?: string;  // If not detached
  exitCode?: number;
  error?: string;
}
```

## Configuration Methods

### brummer.getConfig

Get current configuration.

**Parameters**:
```typescript
{
  key?: string;  // Specific config key
}
```

**Returns**:
```typescript
{
  config: Record<string, any> | any;
}
```

### brummer.setConfig

Update configuration.

**Parameters**:
```typescript
{
  key: string;
  value: any;
  persist?: boolean;  // Save to file
}
```

**Returns**:
```typescript
{
  success: boolean;
  previous: any;
}
```

## Event Subscription

### brummer.subscribe

Subscribe to Brummer events.

**Parameters**:
```typescript
{
  events: Array<
    | 'process.start'
    | 'process.stop'
    | 'process.error'
    | 'log.error'
    | 'url.detected'
    | 'build.complete'
    | 'test.complete'
  >;
}
```

**Returns**: Event stream

**Example**:
```javascript
const events = await mcp.call('brummer.subscribe', {
  events: ['process.error', 'build.complete']
});

events.on('event', (data) => {
  console.log(data.type, data.payload);
});
```

### brummer.unsubscribe

Unsubscribe from events.

**Parameters**:
```typescript
{
  subscriptionId: string;
}
```

**Returns**:
```typescript
{
  success: boolean;
}
```

## Utility Methods

### brummer.ping

Check if MCP server is responsive.

**Parameters**: None

**Returns**:
```typescript
{
  pong: true;
  timestamp: string;
  version: string;
}
```

### brummer.getStats

Get system statistics.

**Parameters**: None

**Returns**:
```typescript
{
  uptime: number;
  totalProcesses: number;
  runningProcesses: number;
  totalLogs: number;
  totalErrors: number;
  memoryUsage: number;
  cpuUsage: number;
}
```

### brummer.exportLogs

Export logs to file.

**Parameters**:
```typescript
{
  format: 'json' | 'csv' | 'text';
  processName?: string;
  since?: string;
  until?: string;
  includeMetadata?: boolean;
}
```

**Returns**:
```typescript
{
  success: boolean;
  path: string;
  size: number;
  entries: number;
}
```

## Error Handling

All methods follow standard JSON-RPC 2.0 error format:

```typescript
{
  jsonrpc: "2.0",
  error: {
    code: number;
    message: string;
    data?: any;
  },
  id: number | string;
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
| 1000 | Process not found |
| 1001 | Process already running |
| 1002 | Process not running |
| 1003 | Permission denied |
| 1004 | Configuration error |

## Rate Limiting

The MCP server implements rate limiting:

- **Default**: 100 requests per minute
- **Burst**: 20 requests
- **Log streaming**: Not rate limited

Configure in `.brummer.yaml`:

```yaml
mcp:
  rate_limit:
    requests_per_minute: 200
    burst: 50
```

## Authentication

Optional authentication for MCP server:

```yaml
mcp:
  auth:
    enabled: true
    token: "your-secret-token"
```

Include token in requests:

```javascript
const mcp = new MCPClient({
  headers: {
    'Authorization': 'Bearer your-secret-token'
  }
});
```

## WebSocket API

For web clients:

```javascript
const ws = new WebSocket('ws://localhost:3280');

ws.onopen = () => {
  ws.send(JSON.stringify({
    jsonrpc: '2.0',
    method: 'brummer.getProcesses',
    id: 1
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);
  console.log(response);
};
```