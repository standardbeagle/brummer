# MCP Implementation Tests

This directory contains comprehensive tests for the Model Context Protocol (MCP) implementation in Brummer.

## Test Structure

The tests are organized into several files:

### Core Protocol Tests
- `streamable_server_test.go` - Tests for the MCP Streamable HTTP transport implementation
  - JSON-RPC 2.0 protocol compliance
  - SSE (Server-Sent Events) streaming
  - Session management
  - CORS handling
  - Concurrent request handling

### Feature Tests
- `tools_test.go` - Tests for MCP tool registration and execution
  - Tool discovery via `tools/list`
  - Tool execution via `tools/call`
  - Tool input validation
  - Streaming tool support

- `resources_test.go` - Tests for MCP resource handling
  - Resource listing via `resources/list`
  - Resource reading via `resources/read`
  - Resource subscriptions
  - Resource update notifications

- `prompts_test.go` - Tests for MCP prompt handling
  - Prompt listing via `prompts/list`
  - Prompt generation via `prompts/get`
  - Argument validation
  - Dynamic content generation

### Integration Tests
- `integration_test.go` - End-to-end protocol tests
  - MCP Inspector CLI integration (when available)
  - Protocol edge cases
  - Streaming edge cases
  - Notification broadcasting
  - Performance benchmarks

## Running Tests

### Run all MCP tests:
```bash
go test -v ./internal/mcp
```

### Run specific test suites:
```bash
# Protocol tests only
go test -v ./internal/mcp -run "TestJSONRPCProtocol|TestSSEStreaming"

# Tool tests only
go test -v ./internal/mcp -run "TestTool"

# Resource tests only
go test -v ./internal/mcp -run "TestResource"

# Prompt tests only
go test -v ./internal/mcp -run "TestPrompt"
```

### Run with coverage:
```bash
go test -v -cover ./internal/mcp
```

### Run benchmarks:
```bash
go test -bench=. ./internal/mcp
```

### Use the test script:
```bash
./test-mcp.sh
```

## Test Approach

The tests follow the recommendations for testing MCP implementations:

1. **Protocol Compliance** - Validates JSON-RPC 2.0 message format, error codes, and response structure
2. **Streaming Support** - Tests SSE connections, message delivery, and connection lifecycle
3. **Tool Registration** - Verifies tools are properly registered with correct schemas
4. **Resource Management** - Tests resource CRUD operations and subscription mechanisms
5. **Concurrent Safety** - Validates thread-safe operation under concurrent load
6. **Edge Cases** - Tests malformed requests, large payloads, unicode handling, etc.

## Known Limitations

Some tests are skipped due to environmental constraints:
- MCP Inspector integration tests require the `mcp-inspector` CLI tool
- Script execution tests require a valid `package.json` with defined scripts
- Process management tests require actual executables in PATH

## Adding New Tests

When adding new MCP features, ensure you:
1. Add unit tests for the feature logic
2. Add integration tests for the MCP protocol endpoints
3. Test edge cases and error conditions
4. Add benchmarks for performance-critical paths
5. Update this README with any new test patterns