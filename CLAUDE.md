# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Current Execution Status

⚠️ **CRITICAL INSTRUCTION**: Always update this section when working on tasks to maintain continuity between sessions.

- **Phase**: AI Coder Feature Implementation - Phase 1 COMPLETED
- **Current Task**: Phase 1 Complete - Core Service, MCP Tools, Configuration System
- **Stage**: PHASE_1_COMPLETE - Foundation ready for Phase 2
- **Started**: January 28, 2025
- **Phase 1 Completed**: January 29, 2025
- **Plan Files**: `/requests/agentic-ai-coders/` - Complete subtask breakdown
- **Execution Strategy**: 3-phase parallel development (8 atomic subtasks)

### Phase 1 Results - AI Coder Foundation:
1. ✅ **COMPLETED**: Created comprehensive subtask plan with 8 atomic tasks
2. ✅ **COMPLETED**: Generated context package with architecture analysis
3. ✅ **COMPLETED**: Documented tmux-style interactive AI coder design
4. ✅ **COMPLETED**: Updated CLAUDE.md with full AI coder documentation
5. ✅ **COMPLETED**: Task 01 - Core Service (AICoderManager implementation)
6. ✅ **COMPLETED**: Task 02 - MCP Tools (6 AI coder control tools)
7. ✅ **COMPLETED**: Task 03 - Configuration System (TOML config)
8. ⏳ **PENDING**: Phase 2 execution (tasks 04-06: TUI, process integration, events)
9. ⏳ **PENDING**: Phase 3 execution (tasks 07-08: testing, documentation)

### AI Coder Design Vision:
"This design makes AI coders feel like pair programming sessions where Brummer acts as the development environment providing real-time feedback to both the human and AI."

### Next Agent Instructions:
- **ALWAYS check this status section first** to understand current work context
- **ALWAYS update this section** when starting/completing tasks
- **Follow the subtask execution guide** at `/requests/agentic-ai-coders/subtasks-execute.md`
- **Use worktrees for parallel development** to avoid merge conflicts

## Current Execution Status
- **Phase**: Lock-Free Architecture - Prototype-First Execution Complete
- **Current Task**: Ready for Phase 3 Implementation with Atomic Operations
- **Stage**: PROTOTYPE_VALIDATED
- **Started**: January 31, 2025
- **Completed**: January 31, 2025
- **Method**: Assumption Testing → Plan Adjustment → Validated Approach
- **Result**: Atomic operations validated as 30-300x faster than mutexes

### Lock-Free Architecture Results:
1. ✅ **COMPLETED**: Phase 1 - Fixed race conditions in scripts_status
2. ✅ **COMPLETED**: Phase 2 - ProcessSnapshot pattern (65% improvement)
3. ✅ **COMPLETED**: Assumption Testing - Channels failed (15-67x slower)
4. ✅ **COMPLETED**: Alternative Testing - Atomics succeeded (30-300x faster)
5. ✅ **COMPLETED**: Plan Updated - Pivot to atomic operations approach
6. ⏳ **PENDING**: Phase 3A - Implement atomic ProcessState
7. ⏳ **PENDING**: Phase 3B - Migrate to sync.Map registry
8. ⏳ **PENDING**: Phase 3C - Integration and optimization

### Key Discovery:
Prototype-first methodology saved weeks - discovered channels are wrong tool for shared state. Atomic operations with immutable structs provide massive performance gains.

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

For detailed architecture documentation, see:
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
- **Architecture**: [Overview](/docs/architecture/overview.md), [Concurrency Patterns](/docs/architecture/concurrency-patterns.md)
- **MCP Integration**: [Overview](/docs/mcp/integration-overview.md), [Configuration](/docs/mcp/server-configuration.md), [Tools](/docs/mcp/tools-and-routing.md)
- **Configuration**: [Examples](/docs/configuration/examples.md), [Hub Mode](/docs/hub-mode.md)
- **Features**: [AI Coders](/docs/features/ai-coders.md), [Test Management](/docs/features/test-management.md), [Environment Variables](/docs/features/environment-variable-management.md)
- **Development**: [Roadmap](/docs/ROADMAP.md) - Comprehensive feature plans and timelines
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