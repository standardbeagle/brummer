# MCP Server Refactoring Plan (Revised)

## Problem Statement
`mcp_server.go` is 1,591 lines and mixes multiple concerns:
- HTTP server lifecycle
- Library/REPL management (8 handler functions)
- Tool/Resource/Prompt registration (12 functions)
- Session management
- Custom JSON-RPC handling (should use MCP SDK)

## Key Insight
**The MCP SDK (`github.com/mark3labs/mcp-go`) already handles the protocol**. We shouldn't be implementing our own JSON-RPC layer. Instead, we should:
1. Break up business logic by domain
2. Use the MCP SDK properly where possible
3. Keep server lifecycle simple

## Current Structure Analysis

### What Stays in `mcp_server.go` (Core Server)
- Type: `MCPServer` struct
- Lifecycle: `NewMCPServer`, `Start`, `Stop`, `GetPort`, `IsRunning`
- HTTP: `setupRoutes`, `handleRequest`, `handleHealth`, `corsMiddleware`
- Events: `setupEventBroadcasting`
- Dependencies: `SetConnectionManager`, `SetAICoderManager`
- **Target**: ~400 lines

### What Gets Extracted

#### 1. Library Handlers → `internal/mcp/handlers/library.go`
**Lines**: ~300 lines
**Functions**:
- `handleLibraryList()`
- `handleLibraryAdd(params LibraryParams)`
- `handleLibraryRemove(params LibraryParams)`
- `handleLibraryUpdate(params LibraryParams)`
- `handleLibrarySearch(params LibraryParams)`
- `handleLibraryCategories()`
- `handleLibraryGet(params LibraryParams)`
- `handleLibraryInject()`
- `getLibraryManager()`
- `registerREPLResponse()`, `unregisterREPLResponse()`, `handleREPLResponse()`

**New Type**:
```go
type LibraryHandler struct {
    server *MCPServer  // Reference back to server
}

func NewLibraryHandler(s *MCPServer) *LibraryHandler
func (h *LibraryHandler) HandleList() (interface{}, error)
// ... etc
```

#### 2. Registration Manager → `internal/mcp/registry/manager.go`
**Lines**: ~400 lines
**Functions**:
- Tool registration: `RegisterTool`, `UnregisterTool`, `RegisterToolsFromInstance`, `UnregisterToolsFromInstance`
- Resource registration: `RegisterResource`, `UnregisterResource`, `RegisterResourcesFromInstance`, `UnregisterResourcesFromInstance`
- Prompt registration: `RegisterPrompt`, `UnregisterPrompt`, `RegisterPromptsFromInstance`, `UnregisterPromptsFromInstance`

**New Type**:
```go
type Registry struct {
    mu        sync.RWMutex
    tools     map[string]MCPTool
    resources map[string]Resource
    prompts   map[string]Prompt

    // Track which instance owns which tools/resources/prompts
    toolOwners     map[string]string  // tool name -> instance ID
    resourceOwners map[string]string
    promptOwners   map[string]string
}

func NewRegistry() *Registry
func (r *Registry) RegisterTool(name string, tool MCPTool) error
// ... etc
```

#### 3. Session Manager → `internal/mcp/session/manager.go`
**Lines**: ~200 lines
**Functions**:
- `sendSSEEvent()`
- `registerResourceUpdateHandler()`, `unregisterResourceUpdateHandler()`
- `notifyResourceUpdate()`, `setupResourceUpdateHandlers()`

**New Type**:
```go
type Manager struct {
    mu       sync.RWMutex
    sessions map[string]*Session  // Renamed from ClientSession

    resourceUpdateHandlers map[string]chan ResourceUpdate
}

func NewManager() *Manager
func (m *Manager) CreateSession(id string) *Session
func (m *Manager) SendSSE(sessionID string, eventType string, data interface{}) error
// ... etc
```

## Refactoring Steps (TDD)

### Step 1: Extract Library Handlers
1. Create `internal/mcp/handlers/library.go`
2. Move `LibraryParams` type
3. Move all `handleLibrary*` functions
4. Create `LibraryHandler` wrapper type
5. Update `mcp_server.go` to use `LibraryHandler`
6. Run tests → verify

### Step 2: Extract Registration Manager
1. Create `internal/mcp/registry/manager.go`
2. Move tool/resource/prompt registration logic
3. Create `Registry` type with ownership tracking
4. Replace `MCPServer.tools/resources/prompts` with `Registry`
5. Run tests → verify

### Step 3: Extract Session Manager
1. Create `internal/mcp/session/manager.go`
2. Move `ClientSession` → `Session`
3. Create `Manager` type
4. Replace `MCPServer.sessions` with `SessionManager`
5. Run tests → verify

### Step 4: Clean Up Server
1. Remove all extracted code from `mcp_server.go`
2. Add delegation to new components
3. Verify < 500 lines
4. Run full test suite
5. Run `make test-race`

## Success Criteria
- ✅ All tests pass
- ✅ No race conditions
- ✅ `mcp_server.go` < 500 lines (from 1,591)
- ✅ Each file has single clear responsibility
- ✅ Easy to find code (clear file organization)

## File Size Targets

| File | Current | Target | Responsibility |
|------|---------|--------|----------------|
| `mcp_server.go` | 1591 | ~450 | HTTP server lifecycle |
| `handlers/library.go` | 0 | ~300 | REPL/library operations |
| `registry/manager.go` | 0 | ~400 | Tool/resource/prompt registration |
| `session/manager.go` | 0 | ~200 | Client session management |
| `tools.go` | 2447 | 2447 | Tool implementations (separate effort) |

**Note**: We're NOT refactoring `tools.go` in this effort. That's a separate, larger refactoring.
