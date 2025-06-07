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

// Make requests
const result = await client.request({
  method: 'brummer.getProcesses'
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
    
    // Set up event listeners
    this.client.notification(({ method, params }) => {
      if (method === 'brummer.event') {
        this.handleEvent(params);
      }
    });
  }

  async getProcesses() {
    const response = await this.client.request({
      method: 'brummer.getProcesses'
    }, {});
    return response.processes;
  }

  async startProcess(name, options = {}) {
    return await this.client.request({
      method: 'brummer.startProcess',
      params: { name, ...options }
    }, {});
  }

  async getLogs(options = {}) {
    return await this.client.request({
      method: 'brummer.getLogs',
      params: options
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
            method="brummer.getProcesses"
        )
        return response["processes"]
    
    async def start_process(self, name, **kwargs):
        return await self.client.request(
            method="brummer.startProcess",
            params={"name": name, **kwargs}
        )
    
    async def get_logs(self, **filters):
        return await self.client.request(
            method="brummer.getLogs",
            params=filters
        )
    
    async def stream_logs(self, process_name=None):
        async for log in self.client.stream(
            method="brummer.streamLogs",
            params={"processName": process_name, "follow": True}
        ):
            yield log
    
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

## Browser Client (WebSocket)

### Vanilla JavaScript

```javascript
class BrummerWebSocketClient {
  constructor(url = 'ws://localhost:3280') {
    this.url = url;
    this.ws = null;
    this.requestId = 0;
    this.pending = new Map();
  }

  connect() {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.url);
      
      this.ws.onopen = () => {
        console.log('Connected to Brummer');
        resolve();
      };
      
      this.ws.onerror = (error) => {
        reject(error);
      };
      
      this.ws.onmessage = (event) => {
        const response = JSON.parse(event.data);
        
        if (response.id && this.pending.has(response.id)) {
          const { resolve, reject } = this.pending.get(response.id);
          this.pending.delete(response.id);
          
          if (response.error) {
            reject(response.error);
          } else {
            resolve(response.result);
          }
        } else if (response.method === 'notification') {
          this.handleNotification(response.params);
        }
      };
    });
  }

  async request(method, params = {}) {
    const id = ++this.requestId;
    
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      
      this.ws.send(JSON.stringify({
        jsonrpc: '2.0',
        id,
        method,
        params
      }));
      
      // Timeout after 30 seconds
      setTimeout(() => {
        if (this.pending.has(id)) {
          this.pending.delete(id);
          reject(new Error('Request timeout'));
        }
      }, 30000);
    });
  }

  async getProcesses() {
    return this.request('brummer.getProcesses');
  }

  async startProcess(name, options = {}) {
    return this.request('brummer.startProcess', { name, ...options });
  }

  async getLogs(filters = {}) {
    return this.request('brummer.getLogs', filters);
  }

  handleNotification(params) {
    console.log('Notification:', params);
    // Handle real-time events
  }

  close() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

// Usage
const client = new BrummerWebSocketClient();

await client.connect();

const processes = await client.getProcesses();
console.log('Processes:', processes);

// Subscribe to events
await client.request('brummer.subscribe', {
  events: ['process.error', 'log.error']
});
```

### React Hook

```jsx
import { useState, useEffect, useCallback } from 'react';

function useBrummer(url = 'ws://localhost:3280') {
  const [client, setClient] = useState(null);
  const [connected, setConnected] = useState(false);
  const [processes, setProcesses] = useState([]);
  const [logs, setLogs] = useState([]);

  useEffect(() => {
    const brummerClient = new BrummerWebSocketClient(url);
    
    brummerClient.connect().then(() => {
      setClient(brummerClient);
      setConnected(true);
      
      // Subscribe to updates
      brummerClient.request('brummer.subscribe', {
        events: ['process.start', 'process.stop', 'log.error']
      });
      
      // Initial data fetch
      brummerClient.getProcesses().then(setProcesses);
    });

    return () => {
      brummerClient.close();
    };
  }, [url]);

  const startProcess = useCallback(async (name) => {
    if (!client) return;
    
    const result = await client.startProcess(name);
    if (result.success) {
      const updatedProcesses = await client.getProcesses();
      setProcesses(updatedProcesses.processes);
    }
    return result;
  }, [client]);

  const stopProcess = useCallback(async (name) => {
    if (!client) return;
    
    const result = await client.request('brummer.stopProcess', { name });
    if (result.success) {
      const updatedProcesses = await client.getProcesses();
      setProcesses(updatedProcesses.processes);
    }
    return result;
  }, [client]);

  const fetchLogs = useCallback(async (filters = {}) => {
    if (!client) return;
    
    const result = await client.getLogs(filters);
    setLogs(result.logs);
    return result;
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
        <div key={process.id}>
          <span>{process.name}</span>
          <span>{process.status}</span>
          {process.status === 'running' ? (
            <button onClick={() => stopProcess(process.name)}>Stop</button>
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
# Send MCP request via CLI
brum mcp-request brummer.getProcesses

# With parameters
brum mcp-request brummer.startProcess '{"name": "dev"}'

# Pretty print response
brum mcp-request brummer.getLogs '{"limit": 10}' | jq
```

### Shell Script Client

```bash
#!/bin/bash

# brummer-client.sh
brummer_request() {
  local method=$1
  local params=${2:-'{}'}
  
  echo "{\"jsonrpc\": \"2.0\", \"method\": \"$method\", \"params\": $params, \"id\": 1}" | \
    brum --mcp | \
    jq -r '.result'
}

# Get all processes
processes=$(brummer_request "brummer.getProcesses")
echo "Processes: $processes"

# Start a process
result=$(brummer_request "brummer.startProcess" '{"name": "dev"}')
echo "Start result: $result"

# Get recent errors
errors=$(brummer_request "brummer.getErrors" '{"limit": 5}')
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
    console.log('\nTesting getProcesses...');
    const processes = await client.getProcesses();
    console.log('✅ Processes:', processes);
    
    // Test starting a process
    console.log('\nTesting startProcess...');
    const startResult = await client.startProcess('dev');
    console.log('✅ Start result:', startResult);
    
    // Test getting logs
    console.log('\nTesting getLogs...');
    const logs = await client.getLogs({ limit: 5 });
    console.log('✅ Logs:', logs);
    
    // Test error detection
    console.log('\nTesting getErrors...');
    const errors = await client.request({
      method: 'brummer.getErrors'
    }, {});
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
   echo '{"jsonrpc":"2.0","method":"brummer.ping","id":1}' | brum --mcp
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