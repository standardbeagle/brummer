# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Building and Running
```bash
# Build the binary
make build                    # Creates ./brum executable
go build -o brum ./cmd/brum/main.go  # Alternative direct build

# Run directly
make run                      # Build and run
./brum                       # Run in directory with package.json
./brum -d ../other-project   # Run in different directory

# Development with hot reload
make dev                      # Uses air for auto-reload

# Installation
make install-user            # Install to ~/.local/bin
make install                 # System-wide install (requires sudo)
```

### Testing and Quality
```bash
# Run tests
make test                    # or: go test -v ./...
go test -v ./internal/logs   # Test specific package

# Code quality
make fmt                     # Format code with go fmt
make lint                    # Run golangci-lint

# Build for all platforms
make build-all               # Creates binaries in ./dist/
```

### CLI Usage
```bash
# Run with CLI arguments to start scripts directly
brum dev                     # Start 'dev' script and switch to logs view
brum dev test               # Start multiple scripts
brum 'node server.js'       # Run arbitrary command
brum -d ../app dev          # Run in different directory

# Options
brum --no-mcp               # Disable MCP server
brum --no-tui               # Run headless (MCP only)
brum -p 8888                # Custom MCP port (default: 7777)
brum --settings             # Show current configuration with sources

# Configuration
brum --settings > .brum.example.toml  # Create example config file
```

## Architecture

Brummer's architecture consists of three main integrated systems: **Process Manager**, **Hub**, and **Proxy Server**. These components work together to provide comprehensive development environment management with MCP integration.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Brummer Architecture                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐         │
│  │ Process Manager │    │   Hub System    │    │  Proxy Server   │         │
│  │                 │    │                 │    │                 │         │
│  │ • npm/yarn/pnpm │    │ • Instance Mgmt │    │ • URL Discovery │         │
│  │ • Script Running│    │ • MCP Routing   │    │ • Request Proxy │         │
│  │ • Log Capture   │────┤ • Health Monitor│────┤ • Telemetry     │         │
│  │ • Event Emission│    │ • Discovery     │    │ • Browser Tools │         │
│  └─────────────────┘    │ • Session Mgmt  │    └─────────────────┘         │
│           │              └─────────────────┘             │                 │
│           │                       │                      │                 │
│           └───────────────────────┼──────────────────────┘                 │
│                                   │                                        │
│                          ┌─────────────────┐                               │
│                          │   Event Bus     │                               │
│                          │                 │                               │
│                          │ • ProcessEvents │                               │
│                          │ • LogLines      │                               │
│                          │ • ErrorEvents   │                               │
│                          │ • URL Detection │                               │
│                          └─────────────────┘                               │
│                                   │                                        │
│                          ┌─────────────────┐                               │
│                          │      TUI        │                               │
│                          │                 │                               │
│                          │ • View Updates  │                               │
│                          │ • User Commands │                               │
│                          │ • Status Display│                               │
│                          └─────────────────┘                               │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Core Components Integration

#### **Process Manager** (`internal/process/manager.go`)
The Process Manager is the foundation that spawns and manages child processes:

- **Script Execution**: Supports npm, yarn, pnpm, bun package managers with automatic detection
- **Process Lifecycle**: Tracks process status (pending → running → stopped/success/failed)
- **Log Processing**: Captures stdout/stderr and emits LogLine events via EventBus
- **URL Detection**: Automatically discovers URLs in process logs for proxy registration
- **Monorepo Support**: Detects workspaces (pnpm/npm/yarn, Lerna, Nx, Rush)
- **Multi-Language**: Auto-detects executables (Go, Rust, Java, Python, Node.js, etc.)
- **Instance Registration**: Automatically registers itself for hub discovery

**Process Lifecycle Flow:**
```
CLI Args → Process Manager → Script/Command Execution
    ↓                              ↓
Event Bus ← Log Processing ← Process stdout/stderr
    ↓                              ↓
TUI/MCP Updates              URL Detection → Proxy Registration
```

#### **Hub System** (`internal/mcp/`)
The Hub manages multiple Brummer instances and coordinates MCP tool routing:

**Connection Manager** (`connection_manager.go`):
- **Instance States**: discovered → connecting → active → retrying → dead
- **Health Monitoring**: Periodic health checks with automatic reconnection
- **Session Tracking**: Maps client sessions to active instances
- **State Transitions**: Records timing and reasons for debugging

**Discovery System** (`internal/discovery/`):
- **File-Based Discovery**: Watches shared directory for instance JSON files
- **Instance Metadata**: ID, name, directory, port, PID, startup time
- **Automatic Cleanup**: Removes stale instances based on process checks
- **Real-Time Updates**: Notifies connection manager of changes

**Hub Client** (`hub_client.go`):
- **HTTP Communication**: JSON-RPC 2.0 client for instance MCP servers
- **Tool Routing**: Routes tools/call requests to appropriate instances
- **Session Management**: Handles connection initialization and cleanup

**Hub Mode** (stdio MCP transport):
- **Multi-Instance Coordination**: Central hub for MCP clients
- **Tool Aggregation**: Exposes tools from all connected instances
- **Session Routing**: Routes client sessions to specific instances

#### **Proxy Server** (`internal/proxy/server.go`)
The Proxy Server provides HTTP interception and browser automation:

**Proxy Modes**:
- **Reverse Mode** (default): Creates shareable URLs for detected endpoints
- **Full Mode**: Traditional HTTP proxy requiring browser configuration

**URL Discovery & Management**:
- **Automatic Detection**: Monitors process logs for HTTP/HTTPS URLs
- **Intelligent Labeling**: Extracts meaningful names from log context
- **Process Association**: Maps URLs to originating processes
- **Port Allocation**: Auto-assigns unique ports for reverse proxy

**Browser Automation**:
- **Screenshot Capture**: Supports PNG, JPEG, WebP formats
- **JavaScript Execution**: Browser REPL with session management
- **Telemetry Collection**: Page performance and error tracking
- **Request Interception**: Captures and analyzes HTTP traffic

### Integration Patterns

#### **Event-Driven Architecture**
Central EventBus (`pkg/events/events.go`) coordinates all components:

```
Process Events: ProcessStarted, ProcessExited, ProcessFailed
Log Events:     LogLine → URL Detection → Proxy Registration
Error Events:   ErrorDetected → Context Extraction → TUI Alerts
Build Events:   BuildStarted, BuildCompleted → Status Updates
Test Events:    TestStarted, TestPassed, TestFailed → Results
```

#### **Data Flow Pipeline**
```
1. Process stdout/stderr → LogStore → EventDetector
2. EventDetector → EventBus → Component Updates
3. URL Detection → Proxy Server → Shareable URLs
4. MCP Requests → Hub Router → Instance Tools
5. Browser Actions → Proxy Server → Telemetry
```

#### **Instance Discovery Flow**
```
1. Brummer Instance Starts → Registers JSON file in shared directory
2. Discovery System → Detects new file → Notifies ConnectionManager
3. ConnectionManager → Attempts connection → Updates state
4. Health Monitor → Periodic checks → Maintains connection state
5. Hub Tools → Route to active instances → Return results
```

#### **MCP Tool Routing**
Tools are categorized and routed based on prefixes:

- **Single-Instance Tools**: `scripts_*`, `logs_*`, `proxy_*` (local instance)
- **Hub Tools**: `hub_*` prefix routes through ConnectionManager
- **Browser Tools**: `browser_*`, `repl_*` coordinate across instances
- **Session Tools**: Client sessions route to specific instances

### Deployment Modes

#### **Single Instance Mode** (Default)
```bash
brum                    # TUI mode with local MCP server
brum --no-tui          # Headless mode, MCP server only
brum -p 8080           # Custom MCP port
```
- Process Manager handles local scripts
- MCP server exposes local tools only
- Proxy server for local URL management

#### **Hub Mode** (Multi-Instance Coordination)
```bash
brum --mcp             # Stdio MCP transport for external clients
```
- Runs as MCP hub without TUI
- Discovers and coordinates multiple instances
- Routes tools between instances
- Session management for clients

#### **Configuration Chain**
```
Command Line Args → Current Directory Config → Parent Dir → ~/.brum.toml
```

### Key Design Patterns

1. **Event-Driven Communication**: Components communicate via EventBus, avoiding tight coupling
2. **State Machine Management**: Connection states with explicit transitions and timing
3. **Discovery-Based Architecture**: File-based instance discovery enables dynamic coordination
4. **Session-Based Routing**: MCP tools route through session-to-instance mapping
5. **Graceful Degradation**: System continues operating when components fail
6. **Resource Cleanup**: Automatic cleanup of processes, connections, and temporary files

### Concurrency Patterns & Race Condition Prevention

#### **Critical Guidelines for Safe Concurrent Programming**

**TUI Components (BubbleTea Architecture)**:
- ✅ **DO**: Use message-passing via tea.Cmd for all state changes
- ✅ **DO**: Handle state modifications only in Update() method  
- ❌ **DON'T**: Modify Model state directly from goroutines
- ❌ **DON'T**: Call methods like `m.logStore.Add()` from tea.Cmd functions

```go
// ✅ CORRECT: Message-based state update
func (m *Model) handleRestartProcess(proc *process.Process) tea.Cmd {
    return func() tea.Msg {
        err := m.processMgr.StopProcess(proc.ID)
        return restartProcessMsg{
            processName: proc.Name,
            message:     fmt.Sprintf("Restart result: %v", err),
            isError:     err != nil,
        }
    }
}

// ❌ INCORRECT: Direct state modification from goroutine
func (m *Model) handleBadRestart(proc *process.Process) tea.Cmd {
    return func() tea.Msg {
        m.logStore.Add("system", "System", "Restarting...", false) // RACE CONDITION
        return processUpdateMsg{}
    }
}
```

**EventBus Usage**:
- ✅ **DO**: Use worker pools with bounded goroutines (CPU cores × 2.5)
- ✅ **DO**: Implement graceful degradation when pools are full
- ❌ **DON'T**: Create unlimited goroutines for event handling

**Process Manager**:
- ✅ **DO**: Use RWMutex for concurrent map operations
- ✅ **DO**: Implement consistent lock ordering to prevent deadlocks  
- ❌ **DON'T**: Access shared maps without synchronization

**Testing Requirements**:
- ✅ **ALWAYS**: Run `go test -race -v ./...` before commits
- ✅ **ALWAYS**: Use `make test-race` for targeted race detection
- ✅ **CI/CD**: Race detection integrated in GitHub Actions pipeline

#### **Code Review Checklist**

Before approving any changes involving concurrency:

1. **TUI Changes**: 
   - [ ] All Model state changes go through Update() method
   - [ ] No direct state modification in tea.Cmd goroutines
   - [ ] Message types defined for complex state updates

2. **EventBus Changes**:
   - [ ] Worker pool limits respected (bounded goroutines)
   - [ ] Graceful handling when pools are full
   - [ ] Proper event handler cleanup

3. **Shared State Access**:
   - [ ] Mutexes used for concurrent map/slice operations
   - [ ] Consistent lock ordering documented
   - [ ] Read/write separation with RWMutex where applicable

4. **Testing Validation**:
   - [ ] `go test -race` passes on modified packages
   - [ ] Integration tests include concurrent scenarios
   - [ ] Performance impact assessed (< 10% regression)

### Error Handling & Recovery

- **Process Failures**: Automatic restart detection and status updates
- **Connection Failures**: Health monitoring with exponential backoff retry
- **Discovery Failures**: Stale instance cleanup and re-discovery
- **Proxy Failures**: URL re-registration and port conflict resolution
- **Session Failures**: Automatic session cleanup and reconnection

## MCP (Model Context Protocol) Integration

Brummer provides comprehensive MCP integration with two operational modes: **Single Instance** and **Hub Mode** for coordinating multiple instances.

### Deployment Architectures

#### **Single Instance Mode** (Default)
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   MCP Client    │────▶│ Brummer Instance│────▶│   Local Tools   │
│ (Claude/VSCode) │     │  (Port 7777)    │     │ • scripts_*     │
└─────────────────┘     └─────────────────┘     │ • logs_*        │
                                                │ • proxy_*       │
                                                │ • browser_*     │
                                                └─────────────────┘
```

#### **Hub Mode** (Multi-Instance Coordination)
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   MCP Client    │────▶│  Brummer Hub    │────▶│   Instance A    │
│ (Claude/VSCode) │     │ (stdio transport│     │  (Port 7778)    │
└─────────────────┘     │   discovery +   │     └─────────────────┘
                        │  routing)       │     ┌─────────────────┐
                        └─────────────────┘────▶│   Instance B    │
                                                │  (Port 7779)    │
                                                └─────────────────┘
                                                ┌─────────────────┐
                                               ▶│   Instance C    │
                                                │  (Port 7780)    │
                                                └─────────────────┘
```

### Server Configuration

#### **Single Instance Configuration**
- **Primary Endpoint**: `http://localhost:7777/mcp` (single URL for all MCP functionality)
- **Protocol**: JSON-RPC 2.0 with Server-Sent Events streaming support
- **Default Port**: 7777 (configurable with `-p` or `--port`)
- **Startup**: Automatically enabled unless `--no-mcp` flag is used

#### **Hub Mode Configuration**
- **Transport**: stdio (JSON-RPC over stdin/stdout)
- **Discovery**: File-based instance discovery in shared directory
- **Routing**: Automatic tool routing to appropriate instances
- **Session Management**: Client session to instance mapping

### Client Configuration

#### **Single Instance Setup**
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

#### **Hub Mode Setup** (Recommended for Multiple Projects)
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

### MCP Tool Categories & Routing

#### **Single-Instance Tools** (Local Execution)
These tools execute on the local instance:

**Script Management:**
- **scripts_list**: List all npm/yarn/pnpm/bun scripts from package.json
- **scripts_run**: Execute a script with real-time output streaming
- **scripts_stop**: Stop a running script process
- **scripts_status**: Check the status of running scripts

**Log Management:**
- **logs_stream**: Stream real-time logs from all processes (supports filtering)
- **logs_search**: Search historical logs with regex patterns and filters

**Proxy & Browser:**
- **proxy_requests**: Get captured HTTP requests from the proxy server
- **browser_open**: Open URLs with automatic proxy configuration
- **browser_refresh**: Refresh connected browser tabs
- **browser_navigate**: Navigate browser tabs to new URLs
- **browser_screenshot**: Capture screenshots of browser tabs
- **repl_execute**: Execute JavaScript in browser context

**Telemetry:**
- **telemetry_sessions**: Access browser telemetry session data
- **telemetry_events**: Stream real-time browser telemetry events

#### **Hub Tools** (Multi-Instance Coordination)
These tools are only available in hub mode and route to instances:

**Instance Management:**
- **instances_list**: List all discovered instances with connection states
- **instances_connect**: Connect to a specific instance (session routing)
- **instances_disconnect**: Disconnect from current instance

**Routed Tools** (with `hub_` prefix):
- **hub_scripts_list**: Route scripts_list to connected instance
- **hub_scripts_run**: Route scripts_run to connected instance
- **hub_logs_stream**: Route logs_stream to connected instance
- **hub_browser_screenshot**: Route browser_screenshot to connected instance
- **hub_repl_execute**: Route repl_execute to connected instance
- (All single-instance tools available with `hub_` prefix)

### Session Management & Routing

#### **Session-Based Tool Routing**
```
1. Client connects to hub with session ID
2. Client calls instances_connect with target instance ID
3. Session is mapped to instance
4. Subsequent hub_* tools route to mapped instance
5. Client can disconnect and connect to different instance
```

#### **Connection State Management**
Instances progress through states with automatic health monitoring:

```
discovered → connecting → active → [retrying] → dead
     ↑                      ↓           ↑
     └──── cleanup ←────────┴───────────┘
```

**State Transitions:**
- **discovered**: Instance file found, not yet connected
- **connecting**: Attempting initial connection
- **active**: Connected and responsive to health checks
- **retrying**: Connection lost, attempting reconnection
- **dead**: Maximum retries exceeded, marked for cleanup

### Tool Execution Flow

#### **Single Instance Flow**
```
MCP Client → HTTP Request → Streamable Server → Tool Handler → Response
```

#### **Hub Mode Flow**
```
MCP Client → stdio → Hub Server → Connection Manager → Instance Client
                                       ↓
Instance Server ← HTTP Request ← Hub Client ← Tool Router
      ↓
Tool Handler → Response → Hub Client → Connection Manager → Hub Server
                                              ↓
                                        stdio → MCP Client
```

### MCP Resources
Structured data access via resources:
- `logs://recent`: Recent log entries from all processes
- `logs://errors`: Recent error log entries only
- `telemetry://sessions`: Active browser telemetry sessions
- `telemetry://errors`: JavaScript errors from browser sessions
- `telemetry://console-errors`: Console error output (console.error calls)
- `proxy://requests`: Recent HTTP requests captured by proxy
- `proxy://mappings`: Active reverse proxy URL mappings
- `processes://active`: Currently running processes
- `scripts://available`: Scripts defined in package.json

### MCP Prompts
Pre-configured debugging prompts:
- **debug_error**: Analyze error logs and suggest fixes
- **performance_analysis**: Analyze telemetry data for performance issues
- **api_troubleshooting**: Examine proxy requests to debug API issues
- **script_configuration**: Help configure npm scripts for common tasks

### MCP Capabilities
- Real-time streaming support for tools marked with `Streaming: true`
- Resource subscription for live updates via WebSocket or SSE
- Session management with automatic cleanup
- Cross-platform compatibility (Windows, macOS, Linux, WSL2)

### MCP Connection Types (Streamable HTTP Transport)
The server implements the official MCP Streamable HTTP transport protocol:

1. **Standard JSON-RPC** (POST to `/mcp` with `Accept: application/json`):
   - Single request/response
   - Batch requests supported

2. **Server-Sent Events** (GET to `/mcp` with `Accept: text/event-stream`):
   - Server-to-client streaming
   - Supports resource subscriptions with real-time updates
   - Automatic heartbeat/ping messages

3. **SSE Response** (POST to `/mcp` with `Accept: text/event-stream`):
   - Client sends requests via POST
   - Server responds with SSE stream
   - Useful for streaming tool responses

Headers:
- `Accept`: Must include appropriate content type
- `Mcp-Session-Id`: Optional session identifier for resumability
- `Content-Type`: `application/json` for requests

Example SSE connection:
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

## Practical Examples & Configuration

### Multi-Project Development Workflow

#### **Scenario: Frontend + Backend + Database**

**Step 1: Set up individual instances**
```bash
# Terminal 1: Frontend (React/Vite)
cd frontend/
brum dev                 # Starts on port 7777

# Terminal 2: Backend (Node.js API)  
cd backend/
brum -p 7778 dev        # Starts on port 7778

# Terminal 3: Database utilities
cd database/
brum -p 7779 migrate    # Starts on port 7779
```

**Step 2: Configure MCP hub for coordination**
```json
// Claude Desktop config
{
  "servers": {
    "my-project-hub": {
      "command": "brum",
      "args": ["--mcp"]
    }
  }
}
```

**Step 3: Use hub tools to coordinate**
```bash
# List all running instances
instances_list

# Connect to frontend instance
instances_connect frontend-abc123

# Run frontend-specific commands
hub_scripts_list        # Lists frontend package.json scripts
hub_logs_stream         # Streams frontend logs
hub_browser_screenshot  # Takes screenshot of frontend

# Switch to backend
instances_connect backend-def456
hub_logs_search "error" # Search backend logs for errors
```

### Proxy Configuration Examples

#### **Automatic URL Detection**
Brummer automatically detects URLs in process logs:

```bash
# Start development server
brum dev

# Logs show: "Local: http://localhost:3000"
# Brummer automatically:
# 1. Detects the URL
# 2. Creates reverse proxy: http://localhost:20888
# 3. Makes it shareable across network
```

#### **Manual Proxy Configuration**
```bash
# Start with specific URL proxying
brum --proxy-url http://localhost:3000 dev

# Use traditional HTTP proxy mode
brum --proxy-mode full --proxy-port 8888

# Configure browser to use proxy:
# HTTP Proxy: localhost:8888
# PAC URL: http://localhost:8888/proxy.pac
```

#### **Multiple URL Handling**
```bash
# Start multiple services
brum "npm run dev & npm run api & npm run docs"

# Brummer detects and proxies:
# Frontend: http://localhost:3000 → http://localhost:20888
# API:      http://localhost:3001 → http://localhost:20889  
# Docs:     http://localhost:3002 → http://localhost:20890
```

### Browser Automation Examples

#### **Screenshot Workflow**
```javascript
// Single instance
browser_screenshot({
  "format": "png",
  "fullPage": true,
  "quality": 90
})

// Hub mode (routes to connected instance)
hub_browser_screenshot({
  "format": "jpeg", 
  "selector": "#main-content",
  "quality": 85
})
```

#### **JavaScript Testing**
```javascript
// Execute JavaScript in browser
repl_execute({
  "code": "document.title = 'Test'; return document.title;"
})

// Hub mode with session
instances_connect("frontend-instance")
hub_repl_execute({
  "code": "console.log('Testing frontend'); return window.location.href;"
})
```

### Advanced Configuration

#### **Multi-Instance Hub Setup**
```toml
# ~/.brum.toml
[instances]
discovery_dir = "~/.brum/instances"
cleanup_interval = "1m"
stale_timeout = "5m"

[hub]
health_check_interval = "30s" 
max_retry_attempts = 3
retry_backoff = "exponential"

[proxy]
mode = "reverse"
base_port = 20888
enable_telemetry = true
```

#### **Project-Specific Configuration**
```toml
# project/.brum.toml
[process]
preferred_package_manager = "pnpm"

[proxy]
mode = "reverse"
proxy_url = "http://localhost:3000"

[mcp]
port = 7777
enable_browser_tools = true
```

### Troubleshooting Common Issues

#### **Instance Discovery Problems**
```bash
# Check instance discovery
ls ~/.brum/instances/          # Should show JSON files

# Manually clean up stale instances
rm ~/.brum/instances/*.json

# Check instance connectivity
instances_list                 # Shows connection states
```

#### **Port Conflicts**
```bash
# Find available port
brum --port 0                  # Auto-assign available port

# Check what's using a port
lsof -i :7777                  # macOS/Linux
netstat -ano | findstr :7777   # Windows
```

#### **Proxy Issues**
```bash
# Reset proxy configuration
brum --no-proxy               # Disable proxy temporarily

# Check proxy mappings
proxy_requests                # Show captured requests

# Force URL re-detection
# Restart process to trigger URL detection
```

#### **Health Monitoring Debug**
```bash
# Instance health information
instances_list | jq '.[] | {id, state, retry_count, time_in_state}'

# Connection state history
instances_list | jq '.[] | .state_stats'
```

### Integration with External Tools

#### **VS Code Integration**
```json
// .vscode/settings.json
{
  "mcp.servers": {
    "brummer": {
      "command": "brum",
      "args": ["--no-tui", "--port", "7777"]
    }
  }
}
```

#### **CI/CD Integration**
```bash
# Headless operation for CI
brum --no-tui --no-proxy test

# Export test results
logs_search "test.*" > test-results.log

# Health check endpoint
curl http://localhost:7777/mcp -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

#### **Docker Integration**
```dockerfile
# Dockerfile
FROM node:18
COPY . /app
WORKDIR /app
RUN npm install && npm install -g brum
EXPOSE 7777 20888-20899
CMD ["brum", "--no-tui", "--port", "7777"]
```

### Performance Optimization

#### **Log Management**
```toml
# .brum.toml
[logs]
max_entries = 10000          # Limit memory usage
max_line_length = 2048       # Truncate long lines
enable_url_detection = true  # Auto-detect URLs
```

#### **Resource Limits**
```bash
# Limit process resources
ulimit -n 1024              # File descriptor limit
ulimit -u 256               # Process limit

# Monitor resource usage
logs_search "memory|cpu"    # Search for resource logs
```

## Important Notes

- The executable is named `brum` (not `brummer`)
- Browser extension code has been removed from the codebase
- The TUI requires a TTY; use `--no-tui` for headless operation
- MCP server runs on port 7777 by default with single endpoint `/mcp`
- Slash commands use Go regex syntax for pattern matching
- Process IDs are generated as `<scriptname>-<timestamp>`
- URLs are automatically extracted from logs and deduplicated per process
- Hub mode requires file-based discovery for instance coordination
- Proxy reverse mode creates shareable URLs for detected endpoints
- Health monitoring maintains connection state with automatic recovery