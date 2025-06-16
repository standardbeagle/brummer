# Test: MCP Duplicate Script Prevention

## Test Steps

1. Start Brummer in MCP mode:
   ```bash
   ./brum --no-tui
   ```

2. In another terminal, use curl to test the MCP endpoint:

### Test 1: Start a script for the first time
```bash
curl -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "scripts_run",
      "arguments": {"name": "dev"}
    },
    "id": 1
  }'
```

Expected: Script starts successfully, returns processId and status

### Test 2: Try to start the same script again
```bash
curl -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "scripts_run",
      "arguments": {"name": "dev"}
    },
    "id": 2
  }'
```

Expected: Returns duplicate=true with current process info, commands to stop/restart, and proxy URLs

### Test 3: Check script status
```bash
curl -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "scripts_status",
      "arguments": {"name": "dev"}
    },
    "id": 3
  }'
```

Expected: Returns process status with proxy URLs and management commands

### Test 4: Stop the script
```bash
# Use the processId from Test 1
curl -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "scripts_stop",
      "arguments": {"processId": "dev-XXXXXXXXXX"}
    },
    "id": 4
  }'
```

## Response Format When Duplicate Detected

When attempting to start a script that's already running, the response should include:

```json
{
  "processId": "dev-1234567890",
  "name": "dev",
  "status": "running",
  "duplicate": true,
  "message": "Script 'dev' is already running with process ID: dev-1234567890",
  "startTime": "2024-01-16T10:30:00Z",
  "runtime": "5m30s",
  "commands": {
    "stop": "scripts/stop {\"processId\": \"dev-1234567890\"}",
    "restart": "First stop the process, then run again",
    "status": "Check status with: scripts/status"
  },
  "proxyUrls": [
    {
      "targetUrl": "http://localhost:3000",
      "proxyUrl": "http://localhost:8889",
      "label": "Frontend"
    }
  ]
}
```

## Key Features

1. **Duplicate Detection**: Prevents starting multiple instances of the same script
2. **Current State Info**: Shows how long the script has been running
3. **Management Commands**: Provides exact commands to stop or check status
4. **Proxy URLs**: Lists all proxy URLs associated with the running process
5. **Streaming Support**: The streaming handler also detects duplicates and sends a special message