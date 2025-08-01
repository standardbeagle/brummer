# MCP Server Configuration

## Server Configuration

### **Single Instance Configuration**
- **Primary Endpoint**: `http://localhost:7777/mcp` (single URL for all MCP functionality)
- **Protocol**: JSON-RPC 2.0 with Server-Sent Events streaming support
- **Default Port**: 7777 (configurable with `-p` or `--port`)
- **Startup**: Automatically enabled unless `--no-mcp` flag is used

### **Hub Mode Configuration**
- **Transport**: stdio (JSON-RPC over stdin/stdout)
- **Discovery**: File-based instance discovery in shared directory
- **Routing**: Automatic tool routing to appropriate instances
- **Session Management**: Client session to instance mapping

## Client Configuration

### **Single Instance Setup**
For MCP clients (Claude Desktop, VSCode, etc.), configure the server executable:
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

For direct HTTP connections, use: `http://localhost:7777/mcp`

### **Hub Mode Setup** (Recommended for Multiple Projects)
```json
{
  "servers": {
    "brummer-hub": {
      "command": "brum",
      "args": ["--mcp"]
    }
  }
}
```

## MCP Connection Types (Streamable HTTP Transport)

The server implements the official MCP Streamable HTTP transport protocol:

### 1. **Standard JSON-RPC** (POST to `/mcp` with `Accept: application/json`):
   - Single request/response
   - Batch requests supported

### 2. **Server-Sent Events** (GET to `/mcp` with `Accept: text/event-stream`):
   - Server-to-client streaming
   - Supports resource subscriptions with real-time updates
   - Automatic heartbeat/ping messages

### 3. **SSE Response** (POST to `/mcp` with `Accept: text/event-stream`):
   - Client sends requests via POST
   - Server responds with SSE stream
   - Useful for streaming tool responses

### Headers:
- `Accept`: Must include appropriate content type
- `Mcp-Session-Id`: Optional session identifier for resumability
- `Content-Type`: `application/json` for requests

### Example SSE connection:
```javascript
const eventSource = new EventSource('http://localhost:7777/mcp');
eventSource.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log('Received:', msg);
};

// Send requests via POST with session ID
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