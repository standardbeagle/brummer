# MCP Server Refactoring Status

## Summary
Started TDD refactoring to break up `mcp_server.go` (1,591 lines) into focused components.

## Key Learnings

### 1. MCP SDK Already Handles Protocol
- The `github.com/mark3labs/mcp-go` SDK implements the MCP protocol
- We shouldn't create our own `protocol` package
- Custom JSON-RPC types in `mcp_server.go` are likely for backward compatibility or specific needs

### 2. Complexity is in Business Logic, Not Protocol
The 1,591 lines break down as:
- **Library/REPL handlers**: ~300 lines (8 functions)
- **Registration system**: ~400 lines (12 functions for tools/resources/prompts)
- **Session management**: ~200 lines
- **HTTP/Server lifecycle**: ~450 lines
- **Custom JSON-RPC**: ~85 lines (types + error helpers)
- **Helper/util functions**: ~156 lines

### 3. Some Handlers Have Deep Dependencies
Example: `handleLibraryInject()` needs:
- Library manager
- Proxy server (for WebSocket broadcast)
- REPL response channel management
- Timeout handling

This makes simple extraction difficult without careful interface design.

## Attempted Approach

### What Was Tried
1. Created `internal/mcp/protocol/` package - **WRONG APPROACH**
   - MCP SDK already provides protocol types
   - Deleted this work

2. Started extracting `internal/mcp/handlers/library.go`
   - Created interface: `LibraryManagerProvider`
   - Moved params and simple handlers
   - Hit complexity with `handleLibraryInject()`

### What Blocked Progress
- `handleLibraryInject()` requires proxy server, REPL channels, timeouts
- Simple extraction creates too many dependencies
- Need better interface design

## Recommended Next Steps

### Option 1: Simpler Refactoring (Recommended)
Instead of full extraction, do **file splitting within same package**:

```
internal/mcp/
├── mcp_server.go          (core server, ~450 lines)
├── server_library.go      (library handlers, ~300 lines)
├── server_registration.go (registration, ~400 lines)
├── server_sessions.go     (sessions, ~200 lines)
└── server_helpers.go      (utils, ~150 lines)
```

**Advantages**:
- No interface complexity
- All methods still on `MCPServer`
- Just logical file organization
- Easy to do incrementally
- Tests still work

**Implementation**:
1. Create `server_library.go` with all `handleLibrary*` methods
2. Create `server_registration.go` with all `Register*/Unregister*` methods
3. Create `server_sessions.go` with session management
4. Leave core in `mcp_server.go`
5. Run tests after each file split

### Option 2: Careful Interface Extraction
If we want true separation:

1. Design interfaces that capture all dependencies:
   ```go
   type ServerDependencies interface {
       GetLibraryManager() *repl.LibraryManager
       GetProxyServer() *proxy.Server
       RegisterREPLResponse(id string) chan interface{}
       UnregisterREPLResponse(id string)
   }
   ```

2. Extract handlers that use this interface
3. Much more work, questionable benefit

### Option 3: Focus on `tools.go` Instead
- `tools.go` is 2,447 lines (larger problem!)
- Might be easier to split by tool category:
  - `tools_process.go` - Process management tools
  - `tools_browser.go` - Browser automation tools
  - `tools_telemetry.go` - Telemetry tools
  - etc.

## Recommendation

**Do Option 1** (file splitting within package) first:
- Quickest path to better organization
- Lowest risk
- Solves the "can't find anything" problem
- Can always extract to separate packages later if needed

The goal is **readability and maintainability**, not architectural purity.

## Files Created (To Be Removed or Reused)
- `/home/beagle/work/brummer/internal/mcp/handlers/library.go` - Partial work
- `/home/beagle/work/brummer/internal/mcp/handlers/library_test.go` - Partial work
- `/home/beagle/work/brummer/docs/refactoring/mcp-server-refactoring.md` - Planning doc

## Time Spent
~1 hour on analysis and first extraction attempt

## Next Session Should
1. Delete partial `handlers/` work
2. Implement Option 1 (file splitting)
3. Start with `server_library.go` - just move functions, no interface changes
4. Test after each file
5. Should take ~30 minutes total
