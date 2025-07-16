---
sidebar_position: 1
---

# MCP Integration Overview

Brummer includes a built-in Model Context Protocol (MCP) server that allows external tools like VSCode, Claude Code, and Cursor to interact with your development processes.

## What is MCP?

The Model Context Protocol (MCP) is an open standard that enables seamless communication between development tools and services. It provides a unified interface for:

- **Process Management** - Start, stop, and monitor processes
- **Log Access** - Stream and search logs in real-time
- **Error Detection** - Get notified of errors and warnings
- **System Status** - Monitor resource usage and performance

## Why Use MCP with Brummer?

### For AI Assistants

AI coding assistants can:
- **Understand your dev environment** - See what's running and its status
- **Debug issues** - Access error logs and stack traces
- **Execute commands** - Run scripts and manage processes
- **Monitor performance** - Track memory and CPU usage

### For IDEs and Editors

Development environments can:
- **Integrate process management** - Control Brummer from your editor
- **Display inline errors** - Show errors next to your code
- **Quick navigation** - Jump to error locations
- **Real-time feedback** - See build and test results instantly

### For Custom Tools

Build your own integrations:
- **Automation scripts** - Programmatically control processes
- **Monitoring dashboards** - Create custom status displays
- **CI/CD pipelines** - Integrate with build systems
- **Chat bots** - Add Brummer commands to Slack/Discord

## Architecture

```
┌─────────────────┐     JSON-RPC 2.0    ┌─────────────────┐
│   MCP Client    │◄──────────────────►│  Brummer MCP    │
│  (VSCode, etc)  │                     │     Server      │
└─────────────────┘                     └─────────────────┘
                                                │
                                                ▼
                                        ┌─────────────────┐
                                        │  Brummer Core   │
                                        │   - Processes   │
                                        │   - Logs        │
                                        │   - Events      │
                                        └─────────────────┘
```

## Key Features

### 1. Process Control

Control your development processes from any MCP client:

```javascript
// Start a development server
await mcp.call('scripts_run', { name: 'dev' });

// Stop a process
const status = await mcp.call('scripts_status', { name: 'dev' });
await mcp.call('scripts_stop', { processId: status.processId });

// Get all processes status
const allStatus = await mcp.call('scripts_status');
```

### 2. Log Streaming

Access logs in real-time:

```javascript
// Search recent logs
const logs = await mcp.call('logs_search', {
  query: 'dev',
  limit: 100
});

// Stream logs as they happen
const stream = await mcp.call('logs_stream', {
  follow: true,
  limit: 100
});
```

### 3. Error Detection

Get notified of errors immediately:

```javascript
// Get recent errors
const errors = await mcp.call('logs_search', {
  query: 'error',
  level: 'error',
  limit: 10
});

// Stream error events
const errorStream = await mcp.call('logs_stream', {
  level: 'error',
  follow: true
});
```

### 4. URL Management

Access detected URLs:

```javascript
// Get HTTP requests from proxy
const requests = await mcp.call('proxy_requests', {
  limit: 50
});

// Open URL in browser
await mcp.call('browser_open', {
  url: 'http://localhost:3000'
});
```

## Quick Start

### 1. Enable MCP Server

Start Brummer with MCP enabled:

```bash
brum --mcp
```

Or configure in `.brum.toml`:

```toml
mcp_port = 7777
no_mcp = false
```

### 2. Connect from VSCode

Add to your VSCode settings:

```json
{
  "mcp.servers": {
    "brummer": {
      "command": "brum",
      "args": ["--mcp"]
    }
  }
}
```

### 3. Connect from Claude Code

Add to Claude Code's MCP configuration:

```json
{
  "mcpServers": {
    "brummer": {
      "command": "brum",
      "args": ["--mcp"]
    }
  }
}
```

## Common Use Cases

### 1. AI-Assisted Debugging

When Claude or another AI assistant is helping you debug:

```
You: "My server is crashing, can you help?"

Claude: "Let me check your server logs..."
[Uses MCP to get recent errors]

Claude: "I found the issue. Your server is crashing due to a missing environment variable. Here's the error:
Error: Missing required environment variable: API_KEY
  at startServer (server.js:15:11)

Would you like me to help you fix this?"
```

### 2. Automated Testing

Run tests and get results programmatically:

```javascript
// Start test process
const testProcess = await mcp.call('scripts_run', { name: 'test' });

// Monitor test logs
const testLogs = await mcp.call('logs_stream', {
  processId: testProcess.processId,
  follow: true
});
```

### 3. Performance Monitoring

Track resource usage:

```javascript
// Get telemetry sessions
const sessions = await mcp.call('telemetry_sessions', {
  limit: 10
});

// Monitor browser performance
const events = await mcp.call('telemetry_events', {
  eventType: 'performance',
  follow: true
});
```

### 4. Build Automation

Integrate with build pipelines:

```javascript
// Start build
const buildProcess = await mcp.call('scripts_run', { name: 'build' });

// Monitor build logs
const buildLogs = await mcp.call('logs_stream', {
  processId: buildProcess.processId,
  follow: true
});
```

## Security Considerations

### Authentication

Enable authentication for production use:

```toml
# Security is handled at the process level
# MCP server runs locally on localhost:7777
```

### Access Control

Limit MCP server access:

```toml
# Tool access is controlled by MCP client configuration
# All tools are available to connected clients
```

### Network Security

For WebSocket transport:

```toml
# Brummer uses Streamable HTTP transport
mcp_port = 7777
# Server only accepts local connections
```

## Advanced Configuration

### Custom Commands

Add custom MCP commands:

```json
// Add custom scripts to package.json
{
  "scripts": {
    "deploy": "./scripts/deploy.sh",
    "db:reset": "npm run db:reset"
  }
}
```

### Event Filtering

Control which events are exposed:

```javascript
// Filter events using tool parameters
const errorLogs = await mcp.call('logs_stream', {
  level: 'error',
  follow: true
});

const processLogs = await mcp.call('logs_stream', {
  processId: 'specific-process-id',
  follow: true
});
```

### Rate Limiting

Prevent abuse:

```toml
# Rate limiting handled by HTTP server
# Default limits are appropriate for development use
```

## Troubleshooting

### MCP Server Not Starting

1. Check if another process is using the port
2. Verify Brummer has necessary permissions
3. Check logs for error messages
4. Try stdio transport instead of websocket

### Connection Issues

1. Ensure MCP server is enabled
2. Check firewall settings
3. Verify authentication tokens match
4. Enable debug logging

### Performance Issues

1. Limit log streaming rate
2. Use filters to reduce data transfer
3. Enable compression for WebSocket
4. Increase buffer sizes

## Next Steps

- [API Reference](./api-reference) - Complete list of MCP methods
- [Client Setup](./client-setup) - Configure various MCP clients
- [Events](./events) - Real-time event documentation
- [Examples](https://github.com/yourusername/brummer/tree/main/examples/mcp) - Sample implementations