# Brummer MCP Hub Implementation Guide

## Overview

This folder contains the complete implementation plan for the Brummer MCP Hub architecture. The implementation should be done in strict order, with each step fully tested before moving to the next.

## Implementation Order

1. **00-lessons-learned.md** - Critical mistakes to avoid from the failed attempt
2. **PRD.md** - Complete product requirements and architecture overview
3. **01-stdio-hub-foundation.md** - Basic stdio MCP server
4. **02-instance-discovery.md** - File watching system
5. **03-connection-management.md** - Channel-based connection manager
6. **04-tool-proxying.md** - Tool forwarding to instances
7. **05-health-monitoring.md** - MCP ping/pong implementation
8. **06-testing-verification.md** - Comprehensive testing plan

## Key Principles

1. **Hub uses stdio ONLY** - Never HTTP transport for the hub
2. **Connections are truth** - Files are just signals
3. **Channels, not mutexes** - All state management via channels
4. **Network first** - No file-based heartbeats
5. **Simple is better** - Don't over-engineer

## Pre-Implementation Checklist

Before starting implementation:

- [ ] Read and understand **00-lessons-learned.md**
- [ ] Review the **PRD.md** for complete architecture
- [ ] Ensure `mark3labs/mcp-go` is in go.mod
- [ ] Clean git workspace (no uncommitted changes)
- [ ] Understand the difference between hub mode and instance mode

## Implementation Checklist

### Step 1: Stdio Hub Foundation
- [ ] Add `--mcp` flag to main.go
- [ ] Create runMCPHub() function
- [ ] Implement basic hub_server.go
- [ ] Test with MCP inspector
- [ ] Verify < 100ms startup

### Step 2: Instance Discovery
- [ ] Update instance registration to happen AFTER listening
- [ ] Create file watcher with fsnotify
- [ ] Implement atomic file writes
- [ ] Test discovery timing
- [ ] Verify no blocking operations

### Step 3: Connection Management
- [ ] Create channel-based ConnectionManager
- [ ] Implement state transitions
- [ ] Add HubClient for HTTP connections
- [ ] Test connection lifecycle
- [ ] Verify no goroutine leaks

### Step 4: Tool Proxying
- [ ] Update hub server to proxy tools
- [ ] Implement session mapping
- [ ] Add all MCP methods to HubClient
- [ ] Test tool forwarding
- [ ] Handle streaming responses

### Step 5: Health Monitoring
- [ ] Implement MCP ping/pong
- [ ] Add health monitor component
- [ ] Configure retry logic
- [ ] Test failure detection
- [ ] Verify reconnection works

### Step 6: Testing & Verification
- [ ] Run all unit tests
- [ ] Execute integration tests
- [ ] Test with real MCP clients
- [ ] Perform stress testing
- [ ] Document any issues found

## Common Pitfalls to Avoid

1. **Don't use mutexes** - Use channels for all state
2. **Don't poll files** - Read once, connect, forget
3. **Don't mix transports** - Hub=stdio, Instance=HTTP
4. **Don't register early** - Only after net.Listen()
5. **Don't block** - All operations need timeouts

## Testing Each Step

After implementing each step:

1. Build the project: `make build`
2. Run unit tests: `go test ./...`
3. Test manually as described in each step
4. Fix any issues before proceeding
5. Commit working code

## Resources

- MCP Protocol: https://modelcontextprotocol.io/
- mcp-go library: https://github.com/mark3labs/mcp-go
- Original design: See DESIGN.md in git history

## Getting Help

If stuck on any step:
1. Re-read the lessons learned document
2. Check the PRD for clarification
3. Review the test cases in step 6
4. Keep it simple - don't over-engineer

Remember: **The hub is just a proxy. Keep it simple.**