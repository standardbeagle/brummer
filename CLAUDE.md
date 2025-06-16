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

### Core Components

**Process Management (`internal/process/manager.go`)**
- Manages lifecycle of child processes spawned from package.json scripts
- Supports npm, yarn, pnpm, bun package managers
- Detects monorepo structures (pnpm/npm/yarn workspaces, Lerna, Nx, Rush)
- Auto-detects executable commands for multiple languages (Go, Rust, Java, Python, etc.)
- Uses context.Context for graceful shutdown

**TUI System (`internal/tui/model.go`)**
- Built with Bubble Tea framework (Model-Update-View pattern)
- Multiple views: Scripts, Processes, Logs, Errors, URLs, Settings
- Event-driven updates via EventBus
- Slash command system for filtering: `/show pattern`, `/hide pattern`
- Process status tracking with visual indicators

**Log Management (`internal/logs/store.go`)**
- Thread-safe log storage with configurable size limit
- Error context extraction with language-specific parsing
- URL detection and deduplication
- Regex-based filtering system
- Priority-based log categorization

**MCP Server (`internal/mcp/streamable_server.go`)**
- JSON-RPC 2.0 compliant Model Context Protocol implementation
- Single endpoint `/mcp` exposing multiple tools, resources, and prompts
- Supports real-time streaming for logs, telemetry, and tool execution
- Session-based client management with Server-Sent Events
- Full MCP protocol support: tools/list, tools/call, resources/list, resources/read, prompts/list, prompts/get

**Event System (`pkg/events/events.go`)**
- Central EventBus for component communication
- Event types: ProcessStarted, ProcessExited, LogLine, ErrorDetected, BuildEvent, TestResult
- Asynchronous event propagation between components

**Configuration System (`internal/config/config.go`)**
- TOML-based configuration with hierarchical override chain
- Loads from: current directory → parent directories → root → `~/.brum.toml`
- Source tracking for debugging configuration values
- Supports MCP port, proxy settings, and package manager preferences
- `--settings` flag displays current config with source file comments

### Key Design Patterns

1. **Variable Shadowing**: Be careful not to shadow package names with variables (e.g., don't use `logs` as a variable name when importing the `logs` package)
   - Never shadow variables in go, double check before naming variables
   - Use different naming conventions for go variables and go types to prevent shadowing

2. **Process Cleanup**: All processes are tracked and cleaned up on exit via `processMgr.Cleanup()`

3. **Log Processing Pipeline**: 
   - Process stdout/stderr → LogStore → EventDetector → EventBus → TUI/MCP updates

4. **Error Context**: The error parser (`internal/logs/error_parser.go`) maintains state to capture multi-line errors with context

## MCP (Model Context Protocol) Integration

### Server Configuration
- **Primary Endpoint**: `http://localhost:7777/mcp` (single URL for all MCP functionality)
- **Protocol**: JSON-RPC 2.0 with Server-Sent Events streaming support
- **Default Port**: 7777 (configurable with `-p` or `--port`)
- **Startup**: Automatically enabled unless `--no-mcp` flag is used

### Client Configuration
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

### Available MCP Tools
- **scripts/list**: List all npm/yarn/pnpm/bun scripts from package.json
- **scripts/run**: Execute a script with real-time output streaming
- **scripts/stop**: Stop a running script process
- **scripts/status**: Check the status of running scripts
- **logs/stream**: Stream real-time logs from all processes (supports filtering)
- **logs/search**: Search historical logs with regex patterns and filters
- **proxy/requests**: Get captured HTTP requests from the proxy server
- **telemetry/sessions**: Access browser telemetry session data
- **telemetry/events**: Stream real-time browser telemetry events
- **browser/open**: Open URLs with automatic proxy configuration
- **browser/refresh**: Refresh connected browser tabs
- **browser/navigate**: Navigate browser tabs to new URLs
- **browser/screenshot**: Capture screenshots of browser tabs (limited without extension)
- **repl/execute**: Execute JavaScript in browser context

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
- Resource subscription for live updates
- Session management with automatic cleanup
- Cross-platform compatibility (Windows, macOS, Linux, WSL2)

## Important Notes

- The executable is named `brum` (not `brummer`)
- Browser extension code has been removed from the codebase
- The TUI requires a TTY; use `--no-tui` for headless operation
- MCP server runs on port 7777 by default with single endpoint `/mcp`
- Slash commands use Go regex syntax for pattern matching
- Process IDs are generated as `<scriptname>-<timestamp>`
- URLs are automatically extracted from logs and deduplicated per process
- Rewrite test-script.sh to write proxy testing code so you don't get your scripts manually approved