# Brummer Hub Mode Documentation

## Overview

Brummer Hub Mode enables MCP clients (like Claude Desktop, VSCode, or Cursor) to discover and control multiple brummer instances through a single MCP connection. Instead of configuring each project separately, you configure the hub once and it automatically discovers all running brummer instances on your system.

## Architecture

```
┌─────────────────┐     stdio      ┌─────────────────┐
│   MCP Client    │◄──────────────►│   Brummer Hub   │
│ (Claude/VSCode) │                │   (brum --mcp)  │
└─────────────────┘                └────────┬────────┘
                                            │ HTTP/MCP
                                   ┌────────┴────────┐
                                   │                 │
                              ┌────▼───┐        ┌────▼───┐
                              │Instance│        │Instance│
                              │  :7778 │        │  :7779 │
                              └────────┘        └────────┘
```

## Installation

### 1. Configure MCP Client

Add the hub to your MCP client configuration:

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):
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

**VSCode/Cursor** (using MCP extension):
```json
{
  "mcp.servers": {
    "brummer-hub": {
      "command": "brum",
      "args": ["--mcp"]
    }
  }
}
```

### 2. Start Brummer Instances

In each project directory, run brummer normally:
```bash
# Project 1
cd ~/projects/frontend
brum

# Project 2
cd ~/projects/backend
brum

# Project 3
cd ~/projects/api
brum
```

Each instance will automatically register itself with the discovery system.

## Usage

### Hub Tools

The hub provides these tools to MCP clients:

#### `instances/list`
Lists all running brummer instances:
```json
[
  {
    "id": "frontend-a1b2c3d4",
    "name": "frontend",
    "directory": "/home/user/projects/frontend",
    "port": 7777,
    "process_pid": 12345,
    "state": "active",
    "connected": true
  },
  {
    "id": "backend-e5f6g7h8",
    "name": "backend",
    "directory": "/home/user/projects/backend",
    "port": 7778,
    "process_pid": 12346,
    "state": "active",
    "connected": true
  }
]
```

#### `instances/connect`
Connects to a specific instance:
```json
{
  "instance_id": "frontend-a1b2c3d4"
}
```

After connecting, all tools from that instance become available with prefixed names:
- `frontend-a1b2c3d4/scripts/list`
- `frontend-a1b2c3d4/scripts/run`
- `frontend-a1b2c3d4/logs/stream`
- etc.

#### `instances/disconnect`
Disconnects from the current instance.

### Tool Proxying

Once connected to an instance, all its tools are available with the instance ID prefix:

```
Original tool: scripts/run
Proxied tool: frontend-a1b2c3d4/scripts/run
```

The hub automatically forwards tool calls to the appropriate instance.

### Example Workflow

1. **List available instances:**
   ```
   User: Show me all running brummer instances
   Assistant: I'll list the available instances...
   [Uses instances/list tool]
   ```

2. **Connect to an instance:**
   ```
   User: Connect to the frontend project
   Assistant: I'll connect to the frontend instance...
   [Uses instances/connect with instance_id: "frontend-a1b2c3d4"]
   ```

3. **Use instance tools:**
   ```
   User: Start the dev server
   Assistant: I'll start the dev server...
   [Uses frontend-a1b2c3d4/scripts/run with name: "dev"]
   ```

4. **Switch instances:**
   ```
   User: Now let's work on the backend
   Assistant: I'll switch to the backend instance...
   [Uses instances/connect with instance_id: "backend-e5f6g7h8"]
   ```

## Features

### Automatic Discovery

- Instances are discovered automatically when they start
- No manual configuration needed for each project
- Discovery works across all user projects

### Health Monitoring

- Hub monitors instance health with MCP ping/pong
- Dead instances are automatically marked and cleaned up
- Connection states: Active → Retrying → Dead

### Session Management

- Each MCP client gets its own session
- Sessions can connect to different instances
- Session state is preserved across tool calls

### Resource Management

- Tools are dynamically registered when connecting
- Resources and prompts are discovered (future feature)
- Automatic cleanup when disconnecting

## Configuration

### Hub Configuration

The hub respects the same `.brum.toml` configuration:

```toml
# Hub-specific settings
[hub]
discovery_interval = "5s"
health_check_interval = "10s"
max_ping_failures = 3
cleanup_interval = "60s"
```

### Instance Registration

Each instance registers itself at:
- Linux/Mac: `~/.local/share/brummer/instances/`
- Windows: `%APPDATA%\brummer\instances\`

Registration files contain:
```json
{
  "id": "frontend-a1b2c3d4",
  "name": "frontend",
  "directory": "/home/user/projects/frontend",
  "port": 7777,
  "pid": 12345,
  "started_at": "2024-01-20T10:30:00Z",
  "last_ping": "2024-01-20T10:35:00Z"
}
```

## Troubleshooting

### No instances found

1. Check if instances are running:
   ```bash
   ls ~/.local/share/brummer/instances/
   ```

2. Verify instances have MCP enabled:
   ```bash
   # Should NOT use --no-mcp flag
   brum  # ✓ Correct
   brum --no-mcp  # ✗ Won't be discoverable
   ```

### Connection failures

1. Check instance health:
   - Look for "Instance X became unhealthy" in hub logs
   - Verify the instance process is still running

2. Check port availability:
   ```bash
   lsof -i :7777  # Check default port
   lsof -i :7778  # Check next port
   ```

### Stale instances

The hub automatically cleans up stale instances, but you can manually clean:
```bash
rm ~/.local/share/brummer/instances/*.json
```

## Limitations

### Current Limitations

1. **Library constraints**: Due to mark3labs MCP library limitations:
   - Resources cannot be dynamically proxied
   - Prompts cannot be dynamically proxied
   - Only tools are fully proxied

2. **Single session**: The stdio transport supports one session at a time

3. **Local only**: Hub only discovers instances on the local machine

### Future Enhancements

1. **Remote instances**: SSH tunneling for remote development
2. **Instance groups**: Tag and manage related instances
3. **Persistent sessions**: Reconnect to previous instance automatically
4. **Full proxying**: Resources and prompts when library supports it

## Security

- Hub only accepts stdio connections (no network exposure)
- Instances only accept localhost connections
- No authentication needed (local user only)
- Instance files are user-readable only

## Example Use Cases

### 1. Microservices Development

Managing multiple services:
```
frontend/ (React app on :3000)
backend/ (Node.js API on :4000)
auth/ (Auth service on :5000)
database/ (DB migrations)
```

All accessible through one hub connection.

### 2. Monorepo Management

Different packages in a monorepo:
```
packages/
  ui/ (Component library)
  api/ (API client)
  docs/ (Documentation site)
  cli/ (CLI tool)
```

Switch between packages without reconfiguring.

### 3. Client Projects

Multiple client projects:
```
client-a/
client-b/
client-c/
internal-tools/
```

One hub configuration for all projects.