# Brummer MCP Hub Architecture - Product Requirements Document

## Executive Summary

This document outlines the complete architecture for the Brummer MCP (Model Context Protocol) Hub system, which enables MCP clients to discover and control multiple brummer process manager instances through a single stdio connection. The hub acts as a multiplexer, allowing LLM tools to find and interact with the correct process manager and browser data for their tasks.

## Problem Statement

Current limitations:
1. MCP clients can only connect to one brummer instance at a time
2. Users must manually configure each project's MCP connection
3. No way to discover running brummer instances across the system
4. Error messages when connecting to non-existent instances

## Solution Overview

A hub-and-spoke architecture where:
- **Hub**: Runs as an MCP server over stdio, providing instance discovery and proxying
- **Instances**: Individual brummer process managers with HTTP-based MCP servers
- **Discovery**: File-based signals with active network connections as source of truth
- **State Management**: Channel-based, lock-free design following Go best practices

## User Stories

### 1. MCP Client Installation
**As a** developer using Claude Desktop/VSCode  
**I want to** install `brum --mcp` as a single MCP server configuration  
**So that** I can access all my running brummer instances without manual configuration

**Acceptance Criteria:**
- Hub starts instantly (< 100ms) over stdio
- Responds to MCP ping using latest protocol version
- Single configuration in MCP client settings:
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

### 2. Instance Discovery
**As a** developer with multiple projects  
**I want to** see all running brummer instances  
**So that** I can choose which project to control

**Acceptance Criteria:**
- `instances/list` tool shows all active instances
- Each instance shows: name, path, port, PID, start time
- List updates in real-time as instances start/stop
- No stale entries from dead processes

### 3. Instance Connection
**As a** developer using LLM tools  
**I want to** connect to a specific brummer instance  
**So that** I can control that project's processes

**Acceptance Criteria:**
- `instances/connect` establishes connection to instance
- All instance tools become available after connection
- Multiple clients can connect to same instance
- Connection persists until explicitly disconnected

### 4. Zero Errors on Startup
**As a** developer  
**I want** the hub to start without errors  
**Even when** no brummer instances are running

**Acceptance Criteria:**
- Hub starts successfully with empty instance list
- No error messages shown to user
- Instances appear as they start
- Graceful handling of all edge cases

## Technical Architecture

### Component Overview

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

### Hub Components

#### 1. Main Server (`cmd/brum/main.go`)
- Detects `--mcp` flag to run in hub mode
- Initializes stdio transport (never HTTP)
- Sets up discovery and connection systems

#### 2. MCP Hub Server (`internal/mcp/hub_server.go`)
- Implements mcp.Server interface
- Provides hub-specific tools:
  - `instances/list`
  - `instances/connect`
  - `instances/disconnect`
- Manages session-to-instance mapping

#### 3. Instance Discovery (`internal/discovery/`)
- **Watcher**: Monitors instance registration files
- **Registry**: Writes instance files after MCP server is listening
- **File Location**: OS-specific config directories
  - Linux/Mac: `~/.local/share/brummer/instances/`
  - Windows: `%APPDATA%\brummer\instances\`

#### 4. Connection Manager (`internal/mcp/connection_manager.go`)
- Channel-based state management (no mutexes)
- Tracks instance connections and health
- Connection states: Starting → Listening → Active → Retrying → Dead
- Cleanup of dead connections

#### 5. Hub Client (`internal/mcp/hub_client.go`)
- HTTP client for instance MCP servers
- Handles request/response proxying
- Manages connection lifecycle

### Instance Components

#### 1. Instance Server (`internal/mcp/server.go`)
- HTTP-based MCP server (existing)
- Registers with discovery after listening
- Provides all process management tools

#### 2. Registration Flow
```go
// Only after successful net.Listen()
port := listener.Addr().(*net.TCPAddr).Port
registry.Register(path, port, name, hasPackageJSON)
```

### Communication Protocols

#### 1. Stdio Protocol (Hub ↔ MCP Client)
- JSON-RPC 2.0 over stdin/stdout
- Supports streaming responses
- Session management via connection

#### 2. HTTP Protocol (Hub ↔ Instances)
- HTTP POST with JSON-RPC 2.0
- Server-Sent Events for streaming
- MCP ping/pong for health checks

#### 3. Discovery Protocol
- Instance creates JSON file after listening
- File contains: ID, port, path, PID, name
- Hub watches directory for changes
- Connects to new instances automatically

### State Management

#### Connection Manager State Machine
```
┌─────────────┐
│   STARTING  │ (Instance MCP server starting)
└──────┬──────┘
       │ net.Listen() succeeds
       ▼
┌─────────────┐
│  LISTENING  │ (Port acquired, file created)
└──────┬──────┘
       │ Hub connects
       ▼
┌─────────────┐
│   ACTIVE    │ (Responding to pings)
└──────┬──────┘
       │ 3 missed pings
       ▼
┌─────────────┐
│  RETRYING   │ (Exponential backoff)
└─────┬───┬───┘
      │   │ Reconnected
      │   └──────────► ACTIVE
      │ Max retries
      ▼
┌─────────────┐
│    DEAD     │ (Cleanup connection)
└─────────────┘
```

#### Channel-Based Operations
All state changes go through channels:
- `registerChan`: Add new connection
- `unregisterChan`: Remove connection
- `ensureChan`: Health check
- `stateChan`: Update state
- `listChan`: Get active connections

### Error Handling

1. **Instance Registration Failures**
   - Retry with exponential backoff
   - Remove file after max retries
   - Log errors but continue

2. **Connection Failures**
   - Mark as retrying, not dead
   - Attempt reconnection
   - Clean up after max retries

3. **File System Errors**
   - Continue operation without files
   - Use in-memory state only
   - Log warnings

## Implementation Plan

### Phase 1: Stdio Hub Foundation
- Implement hub mode detection
- Create stdio-based MCP server
- Add basic tools (list, connect)

### Phase 2: Instance Discovery
- File watcher implementation
- Registration after listening
- Directory management

### Phase 3: Connection Management
- Channel-based state manager
- Connection lifecycle
- Session mapping

### Phase 4: Tool Proxying
- Request forwarding
- Response streaming
- Error propagation

### Phase 5: Health Monitoring
- MCP ping implementation
- Timeout detection
- Automatic cleanup

### Phase 6: Testing & Verification
- Unit tests for each component
- Integration tests
- End-to-end scenarios

## Success Metrics

1. **Performance**
   - Hub startup time < 100ms
   - Instance discovery < 50ms
   - Tool response time < 200ms overhead

2. **Reliability**
   - Zero crashes in 24-hour operation
   - Correct detection of instance death
   - No orphaned connections

3. **Usability**
   - Single configuration for all projects
   - No error messages on normal operation
   - Intuitive tool interface

## Security Considerations

1. **Local Only**
   - Hub only accepts stdio connections
   - Instances only accept localhost
   - No remote access

2. **Process Isolation**
   - Each instance runs as separate process
   - No shared memory between instances
   - Clean shutdown on disconnect

3. **File Permissions**
   - Instance files readable by user only
   - Temp files in secure directories
   - No sensitive data in files

## Future Enhancements

1. **Remote Instances**
   - SSH tunneling support
   - Authentication tokens
   - Encrypted connections

2. **Instance Groups**
   - Tag instances by project
   - Bulk operations
   - Workspace management

3. **Persistent Sessions**
   - Reconnect to same instance
   - Session state preservation
   - Command history

## Appendix: Design Principles

Following DESIGN.md principles:
1. **Lock-free**: All state management via channels
2. **Network-based**: Connections determine availability
3. **Fail-fast**: Report errors immediately
4. **No blocking**: All operations have timeouts

## Glossary

- **MCP**: Model Context Protocol - Standard for LLM tool integration
- **Hub**: Central multiplexer for instance discovery and control
- **Instance**: Individual brummer process manager
- **stdio**: Standard input/output communication
- **SSE**: Server-Sent Events for streaming responses