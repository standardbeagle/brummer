# CLAUDE.md

## 🚨 CRITICAL TUI TESTING RULE

**The Bash tool runs in a non-TTY environment, but the system has TTY support available.**

- **Bash tool**: `timeout 5s ./brum --no-tui` ✅ Works (no TTY needed)
- **Bash tool**: `timeout 5s ./brum` ❌ Fails (needs TTY, no /dev/tty)
- **TUI testing**: MUST use integration tests or scripts with TTY allocation

### **How to Test TUI Properly:**
```bash
# ✅ Use integration tests (they have TTY allocation)
BRUMMER_BINARY=./brum go test -tags=integration ./test -v -run TestFullStackTUI

# ✅ Use scripts that allocate TTY properly
./test-tty-script.sh  # if exists

# ❌ NEVER use direct Bash tool for TUI testing
timeout 5s ./brum  # This will fail with "no such device or address"
```

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Current Development Status

⚠️ **CRITICAL INSTRUCTION**: Always update this section when working on tasks to maintain continuity between sessions.

- **Current State**: Keyboard handling architecture completely simplified (007-build-a-testing branch)
- **Recent Work**: Fixed critical keyboard bug - commands were being discarded in InputService.RouteKeyMessage (line 347)
  - Root cause: Service layer returned bool instead of tea.Cmd, breaking BubbleTea command chain
  - Solution: Implemented KeyHandlerRegistry pattern - single routing path with priority-based handlers
  - Removed complex multi-layer keyboard routing through MessageRouter/ViewSpecificHandler/InputComponent
- **Architecture**: Event-driven with Process Manager, Hub System, Proxy Server, and AI Coder integration
- **Branch**: 007-build-a-testing (keyboard handling simplification)

### Major Completed Work:
- ✅ Message Router pattern implementation for TUI model refactoring
- ✅ Lock-free architecture with atomic operations (30-300x performance improvement)
- ✅ AI Coder foundation with tmux-style interactive coding sessions
- ✅ Hub mode for multi-instance coordination via MCP
- ✅ Comprehensive MCP server with browser automation and telemetry

### Next Agent Instructions:
- **ALWAYS check this status section first** to understand current work context
- **ALWAYS update this section** when starting/completing tasks
- **Run `make test-race` before any concurrency changes** to verify thread safety
- **Prefer atomic operations over mutexes** for shared state (see concurrency patterns)
- **Follow event-driven architecture** - use EventBus for component communication

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
# Unit tests (fast, reliable packages)
make test                    # Core unit tests
make test-fast              # Fastest unit tests only
make test-verbose           # Unit tests with verbose output

# Concurrency testing (CRITICAL for race conditions)
make test-race              # Race detection on core packages
make test-race-all          # Comprehensive race detection
CGO_ENABLED=1 go test -race -timeout 10m ./...  # Manual race detection

# Regression testing
make test-regression        # Full integration test suite
make test-mcp              # MCP-specific tests
make test-proxy            # Proxy server tests
make test-processes        # Process management tests

# Code quality
make fmt                   # Format code with go fmt
make lint                  # Run golangci-lint
make test-coverage         # Coverage report (coverage.html)

# Build for all platforms
make build-all             # Creates binaries in ./dist/
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

### High-Level System Design
Brummer consists of three integrated systems:
- **Process Manager**: Script execution, log capture, URL discovery (`internal/process/`)
- **Hub System**: Multi-instance coordination, MCP routing (`internal/mcp/`) 
- **Proxy Server**: HTTP interception, browser automation (`internal/proxy/`)

Components communicate via **event-driven architecture** using EventBus (`pkg/events/events.go`).

### Critical Concurrency Patterns
**Lock-Free Philosophy**: Prefer atomic operations over mutexes for performance
- Use `sync.Map` for concurrent map operations
- Use atomic operations with immutable structs for shared state
- **TUI Components**: ONLY modify model state in BubbleTea Update() method
- **EventBus**: Use bounded worker pools, never unlimited goroutines

**Testing Requirements**:
- ALWAYS run `make test-race` before commits involving concurrency
- Race detection is integrated in CI/CD pipeline
- Use message-passing via tea.Cmd for TUI state changes

For detailed patterns, see:
- [Architecture Overview](/docs/architecture/overview.md) - Core components and integration patterns
- [Concurrency Patterns](/docs/architecture/concurrency-patterns.md) - Race condition prevention and safe concurrent programming

## MCP (Model Context Protocol) Integration

For comprehensive MCP documentation, see:
- [MCP Integration Overview](/docs/mcp/integration-overview.md) - Architecture and design philosophy
- [Server Configuration](/docs/mcp/server-configuration.md) - Setup for single instance and hub modes
- [Tools and Routing](/docs/mcp/tools-and-routing.md) - Complete tool reference and routing patterns
- [Hub Mode Guide](/docs/hub-mode.md) - Multi-instance coordination
- [MCP Examples](/docs/mcp-examples.md) - Real-world usage examples

## Practical Examples & Configuration

For practical usage examples and configuration:
- [Configuration Examples](/docs/configuration/examples.md) - Multi-project workflows, proxy setup, browser automation
- [Troubleshooting Guide](/docs/troubleshooting.md) - Common issues and solutions

## Agentic AI Coders Feature

For detailed AI Coders documentation, see:
- [AI Coders Feature Guide](/docs/features/ai-coders.md) - Tmux-style interactive AI coding sessions
- [Environment Variable Management](/docs/features/environment-variable-management.md) - Unified .env file management

## Documentation Index

### Essential References
- **Commands**: Build, run, test commands and CLI usage above
- **Architecture**: [Overview](/docs/architecture/overview.md), [Concurrency Patterns](/docs/architecture/concurrency-patterns.md), [Key Handling](/docs/architecture/key-handling.md)
- **MCP Integration**: [Overview](/docs/mcp/integration-overview.md), [Configuration](/docs/mcp/server-configuration.md), [Tools](/docs/mcp/tools-and-routing.md)
- **Configuration**: [Examples](/docs/configuration/examples.md), [Hub Mode](/docs/hub-mode.md)
- **Features**: [AI Coders](/docs/features/ai-coders.md), [Test Management](/docs/features/test-management.md), [Environment Variables](/docs/features/environment-variable-management.md)
- **Development**: [Roadmap](/docs/ROADMAP.md), [Keyboard Migration Guide](/docs/development/keyboard-migration-guide.md)
- **Troubleshooting**: [Common Issues](/docs/troubleshooting.md)

## Important Notes

- The executable is named `brum` (not `brummer`)
- The TUI requires a TTY; use `--no-tui` for headless operation
- MCP server runs on port 7777 by default with single endpoint `/mcp`
- Process IDs are generated as `<scriptname>-<timestamp>`
- URLs are automatically extracted from logs and deduplicated per process
- Hub mode requires file-based discovery for instance coordination
- Proxy reverse mode creates shareable URLs for detected endpoints

### Slash Command Routing
When an AI coder session is active, slash commands are context-aware:
- **"/" at start of line**: Opens Brummer command palette
- **"/" mid-line**: Sent to AI coder as regular input
- **When terminal not focused**: Always opens Brummer command palette
