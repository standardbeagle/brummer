---
sidebar_position: 3
---

# Client Setup

Learn how to set up various clients to work with Brummer's MCP server.

## VSCode Extension

### Installation

1. Install the MCP VSCode extension:
   ```bash
   code --install-extension mcp.vscode-mcp-client
   ```

2. Configure MCP settings in VSCode:
   ```json
   {
     "mcp.servers": {
       "brummer": {
         "command": "brum",
         "args": ["--mcp"],
         "env": {
           "BRUMMER_MCP_MODE": "stdio"
         }
       }
     }
   }
   ```

### Usage

Access Brummer from VSCode:

1. Open Command Palette (`Ctrl+Shift+P`)
2. Type "MCP: Connect to Brummer"
3. Use MCP panel to interact with processes

### Features

- View running processes
- Start/stop scripts
- Search logs
- View errors inline
- Quick navigation to error locations

## Claude Code

### Configuration

1. Add Brummer to Claude Code's MCP config:

```json
// ~/Library/Application Support/Claude/claude_desktop_config.json (macOS)
// ~/.config/Claude/claude_desktop_config.json (Linux)
// %APPDATA%\Claude\claude_desktop_config.json (Windows)
{
  "mcpServers": {
    "brummer": {
      "command": "brum",
      "args": ["--mcp"],
      "env": {
        "BRUMMER_MCP_MODE": "stdio"
      }
    }
  }
}
```

2. Restart Claude Code

### Available Commands

Claude can now:
- Monitor your development processes
- Access build logs and errors
- Execute npm scripts
- Check server status
- Debug issues with full context

Example prompts:
- "Check if the dev server is running"
- "Show me recent errors from the build process"
- "Restart the API server"
- "What's causing the test failures?"

## Cursor

### Setup

1. Install Cursor MCP extension
2. Add Brummer configuration:

```json
// .cursor/settings.json
{
  "mcp.servers": [
    {
      "name": "brummer",
      "command": "brum",
      "args": ["--mcp"],
      "env": {
        "BRUMMER_MCP_MODE": "stdio"
      }
    }
  ]
}
```

### Integration Features

- Inline error display
- Quick fixes from Brummer errors
- Process status in status bar
- Log search from editor

## Node.js Client

### Installation

```bash
npm install @modelcontextprotocol/sdk
```

### Basic Client

```javascript
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';

const transport = new StdioClientTransport({
  command: 'brum',
  args: ['--mcp']
});

const client = new Client({
  name: 'my-mcp-client',
  version: '1.0.0'
}, {
  capabilities: {}
});

await client.connect(transport);

// Make requests using tools/call
const result = await client.request({
  method: 'tools/call',
  params: {
    name: 'scripts_status'
  }
}, {});

console.log(result);
```

### Advanced Client

```javascript
class BrummerClient {
  constructor() {
    this.client = null;
    this.transport = null;
  }

  async connect() {
    this.transport = new StdioClientTransport({
      command: 'brum',
      args: ['--mcp'],
      env: {
        ...process.env,
        BRUMMER_MCP_MODE: 'stdio'
      }
    });

    this.client = new Client({
      name: 'brummer-client',
      version: '1.0.0'
    }, {
      capabilities: {
        tools: true,
        resources: true
      }
    });

    await this.client.connect(this.transport);
    
    // Client is now ready for tool calls
  }

  async getProcesses() {
    const response = await this.client.request({
      method: 'tools/call',
      params: {
        name: 'scripts_status'
      }
    }, {});
    return response.result;
  }

  async startProcess(name, options = {}) {
    return await this.client.request({
      method: 'tools/call',
      params: {
        name: 'scripts_run',
        arguments: { name, ...options }
      }
    }, {});
  }

  async getLogs(options = {}) {
    return await this.client.request({
      method: 'tools/call',
      params: {
        name: 'logs_stream',
        arguments: options
      }
    }, {});
  }

  handleEvent(event) {
    console.log('Event:', event.type, event.data);
  }

  async disconnect() {
    await this.client.close();
  }
}

// Usage
const brummer = new BrummerClient();
await brummer.connect();

const processes = await brummer.getProcesses();
console.log('Running processes:', processes);

await brummer.disconnect();
```

## Python Client

### Installation

```bash
pip install mcp-client
```

### Example Client

```python
import asyncio
from mcp import Client, StdioTransport

class BrummerClient:
    def __init__(self):
        self.client = None
        
    async def connect(self):
        transport = StdioTransport(
            command="brum",
            args=["--mcp"]
        )
        
        self.client = Client(
            name="python-brummer-client",
            version="1.0.0"
        )
        
        await self.client.connect(transport)
    
    async def get_processes(self):
        response = await self.client.request(
            method="tools/call",
            params={
                "name": "scripts_status"
            }
        )
        return response["result"]
    
    async def start_process(self, name, **kwargs):
        return await self.client.request(
            method="tools/call",
            params={
                "name": "scripts_run",
                "arguments": {"name": name, **kwargs}
            }
        )
    
    async def get_logs(self, **filters):
        return await self.client.request(
            method="tools/call",
            params={
                "name": "logs_stream",
                "arguments": filters
            }
        )
    
    async def stream_logs(self, process_name=None):
        # Note: Streaming requires HTTP transport
        response = await self.client.request(
            method="tools/call",
            params={
                "name": "logs_stream",
                "arguments": {"processId": process_name, "follow": True}
            }
        )
        return response["result"]
    
    async def disconnect(self):
        await self.client.close()

# Usage
async def main():
    client = BrummerClient()
    await client.connect()
    
    # Get all processes
    processes = await client.get_processes()
    for process in processes:
        print(f"{process['name']}: {process['status']}")
    
    # Start a process
    result = await client.start_process("dev")
    print(f"Started: {result['success']}")
    
    # Stream logs
    async for log in client.stream_logs("dev"):
        print(f"[{log['level']}] {log['message']}")
    
    await client.disconnect()

if __name__ == "__main__":
    asyncio.run(main())
```

## Browser Client (HTTP)

### Vanilla JavaScript

```javascript
class BrummerHTTPClient {
  constructor(baseUrl = 'http://localhost:7777') {
    this.baseUrl = baseUrl;
    this.requestId = 0;
  }

  async request(method, params = {}) {
    const id = ++this.requestId;
    
    const response = await fetch(`${this.baseUrl}/mcp`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id,
        method,
        params
      })
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const result = await response.json();
    
    if (result.error) {
      throw new Error(result.error.message);
    }
    
    return result.result;
  }

  async getProcesses() {
    return this.request('tools/call', {
      name: 'scripts_status'
    });
  }

  async startProcess(name, options = {}) {
    return this.request('tools/call', {
      name: 'scripts_run',
      arguments: { name, ...options }
    });
  }

  async getLogs(filters = {}) {
    return this.request('tools/call', {
      name: 'logs_stream',
      arguments: filters
    });
  }

  async streamLogs(filters = {}) {
    // Server-Sent Events for streaming
    const eventSource = new EventSource(`${this.baseUrl}/mcp`);
    
    return new Promise((resolve, reject) => {
      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === 'log') {
            // Handle log entry
            console.log('Log:', data.data);
          }
        } catch (error) {
          reject(error);
        }
      };
      
      eventSource.onerror = (error) => {
        reject(error);
      };
    });
  }
}

// Usage
const client = new BrummerHTTPClient();

const processes = await client.getProcesses();
console.log('Processes:', processes);

// Start streaming logs
client.streamLogs({ follow: true });
```

### React Hook

```jsx
import { useState, useEffect, useCallback } from 'react';

function useBrummer(baseUrl = 'http://localhost:7777') {
  const [client, setClient] = useState(null);
  const [connected, setConnected] = useState(false);
  const [processes, setProcesses] = useState([]);
  const [logs, setLogs] = useState([]);

  useEffect(() => {
    const brummerClient = new BrummerHTTPClient(baseUrl);
    setClient(brummerClient);
    setConnected(true);
    
    // Initial data fetch
    brummerClient.getProcesses().then((result) => {
      setProcesses(Array.isArray(result) ? result : [result]);
    }).catch(console.error);
  }, [baseUrl]);

  const startProcess = useCallback(async (name) => {
    if (!client) return;
    
    try {
      const result = await client.startProcess(name);
      // Refresh processes list
      const updatedProcesses = await client.getProcesses();
      setProcesses(Array.isArray(updatedProcesses) ? updatedProcesses : [updatedProcesses]);
      return result;
    } catch (error) {
      console.error('Failed to start process:', error);
      throw error;
    }
  }, [client]);

  const stopProcess = useCallback(async (processId) => {
    if (!client) return;
    
    try {
      const result = await client.request('tools/call', {
        name: 'scripts_stop',
        arguments: { processId }
      });
      // Refresh processes list
      const updatedProcesses = await client.getProcesses();
      setProcesses(Array.isArray(updatedProcesses) ? updatedProcesses : [updatedProcesses]);
      return result;
    } catch (error) {
      console.error('Failed to stop process:', error);
      throw error;
    }
  }, [client]);

  const fetchLogs = useCallback(async (filters = {}) => {
    if (!client) return;
    
    try {
      const result = await client.getLogs(filters);
      setLogs(result);
      return result;
    } catch (error) {
      console.error('Failed to fetch logs:', error);
      throw error;
    }
  }, [client]);

  return {
    connected,
    processes,
    logs,
    startProcess,
    stopProcess,
    fetchLogs
  };
}

// Component usage
function ProcessManager() {
  const { connected, processes, startProcess, stopProcess } = useBrummer();

  if (!connected) {
    return <div>Connecting to Brummer...</div>;
  }

  return (
    <div>
      <h2>Processes</h2>
      {processes.map(process => (
        <div key={process.processId || process.name}>
          <span>{process.name}</span>
          <span>{process.status}</span>
          {process.status === 'running' ? (
            <button onClick={() => stopProcess(process.processId)}>Stop</button>
          ) : (
            <button onClick={() => startProcess(process.name)}>Start</button>
          )}
        </div>
      ))}
    </div>
  );
}
```

## CLI Client

### Direct Commands

```bash
# Send MCP request via curl
curl -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scripts_status"}}'

# Start a process
curl -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scripts_run","arguments":{"name":"dev"}}}'

# Get logs with pretty print
curl -X POST http://localhost:7777/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"logs_search","arguments":{"query":".*","limit":10}}}' | jq
```

### Shell Script Client

```bash
#!/bin/bash

# brummer-client.sh
brummer_request() {
  local tool_name=$1
  local args=${2:-'{}'}
  
  curl -s -X POST http://localhost:7777/mcp \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"$tool_name\",\"arguments\":$args}}" | \
    jq -r '.result'
}

# Get all processes
processes=$(brummer_request "scripts_status")
echo "Processes: $processes"

# Start a process
result=$(brummer_request "scripts_run" '{"name": "dev"}')
echo "Start result: $result"

# Get recent errors
errors=$(brummer_request "logs_search" '{"query": "error", "level": "error", "limit": 5}')
echo "Recent errors: $errors"
```

## Testing MCP Integration

### Test Script

```javascript
// test-mcp.js
import { BrummerClient } from './brummer-client.js';

async function testMCPIntegration() {
  const client = new BrummerClient();
  
  try {
    console.log('Connecting to Brummer MCP server...');
    await client.connect();
    console.log('✅ Connected');
    
    // Test getting processes
    console.log('\nTesting scripts_status...');
    const processes = await client.getProcesses();
    console.log('✅ Processes:', processes);
    
    // Test starting a process
    console.log('\nTesting scripts_run...');
    const startResult = await client.startProcess('dev');
    console.log('✅ Start result:', startResult);
    
    // Test getting logs
    console.log('\nTesting logs_stream...');
    const logs = await client.getLogs({ limit: 5 });
    console.log('✅ Logs:', logs);
    
    // Test log search
    console.log('\nTesting logs_search...');
    const errors = await client.request('tools/call', {
      name: 'logs_search',
      arguments: { query: 'error', level: 'error', limit: 5 }
    });
    console.log('✅ Errors:', errors);
    
    console.log('\n✅ All tests passed!');
  } catch (error) {
    console.error('❌ Test failed:', error);
  } finally {
    await client.disconnect();
  }
}

testMCPIntegration();
```

## Troubleshooting

### Connection Issues

1. **Verify Brummer is running with MCP**:
   ```bash
   ps aux | grep "brum --mcp"
   ```

2. **Check if MCP server is responding**:
   ```bash
   curl -X POST http://localhost:7777/mcp \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
   ```

3. **Enable debug logging**:
   ```bash
   BRUMMER_MCP_DEBUG=true brum --mcp
   ```

### Common Errors

- **"Method not found"**: Ensure you're using the correct method name
- **"Invalid params"**: Check parameter types and required fields
- **"Connection refused"**: Verify Brummer is running and MCP is enabled
- **"Timeout"**: Increase client timeout or check system resources

## Best Practices

1. **Always handle connection errors** gracefully
2. **Implement reconnection logic** for long-running clients
3. **Use request IDs** for tracking multiple concurrent requests
4. **Subscribe to events** instead of polling for real-time updates
5. **Batch requests** when fetching multiple resources
6. **Clean up connections** properly when done
7. **Log MCP interactions** for debugging