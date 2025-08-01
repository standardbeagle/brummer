# MCP Tool Categories & Routing

## Single-Instance Tools (Local Execution)
These tools execute on the local instance:

### Script Management:
- **scripts_list**: List all npm/yarn/pnpm/bun scripts from package.json
- **scripts_run**: Execute a script with real-time output streaming
- **scripts_stop**: Stop a running script process
- **scripts_status**: Check the status of running scripts

### Log Management:
- **logs_stream**: Stream real-time logs from all processes (supports filtering)
- **logs_search**: Search historical logs with regex patterns and filters

### Proxy & Browser:
- **proxy_requests**: Get captured HTTP requests from the proxy server
- **browser_open**: Open URLs with automatic proxy configuration
- **browser_refresh**: Refresh connected browser tabs
- **browser_navigate**: Navigate browser tabs to new URLs
- **browser_screenshot**: Capture screenshots of browser tabs
- **repl_execute**: Execute JavaScript in browser context

### Telemetry:
- **telemetry_sessions**: Access browser telemetry session data
- **telemetry_events**: Stream real-time browser telemetry events

## Hub Tools (Multi-Instance Coordination)
These tools are only available in hub mode and route to instances:

### Instance Management:
- **instances_list**: List all discovered instances with connection states
- **instances_connect**: Connect to a specific instance (session routing)
- **instances_disconnect**: Disconnect from current instance

### Routed Tools (with `hub_` prefix):
- **hub_scripts_list**: Route scripts_list to connected instance
- **hub_scripts_run**: Route scripts_run to connected instance
- **hub_logs_stream**: Route logs_stream to connected instance
- **hub_browser_screenshot**: Route browser_screenshot to connected instance
- **hub_repl_execute**: Route repl_execute to connected instance
- (All single-instance tools available with `hub_` prefix)

## Session Management & Routing

### Session-Based Tool Routing
```
1. Client connects to hub with session ID
2. Client calls instances_connect with target instance ID
3. Session is mapped to instance
4. Subsequent hub_* tools route to mapped instance
5. Client can disconnect and connect to different instance
```

### Connection State Management
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

## Tool Execution Flow

### Single Instance Flow
```
MCP Client → HTTP Request → Streamable Server → Tool Handler → Response
```

### Hub Mode Flow
```
MCP Client → stdio → Hub Server → Connection Manager → Instance Client
                                       ↓
Instance Server ← HTTP Request ← Hub Client ← Tool Router
      ↓
Tool Handler → Response → Hub Client → Connection Manager → Hub Server
                                              ↓
                                        stdio → MCP Client
```

## MCP Resources
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

## MCP Prompts
Pre-configured debugging prompts:
- **debug_error**: Analyze error logs and suggest fixes
- **performance_analysis**: Analyze telemetry data for performance issues
- **api_troubleshooting**: Examine proxy requests to debug API issues
- **script_configuration**: Help configure npm scripts for common tasks

## MCP Capabilities
- Real-time streaming support for tools marked with `Streaming: true`
- Resource subscription for live updates via WebSocket or SSE
- Session management with automatic cleanup
- Cross-platform compatibility (Windows, macOS, Linux, WSL2)