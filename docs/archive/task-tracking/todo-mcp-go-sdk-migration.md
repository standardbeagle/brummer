# Todo: Migration to Official MCP Go SDK

**Generated from**: Full Planning on 2025-07-12
**Next Phase**: Implementation with Feature Flags
**Analysis Date**: 2025-07-12
**Risk Level**: HIGH | **Project Phase**: Production 
**Estimated Effort**: 16-24 hours | **Files**: ~15 files
**Feature Flag Required**: YES (mandatory for all MCP functionality replacement)

## Context & Background

**Request**: Migrate from custom MCP implementation to official Model Context Protocol Go SDK
**Business Impact**: Ensure compatibility with latest MCP spec and reduce maintenance burden
**Technical Debt**: Current custom implementation may drift from spec; official SDK provides better long-term support

### Codebase Context

**Current MCP Implementation**: 
- ✅ Custom JSON-RPC 2.0 server in `internal/mcp/streamable_server.go` (1,188 lines)
- ✅ Legacy MCP server in `internal/mcp/server.go` (468 lines) 
- ✅ Hub routing and multi-instance coordination in `internal/mcp/hub_*.go`
- ✅ Tool/resource/prompt registration and management
- ❌ No official SDK integration
- ⚠️ Mix of custom transport with some compatibility layers

**Key Architecture Components**:
- `StreamableServer` - Main MCP server with HTTP transport and SSE streaming
- `ConnectionManager` - Hub coordination for multi-instance routing  
- `HubClient` - HTTP client for instance communication
- Tool/Resource/Prompt registries with dynamic management
- Browser automation and proxy integration

**Dependencies**: 
- `github.com/mark3labs/mcp-go@v0.32.0` - Unofficial MCP SDK (currently used minimally)
- `github.com/gorilla/mux@v1.8.1` - HTTP routing
- `github.com/gorilla/websocket@v1.5.3` - WebSocket support
- `github.com/google/uuid@v1.6.0` - Session management

### External Context Sources

**Primary Documentation**:
- [Official Go SDK](https://github.com/modelcontextprotocol/go-sdk) - Stable release expected August 2025, maintained by Google
- [MCP Specification 2025-06-18](https://modelcontextprotocol.io/specification/2025-06-18) - Latest protocol specification
- [SDK Design Document](https://github.com/modelcontextprotocol/go-sdk/blob/main/design/design.md) - Architecture patterns and examples

**Key Insights from Research**:
- **Stability**: Official SDK still in development, breaking changes expected until August 2025
- **Architecture**: Single `mcp` package with generics for type safety and schema inference
- **Transport**: Official support for stdio, SSE, and streamable HTTP
- **Migration Path**: Can run in parallel during transition period

**Standards Applied**:
- [MCP Streamable HTTP Transport](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports) - Current implementation follows this
- [JSON-RPC 2.0](https://www.jsonrpc.org/specification) - Protocol foundation

### Current vs Target Architecture

**Current Custom Implementation**:
```
Custom StreamableServer → Custom JSON-RPC → Custom Transport
     ↓                           ↓               ↓
Tool Registry              Message Routing    HTTP+SSE
Resource Registry          Error Handling     WebSocket  
Prompt Registry           Session Management  Legacy REST
```

**Target Official SDK**:
```
mcp.Server → mcp.Transport → Official Protocol Implementation
     ↓             ↓                    ↓
Generic Tools   stdio/SSE/HTTP    Spec Compliance
Auto Schema     Type Safety       Future Updates
```

## Implementation Plan

### Phase 1: SDK Integration Foundation (Risk: HIGH)
**Files**: `go.mod`, `internal/mcp/sdk_server.go` (new), `internal/mcp/feature_flags.go` (new)
**Objective**: Add official SDK and create parallel implementation
**Validation**: Official SDK server starts successfully alongside current implementation

- [ ] **Task 1.1**: Add official MCP Go SDK dependency 
  - **Risk**: HIGH - SDK still in development, breaking changes expected
  - **Files**: `go.mod`
  - **Action**: `go get github.com/modelcontextprotocol/go-sdk@latest`
  - **Success Criteria**: 
    - [ ] SDK dependency added successfully
    - [ ] No conflicts with existing dependencies
    - [ ] Build succeeds: `go build ./...`
  - **Rollback**: `go mod tidy` to remove if build fails

- [ ] **Task 1.2**: Create feature flag system for MCP implementation choice
  - **Risk**: MEDIUM - Critical for safe deployment
  - **Files**: `internal/mcp/feature_flags.go`, `internal/config/config.go`
  - **Action**: Add `use_official_mcp_sdk` boolean flag (default: false)
  - **Success Criteria**:
    - [ ] Feature flag controllable via config file and CLI
    - [ ] Current implementation remains default
    - [ ] Flag accessible throughout MCP package
  - **Implementation**: Environment variable `BRUMMER_USE_OFFICIAL_MCP_SDK=true`

- [ ] **Task 1.3**: Create parallel SDK-based server implementation
  - **Risk**: HIGH - New patterns and potential compatibility issues
  - **Files**: `internal/mcp/sdk_server.go` (new)
  - **Action**: Implement basic `mcp.Server` with core tools
  - **Success Criteria**:
    - [ ] SDK server starts on different port (e.g. 7778)
    - [ ] Basic initialize/tools/list methods work
    - [ ] No interference with existing server
  - **Validation**: `curl http://localhost:7778/mcp -d '{"jsonrpc":"2.0","id":1,"method":"initialize"}'`

### Phase 2: Tool Migration (Risk: MEDIUM)
**Files**: `internal/mcp/sdk_tools.go` (new), `internal/mcp/tools.go` (update)
**Objective**: Migrate core tools to use official SDK patterns
**Validation**: All current tools work with SDK implementation

- [ ] **Task 2.1**: Migrate script management tools
  - **Risk**: MEDIUM - Complex integration with ProcessManager
  - **Files**: `internal/mcp/sdk_tools.go`
  - **Action**: Convert `scripts_list`, `scripts_run`, `scripts_stop`, `scripts_status` to SDK
  - **Success Criteria**:
    - [ ] Tools use SDK's generic tool binding
    - [ ] Type-safe parameter validation
    - [ ] Integration with existing ProcessManager unchanged
  - **Pattern**: `mcp.AddTool(server, &mcp.Tool{Name: "scripts_list"}, handler)`

- [ ] **Task 2.2**: Migrate log management tools  
  - **Risk**: MEDIUM - Streaming considerations
  - **Files**: `internal/mcp/sdk_tools.go`
  - **Action**: Convert `logs_stream`, `logs_search` to SDK patterns
  - **Success Criteria**:
    - [ ] Streaming logs work with SDK transport
    - [ ] Search functionality preserved
    - [ ] Performance equivalent to current implementation

- [ ] **Task 2.3**: Migrate proxy and browser tools
  - **Risk**: MEDIUM - Complex browser automation integration
  - **Files**: `internal/mcp/sdk_tools.go`
  - **Action**: Convert proxy/browser tools to SDK
  - **Success Criteria**:
    - [ ] Browser automation preserved
    - [ ] Proxy request capture works
    - [ ] Screenshot and REPL tools functional

### Phase 3: Transport and Hub Integration (Risk: HIGH)
**Files**: `internal/mcp/sdk_hub.go` (new), `internal/mcp/connection_manager.go` (update)
**Objective**: Integrate SDK with hub coordination system
**Validation**: Multi-instance coordination works with SDK

- [ ] **Task 3.1**: Implement hub coordination with SDK
  - **Risk**: HIGH - Complex multi-instance routing
  - **Files**: `internal/mcp/sdk_hub.go`
  - **Action**: Adapt ConnectionManager to work with SDK servers
  - **Success Criteria**:
    - [ ] Hub can route to SDK-based instances
    - [ ] Tool discovery works across SDK and custom instances
    - [ ] Session management preserved

- [ ] **Task 3.2**: Transport compatibility layer
  - **Risk**: HIGH - Ensure all transport modes work
  - **Files**: `internal/mcp/sdk_server.go`
  - **Action**: Verify stdio, HTTP, and SSE transports work
  - **Success Criteria**:
    - [ ] All current MCP clients can connect
    - [ ] Performance equivalent to current implementation
    - [ ] Streaming and batching work correctly

### Phase 4: Resource and Prompt Migration (Risk: MEDIUM)
**Files**: `internal/mcp/sdk_resources.go` (new), `internal/mcp/sdk_prompts.go` (new)
**Objective**: Migrate resources and prompts to SDK patterns
**Validation**: All resources and prompts work with SDK

- [ ] **Task 4.1**: Migrate MCP resources
  - **Risk**: MEDIUM - Resource subscription complexity
  - **Files**: `internal/mcp/sdk_resources.go`
  - **Action**: Convert logs, telemetry, proxy resources to SDK
  - **Success Criteria**:
    - [ ] Resource subscriptions work
    - [ ] Real-time updates preserved
    - [ ] URI patterns maintained for compatibility

- [ ] **Task 4.2**: Migrate MCP prompts
  - **Risk**: LOW - Simpler than tools and resources
  - **Files**: `internal/mcp/sdk_prompts.go`
  - **Action**: Convert debugging prompts to SDK format
  - **Success Criteria**:
    - [ ] All current prompts work
    - [ ] Parameter validation preserved

### Phase 5: Testing and Validation (Risk: MEDIUM)
**Files**: `internal/mcp/sdk_test.go` (new), `test/mcp_sdk_test.go` (new)
**Objective**: Comprehensive testing of SDK implementation
**Validation**: All tests pass, performance equivalent

- [ ] **Task 5.1**: Create SDK-specific tests
  - **Risk**: MEDIUM - Need comprehensive coverage
  - **Files**: `internal/mcp/sdk_test.go`
  - **Action**: Test all SDK tools, resources, prompts
  - **Success Criteria**:
    - [ ] 100% test coverage for SDK implementation
    - [ ] Integration tests pass
    - [ ] Performance benchmarks meet requirements

- [ ] **Task 5.2**: End-to-end testing with real clients
  - **Risk**: MEDIUM - Real-world compatibility
  - **Files**: `test/mcp_sdk_e2e_test.go`
  - **Action**: Test with Claude Desktop, VS Code, etc.
  - **Success Criteria**:
    - [ ] All MCP clients work with SDK server
    - [ ] Hub coordination functions correctly
    - [ ] No regressions in functionality

### Phase 6: Migration Completion (Risk: LOW)
**Files**: Multiple files for cleanup
**Objective**: Switch default to SDK, remove custom implementation
**Validation**: Production deployment successful

- [ ] **Task 6.1**: Switch default to SDK implementation
  - **Risk**: LOW - Feature flag makes this safe
  - **Files**: `internal/config/config.go`
  - **Action**: Change default `use_official_mcp_sdk` to `true`
  - **Success Criteria**:
    - [ ] SDK becomes default implementation
    - [ ] Custom implementation still available via flag
    - [ ] All functionality preserved

- [ ] **Task 6.2**: Remove custom implementation (Future)
  - **Risk**: LOW - Can be done after successful deployment
  - **Files**: `internal/mcp/server.go`, `internal/mcp/streamable_server.go`
  - **Action**: Clean up old implementation after proven stable
  - **Timeline**: 2-4 weeks after successful SDK deployment

## Gotchas & Considerations

**Known Issues**:
- **SDK Stability**: Official SDK still in development until August 2025
- **Breaking Changes**: Expect API changes before stable release
- **Transport Differences**: May need adaptation layer for current clients
- **Performance**: SDK overhead vs custom implementation needs measurement

**Edge Cases**:
- **Concurrent Access**: Ensure SDK handles multi-session scenarios
- **Resource Subscriptions**: Complex real-time update patterns
- **Hub Coordination**: Multi-instance routing may need custom logic
- **Error Handling**: SDK error patterns vs current custom errors

**Performance Considerations**:
- **Memory Usage**: SDK may have different memory patterns
- **Latency**: Generic type safety might add overhead
- **Throughput**: Streaming performance needs validation
- **Resource Usage**: Monitor CPU/memory impact

**Backwards Compatibility**:
- **MCP Clients**: All current clients must continue working
- **API Contracts**: Tool/resource URIs and schemas preserved
- **Hub Protocol**: Instance coordination unchanged
- **Configuration**: Existing config files remain valid

**Security**:
- **SDK Vulnerabilities**: Monitor official SDK for security updates
- **Transport Security**: Ensure all security measures preserved
- **Input Validation**: SDK validation vs custom validation
- **Session Management**: Security patterns maintained

## Migration Strategy & Risk Mitigation

### Feature Flag Strategy
```go
// Feature flag controls implementation choice
if config.UseOfficialMCPSDK {
    server = NewSDKServer(...)  // Official SDK
} else {
    server = NewStreamableServer(...)  // Current custom
}
```

### Parallel Operation Period
- **Duration**: 4-6 weeks minimum in production
- **Monitoring**: Compare performance, error rates, functionality
- **Rollback**: Instant via feature flag toggle
- **Validation**: All current functionality verified

### Deployment Strategy
1. **Development**: SDK implementation with feature flag (default: false)
2. **Testing**: Parallel testing of both implementations
3. **Staging**: Feature flag enabled, comprehensive testing
4. **Production**: Gradual rollout with monitoring
5. **Cleanup**: Remove custom implementation after proven stable

## Definition of Done

- [ ] Official MCP Go SDK integrated successfully
- [ ] All current tools/resources/prompts work with SDK
- [ ] Hub coordination preserved and functional
- [ ] All MCP clients (Claude Desktop, VS Code) work correctly
- [ ] Performance equivalent or better than custom implementation
- [ ] Feature flag system allows safe rollback
- [ ] Comprehensive tests pass: `make test`
- [ ] No security regressions
- [ ] Documentation updated for SDK usage
- [ ] Production deployment successful with monitoring

## Execution Notes

- **Start with**: Task 1.1 (Add SDK dependency) - Foundation for all other work
- **Validation**: Run `make test` and `make build` after each task
- **Commit pattern**: `mcp: [action taken]` (e.g., "mcp: add official SDK dependency")
- **Monitoring**: Track memory usage, latency, and error rates throughout migration
- **Communication**: Feature flag allows easy A/B testing and rollback

## Risk Communication

⚠️ **HIGH RISK ITEMS REQUIRING APPROVAL**:
- Official SDK still in development (breaking changes expected until August 2025)
- Complete replacement of core MCP functionality (business critical)
- Multi-instance hub coordination complexity
- Potential performance impact from SDK overhead

✅ **MITIGATION**: Feature flag system allows instant rollback to current implementation
🛑 **RECOMMENDATION**: Wait for August 2025 stable release OR proceed with feature flag safety net

**Next**: Begin implementation with Task 1.1 - Add SDK dependency with feature flag protection