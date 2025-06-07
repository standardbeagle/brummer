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
await mcp.call('brummer.startProcess', { name: 'dev' });

// Stop a process
await mcp.call('brummer.stopProcess', { name: 'dev' });

// Get process status
const status = await mcp.call('brummer.getProcesses');
```

### 2. Log Streaming

Access logs in real-time:

```javascript
// Get recent logs
const logs = await mcp.call('brummer.getLogs', {
  processName: 'dev',
  limit: 100
});

// Stream logs as they happen
const stream = await mcp.call('brummer.streamLogs', {
  follow: true
});
```

### 3. Error Detection

Get notified of errors immediately:

```javascript
// Get recent errors
const errors = await mcp.call('brummer.getErrors', {
  severity: 'error',
  limit: 10
});

// Subscribe to error events
await mcp.subscribe('error', (error) => {
  console.log('Error detected:', error);
});
```

### 4. URL Management

Access detected URLs:

```javascript
// Get all detected URLs
const urls = await mcp.call('brummer.getUrls');

// Check URL status
const status = await mcp.call('brummer.checkUrl', {
  url: 'http://localhost:3000'
});
```

## Quick Start

### 1. Enable MCP Server

Start Brummer with MCP enabled:

```bash
brum --mcp
```

Or configure in `.brummer.yaml`:

```yaml
mcp:
  enabled: true
  transport: stdio  # or websocket
  port: 3280       # for websocket
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
await mcp.call('brummer.startProcess', { name: 'test' });

// Wait for completion
await mcp.subscribe('test.complete', (result) => {
  if (result.failed > 0) {
    // Handle test failures
  }
});
```

### 3. Performance Monitoring

Track resource usage:

```javascript
// Get process stats
const stats = await mcp.call('brummer.getProcessStats', {
  name: 'dev'
});

console.log(`Memory: ${stats.memory / 1024 / 1024}MB`);
console.log(`CPU: ${stats.cpu}%`);
```

### 4. Build Automation

Integrate with build pipelines:

```javascript
// Start build
await mcp.call('brummer.startProcess', { name: 'build' });

// Monitor progress
await mcp.subscribe('build.progress', (progress) => {
  console.log(`Build ${progress.percent}% complete`);
});
```

## Security Considerations

### Authentication

Enable authentication for production use:

```yaml
mcp:
  auth:
    enabled: true
    token: "your-secret-token"
```

### Access Control

Limit MCP server access:

```yaml
mcp:
  allowed_methods:
    - brummer.getProcesses
    - brummer.getLogs
  denied_methods:
    - brummer.stopProcess
```

### Network Security

For WebSocket transport:

```yaml
mcp:
  transport: websocket
  host: localhost  # Only local connections
  port: 3280
  ssl:
    enabled: true
    cert: /path/to/cert.pem
    key: /path/to/key.pem
```

## Advanced Configuration

### Custom Commands

Add custom MCP commands:

```yaml
mcp:
  custom_commands:
    - name: "deploy"
      description: "Deploy to staging"
      script: "./scripts/deploy.sh"
      
    - name: "db:reset"
      description: "Reset database"
      script: "npm run db:reset"
```

### Event Filtering

Control which events are exposed:

```yaml
mcp:
  events:
    include:
      - process.*
      - log.error
      - build.complete
    exclude:
      - log.debug
      - process.heartbeat
```

### Rate Limiting

Prevent abuse:

```yaml
mcp:
  rate_limit:
    enabled: true
    requests_per_minute: 100
    burst: 20
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