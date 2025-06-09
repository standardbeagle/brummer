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

**MCP Server (`internal/mcp/server.go`)**
- RESTful API for external tool integration
- Server-Sent Events (SSE) for real-time updates
- Token-based authentication for clients
- Endpoints for script execution, log retrieval, process management

**Event System (`pkg/events/events.go`)**
- Central EventBus for component communication
- Event types: ProcessStarted, ProcessExited, LogLine, ErrorDetected, BuildEvent, TestResult
- Asynchronous event propagation between components

### Key Design Patterns

1. **Variable Shadowing**: Be careful not to shadow package names with variables (e.g., don't use `logs` as a variable name when importing the `logs` package)
   - Never shadow variables in go, double check before naming variables
   - Use different naming conventions for go variables and go types to prevent shadowing

2. **Process Cleanup**: All processes are tracked and cleaned up on exit via `processMgr.Cleanup()`

3. **Log Processing Pipeline**: 
   - Process stdout/stderr → LogStore → EventDetector → EventBus → TUI/MCP updates

4. **Error Context**: The error parser (`internal/logs/error_parser.go`) maintains state to capture multi-line errors with context

## Important Notes

- The executable is named `brum` (not `brummer`)
- Browser extension code has been removed from the codebase
- The TUI requires a TTY; use `--no-tui` for headless operation
- MCP server runs on port 7777 by default
- Slash commands use Go regex syntax for pattern matching
- Process IDs are generated as `<scriptname>-<timestamp>`
- URLs are automatically extracted from logs and deduplicated per process
- Rewrite test-script.sh to write proxy testing code so you don't get your scripts manually approved