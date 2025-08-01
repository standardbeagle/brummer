# Brummer Documentation

This directory contains comprehensive documentation for Brummer, organized into logical sections for easy navigation.

## Documentation Structure

### Core Documentation
- **[CLAUDE.md](../CLAUDE.md)** - Main guidance file for Claude Code with essential commands and references

### Architecture
- **[Overview](architecture/overview.md)** - System architecture, components, and integration patterns
- **[Concurrency Patterns](architecture/concurrency-patterns.md)** - Race condition prevention and safe concurrent programming

### MCP Integration
- **[Integration Overview](mcp/integration-overview.md)** - Architecture and design philosophy
- **[Server Configuration](mcp/server-configuration.md)** - Setup for single instance and hub modes
- **[Tools and Routing](mcp/tools-and-routing.md)** - Complete tool reference and routing patterns
- **[MCP Examples](mcp-examples.md)** - Real-world usage examples
- **[MCP Streaming Data](mcp-streaming-data.md)** - Streaming capabilities and implementation
- **[Hub Mode](hub-mode.md)** - Multi-instance coordination guide

### Configuration & Usage
- **[Configuration Examples](configuration/examples.md)** - Multi-project workflows, proxy setup, browser automation
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

### Features
- **[AI Coders](features/ai-coders.md)** - Tmux-style interactive AI coding sessions
- **[Environment Variable Management](features/environment-variable-management.md)** - Unified .env file management

### Development Resources
- **[Development Roadmap](ROADMAP.md)** - Comprehensive feature roadmap and implementation timeline
- **[Development Plans](development/)** - Implementation analyses and technical planning
- **[Testing Documentation](TESTING_STRATEGY.md)** - Testing approaches and strategies
- **[Distribution](DISTRIBUTION.md)** - Packaging and distribution information
- **[Publishing](PUBLISHING.md)** - Release and publishing processes

## Quick Navigation

- **New Users**: Start with [CLAUDE.md](../CLAUDE.md) for essential commands, then [Architecture Overview](architecture/overview.md)
- **MCP Integration**: Begin with [MCP Integration Overview](mcp/integration-overview.md)
- **Multi-Instance Setup**: See [Hub Mode](hub-mode.md) and [Configuration Examples](configuration/examples.md)
- **AI Features**: Explore [AI Coders](features/ai-coders.md) for interactive AI coding sessions
- **Troubleshooting**: Check [Troubleshooting](troubleshooting.md) for common issues

## Documentation Philosophy

This documentation follows a **progressive disclosure** approach:
1. **Essential information** in the main CLAUDE.md file
2. **Detailed guides** in topical subdirectories
3. **Implementation details** in development documentation
4. **Quick reference** through cross-linked navigation

This structure ensures Claude Code can quickly find essential information while providing comprehensive details when needed.