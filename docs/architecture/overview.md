# Brummer Architecture Overview

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

## Core Components Integration

### **Process Manager** (`internal/process/manager.go`)
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

### **Hub System** (`internal/mcp/`)
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

### **Proxy Server** (`internal/proxy/server.go`)
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

## Integration Patterns

### **Event-Driven Architecture**
Central EventBus (`pkg/events/events.go`) coordinates all components:

```
Process Events: ProcessStarted, ProcessExited, ProcessFailed
Log Events:     LogLine → URL Detection → Proxy Registration
Error Events:   ErrorDetected → Context Extraction → TUI Alerts
Build Events:   BuildStarted, BuildCompleted → Status Updates
Test Events:    TestStarted, TestPassed, TestFailed → Results
```

### **Data Flow Pipeline**
```
1. Process stdout/stderr → LogStore → EventDetector
2. EventDetector → EventBus → Component Updates
3. URL Detection → Proxy Server → Shareable URLs
4. MCP Requests → Hub Router → Instance Tools
5. Browser Actions → Proxy Server → Telemetry
```

### **Instance Discovery Flow**
```
1. Brummer Instance Starts → Registers JSON file in shared directory
2. Discovery System → Detects new file → Notifies ConnectionManager
3. ConnectionManager → Attempts connection → Updates state
4. Health Monitor → Periodic checks → Maintains connection state
5. Hub Tools → Route to active instances → Return results
```

### **MCP Tool Routing**
Tools are categorized and routed based on prefixes:

- **Single-Instance Tools**: `scripts_*`, `logs_*`, `proxy_*` (local instance)
- **Hub Tools**: `hub_*` prefix routes through ConnectionManager
- **Browser Tools**: `browser_*`, `repl_*` coordinate across instances
- **Session Tools**: Client sessions route to specific instances

## Deployment Modes

### **Single Instance Mode** (Default)
```bash
brum                    # TUI mode with local MCP server
brum --no-tui          # Headless mode, MCP server only
brum -p 8080           # Custom MCP port
```
- Process Manager handles local scripts
- MCP server exposes local tools only
- Proxy server for local URL management

### **Hub Mode** (Multi-Instance Coordination)
```bash
brum --mcp             # Stdio MCP transport for external clients
```
- Runs as MCP hub without TUI
- Discovers and coordinates multiple instances
- Routes tools between instances
- Session management for clients

### **Configuration Chain**
```
Command Line Args → Current Directory Config → Parent Dir → ~/.brum.toml
```

## Key Design Patterns

1. **Event-Driven Communication**: Components communicate via EventBus, avoiding tight coupling
2. **State Machine Management**: Connection states with explicit transitions and timing
3. **Discovery-Based Architecture**: File-based instance discovery enables dynamic coordination
4. **Session-Based Routing**: MCP tools route through session-to-instance mapping
5. **Graceful Degradation**: System continues operating when components fail
6. **Resource Cleanup**: Automatic cleanup of processes, connections, and temporary files

## Error Handling & Recovery

- **Process Failures**: Automatic restart detection and status updates
- **Connection Failures**: Health monitoring with exponential backoff retry
- **Discovery Failures**: Stale instance cleanup and re-discovery
- **Proxy Failures**: URL re-registration and port conflict resolution
- **Session Failures**: Automatic session cleanup and reconnection